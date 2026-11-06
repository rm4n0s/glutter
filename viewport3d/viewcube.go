package viewport3d

import (
	"cmp"
	"math"
	"slices"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	defaultViewCubeSize = 104.0
	// cubeHalf is the half of the cube's edge in the picture, in the size of the widget: the corners of the cube (the longest
	// reach, root of 3 times this) stay inside the widget.
	cubeHalf = 0.27
	// cubeBand is how far from the center of a face its edge and corner cells start, in the half of its size.
	cubeBand  = 0.62
	cubeLabel = 11.0
)

// The faces of the cube, in the order of ViewCubeOpts.Labels.
const (
	cubeFront = iota
	cubeBack
	cubeRight
	cubeLeft
	cubeTop
	cubeBottom
)

var defaultCubeLabels = [6]string{"FRONT", "BACK", "RIGHT", "LEFT", "TOP", "BOTTOM"}

// cubeFaceDef is a face of the cube in the cube's own axes (x to the right of the model, y up, z to the front of it): its normal,
// and the directions that are up and to the right in the face when it is seen from outside, so that its label reads upright.
type cubeFaceDef struct {
	normal, up, right vecmath.Vec3
}

var cubeFaces = [6]cubeFaceDef{
	cubeFront:  {vecmath.Vec3Z, vecmath.Vec3Y, vecmath.Vec3X},
	cubeBack:   {vecmath.Vec3Z.Neg(), vecmath.Vec3Y, vecmath.Vec3X.Neg()},
	cubeRight:  {vecmath.Vec3X, vecmath.Vec3Y, vecmath.Vec3Z.Neg()},
	cubeLeft:   {vecmath.Vec3X.Neg(), vecmath.Vec3Y, vecmath.Vec3Z},
	cubeTop:    {vecmath.Vec3Y, vecmath.Vec3Z.Neg(), vecmath.Vec3X},
	cubeBottom: {vecmath.Vec3Y.Neg(), vecmath.Vec3Z, vecmath.Vec3X},
}

// cubeCell is a part of a face that goes to a view: the face's center (a face), a strip along an edge (an edge: the face and the
// neighbour) or a corner square (a corner: the three faces). ia and ib are -1, 0 or 1 across and up the face.
type cubeCell struct {
	face   int
	ia, ib int
	valid  bool
}

// kind is 0 for a face, 1 for an edge and 2 for a corner.
func (c cubeCell) kind() int {
	n := 0
	if c.ia != 0 {
		n++
	}
	if c.ib != 0 {
		n++
	}
	return n
}

// direction is the direction in the cube's own axes from the model to the viewer that the cell goes to: the normal of the face,
// plus the neighbouring faces' normals for an edge or a corner.
func (c cubeCell) direction() vecmath.Vec3 {
	f := cubeFaces[c.face]
	return f.normal.Add(f.right.Scale(float64(c.ia))).Add(f.up.Scale(float64(c.ib)))
}

// ViewCubeOpts are the optional parameters of NewViewCube.
//
// Flutter: no counterpart
type ViewCubeOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Size is the width and the height of the widget in logical pixels; the default is 104.
	Size float64
	// Labels are the names of the faces: front, back, right, left, top and bottom. An empty one is the default.
	Labels [6]string
	// Duration is how long the camera takes to go to a view; the default is 300 ms, and Some(0) goes at once.
	Duration foundation.Opt[time.Duration]
	// OnView is called with the direction in the world, from the model to the viewer, that a tap on the cube goes to, before the
	// camera starts to move.
	OnView func(direction vecmath.Vec3)
	// SemanticsLabel describes the cube to assistive technology; the default is "View cube".
	SemanticsLabel string
}

// ViewCube is a cube in a corner of a view that shows how the camera is turned, and goes to a view when it is tapped (SPEC §27.8,
// "Overlays"): a face goes to the standard view that looks at it, an edge to the view between two faces and a corner to the view
// between three, with the camera's animated move (the target and the distance stay). It turns with the camera, the part under the
// pointer is highlighted, and each face is in the semantics tree with a label and a tap action. NewView draws one in a corner with
// ViewOpts.ViewCube.
//
// Flutter: no counterpart
type ViewCube struct {
	widgets.StatefulWidget
	camera *Camera
	o      ViewCubeOpts
}

