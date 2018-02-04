package quic

import (
	"crypto/tls"
	"net"
	"sync/atomic"

	"github.com/SentimensRG/ctx"
	"github.com/go-mangos/mangos"
	quic "github.com/lucas-clemente/quic-go"
	"github.com/pkg/errors"
)

type listenDeleter interface {
	DelListener(netlocator)
}

type refcntListener struct {
	ctx.Doner
	gc     func()
	refcnt int32
	quic.Listener
}

func newRefCntListener(n netlocator, l quic.Listener, d listenDeleter) *refcntListener {
	cq := make(chan struct{})
	return &refcntListener{
		Listener: l,
		Doner:    ctx.Lift(cq),
		gc: func() {
			close(cq)
			d.DelListener(n)
		},
	}
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
		panic("already closed")
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

func (lm *listenMux) LoadListener(n netlocator, tc *tls.Config, qc *quic.Config) error {
	lock.Lock()
	defer lock.Unlock()

	var ok bool
	if lm.l, ok = lm.mux.GetListener(n); !ok {

		// We don't have a listener for this netloc yet, so create it.
		ql, err := quic.ListenAddr(n.Netloc(), tc, qc)
		if err != nil {
			return err
		}

		// Init refcnt to track the Listener's usage and clean up when we're done
		lm.l = newRefCntListener(n, ql, lm.mux)
		lm.mux.AddListener(n, lm.l)
	}

	lm.l.Incr()
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
			lock.Lock()
			defer lock.Unlock()

			sess := newRefCntSession(sess, lm.mux)
			lm.mux.AddSession(sess.RemoteAddr(), sess.Incr())

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
	netloc

	*listenMux

	opt  *options
	sock mangos.Socket
}

func (l *listener) Listen() error {
	tc, qc := getQUICCfg(l.opt)
	return errors.Wrap(l.LoadListener(l.netloc, tc, qc), "listen quic")
}

func (l listener) Accept() (mangos.Pipe, error) {
	conn, err := l.listenMux.Accept(l.Path)
	if err != nil {
		return nil, errors.Wrap(err, "mux accept")
	}

	return mangos.NewConnPipe(conn, l.sock)
}

func (l listener) Close() error {
	return l.listenMux.Close(l.Path)
}

func (l listener) SetOption(name string, value interface{}) error {
	return l.opt.set(name, value)
}

func (l listener) GetOption(name string) (interface{}, error) {
	return l.opt.get(name)
}

func (l listener) Address() string { return l.URL.String() }
