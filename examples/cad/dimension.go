package main

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
)

var dimensionColor = ui.NewColor(0xFFFFC857)

// dimensionPainter draws the marks of a measure and the dimension line between them over the view: it projects the points with the
// camera, and repaints when the camera moves (it listens to it).
type dimensionPainter struct {
	rendering.CustomPainterBase
	camera *viewport3d.Camera
	marks  []vecmath.Vec3
}

func newDimensionPainter(camera *viewport3d.Camera, marks []vecmath.Vec3) *dimensionPainter {
	return &dimensionPainter{CustomPainterBase: rendering.NewCustomPainterBase(camera), camera: camera, marks: append([]vecmath.Vec3(nil), marks...)}
}

// Paint draws a ring on each mark and the line between two.
func (p *dimensionPainter) Paint(canvas *ui.Canvas, size ui.Size) {
	paint := ui.NewPaint()
	paint.SetIsAntiAlias(true)
	paint.SetColor(dimensionColor)
	paint.SetStyle(ui.PaintingStyleStroke)
	paint.SetStrokeWidth(2)
	var at []ui.Offset
	for _, mark := range p.marks {
		if o, ok := p.camera.Project(mark); ok {
			at = append(at, o)
			canvas.DrawCircle(o, 6, paint)
		}
	}
	if len(at) == 2 {
		canvas.DrawLine(at[0], at[1], paint)
		// A tick across the line at each end, the way a dimension is drawn.
		d := at[1].Sub(at[0])
		if l := d.Distance(); l > 0 {
			n := ui.Offset{Dx: -d.Dy / l * 8, Dy: d.Dx / l * 8}
			for _, o := range at {
				canvas.DrawLine(o.Sub(n), o.Add(n), paint)
			}
		}
	}
}

// ShouldRepaint is true when the marks are not the same.
func (p *dimensionPainter) ShouldRepaint(old rendering.CustomPainter) bool {
	o, ok := old.(*dimensionPainter)
	if !ok || len(o.marks) != len(p.marks) || o.camera != p.camera {
		return true
	}
	for i := range p.marks {
		if o.marks[i] != p.marks[i] {
			return true
		}
	}
	return false
}

// HitTest leaves the default: a foreground painter is not hit.
func (p *dimensionPainter) HitTest(ui.Offset) foundation.Opt[bool] { return foundation.None[bool]() }
