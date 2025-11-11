package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"net"

	"log/slog"

	"github.com/damonto/euicc-go/driver/localnet"
	"github.com/damonto/euicc-go/driver/qmi"
	"github.com/damonto/euicc-go/driver/qmi/core"
	"github.com/damonto/euicc-go/lpa"
)

var (
	options lpa.Options
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	//deviceFlag := flag.String("device", "", "Device if required: /dev/cwm0")
	protoFlag := flag.String("proto", "qrtr", "Protocol: qmi, qrtr, mbim, at")
	slotFlag := flag.Int("slot", 2, "SIM slot where eSIM is installed (for QMI/QRTR)")
	//aidFlag := flag.ByteArray("aid", null, "AID opts value")
	mssFlag := flag.Int("mss", 0, "MSS opts value")
	bindAddrFlag := flag.String("bindAddr", "0.0.0.0", "Binding address (default 0.0.0.0)")
	bindPortFlag := flag.Int("bindPort", 8080, "Binding port (default 8080)")

	flag.Parse()

	//options.AID = *aidFlag
	options.MSS = *mssFlag
	options.AdminProtocolVersion = "2"

	switch *protoFlag {
	//case "at":
	//case "mbim":
	//case "qmi":
	case "qrtr":
		gob.Register(core.QMIError(0))
		var err error
		options.Channel, err = qmi.NewQRTR(uint8(*slotFlag))
		if err != nil {
			panic(err)
		}
	default:
		panic("No handler for the specified protocol")
	}

	if err := options.Normalize(); err != nil {
		panic("Error during options normalization")
	}

	defer options.Channel.Disconnect()

	// udp/tcp server here
	addr := net.UDPAddr{
		Port: *bindPortFlag,
		IP:   net.ParseIP(*bindAddrFlag),
	}

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		fmt.Println("Error on socket server listening:", err)
		return
	}
	defer conn.Close()

	buffer := make([]byte, 512)

outer:
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Printf("error reading from socket %s\n", err)
			break
		}

		pcRcv := localnet.PacketCmd{}
		err = pcRcv.Decode(buffer[:n])
		if err != nil {
			fmt.Printf("Error decoding packet. Closing server\n")
			break
		}

		fmt.Printf("\n=== DEBUG PACKET ===\n")
		fmt.Printf("Cmd: %s\n", pcRcv.Cmd)
		fmt.Printf("Body hex: %X\n", pcRcv.Body)
		fmt.Printf("==================\n\n")

		pcSnd := localnet.PacketCmd{
			Cmd:  "response",
			Body: nil,
			Err:  nil,
		}

		switch pcRcv.Cmd {
		case "exit":
			fmt.Printf("Receiving 'exit' command. Closing server\n")
			break outer

		case "connect":
			pcSnd.Err = options.Channel.Connect()

		case "disconnect":
			pcSnd.Err = options.Channel.Disconnect()

		case "openlogicalchannel":
			var channel byte
			channel, pcSnd.Err = options.Channel.OpenLogicalChannel(pcRcv.Body)
			pcSnd.Body = []byte{channel}

		case "closelogicalchannel":
			pcSnd.Err = options.Channel.CloseLogicalChannel(pcRcv.Body[0])

		case "transmit":
			pcSnd.Body, pcSnd.Err = options.Channel.Transmit(pcRcv.Body)
			if pcSnd.Err != nil {
				fmt.Printf("Error on transmit: %s\n", pcSnd.Err)
			}
			fmt.Printf("Receiving raw from channel: %X\n", pcSnd.Body)

		default:
			fmt.Printf("Receiving unknown command. Closing server\n")
			break outer
		}

		byteArrayResponse, err := pcSnd.Encode()
		if err != nil {
			fmt.Printf("Error encoding response: %s\n", err)
			break
		}

		_, err = conn.WriteToUDP(byteArrayResponse, remoteAddr)
		if err != nil {
			fmt.Printf("Error sending response to the client: %s\n", err)
			break
		}

	}
}
