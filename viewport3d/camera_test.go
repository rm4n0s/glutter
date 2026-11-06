package viewport3d_test

import (
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
	"github.com/rm4n0s/glutter/widgets"
)

func nearV(a, b vecmath.Vec3, eps float64) bool { return a.NearlyEqual(b, eps) }

func nearF(a, b, eps float64) bool { return math.Abs(a-b) <= eps }

// newCamera makes a camera with a view of 800 x 600 logical pixels.
func newCamera(opts ...viewport3d.CameraOpts) *viewport3d.Camera {
	c := viewport3d.NewCamera(opts...)
	c.SetViewportSize(ui.Size{Width: 800, Height: 600})
	return c
}

func TestCameraDefaults(t *testing.T) {
	c := newCamera()
	tests := []struct {
		name      string
		got, want vecmath.Vec3
	}{
		{"eye", c.Eye(), vecmath.Vec3{Z: 10}},
		{"forward", c.Forward(), vecmath.Vec3{Z: -1}},
		{"right", c.Right(), vecmath.Vec3X},
		{"view up", c.ViewUp(), vecmath.Vec3Y},
		{"world up", c.Up(), vecmath.Vec3Y},
		{"target", c.Target(), vecmath.Vec3Zero},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !nearV(tt.got, tt.want, 1e-12) {
				t.Errorf("got %v, want %v", tt.got, tt.want)
			}
		})
	}
	if c.Kind() != viewport3d.ProjectionKindPerspective || c.Distance() != 10 || !nearF(c.FieldOfView(), math.Pi/4, 1e-15) {
		t.Errorf("kind %v, distance %v, field of view %v", c.Kind(), c.Distance(), c.FieldOfView())
	}
	if !math.IsInf(c.Far(), 1) {
		t.Errorf("the far plane of a perspective camera is %v, want infinity", c.Far())
	}
}

func TestCameraMatrices(t *testing.T) {
	for _, kind := range []viewport3d.ProjectionKind{viewport3d.ProjectionKindPerspective, viewport3d.ProjectionKindOrthographic} {
		c := newCamera(viewport3d.CameraOpts{Projection: kind, Target: vecmath.Vec3{X: 3, Y: -2, Z: 5}, Distance: 7,
			Orientation: vecmath.NewQuatAxisAngle(vecmath.Vec3{X: 1, Y: 2, Z: 3}, 0.9)})
		v := c.ViewMatrix()
		if got := v.TransformPoint(c.Eye()); !nearV(got, vecmath.Vec3Zero, 1e-12) {
			t.Errorf("kind %v: the eye is at %v in view space", kind, got)
		}
		if got, want := v.TransformPoint(c.Target()), (vecmath.Vec3{Z: -7}); !nearV(got, want, 1e-12) {
			t.Errorf("kind %v: the target is at %v in view space, want %v", kind, got, want)
		}
		// The view with the eye taken off is the relative view of a world whose origin is the eye.
		p := vecmath.Vec3{X: 1, Y: 2, Z: 3}
		want := v.TransformPoint(p)
		got := c.RelativeViewMatrix().TransformPoint(p.Sub(c.Eye()))
		if !nearV(got, want, 1e-12) {
			t.Errorf("kind %v: relative view gives %v, view gives %v", kind, got, want)
		}
		// ModelView and ModelViewProjection are float32 of the same product.
		world := vecmath.NewMat4TRS(vecmath.Vec3{X: 4, Y: 5, Z: 6}, vecmath.NewQuatAxisAngle(vecmath.Vec3Y, 0.4), vecmath.Vec3{X: 2, Y: 2, Z: 2})
		mv := c.ModelView(world)
		wantMV := v.Mul(world).Float32()
		for i := range mv {
			if math.Abs(float64(mv[i]-wantMV[i])) > 1e-4 {
				t.Errorf("kind %v: ModelView[%d] = %v, want %v", kind, i, mv[i], wantMV[i])
			}
		}
		mvp := c.ModelViewProjection(world)
		wantMVP := c.ViewProjectionMatrix().Mul(world).Float32()
		for i := range mvp {
			if math.Abs(float64(mvp[i]-wantMVP[i])) > 1e-3*math.Max(1, math.Abs(float64(wantMVP[i]))) {
				t.Errorf("kind %v: ModelViewProjection[%d] = %v, want %v", kind, i, mvp[i], wantMVP[i])
			}
		}
	}
}

