package main

import (
	"fmt"
	"sync"

	"github.com/the-code-genin/coalition-p2p"
)

var wg sync.WaitGroup

func main() {
	noNodes := int(coalition.DefaultConcurrentRequests)
	hosts := make([]*coalition.Host, 0)
	for i := 0; i < noNodes; i++ {
		// Create new host
		host, err := coalition.NewHost()
		if err != nil {
			panic(err)
		}
		defer host.Close()
		go host.Listen()

		// Print host address
		addrs, err := host.Addresses()
		if err != nil {
			panic(err)
		}
		fmt.Printf("Node listening on [%s]\n", addrs[0])

		// Connect to all previous nodes
		for i := 0; i < len(hosts); i++ {
			prevHost := hosts[i]
			if err := prevHost.Ping(addrs[0]); err != nil {
				panic(err)
			}
		}

		hosts = append(hosts, host)
	}

	fmt.Printf("[%d] boot nodes online\n", noNodes)
	wg.Add(1)
	wg.Wait()
}
