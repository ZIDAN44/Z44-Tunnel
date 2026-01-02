package common

import (
	"crypto/tls"
	"io"
	"log"
	"net"
	"time"

	"github.com/hashicorp/yamux"
)

// CloseConn safely closes a connection, logging errors
func CloseConn(conn net.Conn) {
	if conn != nil {
		if err := conn.Close(); err != nil {
			log.Printf("Warning: failed to close connection: %v", err)
		}
	}
}

// CloseListener safely closes a listener, logging errors
func CloseListener(ln net.Listener) {
	if ln != nil {
		if err := ln.Close(); err != nil {
			log.Printf("Warning: failed to close listener: %v", err)
		}
	}
}

// CloseSession safely closes a yamux session, logging errors
func CloseSession(session interface{ Close() error }) {
	if session != nil {
		if err := session.Close(); err != nil {
			log.Printf("Warning: failed to close session: %v", err)
		}
	}
}

// YamuxConfig returns a configured yamux config
func YamuxConfig(keepAlive, writeTimeout time.Duration) *yamux.Config {
	cfg := yamux.DefaultConfig()
	cfg.KeepAliveInterval = keepAlive
	cfg.ConnectionWriteTimeout = writeTimeout
	cfg.LogOutput = io.Discard
	return cfg
}

// SetKeepAlive sets TCP keepalive on a TLS connection
func SetKeepAlive(conn net.Conn, period time.Duration) {
	if tlsConn, ok := conn.(*tls.Conn); ok {
		if netConn := tlsConn.NetConn(); netConn != nil {
			if tc, ok := netConn.(*net.TCPConn); ok {
				tc.SetKeepAlive(true)
				tc.SetKeepAlivePeriod(period)
			}
		}
	}
}
