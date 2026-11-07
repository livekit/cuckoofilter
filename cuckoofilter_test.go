package cuckoo

import (
	"bufio"
	"fmt"
	"math"
	"math/rand"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// optFloatNear considers float64 as equal if the relative delta is small.
var optFloatNear = cmp.Comparer(func(x, y float64) bool {
	delta := math.Abs(x - y)
	mean := math.Abs(x+y) / 2.0
	return delta/mean < 0.00001
})

func TestInsertion(t *testing.T) {
	cf := NewFilter(1000000)
	fd, err := os.Open("/usr/share/dict/words")
	if err != nil {
		t.Skipf("failed reading words: %v", err)
	}
	scanner := bufio.NewScanner(fd)

	var values [][]byte
	var lineCount uint
	for scanner.Scan() {
		s := []byte(scanner.Text())
		if cf.Insert(s) {
			lineCount++
		}
		values = append(values, s)
	}

	if got, want := cf.Count(), lineCount; got != want {
		t.Errorf("After inserting: Count() = %d, want %d", got, want)
	}
	if got, want := cf.LoadFactor(), float64(0.95); got >= want {
		t.Errorf("After inserting: LoadFactor() = %f, want less than %f.", got, want)
	}

	for _, v := range values {
		cf.Delete(v)
	}

	if got, want := cf.Count(), uint(0); got != want {
		t.Errorf("After deleting: Count() = %d, want %d", got, want)
	}
	if got, want := cf.LoadFactor(), float64(0); got != want {
		t.Errorf("After deleting: LoadFactor() = %f, want %f", got, want)
	}
}

func TestLookup(t *testing.T) {
	cf := NewFilter(4)
	cf.Insert([]byte("one"))
	cf.Insert([]byte("two"))
	cf.Insert([]byte("three"))

	testCases := []struct {
		word string
		want bool
	}{
		{"one", true},
		{"two", true},
		{"three", true},
		{"four", false},
		{"five", false},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("cf.Lookup(%q)", tc.word), func(t *testing.T) {
			t.Parallel()
			if got := cf.Lookup([]byte(tc.word)); got != tc.want {
				t.Errorf("cf.Lookup(%q) got %v, want %v", tc.word, got, tc.want)
			}
		})
	}
}

func TestFilter_LookupLarge(t *testing.T) {
	const size = 10000
	insertFail := 0
	cf := NewFilter(size)
	for i := 0; i < size; i++ {
		if !cf.Insert([]byte{byte(i)}) {
			insertFail++
		}
	}
	fn := 0
	for i := 0; i < size; i++ {
		if !cf.Lookup([]byte{byte(i)}) {
			fn++
		}
	}

	if fn != 0 {
		t.Errorf("cf.Lookup() with %d items. False negatives = %d, want 0. Insert failed %d times", size, fn, insertFail)
	}
}

func TestFilter_Insert(t *testing.T) {
	filter := NewFilter(10000)
	rng := rand.New(rand.NewSource(int64(42)))

	hash := make([]byte, 32)
	for i := 0; i < 100; i++ {
		rng.Read(hash)
		filter.Insert(hash)
	}

	if got, want := filter.Count(), uint(100); got != want {
		t.Errorf("inserting 100 items, Count() = %d, want %d", got, want)
	}
}

func BenchmarkFilter_Reset(b *testing.B) {
	const cap = 10000
	filter := NewFilter(cap)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		filter.Reset()
	}
}

const (
	benchKeyMinLen  = 16
	benchKeyMaxLen  = 256
	benchKeyMeanLen = 80
)

// benchmarkKeys returns size keys with lengths in [16, 256] averaging ~80 bytes,
// skewed toward short keys like typical identifiers.
func benchmarkKeys(b *testing.B, size int) [][]byte {
	b.Helper()
	rng := rand.New(rand.NewSource(int64(size)))
	keys := make([][]byte, size)
	// Slightly above mean-min to compensate for the tail clipped at max.
	const scale = benchKeyMeanLen - benchKeyMinLen + 1.5
	for i := range keys {
		n := benchKeyMinLen + int(rng.ExpFloat64()*scale)
		if n > benchKeyMaxLen {
			n = benchKeyMaxLen
		}
		keys[i] = make([]byte, n)
		for j := range keys[i] {
			keys[i][j] = byte('!' + rng.Intn('~'-'!'+1))
		}
	}
	return keys
}

