package coalition

import (
	"encoding/hex"
	"fmt"
)

// Handles ping requests which returns a pong response
func PingHandler(*Host, *Peer, RPCRequest) (interface{}, error) {
	return PingResponse, nil
}

// Handles find_node requests which finds nodes near a key
// The nodes are sorted from closest to farthest from the key
func FindNodeHandler(host *Host, remotePeer *Peer, req RPCRequest) (interface{}, error) {
	keyHex, ok := req.Data.(string)
	if !ok {
		return nil, fmt.Errorf("node key not found in request body")
	}
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, err
	}
	peers, err := host.table.SortPeersByProximity(key)
	if err != nil {
		return nil, err
	}
	addrs := make([]string, 0)
	for _, peer := range peers {
		addrs = append(addrs, peer.Address())
	}
	return addrs, nil
}
