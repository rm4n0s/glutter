package meshopt

import (
	"math/bits"

	"github.com/rm4n0s/errors"
)

// This file is translated to Go and changed from src/vertexcodec.cpp of meshoptimizer 1.3 (MIT license,
// third_party/meshoptimizer/LICENSE): the scalar decoder of vertex buffers of versions 0 and 1.

const (
	vertexHeader = 0xa0
	// decodeVertexVersion is the newest version of the vertex format that this package reads.
	decodeVertexVersion = 1

	vertexBlockSizeBytes = 8192
	vertexBlockMaxSize   = 256
	byteGroupSize        = 16
	// byteGroupDecodeLimit is the bytes that a group may read, so that a group is decoded only when this many bytes are left.
	byteGroupDecodeLimit = 24
	tailMinSizeV0        = 32
	tailMinSizeV1        = 24
)

var (
	bitsV0 = [4]int{0, 2, 4, 8}
	bitsV1 = [5]int{0, 1, 2, 4, 8}
)

// vertexBlockSize is the number of vertices of a block: the whole block fits the scratch buffer and is a multiple of a byte group.
// It is part of the format.
func vertexBlockSize(vertexSize int) int {
	result := (vertexBlockSizeBytes / vertexSize) &^ (byteGroupSize - 1)
	return min(result, vertexBlockMaxSize)
}

// DecodeVertexBuffer decodes count vertices of size bytes each (a multiple of four, at most 256) from data into dst, which has
// count*size bytes. It fails with TagMeshoptVertexInvalid when the data is not a version 0 or 1 vertex buffer of exactly that many
// vertices.
//
// Flutter: no counterpart
func DecodeVertexBuffer(dst []byte, count, size int, data []byte) error {
	reason := decodeVertexBuffer(dst, count, size, data)
	if reason != "" {
		return errors.New(TagMeshoptVertexInvalid, "the vertex buffer is malformed", "reason", reason, "count", count, "size", size, "bytes", len(data))
	}
	return nil
}

// decodeVertexBuffer is meshopt_decodeVertexBuffer; it returns the reason of a failure, or "" when it worked.
func decodeVertexBuffer(dst []byte, count, size int, buf []byte) string {
	if size <= 0 || size > 256 || size%4 != 0 {
		return "the vertex size is not a multiple of four up to 256"
	}
	if count < 0 || len(dst) != count*size {
		return "the destination does not have count*size bytes"
	}
	if len(buf) < 1 {
		return "too short"
	}
	header := buf[0]
	if header&0xf0 != vertexHeader {
		return "bad header"
	}
	version := int(header & 0x0f)
	if version > decodeVertexVersion {
		return "unknown version"
	}

	tailSize := size
	tailMin := tailMinSizeV0
	if version != 0 {
		tailSize += size / 4
		tailMin = tailMinSizeV1
	}
	tailPad := max(tailSize, tailMin)
	if len(buf)-1 < tailPad {
		return "too short"
	}

	tail := buf[len(buf)-tailSize:]
	var lastVertex [256]byte
	copy(lastVertex[:size], tail[:size])
	var channels []byte
	if version != 0 {
		channels = tail[size:]
	}

	blockSize := vertexBlockSize(size)
	data := buf[1:]
	for offset := 0; offset < count; {
		n := min(blockSize, count-offset)
		var ok bool
		data, ok = decodeVertexBlock(data, dst[offset*size:(offset+n)*size], n, size, &lastVertex, channels, version)
		if !ok {
			return "truncated data"
		}
		offset += n
	}

	if len(data) != tailPad {
		return "extra data"
	}
	return ""
}

// decodeBytesGroup decodes 16 bytes of the given bit length from data and returns the data after them; the caller has checked that
// byteGroupDecodeLimit bytes are there.
func decodeBytesGroup(data []byte, buffer []byte, bitLen int) []byte {
	switch bitLen {
	case 0:
		clear(buffer[:byteGroupSize])
		return data
	case 8:
		copy(buffer[:byteGroupSize], data[:byteGroupSize])
		return data[byteGroupSize:]
	}

	// the values of the bit length sit in the header bytes; a value of all ones says that the byte follows in the variable part
	var perByte, headerBytes int
	switch bitLen {
	case 1:
		perByte, headerBytes = 8, 2
	case 2:
		perByte, headerBytes = 4, 4
	default: // 4
		perByte, headerBytes = 2, 8
	}
	variable := data[headerBytes:]
	out := 0
	mask := byte(1<<bitLen - 1)
	for h := 0; h < headerBytes; h++ {
		b := data[h]
		if bitLen == 1 {
			// 1-bit values are in the reverse order of the other groups
			b = bits.Reverse8(b)
		}
		for i := 0; i < perByte; i++ {
			enc := b >> (8 - bitLen)
			b <<= bitLen
			if enc == mask {
				buffer[out] = variable[0]
				variable = variable[1:]
			} else {
				buffer[out] = enc
			}
			out++
		}
	}
	return variable
}

