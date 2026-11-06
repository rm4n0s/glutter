package viewport3d

import (
	"math"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
)

// ProjectionKind is how a camera makes the third dimension flat.
//
// Flutter: no counterpart
type ProjectionKind int

const (
	// ProjectionKindUnspecified is the zero value: a perspective projection.
	ProjectionKindUnspecified ProjectionKind = iota
	// ProjectionKindPerspective makes distant things smaller (reversed-Z, the far plane at infinity).
	ProjectionKindPerspective
	// ProjectionKindOrthographic keeps sizes whatever the distance, as a drawing does (reversed-Z).
	ProjectionKindOrthographic
)

// StandardView is one of the views that a CAD program names: seen along an axis, or from a corner.
//
// Flutter: no counterpart
type StandardView int

const (
	// StandardViewUnspecified is the zero value: no view.
	StandardViewUnspecified StandardView = iota
	// StandardViewFront looks at the front of the model, from the camera's Front direction.
	StandardViewFront
	// StandardViewBack looks at the back of the model.
	StandardViewBack
	// StandardViewLeft looks at the left side of the model.
	StandardViewLeft
	// StandardViewRight looks at the right side of the model.
	StandardViewRight
	// StandardViewTop looks down on the model, from the Up direction.
	StandardViewTop
	// StandardViewBottom looks up at the model.
	StandardViewBottom
	// StandardViewIsometric looks from the corner between the front, the right and the top.
	StandardViewIsometric
)

// String names the view.
//
// Flutter: no counterpart
func (v StandardView) String() string {
	switch v {
	case StandardViewFront:
		return "front"
	case StandardViewBack:
		return "back"
	case StandardViewLeft:
		return "left"
	case StandardViewRight:
		return "right"
	case StandardViewTop:
		return "top"
	case StandardViewBottom:
		return "bottom"
	case StandardViewIsometric:
		return "isometric"
	}
	return "unspecified"
}

// CameraPose is where a camera looks from: the point it orbits, how far it is from it and how it is turned.
//
// Flutter: no counterpart
type CameraPose struct {
	// Target is the point that the camera looks at and orbits.
	Target vecmath.Vec3
	// Distance is the distance from the eye to Target.
	Distance float64
	// Orientation turns the camera's axes into the world's: it looks along Orientation*(0, 0, -1), and its up is
	// Orientation*(0, 1, 0).
	Orientation vecmath.Quat
}

const (
	defaultFieldOfView    = math.Pi / 4
	defaultDistance       = 10.0
	defaultElevationLimit = 89.5 * math.Pi / 180
	// fitMargin leaves a little space around what FitTo fits.
	fitMargin = 1.05
	// defaultAnimationDuration is the length of AnimateTo without a duration.
	defaultAnimationDuration = 300 * time.Millisecond
)

// CameraOpts are the optional parameters of NewCamera.
//
// Flutter: no counterpart
type CameraOpts struct {
	// Projection is the kind of projection; the default is perspective.
	Projection ProjectionKind
	// FieldOfView is the vertical angle of view in radians, of a perspective projection; the default is pi/4. An
	// orthographic projection takes it too: its view is as tall as a perspective one at the target's distance, so that a
	// switch between the two keeps the size of what is at the target.
	FieldOfView float64
	// Target is the point that the camera looks at; the default is the origin.
	Target vecmath.Vec3
	// Distance is the distance from the eye to the target; the default is 10.
	Distance float64
	// Orientation turns the camera; the zero value is the identity: it looks down -z, with +y up.
	Orientation vecmath.Quat
	// Up is the direction that is up in the world, which the orbit turns around and the standard views use; the default
	// is +y.
	Up vecmath.Vec3
	// Front is the direction from the model to the viewer in the front view; the default is +z for a world with +y up, and
	// -y for one with +z up.
	Front vecmath.Vec3
	// MinDistance and MaxDistance limit the distance of the eye to the target; the defaults are 1e-9 and 1e15.
	MinDistance, MaxDistance float64
	// ElevationLimit is how far, in radians, the orbit may raise or lower the camera above the horizontal, which stops it
	// short of the pole, where the camera would flip. The default is 89.5 degrees; pi/2 or more removes the limit.
	ElevationLimit float64
	// Near sets the near plane of a perspective projection, instead of the one made from the scene's bounds.
	Near float64
}

