package coalition

import (
	"bytes"
	"sync"
)

// Find network peers closest to a search key
func (host *Host) FindClosestNodes(searchKey []byte) ([]*Peer, error) {
	var wg sync.WaitGroup
	var mutex sync.Mutex

	hostKey := host.PeerKey()
	concurrentRequests := int(host.concurrentRequests)

	activeNodes, inactiveNodes := host.filterDeadNodes(host.RouteTable().Peers())
	prevLookUpRes := make([]*Peer, 0)
	currentLookUpRes := SortPeersByClosest(
		activeNodes,
		searchKey,
	)
	if len(currentLookUpRes) == 0 {
		return make([]*Peer, 0), nil
	}

	// Do a recursive search until all closest nodes have been found
	for {
		// Current lookups becomes part of previous lookups
		prevLookUpRes = append(prevLookUpRes, currentLookUpRes...)
		prevLookUpRes = SortPeersByClosest(prevLookUpRes, searchKey)

		// Max distance of furtherest peer in prev lookups
		maxDistance := XORBytes(prevLookUpRes[len(prevLookUpRes)-1].Key(), searchKey)

		// Find closer set of nodes to the key from the lookup nodes
		newRes := make([]*Peer, 0)
		for i := 0; i < concurrentRequests && i < len(currentLookUpRes); i++ {
			wg.Add(1)
			go func(lookupNode *Peer) {
				defer wg.Done()

				// Find closest nodes from the lookup node's routing table
				lookupNodeAddr, err := lookupNode.Address()
				if err != nil {
					return
				}
				responseAddrs, err := host.FindNode(lookupNodeAddr, searchKey)
				if err != nil {
					return
				}

				mutex.Lock()
				defer mutex.Unlock()

				for _, responseAddr := range responseAddrs {
					peer, err := NewPeerFromAddress(responseAddr)
					if err != nil {
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

					// Skip inactive nodes
					inactive := false
					for i := 0; i < len(inactiveNodes); i++ {
						if bytes.Equal(inactiveNodes[i].Key(), peer.Key()) {
							inactive = true
							break
						}
					}
					if inactive {
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

					// Ensure this peer is closer than the furtherest of the previous peers
					// And ensure the peer isn't this host
					distance := XORBytes(peer.Key(), searchKey)
					if bytes.Compare(distance, maxDistance) > 0 || bytes.Equal(peer.Key(), hostKey[:]) {
						prevLookUpRes = append(prevLookUpRes, peer)
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

		// Refresh lookup nodes for next look up
		// Dead nodes are noted to prevent interference with future results
		activeNodes, deadNodes := host.filterDeadNodes(newRes)
		currentLookUpRes = SortPeersByClosest(activeNodes, searchKey)
		inactiveNodes = append(inactiveNodes, deadNodes...)
	}

	// Sort all lookups from closest to farthest
	// Return at most max peers
	prevLookUpRes = SortPeersByClosest(prevLookUpRes, searchKey)
	if len(prevLookUpRes) >= int(host.maxPeers) {
		return prevLookUpRes[:host.maxPeers], nil
	}
	return prevLookUpRes, nil
}
