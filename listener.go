package quic

import (
	"io"
	"net/url"

	"github.com/SentimensRG/ctx"
	"github.com/go-mangos/mangos"
	"github.com/pkg/errors"
)

type listener struct {
	*url.URL
	cq   chan struct{}
	ch   chan io.ReadWriteCloser
	opt  *options
	sock mangos.Socket
}

func (l *listener) Listen() (err error) {
	var r *router
	if r, err = transport.Bind(l.URL, l.ch, l.opt); err != nil {
		err = errors.Wrap(err, "transport")
	} else {
		r.Incr()
		ctx.Defer(ctx.Lift(l.cq), r.Decr)

		if err = r.RegisterPath(l.URL.Path, l.ch); err != nil {
			err = errors.Wrap(err, "register path")
		} else {
			ctx.Defer(ctx.Lift(l.cq), r.Cleanup(l.URL.Path))
		}
	}

	return
}

func (l listener) Accept() (mangos.Pipe, error) {
	rwc, ok := <-l.ch
	if !ok {
		return nil, errors.New("transport closed")
	}
	return &pipe{
		ReadWriteCloser: rwc,
		proto:           l.sock.GetProtocol(),
	}, nil

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

func (l listener) Address() string { return l.URL.String() }
