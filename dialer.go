package quic

import (
	"github.com/go-mangos/mangos"
	"github.com/pkg/errors"
)

type dialer struct{}

func (dialer) Dial() (mangos.Pipe, error) {
	return nil, errors.New("DIAL NOT IMPLEMENTED")
}

func (dialer) SetOption(name string, value interface{}) error {
	return errors.New("DIALER::SETOPT NOT IMPLEMENTED")
}

func (dialer) GetOption(name string) (interface{}, error) {
	return nil, errors.New("DIALER::GETOPT NOT IMPLEMENTED")
}
