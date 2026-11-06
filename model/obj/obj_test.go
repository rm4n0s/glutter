package obj_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/model"
	"github.com/rm4n0s/glutter/model/obj"
)

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

func load(t *testing.T, name string, o obj.Options) *model.Model {
	t.Helper()
	o.Logger = quiet
	m, err := obj.Load(context.Background(), os.DirFS("testdata"), name, o)
	if err != nil {
		t.Fatalf("Load(%s): %v", name, err)
	}
	return m
}

func decode(t *testing.T, text string, o obj.Options) *model.Model {
	t.Helper()
	o.Logger = quiet
	m, err := obj.Decode(context.Background(), strings.NewReader(text), o)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	return m
}

// checkMesh checks what every mesh has to be.
func checkMesh(t *testing.T, m *model.Mesh) {
	t.Helper()
	n := m.VertexCount()
	per := map[model.Primitive]int{model.PrimitiveTriangles: 3, model.PrimitiveLines: 2, model.PrimitivePoints: 1}[m.Primitive]
	if len(m.Indices)%per != 0 {
		t.Errorf("%d indices of %v", len(m.Indices), m.Primitive)
	}
	for _, i := range m.Indices {
		if int(i) >= n {
			t.Fatalf("index %d of %d vertices", i, n)
		}
	}
	if m.Normals != nil && len(m.Normals) != 3*n {
		t.Errorf("%d normal numbers for %d vertices", len(m.Normals), n)
	}
	if m.TexCoords != nil && len(m.TexCoords) != 2*n {
		t.Errorf("%d texture numbers for %d vertices", len(m.TexCoords), n)
	}
	if m.Colors != nil && len(m.Colors) != 4*n {
		t.Errorf("%d color bytes for %d vertices", len(m.Colors), n)
	}
}

// triangleArea is the area of the triangles of the mesh, and the sum of their normals' z (all up, for a polygon seen from above).
func area(m *model.Mesh) (total float64, down int) {
	for i := 0; i+2 < len(m.Indices); i += 3 {
		a, b, c := m.Position(int(m.Indices[i])), m.Position(int(m.Indices[i+1])), m.Position(int(m.Indices[i+2]))
		n := b.Sub(a).Cross(c.Sub(a))
		total += n.Length() / 2
		if n.Z < 0 {
			down++
		}
	}
	return total, down
}

func TestTheCube(t *testing.T) {
	m := load(t, "cube.obj", obj.Options{})
	if m.Name != "cube" {
		t.Errorf("name %q", m.Name)
	}
	if len(m.Nodes) != 2 || m.Nodes[0].Name != "bottom_and_top" || m.Nodes[1].Name != "sides" || len(m.Nodes[1].Children) != 4 {
		t.Fatalf("nodes: %d top nodes", len(m.Nodes))
	}
	for i, want := range []string{"front", "back", "right", "left"} {
		if got := m.Nodes[1].Children[i].Name; got != want {
			t.Errorf("group %d is %q, want %q", i, got, want)
		}
	}
	if len(m.Meshes) != 5 || m.TriangleCount() != 12 {
		t.Fatalf("%d meshes and %d triangles, want 5 and 12", len(m.Meshes), m.TriangleCount())
	}
	for _, mesh := range m.Meshes {
		checkMesh(t, mesh)
		if mesh.Normals == nil || mesh.TexCoords == nil {
			t.Errorf("mesh %q has normals %v texture coordinates %v", mesh.Name, mesh.Normals != nil, mesh.TexCoords != nil)
		}
	}
	red := m.Nodes[0].Meshes[0]
	if red.VertexCount() != 8 || red.PrimitiveCount() != 4 || red.Material == nil || red.Material.Name != "red" {
		t.Errorf("red mesh: %d vertices, %d triangles, material %v", red.VertexCount(), red.PrimitiveCount(), red.Material)
	}
	// The texture coordinate v is turned: the first texture coordinate (0, 0) is the top left corner.
	if red.TexCoords[0] != 0 || red.TexCoords[1] != 1 {
		t.Errorf("first texture coordinate %v, want (0, 1)", red.TexCoords[:2])
	}
	if b := m.Bounds(); b.Min.X != 0 || b.Max.X != 1 || b.Min.Y != 0 || b.Max.Y != 1 || b.Min.Z != 0 || b.Max.Z != 1 {
		t.Errorf("bounds %v", b)
	}
	if len(m.Materials) != 2 {
		t.Fatalf("%d materials", len(m.Materials))
	}
	r, b := m.Materials[0], m.Materials[1]
	if r.Name != "red" || r.BaseColor != [4]float32{1, 0, 0, 1} || r.Specular != [3]float32{0.5, 0.5, 0.5} || r.Shininess != 64 || r.AlphaMode != model.AlphaOpaque {
		t.Errorf("red: %+v", r)
	}
	if b.Name != "blue" || b.BaseColor != [4]float32{0, 0, 1, 0.5} || b.AlphaMode != model.AlphaBlend {
		t.Errorf("blue: %+v", b)
	}
	if b.BaseColorImage == nil || b.BaseColorImage.Name != "pixel.png" || b.BaseColorImage.Type != "image/png" || len(b.BaseColorImage.Data) < 50 ||
		len(m.Images) != 1 || m.Images[0] != b.BaseColorImage {
		t.Errorf("blue image: %+v, images %d", b.BaseColorImage, len(m.Images))
	}
	for _, g := range m.Nodes[1].Children {
		if len(g.Meshes) != 1 || g.Meshes[0].Material != b {
			t.Errorf("group %s: %d meshes", g.Name, len(g.Meshes))
		}
	}
}

