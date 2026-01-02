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

// GetSecureCipherSuites returns a list of secure cipher suites for TLS 1.2
// TLS 1.3 cipher suites are secure by default and not configurable
func GetSecureCipherSuites() []uint16 {
	return []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
	}
}
