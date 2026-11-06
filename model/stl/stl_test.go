package stl_test

import (
	"bytes"
	"context"
	"encoding/binary"
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
	"github.com/rm4n0s/glutter/model/stl"
)

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

// tri is a triangle of a hand-made file.
type tri struct {
	normal [3]float32
	v      [3][3]float32
	attr   uint16
}

func binaryFile(header string, tris ...tri) []byte {
	b := make([]byte, 80, 84+50*len(tris))
	copy(b, header)
	b = binary.LittleEndian.AppendUint32(b, uint32(len(tris)))
	for _, t := range tris {
		for _, f := range t.normal {
			b = binary.LittleEndian.AppendUint32(b, math.Float32bits(f))
		}
		for _, v := range t.v {
			for _, f := range v {
				b = binary.LittleEndian.AppendUint32(b, math.Float32bits(f))
			}
		}
		b = binary.LittleEndian.AppendUint16(b, t.attr)
	}
	return b
}

func readTestdata(t testing.TB, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func mustDecode(t *testing.T, data []byte, o stl.Options) *model.Model {
	t.Helper()
	o.Logger = quiet
	m, err := stl.Decode(context.Background(), bytes.NewReader(data), o)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	return m
}

// checkMesh checks what every mesh has to be: the indices are in range, the arrays are as long as the vertices, the normals have
// length 1.
func checkMesh(t *testing.T, m *model.Mesh) {
	t.Helper()
	n := m.VertexCount()
	if len(m.Indices)%3 != 0 {
		t.Errorf("%d indices are not whole triangles", len(m.Indices))
	}
	for _, i := range m.Indices {
		if int(i) >= n {
			t.Fatalf("index %d of %d vertices", i, n)
		}
	}
	if len(m.Normals) != 3*n {
		t.Fatalf("%d normal numbers for %d vertices", len(m.Normals), n)
	}
	for i := 0; i < n; i++ {
		l := math.Sqrt(float64(m.Normals[3*i]*m.Normals[3*i] + m.Normals[3*i+1]*m.Normals[3*i+1] + m.Normals[3*i+2]*m.Normals[3*i+2]))
		if math.Abs(l-1) > 1e-5 {
			t.Fatalf("the normal of vertex %d has length %v", i, l)
		}
	}
}

func TestDecodeCube(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		opts     stl.Options
		vertices int
		colored  bool
	}{
		{"ascii, creased at the corners", "cube.ascii.stl", stl.Options{}, 24, false},
		{"binary, creased at the corners", "cube.binary.stl", stl.Options{}, 24, false},
		{"binary whose header begins with solid", "cube.solidheader.stl", stl.Options{}, 24, false},
		{"smooth across every edge", "cube.binary.stl", stl.Options{CreaseAngle: -1}, 8, false},
		{"a crease angle above 90 degrees", "cube.binary.stl", stl.Options{CreaseAngle: 1.6}, 8, false},
		{"flat", "cube.binary.stl", stl.Options{Flat: true}, 36, false},
		{"each corner of the red faces is a vertex for the red and one for the gray", "cube.colored.stl", stl.Options{CreaseAngle: -1}, 16, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := mustDecode(t, readTestdata(t, tt.file), tt.opts)
			if len(m.Nodes) != 1 || len(m.Meshes) != 1 || len(m.Nodes[0].Meshes) != 1 || m.Nodes[0].Meshes[0] != m.Meshes[0] {
				t.Fatalf("model of %d nodes and %d meshes", len(m.Nodes), len(m.Meshes))
			}
			mesh := m.Meshes[0]
			checkMesh(t, mesh)
			if mesh.PrimitiveCount() != 12 || mesh.Primitive != model.PrimitiveTriangles {
				t.Errorf("%d primitives of %v, want 12 triangles", mesh.PrimitiveCount(), mesh.Primitive)
			}
			if mesh.VertexCount() != tt.vertices {
				t.Errorf("%d vertices, want %d", mesh.VertexCount(), tt.vertices)
			}
			if (mesh.Colors != nil) != tt.colored {
				t.Errorf("colors %v, want %v", mesh.Colors != nil, tt.colored)
			}
			if b := m.Bounds(); b.Min.X != 0 || b.Max.X != 1 || b.Min.Z != 0 || b.Max.Z != 1 {
				t.Errorf("bounds %v", b)
			}
			if m.TriangleCount() != 12 {
				t.Errorf("TriangleCount %d", m.TriangleCount())
			}
		})
	}
}