func TestPolygons(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		tris  int
		area  float64
		holds string
	}{
		{"a triangle", "v 0 0 0\nv 1 0 0\nv 0 1 0\nf 1 2 3\n", 1, 0.5, ""},
		{"a convex quad is a fan", "v 0 0 0\nv 1 0 0\nv 1 1 0\nv 0 1 0\nf 1 2 3 4\n", 2, 1, ""},
		{"a convex pentagon", "v 0 0 0\nv 2 0 0\nv 3 1 0\nv 1 3 0\nv -1 1 0\nf 1 2 3 4 5\n", 3, 7, ""},
		{"negative indices", "v 0 0 0\nv 1 0 0\nv 1 1 0\nv 0 1 0\nf -4 -3 -2 -1\n", 2, 1, ""},
		{"negative indices count what has been read at the face", "v 0 0 0\nv 1 0 0\nv 0 1 0\nf -3 -2 -1\nv 5 5 5\nv 6 5 5\nv 5 6 5\nf -3 -2 -1\n", 2, 1, ""},
		{"a concave arrow head", "v 0 0 0\nv 2 1 0\nv 4 0 0\nv 2 4 0\nf 1 2 3 4\n", 2, 6, ""},
		{"a polygon in a vertical plane", "v 0 0 0\nv 2 0 0\nv 2 0 1\nv 1 0 1\nv 1 0 2\nv 0 0 2\nf 1 2 3 4 5 6\n", 4, 3, ""},
		{"corners with texture coordinates and normals", "v 0 0 0\nv 1 0 0\nv 1 1 0\nv 0 1 0\nvt 0 0\nvt 1 0\nvt 1 1\nvt 0 1\nvn 0 0 1\nf 1/1/1 2/2/1 3/3/1 4/4/1\n", 2, 1, ""},
		{"a line that continues", "v 0 0 0\nv 1 0 0\nv 1 1 0\nv 0 1 0\nf 1 2 \\\n 3 \\\n 4\n", 2, 1, ""},
		{"windows line ends and blanks", "v 0 0 0\r\nv 1 0 0\r\nv 0 1 0\r\n\r\n  f   1   2\t3 \r\n", 1, 0.5, ""},
		{"a byte order mark", "\xEF\xBB\xBFv 0 0 0\nv 1 0 0\nv 0 1 0\nf 1 2 3", 1, 0.5, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := decode(t, tt.text, obj.Options{})
			total := 0
			areaSum := 0.0
			for _, mesh := range m.Meshes {
				checkMesh(t, mesh)
				total += mesh.PrimitiveCount()
				a, down := area(mesh)
				areaSum += a
				if down != 0 {
					t.Errorf("%d triangles turned down", down)
				}
			}
			if total != tt.tris || math.Abs(areaSum-tt.area) > 1e-9 {
				t.Errorf("%d triangles of area %v, want %d of %v", total, areaSum, tt.tris, tt.area)
			}
		})
	}
	m := load(t, "concave.obj", obj.Options{})
	if a, down := area(m.Meshes[0]); m.Meshes[0].PrimitiveCount() != 4 || math.Abs(a-3) > 1e-9 || down != 0 {
		t.Errorf("the L: %d triangles, area %v, %d down", m.Meshes[0].PrimitiveCount(), a, down)
	}
}

