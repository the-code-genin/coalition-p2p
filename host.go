package framework

import (
	"fmt"
	"net"
)

type Host struct {
	listener net.Listener
}

// Close the host and any associated resources
func (host *Host) Close() error {
	return host.listener.Close()
}

// Create a new P2P host on the specified host
func NewHost(port int) (*Host, error) {
	// Open listening port at the port
	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return nil, err
	}

	// Create a new host
	host := Host{
		listener,
	}

	return &host, nil
}
