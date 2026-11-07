// Package meshopt decodes the data of EXT_meshopt_compression (SPEC §28.5): the vertex buffers, the index buffers and the index
// sequences of meshoptimizer's formats (versions 0 and 1), and its four vertex filters. It is plain Go with no OS tags.
//
// The decoders are translated to Go and changed from src/vertexcodec.cpp, src/indexcodec.cpp and src/vertexfilter.cpp of
// meshoptimizer 1.3 (MIT license, third_party/meshoptimizer/LICENSE). Only the scalar paths are ported (the SIMD ones give the same
// bytes), every read is checked against the end of the data, and a failure is an error with a reason instead of a negative code.
package meshopt

import (
	"github.com/rm4n0s/errors"
)

const (
	indexHeader    = 0xe0
	sequenceHeader = 0xd0
	// decodeIndexVersion is the newest version of the index formats that this package reads.
	decodeIndexVersion = 1
)

// codeAuxLast is the number of bytes of the table of codes at the end of an index buffer.
const codeAuxLast = 16

// indexReader reads the bytes of an index buffer's data with a position that is never past the end of the buffer.
type indexReader struct {
	data []byte
	pos  int
}

// vbyte reads a 32 bit value in up to five groups of seven bits. The loop ends for malformed data too.
func (r *indexReader) vbyte() (uint32, bool) {
	if r.pos >= len(r.data) {
		return 0, false
	}
	lead := r.data[r.pos]
	r.pos++
	if lead < 128 {
		return uint32(lead), true
	}
	result := uint32(lead & 127)
	shift := uint(7)
	for i := 0; i < 4; i++ {
		if r.pos >= len(r.data) {
			return 0, false
		}
		group := r.data[r.pos]
		r.pos++
		result |= uint32(group&127) << shift
		shift += 7
		if group < 128 {
			break
		}
	}
	return result, true
}

// index reads a zigzag delta from last.
func (r *indexReader) index(last uint32) (uint32, bool) {
	v, ok := r.vbyte()
	if !ok {
		return 0, false
	}
	d := (v >> 1) ^ -(v & 1)
	return last + d, true
}

// DecodeIndexBuffer decodes the triangle index buffer in data into indices, whose length is the number of indices (a multiple of
// three). It fails with TagMeshoptIndexInvalid when the data is not a version 0 or 1 index buffer of exactly that many indices.
//
// Flutter: no counterpart
func DecodeIndexBuffer(indices []uint32, data []byte) error {
	reason := decodeIndexBuffer(indices, data)
	if reason != "" {
		return errors.New(TagMeshoptIndexInvalid, "the index buffer is malformed", "reason", reason, "bytes", len(data))
	}
	return nil
}

