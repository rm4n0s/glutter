package viewport3d_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"log/slog"
	"math"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/model"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
)

func pngBytes(t *testing.T, w, h int, c color.NRGBA) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// triangleMesh is a mesh of one triangle at the origin.
func triangleMesh(name string) *model.Mesh {
	return &model.Mesh{
		Name: name, Positions: []float32{0, 0, 0, 1, 0, 0, 0, 1, 0}, Normals: []float32{0, 0, 1, 0, 0, 1, 0, 0, 1},
		TexCoords: []float32{0, 0, 1, 0, 0, 1}, Indices: []uint32{0, 1, 2},
	}
}

func TestUploadMakesItemsMeshesMaterialsAndTextures(t *testing.T) {
	rig := newDrawRig(t, viewport3d.NewScene(), frontCamera())
	dev := rig.frame.Device
	tex := &model.Image{Name: "px.png", Type: "image/png", Data: pngBytes(t, 4, 2, color.NRGBA{R: 200, G: 100, B: 50, A: 255})}
	red := &model.Material{Name: "red", BaseColor: [4]float32{1, 0, 0, 1}, Specular: [3]float32{1, 1, 1}, Shininess: 100}
	glass := &model.Material{Name: "glass", BaseColor: [4]float32{0, 0, 1, 0.5}, AlphaMode: model.AlphaBlend, BaseColorImage: tex, DoubleSided: true}
	a, b := triangleMesh("a"), triangleMesh("b")
	a.Material, b.Material = red, glass
	lines := &model.Mesh{Primitive: model.PrimitiveLines, Positions: []float32{0, 0, 0, 1, 1, 1}, Indices: []uint32{0, 1}, Material: red}
	points := &model.Mesh{Primitive: model.PrimitivePoints, Positions: []float32{0, 0, 0}, Indices: []uint32{0}}
	colored := triangleMesh("colored")
	colored.Colors = []byte{255, 0, 0, 255, 0, 255, 0, 255, 0, 0, 255, 255}
	empty := &model.Mesh{Name: "empty"}
	// A mesh 50 km from the origin, in float64.
	far := &model.Mesh{Positions64: []float64{5e7 + 0.001, 0, 0, 5e7 + 1.001, 0, 0, 5e7 + 0.001, 1, 0}, Indices: []uint32{0, 1, 2}}

	root := model.NewNode("root")
	root.Transform = vecmath.NewMat4Translation(vecmath.Vec3{X: 10})
	child := model.NewNode("child")
	child.Transform = vecmath.NewMat4Translation(vecmath.Vec3{Y: 5})
	child.Meshes = []*model.Mesh{a, b, empty}
	other := model.NewNode("other")
	other.Meshes = []*model.Mesh{a, lines, points, colored, far}
	root.Children = []*model.Node{child}
	root.Meshes = []*model.Mesh{a}
	m := &model.Model{Nodes: []*model.Node{root, other}}

	before := rig.rec.Device().Live()
	u, err := viewport3d.Upload(context.Background(), dev, m, viewport3d.UploadOpts{FirstPickID: 100})
	if err != nil {
		t.Fatal(err)
	}
	// root: a; child: a, b (empty is skipped); other: a, lines, points, colored, far.
	if len(u.Parts) != 8 || len(u.Items()) != 8 {
		t.Fatalf("%d parts", len(u.Parts))
	}
	// The mesh of a model mesh is shared by its items.
	if u.Parts[0].Item.Mesh() != u.Parts[1].Item.Mesh() || u.Parts[0].Item.Mesh() != u.Parts[3].Item.Mesh() || u.Parts[0].Item.Mesh() == u.Parts[2].Item.Mesh() {
		t.Error("the items of a mesh do not share it")
	}
	if u.Parts[0].Item.Material() != u.Parts[1].Item.Material() {
		t.Error("the items of a mesh do not share its material")
	}
	// The transforms are the products down the tree.
	wantT := []vecmath.Vec3{{X: 10}, {X: 10, Y: 5}, {X: 10, Y: 5}, {}, {}, {}, {}, {}}
	for i, w := range wantT {
		if got := u.Parts[i].Item.Transform().Translation(); got != w {
			t.Errorf("part %d at %v, want %v", i, got, w)
		}
	}
	for i, p := range u.Parts {
		if p.Item.PickID() != uint32(100+i) {
			t.Errorf("part %d has the pick id %d", i, p.Item.PickID())
		}
		if p.Node == nil || p.Mesh == nil {
			t.Errorf("part %d has no source", i)
		}
	}
	if u.Parts[0].Node != root || u.Parts[1].Node != child || u.Parts[2].Mesh != b || u.Parts[3].Node != other {
		t.Error("the sources of the parts")
	}
	// The materials, by primitive.
	phong, ok := u.Parts[0].Item.Material().(*viewport3d.PhongMaterial)
	if !ok || phong.Color() != ui.NewColorFromARGB(255, 255, 0, 0) || phong.Shininess() != 100 || phong.Texture() != nil || phong.Transparent() || phong.DoubleSided() {
		t.Errorf("red material %#v", u.Parts[0].Item.Material())
	}
	glassM, ok := u.Parts[2].Item.Material().(*viewport3d.PhongMaterial)
	if !ok || !glassM.Transparent() || !glassM.DoubleSided() || glassM.Texture() == nil || glassM.Color().A < 0.49 || glassM.Color().A > 0.51 {
		t.Errorf("glass material %#v", u.Parts[2].Item.Material())
	}
	if _, ok := u.Parts[4].Item.Material().(*viewport3d.LineMaterial); !ok {
		t.Errorf("lines have %T", u.Parts[4].Item.Material())
	}
	if _, ok := u.Parts[5].Item.Material().(*viewport3d.PointMaterial); !ok {
		t.Errorf("points have %T", u.Parts[5].Item.Material())
	}
	if !u.Parts[6].Item.Material().(*viewport3d.PhongMaterial).VertexColors() {
		t.Error("a mesh with colors has a material without vertex colors")
	}
	if u.Parts[3].Item.Mesh().Primitive() != viewport3d.PrimitiveTriangles || u.Parts[4].Item.Mesh().Primitive() != viewport3d.PrimitiveLines ||
		u.Parts[5].Item.Mesh().Primitive() != viewport3d.PrimitivePoints {
		t.Error("primitives")
	}
	// The mesh in float64 has its origin in float64 and small float32 positions.
	farMesh := u.Parts[7].Item.Mesh()
	if o := farMesh.Origin(); math.Abs(o.X-(5e7+0.501)) > 1e-6 || math.Abs(o.Y-0.5) > 1e-9 {
		t.Errorf("origin %v", o)
	}
	for _, p := range farMesh.Data().Positions {
		if math.Abs(float64(p)) > 1 {
			t.Errorf("a position of %v relative to the origin", p)
		}
	}
	if b := farMesh.Bounds(); math.Abs(b.Min.X-(5e7+0.001)) > 1e-6 || math.Abs(b.Max.X-(5e7+1.001)) > 1e-6 {
		t.Errorf("bounds %v", b)
	}
	if got := u.Bounds; got.IsEmpty() || got.Max.X < 5e7 {
		t.Errorf("bounds of the model %v", got)
	}
	// One texture, shared, and the model's bounds.
	if live := rig.rec.Device().Live(); live.Textures != before.Textures+1 {
		t.Errorf("%d textures alive, want %d", live.Textures, before.Textures+1)
	}

	// The items draw in a scene.
	rig.scene.Add(u.Items()...)
	calls := rig.redraw()
	if len(rig.draws(calls)) == 0 {
		t.Errorf("nothing drawn: %v", calls)
	}
	rig.scene.Remove(u.Items()...)
	rig.redraw()
	u.Release()
	if len(u.Parts) != 0 {
		t.Error("Release left the parts")
	}
	if live := rig.rec.Device().Live(); live.Textures > before.Textures {
		t.Errorf("%d textures alive after Release, want %d", live.Textures, before.Textures)
	}
}

