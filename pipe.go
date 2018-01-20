package quic

import (
	"github.com/go-mangos/mangos"
	"github.com/pkg/errors"
)

type pipe struct {
}

func (p pipe) Close() error {
	return errors.New("PIPE::CLOSE NOT IMPLEMENTED")
}

func (pipe) Send(*mangos.Message) error {
	return errors.New("PIPE::SEND NOT IMPLEMENTED")
}

func (pipe) Recv() (*mangos.Message, error) {
	return nil, errors.New("PIPE::RECV NOT IMPLEMENTED")
}

func (pipe) LocalProtocol() uint16  { panic("PIPE::LOCALPROTOCOL NOT IMPLEMENTED") }
func (pipe) RemoteProtocol() uint16 { panic("PIPE::LOCALPROTOCOL NOT IMPLEMENTED") }

func (p pipe) IsOpen() bool {
	panic(errors.New("PIPE:ISOPEN NOT IMPLEMENTED"))
}

func (pipe) GetProp(prop string) (interface{}, error) {
	return nil, errors.New("PIPE:GETPROP NOT IMPLEMENTED")
}
