package main

import (
	"fmt"
	"log"
	"os"

	"github.com/the-code-genin/coalition-p2p"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Remote node address must be specified as the argument")
	}

	// Create a new DHT host
	host, err := coalition.NewHost()
	if err != nil {
		panic(err)
	}
	defer host.Close()
	dht := coalition.NewDHT(host)

	// Connect to the bootnode
	if err := host.Ping(os.Args[1]); err != nil {
		panic(err)
	}
	fmt.Println("Completed bootstrap")

	// Print host address
	addrs, err := host.Addresses()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Finding dht closest nodes to host from [%s]\n", addrs[0])

	// Find closest nodes to the host
	hostKey := host.PeerKey()
	nodes, err := dht.FindClosestNodes(hostKey[:])
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
