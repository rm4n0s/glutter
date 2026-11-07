package ply_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/model"
	"github.com/rm4n0s/glutter/model/ply"
)

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

func decode(t *testing.T, data []byte) *model.Model {
	t.Helper()
	m, err := ply.Decode(context.Background(), bytes.NewReader(data), ply.Options{Name: "m", Logger: quiet})
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
	if (m.Positions == nil) == (m.Positions64 == nil) {
		t.Errorf("positions %d, positions64 %d: exactly one is set", len(m.Positions), len(m.Positions64))
	}
}

// writer writes the body of a binary file in the byte order.
type writer struct {
	order binary.ByteOrder
	buf   bytes.Buffer
}

func (w *writer) u8(v uint8) { w.buf.WriteByte(v) }
func (w *writer) u16(v uint16) {
	var b [2]byte
	w.order.PutUint16(b[:], v)
	w.buf.Write(b[:])
}
func (w *writer) u32(v uint32) {
	var b [4]byte
	w.order.PutUint32(b[:], v)
	w.buf.Write(b[:])
}
func (w *writer) f32(v float32) { w.u32(math.Float32bits(v)) }
func (w *writer) f64(v float64) {
	var b [8]byte
	w.order.PutUint64(b[:], math.Float64bits(v))
	w.buf.Write(b[:])
}

// everything is a file of a triangle and a quad with every property, in each of the three formats.
//
//	vertices 0 (0,0,0) 1 (1,0,0) 2 (1,1,0) 3 (0,1,0), the normals (0,0,2), the colors 10 20 30 40 (+i), and the texture coordinates (i/4, 0.25)
func everything(format string) []byte {
	head := "ply\nformat " + format + " 1.0\ncomment made by a test\nobj_info something\n" +
		"element vertex 4\nproperty float x\nproperty float y\nproperty float z\nproperty float nx\nproperty float ny\nproperty float nz\n" +
		"property uchar red\nproperty uchar green\nproperty uchar blue\nproperty uchar alpha\nproperty float s\nproperty float t\n" +
		"element face 2\nproperty list uchar int vertex_indices\nend_header\n"
	xy := [4][2]float32{{0, 0}, {1, 0}, {1, 1}, {0, 1}}
	if format == "ascii" {
		var b strings.Builder
		b.WriteString(head)
		for i := 0; i < 4; i++ {
			fmt.Fprintf(&b, "%g %g 0 0 0 2 %d %d %d %d %g 0.25\n", xy[i][0], xy[i][1], 10+i, 20+i, 30+i, 40+i, float64(i)/4)
		}
		b.WriteString("3 0 1 2\n4 0 1 2 3\n")
		return []byte(b.String())
	}
	w := &writer{order: binary.LittleEndian}
	if format == "binary_big_endian" {
		w.order = binary.BigEndian
	}
	w.buf.WriteString(head)
	for i := 0; i < 4; i++ {
		w.f32(xy[i][0])
		w.f32(xy[i][1])
		w.f32(0)
		w.f32(0)
		w.f32(0)
		w.f32(2)
		w.u8(uint8(10 + i))
		w.u8(uint8(20 + i))
		w.u8(uint8(30 + i))
		w.u8(uint8(40 + i))
		w.f32(float32(i) / 4)
		w.f32(0.25)
	}
	for _, f := range [][]int32{{0, 1, 2}, {0, 1, 2, 3}} {
		w.u8(uint8(len(f)))
		for _, v := range f {
			w.u32(uint32(v))
		}
	}
	return w.buf.Bytes()
}

func TestEveryPropertyIsReadInEveryFormat(t *testing.T) {
	for _, format := range []string{"ascii", "binary_little_endian", "binary_big_endian"} {
		t.Run(format, func(t *testing.T) {
			m := decode(t, everything(format))
			if m.Name != "m" || len(m.Nodes) != 1 || m.Nodes[0].Name != "m" || len(m.Meshes) != 1 || len(m.Nodes[0].Meshes) != 1 || m.Nodes[0].Meshes[0] != m.Meshes[0] {
				t.Fatalf("model %q with %d nodes and %d meshes", m.Name, len(m.Nodes), len(m.Meshes))
			}
			mesh := m.Meshes[0]
			checkMesh(t, mesh)
			if mesh.Primitive != model.PrimitiveTriangles || mesh.Name != "m" || mesh.VertexCount() != 4 {
				t.Fatalf("primitive %v, name %q, %d vertices", mesh.Primitive, mesh.Name, mesh.VertexCount())
			}
			// The triangle, and the quad as a fan of two.
			if want := []uint32{0, 1, 2, 0, 1, 2, 0, 2, 3}; len(mesh.Indices) != 9 || mesh.Indices[0] != 0 || mesh.Indices[1] != 1 || mesh.Indices[2] != 2 || !sameTriangles(mesh.Indices[3:], want[3:]) {
				t.Errorf("indices %v", mesh.Indices)
			}
			if got := mesh.Position(2); got.X != 1 || got.Y != 1 || got.Z != 0 {
				t.Errorf("position 2 = %v", got)
			}
			for i := 0; i < 4; i++ {
				if mesh.Normals[3*i] != 0 || mesh.Normals[3*i+1] != 0 || mesh.Normals[3*i+2] != 1 {
					t.Errorf("normal %d = %v, want (0, 0, 1): normalized", i, mesh.Normals[3*i:3*i+3])
				}
				if c := mesh.Colors[4*i : 4*i+4]; c[0] != byte(10+i) || c[1] != byte(20+i) || c[2] != byte(30+i) || c[3] != byte(40+i) {
					t.Errorf("color %d = %v", i, c)
				}
				if u, v := mesh.TexCoords[2*i], mesh.TexCoords[2*i+1]; u != float32(i)/4 || v != 0.75 {
					t.Errorf("texture coordinate %d = (%v, %v), want (%v, 0.75): v is turned", i, u, v, float32(i)/4)
				}
			}
			if got := m.Bounds(); got.Min.X != 0 || got.Max.X != 1 || got.Max.Y != 1 {
				t.Errorf("bounds %v", got)
			}
		})
	}
}