// Camera is the point of view of a 3D view: it orbits a target at a distance, with a perspective or an orthographic
// projection. It is a Listenable: it notifies its listeners when anything that changes the picture changes (the view
// listens to it to draw). It keeps float64 numbers (D30); ModelView and ModelViewProjection give the GPU matrices that are
// relative to the eye.
//
// The camera looks along its orientation's -z, with y up; a pixel of the view is a logical pixel of its widget, with the
// origin at the top left. Use it on the UI thread.
//
// Flutter: no counterpart
type Camera struct {
	foundation.ChangeNotifier

	kind           ProjectionKind
	fovY           float64
	target         vecmath.Vec3
	distance       float64
	orientation    vecmath.Quat
	up, front      vecmath.Vec3
	minDistance    float64
	maxDistance    float64
	elevationLimit float64
	nearOverride   float64
	bounds         vecmath.Box3
	size           ui.Size
	anim           *cameraAnimation
}

// NewCamera creates a camera that looks at the origin from a distance of 10, down -z.
//
// Flutter: no counterpart
func NewCamera(opts ...CameraOpts) *Camera {
	o := firstOpts(opts)
	c := &Camera{
		kind:           o.Projection,
		fovY:           o.FieldOfView,
		target:         o.Target,
		distance:       o.Distance,
		orientation:    o.Orientation.Normalize(),
		up:             o.Up.Normalize(),
		front:          o.Front.Normalize(),
		minDistance:    o.MinDistance,
		maxDistance:    o.MaxDistance,
		elevationLimit: o.ElevationLimit,
		nearOverride:   o.Near,
		bounds:         vecmath.EmptyBox3(),
	}
	if c.kind == ProjectionKindUnspecified {
		c.kind = ProjectionKindPerspective
	}
	if c.fovY <= 0 {
		c.fovY = defaultFieldOfView
	}
	if c.distance <= 0 {
		c.distance = defaultDistance
	}
	if c.up.LengthSq() == 0 {
		c.up = vecmath.Vec3Y
	}
	if c.front.LengthSq() == 0 {
		c.front = defaultFront(c.up)
	}
	if c.minDistance <= 0 {
		c.minDistance = 1e-9
	}
	if c.maxDistance <= 0 {
		c.maxDistance = 1e15
	}
	if c.elevationLimit <= 0 {
		c.elevationLimit = defaultElevationLimit
	}
	c.distance = c.clampDistance(c.distance)
	return c
}

// defaultFront is the front direction for a world with the given up direction.
func defaultFront(up vecmath.Vec3) vecmath.Vec3 {
	switch {
	case up.NearlyEqual(vecmath.Vec3Y, 1e-6):
		return vecmath.Vec3Z
	case up.NearlyEqual(vecmath.Vec3Z, 1e-6):
		return vecmath.Vec3Y.Neg()
	}
	return up.Perpendicular()
}

// Dispose stops an animation and releases the camera's listeners.
//
// Flutter: no counterpart
func (c *Camera) Dispose() {
	c.StopAnimation()
	c.ChangeNotifier.Dispose()
}

// changed tells the listeners that the picture changes.
func (c *Camera) changed() { c.NotifyListeners() }

func (c *Camera) clampDistance(d float64) float64 {
	return math.Max(c.minDistance, math.Min(c.maxDistance, d))
}

// --- state -------------------------------------------------------------------------------------------------------

// Kind is the projection of the camera.
//
// Flutter: no counterpart
func (c *Camera) Kind() ProjectionKind { return c.kind }

// SetKind changes the projection. The view keeps the size of what is at the target.
//
// Flutter: no counterpart
func (c *Camera) SetKind(k ProjectionKind) {
	if k == ProjectionKindUnspecified {
		k = ProjectionKindPerspective
	}
	if c.kind != k {
		c.kind = k
		c.changed()
	}
}

// FieldOfView is the vertical angle of view in radians.
//
// Flutter: no counterpart
func (c *Camera) FieldOfView() float64 { return c.fovY }

// SetFieldOfView changes the vertical angle of view, in radians, which must be between 0 and pi.
//
// Flutter: no counterpart
func (c *Camera) SetFieldOfView(fovY float64) {
	if fovY > 0 && fovY < math.Pi && c.fovY != fovY {
		c.fovY = fovY
		c.changed()
	}
}

// Target is the point that the camera looks at and orbits.
//
// Flutter: no counterpart
func (c *Camera) Target() vecmath.Vec3 { return c.target }

