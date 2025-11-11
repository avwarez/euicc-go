package main

import (
	"flag"
	"fmt"
	"net"

	"log/slog"

	"github.com/damonto/euicc-go/driver/at"
	"github.com/damonto/euicc-go/driver/localnet"
	"github.com/damonto/euicc-go/driver/mbim"
	"github.com/damonto/euicc-go/driver/qmi"
	"github.com/damonto/euicc-go/lpa"
)

var (
	options lpa.Options
	device  string
	proto   string
	slot    uint8
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	deviceFlag := flag.String("device", "/dev/cdc-wdm0", "Device, if required")
	protoFlag := flag.String("proto", "qrtr", "Protocol: qmi, qrtr, mbim, at")
	slotFlag := flag.Int("slot", 2, "SIM slot where eSIM is installed (for QMI/QRTR/MBIM)")
	//aidFlag := flag.ByteArray("aid", null, "AID opts value")
	mssFlag := flag.Int("mss", 0, "MSS opts value")
	bindAddrFlag := flag.String("bindAddr", "0.0.0.0", "Binding address")
	bindPortFlag := flag.Int("bindPort", 8080, "Binding port")

	flag.Parse()

	device = *deviceFlag
	proto = *protoFlag
	slot = uint8(*slotFlag)
	//options.AID = *aidFlag
	options.MSS = *mssFlag
	options.AdminProtocolVersion = "2"

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

		fmt.Printf("DEBUG Cmd: %s Body hex: %X\n", pcRcv.Cmd, pcRcv.Body)

		pcSnd := localnet.PacketCmd{
			Cmd:  "response",
			Body: nil,
			Err:  "",
		}

		switch pcRcv.Cmd {

		case "connect":

			if options.Channel != nil {
				err = fmt.Errorf("error: channel already open, retry later")
			} else {
				switch proto {
				case "at":
					options.Channel, err = at.New(device)
				/*case "ccid":
				options.Channel, err = ccid.New() */
				case "mbim":
					options.Channel, err = mbim.New(device, slot)
				case "qmi":
					options.Channel, err = qmi.New(device, slot)
				case "qrtr":
					options.Channel, err = qmi.NewQRTR(slot)
				default:
					err = fmt.Errorf("error: no handler for the specified protocol %s", proto)
				}
			}

			if err != nil {
				pcSnd.Err = err.Error()
			} else {

				err = options.Channel.Connect()
				if err != nil {
					pcSnd.Err = err.Error()
					options.Channel = nil
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
