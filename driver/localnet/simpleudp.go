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
	device     string
	proto      string
	slot       uint8
}

func NewUDP(serverAddr string, device string, proto string, slot uint8) (apdu.SmartCardChannel, error) {
	rAddr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		return nil, fmt.Errorf("error resolving address: %s %w", serverAddr, err)
	}

	netctx := &NetContext{serverAddr: serverAddr, rAddr: rAddr, device: device, proto: proto, slot: slot}
	return netctx, nil
}

func (c *NetContext) Connect() error {
	conn, err := net.DialUDP("udp", nil, c.rAddr)
	if err != nil {
		return fmt.Errorf("error establishing connection with %s %w", c.rAddr, err)
	}
	c.conn = conn

	_, err = zzz(c.conn, "connect", nil, c.device, c.proto, c.slot)
	return err
}

func (c *NetContext) Disconnect() error {
	var err error
	if c.conn != nil {
		_, err = zzz(c.conn, "disconnect", nil, c.device, c.proto, c.slot)
		c.conn.Close()
		c.conn = nil
	}
	return err
}

func (c *NetContext) Transmit(command []byte) ([]byte, error) {
	return zzz(c.conn, "transmit", command, c.device, c.proto, c.slot)
}

func (c *NetContext) OpenLogicalChannel(AID []byte) (byte, error) {
	gob.Register(core.QMIError(0))
	bb, er := zzz(c.conn, "openlogicalchannel", AID, c.device, c.proto, c.slot)
	return bb[0], er
}

func (c *NetContext) CloseLogicalChannel(channel byte) error {
	_, er := zzz(c.conn, "closelogicalchannel", []byte{channel}, c.device, c.proto, c.slot)
	return er
}

func zzz(cn *net.UDPConn, cm string, bd []byte, de string, pr string, sl uint8) (by []byte, er error) {

	pcSnd := PacketCmd{
		Cmd:    cm,
		Body:   bd,
		Err:    "",
		Device: de,
		Proto:  pr,
		Slot:   sl,
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
