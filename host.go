package coalition

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// Represents a basic p2p node with an optimized kbucket peer store
type Host struct {
	listener           net.Listener
	table              *RouteTable
	key                ed25519.PrivateKey
	rpcHandlers        RPCHandlerFuncMap
	closed             bool
	maxPeers           int64
	pingPeriod         int64
	latencyPeriod      int64
	concurrentRequests int64
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
	ipAddrs, err := GetPublicIP4Addresses()
	if err != nil {
		return nil, nil
	}

	port, err := host.Port()
	if err != nil {
		return nil, nil
	}

	key := host.PeerKey()

	res := make([]string, 0)
	for _, ipAddr := range ipAddrs {
		nodeAddr, err := FormatNodeAddress(key[:], ipAddr, port)
		if err != nil {
			return nil, err
		}
		res = append(res, nodeAddr)
	}
	return res, nil
}

// Generate a peer signature from a digest by signing with the host's private key
func (host *Host) Sign(digest []byte) ([PeerSignatureSize]byte, error) {
	output := *new([PeerSignatureSize]byte)
	payload := make([]byte, 0)
	payload = append(payload, host.PublicKey()...)
	payload = append(payload, ed25519.Sign(host.key, digest)...)
	if len(payload) != PeerSignatureSize {
		return output, fmt.Errorf("error occured while signing digest")
	}
	if i := copy(output[:], payload); i != PeerSignatureSize {
		return output, fmt.Errorf("error occured while signing digest")
	}
	return output, nil
}

// Returns the host's route table
func (host *Host) RouteTable() *RouteTable {
	return host.table
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

	// Prepare request signature
	requestHash := sha256.Sum256(serializedRequest)
	requestSignature, err := host.Sign(requestHash[:])
	if err != nil {
		return nil, err
	}

	// Get the host's listening port
	hostPort, err := host.Port()
	if err != nil {
		return nil, err
	}

	// Prepare the full request payload
	requestPayload := make([]byte, 0)
	requestPayload = append(requestPayload, Uint64ToBytes(uint64(hostPort))...)
	requestPayload = append(requestPayload, requestSignature[:]...)
	requestPayload = append(requestPayload, serializedRequest...)

	// Send the request
	if err := WriteToConn(conn, requestPayload); err != nil {
		return nil, err
	}

	// Read the payload from the connection
	responsePayload, err := ReadFromConn(conn)
	if err != nil {
		return nil, err
	} else if len(responsePayload) <= PeerSignatureSize {
		return nil, fmt.Errorf("incomplete response body")
	}

	// Parse the peer signature and response from the response payload
	peerSignature := responsePayload[:PeerSignatureSize]
	peerResponse := responsePayload[PeerSignatureSize:]

	// Verify the peer key in the response payload
	responseHash := sha256.Sum256(peerResponse)
	peerKey, err := RecoverPeerKeyFromPeerSignature(peerSignature, responseHash[:])
	if err != nil {
		return nil, err
	} else if !bytes.Equal(peerKey, remotePeerKey) {
		return nil, fmt.Errorf("peer key in address does not match peer key in response")
	}

	// Update the host's route table
	_, err = host.table.Insert(
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

// A long running service that pings all peers within it's route table
// It only pings the peer if it's last seen interval is greater than the ping period
func (host *Host) startPingService() {
	for !host.closed {
		for _, peer := range host.RouteTable().Peers() {
			if time.Now().Unix()-peer.LastSeen() < host.pingPeriod {
				continue
			}
			peerAddr, err := peer.Address()
			if err != nil {
				continue
			}
			host.Ping(peerAddr)
		}
		time.Sleep(time.Duration(host.pingPeriod))
	}
}

// A long running service that prunes inactive peers within it's route table
func (host *Host) startLatencyPruneService() {
	for !host.closed {
		for _, peer := range host.RouteTable().Peers() {
			if time.Now().Unix()-peer.LastSeen() < host.latencyPeriod {
				continue
			}
			host.table.Remove(peer.Key())
		}
		time.Sleep(time.Duration(host.latencyPeriod))
	}
}

// Close the host and any associated resources
func (host *Host) Close() {
	host.closed = true
	host.listener.Close()
}

// Create a new P2P host
func NewHost(
	options ...Option,
) (*Host, error) {
	// Parse the peer private key
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
	pingPeriod := getOption(PingPeriodOption, options, DefaultPingPeriod).(int64)
	latencyPeriod := getOption(LatencyPeriodOption, options, DefaultLatencyPeriod).(int64)
	concurrentRequests := getOption(ConcurrentRequestsOption, options, DefaultConcurrentRequests).(int64)
	if pingPeriod >= latencyPeriod {
		return nil, fmt.Errorf("ping period should be less than latency period")
	} else if maxPeers < 1 {
		return nil, fmt.Errorf("max peers must be >= 1")
	} else if concurrentRequests > maxPeers {
		return nil, fmt.Errorf("concurrent requests must be <= max peers")
	}

	// Create a peer store
	peerKey := sha1.Sum([]byte(key.Public().(ed25519.PublicKey)))
	table, err := NewRouteTable(peerKey[:], maxPeers, latencyPeriod)
	if err != nil {
		return nil, err
	}

	// Start listening on the tcp port
	port := getOption(PortOption, options, 0).(int)
	listener, err := net.Listen("tcp4", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return nil, err
	}

	// Create a new host
	rpcHandlers := make(RPCHandlerFuncMap)
	host := &Host{
		listener,
		table,
		key,
		rpcHandlers,
		false,
		maxPeers,
		pingPeriod,
		latencyPeriod,
		concurrentRequests,
	}

	// Register standard RPC methods
	host.RegisterRPCMethod(PingMethod, PingHandler)
	host.RegisterRPCMethod(FindNodeMethod, FindNodeHandler)

	// Fire up long running services
	go host.startPingService()
	go host.startLatencyPruneService()

	return host, nil
}
