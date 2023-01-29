package coalition

import (
	"bufio"
	"crypto/ed25519"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"net"
)

// Represents a basic p2p node with an optimized kbucket peer store
type Host struct {
	listener           net.Listener
	store              *PeerStore
	key                ed25519.PrivateKey
	rpcHandlers        RPCHandlerFuncMap
	closed             bool
	maxPeers           int64
	concurrentRequests int64
	pingPeriod         int64
	latencyPeriod      int64
}

// Return the listening IPv4 address
func (host *Host) IPAddress() (string, error) {
	tcpAddr, ok := host.listener.Addr().(*net.TCPAddr)
	if !ok {
		return "", fmt.Errorf("unable to parse host port")
	}
	return tcpAddr.IP.To4().String(), nil
}

// Return the listening tcp port
func (host *Host) Port() (int, error) {
	tcpAddr, ok := host.listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("unable to parse host port")
	}
	return tcpAddr.Port, nil
}

// Return the host's fully qualified tcp address i.e tcp ipv4:port
func (host *Host) Address() (string, error) {
	address, err := host.IPAddress()
	if err != nil {
		return "", nil
	}

	port, err := host.Port()
	if err != nil {
		return "", nil
	}

	return fmt.Sprintf("%s:%d", address, port), nil
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

// Returns the host's connected peers
func (host *Host) Peers() []*Peer {
	return host.store.Peers()
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

// Create a new P2P host on the specified ipv4 port with the Ed25519 private key
// port: TCP port to listen for RPC requests
// key: ed25519 private key to sign messages
// rpcHandlers: RPC request handlers
// maxPeers: The kbucket replication paramter
// concurrentRequests: The kademlia concurrent requests paramter
// latencyPeriod: The amount of time in seconds to check if a node is still online
// pingPeriod: The ping period in seconds to check if nodes are alive, should be < latencyPeriod
func NewHost(
	port int,
	key ed25519.PrivateKey,
	rpcHandlers RPCHandlerFuncMap,
	maxPeers int64,
	concurrentRequests int64,
	latencyPeriod int64,
	pingPeriod int64,
) (*Host, error) {
	if pingPeriod >= latencyPeriod {
		return nil, fmt.Errorf("ping period should be less than latency period")
	} else if concurrentRequests < 1 {
		return nil, fmt.Errorf("concurrent requests must be >= 1")
	} else if maxPeers < 1 {
		return nil, fmt.Errorf("max peers must be >= 1")
	}

	// Create a peer store
	store, err := NewPeerStore(nil, maxPeers, latencyPeriod)
	if err != nil {
		return nil, err
	}

	// Start listening on the tcp port
	listener, err := net.Listen("tcp4", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return nil, err
	}

	// Create a new host
	host := &Host{
		listener,
		store,
		key,
		rpcHandlers,
		false,
		maxPeers,
		concurrentRequests,
		pingPeriod,
		latencyPeriod,
	}
	peerKey := host.PeerKey()
	store.locusKey = peerKey[:]

	return host, nil
}
