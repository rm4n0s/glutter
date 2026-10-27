package ui

import "math"

// The fingerprint of a path (SPEC §9.3, "Path keys"). A renderer keys the cached coverage of a
// path, and its clip masks, by the path's contents, so it asks for the fingerprint of every
// path in every frame: it cannot cost a pass over the points. The path therefore keeps the
// fingerprint up to date as it changes (eagerly, so that two goroutines that draw one detached
// scene only ever read it; SPEC §9.9). It is made of three sums, one each for the verbs, the
// points and the conic weights, of a strong hash of each element together with its index, so
// that appending an element takes constant time and the order of the elements still counts.
// Fingerprint then mixes the sums with the counts and the fill type. The bounds of the path, which
// the renderers also ask for in every frame, are kept the same way.

// pathDerived is what a path keeps up to date as it changes: the sums that make up its
// fingerprint, and its bounds. Its zero value is that of an empty path.
type pathDerived struct {
	verbs, points, weights uint64
	// bounds holds the points that are finite, and nonFinite says that one is not (GetBounds is
	// then the empty rectangle).
	bounds    boundsAcc
	nonFinite bool
}

// addPoint adds a point at index i to what is derived.
func (d *pathDerived) addPoint(i int, pt Offset) {
	d.points += pointHash(i, pt)
	if !pt.IsFinite() {
		d.nonFinite = true
		return
	}
	d.bounds.add(pt)
}

// mix64 is the finalizer of SplitMix64: every bit of the input changes about half of the bits of
// the output.
func mix64(z uint64) uint64 {
	z += 0x9E3779B97F4A7C15
	z = (z ^ z>>30) * 0xBF58476D1CE4E5B9
	z = (z ^ z>>27) * 0x94D049BB133111EB
	return z ^ z>>31
}

// verbHash is the hash of the verb at index i.
func verbHash(i int, v PathVerb) uint64 { return mix64(uint64(i)<<8 | uint64(v)) }

// pointHash is the hash of the point at index i.
func pointHash(i int, pt Offset) uint64 {
	return mix64(mix64(math.Float64bits(pt.Dx)^uint64(i)*0xD6E8FEB86659FD93) ^ math.Float64bits(pt.Dy))
}

// weightHash is the hash of the conic weight at index i.
func weightHash(i int, w float64) uint64 {
	return mix64(math.Float64bits(w) ^ uint64(i)*0xA0761D6478BD642F)
}

// appendVerb adds a verb to the path and to its fingerprint.
func (p *Path) appendVerb(v PathVerb) {
	p.derived.verbs += verbHash(len(p.verbs), v)
	p.verbs = append(p.verbs, v)
}

// appendPoint adds a point to the path and to its fingerprint.
func (p *Path) appendPoint(x, y float64) {
	pt := Offset{x, y}
	p.derived.addPoint(len(p.points), pt)
	p.points = append(p.points, pt)
}

// appendWeight adds a conic weight to the path and to its fingerprint.
func (p *Path) appendWeight(w float64) {
	p.derived.weights += weightHash(len(p.weights), w)
	p.weights = append(p.weights, w)
}

// rehash makes the fingerprint again from the contents, after an operation that rewrites them.
func (p *Path) rehash() {
	var h pathDerived
	for i, v := range p.verbs {
		h.verbs += verbHash(i, v)
	}
	for i, pt := range p.points {
		h.addPoint(i, pt)
	}
	for i, w := range p.weights {
		h.weights += weightHash(i, w)
	}
	p.derived = h
}

// Fingerprint returns a 64-bit hash of the path's contents: its verbs, points,
// conic weights and fill type. Two paths with the same contents have the same
// fingerprint, so a renderer can recognize a path it has drawn before even when
// the app built it again. Different paths almost never share one; a renderer that
// needs certainty compares the paths too.
//
// The path keeps the fingerprint up to date as it changes, so this takes constant time
// and writes nothing: it is safe to call from two goroutines while nothing changes the path
// (SPEC §9.3, §9.9).
//
// Flutter: no counterpart
func (p *Path) Fingerprint() uint64 {
	h := mix64(uint64(p.FillType()))
	h = mix64(h ^ uint64(len(p.verbs)))
	h = mix64(h ^ p.derived.verbs)
	h = mix64(h ^ uint64(len(p.points)))
	h = mix64(h ^ p.derived.points)
	h = mix64(h ^ uint64(len(p.weights)))
	return mix64(h ^ p.derived.weights)
}
