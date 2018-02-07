package quic

import (
	"context"
	"net"

	quic "github.com/lucas-clemente/quic-go"
)

var ( // interface constraints
	_ net.Addr      = mockAddrNetloc("")
	_ netlocator    = mockAddrNetloc("")
	_ quic.Listener = &mockLstn{}
	_ quic.Session  = &mockSess{}
)

type mockAddrNetloc string

func (mockAddrNetloc) Network() string  { return "quic" }
func (m mockAddrNetloc) String() string { return string(m) }
func (m mockAddrNetloc) Netloc() string { return m.String() }

type mockLstn struct {
	closed      bool
	sessFactory func() *mockSess
}

func (m mockLstn) Accept() (quic.Session, error) {
	if m.sessFactory == nil {
		return &mockSess{}, nil
	}
	return m.sessFactory(), nil
}

func (mockLstn) Addr() net.Addr                 { return mockAddrNetloc("") }
func (mockLstn) Listen() (quic.Listener, error) { return nil, nil }
func (m *mockLstn) Close() error {
	m.closed = true
	return nil
}

type mockSess struct {
	closed         bool
	contextFactory func() context.Context
}

func (mockSess) AcceptStream() (quic.Stream, error) { return nil, nil }

func (m mockSess) Context() context.Context {
	if m.contextFactory == nil {
		return context.TODO()
	}
	return m.contextFactory()
}

func (mockSess) LocalAddr() net.Addr                  { return mockAddrNetloc("") }
func (mockSess) OpenStream() (quic.Stream, error)     { return nil, nil }
func (mockSess) OpenStreamSync() (quic.Stream, error) { return nil, nil }
func (mockSess) RemoteAddr() net.Addr                 { return mockAddrNetloc("") }
func (m *mockSess) Close(error) error {
	m.closed = true
	return nil
}

func (*mockSess) ConnectionState() quic.ConnectionState { return quic.ConnectionState{} }
