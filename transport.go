package quic

import (
	"net/url"

	"github.com/go-mangos/mangos"
	"github.com/pkg/errors"
)

type options map[string]interface{}

// GetOption retrieves an option value.
func (o options) get(name string) (interface{}, error) {
	if o == nil {
		return nil, mangos.ErrBadOption
	}
	v, ok := o[name]
	if !ok {
		return nil, mangos.ErrBadOption
	}
	return v, nil
}

// SetOption sets an option.  We have none, so just ErrBadOption.
func (o options) set(name string, val interface{}) error {
	return mangos.ErrBadOption
}

type quicTrans struct{}

func (quicTrans) Scheme() string { return "quic" }

func (quicTrans) NewDialer(addr string, sock mangos.Socket) (mangos.PipeDialer, error) {
	return nil, errors.New("NEWDIALER NOT IMPLEMENTED")
}

func (quicTrans) NewListener(addr string, sock mangos.Socket) (mangos.PipeListener, error) {
	var err error
	l := new(listener)
	l.sock = sock // needed to create pipes

	if l.url, err = url.ParseRequestURI(addr); err != nil {
		return nil, errors.Wrap(err, "URI parse")
	}

	return l, nil
}

// NewTransport allocates a new quic:// transport.
func NewTransport() mangos.Transport {
	return quicTrans{}
}
