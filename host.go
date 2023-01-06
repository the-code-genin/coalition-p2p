package framework

import (
	"crypto/ed25519"
	"crypto/sha1"
	"fmt"
	"net"
	"regexp"
	"strconv"
)

type Host struct {
	listener net.Listener
	key      ed25519.PrivateKey
	closed   bool
}

// Return the listening address
func (host *Host) Address() string {
	return host.listener.Addr().String()
}

// Return the listening port
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

// Return the host ed25519 public key
func (host *Host) PublicKey() ed25519.PublicKey {
	return host.key.Public().(ed25519.PublicKey)
}

// Sign digest with the host key
func (host *Host) Sign(digest []byte) []byte {
	return ed25519.Sign(host.key, digest)
}

// Returns the 160-bit hash of the public key as the peer ID
func (host *Host) PeerID() [sha1.Size]byte {
	pk := host.key.Public().(ed25519.PublicKey)
	return sha1.Sum([]byte(pk))
}

// Start listening for connections on the specified port for RPC requests
func (host *Host) Listen(conns chan net.Conn) {
	defer close(conns)

	for !host.closed {
		conn, err := host.listener.Accept()
		if err != nil {
			continue
		}
		conns <- conn
	}
}

// Close the host and any associated resources
func (host *Host) Close() {
	host.closed = true
	host.listener.Close()
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
	host := &Host{
		listener,
		key,
		false,
	}

	return host, nil
}
