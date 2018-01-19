package quic

import (
	"log"
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

type listener struct {
}

func (listener) Listen() error {
	return errors.New("NOT IMPLEMENTED")
}

func (listener) Accept() (mangos.Pipe, error) {
	return nil, errors.New("NOT IMPLEMENTED")
}

func (listener) Close() error {
	return errors.New("NOT IMPLEMENTED")
}

func (listener) SetOption(name string, value interface{}) error {
	return errors.New("NOT IMPLEMENTED")
}

func (listener) GetOption(name string) (interface{}, error) {
	return nil, errors.New("NOT IMPLEMENTED")
}

func (listener) Address() string {
	return "NOT IMPLEMENTED"
}

type quicTrans struct{}

func (quicTrans) Scheme() string { return "quic" }

func (quicTrans) NewDialer(addr string, sock mangos.Socket) (mangos.PipeDialer, error) {
	return nil, errors.New("NOT IMPLEMENTED")
}

func (quicTrans) NewListener(addr string, sock mangos.Socket) (mangos.PipeListener, error) {
	u, err := url.ParseRequestURI(addr)
	if err != nil {
		return nil, errors.Wrap(err, "URI parse")
	}

	log.Fatal(u.String())
	return nil, errors.New("NOT IMPLEMENTED")
}

// NewTransport allocates a new quic:// transport.
func NewTransport() mangos.Transport {
	return quicTrans{}
}
