package viewport3d_test

import (
	"math"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
	"github.com/rm4n0s/glutter/widgets"
)

// navRig is a tester with a navigation over the whole view of 800 x 600 logical pixels, around a camera that looks at the
// origin from 10 units, with a scene of a box 4 units wide.
type navRig struct {
	ts  *gluttertest.Tester
	cam *viewport3d.Camera
}

var sceneBox = vecmath.NewBox3(vecmath.Vec3{X: -2, Y: -2, Z: -2}, vecmath.Vec3{X: 2, Y: 2, Z: 2})

func newNavRig(t *testing.T, opts viewport3d.NavigationOpts, camOpts ...viewport3d.CameraOpts) *navRig {
	t.Helper()
	r := &navRig{ts: gluttertest.New(t), cam: viewport3d.NewCamera(camOpts...)}
	r.cam.SetBounds(sceneBox)
	r.ts.PumpWidget(viewport3d.NewNavigation(r.cam, widgets.NewSizedBoxExpand(nil), opts))
	return r
}

var center = ui.Offset{Dx: 400, Dy: 300}

// drag presses a pointer of the kind at a point, moves it by the steps, and lifts it.
func (r *navRig) drag(kind ui.PointerDeviceKind, buttons int, from ui.Offset, steps ...ui.Offset) {
	r.ts.Binding.Scheduler()
	g := r.ts.StartGesture(from, gluttertest.GestureOpts{Kind: kind, Buttons: foundation.Some(buttons)})
	for _, s := range steps {
		g.MoveBy(s)
	}
	g.Up()
	r.ts.Pump()
}

func (r *navRig) mouseDrag(from ui.Offset, steps ...ui.Offset) {
	r.drag(ui.PointerDeviceKindMouse, gestures.PrimaryButton, from, steps...)
}

// steps splits a drag into moves of 10 pixels or less.
func steps(dx, dy float64) []ui.Offset {
	n := int(math.Ceil(math.Max(math.Abs(dx), math.Abs(dy)) / 10))
	out := make([]ui.Offset, n)
	for i := range out {
		out[i] = ui.Offset{Dx: dx / float64(n), Dy: dy / float64(n)}
	}
	return out
}

func eyeAround(yaw, pitch, d float64) vecmath.Vec3 {
	return vecmath.Vec3{X: d * math.Cos(pitch) * math.Sin(yaw), Y: d * math.Sin(pitch), Z: d * math.Cos(pitch) * math.Cos(yaw)}
}

func TestNavigationSetsTheViewportOfTheCamera(t *testing.T) {
	r := newNavRig(t, viewport3d.NavigationOpts{})
	if got := r.cam.ViewportSize(); got != (ui.Size{Width: 800, Height: 600}) {
		t.Errorf("the camera's viewport = %v, want the navigation's 800 x 600", got)
	}
}

func TestMouseDragOrbits(t *testing.T) {
	tests := []struct {
		name       string
		dx, dy     float64
		yaw, pitch float64
	}{
		{"right", 100, 0, -1, 0},
		{"left", -60, 0, 0.6, 0},
		{"down", 0, 50, 0, 0.5},
		{"up", 0, -30, 0, -0.3},
		{"diagonal", 40, 80, -0.4, 0.8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newNavRig(t, viewport3d.NavigationOpts{})
			r.mouseDrag(center, steps(tt.dx, tt.dy)...)
			// The camera turns the way the scene follows the pointer: a drag to the right moves the camera to the left.
			if got := r.cam.Eye().Sub(r.cam.Target()); !nearV(got, eyeAround(tt.yaw, tt.pitch, 10), 1e-5) {
				t.Errorf("eye = %v, want %v", got, eyeAround(tt.yaw, tt.pitch, 10))
			}
			if !nearV(r.cam.Target(), vecmath.Vec3Zero, 1e-12) {
				t.Errorf("an orbit moved the target to %v", r.cam.Target())
			}
		})
	}
	t.Run("the speed", func(t *testing.T) {
		r := newNavRig(t, viewport3d.NavigationOpts{OrbitSpeed: 0.02})
		r.mouseDrag(center, steps(25, 0)...)
		if got := r.cam.Eye(); !nearV(got, eyeAround(-0.5, 0, 10), 1e-5) {
			t.Errorf("eye = %v, want %v", got, eyeAround(-0.5, 0, 10))
		}
	})
	t.Run("around the point under the pointer", func(t *testing.T) {
		pivot := vecmath.Vec3{X: 1.5, Y: 0.5, Z: 1}
		var asked []ui.Offset
		r := newNavRig(t, viewport3d.NavigationOpts{PivotAt: func(at ui.Offset) (vecmath.Vec3, bool) {
			asked = append(asked, at)
			return pivot, true
		}})
		before, _ := r.cam.Project(pivot)
		r.mouseDrag(center, steps(80, 40)...)
		after, ok := r.cam.Project(pivot)
		if !ok || !nearF(before.Dx, after.Dx, 1e-5) || !nearF(before.Dy, after.Dy, 1e-5) {
			t.Errorf("the pivot moved on the screen from %v to %v", before, after)
		}
		if len(asked) != 1 || asked[0] != center {
			t.Errorf("PivotAt was asked %v, want once, at the press %v", asked, center)
		}
		if r.cam.Target().DistanceTo(vecmath.Vec3Zero) < 1e-3 {
			t.Error("the target should move when the orbit is around another point")
		}
	})
	t.Run("a pointer with no point under it orbits the target", func(t *testing.T) {
		r := newNavRig(t, viewport3d.NavigationOpts{PivotAt: func(ui.Offset) (vecmath.Vec3, bool) { return vecmath.Vec3{}, false }})
		r.mouseDrag(center, steps(100, 0)...)
		if got := r.cam.Eye(); !nearV(got, eyeAround(-1, 0, 10), 1e-5) {
			t.Errorf("eye = %v", got)
		}
	})
}

