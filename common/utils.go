package common

import (
	"crypto/tls"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/yamux"
)

// isClosedError checks if error is from closing an already-closed connection
func isClosedError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "use of closed network connection")
}

// CloseConn safely closes a connection, logging errors
func CloseConn(conn net.Conn) {
	if conn != nil {
		if err := conn.Close(); err != nil && !isClosedError(err) {
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
		if err := session.Close(); err != nil && !isClosedError(err) {
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
	cfg.AcceptBacklog = 256 // Limit pending stream accepts
	return cfg
}

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	mu         sync.Mutex
	tokens     int
	maxTokens  int
	refillRate time.Duration
	lastRefill time.Time
}

// NewRateLimiter creates a new rate limiter
// maxTokens: maximum tokens in bucket
// refillRate: time between token refills (1 token per refillRate)
func NewRateLimiter(maxTokens int, refillRate time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a token is available and consumes it
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)

	// Refill tokens based on elapsed time
	if elapsed > 0 {
		tokensToAdd := int(elapsed / rl.refillRate)
		if tokensToAdd > 0 {
			rl.tokens = min(rl.tokens+tokensToAdd, rl.maxTokens)
			rl.lastRefill = rl.lastRefill.Add(time.Duration(tokensToAdd) * rl.refillRate)
		}
	}

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}
	return false
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
