package gltf_test

import (
	"encoding/base64"
	"encoding/json"
	"slices"
	"testing"
)

// The compressed data of these tests are the vectors of demo/tests.cpp of meshoptimizer 1.3 (MIT license,
// third_party/meshoptimizer/LICENSE), or a stream that encodeLiteralV1 makes.

var (
	meshoptIndexDataV0   = []byte{0xe0, 0xf0, 0x10, 0xfe, 0xff, 0xf0, 0x0c, 0xff, 0x02, 0x02, 0x02, 0x00, 0x76, 0x87, 0x56, 0x67, 0x78, 0xa9, 0x86, 0x65, 0x89, 0x68, 0x98, 0x01, 0x69, 0x00, 0x00}
	meshoptSequenceDataV = []byte{0xd1, 0x00, 0x04, 0xcd, 0x01, 0x04, 0x07, 0x98, 0x1f, 0x00, 0x00, 0x00, 0x00}
)

// encodeLiteralV1 is a version 1 vertex stream of vertices of 4 bytes where every column is stored as literal bytes of zigzag
// deltas (the format's simplest encoding), for at most 256 vertices.
func encodeLiteralV1(vertices []byte) []byte {
	n := len(vertices) / 4
	out := []byte{0xa1, 0xff}
	for col := range 4 {
		var prev byte
		for i := range n {
			cur := vertices[4*i+col]
			d := int8(cur - prev)
			out = append(out, byte(d<<1)^byte(d>>7))
			prev = cur
		}
	}
	// the data ends with the tail: padded to 24 bytes, the last vertex (all zero for the first block), and the channels (bytes)
	out = append(out, make([]byte, 19)...)
	out = append(out, 0, 0, 0, 0, 0)
	return out
}

// meshoptFile is a glTF file whose first buffer view is compressed: compressed is the first part of buffer 0, and plain the rest
// (accessors over it come after). extension is the JSON of the extension of the view; name is the extension's name.
type meshoptFile struct {
	name       string
	compressed []byte
	extension  obj
	plain      []byte
	accessors  list
	primitive  obj
	extra      obj
}

func (f meshoptFile) json(t testing.TB) []byte {
	t.Helper()
	data := pad(append([]byte(nil), f.compressed...), 0)
	plainStart := len(data)
	data = append(data, f.plain...)
	ext := obj{"buffer": 0, "byteOffset": 0, "byteLength": len(f.compressed)}
	for k, v := range f.extension {
		ext[k] = v
	}
	views := list{
		obj{"buffer": 1, "byteOffset": 0, "byteLength": 16, "extensions": obj{f.name: ext}},
		obj{"buffer": 0, "byteOffset": plainStart, "byteLength": len(f.plain)},
	}
	doc := obj{
		"asset":              obj{"version": "2.0"},
		"extensionsUsed":     list{f.name, "KHR_mesh_quantization"},
		"extensionsRequired": list{f.name},
		"buffers": list{
			obj{"byteLength": len(data), "uri": "data:application/octet-stream;base64," + base64.StdEncoding.EncodeToString(data)},
			obj{"byteLength": 16, "extensions": obj{f.name: obj{"fallback": true}}},
		},
		"bufferViews": views,
		"accessors":   f.accessors,
		"meshes":      list{obj{"primitives": list{f.primitive}}},
		"nodes":       list{obj{"mesh": 0}},
		"scenes":      list{obj{"nodes": list{0}}},
	}
	for k, v := range f.extra {
		doc[k] = v
	}
	js, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return js
}