func TestProjectionAndRaysAreInverses(t *testing.T) {
	r := rand.New(rand.NewSource(11))
	for _, kind := range []viewport3d.ProjectionKind{viewport3d.ProjectionKindPerspective, viewport3d.ProjectionKindOrthographic} {
		c := newCamera(viewport3d.CameraOpts{Projection: kind, Target: vecmath.Vec3{X: 3, Y: -2, Z: 5}, Distance: 7,
			Orientation: vecmath.NewQuatAxisAngle(vecmath.Vec3{X: 1, Y: 2, Z: 3}, 0.9)})
		c.SetBounds(vecmath.NewBox3(vecmath.Vec3{X: -5, Y: -5, Z: -5}, vecmath.Vec3{X: 10, Y: 10, Z: 10}))
		if center, ok := c.Project(c.Target()); !ok || !nearF(center.Dx, 400, 1e-9) || !nearF(center.Dy, 300, 1e-9) {
			t.Errorf("kind %v: the target is drawn at %v, %v, want the center", kind, center, ok)
		}
		for i := 0; i < 200; i++ {
			at := ui.Offset{Dx: r.Float64() * 800, Dy: r.Float64() * 600}
			ray := c.Ray(at)
			if !nearF(ray.Dir.Length(), 1, 1e-12) {
				t.Fatalf("kind %v: the ray's direction has length %v", kind, ray.Dir.Length())
			}
			p := ray.At(2 + r.Float64()*20)
			back, ok := c.Project(p)
			if !ok || !nearF(back.Dx, at.Dx, 1e-6) || !nearF(back.Dy, at.Dy, 1e-6) {
				t.Fatalf("kind %v: a point on the ray through %v is drawn at %v (%v)", kind, at, back, ok)
			}
			ndc, ok := c.ProjectNDC(p)
			if !ok {
				t.Fatalf("kind %v: ProjectNDC failed for a point in front", kind)
			}
			if got, ok := c.Unproject(at, ndc.Z); !ok || !nearV(got, p, 1e-6*p.Sub(c.Eye()).Length()) {
				t.Fatalf("kind %v: Unproject(%v, %v) = %v (%v), want %v", kind, at, ndc.Z, got, ok, p)
			}
		}
	}
	// A point behind a perspective camera is not drawn.
	c := newCamera()
	if _, ok := c.Project(vecmath.Vec3{Z: 20}); ok {
		t.Error("a point behind the camera was projected")
	}
	if _, ok := c.Unproject(ui.Offset{Dx: 400, Dy: 300}, 0); ok {
		t.Error("a depth of 0 (infinity) was unprojected")
	}
	// The screen's y goes down, and x goes right.
	right, _ := c.Project(vecmath.Vec3{X: 1})
	up, _ := c.Project(vecmath.Vec3{Y: 1})
	if right.Dx <= 400 || up.Dy >= 300 {
		t.Errorf("+x is drawn at %v and +y at %v: x must go right and y up the screen", right, up)
	}
}

func TestDepthIsReversedZ(t *testing.T) {
	c := newCamera()
	near, far := vecmath.Vec3{Z: 8}, vecmath.Vec3{Z: -100}
	n, _ := c.ProjectNDC(near)
	f, _ := c.ProjectNDC(far)
	if !(n.Z > f.Z && n.Z <= 1 && f.Z >= 0) {
		t.Errorf("depth of a near point %v and of a far point %v: want near > far, both in [0, 1]", n.Z, f.Z)
	}
	o := newCamera(viewport3d.CameraOpts{Projection: viewport3d.ProjectionKindOrthographic})
	o.SetBounds(vecmath.NewBox3(vecmath.Vec3{X: -1, Y: -1, Z: -1}, vecmath.Vec3{X: 1, Y: 1, Z: 1}))
	n, _ = o.ProjectNDC(vecmath.Vec3{Z: 1})
	f, _ = o.ProjectNDC(vecmath.Vec3{Z: -1})
	if !(n.Z > f.Z && n.Z <= 1 && f.Z >= 0) {
		t.Errorf("orthographic depth of a near point %v and of a far point %v", n.Z, f.Z)
	}
}

func TestNearPlaneFromTheBounds(t *testing.T) {
	box := vecmath.NewBox3(vecmath.Vec3{X: -1, Y: -1, Z: -1}, vecmath.Vec3{X: 1, Y: 1, Z: 1})
	r := math.Sqrt(3)
	tests := []struct {
		name string
		cam  *viewport3d.Camera
		want float64
	}{
		{"no bounds: a hundredth of the distance", newCamera(), 0.1},
		{"outside the bounding sphere", func() *viewport3d.Camera { c := newCamera(); c.SetBounds(box); return c }(), (10 - r) / 2},
		{"inside the bounding sphere", func() *viewport3d.Camera {
			c := newCamera(viewport3d.CameraOpts{Distance: 1})
			c.SetBounds(box)
			return c
		}(), 0.01},
		{"given", newCamera(viewport3d.CameraOpts{Near: 0.25}), 0.25},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cam.Near(); !nearF(got, tt.want, 1e-12) {
				t.Errorf("Near = %v, want %v", got, tt.want)
			}
			// The reversed-Z depth of the near plane is 1.
			p := tt.cam.Eye().Add(tt.cam.Forward().Scale(tt.cam.Near()))
			if ndc, ok := tt.cam.ProjectNDC(p); !ok || !nearF(ndc.Z, 1, 1e-9) {
				t.Errorf("the near plane has the depth %v (%v), want 1", ndc.Z, ok)
			}
		})
	}
	// The scene is never cut by the near plane: the nearest point of the bounding box is beyond it.
	c := newCamera()
	c.SetBounds(box)
	if nearest := box.DistanceTo(c.Eye()); c.Near() >= nearest {
		t.Errorf("the near plane %v is not in front of the scene, which starts at %v", c.Near(), nearest)
	}
	// An orthographic camera's range holds the bounding sphere, even behind the eye.
	o := newCamera(viewport3d.CameraOpts{Projection: viewport3d.ProjectionKindOrthographic, Distance: 0.5})
	o.SetBounds(box)
	if o.Near() >= -0.5 || o.Far() <= 0.5+r {
		t.Errorf("orthographic range [%v, %v] does not hold the scene around the eye", o.Near(), o.Far())
	}
}

