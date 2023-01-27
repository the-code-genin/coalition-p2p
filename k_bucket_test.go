package framework

import (
	"crypto/rand"
	"testing"
	"time"
)

func TestKBucketStorage(t *testing.T) {
	replicationParam := int64(20)
	pingPeriod := int64(time.Hour.Seconds())
	bucket, err := NewKBucket(replicationParam, pingPeriod)
	if err != nil {
		t.Error(err)
	}

	// Add three times the number of replication entries
	for i := int64(0); i < replicationParam*3; i++ {
		key := make([]byte, PeerIDSize)
		if _, err := rand.Read(key); err != nil {
			t.Error(err)
		}

		if bucket.Insert(key, i) && i >= replicationParam {
			t.Errorf("A new entry should not be inserted")
		}
	}
}

func TestKBucketStorageExpiry(t *testing.T) {
	replicationParam := int64(20)
	pingPeriod := int64(-1)
	bucket, err := NewKBucket(replicationParam, pingPeriod)
	if err != nil {
		t.Error(err)
	}

	// Add three times the number of replication entries
	for i := int64(0); i < replicationParam*3; i++ {
		key := make([]byte, PeerIDSize)
		if _, err := rand.Read(key); err != nil {
			t.Error(err)
		}

		if !bucket.Insert(key, i) {
			t.Errorf("A new entry should be inserted")
		}
	}
}
