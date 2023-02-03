package coalition

import (
	"bufio"
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"testing"
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
		Identity(hostPrivKey),
	)
	if err != nil {
		t.Error(err)
	}
	defer host.Close()
	go host.Listen()

	// Dial host
	address, err := host.Address()
	if err != nil {
		t.Error(err)
	}
	_, ip4Address, port, err := ParseNodeAddress(address)
	if err != nil {
		t.Error(err)
	}
	conn, err := net.Dial("tcp4", fmt.Sprintf("%s:%d", ip4Address, port))
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
		Method:  PingMethod,
		Data:    nil,
	})
	if err != nil {
		t.Error(err)
	}
	hash := sha256.Sum256(serializedRequest)

	// Send the full request payload
	requestPayload := make([]byte, 0)
	requestPayload = append(requestPayload, clientPubKey...)
	requestPayload = append(requestPayload, ed25519.Sign(clientPrivKey, hash[:])...)
	requestPayload = append(requestPayload, serializedRequest...)
	requestPayload = append(requestPayload, '\n')
	if i, err := conn.Write(requestPayload); err != nil {
		t.Error(err)
	} else if i != len(requestPayload) {
		t.Errorf("Unable to send full request body")
	}

	// Read response payload
	responsePayload, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		t.Error(err)
	} else if len(responsePayload) < PeerSignatureSize+1 {
		t.Errorf("Unable to read signature from response payload")
	}

	// Parse the peer signature and request from the payload
	publicKey := responsePayload[:ed25519.PublicKeySize]
	ecSignature := responsePayload[ed25519.PublicKeySize:PeerSignatureSize]
	peerResponse := responsePayload[PeerSignatureSize : len(responsePayload)-1]

	// Verify the peer signature
	hash = sha256.Sum256(peerResponse)
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
	} else if response.Data.(string) != PingResponse {
		t.Error("expected pong response from host")
	}
}
