package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/model"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
)

const frame = 16 * time.Millisecond

// newViewer starts the app on a tester with a recording GPU device and returns its state.
func newViewer(t *testing.T) (*gluttertest.Tester, *cadState) {
	t.Helper()
	ts := gluttertest.New(t, gluttertest.Opts{GPU: true})
	ts.SetViewSize(ui.Size{Width: 1100, Height: 680})
	ts.PumpWidget(NewCadApp())
	ts.Pump(frame)
	ts.Pump(frame)
	return ts, gluttertest.StateOf[*cadState](ts, gluttertest.ByType[*CadPage]())
}

func TestTheViewerShowsTheBuiltInPart(t *testing.T) {
	ts, s := newViewer(t)
	if s.up == nil || len(s.scene.Items()) != 4 || len(s.up.Parts) != 4 {
		t.Fatalf("the part has %d items", len(s.scene.Items()))
	}
	if !strings.Contains(s.status, "the built-in part") || s.mdl.TriangleCount() == 0 {
		t.Errorf("status %q", s.status)
	}
	// The grid is under the part, and it is a good size for it.
	grid := s.scene.Style().Grid
	if !grid.Show || grid.Origin.Z != 0 || grid.Normal != vecmath.Vec3Z || grid.Spacing != 5 || grid.MajorEvery != 5 {
		t.Errorf("grid %+v", grid)
	}
	if s.scene.Style().DisplayMode != viewport3d.DisplayModeShadedWithEdges {
		t.Errorf("display mode %v", s.scene.Style().DisplayMode)
	}
	// The camera looks at the middle of the part from the isometric view, up being +z.
	if c := s.camera.Target(); c.Sub(s.up.Bounds.Center()).Length() > 1e-6 {
		t.Errorf("the camera looks at %v, the part's middle is %v", c, s.up.Bounds.Center())
	}
	for _, text := range []string{"Open…", "Sample", "Part", "Shaded", "Edges", "Wire", "Hidden", "Cut", "Measure"} {
		// A page has no file dialog: the web build has the sample and the part only.
		if text == "Open…" && foundation.IsWeb {
			continue
		}
		if len(ts.Elements(gluttertest.ByText(text))) == 0 {
			t.Errorf("no %q in the panel", text)
		}
	}
}

func TestTheDisplayModeButtonsChangeTheStyleOfTheScene(t *testing.T) {
	ts, s := newViewer(t)
	tests := []struct {
		button string
		want   viewport3d.DisplayMode
	}{
		{"Wire", viewport3d.DisplayModeWireframe},
		{"Hidden", viewport3d.DisplayModeHiddenLine},
		{"Shaded", viewport3d.DisplayModeShaded},
		{"Edges", viewport3d.DisplayModeShadedWithEdges},
	}
	for _, tt := range tests {
		ts.Tap(gluttertest.ByText(tt.button))
		ts.Pump(frame)
		if got := s.scene.Style().DisplayMode; got != tt.want || s.mode != tt.want {
			t.Errorf("after %q the scene draws %v, want %v", tt.button, got, tt.want)
		}
		// A change of mode keeps the grid.
		if !s.scene.Style().Grid.Show {
			t.Errorf("after %q the grid is gone", tt.button)
		}
	}
}

func TestTheSectionPlaneFollowsTheSwitchTheAxisAndTheSlider(t *testing.T) {
	ts, s := newViewer(t)
	if len(s.scene.SectionPlanes()) != 0 {
		t.Fatal("a plane before the switch is on")
	}
	ts.Tap(gluttertest.ByText("Cut"))
	ts.Pump(frame)
	planes := s.scene.SectionPlanes()
	if len(planes) != 1 || !s.sectionOn {
		t.Fatalf("%d planes after the switch", len(planes))
	}
	center := s.up.Bounds.Center()
	// In the middle of the part, perpendicular to x: what is on the +x side is cut away.
	if d := planes[0].DistanceTo(center); d > 1e-9 || d < -1e-9 {
		t.Errorf("the plane is %v from the middle of the part", d)
	}
	if n := planes[0].Normal; n != vecmath.Vec3X {
		t.Errorf("normal %v", n)
	}
	if planes[0].DistanceTo(vecmath.Vec3{X: center.X + 10, Y: center.Y, Z: center.Z}) <= 0 {
		t.Error("a point on the +x side is not cut away")
	}
	ts.Tap(gluttertest.ByText("Z"))
	ts.Pump(frame)
	if n := s.scene.SectionPlanes()[0].Normal; n != vecmath.Vec3Z {
		t.Errorf("normal %v after the Z button", n)
	}
	// The slider moves the plane from the low end of the part to the high one.
	before := s.sectionAt
	ts.Drag(gluttertest.ByType[*m.Slider](), ui.Offset{Dx: 60})
	ts.Pump(frame)
	if s.sectionAt <= before {
		t.Errorf("the slider is at %v after a drag to the right, was %v", s.sectionAt, before)
	}
	z := s.up.Bounds.Min.Z + s.sectionAt*(s.up.Bounds.Max.Z-s.up.Bounds.Min.Z)
	if d := s.scene.SectionPlanes()[0].DistanceTo(vecmath.Vec3{X: center.X, Y: center.Y, Z: z}); d > 1e-9 || d < -1e-9 {
		t.Errorf("the plane is %v from where the slider says", d)
	}
	ts.Tap(gluttertest.ByText("Cut"))
	ts.Pump(frame)
	if len(s.scene.SectionPlanes()) != 0 {
		t.Error("the plane stays after the switch is off")
	}
}

