package main

import (
	"log"
	"time"

	"github.com/SentimensRG/ctx"
	"github.com/SentimensRG/ctx/sigctx"
	quic "github.com/lthibault/quic-mangos"
	"github.com/nanomsg/mangos/protocol/pair"
)

const (
	addr      = "quic://127.0.0.1:9001/some/arbitrary/path/"
	dialDelay = time.Millisecond * 1
)

func main() {
	s0, err := pair.NewSocket()
	if err != nil {
		panic(err)
	}

	s1, err := pair.NewSocket()
	if err != nil {
		panic(err)
	}

	s0.AddTransport(quic.NewTransport())
	s1.AddTransport(quic.NewTransport())

	if err = s0.Listen(addr); err != nil {
		panic(err)
	}

	if err = s1.Dial(addr); err != nil {
		panic(err)
	}

	// sockets connect asyncrhonously.  This is a limitation of go-mangos itself
	// so just wait 1s for everythign to set up properly before continuing.
	<-time.After(dialDelay)

	// context that completes on SIGINT or SIGTERM
	c := sigctx.New()

	go ctx.FTick(c, func() {
		if err := s0.Send([]byte("HELLO FROM SOCKET 0")); err != nil {
			panic(err)
		}
		<-time.After(time.Millisecond * 250)
	})

	go ctx.FTick(c, func() {
		if msg, err := s1.Recv(); err != nil {
			panic(err)
		} else {
			log.Println(string(msg))
		}
	})

	<-c.Done()
}
