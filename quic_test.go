package quic

import (
	"testing"

	"github.com/go-mangos/mangos"
)

func TestNewDialer(t *testing.T) {
	var sock mangos.Socket

	addr := "quic://127.0.0.1:9001/clean//up/"
	trans := NewTransport()

	p, err := trans.NewDialer(addr, sock)
	if err != nil {
		t.Error(err)
	}
	d := p.(*dialer)

	if d.Path != "/clean/up" {
		t.Errorf("expected /clean/up, got %s", d.Path)
	}

	if d.opt == nil {
		t.Error("opt is nil")
	} else if d.sock != sock {
		t.Error("sock parameter points to unexpected location")
	} else if d.muxDialer == nil {
		t.Error("muxDialer is nil")
	}
}

func TestNewListener(t *testing.T) {

}

func TestMultiplexer(t *testing.T) {

}

// func TestIntegration(t *testing.T) {
// 	s0, err := pair.NewSocket()
// 	if err != nil {
// 		t.Errorf("bind sock create: %s", err)
// 	}

// 	s1, err := pair.NewSocket()
// 	if err != nil {
// 		t.Errorf("conn sock create: %s", err)
// 	}

// 	s0.AddTransport(NewTransport())
// 	s1.AddTransport(NewTransport())

// 	if err = s0.Listen("quic://localhost:9090/"); err != nil {
// 		t.Errorf("s0 listen: %s", err)
// 	}

// 	if err = s1.Dial("quic://localhost:9090/"); err != nil {
// 		t.Errorf("s1 dial: %s", err)
// 	}

// 	t.Log(" SENDING ...")
// 	if err = s0.Send([]byte("OH HAI!")); err != nil {
// 		t.Errorf("send: %s", err)
// 	}

// 	t.Log(" RECVING ...")
// 	if b, err := s1.Recv(); err != nil {
// 		t.Errorf("recv: %s", err)
// 	} else {
// 		t.Log("[ RECV ] ", string(b))
// 	}
// }
