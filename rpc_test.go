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
		3000, // Port
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
	fmt.Println("Dial")

	// Generate a key pair for the client
	clientPubKey, clientPrivKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Error(err)
	}

	// Prepare ping request
	payload, err := json.Marshal(&RPCRequest{
		Version: 1,
		Method:  "ping",
		Data:    nil,
	})
	if err != nil {
		t.Error(err)
	}
	payload = append(payload, '\n')

	// Prepare the ping peer signature
	hash := sha256.Sum256(payload)
	payload = append(payload, clientPubKey...)
	payload = append(payload, ed25519.Sign(clientPrivKey, hash[:])...)
	payload = append(payload, '\n')

	// Serialize the payload to the connection
	if _, err = conn.Write(payload); err != nil {
		t.Error(err)
	}
	fmt.Println("Serialized")

	// Read response payload
	responsePayload, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		t.Error(err)
	}

	// Read and verify the response peer signature
	responseSignature, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		t.Error(err)
	} else if len(responseSignature) != PeerSignatureSize {
		t.Errorf("invalid response signature length from host")
	} else if !bytes.Equal(responseSignature[:ed25519.PublicKeySize], hostPubKey) {
		t.Errorf("invalid response signature public key from host")
	} else if !ed25519.Verify(responseSignature[:ed25519.PublicKeySize], hash[:], responseSignature[ed25519.PublicKeySize:]) {
		t.Errorf("invalid response signature from host")
	}

	var response RPCResponse
	if err = json.Unmarshal(responsePayload, &response); err != nil {
		t.Error(err)
	} else if !response.Success {
		t.Error(response.Data.(string))
	} else if response.Data.(string) != "pong" {
		t.Error("expected pong response from host")
	}
}
