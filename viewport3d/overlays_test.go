package viewport3d_test

import (
	"bytes"
	"image"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
	. "github.com/rm4n0s/glutter/widgets"
)

// cameraAt is a camera with a standard view, and then turned.
func cameraAt(v viewport3d.StandardView, yaw, pitch float64) *viewport3d.Camera {
	cam := viewport3d.NewCamera()
	cam.SetStandardView(v)
	cam.Orbit(yaw, pitch)
	return cam
}

// overlayRow puts the widgets side by side on a light background.
func overlayRow(ws ...Widget) Widget {
	var kids []Widget
	for _, w := range ws {
		kids = append(kids, NewPadding(painting.NewEdgeInsetsAll(8), w))
	}
	roboto := painting.NewTextStyle(painting.TextStyleOpts{FontFamily: "Roboto"})
	return NewDirectionality(ui.TextDirectionLTR, NewDefaultTextStyle(roboto, NewColoredBox(ui.NewColor(0xFF22272E), NewCenter(NewRow(kids, RowOpts{MainAxisSize: rendering.MainAxisSizeMin})))))
}

func TestTheViewCubeFromSeveralSides(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 480, Height: 140}})
	cams := []*viewport3d.Camera{
		cameraAt(viewport3d.StandardViewFront, 0, 0),
		cameraAt(viewport3d.StandardViewIsometric, 0, 0),
		cameraAt(viewport3d.StandardViewTop, 0, 0),
		cameraAt(viewport3d.StandardViewFront, 0.9, -0.5),
	}
	var cubes []Widget
	for _, c := range cams {
		cubes = append(cubes, viewport3d.NewViewCube(c))
	}
	ts.PumpWidget(overlayRow(cubes...))
	ts.MatchesGolden("viewcube")
}

func TestTheAxesFromSeveralSides(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 100}})
	cams := []*viewport3d.Camera{
		cameraAt(viewport3d.StandardViewFront, 0, 0),
		cameraAt(viewport3d.StandardViewIsometric, 0, 0),
		cameraAt(viewport3d.StandardViewTop, 0, 0),
		cameraAt(viewport3d.StandardViewFront, 0.9, -0.5),
	}
	var axes []Widget
	for _, c := range cams {
		axes = append(axes, viewport3d.NewAxes(c))
	}
	ts.PumpWidget(overlayRow(axes...))
	ts.MatchesGolden("axes")
}

// cubePoint is where a point of the cube's own axes is in the picture of a cube widget of the size, for a camera with the world's
// default axes (up y, front z): the camera's view rotation takes the point, and the picture's y is down.
func cubePoint(cam *viewport3d.Camera, local vecmath.Vec3, size float64) ui.Offset {
	v := cam.RelativeViewMatrix().TransformDirection(local)
	half := size * 0.27
	return ui.Offset{Dx: size/2 + v.X*half, Dy: size/2 - v.Y*half}
}

