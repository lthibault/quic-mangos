package quic

import (
	"net/url"

	"github.com/go-mangos/mangos"
	quic "github.com/lucas-clemente/quic-go"
	"github.com/pkg/errors"
)

type listener struct {
	*url.URL

	quic.Listener

	opt  *options
	sock mangos.Socket
}

func (l *listener) Listen() error {
	tc, qc := getQUICCfg(l.opt)

	var err error
	if l.Listener, err = quic.ListenAddr(l.Host, tc, qc); err != nil {
		return errors.Wrap(err, "listen quic")
	}

	return nil
}

func (l listener) Accept() (mangos.Pipe, error) {
	sess, err := l.Listener.Accept()
	if err != nil {
		return nil, errors.Wrap(err, "accept session")
	}

	stream, err := sess.AcceptStream()
	if err != nil {
		return nil, errors.Wrap(err, "accept stream")
	}

	return mangos.NewConnPipe(&conn{Stream: stream, Session: sess}, l.sock)
}

func (l listener) SetOption(name string, value interface{}) error {
	return l.opt.set(name, value)
}

func (l listener) GetOption(name string) (interface{}, error) {
	return l.opt.get(name)
}

func (l listener) Address() string { return l.URL.String() }
