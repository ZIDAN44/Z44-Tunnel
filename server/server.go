package main

import (
	"crypto/tls"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"z44-tunnel/common"

	"github.com/hashicorp/yamux"
)

// Server constants
const (
	TunnelPort           = ":49153"
	PingInterval         = 5 * time.Second
	WriteTimeout         = 10 * time.Second
	HandshakeTimeout     = 10 * time.Second
	KeepAlive            = 10 * time.Second
	MaxConcurrentStreams = 1000                  // Maximum concurrent streams per session
	StreamRateLimit      = 100                   // Maximum tokens in bucket
	StreamRefillRate     = 10 * time.Millisecond // Refill rate (100 streams/sec max)
)

// TunnelServer manages the server state
type TunnelServer struct {
	mu            sync.RWMutex
	activeSession *yamux.Session
	listeners     map[int]net.Listener
	streamCount   int
	rateLimiter   *common.RateLimiter
}

// NewTunnelServer creates a new tunnel server instance
func NewTunnelServer() *TunnelServer {
	return &TunnelServer{
		listeners:   make(map[int]net.Listener),
		rateLimiter: common.NewRateLimiter(StreamRateLimit, StreamRefillRate),
	}
}

// SetActiveSession sets the active yamux session
func (s *TunnelServer) SetActiveSession(session *yamux.Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeSession != nil {
		s.activeSession.Close()
	}
	s.activeSession = session
	s.streamCount = 0
}

// GetActiveSession returns the active yamux session
func (s *TunnelServer) GetActiveSession() *yamux.Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.activeSession
}

// ClearActiveSession clears the active session if it matches
func (s *TunnelServer) ClearActiveSession(session *yamux.Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeSession == session {
		s.activeSession = nil
		s.streamCount = 0
	}
}

// IncrementStreamCount increments the stream count if under limit
func (s *TunnelServer) IncrementStreamCount() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.streamCount >= MaxConcurrentStreams {
		return false
	}
	s.streamCount++
	return true
}

// DecrementStreamCount decrements the stream count
func (s *TunnelServer) DecrementStreamCount() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.streamCount > 0 {
		s.streamCount--
	}
}

// AddListener adds a listener for a port
func (s *TunnelServer) AddListener(port int, listener net.Listener) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners[port] = listener
}

// HasListener checks if a listener exists for a port
func (s *TunnelServer) HasListener(port int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.listeners[port]
	return exists
}

// Handshake is an alias for common.Handshake
type Handshake = common.Handshake

func main() {
	// Recover from panics
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Fatal panic recovered: %v", r)
			os.Exit(1)
		}
	}()

	// Load TLS configuration
	tlsConfig, err := LoadTLSConfig()
	if err != nil {
		log.Fatalf("Failed to load TLS configuration: %v", err)
	}

	// Start TLS listener
	ln, err := tls.Listen("tcp", TunnelPort, tlsConfig)
	if err != nil {
		log.Fatalf("Failed to start TLS listener on %s: %v", TunnelPort, err)
	}
	defer common.CloseListener(ln)

	log.Printf("🚀 Server ready on %s", TunnelPort)

	// Create server instance
	server := NewTunnelServer()

	// Main accept loop
	for {
		conn, err := ln.Accept()
		if err != nil {
			// Check if listener is closed
			if netErr, ok := err.(net.Error); ok && !netErr.Temporary() {
				log.Printf("Listener closed, shutting down: %v", err)
				return
			}
			log.Printf("Accept error: %v", err)
			time.Sleep(100 * time.Millisecond) // Prevent CPU spike
			continue
		}

		common.SetKeepAlive(conn, KeepAlive)
		go func(c net.Conn) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Panic in handleClient: %v", r)
				}
			}()
			handleClient(c, server)
		}(conn)
	}
}
