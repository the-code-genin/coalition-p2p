package coalition

import (
	"bufio"
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
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

// Return the host's ed25519 public key
func (host *Host) PublicKey() ed25519.PublicKey {
	return host.key.Public().(ed25519.PublicKey)
}

// Returns the 160-bit sha1 hash of the host's public key as the host's peer key
func (host *Host) PeerKey() [PeerKeySize]byte {
	return sha1.Sum([]byte(host.PublicKey()))
}

// Return the listening tcp port
func (host *Host) Port() (int, error) {
	tcpAddr, ok := host.listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("unable to parse host port")
	}
	return tcpAddr.Port, nil
}

// Return the host's peer addresses
func (host *Host) Addresses() ([]string, error) {
	addrs, err := GetPublicIP4Addresses()
	if err != nil {
		return nil, nil
	}

	port, err := host.Port()
	if err != nil {
		return nil, nil
	}

	key := host.PeerKey()

	res := make([]string, 0)
	for _, addr := range addrs {
		res = append(res, FormatNodeAddress(key[:], addr, port))
	}
	return res, nil
}

// Sign a digest with the host's private key
func (host *Host) Sign(digest []byte) []byte {
	return ed25519.Sign(host.key, digest)
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

// Send a message to the node at the address
func (host *Host) SendMessage(
	address string,
	version int,
	method string,
	data interface{},
) (interface{}, error) {
	// Parse the node address
	remotePeerKey, remoteIP4Address, remotePort, err := ParseNodeAddress(address)
	if err != nil {
		panic(err)
	}

	// Dial node
	conn, err := net.Dial("tcp4", fmt.Sprintf("%s:%d", remoteIP4Address, remotePort))
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
	hash := sha256.Sum256(serializedRequest)

	// Prepare the full request payload
	requestPayload := make([]byte, 0)
	requestPayload = append(requestPayload, host.PublicKey()...)
	requestPayload = append(requestPayload, host.Sign(hash[:])...)
	requestPayload = append(requestPayload, serializedRequest...)

	// Send the request
	payloadSizeBuffer := make([]byte, 8)
	big.NewInt(int64(len(requestPayload))).FillBytes(payloadSizeBuffer)
	if _, err := conn.Write(payloadSizeBuffer); err != nil {
		return nil, err
	}
	if _, err := conn.Write(requestPayload); err != nil {
		return nil, err
	}

	// Parse the size of the response payload in bytes
	responseReader := bufio.NewReader(conn)
	payloadSizeBuffer = make([]byte, 8)
	_, err = io.ReadFull(responseReader, payloadSizeBuffer)
	if err != nil {
		return nil, err
	}
	payloadSize := new(big.Int).SetBytes(payloadSizeBuffer).Int64()

	// Read response payload
	responsePayload := make([]byte, payloadSize)
	_, err = io.ReadFull(responseReader, responsePayload)
	if err != nil {
		return nil, err
	} else if len(responsePayload) < PeerSignatureSize+1 {
		return nil, fmt.Errorf("incomplete response body")
	}

	// Parse the peer signature and response from the response payload
	publicKey := responsePayload[:ed25519.PublicKeySize]
	ecSignature := responsePayload[ed25519.PublicKeySize:PeerSignatureSize]
	peerResponse := responsePayload[PeerSignatureSize:]

	// Verify the peer key
	peerKey := sha1.Sum(publicKey)
	if !bytes.Equal(peerKey[:], remotePeerKey) {
		return nil, fmt.Errorf("peer key in address does not match peer key in response")
	}

	// Verify the peer signature
	hash = sha256.Sum256(peerResponse)
	if !ed25519.Verify(publicKey, hash[:], ecSignature) {
		return nil, fmt.Errorf("invalid peer signature")
	}

	// Update the host's peer store
	_, err = host.store.Insert(
		remotePeerKey,
		remoteIP4Address,
		remotePort,
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

// Registers a new RPC method or overrites an existing method
func (host *Host) RegisterRPCMethod(
	methodName string,
	handler RPCHandlerFunc,
) {
	host.rpcHandlers[methodName] = handler
}

// Close the host and any associated resources
func (host *Host) Close() {
	host.closed = true
	host.listener.Close()
}

// Create a new P2P host on the specified ip4 port
// Options can be passed to configure the node
func NewHost(
	port int,
	options ...Option,
) (*Host, error) {
	// Parse the peer key
	key, ok := getOption(PrivateKeyOption, options, nil).(ed25519.PrivateKey)
	if !ok {
		_, privKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, err
		}
		key = privKey
	}

	// Parse the kademlia parameters
	maxPeers := getOption(MaxPeersOption, options, DefaultMaxPeers).(int64)
	concurrentRequests := getOption(ConcurrentRequestsOption, options, DefaultConcurrentRequests).(int64)
	pingPeriod := getOption(PingPeriodOption, options, DefaultPingPeriod).(int64)
	latencyPeriod := getOption(LatencyPeriodOption, options, DefaultLatencyPeriod).(int64)
	if pingPeriod >= latencyPeriod {
		return nil, fmt.Errorf("ping period should be less than latency period")
	} else if concurrentRequests < 1 {
		return nil, fmt.Errorf("concurrent requests must be >= 1")
	} else if maxPeers < 1 {
		return nil, fmt.Errorf("max peers must be >= 1")
	}

	// Create a peer store
	peerKey := sha1.Sum([]byte(key.Public().(ed25519.PublicKey)))
	store, err := NewPeerStore(peerKey[:], maxPeers, latencyPeriod)
	if err != nil {
		return nil, err
	}

	// Start listening on the tcp port
	listener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, err
	}

	// Create a new host
	rpcHandlers := make(RPCHandlerFuncMap)
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

	// Register the ping RPC method with returns a pong response
	host.RegisterRPCMethod(
		PingMethod,
		func(*Host, *Peer, RPCRequest) (interface{}, error) {
			return PingResponse, nil
		},
	)

	// Register the find_node RPC method which finds nodes near a key
	// The nodes are sorted from closest to farthest from the key
	host.RegisterRPCMethod(
		FindNodeMethod,
		func(host *Host, _ *Peer, req RPCRequest) (interface{}, error) {
			keyHex, ok := req.Data.(string)
			if !ok {
				return nil, fmt.Errorf("node key not found in request body")
			}
			key, err := hex.DecodeString(keyHex)
			if err != nil {
				return nil, err
			}
			peers, err := host.store.SortPeersByProximity(key)
			if err != nil {
				return nil, err
			}
			addrs := make([]string, 0)
			for _, peer := range peers {
				addrs = append(addrs, peer.Address())
			}
			return addrs, nil
		},
	)

	return host, nil
}