func TestShiftAndMiddleDragsPan(t *testing.T) {
	// A pan follows the pointer: the target's plane moves with it.
	check := func(t *testing.T, r *navRig, dx, dy float64) {
		t.Helper()
		got, ok := r.cam.Project(vecmath.Vec3Zero)
		if !ok || !nearF(got.Dx, 400+dx, 1e-4) || !nearF(got.Dy, 300+dy, 1e-4) {
			t.Errorf("the point that was at the center is drawn at %v, want (%v, %v)", got, 400+dx, 300+dy)
		}
		if !nearF(r.cam.Distance(), 10, 1e-12) {
			t.Errorf("a pan changed the distance to %v", r.cam.Distance())
		}
	}
	t.Run("shift and the left button", func(t *testing.T) {
		r := newNavRig(t, viewport3d.NavigationOpts{})
		r.ts.SendKeyDownEvent(services.LogicalKeyboardKeyShiftLeft)
		r.mouseDrag(center, steps(70, -30)...)
		r.ts.SendKeyUpEvent(services.LogicalKeyboardKeyShiftLeft)
		check(t, r, 70, -30)
		if r.cam.Eye().Sub(r.cam.Target()).DistanceTo(vecmath.Vec3{Z: 10}) > 1e-9 {
			t.Error("a pan turned the camera")
		}
	})
	t.Run("the middle button", func(t *testing.T) {
		r := newNavRig(t, viewport3d.NavigationOpts{})
		r.drag(ui.PointerDeviceKindMouse, gestures.MiddleMouseButton, center, steps(-50, 40)...)
		check(t, r, -50, 40)
	})
	t.Run("the speed", func(t *testing.T) {
		r := newNavRig(t, viewport3d.NavigationOpts{PanSpeed: 2})
		r.drag(ui.PointerDeviceKindMouse, gestures.MiddleMouseButton, center, steps(30, 0)...)
		check(t, r, 60, 0)
	})
}

func TestEachBindingCanBeChanged(t *testing.T) {
	right := gestures.SecondaryButton
	tests := []struct {
		name   string
		opts   viewport3d.NavigationOpts
		button int
		shift  bool
		want   string // orbit, pan, zoom or none
	}{
		{"left to pan", viewport3d.NavigationOpts{LeftDrag: viewport3d.NavigationActionPan}, gestures.PrimaryButton, false, "pan"},
		{"left to zoom", viewport3d.NavigationOpts{LeftDrag: viewport3d.NavigationActionZoom}, gestures.PrimaryButton, false, "zoom"},
		{"left off", viewport3d.NavigationOpts{LeftDrag: viewport3d.NavigationActionNone}, gestures.PrimaryButton, false, "none"},
		{"shift left to orbit", viewport3d.NavigationOpts{ShiftLeftDrag: viewport3d.NavigationActionOrbit}, gestures.PrimaryButton, true, "orbit"},
		{"shift left off", viewport3d.NavigationOpts{ShiftLeftDrag: viewport3d.NavigationActionNone}, gestures.PrimaryButton, true, "none"},
		{"middle to orbit", viewport3d.NavigationOpts{MiddleDrag: viewport3d.NavigationActionOrbit}, gestures.MiddleMouseButton, false, "orbit"},
		{"middle off", viewport3d.NavigationOpts{MiddleDrag: viewport3d.NavigationActionNone}, gestures.MiddleMouseButton, false, "none"},
		{"right is off by default", viewport3d.NavigationOpts{}, right, false, "none"},
		{"right to orbit", viewport3d.NavigationOpts{RightDrag: viewport3d.NavigationActionOrbit}, right, false, "orbit"},
		{"right to pan", viewport3d.NavigationOpts{RightDrag: viewport3d.NavigationActionPan}, right, false, "pan"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newNavRig(t, tt.opts)
			if tt.shift {
				r.ts.SendKeyDownEvent(services.LogicalKeyboardKeyShiftLeft)
			}
			r.drag(ui.PointerDeviceKindMouse, tt.button, center, steps(0, 60)...)
			if tt.shift {
				r.ts.SendKeyUpEvent(services.LogicalKeyboardKeyShiftLeft)
			}
			var got string
			switch {
			case nearF(r.cam.Eye().Y, 0, 1e-9) && nearF(r.cam.Distance(), 10, 1e-9) && nearV(r.cam.Target(), vecmath.Vec3Zero, 1e-9):
				got = "none"
			case !nearF(r.cam.Distance(), 10, 1e-9):
				got = "zoom"
			case !nearV(r.cam.Target(), vecmath.Vec3Zero, 1e-9):
				got = "pan"
			case r.cam.Eye().Y > 1e-6:
				got = "orbit"
			default:
				got = "?"
			}
			if got != tt.want {
				t.Errorf("a drag did %q, want %q (eye %v, target %v, distance %v)", got, tt.want, r.cam.Eye(), r.cam.Target(), r.cam.Distance())
			}
		})
	}
}

