package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/the-code-genin/coalition-p2p"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Remote node address must be specified as the argument")
	}

	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	host, err := coalition.NewHost(
		3001,
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

	peerKey := host.PeerKey()
	fmt.Printf("Sending ping from %s\n", hex.EncodeToString(peerKey[:]))
	response, err := host.SendMessage(os.Args[1], 1, "ping", nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(response.(string))
	fmt.Printf("Peers: %d\n", len(host.Peers()))
}