func TestTappingTheViewCubeGoesToTheView(t *testing.T) {
	tests := []struct {
		name  string
		start viewport3d.StandardView
		// local is the point of the cube that is tapped, in its own axes.
		local vecmath.Vec3
		// want is the direction from the model to the viewer.
		want vecmath.Vec3
	}{
		{"the top face", viewport3d.StandardViewIsometric, vecmath.Vec3{Y: 1}, vecmath.Vec3Y},
		{"the front face", viewport3d.StandardViewIsometric, vecmath.Vec3{Z: 1}, vecmath.Vec3Z},
		{"the right face", viewport3d.StandardViewIsometric, vecmath.Vec3{X: 1}, vecmath.Vec3X},
		{"the edge between the top and the front", viewport3d.StandardViewIsometric, vecmath.Vec3{Y: 1, Z: 0.85}, vecmath.Vec3{Y: 1, Z: 1}},
		{"the edge between the front and the right", viewport3d.StandardViewIsometric, vecmath.Vec3{X: 1, Z: 0.85}, vecmath.Vec3{X: 1, Z: 1}},
		{"the corner of the top, the front and the right", viewport3d.StandardViewIsometric, vecmath.Vec3{X: 0.85, Y: 1, Z: 0.85}, vecmath.Vec3{X: 1, Y: 1, Z: 1}},
		{"the front face from the front goes nowhere new", viewport3d.StandardViewFront, vecmath.Vec3{Z: 1}, vecmath.Vec3Z},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 200, Height: 200}})
			cam := viewport3d.NewCamera(viewport3d.CameraOpts{Target: vecmath.Vec3{X: 1, Y: 2, Z: 3}, Distance: 7})
			cam.SetStandardView(tt.start)
			var asked []vecmath.Vec3
			cube := viewport3d.NewViewCube(cam, viewport3d.ViewCubeOpts{OnView: func(d vecmath.Vec3) { asked = append(asked, d) }})
			ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewCenter(cube)))
			origin := ts.TopLeft(gluttertest.ByType[*viewport3d.ViewCube]())
			ts.TapAt(origin.Add(cubePoint(cam, tt.local, 104)))
			ts.Pump(16 * time.Millisecond)
			if len(asked) != 1 || !nearVec3(asked[0], tt.want.Normalize(), 1e-9) {
				t.Fatalf("OnView was asked for %v, want once for %v", asked, tt.want.Normalize())
			}
			ts.PumpAndSettle()
			if viewer := cam.Orientation().Rotate(vecmath.Vec3Z); !nearVec3(viewer, tt.want.Normalize(), 1e-6) {
				t.Errorf("the viewer is at %v, want %v", viewer, tt.want.Normalize())
			}
			if cam.Target() != (vecmath.Vec3{X: 1, Y: 2, Z: 3}) || cam.Distance() != 7 {
				t.Errorf("the target is %v and the distance %v: they stay", cam.Target(), cam.Distance())
			}
			if cam.Animating() {
				t.Errorf("the camera is still moving")
			}
		})
	}
}

func nearVec3(a, b vecmath.Vec3, tol float64) bool { return a.Sub(b).Length() <= tol }

func TestTheViewCubeMovesTheCameraWithAnAnimation(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 200, Height: 200}})
	cam := viewport3d.NewCamera()
	cam.SetStandardView(viewport3d.StandardViewFront)
	ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewCenter(viewport3d.NewViewCube(cam))))
	origin := ts.TopLeft(gluttertest.ByType[*viewport3d.ViewCube]())
	// The right strip of the front face is the edge with the right face: it goes to the front-right view.
	ts.TapAt(origin.Add(cubePoint(cam, vecmath.Vec3{X: 0.85, Z: 1}, 104)))
	ts.Pump(16 * time.Millisecond)
	ts.Pump(100 * time.Millisecond)
	if !cam.Animating() {
		t.Fatalf("the camera does not move")
	}
	mid := cam.Orientation().Rotate(vecmath.Vec3Z)
	if mid.X <= 0.01 || mid.X >= 0.7 {
		t.Errorf("a third of the way the viewer is at %v: between the front and the front-right view", mid)
	}
	ts.PumpAndSettle()
	if end := cam.Orientation().Rotate(vecmath.Vec3Z); !nearVec3(end, vecmath.Vec3{X: 1, Z: 1}.Normalize(), 1e-6) {
		t.Errorf("the viewer ends at %v", end)
	}
	// A cube that goes at once: Some(0).
	ts2 := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 200, Height: 200}})
	cam2 := viewport3d.NewCamera()
	cam2.SetStandardView(viewport3d.StandardViewFront)
	ts2.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewCenter(viewport3d.NewViewCube(cam2, viewport3d.ViewCubeOpts{Duration: foundation.Some(time.Duration(0))}))))
	ts2.TapAt(ts2.TopLeft(gluttertest.ByType[*viewport3d.ViewCube]()).Add(cubePoint(cam2, vecmath.Vec3{X: 0.85, Z: 1}, 104)))
	if end := cam2.Orientation().Rotate(vecmath.Vec3Z); !nearVec3(end, vecmath.Vec3{X: 1, Z: 1}.Normalize(), 1e-6) {
		t.Errorf("with no duration the viewer is at %v at once", end)
	}
}

func TestTheViewCubeTurnsWithTheCamera(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 200, Height: 200}})
	cam := viewport3d.NewCamera()
	ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewCenter(NewColumn([]Widget{viewport3d.NewViewCube(cam), viewport3d.NewAxes(cam)}))))
	if ts.Binding.Scheduler().HasScheduledFrame() {
		t.Fatalf("an idle cube asks for frames")
	}
	cam.Orbit(0.4, 0.2)
	if !ts.Binding.Scheduler().HasScheduledFrame() {
		t.Errorf("the camera moved and the cube and the axes did not ask to paint")
	}
	ts.Pump(16 * time.Millisecond)
	if ts.Binding.Scheduler().HasScheduledFrame() {
		t.Errorf("a frame is still asked for after the cube painted")
	}
}

