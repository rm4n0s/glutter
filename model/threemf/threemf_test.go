package threemf_test

import (
	"archive/zip"
	"bytes"
	"context"
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
	"github.com/rm4n0s/glutter/model/threemf"
	"github.com/rm4n0s/glutter/vecmath"
)

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

const (
	coreNS     = "http://schemas.microsoft.com/3dmanufacturing/core/2015/02"
	materialNS = "http://schemas.microsoft.com/3dmanufacturing/material/2015/02"
	relsNS     = "http://schemas.openxmlformats.org/package/2006/relationships"
	modelRel   = "http://schemas.microsoft.com/3dmanufacturing/2013/01/3dmodel"
)

// archive makes a zip file of the parts (name and content).
func archive(parts map[string]string) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range parts {
		w, err := zw.Create(name)
		if err != nil {
			panic(err)
		}
		if _, err := io.WriteString(w, content); err != nil {
			panic(err)
		}
	}
	if err := zw.Close(); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// pack is a 3MF package whose model part is the XML.
func pack(modelXML string) []byte {
	return archive(map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="model" ContentType="application/vnd.ms-package.3dmanufacturing-3dmodel+xml"/></Types>`,
		"_rels/.rels":         `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="` + relsNS + `"><Relationship Target="/3D/3dmodel.model" Id="rel0" Type="` + modelRel + `"/></Relationships>`,
		"3D/3dmodel.model":    modelXML,
	})
}

// document is a model part with the namespaces, the unit and the resources and build that the test gives.
func document(unit, resources, build string) string {
	u := ""
	if unit != "" {
		u = ` unit="` + unit + `"`
	}
	return `<?xml version="1.0" encoding="UTF-8"?><model` + u + ` xmlns="` + coreNS + `" xmlns:m="` + materialNS + `" xml:lang="en-US"><metadata name="Title">t</metadata><resources>` +
		resources + `</resources><build>` + build + `</build></model>`
}

// tetra is the mesh of a tetrahedron with a corner at the origin and edges of 10 along the axes.
const tetra = `<mesh><vertices><vertex x="0" y="0" z="0"/><vertex x="10" y="0" z="0"/><vertex x="0" y="10" z="0"/><vertex x="0" y="0" z="10"/></vertices>` +
	`<triangles><triangle v1="0" v2="2" v3="1"/><triangle v1="0" v2="1" v3="3"/><triangle v1="0" v2="3" v3="2"/><triangle v1="1" v2="2" v3="3"/></triangles></mesh>`

func decode(t *testing.T, data []byte, o threemf.Options) *model.Model {
	t.Helper()
	o.Logger = quiet
	m, err := threemf.Decode(context.Background(), bytes.NewReader(data), o)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	return m
}

// checkMesh checks what every mesh has to be.
func checkMesh(t *testing.T, m *model.Mesh) {
	t.Helper()
	n := m.VertexCount()
	if len(m.Indices)%3 != 0 || m.Primitive != model.PrimitiveTriangles {
		t.Errorf("%d indices of %v", len(m.Indices), m.Primitive)
	}
	for _, i := range m.Indices {
		if int(i) >= n {
			t.Fatalf("index %d of %d vertices", i, n)
		}
	}
	if m.Colors != nil && len(m.Colors) != 4*n {
		t.Errorf("%d color bytes for %d vertices", len(m.Colors), n)
	}
	if (m.Positions == nil) == (m.Positions64 == nil) {
		t.Errorf("positions %d, positions64 %d: exactly one is set", len(m.Positions), len(m.Positions64))
	}
}

func TestATetrahedron(t *testing.T) {
	m := decode(t, pack(document("millimeter", `<object id="1" name="tetra" type="model">`+tetra+`</object>`, `<item objectid="1" partnumber="part-1"/>`)), threemf.Options{Name: "x"})
	if m.Name != "x" || len(m.Nodes) != 1 || m.Nodes[0].Name != "part-1" || len(m.Meshes) != 1 || len(m.Materials) != 0 {
		t.Fatalf("model %q with %d nodes, %d meshes and %d materials", m.Name, len(m.Nodes), len(m.Meshes), len(m.Materials))
	}
	mesh := m.Meshes[0]
	checkMesh(t, mesh)
	if m.Nodes[0].Meshes[0] != mesh || mesh.Name != "tetra" || mesh.VertexCount() != 4 || mesh.PrimitiveCount() != 4 || mesh.Material != nil || mesh.Colors != nil || mesh.Normals != nil {
		t.Errorf("mesh %q with %d vertices and %d triangles, material %v", mesh.Name, mesh.VertexCount(), mesh.PrimitiveCount(), mesh.Material)
	}
	if b := m.Bounds(); b.Min.X != 0 || b.Max.X != 10 || b.Max.Y != 10 || b.Max.Z != 10 {
		t.Errorf("bounds %v", b)
	}
	if m.Units.Name != "mm" || m.Units.MetersPerUnit != 0.001 {
		t.Errorf("units %+v", m.Units)
	}
	if got := m.TriangleCount(); got != 4 {
		t.Errorf("%d triangles", got)
	}
}

func TestUnits(t *testing.T) {
	tests := []struct {
		unit string
		name string
		m    float64
	}{
		{"", "mm", 0.001}, {"micron", "um", 1e-6}, {"millimeter", "mm", 0.001}, {"centimeter", "cm", 0.01}, {"inch", "in", 0.0254}, {"foot", "ft", 0.3048}, {"meter", "m", 1},
	}
	for _, tt := range tests {
		t.Run(tt.unit, func(t *testing.T) {
			m := decode(t, pack(document(tt.unit, `<object id="1">`+tetra+`</object>`, `<item objectid="1"/>`)), threemf.Options{})
			if m.Units.Name != tt.name || m.Units.MetersPerUnit != tt.m {
				t.Errorf("units %+v, want %s and %v", m.Units, tt.name, tt.m)
			}
		})
	}
}

func apply(m vecmath.Mat4, p vecmath.Vec3) vecmath.Vec3 { return m.TransformPoint(p) }

func near(a, b vecmath.Vec3) bool { return a.Sub(b).Length() < 1e-9 }

func TestComponentsAndTransforms(t *testing.T) {
	// Object 2 is two copies of object 1, one moved by 10 along x and one turned a quarter about z (a row vector times the matrix:
	// x' = m00 x + m10 y + m20 z + m30); object 3 holds object 2 twice, the second scaled and moved; the build has two items.
	resources := `<object id="1" name="leaf">` + tetra + `</object>` +
		`<object id="2" name="pair"><components><component objectid="1" transform="1 0 0 0 1 0 0 0 1 10 0 0"/><component objectid="1" transform="0 1 0 -1 0 0 0 0 1 0 0 0"/></components></object>` +
		`<object id="3" name="four"><components><component objectid="2"/><component objectid="2" transform="2 0 0 0 2 0 0 0 2 0 0 100"/></components></object>`
	m := decode(t, pack(document("", resources, `<item objectid="3"/><item objectid="1" transform="1 0 0 0 1 0 0 0 1 -5 6 7" partnumber="loose"/>`)), threemf.Options{})
	if len(m.Meshes) != 1 || len(m.Nodes) != 2 {
		t.Fatalf("%d meshes and %d nodes: the object that is used five times is built once", len(m.Meshes), len(m.Nodes))
	}
	four := m.Nodes[0]
	if four.Name != "four" || len(four.Children) != 2 || len(four.Children[0].Children) != 2 || four.Children[0].Children[0].Meshes[0] != m.Meshes[0] || four.Children[1].Children[1].Meshes[0] != m.Meshes[0] {
		t.Fatalf("the tree of %q is not what the file says", four.Name)
	}
	one := vecmath.Vec3{X: 1}
	if got := apply(four.Children[0].Children[0].Transform, one); !near(got, vecmath.Vec3{X: 11}) {
		t.Errorf("a moved copy puts (1,0,0) at %v", got)
	}
	if got := apply(four.Children[0].Children[1].Transform, one); !near(got, vecmath.Vec3{Y: 1}) {
		t.Errorf("a turned copy puts (1,0,0) at %v: a quarter turn counter-clockwise is (0, 1, 0)", got)
	}
	if got := apply(four.Children[1].Transform, vecmath.Vec3{X: 1, Y: 1, Z: 1}); !near(got, vecmath.Vec3{X: 2, Y: 2, Z: 102}) {
		t.Errorf("a scaled and moved pair puts (1,1,1) at %v", got)
	}
	loose := m.Nodes[1]
	if loose.Name != "loose" || !near(apply(loose.Transform, vecmath.Vec3{}), vecmath.Vec3{X: -5, Y: 6, Z: 7}) {
		t.Errorf("the item %q is at %v", loose.Name, apply(loose.Transform, vecmath.Vec3{}))
	}
	// The bounds go through every transform: the leaf's 0 to 10, moved up to 100 + 20 by the scaled pair.
	if b := m.Bounds(); b.Max.Z < 119 || b.Min.X > -5 {
		t.Errorf("bounds %v", b)
	}
}

func TestBaseMaterials(t *testing.T) {
	resources := `<basematerials id="1"><base name="red" displaycolor="#FF0000"/><base name="glass" displaycolor="#0000FF80"/></basematerials>` +
		`<object id="2" name="two" pid="1" pindex="1">` + strings.Replace(tetra, `<triangle v1="0" v2="1" v3="3"/>`, `<triangle v1="0" v2="1" v3="3" pid="1" p1="0"/>`, 1) + `</object>` +
		`<object id="3" name="three" pid="1" pindex="0">` + tetra + `</object>`
	m := decode(t, pack(document("", resources, `<item objectid="2"/><item objectid="3"/>`)), threemf.Options{})
	if len(m.Materials) != 2 || len(m.Meshes) != 3 {
		t.Fatalf("%d materials and %d meshes", len(m.Materials), len(m.Meshes))
	}
	byName := map[string]*model.Material{}
	for _, mat := range m.Materials {
		byName[mat.Name] = mat
	}
	red, glass := byName["red"], byName["glass"]
	if red == nil || glass == nil {
		t.Fatalf("materials %v", byName)
	}
	if red.BaseColor != [4]float32{1, 0, 0, 1} || red.AlphaMode != model.AlphaOpaque {
		t.Errorf("red %+v", red)
	}
	if glass.BaseColor != [4]float32{0, 0, 1, 128.0 / 255} || glass.AlphaMode != model.AlphaBlend {
		t.Errorf("glass %+v", glass)
	}
	// Object two: three triangles of the object's glass and one of red; object three: four of red. The materials are shared.
	counts := map[*model.Material]int{}
	for _, mesh := range m.Meshes {
		checkMesh(t, mesh)
		counts[mesh.Material] += mesh.PrimitiveCount()
	}
	if counts[glass] != 3 || counts[red] != 5 {
		t.Errorf("glass %d triangles and red %d, want 3 and 5", counts[glass], counts[red])
	}
	if len(m.Nodes[0].Meshes) != 2 || len(m.Nodes[1].Meshes) != 1 {
		t.Errorf("the nodes have %d and %d meshes", len(m.Nodes[0].Meshes), len(m.Nodes[1].Meshes))
	}
}

func TestColorGroups(t *testing.T) {
	// Triangle 0 has three colors (red, green, blue), triangle 1 the one color of the group's third (blue) at every corner, triangle 2 is
	// of the group's first color, and triangle 3 has none: no pid and the object has none.
	resources := `<m:colorgroup id="7"><m:color color="#FF0000"/><m:color color="#00FF00"/><m:color color="#0000FF80"/></m:colorgroup>` +
		`<object id="1"><mesh><vertices><vertex x="0" y="0" z="0"/><vertex x="1" y="0" z="0"/><vertex x="0" y="1" z="0"/><vertex x="0" y="0" z="1"/></vertices><triangles>` +
		`<triangle v1="0" v2="2" v3="1" pid="7" p1="0" p2="1" p3="2"/><triangle v1="0" v2="1" v3="3" pid="7" p1="2"/><triangle v1="0" v2="3" v3="2" pid="7" p1="0"/><triangle v1="1" v2="2" v3="3"/></triangles></mesh></object>`
	m := decode(t, pack(document("", resources, `<item objectid="1"/>`)), threemf.Options{})
	mesh := m.Meshes[0]
	checkMesh(t, mesh)
	if len(m.Materials) != 0 || mesh.Material != nil || mesh.Colors == nil || mesh.PrimitiveCount() != 4 {
		t.Fatalf("%d materials, material %v, colors %v, %d triangles", len(m.Materials), mesh.Material, mesh.Colors != nil, mesh.PrimitiveCount())
	}
	// The colors that each triangle's corners have.
	colorOf := func(tri, corner int) [4]byte {
		i := mesh.Indices[3*tri+corner]
		return [4]byte{mesh.Colors[4*i], mesh.Colors[4*i+1], mesh.Colors[4*i+2], mesh.Colors[4*i+3]}
	}
	red, green, blue, white := [4]byte{255, 0, 0, 255}, [4]byte{0, 255, 0, 255}, [4]byte{0, 0, 255, 128}, [4]byte{255, 255, 255, 255}
	want := [4][3][4]byte{{red, green, blue}, {blue, blue, blue}, {red, red, red}, {white, white, white}}
	for tri := 0; tri < 4; tri++ {
		for c := 0; c < 3; c++ {
			if got := colorOf(tri, c); got != want[tri][c] {
				t.Errorf("triangle %d corner %d has the color %v, want %v", tri, c, got, want[tri][c])
			}
		}
	}
	// No two corners with different colors share a vertex, and the corners with one color and one position do.
	seen := map[uint32][4]byte{}
	for tri := 0; tri < 4; tri++ {
		for c := 0; c < 3; c++ {
			i := mesh.Indices[3*tri+c]
			if prev, ok := seen[i]; ok && prev != colorOf(tri, c) {
				t.Errorf("vertex %d has two colors", i)
			}
			seen[i] = colorOf(tri, c)
		}
	}
	if mesh.VertexCount() >= 12 {
		t.Errorf("%d vertices for 12 corners: the ones that look the same are shared", mesh.VertexCount())
	}
}

func TestPartsThatAreNotReadAreLeftOut(t *testing.T) {
	resources := `<m:texture2d id="5" path="/3D/Texture/a.png" contenttype="image/png"/><m:texture2dgroup id="6" texid="5"><m:tex2coord u="0" v="0"/></m:texture2dgroup>` +
		`<object id="1">` + strings.Replace(tetra, `<triangle v1="0" v2="2" v3="1"/>`, `<triangle v1="0" v2="2" v3="1" pid="6" p1="0"/>`, 1) + `</object>`
	data := document("", resources, `<item objectid="1"/>`)
	data = strings.Replace(data, `<model `, `<model requiredextensions="m" `, 1)
	var logged bytes.Buffer
	m, err := threemf.Decode(context.Background(), bytes.NewReader(pack(data)), threemf.Options{Logger: slog.New(slog.NewTextHandler(&logged, nil))})
	if err != nil {
		t.Fatal(err)
	}
	if m.Meshes[0].PrimitiveCount() != 4 || len(m.Meshes) != 1 || m.Meshes[0].Colors != nil {
		t.Errorf("%d meshes, %d triangles", len(m.Meshes), m.Meshes[0].PrimitiveCount())
	}
	for _, want := range []string{"texture2d", "texture2dgroup"} {
		if !strings.Contains(logged.String(), want) {
			t.Errorf("the log does not name %q: %s", want, logged.String())
		}
	}
}

func TestTheModelPartIsFoundByTheRelationship(t *testing.T) {
	doc := document("", `<object id="1">`+tetra+`</object>`, `<item objectid="1"/>`)
	rels := func(target string) string {
		return `<?xml version="1.0"?><Relationships xmlns="` + relsNS + `"><Relationship Id="a" Type="http://example.com/other" Target="/x.txt"/><Relationship Id="b" Type="` + modelRel + `" Target="` + target + `"/></Relationships>`
	}
	tests := []struct {
		name  string
		parts map[string]string
	}{
		{"another name", map[string]string{"_rels/.rels": rels("/3D/part.model"), "3D/part.model": doc}},
		{"a name in another case", map[string]string{"_rels/.rels": rels("/3d/PART.model"), "3D/Part.model": doc}},
		{"a target without its slash", map[string]string{"_rels/.rels": rels("model.xml"), "model.xml": doc}},
		{"no relationships", map[string]string{"3D/3dmodel.model": doc}},
		{"no relationship of the type", map[string]string{"_rels/.rels": `<Relationships xmlns="` + relsNS + `"/>`, "3D/3dmodel.model": doc}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := decode(t, archive(tt.parts), threemf.Options{})
			if m.Meshes[0].PrimitiveCount() != 4 {
				t.Errorf("%d triangles", m.Meshes[0].PrimitiveCount())
			}
		})
	}
}

func TestPositionsInFloat64(t *testing.T) {
	big := `<mesh><vertices><vertex x="0.1" y="0" z="0"/><vertex x="1" y="0" z="0"/><vertex x="0" y="1" z="0"/></vertices><triangles><triangle v1="0" v2="1" v3="2"/></triangles></mesh>`
	exact := strings.Replace(big, "0.1", "0.5", 1)
	for _, tt := range []struct {
		name string
		mesh string
		wide bool
		want bool
	}{{"asked, with digits", big, true, true}, {"not asked", big, false, false}, {"asked, with exact numbers", exact, true, false}} {
		t.Run(tt.name, func(t *testing.T) {
			m := decode(t, pack(document("", `<object id="1">`+tt.mesh+`</object>`, `<item objectid="1"/>`)), threemf.Options{Float64: tt.wide})
			mesh := m.Meshes[0]
			checkMesh(t, mesh)
			if (mesh.Positions64 != nil) != tt.want {
				t.Errorf("Positions64 set is %v, want %v", mesh.Positions64 != nil, tt.want)
			}
		})
	}
}

func TestTrianglesThatRepeatAVertexAreDropped(t *testing.T) {
	doc := strings.Replace(tetra, `<triangle v1="0" v2="2" v3="1"/>`, `<triangle v1="0" v2="2" v3="2"/><triangle v1="1" v2="1" v3="1"/>`, 1)
	var logged bytes.Buffer
	m, err := threemf.Decode(context.Background(), bytes.NewReader(pack(document("", `<object id="1">`+doc+`</object>`, `<item objectid="1"/>`))), threemf.Options{Logger: slog.New(slog.NewTextHandler(&logged, nil))})
	if err != nil {
		t.Fatal(err)
	}
	if m.Meshes[0].PrimitiveCount() != 3 || !strings.Contains(logged.String(), "triangles=2") {
		t.Errorf("%d triangles, log %q", m.Meshes[0].PrimitiveCount(), logged.String())
	}
}

func TestErrors(t *testing.T) {
	const run = "Decode->decodeBytes->decoder.run->"
	obj := func(inner string) string { return `<object id="1">` + inner + `</object>` }
	item := `<item objectid="1"/>`
	good := obj(tetra)
	withModel := func(attrs, resources, build string) []byte {
		return pack(strings.Replace(document("", resources, build), `<model `, `<model `+attrs+` `, 1))
	}
	tests := []struct {
		name   string
		data   []byte
		route  string
		reason string
	}{
		{"not a zip", []byte("this is not a zip file"), "Decode->decodeBytes." + threemf.TagThreeMFInvalid, "not a zip archive"},
		{"no model part", archive(map[string]string{"readme.txt": "x"}), "Decode->decodeBytes->findModelPart." + threemf.TagThreeMFInvalid, "no model part"},
		{"a relationship to a part that is not there", archive(map[string]string{"_rels/.rels": `<Relationships xmlns="` + relsNS + `"><Relationship Id="a" Type="` + modelRel + `" Target="/gone.model"/></Relationships>`}), "findModelPart." + threemf.TagThreeMFInvalid, "no model part"},
		{"relationships that are not XML", archive(map[string]string{"_rels/.rels": "<Relationships", "3D/3dmodel.model": document("", good, item)}), "findModelPart." + threemf.TagThreeMFInvalid, "bad relationships"},
		{"XML that is cut", pack(`<?xml version="1.0"?><model xmlns="` + coreNS + `"><resources>`), run + "decoder.readModel->decoder.readResources->decoder.next." + threemf.TagThreeMFInvalid, "bad xml"},
		{"text that is not XML", pack("this is not xml"), run + "decoder.next." + threemf.TagThreeMFInvalid, "truncated"},
		{"a root that is cut in a tag", pack(`<?xml version="1.0"?><model xmlns="` + coreNS + `"><resources></resources><build></mo`), run + "decoder.readModel->decoder.readBuild->decoder.next." + threemf.TagThreeMFInvalid, "bad xml"},
		{"an empty model part", pack(""), run + "decoder.next." + threemf.TagThreeMFInvalid, "truncated"},
		{"a root that is not a model", pack(`<?xml version="1.0"?><scene xmlns="` + coreNS + `"/>`), "decoder.run." + threemf.TagThreeMFInvalid, "not a 3mf model"},
		{"a model of another namespace", pack(`<?xml version="1.0"?><model xmlns="http://example.com/x"/>`), "decoder.run." + threemf.TagThreeMFInvalid, "not a 3mf model"},
		{"a unit that is not one", pack(strings.Replace(document("", good, item), `<model `, `<model unit="furlong" `, 1)), "decoder.readModel." + threemf.TagThreeMFInvalid, "bad unit"},
		{"a required extension that is not read", withModel(`requiredextensions="p" xmlns:p="http://schemas.microsoft.com/3dmanufacturing/production/2015/06"`, good, item), "decoder.readModel." + threemf.TagThreeMFInvalid, "required extension not read"},
		{"a required extension with no namespace", withModel(`requiredextensions="q"`, good, item), "decoder.readModel." + threemf.TagThreeMFInvalid, "required extension not read"},
		{"an object with no id", pack(document("", `<object>`+tetra+`</object>`, item)), "decoder.readObject->decoder.id." + threemf.TagThreeMFInvalid, "no id"},
		{"an id that is zero", pack(document("", `<object id="0">`+tetra+`</object>`, item)), "decoder.id." + threemf.TagThreeMFInvalid, "bad id"},
		{"an id that is not a number", pack(document("", `<object id="a">`+tetra+`</object>`, item)), "decoder.id." + threemf.TagThreeMFInvalid, "bad id"},
		{"two resources with one id", pack(document("", good+`<basematerials id="1"><base name="a" displaycolor="#000000"/></basematerials>`, item)), "decoder.readBaseMaterials->decoder.id." + threemf.TagThreeMFInvalid, "id twice"},
		{"a base material with a bad color", pack(document("", `<basematerials id="2"><base name="a" displaycolor="red"/></basematerials>`+good, item)), "decoder.readBaseMaterials." + threemf.TagThreeMFInvalid, "bad color"},
		{"a base material with no color", pack(document("", `<basematerials id="2"><base name="a"/></basematerials>`+good, item)), "decoder.readBaseMaterials." + threemf.TagThreeMFInvalid, "bad color"},
		{"a group of base materials with none", pack(document("", `<basematerials id="2"></basematerials>`+good, item)), "decoder.readBaseMaterials." + threemf.TagThreeMFInvalid, "empty group"},
		{"a color group with a bad color", pack(document("", `<m:colorgroup id="2"><m:color color="#12345"/></m:colorgroup>`+good, item)), "decoder.readColorGroup." + threemf.TagThreeMFInvalid, "bad color"},
		{"an empty color group", pack(document("", `<m:colorgroup id="2"/>`+good, item)), "decoder.readColorGroup." + threemf.TagThreeMFInvalid, "empty group"},
		{"an empty object", pack(document("", `<object id="1"/>`, item)), "decoder.readObject." + threemf.TagThreeMFInvalid, "empty object"},
		{"a mesh and components", pack(document("", `<object id="1">`+tetra+`</object><object id="2">`+tetra+`<components><component objectid="1"/></components></object>`, item)), "decoder.readObject." + threemf.TagThreeMFInvalid, "mesh and components"},
		{"a pid that is not a number", pack(document("", `<object id="1" pid="x">`+tetra+`</object>`, item)), "decoder.readObject." + threemf.TagThreeMFInvalid, "bad id"},
		{"a pindex that is not a number", pack(document("", `<basematerials id="2"><base name="a" displaycolor="#000000"/></basematerials><object id="1" pid="2" pindex="x">`+tetra+`</object>`, item)), "decoder.readObject." + threemf.TagThreeMFInvalid, "bad index"},
		{"a mesh with no vertex", pack(document("", obj(`<mesh><vertices/><triangles/></mesh>`), item)), "decoder.readMesh." + threemf.TagThreeMFInvalid, "no vertices"},
		{"a mesh with no triangle", pack(document("", obj(`<mesh><vertices><vertex x="0" y="0" z="0"/></vertices><triangles/></mesh>`), item)), "decoder.readMesh." + threemf.TagThreeMFInvalid, "no triangles"},
		{"a coordinate that is not a number", pack(document("", obj(`<mesh><vertices><vertex x="0" y="a" z="0"/></vertices><triangles/></mesh>`), item)), "decoder.readVertices." + threemf.TagThreeMFInvalid, "bad number"},
		{"a coordinate that is infinite", pack(document("", obj(`<mesh><vertices><vertex x="0" y="1e999" z="0"/></vertices><triangles/></mesh>`), item)), "decoder.readVertices." + threemf.TagThreeMFInvalid, "bad number"},
		{"a coordinate that is missing", pack(document("", obj(`<mesh><vertices><vertex x="0" y="0"/></vertices><triangles/></mesh>`), item)), "decoder.readVertices." + threemf.TagThreeMFInvalid, "bad number"},
		{"a coordinate beyond float32", pack(document("", obj(`<mesh><vertices><vertex x="0" y="0" z="1e100"/></vertices><triangles/></mesh>`), item)), "decoder.readVertices." + threemf.TagThreeMFInvalid, "out of range"},
		{"a corner that is not a number", pack(document("", obj(`<mesh><vertices><vertex x="0" y="0" z="0"/></vertices><triangles><triangle v1="0" v2="a" v3="0"/></triangles></mesh>`), item)), "decoder.readTriangles." + threemf.TagThreeMFInvalid, "bad index"},
		{"a corner that is missing", pack(document("", obj(`<mesh><vertices><vertex x="0" y="0" z="0"/></vertices><triangles><triangle v1="0" v2="0"/></triangles></mesh>`), item)), "decoder.readTriangles." + threemf.TagThreeMFInvalid, "bad index"},
		{"a corner beyond the vertices", pack(document("", obj(`<mesh><vertices><vertex x="0" y="0" z="0"/></vertices><triangles><triangle v1="0" v2="1" v3="2"/></triangles></mesh>`), item)), "decoder.readTriangles." + threemf.TagThreeMFInvalid, "index out of range"},
		{"a triangle pid that is not a number", pack(document("", obj(strings.Replace(tetra, `<triangle v1="0" v2="2" v3="1"/>`, `<triangle v1="0" v2="2" v3="1" pid="q"/>`, 1)), item)), "decoder.readTriangles." + threemf.TagThreeMFInvalid, "bad id"},
		{"a p1 that is not a number", pack(document("", `<basematerials id="2"><base name="a" displaycolor="#000000"/></basematerials>`+obj(strings.Replace(tetra, `<triangle v1="0" v2="2" v3="1"/>`, `<triangle v1="0" v2="2" v3="1" pid="2" p1="q"/>`, 1)), item)), "decoder.readTriangles." + threemf.TagThreeMFInvalid, "bad index"},
		{"a p2 that is not a number", pack(document("", `<basematerials id="2"><base name="a" displaycolor="#000000"/></basematerials>`+obj(strings.Replace(tetra, `<triangle v1="0" v2="2" v3="1"/>`, `<triangle v1="0" v2="2" v3="1" pid="2" p1="0" p2="q"/>`, 1)), item)), "decoder.readTriangles." + threemf.TagThreeMFInvalid, "bad index"},
		{"a group that is not there", pack(document("", obj(strings.Replace(tetra, `<triangle v1="0" v2="2" v3="1"/>`, `<triangle v1="0" v2="2" v3="1" pid="9" p1="0"/>`, 1)), item)), "decoder.readTriangles." + threemf.TagThreeMFInvalid, "unknown property group"},
		{"a property beyond the group", pack(document("", `<basematerials id="2"><base name="a" displaycolor="#000000"/></basematerials>`+obj(strings.Replace(tetra, `<triangle v1="0" v2="2" v3="1"/>`, `<triangle v1="0" v2="2" v3="1" pid="2" p1="3"/>`, 1)), item)), "decoder.readTriangles." + threemf.TagThreeMFInvalid, "index out of range"},
		{"a component that is not defined before", pack(document("", `<object id="2"><components><component objectid="1"/></components></object>`+good, item)), "decoder.readComponents->decoder.objectRef." + threemf.TagThreeMFInvalid, "unknown object"},
		{"a component with no object", pack(document("", good+`<object id="2"><components><component/></components></object>`, item)), "decoder.readComponents->decoder.objectRef." + threemf.TagThreeMFInvalid, "bad id"},
		{"a component with a bad transform", pack(document("", good+`<object id="2"><components><component objectid="1" transform="1 0 0"/></components></object>`, item)), "decoder.readComponents." + threemf.TagThreeMFInvalid, "bad transform"},
		{"a component in another part", pack(document("", good+`<object id="2"><components><component objectid="1" p:path="/3D/o.model" xmlns:p="http://schemas.microsoft.com/3dmanufacturing/production/2015/06"/></components></object>`, `<item objectid="2"/>`)), "decoder.readComponents." + threemf.TagThreeMFInvalid, "component in another part"},
		{"an item that is not defined", pack(document("", good, `<item objectid="4"/>`)), "decoder.readBuild->decoder.objectRef." + threemf.TagThreeMFInvalid, "unknown object"},
		{"an item with a bad transform", pack(document("", good, `<item objectid="1" transform="a b c d e f g h i j k l"/>`)), "decoder.readBuild." + threemf.TagThreeMFInvalid, "bad transform"},
		{"a transform that is not finite", pack(document("", good, `<item objectid="1" transform="1 0 0 0 1 0 0 0 1 NaN 0 0"/>`)), "decoder.readBuild." + threemf.TagThreeMFInvalid, "bad transform"},
		{"no build", pack(`<?xml version="1.0"?><model xmlns="` + coreNS + `"><resources>` + good + `</resources></model>`), "decoder.run." + threemf.TagThreeMFInvalid, "no items"},
		{"a build with no item", pack(document("", good, ``)), "decoder.run." + threemf.TagThreeMFInvalid, "no items"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := threemf.Decode(context.Background(), bytes.NewReader(tt.data), threemf.Options{Logger: quiet})
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
		})
	}
}

func TestTheComponentsAreLimited(t *testing.T) {
	// Sixty-six objects, each holding the one before: the nesting is too deep.
	var deep strings.Builder
	deep.WriteString(`<object id="1">` + tetra + `</object>`)
	for i := 2; i <= 66; i++ {
		fmt.Fprintf(&deep, `<object id="%d"><components><component objectid="%d"/></components></object>`, i, i-1)
	}
	// Twenty-three objects, each holding the one before twice: 2^22 nodes, more than the build may make.
	var wide strings.Builder
	wide.WriteString(`<object id="1">` + tetra + `</object>`)
	for i := 2; i <= 23; i++ {
		fmt.Fprintf(&wide, `<object id="%d"><components><component objectid="%d"/><component objectid="%d"/></components></object>`, i, i-1, i-1)
	}
	tests := []struct {
		name      string
		resources string
		build     string
		reason    string
		route     string
	}{
		{"nested too deep", deep.String(), `<item objectid="66"/>`, "components too deep", "decoder.readComponents."},
		{"as deep as it may be", strings.Join(strings.SplitAfter(deep.String(), "</object>")[:64], ""), `<item objectid="64"/>`, "", ""},
		{"too many nodes", wide.String(), `<item objectid="23"/>`, "too many nodes", "decoder.readBuild->decoder.node->decoder.node"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := threemf.Decode(context.Background(), bytes.NewReader(pack(document("", tt.resources, tt.build))), threemf.Options{Logger: quiet})
			if tt.reason == "" {
				if err != nil || len(m.Nodes) != 1 {
					t.Fatalf("error %v", err)
				}
				return
			}
			appErr, ok := errors.FromError(err)
			if !ok || appErr.Metadata["reason"] != tt.reason || !appErr.HasRoute(tt.route) {
				t.Errorf("error %v (route %v)", err, appErr)
			}
		})
	}
}

func TestAPartThatIsTooLarge(t *testing.T) {
	// The model part is a mebibyte of blanks and a little more: the unpacked size is over the limit that the options give, though
	// the archive is small.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("3D/3dmodel.model")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(w, `<?xml version="1.0"?><model xmlns="`+coreNS+`"><resources>`+strings.Repeat(" ", 1<<20)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if buf.Len() > 4096 {
		t.Fatalf("the archive is %d bytes", buf.Len())
	}
	_, err = threemf.Decode(context.Background(), bytes.NewReader(buf.Bytes()), threemf.Options{Logger: quiet, MaxModelBytes: 1 << 19})
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Metadata["reason"] != "model part too large" || !appErr.HasRoute("decoder.next."+threemf.TagThreeMFInvalid) {
		t.Errorf("error %v", err)
	}
	// With room for it the same part is read to its end (and is not a model).
	_, err = threemf.Decode(context.Background(), bytes.NewReader(buf.Bytes()), threemf.Options{Logger: quiet, MaxModelBytes: 1 << 22})
	if appErr, ok := errors.FromError(err); !ok || appErr.Metadata["reason"] == "model part too large" {
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
	data := pack(document("", `<object id="1">`+tetra+`</object>`, `<item objectid="1"/>`))
	fsys := fstest.MapFS{"a.3mf": {Data: data}}
	tests := []struct {
		name  string
		run   func() error
		route string
		cause error
		meta  string
	}{
		{"a reader that fails", func() error {
			_, err := threemf.Decode(context.Background(), &failingReader{after: 100}, threemf.Options{})
			return err
		}, "Decode->ReadAll." + model.TagModelReadFailed, io.ErrUnexpectedEOF, "offset"},
		{"a context that is done", func() error {
			_, err := threemf.Decode(cancelled, bytes.NewReader(data), threemf.Options{})
			return err
		}, "Decode->ReadAll." + model.TagModelReadFailed, context.Canceled, "offset"},
		{"a file that is not there", func() error {
			_, err := threemf.Load(context.Background(), fstest.MapFS{}, "none.3mf", threemf.Options{})
			return err
		}, "Load." + model.TagModelReadFailed, os.ErrNotExist, "file"},
		{"a file in a context that is done", func() error {
			_, err := threemf.Load(cancelled, fsys, "a.3mf", threemf.Options{})
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

// gridModel is a model part of a height field of n by n squares: 2*n*n triangles.
func gridModel(n int) string {
	var b strings.Builder
	b.WriteString(`<object id="1" type="model"><mesh><vertices>`)
	for j := 0; j <= n; j++ {
		for i := 0; i <= n; i++ {
			fmt.Fprintf(&b, `<vertex x="%d.%03d" y="%d.%03d" z="%.6f"/>`, i, (i*37)%1000, j, (j*91)%1000, math.Sin(float64(i)*0.1)*math.Cos(float64(j)*0.13))
		}
	}
	b.WriteString(`</vertices><triangles>`)
	at := func(i, j int) int { return j*(n+1) + i }
	for j := 0; j < n; j++ {
		for i := 0; i < n; i++ {
			a, bb, c, d := at(i, j), at(i+1, j), at(i+1, j+1), at(i, j+1)
			fmt.Fprintf(&b, `<triangle v1="%d" v2="%d" v3="%d"/><triangle v1="%d" v2="%d" v3="%d"/>`, a, bb, c, a, c, d)
		}
	}
	b.WriteString(`</triangles></mesh></object>`)
	return document("millimeter", b.String(), `<item objectid="1"/>`)
}

func TestTheContextEndsTheWorkInTheMiddle(t *testing.T) {
	data := pack(gridModel(100))
	_, err := threemf.Decode(&stopAfter{Context: context.Background(), n: 2}, bytes.NewReader(data), threemf.Options{})
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != model.TagModelReadFailed || !errors.Is(err, context.Canceled) || !appErr.HasRoute("decoder.next."+model.TagModelReadFailed) {
		t.Fatalf("error %v", err)
	}
	if _, ok := appErr.Metadata["line"]; !ok || appErr.Metadata["part"] != "3D/3dmodel.model" {
		t.Errorf("metadata %v", appErr.Metadata)
	}
}

func TestLoadNamesTheModelAndTheErrors(t *testing.T) {
	good := pack(document("", `<object id="1">`+tetra+`</object>`, `<item objectid="1"/>`))
	fsys := fstest.MapFS{"x/part.3mf": {Data: good}, "x/bad.3mf": {Data: []byte("nothing")}}
	m, err := threemf.Load(context.Background(), fsys, "x/part.3mf", threemf.Options{Logger: quiet})
	if err != nil || m.Name != "part" {
		t.Fatalf("error %v", err)
	}
	m, err = threemf.Load(context.Background(), fsys, "x/part.3mf", threemf.Options{Name: "given", Logger: quiet})
	if err != nil || m.Name != "given" {
		t.Errorf("error %v", err)
	}
	_, err = threemf.Load(context.Background(), fsys, "x/bad.3mf", threemf.Options{Logger: quiet})
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != threemf.TagThreeMFInvalid || appErr.Metadata["file"] != "x/bad.3mf" {
		t.Errorf("error %v", err)
	}
}

func TestTheSampleFilesAreRead(t *testing.T) {
	tests := []struct {
		file      string
		nodes     int
		meshes    int
		materials int
		triangles int
		unit      string
	}{
		{"tetrahedron.3mf", 1, 1, 0, 4, "mm"},
		{"assembly.3mf", 2, 2, 2, 12, "in"},
		{"colors.3mf", 1, 1, 0, 2, "m"},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			m, err := threemf.Load(context.Background(), os.DirFS("testdata"), tt.file, threemf.Options{Logger: quiet})
			if err != nil {
				t.Fatal(err)
			}
			for _, mesh := range m.Meshes {
				checkMesh(t, mesh)
			}
			if len(m.Nodes) != tt.nodes || len(m.Meshes) != tt.meshes || len(m.Materials) != tt.materials || m.TriangleCount() != tt.triangles || m.Units.Name != tt.unit {
				t.Errorf("%d nodes, %d meshes, %d materials, %d triangles, unit %s", len(m.Nodes), len(m.Meshes), len(m.Materials), m.TriangleCount(), m.Units.Name)
			}
		})
	}
}

func fuzzSeeds(f *testing.F) {
	entries, err := os.ReadDir("testdata")
	if err != nil {
		f.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".3mf") {
			data, err := os.ReadFile("testdata/" + e.Name())
			if err != nil {
				f.Fatal(err)
			}
			f.Add(data)
		}
	}
	f.Add(pack(document("inch", `<object id="1">`+tetra+`</object>`, `<item objectid="1"/>`)))
	f.Add(pack(document("", `<m:colorgroup id="7"><m:color color="#FF0000"/></m:colorgroup><object id="1">`+strings.Replace(tetra, `<triangle v1="0" v2="2" v3="1"/>`, `<triangle v1="0" v2="2" v3="1" pid="7" p1="0"/>`, 1)+`</object>`, `<item objectid="1"/>`)))
	f.Add([]byte("PK\x03\x04"))
	f.Add([]byte{})
}

func FuzzDecode(f *testing.F) {
	fuzzSeeds(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		m, err := threemf.Decode(context.Background(), bytes.NewReader(data), threemf.Options{Logger: quiet, Float64: len(data)%2 == 0})
		if err != nil {
			if _, ok := errors.FromError(err); !ok {
				t.Fatalf("an error that is not an *errors.Error: %T %v", err, err)
			}
			return
		}
		if len(m.Nodes) == 0 || len(m.Meshes) == 0 {
			t.Fatalf("%d nodes and %d meshes with no error", len(m.Nodes), len(m.Meshes))
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

func TestAHeightFieldKeepsItsGrid(t *testing.T) {
	const n = 40
	m := decode(t, pack(gridModel(n)), threemf.Options{})
	mesh := m.Meshes[0]
	checkMesh(t, mesh)
	if mesh.PrimitiveCount() != 2*n*n || mesh.VertexCount() != (n+1)*(n+1) {
		t.Errorf("%d triangles, %d vertices", mesh.PrimitiveCount(), mesh.VertexCount())
	}
}

func BenchmarkDecode1M(b *testing.B) {
	// 708 by 708 squares make 1,002,528 triangles.
	data := pack(gridModel(708))
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m, err := threemf.Decode(context.Background(), bytes.NewReader(data), threemf.Options{Logger: quiet})
		if err != nil {
			b.Fatal(err)
		}
		if m.TriangleCount() != 1002528 {
			b.Fatal(m.TriangleCount())
		}
	}
}
