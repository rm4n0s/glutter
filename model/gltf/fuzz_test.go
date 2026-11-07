package gltf_test

import (
	"bytes"
	"context"
	"math"
	"os"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/model"
	"github.com/rm4n0s/glutter/model/gltf"
)

// checkModel checks what a model that Decode gave must be, whatever the file was.
func checkModel(t *testing.T, m *model.Model) {
	t.Helper()
	for _, mesh := range m.Meshes {
		n := mesh.VertexCount()
		per := map[model.Primitive]int{model.PrimitiveTriangles: 3, model.PrimitiveLines: 2, model.PrimitivePoints: 1}[mesh.Primitive]
		if len(mesh.Indices) == 0 || len(mesh.Indices)%per != 0 {
			t.Fatalf("%d indices of %v", len(mesh.Indices), mesh.Primitive)
		}
		for _, i := range mesh.Indices {
			if int(i) >= n {
				t.Fatalf("index %d of %d vertices", i, n)
			}
		}
		for _, p := range mesh.Positions {
			if math.IsNaN(float64(p)) || math.IsInf(float64(p), 0) {
				t.Fatalf("a position is %v", p)
			}
		}
		sizes := []struct {
			name      string
			got, want int
		}{
			{"normals", len(mesh.Normals), 3 * n}, {"tangents", len(mesh.Tangents), 4 * n}, {"texture coordinates", len(mesh.TexCoords), 2 * n},
			{"second texture coordinates", len(mesh.TexCoords1), 2 * n}, {"colors", len(mesh.Colors), 4 * n},
			{"joints", len(mesh.Joints), 4 * n}, {"weights", len(mesh.Weights), 4 * n},
		}
		for _, s := range sizes {
			if s.got != 0 && s.got != s.want {
				t.Fatalf("%d numbers of %s for %d vertices", s.got, s.name, n)
			}
		}
		for _, mt := range mesh.MorphTargets {
			if (mt.Positions != nil && len(mt.Positions) != 3*n) || (mt.Normals != nil && len(mt.Normals) != 3*n) || (mt.Tangents != nil && len(mt.Tangents) != 3*n) {
				t.Fatalf("a morph target of the wrong size for %d vertices", n)
			}
		}
		if mesh.MorphWeights != nil && len(mesh.MorphWeights) != len(mesh.MorphTargets) {
			t.Fatalf("%d weights for %d targets", len(mesh.MorphWeights), len(mesh.MorphTargets))
		}
	}
	for _, s := range m.Skins {
		if len(s.InverseBindMatrices) != len(s.Joints) {
			t.Fatalf("%d matrices for %d joints", len(s.InverseBindMatrices), len(s.Joints))
		}
	}
	for _, a := range m.Animations {
		for _, c := range a.Channels {
			for i, tm := range c.Times {
				if tm < 0 || (i > 0 && tm <= c.Times[i-1]) {
					t.Fatalf("times %v", c.Times)
				}
			}
			per := 1
			if c.Interpolation == model.InterpolationCubicSpline {
				per = 3
			}
			if w := c.Width(); w == 0 || len(c.Values) != per*len(c.Times)*w {
				t.Fatalf("%d values for %d times of width %d", len(c.Values), len(c.Times), w)
			}
		}
	}
	// the nodes are a forest: walking from the top ends
	seen := map[*model.Node]bool{}
	var walk func(n *model.Node)
	walk = func(n *model.Node) {
		if seen[n] {
			t.Fatalf("a node is reached twice")
		}
		seen[n] = true
		for _, c := range n.Children {
			walk(c)
		}
	}
	for _, n := range m.Nodes {
		walk(n)
	}
}

func FuzzDecode(f *testing.F) {
	for _, name := range []string{
		"Triangle/glTF-Embedded/Triangle.gltf", "BoxVertexColors/glTF-Binary/BoxVertexColors.glb",
		"AnimatedMorphCube/glTF-Binary/AnimatedMorphCube.glb", "SimpleInstancing/glTF-Binary/SimpleInstancing.glb",
	} {
		data, err := os.ReadFile("testdata/sample-assets/" + name)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(data)
	}
	f.Add(oneTriangle().json())
	f.Add(oneTriangle().glb())
	f.Add([]byte(`{"asset":{"version":"2.0"}}`))
	f.Add([]byte{})
	f.Add([]byte("glTF"))
	f.Fuzz(func(t *testing.T, data []byte) {
		m, err := gltf.Decode(context.Background(), bytes.NewReader(data), gltf.Options{Logger: discard()})
		if err != nil {
			if _, ok := errors.FromError(err); !ok {
				t.Fatalf("an error that is not an *errors.Error: %T %v", err, err)
			}
			return
		}
		checkModel(t, m)
	})
}

// FuzzDecodeStructured fuzzes the parts that a JSON mutation hardly reaches: the bytes of the binary buffer of a file with
// strided, sparse and quantized accessors, skins, morph targets and animations.
func FuzzDecodeStructured(f *testing.F) {
	seed := func() *builder {
		b := newBuilder()
		b.doc["extensionsUsed"] = list{"KHR_mesh_quantization"}
		pos := b.data(u16s(0, 0, 0, 100, 0, 0, 0, 100, 0), ctUshort, "VEC3", 3, obj{"normalized": true})
		idx := b.data(u16s(0, 1, 2), ctUshort, "SCALAR", 3, nil)
		base := b.view(f32s(0, 0, 0, 1, 0, 0, 0, 1, 0), 0)
		sparse := obj{"count": 1, "indices": obj{"bufferView": b.view([]byte{1}, 0), "componentType": ctUbyte}, "values": obj{"bufferView": b.view(f32s(2, 2, 2), 0)}}
		d := b.accessor(base, ctFloat, "VEC3", 3, obj{"sparse": sparse})
		j := b.data([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, ctUbyte, "VEC4", 3, nil)
		w := b.data(f32s(1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0), ctFloat, "VEC4", 3, nil)
		ibm := b.data(f32s(1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1), ctFloat, "MAT4", 1, nil)
		times := b.data(f32s(0, 1), ctFloat, "SCALAR", 2, nil)
		rot := b.data(f32s(0, 0, 0, 1, 0, 0, 1, 0), ctFloat, "VEC4", 2, nil)
		b.scene(obj{"attributes": obj{"POSITION": pos, "JOINTS_0": j, "WEIGHTS_0": w}, "indices": idx, "targets": list{obj{"POSITION": d}}})
		node0(b)["skin"] = 0
		b.doc["skins"] = list{obj{"joints": list{0}, "inverseBindMatrices": ibm}}
		b.doc["animations"] = list{obj{
			"samplers": list{obj{"input": times, "output": rot}},
			"channels": list{obj{"sampler": 0, "target": obj{"node": 0, "path": "rotation"}}},
		}}
		return b
	}()
	f.Add(seed.bin.Bytes())
	f.Fuzz(func(t *testing.T, bin []byte) {
		// the same document over other bytes: the buffer keeps its length, so the views and the accessors stay inside it
		b := *seed
		var data bytes.Buffer
		data.Write(bin)
		for data.Len() < seed.bin.Len() {
			data.WriteByte(0)
		}
		b.bin = *bytes.NewBuffer(data.Bytes()[:seed.bin.Len()])
		m, err := gltf.Decode(context.Background(), bytes.NewReader(b.glb()), gltf.Options{Logger: discard()})
		if err != nil {
			if _, ok := errors.FromError(err); !ok {
				t.Fatalf("an error that is not an *errors.Error: %T %v", err, err)
			}
			return
		}
		checkModel(t, m)
	})
}
