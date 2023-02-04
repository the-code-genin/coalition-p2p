package main

import (
	"fmt"
	"sync"

	"github.com/the-code-genin/coalition-p2p"
)

var wg sync.WaitGroup

func main() {
	basePort := 3000
	hosts := make([]*coalition.Host, 0)
	for i := 0; i < int(coalition.DefaultMaxPeers * 4); i++ {
		host, err := coalition.NewHost(basePort + i)
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

		// Connect previous host to this host
		for _, prevHost := range hosts {
			if err := prevHost.Ping(addrs[0]); err != nil {
				panic(err)
			}
		}

		hosts = append(hosts, host)
	}

	fmt.Println("Boot nodes online")
	wg.Add(1)
	wg.Wait()
}