// decodeBytes decodes len(buffer) bytes (a multiple of the group size) in groups, each with the bit length that its two bits of the
// header choose from lengths. It returns the data after them, or false when the data is too short.
func decodeBytes(data []byte, buffer []byte, lengths []int) ([]byte, bool) {
	// the groups are counted up to four to get the number of header bytes
	headerSize := (len(buffer)/byteGroupSize + 3) / 4
	if len(data) < headerSize {
		return nil, false
	}
	header := data[:headerSize]
	data = data[headerSize:]

	for i := 0; i < len(buffer); i += byteGroupSize {
		if len(data) < byteGroupDecodeLimit {
			return nil, false
		}
		headerOffset := i / byteGroupSize
		bitsK := int(header[headerOffset/4]>>((headerOffset%4)*2)) & 3
		data = decodeBytesGroup(data, buffer[i:], lengths[bitsK])
	}
	return data, true
}

func rotate32(v uint32, r int) uint32 {
	return v<<r | v>>((32-r)&31)
}

func unzigzag8(v uint8) uint8    { return -(v & 1) ^ (v >> 1) }
func unzigzag16(v uint16) uint16 { return -(v & 1) ^ (v >> 1) }

// decodeDeltas decodes the columns k to k+width of the block: element i of the column is a delta from the one before, the first
// from the last vertex of the block before. width is 1, 2 or 4 bytes; with xor the deltas are xors with a rotation.
func decodeDeltas(buffer, transposed []byte, count, size, k, width int, lastVertex []byte, rot int, xor bool) {
	for col := 0; col < 4; col += width {
		offset := k + col
		var p uint32
		for j := 0; j < width; j++ {
			p |= uint32(lastVertex[k+col+j]) << (8 * j)
		}
		for i := 0; i < count; i++ {
			var v uint32
			for j := 0; j < width; j++ {
				v |= uint32(buffer[col*count+i+count*j]) << (8 * j)
			}
			switch {
			case xor:
				v = rotate32(v, rot) ^ p
			case width == 1:
				v = uint32(unzigzag8(uint8(v))) + p
			case width == 2:
				v = uint32(unzigzag16(uint16(v))) + p
			}
			for j := 0; j < width; j++ {
				transposed[offset+j] = byte(v >> (8 * j))
			}
			p = v
			offset += size
		}
	}
}

// decodeVertexBlock decodes one block of count vertices into vertices (count*size bytes) and updates lastVertex. It returns the
// data after the block, or false for data that is too short or a channel that is not valid.
func decodeVertexBlock(data []byte, vertices []byte, count, size int, lastVertex *[256]byte, channels []byte, version int) ([]byte, bool) {
	var buffer [vertexBlockMaxSize * 4]byte
	countAligned := (count + byteGroupSize - 1) &^ (byteGroupSize - 1)

	controlSize := 0
	if version != 0 {
		controlSize = size / 4
	}
	if len(data) < controlSize {
		return nil, false
	}
	control := data[:controlSize]
	data = data[controlSize:]

	for k := 0; k < size; k += 4 {
		var ctrlByte byte
		if version != 0 {
			ctrlByte = control[k/4]
		}

		for j := 0; j < 4; j++ {
			ctrl := int(ctrlByte>>(j*2)) & 3
			column := buffer[j*count : j*count+countAligned]
			switch ctrl {
			case 3:
				// literal encoding
				if len(data) < count {
					return nil, false
				}
				copy(column[:count], data[:count])
				data = data[count:]
			case 2:
				// zero encoding
				clear(column[:count])
			default:
				lengths := bitsV0[:]
				if version != 0 {
					lengths = bitsV1[ctrl:]
				}
				var ok bool
				if data, ok = decodeBytes(data, column, lengths); !ok {
					return nil, false
				}
			}
		}

		channel := 0
		if version != 0 {
			channel = int(channels[k/4])
		}
		switch channel & 3 {
		case 0:
			decodeDeltas(buffer[:], vertices, count, size, k, 1, lastVertex[:], 0, false)
		case 1:
			decodeDeltas(buffer[:], vertices, count, size, k, 2, lastVertex[:], 0, false)
		case 2:
			decodeDeltas(buffer[:], vertices, count, size, k, 4, lastVertex[:], (32-(channel>>4))&31, true)
		default:
			return nil, false // not a channel type
		}
	}

	copy(lastVertex[:size], vertices[size*(count-1):size*count])
	return data, true
}

// TagMeshoptVertexInvalid is the tag of the error of a vertex buffer that is malformed: a header that is not one, a version that is
// not known, data that is too short or too long for the count and the size, a channel that is not a type. The metadata has
// "reason", "count", "size" and "bytes".
//
// Flutter: no counterpart
const TagMeshoptVertexInvalid = "MeshoptVertexInvalid"
