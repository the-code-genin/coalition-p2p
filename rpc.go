package coalition

import (
	"bufio"
	"crypto/ed25519"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/json"
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

		// Serialize the peer response
		serializedResponse, err := json.Marshal(&response)
		if err != nil {
			return
		}

		// Prepare the peer signature for the serialized response
		hash := sha256.Sum256(serializedResponse)
		signature := make([]byte, 0)
		signature = append(signature, host.PublicKey()...)
		signature = append(signature, host.Sign(hash[:])...)

		// Send the full response payload
		payload := make([]byte, 0)
		payload = append(payload, signature...)
		payload = append(payload, serializedResponse...)
		payload = append(payload, '\n')
		conn.Write(payload)
	}()

	// Read the payload from the connection
	payload, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		response.Data = err.Error()
		return
	} else if len(payload) < PeerSignatureSize {
		response.Data = "Unable to read signature from request payload"
		return
	}

	// Parse the peer signature and request from the payload
	peerSignature := payload[:PeerSignatureSize]
	peerRequest := payload[PeerSignatureSize : len(payload)-1]

	// Verify the peer signature
	hash := sha256.Sum256(peerRequest)
	publicKey := peerSignature[:ed25519.PublicKeySize]
	ecSignature := peerSignature[ed25519.PublicKeySize:]
	if !ed25519.Verify(publicKey, hash[:], ecSignature) {
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
	if err := json.Unmarshal(peerRequest, &request); err != nil {
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
