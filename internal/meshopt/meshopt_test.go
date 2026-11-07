package meshopt

import (
	"bytes"
	"encoding/binary"
	"slices"
	"testing"

	"github.com/rm4n0s/errors"
)

// The vectors are those of demo/tests.cpp of meshoptimizer 1.3 (MIT license, third_party/meshoptimizer/LICENSE).

var (
	indexBuffer    = []uint32{0, 1, 2, 2, 1, 3, 4, 6, 5, 7, 8, 9}
	indexDataV0    = []byte{0xe0, 0xf0, 0x10, 0xfe, 0xff, 0xf0, 0x0c, 0xff, 0x02, 0x02, 0x02, 0x00, 0x76, 0x87, 0x56, 0x67, 0x78, 0xa9, 0x86, 0x65, 0x89, 0x68, 0x98, 0x01, 0x69, 0x00, 0x00}
	indexTricky    = []uint32{0, 1, 2, 2, 1, 3, 0, 1, 2, 2, 1, 5, 2, 1, 4}
	indexDataV1    = []byte{0xe1, 0xf0, 0x10, 0xfe, 0x1f, 0x3d, 0x00, 0x0a, 0x00, 0x76, 0x87, 0x56, 0x67, 0x78, 0xa9, 0x86, 0x65, 0x89, 0x68, 0x98, 0x01, 0x69, 0x00, 0x00}
	indexSequence  = []uint32{0, 1, 51, 2, 49, 1000}
	sequenceDataV1 = []byte{0xd1, 0x00, 0x04, 0xcd, 0x01, 0x04, 0x07, 0x98, 0x1f, 0x00, 0x00, 0x00, 0x00}
)

