package framework

import (
	"bytes"
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
		key := make([]byte, PeerKeySize)
		if _, err := rand.Read(key); err != nil {
			t.Error(err)
		}

		if bucket.Insert(key) && i >= replicationParam {
			t.Errorf("A new entry should not be inserted")
		}
	}

	if len(bucket.Entries()) != int(replicationParam) {
		t.Errorf("bucket should have %d entries", replicationParam)
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
		key := make([]byte, PeerKeySize)
		if _, err := rand.Read(key); err != nil {
			t.Error(err)
		}

		if !bucket.Insert(key) {
			t.Errorf("A new entry should be inserted")
		}
	}
}

func TestKBucketRemoval(t *testing.T) {
	replicationParam := int64(20)
	pingPeriod := int64(time.Hour.Seconds())
	bucket, err := NewKBucket(replicationParam, pingPeriod)
	if err != nil {
		t.Error(err)
	}

	// Populate the bucket
	for i := int64(0); i < replicationParam; i++ {
		key := make([]byte, PeerKeySize)
		if _, err := rand.Read(key); err != nil {
			t.Error(err)
		}

		if !bucket.Insert(key) {
			t.Errorf("A new entry should be inserted")
		}
	}

	entries := bucket.Entries()
	bucket.Remove(entries[0].key)
	if len(bucket.Entries()) != len(entries)-1 {
		t.Errorf("bucket should have %d entries", len(entries)-1)
	} else if bytes.Equal(bucket.Entries()[0].key, entries[0].key) {
		t.Errorf("bucket entry not deleted")
	}
}
