package quic

import (
	"net/url"
	"path/filepath"

	"github.com/go-mangos/mangos"
	"github.com/pkg/errors"
)

const (
	// OptionTLSConfig maps to a *tls.Config value
	OptionTLSConfig = "QUIC-TLS-CONFIG"
	// OptionQUICConfig maps to a *quic.Config value
	OptionQUICConfig = "QUIC-UDP-CONFIG"
	// OptionAcceptTimeout limits the amount of time we wait to accept a connection
)

type transport struct{}

func (transport) Scheme() string { return "quic" }

func (t transport) NewDialer(addr string, sock mangos.Socket) (mangos.PipeDialer, error) {
	u, err := url.ParseRequestURI(addr)
	if err != nil {
		return nil, errors.Wrap(err, "url parse")
	}

	u.Path = filepath.Clean(u.Path)

	return &dialer{
		netloc:  netloc{u},
		Socket:  sock,
		dialMux: newDialMux(sock, mux),
	}, nil
}

func (t transport) NewListener(addr string, sock mangos.Socket) (mangos.PipeListener, error) {
	u, err := url.ParseRequestURI(addr)
	if err != nil {
		return nil, errors.Wrap(err, "url parse")
	}

	u.Path = filepath.Clean(u.Path)

	return &listener{
		netloc:    netloc{u},
		Socket:    sock,
		listenMux: newListenMux(mux),
	}, nil
}

// NewTransport allocates a new quic:// transport.
func NewTransport() mangos.Transport { return transport{} }