func TestDecodeIndexBuffer(t *testing.T) {
	more := slices.Clone(indexDataV0)
	more[0] = 0xe1
	tests := []struct {
		name string
		data []byte
		want []uint32
	}{
		{"v0", indexDataV0, indexBuffer},
		{"v1", indexDataV1, indexTricky},
		{"v1 more", more, indexBuffer},
		{"v1 three edges", []byte{0xe1, 0xf0, 0x20, 0x30, 0x40, 0x00, 0x76, 0x87, 0x56, 0x67, 0x78, 0xa9, 0x86, 0x65, 0x89, 0x68, 0x98, 0x01, 0x69, 0x00, 0x00}, []uint32{0, 1, 2, 1, 0, 3, 2, 1, 4, 0, 2, 5}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := make([]uint32, len(tt.want))
			if err := DecodeIndexBuffer(got, tt.data); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDecodeIndexSequence(t *testing.T) {
	got := make([]uint32, len(indexSequence))
	if err := DecodeIndexSequence(got, sequenceDataV1); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, indexSequence) {
		t.Errorf("got %v, want %v", got, indexSequence)
	}
}

func TestDecodeIndexInvalid(t *testing.T) {
	withByte := func(b []byte, i int, v byte) []byte {
		c := slices.Clone(b)
		c[i] = v
		return c
	}
	tests := []struct {
		name   string
		decode func() error
		route  string
		reason string
	}{
		{"buffer too short", func() error { return DecodeIndexBuffer(make([]uint32, 12), indexDataV0[:10]) }, "DecodeIndexBuffer.MeshoptIndexInvalid", "too short"},
		{"buffer bad header", func() error { return DecodeIndexBuffer(make([]uint32, 12), withByte(indexDataV0, 0, 0x10)) }, "DecodeIndexBuffer.MeshoptIndexInvalid", "bad header"},
		{"buffer unknown version", func() error { return DecodeIndexBuffer(make([]uint32, 12), withByte(indexDataV0, 0, 0xe2)) }, "DecodeIndexBuffer.MeshoptIndexInvalid", "unknown version"},
		{"buffer extra byte", func() error { return DecodeIndexBuffer(make([]uint32, 12), append(slices.Clone(indexDataV0), 0)) }, "DecodeIndexBuffer.MeshoptIndexInvalid", "extra data"},
		{"buffer not triangles", func() error { return DecodeIndexBuffer(make([]uint32, 10), indexDataV0) }, "DecodeIndexBuffer.MeshoptIndexInvalid", "the number of indices is not a multiple of three"},
		{"buffer fewer triangles", func() error { return DecodeIndexBuffer(make([]uint32, 9), indexDataV0) }, "DecodeIndexBuffer.MeshoptIndexInvalid", "extra data"},
		{"sequence too short", func() error { return DecodeIndexSequence(make([]uint32, 6), sequenceDataV1[:8]) }, "DecodeIndexSequence.MeshoptIndexInvalid", "too short"},
		{"sequence bad header", func() error { return DecodeIndexSequence(make([]uint32, 6), withByte(sequenceDataV1, 0, 0xe1)) }, "DecodeIndexSequence.MeshoptIndexInvalid", "bad header"},
		{"sequence unknown version", func() error { return DecodeIndexSequence(make([]uint32, 6), withByte(sequenceDataV1, 0, 0xd2)) }, "DecodeIndexSequence.MeshoptIndexInvalid", "unknown version"},
		{"sequence extra byte", func() error { return DecodeIndexSequence(make([]uint32, 5), sequenceDataV1) }, "DecodeIndexSequence.MeshoptIndexInvalid", "extra data"},
		{"sequence open vbyte", func() error {
			return DecodeIndexSequence(make([]uint32, 2), []byte{0xd1, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x00})
		}, "DecodeIndexSequence.MeshoptIndexInvalid", "truncated data"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.decode()
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
			if got := appErr.Metadata["reason"]; got != tt.reason {
				t.Errorf("reason = %v, want %q", got, tt.reason)
			}
		})
	}
}

// vertexBuffer is kVertexBuffer: four vertices of 12 bytes (three uint16, two uint8, two uint16).
func vertexBuffer() []byte {
	var out []byte
	for _, v := range [][7]uint16{{0, 0, 0, 0, 0, 0, 0}, {300, 0, 0, 0, 0, 500, 0}, {0, 300, 0, 0, 0, 0, 500}, {300, 300, 0, 0, 0, 500, 500}} {
		out = binary.LittleEndian.AppendUint16(out, v[0])
		out = binary.LittleEndian.AppendUint16(out, v[1])
		out = binary.LittleEndian.AppendUint16(out, v[2])
		out = append(out, byte(v[3]), byte(v[4]))
		out = binary.LittleEndian.AppendUint16(out, v[5])
		out = binary.LittleEndian.AppendUint16(out, v[6])
	}
	return out
}

func uint16s(v ...uint16) []byte {
	var out []byte
	for _, x := range v {
		out = binary.LittleEndian.AppendUint16(out, x)
	}
	return out
}

var (
	vertexDataV0 = []byte{
		0xa0, 0x01, 0x3f, 0x00, 0x00, 0x00, 0x58, 0x57, 0x58, 0x01, 0x26, 0x00, 0x00, 0x00, 0x01, 0x0c,
		0x00, 0x00, 0x00, 0x58, 0x01, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x3f, 0x00,
		0x00, 0x00, 0x17, 0x18, 0x17, 0x01, 0x26, 0x00, 0x00, 0x00, 0x01, 0x0c, 0x00, 0x00, 0x00, 0x17,
		0x01, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00,
	}
	vertexDataV1 = []byte{
		0xa1, 0xee, 0xaa, 0xee, 0x00, 0x4b, 0x4b, 0x4b, 0x00, 0x00, 0x4b, 0x00, 0x00, 0x7d, 0x7d, 0x7d,
		0x00, 0x00, 0x7d, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x62, 0x00, 0x62,
	}
	// vertexDataV1Custom is a valid v1 encoding that uses every feature of the format; no encoder makes it.
	vertexDataV1Custom = []byte{
		0xa1, 0xd4, 0x94, 0xd4, 0x01, 0x0e, 0x00, 0x58, 0x57, 0x58, 0x02, 0x02, 0x12, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x58, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x0e, 0x00, 0x7d, 0x7d, 0x7d, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x7d, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x62,
	}
)

func TestDecodeVertexBuffer(t *testing.T) {
	tests := []struct {
		name  string
		data  []byte
		count int
		size  int
		want  []byte
	}{
		{"v0", vertexDataV0, 4, 12, vertexBuffer()},
		{"v1", vertexDataV1, 4, 12, vertexBuffer()},
		{"v1 custom", vertexDataV1Custom, 4, 12, vertexBuffer()},
		{
			"v0 more", []byte{
				0xa0, 0x00, 0x01, 0x2a, 0xaa, 0xaa, 0xaa, 0x02, 0x04, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44,
				0x03, 0x00, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
				0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00,
			}, 16, 4, []byte{
				0, 0, 0, 0, 0, 1, 2, 8, 0, 2, 4, 16, 0, 3, 6, 24,
				0, 4, 8, 32, 0, 5, 10, 40, 0, 6, 12, 48, 0, 7, 14, 56,
				0, 8, 16, 64, 0, 9, 18, 72, 0, 10, 20, 80, 0, 11, 22, 88,
				0, 12, 24, 96, 0, 13, 26, 104, 0, 14, 28, 112, 0, 15, 30, 120,
			},
		},
		{
			"v0 mode 2", []byte{
				0xa0, 0x02, 0x08, 0x88, 0x88, 0x88, 0x88, 0x88, 0x88, 0x88, 0x02, 0x0a, 0xaa, 0xaa, 0xaa, 0xaa,
				0xaa, 0xaa, 0xaa, 0x02, 0x0c, 0xcc, 0xcc, 0xcc, 0xcc, 0xcc, 0xcc, 0xcc, 0x02, 0x0e, 0xee, 0xee,
				0xee, 0xee, 0xee, 0xee, 0xee, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00,
			}, 16, 4, []byte{
				0, 0, 0, 0, 4, 5, 6, 7, 8, 10, 12, 14, 12, 15, 18, 21,
				16, 20, 24, 28, 20, 25, 30, 35, 24, 30, 36, 42, 28, 35, 42, 49,
				32, 40, 48, 56, 36, 45, 54, 63, 40, 50, 60, 70, 44, 55, 66, 77,
				48, 60, 72, 84, 52, 65, 78, 91, 56, 70, 84, 98, 60, 75, 90, 105,
			},
		},
		{
			"v1 deltas", []byte{
				0xa1, 0x99, 0x99, 0x01, 0x2a, 0xaa, 0xaa, 0xaa, 0x02, 0x04, 0x44, 0x44, 0x44, 0x43, 0x33, 0x33,
				0x33, 0x02, 0x06, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x02, 0x08, 0x88, 0x88, 0x88, 0x87,
				0x77, 0x77, 0x77, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0xf8, 0x00, 0xf8, 0x00, 0xf0, 0x00, 0xf0, 0x00, 0x01, 0x01,
			}, 16, 8, uint16s(
				248, 248, 240, 240, 249, 250, 243, 244, 250, 252, 246, 248, 251, 254, 249, 252,
				252, 256, 252, 256, 253, 258, 255, 260, 254, 260, 258, 264, 255, 262, 261, 268,
				256, 264, 264, 272, 257, 262, 267, 268, 258, 260, 270, 264, 259, 258, 273, 260,
				260, 256, 276, 256, 261, 254, 279, 252, 262, 252, 282, 248, 263, 250, 285, 244,
			),
		},
		{
			"v1 sentinel count", []byte{
				0xa1, 0xa9,
				0x01, 0xfc, 0x3c, 0xcc, 0xcc,
				0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
				0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00,
			}, 13, 4, []byte{
				0xff, 0, 0, 0, 0xfe, 0, 0, 0, 0xfd, 0, 0, 0, 0xfd, 0, 0, 0,
				0xfd, 0, 0, 0, 0xfc, 0, 0, 0, 0xfb, 0, 0, 0, 0xfb, 0, 0, 0,
				0xfa, 0, 0, 0, 0xfa, 0, 0, 0, 0xf9, 0, 0, 0, 0xf9, 0, 0, 0,
				0xf8, 0, 0, 0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := make([]byte, len(tt.want))
			if err := DecodeVertexBuffer(got, tt.count, tt.size, tt.data); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDecodeVertexInvalid(t *testing.T) {
	broken := slices.Clone(vertexDataV0)
	broken[0] = 0
	unknown := slices.Clone(vertexDataV0)
	unknown[0] = 0xa2
	// a channel that is not a type: channel 3 in the tail of a v1 buffer
	badChannel := slices.Clone(vertexDataV1)
	badChannel[len(badChannel)-3] = 3
	tests := []struct {
		name   string
		dst    int
		count  int
		size   int
		data   []byte
		reason string
	}{
		{"empty", 48, 4, 12, nil, "too short"},
		{"short", 48, 4, 12, vertexDataV0[:20], "too short"},
		{"extra byte", 48, 4, 12, append(slices.Clone(vertexDataV0), 0), "extra data"},
		{"bad header", 48, 4, 12, broken, "bad header"},
		{"unknown version", 48, 4, 12, unknown, "unknown version"},
		{"bad channel", 48, 4, 12, badChannel, "truncated data"},
		{"truncated blocks", 48, 4, 12, vertexDataV0[:70], "truncated data"},
		{"bad size", 48, 4, 10, vertexDataV0, "the vertex size is not a multiple of four up to 256"},
		{"bad destination", 40, 4, 12, vertexDataV0, "the destination does not have count*size bytes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DecodeVertexBuffer(make([]byte, tt.dst), tt.count, tt.size, tt.data)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute("DecodeVertexBuffer.MeshoptVertexInvalid") {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
			if got := appErr.Metadata["reason"]; got != tt.reason {
				t.Errorf("reason = %v, want %q", got, tt.reason)
			}
		})
	}
}

func TestDecodeFilter(t *testing.T) {
	u32s := func(v ...uint32) []byte {
		var out []byte
		for _, x := range v {
			out = binary.LittleEndian.AppendUint32(out, x)
		}
		return out
	}
	tests := []struct {
		name   string
		filter Filter
		stride int
		data   []byte
		want   []byte
	}{
		{"oct8", FilterOctahedral, 4, []byte{0, 1, 127, 0, 0, 187, 127, 1, 255, 1, 127, 0, 14, 130, 127, 1}, []byte{0, 1, 127, 0, 0, 159, 82, 1, 255, 1, 127, 0, 1, 130, 241, 1}},
		{"oct12", FilterOctahedral, 8,
			uint16s(0, 1, 2047, 0, 0, 1870, 2047, 1, 2017, 1, 2047, 0, 14, 1300, 2047, 1),
			uint16s(0, 16, 32767, 0, 0, 32621, 3088, 1, 32764, 16, 471, 0, 307, 28541, 16093, 1)},
		{"quat12", FilterQuaternion, 8,
			uint16s(0, 1, 0, 0x7fc, 0, 1870, 0, 0x7fd, 2017, 1, 0, 0x7fe, 14, 1300, 0, 0x7ff),
			uint16s(32767, 0, 11, 0, 0, 25013, 0, 21166, 11, 0, 23504, 22830, 158, 14715, 0, 29277)},
		{"exp", FilterExponential, 4, u32s(0, 0xff000003, 0x02fffff7, 0xfe7fffff), u32s(0, 0x3fc00000, 0xc2100000, 0x49fffffe)},
		{"none", FilterNone, 4, []byte{1, 2, 3, 4}, []byte{1, 2, 3, 4}},
	}
	for _, tt := range tests {
		// a count of 4 is the aligned path of the original, a count of 3 its tail
		for _, count := range []int{len(tt.data) / tt.stride, len(tt.data)/tt.stride - 1} {
			if count < 1 {
				continue
			}
			t.Run(tt.name, func(t *testing.T) {
				got := slices.Clone(tt.data)
				if err := DecodeFilter(tt.filter, got, count, tt.stride); err != nil {
					t.Fatal(err)
				}
				n := count * tt.stride
				if !bytes.Equal(got[:n], tt.want[:n]) {
					t.Errorf("got %v, want %v", got[:n], tt.want[:n])
				}
				if !bytes.Equal(got[n:], tt.data[n:]) {
					t.Errorf("data after the count changed: %v", got[n:])
				}
			})
		}
	}
}

func TestDecodeFilterColor(t *testing.T) {
	// y co cg a of 8 bits: a full scale white, and a gray of 64 of an alpha scale of 127 (its alpha has one bit less)
	got := []byte{255, 0, 0, 255, 64, 0, 0, 127}
	if err := DecodeFilter(FilterColor, got, 2, 4); err != nil {
		t.Fatal(err)
	}
	want := []byte{255, 255, 255, 255, 129, 129, 129, 255}
	if !bytes.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDecodeFilterInvalid(t *testing.T) {
	tests := []struct {
		name   string
		filter Filter
		count  int
		stride int
		size   int
	}{
		{"octahedral stride", FilterOctahedral, 1, 6, 6},
		{"quaternion stride", FilterQuaternion, 1, 4, 4},
		{"exponential stride", FilterExponential, 1, 6, 6},
		{"color stride", FilterColor, 1, 12, 12},
		{"unknown filter", Filter(9), 1, 4, 4},
		{"short data", FilterOctahedral, 2, 4, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DecodeFilter(tt.filter, make([]byte, tt.size), tt.count, tt.stride)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute("DecodeFilter.MeshoptFilterInvalid") {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
		})
	}
}

func FuzzDecode(f *testing.F) {
	f.Add(indexDataV0, uint8(0), uint8(12))
	f.Add(indexDataV1, uint8(0), uint8(15))
	f.Add(sequenceDataV1, uint8(1), uint8(6))
	f.Add(vertexDataV0, uint8(2), uint8(4))
	f.Add(vertexDataV1, uint8(2), uint8(4))
	f.Add(vertexDataV1Custom, uint8(2), uint8(4))
	f.Fuzz(func(t *testing.T, data []byte, kind, count uint8) {
		n := int(count)
		switch kind % 3 {
		case 0:
			_ = DecodeIndexBuffer(make([]uint32, n-n%3), data)
		case 1:
			_ = DecodeIndexSequence(make([]uint32, n), data)
		default:
			for _, size := range []int{4, 8, 12, 16, 64, 256} {
				_ = DecodeVertexBuffer(make([]byte, n*size), n, size, data)
			}
		}
	})
}

func FuzzDecodeFilter(f *testing.F) {
	f.Add(uint8(1), uint8(4), []byte{0, 1, 127, 0, 0, 187, 127, 1})
	f.Add(uint8(2), uint8(8), uint16s(0, 1, 0, 0x7fc))
	f.Add(uint8(3), uint8(4), []byte{3, 0, 0, 0xff})
	f.Add(uint8(4), uint8(4), []byte{255, 0, 0, 255})
	f.Fuzz(func(t *testing.T, filter, stride uint8, data []byte) {
		if stride == 0 {
			return
		}
		_ = DecodeFilter(Filter(filter%6), data, len(data)/int(stride), int(stride))
	})
}
