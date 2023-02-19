package coalition

import (
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
	_, hostPrivKey, err := ed25519.GenerateKey(rand.Reader)
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
	addrs, err := host.Addresses()
	if err != nil {
		t.Error(err)
	}
	_, ip4Address, port, err := ParseNodeAddress(addrs[0])
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

	// Prepare the full request payload
	requestPayload := make([]byte, 0)
	requestPayload = append(requestPayload, clientPubKey...)
	requestPayload = append(requestPayload, ed25519.Sign(clientPrivKey, hash[:])...)
	requestPayload = append(requestPayload, serializedRequest...)

	// Send the request
	if err := WriteToConn(conn, requestPayload); err != nil {
		t.Error(err)
	}

	// Read response payload
	responsePayload, err := ReadFromConn(conn)
	if err != nil {
		t.Error(err)
	} else if len(responsePayload) <= PeerSignatureSize {
		t.Errorf("incomplete response body")
	}

	// Parse the peer signature and request from the payload
	peerSignature := responsePayload[:PeerSignatureSize]
	peerResponse := responsePayload[PeerSignatureSize:]

	// Verify the peer signature
	responseHash := sha256.Sum256(peerResponse)
	peerKey, err := RecoverPeerKeyFromPeerSignature(peerSignature, responseHash[:])
	hostPeerKey := host.PeerKey()
	if err != nil {
		t.Error(err)
	} else if !bytes.Equal(peerKey, hostPeerKey[:]) {
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
