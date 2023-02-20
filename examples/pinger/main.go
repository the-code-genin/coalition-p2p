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

	host, err := coalition.NewHost()
	if err != nil {
		panic(err)
	}
	defer host.Close()

	addrs, err := host.Addresses()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Sending [ping] from [%s]\n", addrs[0])
	response, err := host.SendMessage(os.Args[1], 1, coalition.PingMethod, nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(response.(string))
}