// sameTriangles says that the triangles of two lists are the same, each as a set of corners and the winding.
func sameTriangles(a, b []uint32) bool {
	if len(a) != len(b) {
		return false
	}
	key := func(t []uint32) [3]uint32 {
		// The corners from the smallest, in the winding.
		i := 0
		for k := 1; k < 3; k++ {
			if t[k] < t[i] {
				i = k
			}
		}
		return [3]uint32{t[i], t[(i+1)%3], t[(i+2)%3]}
	}
	for i := 0; i < len(a); i += 3 {
		if key(a[i:i+3]) != key(b[i:i+3]) {
			return false
		}
	}
	return true
}

func TestPositionTypes(t *testing.T) {
	tests := []struct {
		name       string
		typ        string
		value      float64
		wantWide   bool
		wantFirst  float64
		binaryOnly bool
	}{
		{"float", "float", 0.1, false, float64(float32(0.1)), false},
		{"float32", "float32", 0.5, false, 0.5, false},
		{"a double that float32 holds", "double", 0.5, false, 0.5, false},
		{"a double that float32 does not hold", "double", 0.1, true, 0.1, false},
		{"float64", "float64", 1234567890.123456, true, 1234567890.123456, false},
		{"a short", "short", -7, false, -7, false},
		{"an int", "int", 16777217, false, 16777216, false},
		{"a uint", "uint", 4000000000, false, 4000000000, false},
		{"a char", "char", -3, false, -3, false},
	}
	for _, tt := range tests {
		for _, format := range []string{"ascii", "binary_little_endian", "binary_big_endian"} {
			t.Run(tt.name+" "+format, func(t *testing.T) {
				head := "ply\nformat " + format + " 1.0\nelement vertex 2\nproperty " + tt.typ + " x\nproperty " + tt.typ + " y\nproperty " + tt.typ + " z\nend_header\n"
				var data []byte
				if format == "ascii" {
					data = []byte(head + fmt.Sprintf("%.17g 2 3\n%.17g 5 6\n", tt.value, tt.value))
				} else {
					w := &writer{order: binary.LittleEndian}
					if format == "binary_big_endian" {
						w.order = binary.BigEndian
					}
					w.buf.WriteString(head)
					for i := 0; i < 2; i++ {
						for _, v := range []float64{tt.value, float64(2 + 3*i), float64(3 + 3*i)} {
							switch tt.typ {
							case "float", "float32":
								w.f32(float32(v))
							case "double", "float64":
								w.f64(v)
							case "short":
								w.u16(uint16(int16(v)))
							case "char":
								w.u8(uint8(int8(v)))
							case "int":
								w.u32(uint32(int32(v)))
							case "uint":
								w.u32(uint32(v))
							}
						}
					}
					data = w.buf.Bytes()
				}
				m := decode(t, data)
				mesh := m.Meshes[0]
				checkMesh(t, mesh)
				if wide := mesh.Positions64 != nil; wide != tt.wantWide {
					t.Fatalf("Positions64 set is %v, want %v", wide, tt.wantWide)
				}
				if got := mesh.Position(0).X; got != tt.wantFirst {
					t.Errorf("x = %.17g, want %.17g", got, tt.wantFirst)
				}
				if got := mesh.Position(1); got.Y != 5 || got.Z != 6 {
					t.Errorf("vertex 1 = %v", got)
				}
			})
		}
	}
}

func TestColorTypes(t *testing.T) {
	tests := []struct {
		name  string
		typ   string
		write func(w *writer, v float64)
		given []float64
		want  []byte
	}{
		{"uchar", "uchar", func(w *writer, v float64) { w.u8(uint8(v)) }, []float64{0, 128, 255}, []byte{0, 128, 255}},
		{"float", "float", func(w *writer, v float64) { w.f32(float32(v)) }, []float64{0, 0.5, 1}, []byte{0, 128, 255}},
		{"double", "double", func(w *writer, v float64) { w.f64(v) }, []float64{0, 0.2, 2}, []byte{0, 51, 255}},
		{"ushort", "ushort", func(w *writer, v float64) { w.u16(uint16(v)) }, []float64{0, 32896, 65535}, []byte{0, 128, 255}},
		{"uint", "uint", func(w *writer, v float64) { w.u32(uint32(v)) }, []float64{0, 2155905152, 4294967295}, []byte{0, 128, 255}},
		{"short", "short", func(w *writer, v float64) { w.u16(uint16(int16(v))) }, []float64{-5, 16448, 32767}, []byte{0, 128, 255}},
		{"char", "char", func(w *writer, v float64) { w.u8(uint8(int8(v))) }, []float64{-5, 64, 127}, []byte{0, 64, 127}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &writer{order: binary.LittleEndian}
			fmt.Fprintf(&w.buf, "ply\nformat binary_little_endian 1.0\nelement vertex 1\nproperty float x\nproperty float y\nproperty float z\n"+
				"property %s red\nproperty %s green\nproperty %s blue\nend_header\n", tt.typ, tt.typ, tt.typ)
			w.f32(0)
			w.f32(0)
			w.f32(0)
			for _, v := range tt.given {
				tt.write(w, v)
			}
			mesh := decode(t, w.buf.Bytes()).Meshes[0]
			checkMesh(t, mesh)
			if !bytes.Equal(mesh.Colors[:3], tt.want) || mesh.Colors[3] != 255 {
				t.Errorf("colors %v, want %v and an alpha of 255", mesh.Colors, tt.want)
			}
		})
	}
}

