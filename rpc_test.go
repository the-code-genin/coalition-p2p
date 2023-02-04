package coalition

import (
	"bufio"
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
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
	payloadSizeBuffer := make([]byte, 8)
	big.NewInt(int64(len(requestPayload))).FillBytes(payloadSizeBuffer)
	if _, err := conn.Write(payloadSizeBuffer); err != nil {
		t.Error(err)
	}
	if _, err := conn.Write(requestPayload); err != nil {
		t.Error(err)
	}

	// Parse the size of the response payload in bytes
	responseReader := bufio.NewReader(conn)
	payloadSizeBuffer = make([]byte, 8)
	_, err = io.ReadFull(responseReader, payloadSizeBuffer)
	if err != nil {
		t.Error(err)
	}
	payloadSize := new(big.Int).SetBytes(payloadSizeBuffer).Int64()

	// Read response payload
	responsePayload := make([]byte, payloadSize)
	_, err = io.ReadFull(responseReader, responsePayload)
	if err != nil {
		t.Error(err)
	} else if len(responsePayload) < PeerSignatureSize+1 {
		t.Errorf("incomplete response body")
	}

	// Parse the peer signature and request from the payload
	publicKey := responsePayload[:ed25519.PublicKeySize]
	ecSignature := responsePayload[ed25519.PublicKeySize:PeerSignatureSize]
	peerResponse := responsePayload[PeerSignatureSize:]

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