func TestWheelZoomsTowardThePointer(t *testing.T) {
	t.Run("in and out", func(t *testing.T) {
		tests := []struct {
			name string
			opts viewport3d.NavigationOpts
			dy   float64
			want float64
		}{
			{"scroll down zooms out", viewport3d.NavigationOpts{}, 100, 10 * math.Exp(0.2)},
			{"scroll up zooms in", viewport3d.NavigationOpts{}, -100, 10 * math.Exp(-0.2)},
			{"inverted", viewport3d.NavigationOpts{InvertZoom: true}, 100, 10 * math.Exp(-0.2)},
			{"faster", viewport3d.NavigationOpts{ZoomSpeed: 2}, -100, 10 * math.Exp(-0.4)},
			{"off", viewport3d.NavigationOpts{Wheel: viewport3d.NavigationActionNone}, 100, 10},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r := newNavRig(t, tt.opts)
				r.ts.ScrollAt(center, ui.Offset{Dy: tt.dy})
				if !nearF(r.cam.Distance(), tt.want, 1e-9) {
					t.Errorf("distance = %v, want %v", r.cam.Distance(), tt.want)
				}
			})
		}
	})
	t.Run("towards the point under the pointer", func(t *testing.T) {
		for _, kind := range []viewport3d.ProjectionKind{viewport3d.ProjectionKindPerspective, viewport3d.ProjectionKindOrthographic} {
			r := newNavRig(t, viewport3d.NavigationOpts{}, viewport3d.CameraOpts{Projection: kind,
				Orientation: vecmath.NewQuatAxisAngle(vecmath.Vec3{X: 1, Y: 1}, 0.7)})
			at := ui.Offset{Dx: 650, Dy: 150}
			p := r.cam.FocalPoint(at)
			r.ts.ScrollAt(at, ui.Offset{Dy: -150})
			got, ok := r.cam.Project(p)
			if !ok || !nearF(got.Dx, at.Dx, 1e-5) || !nearF(got.Dy, at.Dy, 1e-5) {
				t.Errorf("%s: the point under the pointer moved to %v, want %v", kindName(kind), got, at)
			}
			if r.cam.Distance() >= 10 {
				t.Errorf("%s: distance = %v, want it to shrink", kindName(kind), r.cam.Distance())
			}
		}
	})
	t.Run("towards the pivot of the app", func(t *testing.T) {
		pivot := vecmath.Vec3{X: 1, Y: 1, Z: 1.5}
		r := newNavRig(t, viewport3d.NavigationOpts{PivotAt: func(ui.Offset) (vecmath.Vec3, bool) { return pivot, true }})
		at := ui.Offset{Dx: 300, Dy: 200}
		before, _ := r.cam.Project(pivot)
		r.ts.ScrollAt(at, ui.Offset{Dy: -100})
		if after, ok := r.cam.Project(pivot); !ok || !nearF(after.Dx, before.Dx, 1e-5) || !nearF(after.Dy, before.Dy, 1e-5) {
			t.Errorf("the pivot moved on the screen from %v to %v", before, after)
		}
	})
	t.Run("the deepest widget gets the wheel", func(t *testing.T) {
		ts := gluttertest.New(t)
		cam := viewport3d.NewCamera()
		list := widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewListView([]widgets.Widget{
			widgets.NewSizedBox(viewport3d.NewNavigation(cam, widgets.NewSizedBoxExpand(nil)), widgets.SizedBoxOpts{Height: foundation.Some(400.0)}),
			widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(2000.0)}),
		}))
		ts.PumpWidget(list)
		ts.ScrollAt(ui.Offset{Dx: 400, Dy: 200}, ui.Offset{Dy: -100})
		if cam.Distance() >= 10 {
			t.Errorf("the wheel did not zoom: distance %v", cam.Distance())
		}
		// Below the navigation the list scrolls.
		ts.ScrollAt(ui.Offset{Dx: 400, Dy: 500}, ui.Offset{Dy: 100})
		ts.Pump()
		if d := cam.Distance(); d != 10*math.Exp(-0.2) {
			t.Errorf("a wheel over the list zoomed the view: distance %v", d)
		}
	})
}

func TestDoubleClickFitsTheScene(t *testing.T) {
	for _, kind := range []ui.PointerDeviceKind{ui.PointerDeviceKindMouse, ui.PointerDeviceKindTouch} {
		t.Run(kind.String(), func(t *testing.T) {
			r := newNavRig(t, viewport3d.NavigationOpts{}, viewport3d.CameraOpts{Target: vecmath.Vec3{X: 5, Y: 5, Z: 5}, Distance: 40})
			want := r.cam.FitPose(sceneBox)
			g := gluttertest.GestureOpts{Kind: kind}
			r.ts.TapAt(center, g)
			r.ts.Pump(50 * time.Millisecond)
			r.ts.TapAt(center, g)
			if !r.cam.Animating() {
				t.Fatal("the double click did not start a move")
			}
			r.ts.Pump()
			r.ts.Pump(150 * time.Millisecond)
			if mid := r.cam.Pose(); mid == want || mid.Distance == 40 {
				t.Errorf("halfway the pose is %v: it should be between the start and %v", mid, want)
			}
			r.ts.Pump(300 * time.Millisecond)
			if r.cam.Animating() || r.cam.Pose() != want {
				t.Errorf("the pose after the move is %v (animating %v), want %v", r.cam.Pose(), r.cam.Animating(), want)
			}
		})
	}
	t.Run("a single click does nothing", func(t *testing.T) {
		r := newNavRig(t, viewport3d.NavigationOpts{})
		pose := r.cam.Pose()
		r.ts.TapAt(center, gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse})
		r.ts.Pump(time.Second)
		if r.cam.Pose() != pose || r.cam.Animating() {
			t.Errorf("a click moved the camera: %v", r.cam.Pose())
		}
	})
	t.Run("off", func(t *testing.T) {
		r := newNavRig(t, viewport3d.NavigationOpts{DoubleClick: viewport3d.NavigationActionNone}, viewport3d.CameraOpts{Distance: 40})
		g := gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse}
		r.ts.TapAt(center, g)
		r.ts.Pump(50 * time.Millisecond)
		r.ts.TapAt(center, g)
		r.ts.Pump(time.Second)
		if r.cam.Distance() != 40 {
			t.Errorf("distance = %v: a double click fitted the scene although it is off", r.cam.Distance())
		}
	})
	t.Run("with no duration the fit is at once", func(t *testing.T) {
		r := newNavRig(t, viewport3d.NavigationOpts{Duration: foundation.Some(time.Duration(0))}, viewport3d.CameraOpts{Distance: 40})
		g := gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse}
		r.ts.TapAt(center, g)
		r.ts.Pump(50 * time.Millisecond)
		r.ts.TapAt(center, g)
		if r.cam.Pose() != r.cam.FitPose(sceneBox) || r.cam.Animating() {
			t.Errorf("pose = %v, animating %v", r.cam.Pose(), r.cam.Animating())
		}
	})
	t.Run("without bounds there is nothing to fit", func(t *testing.T) {
		r := newNavRig(t, viewport3d.NavigationOpts{}, viewport3d.CameraOpts{Distance: 40})
		r.cam.SetBounds(vecmath.EmptyBox3())
		g := gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse}
		r.ts.TapAt(center, g)
		r.ts.Pump(50 * time.Millisecond)
		r.ts.TapAt(center, g)
		r.ts.Pump(time.Second)
		if r.cam.Distance() != 40 || r.cam.Animating() {
			t.Errorf("distance = %v, animating %v", r.cam.Distance(), r.cam.Animating())
		}
	})
}

