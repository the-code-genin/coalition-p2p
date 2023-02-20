package coalition

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
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

		// Prepare response signature
		responseHash := sha256.Sum256(serializedResponse)
		responseSignature, err := host.Sign(responseHash[:])
		if err != nil {
			return
		}

		// Prepare the full response payload
		payload := make([]byte, 0)
		payload = append(payload, responseSignature[:]...)
		payload = append(payload, serializedResponse...)

		// Return the response
		WriteToConn(conn, payload)
	}()

	// Read the payload from the connection
	payload, err := ReadFromConn(conn)
	if err != nil {
		response.Data = err.Error()
		return
	} else if len(payload) <= Int64Len+PeerSignatureSize {
		response.Data = "Incomplete request body"
		return
	}

	// Parse the peer listening port, signature and request from the payload
	peerPort := BytesToInt64(payload[:Int64Len])
	peerSignature := payload[Int64Len : Int64Len+PeerSignatureSize]
	peerRequest := payload[Int64Len+PeerSignatureSize:]

	// Verify the peer signature and recover the peer key
	requestHash := sha256.Sum256(peerRequest)
	peerKey, err := RecoverPeerKeyFromPeerSignature(peerSignature, requestHash[:])
	if err != nil {
		response.Data = err.Error()
		return
	}

	// Parse the remote peer details
	peer := &Peer{
		peerKey[:],
		conn.RemoteAddr().(*net.TCPAddr).IP.To4().String(),
		int(peerPort),
		int64(time.Now().Unix()),
	}

	// Attempt to connect to the peer to ensure the peer can accept RPC requests
	tmpConn, err := net.Dial("tcp4", fmt.Sprintf("%s:%d", peer.IPAddress(), peer.Port()))
	if err == nil {
		tmpConn.Close()

		// Update the host's peer store
		_, err := host.RouteTable().Insert(
			peer.Key(),
			peer.IPAddress(),
			peer.Port(),
		)
		if err != nil {
			response.Data = err.Error()
			return
		}
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
		peer,
		request,
	)
	if err != nil {
		response.Data = err.Error()
		return
	}
	response.Success = true
}
