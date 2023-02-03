package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/the-code-genin/coalition-p2p"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Remote node address must be specified as the argument")
	}

	host, err := coalition.NewHost(3002)
	if err != nil {
		panic(err)
	}
	defer host.Close()

	address, err := host.Address()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Sending [find_node] from [%s]\n", address)
	peerKey := host.PeerKey()
	response, err := host.SendMessage(
		os.Args[1],
		1,
		coalition.FindNodeMethod,
		hex.EncodeToString(peerKey[:]),
	)
	if err != nil {
		panic(err)
	}
	fmt.Println(response.([]interface{}))
}