// SetTarget moves the target; the eye moves with it.
//
// Flutter: no counterpart
func (c *Camera) SetTarget(t vecmath.Vec3) {
	if c.target != t && t.IsFinite() {
		c.target = t
		c.changed()
	}
}

// Distance is the distance from the eye to the target.
//
// Flutter: no counterpart
func (c *Camera) Distance() float64 { return c.distance }

// SetDistance moves the eye along the line of sight, to the distance from the target, within the camera's limits.
//
// Flutter: no counterpart
func (c *Camera) SetDistance(d float64) {
	d = c.clampDistance(d)
	if c.distance != d && !math.IsNaN(d) {
		c.distance = d
		c.changed()
	}
}

// Orientation turns the camera's axes into the world's: the camera looks along Orientation*(0, 0, -1).
//
// Flutter: no counterpart
func (c *Camera) Orientation() vecmath.Quat { return c.orientation }

// SetOrientation turns the camera around the target.
//
// Flutter: no counterpart
func (c *Camera) SetOrientation(q vecmath.Quat) {
	q = q.Normalize()
	if c.orientation != q {
		c.orientation = q
		c.changed()
	}
}

// Pose is where the camera looks from.
//
// Flutter: no counterpart
func (c *Camera) Pose() CameraPose {
	return CameraPose{Target: c.target, Distance: c.distance, Orientation: c.orientation}
}

// SetPose puts the camera in the pose, within its limits, and stops an animation.
//
// Flutter: no counterpart
func (c *Camera) SetPose(p CameraPose) {
	c.StopAnimation()
	c.setPose(p)
}

func (c *Camera) setPose(p CameraPose) {
	p.Orientation = p.Orientation.Normalize()
	p.Distance = c.clampDistance(p.Distance)
	if !p.Target.IsFinite() || math.IsNaN(p.Distance) || c.Pose() == p {
		return
	}
	c.target, c.distance, c.orientation = p.Target, p.Distance, p.Orientation
	c.changed()
}

// Up is the direction that is up in the world.
//
// Flutter: no counterpart
func (c *Camera) Up() vecmath.Vec3 { return c.up }

// Eye is where the camera is.
//
// Flutter: no counterpart
func (c *Camera) Eye() vecmath.Vec3 { return c.target.Sub(c.Forward().Scale(c.distance)) }

// Forward is the direction that the camera looks in.
//
// Flutter: no counterpart
func (c *Camera) Forward() vecmath.Vec3 { return c.orientation.Rotate(vecmath.Vec3{Z: -1}) }

// Right is the direction to the right of the view.
//
// Flutter: no counterpart
func (c *Camera) Right() vecmath.Vec3 { return c.orientation.Rotate(vecmath.Vec3X) }

// ViewUp is the direction to the top of the view, which differs from Up when the camera looks up or down or has rolled.
//
// Flutter: no counterpart
func (c *Camera) ViewUp() vecmath.Vec3 { return c.orientation.Rotate(vecmath.Vec3Y) }

// Bounds is the box of the scene that the near plane and FitTo are made from; it is empty until SetBounds.
//
// Flutter: no counterpart
func (c *Camera) Bounds() vecmath.Box3 { return c.bounds }

// SetBounds tells the camera how big the scene is. The near plane is made from it, and a double click fits it.
//
// Flutter: no counterpart
func (c *Camera) SetBounds(b vecmath.Box3) {
	if c.bounds != b {
		c.bounds = b
		c.changed()
	}
}

// ViewportSize is the size of the view in logical pixels, as the widget told the camera.
//
// Flutter: no counterpart
func (c *Camera) ViewportSize() ui.Size { return c.size }

// SetViewportSize tells the camera the size of its view in logical pixels. The widget does it at layout; it does not
// notify, since the widget redraws.
//
// Flutter: no counterpart
func (c *Camera) SetViewportSize(s ui.Size) { c.size = s }

// aspect is the width over the height of the view; 1 while the size is unknown.
func (c *Camera) aspect() float64 {
	if c.size.Width <= 0 || c.size.Height <= 0 {
		return 1
	}
	return c.size.Width / c.size.Height
}

// halfHeight is half of the height, in the units of the world, of what the view shows at the target's distance.
func (c *Camera) halfHeight() float64 { return c.distance * math.Tan(c.fovY/2) }

