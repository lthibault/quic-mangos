package quic

import (
	"bytes"
	"net"
	"net/url"
	"testing"
)

func TestNetloc(t *testing.T) {
	u, _ := url.ParseRequestURI("quic://127.0.0.1:9001/hello")
	n := netloc{URL: u}

	if n.Netloc() != u.Host {
		t.Errorf("mismatch between %s an %s", n.Netloc(), u.Host)
	}
}

type bufCloser struct {
	*bytes.Buffer
	closed bool
}

func (b *bufCloser) Close() (err error) {
	b.closed = true
	return
}

func TestNegotiator(t *testing.T) {
	const path = "/some/path"

	buf := &bufCloser{Buffer: new(bytes.Buffer)}
	n := newNegotiator(buf)

	t.Run("PathNegotiation", func(t *testing.T) {
		defer buf.Reset()

		t.Run("WriteHeaders", func(t *testing.T) {
			if err := n.WriteHeaders(path); err != nil {
				t.Error(err)
			}

			if buf.String() != path+"\n" {
				t.Errorf("unexpected value in buffer: %v", buf.Bytes())
			}
		})

		t.Run("Readheaders", func(t *testing.T) {
			if p, err := n.ReadHeaders(); err != nil {
				t.Error(err)
			} else if p != path {
				t.Errorf("expected path `%s`, got `%s`", path, p)
			}
		})
	})

	t.Run("Accept/Ack", func(t *testing.T) {
		defer buf.Reset()

		t.Run("Accept", func(t *testing.T) {
			if err := n.Accept(); err != nil {
				t.Error(err)
			}
		})

		t.Run("Ack", func(t *testing.T) {
			if err := n.Ack(); err != nil {
				t.Error(err)
			}
		})
	})

	t.Run("Abort/Ack", func(t *testing.T) {
		defer buf.Reset()

		t.Run("Abort", func(t *testing.T) {
			if err := n.Abort(404, "not found"); err != nil {
				t.Error(err)
			} else if buf.String() != "404:not found" {
				t.Errorf("expected `404:not found`, got `%s`", buf.String())
			}
		})

		t.Run("Ack", func(t *testing.T) {
			if err := n.Ack(); err == nil {
				t.Error("no error reported for aborted transaction")
			}
		})
	})
}

func TestRouter(t *testing.T) {
	r := newRouter()
	ch := make(chan net.Conn)
	const path = "/some/path"

	t.Run("Add", func(t *testing.T) {
		if !r.Add(path, ch) {
			t.Errorf("failed to add channel to path %s", path)
		}

		if r.Add(path, ch) {
			t.Error("slot not detected as occupied")
		}
	})

	t.Run("Get", func(t *testing.T) {
		if c, ok := r.Get(path); !ok {
			t.Error("value not retrieved")
		} else if c != ch {
			t.Error("mismatch between retrieved values")
		}
	})

	t.Run("Del", func(t *testing.T) {
		r.Del(path)
		if _, ok := r.Get(path); ok {
			t.Error("value not deleted")
		}
	})
}

func TestRefcntSession(t *testing.T) {
	sess := &mockSess{}
	rfcs := newRefCntSession(sess, newMux())

	t.Run("CtrDefault=0", func(t *testing.T) {
		if rfcs.refcnt != 0 {
			t.Errorf("expected 0, got %d", rfcs.refcnt)
		}
	})

	t.Run("Incr", func(t *testing.T) {
		if rfcs.Incr().refcnt != 1 {
			t.Errorf("expected 1, got %d", rfcs.refcnt)
		}
	})

	t.Run("DecrAndClose", func(t *testing.T) {
		t.Run("Decr", func(t *testing.T) {
			if err := rfcs.Incr().DecrAndClose(); err != nil {
				t.Error(err)
			} else if rfcs.refcnt != 1 {
				t.Errorf("expected 1, got %d", rfcs.refcnt)
			}
		})

		t.Run("Close", func(t *testing.T) {
			if err := rfcs.DecrAndClose(); err != nil {
				t.Error(err)
			} else if !sess.closed {
				t.Error("session not closed")
			} else if rfcs.refcnt != 0 {
				t.Errorf("expected 0, got %d", rfcs.refcnt)
			}
		})

		t.Run("OverClose", func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Error("should have panicked")
				}
			}()
			rfcs.DecrAndClose()
		})
	})
}

func TestMultiplexer(t *testing.T) {
	mx := newMux()
	u, _ := url.ParseRequestURI("quic://127.0.0.1:9001/hello")
	n := &netloc{URL: u}

	t.Run("TestListenerOps", func(t *testing.T) {

		rfcl := new(refcntListener)

		t.Run("AddListener", func(t *testing.T) {
			mx.AddListener(n, rfcl)

			if l, ok := mx.listeners[n.Netloc()]; !ok {
				t.Error("listener was not added to map")
			} else if l != rfcl {
				t.Error("listener pointer mismatch")
			}
		})

		t.Run("GetListener", func(t *testing.T) {
			if l, ok := mx.GetListener(n); !ok {
				t.Error("listener was not found in map")
			} else if l != rfcl {
				t.Error("listener pointer mismatch")
			}
		})

		t.Run("DelListener", func(t *testing.T) {
			mx.DelListener(n)
			if _, ok := mx.listeners[n.Netloc()]; ok {
				t.Error("listener not removed")
			}
		})
	})

	t.Run("TestSessionOps", func(t *testing.T) {

		rfcs := new(refcntSession)

		t.Run("AddSession", func(t *testing.T) {
			mx.AddSession(mockAddrNetloc(n.String()), rfcs)

			if s, ok := mx.sessions[n.String()]; !ok {
				t.Error("session was not added to map")
			} else if s != rfcs {
				t.Error("session pointer mismatch")
			}
		})

		t.Run("GetSession", func(t *testing.T) {
			if s, ok := mx.GetSession(mockAddrNetloc(n.String())); !ok {
				t.Error("session was not found in map")
			} else if s != rfcs {
				t.Error("session pointer mismatch")
			}
		})

		t.Run("DelSession", func(t *testing.T) {
			mx.DelSession(mockAddrNetloc(n.String()))
			if _, ok := mx.sessions[n.String()]; ok {
				t.Error("session not removed")
			}
		})
	})

	t.Run("TestRouterOps", func(t *testing.T) {
		t.Run("RegisterPath", func(t *testing.T) {
			ch := make(chan net.Conn)

			t.Run("SlotFree", func(t *testing.T) {
				if err := mx.RegisterPath(n.Path, ch); err != nil {
					t.Error(err)
				}
			})

			t.Run("SlotOccupied", func(t *testing.T) {
				if err := mx.RegisterPath(n.Path, ch); err == nil {
					t.Errorf("expected %s to be occupied, was free", n.Path)
				}
			})
		})

		t.Run("UnregisterPath", func(t *testing.T) {
			t.Run("SlotOccupied", func(t *testing.T) {
				mx.UnregisterPath(n.Path)
				if _, ok := mx.routes.Get(n.Path); ok {
					t.Error("value not removed from radix tree")
				}
			})

			t.Run("SlotFree", func(t *testing.T) {
				// make sure nothing weird happens (e.g. panics)
				mx.UnregisterPath(n.Path)
			})
		})

		// t.Run("Serve", func(t *testing.T) {
		// this is too hard to test for now ... :/
		// })

		// t.Run("routeStream", func(t *testing.T) {
		// this is too hard to test for now ... :/
		// })
	})
}
