package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/the-code-genin/coalition-p2p"
)

func main() {
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

	port := 3000
	host, err := coalition.NewHost(
		port,
		privKey,
		coalition.RPCHandlerFuncMap{
			"ping": func(
				h *coalition.Host,
				peerKey [coalition.PeerKeySize]byte,
				req coalition.RPCRequest,
			) (interface{}, error) {
				fmt.Printf("Received ping from %s\n", hex.EncodeToString(peerKey[:]))
				fmt.Printf("Peers: %d\n", len(h.Peers()))
				return "pong", nil
			},
		},
		20,                                 // Max peers
		3,                                  // Max concurrent requests
		int64(time.Hour.Seconds()),         // LatencyPeriod
		int64((time.Minute * 5).Seconds()), // PingPeriod
	)
	if err != nil {
		panic(err)
	}
	defer host.Close()

	address, err := host.Address()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Node address [%s]\n", address)
	host.Listen()
}
