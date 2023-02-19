package coalition

import (
	"bytes"
	"math/big"
	"sync"
)

type DHT struct {
	host *Host
}

// Find peers closest to a search key
func (dht *DHT) FindClosestNodes(searchKey []byte) ([]*Peer, error) {
	var wg sync.WaitGroup
	var mutex sync.Mutex

	maxPeers := int(dht.host.maxPeers)
	prevLookUpRes := make([]*Peer, 0)
	currentLookUpRes := dht.host.RouteTable().Peers()

	// Do a recursive search until all closest nodes have been found
	for {
		// Current lookups becomes previous lookups
		prevLookUpRes = append(prevLookUpRes, currentLookUpRes...)

		// Find up to max peers closest set of nodes to the key from the lookup nodes
		newRes := make([]*Peer, 0)
		for i := 0; len(newRes) < maxPeers && i < len(currentLookUpRes); i++ {
			wg.Add(1)
			go func(lookupNode *Peer) {
				defer wg.Done()

				// Skip this DHT's host
				if bytes.Equal(lookupNode.Key(), dht.host.key[:]) {
					return
				}

				lookupNodeAddr, err := lookupNode.Address()
				if err != nil {
					return
				}
				responseAddrs, err := dht.host.FindNode(lookupNodeAddr, searchKey)
				if err != nil {
					return
				}

				mutex.Lock()
				defer mutex.Unlock()

				if len(newRes) >= maxPeers {
					return
				}

				for _, responseAddr := range responseAddrs {
					peer, err := NewPeerFromAddress(responseAddr)
					if err != nil {
						return
					}

					// Skip this DHT's host
					if bytes.Equal(peer.Key(), dht.host.key[:]) {
						continue
					}

					// Skip old look ups
					old := false
					for i := 0; i < len(prevLookUpRes); i++ {
						if bytes.Equal(prevLookUpRes[i].Key(), peer.Key()) {
							old = true
							break
						}
					}
					if old {
						continue
					}

					// Skip duplicates
					duplicate := false
					for i := 0; i < len(newRes); i++ {
						if bytes.Equal(newRes[i].Key(), peer.Key()) {
							duplicate = true
							break
						}
					}
					if duplicate {
						continue
					}

					newRes = append(newRes, peer)
				}
			}(currentLookUpRes[i])
		}
		wg.Wait()

		// Unable to find closer nodes to the search key
		if len(newRes) == 0 {
			break
		}

		// Sort the network response from closest to farthest from the search key
		newRes = MergeSortPeers(
			newRes,
			make([]*Peer, 0),
			func(peerA, peerB *Peer) int {
				distanceA := new(big.Int).Xor(
					new(big.Int).SetBytes(peerA.Key()),
					new(big.Int).SetBytes(searchKey),
				).Bytes()
				distanceB := new(big.Int).Xor(
					new(big.Int).SetBytes(peerB.Key()),
					new(big.Int).SetBytes(searchKey),
				).Bytes()
				return bytes.Compare(distanceB, distanceA)
			},
		)

		// Refresh lookup nodes for next look up
		// Dead nodes are filtered out
		// If the node has been queried before it is skipped
		currentLookUpRes = make([]*Peer, 0)
		for _, peer := range newRes {
			peerAddr, err := peer.Address()
			if err != nil {
				continue
			}

			// Check if alive
			if err := dht.host.Ping(peerAddr); err != nil {
				continue
			}
			currentLookUpRes = append(currentLookUpRes, peer)
		}
	}

	// Sort all lookups from closest to farthest
	prevLookUpRes = MergeSortPeers(
		prevLookUpRes,
		make([]*Peer, 0),
		func(peerA, peerB *Peer) int {
			distanceA := new(big.Int).Xor(
				new(big.Int).SetBytes(peerA.Key()),
				new(big.Int).SetBytes(searchKey),
			).Bytes()
			distanceB := new(big.Int).Xor(
				new(big.Int).SetBytes(peerB.Key()),
				new(big.Int).SetBytes(searchKey),
			).Bytes()
			return bytes.Compare(distanceB, distanceA)
		},
	)

	if len(prevLookUpRes) >= int(dht.host.maxPeers) {
		return prevLookUpRes[:dht.host.maxPeers], nil
	}
	return prevLookUpRes, nil
}

func NewDHT(host *Host) *DHT {
	return &DHT{host}
}
