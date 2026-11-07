package gltf_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"math"
	"slices"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/rm4n0s/glutter/model"
	"github.com/rm4n0s/glutter/model/gltf"
)

func discard() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// records is a log handler that keeps the messages.
type records struct {
	mu   sync.Mutex
	msgs []string
}

func (r *records) Enabled(context.Context, slog.Level) bool { return true }
func (r *records) Handle(_ context.Context, rec slog.Record) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.msgs = append(r.msgs, rec.Message)
	return nil
}
func (r *records) WithAttrs([]slog.Attr) slog.Handler { return r }
func (r *records) WithGroup(string) slog.Handler      { return r }

func approx(a, b []float32, eps float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if math.Abs(float64(a[i]-b[i])) > eps {
			return false
		}
	}
	return true
}

func TestTriangle(t *testing.T) {
	m := mustDecode(t, oneTriangle().json())
	if len(m.Meshes) != 1 || len(m.Nodes) != 1 {
		t.Fatalf("meshes %d nodes %d", len(m.Meshes), len(m.Nodes))
	}
	mesh := m.Meshes[0]
	if want := []float32{0, 0, 0, 1, 0, 0, 0, 1, 0}; !slices.Equal(mesh.Positions, want) {
		t.Errorf("positions %v, want %v", mesh.Positions, want)
	}
	if want := []uint32{0, 1, 2}; !slices.Equal(mesh.Indices, want) {
		t.Errorf("indices %v, want %v", mesh.Indices, want)
	}
	if m.Units.MetersPerUnit != 1 || m.TriangleCount() != 1 {
		t.Errorf("units %v, triangles %d", m.Units, m.TriangleCount())
	}
	if m.Nodes[0].TRS == nil || m.Nodes[0].Transform != ([16]float64)(model.NewNode("").Transform) {
		t.Errorf("a node with no transform has the identity")
	}
}

func TestAccessorComponents(t *testing.T) {
	tests := []struct {
		name       string
		ct         int
		normalized bool
		data       []byte
		want       []float32
	}{
		{"float", ctFloat, false, f32s(0.25, 0.75), []float32{0.25, 0.75}},
		{"ubyte normalized", ctUbyte, true, []byte{0, 255}, []float32{0, 1}},
		{"ushort normalized", ctUshort, true, u16s(0, 65535), []float32{0, 1}},
		{"byte normalized", ctByte, true, []byte{0x81, 127}, []float32{-1, 1}},
		{"byte normalized low", ctByte, true, []byte{0x80, 0}, []float32{-1, 0}},
		{"short normalized", ctShort, true, u16s(0x8000, 32767), []float32{-1, 1}},
		{"ubyte", ctUbyte, false, []byte{3, 200}, []float32{3, 200}},
		{"byte", ctByte, false, []byte{0xff, 5}, []float32{-1, 5}},
		{"ushort", ctUshort, false, u16s(1000, 65535), []float32{1000, 65535}},
		{"short", ctShort, false, u16s(0xfff0, 7), []float32{-16, 7}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newBuilder()
			b.doc["extensionsUsed"] = list{"KHR_mesh_quantization"}
			p := b.triangle()
			// two texture coordinates for each of the three vertices, from the table's two values
			var raw []byte
			for range 3 {
				raw = append(raw, tt.data...)
			}
			uv := b.data(raw, tt.ct, "VEC2", 3, obj{"normalized": tt.normalized})
			b.scene(obj{"attributes": obj{"POSITION": p, "TEXCOORD_0": uv}})
			m := mustDecode(t, b.json())
			var want []float32
			for range 3 {
				want = append(want, tt.want...)
			}
			if got := m.Meshes[0].TexCoords; !approx(got, want, 1e-6) {
				t.Errorf("texture coordinates %v, want %v", got, want)
			}
		})
	}
}