// unitsPerPixel is the size of a logical pixel in the world, at the target's distance.
func (c *Camera) unitsPerPixel() float64 {
	h := c.size.Height
	if h <= 0 {
		h = 1
	}
	return 2 * c.halfHeight() / h
}

// --- matrices ----------------------------------------------------------------------------------------------------

// Near is the distance of the near plane. A perspective camera takes it from the scene's bounds: half the distance to
// the nearest point that the scene's bounding sphere can have, so that nothing of the scene is cut, and a hundredth of
// the target's distance when the eye is inside the sphere. CameraOpts.Near sets it instead. An orthographic camera's
// near plane is the front of the bounding sphere, which may be behind the eye (a negative distance).
//
// Flutter: no counterpart
func (c *Camera) Near() float64 {
	if c.kind == ProjectionKindOrthographic {
		near, _ := c.orthoDepthRange()
		return near
	}
	if c.nearOverride > 0 {
		return c.nearOverride
	}
	near := c.distance * 0.01
	if !c.bounds.IsEmpty() {
		s := c.bounds.BoundingSphere()
		if d := c.Eye().DistanceTo(s.Center); d > s.Radius*(1+1e-9) {
			near = (d - s.Radius) * 0.5
		}
	}
	return math.Max(near, 1e-12)
}

// Far is the distance of the far plane: infinity for a perspective camera.
//
// Flutter: no counterpart
func (c *Camera) Far() float64 {
	if c.kind == ProjectionKindOrthographic {
		_, far := c.orthoDepthRange()
		return far
	}
	return math.Inf(1)
}

// orthoDepthRange is the near and far distance of an orthographic view: the bounding sphere of the scene, or a wide range
// around the target when the bounds are unknown.
func (c *Camera) orthoDepthRange() (near, far float64) {
	if c.bounds.IsEmpty() {
		return -100 * c.distance, 100 * c.distance
	}
	s := c.bounds.BoundingSphere()
	along := s.Center.Sub(c.Eye()).Dot(c.Forward())
	pad := math.Max(s.Radius*1.01, c.distance*0.01)
	return along - pad, along + pad
}

// ProjectionMatrix is the projection in float64, in WebGPU's clip space with reversed-Z (SPEC §27.7).
//
// Flutter: no counterpart
func (c *Camera) ProjectionMatrix() vecmath.Mat4 {
	if c.kind == ProjectionKindOrthographic {
		near, far := c.orthoDepthRange()
		h := c.halfHeight()
		w := h * c.aspect()
		return vecmath.Orthographic(-w, w, -h, h, near, far)
	}
	return vecmath.Perspective(c.fovY, c.aspect(), c.Near())
}

// RelativeViewMatrix turns directions of the world into directions of the view: the view matrix without the eye's
// translation. The GPU matrices use it with points that are relative to the eye.
//
// Flutter: no counterpart
func (c *Camera) RelativeViewMatrix() vecmath.Mat4 { return c.orientation.Conjugate().Mat4() }

// ViewMatrix turns points of the world into points of the view, in float64.
//
// Flutter: no counterpart
func (c *Camera) ViewMatrix() vecmath.Mat4 {
	return c.RelativeViewMatrix().Mul(vecmath.NewMat4Translation(c.Eye().Neg()))
}

// ViewProjectionMatrix is the projection times the view, in float64.
//
// Flutter: no counterpart
func (c *Camera) ViewProjectionMatrix() vecmath.Mat4 { return c.ProjectionMatrix().Mul(c.ViewMatrix()) }

// RelativeToEye returns the transform of an item with the eye taken off its translation, in float64: the model matrix
// of an item in a world whose origin is the eye. Where the item is far from the world's origin the subtraction is exact
// in float64, and what is left is small near the eye (D30).
//
// Flutter: no counterpart
func (c *Camera) RelativeToEye(world vecmath.Mat4) vecmath.Mat4 {
	return world.WithTranslation(world.Translation().Sub(c.Eye()))
}

// ModelView is the float32 matrix that takes the vertices of an item (relative to its mesh's origin) into the view,
// for an item whose float64 transform is world (D30). The matrix works in a world whose origin is the eye, so the GPU's
// float32 sees small numbers near the camera. The item's world transform must hold the mesh's origin.
//
// Flutter: no counterpart
func (c *Camera) ModelView(world vecmath.Mat4) [16]float32 {
	return c.RelativeViewMatrix().Mul(c.RelativeToEye(world)).Float32()
}

