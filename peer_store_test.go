package coalition

import (
	// "bytes"
	"crypto/rand"
	"testing"
	"time"
)

func TestPeerStorage(t *testing.T) {
	locusKey := make([]byte, PeerKeySize)
	if _, err := rand.Read(locusKey); err != nil {
		t.Error(err)
	}
	maxPeers := int64(20)
	pingPeriod := int64(time.Hour.Seconds())
	store, err := NewPeerStore(locusKey, maxPeers, pingPeriod)
	if err != nil {
		t.Error(err)
	}

	// Add three times the number of replication entries
	for i := int64(0); i < maxPeers*3; i++ {
		key := make([]byte, PeerKeySize)
		if _, err := rand.Read(key); err != nil {
			t.Error(err)
		}

		inserted, err := store.Insert(key, "0.0.0.0", int(i))
		if err != nil {
			t.Error(err)
		}
		if inserted && len(store.Peers()) > int(maxPeers) {
			t.Errorf("A new peer should not be inserted over the max")
		}
	}

	if len(store.Peers()) != int(maxPeers) {
		t.Errorf("store should have %d peers", maxPeers)
	}
}

// func TestPeerStoreExpiry(t *testing.T) {
// 	locusKey := make([]byte, PeerKeySize)
// 	if _, err := rand.Read(locusKey); err != nil {
// 		t.Error(err)
// 	}
// 	maxPeers := int64(20)
// 	pingPeriod := int64(-1)
// 	store, err := NewPeerStore(locusKey, maxPeers, pingPeriod)
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	// Add three times the number of replication entries
// 	for i := int64(0); i < maxPeers*3; i++ {
// 		key := make([]byte, PeerKeySize)
// 		if _, err := rand.Read(key); err != nil {
// 			t.Error(err)
// 		}

// 		if !store.Insert(key, "0.0.0.0", int(i)) {
// 			t.Errorf("A new peer should be inserted")
// 		}
// 	}
// }

// func TestPeerStoreRemove(t *testing.T) {
// 	locusKey := make([]byte, PeerKeySize)
// 	if _, err := rand.Read(locusKey); err != nil {
// 		t.Error(err)
// 	}
// 	maxPeers := int64(20)
// 	pingPeriod := int64(time.Hour.Seconds())
// 	store, err := NewPeerStore(locusKey, maxPeers, pingPeriod)
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	// Populate the bucket
// 	for i := int64(0); i < maxPeers; i++ {
// 		key := make([]byte, PeerKeySize)
// 		if _, err := rand.Read(key); err != nil {
// 			t.Error(err)
// 		}

// 		if !store.Insert(key, "0.0.0.0", int(i)) {
// 			t.Errorf("A new peer should be inserted")
// 		}
// 	}

// 	peers := store.Peers()
// 	store.Remove(peers[0].key)
// 	if len(store.Peers()) != len(peers)-1 {
// 		t.Errorf("bucket should have %d entries", len(peers)-1)
// 	} else if bytes.Equal(store.Peers()[0].key, peers[0].key) {
// 		t.Errorf("bucket entry not deleted")
// 	}
// }
