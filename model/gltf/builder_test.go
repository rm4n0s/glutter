package gltf_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"math"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/model"
	"github.com/rm4n0s/glutter/model/gltf"
)

// obj is a JSON object of a glTF file that a test writes.
type obj = map[string]any

// list is a JSON array.
type list = []any

// The component types of an accessor.
const (
	ctByte   = 5120
	ctUbyte  = 5121
	ctShort  = 5122
	ctUshort = 5123
	ctUint   = 5125
	ctFloat  = 5126
)

// builder makes the glTF file of a test: a binary buffer with buffer views, accessors and the JSON around them.
type builder struct {
	bin       bytes.Buffer
	views     list
	accessors list
	doc       obj
}

func newBuilder() *builder {
	return &builder{doc: obj{"asset": obj{"version": "2.0"}}}
}

func f32s(v ...float32) []byte {
	var out []byte
	for _, x := range v {
		out = binary.LittleEndian.AppendUint32(out, math.Float32bits(x))
	}
	return out
}

func u16s(v ...uint16) []byte {
	var out []byte
	for _, x := range v {
		out = binary.LittleEndian.AppendUint16(out, x)
	}
	return out
}

func u32s(v ...uint32) []byte {
	var out []byte
	for _, x := range v {
		out = binary.LittleEndian.AppendUint32(out, x)
	}
	return out
}

// view adds a buffer view of the bytes, at a multiple of four bytes of the buffer, and returns its index. stride is the
// byteStride, 0 for none.
func (b *builder) view(data []byte, stride int) int {
	for b.bin.Len()%4 != 0 {
		b.bin.WriteByte(0)
	}
	v := obj{"buffer": 0, "byteOffset": b.bin.Len(), "byteLength": len(data)}
	if stride != 0 {
		v["byteStride"] = stride
	}
	b.bin.Write(data)
	b.views = append(b.views, v)
	return len(b.views) - 1
}

// accessor adds an accessor over the buffer view and returns its index; extra is merged into the JSON of the accessor.
func (b *builder) accessor(view, componentType int, typ string, count int, extra obj) int {
	a := obj{"componentType": componentType, "type": typ, "count": count}
	if view >= 0 {
		a["bufferView"] = view
	}
	for k, v := range extra {
		a[k] = v
	}
	b.accessors = append(b.accessors, a)
	return len(b.accessors) - 1
}

// data makes an accessor of the bytes in a view of their own.
func (b *builder) data(data []byte, componentType int, typ string, count int, extra obj) int {
	return b.accessor(b.view(data, 0), componentType, typ, count, extra)
}

// json is the JSON of the file, with the buffer as a data URI.
func (b *builder) json() []byte {
	doc := obj{}
	for k, v := range b.doc {
		doc[k] = v
	}
	if b.bin.Len() > 0 {
		doc["buffers"] = list{obj{"byteLength": b.bin.Len(), "uri": "data:application/octet-stream;base64," + base64.StdEncoding.EncodeToString(b.bin.Bytes())}}
		doc["bufferViews"] = b.views
		doc["accessors"] = b.accessors
	}
	out, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	return out
}

// glb is the file as a binary container, the buffer in its chunk.
func (b *builder) glb() []byte {
	doc := obj{}
	for k, v := range b.doc {
		doc[k] = v
	}
	if b.bin.Len() > 0 {
		doc["buffers"] = list{obj{"byteLength": b.bin.Len()}}
		doc["bufferViews"] = b.views
		doc["accessors"] = b.accessors
	}
	js, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	return makeGLB(js, b.bin.Bytes())
}

func pad(data []byte, with byte) []byte {
	for len(data)%4 != 0 {
		data = append(data, with)
	}
	return data
}

// makeGLB is the container of a JSON chunk and, when bin is not empty, a binary chunk.
func makeGLB(js, bin []byte) []byte {
	js = pad(append([]byte(nil), js...), ' ')
	bin = pad(append([]byte(nil), bin...), 0)
	total := 12 + 8 + len(js)
	if len(bin) > 0 {
		total += 8 + len(bin)
	}
	out := u32s(0x46546C67, 2, uint32(total))
	out = append(out, u32s(uint32(len(js)), 0x4E4F534A)...)
	out = append(out, js...)
	if len(bin) > 0 {
		out = append(out, u32s(uint32(len(bin)), 0x004E4942)...)
		out = append(out, bin...)
	}
	return out
}

// triangle adds a triangle with positions (and the extra attributes) and returns the index of its accessor of positions; the
// document gets a mesh, a node and a scene when withScene is true.
func (b *builder) triangle() int {
	return b.data(f32s(0, 0, 0, 1, 0, 0, 0, 1, 0), ctFloat, "VEC3", 3, obj{"min": list{0, 0, 0}, "max": list{1, 1, 0}})
}

// scene puts the mesh made of the primitives in a node of a scene.
func (b *builder) scene(primitives ...obj) {
	var ps list
	for _, p := range primitives {
		ps = append(ps, p)
	}
	b.doc["meshes"] = list{obj{"primitives": ps}}
	b.doc["nodes"] = list{obj{"mesh": 0}}
	b.doc["scenes"] = list{obj{"nodes": list{0}}}
	b.doc["scene"] = 0
}

// oneTriangle is the file of a triangle of positions.
func oneTriangle() *builder {
	b := newBuilder()
	p := b.triangle()
	b.scene(obj{"attributes": obj{"POSITION": p}})
	return b
}

// decodeJSON reads the JSON of a builder.
func decodeJSON(t testing.TB, js []byte) (*model.Model, error) {
	t.Helper()
	return gltf.Decode(context.Background(), bytes.NewReader(js), gltf.Options{Logger: discard()})
}

func mustDecode(t testing.TB, js []byte) *model.Model {
	t.Helper()
	m, err := decodeJSON(t, js)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	return m
}

// failure checks that err is a tagged error with the route and, for the metadata, the pairs of want.
func failure(t testing.TB, err error, route string, want map[string]any) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected the error %s, got none", route)
	}
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if !appErr.HasRoute(route) {
		t.Errorf("unexpected failure path: %s (want %s)", appErr.Route(), route)
	}
	for k, v := range want {
		if got := appErr.Metadata[k]; got != v {
			t.Errorf("metadata %q = %v, want %v", k, got, v)
		}
	}
}