func TestPicksShowTheItemAndMeasuresShowTheDistance(t *testing.T) {
	ts, s := newViewer(t)
	boss := s.up.Parts[1]
	if boss.Node.Name != "boss" {
		t.Fatalf("part 1 is %q", boss.Node.Name)
	}
	hit := func(item *viewport3d.Item, p vecmath.Vec3) viewport3d.PickResult {
		return viewport3d.PickResult{Hit: true, Item: item, PickID: item.PickID(), Primitive: viewport3d.PrimitiveTriangles, Index: 7, Point: p, Normal: vecmath.Vec3Z}
	}
	post := s.up.Parts[2].Item

	// A pick selects the item and tells what was hit.
	s.SetState(func() { s.showPick(hit(boss.Item, vecmath.Vec3{X: -20, Y: 12, Z: 55}), nil) })
	ts.Pump(frame)
	if sel := s.scene.Selection(); len(sel) != 1 || sel[0] != boss.Item {
		t.Errorf("selection %v", sel)
	}
	for _, want := range []string{"Picked boss", "triangle 7", "point (-20.000, 12.000, 55.000)", "normal (0.00, 0.00, 1.00)"} {
		if !strings.Contains(s.picked, want) {
			t.Errorf("the pick says %q, want %q in it", s.picked, want)
		}
	}
	if len(ts.Elements(gluttertest.ByTextContaining("Picked boss"))) == 0 {
		t.Error("the panel does not show the pick")
	}
	// Nothing hit, and a pick that failed.
	s.SetState(func() { s.showPick(viewport3d.PickResult{}, nil) })
	if len(s.scene.Selection()) != 0 || s.picked != "Nothing there." {
		t.Errorf("a miss: %q, selection %v", s.picked, s.scene.Selection())
	}
	s.SetState(func() { s.showPick(viewport3d.PickResult{}, errors.New(tagCadLoadFailed, "a pick failed")) })
	if !strings.Contains(s.picked, "The pick failed") {
		t.Errorf("a failed pick: %q", s.picked)
	}

	// Measure: two picks make the dimension, a third starts again.
	ts.Tap(gluttertest.ByText("Measure"))
	ts.Pump(frame)
	if !s.measure {
		t.Fatal("measure is off after the switch")
	}
	a, b := vecmath.Vec3{X: 30, Y: -18, Z: 34}, vecmath.Vec3{X: 30, Y: 18, Z: 34}
	s.SetState(func() { s.showPick(hit(post, a), nil) })
	ts.Pump(frame)
	if len(s.marks) != 1 || len(ts.Elements(gluttertest.ByText("Pick the second point."))) == 0 {
		t.Errorf("after one pick: %d marks", len(s.marks))
	}
	s.SetState(func() { s.showPick(hit(post, b), nil) })
	ts.Pump(frame)
	if d, label := s.distance(); d != 36 || label != "36.00 mm" {
		t.Errorf("distance %v %q", d, label)
	}
	for _, text := range []string{"Distance 36.00 mm", "36.00 mm"} {
		if len(ts.Elements(gluttertest.ByText(text))) == 0 {
			t.Errorf("no %q on the screen", text)
		}
	}
	if len(s.scene.Selection()) != 0 && s.scene.Selection()[0] == post {
		t.Error("a pick in measure mode selected an item")
	}
	s.SetState(func() { s.showPick(hit(post, a), nil) })
	if len(s.marks) != 1 {
		t.Errorf("a third pick left %d marks", len(s.marks))
	}
	// Turning measure off clears the marks.
	ts.Tap(gluttertest.ByText("Measure"))
	ts.Pump(frame)
	if len(s.marks) != 0 || s.measure {
		t.Error("measure left marks")
	}
}

