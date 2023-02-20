package main

import (
	"fmt"
	"log"
	"os"

	"github.com/the-code-genin/coalition-p2p"
)

func main() {
	if len(os.Args) <= 1 {
		log.Fatalf("At least one remote node address must be specified in the arguments")
	}

	host, err := coalition.NewHost(4060)
	if err != nil {
		panic(err)
	}
	defer host.Close()

	addrs, err := host.Addresses()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Bridging from from [%s]\n", addrs[0])

	for i := 1; i < len(os.Args); i++ {
		if err := host.Ping(os.Args[i]); err != nil {
			panic(err)
		}
	}

	host.Listen()
}
