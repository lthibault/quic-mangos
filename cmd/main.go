package main

import (
	"log"
	"strconv"

	"github.com/go-mangos/mangos/protocol/pair"
	quic "github.com/lthibault/quic-mangos"
)

func maybeError(err error) {
	if err != nil {
		log.Fatal("was error:", err)
	}
}

func main() {
	s0, err := pair.NewSocket()
	if err != nil {
		log.Fatal("S0", err)
	}

	s1, err := pair.NewSocket()
	if err != nil {
		log.Fatal("S0", err)
	}

	s0.AddTransport(quic.NewTransport())
	s1.AddTransport(quic.NewTransport())

	maybeError(s0.Listen("quic://127.0.0.1:9001/foo"))
	maybeError(s1.Dial("quic://127.0.0.1:9001/foo"))

	go func() {
		defer s0.Close()
		for i := 0; i < 10; i++ {
			maybeError(s0.Send([]byte(strconv.Itoa(i))))
		}
	}()

	for i := 0; i < 10; i++ {
		msg, err := s1.Recv()
		if err != nil {
			log.Fatal("recv:", err)
		}
		log.Println(msg)
	}

	s1.Close()
}
