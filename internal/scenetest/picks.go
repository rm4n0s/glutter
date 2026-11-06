package scenetest

import (
	"fmt"
	"math"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
)

// PickCheck is a scene, a pick on it and what the pick must find (SPEC §27.8, "Picking"): the display tests of each backend run
// them, so that every backend picks the same.
type PickCheck struct {
	Name string
	Case Case
	// Ask makes the picks before the frame is drawn, and returns a function that says when their results have come and one that
	// says what is wrong with them ("" for nothing).
	Ask func(r *viewport3d.Renderer, scene *viewport3d.Scene, cam *viewport3d.Camera) (settled func() bool, verdict func() string)
}

// RunPickCheck draws the check's scene with its picks and returns what is wrong ("" when nothing is). wait is as in Render.
func RunPickCheck(d *gpu.Device, wait func(done func() bool), c PickCheck) (string, error) {
	var verdict func() string
	_, err := RenderAsking(d, wait, c.Case, func(r *viewport3d.Renderer, scene *viewport3d.Scene, cam *viewport3d.Camera) func() bool {
		var settled func() bool
		settled, verdict = c.Ask(r, scene, cam)
		return settled
	})
	if err != nil {
		return "", err
	}
	return verdict(), nil
}

// pickView is a camera of 400 x 300 pixels for a scene.
const pickW, pickH = 400, 300

func pickCase(name string, build func() (*viewport3d.Scene, *viewport3d.Camera)) Case {
	return Case{Name: name, Width: pickW, Height: pickH, Build: func(d *gpu.Device) (*viewport3d.Scene, *viewport3d.Camera, func()) {
		s, cam := build()
		cam.SetViewportSize(ui.Size{Width: pickW, Height: pickH})
		return s, cam, nil
	}}
}

func near(a, b vecmath.Vec3, eps float64) bool { return a.Sub(b).Length() <= eps }

// pickOne makes a pick at a point and checks the result with verify, which says what is wrong with it.
func pickOne(at func(cam *viewport3d.Camera) ui.Offset, verify func(res viewport3d.PickResult) string) func(*viewport3d.Renderer, *viewport3d.Scene, *viewport3d.Camera) (func() bool, func() string) {
	return func(r *viewport3d.Renderer, _ *viewport3d.Scene, cam *viewport3d.Camera) (func() bool, func() string) {
		var res viewport3d.PickResult
		var perr error
		done := false
		r.Pick(at(cam), func(p viewport3d.PickResult, err error) { res, perr, done = p, err, true })
		return func() bool { return done }, func() string {
			if perr != nil {
				return fmt.Sprintf("the pick failed: %v", perr)
			}
			return verify(res)
		}
	}
}

// pickRect makes a pick of a rectangle and checks the items it found against the wanted ones.
func pickRect(rect ui.Rect, items func() []*viewport3d.Item, want []int) func(*viewport3d.Renderer, *viewport3d.Scene, *viewport3d.Camera) (func() bool, func() string) {
	return func(r *viewport3d.Renderer, _ *viewport3d.Scene, _ *viewport3d.Camera) (func() bool, func() string) {
		var got viewport3d.PickRectResult
		var perr error
		done := false
		r.PickRect(rect, func(p viewport3d.PickRectResult, err error) { got, perr, done = p, err, true })
		return func() bool { return done }, func() string {
			if perr != nil {
				return fmt.Sprintf("the pick failed: %v", perr)
			}
			all := items()
			if len(got.Items) != len(want) {
				return fmt.Sprintf("%d items, want %d", len(got.Items), len(want))
			}
			for i, w := range want {
				if got.Items[i] != all[w] {
					return fmt.Sprintf("item %d is not the expected one", i)
				}
			}
			return ""
		}
	}
}

func centerOfView(*viewport3d.Camera) ui.Offset { return ui.Offset{Dx: pickW / 2, Dy: pickH / 2} }

func translation(x, y, z float64) foundation.Opt[vecmath.Mat4] {
	return foundation.Some(vecmath.NewMat4Translation(vecmath.Vec3{X: x, Y: y, Z: z}))
}

