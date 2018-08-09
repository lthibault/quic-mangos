package quic

import (
	"testing"

	"github.com/nanomsg/mangos"
)

func TestCanary(t *testing.T) {}

func TestNewTransport(t *testing.T) {
	if NewTransport().Scheme() != "quic" {
		t.Errorf("expected sheme to be `quic `, got %s", NewTransport().Scheme())
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

		if d.sock != sock {
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

		if l.sock != sock {
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