func TestTouchGestures(t *testing.T) {
	finger := func(r *navRig, at ui.Offset) *gluttertest.TestGesture {
		return r.ts.StartGesture(at, gluttertest.GestureOpts{Kind: ui.PointerDeviceKindTouch})
	}
	t.Run("one finger orbits", func(t *testing.T) {
		r := newNavRig(t, viewport3d.NavigationOpts{})
		g := finger(r, center)
		for _, s := range steps(-50, 20) {
			g.MoveBy(s)
		}
		g.Up()
		if got := r.cam.Eye(); !nearV(got, eyeAround(0.5, 0.2, 10), 1e-5) {
			t.Errorf("eye = %v, want %v", got, eyeAround(0.5, 0.2, 10))
		}
	})
	t.Run("one finger can pan instead", func(t *testing.T) {
		r := newNavRig(t, viewport3d.NavigationOpts{TouchDrag: viewport3d.NavigationActionPan})
		g := finger(r, center)
		for _, s := range steps(40, 0) {
			g.MoveBy(s)
		}
		g.Up()
		if got, _ := r.cam.Project(vecmath.Vec3Zero); !nearF(got.Dx, 440, 1e-4) {
			t.Errorf("the target is drawn at %v, want x of 440", got)
		}
	})
	t.Run("two fingers pan, pinch and twist", func(t *testing.T) {
		tests := []struct {
			name  string
			opts  viewport3d.NavigationOpts
			moves func(a, b *gluttertest.TestGesture)
			check func(t *testing.T, r *navRig)
		}{
			{"pan: both fingers move together", viewport3d.NavigationOpts{}, func(a, b *gluttertest.TestGesture) {
				for _, s := range steps(60, 20) {
					a.MoveBy(s)
					b.MoveBy(s)
				}
			}, func(t *testing.T, r *navRig) {
				// The recognizer is accepted after the fingers moved a little: what they did before is not in the pan.
				got, _ := r.cam.Project(vecmath.Vec3Zero)
				if got.Dx < 430 || got.Dx > 460.001 || got.Dy < 308 || got.Dy > 320.001 || !nearF((got.Dx-400)/(got.Dy-300), 3, 0.1) {
					t.Errorf("the target is drawn at %v, want it moved most of the way to (460, 320), in that direction", got)
				}
				if !nearF(r.cam.Distance(), 10, 1e-9) {
					t.Errorf("a pan changed the distance to %v", r.cam.Distance())
				}
			}},
			{"pinch: the fingers spread to twice the span", viewport3d.NavigationOpts{}, func(a, b *gluttertest.TestGesture) {
				for i := 0; i < 10; i++ {
					a.MoveBy(ui.Offset{Dx: -10})
					b.MoveBy(ui.Offset{Dx: 10})
				}
			}, func(t *testing.T, r *navRig) {
				// The span went from 200 to 400: the scene is twice as big.
				if !nearF(r.cam.Distance(), 5, 0.05) {
					t.Errorf("distance = %v, want about 5", r.cam.Distance())
				}
			}},
			{"pinch: the fingers close", viewport3d.NavigationOpts{}, func(a, b *gluttertest.TestGesture) {
				for i := 0; i < 5; i++ {
					a.MoveBy(ui.Offset{Dx: 10})
					b.MoveBy(ui.Offset{Dx: -10})
				}
			}, func(t *testing.T, r *navRig) {
				if !nearF(r.cam.Distance(), 20, 0.2) {
					t.Errorf("distance = %v, want about 20 (the span halves)", r.cam.Distance())
				}
			}},
			{"twist: the fingers turn clockwise, and the scene turns with them", viewport3d.NavigationOpts{}, func(a, b *gluttertest.TestGesture) {
				// a is on the left, b on the right: b moves down and a up, around the center.
				for i := 0; i < 10; i++ {
					a.MoveBy(ui.Offset{Dy: -6})
					b.MoveBy(ui.Offset{Dy: 6})
				}
			}, func(t *testing.T, r *navRig) {
				p, _ := r.cam.Project(vecmath.Vec3{X: 1})
				if p.Dy <= 300+1 {
					t.Errorf("the point to the right of the target is drawn at %v: it should have gone down with the finger", p)
				}
				// The camera rolled around its line of sight (the fingers also spread a little, which zoomed).
				if dir := r.cam.Eye().Sub(r.cam.Target()).Normalize(); !nearV(dir, vecmath.Vec3Z, 1e-3) {
					t.Errorf("a twist turned the line of sight: the eye is in the direction %v", dir)
				}
			}},
			{"the pinch is off", viewport3d.NavigationOpts{Disable: viewport3d.NavigationDisablePinch}, func(a, b *gluttertest.TestGesture) {
				for i := 0; i < 10; i++ {
					a.MoveBy(ui.Offset{Dx: -10})
					b.MoveBy(ui.Offset{Dx: 10})
				}
			}, func(t *testing.T, r *navRig) {
				if r.cam.Distance() != 10 {
					t.Errorf("distance = %v, want 10", r.cam.Distance())
				}
			}},
			{"the twist is off", viewport3d.NavigationOpts{Disable: viewport3d.NavigationDisableTwist}, func(a, b *gluttertest.TestGesture) {
				for i := 0; i < 10; i++ {
					a.MoveBy(ui.Offset{Dy: -6})
					b.MoveBy(ui.Offset{Dy: 6})
				}
			}, func(t *testing.T, r *navRig) {
				if !nearV(r.cam.ViewUp(), vecmath.Vec3Y, 1e-12) {
					t.Errorf("view up = %v: the camera rolled", r.cam.ViewUp())
				}
			}},
			{"the two finger pan is off", viewport3d.NavigationOpts{Disable: viewport3d.NavigationDisableTwoFingerPan}, func(a, b *gluttertest.TestGesture) {
				for _, s := range steps(60, 20) {
					a.MoveBy(s)
					b.MoveBy(s)
				}
			}, func(t *testing.T, r *navRig) {
				if r.cam.Target().Length() > 0.05 {
					t.Errorf("target = %v: the camera panned", r.cam.Target())
				}
			}},
			{"the pinch speed", viewport3d.NavigationOpts{PinchSpeed: 2}, func(a, b *gluttertest.TestGesture) {
				for i := 0; i < 10; i++ {
					a.MoveBy(ui.Offset{Dx: -10})
					b.MoveBy(ui.Offset{Dx: 10})
				}
			}, func(t *testing.T, r *navRig) {
				if !nearF(r.cam.Distance(), 2.5, 0.05) {
					t.Errorf("distance = %v, want about 2.5 (the ratio squared)", r.cam.Distance())
				}
			}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r := newNavRig(t, tt.opts)
				a, b := finger(r, ui.Offset{Dx: 300, Dy: 300}), finger(r, ui.Offset{Dx: 500, Dy: 300})
				tt.moves(a, b)
				a.Up()
				b.Up()
				tt.check(t, r)
			})
		}
	})
}