// NewViewCube creates the cube of a camera.
//
// Flutter: no counterpart
func NewViewCube(camera *Camera, opts ...ViewCubeOpts) *ViewCube {
	o := firstOpts(opts)
	if o.Size <= 0 {
		o.Size = defaultViewCubeSize
	}
	if o.SemanticsLabel == "" {
		o.SemanticsLabel = "View cube"
	}
	for i := range o.Labels {
		if o.Labels[i] == "" {
			o.Labels[i] = defaultCubeLabels[i]
		}
	}
	return &ViewCube{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), camera: camera, o: o}
}

// Camera is the camera that the cube follows and moves.
//
// Flutter: no counterpart
func (v *ViewCube) Camera() *Camera { return v.camera }

// CreateState creates the state of the widget.
//
// Flutter: no counterpart
func (v *ViewCube) CreateState() widgets.State { return &viewCubeState{} }

type viewCubeState struct {
	widgets.StateOf[*ViewCube]
	widgets.TickerProvider

	labels *labels
	hover  cubeCell
	// animating is whether the camera's move is the cube's.
	animating bool
}

// geometry is the cube as the camera shows it now.
func (s *viewCubeState) geometry() *cubeGeometry {
	w := s.Widget()
	return newCubeGeometry(w.camera, w.o.Size)
}

// Build makes the cube: a painting with the pointer's hover and tap over it.
func (s *viewCubeState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	base := widgets.DefaultTextStyleOf(bc).Style()
	if s.labels == nil {
		s.labels = newLabels(widgets.WidgetsBindingOf(bc).PlatformDispatcher().TextEngine(), base, cubeLabel, ui.FontWeightBold)
	}
	s.labels.setBase(base)
	painter := &cubePainter{CustomPainterBase: rendering.NewCustomPainterBase(w.camera), camera: w.camera, labels: s.labels, names: w.o.Labels, hover: s.hover,
		onGo: s.goTo}
	paint := widgets.NewCustomPaint(nil, widgets.CustomPaintOpts{Painter: painter})
	sized := widgets.NewSizedBox(paint, widgets.SizedBoxOpts{Width: foundation.Some(w.o.Size), Height: foundation.Some(w.o.Size)})
	hovered := widgets.NewMouseRegion(sized, widgets.MouseRegionOpts{
		Cursor:  services.SystemMouseCursors.Click,
		OnHover: func(e *gestures.PointerHoverEvent) { s.setHover(s.geometry().cellAt(e.LocalPosition())) },
		OnExit:  func(*gestures.PointerExitEvent) { s.setHover(cubeCell{}) },
	})
	return widgets.NewSemantics(widgets.NewGestureDetector(hovered, widgets.GestureDetectorOpts{
		OnTapUp: func(d gestures.TapUpDetails) {
			if cell := s.geometry().cellAt(d.LocalPosition); cell.valid {
				s.goTo(cell)
			}
		},
	}), widgets.SemanticsOpts{Label: foundation.Some(w.o.SemanticsLabel), Container: true, ExplicitChildNodes: true})
}

// setHover changes the highlighted cell.
func (s *viewCubeState) setHover(c cubeCell) {
	if c != s.hover {
		s.SetState(func() { s.hover = c })
	}
}

// goTo moves the camera to the view of a cell.
func (s *viewCubeState) goTo(c cubeCell) {
	w := s.Widget()
	dir := worldDirection(w.camera, c.direction()).Normalize()
	if w.o.OnView != nil {
		w.o.OnView(dir)
	}
	s.animating = true
	w.camera.AnimateTo(w.camera.ViewDirectionPose(dir), &s.TickerProvider, AnimateOpts{Duration: w.o.Duration, OnEnd: func(bool) { s.animating = false }})
}

// Dispose stops the cube's move and releases the labels.
func (s *viewCubeState) Dispose() {
	if s.animating {
		s.Widget().camera.StopAnimation()
	}
	if s.labels != nil {
		s.labels.dispose()
	}
}

// worldDirection turns a direction in the cube's own axes (x to the right of the model, y up, z to the front) into the world's.
func worldDirection(c *Camera, local vecmath.Vec3) vecmath.Vec3 {
	right := c.up.Cross(c.front).Normalize()
	return right.Scale(local.X).Add(c.up.Scale(local.Y)).Add(c.front.Scale(local.Z))
}