func TestStridedAccessor(t *testing.T) {
	// positions and normals interleaved in one view of a stride of 24, and 4 bytes of padding after each index of a stride of 8
	b := newBuilder()
	inter := f32s(0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 1, 0, 1, 0, 0, 0, 1)
	v := b.view(inter, 24)
	pos := b.accessor(v, ctFloat, "VEC3", 3, nil)
	nrm := b.accessor(v, ctFloat, "VEC3", 3, obj{"byteOffset": 12})
	idx := b.accessor(b.view(append(u16s(2, 0, 0, 0), append(u16s(1, 0, 0, 0), u16s(0, 0, 0, 0)...)...), 8), ctUshort, "SCALAR", 3, nil)
	b.scene(obj{"attributes": obj{"POSITION": pos, "NORMAL": nrm}, "indices": idx})
	m := mustDecode(t, b.json())
	mesh := m.Meshes[0]
	if want := []float32{0, 0, 0, 1, 0, 0, 0, 1, 0}; !slices.Equal(mesh.Positions, want) {
		t.Errorf("positions %v", mesh.Positions)
	}
	if want := []float32{0, 0, 1, 0, 0, 1, 0, 0, 1}; !slices.Equal(mesh.Normals, want) {
		t.Errorf("normals %v", mesh.Normals)
	}
	if want := []uint32{2, 1, 0}; !slices.Equal(mesh.Indices, want) {
		t.Errorf("indices %v", mesh.Indices)
	}
}