func TestAFileWithNoFacesIsAPointCloud(t *testing.T) {
	data := "ply\nformat ascii 1.0\nelement vertex 3\nproperty float x\nproperty float y\nproperty float z\nproperty uchar red\nproperty uchar green\nproperty uchar blue\nend_header\n" +
		"0 0 0 255 0 0\n1 0 0 0 255 0\n0 1 0 0 0 255\n"
	for _, withFaces := range []bool{false, true} {
		text := data
		if withFaces {
			text += ""
			text = strings.Replace(text, "end_header", "element face 0\nproperty list uchar int vertex_indices\nend_header", 1)
		}
		m := decode(t, []byte(text))
		mesh := m.Meshes[0]
		checkMesh(t, mesh)
		if mesh.Primitive != model.PrimitivePoints || mesh.PrimitiveCount() != 3 || mesh.Indices[2] != 2 || mesh.Colors[4] != 0 || mesh.Colors[5] != 255 {
			t.Errorf("faces element %v: %v with %d points, indices %v, colors %v", withFaces, mesh.Primitive, mesh.PrimitiveCount(), mesh.Indices, mesh.Colors)
		}
		if b := m.Bounds(); b.Max.X != 1 || b.Max.Y != 1 {
			t.Errorf("bounds %v", b)
		}
	}
}

// polygonArea is the area of a polygon in the plane z = 0, positive when counter-clockwise.
func polygonArea(xy [][2]float64) float64 {
	a := 0.0
	for i := range xy {
		p, q := xy[i], xy[(i+1)%len(xy)]
		a += p[0]*q[1] - q[0]*p[1]
	}
	return a / 2
}

func TestPolygonsAreCutIntoTriangles(t *testing.T) {
	// The L of three squares is concave: a fan from its first corner would leave its notch covered.
	L := [][2]float64{{0, 0}, {2, 0}, {2, 1}, {1, 1}, {1, 2}, {0, 2}}
	square := [][2]float64{{0, 0}, {1, 0}, {1, 1}, {0, 1}}
	hex := [][2]float64{{2, 0}, {1, 1.7}, {-1, 1.7}, {-2, 0}, {-1, -1.7}, {1, -1.7}}
	tests := []struct {
		name string
		poly [][2]float64
		area float64
	}{{"a square", square, 1}, {"the L", L, 3}, {"a hexagon", hex, polygonArea(hex)}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b strings.Builder
			fmt.Fprintf(&b, "ply\nformat ascii 1.0\nelement vertex %d\nproperty float x\nproperty float y\nproperty float z\nelement face 1\nproperty list uchar uint vertex_index\nend_header\n", len(tt.poly))
			for _, p := range tt.poly {
				fmt.Fprintf(&b, "%g %g 0\n", p[0], p[1])
			}
			fmt.Fprintf(&b, "%d", len(tt.poly))
			for i := range tt.poly {
				fmt.Fprintf(&b, " %d", i)
			}
			b.WriteString("\n")
			mesh := decode(t, []byte(b.String())).Meshes[0]
			checkMesh(t, mesh)
			if mesh.PrimitiveCount() != len(tt.poly)-2 {
				t.Fatalf("%d triangles for %d corners", mesh.PrimitiveCount(), len(tt.poly))
			}
			sum := 0.0
			for i := 0; i < len(mesh.Indices); i += 3 {
				tri := make([][2]float64, 3)
				for k := range tri {
					p := mesh.Position(int(mesh.Indices[i+k]))
					tri[k] = [2]float64{p.X, p.Y}
				}
				a := polygonArea(tri)
				if a < -1e-9 {
					t.Errorf("triangle %d turns the wrong way", i/3)
				}
				sum += a
			}
			if math.Abs(sum-tt.area) > 1e-6 {
				t.Errorf("the triangles cover %v, the polygon is %v", sum, tt.area)
			}
		})
	}
}

func TestOtherElementsAndPropertiesAreSkipped(t *testing.T) {
	for _, format := range []string{"ascii", "binary_little_endian", "binary_big_endian"} {
		t.Run(format, func(t *testing.T) {
			head := "ply\nformat " + format + " 1.0\nelement edge 2\nproperty int vertex1\nproperty int vertex2\nproperty uchar flags\n" +
				"element vertex 3\nproperty float x\nproperty float y\nproperty float z\nproperty float confidence\nproperty list uchar float weights\n" +
				"element face 1\nproperty uchar flags\nproperty list uchar int vertex_indices\nproperty list uchar float texcoord\n" +
				"element material 1\nproperty list uchar uchar name\nend_header\n"
			var data []byte
			if format == "ascii" {
				data = []byte(head + "0 1 7\n1 2 7\n0 0 0 0.5 2 1.5 2.5\n1 0 0 0.5 0\n0 1 0 0.5 1 9\n7 3 0 1 2 6 0 0 1 0 1 1\n2 104 105\n")
			} else {
				w := &writer{order: binary.LittleEndian}
				if format == "binary_big_endian" {
					w.order = binary.BigEndian
				}
				w.buf.WriteString(head)
				for _, e := range [][3]uint32{{0, 1, 7}, {1, 2, 7}} {
					w.u32(e[0])
					w.u32(e[1])
					w.u8(uint8(e[2]))
				}
				for i := 0; i < 3; i++ {
					w.f32(float32(i % 2))
					w.f32(float32(i / 2))
					w.f32(0)
					w.f32(0.5)
					w.u8(uint8(i))
					for k := 0; k < i; k++ {
						w.f32(float32(k) + 1.5)
					}
				}
				w.u8(7)
				w.u8(3)
				w.u32(0)
				w.u32(1)
				w.u32(2)
				w.u8(6)
				for k := 0; k < 6; k++ {
					w.f32(float32(k % 2))
				}
				w.u8(2)
				w.u8('h')
				w.u8('i')
				data = w.buf.Bytes()
			}
			var logged bytes.Buffer
			m, err := ply.Decode(context.Background(), bytes.NewReader(data), ply.Options{Logger: slog.New(slog.NewTextHandler(&logged, nil))})
			if err != nil {
				t.Fatal(err)
			}
			mesh := m.Meshes[0]
			checkMesh(t, mesh)
			if mesh.PrimitiveCount() != 1 || mesh.VertexCount() != 3 || mesh.Indices[2] != 2 {
				t.Errorf("%d triangles, indices %v", mesh.PrimitiveCount(), mesh.Indices)
			}
			for _, want := range []string{"edge", "material", "confidence", "weights", "flags", "texcoord"} {
				if !strings.Contains(logged.String(), want) {
					t.Errorf("the log does not name %q:\n%s", want, logged.String())
				}
			}
		})
	}
}

