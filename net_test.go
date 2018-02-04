package quic

import (
	"bytes"
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

}

func TestRefcntSession(t *testing.T) {

}