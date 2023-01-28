package coalition

import (
	"bytes"
	"fmt"
	"math"
	"math/big"
	"time"
)

// A network peer
type Peer struct {
	key       []byte
	ipAddress string
	port      int
	lastSeen  int64
}

// Return the peer key
func (peer *Peer) Key() []byte {
	return peer.key
}

// Return the peer IPv4 Address
func (peer *Peer) IPAddress() string {
	return peer.ipAddress
}

// Return the peer listening port
func (peer *Peer) Port() int {
	return peer.port
}

// Return the fully qualified peer address
func (peer *Peer) Address() string {
	return fmt.Sprintf("%s:%d", peer.ipAddress, peer.port)
}

// Return the peer's last seen timestamp
func (peer *Peer) LastSeen() int64 {
	return peer.lastSeen
}

// Return the peer's XOR distance from a key
func (peer *Peer) Distance(key []byte) (*big.Int, error) {
	if len(key) != len(peer.key) {
		return nil, fmt.Errorf("peer key length miss-match")
	}
	xorBytes, err := XORBytes(peer.key, key)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetBytes(xorBytes), nil
}

type PeerStore struct {
	locusKey    []byte
	maxPeers    int64
	pingPeriod  int64
	peers       []*Peer
	minDistance *big.Int
	maxDistance *big.Int
}

// Do a merge sort on two KBucketEntry arrays
func (store *PeerStore) mergeSort(bucketA, bucketB []*Peer) []*Peer {
	output := make([]*Peer, 0)

	// Atomic sub array
	if len(bucketA) == 0 && len(bucketB) == 0 {
		return output
	} else if len(bucketA) == 1 && len(bucketB) == 0 {
		return bucketA
	} else if len(bucketB) == 1 && len(bucketA) == 0 {
		return bucketB
	}

	// Sort bucketA
	midPointA := int(math.Ceil(float64(len(bucketA)) / 2))
	sortedA := store.mergeSort(bucketA[:midPointA], bucketA[midPointA:])

	// Sort bucketB
	midPointB := int(math.Ceil(float64(len(bucketB)) / 2))
	sortedB := store.mergeSort(bucketB[:midPointB], bucketB[midPointB:])

	// Merge arrays
	for i, j := 0, 0; i < len(sortedA) || j < len(sortedB); {
		currA, currB := math.MaxInt, math.MaxInt
		if i < len(sortedA) {
			currA = int(sortedA[i].lastSeen)
		}
		if j < len(sortedB) {
			currB = int(sortedB[j].lastSeen)
		}

		if currA <= currB {
			output = append(output, sortedA[i])
			i++
		} else {
			output = append(output, sortedB[j])
			j++
		}
	}

	return output
}

// Sort the PeerStore from recently seen to least recently seen peer
func (store *PeerStore) sort() {
	store.peers = store.mergeSort(
		store.peers,
		make([]*Peer, 0),
	)
}

// Remove a peer
func (store *PeerStore) Remove(key []byte) error {
	// Get the peer index
	peerIndex := -1
	for index, peer := range store.peers {
		if bytes.Equal(peer.key, key) {
			peerIndex = index
			break
		}
	}
	if peerIndex == -1 {
		return fmt.Errorf("peer not found")
	}

	// Splice the peer to be removed
	partA := store.peers[:peerIndex]
	partB := store.peers[peerIndex+1:]
	store.peers = make([]*Peer, 0)
	store.peers = append(store.peers, partA...)
	store.peers = append(store.peers, partB...)

	return nil
}

// Insert/update a peer in the store and k-bucket
// Returns true if updated/inserted
func (store *PeerStore) Insert(
	key []byte,
	ipAddress string,
	port int,
) bool {
	// If the peer is already in the store
	entryIndex := -1
	for index, peer := range store.peers {
		if bytes.Equal(peer.key, key) {
			entryIndex = index
			break
		}
	}
	if entryIndex != -1 {
		peer := store.peers[entryIndex]
		peer.ipAddress = ipAddress
		peer.port = port
		peer.lastSeen = time.Now().Unix()
		return true
	}

	// New peer
	peer := &Peer{key, ipAddress, port, time.Now().Unix()}
	distanceFromLocus, err := peer.Distance(store.locusKey)
	if err != nil {
		return false
	} else if distanceFromLocus.Cmp(store.minDistance) == -1 {
		// Distance from locus is less than minimum allowed distance
		return false
	} else if distanceFromLocus.Cmp(store.maxDistance) == 1 {
		// Distance from locus is more than maximum allowed distance
		return false
	}

	// If the store is not full, append the new entry
	if len(store.peers) < int(store.maxPeers) {
		store.peers = append(store.peers, peer)
		return true
	}

	// If the least recently seen peer hasn't been seen in over ping period seconds replace it
	store.sort()
	leastSeenPeer := store.peers[len(store.peers)-1]
	if time.Now().Unix()-leastSeenPeer.lastSeen > store.pingPeriod {
		store.peers[len(store.peers)-1] = peer
		return true
	}

	return false
}

// Gets a peer by it's key if it exists
func (store *PeerStore) Get(key []byte) *Peer {
	for _, peer := range store.peers {
		if bytes.Equal(peer.key, key) {
			return peer
		}
	}
	return nil
}

// Get a sorted list of stored peers
func (store *PeerStore) Peers() []*Peer {
	store.sort()
	return store.peers
}

// locusKey: the host node's peer key
// maxPeers: must be >= 1
// pingPeriod: <= 0 means that new peers will always be inserted into the store
func NewPeerStore(
	locusKey []byte,
	maxPeers int64,
	pingPeriod int64,
) (*PeerStore, error) {
	if maxPeers < 1 {
		return nil, fmt.Errorf("max peers must be >= 1")
	}

	// Calculate min and max allowed distance from locus
	locusNo := big.NewInt(0).SetBytes(locusKey)
	minDistance := new(big.Int).Exp(
		big.NewInt(2),
		locusNo,
		nil,
	)
	maxDistance := new(big.Int).Exp(
		big.NewInt(2),
		new(big.Int).Add(locusNo, big.NewInt(1)),
		nil,
	)

	store := &PeerStore{
		locusKey:    locusKey,
		maxPeers:    maxPeers,
		pingPeriod:  pingPeriod,
		peers:       make([]*Peer, 0),
		minDistance: minDistance,
		maxDistance: maxDistance,
	}
	return store, nil
}