func TestTheViewCubeHighlightsWhatIsUnderThePointer(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 200, Height: 200}})
	cam := viewport3d.NewCamera()
	cam.SetStandardView(viewport3d.StandardViewIsometric)
	ts.PumpWidget(overlayRow(viewport3d.NewViewCube(cam)))
	plain := ts.Screenshot()
	origin := ts.TopLeft(gluttertest.ByType[*viewport3d.ViewCube]())
	ts.Hover(origin.Add(cubePoint(cam, vecmath.Vec3{Y: 1}, 104)))
	ts.Pump(16 * time.Millisecond)
	hovered := ts.Screenshot()
	if same(plain, hovered) {
		t.Fatalf("hovering over the top face changed nothing")
	}
	ts.MatchesGolden("viewcube-hover")
	// Moving to a corner highlights the corner and not the face.
	ts.Hover(origin.Add(cubePoint(cam, vecmath.Vec3{X: 0.85, Y: 1, Z: 0.85}, 104)))
	ts.Pump(16 * time.Millisecond)
	corner := ts.Screenshot()
	if same(hovered, corner) || same(plain, corner) {
		t.Errorf("the corner is not highlighted by itself")
	}
	// Leaving the cube puts it back.
	ts.Hover(ui.Offset{Dx: 1, Dy: 1})
	ts.Pump(16 * time.Millisecond)
	if !same(plain, ts.Screenshot()) {
		t.Errorf("the cube is still highlighted after the pointer left")
	}
}

func same(a, b *image.RGBA) bool { return bytes.Equal(a.Pix, b.Pix) }

func TestTheFacesOfTheViewCubeAreInTheSemanticsTree(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 200, Height: 200}})
	ts.EnsureSemantics()
	cam := viewport3d.NewCamera()
	cam.SetStandardView(viewport3d.StandardViewIsometric)
	ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewCenter(viewport3d.NewViewCube(cam, viewport3d.ViewCubeOpts{
		Labels: [6]string{3: "LINKS"}}))))
	paint := ts.RenderObject(gluttertest.ByType[*CustomPaint]())
	node := paint.AsRenderObject().DebugSemantics()
	if node == nil {
		t.Fatal("the cube has no semantics node")
	}
	byLabel := map[string]*semantics.SemanticsNode{}
	node.VisitChildren(func(n *semantics.SemanticsNode) bool { byLabel[n.Label()] = n; return true })
	// Seen from the corner, the front, the right and the top are in the tree, and the three that look away are not.
	for _, want := range []string{"FRONT view", "RIGHT view", "TOP view"} {
		if byLabel[want] == nil {
			t.Errorf("no node %q; nodes: %v", want, len(byLabel))
		}
	}
	if len(byLabel) != 3 {
		t.Errorf("%d face nodes, want the 3 that look at the viewer", len(byLabel))
	}
	// A tap action on a face goes to its view.
	top := byLabel["TOP view"]
	if top == nil {
		return
	}
	paint.AsRenderObject().Owner().SemanticsOwner().PerformAction(top.ID(), ui.SemanticsActionTap, nil)
	ts.PumpAndSettle()
	if viewer := cam.Orientation().Rotate(vecmath.Vec3Z); !nearVec3(viewer, vecmath.Vec3Y, 1e-6) {
		t.Errorf("after the tap action on the top face the viewer is at %v", viewer)
	}
	// The label of the cube itself.
	if !ts.Exists(gluttertest.BySemanticsLabel("View cube")) {
		t.Errorf("the cube has no label of its own")
	}
}

func TestTheAxesAreLabelled(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 200, Height: 200}})
	ts.EnsureSemantics()
	ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewCenter(viewport3d.NewAxes(viewport3d.NewCamera(), viewport3d.AxesOpts{SemanticsLabel: "World axes"}))))
	if !ts.Exists(gluttertest.BySemanticsLabel("World axes")) {
		t.Errorf("the axes have no label")
	}
}
