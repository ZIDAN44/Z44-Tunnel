package main

import (
	"crypto/tls"
	"fmt"

	"z44-tunnel/common"
)

// LoadTLSConfig loads the TLS configuration for the client
func LoadTLSConfig(serverAddr string) (*tls.Config, error) {
	// Load CA certificate pool
	pool, err := common.LoadCACertPool("certs/ca.pem")
	if err != nil {
		return nil, err
	}

	// Load client certificate
	cert, err := common.LoadCertKeyPair("certs/client-cert.pem", "certs/client-key.pem")
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
		ServerName:   serverAddr,
	}, nil
}
