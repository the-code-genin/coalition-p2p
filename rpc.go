package coalition

import (
	"bufio"
	"crypto/ed25519"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
)

type RPCRequest struct {
	Version int         `json:"version"`
	Method  string      `json:"method"`
	Data    interface{} `json:"data"`
}

type RPCResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
}

type RPCHandlerFunc func(
	*Host,
	[PeerKeySize]byte,
	RPCRequest,
) (interface{}, error)

type RPCHandlerFuncMap map[string]RPCHandlerFunc

func HandleRPCConnection(host *Host, conn net.Conn) {
	response := RPCResponse{
		Success: false,
	}

	// Serialize the response to the connection after execution
	defer func() {
		defer conn.Close()

		// Prepare the response payload
		payload, err := json.Marshal(&response)
		if err != nil {
			return
		}
		payload = append(payload, '\n')

		// Prepare the response signature
		hash := sha256.Sum256(payload)
		payload = append(payload, host.PublicKey()...)
		payload = append(payload, host.Sign(hash[:])...)
		payload = append(payload, '\n')

		conn.Write(payload)
	}()

	// Read the payload from the connection
	payload, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		response.Data = err.Error()
		return
	}
	fmt.Println("Read payload")

	// Read the peer's signature from the connection
	peerSignature, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		fmt.Println(err.Error())
		response.Data = err.Error()
		return
	} else if len(peerSignature) != PeerSignatureSize {
		fmt.Println("Invalid peer signature")
		response.Data = "Invalid peer signature"
		return
	}
	fmt.Println("Read signature")

	// Verify the peer signature
	hash := sha256.Sum256(payload)
	publicKey := peerSignature[:ed25519.PublicKeySize]
	if !ed25519.Verify(publicKey, hash[:], peerSignature[ed25519.PublicKeySize:]) {
		response.Data = "Invalid peer signature"
		return
	}

	// Parse the peer information and Uupdate the host's peer store
	peerKey := sha1.Sum(publicKey)
	peerAddr := conn.RemoteAddr().(*net.TCPAddr)
	_, err = host.store.Insert(
		peerKey[:],
		peerAddr.IP.To4().String(),
		peerAddr.Port,
	)
	if err != nil {
		response.Data = err.Error()
		return
	}

	// Parse the RPC request from the payload
	var request RPCRequest
	if err := json.Unmarshal(payload, &request); err != nil {
		response.Data = err.Error()
		return
	}

	// Get the registered handler for the RPC request
	handler, exists := host.rpcHandlers[request.Method]
	if !exists {
		response.Data = "Unknown RPC method"
		return
	}

	// Handle the RPC request
	response.Data, err = handler(
		host,
		peerKey,
		request,
	)
	if err != nil {
		response.Data = err.Error()
		return
	}
	response.Success = true
}
