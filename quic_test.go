package quic

import (
	"testing"

	"github.com/go-mangos/mangos"
	"github.com/go-mangos/mangos/protocol/pair"
)

type mockAddrNetloc string

func (mockAddrNetloc) Network() string  { return "quic" }
func (m mockAddrNetloc) String() string { return string(m) }
func (m mockAddrNetloc) Netloc() string { return m.String() }

func TestNewTransport(t *testing.T) {
	trans := NewTransport().(*transport)
	if trans.opt == nil {
		t.Error("opt is nil")
	} else if trans.Scheme() != "quic" {
		t.Errorf("expected sheme to be `quic `, got %s", trans.Scheme())
	}
}

func TestNewDialer(t *testing.T) {
	var sock mangos.Socket
	trans := NewTransport()

	t.Run("SuccessfulInit", func(t *testing.T) {
		addr := "quic://127.0.0.1:9001/clean//up/"

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
		} else if d.dialMux == nil {
			t.Error("muxDialer is nil")
		}
	})

	t.Run("BadURL", func(t *testing.T) {
		addr := "xxx"
		if _, err := trans.NewDialer(addr, sock); err == nil {
			t.Error("should have failed due to invalid URL")
		}
	})
}

func TestNewListener(t *testing.T) {
	var sock mangos.Socket
	trans := NewTransport()

	t.Run("SuccessfulInit", func(t *testing.T) {
		addr := "quic://127.0.0.1:9001/clean//up/"

		p, err := trans.NewListener(addr, sock)
		if err != nil {
			t.Error(err)
		}
		l := p.(*listener)

		if l.Path != "/clean/up" {
			t.Errorf("expected /clean/up, got %s", l.Path)
		}

		if l.opt == nil {
			t.Error("opt is nil")
		} else if l.sock != sock {
			t.Error("sock parameter points to unexpected location")
		} else if l.listenMux == nil {
			t.Error("listenMux is nil")
		}
	})

	t.Run("BadURL", func(t *testing.T) {
		addr := "xxx"
		if _, err := trans.NewListener(addr, sock); err == nil {
			t.Error("should have failed due to invalid URL")
		}
	})
}

func TestIntegration(t *testing.T) {
	s0, err := pair.NewSocket()
	if err != nil {
		t.Errorf("bind sock create: %s", err)
	}

	s1, err := pair.NewSocket()
	if err != nil {
		t.Errorf("conn sock create: %s", err)
	}

	s0.AddTransport(NewTransport())
	s1.AddTransport(NewTransport())

	if err = s0.Listen("quic://localhost:9090/test"); err != nil {
		t.Errorf("s0 listen: %s", err)
	}

	if err = s1.Dial("quic://localhost:9090/test"); err != nil {
		t.Errorf("s1 dial: %s", err)
	}

	t.Log(" SENDING ...")
	if err = s0.Send([]byte("OH HAI!")); err != nil {
		t.Errorf("send: %s", err)
	}

	t.Log(" RECVING ...")
	if b, err := s1.Recv(); err != nil {
		t.Errorf("recv: %s", err)
	} else {
		t.Log("[ RECV ] ", string(b))
	}
}
