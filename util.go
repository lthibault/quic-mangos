package quic

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"sync"

	"github.com/go-mangos/mangos"
	quic "github.com/lucas-clemente/quic-go"
)

type options struct {
	sync.RWMutex
	opt map[string]interface{}
}

func newOpt() *options { return &options{opt: make(map[string]interface{})} }

// GetOption retrieves an option value.
func (o *options) get(name string) (interface{}, error) {
	o.RLock()
	defer o.RUnlock()

	v, ok := o.opt[name]
	if !ok {
		return nil, mangos.ErrBadOption
	}
	return v, nil
}

// SetOption sets an option.  We have none, so just ErrBadOption.
func (o *options) set(name string, val interface{}) (err error) {
	o.Lock()
	defer o.Unlock()

	switch name {
	case OptionQUICConfig, OptionTLSConfig:
		o.opt[name] = val
	default:
		err = mangos.ErrBadOption
	}
	return
}

// Setup a bare-bones TLS config for the server
func generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{Certificates: []tls.Certificate{tlsCert}, InsecureSkipVerify: true}
}

func getQUICCfg(opt *options) (tc *tls.Config, qc *quic.Config) {
	if v, err := opt.get(OptionTLSConfig); err != nil {
		tc = generateTLSConfig()
	} else {
		tc = v.(*tls.Config)
	}

	// It's acceptable for qc to be nil
	if v, err := opt.get(OptionQUICConfig); err == nil {
		qc = v.(*quic.Config)
	}

	return
}
