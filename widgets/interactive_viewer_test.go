package widgets_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/interactive_viewer_test.dart.
// Ported: the first part of the viewer group (fit, boundaries, pan axis, mouse
// scaling, the interaction callbacks, the scale factor) and the static helper
// groups. The inertia, trackpad, rotation and builder cases are not ported yet.

var infMargin = painting.NewEdgeInsetsAll(math.Inf(1))

func interactiveViewerApp(c *TransformationController, size float64, opts InteractiveViewerOpts) Widget {
	opts.TransformationController = c
	return NewCenter(NewInteractiveViewer(NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(size), Height: foundation.Some(size)}), opts))
}

// pinch puts two pointers down, moves them and lifts them.
func pinch(ts *gluttertest.Tester, start1, start2, end1, end2 ui.Offset) {
	g1 := ts.StartGesture(start1)
	g2 := ts.StartGesture(start2)
	ts.Pump()
	g1.MoveTo(end1)
	g2.MoveTo(end2)
	ts.Pump()
	g1.Up()
	g2.Up()
	ts.PumpAndSettle()
}

func dragFromTo(ts *gluttertest.Tester, from, to ui.Offset) {
	g := ts.StartGesture(from)
	ts.Pump()
	g.MoveTo(to)
	ts.Pump()
	g.Up()
	ts.PumpAndSettle()
}

func isIdentity(c *TransformationController) bool { return c.Value().Equals(ui.NewMatrix4Identity()) }

func TestInteractiveViewerChildFitsInViewport(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	c := NewTransformationController(nil)
	defer c.Dispose()
	ts.PumpWidget(interactiveViewerApp(c, 200, InteractiveViewerOpts{}))
	if !isIdentity(c) {
		t.Fatal("not identity at the start")
	}
	// Panning does not work: the child fits and the boundary is tight.
	childOffset := ts.TopLeft(gluttertest.ByType[*SizedBox]())
	interior := childOffset.Add(ui.Offset{Dx: 20, Dy: 20})
	dragFromTo(ts, interior, childOffset)
	if !isIdentity(c) {
		t.Error("panned")
	}
	// Pinch to zoom works.
	pinch(ts, interior, interior.Add(ui.Offset{Dx: 10}), interior.Sub(ui.Offset{Dx: 10}), interior.Add(ui.Offset{Dx: 20}))
	if isIdentity(c) {
		t.Error("did not zoom")
	}
}

func TestInteractiveViewerBoundarySlightlyBiggerThanChild(t *testing.T) {
	const margin = 10.0
	ts := gluttertest.New(t, gluttertest.Opts{})
	c := NewTransformationController(nil)
	defer c.Dispose()
	ts.PumpWidget(interactiveViewerApp(c, 200, InteractiveViewerOpts{BoundaryMargin: painting.NewEdgeInsetsAll(margin)}))
	childOffset := ts.TopLeft(gluttertest.ByType[*SizedBox]())
	interior := childOffset.Add(ui.Offset{Dx: 20, Dy: 20})
	// Panning works only until it hits the boundary.
	dragFromTo(ts, interior, childOffset)
	if tr := c.Value().GetTranslation(); tr.X != -margin || tr.Y != -margin {
		t.Errorf("translation = %v, want -%v", tr, margin)
	}
	// Pinch to zoom also works only until it expands to the boundary.
	s1, s2 := interior, interior.Add(ui.Offset{Dx: 20})
	pinch(ts, s1, s2, s1.Add(ui.Offset{Dx: 5}), s2.Sub(ui.Offset{Dx: 5}))
	// The new scale makes the original size (200) as big as the boundary (220).
	if got := c.Value().GetMaxScaleOnAxis(); math.Abs(got-200.0/220.0) > 1e-6 {
		t.Errorf("scale = %v, want %v", got, 200.0/220.0)
	}
}

