package cuckoo

import (
	"bytes"
	"fmt"
)

// fingerprint represents a single entry in a bucket.
type fingerprint uint16

// bucket keeps track of fingerprints hashing to the same index.
type bucket [bucketSize]fingerprint

const (
	nullFp              = 0
	bucketSize          = 4
	fingerprintSizeBits = 16
	maxFingerprint      = (1 << fingerprintSizeBits) - 1
)

// insert a fingerprint into a bucket. Returns true if there was enough space and insertion succeeded.
// Note it allows inserting the same fingerprint multiple times.
func (b *bucket) insert(fp fingerprint) bool {
	if i := b.index(nullFp); i != 4 {
		b[i] = fp
		return true
	}
	return false
}

// delete a fingerprint from a bucket.
// Returns true if the fingerprint was present and successfully removed.
func (b *bucket) delete(fp fingerprint) bool {
	if i := b.index(fp); i != 4 {
		b[i] = nullFp
		return true
	}
	return false
}

func (b *bucket) contains(needle fingerprint) bool {
	return b.index(needle) != 4
}

func (b *bucket) index(needle fingerprint) uint8 {
	if b[0] == needle {
		return 0
	}
	if b[1] == needle {
		return 1
	}
	if b[2] == needle {
		return 2
	}
	if b[3] == needle {
		return 3
	}
	return 4
}

// reset deletes all fingerprints in the bucket.
func (b *bucket) reset() {
	*b = [bucketSize]fingerprint{nullFp, nullFp, nullFp, nullFp}
}

func (b *bucket) String() string {
	var buf bytes.Buffer
	buf.WriteString("[")
	for _, by := range b {
		buf.WriteString(fmt.Sprintf("%5d ", by))
	}
	buf.WriteString("]")
	return buf.String()
}
