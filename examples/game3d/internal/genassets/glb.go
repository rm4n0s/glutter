package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"math"
)

// glb writes a glTF 2.0 binary file (.glb): the JSON of the document and one buffer. It knows what the generators of this
// program need (meshes, a skin, animations and materials) and nothing else.
type glb struct {
	bin         bytes.Buffer
	BufferViews []map[string]any
	Accessors   []map[string]any
	Meshes      []map[string]any
	Materials   []map[string]any
	Nodes       []map[string]any
	Skins       []map[string]any
	Animations  []map[string]any
}

const (
	componentFloat  = 5126
	componentUShort = 5123
	targetArray     = 34962
	targetElement   = 34963
)

// align pads the buffer to a multiple of four bytes.
func (g *glb) align() {
	for g.bin.Len()%4 != 0 {
		g.bin.WriteByte(0)
	}
}

// view appends the data as a buffer view and returns its index.
func (g *glb) view(data []byte, target int) int {
	g.align()
	v := map[string]any{"buffer": 0, "byteOffset": g.bin.Len(), "byteLength": len(data)}
	if target != 0 {
		v["target"] = target
	}
	g.bin.Write(data)
	g.BufferViews = append(g.BufferViews, v)
	return len(g.BufferViews) - 1
}

func (g *glb) accessor(view, component, count int, kind string, extra map[string]any) int {
	a := map[string]any{"bufferView": view, "componentType": component, "count": count, "type": kind}
	for k, v := range extra {
		a[k] = v
	}
	g.Accessors = append(g.Accessors, a)
	return len(g.Accessors) - 1
}

func floatBytes(v []float32) []byte {
	b := make([]byte, 4*len(v))
	for i, f := range v {
		binary.LittleEndian.PutUint32(b[4*i:], math.Float32bits(f))
	}
	return b
}

func ushortBytes(v []uint16) []byte {
	b := make([]byte, 2*len(v))
	for i, u := range v {
		binary.LittleEndian.PutUint16(b[2*i:], u)
	}
	return b
}

// floats adds a float accessor of the given type ("SCALAR", "VEC3", ...) and returns it.
func (g *glb) floats(v []float32, kind string, target int, withBounds bool) int {
	width := map[string]int{"SCALAR": 1, "VEC2": 2, "VEC3": 3, "VEC4": 4, "MAT4": 16}[kind]
	extra := map[string]any{}
	if withBounds {
		lo, hi := make([]float64, width), make([]float64, width)
		for i := range lo {
			lo[i], hi[i] = math.Inf(1), math.Inf(-1)
		}
		for i, f := range v {
			c := i % width
			lo[c], hi[c] = math.Min(lo[c], float64(f)), math.Max(hi[c], float64(f))
		}
		extra["min"], extra["max"] = lo, hi
	}
	return g.accessor(g.view(floatBytes(v), target), componentFloat, len(v)/width, kind, extra)
}

func (g *glb) ushorts(v []uint16, kind string, target int) int {
	width := map[string]int{"SCALAR": 1, "VEC4": 4}[kind]
	return g.accessor(g.view(ushortBytes(v), target), componentUShort, len(v)/width, kind, nil)
}

// bytes returns the file.
func (g *glb) bytes(scene []int) []byte {
	g.align()
	doc := map[string]any{
		"asset":       map[string]any{"version": "2.0", "generator": "examples/game3d/internal/genassets"},
		"scene":       0,
		"scenes":      []any{map[string]any{"nodes": scene}},
		"nodes":       g.Nodes,
		"buffers":     []any{map[string]any{"byteLength": g.bin.Len()}},
		"bufferViews": g.BufferViews,
		"accessors":   g.Accessors,
	}
	for name, list := range map[string][]map[string]any{"meshes": g.Meshes, "materials": g.Materials, "skins": g.Skins, "animations": g.Animations} {
		if len(list) > 0 {
			doc[name] = list
		}
	}
	js, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	for len(js)%4 != 0 {
		js = append(js, ' ')
	}
	total := 12 + 8 + len(js) + 8 + g.bin.Len()
	out := make([]byte, 0, total)
	out = append(out, "glTF"...)
	out = binary.LittleEndian.AppendUint32(out, 2)
	out = binary.LittleEndian.AppendUint32(out, uint32(total))
	out = binary.LittleEndian.AppendUint32(out, uint32(len(js)))
	out = append(out, "JSON"...)
	out = append(out, js...)
	out = binary.LittleEndian.AppendUint32(out, uint32(g.bin.Len()))
	out = append(out, "BIN\x00"...)
	out = append(out, g.bin.Bytes()...)
	return out
}

