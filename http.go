package quic

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"sync"
	"unsafe"

	"github.com/SentimensRG/ctx"
	"github.com/SentimensRG/ctx/refctx"
	quic "github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/h2quic"

	radix "github.com/armon/go-radix"
	"github.com/pkg/errors"
)

var transport transporter = &serverPool{svr: make(map[string]*server)}

type transporter interface {
	MaybeInit(string, *options) (*router, error)
}

type server struct {
	ctx.Doner
	err error
	ch  chan error
	*router
	h2 *h2quic.Server
}

func newServer(netloc string, tlsc *tls.Config, qconf *quic.Config) *server {
	s := new(server)
	c, cancel := context.WithCancel(context.Background())

	var ctr *refctx.RefCtr
	s.Doner, ctr = refctx.WithRefCount(c)
	s.router = newRouter(ctr)

	s.h2 = newH2(netloc, s.router, tlsc, qconf)
	s.ch = make(chan error, 1)

	go func() {
		s.ch <- s.h2.ListenAndServe()
		cancel()
		close(s.ch)
	}()

	return s
}

func (s *server) Err() error {
	if s.err == nil {
		select {
		case s.err = <-s.ch:
		default:
		}
	}

	return s.err
}

type serverPool struct {
	sync.RWMutex
	svr map[string]*server
}

func (p *serverPool) GC(svr *h2quic.Server) func() {
	return func() {
		p.Lock()
		svr.Close()
		delete(p.svr, svr.Addr)
		p.Unlock()
	}
}

func (p *serverPool) MaybeInit(netloc string, opt *options) (*router, error) {
	p.Lock()
	defer p.Unlock()

	if r, ok := p.svr[netloc]; ok {
		return r.router, nil
	}

	var tlsc *tls.Config
	if v, err := opt.get(OptionTLSConfig); err != nil {
		tlsc = generateTLSConfig()
	} else {
		tlsc = v.(*tls.Config)
	}

	var qconf *quic.Config
	if v, err := opt.get(OptionQUICConfig); err == nil {
		qconf = v.(*quic.Config)
	}

	svr := newServer(netloc, tlsc, qconf)
	p.svr[netloc] = svr
	ctx.Defer(svr, p.GC(svr.h2))

	return svr.router, svr.Err()
}

func newH2(addr string, h http.Handler, tlsc *tls.Config, cfg *quic.Config) *h2quic.Server {
	return &h2quic.Server{
		Server: &http.Server{
			Addr:      addr,
			Handler:   h,
			TLSConfig: tlsc,
		},
		QuicConfig: cfg,
	}
}

type fullDuplexConn struct {
	io.Writer
	io.ReadCloser
}

type rwcGuard radix.Tree

func (g *rwcGuard) Get(path string) (ch chan<- io.ReadWriteCloser, ok bool) {
	var v interface{}
	if v, ok = (*radix.Tree)(unsafe.Pointer(g)).Get(path); ok {
		ch = v.(chan<- io.ReadWriteCloser)
	}
	return
}

func (g *rwcGuard) Exist(path string) (exist bool) {
	_, exist = (*radix.Tree)(unsafe.Pointer(g)).Get(path)
	return
}

func (g *rwcGuard) Del(path string) {
	(*radix.Tree)(unsafe.Pointer(g)).Delete(path)
}

func (g *rwcGuard) Add(path string, ch chan<- io.ReadWriteCloser) (ok bool) {
	if ok = g.Exist(path); ok {
		(*radix.Tree)(unsafe.Pointer(g)).Insert(path, ch)
	}
	return
}

type router struct {
	sync.RWMutex
	*refctx.RefCtr
	path *rwcGuard
}

func newRouter(r *refctx.RefCtr) *router {
	return &router{RefCtr: r, path: (*rwcGuard)(radix.New())}
}

func (rtr *router) RegisterPath(path string, ch chan<- io.ReadWriteCloser) error {
	rtr.Lock()
	defer rtr.Unlock()

	if !rtr.path.Exist(path) {
		return errors.Errorf("handler exists at %s", path)
	}

	rtr.path.Add(path, ch)
	return nil
}

func (rtr *router) Cleanup(path string) func() {
	return func() {
		rtr.Lock()
		rtr.path.Del(path)
		rtr.Unlock()
	}
}

func (rtr *router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rtr.RLock()

	if c, ok := rtr.path.Get(r.URL.Path); !ok {
		http.Error(w, "no listener at "+r.URL.Path, http.StatusNotFound)
	} else {
		c <- &fullDuplexConn{
			Writer:     w,
			ReadCloser: r.Body,
		}
	}

	rtr.RUnlock()
}
