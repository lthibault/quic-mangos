package quic

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"sync"
	"unsafe"

	"github.com/SentimensRG/ctx"
	"github.com/SentimensRG/ctx/refctx"
	quic "github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/h2quic"
	"golang.org/x/sync/errgroup"

	radix "github.com/armon/go-radix"
	"github.com/pkg/errors"
)

var transport transporter = &trans{
	serverPool: &serverPool{svr: make(map[string]*server)},
	client:     &client{&http.Client{Transport: &h2quic.RoundTripper{}}},
}

type transporter interface {
	Bind(*url.URL, chan<- io.ReadWriteCloser, *options) (*router, error)
	Connect(*url.URL) (io.ReadWriteCloser, error)
}

type trans struct {
	*serverPool
	*client
}

type doer interface {
	Do(*http.Request) (*http.Response, error)
}

type clientConn struct {
	io.ReadCloser
	io.WriteCloser
}

func (c clientConn) Close() error {
	var g errgroup.Group
	g.Go(c.ReadCloser.Close)
	g.Go(c.WriteCloser.Close)
	return g.Wait()
}

type client struct{ doer }

func (c *client) Connect(u *url.URL) (io.ReadWriteCloser, error) {
	pr, pw := io.Pipe()
	req, err := http.NewRequest(http.MethodConnect, u.String(), pr)
	if err != nil {
		return nil, errors.Wrap(err, "new request")
	}

	res, err := c.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "http connect")
	}

	return &clientConn{
		WriteCloser: pw,
		ReadCloser:  res.Body,
	}, nil
}

type server struct {
	ctx.Doner
	err error
	ch  chan error
	*router
	h2     *h2quic.Server
	cancel func()
}

func newServer(netloc string, tlsc *tls.Config, qconf *quic.Config) *server {
	s := new(server)
	var c context.Context
	c, s.cancel = context.WithCancel(context.Background())

	var ctr *refctx.RefCtr
	s.Doner, ctr = refctx.WithRefCount(c)
	s.router = newRouter(ctr)

	s.h2 = newH2(netloc, s.router, tlsc, qconf)
	s.ch = make(chan error, 1)

	go func() {
		s.ch <- s.h2.ListenAndServe()
		_ = s.Close()
	}()

	return s
}

func (s server) Close() error {
	s.cancel()
	close(s.ch)
	return s.h2.Close()
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

func (p *serverPool) Bind(u *url.URL, ch chan<- io.ReadWriteCloser, opt *options) (*router, error) {
	p.Lock()
	defer p.Unlock()

	if r, ok := p.svr[u.Host]; ok {
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

	svr := newServer(u.Host, tlsc, qconf)
	if err := svr.RegisterPath(u.Path, ch); err != nil {
		// TODO:  shut down the server
		return nil, errors.Wrap(err, "register path")
	}

	p.svr[u.Host] = svr
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

type serverConn struct {
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

	if rtr.path.Exist(path) {
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
		c <- &serverConn{
			Writer:     w,
			ReadCloser: r.Body,
		}
	}

	rtr.RUnlock()
}