// material adds a metallic-roughness material and returns its index.
func (g *glb) material(name string, color [4]float64, metallic, roughness float64, emissive [3]float64) int {
	m := map[string]any{
		"name": name,
		"pbrMetallicRoughness": map[string]any{
			"baseColorFactor": color, "metallicFactor": metallic, "roughnessFactor": roughness,
		},
	}
	if emissive != [3]float64{} {
		m["emissiveFactor"] = emissive
	}
	g.Materials = append(g.Materials, m)
	return len(g.Materials) - 1
}

// geometry is a list of boxes' triangles: positions, normals, joints and weights for each vertex and the indices.
type geometry struct {
	pos, nrm []float32
	joints   []uint16
	idx      []uint16
}

// box adds an axis-aligned box of half sizes (hx, hy, hz) around (cx, cy, cz), all of whose vertices belong to one joint.
func (b *geometry) box(cx, cy, cz, hx, hy, hz float64, joint uint16, skinned bool) {
	type face struct {
		n    [3]float64
		u, v [3]float64
	}
	faces := []face{
		{[3]float64{1, 0, 0}, [3]float64{0, 0, -1}, [3]float64{0, 1, 0}},
		{[3]float64{-1, 0, 0}, [3]float64{0, 0, 1}, [3]float64{0, 1, 0}},
		{[3]float64{0, 1, 0}, [3]float64{1, 0, 0}, [3]float64{0, 0, -1}},
		{[3]float64{0, -1, 0}, [3]float64{1, 0, 0}, [3]float64{0, 0, 1}},
		{[3]float64{0, 0, 1}, [3]float64{1, 0, 0}, [3]float64{0, 1, 0}},
		{[3]float64{0, 0, -1}, [3]float64{-1, 0, 0}, [3]float64{0, 1, 0}},
	}
	half := [3]float64{hx, hy, hz}
	centre := [3]float64{cx, cy, cz}
	for _, f := range faces {
		base := uint16(len(b.pos) / 3)
		for _, c := range [4][2]float64{{-1, -1}, {1, -1}, {1, 1}, {-1, 1}} {
			var p [3]float64
			for i := 0; i < 3; i++ {
				p[i] = centre[i] + half[i]*(f.n[i]+c[0]*f.u[i]+c[1]*f.v[i])
			}
			b.pos = append(b.pos, float32(p[0]), float32(p[1]), float32(p[2]))
			b.nrm = append(b.nrm, float32(f.n[0]), float32(f.n[1]), float32(f.n[2]))
			if skinned {
				b.joints = append(b.joints, joint, 0, 0, 0)
			}
		}
		b.idx = append(b.idx, base, base+1, base+2, base, base+2, base+3)
	}
}

// primitive adds the geometry as a primitive of a mesh with the material, with the attributes that it has.
func (g *glb) primitive(b *geometry, material int) map[string]any {
	attrs := map[string]any{
		"POSITION": g.floats(b.pos, "VEC3", targetArray, true),
		"NORMAL":   g.floats(b.nrm, "VEC3", targetArray, false),
	}
	if len(b.joints) > 0 {
		attrs["JOINTS_0"] = g.ushorts(b.joints, "VEC4", targetArray)
		w := make([]float32, 0, len(b.joints))
		for i := 0; i < len(b.joints); i += 4 {
			w = append(w, 1, 0, 0, 0)
		}
		attrs["WEIGHTS_0"] = g.floats(w, "VEC4", targetArray, false)
	}
	return map[string]any{"attributes": attrs, "indices": g.ushorts(b.idx, "SCALAR", targetElement), "material": material}
}
