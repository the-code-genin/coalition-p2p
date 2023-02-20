package main

import (
	"fmt"
	"sync"

	"github.com/the-code-genin/coalition-p2p"
)

func main() {
	var wg sync.WaitGroup
	hosts := make([]*coalition.Host, 0)
	for i := 0; i < int(coalition.DefaultConcurrentRequests); i++ {
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

	fmt.Printf("[%d] boot nodes online\n", len(hosts))
	wg.Add(1)
	wg.Wait()
}