// cubeGeometry is the cube as a camera shows it, in the pixels of a widget of the size.
type cubeGeometry struct {
	camera *Camera
	view   vecmath.Mat4
	center ui.Offset
	half   float64
}

func newCubeGeometry(c *Camera, size float64) *cubeGeometry {
	return &cubeGeometry{camera: c, view: c.RelativeViewMatrix(), center: ui.Offset{Dx: size / 2, Dy: size / 2}, half: size * cubeHalf}
}

// proj is a vector in the cube's own axes as the picture shows it: a vector of pixels, and the depth towards the viewer.
func (g *cubeGeometry) proj(local vecmath.Vec3) (ui.Offset, float64) {
	v := g.view.TransformDirection(worldDirection(g.camera, local))
	return ui.Offset{Dx: v.X * g.half, Dy: -v.Y * g.half}, v.Z
}

// faceFrame is a face in the picture: where its center is and the vectors of pixels of a step of its width and of its height.
type faceFrame struct {
	index        int
	origin       ui.Offset
	right, up    ui.Offset
	depth        float64
	facingViewer bool
}

func (g *cubeGeometry) frame(i int) faceFrame {
	f := cubeFaces[i]
	origin, depth := g.proj(f.normal)
	right, _ := g.proj(f.right)
	up, _ := g.proj(f.up)
	return faceFrame{index: i, origin: g.center.Add(origin), right: right, up: up, depth: depth, facingViewer: depth > 1e-6}
}

// visible returns the faces that look at the viewer, the farthest first.
func (g *cubeGeometry) visible() []faceFrame {
	var out []faceFrame
	for i := range cubeFaces {
		if f := g.frame(i); f.facingViewer {
			out = append(out, f)
		}
	}
	slices.SortFunc(out, func(a, b faceFrame) int { return cmp.Compare(a.depth, b.depth) })
	return out
}

// at is the point of the picture at (a, b) in the face: a across and b up, each from -1 to 1.
func (f faceFrame) at(a, b float64) ui.Offset {
	return f.origin.Add(f.right.Scale(a, a)).Add(f.up.Scale(b, b))
}

// cellAt is the cell of the cube under a point of the picture, or an invalid one.
func (g *cubeGeometry) cellAt(p ui.Offset) cubeCell {
	for _, f := range g.visible() {
		// Solve origin + a * right + b * up = p for a and b.
		det := f.right.Dx*f.up.Dy - f.right.Dy*f.up.Dx
		if math.Abs(det) < 1e-9 {
			continue
		}
		d := p.Sub(f.origin)
		// up is the screen's vector of the face's up, so the screen's y is negated by proj already.
		a := (d.Dx*f.up.Dy - d.Dy*f.up.Dx) / det
		b := (f.right.Dx*d.Dy - f.right.Dy*d.Dx) / det
		if math.Abs(a) > 1 || math.Abs(b) > 1 {
			continue
		}
		return cubeCell{face: f.index, ia: band(a), ib: band(b), valid: true}
	}
	return cubeCell{}
}

// band says which third of a face a coordinate is in: -1 and 1 are the edge strips.
func band(x float64) int {
	switch {
	case x < -cubeBand:
		return -1
	case x > cubeBand:
		return 1
	}
	return 0
}

// cubePainter draws the cube.
type cubePainter struct {
	rendering.CustomPainterBase
	camera *Camera
	labels *labels
	names  [6]string
	hover  cubeCell
	onGo   func(cubeCell)
}

var cubeColors = struct{ face, edge, line, ink, hover, hoverInk ui.Color }{
	face: ui.NewColor(0xFFE3E7EC), edge: ui.NewColor(0xFFC1C9D2), line: ui.NewColor(0xFF7A8390),
	ink: ui.NewColor(0xFF2B3138), hover: ui.NewColor(0xFF4C8DFF), hoverInk: ui.NewColor(0xFFFFFFFF),
}

// ShouldRepaint says whether the new painter draws another cube.
func (p *cubePainter) ShouldRepaint(old rendering.CustomPainter) bool {
	o, ok := old.(*cubePainter)
	return !ok || o.camera != p.camera || o.hover != p.hover || o.names != p.names
}

