package gpu

import "math"

// Hasher is a 64-bit hash that the cache keys of the renderers are made with. It takes a whole
// 64-bit word at a time (a multiply and a shift each, mixing every bit of the word into every
// bit of the state, so that two numbers that differ in one bit, as the signs of two coordinates
// do, never cancel), where the FNV-1a it replaced took one byte at a time.
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
	x := (uint64(*h) ^ v) * 0x9E3779B97F4A7C15
	x ^= x >> 32
	x *= 0xD6E8FEB86659FD93
	x ^= x >> 29
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
