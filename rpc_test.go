package coalition

import (
	"bufio"
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"net"
	"testing"
	"time"
)

func TestRPCServer(t *testing.T) {
	// Generate a key pair for the host
	hostPubKey, hostPrivKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Error(err)
	}

	// Create a new host on port 3000
	host, err := NewHost(
		3001, // Port
		hostPrivKey,
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
	go host.Listen()
	defer host.Close()

	// Dial server
	address, err := host.Address()
	if err != nil {
		t.Error(err)
	}
	conn, err := net.Dial("tcp4", address)
	if err != nil {
		t.Error(err)
	}
	defer conn.Close()

	// Generate a key pair for the client
	clientPubKey, clientPrivKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Error(err)
	}

	// Prepare serialized ping request
	serializedRequest, err := json.Marshal(&RPCRequest{
		Version: 1,
		Method:  "ping",
		Data:    nil,
	})
	if err != nil {
		t.Error(err)
	}

	// Prepare the peer signature for the serialized request
	hash := sha256.Sum256(serializedRequest)
	signature := make([]byte, 0)
	signature = append(signature, clientPubKey...)
	signature = append(signature, ed25519.Sign(clientPrivKey, hash[:])...)

	// Send the full request payload
	requestPayload := make([]byte, 0)
	requestPayload = append(requestPayload, signature...)
	requestPayload = append(requestPayload, serializedRequest...)
	requestPayload = append(requestPayload, '\n')
	if _, err = conn.Write(requestPayload); err != nil {
		t.Error(err)
	}

	// Read response payload
	responsePayload, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		t.Error(err)
	} else if len(responsePayload) < PeerSignatureSize {
		t.Errorf("Unable to read signature from response payload")
	}

	// Parse the peer signature and response from the response payload
	peerSignature := responsePayload[:PeerSignatureSize]
	peerResponse := responsePayload[PeerSignatureSize : len(responsePayload)-1]

	// Verify the peer signature
	hash = sha256.Sum256(peerResponse)
	publicKey := peerSignature[:ed25519.PublicKeySize]
	ecSignature := peerSignature[ed25519.PublicKeySize:]
	if !ed25519.Verify(publicKey, hash[:], ecSignature) {
		t.Errorf("Invalid peer signature")
	} else if !bytes.Equal(hostPubKey, publicKey) {
		t.Errorf("Response payload not signed by host")
	}

	// Parse the RPC response from the payload
	var response RPCResponse
	if err = json.Unmarshal(peerResponse, &response); err != nil {
		t.Error(err)
	} else if !response.Success {
		t.Error(response.Data.(string))
	} else if response.Data.(string) != "pong" {
		t.Error("expected pong response from host")
	}
}
