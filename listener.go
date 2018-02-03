package quic

import (
	"crypto/tls"
	"net"
	"net/url"

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

// listenMux implements muxListener
type listenMux struct {
	ql quic.Listener
}

func newListenMux(m multiplexer) *listenMux {
	return nil
}

func (lm listenMux) LoadListener(n netlocator, tc *tls.Config, qc *quic.Config) error {

	// TODO:  if we already have a listener on the netloc, load it into the listenmux
	// TODO:  if we _don't_ have a listener on the netloc, init and load into the listenmux
	// TODO:  incr the listener

	return errors.New("LOADLISTENER NOT IMPLEMENTED")
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
	// TODO:  decrement a counter such that the underlying listener is closed
	// when it equals 0
	return errors.New("CLOSE NOT IMPLEMENTED")
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