func TestTrackpadPinchZooms(t *testing.T) {
	tests := []struct {
		name  string
		opts  viewport3d.NavigationOpts
		scale float64
		want  float64
	}{
		{"spread", viewport3d.NavigationOpts{}, 2, 5},
		{"close", viewport3d.NavigationOpts{}, 0.5, 20},
		{"off", viewport3d.NavigationOpts{Disable: viewport3d.NavigationDisableTrackpad}, 2, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newNavRig(t, tt.opts)
			g := r.ts.StartTrackpadGesture(center)
			for i := 1; i <= 10; i++ {
				g.Update(ui.Offset{}, 1+(tt.scale-1)*float64(i)/10, 0)
			}
			g.End()
			if !nearF(r.cam.Distance(), tt.want, 1e-6) {
				t.Errorf("distance = %v, want %v", r.cam.Distance(), tt.want)
			}
			if !nearV(r.cam.Target().Sub(vecmath.Vec3Zero), vecmath.Vec3Zero, 1e-6) {
				t.Errorf("a pinch at the center moved the target to %v", r.cam.Target())
			}
		})
	}
	t.Run("a twist and a pan of the trackpad do nothing", func(t *testing.T) {
		r := newNavRig(t, viewport3d.NavigationOpts{})
		g := r.ts.StartTrackpadGesture(center)
		for i := 1; i <= 10; i++ {
			g.Update(ui.Offset{Dx: float64(i) * 5, Dy: float64(i) * 3}, 1.0001, float64(i)*0.05)
		}
		g.End()
		if !nearF(r.cam.Distance(), 10, 0.01) || !nearV(r.cam.ViewUp(), vecmath.Vec3Y, 1e-9) || !nearV(r.cam.Target(), vecmath.Vec3Zero, 1e-3) {
			t.Errorf("the camera moved: distance %v, up %v, target %v", r.cam.Distance(), r.cam.ViewUp(), r.cam.Target())
		}
	})
}