func TestMeshoptFilterThroughGLTF(t *testing.T) {
	// the octahedral vectors of meshoptimizer: four normals of 4 bytes, and what the filter makes of them
	input := []byte{0, 1, 127, 0, 0, 187, 127, 1, 255, 1, 127, 0, 14, 130, 127, 1}
	decoded := []byte{0, 1, 127, 0, 0, 159, 82, 1, 255, 1, 127, 0, 1, 130, 241, 1}
	var want []float32
	for i := range 4 {
		for c := range 3 {
			want = append(want, max(float32(int8(decoded[4*i+c]))/127, -1))
		}
	}
	positions := f32s(0, 0, 0, 1, 0, 0, 0, 1, 0, 1, 1, 0)
	for _, name := range []string{"EXT_meshopt_compression", "KHR_meshopt_compression"} {
		t.Run(name, func(t *testing.T) {
			f := meshoptFile{
				name: name, compressed: encodeLiteralV1(input),
				extension: obj{"byteStride": 4, "mode": "ATTRIBUTES", "filter": "OCTAHEDRAL", "count": 4},
				plain:     positions,
				accessors: list{
					obj{"bufferView": 0, "componentType": ctByte, "normalized": true, "type": "VEC3", "count": 4},
					obj{"bufferView": 1, "componentType": ctFloat, "type": "VEC3", "count": 4},
				},
				primitive: obj{"attributes": obj{"NORMAL": 0, "POSITION": 1}, "mode": 0},
			}
			m := mustDecode(t, f.json(t))
			if got := m.Meshes[0].Normals; !approx(got, want, 1e-6) {
				t.Errorf("normals %v, want %v", got, want)
			}
		})
	}

	// without the filter the bytes are the input
	f := meshoptFile{
		name: "EXT_meshopt_compression", compressed: encodeLiteralV1(input),
		extension: obj{"byteStride": 4, "mode": "ATTRIBUTES", "count": 4},
		plain:     positions,
		accessors: list{
			obj{"bufferView": 0, "componentType": ctUbyte, "normalized": true, "type": "VEC4", "count": 4},
			obj{"bufferView": 1, "componentType": ctFloat, "type": "VEC3", "count": 4},
		},
		primitive: obj{"attributes": obj{"COLOR_0": 0, "POSITION": 1}, "mode": 0},
	}
	m := mustDecode(t, f.json(t))
	if len(m.Meshes[0].Colors) == 0 {
		t.Errorf("colors")
	}
}

func TestMeshoptIndices(t *testing.T) {
	var positions []float32
	for i := range 10 {
		positions = append(positions, float32(i), 0, 0)
	}
	want := []uint32{0, 1, 2, 2, 1, 3, 4, 6, 5, 7, 8, 9}
	tests := []struct {
		name   string
		stride int
		ct     int
	}{
		{"uint32", 4, ctUint},
		{"uint16", 2, ctUshort},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := meshoptFile{
				name: "EXT_meshopt_compression", compressed: meshoptIndexDataV0,
				extension: obj{"byteStride": tt.stride, "mode": "TRIANGLES", "count": 12},
				plain:     f32s(positions...),
				accessors: list{
					obj{"bufferView": 0, "componentType": tt.ct, "type": "SCALAR", "count": 12},
					obj{"bufferView": 1, "componentType": ctFloat, "type": "VEC3", "count": 10},
				},
				primitive: obj{"attributes": obj{"POSITION": 1}, "indices": 0},
			}
			m := mustDecode(t, f.json(t))
			if got := m.Meshes[0].Indices; !slices.Equal(got, want) {
				t.Errorf("indices %v, want %v", got, want)
			}
		})
	}

	t.Run("a sequence", func(t *testing.T) {
		var many []float32
		for range 1001 {
			many = append(many, 0, 0, 0)
		}
		f := meshoptFile{
			name: "EXT_meshopt_compression", compressed: meshoptSequenceDataV,
			extension: obj{"byteStride": 4, "mode": "INDICES", "count": 6},
			plain:     f32s(many...),
			accessors: list{
				obj{"bufferView": 0, "componentType": ctUint, "type": "SCALAR", "count": 6},
				obj{"bufferView": 1, "componentType": ctFloat, "type": "VEC3", "count": 1001},
			},
			primitive: obj{"attributes": obj{"POSITION": 1}, "indices": 0, "mode": 0},
		}
		m := mustDecode(t, f.json(t))
		if got, want := m.Meshes[0].Indices, []uint32{0, 1, 51, 2, 49, 1000}; !slices.Equal(got, want) {
			t.Errorf("indices %v, want %v", got, want)
		}
	})
}

