package main

import (
	"flag"
	"fmt"
	"net"

	"log/slog"

	"github.com/damonto/euicc-go/apdu"
	"github.com/damonto/euicc-go/driver"
	"github.com/damonto/euicc-go/driver/localnet"
	"github.com/damonto/euicc-go/driver/qmi"
	"github.com/damonto/euicc-go/lpa"
	sgp22 "github.com/damonto/euicc-go/v2"
)

/*type ReqHead struct {
	CLA byte
	INS byte
	P1  byte
	P2  byte
	LEN byte
}*/

var (
	options     lpa.Options
	APDU        sgp22.Transmitter
	transmitter driver.Transmitter
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

	var err1 error
	if transmitter, err1 = driver.NewTransmitter(options.Logger, options.Channel, options.AID, options.MSS); err1 != nil {
		panic("Error during creation of new transmitter")
	}
	APDU = transmitter
	defer transmitter.Close()

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
			fmt.Println("Error reading from socket:", err)
			continue
		}

		fmt.Printf("Message from %s: %s\n", remoteAddr, fmt.Sprintf("%X", buffer[:n]))

		if n < 5 {
			fmt.Println("Recevied packet too short, skipping")
			continue
		}

		/*req := &ReqHead{
			CLA: buffer[0],
			INS: buffer[1],
			P1:  buffer[2],
			P2:  buffer[3],
			LEN: buffer[4],
		}*/

		fmt.Println("Decodifica messaggio")
		pcRcv := localnet.PacketCmd{}
		err = pcRcv.Decode(buffer[:n])
		if err != nil {
			fmt.Println("Errore nella decodifica del messaggio. Chiusura del server...")
			break
		}

		fmt.Printf("\n=== DEBUG PACKET ===\n")
		fmt.Printf("Cmd: %s\n", pcRcv.Cmd)
		fmt.Printf("Body len: %d\n", len(pcRcv.Body))
		fmt.Printf("Body hex: %X\n", pcRcv.Body)
		if len(pcRcv.Body) > 0 {
			fmt.Printf("Body[0]: 0x%02X\n", pcRcv.Body[0])
		}
		if len(pcRcv.Body) > 4 {
			fmt.Printf("Body[4] (presumed len): 0x%02X (%d)\n", pcRcv.Body[4], pcRcv.Body[4])
		}
		fmt.Printf("==================\n\n")

		fmt.Printf("Analisi comando %s \n", pcRcv.Cmd)
		switch pcRcv.Cmd {
		case "exit":
			fmt.Println("Comando 'exit' ricevuto. Chiusura del server...")
			break outer

		case "transmit":
			fmt.Printf("DEBUG: Body ricevuto (len=%d): %X\n", len(pcRcv.Body), pcRcv.Body)

			if len(pcRcv.Body) == 0 {
				fmt.Println("ERROR: Body vuoto")
				pcSnd := localnet.PacketCmd{
					Cmd:  "response",
					Body: nil,
					Err:  fmt.Errorf("empty body"),
				}
				byteArrayResponse, _ := pcSnd.Encode()
				conn.WriteToUDP(byteArrayResponse, remoteAddr)
				continue
			}

			var rawCommand []byte

			// Controlla se c'è un header con lunghezza
			if len(pcRcv.Body) >= 5 {
				lenByte := pcRcv.Body[4]

				if lenByte == 0xFE || lenByte == 0xFF {
					// Lunghezza estesa: usa tutto il body dal byte 5 in poi
					fmt.Printf("Lunghezza estesa (0x%02X), uso tutto il body dal byte 5\n", lenByte)
					rawCommand = pcRcv.Body[5:]
				} else {
					// Lunghezza normale
					dataLen := int(lenByte)
					if len(pcRcv.Body) >= 5+dataLen {
						rawCommand = pcRcv.Body[5 : 5+dataLen]
					} else {
						// Usa quello che c'è disponibile
						fmt.Printf("WARNING: Lunghezza dichiarata (%d) > disponibile, uso tutto\n", dataLen)
						rawCommand = pcRcv.Body[5:]
					}
				}
			} else {
				// Body troppo corto, usa tutto
				fmt.Println("WARNING: Body troppo corto per header, uso tutto")
				rawCommand = pcRcv.Body
			}

			fmt.Printf("Transmitting raw (%d bytes): %s\n", len(rawCommand), fmt.Sprintf("%X", rawCommand))
			byteArrayResponse, err := transmitter.TransmitRaw(rawCommand)
			if err != nil {
				fmt.Println("Error using lib raw transport:", err)
				// In caso di errore, crea una risposta di errore appropriata
				// Status word 6F00 = errore generico
				byteArrayResponse = []byte{0x6F, 0x00}
			} else {
				// pezza brutta brutta
				byteArrayResponse = append(byteArrayResponse, 0x90, 0x00)
				// pezza brutta brutta
			}

			// Controlla che la risposta abbia almeno 2 byte prima di processarla
			if len(byteArrayResponse) >= 2 {
				resp := apdu.Response(byteArrayResponse)
				fmt.Printf("SW:      0x%04X\n", resp.SW())
				fmt.Printf("SW1:     0x%02X\n", resp.SW1())
				fmt.Printf("SW2:     0x%02X\n", resp.SW2())
				fmt.Printf("OK?      %v\n", resp.OK())
				fmt.Printf("HasMore? %v\n", resp.HasMore())
			} else {
				fmt.Printf("WARNING: Risposta troppo corta (%d bytes)\n", len(byteArrayResponse))
			}

			fmt.Printf("Receiving raw: %s\n", fmt.Sprintf("%X", byteArrayResponse))

			pcSnd := localnet.PacketCmd{
				Cmd:  "response",
				Body: byteArrayResponse,
				Err:  err,
			}
			byteArrayResponse, err = pcSnd.Encode()
			if err != nil {
				fmt.Println("Error encoding response:", err)
				break outer
			}

			_, err = conn.WriteToUDP(byteArrayResponse, remoteAddr)
			if err != nil {
				fmt.Println("Error sending response to the client:", err)
			}
		default:
			fmt.Println("Comando sconosciuto ricevuto. Chiusura del server...")
			break outer
		}

		// 0x65=e 0x78=x 0x69=i 0x74=t => exit
		/*if req.CLA == 0x65 && req.INS == 0x78 && req.P1 == 0x69 && req.P2 == 0x74 {
			fmt.Println("Comando 'exit' ricevuto. Chiusura del server...")
			break
		}*/

		/*if req.CLA == 0x80 && req.INS == 0xE2 && req.LEN > 0 {

			fmt.Printf("Transmitting raw: %s\n", fmt.Sprintf("%X", buffer[5:5+req.LEN]))
			byteArrayResponse, err := transmitter.TransmitRaw(buffer[5 : 5+req.LEN])
			if err != nil {
				fmt.Println("Error using lib raw transport:", err)
			}
			// pezza brutta brutta
			byteArrayResponse = append(byteArrayResponse, 0x90, 0x00)
			// pezza brutta brutta
			resp := apdu.Response(byteArrayResponse)

			fmt.Printf("SW:   0x%04X\n", resp.SW())
			fmt.Printf("SW1:  0x%02X\n", resp.SW1())
			fmt.Printf("SW2:  0x%02X\n", resp.SW2())
			fmt.Printf("OK?   %v\n", resp.OK())
			fmt.Printf("HasMore?   %v\n", resp.HasMore())

			fmt.Printf("Receiving raw: %s\n", fmt.Sprintf("%X", byteArrayResponse))

			_, err = conn.WriteToUDP(byteArrayResponse, remoteAddr)
			if err != nil {
				fmt.Println("Error sending response to the client:", err)
			}
		}*/

	}
}