func TestKeyboard(t *testing.T) {
	tests := []struct {
		name string
		key  services.LogicalKeyboardKey
		char string
		opts viewport3d.NavigationOpts
		eye  vecmath.Vec3 // from the target
	}{
		{"left", services.LogicalKeyboardKeyArrowLeft, "", viewport3d.NavigationOpts{}, eyeAround(-0.1, 0, 10)},
		{"right", services.LogicalKeyboardKeyArrowRight, "", viewport3d.NavigationOpts{}, eyeAround(0.1, 0, 10)},
		{"up", services.LogicalKeyboardKeyArrowUp, "", viewport3d.NavigationOpts{}, eyeAround(0, 0.1, 10)},
		{"down", services.LogicalKeyboardKeyArrowDown, "", viewport3d.NavigationOpts{}, eyeAround(0, -0.1, 10)},
		{"a bigger step", services.LogicalKeyboardKeyArrowRight, "", viewport3d.NavigationOpts{KeyboardOrbitStep: 0.5}, eyeAround(0.5, 0, 10)},
		{"plus", services.LogicalKeyboardKeyAdd, "+", viewport3d.NavigationOpts{}, vecmath.Vec3{Z: 10 / 1.1}},
		{"equal", services.LogicalKeyboardKeyEqual, "=", viewport3d.NavigationOpts{}, vecmath.Vec3{Z: 10 / 1.1}},
		{"numpad plus", services.LogicalKeyboardKeyNumpadAdd, "+", viewport3d.NavigationOpts{}, vecmath.Vec3{Z: 10 / 1.1}},
		{"minus", services.LogicalKeyboardKeyMinus, "-", viewport3d.NavigationOpts{}, vecmath.Vec3{Z: 11}},
		{"numpad minus", services.LogicalKeyboardKeyNumpadSubtract, "-", viewport3d.NavigationOpts{}, vecmath.Vec3{Z: 11}},
		{"a bigger zoom step", services.LogicalKeyboardKeyMinus, "-", viewport3d.NavigationOpts{KeyboardZoomStep: 2}, vecmath.Vec3{Z: 20}},
		{"the keys are off", services.LogicalKeyboardKeyArrowLeft, "", viewport3d.NavigationOpts{Disable: viewport3d.NavigationDisableKeyboard}, vecmath.Vec3{Z: 10}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.opts.Autofocus = true
			r := newNavRig(t, tt.opts)
			r.ts.SendKeyEvent(tt.key, gluttertest.KeyOpts{Character: tt.char})
			if got := r.cam.Eye().Sub(r.cam.Target()); !nearV(got, tt.eye, 1e-9) {
				t.Errorf("eye = %v, want %v", got, tt.eye)
			}
		})
	}
	t.Run("a press on the view takes the focus", func(t *testing.T) {
		r := newNavRig(t, viewport3d.NavigationOpts{})
		r.ts.SendKeyEvent(services.LogicalKeyboardKeyArrowLeft)
		if !nearV(r.cam.Eye(), vecmath.Vec3{Z: 10}, 1e-12) {
			t.Fatalf("a key moved the camera before the view had the focus: %v", r.cam.Eye())
		}
		r.ts.TapAt(center, gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse})
		r.ts.Pump(time.Second)
		r.ts.SendKeyEvent(services.LogicalKeyboardKeyArrowLeft)
		if !nearV(r.cam.Eye(), eyeAround(-0.1, 0, 10), 1e-9) {
			t.Errorf("eye = %v after the press and the key", r.cam.Eye())
		}
	})
	t.Run("the shortcuts and the actions are the app's to change", func(t *testing.T) {
		var orbits []viewport3d.OrbitIntent
		r := newNavRig(t, viewport3d.NavigationOpts{
			Autofocus: true,
			Shortcuts: widgets.ShortcutMap{
				{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyKeyA), Intent: viewport3d.OrbitIntent{Yaw: 2}},
				{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyHome), Intent: viewport3d.FitIntent{}},
				{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyKeyT), Intent: viewport3d.StandardViewIntent{View: viewport3d.StandardViewTop}},
				{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyKeyP), Intent: viewport3d.PanIntent{Dx: 20}},
			},
			Actions: widgets.ActionMap{
				widgets.IntentTypeOf[viewport3d.OrbitIntent](): widgets.NewCallbackAction(func(i viewport3d.OrbitIntent) any {
					orbits = append(orbits, i)
					return nil
				}),
			},
		})
		// The default keys are replaced.
		r.ts.SendKeyEvent(services.LogicalKeyboardKeyArrowLeft)
		if !nearV(r.cam.Eye(), vecmath.Vec3{Z: 10}, 1e-12) || len(orbits) != 0 {
			t.Errorf("an arrow still works: eye %v, orbits %v", r.cam.Eye(), orbits)
		}
		// Another action replaces the navigation's.
		r.ts.SendKeyEvent(services.LogicalKeyboardKeyKeyA)
		if len(orbits) != 1 || orbits[0].Yaw != 2 || !nearV(r.cam.Eye(), vecmath.Vec3{Z: 10}, 1e-12) {
			t.Errorf("the app's action got %v, and the camera is at %v", orbits, r.cam.Eye())
		}
		// The other intents are the navigation's.
		r.ts.SendKeyEvent(services.LogicalKeyboardKeyKeyP)
		if got, _ := r.cam.Project(vecmath.Vec3Zero); !nearF(got.Dx, 420, 1e-4) {
			t.Errorf("a pan of 20 pixels drew the target at %v", got)
		}
		r.ts.SendKeyEvent(services.LogicalKeyboardKeyKeyT)
		r.ts.Pump()
		r.ts.Pump(time.Second)
		if got := r.cam.Eye().Sub(r.cam.Target()).Normalize(); !nearV(got, vecmath.Vec3Y, 1e-9) {
			t.Errorf("after the top view the eye is in the direction %v", got)
		}
		r.ts.SendKeyEvent(services.LogicalKeyboardKeyHome)
		r.ts.Pump()
		r.ts.Pump(time.Second)
		if want := r.cam.FitPose(sceneBox); r.cam.Pose() != want {
			t.Errorf("after the fit the pose is %v, want %v", r.cam.Pose(), want)
		}
	})
}

func TestRemovingTheNavigationStopsItsMove(t *testing.T) {
	r := newNavRig(t, viewport3d.NavigationOpts{}, viewport3d.CameraOpts{Distance: 40})
	g := gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse}
	r.ts.TapAt(center, g)
	r.ts.Pump(50 * time.Millisecond)
	r.ts.TapAt(center, g)
	r.ts.Pump()
	r.ts.Pump(100 * time.Millisecond)
	if !r.cam.Animating() {
		t.Fatal("no move running")
	}
	r.ts.PumpWidget(widgets.NewSizedBoxShrink(nil))
	if r.cam.Animating() {
		t.Error("the move goes on after the navigation was removed")
	}
	r.ts.Pump(time.Second)
}

