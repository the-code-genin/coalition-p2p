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
		go HandleRPCConnection(host, conn)
	}
}

// Close the host and any associated resources
func (host *Host) Close() {
	host.closed = true
	host.listener.Close()
}

// Send a message to the node at the full qualified ipv4:port address
func (host *Host) SendMessage(
	address string,
	version int,
	method string,
	data interface{},
) (interface{}, error) {
	// Dial node
	conn, err := net.Dial("tcp4", address)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Prepare serialized request
	serializedRequest, err := json.Marshal(&RPCRequest{
		version,
		method,
		data,
	})
	if err != nil {
		return nil, err
	}

	// Prepare the peer signature for the serialized request
	hash := sha256.Sum256(serializedRequest)
	signature := make([]byte, 0)
	signature = append(signature, host.PublicKey()...)
	signature = append(signature, host.Sign(hash[:])...)

	// Send the full request payload
	requestPayload := make([]byte, 0)
	requestPayload = append(requestPayload, signature...)
	requestPayload = append(requestPayload, serializedRequest...)
	requestPayload = append(requestPayload, '\n')
	if _, err = conn.Write(requestPayload); err != nil {
		return nil, err
	}

	// Read response payload
	responsePayload, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		return nil, err
	} else if len(responsePayload) < PeerSignatureSize {
		return nil, fmt.Errorf("Unable to read signature from response payload")
	}

	// Parse the peer signature and response from the response payload
	peerSignature := responsePayload[:PeerSignatureSize]
	peerResponse := responsePayload[PeerSignatureSize : len(responsePayload)-1]

	// Verify the peer signature
	hash = sha256.Sum256(peerResponse)
	publicKey := peerSignature[:ed25519.PublicKeySize]
	ecSignature := peerSignature[ed25519.PublicKeySize:]
	if !ed25519.Verify(publicKey, hash[:], ecSignature) {
		return nil, fmt.Errorf("Invalid peer signature")
	}

	// Parse the peer information and update the host's peer store
	peerKey := sha1.Sum(publicKey)
	peerAddr := conn.RemoteAddr().(*net.TCPAddr)
	_, err = host.store.Insert(
		peerKey[:],
		peerAddr.IP.To4().String(),
		peerAddr.Port,
	)
	if err != nil {
		return nil, err
	}

	// Parse the RPC response from the payload
	var response RPCResponse
	if err = json.Unmarshal(peerResponse, &response); err != nil {
		return nil, err
	} else if !response.Success {
		return nil, fmt.Errorf(response.Data.(string))
	}
	return response.Data, nil
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