func TestLinesAndPoints(t *testing.T) {
	m := load(t, "lines-points.obj", obj.Options{})
	got := map[model.Primitive]int{}
	for _, mesh := range m.Meshes {
		checkMesh(t, mesh)
		got[mesh.Primitive] += mesh.PrimitiveCount()
	}
	// l 1 2 3 4 1 is four segments, l -5 -1 is one; p 5 and p 1 3 are three points; f is one triangle.
	if got[model.PrimitiveLines] != 5 || got[model.PrimitivePoints] != 3 || got[model.PrimitiveTriangles] != 1 || len(m.Meshes) != 3 {
		t.Errorf("primitives %v in %d meshes", got, len(m.Meshes))
	}
	if len(m.Nodes) != 1 || len(m.Nodes[0].Meshes) != 3 || m.Nodes[0].Name != "lines-points" {
		t.Errorf("nodes %d", len(m.Nodes))
	}
	if b := m.Bounds(); b.Max.X != 5 || b.Max.Z != 5 {
		t.Errorf("bounds %v", b)
	}
	if m.TriangleCount() != 1 {
		t.Errorf("TriangleCount %d counts only triangles", m.TriangleCount())
	}
}

func TestVertexColors(t *testing.T) {
	m := load(t, "colors.obj", obj.Options{})
	mesh := m.Meshes[0]
	checkMesh(t, mesh)
	if want := []byte{255, 0, 0, 255, 0, 255, 0, 255, 0, 0, 255, 255, 255, 255, 255, 255}; !bytes.Equal(mesh.Colors, want) {
		t.Errorf("colors %v, want %v", mesh.Colors, want)
	}
	// A mesh that was begun before any position had colors keeps the white of its first vertices.
	m = decode(t, "v 0 0 0\nv 1 0 0\nv 0 1 0\nf 1 2 3\nv 0 0 0 0 0 0\nv 1 1 1\nf 1 2 4\nf 4 5 3\n", obj.Options{})
	checkMesh(t, m.Meshes[0])
	if want := []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 0, 0, 0, 255, 255, 255, 255, 255}; !bytes.Equal(m.Meshes[0].Colors[:20], want) || len(m.Meshes[0].Colors) != 4*m.Meshes[0].VertexCount() {
		t.Errorf("colors %v for %d vertices", m.Meshes[0].Colors, m.Meshes[0].VertexCount())
	}
	// The first positions have none, and a later one has: they are white.
	m = decode(t, "v 0 0 0\nv 1 0 0\nv 0 1 0 0.5 0.5 0.5\nf 1 2 3\n", obj.Options{})
	if want := []byte{255, 255, 255, 255, 255, 255, 255, 255, 128, 128, 128, 255}; !bytes.Equal(m.Meshes[0].Colors, want) {
		t.Errorf("colors %v, want %v", m.Meshes[0].Colors, want)
	}
}

func TestFloat64(t *testing.T) {
	const text = "v 1234567890.123456789 0 0\nv 1234567891.5 0 0\nv 1234567890.123456789 1 0\nf 1 2 3\n"
	m := decode(t, text, obj.Options{Float64: true})
	mesh := m.Meshes[0]
	if mesh.Positions != nil || len(mesh.Positions64) != 9 || math.Abs(mesh.Positions64[0]-1234567890.123456789) > 1e-9 {
		t.Errorf("positions %v positions64 %v", mesh.Positions, mesh.Positions64)
	}
	m = decode(t, text, obj.Options{})
	if m.Meshes[0].Positions64 != nil || len(m.Meshes[0].Positions) != 9 {
		t.Errorf("not asked for: positions64 %v", m.Meshes[0].Positions64)
	}
	m = decode(t, "v 0.5 0 0\nv 1 0 0\nv 0 1 0\nf 1 2 3\n", obj.Options{Float64: true})
	if m.Meshes[0].Positions64 != nil || len(m.Meshes[0].Positions) != 9 {
		t.Errorf("a file that float32 holds: positions64 %v", m.Meshes[0].Positions64)
	}
}