func TestSparseAccessor(t *testing.T) {
	tests := []struct {
		name    string
		indices []byte
		ict     int
		base    bool
		want    []float32
	}{
		{"on a base, ubyte indices", []byte{1, 2}, ctUbyte, true, []float32{0, 0, 0, 9, 9, 9, 8, 8, 8}},
		{"on a base, ushort indices", u16s(0, 2), ctUshort, true, []float32{9, 9, 9, 1, 0, 0, 8, 8, 8}},
		{"on a base, uint indices", u32s(0, 1), ctUint, true, []float32{9, 9, 9, 8, 8, 8, 0, 1, 0}},
		{"on zeros", []byte{1, 2}, ctUbyte, false, []float32{0, 0, 0, 9, 9, 9, 8, 8, 8}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newBuilder()
			base := -1
			if tt.base {
				base = b.view(f32s(0, 0, 0, 1, 0, 0, 0, 1, 0), 0)
			}
			sparse := obj{
				"count":   2,
				"indices": obj{"bufferView": b.view(tt.indices, 0), "componentType": tt.ict},
				"values":  obj{"bufferView": b.view(f32s(9, 9, 9, 8, 8, 8), 0)},
			}
			pos := b.accessor(base, ctFloat, "VEC3", 3, obj{"sparse": sparse})
			b.scene(obj{"attributes": obj{"POSITION": pos}})
			m := mustDecode(t, b.json())
			if got := m.Meshes[0].Positions; !slices.Equal(got, tt.want) {
				t.Errorf("positions %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIndexTypes(t *testing.T) {
	tests := []struct {
		name string
		ct   int
		data []byte
	}{
		{"ubyte", ctUbyte, []byte{2, 1, 0}},
		{"ushort", ctUshort, u16s(2, 1, 0)},
		{"uint", ctUint, u32s(2, 1, 0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newBuilder()
			p := b.triangle()
			i := b.data(tt.data, tt.ct, "SCALAR", 3, nil)
			b.scene(obj{"attributes": obj{"POSITION": p}, "indices": i})
			m := mustDecode(t, b.json())
			if want := []uint32{2, 1, 0}; !slices.Equal(m.Meshes[0].Indices, want) {
				t.Errorf("indices %v", m.Meshes[0].Indices)
			}
		})
	}
}

func TestPrimitiveModes(t *testing.T) {
	tests := []struct {
		name      string
		mode      int
		indices   []uint16
		primitive model.Primitive
		want      []uint32
	}{
		{"points", 0, []uint16{0, 1, 2}, model.PrimitivePoints, []uint32{0, 1, 2}},
		{"lines", 1, []uint16{0, 1, 1, 2}, model.PrimitiveLines, []uint32{0, 1, 1, 2}},
		{"line loop", 2, []uint16{0, 1, 2}, model.PrimitiveLines, []uint32{0, 1, 1, 2, 2, 0}},
		{"line strip", 3, []uint16{0, 1, 2}, model.PrimitiveLines, []uint32{0, 1, 1, 2}},
		{"triangles", 4, []uint16{0, 1, 2}, model.PrimitiveTriangles, []uint32{0, 1, 2}},
		{"triangle strip", 5, []uint16{0, 1, 2, 3, 4}, model.PrimitiveTriangles, []uint32{0, 1, 2, 2, 1, 3, 2, 3, 4}},
		{"triangle fan", 6, []uint16{0, 1, 2, 3, 4}, model.PrimitiveTriangles, []uint32{0, 1, 2, 0, 2, 3, 0, 3, 4}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newBuilder()
			p := b.data(f32s(0, 0, 0, 1, 0, 0, 0, 1, 0, 1, 1, 0, 2, 1, 0), ctFloat, "VEC3", 5, nil)
			i := b.data(u16s(tt.indices...), ctUshort, "SCALAR", len(tt.indices), nil)
			b.scene(obj{"attributes": obj{"POSITION": p}, "indices": i, "mode": tt.mode})
			m := mustDecode(t, b.json())
			mesh := m.Meshes[0]
			if mesh.Primitive != tt.primitive || !slices.Equal(mesh.Indices, tt.want) {
				t.Errorf("primitive %v indices %v, want %v %v", mesh.Primitive, mesh.Indices, tt.primitive, tt.want)
			}
		})
	}
}

func TestVertexAttributes(t *testing.T) {
	b := newBuilder()
	p := b.triangle()
	n := b.data(f32s(0, 0, 1, 0, 0, 1, 0, 0, 1), ctFloat, "VEC3", 3, nil)
	tan := b.data(f32s(1, 0, 0, 1, 1, 0, 0, 1, 1, 0, 0, -1), ctFloat, "VEC4", 3, nil)
	uv0 := b.data(f32s(0, 0, 1, 0, 0, 1), ctFloat, "VEC2", 3, nil)
	uv1 := b.data(f32s(0.5, 0.5, 0.25, 0.25, 0.75, 0.75), ctFloat, "VEC2", 3, nil)
	rgb := b.data(f32s(1, 0, 0, 0, 1, 0, 0, 0, 1), ctFloat, "VEC3", 3, nil)
	j := b.data([]byte{0, 1, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0}, ctUbyte, "VEC4", 3, nil)
	w := b.data(f32s(1, 0, 0, 0, 0.5, 0.5, 0, 0, 0, 1, 0, 0), ctFloat, "VEC4", 3, nil)
	b.scene(obj{"attributes": obj{
		"POSITION": p, "NORMAL": n, "TANGENT": tan, "TEXCOORD_0": uv0, "TEXCOORD_1": uv1, "COLOR_0": rgb,
		"JOINTS_0": j, "WEIGHTS_0": w, "_CUSTOM": 99,
	}})
	m := mustDecode(t, b.json())
	mesh := m.Meshes[0]
	if len(mesh.Normals) != 9 || len(mesh.Tangents) != 12 || len(mesh.TexCoords) != 6 || len(mesh.TexCoords1) != 6 {
		t.Errorf("attributes: %d %d %d %d", len(mesh.Normals), len(mesh.Tangents), len(mesh.TexCoords), len(mesh.TexCoords1))
	}
	if want := []byte{255, 0, 0, 255, 0, 255, 0, 255, 0, 0, 255, 255}; !bytes.Equal(mesh.Colors, want) {
		t.Errorf("colors %v, want %v", mesh.Colors, want)
	}
	if want := []uint16{0, 1, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0}; !slices.Equal(mesh.Joints, want) {
		t.Errorf("joints %v", mesh.Joints)
	}
	if want := []float32{1, 0, 0, 0, 0.5, 0.5, 0, 0, 0, 1, 0, 0}; !slices.Equal(mesh.Weights, want) {
		t.Errorf("weights %v", mesh.Weights)
	}
}

func TestQuantization(t *testing.T) {
	build := func(declare bool) []byte {
		b := newBuilder()
		if declare {
			b.doc["extensionsUsed"] = list{"KHR_mesh_quantization"}
			b.doc["extensionsRequired"] = list{"KHR_mesh_quantization"}
		}
		pos := b.data(u16s(0, 0, 0, 32767, 0, 0, 0, 16384, 0), ctShort, "VEC3", 3, obj{"normalized": true})
		b.scene(obj{"attributes": obj{"POSITION": pos}})
		return b.json()
	}
	m := mustDecode(t, build(true))
	if got, want := m.Meshes[0].Positions, []float32{0, 0, 0, 1, 0, 0, 0, 16384.0 / 32767, 0}; !approx(got, want, 1e-6) {
		t.Errorf("positions %v, want %v", got, want)
	}
	_, err := decodeJSON(t, build(false))
	failure(t, err, "loader.checkAttribute.GLTFInvalid", map[string]any{"path": "/meshes/0/primitives/0/attributes/POSITION", "reason": "bad componentType"})
}

func TestMorphTargets(t *testing.T) {
	b := newBuilder()
	p := b.triangle()
	d := b.data(f32s(0, 0, 1, 0, 0, 1, 0, 0, 1), ctFloat, "VEC3", 3, nil)
	dn := b.data(f32s(1, 0, 0, 1, 0, 0, 1, 0, 0), ctFloat, "VEC3", 3, nil)
	b.scene(obj{"attributes": obj{"POSITION": p}, "targets": list{obj{"POSITION": d}, obj{"POSITION": d, "NORMAL": dn}}})
	b.doc["meshes"].(list)[0].(obj)["weights"] = list{0.5, 0}
	b.doc["meshes"].(list)[0].(obj)["extras"] = obj{"targetNames": list{"up", "up and out"}}
	b.doc["nodes"].(list)[0].(obj)["weights"] = list{0, 1}
	m := mustDecode(t, b.json())
	mesh := m.Meshes[0]
	if len(mesh.MorphTargets) != 2 || mesh.MorphTargets[1].Name != "up and out" || len(mesh.MorphTargets[1].Normals) != 9 || mesh.MorphTargets[0].Normals != nil {
		t.Errorf("targets %+v", mesh.MorphTargets)
	}
	if !slices.Equal(mesh.MorphWeights, []float32{0.5, 0}) || !slices.Equal(m.Nodes[0].MorphWeights, []float32{0, 1}) {
		t.Errorf("weights %v, node weights %v", mesh.MorphWeights, m.Nodes[0].MorphWeights)
	}
}

func TestSkinAndAnimation(t *testing.T) {
	b := newBuilder()
	p := b.triangle()
	j := b.data([]byte{0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0}, ctUbyte, "VEC4", 3, nil)
	w := b.data(f32s(1, 0, 0, 0, 0.5, 0.5, 0, 0, 0, 1, 0, 0), ctFloat, "VEC4", 3, nil)
	var ibm []float32
	for k := range 2 {
		ibm = append(ibm, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, float32(-k), 0, 0, 1)
	}
	ibmAcc := b.data(f32s(ibm...), ctFloat, "MAT4", 2, nil)
	times := b.data(f32s(0, 1, 2), ctFloat, "SCALAR", 3, nil)
	rot := b.data(f32s(0, 0, 0, 1, 0, 0, 0.7071068, 0.7071068, 0, 0, 1, 0), ctFloat, "VEC4", 3, nil)
	trans := b.data(f32s(0, 0, 0, 1, 0, 0, 1, 1, 0, 2, 2, 0, 3, 3, 0, 4, 4, 0, 5, 5, 0, 6, 6, 0, 7, 7, 0), ctFloat, "VEC3", 9, nil)
	b.doc["meshes"] = list{obj{"primitives": list{obj{"attributes": obj{"POSITION": p, "JOINTS_0": j, "WEIGHTS_0": w}}}}}
	b.doc["nodes"] = list{
		obj{"mesh": 0, "skin": 0},
		obj{"children": list{2}, "translation": list{1, 2, 3}},
		obj{"rotation": list{0, 0, 0.7071068, 0.7071068}, "scale": list{2, 2, 2}},
	}
	b.doc["skins"] = list{obj{"joints": list{1, 2}, "inverseBindMatrices": ibmAcc, "skeleton": 1}}
	b.doc["scenes"] = list{obj{"nodes": list{0, 1}}}
	b.doc["animations"] = list{obj{
		"name": "wave",
		"samplers": list{
			obj{"input": times, "output": rot, "interpolation": "STEP"},
			obj{"input": times, "output": trans, "interpolation": "CUBICSPLINE"},
		},
		"channels": list{
			obj{"sampler": 0, "target": obj{"node": 2, "path": "rotation"}},
			obj{"sampler": 1, "target": obj{"node": 1, "path": "translation"}},
			obj{"sampler": 0, "target": obj{"node": 1, "path": "rotation"}},
		},
	}}
	m := mustDecode(t, b.json())

	skin := m.Skins[0]
	if len(skin.Joints) != 2 || skin.Joints[0] != m.Nodes[1] || skin.Skeleton != m.Nodes[1] || m.Nodes[0].Skin != skin {
		t.Errorf("skin %+v", skin)
	}
	if skin.InverseBindMatrices[1][12] != -1 {
		t.Errorf("inverse bind matrix %v", skin.InverseBindMatrices[1])
	}
	child := m.Nodes[1].Children[0]
	if child != m.Nodes[1].Children[0] || child.TRS == nil || child.TRS.Scale.X != 2 || math.Abs(child.Transform[0]) > 1e-6 || math.Abs(child.Transform[1]-2) > 1e-6 {
		t.Errorf("child %+v %v", child.TRS, child.Transform)
	}
	anim := m.Animations[0]
	if anim.Name != "wave" || len(anim.Channels) != 3 || anim.Duration() != 2 {
		t.Fatalf("animation %+v", anim)
	}
	c0, c1, c2 := anim.Channels[0], anim.Channels[1], anim.Channels[2]
	if c0.Node != child || c0.Path != model.PathRotation || c0.Interpolation != model.InterpolationStep || c0.Width() != 4 {
		t.Errorf("channel 0 %+v", c0)
	}
	if c1.Node != m.Nodes[1] || c1.Path != model.PathTranslation || c1.Interpolation != model.InterpolationCubicSpline || c1.Width() != 3 || len(c1.Values) != 27 {
		t.Errorf("channel 1 %+v", c1)
	}
	if &c0.Times[0] != &c2.Times[0] || &c0.Values[0] != &c2.Values[0] || &c0.Times[0] == &c1.Times[0] {
		t.Errorf("channels of one sampler share their data, and those of two samplers do not")
	}
}

func TestScenes(t *testing.T) {
	b := newBuilder()
	b.doc["nodes"] = list{obj{"name": "a", "children": list{1}}, obj{"name": "b"}, obj{"name": "c"}}
	tests := []struct {
		name   string
		scenes list
		scene  any
		want   []string
	}{
		{"no scenes: the nodes that are not children", nil, nil, []string{"a", "c"}},
		{"the first scene", list{obj{"nodes": list{2}}, obj{"nodes": list{0}}}, nil, []string{"c"}},
		{"the default scene", list{obj{"nodes": list{2}}, obj{"name": "second", "nodes": list{0, 1}}}, 1, []string{"a", "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b.doc["scenes"], b.doc["scene"] = tt.scenes, tt.scene
			if tt.scenes == nil {
				delete(b.doc, "scenes")
			}
			if tt.scene == nil {
				delete(b.doc, "scene")
			}
			m := mustDecode(t, b.json())
			var got []string
			for _, n := range m.Nodes {
				got = append(got, n.Name)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("top nodes %v, want %v", got, tt.want)
			}
			if len(m.Scenes) != len(tt.scenes) {
				t.Errorf("scenes %d", len(m.Scenes))
			}
		})
	}
}

func TestMaterials(t *testing.T) {
	b := newBuilder()
	p := b.triangle()
	uv := b.data(f32s(0, 0, 1, 0, 0, 1), ctFloat, "VEC2", 3, nil)
	b.doc["extensionsUsed"] = list{"KHR_texture_transform", "KHR_materials_emissive_strength", "KHR_materials_unlit"}
	b.doc["extensionsRequired"] = list{"KHR_texture_transform"}
	png := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
	b.doc["images"] = list{obj{"uri": png, "name": "pixel"}}
	b.doc["samplers"] = list{obj{"magFilter": 9728, "minFilter": 9987, "wrapS": 33071, "wrapT": 33648}}
	b.doc["textures"] = list{obj{"source": 0, "sampler": 0, "name": "tex"}}
	b.doc["materials"] = list{
		obj{
			"name": "full",
			"pbrMetallicRoughness": obj{
				"baseColorFactor": list{0.5, 0.25, 1, 0.5}, "metallicFactor": 0.25, "roughnessFactor": 0.75,
				"baseColorTexture": obj{"index": 0, "texCoord": 1, "extensions": obj{"KHR_texture_transform": obj{
					"offset": list{0.5, 0.25}, "rotation": 1.5707964, "scale": list{2, 3}, "texCoord": 0,
				}}},
				"metallicRoughnessTexture": obj{"index": 0},
			},
			"normalTexture":    obj{"index": 0, "scale": 0.5},
			"occlusionTexture": obj{"index": 0, "strength": 0.25},
			"emissiveTexture":  obj{"index": 0},
			"emissiveFactor":   list{1, 0.5, 0},
			"alphaMode":        "MASK", "alphaCutoff": 0.25, "doubleSided": true,
			"extensions": obj{"KHR_materials_emissive_strength": obj{"emissiveStrength": 4}},
		},
		obj{"name": "default"},
		obj{"name": "unlit", "alphaMode": "BLEND", "extensions": obj{"KHR_materials_unlit": obj{}}},
	}
	b.scene(obj{"attributes": obj{"POSITION": p, "TEXCOORD_0": uv}, "material": 0}, obj{"attributes": obj{"POSITION": p}, "material": 1}, obj{"attributes": obj{"POSITION": p}, "material": 2})
	m := mustDecode(t, b.json())

	full := m.Materials[0]
	if !full.PBR || full.Name != "full" || full.BaseColor != [4]float32{0.5, 0.25, 1, 0.5} || full.Metallic != 0.25 || full.Roughness != 0.75 {
		t.Errorf("material %+v", full)
	}
	if full.Emissive != [3]float32{4, 2, 0} || full.AlphaMode != model.AlphaMask || full.AlphaCutoff != 0.25 || !full.DoubleSided || full.NormalScale != 0.5 || full.OcclusionStrength != 0.25 {
		t.Errorf("material %+v", full)
	}
	tex := full.BaseColorTexture.Texture
	if tex != m.Textures[0] || tex.Name != "tex" || tex.Image != m.Images[0] || full.BaseColorImage != tex.Image || tex.Image.Type != "image/png" || tex.Image.Name != "pixel" || tex.Image.URI != "" {
		t.Errorf("texture %+v", tex)
	}
	if want := (model.Sampler{Mag: model.FilterNearest, Min: model.FilterLinear, Mipmap: model.FilterLinear, WrapU: model.WrapClampToEdge, WrapV: model.WrapMirroredRepeat}); tex.Sampler != want {
		t.Errorf("sampler %+v", tex.Sampler)
	}
	ref := full.BaseColorTexture
	if ref.TexCoord != 0 || ref.Transform == nil || ref.Transform.Scale != [2]float32{2, 3} || ref.Transform.Offset != [2]float32{0.5, 0.25} {
		t.Errorf("reference %+v %+v", ref, ref.Transform)
	}
	// a quarter turn of (1, 0) after the scale (2, 0): (0, -2), then the offset
	if u, v := ref.Transform.Apply(1, 0); math.Abs(float64(u-0.5)) > 1e-6 || math.Abs(float64(v-(0.25-2))) > 1e-6 {
		t.Errorf("transform of (1, 0) is (%v, %v)", u, v)
	}
	if full.MetallicRoughnessTexture == nil || full.NormalTexture == nil || full.OcclusionTexture == nil || full.EmissiveTexture == nil {
		t.Errorf("textures %+v", full)
	}

	def := m.Materials[1]
	if !def.PBR || def.BaseColor != [4]float32{1, 1, 1, 1} || def.Metallic != 1 || def.Roughness != 1 || def.AlphaMode != model.AlphaOpaque || def.AlphaCutoff != 0.5 || def.Color() != def.BaseColor {
		t.Errorf("default material %+v", def)
	}
	if u := m.Materials[2]; !u.Unlit || u.AlphaMode != model.AlphaBlend {
		t.Errorf("unlit material %+v", u)
	}
	if m.Meshes[0].Material != full || m.Meshes[1].Material != def {
		t.Errorf("the meshes use the materials")
	}
}

func TestLightsAndCameras(t *testing.T) {
	b := oneTriangle()
	b.doc["extensionsUsed"] = list{"KHR_lights_punctual"}
	b.doc["extensionsRequired"] = list{"KHR_lights_punctual"}
	b.doc["extensions"] = obj{"KHR_lights_punctual": obj{"lights": list{
		obj{"type": "directional", "name": "sun", "intensity": 3},
		obj{"type": "point", "color": list{1, 0.5, 0}, "range": 10},
		obj{"type": "spot", "spot": obj{"innerConeAngle": 0.25, "outerConeAngle": 0.5}},
	}}}
	b.doc["cameras"] = list{
		obj{"type": "perspective", "name": "p", "perspective": obj{"yfov": 0.5, "znear": 0.1, "aspectRatio": 1.5}},
		obj{"type": "orthographic", "orthographic": obj{"xmag": 2, "ymag": 1, "znear": 0.5, "zfar": 20}},
	}
	b.doc["nodes"] = list{
		obj{"mesh": 0, "children": list{1, 2, 3, 4}},
		obj{"extensions": obj{"KHR_lights_punctual": obj{"light": 0}}},
		obj{"extensions": obj{"KHR_lights_punctual": obj{"light": 1}}},
		obj{"extensions": obj{"KHR_lights_punctual": obj{"light": 2}}, "camera": 0},
		obj{"camera": 1},
	}
	m := mustDecode(t, b.json())
	if len(m.Lights) != 3 || len(m.Cameras) != 2 {
		t.Fatalf("lights %d cameras %d", len(m.Lights), len(m.Cameras))
	}
	kids := m.Nodes[0].Children
	if l := kids[0].Light; l.Type != model.LightDirectional || l.Name != "sun" || l.Intensity != 3 || l.Color != [3]float32{1, 1, 1} {
		t.Errorf("directional light %+v", l)
	}
	if l := kids[1].Light; l.Type != model.LightPoint || l.Color != [3]float32{1, 0.5, 0} || l.Range != 10 || l.Intensity != 1 {
		t.Errorf("point light %+v", l)
	}
	if l := kids[2].Light; l.Type != model.LightSpot || l.InnerConeAngle != 0.25 || l.OuterConeAngle != 0.5 {
		t.Errorf("spot light %+v", l)
	}
	if c := kids[2].Camera; c.Orthographic || c.YFov != 0.5 || c.ZNear != 0.1 || c.ZFar != 0 || c.AspectRatio != 1.5 || c.Name != "p" {
		t.Errorf("perspective camera %+v", c)
	}
	if c := kids[3].Camera; !c.Orthographic || c.XMag != 2 || c.YMag != 1 || c.ZNear != 0.5 || c.ZFar != 20 {
		t.Errorf("orthographic camera %+v", c)
	}
	if m.Lights[2].Type.String() != "spot" {
		t.Errorf("light type name")
	}
}

func TestExtensions(t *testing.T) {
	t.Run("an optional extension that is not read is logged once", func(t *testing.T) {
		b := oneTriangle()
		b.doc["extensionsUsed"] = list{"KHR_materials_clearcoat", "KHR_materials_clearcoat", "EXT_mesh_gpu_instancing"}
		rec := &records{}
		_, err := gltf.Decode(context.Background(), bytes.NewReader(b.json()), gltf.Options{Logger: slog.New(rec)})
		if err != nil {
			t.Fatal(err)
		}
		if len(rec.msgs) != 2 {
			t.Errorf("messages %v, want one for each extension", rec.msgs)
		}
	})
	for _, name := range []string{"KHR_draco_mesh_compression", "KHR_texture_basisu", "EXT_texture_webp", "VENDOR_something"} {
		t.Run("required "+name, func(t *testing.T) {
			b := oneTriangle()
			b.doc["extensionsUsed"] = list{"KHR_lights_punctual", name}
			b.doc["extensionsRequired"] = list{"KHR_lights_punctual", name}
			_, err := decodeJSON(t, b.json())
			failure(t, err, "loader.build.GLTFExtensionUnsupported", map[string]any{"extension": name, "path": "/extensionsRequired/1"})
		})
	}
}

func TestGLB(t *testing.T) {
	b := newBuilder()
	p := b.triangle()
	i := b.data(u16s(0, 1, 2), ctUshort, "SCALAR", 3, nil)
	b.scene(obj{"attributes": obj{"POSITION": p}, "indices": i})
	m, err := gltf.Decode(context.Background(), bytes.NewReader(b.glb()))
	if err != nil {
		t.Fatal(err)
	}
	if m.TriangleCount() != 1 || len(m.Meshes[0].Positions) != 9 {
		t.Errorf("glb: %d triangles", m.TriangleCount())
	}
}

func TestLoadFromFS(t *testing.T) {
	b := newBuilder()
	p := b.triangle()
	b.scene(obj{"attributes": obj{"POSITION": p}, "material": 0})
	b.doc["images"] = list{obj{"uri": "tex%20ture.png"}}
	b.doc["textures"] = list{obj{"source": 0}}
	b.doc["materials"] = list{obj{"pbrMetallicRoughness": obj{"baseColorTexture": obj{"index": 0}}}}
	doc := b.doc
	doc["buffers"] = list{obj{"byteLength": b.bin.Len(), "uri": "mesh.bin"}}
	doc["bufferViews"] = b.views
	doc["accessors"] = b.accessors
	js := marshal(t, doc)
	fsys := fstest.MapFS{
		"models/tri.gltf":      {Data: js},
		"models/mesh.bin":      {Data: b.bin.Bytes()},
		"models/tex ture.png":  {Data: []byte("\x89PNG\r\n\x1a\nxxxx")},
		"models/tri-bad.gltf":  {Data: bytes.ReplaceAll(js, []byte("mesh.bin"), []byte("../../mesh.bin"))},
		"models/tri-gone.gltf": {Data: bytes.ReplaceAll(js, []byte("mesh.bin"), []byte("gone.bin"))},
		"models/tri-abs.gltf":  {Data: bytes.ReplaceAll(js, []byte("mesh.bin"), []byte("/mesh.bin"))},
		"mesh.bin":             {Data: b.bin.Bytes()},
	}

	m, err := gltf.Load(context.Background(), fsys, "models/tri.gltf", gltf.Options{Logger: discard()})
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "tri" || m.TriangleCount() != 1 {
		t.Errorf("model %q with %d triangles", m.Name, m.TriangleCount())
	}
	if img := m.Images[0]; img.URI != "tex%20ture.png" || img.Type != "image/png" || len(img.Data) != 12 {
		t.Errorf("image %+v", img)
	}

	tests := []struct {
		name  string
		file  string
		route string
		meta  map[string]any
	}{
		{"missing file", "models/tri-gone.gltf", "loader.resource.ModelReadFailed", map[string]any{"path": "/buffers/0/uri", "resource": "models/gone.bin", "file": "models/tri-gone.gltf"}},
		{"outside the directory", "models/tri-bad.gltf", "loader.resource.GLTFInvalid", map[string]any{"path": "/buffers/0/uri", "reason": "bad URI"}},
		{"absolute", "models/tri-abs.gltf", "loader.resource.GLTFInvalid", map[string]any{"path": "/buffers/0/uri", "reason": "bad URI"}},
		{"the file is missing", "models/none.gltf", "Load.ModelReadFailed", map[string]any{"file": "models/none.gltf"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := gltf.Load(context.Background(), fsys, tt.file)
			failure(t, err, tt.route, tt.meta)
		})
	}

	// without a file system a path cannot be read
	_, err = gltf.Decode(context.Background(), bytes.NewReader(js))
	failure(t, err, "loader.resource.ModelReadFailed", map[string]any{"path": "/images/0/uri"})
}

func TestDecodeCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := gltf.Decode(ctx, bytes.NewReader(oneTriangle().json()))
	failure(t, err, "ReadAll.ModelReadFailed", nil)
}

func TestSharedMeshIsBuiltOnce(t *testing.T) {
	b := oneTriangle()
	b.doc["nodes"] = list{obj{"mesh": 0, "children": list{1}}, obj{"mesh": 0, "translation": list{1, 0, 0}}}
	m := mustDecode(t, b.json())
	if len(m.Meshes) != 1 || m.Nodes[0].Meshes[0] != m.Nodes[0].Children[0].Meshes[0] || m.TriangleCount() != 2 {
		t.Errorf("meshes %d, triangles %d", len(m.Meshes), m.TriangleCount())
	}
}

func marshal(t testing.TB, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestEmptyPrimitiveIsLeftOut(t *testing.T) {
	b := newBuilder()
	p := b.triangle()
	i := b.data(u16s(0, 1), ctUshort, "SCALAR", 2, nil)
	b.scene(obj{"attributes": obj{"POSITION": p}, "indices": i, "mode": 5}, obj{"attributes": obj{"POSITION": p}})
	m := mustDecode(t, b.json())
	if len(m.Meshes) != 1 {
		t.Errorf("meshes %d", len(m.Meshes))
	}
}
