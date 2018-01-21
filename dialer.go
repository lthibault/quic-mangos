package quic

import (
	"net/url"

	"github.com/go-mangos/mangos"
	"github.com/pkg/errors"
)

type dialer struct {
	*url.URL
	opt  *options
	sock mangos.Socket
}

func (d dialer) Dial() (mangos.Pipe, error) {
	rwc, err := transport.Connect(d.URL)
	if err != nil {
		return nil, errors.Wrap(err, "dial")
	}

	return &pipe{ReadWriteCloser: rwc}, nil
}

func (d dialer) SetOption(name string, value interface{}) error {
	return d.opt.set(name, value)
}

func (d dialer) GetOption(name string) (interface{}, error) {
	return d.opt.get(name)
}