func TestTheNormalsOfACubeAreTheNormalsOfItsFaces(t *testing.T) {
	m := mustDecode(t, readTestdata(t, "cube.binary.stl"), stl.Options{})
	mesh := m.Meshes[0]
	// Every vertex of a creased cube belongs to one face: its normal is an axis, and it is on that face.
	for i := 0; i < mesh.VertexCount(); i++ {
		p := mesh.Position(i)
		n := [3]float32{mesh.Normals[3*i], mesh.Normals[3*i+1], mesh.Normals[3*i+2]}
		axes := 0
		for k, c := range n {
			if c != 0 {
				axes++
				coord := []float64{p.X, p.Y, p.Z}[k]
				if want := (1 + float64(c)) / 2; coord != want {
					t.Errorf("vertex %d at %v has the normal %v", i, p, n)
				}
			}
		}
		if axes != 1 {
			t.Errorf("vertex %d has the normal %v", i, n)
		}
	}
	// Smooth across everything, the normal of a corner points away from the middle.
	m = mustDecode(t, readTestdata(t, "cube.binary.stl"), stl.Options{CreaseAngle: -1})
	mesh = m.Meshes[0]
	for i := 0; i < mesh.VertexCount(); i++ {
		p := mesh.Position(i)
		for k, c := range []float64{p.X - 0.5, p.Y - 0.5, p.Z - 0.5} {
			if float64(mesh.Normals[3*i+k])*c <= 0 {
				t.Errorf("vertex %d at %v has the normal %v", i, p, mesh.Normals[3*i:3*i+3])
			}
		}
	}
}

func TestColorsOfBinaryFiles(t *testing.T) {
	m := mustDecode(t, readTestdata(t, "cube.colored.stl"), stl.Options{Flat: true})
	mesh := m.Meshes[0]
	if len(mesh.Colors) != 4*mesh.VertexCount() {
		t.Fatalf("%d color bytes for %d vertices", len(mesh.Colors), mesh.VertexCount())
	}
	red, gray := 0, 0
	for i := 0; i < mesh.VertexCount(); i++ {
		c := mesh.Colors[4*i : 4*i+4]
		switch {
		case c[0] == 255 && c[1] == 0 && c[2] == 0 && c[3] == 255:
			red++
		case c[0] == 199 && c[3] == 255:
			gray++
		default:
			t.Errorf("color %v", c)
		}
	}
	// The first four triangles are red, the others have no color and get the default.
	if red != 12 || gray != 24 {
		t.Errorf("%d red vertices and %d gray ones, want 12 and 24", red, gray)
	}
	// The conventions in detail: five bits per channel, bit 15 says that the color is valid.
	tests := []struct {
		name string
		attr uint16
		want []byte
	}{
		{"no bit 15 is no color", 0x7FFF, nil},
		{"all zero is no color", 0, nil},
		{"black", 0x8000, []byte{0, 0, 0, 255}},
		{"white", 0xFFFF, []byte{255, 255, 255, 255}},
		{"red is bits 10 to 14", 0x8000 | 31<<10, []byte{255, 0, 0, 255}},
		{"green is bits 5 to 9", 0x8000 | 31<<5, []byte{0, 255, 0, 255}},
		{"blue is bits 0 to 4", 0x8000 | 31, []byte{0, 0, 255, 255}},
		{"half", 0x8000 | 16<<10, []byte{132, 0, 0, 255}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := binaryFile("", tri{v: [3][3]float32{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}}, attr: tt.attr})
			mesh := mustDecode(t, data, stl.Options{}).Meshes[0]
			if !bytes.Equal(mesh.Colors, bytes.Repeat(tt.want, 3)) {
				t.Errorf("colors %v, want %v three times", mesh.Colors, tt.want)
			}
		})
	}
}