func TestMeshoptInvalid(t *testing.T) {
	positions := f32s(0, 0, 0, 1, 0, 0, 0, 1, 0, 1, 1, 0)
	input := []byte{0, 1, 127, 0, 0, 187, 127, 1, 255, 1, 127, 0, 14, 130, 127, 1}
	attributes := func(compressed []byte, ext obj) meshoptFile {
		return meshoptFile{
			name: "EXT_meshopt_compression", compressed: compressed, extension: ext, plain: positions,
			accessors: list{
				obj{"bufferView": 0, "componentType": ctUbyte, "normalized": true, "type": "VEC4", "count": 4},
				obj{"bufferView": 1, "componentType": ctFloat, "type": "VEC3", "count": 4},
			},
			primitive: obj{"attributes": obj{"COLOR_0": 0, "POSITION": 1}, "mode": 0},
		}
	}
	good := encodeLiteralV1(input)
	broken := slices.Clone(good)
	broken[0] = 0x10
	extra := append(slices.Clone(good), 0)
	const where = "/bufferViews/0/extensions/EXT_meshopt_compression"
	tests := []struct {
		name   string
		file   meshoptFile
		reason string
		cause  string
	}{
		{"a header that is not one", attributes(broken, obj{"byteStride": 4, "mode": "ATTRIBUTES", "count": 4}), "meshopt", "bad header"},
		{"data that is too long", attributes(extra, obj{"byteStride": 4, "mode": "ATTRIBUTES", "count": 4}), "meshopt", "extra data"},
		{"data that is too short", attributes(good[:20], obj{"byteStride": 4, "mode": "ATTRIBUTES", "count": 4}), "meshopt", "too short"},
		{"more vertices than the data has", attributes(good, obj{"byteStride": 4, "mode": "ATTRIBUTES", "count": 8}), "meshopt", "extra data"},
		{"a filter that does not fit the stride", attributes(good, obj{"byteStride": 4, "mode": "ATTRIBUTES", "count": 4, "filter": "QUATERNION"}), "meshopt", "the stride of the quaternion filter is 8"},
		{"an unknown filter", attributes(good, obj{"byteStride": 4, "mode": "ATTRIBUTES", "count": 4, "filter": "SPHERICAL"}), "bad filter", ""},
		{"an unknown mode", attributes(good, obj{"byteStride": 4, "mode": "VERTICES", "count": 4}), "bad mode", ""},
		{"no mode", attributes(good, obj{"byteStride": 4, "count": 4}), "missing property", ""},
		{"no count", attributes(good, obj{"byteStride": 4, "mode": "ATTRIBUTES"}), "missing property", ""},
		{"a count of 0", attributes(good, obj{"byteStride": 4, "mode": "ATTRIBUTES", "count": 0}), "bad count", ""},
		{"a stride that is not a multiple of four", attributes(good, obj{"byteStride": 6, "mode": "ATTRIBUTES", "count": 4}), "bad byteStride", ""},
		{"indices with a filter", attributes(meshoptIndexDataV0, obj{"byteStride": 4, "mode": "TRIANGLES", "count": 12, "filter": "OCTAHEDRAL"}), "bad filter", ""},
		{"indices of a stride of 3", attributes(meshoptIndexDataV0, obj{"byteStride": 3, "mode": "TRIANGLES", "count": 12}), "bad byteStride", ""},
		{"indices of the wrong count", attributes(meshoptIndexDataV0, obj{"byteStride": 4, "mode": "TRIANGLES", "count": 9}), "meshopt", "extra data"},
		{"indices that are not triangles", attributes(meshoptIndexDataV0, obj{"byteStride": 4, "mode": "TRIANGLES", "count": 10}), "meshopt", "the number of indices is not a multiple of three"},
		{"a sequence that does not fit 16 bits", attributes([]byte{0xd1, 0xc0, 0x8b, 0x11, 0, 0, 0, 0}, obj{"byteStride": 2, "mode": "INDICES", "count": 1}), "index out of range", ""},
		{"compressed data past its buffer", attributes(good, obj{"byteStride": 4, "mode": "ATTRIBUTES", "count": 4, "byteLength": 1 << 20}), "view out of bounds", ""},
		{"a buffer that is not there", attributes(good, obj{"byteStride": 4, "mode": "ATTRIBUTES", "count": 4, "buffer": 5}), "index out of range", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decodeJSON(t, tt.file.json(t))
			wantFn := "loader.meshoptView.GLTFInvalid"
			meta := map[string]any{"reason": tt.reason}
			if tt.cause != "" {
				meta["cause"] = tt.cause
			}
			if tt.name == "a buffer that is not there" {
				wantFn = "loader.buffer.GLTFInvalid"
			}
			if tt.name != "a buffer that is not there" {
				meta["path"] = where
			}
			failure(t, err, wantFn, meta)
		})
	}
}
