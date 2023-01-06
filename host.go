package framework

import (
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"
	"net"
	"regexp"
	"strconv"
)

type Host struct {
	listener net.Listener
	key      ed25519.PrivateKey
}

// Return the listening address
func (host *Host) Address() string {
	return host.listener.Addr().String()
}

func (host *Host) Port() (int, error) {
	regExp := regexp.MustCompile(`^.+\:(\d+)$`)
	match := regExp.FindAllStringSubmatch(host.Address(), -1)
	if len(match) != 1 {
		return 0, fmt.Errorf("unable to parse host port")
	} else if len(match[0]) != 2 {
		return 0, fmt.Errorf("unable to parse host port")
	}

	return strconv.Atoi(match[0][1])
}

// Returns the 256-bit hash of the public key as the peer ID
func (host *Host) PeerID() [32]byte {
	pk := host.key.Public().(ed25519.PublicKey)
	return sha256.Sum256([]byte(pk))
}

// Close the host and any associated resources
func (host *Host) Close() error {
	return host.listener.Close()
}

// Create a new P2P host on the specified port with the Ed25519 private key
func NewHost(
	port int,
	key ed25519.PrivateKey,
) (*Host, error) {
	// Start listening on the port
	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return nil, err
	}

	// Create a new host
	host := Host{
		listener,
		key,
	}

	return &host, nil
}
