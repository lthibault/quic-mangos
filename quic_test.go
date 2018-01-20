package quic

import (
	"net"
	"net/url"
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

func TestScratch(t *testing.T) {
	u, _ := url.Parse("http://google.com/:80")
	ip, err := net.ResolveIPAddr("ip", u.Host)
	if err != nil {
		t.Error(err)
	}
	t.Log(ip)

}