func TestOrbit(t *testing.T) {
	tests := []struct {
		name       string
		yaw, pitch float64
		eye        vecmath.Vec3
	}{
		{"a quarter turn to the right", math.Pi / 2, 0, vecmath.Vec3{X: 10}},
		{"a quarter turn to the left", -math.Pi / 2, 0, vecmath.Vec3{X: -10}},
		{"a half turn", math.Pi, 0, vecmath.Vec3{Z: -10}},
		{"up by 30 degrees", 0, math.Pi / 6, vecmath.Vec3{Y: 5, Z: 10 * math.Cos(math.Pi/6)}},
		{"down by 30 degrees", 0, -math.Pi / 6, vecmath.Vec3{Y: -5, Z: 10 * math.Cos(math.Pi/6)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCamera()
			c.Orbit(tt.yaw, tt.pitch)
			if !nearV(c.Eye(), tt.eye, 1e-9) {
				t.Errorf("eye = %v, want %v", c.Eye(), tt.eye)
			}
			if !nearV(c.Target(), vecmath.Vec3Zero, 1e-12) || !nearF(c.Distance(), 10, 1e-12) {
				t.Errorf("target %v, distance %v changed", c.Target(), c.Distance())
			}
			// The camera looks at the target.
			if got := c.Target().Sub(c.Eye()).Normalize(); !nearV(got, c.Forward(), 1e-9) {
				t.Errorf("forward = %v, want %v", c.Forward(), got)
			}
		})
	}
	t.Run("the elevation stops short of the pole", func(t *testing.T) {
		c := newCamera()
		c.Orbit(0, 100)
		limit := 89.5 * math.Pi / 180
		if want := 10 * math.Sin(limit); !nearF(c.Eye().Y, want, 1e-9) {
			t.Errorf("eye.Y = %v, want %v", c.Eye().Y, want)
		}
		c.Orbit(1, 0) // the camera can still turn around the up axis
		if !nearF(c.Eye().Y, 10*math.Sin(limit), 1e-9) {
			t.Errorf("a yaw changed the elevation: eye.Y = %v", c.Eye().Y)
		}
		c.Orbit(0, -200)
		if want := -10 * math.Sin(limit); !nearF(c.Eye().Y, want, 1e-9) {
			t.Errorf("the lower limit: eye.Y = %v, want %v", c.Eye().Y, want)
		}
	})
	t.Run("the limit can be removed", func(t *testing.T) {
		c := newCamera(viewport3d.CameraOpts{ElevationLimit: math.Pi})
		c.Orbit(0, math.Pi) // over the pole
		if !nearV(c.Eye(), vecmath.Vec3{Z: -10}, 1e-9) {
			t.Errorf("eye = %v, want over the top to (0, 0, -10)", c.Eye())
		}
	})
	t.Run("orbit around a point keeps it where it is on the screen", func(t *testing.T) {
		c := newCamera()
		pivot := vecmath.Vec3{X: 2, Y: 1, Z: -3}
		before, _ := c.Project(pivot)
		c.OrbitAround(pivot, 0.7, 0.3)
		after, ok := c.Project(pivot)
		if !ok || !nearF(before.Dx, after.Dx, 1e-7) || !nearF(before.Dy, after.Dy, 1e-7) {
			t.Errorf("the pivot moved on the screen from %v to %v", before, after)
		}
		if c.Target().DistanceTo(pivot) < 1e-3 {
			t.Error("the target should have moved with the camera")
		}
	})
	t.Run("z up", func(t *testing.T) {
		c := newCamera(viewport3d.CameraOpts{Up: vecmath.Vec3Z, Orientation: vecmath.NewQuatAxisAngle(vecmath.Vec3X, math.Pi/2)})
		if !nearV(c.Eye(), vecmath.Vec3{Y: -10}, 1e-9) {
			t.Fatalf("eye = %v", c.Eye())
		}
		c.Orbit(math.Pi/2, 0)
		if !nearV(c.Eye(), vecmath.Vec3{X: 10}, 1e-9) {
			t.Errorf("eye after a quarter turn = %v, want (10, 0, 0)", c.Eye())
		}
		c.Orbit(0, math.Pi/6)
		if c.Eye().Z <= 0 {
			t.Errorf("the camera should have risen along z: %v", c.Eye())
		}
	})
}