func TestMissingFilesGiveTheDefaultMaterialAndALine(t *testing.T) {
	var logged bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logged, nil))
	m, err := obj.Load(context.Background(), os.DirFS("testdata"), "missing-mtl.obj", obj.Options{Logger: logger})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Materials) != 1 || m.Materials[0].Name != "ghost" || m.Materials[0].BaseColor != ([4]float32{}) || m.Materials[0].Color() != model.DefaultColor {
		t.Errorf("materials %+v", m.Materials)
	}
	if m.Meshes[0].Material != m.Materials[0] {
		t.Error("the mesh does not have the material")
	}
	// One line for each library that is not there.
	if lines := strings.Count(logged.String(), "\n"); lines != 2 || !strings.Contains(logged.String(), "nothing.mtl") || !strings.Contains(logged.String(), "other.mtl") {
		t.Errorf("log:\n%s", logged.String())
	}

	// Without a file system there is one line, whatever the number of libraries.
	logged.Reset()
	if _, err := obj.Decode(context.Background(), strings.NewReader("mtllib a.mtl\nmtllib b.mtl\nv 0 0 0\nv 1 0 0\nv 0 1 0\nf 1 2 3\n"), obj.Options{Logger: logger}); err != nil {
		t.Fatal(err)
	}
	if lines := strings.Count(logged.String(), "\n"); lines != 1 {
		t.Errorf("log:\n%s", logged.String())
	}

	// A missing image: the material has none, and there is a line.
	logged.Reset()
	fsys := fstest.MapFS{
		"d/a.obj": {Data: []byte("mtllib m.mtl\nusemtl x\nv 0 0 0\nv 1 0 0\nv 0 1 0\nf 1 2 3\n")},
		"d/m.mtl": {Data: []byte("newmtl x\nmap_Kd -o 0 0 0 gone.png\nKd 0.5 0.5 0.5\n")},
	}
	m, err = obj.Load(context.Background(), fsys, "d/a.obj", obj.Options{Logger: logger})
	if err != nil {
		t.Fatal(err)
	}
	if mat := m.Materials[0]; mat.BaseColorImage != nil || mat.BaseColor != [4]float32{0.5, 0.5, 0.5, 1} || len(m.Images) != 0 {
		t.Errorf("material %+v", mat)
	}
	if lines := strings.Count(logged.String(), "\n"); lines != 1 || !strings.Contains(logged.String(), "gone.png") {
		t.Errorf("log:\n%s", logged.String())
	}
}

func TestMaterialLibraryDetails(t *testing.T) {
	tests := []struct {
		name string
		mtl  string
		want model.Material
	}{
		{"Kd alone is opaque", "newmtl m\nKd 0.1 0.2 0.3\n", model.Material{Name: "m", BaseColor: [4]float32{0.1, 0.2, 0.3, 1}}},
		{"d alone keeps the default color", "newmtl m\nd 0.25\n", model.Material{Name: "m", BaseColor: [4]float32{model.DefaultColor[0], model.DefaultColor[1], model.DefaultColor[2], 0.25}, AlphaMode: model.AlphaBlend}},
		{"d before Kd", "newmtl m\nd 0.5\nKd 1 1 0\n", model.Material{Name: "m", BaseColor: [4]float32{1, 1, 0, 0.5}, AlphaMode: model.AlphaBlend}},
		{"d with the halo option", "newmtl m\nd -halo 0.5\nKd 1 1 0\n", model.Material{Name: "m", BaseColor: [4]float32{1, 1, 0, 0.5}, AlphaMode: model.AlphaBlend}},
		{"black is black", "newmtl m\nKd 0 0 0\n", model.Material{Name: "m", BaseColor: [4]float32{0, 0, 0, 1}}},
		{"specular and shininess", "newmtl m\nKs 1 0.5 0.25\nNs 900\n", model.Material{Name: "m", BaseColor: [4]float32{model.DefaultColor[0], model.DefaultColor[1], model.DefaultColor[2], 1}, Specular: [3]float32{1, 0.5, 0.25}, Shininess: 900}},
		{"a name with blanks, other keys, comments and case", "# c\nNEWMTL a name\nka 1 1 1\nillum 2\nKD 0 1 0\nmap_Bump x.png\n", model.Material{Name: "a name", BaseColor: [4]float32{0, 1, 0, 1}}},
		{"d above 1 is 1", "newmtl m\nd 7\nKd 1 0 0\n", model.Material{Name: "m", BaseColor: [4]float32{1, 0, 0, 1}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := fstest.MapFS{
				"a.obj": {Data: []byte("mtllib m.mtl\nusemtl " + tt.want.Name + "\nv 0 0 0\nv 1 0 0\nv 0 1 0\nf 1 2 3\n")},
				"m.mtl": {Data: []byte(tt.mtl)},
			}
			m, err := obj.Load(context.Background(), fsys, "a.obj", obj.Options{Logger: quiet})
			if err != nil {
				t.Fatal(err)
			}
			if len(m.Materials) != 1 || *m.Materials[0] != tt.want {
				t.Errorf("material %+v, want %+v", m.Materials, tt.want)
			}
		})
	}
	// Library names with blanks, and a library in another directory whose image is relative to it.
	fsys := fstest.MapFS{
		"models/a.obj":                {Data: []byte("mtllib lib/my materials.mtl\nusemtl x\nv 0 0 0\nv 1 0 0\nv 0 1 0\nf 1 2 3\n")},
		"models/lib/my materials.mtl": {Data: []byte("newmtl x\nmap_Kd tex.png\n")},
		"models/lib/tex.png":          {Data: []byte("\x89PNG\r\n\x1a\nxx")},
	}
	m, err := obj.Load(context.Background(), fsys, "models/a.obj", obj.Options{Logger: quiet})
	if err != nil {
		t.Fatal(err)
	}
	if img := m.Materials[0].BaseColorImage; img == nil || img.URI != "models/lib/tex.png" || img.Type != "image/png" {
		t.Errorf("image %+v", img)
	}
	// A path that leaves the file system is not read.
	fsys = fstest.MapFS{
		"a.obj":   {Data: []byte("mtllib ../secret.mtl\nusemtl x\nv 0 0 0\nv 1 0 0\nv 0 1 0\nf 1 2 3\n")},
		"b/c.obj": {Data: []byte("mtllib ../a.mtl\nv 0 0 0\nv 1 0 0\nv 0 1 0\nf 1 2 3\n")},
		"a.mtl":   {Data: []byte("newmtl x\nKd 1 0 0\n")},
	}
	m, err = obj.Load(context.Background(), fsys, "a.obj", obj.Options{Logger: quiet})
	if err != nil || m.Materials[0].BaseColor != ([4]float32{}) {
		t.Errorf("a library outside the file system was read: %v %+v", err, m.Materials)
	}
	m, err = obj.Load(context.Background(), fsys, "b/c.obj", obj.Options{Logger: quiet})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Materials) != 1 || m.Materials[0].BaseColor != [4]float32{1, 0, 0, 1} {
		t.Errorf("a library one directory up: %+v", m.Materials)
	}
}

