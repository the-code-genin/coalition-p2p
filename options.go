package coalition

import "crypto/ed25519"

type Option struct {
	Name  string
	Value interface{}
}

// Quick helper to get an option within a list of options
func getOption(name string, opList []Option, defaultValue interface{}) interface{} {
	for i := 0; i < len(opList); i++ {
		if opList[i].Name == name {
			return opList[i].Value
		}
	}
	return defaultValue
}

// The listening port to be used by the host
func Port(port int) Option {
	return Option{PortOption, port}
}

// The private key to be used by the host
func Identity(key ed25519.PrivateKey) Option {
	return Option{PrivateKeyOption, key}
}

// The kademlia replication parameter
func MaxPeers(peers int64) Option {
	return Option{MaxPeersOption, peers}
}

// The kademlia concurrent requests parameter
func ConcurrentRequests(requests int64) Option {
	return Option{ConcurrentRequestsOption, requests}
}

// The ping interval in seconds
func PingPeriod(period int64) Option {
	return Option{PingPeriodOption, period}
}

// The latency period in seconds
func LatencyPeriod(period int64) Option {
	return Option{LatencyPeriodOption, period}
}
