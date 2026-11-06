package viewport3d

import (
	"cmp"
	"slices"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/widgets"
)

// The default sizes and colors of the axes.
const (
	defaultAxesSize = 72.0
	axesLabelSize   = 11.0
)

var (
	defaultAxisColors = [3]ui.Color{ui.NewColor(0xFFE5534B), ui.NewColor(0xFF57AB5A), ui.NewColor(0xFF4C8DFF)}
	defaultAxisNames  = [3]string{"X", "Y", "Z"}
)

// AxesOpts are the optional parameters of NewAxes.
//
// Flutter: no counterpart
type AxesOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Size is the width and the height of the widget in logical pixels; the default is 72.
	Size float64
	// Colors are the colors of the x, y and z axes; the defaults are red, green and blue. A zero color is the default.
	Colors [3]ui.Color
	// Labels are the names of the axes; the defaults are "X", "Y" and "Z". An empty one is the default.
	Labels [3]string
	// SemanticsLabel describes the axes to assistive technology; the default is "Axes".
	SemanticsLabel string
}

// Axes is a small picture of the world's axes as the camera sees them, for a corner of a view (SPEC §27.8, "Overlays"): a line
// from the center for each positive axis with its name at the tip, and a short dim stub for the negative one. It turns with the
// camera and costs nothing while the camera stands still. NewView draws one in a corner with ViewOpts.Axes.
//
// Flutter: no counterpart
type Axes struct {
	widgets.StatefulWidget
	camera *Camera
	o      AxesOpts
}

// NewAxes creates the axes of a camera.
//
// Flutter: no counterpart
func NewAxes(camera *Camera, opts ...AxesOpts) *Axes {
	o := firstOpts(opts)
	if o.Size <= 0 {
		o.Size = defaultAxesSize
	}
	if o.SemanticsLabel == "" {
		o.SemanticsLabel = "Axes"
	}
	return &Axes{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), camera: camera, o: o}
}

// Camera is the camera that the axes follow.
//
// Flutter: no counterpart
func (a *Axes) Camera() *Camera { return a.camera }

// CreateState creates the state of the widget.
//
// Flutter: no counterpart
func (a *Axes) CreateState() widgets.State { return &axesState{} }

type axesState struct {
	widgets.StateOf[*Axes]
	labels *labels
}

// Build makes the painting.
func (s *axesState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	base := widgets.DefaultTextStyleOf(bc).Style()
	if s.labels == nil {
		s.labels = newLabels(widgets.WidgetsBindingOf(bc).PlatformDispatcher().TextEngine(), base, axesLabelSize, ui.FontWeightBold)
	}
	s.labels.setBase(base)
	colors, names := defaultAxisColors, defaultAxisNames
	for i := range colors {
		if w.o.Colors[i] != (ui.Color{}) {
			colors[i] = w.o.Colors[i]
		}
		if w.o.Labels[i] != "" {
			names[i] = w.o.Labels[i]
		}
	}
	painter := &axesPainter{CustomPainterBase: rendering.NewCustomPainterBase(w.camera), camera: w.camera, labels: s.labels, colors: colors, names: names}
	return widgets.NewSemantics(widgets.NewSizedBox(widgets.NewCustomPaint(nil, widgets.CustomPaintOpts{Painter: painter}),
		widgets.SizedBoxOpts{Width: foundation.Some(w.o.Size), Height: foundation.Some(w.o.Size)}),
		widgets.SemanticsOpts{Label: foundation.Some(w.o.SemanticsLabel)})
}

// Dispose releases the labels.
func (s *axesState) Dispose() {
	if s.labels != nil {
		s.labels.dispose()
	}
}

// axesPainter draws the axes of the camera.
type axesPainter struct {
	rendering.CustomPainterBase
	camera *Camera
	labels *labels
	colors [3]ui.Color
	names  [3]string
}

// axisDirections are the world's axes.
var axisDirections = [3]vecmath.Vec3{vecmath.Vec3X, vecmath.Vec3Y, vecmath.Vec3Z}

// ShouldRepaint says whether the new painter draws other axes.
func (p *axesPainter) ShouldRepaint(old rendering.CustomPainter) bool {
	o, ok := old.(*axesPainter)
	return !ok || o.camera != p.camera || o.colors != p.colors || o.names != p.names
}

// axisTip is where the tip of a world axis is in the picture: the axis turned by the camera's view, in the pixels of a picture
// of the size, with the depth in the view (positive is towards the viewer).
func axisTip(view *vecmath.Mat4, axis vecmath.Vec3, length float64) (tip ui.Offset, depth float64) {
	v := view.TransformDirection(axis)
	return ui.Offset{Dx: v.X * length, Dy: -v.Y * length}, v.Z
}

// Paint draws the three axes, the farthest first.
func (p *axesPainter) Paint(canvas *ui.Canvas, size ui.Size) {
	center := ui.Offset{Dx: size.Width / 2, Dy: size.Height / 2}
	length := min(size.Width, size.Height) * 0.34
	view := p.camera.RelativeViewMatrix()
	type axis struct {
		i     int
		tip   ui.Offset
		depth float64
	}
	axes := make([]axis, 3)
	for i, a := range axisDirections {
		tip, depth := axisTip(&view, a, length)
		axes[i] = axis{i, tip, depth}
	}
	slices.SortFunc(axes, func(a, b axis) int { return cmp.Compare(a.depth, b.depth) })
	disc := ui.NewPaint()
	disc.SetIsAntiAlias(true)
	disc.SetColor(ui.NewColor(0x22808890))
	canvas.DrawCircle(center, min(size.Width, size.Height)/2-1, disc)
	for _, a := range axes {
		color := p.colors[a.i]
		stub := ui.NewPaint()
		stub.SetIsAntiAlias(true)
		stub.SetStyle(ui.PaintingStyleStroke)
		stub.SetStrokeWidth(1.5)
		stub.SetStrokeCap(ui.StrokeCapRound)
		stub.SetColor(color.WithOpacity(0.45))
		canvas.DrawLine(center, center.Sub(a.tip.Scale(0.35, 0.35)), stub)
		line := ui.NewPaint()
		line.SetIsAntiAlias(true)
		line.SetStyle(ui.PaintingStyleStroke)
		line.SetStrokeWidth(2.5)
		line.SetStrokeCap(ui.StrokeCapRound)
		line.SetColor(color)
		end := center.Add(a.tip)
		canvas.DrawLine(center, end, line)
		dot := ui.NewPaint()
		dot.SetIsAntiAlias(true)
		dot.SetColor(color)
		canvas.DrawCircle(end, 3.5, dot)
		// The name, a little beyond the tip; an axis that points at the viewer has it over the center.
		out := a.tip
		if n := out.Distance(); n > 1e-6 {
			out = out.Scale((n+9)/n, (n+9)/n)
		}
		p.labels.paintCentered(canvas, p.names[a.i], color, center.Add(out))
	}
}
