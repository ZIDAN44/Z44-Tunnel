package main

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"log"
	"net"
	"time"

	"z44-tunnel/common"

	"github.com/hashicorp/yamux"
)

// Tunnel constants
const (
	RetryDelay          = 200 * time.Millisecond
	DialTimeout         = 10 * time.Second
	TCPKeepAlive        = 10 * time.Second
	PingInterval        = 5 * time.Second
	WriteTimeout        = 10 * time.Second
	LocalServiceTimeout = 10 * time.Second
)

// Tunnel manages the connection to the server
type Tunnel struct {
	addr      string
	tlsConfig *tls.Config
	cfg       *Config
	portMap   map[int]string
}

// NewTunnel creates a new tunnel instance
func NewTunnel(addr string, tlsConfig *tls.Config, cfg *Config, portMap map[int]string) *Tunnel {
	return &Tunnel{
		addr:      addr,
		tlsConfig: tlsConfig,
		cfg:       cfg,
		portMap:   portMap,
	}
}

// Run starts the tunnel connection loop
func (t *Tunnel) Run() {
	for {
		log.Printf("Connecting to %s...", t.addr)
		t.connect()
		log.Println("Disconnected. Retrying in 200ms...")
		time.Sleep(RetryDelay)
	}
}

// connect establishes a connection to the server
func (t *Tunnel) connect() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in connect: %v", r)
		}
	}()

	raw, err := (&net.Dialer{Timeout: DialTimeout, KeepAlive: TCPKeepAlive}).Dial("tcp", t.addr)
	if err != nil {
		log.Printf("Failed to dial %s: %v", t.addr, err)
		return
	}
	defer common.CloseConn(raw)

	conn := tls.Client(raw, t.tlsConfig)
	if err := conn.Handshake(); err != nil {
		log.Printf("TLS handshake failed: %v", err)
		common.CloseConn(conn)
		return
	}
	log.Println("✅ Connected!")

	session, err := yamux.Client(conn, common.YamuxConfig(PingInterval, WriteTimeout))
	if err != nil {
		log.Printf("Failed to create yamux session: %v", err)
		return
	}
	defer common.CloseSession(session)

	if err := t.sendHandshake(session); err != nil {
		log.Printf("Failed to send handshake: %v", err)
		return
	}

	for {
		stream, err := session.Accept()
		if err != nil {
			if err != io.EOF && err.Error() != "keepalive timeout" {
				log.Printf("Failed to accept stream: %v", err)
			}
			return
		}
		go func(s net.Conn) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Panic in handleStream: %v", r)
				}
			}()
			handleStream(s, t.portMap)
		}(stream)
	}
}

// sendHandshake sends the initial handshake to the server
func (t *Tunnel) sendHandshake(session *yamux.Session) error {
	stream, err := session.Open()
	if err != nil {
		return err
	}
	defer common.CloseConn(stream)

	return json.NewEncoder(stream).Encode(struct {
		Mappings []common.Mapping `json:"mappings"`
	}{Mappings: t.cfg.Mappings})
}
