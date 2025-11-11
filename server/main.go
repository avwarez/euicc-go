package main

import (
	"flag"
	"fmt"
	"net"

	"log/slog"

	"github.com/damonto/euicc-go/driver/localnet"
	"github.com/damonto/euicc-go/driver/qmi"
	"github.com/damonto/euicc-go/lpa"
)

var (
	options lpa.Options
	proto   string
	slot    uint8
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

	proto = *protoFlag
	slot = uint8(*slotFlag)
	//options.AID = *aidFlag
	options.MSS = *mssFlag
	options.AdminProtocolVersion = "2"

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
			Err:  "",
		}

		switch pcRcv.Cmd {

		case "connect":

			if options.Channel == nil {
				switch proto {
				//case "at":
				//case "mbim":
				//case "qmi":
				case "qrtr":
					options.Channel, err = qmi.NewQRTR(slot)
					if err != nil {
						pcSnd.Err = err.Error()
					} else {

						err = options.Channel.Connect()
						if err != nil {
							pcSnd.Err = err.Error()
							options.Channel = nil
						}
					}
				default:
					pcSnd.Err = "No handler for the specified protocol"
				}
			}

		case "disconnect":
			err = options.Channel.Disconnect()
			options.Channel = nil
			if err != nil {
				pcSnd.Err = err.Error()
			}

		case "openlogicalchannel":
			var channel byte
			channel, err = options.Channel.OpenLogicalChannel(pcRcv.Body)
			pcSnd.Body = []byte{channel}
			if err != nil {
				pcSnd.Err = err.Error()
			}

		case "closelogicalchannel":
			err = options.Channel.CloseLogicalChannel(pcRcv.Body[0])
			if err != nil {
				pcSnd.Err = err.Error()
			}

		case "transmit":
			pcSnd.Body, err = options.Channel.Transmit(pcRcv.Body)
			if err != nil {
				fmt.Printf("Error on transmit: %s\n", err)
				pcSnd.Err = err.Error()
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
