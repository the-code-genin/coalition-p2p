package framework

import (
	"bufio"
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha1"
	"encoding/json"
	"net"
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
		port,
		privKey,
		map[string]RPCHandlerFunc{},
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

func TestRPCServer(t *testing.T) {
	// Generate a key pair
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Error(err)
	}

	// Create a new host on port 3000
	host, err := NewHost(
		3000,
		privKey,
		RPCHandlerFuncMap{
			"ping": func(RPCRequest) (interface{}, error) {
				return "pong", nil
			},
		},
	)
	if err != nil {
		t.Error(err)
	}
	go host.Listen()
	defer host.Close()

	// Dail server
	conn, err := net.Dial("tcp", host.Address())
	if err != nil {
		t.Error(err)
	}
	defer conn.Close()

	// Send ping request
	data, err := json.Marshal(&RPCRequest{
		Version: 1,
		Method:  "ping",
		Data:    nil,
	})
	if err != nil {
		t.Error(err)
	}
	data = append(data, '\n')
	if _, err = conn.Write(data); err != nil {
		t.Error(err)
	}

	// Parse response
	rawResponse, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		t.Error(err)
	}
	var response RPCResponse
	if err = json.Unmarshal(rawResponse, &response); err != nil {
		t.Error(err)
	} else if !response.Success {
		t.Error(response.Data.(string))
	} else if response.Data.(string) != "pong" {
		t.Error("expected pong")
	}
}
