package coalition

import (
	"crypto/ed25519"
	"crypto/sha1"
	"time"
)

// Size of peer key in bytes
const PeerKeySize = sha1.Size

// Size of peer payload signature in bytes
const PeerSignatureSize = ed25519.PublicKeySize + ed25519.SignatureSize

// Peer identity option
const PrivateKeyOption = "private_key"

// Peer listening port option
const PortOption = "port"

// Kademlia replication parameter
const MaxPeersOption = "max_peers"
const DefaultMaxPeers = int64(PeerKeySize * 8)

// Ping RPC method
const PingPeriodOption = "ping_period"
const PingMethod = "ping"
const PingResponse = "pong"
const DefaultPingPeriod = int64(time.Minute * 5)

// Node latency period before it's eligible to be kicked off the routing table
const LatencyPeriodOption = "latency_period"
const DefaultLatencyPeriod = int64(time.Hour)

// RPC method to list peers near a certain key
const FindNodeMethod = "find_node"

// Size of 64 bit integers in bytes
const Int64Len = 8

// TCP Read/Write deadlines
const TCPIODeadline = time.Second * 100
