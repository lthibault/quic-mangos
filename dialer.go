package quic

import (
	"net/url"

	"github.com/go-mangos/mangos"
	quic "github.com/lucas-clemente/quic-go"
	"github.com/pkg/errors"
)

type dialer struct {
	*url.URL
	opt  *options
	sock mangos.Socket
}

func (d dialer) Dial() (mangos.Pipe, error) {
	tc, qc := getQUICCfg(d.opt)
	sess, err := quic.DialAddr(d.Host, tc, qc)
	if err != nil {
		return nil, errors.Wrap(err, "dial quic")
	}

	stream, err := sess.OpenStreamSync()
	if err != nil {
		return nil, errors.Wrap(err, "open stream")
	}

	return mangos.NewConnPipe(&conn{Stream: stream, Session: sess}, d.sock)
}

func (d dialer) SetOption(name string, value interface{}) error {
	return d.opt.set(name, value)
}

func (d dialer) GetOption(name string) (interface{}, error) {
	return d.opt.get(name)
}
