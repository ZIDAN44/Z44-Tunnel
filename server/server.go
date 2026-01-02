package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/yamux"
)

// --- Configuration ---
const (
	TunnelPort       = ":49153"
	PingInterval     = 5 * time.Second
	WriteTimeout     = 10 * time.Second
	HandshakeTimeout = 10 * time.Second
	KeepAlive        = 10 * time.Second
)

// --- State ---
type TunnelServer struct {
	mu            sync.RWMutex
	activeSession *yamux.Session
	listeners     map[int]net.Listener
}

var serverState = &TunnelServer{
	listeners: make(map[int]net.Listener),
}

type Handshake struct {
	Mappings []Mapping `json:"mappings"`
}

type Mapping struct {
	RemotePort int `json:"remote_port"`
}

func main() {
	// Load mTLS Config
	caCert, err := os.ReadFile("certs/ca.pem")
	if err != nil {
		log.Fatal("Missing certs/ca.pem")
	}
	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(caCert)

	cert, err := tls.LoadX509KeyPair("certs/server-cert.pem", "certs/server-key.pem")
	if err != nil {
		log.Fatal("Missing server certs:", err)
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}

	// Start Listener
	ln, err := tls.Listen("tcp", TunnelPort, config)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("🚀 Server ready on %s", TunnelPort)

	// Main Accept Loop
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Accept error:", err)
			time.Sleep(100 * time.Millisecond) // Prevent CPU spike
			continue
		}

		// OS-Level KeepAlive
		if tc, ok := conn.(*tls.Conn).NetConn().(*net.TCPConn); ok {
			tc.SetKeepAlive(true)
			tc.SetKeepAlivePeriod(KeepAlive)
		}

		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	defer conn.Close()
	log.Println("New connection:", conn.RemoteAddr())

	// Yamux Session
	cfg := yamux.DefaultConfig()
	cfg.KeepAliveInterval = PingInterval
	cfg.ConnectionWriteTimeout = WriteTimeout
	cfg.LogOutput = io.Discard

	session, err := yamux.Server(conn, cfg)
	if err != nil {
		return
	}

	// Read Handshake
	stream, err := session.Accept()
	if err != nil {
		return
	}
	var h Handshake
	if err = json.NewDecoder(stream).Decode(&h); err != nil {
		return
	}
	stream.Close()

	// Hot-Swap Active Session
	serverState.mu.Lock()
	if serverState.activeSession != nil {
		serverState.activeSession.Close()
	}
	serverState.activeSession = session

	// Ensure Listeners Exist
	for _, m := range h.Mappings {
		if _, exists := serverState.listeners[m.RemotePort]; !exists {
			l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", m.RemotePort))
			if err == nil {
				serverState.listeners[m.RemotePort] = l
				log.Printf("✅ Forwarding port %d", m.RemotePort)
				go forwardLoop(l, m.RemotePort)
			}
		}
	}
	serverState.mu.Unlock()

	// Block Until Disconnect
	<-session.CloseChan()
	log.Println("⚠️ Client disconnected")

	// Cleanup
	serverState.mu.Lock()
	if serverState.activeSession == session {
		serverState.activeSession = nil
	}
	serverState.mu.Unlock()
}

func forwardLoop(ln net.Listener, port int) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		serverState.mu.RLock()
		sess := serverState.activeSession
		serverState.mu.RUnlock()

		if sess == nil || sess.IsClosed() {
			conn.Close()
			continue
		}

		stream, err := sess.Open()
		if err != nil {
			conn.Close()
			continue
		}

		// Protocol: Send Port -> Wait for "OK" -> Pipe
		fmt.Fprintf(stream, "%d\n", port)

		stream.SetReadDeadline(time.Now().Add(HandshakeTimeout))
		buf := make([]byte, 3)
		if _, err := io.ReadFull(stream, buf); err != nil || string(buf[:2]) != "OK" {
			log.Printf("❌ Zombie detected on port %d. Killing session.", port)
			sess.Close()
			conn.Close()
			stream.Close()
			continue
		}
		stream.SetReadDeadline(time.Time{})

		go func() {
			defer conn.Close()
			defer stream.Close()
			go io.Copy(conn, stream)
			io.Copy(stream, conn)
		}()
	}
}