func TestInteractiveViewerChildBiggerThanViewport(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	c := NewTransformationController(nil)
	defer c.Dispose()
	ts.PumpWidget(interactiveViewerApp(c, 2000, InteractiveViewerOpts{Constrained: foundation.Some(false), ScaleEnabled: foundation.Some(false)}))
	childOffset := ts.TopLeft(gluttertest.ByType[*SizedBox]())
	interior := childOffset.Add(ui.Offset{Dx: 20, Dy: 20})
	// Moving against the boundary does not work.
	dragFromTo(ts, childOffset, interior)
	if !isIdentity(c) {
		t.Error("panned against the boundary")
	}
	// Pinching to zoom does not work: it is disabled.
	pinch(ts, interior, interior.Add(ui.Offset{Dx: 10}), interior.Sub(ui.Offset{Dx: 10}), interior.Add(ui.Offset{Dx: 20}))
	if !isIdentity(c) {
		t.Error("scaled")
	}
	// Pinching to rotate does not work either.
	pinch(ts, interior, interior.Add(ui.Offset{Dx: 10}), interior.Add(ui.Offset{Dx: 5, Dy: 5}), interior.Sub(ui.Offset{Dx: 5, Dy: 5}))
	if !isIdentity(c) {
		t.Error("rotated")
	}
	// Dragging to pan away from the boundary works.
	dragFromTo(ts, interior, childOffset)
	if isIdentity(c) {
		t.Error("did not pan")
	}
}

func TestInteractiveViewerChildHasNoDimensions(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	c := NewTransformationController(nil)
	defer c.Dispose()
	ts.PumpWidget(NewCenter(NewInteractiveViewer(NewSizedBoxShrink(nil), InteractiveViewerOpts{
		Constrained: foundation.Some(false), ScaleEnabled: foundation.Some(false), TransformationController: c})))
	childOffset := ts.TopLeft(gluttertest.ByType[*SizedBox]())
	dragFromTo(ts, childOffset, childOffset.Add(ui.Offset{Dx: 20, Dy: 20}))
	if !isIdentity(c) {
		t.Error("the transformation changed")
	}
	_ = requireErrorTag(t, takeFirst(t, ts), "InteractiveViewerChildEmpty")
}

func TestInteractiveViewerNoBoundary(t *testing.T) {
	const minScale = 0.8
	ts := gluttertest.New(t, gluttertest.Opts{})
	c := NewTransformationController(nil)
	defer c.Dispose()
	ts.PumpWidget(interactiveViewerApp(c, 200, InteractiveViewerOpts{BoundaryMargin: infMargin}))
	// Dragging works although the viewport fits around the child: there is no boundary.
	childOffset := ts.TopLeft(gluttertest.ByType[*SizedBox]())
	interior := childOffset.Add(ui.Offset{Dx: 20, Dy: 20})
	dragFromTo(ts, interior, childOffset)
	if tr := c.Value().GetTranslation(); float64(tr.X) != childOffset.Dx-interior.Dx || float64(tr.Y) != childOffset.Dy-interior.Dy {
		t.Errorf("translation = %v", tr)
	}
	// It is also possible to zoom out and see beyond the child.
	s1, s2 := interior, interior.Add(ui.Offset{Dx: 20})
	pinch(ts, s1, s2, s1.Add(ui.Offset{Dx: 5}), s2.Sub(ui.Offset{Dx: 5}))
	if got := c.Value().GetMaxScaleOnAxis(); math.Abs(got-minScale) > 1e-6 {
		t.Errorf("scale = %v, want %v", got, minScale)
	}
}