func TestFacesMayComeBeforeTheVertices(t *testing.T) {
	text := "ply\nformat ascii 1.0\nelement face 2\nproperty list uchar int vertex_indices\nelement vertex 4\nproperty float x\nproperty float y\nproperty float z\nend_header\n" +
		"3 0 1 2\n4 0 1 2 3\n0 0 0\n1 0 0\n1 1 0\n0 1 0\n"
	mesh := decode(t, []byte(text)).Meshes[0]
	checkMesh(t, mesh)
	if mesh.PrimitiveCount() != 3 {
		t.Errorf("%d triangles, indices %v", mesh.PrimitiveCount(), mesh.Indices)
	}
}

func TestFacesOfFewerThanThreeCornersAreDropped(t *testing.T) {
	text := "ply\nformat ascii 1.0\nelement vertex 3\nproperty float x\nproperty float y\nproperty float z\nelement face 3\nproperty list uchar int vertex_indices\nend_header\n" +
		"0 0 0\n1 0 0\n0 1 0\n2 0 1\n3 0 1 2\n0\n"
	var logged bytes.Buffer
	m, err := ply.Decode(context.Background(), strings.NewReader(text), ply.Options{Logger: slog.New(slog.NewTextHandler(&logged, nil))})
	if err != nil {
		t.Fatal(err)
	}
	if m.Meshes[0].PrimitiveCount() != 1 || !strings.Contains(logged.String(), "fewer than three") || !strings.Contains(logged.String(), "faces=2") {
		t.Errorf("%d triangles, log %q", m.Meshes[0].PrimitiveCount(), logged.String())
	}
}

func TestNormalsOfLengthZeroAreNotUsed(t *testing.T) {
	text := "ply\nformat ascii 1.0\nelement vertex 2\nproperty float x\nproperty float y\nproperty float z\nproperty float nx\nproperty float ny\nproperty float nz\nend_header\n" +
		"0 0 0 0 0 1\n1 0 0 0 0 0\n"
	var logged bytes.Buffer
	m, err := ply.Decode(context.Background(), strings.NewReader(text), ply.Options{Logger: slog.New(slog.NewTextHandler(&logged, nil))})
	if err != nil {
		t.Fatal(err)
	}
	if m.Meshes[0].Normals != nil || !strings.Contains(logged.String(), "length 0") {
		t.Errorf("normals %v, log %q", m.Meshes[0].Normals, logged.String())
	}
}

func TestColorsAndCoordinatesThatAreNotWholeAreIgnored(t *testing.T) {
	text := "ply\nformat ascii 1.0\nelement vertex 1\nproperty float x\nproperty float y\nproperty float z\nproperty uchar red\nproperty uchar green\nproperty float s\nend_header\n0 0 0 1 2 0.5\n"
	var logged bytes.Buffer
	m, err := ply.Decode(context.Background(), strings.NewReader(text), ply.Options{Logger: slog.New(slog.NewTextHandler(&logged, nil))})
	if err != nil {
		t.Fatal(err)
	}
	if mesh := m.Meshes[0]; mesh.Colors != nil || mesh.TexCoords != nil || !strings.Contains(logged.String(), "red, green and blue together") {
		t.Errorf("colors %v, texture coordinates %v, log %q", mesh.Colors, mesh.TexCoords, logged.String())
	}
}

func TestTextureCoordinateNames(t *testing.T) {
	for _, names := range [][2]string{{"s", "t"}, {"u", "v"}, {"texture_u", "texture_v"}, {"texture_s", "texture_t"}} {
		text := "ply\nformat ascii 1.0\nelement vertex 1\nproperty float x\nproperty float y\nproperty float z\nproperty float " + names[0] + "\nproperty float " + names[1] + "\nend_header\n0 0 0 0.25 0.25\n"
		mesh := decode(t, []byte(text)).Meshes[0]
		if mesh.TexCoords == nil || mesh.TexCoords[0] != 0.25 || mesh.TexCoords[1] != 0.75 {
			t.Errorf("%v: texture coordinates %v", names, mesh.TexCoords)
		}
	}
}

func TestLineEndingsOfTheHeader(t *testing.T) {
	text := "ply\r\nformat ascii 1.0\r\nelement vertex 1\r\nproperty float x\r\nproperty float y\r\nproperty float z\r\nend_header\r\n1 2 3\r\n"
	mesh := decode(t, []byte(text)).Meshes[0]
	if p := mesh.Position(0); p.X != 1 || p.Y != 2 || p.Z != 3 {
		t.Errorf("position %v", p)
	}
	// A binary body after "end_header\r\n" starts at the byte after the line feed.
	w := &writer{order: binary.LittleEndian}
	w.buf.WriteString("ply\r\nformat binary_little_endian 1.0\r\nelement vertex 1\r\nproperty float x\r\nproperty float y\r\nproperty float z\r\nend_header\r\n")
	w.f32(1)
	w.f32(2)
	w.f32(3)
	mesh = decode(t, w.buf.Bytes()).Meshes[0]
	if p := mesh.Position(0); p.X != 1 || p.Y != 2 || p.Z != 3 {
		t.Errorf("binary position %v", p)
	}
}

func TestTheBodyMayWrapItsLines(t *testing.T) {
	text := "ply\nformat ascii 1.0\nelement vertex 2\nproperty float x\nproperty float y\nproperty float z\nend_header\n1 2\n3 4 5\n\n6\n"
	mesh := decode(t, []byte(text)).Meshes[0]
	if p := mesh.Position(1); p.X != 4 || p.Y != 5 || p.Z != 6 {
		t.Errorf("position %v", p)
	}
}

