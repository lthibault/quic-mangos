package quic

import (
	"net"
	"net/url"
)

// TODO:  re-implement
var transport transporter

type (
	transporter interface {
		Bind(*url.URL, chan<- net.Conn, *options) (router, error)
		Connect(*url.URL) (net.Conn, error)
	}

	router interface {
		RegisterPath(string, chan<- net.Conn) error
	}
)