// Paint draws the faces that look at the viewer, each as nine cells with its name on the middle one.
func (p *cubePainter) Paint(canvas *ui.Canvas, size ui.Size) {
	g := newCubeGeometry(p.camera, math.Min(size.Width, size.Height))
	edges := [4]float64{-1, -cubeBand, cubeBand, 1}
	fill := ui.NewPaint()
	fill.SetIsAntiAlias(true)
	stroke := ui.NewPaint()
	stroke.SetIsAntiAlias(true)
	stroke.SetStyle(ui.PaintingStyleStroke)
	stroke.SetStrokeWidth(1)
	stroke.SetStrokeJoin(ui.StrokeJoinRound)
	stroke.SetColor(cubeColors.line)
	for _, f := range g.visible() {
		for ia := -1; ia <= 1; ia++ {
			for ib := -1; ib <= 1; ib++ {
				cell := cubeCell{face: f.index, ia: ia, ib: ib, valid: true}
				a0, a1, b0, b1 := edges[ia+1], edges[ia+2], edges[ib+1], edges[ib+2]
				path := ui.NewPath()
				for k, pt := range [4][2]float64{{a0, b0}, {a1, b0}, {a1, b1}, {a0, b1}} {
					at := f.at(pt[0], pt[1])
					if k == 0 {
						path.MoveTo(at.Dx, at.Dy)
					} else {
						path.LineTo(at.Dx, at.Dy)
					}
				}
				path.Close()
				switch {
				case cell == p.hover:
					fill.SetColor(cubeColors.hover)
				case cell.kind() == 0:
					fill.SetColor(cubeColors.face)
				default:
					fill.SetColor(cubeColors.edge)
				}
				canvas.DrawPath(path, fill)
				canvas.DrawPath(path, stroke)
			}
		}
		ink := cubeColors.ink
		if p.hover.valid && p.hover.face == f.index && p.hover.kind() == 0 {
			ink = cubeColors.hoverInk
		}
		p.paintLabel(canvas, g, f, ink)
	}
}

// paintLabel draws the name of a face on it, in the plane of the face: the canvas is turned and sheared by the face's frame, so
// that the text reads upright and foreshortened as the face is.
func (p *cubePainter) paintLabel(canvas *ui.Canvas, g *cubeGeometry, f faceFrame, ink ui.Color) {
	// A step of a pixel of the label is one half-size of the face over the picture's half size of the cube.
	k := 1 / g.half
	m := ui.NewMatrix4FromList([16]float32{
		float32(f.right.Dx * k), float32(f.right.Dy * k), 0, 0,
		float32(-f.up.Dx * k), float32(-f.up.Dy * k), 0, 0,
		0, 0, 1, 0,
		float32(f.origin.Dx), float32(f.origin.Dy), 0, 1,
	})
	canvas.Save()
	canvas.Transform(m)
	p.labels.paintCentered(canvas, p.names[f.index], ink, ui.Offset{})
	canvas.Restore()
}

// SemanticsBuilder gives each face that looks at the viewer a node with its name and a tap that goes to its view.
func (p *cubePainter) SemanticsBuilder() rendering.SemanticsBuilderCallback {
	return func(size ui.Size) []rendering.CustomPainterSemantics {
		g := newCubeGeometry(p.camera, math.Min(size.Width, size.Height))
		var out []rendering.CustomPainterSemantics
		for _, f := range g.visible() {
			minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
			for _, c := range [4][2]float64{{-1, -1}, {1, -1}, {1, 1}, {-1, 1}} {
				at := f.at(c[0], c[1])
				minX, maxX = math.Min(minX, at.Dx), math.Max(maxX, at.Dx)
				minY, maxY = math.Min(minY, at.Dy), math.Max(maxY, at.Dy)
			}
			cell := cubeCell{face: f.index, valid: true}
			label := p.names[f.index]
			out = append(out, rendering.CustomPainterSemantics{
				Key:  foundation.NewValueKey(f.index),
				Rect: ui.NewRectFromLTRB(minX, minY, maxX, maxY),
				Properties: &semantics.SemanticsProperties{
					Label:         foundation.Some(label + " view"),
					TextDirection: ui.TextDirectionLTR,
					Button:        foundation.Some(true),
					OnTap:         func() { p.onGo(cell) },
				},
			})
		}
		return out
	}
}
