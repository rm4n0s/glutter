package viewport3d_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/scenetest"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
)

// counter counts the notifications of a listenable.
type counter struct{ n int }

func listen(l interface {
	AddListener(foundation.VoidCallback) foundation.ListenerHandle
}) *counter {
	c := &counter{}
	l.AddListener(func() { c.n++ })
	return c
}

func translation(x, y, z float64) vecmath.Mat4 {
	return vecmath.NewMat4Translation(vecmath.Vec3{X: x, Y: y, Z: z})
}

func TestTheSceneTellsItsListenersOfEveryChange(t *testing.T) {
	box := scenetest.Box(1, 1, 1)
	mat := viewport3d.NewPhongMaterial()
	other := viewport3d.NewUnlitMaterial()
	lineMesh := scenetest.Segments([3]float32{0, 0, 0}, [3]float32{1, 0, 0})
	lines := viewport3d.NewLineMaterial()
	points := viewport3d.NewPointMaterial()
	tests := []struct {
		name   string
		change func(s *viewport3d.Scene, it *viewport3d.Item)
		want   int
	}{
		{"an item added", func(s *viewport3d.Scene, it *viewport3d.Item) { s.Add(viewport3d.NewItem(box, mat)) }, 1},
		{"an item removed", func(s *viewport3d.Scene, it *viewport3d.Item) { s.Remove(it) }, 1},
		{"the scene cleared", func(s *viewport3d.Scene, it *viewport3d.Item) { s.Clear() }, 1},
		{"a style", func(s *viewport3d.Scene, it *viewport3d.Item) { s.SetStyle(viewport3d.SceneStyle{EdgeWidth: 2}) }, 1},
		{"a transform", func(s *viewport3d.Scene, it *viewport3d.Item) { it.SetTransform(translation(1, 0, 0)) }, 1},
		{"the same transform", func(s *viewport3d.Scene, it *viewport3d.Item) { it.SetTransform(vecmath.Mat4Identity()) }, 0},
		{"a hidden item", func(s *viewport3d.Scene, it *viewport3d.Item) { it.SetVisible(false) }, 1},
		{"the same visibility", func(s *viewport3d.Scene, it *viewport3d.Item) { it.SetVisible(true) }, 0},
		{"a layer", func(s *viewport3d.Scene, it *viewport3d.Item) { it.SetLayers(4) }, 1},
		{"a pick ID", func(s *viewport3d.Scene, it *viewport3d.Item) { it.SetPickID(7) }, 1},
		{"a material", func(s *viewport3d.Scene, it *viewport3d.Item) { it.SetMaterial(other) }, 1},
		{"a color of its material", func(s *viewport3d.Scene, it *viewport3d.Item) { mat.SetColor(ui.NewColor(0xFF112233)) }, 1},
		{"the same color", func(s *viewport3d.Scene, it *viewport3d.Item) { mat.SetColor(mat.Color()) }, 0},
		{"a transparency of its material", func(s *viewport3d.Scene, it *viewport3d.Item) { mat.SetTransparent(true) }, 1},
		{"a mesh", func(s *viewport3d.Scene, it *viewport3d.Item) { it.SetMesh(scenetest.Sphere(1, 8, 6)) }, 1},
		{"a material of an item that left", func(s *viewport3d.Scene, it *viewport3d.Item) { s.Remove(it); mat.SetColor(ui.NewColor(0xFF000001)) }, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := viewport3d.NewScene()
			it := viewport3d.NewItem(box, mat)
			s.Add(it)
			c := listen(s)
			tt.change(s, it)
			if c.n != tt.want {
				t.Errorf("%d notifications, want %d", c.n, tt.want)
			}
		})
	}
	// Materials of other kinds reach the scene too.
	s := viewport3d.NewScene()
	s.Add(viewport3d.NewItem(lineMesh, lines), viewport3d.NewItem(scenetest.Points([3]float32{}), points))
	c := listen(s)
	lines.SetWidth(3)
	lines.SetStipple([]float64{4, 4})
	lines.SetColor(ui.NewColor(0xFFFFFFFF))
	points.SetSize(9)
	points.SetShape(viewport3d.PointShapeSquare)
	points.SetColor(ui.NewColor(0xFF00FF00))
	if c.n != 6 {
		t.Errorf("%d notifications of the line and point materials, want 6", c.n)
	}
}

func TestAMaterialLeftByItsLastItemIsNoLongerListenedTo(t *testing.T) {
	s := viewport3d.NewScene()
	mat := viewport3d.NewPhongMaterial()
	a, b := viewport3d.NewItem(scenetest.Box(1, 1, 1), mat), viewport3d.NewItem(scenetest.Box(1, 1, 1), mat)
	s.Add(a, b)
	c := listen(s)
	s.Remove(a)
	c.n = 0
	mat.SetColor(ui.NewColor(0xFF010203))
	if c.n != 1 {
		t.Errorf("with one item left the material made %d notifications, want 1 (the scene has one listener on it)", c.n)
	}
	s.Remove(b)
	c.n = 0
	mat.SetColor(ui.NewColor(0xFF040506))
	if c.n != 0 {
		t.Errorf("a material that no item of the scene uses made %d notifications", c.n)
	}
}

