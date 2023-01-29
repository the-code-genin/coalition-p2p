package coalition

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha1"
	"testing"
	"time"
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
		port,
		privKey,
		RPCHandlerFuncMap{},
		20,                                 // Max peers
		3,                                  // Max concurrent requests
		int64(time.Hour.Seconds()),         // LatencyPeriod
		int64((time.Minute * 5).Seconds()), // PingPeriod
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
	// Create hostA listening on port 3000
	_, privKeyA, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Error(err)
	}
	portA := 3002
	hostA, err := NewHost(
		portA,
		privKeyA,
		RPCHandlerFuncMap{
			"ping": func(*Host, [PeerKeySize]byte, RPCRequest) (interface{}, error) {
				return "pong", nil
			},
		},
		20,                                 // Max peers
		3,                                  // Max concurrent requests
		int64(time.Hour.Seconds()),         // LatencyPeriod
		int64((time.Minute * 5).Seconds()), // PingPeriod
	)
	if err != nil {
		t.Error(err)
	}
	go hostA.Listen()
	defer hostA.Close()

	// Create hostB listening on port 3001
	_, privKeyB, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Error(err)
	}
	portB := 3003
	hostB, err := NewHost(
		portB,
		privKeyB,
		RPCHandlerFuncMap{
			"ping": func(*Host, [PeerKeySize]byte, RPCRequest) (interface{}, error) {
				return "pong", nil
			},
		},
		20,                                 // Max peers
		3,                                  // Max concurrent requests
		int64(time.Hour.Seconds()),         // LatencyPeriod
		int64((time.Minute * 5).Seconds()), // PingPeriod
	)
	if err != nil {
		t.Error(err)
	}
	go hostB.Listen()
	defer hostB.Close()

	if len(hostA.Peers()) != 0 {
		t.Errorf("Host A should not have any peers")
	} else if len(hostB.Peers()) != 0 {
		t.Errorf("Host B should not have any peers")
	}

	// Send a ping message to hostA from hostB
	hostAddressA, err := hostA.Address()
	if err != nil {
		t.Error(err)
	}
	_, err = hostB.SendMessage(hostAddressA, 1, "ping", nil)
	if err != nil {
		t.Error(err)
	}

	if len(hostA.Peers()) != 1 {
		t.Errorf("Host A should have one peer")
	} else if len(hostB.Peers()) != 1 {
		t.Errorf("Host B should have one peer")
	}
}
