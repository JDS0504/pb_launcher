package networktools

import (
	"fmt"
	"net"
)

// GetAvailablePort tries to bind to a random available port on the given IP address segment.
// Example: ipSegment = "127.0.0.2" → returns ("127.0.0.2", 49231, nil)
func GetAvailablePort(ipAddress string) (string, int, error) {
	if net.ParseIP(ipAddress) == nil {
		return "", 0, fmt.Errorf("invalid IP address: %s", ipAddress)
	}

	listener, err := net.Listen("tcp", ipAddress+":0")
	if err != nil {
		return "", 0, fmt.Errorf("failed to bind to %s: %w", ipAddress, err)
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return ipAddress, addr.Port, nil
}

// IsPortAvailable checks if a specific port on a given IP address is free.
func IsPortAvailable(ipAddress string, port int) bool {
	if net.ParseIP(ipAddress) == nil {
		return false
	}
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", ipAddress, port))
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

