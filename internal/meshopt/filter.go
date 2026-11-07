package meshopt

import (
	"encoding/binary"
	"math"

	"github.com/rm4n0s/errors"
)

// This file is translated to Go and changed from src/vertexfilter.cpp of meshoptimizer 1.3 (MIT license,
// third_party/meshoptimizer/LICENSE): the scalar decoders of the four filters. The float32 operations are written so that no
// multiply is fused with an add (a conversion to float32 rounds), which gives the same bytes on every architecture.

// Filter is a vertex filter of EXT_meshopt_compression: what is done to the decoded vertices to get their attributes.
//
// Flutter: no counterpart
type Filter int

// The filters of the extension. FilterNone leaves the bytes as the vertex decoder made them.
//
// Flutter: no counterpart
const (
	FilterNone Filter = iota
	// FilterOctahedral is for normals and tangents of 4 bytes (8 bit components) or 8 bytes (16 bit components).
	FilterOctahedral
	// FilterQuaternion is for rotations of 8 bytes (16 bit components).
	FilterQuaternion
	// FilterExponential is for floats of a multiple of 4 bytes, each a 24 bit mantissa and an 8 bit exponent.
	FilterExponential
	// FilterColor is for colors of 4 bytes (8 bit components) or 8 bytes (16 bit components).
	FilterColor
)

// DecodeFilter applies the filter to count elements of stride bytes each in data, in place. It fails with TagMeshoptFilterInvalid
// when the stride is not one that the filter has.
//
// Flutter: no counterpart
func DecodeFilter(filter Filter, data []byte, count, stride int) error {
	reason := decodeFilter(filter, data, count, stride)
	if reason != "" {
		return errors.New(TagMeshoptFilterInvalid, "the filter cannot be applied", "reason", reason, "filter", int(filter), "count", count, "stride", stride)
	}
	return nil
}

// decodeFilter returns the reason of a failure, or "" when it worked.
func decodeFilter(filter Filter, data []byte, count, stride int) string {
	if count < 0 || stride <= 0 || len(data) < count*stride {
		return "the data is shorter than count*stride"
	}
	data = data[:count*stride]
	switch filter {
	case FilterNone:
	case FilterOctahedral:
		switch stride {
		case 4:
			decodeFilterOct8(data, count)
		case 8:
			decodeFilterOct16(data, count)
		default:
			return "the stride of the octahedral filter is 4 or 8"
		}
	case FilterQuaternion:
		if stride != 8 {
			return "the stride of the quaternion filter is 8"
		}
		decodeFilterQuat(data, count)
	case FilterExponential:
		if stride%4 != 0 {
			return "the stride of the exponential filter is a multiple of 4"
		}
		decodeFilterExp(data, count*(stride/4))
	case FilterColor:
		switch stride {
		case 4:
			decodeFilterColor8(data, count)
		case 8:
			decodeFilterColor16(data, count)
		default:
			return "the stride of the color filter is 4 or 8"
		}
	default:
		return "the filter is not known"
	}
	return ""
}

func abs32(v float32) float32 { return float32(math.Abs(float64(v))) }

func sqrt32(v float32) float32 { return float32(math.Sqrt(float64(v))) }

// roundSigned is int(v + 0.5) for v >= 0 and int(v - 0.5) otherwise, with the conversion that C does (towards zero).
func roundSigned(x, s float32) int {
	h := float32(0.5)
	if x < 0 {
		h = -0.5
	}
	return int(float32(x*s) + h)
}

// octahedral decodes one (x, y, z) of a normal in place; limit is the largest value of a component.
func octahedral(x, y, z float32, limit float32) (int, int, int) {
	z -= abs32(x)
	z -= abs32(y)

	// fix the octahedral coordinates for z < 0
	t := float32(0)
	if z < 0 {
		t = z
	}
	if x >= 0 {
		x += t
	} else {
		x -= t
	}
	if y >= 0 {
		y += t
	} else {
		y -= t
	}

	// the length of the normal and the scale
	l := sqrt32(float32(float32(x*x)+float32(y*y)) + float32(z*z))
	s := limit / l
	return roundSigned(x, s), roundSigned(y, s), roundSigned(z, s)
}

func decodeFilterOct8(data []byte, count int) {
	for i := 0; i < count; i++ {
		e := data[i*4 : i*4+4]
		x, y, z := float32(int8(e[0])), float32(int8(e[1])), float32(int8(e[2]))
		xf, yf, zf := octahedral(x, y, z, 127)
		e[0], e[1], e[2] = byte(int8(xf)), byte(int8(yf)), byte(int8(zf))
	}
}

