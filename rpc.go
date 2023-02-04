package coalition

import (
	"bufio"
	"crypto/ed25519"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/json"
	"io"
	"math/big"
	"net"
	"time"
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
	*Peer,
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
		hash := sha256.Sum256(serializedResponse)

		// Prepare the full response payload
		payload := make([]byte, 0)
		payload = append(payload, host.PublicKey()...)
		payload = append(payload, host.Sign(hash[:])...)
		payload = append(payload, serializedResponse...)

		// Return the response
		payloadSizeBuffer := make([]byte, 8)
		big.NewInt(int64(len(payload))).FillBytes(payloadSizeBuffer)
		conn.Write(payloadSizeBuffer)
		conn.Write(payload)
	}()

	// Parse the size of the request payload in bytes
	requestReader := bufio.NewReader(conn)
	payloadSizeBuffer := make([]byte, 8)
	_, err := io.ReadFull(requestReader, payloadSizeBuffer)
	if err != nil {
		response.Data = err.Error()
		return
	}
	payloadSize := new(big.Int).SetBytes(payloadSizeBuffer).Int64()

	// Read the payload from the connection
	payload := make([]byte, payloadSize)
	_, err = io.ReadFull(requestReader, payload)
	if err != nil {
		response.Data = err.Error()
		return
	} else if len(payload) < PeerSignatureSize+1 {
		response.Data = "Incomplete request body"
		return
	}

	// Parse the peer signature and request from the payload
	publicKey := payload[:ed25519.PublicKeySize]
	ecSignature := payload[ed25519.PublicKeySize:PeerSignatureSize]
	peerRequest := payload[PeerSignatureSize:]

	// Verify the peer signature
	hash := sha256.Sum256(peerRequest)
	if !ed25519.Verify(publicKey, hash[:], ecSignature) {
		response.Data = "Invalid peer signature"
		return
	}

	// Parse the peer information and update the host's peer store
	peerKey := sha1.Sum(publicKey)
	peerAddr := conn.RemoteAddr().(*net.TCPAddr)
	_, err = host.RouteTable().Insert(
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
	peer := &Peer{
		peerKey[:],
		peerAddr.IP.To4().String(),
		peerAddr.Port,
		int64(time.Now().Unix()),
	}
	response.Data, err = handler(
		host,
		peer,
		request,
	)
	if err != nil {
		response.Data = err.Error()
		return
	}
	response.Success = true
}
