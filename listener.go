package quic

import (
	"net"
	"net/url"

	"github.com/SentimensRG/ctx"
	"github.com/go-mangos/mangos"
	"github.com/pkg/errors"
)

type listener struct {
	*url.URL

	d      ctx.Doner
	cancel func()

	ch   chan net.Conn
	opt  *options
	sock mangos.Socket
}

func (l *listener) Listen() (err error) {
	var r router
	if r, err = transport.Bind(l.URL, l.ch, l.opt); err != nil {
		err = errors.Wrap(err, "transport")
	} else if err = r.RegisterPath(l.URL.Path, l.ch); err != nil {
		l.cancel()
	}

	return
}

func (l listener) Accept() (mangos.Pipe, error) {
	conn, ok := <-l.ch
	if !ok {
		return nil, errors.New("transport closed")
	}
	return mangos.NewConnPipe(conn, l.sock)

}

func (l listener) Close() (err error) {
	l.cancel()
	return
}

func (l listener) SetOption(name string, value interface{}) error {
	return l.opt.set(name, value)
}

func (l listener) GetOption(name string) (interface{}, error) {
	return l.opt.get(name)
}

func (l listener) Address() string { return l.URL.String() }
