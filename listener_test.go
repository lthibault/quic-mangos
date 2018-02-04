package quic

import (
	"crypto/tls"
	"testing"
	"time"

	quic "github.com/lucas-clemente/quic-go"
)

func TestRefcntListener(t *testing.T) {
	l := &mockLstn{}
	rfcl := newRefCntListener(mockAddrNetloc(""), l, mux)

	t.Run("CtrDefault=0", func(t *testing.T) {
		if rfcl.refcnt != 0 {
			t.Errorf("expected 0, got %d", rfcl.refcnt)
		}
	})

	t.Run("Incr", func(t *testing.T) {
		if rfcl.Incr().refcnt != 1 {
			t.Errorf("expected 1, got %d", rfcl.refcnt)
		}
	})

	t.Run("DecrAndClose", func(t *testing.T) {
		t.Run("Decr", func(t *testing.T) {
			if err := rfcl.Incr().DecrAndClose(); err != nil {
				t.Error(err)
			} else if rfcl.refcnt != 1 {
				t.Errorf("expected 1, got %d", rfcl.refcnt)
			}
		})

		t.Run("Close", func(t *testing.T) {
			if err := rfcl.DecrAndClose(); err != nil {
				t.Error(err)
			} else if !l.closed {
				t.Error("listener not closed")
			} else if rfcl.refcnt != 0 {
				t.Errorf("expected 0, got %d", rfcl.refcnt)
			}
		})

		t.Run("OverClose", func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Error("should have panicked")
				}
			}()
			rfcl.DecrAndClose()
		})
	})
}

func TestListenMux(t *testing.T) {
	const netloc = mockAddrNetloc("localhost:9001")

	t.Run("LoadListener", func(t *testing.T) {
		mx := newMux()
		lm := newListenMux(mx, func(string, *tls.Config, *quic.Config) (quic.Listener, error) {
			return &mockLstn{}, nil
		})

		if lm.l != nil {
			t.Error("listener is not nil upon init")
		}

		t.Run("TestMutexFree", func(t *testing.T) {
			ch := make(chan struct{})
			go func() {
				lm.mux.Lock()
				lm.mux.Unlock()
				close(ch)
			}()

			select {
			case <-ch:
			case <-time.After(time.Millisecond):
				t.Error("could not cycle lock")
			}
		})

		t.Run("FirstLoad", func(t *testing.T) {
			if err := lm.LoadListener(netloc, nil, nil); err != nil {
				t.Error(err)
			} else if lm.l == nil {
				t.Error("listener not loaded")
			}
		})

		t.Run("SubsequentLoad", func(t *testing.T) {
			lm.factory = func(string, *tls.Config, *quic.Config) (quic.Listener, error) {
				t.Error("listener was already loaded; should not have been reloaded")
				return nil, nil
			}

			if err := lm.LoadListener(netloc, nil, nil); err != nil {
				t.Error(err)
			} else if lm.l == nil {
				t.Error("listener not loaded")
			}
		})
	})

	// t.Run("Accept", func(t *testing.T) {
	// 	c, cancel := context.WithCancel(context.Background())
	// 	defer cancel()

	// 	mx := newMux()
	// 	lm := newListenMux(mx, func(string, *tls.Config, *quic.Config) (quic.Listener, error) {
	// 		return &mockLstn{sessFactory: func() *mockSess {
	// 			return &mockSess{contextFactory: func() context.Context {
	// 				return c
	// 			}}
	// 		}}, nil
	// 	})
	// 	defer lm.Close("") // make sure we break out of the FTick in Accept

	// 	// TODO:  check if path was registered (chan exists at mux.router path)
	// 	// TODO:  check if session was added to mx.sessions
	// })
}

func TestListener(t *testing.T) {

}
