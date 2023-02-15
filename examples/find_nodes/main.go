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
	bootnodeAddr, err := bootNode.Address()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Sending [find_node] from [%s]\n", addrs[0])
	fmt.Printf("Boot node [%s]\n", bootnodeAddr)
	fmt.Printf("Search key [%s]\n", hex.EncodeToString(searchKey))

	maxPeers := int(coalition.DefaultMaxPeers)
	prevLookUpRes := []*coalition.Peer{bootNode}
	currentLookUpRes := []*coalition.Peer{bootNode}
	for {
		// Find up to max peers closest set of nodes to the key from the lookup nodes
		newRes := make([]*coalition.Peer, 0)
		for i := 0; len(newRes) < maxPeers && i < len(currentLookUpRes); i++ {
			wg.Add(1)
			go func(lookupNode *coalition.Peer) {
				defer wg.Done()
				lookupNodeAddr, err := lookupNode.Address()
				if err != nil {
					panic(err)
				}
				responseAddrs, err := host.FindNode(lookupNodeAddr, searchKey)
				if err != nil {
					panic(err)
				}

				mutex.Lock()
				defer mutex.Unlock()

				if len(newRes) >= maxPeers {
					return
				}

				for _, responseAddr := range responseAddrs {
					peer, err := coalition.NewPeerFromAddress(responseAddr)
					if err != nil {
						panic(err)
					}

					// Found search key
					if bytes.Equal(peer.Key(), searchKey) {
						peerAddr, err := peer.Address()
						if err != nil {
							panic(err)
						}
						fmt.Printf("Found node at [%s]\n", peerAddr)
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

		if len(newRes) == 0 {
			log.Fatal("unable to find node in network")
		}

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
		for _, peer := range newRes {
			fmt.Println(peer.Address())
		}
		fmt.Println()

		// Refresh lookup nodes for next look up
		// Dead nodes are filtered out
		// If the node has been queried before it is skipped
		prevLookUpRes = append(prevLookUpRes, currentLookUpRes...)
		currentLookUpRes = make([]*coalition.Peer, 0)
		for _, peer := range newRes {
			peerAddr, err := peer.Address()
			if err != nil {
				panic(err)
			}

			// Check if alive
			if err := host.Ping(peerAddr); err != nil {
				continue
			}
			currentLookUpRes = append(currentLookUpRes, peer)
		}
	}
}
