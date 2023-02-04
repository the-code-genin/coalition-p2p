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
	bootNode := os.Args[1]
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
	fmt.Printf("Boot node [%s]\n", bootNode)
	fmt.Printf("Search key [%s]\n", hex.EncodeToString(searchKey))

	concurrentReqs := int(coalition.DefaultConcurrentRequests)
	lookUpNodes := []string{bootNode}
	var lookUpRes []*coalition.Peer
	for i := 0; i < 10; i++ {
		// Find closest set nodes to the key from the lookup nodes
		lookUpRes = make([]*coalition.Peer, 0)
		for i := 0; i < concurrentReqs && i < len(lookUpNodes); i++ {
			wg.Add(1)
			go func(addr string) {
				defer wg.Done()
				responseAddrs, err := host.FindNode(addr, searchKey)
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

					if bytes.Equal(peer.Key(), searchKey) {
						fmt.Printf("Found node at [%s]\n", peer.Address())
						log.Fatalf("\n")
					} else if bytes.Equal(peer.Key(), hostKey[:]) {
						continue
					}

					duplicate := false
					for i := 0; i < len(lookUpRes); i++ {
						if bytes.Equal(lookUpRes[i].Key(), peer.Key()) {
							duplicate = true
							break
						}
					}
					if !duplicate {
						lookUpRes = append(lookUpRes, peer)
					}
				}
			}(lookUpNodes[i])
		}
		wg.Wait()

		// Sort the network response from closest to farthest
		lookUpRes = coalition.MergeSortPeers(
			lookUpRes,
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
		fmt.Printf("Got [%d] close peers from network\n", len(lookUpRes))
		for _, addr := range lookUpRes {
			fmt.Println(addr.Address())
		}
		fmt.Println()

		// Refresh lookup nodes for next look up
		// Dead nodes are filtered out
		lookUpNodes = make([]string, 0)
		for _, peer := range lookUpRes {
			if err := host.Ping(peer.Address()); err != nil {
				continue
			}
			lookUpNodes = append(lookUpNodes, peer.Address())
			if len(lookUpNodes) == concurrentReqs {
				break
			}
		}
	}
}