// PickChecks are the picks of the display tests.
func PickChecks() []PickCheck {
	var checks []PickCheck
	add := func(name string, c Case, ask func(*viewport3d.Renderer, *viewport3d.Scene, *viewport3d.Camera) (func() bool, func() string)) {
		checks = append(checks, PickCheck{Name: name, Case: c, Ask: ask})
	}

	// Each face of a cube, seen from each of the standard views: the point is on the face and the normal is the face's.
	box := Box(2, 2, 2)
	for _, v := range []struct {
		name   string
		view   viewport3d.StandardView
		normal vecmath.Vec3
		face   int
	}{
		{"front", viewport3d.StandardViewFront, vecmath.Vec3Z, 4},
		{"back", viewport3d.StandardViewBack, vecmath.Vec3Z.Neg(), 5},
		{"right", viewport3d.StandardViewRight, vecmath.Vec3X, 0},
		{"left", viewport3d.StandardViewLeft, vecmath.Vec3X.Neg(), 1},
		{"top", viewport3d.StandardViewTop, vecmath.Vec3Y, 2},
		{"bottom", viewport3d.StandardViewBottom, vecmath.Vec3Y.Neg(), 3},
	} {
		v := v
		c := pickCase("pick-"+v.name, func() (*viewport3d.Scene, *viewport3d.Camera) {
			s := viewport3d.NewScene()
			s.Add(viewport3d.NewItem(box, viewport3d.NewPhongMaterial(), viewport3d.ItemOpts{PickID: 7}))
			cam := viewport3d.NewCamera(viewport3d.CameraOpts{Up: vecmath.Vec3Y, Distance: 10})
			cam.SetBounds(s.Bounds())
			cam.SetStandardView(v.view)
			return s, cam
		})
		add("the "+v.name+" face of a cube", c, pickOne(centerOfView, func(res viewport3d.PickResult) string {
			switch {
			case !res.Hit || res.Item == nil || res.PickID != 7 || res.Primitive != viewport3d.PrimitiveTriangles:
				return fmt.Sprintf("result %+v", res)
			case !near(res.Normal, v.normal, 1e-6):
				return fmt.Sprintf("the normal is %v, want %v", res.Normal, v.normal)
			case math.Abs(res.Point.Dot(v.normal)-1) > 1e-5 || !near(res.Point, v.normal, 0.03):
				// On the face to a millionth, and within a pixel of its middle (the pixel's center is half a pixel off).
				return fmt.Sprintf("the point is %v, want %v", res.Point, v.normal)
			case res.Index/2 != v.face:
				return fmt.Sprintf("the triangle is %d, want one of face %d", res.Index, v.face)
			}
			return ""
		}))
	}

	// Nothing under the point.
	nothing := pickCase("pick-nothing", func() (*viewport3d.Scene, *viewport3d.Camera) {
		s := viewport3d.NewScene()
		s.Add(viewport3d.NewItem(Box(1, 1, 1), viewport3d.NewPhongMaterial()))
		cam := viewport3d.NewCamera(viewport3d.CameraOpts{Distance: 10})
		cam.SetBounds(s.Bounds())
		return s, cam
	})
	for _, tt := range []struct {
		name string
		at   ui.Offset
		hit  bool
	}{
		{"the middle of a view with a cube in it", ui.Offset{Dx: 200, Dy: 150}, true},
		{"a corner of the view", ui.Offset{Dx: 5, Dy: 5}, false},
		{"a point outside the view, to the left", ui.Offset{Dx: -50, Dy: 150}, false},
		{"a point far outside the view", ui.Offset{Dx: 900, Dy: 900}, false},
	} {
		tt := tt
		add("pick of "+tt.name, nothing, pickOne(func(*viewport3d.Camera) ui.Offset { return tt.at }, func(res viewport3d.PickResult) string {
			if res.Hit != tt.hit {
				return fmt.Sprintf("hit %v, want %v (%+v)", res.Hit, tt.hit, res)
			}
			return ""
		}))
	}

	// A line and a point, picked within their width.
	lines := pickCase("pick-line", func() (*viewport3d.Scene, *viewport3d.Camera) {
		s := viewport3d.NewScene()
		s.Add(viewport3d.NewItem(Segments([3]float32{-1, 0, 0}, [3]float32{0, 0, 0}, [3]float32{0, 0, 0}, [3]float32{1, 0, 0}),
			viewport3d.NewLineMaterial(viewport3d.LineOpts{Width: 2}), viewport3d.ItemOpts{PickID: 3}))
		s.Add(viewport3d.NewItem(Points([3]float32{0, 0.8, 0}, [3]float32{0.5, 0.8, 0}), viewport3d.NewPointMaterial(viewport3d.PointOpts{Size: 8}), viewport3d.ItemOpts{PickID: 4}))
		cam := viewport3d.NewCamera(viewport3d.CameraOpts{Distance: 6})
		cam.SetBounds(s.Bounds())
		return s, cam
	})
	over := func(p vecmath.Vec3, dx, dy float64) func(*viewport3d.Camera) ui.Offset {
		return func(cam *viewport3d.Camera) ui.Offset {
			o, _ := cam.Project(p)
			return ui.Offset{Dx: o.Dx + dx, Dy: o.Dy + dy}
		}
	}
	for _, tt := range []struct {
		name  string
		at    func(*viewport3d.Camera) ui.Offset
		hit   bool
		id    uint32
		prim  viewport3d.Primitive
		index int
		point vecmath.Vec3
	}{
		{"on the second segment of a line", over(vecmath.Vec3{X: 0.5}, 0, 0), true, 3, viewport3d.PrimitiveLines, 1, vecmath.Vec3{X: 0.5}},
		{"on the first segment of a line", over(vecmath.Vec3{X: -0.5}, 0, 0), true, 3, viewport3d.PrimitiveLines, 0, vecmath.Vec3{X: -0.5}},
		{"1.5 pixels from a line of 2 pixels", over(vecmath.Vec3{X: 0.5}, 0, 1.5), true, 3, viewport3d.PrimitiveLines, 1, vecmath.Vec3{X: 0.5}},
		{"6 pixels from a line of 2 pixels", over(vecmath.Vec3{X: 0.5}, 0, 6), false, 0, 0, 0, vecmath.Vec3{}},
		{"the second point of a mesh of points", over(vecmath.Vec3{X: 0.5, Y: 0.8}, 0, 0), true, 4, viewport3d.PrimitivePoints, 1, vecmath.Vec3{X: 0.5, Y: 0.8}},
		{"3 pixels from the first point", over(vecmath.Vec3{Y: 0.8}, 3, 0), true, 4, viewport3d.PrimitivePoints, 0, vecmath.Vec3{Y: 0.8}},
	} {
		tt := tt
		add("pick of "+tt.name, lines, pickOne(tt.at, func(res viewport3d.PickResult) string {
			switch {
			case res.Hit != tt.hit:
				return fmt.Sprintf("hit %v, want %v (%+v)", res.Hit, tt.hit, res)
			case !tt.hit:
				return ""
			case res.PickID != tt.id || res.Primitive != tt.prim || res.Index != tt.index:
				return fmt.Sprintf("pick ID %d, %v %d; want %d, %v %d", res.PickID, res.Primitive, res.Index, tt.id, tt.prim, tt.index)
			case res.Normal != (vecmath.Vec3{}):
				return fmt.Sprintf("a line or a point has no normal, got %v", res.Normal)
			case !near(res.Point, tt.point, 0.08):
				return fmt.Sprintf("the point is %v, want near %v", res.Point, tt.point)
			}
			return ""
		}))
	}

	// A part 50 km from the origin: the point is rebuilt from the depth in float64.
	far := pickCase("pick-far", func() (*viewport3d.Scene, *viewport3d.Camera) {
		data := Box(1, 1, 1).Data()
		data.Origin = FarOrigin
		mesh := mustMesh(data)
		s := viewport3d.NewScene()
		s.Add(viewport3d.NewItem(mesh, viewport3d.NewPhongMaterial()))
		cam := viewport3d.NewCamera(viewport3d.CameraOpts{Target: FarOrigin, Distance: 4})
		cam.SetBounds(s.Bounds())
		return s, cam
	})
	add("pick of a part 50 km from the origin", far, pickOne(centerOfView, func(res viewport3d.PickResult) string {
		// The face is at z + 0.5 to a hundred-thousandth, and the pixel's center is half a pixel (0.006) from the middle of it.
		want := FarOrigin.Add(vecmath.Vec3{Z: 0.5})
		if !res.Hit || math.Abs(res.Point.Z-want.Z) > 1e-5 || !near(res.Point, want, 0.01) || !near(res.Normal, vecmath.Vec3Z, 1e-6) {
			return fmt.Sprintf("the point is %v and the normal %v, want %v and +z (%+v)", res.Point, res.Normal, want, res)
		}
		return ""
	}))

	// A box selection of three cubes in a row.
	var row []*viewport3d.Item
	cubes := pickCase("pick-rect", func() (*viewport3d.Scene, *viewport3d.Camera) {
		s := viewport3d.NewScene()
		row = nil
		for i := -1; i <= 1; i++ {
			it := viewport3d.NewItem(Box(1, 1, 1), viewport3d.NewPhongMaterial(), viewport3d.ItemOpts{Transform: translation(float64(i)*2, 0, 0)})
			row = append(row, it)
			s.Add(it)
		}
		cam := viewport3d.NewCamera(viewport3d.CameraOpts{Distance: 12})
		cam.SetBounds(s.Bounds())
		return s, cam
	})
	items := func() []*viewport3d.Item { return row }
	for _, tt := range []struct {
		name string
		rect ui.Rect
		want []int
	}{
		{"the whole view", ui.Rect{Left: 0, Top: 0, Right: pickW, Bottom: pickH}, []int{0, 1, 2}},
		{"the left half of the view", ui.Rect{Left: 0, Top: 0, Right: 200, Bottom: pickH}, []int{0, 1}},
		{"the right half of the view", ui.Rect{Left: 200, Top: 0, Right: pickW, Bottom: pickH}, []int{1, 2}},
		{"the middle cube only", ui.Rect{Left: 185, Top: 135, Right: 215, Bottom: 165}, []int{1}},
		{"the top of the view, where there is nothing", ui.Rect{Left: 0, Top: 0, Right: pickW, Bottom: 40}, nil},
	} {
		add("box selection of "+tt.name, cubes, pickRect(tt.rect, items, tt.want))
	}

	// What is in front hides what is behind, and what a section plane cuts away is not picked.
	stacked := func(name string, planes ...vecmath.Plane) Case {
		c := pickCase(name, func() (*viewport3d.Scene, *viewport3d.Camera) {
			s := viewport3d.NewScene()
			mat := viewport3d.NewPhongMaterial()
			s.Add(viewport3d.NewItem(Box(1, 1, 1), mat, viewport3d.ItemOpts{PickID: 1, Transform: translation(0, 0, -2)}))
			s.Add(viewport3d.NewItem(Box(1, 1, 1), mat, viewport3d.ItemOpts{PickID: 2, Transform: translation(0, 0, 2)}))
			s.SetSectionPlanes(planes...)
			cam := viewport3d.NewCamera(viewport3d.CameraOpts{Distance: 10})
			cam.SetBounds(s.Bounds())
			return s, cam
		})
		if len(planes) > 0 {
			c.Depth = gpu.TextureFormatDepth24PlusStencil8 // the caps of the section need a stencil
		}
		return c
	}
	add("pick of two cubes, one in front of the other", stacked("pick-front"), pickOne(centerOfView, func(res viewport3d.PickResult) string {
		if !res.Hit || res.PickID != 2 || math.Abs(res.Point.Z-2.5) > 1e-4 {
			return fmt.Sprintf("the nearer cube should be picked at z 2.5: %+v", res)
		}
		return ""
	}))
	add("pick of two cubes with the nearer cut away", stacked("pick-cut", vecmath.Plane{Normal: vecmath.Vec3Z}), pickOne(centerOfView, func(res viewport3d.PickResult) string {
		if !res.Hit || res.PickID != 1 {
			return fmt.Sprintf("the cube behind the plane should be picked: %+v", res)
		}
		return ""
	}))
	return checks
}
