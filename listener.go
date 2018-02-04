package quic

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"sync/atomic"

	"github.com/SentimensRG/ctx"
	"github.com/go-mangos/mangos"
	quic "github.com/lucas-clemente/quic-go"
	"github.com/pkg/errors"
)

type muxListener interface {
	LoadListener(netlocator, *tls.Config, *quic.Config) error
	Accept(string) (net.Conn, error)
	Close(string) error
}

type netlocator interface {
	Hostname() string
	Port() string
}

type refcntListener struct {
	ctx.Doner
	gc     func()
	refcnt int32
	quic.Listener
}

func newRefCntListener(n netlocator, l quic.Listener, mux multiplexer) *refcntListener {
	cq := make(chan struct{})
	return &refcntListener{
		Listener: l,
		refcnt:   1,
		Doner:    ctx.Lift(cq),
		gc: func() {
			close(cq)
			mux.DelListener(n)
			mux = nil // for safety.  make sure subsequent usage panics
		},
	}
}

func (r *refcntListener) Incr() *refcntListener {
	atomic.AddInt32(&r.refcnt, 1)
	return r
}

func (r *refcntListener) DecrAndClose() (err error) {
	if atomic.AddInt32(&r.refcnt, -1) == 0 {
		err = r.Close()
		r.gc() // will panic if closed more than once
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
		lm.l = newRefCntListener(n, ql, lm.mux)

		lm.mux.SetListener(n, lm.l)
	}

	return nil
}

func (lm listenMux) Accept(path string) (conn net.Conn, err error) {
	chConn := make(chan net.Conn)

	if err = lm.mux.RegisterPath(path, chConn); err != nil {
		err = errors.Wrapf(err, "register path %s", path)
		return
	}

	// Start the listen loop, which will produce sessions, accept their
	// streams, and route them to the appropriate endpoint.
	go ctx.FTick(lm.l, func() {
		if sess, err := lm.l.Accept(); err == nil {
			go lm.mux.Serve(sess)
		}
	})

	return <-chConn, nil
}

func (lm listenMux) Close(path string) error {
	lm.mux.UnregisterPath(path)
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

func (l listener) Close() error {
	return l.muxListener.Close(l.Path)
}

func (l listener) SetOption(name string, value interface{}) error {
	return l.opt.set(name, value)
}

func (l listener) GetOption(name string) (interface{}, error) {
	return l.opt.get(name)
}

func (l listener) Address() string { return l.URL.String() }
