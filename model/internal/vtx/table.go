// Package vtx is the table that welds equal vertices for the loaders of package model: a hash table from a key of three integers
// (the bits of a position, or its cell on a grid) to the number of the vertex that was first seen with it.
package vtx

import "math/bits"

// Key is the three integers of a position.
type Key [3]int64

// Table maps keys to consecutive numbers 0, 1, 2 ... in the order in which the keys first came.
type Table struct {
	keys  []Key
	slots []uint32 // number + 1, or 0 for an empty slot
	mask  uint64
}

// New makes a table with room for about hint keys.
func New(hint int) *Table {
	size := 16
	for size < 2*hint {
		size <<= 1
	}
	return &Table{keys: make([]Key, 0, hint), slots: make([]uint32, size), mask: uint64(size - 1)}
}

// Len is the number of different keys.
func (t *Table) Len() int { return len(t.keys) }

// Hash is the hash of a key; its high bits are as good as its low ones, so a caller can share the keys among tables by them.
func Hash(k Key) uint64 {
	h := uint64(k[0])*0x9E3779B97F4A7C15 ^ bits.RotateLeft64(uint64(k[1])*0xC2B2AE3D27D4EB4F, 23) ^ bits.RotateLeft64(uint64(k[2])*0x165667B19E3779F9, 41)
	h ^= h >> 29
	h *= 0xBF58476D1CE4E5B9
	h ^= h >> 32
	return h
}

// Find returns the number of the key, adding the key with the next number when it is new (added is true).
func (t *Table) Find(k Key) (n uint32, added bool) { return t.FindHash(k, Hash(k)) }

// FindHash is Find for a key whose hash the caller has.
func (t *Table) FindHash(k Key, h uint64) (n uint32, added bool) {
	if 2*(len(t.keys)+1) > len(t.slots) {
		t.grow()
	}
	i := h & t.mask
	for {
		s := t.slots[i]
		if s == 0 {
			n = uint32(len(t.keys))
			t.keys = append(t.keys, k)
			t.slots[i] = n + 1
			return n, true
		}
		if t.keys[s-1] == k {
			return s - 1, false
		}
		i = (i + 1) & t.mask
	}
}

func (t *Table) grow() {
	slots := make([]uint32, 2*len(t.slots))
	mask := uint64(len(slots) - 1)
	for n, k := range t.keys {
		i := Hash(k) & mask
		for slots[i] != 0 {
			i = (i + 1) & mask
		}
		slots[i] = uint32(n) + 1
	}
	t.slots, t.mask = slots, mask
}
