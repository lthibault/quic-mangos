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

const (
	statusOK        = iota
	statusBadHeader = iota
	statusBadPath   = iota
	statusBadProto  = iota
)

var mux = newMux()

type hasher interface {
	Hash() uint64
}

type path interface {
	hasher
	String() string
}

type pathString struct {
	sync.Once
	s string
	i uint64
}

func asPath(s string) *pathString { return &pathString{s: s} }

func (p *pathString) String() string { return p.s }

func (p *pathString) Hash() (i uint64) {
	p.Do(func() {
		hasher := md5.New()
		hasher.Write([]byte(p.s))
		p.i = binary.BigEndian.Uint64(hasher.Sum(nil)[:8])
	})

	return p.i
}

type pathHash uint64

func (p pathHash) Hash() uint64 { return uint64(p) }

type netlocator interface {
	Netloc() string
}

type netloc struct{ *url.URL }

func (n netloc) Netloc() string { return n.Host }

// connHeader is exchanged during the initial handshake.
type connHeader struct {
	Proto uint16
	Path  uint64 // 64-bit hash of the path
}

func (h *connHeader) Hash() uint64 { return h.Path }

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

func (m *multiplexer) RegisterPath(p path, ch chan<- *connRequest) (err error) {
	if !m.routes.Add(p, ch) {
		err = errors.Errorf("route already registered for %s", p.String())
	}
	return
}

func (m *multiplexer) UnregisterPath(p path) { m.routes.Del(p) }

func (m *multiplexer) Serve(sess quic.Session) {
	for _ = range ctx.Tick(sess.Context()) {
		stream, err := sess.AcceptStream()
		if err != nil {
			continue
		}

		go m.handshake(&streamNegotiator{stream})
	}
}

func (m *multiplexer) handshake(sn *streamNegotiator) {
	h, err := sn.Header()
	if err != nil {
		sn.Abort(statusBadHeader)
	} else if ch, ok := m.routes.Get(h); !ok {
		sn.Abort(statusBadPath)
	} else {
		ch <- &connRequest{H: h, Stream: sn.Stream}
	}

	// var h = new(connHeader)
	// if err := h.Load(s); err != nil {
	// 	// TODO:  find some way to report "invalid header" errors back to the dialer
	// 	_ = s.Close()
	// } else if ch, ok := m.routes.Get(pathHash(h.Hash())); !ok {
	// 	// TODO:  find some way to report "refused - nobody there" errors back to the dialer
	// 	_ = s.Close()
	// } else {
	// 	ch <- s
	// }
}

type router struct {
	sync.RWMutex
	routes map[uint64]chan<- *connRequest
}

func newRouter() *router { return &router{routes: make(map[uint64]chan<- *connRequest)} }

func (r *router) Get(h hasher) (ch chan<- *connRequest, ok bool) {
	r.RLock()
	defer r.RUnlock()

	// TODO:  hash the path
	ch, ok = r.routes[h.Hash()]
	return
}

func (r *router) Add(p path, ch chan<- *connRequest) (ok bool) {
	r.Lock()
	// TODO: hash the path
	if _, ok = r.routes[p.Hash()]; !ok {
		r.routes[p.Hash()] = ch
	}
	r.Unlock()
	ok = !ok // turn "value not found" into "value successfully inserted"
	return
}

func (r *router) Del(p path) {
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
	s         quic.Stream
	maxrx     int64 // NOTE: (probably) set via socket value in NewQUICConn
	num, pnum uint16
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

func (p quicPipe) LocalProtocol() uint16  { return p.num }
func (p quicPipe) RemoteProtocol() uint16 { return p.pnum }

func (p quicPipe) IsOpen() bool {
	select {
	case <-p.s.Context().Done():
		return false
	default:
		return true
	}
}

func (p quicPipe) GetProp(name string) (interface{}, error) {
	return nil, mangos.ErrBadProperty
}

func newQUICPipe(s quic.Stream, sock mangos.Socket) (p *quicPipe) {
	proto := sock.GetProtocol()

	p = &quicPipe{s: s, num: proto.Number(), pnum: proto.PeerNumber()}

	if v, e := sock.GetOption(mangos.OptionMaxRecvSize); e == nil {
		// socket guarantees this is an integer
		p.maxrx = int64(v.(int))
	}

	return
}

type streamNegotiator struct{ quic.Stream }

func (n *streamNegotiator) Header() (*connHeader, error) {
	h := new(connHeader)
	return h, binary.Read(n, binary.BigEndian, h)
}

// Abort attempts to notify the peer of the problem before closing the connection
func (n *streamNegotiator) Abort(status uint8) {
	binary.Write(n, binary.BigEndian, &connResp{Status: status})
}

type connRequest struct {
	H *connHeader
	quic.Stream
}

type connResp struct {
	Status uint8
	*connHeader
}

func dialPipe(path hasher, s quic.Stream, sock mangos.Socket) (p *quicPipe, err error) {
	h := &connHeader{
		Proto: sock.GetProtocol().Number(),
		Path:  path.Hash(),
	}

	if err = binary.Write(s, binary.BigEndian, h); err != nil {
		return
	}

	var resp connResp
	if err = binary.Read(s, binary.BigEndian, &resp); err != nil {
		return
	}

	switch resp.Status {
	case statusOK:

		if resp.Proto != sock.GetProtocol().PeerNumber() {
			err = errors.New("invalid peer protocol")
		} else {
			p = &quicPipe{s: s, num: h.Proto, pnum: resp.Proto}
			if v, e := sock.GetOption(mangos.OptionMaxRecvSize); e == nil {
				// socket guarantees this is an integer
				p.maxrx = int64(v.(int))
			}
		}

	case statusBadHeader:
		err = errors.New("missing or malformed header")
	case statusBadPath:
		err = errors.New("refused - nobody there")
	}

	return
}

func listenPipe(path hasher, r *connRequest, sock mangos.Socket) (*quicPipe, error) {

	resp := new(connResp)

	if r.H.Proto != sock.GetProtocol().PeerNumber() {
		resp.Status = statusBadProto
	} else {
		resp.Proto = sock.GetProtocol().Number()
		resp.Path = path.Hash()
	}

	if err := binary.Write(r, binary.BigEndian, resp); err != nil {
		return nil, err
	}

	p := &quicPipe{s: r, num: r.H.Proto, pnum: resp.Proto}
	if v, e := sock.GetOption(mangos.OptionMaxRecvSize); e == nil {
		// socket guarantees this is an integer
		p.maxrx = int64(v.(int))
	}

	return p, nil
}
