package gltf

import (
	"encoding/binary"
	"math"

	"github.com/rm4n0s/errors"
)

// The component types of an accessor (OpenGL's constants).
const (
	ctByte   = 5120
	ctUbyte  = 5121
	ctShort  = 5122
	ctUshort = 5123
	ctUint   = 5125
	ctFloat  = 5126
)

// maxZeroComponents is the most components that an accessor with no buffer view (all zeros, or zeros with a sparse substitution)
// may have: the bytes of the others are in the file, which bounds them.
const maxZeroComponents = 1 << 24

// accessor is a checked accessor: where its elements are and how they are laid out.
type accessor struct {
	where      string
	count      int
	ncomp      int
	rows       int
	typ        string
	ct         int
	normalized bool
	compSize   int
	colStride  int
	elemSize   int

	// base is the data of the buffer view, nil for an accessor that has none (all zeros); first is the byte of the first element in
	// it and stride the distance of two elements.
	base   []byte
	first  int
	stride int

	sparse *sparse
}

// sparse is a checked sparse substitution: count elements, at the indices of idx, take the values of val.
type sparse struct {
	count  int
	idxCT  int
	idx    []byte
	values []byte
}

func componentSize(ct int) int {
	switch ct {
	case ctByte, ctUbyte:
		return 1
	case ctShort, ctUshort:
		return 2
	case ctUint, ctFloat:
		return 4
	}
	return 0
}

// typeShape is the number of components of an accessor type, and for a matrix its number of rows.
func typeShape(t string) (ncomp, rows int) {
	switch t {
	case "SCALAR":
		return 1, 1
	case "VEC2":
		return 2, 2
	case "VEC3":
		return 3, 3
	case "VEC4":
		return 4, 4
	case "MAT2":
		return 4, 2
	case "MAT3":
		return 9, 3
	case "MAT4":
		return 16, 4
	}
	return 0, 0
}

// accessor checks the accessor i and gives its layout; every element, and every sparse index and value, is inside its buffer view.
func (l *loader) accessor(i int, where string) (*accessor, error) {
	if i < 0 || i >= len(l.doc.Accessors) {
		return nil, errors.New(TagGLTFInvalid, "an accessor does not exist", "path", where, "reason", "index out of range", "index", i)
	}
	if a, ok := l.accessors[i]; ok {
		return a, nil
	}
	aj := &l.doc.Accessors[i]
	here := ptr("accessors", i)
	a := &accessor{where: here, count: aj.Count, typ: aj.Type, ct: aj.ComponentType, normalized: aj.Normalized}

	a.compSize = componentSize(aj.ComponentType)
	if a.compSize == 0 {
		return nil, errors.New(TagGLTFInvalid, "an accessor has a componentType that is not known", "path", ptr("accessors", i, "componentType"), "reason", "bad componentType", "componentType", aj.ComponentType)
	}
	a.ncomp, a.rows = typeShape(aj.Type)
	if a.ncomp == 0 {
		return nil, errors.New(TagGLTFInvalid, "an accessor has a type that is not known", "path", ptr("accessors", i, "type"), "reason", "bad type", "type", aj.Type)
	}
	if aj.Count < 1 {
		return nil, errors.New(TagGLTFInvalid, "an accessor has no elements", "path", ptr("accessors", i, "count"), "reason", "bad count", "count", aj.Count)
	}
	if aj.Normalized && (aj.ComponentType == ctUint || aj.ComponentType == ctFloat) {
		return nil, errors.New(TagGLTFInvalid, "an accessor is normalized and is not a byte or a short", "path", ptr("accessors", i, "normalized"), "reason", "bad normalized")
	}
	if aj.ByteOffset < 0 {
		return nil, errors.New(TagGLTFInvalid, "an accessor has a negative byteOffset", "path", ptr("accessors", i, "byteOffset"), "reason", "bad byteOffset")
	}

	// the columns of a matrix start at a multiple of four bytes
	if a.ncomp == a.rows {
		a.colStride = a.ncomp * a.compSize
		a.elemSize = a.colStride
	} else {
		colBytes := a.rows * a.compSize
		a.colStride = (colBytes + 3) &^ 3
		a.elemSize = a.colStride * (a.ncomp / a.rows)
	}

	if aj.BufferView != nil {
		v, err := l.view(*aj.BufferView, ptr("accessors", i, "bufferView"))
		if err != nil {
			return nil, err
		}
		a.base, a.first, a.stride = v.data, aj.ByteOffset, v.stride
		if a.stride == 0 {
			a.stride = a.elemSize
		}
		if a.stride < a.elemSize {
			return nil, errors.New(TagGLTFInvalid, "an accessor has an element longer than the byteStride of its view", "path", here, "reason", "bad byteStride", "byteStride", a.stride, "elementSize", a.elemSize)
		}
		// the last element may end before the stride does
		end := a.first + (aj.Count-1)*a.stride + a.elemSize
		if aj.Count-1 > (math.MaxInt-a.first-a.elemSize)/a.stride || end > len(a.base) {
			return nil, errors.New(TagGLTFInvalid, "an accessor reaches past its buffer view", "path", here, "reason", "accessor out of bounds", "end", end, "viewLength", len(a.base))
		}
	}

	if aj.BufferView == nil && aj.Count > maxZeroComponents/a.ncomp {
		return nil, errors.New(TagGLTFInvalid, "an accessor without a buffer view has more elements than the loader makes", "path", ptr("accessors", i, "count"), "reason", "bad count", "count", aj.Count)
	}

	if sj := aj.Sparse; sj != nil {
		sp, err := l.sparse(sj, a, i)
		if err != nil {
			return nil, err
		}
		a.sparse = sp
	}

	l.accessors[i] = a
	return a, nil
}

