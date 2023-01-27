package framework

import (
	"bufio"
	"crypto/ed25519"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strconv"
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

type RPCHandlerFunc func(RPCRequest) (interface{}, error)

type RPCHandlerFuncMap map[string]RPCHandlerFunc

type Host struct {
	listener    net.Listener
	key         ed25519.PrivateKey
	rpcHandlers RPCHandlerFuncMap
	closed      bool
}

// Return the listening address
func (host *Host) Address() string {
	return host.listener.Addr().String()
}

// Return the listening port
func (host *Host) Port() (int, error) {
	regExp := regexp.MustCompile(`^.+\:(\d+)$`)
	match := regExp.FindAllStringSubmatch(host.Address(), -1)
	if len(match) != 1 {
		return 0, fmt.Errorf("unable to parse host port")
	} else if len(match[0]) != 2 {
		return 0, fmt.Errorf("unable to parse host port")
	}

	return strconv.Atoi(match[0][1])
}

// Return the host ed25519 public key
func (host *Host) PublicKey() ed25519.PublicKey {
	return host.key.Public().(ed25519.PublicKey)
}

// Sign digest with the host key
func (host *Host) Sign(digest []byte) []byte {
	return ed25519.Sign(host.key, digest)
}

// Returns the 160-bit hash of the public key as the peer ID
func (host *Host) PeerKey() [PeerKeySize]byte {
	pk := host.key.Public().(ed25519.PublicKey)
	return sha1.Sum([]byte(pk))
}

// Start listening for connections on the specified port for RPC requests
func (host *Host) Listen() {
	for !host.closed {
		conn, err := host.listener.Accept()
		if err != nil {
			continue
		}

		go func(conn net.Conn) {
			response := RPCResponse{
				Success: false,
			}
			defer func() {
				defer conn.Close()

				// Prepare the response payload
				payload, err := json.Marshal(&response)
				if err != nil {
					return
				}

				payload = append(payload, '\n')
				conn.Write(payload)
			}()

			// Read data from the connection
			data, err := bufio.NewReader(conn).ReadBytes('\n')
			if err != nil {
				response.Data = err.Error()
				return
			}

			// Parse the RPC message
			var request RPCRequest
			if err := json.Unmarshal(data, &request); err != nil {
				response.Data = err.Error()
				return
			}

			// Get the handler for the RPC message
			handler, exists := host.rpcHandlers[request.Method]
			if !exists {
				response.Data = "Unknown RPC method"
				return
			}

			// Process the response
			response.Data, err = handler(request)
			if err != nil {
				response.Data = err.Error()
				return
			}
			response.Success = true
		}(conn)
	}
}

// Close the host and any associated resources
func (host *Host) Close() {
	host.closed = true
	host.listener.Close()
}

// Create a new P2P host on the specified port with the Ed25519 private key
func NewHost(
	port int,
	key ed25519.PrivateKey,
	rpcHandlers RPCHandlerFuncMap,
) (*Host, error) {
	// Start listening on the port
	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return nil, err
	}

	// Create a new host
	host := &Host{
		listener,
		key,
		rpcHandlers,
		false,
	}

	return host, nil
}
