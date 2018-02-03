package quic

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"

	quic "github.com/lucas-clemente/quic-go"
)

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

type conn struct {
	quic.Session
	quic.Stream
}

func (c conn) Close() error { return c.Stream.Close() }