func TestRoll(t *testing.T) {
	c := newCamera()
	eye := c.Eye()
	c.Roll(math.Pi / 2)
	if !nearV(c.Eye(), eye, 1e-12) || !nearV(c.Forward(), vecmath.Vec3{Z: -1}, 1e-12) {
		t.Errorf("a roll moved the eye to %v or turned the line of sight to %v", c.Eye(), c.Forward())
	}
	// A clockwise roll of the camera: its up tilts towards the right of the view, so the scene turns counterclockwise.
	if !nearV(c.ViewUp(), vecmath.Vec3X, 1e-12) {
		t.Errorf("view up = %v, want +x", c.ViewUp())
	}
	if p, _ := c.Project(vecmath.Vec3{Y: 1}); p.Dx >= 400 {
		t.Errorf("a point above the target is drawn at %v, want it to the left (the scene turned counterclockwise)", p)
	}
}

func TestPan(t *testing.T) {
	for _, kind := range []viewport3d.ProjectionKind{viewport3d.ProjectionKindPerspective, viewport3d.ProjectionKindOrthographic} {
		c := newCamera(viewport3d.CameraOpts{Projection: kind, Orientation: vecmath.NewQuatAxisAngle(vecmath.Vec3{X: 1, Y: 1}, 0.8), Distance: 12})
		p := c.Target()
		c.Pan(50, -30)
		got, ok := c.Project(p)
		if !ok || !nearF(got.Dx, 450, 1e-6) || !nearF(got.Dy, 270, 1e-6) {
			t.Errorf("kind %v: the point under the target went to %v, want (450, 270): it follows the drag", kind, got)
		}
	}
}

func TestZoom(t *testing.T) {
	for _, kind := range []viewport3d.ProjectionKind{viewport3d.ProjectionKindPerspective, viewport3d.ProjectionKindOrthographic} {
		t.Run(kindName(kind), func(t *testing.T) {
			c := newCamera(viewport3d.CameraOpts{Projection: kind, Orientation: vecmath.NewQuatAxisAngle(vecmath.Vec3{X: 1, Y: 1}, 0.8)})
			c.SetBounds(vecmath.NewBox3(vecmath.Vec3{X: -5, Y: -5, Z: -5}, vecmath.Vec3{X: 5, Y: 5, Z: 5}))
			target := c.Target()
			c.Zoom(0.5)
			if !nearF(c.Distance(), 5, 1e-12) || !nearV(c.Target(), target, 1e-12) {
				t.Errorf("Zoom(0.5): distance %v, target %v", c.Distance(), c.Target())
			}
			at := ui.Offset{Dx: 600, Dy: 200}
			p := c.FocalPoint(at)
			c.ZoomAt(0.4, at)
			if got, ok := c.Project(p); !ok || !nearF(got.Dx, at.Dx, 1e-6) || !nearF(got.Dy, at.Dy, 1e-6) {
				t.Errorf("ZoomAt: the point under the pointer moved to %v, want %v", got, at)
			}
			if !nearF(c.Distance(), 2, 1e-9) {
				t.Errorf("distance = %v, want 2", c.Distance())
			}
			// The target is still where the camera looks.
			if got := c.Eye().Add(c.Forward().Scale(c.Distance())); !nearV(got, c.Target(), 1e-9) {
				t.Errorf("eye + forward*distance = %v, target %v", got, c.Target())
			}
			// ZoomToward a point of the world.
			world := vecmath.Vec3{X: 1, Y: -2, Z: 0.5}
			before, _ := c.Project(world)
			c.ZoomToward(2, world)
			if after, ok := c.Project(world); !ok || !nearF(after.Dx, before.Dx, 1e-6) || !nearF(after.Dy, before.Dy, 1e-6) {
				t.Errorf("ZoomToward: the point went from %v to %v", before, after)
			}
		})
	}
	t.Run("the limits", func(t *testing.T) {
		c := newCamera(viewport3d.CameraOpts{MinDistance: 2, MaxDistance: 20})
		c.Zoom(0.001)
		if c.Distance() != 2 {
			t.Errorf("distance = %v, want the minimum 2", c.Distance())
		}
		c.Zoom(1e6)
		if c.Distance() != 20 {
			t.Errorf("distance = %v, want the maximum 20", c.Distance())
		}
		eye := c.Eye()
		c.Zoom(math.NaN())
		c.Zoom(-1)
		c.Zoom(math.Inf(1))
		if c.Eye() != eye || c.Distance() != 20 {
			t.Errorf("a factor that is not a number changed the camera: eye %v, distance %v", c.Eye(), c.Distance())
		}
	})
}

func kindName(k viewport3d.ProjectionKind) string {
	if k == viewport3d.ProjectionKindOrthographic {
		return "orthographic"
	}
	return "perspective"
}

func TestSwitchingProjectionKeepsTheSizeAtTheTarget(t *testing.T) {
	c := newCamera()
	p := vecmath.Vec3{X: 1, Y: 0.5}
	persp, _ := c.Project(p)
	c.SetKind(viewport3d.ProjectionKindOrthographic)
	ortho, _ := c.Project(p)
	if !nearF(persp.Dx, ortho.Dx, 1e-9) || !nearF(persp.Dy, ortho.Dy, 1e-9) {
		t.Errorf("a point on the target's plane is drawn at %v (perspective) and %v (orthographic)", persp, ortho)
	}
	c.SetKind(viewport3d.ProjectionKindUnspecified)
	if c.Kind() != viewport3d.ProjectionKindPerspective {
		t.Errorf("kind = %v, want perspective", c.Kind())
	}
}

