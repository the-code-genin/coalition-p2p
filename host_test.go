package coalition

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha1"
	"testing"
)

func TestNewHost(t *testing.T) {
	// Generate a key pair
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Error(err)
	}

	// Create a new host
	port := 3000
	host, err := NewHost(
		Port(port),
		Identity(privKey),
	)
	if err != nil {
		t.Error(err)
	}
	defer host.Close()

	// Ensure host has correct peer ID
	hostID := host.PeerKey()
	expectedID := sha1.Sum(pubKey)
	if !bytes.Equal(expectedID[:], hostID[:]) {
		t.Errorf("unexpected host ID")
	}

	// Ensure host is listening on the right port
	hostPort, err := host.Port()
	if err != nil {
		t.Error(err)
	} else if hostPort != port {
		t.Errorf("unexpected host port")
	}
}

func TestConnection(t *testing.T) {
	// Create hostA
	hostA, err := NewHost()
	if err != nil {
		t.Error(err)
	}
	go hostA.Listen()
	defer hostA.Close()

	// Create hostB
	hostB, err := NewHost()
	if err != nil {
		t.Error(err)
	}
	go hostB.Listen()
	defer hostB.Close()

	if len(hostA.RouteTable().Peers()) != 0 {
		t.Errorf("Host A should not have any peers")
	} else if len(hostB.RouteTable().Peers()) != 0 {
		t.Errorf("Host B should not have any peers")
	}

	// Send a ping message to hostA from hostB
	addrs, err := hostA.Addresses()
	if err != nil {
		t.Error(err)
	}
	if err := hostB.Ping(addrs[0]); err != nil {
		t.Error(err)
	}

	if len(hostA.RouteTable().Peers()) != 1 {
		t.Errorf("Host A should have one peer")
	} else if len(hostB.RouteTable().Peers()) != 1 {
		t.Errorf("Host B should have one peer")
	}
}
