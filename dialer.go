package quic

import (
	"fmt"
	"net"

	"github.com/go-mangos/mangos"
	"github.com/pkg/errors"
)

type dialer struct {
	opt        *options
	ip         *net.IPAddr
	port, path string
	sock       mangos.Socket
}

func (d dialer) netloc() string {
	return fmt.Sprintf("%s:%s", d.ip.String(), d.port)
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