func TestFitTo(t *testing.T) {
	boxes := []vecmath.Box3{
		vecmath.NewBox3(vecmath.Vec3{X: -1, Y: -1, Z: -1}, vecmath.Vec3{X: 1, Y: 1, Z: 1}),
		vecmath.NewBox3(vecmath.Vec3{X: 100, Y: 200, Z: 300}, vecmath.Vec3{X: 140, Y: 210, Z: 380}),
		vecmath.NewBox3(vecmath.Vec3{X: 0, Y: 0, Z: 0}, vecmath.Vec3{X: 0.001, Y: 50, Z: 0.001}), // a thin pole
	}
	for _, kind := range []viewport3d.ProjectionKind{viewport3d.ProjectionKindPerspective, viewport3d.ProjectionKindOrthographic} {
		for i, box := range boxes {
			c := newCamera(viewport3d.CameraOpts{Projection: kind, Orientation: vecmath.NewQuatAxisAngle(vecmath.Vec3{X: 1, Y: 2, Z: -1}, 0.7)})
			c.FitTo(box)
			if !nearV(c.Target(), box.Center(), 1e-9) {
				t.Errorf("%s box %d: the target is %v, want the center %v", kindName(kind), i, c.Target(), box.Center())
			}
			farthest := 0.0
			for j := 0; j < 8; j++ {
				ndc, ok := c.ProjectNDC(box.Corner(j))
				if !ok {
					t.Fatalf("%s box %d: corner %d is behind the camera", kindName(kind), i, j)
				}
				farthest = math.Max(farthest, math.Max(math.Abs(ndc.X), math.Abs(ndc.Y)))
			}
			if farthest > 1 || farthest < 0.9 {
				t.Errorf("%s box %d: the farthest corner is at %v of the half view, want just inside (0.9..1)", kindName(kind), i, farthest)
			}
			// Closer than the fit, a corner is out of the view.
			closer := c.Pose()
			closer.Distance *= 0.8
			c.SetPose(closer)
			out := false
			for j := 0; j < 8; j++ {
				if ndc, ok := c.ProjectNDC(box.Corner(j)); !ok || math.Abs(ndc.X) > 1 || math.Abs(ndc.Y) > 1 {
					out = true
				}
			}
			if !out {
				t.Errorf("%s box %d: at 0.8 of the fit distance the box still fits, so the fit is not the closest", kindName(kind), i)
			}
		}
	}
	t.Run("an empty box changes nothing", func(t *testing.T) {
		c := newCamera()
		pose := c.Pose()
		c.FitTo(vecmath.EmptyBox3())
		if c.Pose() != pose {
			t.Errorf("pose = %v, want %v", c.Pose(), pose)
		}
	})
	t.Run("a point moves the target only", func(t *testing.T) {
		c := newCamera()
		c.FitTo(vecmath.NewBox3(vecmath.Vec3{X: 4, Y: 5, Z: 6}, vecmath.Vec3{X: 4, Y: 5, Z: 6}))
		if c.Target() != (vecmath.Vec3{X: 4, Y: 5, Z: 6}) || c.Distance() <= 0 {
			t.Errorf("target %v, distance %v", c.Target(), c.Distance())
		}
	})
	t.Run("a narrow view fits the width", func(t *testing.T) {
		c := viewport3d.NewCamera()
		c.SetViewportSize(ui.Size{Width: 200, Height: 800})
		wide := vecmath.NewBox3(vecmath.Vec3{X: -10, Y: -1, Z: -1}, vecmath.Vec3{X: 10, Y: 1, Z: 1})
		c.FitTo(wide)
		for j := 0; j < 8; j++ {
			if ndc, ok := c.ProjectNDC(wide.Corner(j)); !ok || math.Abs(ndc.X) > 1 {
				t.Errorf("corner %d is outside the narrow view: %v", j, ndc)
			}
		}
	})
}