// sparse checks the sparse part of the accessor i.
func (l *loader) sparse(sj *sparseJSON, a *accessor, i int) (*sparse, error) {
	here := ptr("accessors", i, "sparse")
	if sj.Count < 1 || sj.Count > a.count {
		return nil, errors.New(TagGLTFInvalid, "a sparse accessor has a count that is not from 1 to the count of the accessor", "path", here+"/count", "reason", "bad count", "count", sj.Count)
	}
	ict := sj.Indices.ComponentType
	if ict != ctUbyte && ict != ctUshort && ict != ctUint {
		return nil, errors.New(TagGLTFInvalid, "the indices of a sparse accessor are not unsigned integers", "path", here+"/indices/componentType", "reason", "bad componentType", "componentType", ict)
	}
	if sj.Indices.BufferView == nil || sj.Values.BufferView == nil {
		return nil, errors.New(TagGLTFInvalid, "a sparse accessor has no buffer view for its indices or its values", "path", here, "reason", "missing property")
	}
	if sj.Indices.ByteOffset < 0 || sj.Values.ByteOffset < 0 {
		return nil, errors.New(TagGLTFInvalid, "a sparse accessor has a negative byteOffset", "path", here, "reason", "bad byteOffset")
	}

	iv, err := l.view(*sj.Indices.BufferView, here+"/indices/bufferView")
	if err != nil {
		return nil, err
	}
	isz := componentSize(ict)
	if sj.Indices.ByteOffset > len(iv.data) || sj.Count > (len(iv.data)-sj.Indices.ByteOffset)/isz {
		return nil, errors.New(TagGLTFInvalid, "the indices of a sparse accessor reach past their buffer view", "path", here+"/indices", "reason", "accessor out of bounds", "viewLength", len(iv.data))
	}
	vv, err := l.view(*sj.Values.BufferView, here+"/values/bufferView")
	if err != nil {
		return nil, err
	}
	if sj.Values.ByteOffset > len(vv.data) || sj.Count > (len(vv.data)-sj.Values.ByteOffset)/a.elemSize {
		return nil, errors.New(TagGLTFInvalid, "the values of a sparse accessor reach past their buffer view", "path", here+"/values", "reason", "accessor out of bounds", "viewLength", len(vv.data))
	}
	sp := &sparse{
		count:  sj.Count,
		idxCT:  ict,
		idx:    iv.data[sj.Indices.ByteOffset : sj.Indices.ByteOffset+sj.Count*isz],
		values: vv.data[sj.Values.ByteOffset : sj.Values.ByteOffset+sj.Count*a.elemSize],
	}

	// the indices increase and are inside the accessor
	prev := -1
	for k := 0; k < sj.Count; k++ {
		idx := sp.index(k)
		if idx <= prev || idx >= a.count {
			return nil, errors.New(TagGLTFInvalid, "the indices of a sparse accessor do not increase inside the accessor", "path", here+"/indices", "reason", "bad sparse index", "item", k, "index", idx)
		}
		prev = idx
	}
	return sp, nil
}

