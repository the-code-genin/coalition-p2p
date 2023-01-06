package framework

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"testing"
)

func TestNewHost(t *testing.T) {
	// Generate a key pair
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Error(err)
	}

	// Create a new host
	host, err := NewHost(3000, privKey)
	if err != nil {
		t.Error(err)
	}

	hostID := host.PeerID()
	expectedID := sha256.Sum256(pubKey)
	if !bytes.Equal(expectedID[:], hostID[:]) {
		t.Errorf("unexpected host ID")
	}

	if err = host.Close(); err != nil {
		t.Error(err)
	}
}