func TestVerticesAreSharedWithinAMeshOnly(t *testing.T) {
	text := "v 0 0 0\nv 1 0 0\nv 1 1 0\nv 0 1 0\nusemtl a\nf 1 2 3\nf 1 3 4\nusemtl b\nf 1 2 4\ng second\nf 1 2 3\n"
	m := decode(t, text, obj.Options{})
	// Mesh a of the root node: 4 vertices for 2 triangles; b: 3; the group's: 3 (the material of b).
	if len(m.Meshes) != 3 {
		t.Fatalf("%d meshes", len(m.Meshes))
	}
	if m.Meshes[0].VertexCount() != 4 || m.Meshes[1].VertexCount() != 3 || m.Meshes[2].VertexCount() != 3 {
		t.Errorf("vertices %d %d %d", m.Meshes[0].VertexCount(), m.Meshes[1].VertexCount(), m.Meshes[2].VertexCount())
	}
	if len(m.Nodes) != 2 || m.Nodes[0].Name != "" || m.Nodes[1].Name != "second" {
		t.Errorf("nodes %d", len(m.Nodes))
	}
	if m.Nodes[0].Meshes[0].Material.Name != "a" || m.Nodes[0].Meshes[1].Material.Name != "b" || m.Nodes[1].Meshes[0].Material.Name != "b" {
		t.Error("materials of the meshes")
	}
	// The same corner with and without a normal are two vertices, and a mesh where only some have normals keeps none.
	var logged bytes.Buffer
	m, err := obj.Decode(context.Background(), strings.NewReader("v 0 0 0\nv 1 0 0\nv 0 1 0\nvn 0 0 1\nf 1//1 2//1 3\n"), obj.Options{Logger: slog.New(slog.NewTextHandler(&logged, nil))})
	if err != nil {
		t.Fatal(err)
	}
	if m.Meshes[0].Normals != nil || !strings.Contains(logged.String(), "no normal") {
		t.Errorf("normals %v, log %q", m.Meshes[0].Normals, logged.String())
	}
}

