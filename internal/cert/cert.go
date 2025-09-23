package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	ErrInvalidCA = errors.New("invalid CA certificate")
)

type Config struct {
	CAPath    string
	CacheSize int
}

type Manager struct {
	caCert       *x509.Certificate
	caPrivateKey *rsa.PrivateKey
	certStore    sync.Map
	config       *Config
}

// GetTLSConfig .
func (m *Manager) GetTLSConfig(host string) (*tls.Config, error) {
	if cert, ok := m.certStore.Load(host); ok {
		return &tls.Config{
			Certificates: []tls.Certificate{cert.(tls.Certificate)},
		}, nil
	}

	cert, privateKey, err := m.generateCert(host)
	if err != nil {
		return nil, err
	}
	tlsCertificate := tls.Certificate{
		Certificate: [][]byte{cert},
		PrivateKey:  privateKey,
	}

	if m.config.CacheSize > 0 {
		m.certStore.Store(host, tlsCertificate)
		m.config.CacheSize--
	}

	return &tls.Config{
		Certificates: []tls.Certificate{tlsCertificate},
	}, nil
}

func (m *Manager) generateCert(host string) ([]byte, *rsa.PrivateKey, error) {
	return nil, nil, nil
}

func (m *Manager) loadOrGenerateCA() error {
	caCertPath := filepath.Join(m.config.CAPath, "ca.crt")
	caKeyPath := filepath.Join(m.config.CAPath, "ca.key")

	// load cert in file path
	if cert, err := tls.LoadX509KeyPair(caCertPath, caKeyPath); err == nil {
		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			return ErrInvalidCA
		}
		m.caCert = x509Cert
		rsaPrivateKey, ok := cert.PrivateKey.(*rsa.PrivateKey)
		if !ok {
			return ErrInvalidCA
		}
		m.caPrivateKey = rsaPrivateKey
		return nil
	}

	// generate CA
	ca := &x509.Certificate{
		SerialNumber:          big.NewInt(2023),
		Subject:               pkix.Name{CommonName: "MITMFOXY CA"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caPrivateKey, err := rsa.GenerateKey(rand.Reader, 4<<10)
	if err != nil {
		return err
	}
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivateKey.PublicKey, caPrivateKey)
	if err != nil {
		return err
	}

	// save cert
	if err := os.MkdirAll(m.config.CAPath, 0o700); err != nil {
		return err
	}
	certOut, err := os.Create(caCertPath)
	if err != nil {
		return err
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: caBytes}); err != nil {
		return err
	}
	certOut.Close()

	keyOut, err := os.Create(caKeyPath)
	if err != nil {
		return err
	}
	if err := pem.Encode(keyOut, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivateKey),
	}); err != nil {
		return err
	}
	keyOut.Close()

	m.caCert = ca
	m.caPrivateKey = caPrivateKey
	return nil
}

func NewManager(conf *Config) (*Manager, error) {
	m := &Manager{config: conf}

	if err := m.loadOrGenerateCA(); err != nil {
		return nil, err
	}

	return m, nil
}
