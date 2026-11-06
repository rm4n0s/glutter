package model_test

import (
	"bytes"
	"context"
	"io"
	"testing"
	"testing/iotest"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/model"
	"github.com/rm4n0s/glutter/vecmath"
)

func TestMeshBounds(t *testing.T) {
	tests := []struct {
		name     string
		mesh     model.Mesh
		min, max vecmath.Vec3
		empty    bool
	}{
		{"every vertex without indices", model.Mesh{Positions: []float32{0, 1, 2, 3, -4, 5}}, vecmath.Vec3{Y: -4, Z: 2}, vecmath.Vec3{X: 3, Y: 1, Z: 5}, false},
		{"only the vertices that the indices use", model.Mesh{Positions: []float32{0, 0, 0, 1, 1, 1, 9, 9, 9}, Indices: []uint32{0, 1}}, vecmath.Vec3{}, vecmath.Vec3{X: 1, Y: 1, Z: 1}, false},
		{"float64", model.Mesh{Positions64: []float64{5e7, 0, 0, 5e7 + 0.25, 1, 0}}, vecmath.Vec3{X: 5e7}, vecmath.Vec3{X: 5e7 + 0.25, Y: 1}, false},
		{"an index outside the vertices is left out", model.Mesh{Positions: []float32{1, 2, 3}, Indices: []uint32{0, 5}}, vecmath.Vec3{X: 1, Y: 2, Z: 3}, vecmath.Vec3{X: 1, Y: 2, Z: 3}, false},
		{"nothing", model.Mesh{}, vecmath.Vec3{}, vecmath.Vec3{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := tt.mesh.Bounds()
			if b.IsEmpty() != tt.empty || (!tt.empty && (b.Min != tt.min || b.Max != tt.max)) {
				t.Errorf("bounds %v, want %v to %v (empty %v)", b, tt.min, tt.max, tt.empty)
			}
		})
	}
}

func TestMeshCounts(t *testing.T) {
	tests := []struct {
		mesh       model.Mesh
		vertices   int
		primitives int
	}{
		{model.Mesh{Positions: make([]float32, 9), Indices: []uint32{0, 1, 2}}, 3, 1},
		{model.Mesh{Primitive: model.PrimitiveLines, Positions: make([]float32, 12), Indices: []uint32{0, 1, 1, 2, 2, 3}}, 4, 3},
		{model.Mesh{Primitive: model.PrimitivePoints, Positions64: make([]float64, 6), Indices: []uint32{0, 1}}, 2, 2},
	}
	for _, tt := range tests {
		if tt.mesh.VertexCount() != tt.vertices || tt.mesh.PrimitiveCount() != tt.primitives {
			t.Errorf("%v: %d vertices and %d primitives, want %d and %d", tt.mesh.Primitive, tt.mesh.VertexCount(), tt.mesh.PrimitiveCount(), tt.vertices, tt.primitives)
		}
	}
	for p, want := range map[model.Primitive]string{model.PrimitiveTriangles: "triangles", model.PrimitiveLines: "lines", model.PrimitivePoints: "points", 9: "unknown"} {
		if p.String() != want {
			t.Errorf("%d is %q, want %q", p, p.String(), want)
		}
	}
}

func TestModelBoundsFollowTheTransforms(t *testing.T) {
	mesh := &model.Mesh{Positions: []float32{0, 0, 0, 1, 1, 1}}
	root := model.NewNode("root")
	root.Transform = vecmath.NewMat4Translation(vecmath.Vec3{X: 10})
	child := model.NewNode("child")
	child.Transform = vecmath.NewMat4Scale(vecmath.Vec3{X: 2, Y: 2, Z: 2})
	child.Meshes = []*model.Mesh{mesh}
	other := model.NewNode("other")
	other.Meshes = []*model.Mesh{mesh, {}}
	root.Children = []*model.Node{child}
	m := &model.Model{Nodes: []*model.Node{root, other}}
	b := m.Bounds()
	if b.Min != (vecmath.Vec3{}) || b.Max != (vecmath.Vec3{X: 12, Y: 2, Z: 2}) {
		t.Errorf("bounds %v", b)
	}
	if m.TriangleCount() != 0 {
		t.Errorf("TriangleCount %d for meshes with no indices", m.TriangleCount())
	}
	if !(&model.Model{}).Bounds().IsEmpty() {
		t.Error("a model with nothing has bounds")
	}
}

func TestMaterialColor(t *testing.T) {
	var none *model.Material
	if none.Color() != model.DefaultColor || (&model.Material{}).Color() != model.DefaultColor {
		t.Error("no color is the default color")
	}
	if c := (&model.Material{BaseColor: [4]float32{0, 0, 0, 1}}).Color(); c != [4]float32{0, 0, 0, 1} {
		t.Errorf("black is %v", c)
	}
}

func TestDetectImageType(t *testing.T) {
	tests := []struct {
		name string
		file string
		data []byte
		want string
	}{
		{"png by name", "a.PNG", nil, "image/png"},
		{"jpeg by name", "dir\\a.jpeg", nil, "image/jpeg"},
		{"the name wins", "a.png", []byte("GIF89a"), "image/png"},
		{"png by bytes", "a", []byte("\x89PNG\r\n\x1a\nxx"), "image/png"},
		{"jpeg by bytes", "a", []byte{0xFF, 0xD8, 0xFF, 0xE0}, "image/jpeg"},
		{"gif by bytes", "a", []byte("GIF89a"), "image/gif"},
		{"bmp by bytes", "a", []byte("BM...."), "image/bmp"},
		{"webp by bytes", "a", []byte("RIFF\x00\x00\x00\x00WEBPVP8 "), "image/webp"},
		{"gif by name", "a.gif", nil, "image/gif"},
		{"bmp by name", "a.bmp", nil, "image/bmp"},
		{"webp by name", "a.webp", nil, "image/webp"},
		{"unknown", "a.xyz", []byte("hello"), ""},
		{"short riff", "a", []byte("RIFF"), ""},
	}
	for _, tt := range tests {
		if got := model.DetectImageType(tt.file, tt.data); got != tt.want {
			t.Errorf("%s: %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestReadAll(t *testing.T) {
	big := bytes.Repeat([]byte("0123456789abcdef"), 1<<18) // 4 MiB: several reads
	tests := []struct {
		name string
		r    io.Reader
		want []byte
	}{
		{"nothing", bytes.NewReader(nil), nil},
		{"small", bytes.NewReader([]byte("abc")), []byte("abc")},
		{"big, sized by Len", bytes.NewReader(big), big},
		{"big, one byte at a time is slow, so half a megabyte", iotest.HalfReader(bytes.NewReader(big)), big},
		{"big, of unknown size", io.MultiReader(bytes.NewReader(big), bytes.NewReader(big[:5])), append(append([]byte(nil), big...), big[:5]...)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := model.ReadAll(context.Background(), tt.r)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Errorf("read %d bytes, want %d", len(got), len(tt.want))
			}
		})
	}
	_, err := model.ReadAll(context.Background(), iotest.ErrReader(io.ErrClosedPipe))
	appErr, ok := errors.FromError(err)
	if !ok || !appErr.HasRoute("ReadAll."+model.TagModelReadFailed) || !errors.Is(err, io.ErrClosedPipe) {
		t.Errorf("error %v", err)
	}
}
