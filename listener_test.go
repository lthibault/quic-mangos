package quic

import "testing"

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

}

func TestListener(t *testing.T) {

}
