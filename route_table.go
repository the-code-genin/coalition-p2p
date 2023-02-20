package coalition

import (
	"bytes"
	"encoding/hex"
	"fmt"
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

// Return the peer ip4 Address
func (peer *Peer) IPAddress() string {
	return peer.ipAddress
}

// Return the peer listening port
func (peer *Peer) Port() int {
	return peer.port
}

// Return the peer address
func (peer *Peer) Address() (string, error) {
	return FormatNodeAddress(peer.key, peer.ipAddress, peer.port)
}

// Return the peer's last seen timestamp
func (peer *Peer) LastSeen() int64 {
	return peer.lastSeen
}

// Create a new peer from the peer details
func NewPeer(key []byte, ipAddress string, port int) *Peer {
	return &Peer{
		key,
		ipAddress,
		port,
		time.Now().Unix(),
	}
}

// Create a new peer from a peer address
func NewPeerFromAddress(address string) (*Peer, error) {
	key, ip4Address, port, err := ParseNodeAddress(address)
	if err != nil {
		return nil, err
	}
	peer := &Peer{
		key,
		ip4Address,
		port,
		time.Now().Unix(),
	}
	return peer, nil
}

// The route table manages an optimized kbucket of network peers
type RouteTable struct {
	locusKey      []byte
	maxPeers      int64
	latencyPeriod int64
	peers         []*Peer
	kbucket       map[string][][]byte
}

// Calculate the KBucket key for a peer as an hex value
func (table *RouteTable) calculateKBucketKey(key []byte) (string, error) {
	if len(key) != len(table.locusKey) {
		return "", fmt.Errorf("key length miss-match")
	}
	distanceFromLocus := new(big.Int).Xor(
		new(big.Int).SetBytes(table.locusKey),
		new(big.Int).SetBytes(key),
	)
	noBits := len(table.locusKey) * 8
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

// Sort the peers in the route table
// From recently seen to least recently seen peer
func (table *RouteTable) SortPeersByLastSeen() []*Peer {
	peers := MergeSortPeers(
		table.peers,
		make([]*Peer, 0),
		func(peerA, peerB *Peer) int {
			return int(peerB.lastSeen - peerA.lastSeen)
		},
	)
	return peers
}

// Sort the peers in the route table by proximity to a certain key
// From closest to farthest
func (table *RouteTable) SortPeersByProximity(key []byte) ([]*Peer, error) {
	if len(key) != len(table.locusKey) {
		return nil, fmt.Errorf("key length miss-match")
	}
	return SortPeersByClosest(table.peers, key), nil
}

// Remove a peer from the route table
func (table *RouteTable) Remove(key []byte) error {
	// Get the peer index in the peer list
	peerIndex := -1
	for index, peer := range table.peers {
		if bytes.Equal(peer.key, key) {
			peerIndex = index
			break
		}
	}
	if peerIndex == -1 {
		return fmt.Errorf("peer not found in peers list")
	}

	// Calculate the peer kbucket key
	peer := table.peers[peerIndex]
	bucketKey, err := table.calculateKBucketKey(peer.key)
	if err != nil {
		return err
	}
	if _, exists := table.kbucket[bucketKey]; !exists {
		return fmt.Errorf("kbucket entry does not exist for peer")
	}

	// Calculate the peer key index in the kbucket entry
	peerKeyIndex := -1
	for index, peerKey := range table.kbucket[bucketKey] {
		if bytes.Equal(peer.key, peerKey) {
			peerKeyIndex = index
			break
		}
	}
	if peerKeyIndex == -1 {
		return fmt.Errorf("peer key not found in kbucket")
	}

	// Splice the peer key from the kbucket entry
	keysA := table.kbucket[bucketKey][:peerKeyIndex]
	keysB := table.kbucket[bucketKey][peerKeyIndex+1:]
	table.kbucket[bucketKey] = make([][]byte, 0)
	table.kbucket[bucketKey] = append(table.kbucket[bucketKey], keysA...)
	table.kbucket[bucketKey] = append(table.kbucket[bucketKey], keysB...)

	// Splice the peer to be removed from the peer list
	peersA := table.peers[:peerIndex]
	peersB := table.peers[peerIndex+1:]
	table.peers = make([]*Peer, 0)
	table.peers = append(table.peers, peersA...)
	table.peers = append(table.peers, peersB...)

	return nil
}

// Insert/update a peer. If the peer already exists in the table, it's last seen is updated.
// A peer will not be inserted if it does not meet the kbucket insertion rules.
// Returns true if peer updated/inserted successfully.
func (table *RouteTable) Insert(
	key []byte,
	ipAddress string,
	port int,
) (bool, error) {
	// Skip inserts for the same node
	if bytes.Equal(key, table.locusKey) {
		return false, nil
	}

	// If the peer is already in the table
	peerIndex := -1
	for index, peer := range table.peers {
		if bytes.Equal(peer.key, key) {
			peerIndex = index
			break
		}
	}
	if peerIndex != -1 {
		peer := table.peers[peerIndex]
		peer.ipAddress = ipAddress
		peer.port = port
		peer.lastSeen = time.Now().Unix()
		return true, nil
	}

	// Create new peer and calculate it's bucket key
	// Create the bucket entry if it does not exist yet
	peer := NewPeer(key, ipAddress, port)
	bucketKey, err := table.calculateKBucketKey(peer.key)
	if err != nil {
		return false, err
	}
	if _, exists := table.kbucket[bucketKey]; !exists {
		table.kbucket[bucketKey] = make([][]byte, 0)
	}

	// If the table is not full, append the new entry
	if len(table.peers) < int(table.maxPeers) {
		table.peers = append(table.peers, peer)
		table.kbucket[bucketKey] = append(table.kbucket[bucketKey], peer.key)
		return true, nil
	}

	// If it's kbucket entry is empty but the table is full
	// Find a bloated entry to prune to make space for the new peer
	if len(table.kbucket[bucketKey]) == 0 {
		pruned := false
		for _, entries := range table.kbucket {
			// Not a bloated entry
			if len(entries) <= 1 {
				continue
			}

			// Remove the last peer in the entry
			entryPeerKey := entries[len(entries)-1]
			if err := table.Remove(entryPeerKey); err != nil {
				return false, err
			}
			pruned = true
			break
		}

		// If a bloat node was pruned
		if pruned {
			table.peers = append(table.peers, peer)
			table.kbucket[bucketKey] = append(table.kbucket[bucketKey], peer.key)
			return true, nil
		}
	}

	// If the least recently seen peer hasn't been seen in over ping period seconds replace it
	peers := table.SortPeersByLastSeen()
	leastSeenPeer := peers[len(peers)-1]
	if time.Now().Unix()-leastSeenPeer.lastSeen > table.latencyPeriod {
		if err := table.Remove(leastSeenPeer.key); err != nil {
			return false, err
		}
		table.peers = append(table.peers, peer)
		table.kbucket[bucketKey] = append(table.kbucket[bucketKey], peer.key)
		return true, nil
	}

	return false, nil
}

// Gets a peer by it's key if it exists
func (table *RouteTable) Get(key []byte) *Peer {
	for _, peer := range table.peers {
		if bytes.Equal(peer.key, key) {
			return peer
		}
	}
	return nil
}

// Get a the list of stored peers
func (table *RouteTable) Peers() []*Peer {
	return table.peers
}

// locusKey: the host node's peer key
// maxPeers: the kbucket replication parameter
// latencyPeriod: grace period in seconds before the node is considered offline
func NewRouteTable(
	locusKey []byte,
	maxPeers int64,
	latencyPeriod int64,
) (*RouteTable, error) {
	if maxPeers < 1 {
		return nil, fmt.Errorf("max peers must be >= 1")
	}

	table := &RouteTable{
		locusKey:      locusKey,
		maxPeers:      maxPeers,
		latencyPeriod: latencyPeriod,
		peers:         make([]*Peer, 0),
		kbucket:       make(map[string][][]byte),
	}
	return table, nil
}