func TestStandardViews(t *testing.T) {
	tests := []struct {
		name      string
		up        vecmath.Vec3
		view      viewport3d.StandardView
		eyeDir    vecmath.Vec3 // from the target to the eye
		screenUp  vecmath.Vec3
		screenRgt vecmath.Vec3
	}{
		{"y up front", vecmath.Vec3Y, viewport3d.StandardViewFront, vecmath.Vec3{Z: 1}, vecmath.Vec3Y, vecmath.Vec3X},
		{"y up back", vecmath.Vec3Y, viewport3d.StandardViewBack, vecmath.Vec3{Z: -1}, vecmath.Vec3Y, vecmath.Vec3X.Neg()},
		{"y up right", vecmath.Vec3Y, viewport3d.StandardViewRight, vecmath.Vec3X, vecmath.Vec3Y, vecmath.Vec3Z.Neg()},
		{"y up left", vecmath.Vec3Y, viewport3d.StandardViewLeft, vecmath.Vec3X.Neg(), vecmath.Vec3Y, vecmath.Vec3Z},
		{"y up top", vecmath.Vec3Y, viewport3d.StandardViewTop, vecmath.Vec3Y, vecmath.Vec3Z.Neg(), vecmath.Vec3X},
		{"y up bottom", vecmath.Vec3Y, viewport3d.StandardViewBottom, vecmath.Vec3Y.Neg(), vecmath.Vec3Z, vecmath.Vec3X},
		{"y up isometric", vecmath.Vec3Y, viewport3d.StandardViewIsometric, vecmath.Vec3{X: 1, Y: 1, Z: 1}.Normalize(), vecmath.Vec3Zero, vecmath.Vec3Zero},
		{"z up front", vecmath.Vec3Z, viewport3d.StandardViewFront, vecmath.Vec3Y.Neg(), vecmath.Vec3Z, vecmath.Vec3X},
		{"z up right", vecmath.Vec3Z, viewport3d.StandardViewRight, vecmath.Vec3X, vecmath.Vec3Z, vecmath.Vec3Y},
		{"z up top", vecmath.Vec3Z, viewport3d.StandardViewTop, vecmath.Vec3Z, vecmath.Vec3Y, vecmath.Vec3X},
		{"z up isometric", vecmath.Vec3Z, viewport3d.StandardViewIsometric, vecmath.Vec3{X: 1, Y: -1, Z: 1}.Normalize(), vecmath.Vec3Zero, vecmath.Vec3Zero},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCamera(viewport3d.CameraOpts{Up: tt.up, Target: vecmath.Vec3{X: 1, Y: 2, Z: 3}, Distance: 6})
			c.SetStandardView(tt.view)
			if got := c.Eye().Sub(c.Target()).Normalize(); !nearV(got, tt.eyeDir, 1e-9) {
				t.Errorf("the eye is in the direction %v, want %v", got, tt.eyeDir)
			}
			if !nearF(c.Distance(), 6, 1e-12) || c.Target() != (vecmath.Vec3{X: 1, Y: 2, Z: 3}) {
				t.Errorf("distance %v or target %v changed", c.Distance(), c.Target())
			}
			if tt.screenUp.LengthSq() > 0 {
				if !nearV(c.ViewUp(), tt.screenUp, 1e-9) || !nearV(c.Right(), tt.screenRgt, 1e-9) {
					t.Errorf("view up %v and right %v, want %v and %v", c.ViewUp(), c.Right(), tt.screenUp, tt.screenRgt)
				}
			} else if c.ViewUp().Dot(tt.up) <= 0 {
				t.Errorf("the isometric view is upside down: view up %v", c.ViewUp())
			}
			if d := c.Orientation().Mat3().Det(); !nearF(d, 1, 1e-9) {
				t.Errorf("the orientation is not a rotation: determinant %v", d)
			}
		})
	}
	t.Run("a custom front", func(t *testing.T) {
		c := newCamera(viewport3d.CameraOpts{Up: vecmath.Vec3Y, Front: vecmath.Vec3X})
		c.SetStandardView(viewport3d.StandardViewFront)
		if !nearV(c.Eye().Normalize(), vecmath.Vec3X, 1e-9) {
			t.Errorf("eye = %v, want along +x", c.Eye())
		}
	})
	t.Run("unspecified keeps the orientation", func(t *testing.T) {
		c := newCamera()
		q := c.Orientation()
		c.SetStandardView(viewport3d.StandardViewUnspecified)
		if c.Orientation() != q {
			t.Errorf("orientation = %v, want %v", c.Orientation(), q)
		}
	})
	if got := viewport3d.StandardViewTop.String(); got != "top" {
		t.Errorf("String = %q", got)
	}
}

func TestListenersAreToldOfChangesOnly(t *testing.T) {
	c := newCamera()
	n := 0
	c.AddListener(func() { n++ })
	for _, step := range []struct {
		name string
		do   func()
		want int
	}{
		{"orbit", func() { c.Orbit(0.1, 0.1) }, 1},
		{"pan", func() { c.Pan(5, 5) }, 1},
		{"zoom", func() { c.Zoom(0.9) }, 1},
		{"roll", func() { c.Roll(0.1) }, 1},
		{"set the target", func() { c.SetTarget(vecmath.Vec3{X: 1}) }, 1},
		{"the same target", func() { c.SetTarget(vecmath.Vec3{X: 1}) }, 0},
		{"set the kind", func() { c.SetKind(viewport3d.ProjectionKindOrthographic) }, 1},
		{"the same kind", func() { c.SetKind(viewport3d.ProjectionKindOrthographic) }, 0},
		{"a zero orbit", func() { c.Orbit(0, 0) }, 0},
		{"a zero pan", func() { c.Pan(0, 0) }, 0},
		{"a zoom of 1", func() { c.Zoom(1) }, 0},
		{"set the bounds", func() { c.SetBounds(vecmath.NewBox3(vecmath.Vec3Zero, vecmath.Vec3{X: 1, Y: 1, Z: 1})) }, 1},
		{"the viewport size", func() { c.SetViewportSize(ui.Size{Width: 10, Height: 10}) }, 0},
		{"a field of view that is not valid", func() { c.SetFieldOfView(4) }, 0},
		{"a field of view", func() { c.SetFieldOfView(1) }, 1},
		{"the same pose", func() { c.SetPose(c.Pose()) }, 0},
	} {
		n = 0
		step.do()
		if n != step.want {
			t.Errorf("%s: the listener was called %d times, want %d", step.name, n, step.want)
		}
	}
}