func TestErrors(t *testing.T) {
	const run = "Decode->decodeBytes->"
	const runBody = "Decode->decodeBytes->decoder.run->"
	vertexHead := "ply\nformat ascii 1.0\nelement vertex 1\nproperty float x\nproperty float y\nproperty float z\n"
	faceHead := vertexHead + "element face 1\nproperty list uchar int vertex_indices\n"
	tests := []struct {
		name   string
		text   string
		route  string
		reason string
		where  string
		at     int
	}{
		{"not a ply file", "plx\nformat ascii 1.0\n", run + "parseHeader." + ply.TagPLYInvalid, "not a ply file", "line", 1},
		{"an empty file", "", run + "parseHeader." + ply.TagPLYInvalid, "no end_header", "line", 0},
		{"a header that never ends", "ply\nformat ascii 1.0\ncomment x\n", run + "parseHeader." + ply.TagPLYInvalid, "no end_header", "line", 3},
		{"two formats", "ply\nformat ascii 1.0\nformat ascii 1.0\nend_header\n", run + "parseHeader." + ply.TagPLYInvalid, "format twice", "line", 3},
		{"an unknown format", "ply\nformat text 1.0\nend_header\n", run + "parseHeader." + ply.TagPLYInvalid, "unknown format", "line", 2},
		{"version 2", "ply\nformat ascii 2.0\nend_header\n", run + "parseHeader." + ply.TagPLYInvalid, "unsupported version", "line", 2},
		{"a format line with more", "ply\nformat ascii 1.0 x\nend_header\n", run + "parseHeader." + ply.TagPLYInvalid, "bad format line", "line", 2},
		{"no format", "ply\nend_header\n", run + "parseHeader." + ply.TagPLYInvalid, "no format", "line", 2},
		{"an unknown line", "ply\nformat ascii 1.0\nstuff 1\nend_header\n", run + "parseHeader." + ply.TagPLYInvalid, "unknown header line", "line", 3},
		{"an element without a count", "ply\nformat ascii 1.0\nelement vertex\nend_header\n", run + "parseHeader." + ply.TagPLYInvalid, "bad element", "line", 3},
		{"an element line with more", "ply\nformat ascii 1.0\nelement vertex 1 2\nend_header\n", run + "parseHeader." + ply.TagPLYInvalid, "bad element", "line", 3},
		{"a count that is not a number", "ply\nformat ascii 1.0\nelement vertex x\nend_header\n", run + "parseHeader." + ply.TagPLYInvalid, "bad count", "line", 3},
		{"a negative count", "ply\nformat ascii 1.0\nelement vertex -1\nend_header\n", run + "parseHeader." + ply.TagPLYInvalid, "bad count", "line", 3},
		{"a count beyond 2^31", "ply\nformat ascii 1.0\nelement vertex 4294967296\nend_header\n", run + "parseHeader." + ply.TagPLYInvalid, "bad count", "line", 3},
		{"an element twice", "ply\nformat ascii 1.0\nelement vertex 1\nelement vertex 1\nend_header\n", run + "parseHeader." + ply.TagPLYInvalid, "element twice", "line", 4},
		{"a property before an element", "ply\nformat ascii 1.0\nproperty float x\nend_header\n", run + "parseHeader." + ply.TagPLYInvalid, "property without element", "line", 3},
		{"a property of an unknown type", "ply\nformat ascii 1.0\nelement vertex 1\nproperty quad x\nend_header\n", run + "parseHeader." + ply.TagPLYInvalid, "bad type", "line", 4},
		{"a property with no name", "ply\nformat ascii 1.0\nelement vertex 1\nproperty float\nend_header\n", run + "parseHeader." + ply.TagPLYInvalid, "bad property", "line", 4},
		{"a property line with more", "ply\nformat ascii 1.0\nelement vertex 1\nproperty float x y\nend_header\n", run + "parseHeader." + ply.TagPLYInvalid, "bad property", "line", 4},
		{"a property twice", "ply\nformat ascii 1.0\nelement vertex 1\nproperty float x\nproperty int x\nend_header\n", run + "parseHeader." + ply.TagPLYInvalid, "property twice", "line", 5},
		{"a list of unknown types", "ply\nformat ascii 1.0\nelement face 1\nproperty list foo int v\nend_header\n", run + "parseHeader." + ply.TagPLYInvalid, "bad type", "line", 4},
		{"a list counted by a float", "ply\nformat ascii 1.0\nelement face 1\nproperty list float int v\nend_header\n", run + "parseHeader." + ply.TagPLYInvalid, "bad list count type", "line", 4},
		{"no vertex element", "ply\nformat ascii 1.0\nelement face 0\nproperty list uchar int vertex_indices\nend_header\n", "Decode->decodeBytes->decoder.run." + ply.TagPLYInvalid, "no vertex element", "line", 5},
		{"vertices without x", "ply\nformat ascii 1.0\nelement vertex 1\nproperty float y\nproperty float z\nend_header\n0 0\n", runBody + "decoder.readVertices." + ply.TagPLYInvalid, "no position", "line", 3},
		{"vertices without y", "ply\nformat ascii 1.0\nelement vertex 1\nproperty float x\nproperty float z\nend_header\n0 0\n", runBody + "decoder.readVertices." + ply.TagPLYInvalid, "no position", "line", 3},
		{"vertices without z", "ply\nformat ascii 1.0\nelement vertex 1\nproperty float x\nproperty float y\nend_header\n0 0\n", runBody + "decoder.readVertices." + ply.TagPLYInvalid, "no position", "line", 3},
		{"no vertices", "ply\nformat ascii 1.0\nelement vertex 0\nproperty float x\nproperty float y\nproperty float z\nend_header\n", runBody + "decoder.readVertices." + ply.TagPLYInvalid, "no vertices", "line", 3},
		{"more vertices than the body holds", vertexHead[:len(vertexHead)-0] + "end_header\n", runBody + "decoder.readVertices." + ply.TagPLYInvalid, "truncated", "line", 8},
		{"a lie about the count of the vertices", strings.Replace(vertexHead, "vertex 1", "vertex 100000", 1) + "end_header\n0 0 0\n", runBody + "decoder.readVertices." + ply.TagPLYInvalid, "truncated", "line", 8},
		{"a position that is not a number", vertexHead + "end_header\n0 0 q\n", runBody + "decoder.readVertices->cursor.value." + ply.TagPLYInvalid, "not a number", "line", 8},
		{"a position that is infinite", vertexHead + "end_header\n0 0 1e999\n", runBody + "decoder.readVertices." + ply.TagPLYInvalid, "not finite", "line", 8},
		{"a position that is nan", vertexHead + "end_header\n0 nan 0\n", runBody + "decoder.readVertices." + ply.TagPLYInvalid, "not finite", "line", 8},
		{"a normal that is not finite", "ply\nformat ascii 1.0\nelement vertex 1\nproperty float x\nproperty float y\nproperty float z\nproperty float nx\nproperty float ny\nproperty float nz\nend_header\n0 0 0 inf 0 0\n", runBody + "decoder.readVertices." + ply.TagPLYInvalid, "not finite", "line", 11},
		{"a texture coordinate that is not finite", "ply\nformat ascii 1.0\nelement vertex 1\nproperty float x\nproperty float y\nproperty float z\nproperty float s\nproperty float t\nend_header\n0 0 0 nan 0\n", runBody + "decoder.readVertices." + ply.TagPLYInvalid, "not finite", "line", 10},
		{"a face with no list of indices", vertexHead + "element face 1\nproperty int flags\nend_header\n0 0 0\n1\n", runBody + "decoder.readFaces." + ply.TagPLYInvalid, "no vertex indices", "line", 7},
		{"a lie about the count of the faces", strings.Replace(faceHead, "face 1", "face 1000000", 1) + "end_header\n0 0 0\n3 0 0 0\n", runBody + "decoder.readFaces." + ply.TagPLYInvalid, "truncated", "line", 10},
		{"an index beyond the vertices", faceHead + "end_header\n0 0 0\n3 0 0 1\n", runBody + "decoder.readFaces." + ply.TagPLYInvalid, "index out of range", "line", 11},
		{"a negative index", faceHead + "end_header\n0 0 0\n3 0 0 -1\n", runBody + "decoder.readFaces." + ply.TagPLYInvalid, "index out of range", "line", 11},
		{"an index that is not whole", faceHead + "end_header\n0 0 0\n3 0 0 0.5\n", runBody + "decoder.readFaces." + ply.TagPLYInvalid, "index out of range", "line", 11},
		{"a count that is not whole", faceHead + "end_header\n0 0 0\n2.5 0 0 0\n", runBody + "decoder.readFaces->cursor.count." + ply.TagPLYInvalid, "bad list count", "line", 11},
		{"a count beyond the limit", faceHead + "end_header\n0 0 0\n2000000 0 0\n", runBody + "decoder.readFaces->cursor.count." + ply.TagPLYInvalid, "bad list count", "line", 11},
		{"a face cut short", faceHead + "end_header\n0 0 0\n3 0 0\n", runBody + "decoder.readFaces->cursor.value." + ply.TagPLYInvalid, "truncated", "line", 12},
		{"no face of three corners", faceHead + "end_header\n0 0 0\n2 0 0\n", runBody + "decoder.finish." + ply.TagPLYInvalid, "no triangles", "line", 7},
		{"a skipped element that is cut short", vertexHead + "element tag 5\nproperty int x\nproperty int y\nend_header\n0 0 0\n1 2\n", runBody + "decoder.skipElement." + ply.TagPLYInvalid, "truncated", "line", 11},
		{"a skipped list that is cut short", vertexHead + "element tag 1\nproperty list uchar int x\nend_header\n0 0 0\n3 1 2\n", runBody + "decoder.skipElement->cursor.skip->cursor.value." + ply.TagPLYInvalid, "truncated", "line", 12},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ply.Decode(context.Background(), strings.NewReader(tt.text), ply.Options{Logger: quiet})
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
			if appErr.Metadata["reason"] != tt.reason || appErr.Metadata[tt.where] != tt.at {
				t.Errorf("reason %v at %s %v, want %q and %d; metadata %v", appErr.Metadata["reason"], tt.where, appErr.Metadata[tt.where], tt.reason, tt.at, appErr.Metadata)
			}
		})
	}
}

