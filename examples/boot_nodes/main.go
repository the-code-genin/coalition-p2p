package main

import (
	"fmt"
	"sync"

	"github.com/the-code-genin/coalition-p2p"
)

var wg sync.WaitGroup

func main() {
	hostA, err := coalition.NewHost(3000)
	if err != nil {
		panic(err)
	}
	defer hostA.Close()

	hostB, err := coalition.NewHost(3001)
	if err != nil {
		panic(err)
	}
	defer hostB.Close()

	addrsA, err := hostA.Addresses()
	if err != nil {
		panic(err)
	}
	addrsB, err := hostB.Addresses()
	if err != nil {
		panic(err)
	}

	go hostA.Listen()
	go hostB.Listen()

	fmt.Printf("NodeA listening on [%s]\n", addrsA[0])
	fmt.Printf("NodeB listening on [%s]\n", addrsB[0])

	// Connect hostB to hostA
	err = hostB.Ping(addrsA[0])
	if err != nil {
		panic(err)
	}

	fmt.Printf("NodeA has [%d] peers\n", len(hostA.RouteTable().Peers()))
	fmt.Printf("NodeB has [%d] peers\n", len(hostB.RouteTable().Peers()))

	fmt.Println("Boot nodes online")
	wg.Add(1)
	wg.Wait()
}
