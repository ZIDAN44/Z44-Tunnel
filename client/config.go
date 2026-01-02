package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"

	"z44-tunnel/common"
)

// Config represents the client configuration
type Config struct {
	ServerAddr string           `json:"server_addr"`
	TunnelPort int              `json:"tunnel_port"`
	Mappings   []common.Mapping `json:"mappings"`
}

// LoadConfig loads and validates the configuration from config.json
func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// validateConfig validates the configuration values
func validateConfig(cfg Config) error {
	if cfg.ServerAddr == "" {
		return fmt.Errorf("server_addr cannot be empty")
	}
	if !common.ValidatePort(cfg.TunnelPort) {
		return fmt.Errorf("tunnel_port must be between 1 and 65535, got %d", cfg.TunnelPort)
	}
	if len(cfg.Mappings) == 0 {
		return fmt.Errorf("mappings cannot be empty")
	}
	for i, m := range cfg.Mappings {
		if !common.ValidatePort(m.RemotePort) {
			return fmt.Errorf("mapping[%d]: invalid remote_port %d", i, m.RemotePort)
		}
		if m.LocalAddr == "" {
			return fmt.Errorf("mapping[%d]: local_addr cannot be empty", i)
		}
		if _, _, err := net.SplitHostPort(m.LocalAddr); err != nil {
			return fmt.Errorf("mapping[%d]: invalid local_addr '%s': %w", i, m.LocalAddr, err)
		}
	}
	return nil
}

// BuildPortMap creates a lookup map from remote port to local address
func BuildPortMap(mappings []common.Mapping) map[int]string {
	portMap := make(map[int]string)
	for _, m := range mappings {
		portMap[m.RemotePort] = m.LocalAddr
	}
	return portMap
}
