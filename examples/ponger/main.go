package main

import (
	"fmt"

	"github.com/the-code-genin/coalition-p2p"
)

func main() {
	host, err := coalition.NewHost(4000)
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
			peerAddr, err := peer.Address()
			if err != nil {
				return nil, err
			}

			fmt.Printf("Received ping from [%s]\n", peerAddr)
			fmt.Printf("Peers [%d]\n", len(h.RouteTable().Peers()))
			return coalition.PingResponse, nil
		},
	)

	addrs, err := host.Addresses()
	if err != nil {
		panic(err)
	}
	for _, addr := range addrs {
		fmt.Printf("Node address [%s]\n", addr)
	}
	host.Listen()
}
