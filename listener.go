package quic

import (
	"net/url"

	"github.com/go-mangos/mangos"
	"github.com/pkg/errors"
)

type listener struct {
	url  *url.URL
	sock mangos.Socket
}

func (listener) Listen() error {

	// Do we have an open quic session matching the URL?
	// How do we match a stream to a path?

	return errors.New("LISTEN NOT IMPLEMENTED")
}

func (listener) Accept() (mangos.Pipe, error) {
	return nil, errors.New("ACCEPT NOT IMPLEMENTED")
}

func (listener) Close() error {
	return errors.New("CLOSE NOT IMPLEMENTED")
}

func (listener) SetOption(name string, value interface{}) error {
	return errors.New("LISTENER::SETOPT NOT IMPLEMENTED")
}

func (listener) GetOption(name string) (interface{}, error) {
	return nil, errors.New("LISTENER::GETOPT NOT IMPLEMENTED")
}

func (listener) Address() string {
	return "ADDRESS NOT IMPLEMENTED"
}
