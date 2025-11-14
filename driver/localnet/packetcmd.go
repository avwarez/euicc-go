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

type PacketCmd struct {
	Cmd    Cmd
	Body   []byte
	Err    string
	Device string
	Proto  string
	Slot   uint8
}

func (p *PacketCmd) Decode(byteArray []byte) error {
	buf := bytes.NewBuffer(byteArray)
	dec := gob.NewDecoder(buf)
	return dec.Decode(p)
}

func (p PacketCmd) Encode() (byteArray []byte, err error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err = enc.Encode(p)
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

func (p PacketCmd) GetDevice() string {
	return p.Device
}

func (p PacketCmd) GetProto() string {
	return p.Proto
}

func (p PacketCmd) GetSlot() uint8 {
	return p.Slot
}

func (p PacketCmd) String() string {
	return fmt.Sprintf("Cmd: %s, Body(hex): %X, Device: %s, Proto: %s, Slot: %d\n", p.GetCmd(), p.GetBody(), p.GetDevice(), p.GetProto(), p.GetSlot())
}