// ModelViewProjection is ModelView with the projection: the float32 matrix that takes an item's vertices into clip space.
//
// Flutter: no counterpart
func (c *Camera) ModelViewProjection(world vecmath.Mat4) [16]float32 {
	return c.ProjectionMatrix().Mul(c.RelativeViewMatrix()).Mul(c.RelativeToEye(world)).Float32()
}

// Frustum is the volume that the camera sees, in world coordinates.
//
// Flutter: no counterpart
func (c *Camera) Frustum() vecmath.Frustum { return vecmath.NewFrustum(c.ViewProjectionMatrix()) }

// --- points of the view ------------------------------------------------------------------------------------------

// ndcOf turns a point of the view, in logical pixels, into x and y of normalized device coordinates.
func (c *Camera) ndcOf(at ui.Offset) (x, y float64) {
	w, h := c.size.Width, c.size.Height
	if w <= 0 || h <= 0 {
		return 0, 0
	}
	return at.Dx/w*2 - 1, 1 - at.Dy/h*2
}

// Ray is the ray of the line of sight through a point of the view, in logical pixels: it starts at the eye (perspective)
// or in the plane of the eye (orthographic), and its direction has length 1.
//
// Flutter: no counterpart
func (c *Camera) Ray(at ui.Offset) vecmath.Ray {
	nx, ny := c.ndcOf(at)
	th := math.Tan(c.fovY / 2)
	right, up, fwd := c.Right(), c.ViewUp(), c.Forward()
	if c.kind == ProjectionKindOrthographic {
		h := c.halfHeight()
		origin := c.Eye().Add(right.Scale(nx * h * c.aspect())).Add(up.Scale(ny * h))
		return vecmath.Ray{Origin: origin, Dir: fwd}
	}
	dir := fwd.Add(right.Scale(nx * th * c.aspect())).Add(up.Scale(ny * th))
	return vecmath.Ray{Origin: c.Eye(), Dir: dir.Normalize()}
}

// Project returns the point of the view, in logical pixels, where a point of the world is drawn, and false when the point
// is behind a perspective camera.
//
// Flutter: no counterpart
func (c *Camera) Project(p vecmath.Vec3) (ui.Offset, bool) {
	ndc, ok := c.ProjectNDC(p)
	if !ok {
		return ui.Offset{}, false
	}
	return ui.Offset{Dx: (ndc.X + 1) / 2 * c.size.Width, Dy: (1 - ndc.Y) / 2 * c.size.Height}, true
}

// ProjectNDC returns the normalized device coordinates of a point of the world: x and y from -1 to 1, and the depth from 0
// (far) to 1 (near) of reversed-Z. It returns false for a point behind a perspective camera.
//
// Flutter: no counterpart
func (c *Camera) ProjectNDC(p vecmath.Vec3) (vecmath.Vec3, bool) {
	view := c.RelativeViewMatrix().TransformPoint(p.Sub(c.Eye()))
	ndc, w := c.ProjectionMatrix().ProjectPoint(view)
	if w <= 0 {
		return vecmath.Vec3{}, false
	}
	return ndc, true
}

// Unproject returns the point of the world that is drawn at a point of the view with a depth of reversed-Z (what a pick
// reads back, SPEC §27.8), and false when the depth is 0 or less for a perspective camera: that point is at infinity.
//
// Flutter: no counterpart
func (c *Camera) Unproject(at ui.Offset, depth float64) (vecmath.Vec3, bool) {
	nx, ny := c.ndcOf(at)
	right, up, fwd := c.Right(), c.ViewUp(), c.Forward()
	if c.kind == ProjectionKindOrthographic {
		near, far := c.orthoDepthRange()
		h := c.halfHeight()
		d := far - depth*(far-near)
		origin := c.Eye().Add(right.Scale(nx * h * c.aspect())).Add(up.Scale(ny * h))
		return origin.Add(fwd.Scale(d)), true
	}
	if depth <= 0 {
		return vecmath.Vec3{}, false
	}
	th := math.Tan(c.fovY / 2)
	d := c.Near() / depth
	dir := fwd.Add(right.Scale(nx * th * c.aspect())).Add(up.Scale(ny * th))
	return c.Eye().Add(dir.Scale(d)), true
}