func TestUserInputStopsAMove(t *testing.T) {
	r := newNavRig(t, viewport3d.NavigationOpts{Autofocus: true}, viewport3d.CameraOpts{Distance: 40})
	fit := func() {
		g := gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse}
		r.ts.TapAt(center, g)
		r.ts.Pump(50 * time.Millisecond)
		r.ts.TapAt(center, g)
		r.ts.Pump()
		r.ts.Pump(100 * time.Millisecond)
	}
	for name, input := range map[string]func(){
		"a drag":  func() { r.mouseDrag(center, steps(30, 0)...) },
		"a wheel": func() { r.ts.ScrollAt(center, ui.Offset{Dy: 10}) },
		"a key":   func() { r.ts.SendKeyEvent(services.LogicalKeyboardKeyArrowLeft) },
	} {
		r.cam.SetPose(viewport3d.CameraPose{Distance: 40, Orientation: vecmath.QuatIdentity()})
		fit()
		if !r.cam.Animating() {
			t.Fatalf("%s: no move to stop", name)
		}
		input()
		if r.cam.Animating() {
			t.Errorf("%s did not stop the move", name)
		}
		r.ts.Pump(time.Second) // no ticker is left to fail the test at the end
	}
}

func TestLockedPointerOrbit(t *testing.T) {
	startLocked := func(t *testing.T, opts viewport3d.NavigationOpts) (*navRig, *gluttertest.TestGesture) {
		t.Helper()
		opts.LockPointer = true
		r := newNavRig(t, opts)
		g := r.ts.StartGesture(center, gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse})
		g.MoveBy(ui.Offset{Dx: 12})
		g.MoveBy(ui.Offset{Dx: 12})
		g.MoveBy(ui.Offset{Dx: 12})
		return r, g
	}
	t.Run("the drag locks the pointer and orbits with its motion", func(t *testing.T) {
		r, g := startLocked(t, viewport3d.NavigationOpts{})
		if !r.ts.PointerLocked(r.ts.View.ViewID()) {
			t.Fatal("the orbit did not lock the pointer")
		}
		yaw := r.cam.Eye().X
		r.ts.MoveLockedMouse(ui.Offset{Dx: 30, Dy: 10})
		if got := r.cam.Eye().X; got >= yaw {
			t.Errorf("a locked motion to the right did not turn the camera: eye.x %v then %v", yaw, got)
		}
		// The motion of the lock is the pointer's own: two moves add up as one.
		a := newNavRig(t, viewport3d.NavigationOpts{LockPointer: true})
		ga := a.ts.StartGesture(center, gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse})
		for i := 0; i < 3; i++ {
			ga.MoveBy(ui.Offset{Dx: 12})
		}
		a.ts.MoveLockedMouse(ui.Offset{Dx: 20})
		a.ts.MoveLockedMouse(ui.Offset{Dx: 10})
		b := newNavRig(t, viewport3d.NavigationOpts{LockPointer: true})
		gb := b.ts.StartGesture(center, gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse})
		for i := 0; i < 3; i++ {
			gb.MoveBy(ui.Offset{Dx: 12})
		}
		b.ts.MoveLockedMouse(ui.Offset{Dx: 30})
		if !nearV(a.cam.Eye(), b.cam.Eye(), 1e-9) {
			t.Errorf("two locked moves of 20 and 10 gave %v, one of 30 gave %v", a.cam.Eye(), b.cam.Eye())
		}
		g.Up()
		if r.ts.PointerLocked(r.ts.View.ViewID()) {
			t.Error("the pointer is still locked after the button went up")
		}
	})
	t.Run("a platform that cannot lock leaves the drag as it is", func(t *testing.T) {
		for _, tag := range []string{ui.TagPointerLockUnavailable, ui.TagPointerLockDenied} {
			r := newNavRig(t, viewport3d.NavigationOpts{LockPointer: true})
			r.ts.FailPointerLock(errors.New(tag, "no"))
			g := r.ts.StartGesture(center, gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse})
			for i := 0; i < 5; i++ {
				g.MoveBy(ui.Offset{Dx: 12})
			}
			g.Up()
			if r.ts.PointerLocked(r.ts.View.ViewID()) {
				t.Errorf("%s: locked", tag)
			}
			if got := r.cam.Eye(); !nearV(got, eyeAround(-0.6, 0, 10), 1e-5) {
				t.Errorf("%s: eye = %v, want the drag's %v", tag, got, eyeAround(-0.6, 0, 10))
			}
		}
	})
	t.Run("a lock that the platform ends does not make the pointer jump", func(t *testing.T) {
		r, g := startLocked(t, viewport3d.NavigationOpts{})
		r.ts.MoveLockedMouse(ui.Offset{Dx: 10})
		eye := r.cam.Eye()
		r.ts.EndPointerLock(r.ts.View.ViewID())
		// The pointer is where it really is: its first motion would jump, so it is skipped.
		g.MoveBy(ui.Offset{Dx: 200})
		if !nearV(r.cam.Eye(), eye, 1e-12) {
			t.Errorf("the jump back moved the camera: %v then %v", eye, r.cam.Eye())
		}
		g.MoveBy(ui.Offset{Dx: 20})
		if nearV(r.cam.Eye(), eye, 1e-6) {
			t.Error("the drag does not go on after the lock ended")
		}
		g.Up()
	})
	t.Run("the lock is for an orbit of the mouse only", func(t *testing.T) {
		r := newNavRig(t, viewport3d.NavigationOpts{LockPointer: true})
		r.drag(ui.PointerDeviceKindMouse, gestures.MiddleMouseButton, center, steps(40, 0)...)
		if r.ts.PointerLocked(r.ts.View.ViewID()) {
			t.Error("a pan locked the pointer")
		}
		g := r.ts.StartGesture(center, gluttertest.GestureOpts{Kind: ui.PointerDeviceKindTouch})
		g.MoveBy(ui.Offset{Dx: 30})
		g.MoveBy(ui.Offset{Dx: 30})
		if r.ts.PointerLocked(r.ts.View.ViewID()) {
			t.Error("a finger locked the pointer")
		}
		g.Up()
	})
	t.Run("the navigation going away gives the pointer back", func(t *testing.T) {
		r, _ := startLocked(t, viewport3d.NavigationOpts{})
		if !r.ts.PointerLocked(r.ts.View.ViewID()) {
			t.Fatal("not locked")
		}
		r.ts.PumpWidget(widgets.NewSizedBoxShrink(nil))
		if r.ts.PointerLocked(r.ts.View.ViewID()) {
			t.Error("the pointer is still locked after the navigation was removed")
		}
	})
}

