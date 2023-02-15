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

const PrivateKeyOption = "private_key"

// Kademlia replication parameter
const MaxPeersOption = "max_peers"
const DefaultMaxPeers = int64(PeerKeySize * 8)

// Ping RPC method
const PingPeriodOption = "ping_period"
const PingMethod = "ping"
const PingResponse = "pong"
const DefaultPingPeriod = int64(time.Minute * 5)

const LatencyPeriodOption = "latency_period"
const DefaultLatencyPeriod = int64(time.Hour)

const FindNodeMethod = "find_node"
