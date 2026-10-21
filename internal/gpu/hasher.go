package gpu

import "math"

// Hasher is a 64-bit FNV-1a hash that the cache keys of the renderers are made with.
//
// Flutter: no counterpart
type Hasher uint64

// NewHasher returns a hash with nothing added.
//
// Flutter: no counterpart
func NewHasher() Hasher { return 14695981039346656037 }

// U64 adds a number.
//
// Flutter: no counterpart
func (h *Hasher) U64(v uint64) {
	x := uint64(*h)
	for i := 0; i < 8; i++ {
		x ^= v & 0xff
		x *= 1099511628211
		v >>= 8
	}
	*h = Hasher(x)
}

// F64 adds a float.
//
// Flutter: no counterpart
func (h *Hasher) F64(v float64) { h.U64(math.Float64bits(v)) }

// I adds an int.
//
// Flutter: no counterpart
func (h *Hasher) I(v int) { h.U64(uint64(int64(v))) }

// B adds a bool.
//
// Flutter: no counterpart
func (h *Hasher) B(v bool) {
	if v {
		h.U64(1)
	} else {
		h.U64(0)
	}
}

// Sum returns the hash.
//
// Flutter: no counterpart
func (h *Hasher) Sum() uint64 { return uint64(*h) }