func TestErrors(t *testing.T) {
	const run = "Decode->decode->decoder.run->"
	tests := []struct {
		name   string
		text   string
		route  string
		reason string
		line   int
	}{
		{"a position that is not a number", "v 0 0 x\n", run + "decoder.vertex." + obj.TagOBJInvalid, "not a number", 1},
		{"a position that is infinite", "v 0 0 1e999\n", run + "decoder.vertex." + obj.TagOBJInvalid, "not a number", 1},
		{"a position with two numbers", "# c\nv 0 0\n", run + "decoder.vertex." + obj.TagOBJInvalid, "short vertex", 2},
		{"a position beyond float32", "v 0 0 1e100\n", run + "decoder.vertex." + obj.TagOBJInvalid, "out of range", 1},
		{"a texture coordinate that is not a number", "vt a\n", run + "decoder.texCoord." + obj.TagOBJInvalid, "not a number", 1},
		{"an empty texture coordinate", "vt\n", run + "decoder.texCoord." + obj.TagOBJInvalid, "short texture coordinate", 1},
		{"a normal that is not a number", "vn 0 0 q\n", run + "decoder.normal." + obj.TagOBJInvalid, "not a number", 1},
		{"a normal with two numbers", "vn 0 0\n", run + "decoder.normal." + obj.TagOBJInvalid, "short normal", 1},
		{"a face with two corners", "v 0 0 0\nv 1 0 0\nf 1 2\n", run + "decoder.face." + obj.TagOBJInvalid, "short face", 3},
		{"a face of nothing", "f\n", run + "decoder.face." + obj.TagOBJInvalid, "short face", 1},
		{"index zero", "v 0 0 0\nv 1 0 0\nv 0 1 0\nf 0 1 2\n", run + "decoder.face." + obj.TagOBJInvalid, "zero index", 4},
		{"an index beyond the positions", "v 0 0 0\nv 1 0 0\nv 0 1 0\nf 1 2 4\n", run + "decoder.face." + obj.TagOBJInvalid, "index out of range", 4},
		{"a negative index beyond the positions", "v 0 0 0\nv 1 0 0\nv 0 1 0\nf -4 1 2\n", run + "decoder.face." + obj.TagOBJInvalid, "index out of range", 4},
		{"a face before its positions", "f 1 2 3\nv 0 0 0\nv 1 0 0\nv 0 1 0\n", run + "decoder.face." + obj.TagOBJInvalid, "index out of range", 1},
		{"an index that is not a number", "v 0 0 0\nv 1 0 0\nv 0 1 0\nf 1 2 x\n", run + "decoder.face." + obj.TagOBJInvalid, "not an index", 4},
		{"a texture index beyond the data", "v 0 0 0\nv 1 0 0\nv 0 1 0\nf 1/1 2/1 3/1\n", run + "decoder.face." + obj.TagOBJInvalid, "index out of range", 4},
		{"a normal index beyond the data", "v 0 0 0\nv 1 0 0\nv 0 1 0\nf 1//1 2//1 3//1\n", run + "decoder.face." + obj.TagOBJInvalid, "index out of range", 4},
		{"four parts in a corner", "v 0 0 0\nv 1 0 0\nv 0 1 0\nvt 0 0\nvn 0 0 1\nf 1/1/1/1 2 3\n", run + "decoder.face." + obj.TagOBJInvalid, "too many parts", 6},
		{"a line of one point", "v 0 0 0\nl 1\n", run + "decoder.lines." + obj.TagOBJInvalid, "short line", 2},
		{"a line to nowhere", "v 0 0 0\nl 1 2\n", run + "decoder.lines." + obj.TagOBJInvalid, "index out of range", 2},
		{"a point to nowhere", "v 0 0 0\np 2\n", run + "decoder.points." + obj.TagOBJInvalid, "index out of range", 2},
		{"points of nothing", "v 0 0 0\np\n", run + "decoder.points." + obj.TagOBJInvalid, "short points", 2},
		{"nothing at all", "", "Decode->decode->decoder.finish." + obj.TagOBJInvalid, "empty", 0},
		{"only positions", "v 0 0 0\nv 1 0 0\n", "Decode->decode->decoder.finish." + obj.TagOBJInvalid, "empty", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := obj.Decode(context.Background(), strings.NewReader(tt.text), obj.Options{Logger: quiet})
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
			if appErr.Metadata["reason"] != tt.reason || appErr.Metadata["line"] != tt.line {
				t.Errorf("reason %v line %v, want %q and %d; metadata %v", appErr.Metadata["reason"], appErr.Metadata["line"], tt.reason, tt.line, appErr.Metadata)
			}
		})
	}
}

func TestTheErrorOfAFileHasItsName(t *testing.T) {
	fsys := fstest.MapFS{"x/bad.obj": {Data: []byte("v 1 2 z\n")}}
	_, err := obj.Load(context.Background(), fsys, "x/bad.obj", obj.Options{})
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != obj.TagOBJInvalid || appErr.Metadata["file"] != "x/bad.obj" || appErr.Metadata["field"] != "z" {
		t.Errorf("error %v", err)
	}
}