func TestWelding(t *testing.T) {
	// Two triangles that share an edge, with the shared corners a little apart.
	data := binaryFile("", tri{v: [3][3]float32{{0, 0, 0}, {1.504, 0, 0}, {0, 1, 0}}}, tri{v: [3][3]float32{{1.506, 0, 0}, {1.5, 1, 0}, {0, 1, 0}}})
	tests := []struct {
		name string
		opts stl.Options
		want int
	}{
		{"equal bit for bit", stl.Options{}, 5},
		{"a tolerance welds what is within a cell", stl.Options{WeldTolerance: 0.01}, 4},
		{"flat welds nothing", stl.Options{Flat: true}, 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.opts.CreaseAngle = -1
			mesh := mustDecode(t, data, tt.opts).Meshes[0]
			checkMesh(t, mesh)
			if mesh.VertexCount() != tt.want {
				t.Errorf("%d vertices, want %d", mesh.VertexCount(), tt.want)
			}
		})
	}
	// -0 is 0.
	neg := math.Float32frombits(0x80000000)
	data = binaryFile("", tri{v: [3][3]float32{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}}}, tri{v: [3][3]float32{{neg, neg, neg}, {0, 1, 0}, {-1, 0, 0}}})
	if got := mustDecode(t, data, stl.Options{}).Meshes[0].VertexCount(); got != 4 {
		t.Errorf("-0 and 0 gave %d vertices, want 4", got)
	}
}

func TestDegenerateAndTurnedTriangles(t *testing.T) {
	var logged bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logged, nil))
	good := tri{normal: [3]float32{0, 0, 1}, v: [3][3]float32{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}}}
	line := tri{v: [3][3]float32{{0, 0, 0}, {1, 0, 0}, {2, 0, 0}}}
	point := tri{v: [3][3]float32{{3, 3, 3}, {3, 3, 3}, {3, 3, 3}}}
	turned := tri{normal: [3]float32{0, 0, -1}, v: [3][3]float32{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}}}
	noNormal := tri{v: [3][3]float32{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}}}

	m, err := stl.Decode(context.Background(), bytes.NewReader(binaryFile("", good, line, point)), stl.Options{Logger: logger})
	if err != nil {
		t.Fatal(err)
	}
	if got := m.Meshes[0].PrimitiveCount(); got != 1 {
		t.Errorf("%d triangles kept, want 1", got)
	}
	if !bytes.Contains(logged.Bytes(), []byte("dropped=2")) {
		t.Errorf("the log is %q, want the number of dropped triangles", logged.String())
	}
	// The stored normal wins over the winding; with none stored the winding stands.
	for _, tt := range []struct {
		name  string
		t     tri
		wantZ float32
	}{{"the normal says down", turned, -1}, {"the normal agrees", good, 1}, {"no normal", noNormal, 1}} {
		mesh := mustDecode(t, binaryFile("", tt.t), stl.Options{}).Meshes[0]
		for i := 0; i < mesh.VertexCount(); i++ {
			if mesh.Normals[3*i+2] != tt.wantZ {
				t.Errorf("%s: normal %v, want z %v", tt.name, mesh.Normals[3*i:3*i+3], tt.wantZ)
			}
		}
	}
	if _, err := stl.Decode(context.Background(), bytes.NewReader(binaryFile("", line)), stl.Options{Logger: quiet}); err == nil {
		t.Error("a file of degenerate triangles only did not fail")
	}
}

func TestASCIIDetails(t *testing.T) {
	two := mustDecode(t, readTestdata(t, "two.ascii.stl"), stl.Options{})
	if len(two.Nodes) != 2 || two.Nodes[0].Name != "first" || two.Nodes[1].Name != "second" || two.Meshes[1].Name != "second" {
		t.Errorf("nodes %d: %+v", len(two.Nodes), two.Nodes)
	}
	// Any case, tabs, windows line ends, exponents, no final newline, a byte order mark, no endsolid.
	text := "\xEF\xBB\xBFSOLID part one\r\n FACET NORMAL 0 0 1E0\r\n\tOUTER LOOP\r\n\t\tVERTEX 0 0 0\r\n VERTEX 1e0 0 +0\r\n VERTEX 0 1.0E+0 0\r\n ENDLOOP\r\n ENDFACET"
	m := mustDecode(t, []byte(text), stl.Options{})
	if m.Meshes[0].Name != "part one" || m.Meshes[0].PrimitiveCount() != 1 {
		t.Errorf("mesh %q of %d triangles", m.Meshes[0].Name, m.Meshes[0].PrimitiveCount())
	}
	// The name option names a solid that has none.
	m = mustDecode(t, []byte("solid\nfacet normal 0 0 1\nouter loop\nvertex 0 0 0\nvertex 1 0 0\nvertex 0 1 0\nendloop\nendfacet\nendsolid\n"), stl.Options{Name: "given"})
	if m.Meshes[0].Name != "given" || m.Name != "given" {
		t.Errorf("names %q %q", m.Name, m.Meshes[0].Name)
	}
}

