package viewport3d_test

import (
	"image"
	"math"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/internal/scenetest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
	. "github.com/rm4n0s/glutter/widgets"
)

var magenta = ui.NewColor(0xFFFF00FF)

// tag is a 20 by 10 pixel child of a bright color that a test can find in a picture.
func tag(key foundation.Key, point vecmath.Vec3, opts ...viewport3d.AnchoredOpts) *viewport3d.Anchored {
	o := viewport3d.AnchoredOpts{}
	if len(opts) > 0 {
		o = opts[0]
	}
	o.Key = key
	return viewport3d.NewAnchored(point, NewSizedBox(NewColoredBox(magenta, nil), SizedBoxOpts{Width: foundation.Some(20.0), Height: foundation.Some(10.0)}), o)
}

// anchorRig is a view of 200 by 100 pixels in the middle of the window, with the children, and a camera that looks at the origin from
// 10 units.
type anchorRig struct {
	ts   *gluttertest.Tester
	rec  *gluttertest.GPURecorder
	cam  *viewport3d.Camera
	view gluttertest.Finder
}

func newAnchorRig(t *testing.T, opts viewport3d.ViewOpts) *anchorRig {
	t.Helper()
	scene := viewport3d.NewScene()
	scene.Add(viewport3d.NewItem(scenetest.Box(1, 1, 1), viewport3d.NewPhongMaterial()))
	cam := viewport3d.NewCamera()
	ts := gluttertest.New(t, gluttertest.Opts{GPU: true})
	rec := ts.GPUDevice()
	ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewCenter(NewSizedBox(viewport3d.NewView(scene, cam, opts), SizedBoxOpts{Width: foundation.Some(200.0), Height: foundation.Some(100.0)}))))
	return &anchorRig{ts: ts, rec: rec, cam: cam, view: gluttertest.ByType[*viewport3d.View]()}
}

// origin is where the view is in the window.
func (r *anchorRig) origin() ui.Offset { return r.ts.TopLeft(r.view) }

// count is the number of pixels of the color in the picture.
func count(img *image.RGBA, c ui.Color) int {
	n := 0
	for i := 0; i < len(img.Pix); i += 4 {
		if img.Pix[i] == uint8(c.R*255+0.5) && img.Pix[i+1] == uint8(c.G*255+0.5) && img.Pix[i+2] == uint8(c.B*255+0.5) {
			n++
		}
	}
	return n
}

func TestAnAnchoredChildIsPutAtThePointItFollows(t *testing.T) {
	key := foundation.NewUniqueKey()
	tests := []struct {
		name      string
		point     vecmath.Vec3
		alignment painting.Alignment
		offset    ui.Offset
	}{
		{"its center on the point", vecmath.Vec3{X: 1, Y: 0.5, Z: 0}, painting.AlignmentCenter, ui.Offset{}},
		{"its top left corner", vecmath.Vec3{X: -1, Y: 0.2, Z: 1}, painting.AlignmentTopLeft, ui.Offset{}},
		{"the middle of its bottom edge, moved by an offset", vecmath.Vec3{X: 0.3, Y: -0.4, Z: 0}, painting.AlignmentBottomCenter, ui.Offset{Dx: 5, Dy: -3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newAnchorRig(t, viewport3d.ViewOpts{Children: []Widget{tag(key, tt.point, viewport3d.AnchoredOpts{Alignment: tt.alignment, Offset: tt.offset})}})
			find := gluttertest.ByKey(key)
			check := func(when string) {
				t.Helper()
				at, ok := r.cam.Project(tt.point)
				if !ok {
					t.Fatalf("%s: the point is behind the camera", when)
				}
				want := r.origin().Add(at).Add(tt.offset).Sub(tt.alignment.AlongSize(ui.Size{Width: 20, Height: 10}))
				if got := r.ts.TopLeft(find); math.Abs(got.Dx-want.Dx) > 1e-3 || math.Abs(got.Dy-want.Dy) > 1e-3 {
					t.Errorf("%s: the child is at %v, want %v", when, got, want)
				}
			}
			check("at first")
			// The child follows each move of the camera, in a frame and with no build.
			for i, move := range []func(){func() { r.cam.Orbit(0.5, 0.2) }, func() { r.cam.Pan(10, -6) }, func() { r.cam.Zoom(1.3) }, func() { r.cam.SetKind(viewport3d.ProjectionKindOrthographic) }} {
				move()
				r.ts.Pump(16 * time.Millisecond)
				check([]string{"after an orbit", "after a pan", "after a zoom", "in an orthographic view"}[i])
			}
		})
	}
}