// FocalPoint returns the point under a point of the view that is on the plane through the target, perpendicular to the
// line of sight: where a navigation without a depth to go by takes the point it zooms or turns around.
//
// Flutter: no counterpart
func (c *Camera) FocalPoint(at ui.Offset) vecmath.Vec3 {
	r := c.Ray(at)
	fwd := c.Forward()
	denom := r.Dir.Dot(fwd)
	if denom <= 1e-12 {
		return c.target
	}
	t := c.target.Sub(r.Origin).Dot(fwd) / denom
	return r.At(t)
}

// --- moving the camera -------------------------------------------------------------------------------------------

// Orbit turns the camera around the target: yaw radians counterclockwise as seen from above (around Up), and pitch radians
// upwards (the camera rises). The elevation stops short of the pole (CameraOpts.ElevationLimit).
//
// Flutter: no counterpart
func (c *Camera) Orbit(yaw, pitch float64) { c.OrbitAround(c.target, yaw, pitch) }

// OrbitAround turns the camera around a point, which stays where it is on the screen. See Orbit.
//
// Flutter: no counterpart
func (c *Camera) OrbitAround(pivot vecmath.Vec3, yaw, pitch float64) {
	if yaw == 0 && pitch == 0 {
		return
	}
	if c.elevationLimit < math.Pi/2 {
		elev := math.Asin(math.Max(-1, math.Min(1, -c.Forward().Dot(c.up))))
		next := math.Max(-c.elevationLimit, math.Min(c.elevationLimit, elev+pitch))
		pitch = next - elev
	}
	rot := vecmath.NewQuatAxisAngle(c.up, yaw).Mul(vecmath.NewQuatAxisAngle(c.Right(), -pitch))
	c.target = pivot.Add(rot.Rotate(c.target.Sub(pivot)))
	c.orientation = rot.Mul(c.orientation).Normalize()
	c.changed()
}

// Roll turns the camera around its line of sight by the angle in radians: positive is clockwise as the viewer sees it, which
// turns the scene counterclockwise.
//
// Flutter: no counterpart
func (c *Camera) Roll(angle float64) {
	if angle == 0 {
		return
	}
	c.orientation = vecmath.NewQuatAxisAngle(c.Forward(), angle).Mul(c.orientation).Normalize()
	c.changed()
}

// Pan moves the camera and its target across the view by a distance in logical pixels, as dragging the scene would: the
// point under the target stays under the pointer. dx is to the right and dy is down.
//
// Flutter: no counterpart
func (c *Camera) Pan(dx, dy float64) {
	if dx == 0 && dy == 0 {
		return
	}
	upp := c.unitsPerPixel()
	c.target = c.target.Sub(c.Right().Scale(dx * upp)).Add(c.ViewUp().Scale(dy * upp))
	c.changed()
}

// Zoom moves the eye towards the target when the factor is below 1 and away when it is above: the distance is multiplied
// by it, within the camera's limits.
//
// Flutter: no counterpart
func (c *Camera) Zoom(factor float64) { c.ZoomToward(factor, c.target) }

// ZoomAt is Zoom that keeps the point of the world under a point of the view (FocalPoint) where it is on the screen.
//
// Flutter: no counterpart
func (c *Camera) ZoomAt(factor float64, at ui.Offset) { c.ZoomToward(factor, c.FocalPoint(at)) }

// ZoomToward is Zoom that keeps a point of the world where it is on the screen: the eye moves along the line to the
// point, and the target with it.
//
// Flutter: no counterpart
func (c *Camera) ZoomToward(factor float64, point vecmath.Vec3) {
	if factor <= 0 || math.IsNaN(factor) || math.IsInf(factor, 0) {
		return
	}
	d := c.clampDistance(c.distance * factor)
	f := d / c.distance
	if f == 1 {
		return
	}
	eye := point.Add(c.Eye().Sub(point).Scale(f))
	c.distance = d
	c.target = eye.Add(c.Forward().Scale(d))
	c.changed()
}

