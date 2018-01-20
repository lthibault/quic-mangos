package quic

import (
	"net/url"

	"github.com/go-mangos/mangos"
	"github.com/pkg/errors"
)

type dialer struct {
	opt  *options
	u    *url.URL
	sock mangos.Socket
}

func (d dialer) Dial() (mangos.Pipe, error) {
	return nil, errors.New("DIALER::DIAL NOT IMPLEMENTED")
}

func (d dialer) SetOption(name string, value interface{}) error {
	return d.opt.set(name, value)
}

func (d dialer) GetOption(name string) (interface{}, error) {
	return d.opt.get(name)
}
