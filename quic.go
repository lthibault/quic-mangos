package quic

import (
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"sync"

	"github.com/SentimensRG/ctx"
	"github.com/go-mangos/mangos"
	quic "github.com/lucas-clemente/quic-go"
	"github.com/pkg/errors"
)

const (
	// OptionTLSConfig maps to a *tls.Config value
	OptionTLSConfig = "QUIC-TLS-CONFIG"
	// OptionQUICConfig maps to a *quic.Config value
	OptionQUICConfig = "QUIC-UDP-CONFIG"
	// OptionAcceptTimeout limits the amount of time we wait to accept a connection
)

type options struct {
	sync.RWMutex
	opt map[string]interface{}
}

// GetOption retrieves an option value.
func (o *options) get(name string) (interface{}, error) {
	o.RLock()
	defer o.RUnlock()

	if o.opt == nil {
		return nil, mangos.ErrBadOption
	}
	v, ok := o.opt[name]
	if !ok {
		return nil, mangos.ErrBadOption
	}
	return v, nil
}

// SetOption sets an option.  We have none, so just ErrBadOption.
func (o *options) set(name string, val interface{}) error {
	o.Lock()
	defer o.Unlock()
	return mangos.ErrBadOption
}

type multiplexer interface {
	sync.Locker

	GetListener(netlocator) (*refcntListener, bool)
	SetListener(netlocator, *refcntListener)
	DelListener(netlocator)

	RegisterPath(string, chan<- net.Conn) error
	UnregisterPath(string)

	Serve(quic.Session)
}

type transport struct {
	sync.Mutex
	opt *options

	routes    *router
	listeners map[string]*refcntListener
}

func (*transport) Scheme() string { return "quic" }

func (t *transport) NewDialer(addr string, sock mangos.Socket) (mangos.PipeDialer, error) {
	u, err := url.ParseRequestURI(addr)
	if err != nil {
		return nil, errors.Wrap(err, "url parse")
	}

	u.Path = filepath.Clean(u.Path)

	return &dialer{
		URL:  u,
		opt:  t.opt,
		sock: sock,
	}, nil
}

func (t *transport) NewListener(addr string, sock mangos.Socket) (mangos.PipeListener, error) {
	u, err := url.ParseRequestURI(addr)
	if err != nil {
		return nil, errors.Wrap(err, "url parse")
	}

	u.Path = filepath.Clean(u.Path)

	return &listener{
		URL:         u,
		opt:         t.opt,
		sock:        sock,
		muxListener: newListenMux(t),
	}, nil
}

func netlocStr(n netlocator) string {
	return fmt.Sprintf("%s:%s", n.Hostname(), n.Port())
}

// Implement multiplexer
func (t *transport) GetListener(n netlocator) (l *refcntListener, ok bool) {
	l, ok = t.listeners[netlocStr(n)]
	return
}

func (t *transport) SetListener(n netlocator, l *refcntListener) {
	t.listeners[netlocStr(n)] = l
}

func (t *transport) DelListener(n netlocator) {
	t.Lock()
	delete(t.listeners, netlocStr(n))
	t.Unlock()
}

func (t *transport) RegisterPath(path string, ch chan<- net.Conn) (err error) {
	if !t.routes.Add(path, ch) {
		err = errors.Errorf("route already registered for %s", path)
	}
	return
}

func (t *transport) UnregisterPath(path string) { t.routes.Del(path) }

func (t *transport) Serve(sess quic.Session) {
	for _ = range ctx.Tick(sess.Context()) {
		stream, err := sess.AcceptStream()
		if err != nil {
			continue
		}

		go t.routeStream(sess, stream)
	}
}

func (t *transport) routeStream(sess quic.Session, stream quic.Stream) {
	n := newNegotiator(stream)

	path, err := n.ReadHeaders()
	if err != nil {
		n.Abort(400, err.Error())
		return
	} else if ch, ok := t.routes.Get(path); !ok {
		n.Abort(404, path)
		return
	} else if err = n.Accept(); err != nil {
		_ = stream.Close()
	} else {
		ch <- &conn{Session: sess, Stream: stream}
	}
}

// NewTransport allocates a new quic:// transport.
func NewTransport() mangos.Transport {
	return &transport{opt: &options{
		opt: make(map[string]interface{})},
	}
}