func TestBenchmarkKeys(t *testing.T) {
	keys := benchmarkKeys(&testing.B{}, 100000)
	var total int
	for _, k := range keys {
		if len(k) < benchKeyMinLen || len(k) > benchKeyMaxLen {
			t.Fatalf("key length %d out of range", len(k))
		}
		total += len(k)
	}
	if mean := float64(total) / float64(len(keys)); math.Abs(mean-benchKeyMeanLen) > 1 {
		t.Errorf("mean key length %.2f, want ~%d", mean, benchKeyMeanLen)
	}
}

var benchSizes = []uint{10_000, 1_000_000}

func BenchmarkFilter_Insert(b *testing.B) {
	for _, size := range benchSizes {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			keys := benchmarkKeys(b, int(float64(size)*0.9))
			filter := NewFilter(size)
			b.ResetTimer()
			for i := 0; i < b.N; {
				b.StopTimer()
				filter.Reset()
				b.StartTimer()
				for _, k := range keys {
					filter.Insert(k)
				}
				i += len(keys)
			}
		})
	}
}

// lookupKeys returns a shuffled mix of present and absent keys.
func lookupKeys(b *testing.B, f *Filter, size uint) [][]byte {
	keys := benchmarkKeys(b, int(float64(size)*0.9))
	for _, k := range keys {
		f.Insert(k)
	}
	n := len(keys) / 2
	lk := append(benchmarkKeys(b, n), keys[:n]...)
	rand.New(rand.NewSource(42)).Shuffle(len(lk), func(i, j int) {
		lk[i], lk[j] = lk[j], lk[i]
	})
	return lk
}

func BenchmarkFilter_Lookup(b *testing.B) {
	for _, size := range benchSizes {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			f := NewFilter(size)
			lk := lookupKeys(b, f, size)
			b.ResetTimer()
			for i := 0; i < b.N; {
				for _, k := range lk {
					f.Lookup(k)
				}
				i += len(lk)
			}
		})
	}
}

func BenchmarkFilter_Delete(b *testing.B) {
	for _, size := range benchSizes {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			f := NewFilter(size)
			lk := lookupKeys(b, f, size)
			snapshot := append([]bucket(nil), f.buckets...)
			count := f.count
			b.ResetTimer()
			for i := 0; i < b.N; {
				b.StopTimer()
				copy(f.buckets, snapshot)
				f.count = count
				b.StartTimer()
				for _, k := range lk {
					f.Delete(k)
				}
				i += len(lk)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	cf := NewFilter(8)
	cf.Insert([]byte("one"))
	cf.Insert([]byte("two"))
	cf.Insert([]byte("three"))

	testCases := []struct {
		word string
		want bool
	}{
		{"four", false},
		{"five", false},
		{"one", true},
		{"two", true},
		{"three", true},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("cf.Delete(%q)", tc.word), func(t *testing.T) {
			if got := cf.Delete([]byte(tc.word)); got != tc.want {
				t.Errorf("cf.Delete(%q) got %v, want %v", tc.word, got, tc.want)
			}
		})
	}
}

func TestDeleteMultipleSame(t *testing.T) {
	cf := NewFilter(4)
	for i := 0; i < 5; i++ {
		if !cf.Insert([]byte("some_item")) {
			t.Error("Failed insert during setup.")
		}
	}

	testCases := []struct {
		word      string
		want      bool
		wantCount uint
	}{
		{"missing", false, 5},
		{"missing2", false, 5},
		{"some_item", true, 4},
		{"some_item", true, 3},
		{"some_item", true, 2},
		{"some_item", true, 1},
		{"some_item", true, 0},
		{"some_item", false, 0},
	}
	t.Logf("Filter state full: %v", cf)
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("cf.Delete(%q)", tc.word), func(t *testing.T) {
			if got, gotCount := cf.Delete([]byte(tc.word)), cf.Count(); got != tc.want || gotCount != tc.wantCount {
				t.Errorf("cf.Delete(%q) = %v, count = %d; want %v, count = %d", tc.word, got, gotCount, tc.want, tc.wantCount)
			}
		})
	}
}

func TestEncodeDecode(t *testing.T) {
	cf := NewFilter(10)
	for i := 0; i < 16; i++ {
		cf.Insert([]byte{byte(i)})
	}
	encoded := cf.Encode()
	got, err := Decode(encoded)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !cmp.Equal(cf, got, cmp.AllowUnexported(Filter{})) {
		t.Errorf("Decode = %v, want %v, encoded = %v", got, cf, encoded)
	}
}
