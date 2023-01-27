package framework

import (
	"bytes"
	"fmt"
)

// A network peer
type Peer struct {
	key       []byte
	ipAddress string
	port      int
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

// Return the peer's XOR distance from a key
func (peer *Peer) Distance(key []byte) int64 {
	return 0
}

type PeerStore struct {
	bucket *KBucket
	peers  []*Peer
}

// Remove a peer
func (store *PeerStore) Remove(key []byte) error {
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

	// Unable to remove the key in the bucket
	if err := store.bucket.Remove(key); err != nil {
		return err
	}

	// Splice the entry to be removed
	partA := store.peers[:peerIndex]
	partB := store.peers[peerIndex+1:]
	store.peers = make([]*Peer, 0)
	store.peers = append(store.peers, partA...)
	store.peers = append(store.peers, partB...)

	return nil
}

// Insert/update a peer in the store and k-bucket
// Returns true if updated/inserted
// Returns an error if there's a processing error
func (store *PeerStore) Insert(
	key []byte,
	ipAddress string,
	port int,
) (bool, error) {
	// If the peer is already in the store
	entryIndex := -1
	for index, peer := range store.peers {
		if bytes.Equal(peer.key, key) {
			entryIndex = index
			break
		}
	}
	if entryIndex != -1 {
		// Update peer information
		peer := store.peers[entryIndex]
		peer.ipAddress = ipAddress
		peer.port = port

		// Update the k-bucket
		if !store.bucket.Insert(peer.key) {
			return false, fmt.Errorf("unable to update peer in k-bucket")
		}
		return true, nil
	}

	// New peer
	peer := &Peer{key, ipAddress, port}
	if !store.bucket.Insert(peer.key) {
		return false, nil
	}
	store.peers = append(store.peers, peer)
	return true, nil
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

// Get list of stored peers
func (store *PeerStore) Peers() []*Peer {
	return store.peers
}

func NewPeerStore(
	maxPeers int64,
	pingPeriod int64,
) (*PeerStore, error) {
	bucket, err := NewKBucket(maxPeers, pingPeriod)
	if err != nil {
		return nil, err
	}

	store := &PeerStore{
		bucket: bucket,
		peers:  make([]*Peer, 0),
	}
	return store, nil
}