type failingReader struct{ after int }

func (r *failingReader) Read(p []byte) (int, error) {
	if r.after <= 0 {
		return 0, io.ErrUnexpectedEOF
	}
	n := min(len(p), r.after)
	r.after -= n
	return n, nil
}

func TestReadFailures(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	cube, err := os.ReadFile("testdata/cube.obj")
	if err != nil {
		t.Fatal(err)
	}
	fsys := fstest.MapFS{"a.obj": {Data: cube}, "cube.mtl": {Data: []byte("newmtl red\n")}}
	tests := []struct {
		name  string
		run   func() error
		route string
		cause error
		meta  string
	}{
		{"a reader that fails", func() error {
			_, err := obj.Decode(context.Background(), &failingReader{after: 100}, obj.Options{})
			return err
		}, "Decode->ReadAll." + model.TagModelReadFailed, io.ErrUnexpectedEOF, "offset"},
		{"a context that is done", func() error {
			_, err := obj.Decode(cancelled, bytes.NewReader(cube), obj.Options{})
			return err
		}, "Decode->ReadAll." + model.TagModelReadFailed, context.Canceled, "offset"},
		{"a file that is not there", func() error {
			_, err := obj.Load(context.Background(), fstest.MapFS{}, "none.obj", obj.Options{})
			return err
		}, "Load." + model.TagModelReadFailed, os.ErrNotExist, "file"},
		{"a file in a context that is done", func() error {
			_, err := obj.Load(cancelled, fsys, "a.obj", obj.Options{})
			return err
		}, "Load->ReadAll." + model.TagModelReadFailed, context.Canceled, "file"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute(tt.route) || !errors.Is(err, tt.cause) {
				t.Errorf("unexpected failure path: %s (cause %v)", appErr.Route(), tt.cause)
			}
			if _, ok := appErr.Metadata[tt.meta]; !ok {
				t.Errorf("no %q in the metadata %v", tt.meta, appErr.Metadata)
			}
		})
	}
}

// cancelAfter is a context that is done after n looks at it, counted from when the reader has given all its data.
type cancelAfter struct {
	context.Context
	n    atomic.Int64
	read atomic.Bool
}

func (c *cancelAfter) Err() error {
	if !c.read.Load() {
		return nil
	}
	if c.n.Add(-1) < 0 {
		return context.Canceled
	}
	return nil
}

type signalEOF struct {
	r   io.Reader
	ctx *cancelAfter
}

func (s signalEOF) Read(p []byte) (int, error) {
	n, err := s.r.Read(p)
	if err == io.EOF {
		s.ctx.read.Store(true)
	}
	return n, err
}

func TestTheContextIsCheckedWhileParsing(t *testing.T) {
	var text strings.Builder
	for i := 0; i < 30000; i++ {
		fmt.Fprintf(&text, "v %d %d 0\n", i, i%7)
	}
	for i := 3; i < 30000; i++ {
		fmt.Fprintf(&text, "f %d %d %d\n", i-2, i-1, i)
	}
	mtl := "newmtl m\n" + strings.Repeat("# comment\n", 20000) + "Kd 1 0 0\n"
	fsys := fstest.MapFS{"a.mtl": {Data: []byte(mtl)}}
	withLibrary := "mtllib a.mtl\n" + text.String()
	tests := []struct {
		name string
		text string
		opts obj.Options
		want string
	}{
		{"the lines", text.String(), obj.Options{}, "Decode->decode->decoder.run."},
		{"a library", withLibrary, obj.Options{FS: fsys}, "decoder.materialLibraries->decoder.parseMTL."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := false
			for after := 0; after < 40; after++ {
				ctx := &cancelAfter{Context: context.Background()}
				ctx.n.Store(int64(after))
				o := tt.opts
				o.Logger = quiet
				_, err := obj.Decode(ctx, signalEOF{strings.NewReader(tt.text), ctx}, o)
				if err == nil {
					break
				}
				appErr, ok := errors.FromError(err)
				if !ok || !errors.Is(err, context.Canceled) {
					t.Fatalf("after %d looks: %T %v", after, err, err)
				}
				if appErr.HasRoute(tt.want + model.TagModelReadFailed) {
					found = true
				}
			}
			if !found {
				t.Errorf("the loader never stopped at %s", tt.want+model.TagModelReadFailed)
			}
		})
	}
}

