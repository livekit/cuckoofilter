package cuckoo

import (
	"math/bits"

	"github.com/zeebo/xxh3"
)

// altHash is xxh3.Hash of fp as 2 little-endian bytes, specialized for that
// input length. It must match xxh3 exactly or previously encoded filters break.
func altHash(fp fingerprint) uint {
	const (
		key32_000 = 0xbe4ba423
		key32_004 = 0x396cfeb8
		prime64_2 = 14029467366897019727
		prime64_3 = 1609587929392839161
	)
	x := uint64(fp)*(1<<24+1)>>8 + 2<<8
	x ^= key32_000 ^ key32_004
	x ^= x >> 33
	x *= prime64_2
	x ^= x >> 29
	x *= prime64_3
	x ^= x >> 32
	return uint(x)
}

func getAltIndex(fp fingerprint, i uint, bucketIndexMask uint) uint {
	return (i ^ altHash(fp)) & bucketIndexMask
}

func getFingerprint(hash uint64) fingerprint {
	// Use most significant bits for fingerprint.
	shifted := hash >> (64 - fingerprintSizeBits)
	// Valid fingerprints are in range [1, maxFingerprint], leaving 0 as the special empty state.
	fp := shifted%(maxFingerprint-1) + 1
	return fingerprint(fp)
}

// getIndexAndFingerprint returns the primary bucket index and fingerprint to be used
func getIndexAndFingerprint(data []byte, bucketIndexMask uint) (uint, fingerprint) {
	hash := xxh3.Hash(data)
	f := getFingerprint(hash)
	// Use least significant bits for deriving index.
	i1 := uint(hash) & bucketIndexMask
	return i1, f
}

func getNextPow2(n uint64) uint {
	return uint(1 << bits.Len64(n-1))
}

// SEE: https://graphics.stanford.edu/~seander/bithacks.html#ZeroInWord
func findZeros(v uint64) uint64 {
	return ^((((v & 0x7FFF7FFF7FFF7FFF) + 0x7FFF7FFF7FFF7FFF) | v) | 0x7FFF7FFF7FFF7FFF)
}

// SEE: https://graphics.stanford.edu/~seander/bithacks.html#ValueInWord
func findValue(x uint64, n uint16) uint64 {
	return findZeros(x ^ (^uint64(0) / (1<<16 - 1) * uint64(n)))
}
