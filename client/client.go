package main

import (
	"log"
	"net"
	"os"
	"strconv"
)

func main() {
	// Recover from panics
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Fatal panic recovered: %v", r)
			os.Exit(1)
		}
	}()

	// Load configuration
	cfg, err := LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Build port map
	portMap := BuildPortMap(cfg.Mappings)

	// Load TLS configuration
	tlsConfig, err := LoadTLSConfig(cfg.ServerAddr)
	if err != nil {
		log.Fatalf("Failed to load TLS configuration: %v", err)
	}

	// Create tunnel address
	addr := net.JoinHostPort(cfg.ServerAddr, strconv.Itoa(cfg.TunnelPort))

	// Create and run tunnel
	tunnel := NewTunnel(addr, tlsConfig, cfg, portMap)
	tunnel.Run()
}
