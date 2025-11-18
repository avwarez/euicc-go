package localnet

import (
	"bytes"
	"encoding/gob"
	"fmt"
)

type Cmd string

const (
	CmdConnect      Cmd = "connect"
	CmdDisconnect   Cmd = "disconnect"
	CmdOpenLogical  Cmd = "openlogicalchannel"
	CmdCloseLogical Cmd = "closelogicalchannel"
	CmdTransmit     Cmd = "transmit"
	CmdResponse     Cmd = "response"
)

type IPacketCmd interface {
	GetCmd() Cmd
	GetBody() []byte
	GetErr() string
}

type IPacketConnect interface {
	IPacketCmd
	GetDevice() string
	GetProto() string
	GetSlot() uint8
}

type PacketCmd struct {
	Cmd  Cmd
	Body []byte
	Err  string
}

type PacketConnect struct {
	PacketCmd
	Device string
	Proto  string
	Slot   uint8
}

func Decode(byteArray []byte) (p IPacketCmd, e error) {
	gob.Register(&PacketCmd{})
	gob.Register(&PacketConnect{})

	buf := bytes.NewBuffer(byteArray)
	dec := gob.NewDecoder(buf)
	e = dec.Decode(&p)
	return p, e
}

func Encode(p IPacketCmd) (byteArray []byte, err error) {
	gob.Register(&PacketCmd{})
	gob.Register(&PacketConnect{})

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err = enc.Encode(&p)
	return buf.Bytes(), err
}

func (p PacketCmd) GetCmd() Cmd {
	return p.Cmd
}

func (p PacketCmd) GetBody() []byte {
	return p.Body
}

func (p PacketCmd) GetErr() string {
	return p.Err
}

func (p PacketConnect) GetDevice() string {
	return p.Device
}

func (p PacketConnect) GetProto() string {
	return p.Proto
}

func (p PacketConnect) GetSlot() uint8 {
	return p.Slot
}

func (p PacketCmd) String() string {
	return fmt.Sprintf("Cmd: %s, Body(hex): %X, Err: %s", p.GetCmd(), p.GetBody(), p.GetErr())
}

func (p PacketConnect) String() string {
	return fmt.Sprintf("%s, Device: %s, Proto: %s, Slot: %d", p.PacketCmd, p.GetDevice(), p.GetProto(), p.GetSlot())
}

func NewPacketCmd(cmd Cmd) IPacketCmd {
	return PacketCmd{cmd, nil, ""}
}

func NewPacketCmdErr(cmd Cmd, err string) IPacketCmd {
	return PacketCmd{cmd, nil, err}
}

func NewPacketBody(cmd Cmd, body []byte) IPacketCmd {
	return PacketCmd{cmd, body, ""}
}

func NewPacketConnect(device string, proto string, slot uint8) IPacketCmd {
	return PacketConnect{PacketCmd{CmdConnect, nil, ""}, device, proto, slot}
}