// FitPose is the pose that shows a box whole, with the camera's orientation: the target is the box's center and the
// distance the smallest at which the corners are in the view, with a small margin. An empty box gives the current pose.
//
// Flutter: no counterpart
func (c *Camera) FitPose(box vecmath.Box3) CameraPose {
	pose := c.Pose()
	if box.IsEmpty() {
		return pose
	}
	center := box.Center()
	pose.Target = center
	th := math.Tan(c.fovY / 2)
	thx := th * c.aspect()
	right, up, back := c.Right(), c.ViewUp(), c.Forward().Neg()
	d := 0.0
	for i := 0; i < 8; i++ {
		rel := box.Corner(i).Sub(center)
		x, y := math.Abs(rel.Dot(right))*fitMargin, math.Abs(rel.Dot(up))*fitMargin
		need := math.Max(x/thx, y/th)
		if c.kind != ProjectionKindOrthographic {
			need += rel.Dot(back)
		}
		d = math.Max(d, need)
	}
	if c.kind != ProjectionKindOrthographic {
		// The eye stays outside the box, even when it is seen end on.
		d = math.Max(d, maxAlong(box, center, back)+box.BoundingSphere().Radius*0.05)
	}
	if d > 0 {
		pose.Distance = c.clampDistance(d)
	}
	return pose
}

// maxAlong is how far the box reaches from its center towards a direction.
func maxAlong(box vecmath.Box3, center, dir vecmath.Vec3) float64 {
	m := 0.0
	for i := 0; i < 8; i++ {
		m = math.Max(m, box.Corner(i).Sub(center).Dot(dir))
	}
	return m
}

// FitTo moves the target to the center of the box and the eye to the distance at which the whole box is in the view (see
// FitPose), keeping the orientation.
//
// Flutter: no counterpart
func (c *Camera) FitTo(box vecmath.Box3) { c.SetPose(c.FitPose(box)) }

// StandardViewOrientation is the orientation of a standard view: the camera looks at the model along an axis (or the
// diagonal of the isometric view) with the world's up upwards in the view. The top and bottom views have the back and the
// front of the model upwards. Right-handed, with the right of the front view at Up x Front.
//
// Flutter: no counterpart
func (c *Camera) StandardViewOrientation(v StandardView) vecmath.Quat {
	front := c.front
	right := c.up.Cross(front).Normalize()
	// dir is the direction from the model to the viewer; screenUp what is upwards in the view.
	var dir, screenUp vecmath.Vec3
	switch v {
	case StandardViewFront:
		dir, screenUp = front, c.up
	case StandardViewBack:
		dir, screenUp = front.Neg(), c.up
	case StandardViewRight:
		dir, screenUp = right, c.up
	case StandardViewLeft:
		dir, screenUp = right.Neg(), c.up
	case StandardViewTop:
		dir, screenUp = c.up, front.Neg()
	case StandardViewBottom:
		dir, screenUp = c.up.Neg(), front
	case StandardViewIsometric:
		dir, screenUp = front.Add(right).Add(c.up).Normalize(), c.up
	default:
		return c.orientation
	}
	backAxis := dir.Normalize()
	xAxis := screenUp.Cross(backAxis).Normalize()
	yAxis := backAxis.Cross(xAxis)
	return vecmath.NewQuatBasis(xAxis, yAxis, backAxis)
}

// StandardViewPose is the pose of a standard view of what the camera looks at: the same target and distance, the view's
// orientation.
//
// Flutter: no counterpart
func (c *Camera) StandardViewPose(v StandardView) CameraPose {
	pose := c.Pose()
	pose.Orientation = c.StandardViewOrientation(v)
	return pose
}

// ViewDirectionOrientation is the orientation of a view of the model from a direction (from the model to the viewer, in the
// world; it need not be a unit vector) with the world's up upwards in the view, as the standard views have it. When the
// direction is that of the up axis (the top and bottom views) the front of the model is downwards in the view, as in
// StandardViewTop and StandardViewBottom. The view cube goes to its edges and corners with it. A direction with no length gives
// the camera's own orientation.
//
// Flutter: no counterpart
func (c *Camera) ViewDirectionOrientation(dir vecmath.Vec3) vecmath.Quat {
	if dir.LengthSq() == 0 || !dir.IsFinite() {
		return c.orientation
	}
	back := dir.Normalize()
	screenUp := c.up
	if math.Abs(back.Dot(c.up)) > 1-1e-9 {
		// Seen along the up axis: the front is down in the view from above, and up in the view from below.
		screenUp = c.front.Neg()
		if back.Dot(c.up) < 0 {
			screenUp = c.front
		}
	}
	xAxis := screenUp.Cross(back).Normalize()
	yAxis := back.Cross(xAxis)
	return vecmath.NewQuatBasis(xAxis, yAxis, back)
}

