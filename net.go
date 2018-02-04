package quic

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"sync"

	radix "github.com/armon/go-radix"
)

type negotiator struct {
	io.ReadWriteCloser
	buf *bytes.Buffer
}

func newNegotiator(pipe io.ReadWriteCloser) *negotiator {
	return &negotiator{ReadWriteCloser: pipe, buf: new(bytes.Buffer)}
}

func (n negotiator) ReadHeaders() (path string, err error) {
	scanner := bufio.NewScanner(n)

	if !scanner.Scan() {
		err = io.EOF
	} else if err = scanner.Err(); err == nil {
		path = scanner.Text()
	}

	return
}

func (n negotiator) Abort(status int, message string) {
	_, err := n.buf.WriteString(fmt.Sprintf("%d:%s", status, message))
	if err != nil {
		panic(err) // should never happen
	}

	_, _ = io.Copy(n, n.buf) // best-effort
	n.buf.Reset()
}

func (n negotiator) Accept() (err error) {
	_, err = n.Write([]byte("\n"))
	return
}

type router struct {
	sync.RWMutex
	routes *radix.Tree
}

func (r *router) Get(path string) (ch chan<- net.Conn, ok bool) {
	r.RLock()
	defer r.RUnlock()

	var v interface{}
	if v, ok = r.routes.Get(path); ok {
		ch = v.(chan<- net.Conn)
	}

	return
}

func (r *router) Add(path string, ch chan<- net.Conn) (ok bool) {
	r.Lock()
	if _, ok = r.routes.Get(path); ok {
		r.routes.Insert(path, ch)
	}
	r.Unlock()
	return
}

func (r *router) Del(path string) {
	r.Lock()
	r.routes.Delete(path)
	r.Unlock()
}