func TestFloat64(t *testing.T) {
	const big = 1234567890.123456789
	text := func(x string) []byte {
		return []byte("solid s\nfacet normal 0 0 1\nouter loop\nvertex " + x + " 0 0\nvertex 1 0 0\nvertex 0 1 0\nendloop\nendfacet\nendsolid s\n")
	}
	tests := []struct {
		name    string
		x       string
		opts    stl.Options
		wide    bool
		wantX   float64
		epsilon float64
	}{
		{"a number with more digits, asked for", "1234567890.123456789", stl.Options{Float64: true}, true, big, 1e-9},
		{"not asked for", "1234567890.123456789", stl.Options{}, false, float64(float32(big)), 0},
		{"asked for, but float32 holds the file", "0.5", stl.Options{Float64: true}, false, 0.5, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mesh := mustDecode(t, text(tt.x), tt.opts).Meshes[0]
			if (mesh.Positions64 != nil) != tt.wide || (mesh.Positions64 == nil) != (mesh.Positions != nil) {
				t.Fatalf("positions %d, positions64 %d", len(mesh.Positions), len(mesh.Positions64))
			}
			found := false
			for i := 0; i < mesh.VertexCount(); i++ {
				if p := mesh.Position(i); p.Y == 0 && p.Z == 0 && p.X != 1 && math.Abs(p.X-tt.wantX) <= tt.epsilon {
					found = true
				}
			}
			if !found && tt.wantX != 0 {
				t.Errorf("no vertex at x = %v", tt.wantX)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	m, err := stl.Load(context.Background(), os.DirFS("testdata"), "cube.binary.stl", stl.Options{Logger: quiet})
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "cube.binary" || m.Meshes[0].Name != "cube.binary" || m.Nodes[0].Name != "cube.binary" {
		t.Errorf("names %q %q %q", m.Name, m.Meshes[0].Name, m.Nodes[0].Name)
	}
	if m.Units.MetersPerUnit != 0 {
		t.Errorf("units %+v: an STL file has none", m.Units)
	}
}

func TestErrors(t *testing.T) {
	cube := readTestdata(t, "cube.binary.stl")
	asciiCube := string(readTestdata(t, "cube.ascii.stl"))
	nan := math.Float32frombits(0x7FC00000)
	inf := float32(math.Inf(1))
	hugeCount := append([]byte(nil), cube[:84]...)
	binary.LittleEndian.PutUint32(hugeCount[80:], 0xFFFFFFFF)
	tests := []struct {
		name   string
		data   []byte
		route  string
		reason string
		key    string
	}{
		{"empty input", nil, "Decode->decodeBytes->decodeBinary." + stl.TagSTLInvalid, "short", "offset"},
		{"shorter than a header", []byte("not stl"), "Decode->decodeBytes->decodeBinary." + stl.TagSTLInvalid, "short", "offset"},
		{"binary cut short", cube[:len(cube)-10], "Decode->decodeBytes->decodeBinary." + stl.TagSTLInvalid, "truncated", "offset"},
		{"a count that no file holds", hugeCount, "Decode->decodeBytes->decodeBinary." + stl.TagSTLInvalid, "truncated", "offset"},
		{"binary with no triangles", readTestdata(t, "empty.binary.stl"), "Decode->decodeBytes->decodeBinary." + stl.TagSTLInvalid, "empty", "offset"},
		{"a coordinate that is NaN", binaryFile("", tri{v: [3][3]float32{{0, 0, 0}, {nan, 0, 0}, {0, 1, 0}}}), "Decode->decodeBytes->decodeBinary." + stl.TagSTLInvalid, "not finite", "offset"},
		{"a coordinate that is infinite", binaryFile("", tri{v: [3][3]float32{{0, 0, 0}, {1, inf, 0}, {0, 1, 0}}}), "Decode->decodeBytes->decodeBinary." + stl.TagSTLInvalid, "not finite", "offset"},
		{"a triangle with no area only", binaryFile("", tri{}), "Decode->decodeBytes->decodeBinary->build[...]." + stl.TagSTLInvalid, "degenerate", ""},
		{"solid with nothing", []byte("solid x\nendsolid x\n"), "Decode->decodeBytes->decodeASCII->parseASCII[...]->parseASCII[...].func1->build[...]." + stl.TagSTLInvalid, "empty", ""},
		{"only a word", []byte("solid"), "Decode->decodeBytes->decodeASCII->parseASCII[...]->parseASCII[...].func1->build[...]." + stl.TagSTLInvalid, "empty", ""},
		{"facet where none can be", []byte("solid x\nfacet normal 0 0 1\nfacet normal 0 0 1\n"), "Decode->decodeBytes->decodeASCII->parseASCII[...]." + stl.TagSTLInvalid, "unexpected facet", "line"},
		{"facet without normal", []byte("solid x\nfacet 0 0 1\n"), "Decode->decodeBytes->decodeASCII->parseASCII[...]." + stl.TagSTLInvalid, "facet normal", "line"},
		{"a normal that is not a number", []byte("solid x\nfacet normal 0 a 1\n"), "Decode->decodeBytes->decodeASCII->parseASCII[...]." + stl.TagSTLInvalid, "not a number", "line"},
		{"a missing number", []byte("solid x\nfacet normal 0 1\n"), "Decode->decodeBytes->decodeASCII->parseASCII[...]." + stl.TagSTLInvalid, "not a number", "line"},
		{"outer without loop", []byte("solid x\nfacet normal 0 0 1\nouter\n"), "Decode->decodeBytes->decodeASCII->parseASCII[...]." + stl.TagSTLInvalid, "unexpected loop", "line"},
		{"vertex outside a loop", []byte("solid x\nvertex 0 0 0\n"), "Decode->decodeBytes->decodeASCII->parseASCII[...]." + stl.TagSTLInvalid, "unexpected vertex", "line"},
		{"four vertices", []byte("solid x\nfacet normal 0 0 1\nouter loop\nvertex 0 0 0\nvertex 0 0 0\nvertex 0 0 0\nvertex 0 0 0\n"), "Decode->decodeBytes->decodeASCII->parseASCII[...]." + stl.TagSTLInvalid, "unexpected vertex", "line"},
		{"a vertex that is not a number", []byte("solid x\nfacet normal 0 0 1\nouter loop\nvertex 0 0 0x\n"), "Decode->decodeBytes->decodeASCII->parseASCII[...]." + stl.TagSTLInvalid, "not a number", "line"},
		{"a vertex that is infinite", []byte("solid x\nfacet normal 0 0 1\nouter loop\nvertex 0 0 inf\n"), "Decode->decodeBytes->decodeASCII->parseASCII[...]." + stl.TagSTLInvalid, "not finite", "line"},
		{"a vertex that is too big", []byte("solid x\nfacet normal 0 0 1\nouter loop\nvertex 0 0 1e999\n"), "Decode->decodeBytes->decodeASCII->parseASCII[...]." + stl.TagSTLInvalid, "not finite", "line"},
		{"a vertex beyond float32", []byte("solid x\nfacet normal 0 0 1\nouter loop\nvertex 0 0 1e100\n"), "Decode->decodeBytes->decodeASCII->parseASCII[...]." + stl.TagSTLInvalid, "out of range", "line"},
		{"endloop with two vertices", []byte("solid x\nfacet normal 0 0 1\nouter loop\nvertex 0 0 0\nvertex 0 0 0\nendloop\n"), "Decode->decodeBytes->decodeASCII->parseASCII[...]." + stl.TagSTLInvalid, "vertex count", "line"},
		{"endfacet too early", []byte("solid x\nfacet normal 0 0 1\nendfacet\n"), "Decode->decodeBytes->decodeASCII->parseASCII[...]." + stl.TagSTLInvalid, "unexpected endfacet", "line"},
		{"endsolid inside a facet", []byte("solid x\nfacet normal 0 0 1\nendsolid\n"), "Decode->decodeBytes->decodeASCII->parseASCII[...]." + stl.TagSTLInvalid, "unexpected endsolid", "line"},
		{"a solid inside a solid", []byte("solid x\nsolid y\n"), "Decode->decodeBytes->decodeASCII->parseASCII[...]." + stl.TagSTLInvalid, "nesting", "line"},
		{"a line that is not STL", []byte("solid x\nhello world\n"), "Decode->decodeBytes->decodeASCII->parseASCII[...]." + stl.TagSTLInvalid, "unknown keyword", "line"},
		{"text after the last solid", []byte(asciiCube + "more\n"), "Decode->decodeBytes->decodeASCII->parseASCII[...]." + stl.TagSTLInvalid, "unknown keyword", "line"},
		{"cut inside a facet", []byte(asciiCube[:strings.Index(asciiCube, "endloop")]), "Decode->decodeBytes->decodeASCII->parseASCII[...]." + stl.TagSTLInvalid, "truncated", "line"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := stl.Decode(context.Background(), bytes.NewReader(tt.data), stl.Options{Logger: quiet})
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
			if got := appErr.Metadata["reason"]; got != tt.reason {
				t.Errorf("reason %v, want %q", got, tt.reason)
			}
			if tt.key != "" {
				if _, ok := appErr.Metadata[tt.key]; !ok {
					t.Errorf("no %q in the metadata %v", tt.key, appErr.Metadata)
				}
			}
		})
	}
}

func TestTheLineOfAnErrorIsTheLineOfTheFault(t *testing.T) {
	_, err := stl.Decode(context.Background(), bytes.NewReader([]byte("solid x\nfacet normal 0 0 1\nouter loop\nvertex 0 0 0\nvertex 1 0 q\n")), stl.Options{})
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("got %T", err)
	}
	if appErr.Metadata["line"] != 5 {
		t.Errorf("line %v, want 5; metadata %v", appErr.Metadata["line"], appErr.Metadata)
	}
	// The offset of a binary fault is the byte of the coordinate.
	nan := math.Float32frombits(0x7FC00000)
	_, err = stl.Decode(context.Background(), bytes.NewReader(binaryFile("", tri{v: [3][3]float32{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}}}, tri{v: [3][3]float32{{0, 0, 0}, {nan, 0, 0}, {0, 1, 0}}})), stl.Options{})
	appErr, ok = errors.FromError(err)
	if !ok {
		t.Fatalf("got %T", err)
	}
	if appErr.Metadata["offset"] != 84+50+12 || appErr.Metadata["triangle"] != 1 {
		t.Errorf("metadata %v, want the offset %d of triangle 1", appErr.Metadata, 84+50+12)
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
	cube := readTestdata(t, "cube.binary.stl")

	tests := []struct {
		name  string
		run   func() error
		route string
		cause error
		meta  string
	}{
		{"a reader that fails", func() error {
			_, err := stl.Decode(context.Background(), &failingReader{after: 100}, stl.Options{})
			return err
		}, "Decode->ReadAll." + model.TagModelReadFailed, io.ErrUnexpectedEOF, "offset"},
		{"a context that is done before the read", func() error {
			_, err := stl.Decode(cancelled, bytes.NewReader(cube), stl.Options{})
			return err
		}, "Decode->ReadAll." + model.TagModelReadFailed, context.Canceled, "offset"},
		{"a file that is not there", func() error {
			_, err := stl.Load(context.Background(), fstest.MapFS{}, "none.stl", stl.Options{})
			return err
		}, "Load." + model.TagModelReadFailed, os.ErrNotExist, "file"},
		{"a file in a context that is done", func() error {
			_, err := stl.Load(cancelled, fstest.MapFS{"a.stl": {Data: cube}}, "a.stl", stl.Options{})
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
			if !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
			if !errors.Is(err, tt.cause) {
				t.Errorf("the cause of %v is not %v", err, tt.cause)
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

// signalEOF tells the context that the data is all read.
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
	big := binaryFile("", gridTriangles(180)...) // 64,800 triangles
	text := []byte("solid big\n")
	for i := 0; i < 20000; i++ {
		text = append(text, "facet normal 0 0 1\nouter loop\nvertex 0 0 0\nvertex 1 0 0\nvertex 0 1 0\nendloop\nendfacet\n"...)
	}
	text = append(text, "endsolid big\n"...)
	// The context is done after more and more looks at it, once the data is read: whatever the number of workers, the loader stops
	// with ModelReadFailed in the parse and then in the build, and finishes when the context outlasts it.
	tests := []struct {
		name   string
		data   []byte
		routes []string
	}{
		{"binary", big, []string{"Decode->decodeBytes->decodeBinary.", "Decode->decodeBytes->decodeBinary->build[...]."}},
		{"ascii", text, []string{"Decode->decodeBytes->decodeASCII->parseASCII[...].", "Decode->decodeBytes->decodeASCII->parseASCII[...]->parseASCII[...].func1->build[...]."}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seen := map[string]bool{}
			finished := false
			for after := 0; after < 400 && !finished; after++ {
				ctx := &cancelAfter{Context: context.Background()}
				ctx.n.Store(int64(after))
				_, err := stl.Decode(ctx, signalEOF{bytes.NewReader(tt.data), ctx}, stl.Options{Logger: quiet})
				if err == nil {
					finished = true
					break
				}
				appErr, ok := errors.FromError(err)
				if !ok || !errors.Is(err, context.Canceled) {
					t.Fatalf("after %d looks: %T %v", after, err, err)
				}
				for _, r := range tt.routes {
					if appErr.HasRoute(r + model.TagModelReadFailed) {
						seen[r] = true
					}
				}
			}
			if !finished {
				t.Error("the loader never finished")
			}
			for _, r := range tt.routes {
				if !seen[r] {
					t.Errorf("the loader never stopped at %s", r+model.TagModelReadFailed)
				}
			}
		})
	}
}

// gridTriangles makes the triangles of a height field of n by n squares.
func gridTriangles(n int) []tri {
	h := func(i, j int) float32 { return float32(math.Sin(float64(i)*0.1) * math.Cos(float64(j)*0.13)) }
	p := func(i, j int) [3]float32 { return [3]float32{float32(i), float32(j), h(i, j)} }
	tris := make([]tri, 0, 2*n*n)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			tris = append(tris, tri{v: [3][3]float32{p(i, j), p(i+1, j), p(i+1, j+1)}}, tri{v: [3][3]float32{p(i, j), p(i+1, j+1), p(i, j+1)}})
		}
	}
	return tris
}

func TestAHeightFieldIsWeldedToItsGrid(t *testing.T) {
	const n = 40
	m := mustDecode(t, binaryFile("", gridTriangles(n)...), stl.Options{CreaseAngle: -1})
	mesh := m.Meshes[0]
	checkMesh(t, mesh)
	if got, want := mesh.VertexCount(), (n+1)*(n+1); got != want {
		t.Errorf("%d vertices, want %d", got, want)
	}
	for i := 0; i < mesh.VertexCount(); i++ {
		if mesh.Normals[3*i+2] <= 0 {
			t.Fatalf("a height field has up normals, vertex %d has %v", i, mesh.Normals[3*i:3*i+3])
		}
	}
}

func fuzzSeeds(f *testing.F) {
	for _, name := range []string{"cube.ascii.stl", "cube.binary.stl", "cube.solidheader.stl", "cube.colored.stl", "two.ascii.stl", "empty.binary.stl"} {
		data, err := os.ReadFile("testdata/" + name)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(data, false, false)
	}
	f.Add([]byte("solid x\nfacet normal 0 0 1\nouter loop\nvertex 0 0 0\nvertex 1 0 0\nvertex 0 1 0\nendloop\nendfacet\nendsolid"), true, true)
	f.Add([]byte("solid"), false, false)
	f.Add([]byte{}, false, false)
}

func FuzzDecode(f *testing.F) {
	fuzzSeeds(f)
	f.Fuzz(func(t *testing.T, data []byte, flat, wide bool) {
		m, err := stl.Decode(context.Background(), bytes.NewReader(data), stl.Options{Flat: flat, Float64: wide, Logger: quiet})
		if err != nil {
			if _, ok := errors.FromError(err); !ok {
				t.Fatalf("an error that is not an *errors.Error: %T %v", err, err)
			}
			return
		}
		for _, mesh := range m.Meshes {
			checkMesh(t, mesh)
			for i := 0; i < mesh.VertexCount(); i++ {
				p := mesh.Position(i)
				if math.IsNaN(p.X+p.Y+p.Z) || math.IsInf(p.X+p.Y+p.Z, 0) {
					t.Fatalf("vertex %d is %v", i, p)
				}
			}
		}
	})
}

func BenchmarkDecodeBinary1M(b *testing.B) {
	// 708 by 708 squares make 1,002,528 triangles.
	data := binaryFile("", gridTriangles(708)...)
	for _, tt := range []struct {
		name string
		opts stl.Options
	}{{"welded and smooth", stl.Options{}}, {"flat", stl.Options{Flat: true}}} {
		b.Run(tt.name, func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				m, err := stl.Decode(context.Background(), bytes.NewReader(data), tt.opts)
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