// A recognizer above the view takes part in the arena as in Flutter: the view's drag wins over a pan of a parent, and a
// tap of a parent still gets its taps.
func TestTheArenaWithAnotherRecognizerAboveTheView(t *testing.T) {
	t.Run("a parent's tap still works", func(t *testing.T) {
		ts := gluttertest.New(t)
		cam := viewport3d.NewCamera()
		taps := 0
		ts.PumpWidget(widgets.NewGestureDetector(viewport3d.NewNavigation(cam, widgets.NewSizedBoxExpand(nil)), widgets.GestureDetectorOpts{
			OnTap: func() { taps++ },
		}))
		for _, kind := range []ui.PointerDeviceKind{ui.PointerDeviceKindMouse, ui.PointerDeviceKindTouch} {
			before := taps
			ts.TapAt(center, gluttertest.GestureOpts{Kind: kind})
			ts.Pump(time.Second) // the view's double click recognizer lets go after its timeout
			if taps != before+1 {
				t.Errorf("%v: the parent's tap was called %d times, want 1", kind, taps-before)
			}
		}
		if cam.Distance() != 10 || cam.Target() != vecmath.Vec3Zero {
			t.Error("a tap moved the camera")
		}
	})
	t.Run("the view's drag wins over a parent's pan, and the parent's pan gets what the view declines", func(t *testing.T) {
		ts := gluttertest.New(t)
		cam := viewport3d.NewCamera()
		pans := 0
		ts.PumpWidget(widgets.NewGestureDetector(viewport3d.NewNavigation(cam, widgets.NewSizedBoxExpand(nil), viewport3d.NavigationOpts{}),
			widgets.GestureDetectorOpts{OnPanUpdate: func(gestures.DragUpdateDetails) { pans++ }}))
		g := ts.StartGesture(center, gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse})
		for _, s := range steps(80, 0) {
			g.MoveBy(s)
		}
		g.Up()
		if pans != 0 {
			t.Errorf("the parent's pan got %d updates of a drag that orbits", pans)
		}
		if cam.Eye().X >= 0 {
			t.Errorf("the view did not orbit: eye %v", cam.Eye())
		}
		// The right button is off by default: the view does not claim it, and the parent's pan (a mouse drag of any
		// button the detector allows) does not either, so nothing moves. A left drag with no action does nothing in the
		// view but still wins over the parent: the view claimed the button.
		cam.SetPose(viewport3d.CameraPose{Distance: 10, Orientation: vecmath.QuatIdentity()})
	})
	t.Run("a vertical list around the view", func(t *testing.T) {
		ts := gluttertest.New(t)
		cam := viewport3d.NewCamera()
		ctl := widgets.NewScrollController()
		ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewListView([]widgets.Widget{
			widgets.NewSizedBox(viewport3d.NewNavigation(cam, widgets.NewSizedBoxExpand(nil)), widgets.SizedBoxOpts{Height: foundation.Some(400.0)}),
			widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(2000.0)}),
		}, widgets.ListViewOpts{Controller: ctl})))
		// A finger that moves across the list's axis orbits the view and the list stays.
		g := ts.StartGesture(ui.Offset{Dx: 400, Dy: 200}, gluttertest.GestureOpts{Kind: ui.PointerDeviceKindTouch})
		for _, s := range steps(60, 0) {
			g.MoveBy(s)
		}
		g.Up()
		ts.Pump(time.Second)
		if ctl.Offset() != 0 {
			t.Errorf("the list scrolled by %v although the finger moved across it", ctl.Offset())
		}
		if cam.Eye().X >= 0 {
			t.Errorf("the view did not orbit: eye %v", cam.Eye())
		}
		// A finger that moves along the list's axis is the list's: its drag is accepted after 18 pixels, and the view's
		// after 36 (the arena's rule, as for any two recognizers in Flutter).
		eye := cam.Eye()
		g = ts.StartGesture(ui.Offset{Dx: 400, Dy: 300}, gluttertest.GestureOpts{Kind: ui.PointerDeviceKindTouch})
		for _, s := range steps(0, -80) {
			g.MoveBy(s)
		}
		g.Up()
		ts.Pump(time.Second)
		if ctl.Offset() <= 0 || cam.Eye() != eye {
			t.Errorf("a vertical finger: the list is at %v and the eye moved from %v to %v: the list should have it", ctl.Offset(), eye, cam.Eye())
		}
		// A finger below the view scrolls the list.
		before := ctl.Offset()
		g = ts.StartGesture(ui.Offset{Dx: 400, Dy: 500}, gluttertest.GestureOpts{Kind: ui.PointerDeviceKindTouch})
		for _, s := range steps(0, -60) {
			g.MoveBy(s)
		}
		g.Up()
		ts.Pump(time.Second)
		if ctl.Offset() <= before {
			t.Errorf("the list did not scroll: offset %v, was %v", ctl.Offset(), before)
		}
	})
}
