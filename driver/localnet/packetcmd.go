package localnet

import (
	"bytes"
	"encoding/gob"
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
