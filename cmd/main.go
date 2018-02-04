package main

import (
	"log"
	"sync"

	"github.com/go-mangos/mangos/protocol/pair"
	quic "github.com/lthibault/quic-mangos"
	"github.com/pkg/errors"
)

func run(wg *sync.WaitGroup, path string) {
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

	if err = s0.Listen(path); err != nil {
		log.Fatal(err)
	}

	if err = s1.Dial(path); err != nil {
		log.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		if err = s0.Send([]byte(path)); err != nil {
			log.Fatal(errors.Wrap(err, "send"))
		}

		if b, err := s1.Recv(); err != nil {
			log.Fatal(errors.Wrap(err, "recv"))
		} else {
			log.Println("[ RECV ] ", string(b))
		}
	}

	wg.Done()
}

func main() {
	var wg sync.WaitGroup
	wg.Add(2)

	go run(&wg, "quic://127.0.0.1:9001/test/1")
	go run(&wg, "quic://127.0.0.1:9001/test/2")

	wg.Wait()
}
