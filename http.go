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

var transport = &serverPool{svr: make(map[string]*router)}

type serverPool struct {
	sync.RWMutex
	svr map[string]*router
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
		r.Incr()
		return r, nil
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

	c, cancel := context.WithCancel(context.Background())

	s := newH2(netloc, tlsc, qconf)
	go func() {
		// TODO:  we should return an error if ListenAndServe fails...
		_ = s.ListenAndServe()
		cancel()
	}()

	c, ctr := refctx.WithRefCount(c)
	r := newRouter(ctr)
	r.Incr()
	p.svr[netloc] = r
	ctx.Defer(c, p.GC(s))

	return r, nil
}

func newH2(addr string, tlsc *tls.Config, cfg *quic.Config) *h2quic.Server {
	return &h2quic.Server{
		Server: &http.Server{
			Addr:      addr,
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
