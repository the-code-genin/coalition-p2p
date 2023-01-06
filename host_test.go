package framework

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
	host, err := NewHost(port, privKey)
	if err != nil {
		t.Error(err)
	}

	// Ensure host has correct peer ID
	hostID := host.PeerID()
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

	if err = host.Close(); err != nil {
		t.Error(err)
	}
}
