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
	for i := 0; i < int(coalition.DefaultMaxPeers); i++ {
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

		// Connect to last 10 hosts
		added := 0
		for i := len(hosts); i > 0 && added < 10; i-- {
			prevHost := hosts[i - 1]
			if err := prevHost.Ping(addrs[0]); err != nil {
				panic(err)
			}
			added++
		}

		hosts = append(hosts, host)
	}

	fmt.Println("Boot nodes online")
	wg.Add(1)
	wg.Wait()
}
