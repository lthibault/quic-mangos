package quic

import (
	"crypto/md5"
	"encoding/binary"
	"io"
	"net"
	"net/url"
	"sync"
	"sync/atomic"

	"github.com/SentimensRG/ctx"
	"github.com/go-mangos/mangos"
	quic "github.com/lucas-clemente/quic-go"
	"github.com/pkg/errors"
)

var mux = newMux()

type path struct {
	s string
	i uint32
}

func asPath(s string) *path { return &path{s: s} }

func (p *path) Hash() (i uint32) {
	if p.i == 0 {
		hasher := md5.New()
		hasher.Write([]byte(p.s))
		p.i = binary.BigEndian.Uint32(hasher.Sum(nil)[:8])
	}
	return p.i
}

type netlocator interface {
	Netloc() string
}

type netloc struct{ *url.URL }

func (n netloc) Netloc() string { return n.Host }

type sessionDropper interface {
	DelSession(net.Addr)
}

type dialMuxer interface {
	sync.Locker
	GetSession(netlocator) (*refcntSession, bool)
	AddSession(net.Addr, *refcntSession)
	sessionDropper
}

type multiplexer struct {
	sync.Mutex
	listeners map[string]*refcntListener
	sessions  map[string]*refcntSession
	routes    *router
}

func newMux() *multiplexer {
	return &multiplexer{
		listeners: make(map[string]*refcntListener),
		sessions:  make(map[string]*refcntSession),
		routes:    newRouter(),
	}
}

func (m *multiplexer) GetListener(n netlocator) (l *refcntListener, ok bool) {
	l, ok = m.listeners[n.Netloc()]
	return
}

func (m *multiplexer) AddListener(n netlocator, l *refcntListener) {
	m.listeners[n.Netloc()] = l
}

func (m *multiplexer) DelListener(n netlocator) {
	m.Lock()
	delete(m.listeners, n.Netloc())
	m.Unlock()
}

func (m *multiplexer) GetSession(n netlocator) (s *refcntSession, ok bool) {
	s, ok = m.sessions[n.Netloc()]
	return
}

func (m *multiplexer) AddSession(a net.Addr, sess *refcntSession) {
	m.sessions[a.String()] = sess
}

func (m *multiplexer) DelSession(a net.Addr) {
	m.Lock()
	delete(m.sessions, a.String())
	m.Unlock()
}

func (m *multiplexer) RegisterPath(p *path, ch chan<- quic.Stream) (err error) {
	if !m.routes.Add(p, ch) {
		err = errors.Errorf("route already registered for %s", p.s)
	}
	return
}

func (m *multiplexer) UnregisterPath(p *path) { m.routes.Del(p) }

func (m *multiplexer) Serve(sess quic.Session) {
	for _ = range ctx.Tick(sess.Context()) {
		stream, err := sess.AcceptStream()
		if err != nil {
			continue
		}

		go m.routeStream(stream)
	}
}

func (m *multiplexer) routeStream(stream quic.Stream) {
	panic("NOT IMPLEMENTED")

	// var n listenNegotiator = newNegotiator(stream)

	// path, err := n.ReadHeaders()
	// if err != nil {
	// 	n.Abort(400, err.Error())
	// 	return
	// } else if ch, ok := m.routes.Get(path); !ok {
	// 	n.Abort(404, path)
	// 	return
	// } else if err = n.Accept(); err != nil {
	// 	_ = stream.Close()
	// } else {
	// 	ch <- stream
	// }
}

type router struct {
	sync.RWMutex
	routes map[uint32]chan<- quic.Stream
}

func newRouter() *router { return &router{routes: make(map[uint32]chan<- quic.Stream)} }

func (r *router) Get(p *path) (ch chan<- quic.Stream, ok bool) {
	r.RLock()
	defer r.RUnlock()

	// TODO:  hash the path
	ch, ok = r.routes[p.Hash()]
	return
}

func (r *router) Add(p *path, ch chan<- quic.Stream) (ok bool) {
	r.Lock()
	// TODO: hash the path
	if _, ok = r.routes[p.Hash()]; !ok {
		r.routes[p.Hash()] = ch
	}
	r.Unlock()
	ok = !ok // turn "value not found" into "value successfully inserted"
	return
}

func (r *router) Del(p *path) {
	r.Lock()
	// TODO: hash the path
	delete(r.routes, p.Hash())
	r.Unlock()
}

type refcntSession struct {
	gc     func()
	refcnt int32
	quic.Session
}

func newRefCntSession(sess quic.Session, d sessionDropper) *refcntSession {
	return &refcntSession{
		Session: sess,
		gc:      func() { d.DelSession(sess.RemoteAddr()) },
	}
}

func (r *refcntSession) Incr() *refcntSession {
	atomic.AddInt32(&r.refcnt, 1)
	return r
}

func (r *refcntSession) DecrAndClose() (err error) {
	if i := atomic.AddInt32(&r.refcnt, -1); i == 0 {
		err = r.Close(nil)
		r.gc()
	} else if i < 0 {
		panic(errors.New("already closed"))
	}
	return
}

type quicPipe struct {
	s     quic.Stream
	maxrx int64 // NOTE: (probably) set via socket value in NewQUICConn
	sock  mangos.Socket
	proto mangos.Protocol
	props map[string]interface{}
}

func (p quicPipe) Send(msg *mangos.Message) error {
	l := uint64(len(msg.Header) + len(msg.Body))

	if msg.Expired() {
		msg.Free()
		return nil
	}

	// send length header
	if err := binary.Write(p.s, binary.BigEndian, l); err != nil {
		return err
	}
	if _, err := p.s.Write(msg.Header); err != nil {
		return err
	}
	// hope this works
	if _, err := p.s.Write(msg.Body); err != nil {
		return err
	}
	msg.Free()
	return nil
}

func (p quicPipe) Recv() (msg *mangos.Message, err error) {
	var sz int64
	if err = binary.Read(p.s, binary.BigEndian, &sz); err != nil {
		return nil, err
	}

	// Limit messages to the maximum receive value, if not
	// unlimited.  This avoids a potential denaial of service.
	if sz < 0 || (p.maxrx > 0 && sz > p.maxrx) {
		return nil, mangos.ErrTooLong
	}
	msg = mangos.NewMessage(int(sz))
	msg.Body = msg.Body[0:sz]
	if _, err = io.ReadFull(p.s, msg.Body); err != nil {
		msg.Free()
		return nil, err
	}
	return msg, nil
}

func (p quicPipe) Close() error { return p.s.Close() }

func (p quicPipe) LocalProtocol() uint16 { return p.proto.Number() }

func (p quicPipe) RemoteProtocol() uint16 { return p.proto.PeerNumber() }

func (p quicPipe) IsOpen() bool {
	select {
	case <-p.s.Context().Done():
		return false
	default:
		return true
	}
}

func (p quicPipe) GetProp(name string) (interface{}, error) {
	if v, ok := p.props[name]; ok {
		return v, nil
	}
	return nil, mangos.ErrBadProperty
}
