package main

import (
	"log"

	"github.com/go-mangos/mangos/protocol/pair"
	quic "github.com/lthibault/quic-mangos"
)

func main() {
	s0, err := pair.NewSocket()
	if err != nil {
		log.Fatalf("bind sock create: %s", err)
	}

	s1, err := pair.NewSocket()
	if err != nil {
		log.Fatalf("conn sock create: %s", err)
	}

	s0.AddTransport(quic.NewTransport())
	s1.AddTransport(quic.NewTransport())

	if err = s0.Listen("quic://localhost:9090/"); err != nil {
		log.Fatalf("s0 listen: %s", err)
	}

	if err = s1.Dial("quic://localhost:9090/"); err != nil {
		log.Fatalf("s1 dial: %s", err)
	}

	log.Println(" SENDING ...")
	if err = s0.Send([]byte("OH HAI!")); err != nil {
		log.Fatalf("send: %s", err)
	}

	log.Println(" RECVING ...")
	if b, err := s1.Recv(); err != nil {
		log.Fatalf("recv: %s", err)
	} else {
		log.Println("[ RECV ] ", string(b))
	}
}