func decodeFilterOct16(data []byte, count int) {
	for i := 0; i < count; i++ {
		e := data[i*8 : i*8+8]
		x := float32(int16(binary.LittleEndian.Uint16(e[0:])))
		y := float32(int16(binary.LittleEndian.Uint16(e[2:])))
		z := float32(int16(binary.LittleEndian.Uint16(e[4:])))
		xf, yf, zf := octahedral(x, y, z, 32767)
		binary.LittleEndian.PutUint16(e[0:], uint16(int16(xf)))
		binary.LittleEndian.PutUint16(e[2:], uint16(int16(yf)))
		binary.LittleEndian.PutUint16(e[4:], uint16(int16(zf)))
	}
}

func decodeFilterQuat(data []byte, count int) {
	scale := float32(32767) / sqrt32(2)
	for i := 0; i < count; i++ {
		e := data[i*8 : i*8+8]
		w3 := int16(binary.LittleEndian.Uint16(e[6:]))

		// the scale is in the high bits of the component
		s := float32(int(w3) | 3)

		// x, y and z as floats, unscaled (the scale is 1/sqrt(2) * 1/s)
		x := float32(int16(binary.LittleEndian.Uint16(e[0:])))
		y := float32(int16(binary.LittleEndian.Uint16(e[2:])))
		z := float32(int16(binary.LittleEndian.Uint16(e[4:])))

		// w is a square root, unscaled, and it is clamped at 0 against the errors of the precision
		ws := float32(s * s)
		ww := float32(float32(float32(float32(ws*2)-float32(x*x))-float32(y*y)) - float32(z*z))
		if ww < 0 {
			ww = 0
		}
		w := sqrt32(ww)

		// the final scale
		ss := scale / s

		xf := roundSigned(x, ss)
		yf := roundSigned(y, ss)
		zf := roundSigned(z, ss)
		wf := int(float32(w*ss) + 0.5)

		qc := int(w3) & 3

		// the order of the output is told by the input
		put := func(slot, v int) {
			binary.LittleEndian.PutUint16(e[((qc+slot)&3)*2:], uint16(int16(v)))
		}
		put(1, xf)
		put(2, yf)
		put(3, zf)
		put(0, wf)
	}
}

func decodeFilterExp(data []byte, count int) {
	for i := 0; i < count; i++ {
		v := binary.LittleEndian.Uint32(data[i*4:])

		// the mantissa and the exponent
		m := int32(v<<8) >> 8
		e := int32(v) >> 24

		// ldexp(float(m), e)
		f := math.Float32frombits(uint32(e+127) << 23)
		f = float32(f * float32(m))
		binary.LittleEndian.PutUint32(data[i*4:], math.Float32bits(f))
	}
}

// color decodes one (r, g, b, a) of fixed point values; co and cg are the sign extended chroma, and limit is the largest value.
func color(y, co, cg, a int, as int, limit float32) (int, int, int, int) {
	as |= as >> 1
	as |= as >> 2
	as |= as >> 4
	as |= as >> 8 // nothing for 8 bits

	r := y + co - cg
	g := y + cg
	b := y - co - cg

	// a has one more bit, to match the others
	a = ((a << 1) & as) | (a & 1)

	ss := limit / float32(as)
	rf := int(float32(float32(r)*ss) + 0.5)
	gf := int(float32(float32(g)*ss) + 0.5)
	bf := int(float32(float32(b)*ss) + 0.5)
	af := int(float32(float32(a)*ss) + 0.5)
	return rf, gf, bf, af
}

func decodeFilterColor8(data []byte, count int) {
	for i := 0; i < count; i++ {
		e := data[i*4 : i*4+4]
		a := int(e[3])
		r, g, b, a := color(int(e[0]), int(int8(e[1])), int(int8(e[2])), a, a, 255)
		e[0], e[1], e[2], e[3] = byte(r), byte(g), byte(b), byte(a)
	}
}

func decodeFilterColor16(data []byte, count int) {
	for i := 0; i < count; i++ {
		e := data[i*8 : i*8+8]
		a := int(binary.LittleEndian.Uint16(e[6:]))
		r, g, b, a := color(int(binary.LittleEndian.Uint16(e[0:])), int(int16(binary.LittleEndian.Uint16(e[2:]))),
			int(int16(binary.LittleEndian.Uint16(e[4:]))), a, a, 65535)
		binary.LittleEndian.PutUint16(e[0:], uint16(r))
		binary.LittleEndian.PutUint16(e[2:], uint16(g))
		binary.LittleEndian.PutUint16(e[4:], uint16(b))
		binary.LittleEndian.PutUint16(e[6:], uint16(a))
	}
}

// TagMeshoptFilterInvalid is the tag of the error of a filter that is asked for a stride it does not have, or for data shorter than
// its count says. The metadata has "reason", "filter", "count" and "stride".
//
// Flutter: no counterpart
const TagMeshoptFilterInvalid = "MeshoptFilterInvalid"