func TestBinaryErrors(t *testing.T) {
	const runBody = "Decode->decodeBytes->decoder.run->"
	head := "ply\nformat binary_little_endian 1.0\nelement vertex 2\nproperty float x\nproperty float y\nproperty float z\n"
	faces := "element face 1\nproperty list uchar int vertex_indices\n"
	body := func(parts ...func(w *writer)) []byte {
		w := &writer{order: binary.LittleEndian}
		for _, p := range parts {
			p(w)
		}
		return w.buf.Bytes()
	}
	text := func(s string) func(w *writer) { return func(w *writer) { w.buf.WriteString(s) } }
	vertices := func(n int) func(w *writer) {
		return func(w *writer) {
			for i := 0; i < 3*n; i++ {
				w.f32(float32(i))
			}
		}
	}
	tests := []struct {
		name   string
		data   []byte
		route  string
		reason string
		offset int
	}{
		{"vertices cut short", body(text(head+"end_header\n"), vertices(1)), runBody + "decoder.readVertices." + ply.TagPLYInvalid, "truncated", len(head) + 11},
		{"a position that is nan", body(text(head+"end_header\n"), func(w *writer) {
			w.f32(float32(math.NaN()))
			w.f32(0)
			w.f32(0)
			vertices(1)(w)
		}), runBody + "decoder.readVertices." + ply.TagPLYInvalid, "not finite", len(head) + 11 + 4},
		{"a face cut short", body(text(head+faces+"end_header\n"), vertices(2), func(w *writer) { w.u8(3); w.u32(0) }), runBody + "decoder.readFaces->cursor.value." + ply.TagPLYInvalid, "truncated", 0},
		{"an index beyond the vertices", body(text(head+faces+"end_header\n"), vertices(2), func(w *writer) {
			w.u8(3)
			w.u32(0)
			w.u32(1)
			w.u32(7)
		}), runBody + "decoder.readFaces." + ply.TagPLYInvalid, "index out of range", 0},
		{"a skipped list that is cut short", body(text(head+"element tag 1\nproperty list uchar int x\nend_header\n"), vertices(2), func(w *writer) { w.u8(200) }), runBody + "decoder.skipElement->cursor.skip." + ply.TagPLYInvalid, "truncated", 0},
		{"skipped records that are cut short", body(text(head+"element tag 10\nproperty int x\nend_header\n"), vertices(2), func(w *writer) { w.u32(1) }), runBody + "decoder.skipElement." + ply.TagPLYInvalid, "truncated", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ply.Decode(context.Background(), bytes.NewReader(tt.data), ply.Options{Logger: quiet})
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
			if appErr.Metadata["reason"] != tt.reason {
				t.Errorf("reason %v, want %q; metadata %v", appErr.Metadata["reason"], tt.reason, appErr.Metadata)
			}
			if _, ok := appErr.Metadata["offset"]; !ok {
				t.Errorf("no offset in %v", appErr.Metadata)
			}
		})
	}
}