func TestAChildThatIsNotAnchoredFillsTheView(t *testing.T) {
	key := foundation.NewUniqueKey()
	r := newAnchorRig(t, viewport3d.ViewOpts{Children: []Widget{NewIgnorePointer(NewSizedBox(nil, SizedBoxOpts{Key: key}))}})
	got := r.ts.Size(gluttertest.ByKey(key))
	if got != (ui.Size{Width: 200, Height: 100}) {
		t.Errorf("the child is %v, want the view's 200 x 100", got)
	}
}

func TestAnAnchoredChildIsHiddenBehindTheCameraAndOutsideTheView(t *testing.T) {
	r := newAnchorRig(t, viewport3d.ViewOpts{Children: []Widget{tag(nil, vecmath.Vec3{})}})
	visible := func() bool { return count(r.ts.Screenshot(), magenta) > 0 }
	if !visible() {
		t.Fatalf("the child is not seen at first")
	}
	tests := []struct {
		name string
		move func()
		want bool
	}{
		// The camera looks away from the point: the point is behind it.
		{"the camera passes the point", func() { r.cam.SetTarget(vecmath.Vec3{Z: -20}) }, false},
		{"and looks back", func() { r.cam.SetTarget(vecmath.Vec3{}) }, true},
		// The point is far to the side, out of the view.
		{"the point is out of the view", func() { r.cam.SetTarget(vecmath.Vec3{X: 20}) }, false},
		{"and comes back", func() { r.cam.SetTarget(vecmath.Vec3{}) }, true},
	}
	for _, tt := range tests {
		tt.move()
		r.ts.Pump(16 * time.Millisecond)
		if got := visible(); got != tt.want {
			t.Errorf("%s: the child is seen %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestTheChildrenOfAViewGetThePointerBeforeTheNavigation(t *testing.T) {
	var tapped, tappedTop []string
	child := func(name string, x float64, taps *[]string) Widget {
		return viewport3d.NewAnchored(vecmath.Vec3{X: x}, NewGestureDetector(NewSizedBox(NewColoredBox(magenta, nil), SizedBoxOpts{Width: foundation.Some(30.0), Height: foundation.Some(30.0)}),
			GestureDetectorOpts{OnTap: func() { *taps = append(*taps, name) }, Behavior: rendering.HitTestBehaviorOpaque}))
	}
	// Two children that overlap: the second is over the first.
	r := newAnchorRig(t, viewport3d.ViewOpts{Children: []Widget{child("under", 0, &tapped), child("over", 0.2, &tappedTop)}})
	o := r.origin()
	r.ts.TapAt(o.Add(ui.Offset{Dx: 100, Dy: 50}))
	if len(tapped)+len(tappedTop) != 1 || len(tappedTop) != 1 {
		t.Errorf("a tap where the two overlap reached %v and %v: only the one over", tapped, tappedTop)
	}
	// Away from both, nothing is tapped; the view's own navigation takes a drag.
	before := r.cam.Pose()
	r.ts.TapAt(o.Add(ui.Offset{Dx: 10, Dy: 10}))
	if len(tapped)+len(tappedTop) != 1 {
		t.Errorf("a tap away from the children reached one")
	}
	r.ts.DragFrom(o.Add(ui.Offset{Dx: 10, Dy: 10}), ui.Offset{Dx: 40, Dy: 0})
	if r.cam.Pose() == before {
		t.Errorf("a drag away from the children did not orbit the camera")
	}
	// A drag that starts on a child is the child's, not the navigation's.
	before = r.cam.Pose()
	r.ts.DragFrom(o.Add(ui.Offset{Dx: 100, Dy: 50}), ui.Offset{Dx: 40, Dy: 0})
	if r.cam.Pose() != before {
		t.Errorf("a drag that began on a child orbited the camera")
	}
}

func TestAnAnchoredChildThatIsHiddenCannotBeTapped(t *testing.T) {
	taps := 0
	child := viewport3d.NewAnchored(vecmath.Vec3{}, NewGestureDetector(NewSizedBox(NewColoredBox(magenta, nil), SizedBoxOpts{Width: foundation.Some(30.0), Height: foundation.Some(30.0)}),
		GestureDetectorOpts{OnTap: func() { taps++ }, Behavior: rendering.HitTestBehaviorOpaque}))
	r := newAnchorRig(t, viewport3d.ViewOpts{Children: []Widget{child}})
	at := r.origin().Add(ui.Offset{Dx: 100, Dy: 50})
	r.ts.TapAt(at)
	if taps != 1 {
		t.Fatalf("the child was tapped %d times, want 1", taps)
	}
	r.cam.SetTarget(vecmath.Vec3{Z: -20})
	r.ts.Pump(16 * time.Millisecond)
	r.ts.TapAt(at)
	if taps != 1 {
		t.Errorf("the child behind the camera was tapped")
	}
}

func TestAnAnchoredChildIsInTheSemanticsTreeWhileItIsShown(t *testing.T) {
	skipUnlessDebug(t)
	child := viewport3d.NewAnchored(vecmath.Vec3{}, NewSemantics(NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(20.0), Height: foundation.Some(10.0)}),
		SemanticsOpts{Label: foundation.Some("Part A"), Container: true, TextDirection: ui.TextDirectionLTR}))
	r := newAnchorRig(t, viewport3d.ViewOpts{Children: []Widget{child}})
	r.ts.EnsureSemantics()
	r.ts.Pump(16 * time.Millisecond)
	owner := r.ts.RenderObject(r.view).AsRenderObject().Owner().SemanticsOwner()
	// has says whether a node of the semantics tree that the owner holds has the label.
	has := func(label string) bool {
		var walk func(n *semantics.SemanticsNode) bool
		walk = func(n *semantics.SemanticsNode) bool {
			if n.Label() == label {
				return true
			}
			found := false
			n.VisitChildren(func(c *semantics.SemanticsNode) bool {
				found = found || walk(c)
				return !found
			})
			return found
		}
		return walk(owner.RootSemanticsNode())
	}
	if !has("Part A") {
		t.Fatalf("the child is not in the semantics tree")
	}
	r.cam.SetTarget(vecmath.Vec3{Z: -20})
	r.ts.Pump(16 * time.Millisecond)
	r.ts.Pump(16 * time.Millisecond)
	if has("Part A") {
		t.Errorf("the child behind the camera is still in the semantics tree")
	}
	r.cam.SetTarget(vecmath.Vec3{})
	r.ts.Pump(16 * time.Millisecond)
	r.ts.Pump(16 * time.Millisecond)
	if !has("Part A") {
		t.Errorf("the child is not back in the semantics tree")
	}
}

func TestAViewShowsItsAxesAndItsViewCubeInTheCorners(t *testing.T) {
	r := newAnchorRig(t, viewport3d.ViewOpts{Axes: &viewport3d.AxesOpts{}, ViewCube: &viewport3d.ViewCubeOpts{}})
	axes, cube := gluttertest.ByType[*viewport3d.Axes](), gluttertest.ByType[*viewport3d.ViewCube]()
	o := r.origin()
	if got, want := r.ts.TopLeft(axes).Sub(o), (ui.Offset{Dx: 8, Dy: 100 - 8 - 72}); got != want {
		t.Errorf("the axes are at %v of the view, want %v (the bottom left)", got, want)
	}
	if got, want := r.ts.TopLeft(cube).Sub(o), (ui.Offset{Dx: 200 - 8 - 104, Dy: 8}); got != want {
		t.Errorf("the view cube is at %v of the view, want %v (the top right)", got, want)
	}
	// The pointer goes through the axes and stops at the cube: a tap on the cube does not reach the navigation (a double tap would
	// fit the scene), and a drag from the axes orbits.
	before := r.cam.Pose()
	r.ts.DragFrom(r.ts.Center(axes), ui.Offset{Dx: 30, Dy: 0})
	if r.cam.Pose() == before {
		t.Errorf("a drag that began on the axes did not orbit the camera")
	}
	// Other corners, and neither.
	custom := newAnchorRig(t, viewport3d.ViewOpts{Axes: &viewport3d.AxesOpts{Size: 40}, AxesAlignment: foundation.Some(painting.AlignmentTopLeft)})
	if got, want := custom.ts.TopLeft(axes).Sub(custom.origin()), (ui.Offset{Dx: 8, Dy: 8}); got != want {
		t.Errorf("the axes are at %v, want %v (the top left)", got, want)
	}
	if custom.ts.Exists(cube) {
		t.Errorf("a view with no ViewCube option has one")
	}
}

func TestTheViewProjectsAPointOfTheWorld(t *testing.T) {
	scene := viewport3d.NewScene()
	cam := viewport3d.NewCamera()
	cam.SetViewportSize(ui.Size{Width: 200, Height: 100})
	view := viewport3d.NewView(scene, cam)
	p := vecmath.Vec3{X: 1, Y: 0.5, Z: 2}
	got, ok := view.Project(p)
	want, wantOK := cam.Project(p)
	if ok != wantOK || got != want {
		t.Errorf("Project = %v, %v; want %v, %v", got, ok, want, wantOK)
	}
	if _, ok := view.Project(vecmath.Vec3{Z: 30}); ok {
		t.Errorf("a point behind the camera is projected")
	}
}

func TestTheDepthOfAnAnchoredChildIsAskedForOnlyWhenItHidesWhenCovered(t *testing.T) {
	tests := []struct {
		name       string
		children   []Widget
		wantPasses int
	}{
		{"no anchored child", nil, 1},
		{"a child that does not hide", []Widget{tag(nil, vecmath.Vec3{})}, 1},
		{"a child that hides when covered", []Widget{tag(nil, vecmath.Vec3{}, viewport3d.AnchoredOpts{HideWhenCovered: true})}, 2},
		{"two of them and one that does not", []Widget{tag(nil, vecmath.Vec3{X: 1}, viewport3d.AnchoredOpts{HideWhenCovered: true}), tag(nil, vecmath.Vec3{}),
			tag(nil, vecmath.Vec3{X: -1}, viewport3d.AnchoredOpts{HideWhenCovered: true})}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scene := viewport3d.NewScene()
			scene.Add(viewport3d.NewItem(scenetest.Box(1, 1, 1), viewport3d.NewPhongMaterial()))
			cam := viewport3d.NewCamera()
			ctl := viewport3d.NewViewController()
			ts := gluttertest.New(t, gluttertest.Opts{GPU: true})
			rec := ts.GPUDevice()
			rec.Reset()
			ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewCenter(NewSizedBox(viewport3d.NewView(scene, cam, viewport3d.ViewOpts{Controller: ctl, Children: tt.children}),
				SizedBoxOpts{Width: foundation.Some(200.0), Height: foundation.Some(100.0)}))))
			if n := countCalls(rec.Calls(), "BeginRenderPass"); n != tt.wantPasses {
				t.Errorf("the first frame made %d passes, want %d", n, tt.wantPasses)
			}
			// The answers do not draw again: nothing asks for a frame, and an idle view makes no pass.
			rec.CompleteReads()
			rec.Reset()
			ts.Pump(16 * time.Millisecond)
			ts.Pump(16 * time.Millisecond)
			if n := countCalls(rec.Calls(), "BeginRenderPass"); n != 0 {
				t.Errorf("the answers of the depth made the view draw %d times", n)
			}
			// A move of the camera draws, and asks again.
			rec.Reset()
			cam.Orbit(0.1, 0)
			ts.Pump(16 * time.Millisecond)
			if n := countCalls(rec.Calls(), "BeginRenderPass"); n != tt.wantPasses {
				t.Errorf("the frame after a move made %d passes, want %d", n, tt.wantPasses)
			}
		})
	}
}
