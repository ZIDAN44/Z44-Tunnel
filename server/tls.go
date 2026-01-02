package main

import (
	"crypto/tls"
	"fmt"

	"z44-tunnel/common"
)

// LoadTLSConfig loads the TLS configuration for the server
func LoadTLSConfig() (*tls.Config, error) {
	// Load CA certificate pool
	caPool, err := common.LoadCACertPool("certs/ca.pem")
	if err != nil {
		return nil, err
	}

	// Load server certificate
	cert, err := common.LoadCertKeyPair("certs/server-cert.pem", "certs/server-key.pem")
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}, nil
}