func TestTheDimensionPainterRepaintsOnlyWhenTheMarksChange(t *testing.T) {
	camera := viewport3d.NewCamera()
	a, b := vecmath.Vec3{X: 1}, vecmath.Vec3{Y: 1}
	p := newDimensionPainter(camera, []vecmath.Vec3{a, b})
	tests := []struct {
		name string
		old  *dimensionPainter
		want bool
	}{
		{"the same marks", newDimensionPainter(camera, []vecmath.Vec3{a, b}), false},
		{"another mark", newDimensionPainter(camera, []vecmath.Vec3{a, a}), true},
		{"one mark fewer", newDimensionPainter(camera, []vecmath.Vec3{a}), true},
		{"another camera", newDimensionPainter(viewport3d.NewCamera(), []vecmath.Vec3{a, b}), true},
	}
	for _, tt := range tests {
		if got := p.ShouldRepaint(tt.old); got != tt.want {
			t.Errorf("%s: ShouldRepaint = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestTheSampleAndThePartReplaceEachOther(t *testing.T) {
	ts, s := newViewer(t)
	ts.Tap(gluttertest.ByText("Sample"))
	ts.Pump(frame)
	if len(s.scene.Items()) != 1 || !strings.Contains(s.status, "the sample gear") || !strings.Contains(s.status, "1152 triangles") {
		t.Errorf("after Sample: %d items, status %q", len(s.scene.Items()), s.status)
	}
	if s.mdl.Units.Name != "mm" {
		t.Errorf("units %+v", s.mdl.Units)
	}
	// The camera fits the gear, which is 48 mm wide and 8 mm thick.
	if size := s.up.Bounds.Size(); size.X < 47 || size.X > 49 || size.Z < 7.9 || size.Z > 8.1 {
		t.Errorf("the gear is %v", size)
	}
	if c := s.camera.Target(); c.Sub(s.up.Bounds.Center()).Length() > 1e-6 {
		t.Errorf("the camera looks at %v", c)
	}
	if g := s.scene.Style().Grid; g.Spacing != 2 {
		t.Errorf("grid spacing %v for a gear of 48 mm", g.Spacing)
	}
	ts.Tap(gluttertest.ByText("Part"))
	ts.Pump(frame)
	if len(s.scene.Items()) != 4 || !strings.Contains(s.status, "the built-in part") {
		t.Errorf("after Part: %d items, status %q", len(s.scene.Items()), s.status)
	}
	// Loading a model forgets the marks, the selection and the measure.
	s.SetState(func() { s.measure, s.marks = true, []vecmath.Vec3{{}, {X: 1}} })
	ts.Tap(gluttertest.ByText("Sample"))
	ts.Pump(frame)
	if s.measure || len(s.marks) != 0 {
		t.Error("a new model keeps the marks of the old")
	}
}

func TestTheDeviceIsFreedWhenThePageGoes(t *testing.T) {
	ts, s := newViewer(t)
	rec := ts.GPUDevice()
	if live := rec.Device().Live(); live.Buffers == 0 {
		t.Fatalf("nothing alive while the part is shown: %+v", live)
	}
	ts.PumpWidget(m.NewSizedBox(nil))
	ts.Pump(frame)
	if live := rec.Device().Live(); live.Buffers != 0 || live.Textures != 0 || live.Pipelines != 0 || live.Samplers != 0 {
		t.Errorf("after the page is gone: %+v", live)
	}
	if s.up != nil {
		t.Error("the page keeps what Upload made")
	}
}

func TestWithoutAGPUTheViewShowsItsFallback(t *testing.T) {
	ts := gluttertest.New(t)
	ts.SetViewSize(ui.Size{Width: 1100, Height: 680})
	ts.PumpWidget(NewCadApp())
	ts.Pump(frame)
	if len(ts.Elements(gluttertest.ByText("This app needs a GPU renderer."))) == 0 {
		t.Error("no fallback text")
	}
	s := gluttertest.StateOf[*cadState](ts, gluttertest.ByType[*CadPage]())
	if s.up != nil || s.device != nil {
		t.Error("a model was uploaded without a device")
	}
}

func TestANarrowWindowPutsThePanelUnderTheView(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{GPU: true})
	for _, tt := range []struct {
		name       string
		size       ui.Size
		panelBelow bool
	}{{"wide", ui.Size{Width: 1100, Height: 680}, false}, {"a phone", ui.Size{Width: 400, Height: 800}, true}} {
		t.Run(tt.name, func(t *testing.T) {
			ts.SetViewSize(tt.size)
			ts.PumpWidget(NewCadApp())
			ts.Pump(frame)
			ts.Pump(frame)
			open := ts.TopLeft(gluttertest.ByText("Display"))
			// Beside the view the panel is on the right; under it, it is lower than the middle of the screen.
			if tt.panelBelow {
				if open.Dy < tt.size.Height/2 {
					t.Errorf("the panel starts at y = %v on a phone", open.Dy)
				}
			} else if open.Dx < tt.size.Width/2 {
				t.Errorf("the panel starts at x = %v on a wide window", open.Dx)
			}
		})
	}
}

func TestLoadFile(t *testing.T) {
	dir := t.TempDir()
	write := func(name, text string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	stlPath := write("tri.STL", "solid t\nfacet normal 0 0 1\nouter loop\nvertex 0 0 0\nvertex 1 0 0\nvertex 0 1 0\nendloop\nendfacet\nendsolid t\n")
	objPath := write("tri.obj", "v 0 0 0\nv 1 0 0\nv 0 1 0\nf 1 2 3\n")
	txtPath := write("tri.txt", "hello")
	tests := []struct {
		name  string
		path  string
		tris  int
		route string
	}{
		{"an STL file, in capitals", stlPath, 1, ""},
		{"an OBJ file", objPath, 1, ""},
		{"another kind of file", txtPath, 0, "loadFile." + tagCadLoadFailed},
		{"a file that is not there", filepath.Join(dir, "none.stl"), 0, "loadFile->Load." + model.TagModelReadFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mdl, err := loadFile(context.Background(), tt.path)
			if tt.route == "" {
				if err != nil {
					t.Fatal(err)
				}
				if mdl.TriangleCount() != tt.tris {
					t.Errorf("%d triangles", mdl.TriangleCount())
				}
				return
			}
			appErr, ok := errors.FromError(err)
			if !ok || !appErr.HasRoute(tt.route) {
				t.Errorf("error %v, route %v", err, appErr)
			}
		})
	}
}

func TestTheBuiltInPartIsMadeOfClosedPieces(t *testing.T) {
	mdl := builtInPart()
	if len(mdl.Nodes) != 4 || len(mdl.Meshes) != 4 || len(mdl.Materials) != 4 || mdl.Units.Name != "mm" {
		t.Fatalf("%d nodes, %d meshes", len(mdl.Nodes), len(mdl.Meshes))
	}
	for _, mesh := range mdl.Meshes {
		// A closed surface: every edge of the triangles is used by two of them, once each way (positions compared, as the vertices
		// are doubled along the edges between the faces).
		type edge struct{ a, b [3]float32 }
		count := map[edge]int{}
		pos := func(i uint32) [3]float32 {
			return [3]float32{mesh.Positions[3*i], mesh.Positions[3*i+1], mesh.Positions[3*i+2]}
		}
		for i := 0; i+2 < len(mesh.Indices); i += 3 {
			for k := 0; k < 3; k++ {
				a, b := pos(mesh.Indices[i+k]), pos(mesh.Indices[i+(k+1)%3])
				if a == b {
					continue
				}
				count[edge{a, b}]++
			}
		}
		for e, n := range count {
			if n != 1 || count[edge{e.b, e.a}] != 1 {
				t.Errorf("%s: the edge %v is used %d times one way and %d the other", mesh.Name, e, n, count[edge{e.b, e.a}])
				break
			}
		}
		// The normals point out of the surface: the triangles' own normal agrees with the one of its vertices.
		for i := 0; i+2 < len(mesh.Indices); i += 3 {
			a, b, c := mesh.Position(int(mesh.Indices[i])), mesh.Position(int(mesh.Indices[i+1])), mesh.Position(int(mesh.Indices[i+2]))
			face := b.Sub(a).Cross(c.Sub(a))
			n := vecmath.Vec3{X: float64(mesh.Normals[3*mesh.Indices[i]]), Y: float64(mesh.Normals[3*mesh.Indices[i]+1]), Z: float64(mesh.Normals[3*mesh.Indices[i]+2])}
			if face.Dot(n) <= 0 {
				t.Errorf("%s: triangle %d turns away from its normal", mesh.Name, i/3)
				break
			}
		}
	}
	if b := mdl.Bounds(); b.Min != (vecmath.Vec3{X: -50, Y: -30}) || b.Max != (vecmath.Vec3{X: 50, Y: 30, Z: 55}) {
		t.Errorf("bounds %v", b)
	}
}

func TestNiceSpacing(t *testing.T) {
	tests := []struct {
		size, want float64
	}{{0, 1}, {-3, 1}, {100, 5}, {115, 5}, {20, 1}, {56, 2}, {1000, 50}, {7e5, 5e4}, {48, 2}}
	for _, tt := range tests {
		if got := niceSpacing(tt.size); got != tt.want {
			t.Errorf("niceSpacing(%v) = %v, want %v", tt.size, got, tt.want)
		}
	}
}
