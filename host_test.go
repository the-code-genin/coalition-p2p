package framework

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

func TestNewHost(t *testing.T) {
	// Generate a key pair
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Error(err)
	}

	// Create a new host
	host, err := NewHost(3000, key)
	if err != nil {
		t.Error(err)
	}

	if err = host.Close(); err != nil {
		t.Error(err)
	}
}
