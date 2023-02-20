package main

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/the-code-genin/coalition-p2p"
)

func main() {
	if len(os.Args) <= 1 {
		log.Fatalf("At least one boot node address must be specified in the arguments")
	}

	// Spawn a 1000 nodes for the cluster
	var wg sync.WaitGroup
	bootNodes := os.Args[1:]
	hosts := make([]*coalition.Host, 0)
	for i := 0; i < 10; i++ {
		host, err := coalition.NewHost()
		if err != nil {
			panic(err)
		}
		defer host.Close()
		go host.Listen()

		addrs, err := host.Addresses()
		if err != nil {
			panic(err)
		}
		fmt.Printf("Node listening on [%s]\n", addrs[0])

		// Ping all boot nodes
		for i := 0; i < len(bootNodes); i++ {
			if err := host.Ping(bootNodes[i]); err != nil {
				panic(err)
			}
		}

		// Initiate a request to find closest nodes to itself
		// It will attempt to connect to all nodes in it's path
		hostKey := host.PeerKey()
		if _, err := host.FindClosestNodes(hostKey[:]); err != nil {
			panic(err)
		}
		hosts = append(hosts, host)
	}

	fmt.Printf("[%d] cluster nodes online\n", len(hosts))
	wg.Add(1)
	wg.Wait()
}
