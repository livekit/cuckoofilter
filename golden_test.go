package cuckoo

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/rand"
	"testing"
)

// goldenHash was produced by master@2afb68a. A mismatch means filters encoded
// by earlier versions will no longer decode or look up correctly.
const goldenHash = "03cacb3758df8c768f324a50368195a4c1030f655320aa71df752d992f1ed224"

func TestGolden(t *testing.T) {
	h := sha256.New()
	rng := rand.New(rand.NewSource(1))
	key := func() []byte {
		b := make([]byte, 16+rng.Intn(240))
		for i := range b {
			b[i] = byte(rng.Uint32())
		}
		return b
	}
	result := func(ok bool) {
		if ok {
			h.Write([]byte{1})
		} else {
			h.Write([]byte{0})
		}
	}

	for _, size := range []uint{8, 1000, 100000} {
		f := NewFilter(size)
		var keys [][]byte
		// Overfill to exercise kickouts and failed inserts.
		for i := 0; i < int(size)*11/10; i++ {
			k := key()
			keys = append(keys, k)
			result(f.Insert(k))
		}
		for i := 0; i < len(keys); i += 3 {
			result(f.Delete(keys[i]))
		}
		for i := 0; i < 20000; i++ {
			k := key()
			if i%2 == 0 {
				k = keys[rng.Intn(len(keys))]
			}
			result(f.Lookup(k))
		}
		h.Write(f.Encode())
		h.Write(binary.LittleEndian.AppendUint64(nil, uint64(f.Count())))
	}

	if got := fmt.Sprintf("%x", h.Sum(nil)); got != goldenHash {
		t.Errorf("golden hash = %s, want %s", got, goldenHash)
	}
}
