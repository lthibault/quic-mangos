package quic

import (
	"crypto/tls"
	"fmt"
	"net"

	"github.com/SentimensRG/ctx"
	"github.com/go-mangos/mangos"
	quic "github.com/lucas-clemente/quic-go"
	"github.com/pkg/errors"
)

type dialMux struct {
	mux  dialMuxer
	sess *refcntSession
	sock mangos.Socket
}

func newDialMux(sock mangos.Socket, m multiplexer) *dialMux {
	return &dialMux{sock: sock, mux: m}
}

func (dm *dialMux) LoadSession(n netlocator, tc *tls.Config, qc *quic.Config) error {
	lock.Lock()
	defer lock.Unlock()

	var ok bool
	if dm.sess, ok = dm.mux.GetSession(n); !ok {

		// We don't have a session for this [ ??? ] yet, so create it
		qs, err := quic.DialAddr(n.Netloc(), tc, qc)
		if err != nil {
			return err
		}

		// Init refcnt to track the Session's usage and clean up when we're done
		dm.sess = newRefCntSession(qs, dm.mux)
		dm.mux.AddSession(qs.RemoteAddr(), dm.sess) // don't add until it's incremented
	}

	dm.sess.Incr()
	return nil
}

func (dm dialMux) Dial(path string) (net.Conn, error) {
	stream, err := dm.sess.OpenStreamSync()
	if err != nil {
		return nil, errors.Wrap(err, "open stream")
	}

	// There's no Close method for mangos.PipeDialer, so we need to decr
	// the ref counter when the stream closes.
	ctx.Defer(stream.Context(), func() { _ = dm.sess.DecrAndClose() })

	// this is where we do the path negotiation
	var n dialNegotiator = newNegotiator(stream)

	if err = n.WriteHeaders(fmt.Sprintf("%s\n", path)); err != nil {
		_ = stream.Close()
		return nil, errors.Wrap(err, "write headers")
	}
	if err = n.Ack(); err != nil {
		return nil, errors.Wrap(err, "ack")
	}

	return &conn{Stream: stream, Session: dm.sess}, nil
}

type dialer struct {
	netloc
	*dialMux
	mangos.Socket
}

func (d dialer) Dial() (mangos.Pipe, error) {
	tc, qc := getQUICCfg(d.Socket)

	if err := d.LoadSession(d.netloc, tc, qc); err != nil {
		return nil, errors.Wrap(err, "dial quic")
	}

	conn, err := d.dialMux.Dial(d.Path)
	if err != nil {
		return nil, errors.Wrap(err, "dial path")
	}

	return mangos.NewConnPipe(conn, d.sock)
}