// A point 50 km from the origin and one 0.1 mm from another, seen from 1 m, are drawn the right number of pixels apart
// with the matrices of the camera (float64 relative to the eye), which float32 alone would not give (D30).
func TestPrecisionAt50KmAndAtTenthsOfAMillimeter(t *testing.T) {
	const height = 1000.0
	pixelsPerMeter := height / (2 * math.Tan(math.Pi/8)) // at 1 m, with a field of view of 45 degrees
	// gpu does what the vertex shader does: a float32 matrix times a float32 vertex, divided by w.
	gpu := func(mvp [16]float32, v [3]float32) (x, y float32) {
		cx := mvp[0]*v[0] + mvp[4]*v[1] + mvp[8]*v[2] + mvp[12]
		cy := mvp[1]*v[0] + mvp[5]*v[1] + mvp[9]*v[2] + mvp[13]
		cw := mvp[3]*v[0] + mvp[7]*v[1] + mvp[11]*v[2] + mvp[15]
		return (cx/cw + 1) / 2 * height, (1 - cy/cw) / 2 * height
	}
	for _, origin := range []vecmath.Vec3{{X: 50000, Y: -20000, Z: 30000}, {}, {X: -50000}} {
		c := viewport3d.NewCamera(viewport3d.CameraOpts{Target: origin, Distance: 1})
		c.SetViewportSize(ui.Size{Width: height, Height: height})
		// An item whose mesh origin is the target: its vertices are float32 offsets, 0 and 0.1 mm.
		world := vecmath.NewMat4Translation(origin)
		mvp := c.ModelViewProjection(world)
		x0, y0 := gpu(mvp, [3]float32{})
		x1, y1 := gpu(mvp, [3]float32{1e-4, 0, 0})
		if got, want := float64(x1-x0), 1e-4*pixelsPerMeter; !nearF(got, want, 1e-3) || !nearF(float64(y1-y0), 0, 1e-3) {
			t.Errorf("origin %v: 0.1 mm apart is %v pixels, want %v", origin, got, want)
		}
		if !nearF(float64(x0), height/2, 1e-3) || !nearF(float64(y0), height/2, 1e-3) {
			t.Errorf("origin %v: the target is drawn at (%v, %v), want the center", origin, x0, y0)
		}
		// 1 m to the side is 1207 pixels (off the view, but measured the same way).
		x2, _ := gpu(mvp, [3]float32{1e-2, 0, 0})
		if got, want := float64(x2-x0), 1e-2*pixelsPerMeter; !nearF(got, want, 1e-2) {
			t.Errorf("origin %v: 1 cm apart is %v pixels, want %v", origin, got, want)
		}
		// The CPU's own projection (float64) agrees with the GPU's float32 to a thousandth of a pixel.
		p := origin.Add(vecmath.Vec3{X: 1e-4})
		if cpu, ok := c.Project(p); !ok || !nearF(cpu.Dx-height/2, 1e-4*pixelsPerMeter, 1e-6) {
			t.Errorf("origin %v: Project of the point 0.1 mm off gives %v (%v)", origin, cpu, ok)
		}
	}
	// With float32 alone, 50 km from the origin, 0.1 mm is lost: the same numbers through float32 world and view matrices.
	origin := vecmath.Vec3{X: 50000, Y: -20000, Z: 30000}
	c := viewport3d.NewCamera(viewport3d.CameraOpts{Target: origin, Distance: 1})
	c.SetViewportSize(ui.Size{Width: height, Height: height})
	vp := c.ViewProjectionMatrix().Float32()
	// The vertex's world position is a float32 too, which has no room for 0.1 mm at 50 km.
	world := func(v [3]float32) [3]float32 {
		return [3]float32{float32(origin.X) + v[0], float32(origin.Y) + v[1], float32(origin.Z) + v[2]}
	}
	x0, _ := gpu(vp, world([3]float32{}))
	x1, _ := gpu(vp, world([3]float32{1e-4, 0, 0}))
	if got, want := float64(x1-x0), 1e-4*pixelsPerMeter; nearF(got, want, 0.05) {
		t.Errorf("float32 world positions gave %v pixels for 0.1 mm at 50 km, which is right (%v): the test no longer shows what it should", got, want)
	}
}

