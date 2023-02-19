package coalition

import (
	"crypto/sha256"
	"encoding/json"
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
	} else if len(payload) <= PeerSignatureSize {
		response.Data = "Incomplete request body"
		return
	}

	// Parse the peer signature and request from the payload
	peerSignature := payload[:PeerSignatureSize]
	peerRequest := payload[PeerSignatureSize:]

	// Verify the peer signature
	requestHash := sha256.Sum256(peerRequest)
	peerKey, err := RecoverPeerKeyFromPeerSignature(peerSignature, requestHash[:])
	if err != nil {
		response.Data = err.Error()
		return
	}

	// Parse the peer information and update the host's peer store
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