// index is the k-th index of the substitution.
func (s *sparse) index(k int) int {
	switch s.idxCT {
	case ctUbyte:
		return int(s.idx[k])
	case ctUshort:
		return int(binary.LittleEndian.Uint16(s.idx[2*k:]))
	}
	return int(binary.LittleEndian.Uint32(s.idx[4*k:]))
}

// offset is the byte, in the data of a view, of component c of element e.
func (a *accessor) componentOffset(c int) int {
	if a.ncomp == a.rows {
		return c * a.compSize
	}
	return (c/a.rows)*a.colStride + (c%a.rows)*a.compSize
}

// float is the value of a component at the start of b, as a float: a normalized integer is mapped to 0 to 1 (or -1 to 1), and any
// other integer is converted as it is.
func (a *accessor) float(b []byte) float32 {
	switch a.ct {
	case ctFloat:
		return math.Float32frombits(binary.LittleEndian.Uint32(b))
	case ctByte:
		v := float32(int8(b[0]))
		if a.normalized {
			return max(v/127, -1)
		}
		return v
	case ctUbyte:
		v := float32(b[0])
		if a.normalized {
			return v / 255
		}
		return v
	case ctShort:
		v := float32(int16(binary.LittleEndian.Uint16(b)))
		if a.normalized {
			return max(v/32767, -1)
		}
		return v
	case ctUshort:
		v := float32(binary.LittleEndian.Uint16(b))
		if a.normalized {
			return v / 65535
		}
		return v
	}
	return float32(binary.LittleEndian.Uint32(b))
}

// floats reads every component of the accessor as floats, count*ncomp of them. A float that is not finite is an error.
func (l *loader) floats(a *accessor) ([]float32, error) {
	out := make([]float32, a.count*a.ncomp)
	if a.base != nil {
		for e := 0; e < a.count; e++ {
			el := a.base[a.first+e*a.stride:]
			for c := 0; c < a.ncomp; c++ {
				out[e*a.ncomp+c] = a.float(el[a.componentOffset(c):])
			}
		}
	}
	if sp := a.sparse; sp != nil {
		for k := 0; k < sp.count; k++ {
			el := sp.values[k*a.elemSize:]
			dst := out[sp.index(k)*a.ncomp:]
			for c := 0; c < a.ncomp; c++ {
				dst[c] = a.float(el[a.componentOffset(c):])
			}
		}
	}
	if a.ct == ctFloat {
		for i, v := range out {
			if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
				return nil, errors.New(TagGLTFInvalid, "an accessor has a float that is not a finite number", "path", a.where, "reason", "not finite", "item", i/a.ncomp)
			}
		}
	}
	return out, nil
}

// uint is the value of an unsigned component at the start of b.
func (a *accessor) uint(b []byte) uint32 {
	switch a.ct {
	case ctUbyte:
		return uint32(b[0])
	case ctUshort:
		return uint32(binary.LittleEndian.Uint16(b))
	}
	return binary.LittleEndian.Uint32(b)
}

// uints reads every component of the accessor, which must be of an unsigned integer type, count*ncomp of them.
func (l *loader) uints(a *accessor) ([]uint32, error) {
	if a.ct != ctUbyte && a.ct != ctUshort && a.ct != ctUint {
		return nil, errors.New(TagGLTFInvalid, "an accessor is not of an unsigned integer type", "path", a.where, "reason", "bad componentType", "componentType", a.ct)
	}
	out := make([]uint32, a.count*a.ncomp)
	if a.base != nil {
		for e := 0; e < a.count; e++ {
			el := a.base[a.first+e*a.stride:]
			for c := 0; c < a.ncomp; c++ {
				out[e*a.ncomp+c] = a.uint(el[a.componentOffset(c):])
			}
		}
	}
	if sp := a.sparse; sp != nil {
		for k := 0; k < sp.count; k++ {
			el := sp.values[k*a.elemSize:]
			dst := out[sp.index(k)*a.ncomp:]
			for c := 0; c < a.ncomp; c++ {
				dst[c] = a.uint(el[a.componentOffset(c):])
			}
		}
	}
	return out, nil
}
