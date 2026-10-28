//go:build amd64 || arm64

package goid

import (
	"sync"
	"testing"
)

// window makes a fake window of words, all zero except those in at (word index to
// value).
func window(at map[int]uint64) [windowWords]uint64 {
	var w [windowWords]uint64
	for i, v := range at {
		w[i] = v
	}
	return w
}

func TestFindOffset(t *testing.T) {
	ids := []uint64{7, 19, 20, 33, 41}
	// build makes samples whose window holds the number of the goroutine at the
	// words in at, and other[k] (not the number) at the words in other.
	build := func(at, same []int, ids []uint64) []sample {
		var out []sample
		for _, id := range ids {
			m := map[int]uint64{}
			for _, w := range at {
				m[w] = id
			}
			for _, w := range same {
				m[w] = 99 // the same on every goroutine
			}
			out = append(out, sample{id: id, words: window(m)})
		}
		return out
	}
	tests := []struct {
		name    string
		samples []sample
		want    int
		wantOK  bool
	}{
		{"exactly one offset", build([]int{19}, []int{3, 4}, ids), 19 * 8, true},
		{"the first word", build([]int{0}, nil, ids), 0, true},
		{"the last word of the window", build([]int{windowWords - 1}, nil, ids), 248, true},
		{"none matches", build(nil, []int{3, 4}, ids), 0, false},
		{"two match", build([]int{19, 20}, nil, ids), 0, false},
		{"three match", build([]int{5, 19, 20}, nil, ids), 0, false},
		{"one matches on some goroutines only", func() []sample {
			s := build([]int{19}, nil, ids)
			s[2].words[19] = 1
			return s
		}(), 0, false},
		{"too few samples", build([]int{19}, nil, ids[:3]), 0, false},
		{"no samples", nil, 0, false},
		{"two goroutines with the same number", build([]int{19}, nil, []uint64{7, 19, 19, 33}), 0, false},
		{"a number 0", build([]int{19}, nil, []uint64{0, 19, 20, 33}), 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := findOffset(tt.samples)
			if ok != tt.wantOK || (ok && got != tt.want) {
				t.Errorf("findOffset = %d, %v; want %d, %v", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestGwordStaysInTheWindow checks the mask in the assembly: whatever the offset,
// the word read is one of the first 256 bytes of the g.
func TestGwordStaysInTheWindow(t *testing.T) {
	tests := []struct {
		name string
		off  uintptr
		want uintptr // the offset that the read actually uses
	}{
		{"zero", 0, 0},
		{"inside", 80, 80},
		{"the last word", 248, 248},
		{"just past the window", 256, 0},
		{"far past the window", 4096 + 16, 16},
		{"unaligned", 83, 80},
		{"all ones", ^uintptr(0), 248},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, want := gword(tt.off), gword(tt.want); got != want {
				t.Errorf("gword(%d) = %#x, gword(%d) = %#x", tt.off, got, tt.want, want)
			}
		})
	}
}

// TestCalibrationFindsTheOffset checks that this Go version, on this machine, lets
// Token read the number from the g, rather than fall back to runtime.Stack (which
// would still be correct, but slow).
func TestCalibrationFindsTheOffset(t *testing.T) {
	if !fastPath() {
		t.Fatal("the calibration found no single offset: Token falls back to runtime.Stack")
	}
	off := uintptr(offsetState.Load() - 1)
	if off >= windowBytes || off%8 != 0 {
		t.Errorf("offset %d is outside the window or unaligned", off)
	}
	if got, want := gword(off), ID(); got != want {
		t.Errorf("the word at the offset = %d, ID = %d", got, want)
	}
	t.Logf("the goroutine number is at byte %d of the g", off)
}

// TestCalibrationFromManyGoroutinesAtOnce runs the first call of Token from many
// goroutines, as the first calls of a program can be.
func TestCalibrationFromManyGoroutinesAtOnce(t *testing.T) {
	tests := []struct {
		name       string
		goroutines int
	}{
		{"two", 2},
		{"fifty", 50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start over, as at the first call of a program. No other test runs
			// now, and no goroutine of this program is calling Token.
			offsetState.Store(0)
			calibrated = sync.Once{}
			tokens := make([]uint64, tt.goroutines)
			var ready, wg sync.WaitGroup
			start := make(chan struct{})
			ready.Add(tt.goroutines)
			wg.Add(tt.goroutines)
			for i := range tokens {
				go func() {
					defer wg.Done()
					ready.Done()
					<-start
					tokens[i] = Token()
					if tokens[i] != ID() {
						t.Errorf("goroutine %d: Token = %d, ID = %d", i, tokens[i], ID())
					}
				}()
			}
			ready.Wait()
			close(start)
			wg.Wait()
			if !fastPath() {
				t.Error("the calibration did not find the offset")
			}
		})
	}
}

// TestTokenFallsBackToID checks Token when the calibration found no single
// offset: it is the number from runtime.Stack.
func TestTokenFallsBackToID(t *testing.T) {
	if !fastPath() {
		t.Skip("already on the fallback")
	}
	saved := offsetState.Load()
	offsetState.Store(fallback)
	t.Cleanup(func() { offsetState.Store(saved) })
	tests := []struct {
		name  string
		depth int
	}{
		{"shallow", 0},
		{"deep", 200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			at(tt.depth, func() {
				if got, want := Token(), ID(); got != want || got == 0 {
					t.Errorf("Token = %d, ID = %d", got, want)
				}
			})
		})
	}
}
