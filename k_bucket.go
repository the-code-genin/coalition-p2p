package framework

import (
	"bytes"
	"fmt"
	"math"
	"time"
)

// A KBucket Entry
type KBucketEntry struct {
	key      []byte
	lastSeen int64
}

func (entry KBucketEntry) Key() []byte {
	return entry.key
}

func (entry KBucketEntry) LastSeen() int64 {
	return entry.lastSeen
}

// K Bucket implementation
type KBucket struct {
	replication   int64
	pingPeriod    int64
	bucketEntries []KBucketEntry
}

// Do a merge sort on two KBucketEntry arrays
func (bucket *KBucket) mergeSort(bucketA, bucketB []KBucketEntry) []KBucketEntry {
	output := make([]KBucketEntry, 0)

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
	sortedA := bucket.mergeSort(bucketA[:midPointA], bucketA[midPointA:])

	// Sort bucketB
	midPointB := int(math.Ceil(float64(len(bucketB)) / 2))
	sortedB := bucket.mergeSort(bucketB[:midPointB], bucketB[midPointB:])

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

// Sort the KBucket from recently seen to least recently seen
func (bucket *KBucket) sort() {
	bucket.bucketEntries = bucket.mergeSort(
		bucket.bucketEntries,
		make([]KBucketEntry, 0),
	)
}

// Return the sorted bucket entries from recently seen to least recently seen
func (bucket *KBucket) Entries() []KBucketEntry {
	bucket.sort()
	return bucket.bucketEntries
}

// Insert a new KBucketEntry
// Returns true if inserted/updated
func (bucket *KBucket) Insert(key []byte) bool {
	// If the entry is already in the K-Bucket
	entryIndex := -1
	for index, entry := range bucket.bucketEntries {
		if bytes.Equal(entry.key, key) {
			entryIndex = index
			break
		}
	}
	if entryIndex != -1 {
		entry := bucket.bucketEntries[entryIndex]
		entry.lastSeen = time.Now().Unix()
		return true
	}

	// A new entry
	entry := KBucketEntry{key, time.Now().Unix()}

	// If the K-Bucket is not full, append the new entry
	if len(bucket.bucketEntries) < int(bucket.replication) {
		bucket.bucketEntries = append(bucket.bucketEntries, entry)
		return true
	}

	// If the least recently seen entry hasn't been seen in over ping period seconds replace it
	bucket.sort()
	leastSeenEntry := bucket.bucketEntries[len(bucket.bucketEntries)-1]
	if time.Now().Unix()-leastSeenEntry.lastSeen > bucket.pingPeriod {
		bucket.bucketEntries[len(bucket.bucketEntries)-1] = entry
		return true
	}

	return false
}

// Remove a KBucketEntry
func (bucket *KBucket) Remove(key []byte) error {
	// Check if the entry is in the K-Bucket
	entryIndex := -1
	for index, entry := range bucket.bucketEntries {
		if bytes.Equal(entry.key, key) {
			entryIndex = index
			break
		}
	}
	if entryIndex == -1 {
		return fmt.Errorf("k-bucket entry not found")
	}

	// Splice the entry to be removed
	partA := bucket.bucketEntries[:entryIndex]
	partB := bucket.bucketEntries[entryIndex+1:]
	bucket.bucketEntries = make([]KBucketEntry, 0)
	bucket.bucketEntries = append(bucket.bucketEntries, partA...)
	bucket.bucketEntries = append(bucket.bucketEntries, partB...)

	return nil
}

// Replication must be >= 1
// A ping period <= 0 means that new entries will always be inserted into the bucket
func NewKBucket(
	replication int64,
	pingPeriod int64,
) (*KBucket, error) {
	if replication < 1 {
		return nil, fmt.Errorf("replication parameter must be >= 1")
	}

	bucket := &KBucket{
		replication:   replication,
		pingPeriod:    pingPeriod,
		bucketEntries: make([]KBucketEntry, 0),
	}
	return bucket, nil
}
