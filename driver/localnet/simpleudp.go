package localnet

import (
	"encoding/gob"
	"errors"
	"fmt"
	"net"

	"github.com/damonto/euicc-go/apdu"
	"github.com/damonto/euicc-go/driver/qmi/core"
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

	_, err = zzz(c.conn, "connect", nil)
	return err
}

func (c *NetContext) Disconnect() error {
	var err error
	if c.conn != nil {
		_, err = zzz(c.conn, "disconnect", nil)
		c.conn.Close()
		c.conn = nil
	}
	return err
}

func (c *NetContext) Transmit(command []byte) ([]byte, error) {
	return zzz(c.conn, "transmit", command)
}

func (c *NetContext) OpenLogicalChannel(AID []byte) (byte, error) {
	gob.Register(core.QMIError(0))
	bb, er := zzz(c.conn, "openlogicalchannel", AID)
	return bb[0], er
}

func (c *NetContext) CloseLogicalChannel(channel byte) error {
	_, er := zzz(c.conn, "closelogicalchannel", []byte{channel})
	return er
}

func zzz(cn *net.UDPConn, cm string, bd []byte) (by []byte, er error) {

	pcSnd := PacketCmd{
		Cmd:  cm,
		Body: bd,
		Err:  "",
	}

	byteToTransmit, err1 := pcSnd.Encode()
	if err1 != nil {
		return nil, fmt.Errorf("error encoding message %X %w", cm, err1)
	}

	_, err2 := cn.Write(byteToTransmit)
	if err2 != nil {
		return nil, fmt.Errorf("error sending message %X %w", cm, err2)
	}

	buffer := make([]byte, 512)
	n, _, err3 := cn.ReadFromUDP(buffer)
	if err3 != nil {
		return nil, fmt.Errorf("error receiving response %X %w", buffer, err3)
	}

	pcRcv := PacketCmd{}
	err4 := pcRcv.Decode(buffer[:n])
	if err4 != nil {
		return nil, fmt.Errorf("error decoding response %X %w", buffer[:n], err4)
	}

	if pcRcv.Err != "" {
		return pcRcv.Body, errors.New(pcRcv.Err)
	}

	return pcRcv.Body, nil
}
