package quic

import (
	"net/url"
	"testing"

	"github.com/go-mangos/mangos"
)

type mockAddrNetloc string

func (mockAddrNetloc) Network() string  { return "quic" }
func (m mockAddrNetloc) String() string { return string(m) }
func (m mockAddrNetloc) Netloc() string { return m.String() }

func TestNewTransport(t *testing.T) {
	trans := NewTransport().(*transport)
	if trans.opt == nil {
		t.Error("opt is nil")
	} else if trans.routes == nil {
		t.Error("router is nil")
	} else if trans.listeners == nil {
		t.Error("listeners is nil")
	} else if trans.sessions == nil {
		t.Error("sessions is nil")
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
		} else if d.muxDialer == nil {
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
		} else if l.muxListener == nil {
			t.Error("muxListener is nil")
		}
	})

	t.Run("BadURL", func(t *testing.T) {
		addr := "xxx"
		if _, err := trans.NewListener(addr, sock); err == nil {
			t.Error("should have failed due to invalid URL")
		}
	})
}

func TestMultiplexer(t *testing.T) {
	trans := NewTransport().(*transport)
	u, _ := url.ParseRequestURI("quic://127.0.0.1:9001/hello")
	n := &netloc{URL: u}

	t.Run("TestListenerOps", func(t *testing.T) {

		rfcl := new(refcntListener)

		t.Run("AddListener", func(t *testing.T) {
			trans.AddListener(n, rfcl)

			if l, ok := trans.listeners[n.Netloc()]; !ok {
				t.Error("listener was not added to map")
			} else if l != rfcl {
				t.Error("listener pointer mismatch")
			}
		})

		t.Run("GetListener", func(t *testing.T) {
			if l, ok := trans.GetListener(n); !ok {
				t.Error("listener was not found in map")
			} else if l != rfcl {
				t.Error("listener pointer mismatch")
			}
		})

		t.Run("DelListener", func(t *testing.T) {
			trans.DelListener(n)
			if _, ok := trans.listeners[n.Netloc()]; ok {
				t.Error("listener not removed")
			}
		})
	})

	t.Run("TestSessionOps", func(t *testing.T) {

		rfcs := new(refcntSession)

		t.Run("AddSession", func(t *testing.T) {
			trans.AddSession(mockAddrNetloc(n.String()), rfcs)

			if s, ok := trans.sessions[n.String()]; !ok {
				t.Error("session was not added to map")
			} else if s != rfcs {
				t.Error("session pointer mismatch")
			}
		})

		t.Run("GetSession", func(t *testing.T) {
			if s, ok := trans.GetSession(mockAddrNetloc(n.String())); !ok {
				t.Error("session was not found in map")
			} else if s != rfcs {
				t.Error("session pointer mismatch")
			}
		})

		t.Run("DelSession", func(t *testing.T) {
			trans.DelSession(mockAddrNetloc(n.String()))
			if _, ok := trans.sessions[n.String()]; ok {
				t.Error("session not removed")
			}
		})
	})

	t.Run("TestRouterOps", func(t *testing.T) {
		t.Run("RegisterPath", func(t *testing.T) {

		})

		t.Run("UnregisterPath", func(t *testing.T) {

		})

		t.Run("Serve", func(t *testing.T) {

		})

		t.Run("routeStream", func(t *testing.T) {

		})
	})
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
