package coalition

import (
	"bytes"
	"encoding/hex"
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

// Return the peer address
func (peer *Peer) Address() string {
	return fmt.Sprintf("%s:%d", peer.ipAddress, peer.port)
}

// Return the peer's last seen timestamp
func (peer *Peer) LastSeen() int64 {
	return peer.lastSeen
}

// The peer store manages a kbucket of network peers
type PeerStore struct {
	locusKey      []byte
	maxPeers      int64
	latencyPeriod int64
	peers         []*Peer
	kbucket       map[string][][]byte
}

// Do a merge sort on two Peer arrays
func (store *PeerStore) mergeSortPeers(
	bucketA, bucketB []*Peer,
	sortFunc func(*Peer, *Peer) int,
) []*Peer {
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
	sortedA := store.mergeSortPeers(bucketA[:midPointA], bucketA[midPointA:], sortFunc)

	// Sort bucketB
	midPointB := int(math.Ceil(float64(len(bucketB)) / 2))
	sortedB := store.mergeSortPeers(bucketB[:midPointB], bucketB[midPointB:], sortFunc)

	// Merge arrays
	for i, j := 0, 0; i < len(sortedA) || j < len(sortedB); {
		var peerA, peerB *Peer
		if i < len(sortedA) {
			peerA = sortedA[i]
		}
		if j < len(sortedB) {
			peerB = sortedB[j]
		}

		// If either array has been exhausted
		if peerA == nil {
			output = append(output, peerB)
			j++
			continue
		} else if peerB == nil {
			output = append(output, peerA)
			i++
			continue
		}

		// Compare the Peers
		if res := sortFunc(peerA, peerB); res >= 0 {
			// PeerB is greater than or equal to peerA
			output = append(output, peerA)
			i++
		} else {
			// PeerA is greater than peerB
			output = append(output, peerB)
			j++
		}
	}

	return output
}

// Calculate the KBucket key for a peer as an hex value
func (store *PeerStore) calculateKBucketKey(key []byte) (string, error) {
	if len(key) != len(store.locusKey) {
		return "", fmt.Errorf("key length miss-match")
	}
	distanceFromLocus := new(big.Int).Xor(
		new(big.Int).SetBytes(store.locusKey),
		new(big.Int).SetBytes(key),
	)
	noBits := len(store.locusKey) * 8
	for i := int64(noBits - 1); i >= 0; i-- {
		key := new(big.Int).Exp(
			new(big.Int).SetInt64(2),
			new(big.Int).SetInt64(i),
			nil,
		)
		if new(big.Int).And(distanceFromLocus, key).Cmp(key) == 0 {
			return hex.EncodeToString(key.Bytes()), nil
		}
	}
	return hex.EncodeToString([]byte{0}), nil
}

// Sort the PeerStore from recently seen to least recently seen peer
func (store *PeerStore) SortPeersByLastSeen() []*Peer {
	peers := store.mergeSortPeers(
		store.peers,
		make([]*Peer, 0),
		func(peerA, peerB *Peer) int {
			return int(peerB.lastSeen - peerA.lastSeen)
		},
	)
	return peers
}

// Sort the PeerStore by proximity to a certain key
// From closest to farthest
func (store *PeerStore) SortPeersByProximity(key []byte) ([]*Peer, error) {
	if len(key) != len(store.locusKey) {
		return nil, fmt.Errorf("key length miss-match")
	}
	peers := store.mergeSortPeers(
		store.peers,
		make([]*Peer, 0),
		func(peerA, peerB *Peer) int {
			distanceA := new(big.Int).Xor(
				new(big.Int).SetBytes(peerA.key),
				new(big.Int).SetBytes(key),
			)
			distanceB := new(big.Int).Xor(
				new(big.Int).SetBytes(peerB.key),
				new(big.Int).SetBytes(key),
			)
			return int(new(big.Int).Sub(distanceB, distanceA).Int64())
		},
	)
	return peers, nil
}

// Remove a peer
func (store *PeerStore) Remove(key []byte) error {
	// Get the peer index in the peer list
	peerIndex := -1
	for index, peer := range store.peers {
		if bytes.Equal(peer.key, key) {
			peerIndex = index
			break
		}
	}
	if peerIndex == -1 {
		return fmt.Errorf("peer not found in peers list")
	}

	// Calculate the peer kbucket key
	peer := store.peers[peerIndex]
	bucketKey, err := store.calculateKBucketKey(peer.key)
	if err != nil {
		return err
	}
	if _, exists := store.kbucket[bucketKey]; !exists {
		return fmt.Errorf("kbucket entry does not exist for peer")
	}

	// Calculate the peer key index in the kbucket entry
	peerKeyIndex := -1
	for index, peerKey := range store.kbucket[bucketKey] {
		if bytes.Equal(peer.key, peerKey) {
			peerKeyIndex = index
			break
		}
	}
	if peerKeyIndex == -1 {
		return fmt.Errorf("peer key not found in kbucket")
	}

	// Splice the peer key from the kbucket entry
	keysA := store.kbucket[bucketKey][:peerKeyIndex]
	keysB := store.kbucket[bucketKey][peerKeyIndex+1:]
	store.kbucket[bucketKey] = make([][]byte, 0)
	store.kbucket[bucketKey] = append(store.kbucket[bucketKey], keysA...)
	store.kbucket[bucketKey] = append(store.kbucket[bucketKey], keysB...)

	// Splice the peer to be removed from the peer list
	peersA := store.peers[:peerIndex]
	peersB := store.peers[peerIndex+1:]
	store.peers = make([]*Peer, 0)
	store.peers = append(store.peers, peersA...)
	store.peers = append(store.peers, peersB...)

	return nil
}

// Insert/update a peer. If the peer already exists in the store, it's last seen is updated.
// A peer will not be inserted if it does not meet the kbucket insertion rules.
// Returns true if peer updated/inserted successfully.
func (store *PeerStore) Insert(
	key []byte,
	ipAddress string,
	port int,
) (bool, error) {
	// If the peer is already in the store
	peerIndex := -1
	for index, peer := range store.peers {
		if bytes.Equal(peer.key, key) {
			peerIndex = index
			break
		}
	}
	if peerIndex != -1 {
		peer := store.peers[peerIndex]
		peer.ipAddress = ipAddress
		peer.port = port
		peer.lastSeen = time.Now().Unix()
		return true, nil
	}

	// Create new peer and calculate it's bucket key
	// Create the bucket entry if it does not exist yet
	peer := &Peer{key, ipAddress, port, time.Now().Unix()}
	bucketKey, err := store.calculateKBucketKey(peer.key)
	if err != nil {
		return false, err
	}
	if _, exists := store.kbucket[bucketKey]; !exists {
		store.kbucket[bucketKey] = make([][]byte, 0)
	}

	// If the store is not full, append the new entry
	if len(store.peers) < int(store.maxPeers) {
		store.peers = append(store.peers, peer)
		store.kbucket[bucketKey] = append(store.kbucket[bucketKey], peer.key)
		return true, nil
	}

	// If it's kbucket entry is empty but the store is full
	// Find a bloated entry to prune to make space for the new peer
	if len(store.kbucket[bucketKey]) == 0 {
		pruned := false
		for _, entries := range store.kbucket {
			// Not a bloated entry
			if len(entries) <= 1 {
				continue
			}

			// Remove the last peer in the entry
			entryPeerKey := entries[len(entries)-1]
			if err := store.Remove(entryPeerKey); err != nil {
				return false, err
			}
			pruned = true
			break
		}

		// If a bloat node was pruned
		if pruned {
			store.peers = append(store.peers, peer)
			store.kbucket[bucketKey] = append(store.kbucket[bucketKey], peer.key)
			return true, nil
		}
	}

	// If the least recently seen peer hasn't been seen in over ping period seconds replace it
	peers := store.SortPeersByLastSeen()
	leastSeenPeer := peers[len(peers)-1]
	if time.Now().Unix()-leastSeenPeer.lastSeen > store.latencyPeriod {
		if err := store.Remove(leastSeenPeer.key); err != nil {
			return false, err
		}
		store.peers = append(store.peers, peer)
		store.kbucket[bucketKey] = append(store.kbucket[bucketKey], peer.key)
		return true, nil
	}

	return false, nil
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

// Get a the list of stored peers
func (store *PeerStore) Peers() []*Peer {
	return store.peers
}

// locusKey: the host node's peer key
// maxPeers: the kbucket replication parameter
// latencyPeriod: grace period in seconds before the node is considered offline
func NewPeerStore(
	locusKey []byte,
	maxPeers int64,
	latencyPeriod int64,
) (*PeerStore, error) {
	if maxPeers < 1 {
		return nil, fmt.Errorf("max peers must be >= 1")
	}

	store := &PeerStore{
		locusKey:      locusKey,
		maxPeers:      maxPeers,
		latencyPeriod: latencyPeriod,
		peers:         make([]*Peer, 0),
		kbucket:       make(map[string][][]byte),
	}
	return store, nil
}
