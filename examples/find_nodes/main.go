package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/the-code-genin/coalition-p2p"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("At least remote node address must be specified as the argument")
	}

	// Create a new DHT host
	host, err := coalition.NewHost()
	if err != nil {
		panic(err)
	}
	defer host.Close()

	// Connect to the bootnode
	if err := host.Ping(os.Args[1]); err != nil {
		panic(err)
	}
	fmt.Println("Completed bootstrap")

	// Parse search key
	var searchKey []byte
	if len(os.Args) > 2 {
		searchKey, err = hex.DecodeString(os.Args[2])
		if err != nil {
			panic(err)
		}
	} else {
		key := host.PeerKey()
		searchKey = key[:]
	}

	// Print host address
	addrs, err := host.Addresses()
	if err != nil {
		panic(err)
	}
	fmt.Printf(
		"Finding closest nodes to [%s] from [%s]\n", 
		hex.EncodeToString(searchKey), 
		addrs[0],
	)

	// Find closest nodes to the search key
	nodes, err := host.FindClosestNodes(searchKey)
	if err != nil {
		panic(err)
	}
	for _, node := range nodes {
		nodeAddr, err := node.Address()
		if err != nil {
			panic(err)
		}
		fmt.Println(nodeAddr)
	}
}
