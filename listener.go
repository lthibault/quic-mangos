package quic

import (
	"io"
	"net/url"

	"github.com/SentimensRG/ctx"
	"github.com/go-mangos/mangos"
	"github.com/pkg/errors"
)

type listener struct {
	cq   chan struct{}
	ch   chan io.ReadWriteCloser
	opt  *options
	u    *url.URL
	sock mangos.Socket
}

func (l *listener) Listen() (err error) {
	// transparently start an h2server if it doesn't exist
	// assign a chan io.ReadWriteCloser to the correct router
	// incr [ something ] to track that a mangos.Socket is using the server

	var r *router
	if r, err = transport.MaybeInit(l.u.Host, l.opt); err != nil {
		err = errors.Wrap(err, "transport")
	} else if err = r.RegisterPath(l.u.Path, l.ch); err != nil {
		err = errors.Wrap(err, "register path")
	} else {
		ctx.Defer(ctx.Lift(l.cq), r.Decr)
	}

	return
}

func (l *listener) Accept() (mangos.Pipe, error) {
	return nil, errors.New("LISTENER::ACCEPT NOT IMPLEMENTED")
}

func (l listener) Close() (err error) {
	close(l.cq)
	return
}

func (l listener) SetOption(name string, value interface{}) error {
	return l.opt.set(name, value)
}

func (l listener) GetOption(name string) (interface{}, error) {
	return l.opt.get(name)
}

func (l listener) Address() string { return l.u.String() }
