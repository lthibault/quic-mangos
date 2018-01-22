package quic

import (
	"bytes"
	"io"

	"github.com/go-mangos/mangos"
	"github.com/pkg/errors"
)

type pipe struct {
	io.ReadWriteCloser
	proto  mangos.Protocol
	closed bool
}

func (p pipe) Send(m *mangos.Message) error {
	var buf []byte

	if m.Expired() {
		m.Free()
		return nil
	}

	if len(m.Header) > 0 {
		buf = make([]byte, 0, len(m.Header)+len(m.Body))
		buf = append(buf, m.Header...)
		buf = append(buf, m.Body...)
	} else {
		buf = m.Body
	}

	if _, err := io.Copy(p, bytes.NewBuffer(buf)); err != nil {
		return errors.Wrap(err, "write msg")
	}

	m.Free()
	return nil
}

func (p pipe) Recv() (*mangos.Message, error) {
	var b bytes.Buffer
	if _, err := io.Copy(&b, p); err != nil {
		return nil, errors.Wrap(err, "read msg")
	}

	m := mangos.NewMessage(0)
	m.Body = (&b).Bytes()
	return m, nil
}

func (p pipe) LocalProtocol() uint16  { return p.proto.Number() }
func (p pipe) RemoteProtocol() uint16 { return p.proto.PeerNumber() }

func (p pipe) IsOpen() bool { return !p.closed }

func (p *pipe) Close() error {
	p.closed = true
	return p.ReadWriteCloser.Close()
}

func (p pipe) GetProp(prop string) (v interface{}, err error) {
	if v, err = p.props.get(prop); err != nil {
		return nil, mangos.ErrBadProperty
	}
	return
}
