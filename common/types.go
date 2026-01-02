package common

// Mapping represents a port mapping
// LocalAddr is optional and only used by client
type Mapping struct {
	RemotePort int    `json:"remote_port"`
	LocalAddr  string `json:"local_addr,omitempty"`
}

// Handshake represents the client handshake data
type Handshake struct {
	Mappings []Mapping `json:"mappings"`
}

// ValidatePort validates that a port is in the valid range
func ValidatePort(port int) bool {
	return port > 0 && port <= 65535
}
