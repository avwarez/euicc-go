package localnet

import (
	"bytes"
	"encoding/gob"
)

type PacketCmd struct {
	Cmd  string
	Body []byte
	Err  string
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
