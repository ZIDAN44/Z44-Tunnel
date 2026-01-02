package main

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/yamux"
)

// --- Configuration ---
const (
	RetryDelay          = 200 * time.Millisecond
	DialTimeout         = 10 * time.Second
	TCPKeepAlive        = 10 * time.Second
	PingInterval        = 5 * time.Second
	WriteTimeout        = 10 * time.Second
	LocalServiceTimeout = 10 * time.Second
)

type Config struct {
	ServerAddr string    `json:"server_addr"`
	TunnelPort int       `json:"tunnel_port"`
	Mappings   []Mapping `json:"mappings"`
}

type Mapping struct {
	RemotePort int    `json:"remote_port"`
	LocalAddr  string `json:"local_addr"`
}

// Global lookup map for speed
var portMap = make(map[int]string)

func main() {
	// Load Config
	f, err := os.Open("config.json")
	if err != nil {
		log.Fatal("Missing config.json")
	}
	var cfg Config
	if json.NewDecoder(f).Decode(&cfg); err != nil {
		log.Fatal("Invalid config.json")
	}
	f.Close()

	// Optimize: Build Map
	for _, m := range cfg.Mappings {
		portMap[m.RemotePort] = m.LocalAddr
	}

	// Load mTLS
	ca, _ := os.ReadFile("certs/ca.pem")
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(ca)

	cert, _ := tls.LoadX509KeyPair("certs/client-cert.pem", "certs/client-key.pem")
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
		ServerName:   cfg.ServerAddr,
	}

	// Connect Loop
	addr := net.JoinHostPort(cfg.ServerAddr, strconv.Itoa(cfg.TunnelPort))
	for {
		log.Printf("Connecting to %s...", addr)
		runTunnel(addr, tlsConfig, cfg)
		log.Println("Disconnected. Retrying in 200ms...")
		time.Sleep(RetryDelay)
	}
}

func runTunnel(addr string, tlsConfig *tls.Config, cfg Config) {
	d := &net.Dialer{Timeout: DialTimeout, KeepAlive: 10 * time.Second}
	raw, err := d.Dial("tcp", addr)
	if err != nil {
		log.Println("Dial failed:", err)
		return
	}

	conn := tls.Client(raw, tlsConfig)
	if err := conn.Handshake(); err != nil {
		conn.Close()
		return
	}
	log.Println("✅ Connected!")

	// Yamux Config
	yCfg := yamux.DefaultConfig()
	yCfg.KeepAliveInterval = PingInterval
	yCfg.ConnectionWriteTimeout = WriteTimeout
	yCfg.LogOutput = io.Discard

	session, err := yamux.Client(conn, yCfg)
	if err != nil {
		return
	}

	// Send Handshake
	stream, err := session.Open()
	if err != nil {
		return
	}
	json.NewEncoder(stream).Encode(struct {
		Mappings []Mapping `json:"mappings"`
	}{Mappings: cfg.Mappings})
	stream.Close()

	// Handle Streams
	for {
		stream, err := session.Accept()
		if err != nil {
			return
		}
		go handleStream(stream)
	}
}

func handleStream(stream net.Conn) {
	defer stream.Close()

	// Read Requested Port
	rd := bufio.NewReader(stream)
	s, err := rd.ReadString('\n')
	if err != nil {
		return
	}
	port, _ := strconv.Atoi(strings.TrimSpace(s))

	// Fast Lookup
	localAddr, ok := portMap[port]
	if !ok {
		return
	}

	// Dial Local App
	local, err := net.DialTimeout("tcp", localAddr, LocalServiceTimeout)
	if err != nil {
		return
	}

	// Send OK
	if _, err := stream.Write([]byte("OK\n")); err != nil {
		local.Close()
		return
	}

	// Pipe
	go func() {
		defer local.Close()
		io.Copy(local, stream)
	}()
	io.Copy(stream, local)
}