func TestInteractiveViewerPanAxis(t *testing.T) {
	tests := []struct {
		name         string
		axis         PanAxis
		wantX, wantY func(childOffset, interior ui.Offset) float64
	}{
		{"free allows panning in all directions for a diagonal gesture", PanAxisUnspecified,
			func(c, i ui.Offset) float64 { return c.Dx - i.Dx }, func(c, i ui.Offset) float64 { return c.Dy - i.Dy }},
		{"aligned allows panning in one direction only for a diagonal gesture", PanAxisAligned,
			func(c, i ui.Offset) float64 { return 0 }, func(c, i ui.Offset) float64 { return c.Dy - i.Dy }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			c := NewTransformationController(nil)
			defer c.Dispose()
			ts.PumpWidget(interactiveViewerApp(c, 200, InteractiveViewerOpts{PanAxis: tc.axis, BoundaryMargin: infMargin}))
			childOffset := ts.TopLeft(gluttertest.ByType[*SizedBox]())
			interior := childOffset.Add(ui.Offset{Dx: 20, Dy: 20})
			dragFromTo(ts, interior, childOffset)
			if tr := c.Value().GetTranslation(); float64(tr.X) != tc.wantX(childOffset, interior) || float64(tr.Y) != tc.wantY(childOffset, interior) {
				t.Errorf("translation = %v, want (%v, %v)", tr, tc.wantX(childOffset, interior), tc.wantY(childOffset, interior))
			}
		})
	}
}

func TestInteractiveViewerScaleWithMouse(t *testing.T) {
	t.Run("can scale", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		c := NewTransformationController(nil)
		defer c.Dispose()
		ts.PumpWidget(interactiveViewerApp(c, 200, InteractiveViewerOpts{}))
		ts.ScrollAt(ts.Center(gluttertest.ByType[*InteractiveViewer]()), ui.Offset{Dy: -20})
		ts.PumpAndSettle()
		if c.Value().GetMaxScaleOnAxis() <= 1 {
			t.Error("did not scale")
		}
	})
	t.Run("cannot scale when scale is disabled", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		c := NewTransformationController(nil)
		defer c.Dispose()
		ts.PumpWidget(interactiveViewerApp(c, 200, InteractiveViewerOpts{ScaleEnabled: foundation.Some(false)}))
		ts.ScrollAt(ts.Center(gluttertest.ByType[*InteractiveViewer]()), ui.Offset{Dy: -20})
		ts.PumpAndSettle()
		if c.Value().GetMaxScaleOnAxis() != 1 {
			t.Error("scaled")
		}
	})
	t.Run("returns the onInteraction properties", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		c := NewTransformationController(nil)
		defer c.Dispose()
		var focalPoint, localFocalPoint ui.Offset
		var scaleChange float64
		var velocity gestures.Velocity
		calledStart := false
		ts.PumpWidget(interactiveViewerApp(c, 200, InteractiveViewerOpts{
			OnInteractionStart: func(gestures.ScaleStartDetails) { calledStart = true },
			OnInteractionUpdate: func(d gestures.ScaleUpdateDetails) {
				scaleChange, focalPoint, localFocalPoint = d.Scale, d.FocalPoint, d.LocalFocalPoint
			},
			OnInteractionEnd: func(d gestures.ScaleEndDetails) { velocity = d.Velocity },
		}))
		center := ts.Center(gluttertest.ByType[*InteractiveViewer]())
		ts.ScrollAt(center, ui.Offset{Dy: -20})
		ts.PumpAndSettle()
		if scaleChange <= 1 || math.Abs(c.Value().GetMaxScaleOnAxis()-scaleChange) > 1e-6 || velocity != gestures.VelocityZero || !calledStart {
			t.Errorf("scaleChange %v, scale %v, velocity %v, started %v", scaleChange, c.Value().GetMaxScaleOnAxis(), velocity, calledStart)
		}
		// The focal points are in the coordinates outside of the viewer; the local
		// one is relative to the viewport.
		if focalPoint != center || localFocalPoint != (ui.Offset{Dx: 100, Dy: 100}) {
			t.Errorf("focal %v local %v", focalPoint, localFocalPoint)
		}
		// The scene point is the local focal point: the center of the scene is the
		// center of the viewport.
		if p := c.ToScene(localFocalPoint); p != (ui.Offset{Dx: 100, Dy: 100}) {
			t.Errorf("scene point = %v", p)
		}
	})
	t.Run("the scaling amount is equal forth and back", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		c := NewTransformationController(nil)
		defer c.Dispose()
		ts.PumpWidget(interactiveViewerApp(c, 1000, InteractiveViewerOpts{Constrained: foundation.Some(false), MaxScale: foundation.Some(100000.0), MinScale: foundation.Some(0.01)}))
		center := ts.Center(gluttertest.ByType[*InteractiveViewer]())
		ts.ScrollAt(center, ui.Offset{Dy: -200})
		ts.PumpAndSettle()
		if got := c.Value().GetMaxScaleOnAxis(); math.Abs(got-math.Exp(1)) > 1e-5 {
			t.Errorf("scale = %v, want %v", got, math.Exp(1))
		}
		ts.ScrollAt(center, ui.Offset{Dy: -200})
		ts.PumpAndSettle()
		if got := c.Value().GetMaxScaleOnAxis(); math.Abs(got-math.Exp(2)) > 1e-4 {
			t.Errorf("scale = %v, want %v", got, math.Exp(2))
		}
		ts.ScrollAt(center, ui.Offset{Dy: 200})
		ts.ScrollAt(center, ui.Offset{Dy: 200})
		ts.PumpAndSettle()
		if got := c.Value().GetMaxScaleOnAxis(); math.Abs(got-1) > 1e-5 {
			t.Errorf("scale = %v, want 1", got)
		}
	})
}