func TestTheHeaderHasLimits(t *testing.T) {
	var many strings.Builder
	many.WriteString("ply\nformat ascii 1.0\n")
	for i := 0; i < 65; i++ {
		fmt.Fprintf(&many, "element e%d 0\n", i)
	}
	many.WriteString("end_header\n")
	var props strings.Builder
	props.WriteString("ply\nformat ascii 1.0\nelement vertex 1\n")
	for i := 0; i < 257; i++ {
		fmt.Fprintf(&props, "property float p%d\n", i)
	}
	props.WriteString("end_header\n")
	tests := []struct {
		name   string
		text   string
		reason string
	}{
		{"too many elements", many.String(), "too many elements"},
		{"too many properties", props.String(), "too many properties"},
		{"too many lines", "ply\nformat ascii 1.0\n" + strings.Repeat("comment x\n", 1<<14+1), "header too long"},
		{"too many bytes", "ply\nformat ascii 1.0\ncomment " + strings.Repeat("x", 1<<20+1) + "\nend_header\n", "header too long"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ply.Decode(context.Background(), strings.NewReader(tt.text), ply.Options{Logger: quiet})
			appErr, ok := errors.FromError(err)
			if !ok || !appErr.HasRoute("Decode->decodeBytes->parseHeader."+ply.TagPLYInvalid) || appErr.Metadata["reason"] != tt.reason {
				t.Errorf("error %v", err)
			}
		})
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
	data := everything("ascii")
	fsys := fstest.MapFS{"a.ply": {Data: data}}
	tests := []struct {
		name  string
		run   func() error
		route string
		cause error
		meta  string
	}{
		{"a reader that fails", func() error {
			_, err := ply.Decode(context.Background(), &failingReader{after: 100}, ply.Options{})
			return err
		}, "Decode->ReadAll." + model.TagModelReadFailed, io.ErrUnexpectedEOF, "offset"},
		{"a context that is done", func() error {
			_, err := ply.Decode(cancelled, bytes.NewReader(data), ply.Options{})
			return err
		}, "Decode->ReadAll." + model.TagModelReadFailed, context.Canceled, "offset"},
		{"a file that is not there", func() error {
			_, err := ply.Load(context.Background(), fstest.MapFS{}, "none.ply", ply.Options{})
			return err
		}, "Load." + model.TagModelReadFailed, os.ErrNotExist, "file"},
		{"a file in a context that is done", func() error {
			_, err := ply.Load(cancelled, fsys, "a.ply", ply.Options{})
			return err
		}, "Load->ReadAll." + model.TagModelReadFailed, context.Canceled, "file"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			appErr, ok := errors.FromError(err)
			if !ok || appErr.Tag != model.TagModelReadFailed || !appErr.HasRoute(tt.route) || !errors.Is(err, tt.cause) {
				t.Fatalf("error %v (route %v)", err, appErr)
			}
			if _, ok := appErr.Metadata[tt.meta]; !ok {
				t.Errorf("no %q in %v", tt.meta, appErr.Metadata)
			}
		})
	}
}

// bigASCII is the text of a file of n points that the context may end in the middle of.
func bigASCII(n int) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "ply\nformat ascii 1.0\nelement vertex %d\nproperty float x\nproperty float y\nproperty float z\nend_header\n", n)
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "%d 0 0\n", i)
	}
	return b.Bytes()
}

// stopAfter is a context that is done after it has been asked n times.
type stopAfter struct {
	context.Context
	n int
}

func (c *stopAfter) Err() error {
	if c.n--; c.n < 0 {
		return context.Canceled
	}
	return nil
}

func TestTheContextEndsTheWorkInTheMiddle(t *testing.T) {
	data := bigASCII(100000)
	_, err := ply.Decode(&stopAfter{Context: context.Background(), n: 3}, bytes.NewReader(data), ply.Options{})
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != model.TagModelReadFailed || !errors.Is(err, context.Canceled) || !appErr.HasRoute("decoder.readVertices."+model.TagModelReadFailed) {
		t.Fatalf("error %v", err)
	}
	if _, ok := appErr.Metadata["line"]; !ok || appErr.Metadata["element"] != "vertex" {
		t.Errorf("metadata %v", appErr.Metadata)
	}
}

func TestLoadNamesTheModelAndTheErrors(t *testing.T) {
	fsys := fstest.MapFS{"x/cloud.ply": {Data: everything("binary_little_endian")}, "x/bad.ply": {Data: []byte("ply\nformat ascii 1.0\nbad\nend_header\n")}}
	m, err := ply.Load(context.Background(), fsys, "x/cloud.ply", ply.Options{Logger: quiet})
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "cloud" || m.Nodes[0].Name != "cloud" || m.Meshes[0].Name != "cloud" {
		t.Errorf("names %q %q %q", m.Name, m.Nodes[0].Name, m.Meshes[0].Name)
	}
	m, err = ply.Load(context.Background(), fsys, "x/cloud.ply", ply.Options{Name: "given", Logger: quiet})
	if err != nil || m.Name != "given" {
		t.Errorf("name %q, error %v", m.Name, err)
	}
	_, err = ply.Load(context.Background(), fsys, "x/bad.ply", ply.Options{Logger: quiet})
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != ply.TagPLYInvalid || appErr.Metadata["file"] != "x/bad.ply" || appErr.Metadata["keyword"] != "bad" {
		t.Errorf("error %v", err)
	}
}