func fuzzSeeds(f *testing.F) {
	for _, name := range []string{"cube.obj", "concave.obj", "lines-points.obj", "colors.obj", "missing-mtl.obj", "cube.mtl"} {
		data, err := os.ReadFile("testdata/" + name)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(data, false)
	}
	f.Add([]byte("v 1e400 0 0\nf 1 1 1\n"), true)
	f.Add([]byte("v 0 0 0\nv 1 0 0\nv 1 1 0\nv 0 1 0\nf 1 2 3 4 \\\n"), false)
	f.Add([]byte("f -1 -1 -1\n"), false)
	f.Add([]byte{}, false)
}

func FuzzDecode(f *testing.F) {
	fuzzSeeds(f)
	cube, err := os.ReadFile("testdata/cube.mtl")
	if err != nil {
		f.Fatal(err)
	}
	f.Fuzz(func(t *testing.T, data []byte, wide bool) {
		fsys := fstest.MapFS{"cube.mtl": {Data: cube}, "pixel.png": {Data: []byte("\x89PNG")}, "nothing.mtl": {Data: data}}
		m, err := obj.Decode(context.Background(), bytes.NewReader(data), obj.Options{Float64: wide, Logger: quiet, FS: fsys})
		if err != nil {
			if _, ok := errors.FromError(err); !ok {
				t.Fatalf("an error that is not an *errors.Error: %T %v", err, err)
			}
			return
		}
		if len(m.Meshes) == 0 || len(m.Nodes) == 0 {
			t.Fatal("a model with no mesh and no error")
		}
		for _, mesh := range m.Meshes {
			checkMesh(t, mesh)
			for i := 0; i < mesh.VertexCount(); i++ {
				if p := mesh.Position(i); math.IsNaN(p.X+p.Y+p.Z) || math.IsInf(p.X+p.Y+p.Z, 0) {
					t.Fatalf("vertex %d is %v", i, p)
				}
			}
		}
	})
}

// gridOBJ is the text of an OBJ file of a height field of n by n squares: 2*n*n triangles, with normals and texture coordinates.
func gridOBJ(n int) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "# grid\n")
	for j := 0; j <= n; j++ {
		for i := 0; i <= n; i++ {
			fmt.Fprintf(&b, "v %d.%03d %d.%03d %.6f\n", i, (i*37)%1000, j, (j*91)%1000, math.Sin(float64(i)*0.1)*math.Cos(float64(j)*0.13))
		}
	}
	for j := 0; j <= n; j++ {
		for i := 0; i <= n; i++ {
			fmt.Fprintf(&b, "vt %.6f %.6f\n", float64(i)/float64(n), float64(j)/float64(n))
		}
	}
	for j := 0; j <= n; j++ {
		for i := 0; i <= n; i++ {
			fmt.Fprintf(&b, "vn %.6f %.6f %.6f\n", 0.1, 0.2, 0.97)
		}
	}
	at := func(i, j int) int { return j*(n+1) + i + 1 }
	for j := 0; j < n; j++ {
		for i := 0; i < n; i++ {
			a, bb, c, d := at(i, j), at(i+1, j), at(i+1, j+1), at(i, j+1)
			fmt.Fprintf(&b, "f %d/%d/%d %d/%d/%d %d/%d/%d\n", a, a, a, bb, bb, bb, c, c, c)
			fmt.Fprintf(&b, "f %d/%d/%d %d/%d/%d %d/%d/%d\n", a, a, a, c, c, c, d, d, d)
		}
	}
	return b.Bytes()
}

func TestAHeightFieldKeepsItsGrid(t *testing.T) {
	const n = 30
	m := decode(t, string(gridOBJ(n)), obj.Options{})
	mesh := m.Meshes[0]
	checkMesh(t, mesh)
	if mesh.PrimitiveCount() != 2*n*n || mesh.VertexCount() != (n+1)*(n+1) || mesh.Normals == nil || mesh.TexCoords == nil {
		t.Errorf("%d triangles, %d vertices", mesh.PrimitiveCount(), mesh.VertexCount())
	}
}

func BenchmarkDecode1M(b *testing.B) {
	// 708 by 708 squares make 1,002,528 triangles.
	data := gridOBJ(708)
	for _, tt := range []struct {
		name string
		opts obj.Options
	}{{"float32", obj.Options{}}, {"float64", obj.Options{Float64: true}}} {
		b.Run(tt.name, func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				m, err := obj.Decode(context.Background(), bytes.NewReader(data), tt.opts)
				if err != nil {
					b.Fatal(err)
				}
				if m.TriangleCount() != 1002528 {
					b.Fatal(m.TriangleCount())
				}
			}
		})
	}
}