// decodeIndexBuffer is meshopt_decodeIndexBuffer; it returns the reason of a failure, or "" when it worked.
func decodeIndexBuffer(indices []uint32, buf []byte) string {
	count := len(indices)
	if count%3 != 0 {
		return "the number of indices is not a multiple of three"
	}

	// the minimum valid encoding is the header, one byte for each triangle and the 16 bytes of the table of codes
	if len(buf) < 1+count/3+codeAuxLast {
		return "too short"
	}
	if buf[0]&0xf0 != indexHeader {
		return "bad header"
	}
	version := int(buf[0] & 0x0f)
	if version > decodeIndexVersion {
		return "unknown version"
	}

	var edgeFifo [16][2]uint32
	for i := range edgeFifo {
		edgeFifo[i] = [2]uint32{0xffffffff, 0xffffffff}
	}
	var vertexFifo [16]uint32
	for i := range vertexFifo {
		vertexFifo[i] = 0xffffffff
	}
	edgeOffset, vertexOffset := 0, 0

	pushEdge := func(a, b uint32) {
		edgeFifo[edgeOffset] = [2]uint32{a, b}
		edgeOffset = (edgeOffset + 1) & 15
	}
	pushVertex := func(v uint32, advance bool) {
		vertexFifo[vertexOffset] = v
		if advance {
			vertexOffset = (vertexOffset + 1) & 15
		}
	}

	var next, last uint32
	fecMax := 15
	if version >= 1 {
		fecMax = 13
	}

	// the table of codes is at the end, so the triangle data has to begin before dataSafeEnd
	codePos := 1
	codeEnd := codePos + count/3
	dataSafeEnd := len(buf) - codeAuxLast
	codeAuxTable := buf[dataSafeEnd:]
	r := indexReader{data: buf[:dataSafeEnd], pos: codeEnd}
	out := 0
	put := func(a, b, c uint32) {
		indices[out] = a
		indices[out+1] = b
		indices[out+2] = c
		out += 3
	}

	for codePos < codeEnd {
		codeTri := buf[codePos]
		codePos++

		if codeTri < 0xf0 {
			fe := int(codeTri >> 4)
			a := edgeFifo[(edgeOffset-1-fe)&15][0]
			b := edgeFifo[(edgeOffset-1-fe)&15][1]
			var c uint32

			fec := int(codeTri & 15)
			if fec < fecMax {
				cf := vertexFifo[(vertexOffset-1-fec)&15]
				advance := fec == 0
				if advance {
					c = next
					next++
				} else {
					c = cf
				}
				pushVertex(c, advance)
			} else {
				if fec != 15 {
					c = last + uint32(int32(fec*2-27))
				} else {
					v, ok := r.index(last)
					if !ok {
						return "truncated data"
					}
					c = v
				}
				last = c
				pushVertex(c, true)
			}

			pushEdge(c, b)
			pushEdge(a, c)
			put(a, b, c)
			continue
		}

		if codeTri < 0xfe {
			// fast path: the codes are in the table
			codeAux := codeAuxTable[codeTri&15]
			feb := int(codeAux >> 4)
			fec := int(codeAux & 15)

			// next goes up for all three vertices before the indices are decoded, as the encoder does
			a := next
			next++

			bf := vertexFifo[(vertexOffset-feb)&15]
			b := bf
			if feb == 0 {
				b = next
				next++
			}
			cf := vertexFifo[(vertexOffset-fec)&15]
			c := cf
			if fec == 0 {
				c = next
				next++
			}

			put(a, b, c)

			pushVertex(a, true)
			pushVertex(b, feb == 0)
			pushVertex(c, fec == 0)

			pushEdge(b, a)
			pushEdge(c, b)
			pushEdge(a, c)
			continue
		}

		// slow path: a whole byte for the codes instead of the table
		if r.pos >= dataSafeEnd {
			return "truncated data"
		}
		codeAux := r.data[r.pos]
		r.pos++

		fea := 15
		if codeTri == 0xfe {
			fea = 0
		}
		feb := int(codeAux >> 4)
		fec := int(codeAux & 15)

		// reset: the codes are 0 but not in the table
		if codeAux == 0 {
			next = 0
		}

		var a, b, c uint32
		if fea == 0 {
			a = next
			next++
		}
		if feb == 0 {
			b = next
			next++
		} else {
			b = vertexFifo[(vertexOffset-feb)&15]
		}
		if fec == 0 {
			c = next
			next++
		} else {
			c = vertexFifo[(vertexOffset-fec)&15]
		}

		// the free indices are deltas, so last changes
		var ok bool
		if fea == 15 {
			if a, ok = r.index(last); !ok {
				return "truncated data"
			}
			last = a
		}
		if feb == 15 {
			if b, ok = r.index(last); !ok {
				return "truncated data"
			}
			last = b
		}
		if fec == 15 {
			if c, ok = r.index(last); !ok {
				return "truncated data"
			}
			last = c
		}

		put(a, b, c)

		pushVertex(a, true)
		pushVertex(b, feb == 0 || feb == 15)
		pushVertex(c, fec == 0 || fec == 15)

		pushEdge(b, a)
		pushEdge(c, b)
		pushEdge(a, c)
	}

	// all the data bytes were read, and the read stopped at the border between the data and the table of codes
	if r.pos != dataSafeEnd {
		return "extra data"
	}
	return ""
}

// DecodeIndexSequence decodes the index sequence in data (the indices of points or lines, or of any list that is not a mesh) into
// indices. It fails with TagMeshoptIndexInvalid when the data is not a version 0 or 1 index sequence of exactly that many indices.
//
// Flutter: no counterpart
func DecodeIndexSequence(indices []uint32, data []byte) error {
	reason := decodeIndexSequence(indices, data)
	if reason != "" {
		return errors.New(TagMeshoptIndexInvalid, "the index sequence is malformed", "reason", reason, "bytes", len(data))
	}
	return nil
}

// decodeIndexSequence is meshopt_decodeIndexSequence; it returns the reason of a failure, or "" when it worked.
func decodeIndexSequence(indices []uint32, buf []byte) string {
	// the minimum valid encoding is the header, one byte for each index and a tail of four bytes
	if len(buf) < 1+len(indices)+4 {
		return "too short"
	}
	if buf[0]&0xf0 != sequenceHeader {
		return "bad header"
	}
	if int(buf[0]&0x0f) > decodeIndexVersion {
		return "unknown version"
	}

	dataSafeEnd := len(buf) - 4
	r := indexReader{data: buf[:dataSafeEnd], pos: 1}
	var last [2]uint32

	for i := range indices {
		if r.pos >= dataSafeEnd {
			return "truncated data"
		}
		v, ok := r.vbyte()
		if !ok {
			return "truncated data"
		}

		// the low bit tells the baseline
		current := v & 1
		v >>= 1

		d := (v >> 1) ^ -(v & 1)
		index := last[current] + d
		last[current] = index
		indices[i] = index
	}

	// all the data bytes were read, and the read stopped at the border between the data and the tail
	if r.pos != dataSafeEnd {
		return "extra data"
	}
	return ""
}

// TagMeshoptIndexInvalid is the tag of the error of an index buffer or an index sequence that is malformed: a header that is not
// one, a version that is not known, data that is too short or too long for the count. The metadata has "reason" and "bytes".
//
// Flutter: no counterpart
const TagMeshoptIndexInvalid = "MeshoptIndexInvalid"
