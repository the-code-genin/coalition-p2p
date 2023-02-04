package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"os"
	"sync"

	"github.com/the-code-genin/coalition-p2p"
)

var wg sync.WaitGroup
var mutex sync.Mutex

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("Remote node address and search key must be specified as the argument")
	}

	// Setup host
	host, err := coalition.NewHost(4000)
	if err != nil {
		panic(err)
	}
	hostKey := host.PeerKey()
	defer host.Close()

	// Parse bootnode and search key
	bootNode, err := coalition.NewPeerFromAddress(os.Args[1])
	if err != nil {
		panic(err)
	}
	searchKey, err := hex.DecodeString(os.Args[2])
	if err != nil {
		panic(err)
	}

	// Output
	addrs, err := host.Addresses()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Sending [find_node] from [%s]\n", addrs[0])
	fmt.Printf("Boot node [%s]\n", bootNode.Address())
	fmt.Printf("Search key [%s]\n", hex.EncodeToString(searchKey))

	maxPeers := int(coalition.DefaultMaxPeers)
	prevLookUpRes := []*coalition.Peer{bootNode}
	currentLookUpRes := []*coalition.Peer{bootNode}
	for i := 0; i < 10; i++ {
		// Find up to max peers closest set of nodes to the key from the lookup nodes
		newRes := make([]*coalition.Peer, 0)
		for i := 0; len(newRes) < maxPeers && i < len(currentLookUpRes); i++ {
			wg.Add(1)
			go func(lookupNode *coalition.Peer) {
				defer wg.Done()
				responseAddrs, err := host.FindNode(lookupNode.Address(), searchKey)
				if err != nil {
					panic(err)
				}

				mutex.Lock()
				defer mutex.Unlock()
				for _, responseAddr := range responseAddrs {
					peer, err := coalition.NewPeerFromAddress(responseAddr)
					if err != nil {
						panic(err)
					}

					// Found search key
					if bytes.Equal(peer.Key(), searchKey) {
						fmt.Printf("Found node at [%s]\n", peer.Address())
						log.Fatalf("\n")
					} else if bytes.Equal(peer.Key(), hostKey[:]) {
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
					for i := 0; i < len(currentLookUpRes); i++ {
						if bytes.Equal(currentLookUpRes[i].Key(), peer.Key()) {
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

		// Sort the network response from closest to farthest
		newRes = coalition.MergeSortPeers(
			newRes,
			make([]*coalition.Peer, 0),
			func(peerA, peerB *coalition.Peer) int {
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

		// Output
		fmt.Printf("Got [%d] new closer peers from network\n", len(newRes))
		for _, addr := range newRes {
			fmt.Println(addr.Address())
		}
		fmt.Println()

		// Refresh lookup nodes for next look up
		// Dead nodes are filtered out
		// If the node has been queried before it is skipped
		prevLookUpRes = append(prevLookUpRes, currentLookUpRes...)
		currentLookUpRes = make([]*coalition.Peer, 0)
		for _, peer := range newRes {
			// Check if alive
			if err := host.Ping(peer.Address()); err != nil {
				continue
			}
			currentLookUpRes = append(currentLookUpRes, peer)
		}
	}
}