func TestInteractiveViewerScaleFactor(t *testing.T) {
	const scrollAmount = 30.0
	ts := gluttertest.New(t, gluttertest.Opts{})
	c := NewTransformationController(nil)
	defer c.Dispose()
	pump := func(factor float64) {
		ts.PumpWidget(interactiveViewerApp(c, 200, InteractiveViewerOpts{BoundaryMargin: infMargin, ScaleFactor: foundation.Some(factor)}))
	}
	scale := func() float64 { return c.Value().GetMaxScaleOnAxis() }
	pump(200)
	center := ts.Center(gluttertest.ByType[*InteractiveViewer]())
	ts.ScrollAt(center, ui.Offset{Dy: scrollAmount})
	ts.PumpAndSettle()
	zoomedOut := scale()
	if zoomedOut >= 1 {
		t.Fatalf("zoomed out scale = %v", zoomedOut)
	}
	ts.ScrollAt(center, ui.Offset{Dy: -scrollAmount})
	ts.PumpAndSettle()
	zoomedIn := scale()
	if zoomedIn <= zoomedOut {
		t.Fatalf("zoomed in scale = %v", zoomedIn)
	}
	// A smaller factor makes the same scroll scale more.
	c.SetValue(ui.NewMatrix4Identity())
	pump(100)
	ts.ScrollAt(center, ui.Offset{Dy: -scrollAmount})
	ts.PumpAndSettle()
	if scale() <= zoomedIn {
		t.Errorf("a smaller scale factor scaled %v, not more than %v", scale(), zoomedIn)
	}
}

func TestInteractiveViewerDoesNotCrashAtZeroArea(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(ltr(NewCenter(NewSizedBoxShrink(NewInteractiveViewer(NewText("X"))))))
	if ts.Count(gluttertest.ByType[*InteractiveViewer]()) != 1 {
		t.Error("no InteractiveViewer")
	}
}

func v3(x, y float32) ui.Vector3 { return ui.Vector3{X: x, Y: y} }