func TestUploadErrors(t *testing.T) {
	rig := newDrawRig(t, viewport3d.NewScene(), frontCamera())
	dev := rig.frame.Device
	good := triangleMesh("good")
	bad := triangleMesh("bad")
	bad.Indices = []uint32{0, 1, 7}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	nodeOf := func(meshes ...*model.Mesh) *model.Model {
		n := model.NewNode("n")
		n.Meshes = meshes
		return &model.Model{Nodes: []*model.Node{n}}
	}
	before := rig.rec.Device().Live()
	tests := []struct {
		name  string
		ctx   context.Context
		model *model.Model
		tag   string
		route string
	}{
		{"an index outside the vertices", context.Background(), nodeOf(good, bad), viewport3d.TagMeshInvalid, "newMesh->NewMesh->Mesh.validate.MeshInvalid"},
		{"a context that is done", cancelled, nodeOf(good), viewport3d.TagUploadFailed, "Upload->Upload.func"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := viewport3d.Upload(tt.ctx, dev, tt.model)
			if u != nil {
				t.Error("a result with an error")
			}
			wantTag(t, err, tt.tag, tt.route)
		})
	}
	if live := rig.rec.Device().Live(); live != before {
		t.Errorf("resources alive after the failures: %+v, before %+v", live, before)
	}
}

func TestUploadLeavesOutAnImageThatCannotBeDecoded(t *testing.T) {
	rig := newDrawRig(t, viewport3d.NewScene(), frontCamera())
	var logged strings.Builder
	logger := slog.New(slog.NewTextHandler(&logged, nil))
	mesh := triangleMesh("m")
	mesh.Material = &model.Material{Name: "m", BaseColorImage: &model.Image{Name: "broken.png", Data: []byte("not an image")}}
	n := model.NewNode("n")
	n.Meshes = []*model.Mesh{mesh}
	u, err := viewport3d.Upload(context.Background(), rig.frame.Device, &model.Model{Nodes: []*model.Node{n}}, viewport3d.UploadOpts{Logger: logger})
	if err != nil {
		t.Fatal(err)
	}
	defer u.Release()
	if tex := u.Parts[0].Item.Material().(*viewport3d.PhongMaterial).Texture(); tex != nil {
		t.Error("a material has a texture of a broken image")
	}
	if !strings.Contains(logged.String(), "broken.png") {
		t.Errorf("log %q", logged.String())
	}
}
