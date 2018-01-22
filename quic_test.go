package quic

import (
	"testing"

	"github.com/go-mangos/mangos/protocol/pair"
	"github.com/go-mangos/mangos/protocol/star"
	"github.com/go-mangos/mangos/transport/inproc"
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

func TestDial(t *testing.T) {
	sock, err := star.NewSocket()
	if err != nil {
		t.Errorf("socket create: %s", err)
	}

	sock.AddTransport(NewTransport())

	if err = sock.Dial("quic://127.0.0.1:9001/test"); err != nil {
		t.Errorf("socket dial: %s", err)
	}
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

	s0.AddTransport(inproc.NewTransport())
	s1.AddTransport(inproc.NewTransport())

	if err = s0.Listen("inproc:///test"); err != nil {
		t.Errorf("s0 listen: %s", err)
	}

	if err = s1.Dial("inproc:///test"); err != nil {
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