func TestTheSampleFilesAreRead(t *testing.T) {
	tests := []struct {
		file      string
		primitive model.Primitive
		count     int
		vertices  int
		colors    bool
	}{
		{"cube.ascii.ply", model.PrimitiveTriangles, 12, 8, true},
		{"cube.binary.ply", model.PrimitiveTriangles, 12, 8, true},
		{"cube.bigendian.ply", model.PrimitiveTriangles, 12, 8, true},
		{"cloud.ply", model.PrimitivePoints, 64, 64, true},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			m, err := ply.Load(context.Background(), os.DirFS("testdata"), tt.file, ply.Options{Logger: quiet})
			if err != nil {
				t.Fatal(err)
			}
			mesh := m.Meshes[0]
			checkMesh(t, mesh)
			if mesh.Primitive != tt.primitive || mesh.PrimitiveCount() != tt.count || mesh.VertexCount() != tt.vertices || (mesh.Colors != nil) != tt.colors {
				t.Errorf("%v of %d with %d vertices, colors %v", mesh.Primitive, mesh.PrimitiveCount(), mesh.VertexCount(), mesh.Colors != nil)
			}
		})
	}
}

func fuzzSeeds(f *testing.F) {
	for _, format := range []string{"ascii", "binary_little_endian", "binary_big_endian"} {
		f.Add(everything(format))
	}
	entries, err := os.ReadDir("testdata")
	if err != nil {
		f.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".ply") {
			data, err := os.ReadFile("testdata/" + e.Name())
			if err != nil {
				f.Fatal(err)
			}
			f.Add(data)
		}
	}
	f.Add([]byte("ply\nformat ascii 1.0\nelement vertex 1\nproperty double x\nproperty double y\nproperty double z\nend_header\n1e400 0 0\n"))
	f.Add([]byte("ply\nformat binary_little_endian 1.0\nelement vertex 2147483647\nproperty float x\nproperty float y\nproperty float z\nend_header\n"))
	f.Add([]byte("ply\nformat ascii 1.0\nelement vertex 3\nproperty float x\nproperty float y\nproperty float z\nelement face 1\nproperty list uchar int vertex_indices\nend_header\n0 0 0\n1 0 0\n0 1 0\n4294967295 0 1 2\n"))
	f.Add([]byte{})
}

func FuzzDecode(f *testing.F) {
	fuzzSeeds(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		m, err := ply.Decode(context.Background(), bytes.NewReader(data), ply.Options{Logger: quiet})
		if err != nil {
			if _, ok := errors.FromError(err); !ok {
				t.Fatalf("an error that is not an *errors.Error: %T %v", err, err)
			}
			return
		}
		if len(m.Meshes) != 1 || len(m.Nodes) != 1 {
			t.Fatalf("%d meshes and %d nodes with no error", len(m.Meshes), len(m.Nodes))
		}
		mesh := m.Meshes[0]
		checkMesh(t, mesh)
		if mesh.PrimitiveCount() == 0 {
			t.Fatal("a mesh with nothing to draw and no error")
		}
		for i := 0; i < mesh.VertexCount(); i++ {
			if p := mesh.Position(i); math.IsNaN(p.X+p.Y+p.Z) || math.IsInf(p.X+p.Y+p.Z, 0) {
				t.Fatalf("vertex %d is %v", i, p)
			}
		}
		for _, n := range mesh.Normals {
			if math.IsNaN(float64(n)) || math.IsInf(float64(n), 0) {
				t.Fatalf("a normal is %v", n)
			}
		}
	})
}

// cloudFile is a file of n points with a normal, a color and a texture coordinate each, in the format.
func cloudFile(n int, format string) []byte {
	head := fmt.Sprintf("ply\nformat %s 1.0\nelement vertex %d\nproperty float x\nproperty float y\nproperty float z\nproperty float nx\nproperty float ny\nproperty float nz\n"+
		"property uchar red\nproperty uchar green\nproperty uchar blue\nend_header\n", format, n)
	if format == "ascii" {
		var b bytes.Buffer
		b.WriteString(head)
		for i := 0; i < n; i++ {
			fmt.Fprintf(&b, "%.6f %.6f %.6f 0 0 1 %d %d %d\n", float64(i%1000)*0.01, float64(i/1000)*0.01, math.Sin(float64(i)*0.001), i%256, (i/7)%256, (i/49)%256)
		}
		return b.Bytes()
	}
	w := &writer{order: binary.LittleEndian}
	if format == "binary_big_endian" {
		w.order = binary.BigEndian
	}
	w.buf.WriteString(head)
	for i := 0; i < n; i++ {
		w.f32(float32(i%1000) * 0.01)
		w.f32(float32(i/1000) * 0.01)
		w.f32(float32(math.Sin(float64(i) * 0.001)))
		w.f32(0)
		w.f32(0)
		w.f32(1)
		w.u8(uint8(i))
		w.u8(uint8(i / 7))
		w.u8(uint8(i / 49))
	}
	return w.buf.Bytes()
}

func TestAMillionPointsAreRead(t *testing.T) {
	if testing.Short() {
		t.Skip("a million points")
	}
	for _, format := range []string{"binary_little_endian", "ascii"} {
		data := cloudFile(1000000, format)
		m := decode(t, data)
		mesh := m.Meshes[0]
		checkMesh(t, mesh)
		if mesh.PrimitiveCount() != 1000000 || mesh.Primitive != model.PrimitivePoints {
			t.Errorf("%s: %d %v", format, mesh.PrimitiveCount(), mesh.Primitive)
		}
	}
}

func BenchmarkDecode1MPoints(b *testing.B) {
	for _, format := range []string{"binary_little_endian", "binary_big_endian", "ascii"} {
		data := cloudFile(1000000, format)
		b.Run(format, func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				m, err := ply.Decode(context.Background(), bytes.NewReader(data), ply.Options{Logger: quiet})
				if err != nil {
					b.Fatal(err)
				}
				if m.Meshes[0].PrimitiveCount() != 1000000 {
					b.Fatal(m.Meshes[0].PrimitiveCount())
				}
			}
		})
	}
}
