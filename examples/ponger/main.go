package main

import (
	"fmt"

	"github.com/the-code-genin/coalition-p2p"
)

func main() {
	host, err := coalition.NewHost(3000)
	if err != nil {
		panic(err)
	}
	defer host.Close()

	// Override the "ping" method
	host.RegisterRPCMethod(
		coalition.PingMethod,
		func(
			h *coalition.Host,
			peer *coalition.Peer,
			req coalition.RPCRequest,
		) (interface{}, error) {
			fmt.Printf("Received ping from [%s]\n", peer.Address())
			fmt.Printf("Peers [%d]\n", len(h.Peers()))
			return coalition.PingResponse, nil
		},
	)

	address, err := host.Address()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Node address [%s]\n", address)
	host.Listen()
}
