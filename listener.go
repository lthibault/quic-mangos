package quic

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"sync/atomic"

	"github.com/go-mangos/mangos"
	quic "github.com/lucas-clemente/quic-go"
	"github.com/pkg/errors"
)

type muxListener interface {
	LoadListener(netlocator, *tls.Config, *quic.Config) error
	Accept(string) (net.Conn, error)
	Close() error
}

type netlocator interface {
	Hostname() string
	Port() string
}

type refcntListener struct {
	gc     func()
	refcnt int32
	quic.Listener
}

func (r *refcntListener) Incr() *refcntListener {
	atomic.AddInt32(&r.refcnt, 1)
	return r
}

func (r *refcntListener) DecrAndClose() (err error) {
	if i := atomic.AddInt32(&r.refcnt, -1); i == 0 {
		err = r.Close()
		r.gc()
	} else if i < 0 {
		err = errors.New("close called on previously-closed Listener")
	}
	return
}

// listenMux implements muxListener
type listenMux struct {
	mux multiplexer
	l   *refcntListener
}

func newListenMux(m multiplexer) *listenMux {
	return &listenMux{mux: m}
}

func listenQUIC(n netlocator, tc *tls.Config, qc *quic.Config) (quic.Listener, error) {
	netloc := fmt.Sprintf("%s:%s", n.Hostname(), n.Port())
	return quic.ListenAddr(netloc, tc, qc)
}

func (lm *listenMux) LoadListener(n netlocator, tc *tls.Config, qc *quic.Config) error {
	lm.mux.Lock()
	defer lm.mux.Unlock()

	var ok bool
	if lm.l, ok = lm.mux.GetListener(n); !ok {

		// We don't have a listener for this netloc yet, so create it.
		ql, err := listenQUIC(n, tc, qc)
		if err != nil {
			return err
		}

		// Init refcnt to track the Listener's usage and clean up when we're done
		lm.l = &refcntListener{
			Listener: ql,
			refcnt:   1,
			gc:       func() { lm.mux.DelListener(n) },
		}
		lm.mux.SetListener(n, lm.l)
	}

	return nil
}

func (lm listenMux) Accept(path string) (net.Conn, error) {

	// TODO:  get a session
	// TODO:  get a stream
	// TODO:  return &conn{Session: sess, Stream: stream}

	// // FROM LISTENER
	// sess, err := l.muxListener.Accept()
	// if err != nil {
	// 	return nil, errors.Wrap(err, "accept session")
	// }

	// stream, err := sess.AcceptStream()
	// if err != nil {
	// 	return nil, errors.Wrap(err, "accept stream")
	// }

	// &conn{Stream: stream, Session: sess}

	return nil, errors.New("ACCEPT NOT IMPLEMENTED")
}

func (lm listenMux) Close() error {
	return lm.l.DecrAndClose()
}

type listener struct {
	*url.URL

	muxListener

	opt  *options
	sock mangos.Socket
}

func (l *listener) Listen() error {
	tc, qc := getQUICCfg(l.opt)
	return errors.Wrap(l.LoadListener(l.URL, tc, qc), "listen quic")
}

func (l listener) Accept() (mangos.Pipe, error) {
	conn, err := l.muxListener.Accept(l.Path)
	if err != nil {
		return nil, errors.Wrap(err, "mux accept")
	}

	return mangos.NewConnPipe(conn, l.sock)
}

func (l listener) SetOption(name string, value interface{}) error {
	return l.opt.set(name, value)
}

func (l listener) GetOption(name string) (interface{}, error) {
	return l.opt.get(name)
}

func (l listener) Address() string { return l.URL.String() }
