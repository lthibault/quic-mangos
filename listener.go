package quic

import (
	"net/url"

	"github.com/go-mangos/mangos"
	"github.com/pkg/errors"
)

type listener struct {
	cq   chan struct{}
	opt  *options
	u    *url.URL
	sock mangos.Socket
}

func (l *listener) Listen() error {
	return nil
}

func (l *listener) Accept() (mangos.Pipe, error) {
	return nil, errors.New("LISTENER::ACCEPT NOT IMPLEMENTED")
}

func (l listener) Close() error {
	return errors.New("LISTENER::CLOSE NOT IMPLEMENTED")
}

func (l listener) SetOption(name string, value interface{}) error {
	return l.opt.set(name, value)
}

func (l listener) GetOption(name string) (interface{}, error) {
	return l.opt.get(name)
}

func (l listener) Address() string { return l.u.String() }
