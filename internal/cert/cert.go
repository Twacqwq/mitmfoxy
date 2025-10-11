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
	"net"
	"os"
	"sync"
	"time"
)

// Cert Manager
type Manager struct {
	certBlock, keyBlock *pem.Block
	store               sync.Map
}

// GetCert issue a certificate based on the upstream server's SNI (continuous optimization)
func (m *Manager) GetCert(serverName string) (*tls.Certificate, error) {
	if m.certBlock == nil || m.keyBlock == nil {
		return nil, errors.New("failed to load pem block")
	}

	if val, ok := m.store.Load(serverName); ok {
		return val.(*tls.Certificate), nil
	}

	rootCert, err := x509.ParseCertificate(m.certBlock.Bytes)
	if err != nil {
		return nil, err
	}

	ip := net.ParseIP(serverName)
	if ip != nil {
		rootCert.IPAddresses = []net.IP{ip}
	} else {
		rootCert.DNSNames = []string{serverName}
	}

	rootKey, err := x509.ParsePKCS8PrivateKey(m.keyBlock.Bytes)
	if err != nil {
		return nil, err
	}

	pemCertData, pemKeyData, err := generateCert(rootCert, rootKey.(*rsa.PrivateKey), serverName)
	if err != nil {
		return nil, err
	}

	certificate, err := tls.X509KeyPair(pemCertData, pemKeyData)
	if err != nil {
		return nil, err
	}

	m.store.Store(serverName, &certificate)

	return &certificate, nil
}

func NewManager(certPath, keyPath string) (*Manager, error) {
	rootCertPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}

	rootKeyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	rootCertBlock, _ := pem.Decode(rootCertPEM)
	if rootCertBlock == nil {
		return nil, errors.New("failed to decode root certificate")
	}

	rootKeyBlock, _ := pem.Decode(rootKeyPEM)
	if rootKeyBlock == nil {
		return nil, errors.New("failed to decode root key")
	}

	return &Manager{
		store:     sync.Map{},
		certBlock: rootCertBlock,
		keyBlock:  rootKeyBlock,
	}, nil
}

func generateCert(rootCert *x509.Certificate, rootKey *rsa.PrivateKey, serverName string) ([]byte, []byte, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(time.Now().UnixNano()/100000), 128))
	if err != nil {
		return nil, nil, err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: serverName,
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, rootCert, &priv.PublicKey, rootKey)
	if err != nil {
		return nil, nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	return certPEM, keyPEM, nil
}
