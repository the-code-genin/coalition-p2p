package coalition

import (
	"crypto/ed25519"
	"time"
)

// Size of peer key in bytes
const PeerKeySize = 20

// Size of peer payload signature in bytes
const PeerSignatureSize = ed25519.PublicKeySize + ed25519.SignatureSize

const PrivateKeyOption = "private_key"

const MaxPeersOption = "max_peers"
const DefaultMaxPeers = int64(20)

const ConcurrentRequestsOption = "concurrent_requests"
const DefaultConcurrentRequests = int64(3)

const PingPeriodOption = "ping_period"
const PingMethod = "ping"
const PingResponse = "pong"
const DefaultPingPeriod = int64(time.Minute * 5)

const LatencyPeriodOption = "latency_period"
const DefaultLatencyPeriod = int64(time.Hour)