func TestSceneBoundsFollowTheVisibleItems(t *testing.T) {
	s := viewport3d.NewScene()
	if !s.Bounds().IsEmpty() {
		t.Errorf("an empty scene has bounds %v", s.Bounds())
	}
	mat := viewport3d.NewPhongMaterial()
	a := viewport3d.NewItem(scenetest.Box(2, 2, 2), mat)
	b := viewport3d.NewItem(scenetest.Box(2, 2, 2), mat, viewport3d.ItemOpts{Transform: foundation.Some(translation(10, 0, 0))})
	s.Add(a, b)
	check := func(name string, wantMinX, wantMaxX float64) {
		t.Helper()
		if got := s.Bounds(); got.Min.X != wantMinX || got.Max.X != wantMaxX {
			t.Errorf("%s: x from %v to %v, want %v to %v", name, got.Min.X, got.Max.X, wantMinX, wantMaxX)
		}
	}
	check("two boxes", -1, 11)
	b.SetVisible(false)
	check("one hidden", -1, 1)
	b.SetVisible(true)
	a.SetTransform(translation(-5, 0, 0))
	check("one moved", -6, 11)
	s.Remove(a)
	check("one removed", 9, 11)
	s.Clear()
	if !s.Bounds().IsEmpty() {
		t.Errorf("a cleared scene has bounds %v", s.Bounds())
	}
}

func TestAnItemBelongsToOneSceneAtATime(t *testing.T) {
	first, second := viewport3d.NewScene(), viewport3d.NewScene()
	it := viewport3d.NewItem(scenetest.Box(1, 1, 1), viewport3d.NewPhongMaterial())
	first.Add(it)
	first.Add(it) // twice is once
	if len(first.Items()) != 1 {
		t.Fatalf("the item is in the scene %d times", len(first.Items()))
	}
	c1, c2 := listen(first), listen(second)
	second.Add(it)
	if len(first.Items()) != 0 || len(second.Items()) != 1 {
		t.Errorf("after the move the scenes have %d and %d items", len(first.Items()), len(second.Items()))
	}
	it.SetTransform(translation(1, 0, 0))
	if c1.n != 0 || c2.n != 2 {
		t.Errorf("notifications %d and %d; the change of the item must reach the scene that has it", c1.n, c2.n)
	}
}

func TestNewItemChecksItsMeshAndItsMaterial(t *testing.T) {
	skipUnlessDebug(t)
	tris, lines, pts := scenetest.Box(1, 1, 1), scenetest.Segments([3]float32{}, [3]float32{1, 0, 0}), scenetest.Points([3]float32{})
	phong, line, point := viewport3d.NewPhongMaterial(), viewport3d.NewLineMaterial(), viewport3d.NewPointMaterial()
	tests := []struct {
		name  string
		build func()
		route string
	}{
		{"no mesh", func() { viewport3d.NewItem(nil, phong) }, "NewItem->checkItem.ItemInvalid"},
		{"no material", func() { viewport3d.NewItem(tris, nil) }, "NewItem->checkItem.ItemInvalid"},
		{"triangles with a line material", func() { viewport3d.NewItem(tris, line) }, "NewItem->checkItem.ItemInvalid"},
		{"lines with a surface material", func() { viewport3d.NewItem(lines, phong) }, "NewItem->checkItem.ItemInvalid"},
		{"points with a line material", func() { viewport3d.NewItem(pts, line) }, "NewItem->checkItem.ItemInvalid"},
		{"lines with a point material", func() { viewport3d.NewItem(lines, point) }, "NewItem->checkItem.ItemInvalid"},
		{"a new material of another primitive", func() { viewport3d.NewItem(tris, phong).SetMaterial(point) }, "Item.SetMaterial->checkItem.ItemInvalid"},
		{"a new mesh of another primitive", func() { viewport3d.NewItem(tris, phong).SetMesh(lines) }, "Item.SetMesh->checkItem.ItemInvalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantTag(t, panicOf(tt.build), viewport3d.TagItemInvalid, tt.route)
		})
	}
}

func TestAnItemTakesTheOriginOfItsMesh(t *testing.T) {
	// A mesh whose vertices are offsets from an origin 50 km away: the item's bounds are in the world, in float64.
	far := vecmath.Vec3{X: 50000, Y: -50000, Z: 30000}
	data := scenetest.Box(2, 2, 2).Data()
	data.Origin = far
	m, err := viewport3d.NewMesh(data)
	if err != nil {
		t.Fatal(err)
	}
	it := viewport3d.NewItem(m, viewport3d.NewPhongMaterial(), viewport3d.ItemOpts{Transform: foundation.Some(translation(0.001, 0, 0))})
	b := it.Bounds()
	if b.Min != (vecmath.Vec3{X: far.X - 1 + 0.001, Y: far.Y - 1, Z: far.Z - 1}) || b.Max != (vecmath.Vec3{X: far.X + 1 + 0.001, Y: far.Y + 1, Z: far.Z + 1}) {
		t.Errorf("bounds = %v", b)
	}
}