func TestStandardAndFitMovesAnimate(t *testing.T) {
	ts := gluttertest.New(t)
	var host *tickerState
	ts.PumpWidget(newTickerHost(&host))
	c := newCamera()
	c.SetBounds(vecmath.NewBox3(vecmath.Vec3{X: -2, Y: -2, Z: -2}, vecmath.Vec3{X: 2, Y: 2, Z: 2}))
	var ended []bool
	n := 0
	c.AddListener(func() { n++ })
	pose := c.StandardViewPose(viewport3d.StandardViewTop)
	pose.Distance = 20
	c.AnimateTo(pose, &host.TickerProvider, viewport3d.AnimateOpts{
		Duration: foundation.Some(400 * time.Millisecond),
		OnEnd:    func(done bool) { ended = append(ended, done) },
	})
	if !c.Animating() {
		t.Fatal("the camera is not animating after AnimateTo")
	}
	start := c.Pose()
	ts.Pump() // the first frame starts the ticker
	ts.Pump(200 * time.Millisecond)
	mid := c.Pose()
	if mid == start || mid == pose {
		t.Fatalf("halfway the pose is %v: start %v, end %v", mid, start, pose)
	}
	// Halfway the distance is the geometric mean (the same factor each frame), by an ease in/out curve at 0.5.
	if want := math.Sqrt(10 * 20); !nearF(mid.Distance, want, 0.5) {
		t.Errorf("halfway the distance is %v, want about %v", mid.Distance, want)
	}
	if !nearF(mid.Orientation.Length(), 1, 1e-9) {
		t.Errorf("the orientation lost its length: %v", mid.Orientation.Length())
	}
	ts.Pump(300 * time.Millisecond)
	if c.Animating() || c.Pose() != pose {
		t.Errorf("after the end: animating %v, pose %v, want %v", c.Animating(), c.Pose(), pose)
	}
	if len(ended) != 1 || !ended[0] {
		t.Errorf("OnEnd calls = %v, want [true]", ended)
	}
	if n < 2 {
		t.Errorf("the listener was called %d times during the move, want a call per frame", n)
	}

	// A second move that another stops reports false; so does a move that SetPose stops.
	ended = nil
	c.AnimateTo(c.StandardViewPose(viewport3d.StandardViewFront), &host.TickerProvider, viewport3d.AnimateOpts{OnEnd: func(done bool) { ended = append(ended, done) }})
	ts.Pump()
	ts.Pump(100 * time.Millisecond)
	c.AnimateTo(c.StandardViewPose(viewport3d.StandardViewRight), &host.TickerProvider, viewport3d.AnimateOpts{OnEnd: func(done bool) { ended = append(ended, done) }})
	ts.Pump()
	ts.Pump(50 * time.Millisecond)
	c.SetPose(pose)
	if c.Animating() {
		t.Error("SetPose left the move running")
	}
	ts.Pump(time.Second)
	if c.Pose() != pose {
		t.Errorf("a stopped move went on: pose %v, want %v", c.Pose(), pose)
	}
	if len(ended) != 2 || ended[0] || ended[1] {
		t.Errorf("OnEnd calls = %v, want [false false]", ended)
	}

	// A duration of zero, and no ticker provider, move at once.
	ended = nil
	front := c.StandardViewPose(viewport3d.StandardViewFront)
	c.AnimateTo(front, &host.TickerProvider, viewport3d.AnimateOpts{Duration: foundation.Some(time.Duration(0)), OnEnd: func(done bool) { ended = append(ended, done) }})
	if c.Pose() != front || c.Animating() || len(ended) != 1 || !ended[0] {
		t.Errorf("a zero duration: pose %v, animating %v, ended %v", c.Pose(), c.Animating(), ended)
	}
	c.AnimateTo(pose, nil)
	if c.Pose() != pose || c.Animating() {
		t.Errorf("without a ticker provider the move is at once: pose %v, animating %v", c.Pose(), c.Animating())
	}
	c.Dispose()
}

// tickerState is a State that gives a camera's animation its ticker.
type tickerState struct {
	widgets.StateOf[*tickerHost]
	widgets.TickerProvider
}

func (s *tickerState) Build(widgets.BuildContext) widgets.Widget {
	return widgets.NewSizedBoxShrink(nil)
}

type tickerHost struct {
	widgets.StatefulWidget
	out **tickerState
}

func newTickerHost(out **tickerState) *tickerHost {
	return &tickerHost{StatefulWidget: widgets.NewStatefulWidgetBase(), out: out}
}

func (w *tickerHost) CreateState() widgets.State {
	s := &tickerState{}
	*w.out = s
	return s
}

func TestAMoveCanStartAnotherFromItsEnd(t *testing.T) {
	ts := gluttertest.New(t)
	var host *tickerState
	ts.PumpWidget(newTickerHost(&host))
	c := newCamera()
	first := c.StandardViewPose(viewport3d.StandardViewTop)
	last := c.StandardViewPose(viewport3d.StandardViewRight)
	var ends []string
	c.AnimateTo(first, &host.TickerProvider, viewport3d.AnimateOpts{
		Duration: foundation.Some(100 * time.Millisecond),
		OnEnd: func(done bool) {
			ends = append(ends, "first")
			c.AnimateTo(last, &host.TickerProvider, viewport3d.AnimateOpts{
				Duration: foundation.Some(100 * time.Millisecond),
				OnEnd:    func(done bool) { ends = append(ends, "second") },
			})
		},
	})
	ts.Pump()
	ts.Pump(150 * time.Millisecond)
	ts.Pump()
	ts.Pump(150 * time.Millisecond)
	if c.Animating() || c.Pose() != last || len(ends) != 2 {
		t.Errorf("animating %v, pose %v (want %v), ends %v", c.Animating(), c.Pose(), last, ends)
	}
	c.Dispose()
}