// ViewDirectionPose is the pose of a view from a direction (ViewDirectionOrientation) of what the camera looks at: the same
// target and distance.
//
// Flutter: no counterpart
func (c *Camera) ViewDirectionPose(dir vecmath.Vec3) CameraPose {
	pose := c.Pose()
	pose.Orientation = c.ViewDirectionOrientation(dir)
	return pose
}

// SetStandardView turns the camera to a standard view at once. AnimateTo(StandardViewPose(v), vsync) moves there instead.
//
// Flutter: no counterpart
func (c *Camera) SetStandardView(v StandardView) { c.SetPose(c.StandardViewPose(v)) }

// --- animated moves ----------------------------------------------------------------------------------------------

// AnimateOpts are the optional parameters of Camera.AnimateTo.
//
// Flutter: no counterpart
type AnimateOpts struct {
	// Duration is the length of the move; the default is 300 ms, and Some(0) moves at once.
	Duration foundation.Opt[time.Duration]
	// Curve shapes the move; the default is animation.Curves.EaseInOut.
	Curve animation.Curve
	// OnEnd is called when the move ends: with true when the camera reached the pose, with false when another move or the
	// user's input stopped it.
	OnEnd func(completed bool)
}

// cameraAnimation is the move of a camera between two poses.
type cameraAnimation struct {
	camera     *Camera
	from, to   CameraPose
	controller *animation.AnimationController
	curved     *animation.CurvedAnimation
	onEnd      func(completed bool)
	done       bool
	handle     foundation.ListenerHandle
}

// AnimateTo moves the camera to the pose over a time, with an AnimationController on the ticker provider that the caller
// gives (a State embeds SingleTickerProvider). The camera turns by the shortest arc, and the distance changes by the same
// factor in each frame. A move that starts stops the one that is running; so does SetPose and the user's navigation.
//
// Flutter: no counterpart
func (c *Camera) AnimateTo(pose CameraPose, vsync scheduler.TickerProvider, opts ...AnimateOpts) {
	o := firstOpts(opts)
	c.StopAnimation()
	duration := o.Duration.OrElse(defaultAnimationDuration)
	if duration <= 0 || vsync == nil {
		c.setPose(pose)
		if o.OnEnd != nil {
			o.OnEnd(true)
		}
		return
	}
	pose.Orientation = pose.Orientation.Normalize()
	pose.Distance = c.clampDistance(pose.Distance)
	a := &cameraAnimation{camera: c, from: c.Pose(), to: pose, onEnd: o.OnEnd}
	a.controller = animation.NewAnimationController(vsync, animation.AnimationControllerOpts{Duration: foundation.Some(duration)})
	var curve animation.Curve = animation.Curves.EaseInOut
	if o.Curve != nil {
		curve = o.Curve
	}
	a.curved = animation.NewCurvedAnimation(a.controller, curve)
	a.handle = a.curved.AddListener(a.tick)
	a.controller.AddStatusListener(a.status)
	c.anim = a
	a.controller.Forward()
}

// Animating says whether a move is running.
//
// Flutter: no counterpart
func (c *Camera) Animating() bool { return c.anim != nil && !c.anim.done }

// StopAnimation stops a running move where it is, and releases its controller.
//
// Flutter: no counterpart
func (c *Camera) StopAnimation() {
	a := c.anim
	if a == nil {
		return
	}
	c.anim = nil
	completed := a.done
	a.curved.RemoveListener(a.handle)
	a.controller.Dispose()
	if a.onEnd != nil && !completed {
		a.onEnd(false)
	}
}

// tick sets the pose of the camera for the value of the animation.
func (a *cameraAnimation) tick() {
	if a.done {
		return
	}
	t := a.curved.Value()
	pose := CameraPose{
		Target:      a.from.Target.Lerp(a.to.Target, t),
		Distance:    a.from.Distance * math.Pow(a.to.Distance/a.from.Distance, t),
		Orientation: a.from.Orientation.Slerp(a.to.Orientation, t),
	}
	a.camera.setPose(pose)
}

// status finishes the move when the controller completes.
func (a *cameraAnimation) status(s animation.AnimationStatus) {
	if s != animation.AnimationStatusCompleted || a.done {
		return
	}
	a.done = true
	a.camera.setPose(a.to)
	if a.onEnd != nil {
		a.onEnd(true)
	}
}
