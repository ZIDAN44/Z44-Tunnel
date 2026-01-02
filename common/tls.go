package common

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// LoadCACertPool loads the CA certificate and creates a cert pool
func LoadCACertPool(caPath string) (*x509.CertPool, error) {
	caCert, err := os.ReadFile(caPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	return pool, nil
}

// LoadCertKeyPair loads a certificate and key pair
func LoadCertKeyPair(certPath, keyPath string) (tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to load certificate: %w", err)
	}
	return cert, nil
}
