package quic

import (
	"testing"

	"github.com/go-mangos/mangos/protocol/star"
)

func TestListen(t *testing.T) {
	sock, err := star.NewSocket()
	if err != nil {
		t.Errorf("socket create: %s", err)
	}

	sock.AddTransport(NewTransport())

	if err = sock.Listen("quic://127.0.0.1:9001/test"); err != nil {
		t.Errorf("socket listen: %s", err)
	}
}