func TestInteractiveViewerGetNearestPointOnLine(t *testing.T) {
	near := func(a, b ui.Vector3, eps float32) bool {
		return math.Abs(float64(a.X-b.X)) <= float64(eps) && math.Abs(float64(a.Y-b.Y)) <= float64(eps)
	}
	tests := []struct {
		name        string
		point, a, b ui.Vector3
		want        ui.Vector3
		epsilon     float32
	}{
		{"does not modify parameters", v3(5, 5), v3(0, 0), v3(10, 0), v3(5, 0), 0},
		{"simple example", v3(0, 5), v3(0, 0), v3(5, 5), v3(2.5, 2.5), 0},
		{"closest to a", v3(-1, -1), v3(0, 0), v3(5, 5), v3(0, 0), 0},
		{"closest to b", v3(6, 6), v3(0, 0), v3(5, 5), v3(5, 5), 0},
		{"point already on the line returns the point", v3(2, 2), v3(0, 0), v3(5, 5), v3(2, 2), 0},
		{"real example", v3(-436.9, 433.6), v3(-1114, -60.3), v3(288.8, 432.7), v3(-356.8, 205.8), 0.1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			point, a, b := tc.point, tc.a, tc.b
			if got := InteractiveViewerGetNearestPointOnLine(point, a, b); !near(got, tc.want, tc.epsilon) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
			if point != tc.point || a != tc.a || b != tc.b {
				t.Error("a parameter was modified")
			}
		})
	}
}

func TestInteractiveViewerGetAxisAlignedBoundingBox(t *testing.T) {
	tests := []struct {
		name       string
		quad, want ui.Quad
	}{
		{"rectangle already axis aligned returns the rectangle",
			ui.NewQuadPoints(v3(0, 0), v3(10, 0), v3(10, 10), v3(0, 10)), ui.NewQuadPoints(v3(0, 0), v3(10, 0), v3(10, 10), v3(0, 10))},
		{"rectangle rotated by 45 degrees",
			ui.NewQuadPoints(v3(0, 5), v3(5, 10), v3(10, 5), v3(5, 0)), ui.NewQuadPoints(v3(0, 0), v3(10, 0), v3(10, 10), v3(0, 10))},
		{"rectangle rotated very slightly",
			ui.NewQuadPoints(v3(0, 1), v3(1, 11), v3(11, 9), v3(9, -1)), ui.NewQuadPoints(v3(0, -1), v3(11, -1), v3(11, 11), v3(0, 11))},
		{"example from hexagon board",
			ui.NewQuadPoints(v3(-462.7, 165.9), v3(690.6, -576.7), v3(1188.1, 196), v3(34.9, 938.6)),
			ui.NewQuadPoints(v3(-462.7, -576.7), v3(1188.1, -576.7), v3(1188.1, 938.6), v3(-462.7, 938.6))},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := InteractiveViewerGetAxisAlignedBoundingBox(tc.quad); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestInteractiveViewerPointIsInside(t *testing.T) {
	quad := ui.NewQuadPoints(v3(0, 0), v3(0, 10), v3(10, 10), v3(10, 0))
	tests := []struct {
		name  string
		point ui.Vector3
		want  bool
	}{{"inside", v3(5, 5), true}, {"outside", v3(12, 0), false}, {"on the edge", v3(0, 0), true}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := InteractiveViewerPointIsInside(tc.point, quad); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestInteractiveViewerGetNearestPointInside(t *testing.T) {
	square := ui.NewQuadPoints(v3(0, 0), v3(0, 10), v3(10, 10), v3(10, 0))
	tilted := ui.NewQuadPoints(v3(0, 0), v3(2, 10), v3(12, 12), v3(10, 2))
	tests := []struct {
		name    string
		point   ui.Vector3
		quad    ui.Quad
		want    ui.Vector3
		epsilon float64
	}{
		{"point already inside quad", v3(5, 5), square, v3(5, 5), 0},
		{"axis aligned quad", v3(5, 15), square, v3(5, 10), 0},
		{"not axis aligned quad", v3(5, 15), tilted, v3(5.8, 10.8), 0.1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := InteractiveViewerGetNearestPointInside(tc.point, tc.quad)
			if math.Abs(float64(got.X-tc.want.X)) > tc.epsilon || math.Abs(float64(got.Y-tc.want.Y)) > tc.epsilon {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
