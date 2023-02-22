package coalition

import (
	"encoding/hex"
	"fmt"
	"sync"
)

// Helper to filter dead nodes from a list of peers
func (host *Host) filterDeadNodes(peers []*Peer) (activeNodes, deadNodes []*Peer) {
	var wg sync.WaitGroup
	var mutex sync.Mutex

	activeNodes = make([]*Peer, 0)
	deadNodes = make([]*Peer, 0)

	addDeadNode := func(peer *Peer) {
		mutex.Lock()
		defer mutex.Unlock()
		deadNodes = append(deadNodes, peer)
	}

	for _, peer := range peers {
		wg.Add(1)
		go func(peer *Peer) {
			defer wg.Done()

			// Get the full peer address
			peerAddr, err := peer.Address()
			if err != nil {
				addDeadNode(peer)
				return
			}

			// Check if peer is alive
			if err := host.Ping(peerAddr); err != nil {
				addDeadNode(peer)
				return
			}

			mutex.Lock()
			defer mutex.Unlock()
			activeNodes = append(activeNodes, peer)
		}(peer)
	}
	wg.Wait()
	return
}

// Send a ping to the host at the address
func (host *Host) Ping(address string) error {
	response, err := host.SendMessage(address, 1, PingMethod, nil)
	if err != nil {
		return err
	}
	data, ok := response.(string)
	if !ok {
		return fmt.Errorf("expected [%s] as response", PingResponse)
	} else if data != PingResponse {
		return fmt.Errorf("expected [%s] as response", PingResponse)
	}
	return nil
}

// Asks a peer for a list of nodes closest to a key on the network
func (host *Host) FindNode(address string, key []byte) ([]string, error) {
	response, err := host.SendMessage(
		address,
		1,
		FindNodeMethod,
		hex.EncodeToString(key),
	)
	if err != nil {
		return nil, err
	}

	data, ok := response.([]interface{})
	if !ok {
		return nil, fmt.Errorf("expected an array of node addresses as response")
	}
	addrs := make([]string, 0)
	for _, raw := range data {
		addr, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("expected a string")
		}
		addrs = append(addrs, addr)
	}
	return addrs, nil
}
