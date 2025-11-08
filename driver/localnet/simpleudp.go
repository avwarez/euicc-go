package localnet

import (
	"fmt"
	"net"

	"github.com/damonto/euicc-go/apdu"
)

type NetContext struct {
	serverAddr string
	rAddr      *net.UDPAddr
	conn       *net.UDPConn
}

func NewUDP(serverAddr string) (apdu.SmartCardChannel, error) {
	rAddr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		return nil, fmt.Errorf("error resolving address: %s %w", serverAddr, err)
	}

	ccid := &NetContext{serverAddr: serverAddr, rAddr: rAddr}
	return ccid, nil
}

func (c *NetContext) Connect() error {
	conn, err := net.DialUDP("udp", nil, c.rAddr)
	if err != nil {
		return fmt.Errorf("error establishing connection with %s %w", c.rAddr, err)
	}
	c.conn = conn
	return nil
}

func (c *NetContext) Disconnect() error {
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return fmt.Errorf("error closing connection %w", err)
	}
	return nil
}

func (c *NetContext) Transmit(command []byte) ([]byte, error) {

	pcSnd := PacketCmd{
		Cmd:  "transmit",
		Body: command,
		Err:  nil,
	}

	byteToTransmit, err1 := pcSnd.Encode()
	if err1 != nil {
		return nil, fmt.Errorf("error encoding message %X %w", command, err1)
	}

	_, err2 := c.conn.Write(byteToTransmit)
	if err2 != nil {
		return nil, fmt.Errorf("error sending message %X %w", command, err2)
	}

	buffer := make([]byte, 512)
	n, _, err3 := c.conn.ReadFromUDP(buffer)
	if err3 != nil {
		return nil, fmt.Errorf("error receiving response %X %w", buffer, err3)
	}

	pcRcv := PacketCmd{}
	err4 := pcRcv.Decode(buffer[:n])

	return pcRcv.Body, err4
}

func (c *NetContext) OpenLogicalChannel(AID []byte) (byte, error) {
	return 0, nil
}

func (c *NetContext) CloseLogicalChannel(channel byte) error {
	return nil
}
