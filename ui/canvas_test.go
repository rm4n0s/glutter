package ui

import (
	"math"
	"reflect"
	"testing"

	"github.com/rm4n0s/errors"
)

// record runs fn on a new canvas and returns the canvas, its recorder and, after
// the recording ended, the picture.
func record(t *testing.T, cull *Rect, fn func(c *Canvas)) *Picture {
	t.Helper()
	rec := NewPictureRecorder()
	var opts []CanvasOpts
	if cull != nil {
		opts = append(opts, CanvasOpts{CullRect: cull})
	}
	c := NewCanvas(rec, opts...)
	fn(c)
	return rec.EndRecording()
}

func opsOf(p *Picture) []DisplayOp { return p.DisplayList().Ops() }

func TestCanvasRecordsEveryOperation(t *testing.T) {
	var paint Paint
	paint.SetColor(NewColor(0xFF6750A4))
	stroke := Paint{}
	stroke.SetStyle(PaintingStyleStroke)
	path := NewPath()
	path.AddRect(Rect{0, 0, 5, 5})
	img := &Image{}
	para := &Paragraph{}
	inner := record(t, nil, func(c *Canvas) { c.DrawCircle(Offset{1, 1}, 1, &paint) })
	verts := NewVertices(VertexModeTriangles, []Offset{{0, 0}, {1, 0}, {0, 1}})
	rr := NewRRectFromRectXY(Rect{0, 0, 10, 10}, 2, 2)
	rs := NewRSuperellipseFromRectXY(Rect{0, 0, 10, 10}, 2, 2)

	tests := []struct {
		name string
		call func(c *Canvas)
		want DisplayOp
	}{
		{"save", func(c *Canvas) { c.Save() }, OpSave{}},
		{"drawColor", func(c *Canvas) { c.DrawColor(NewColor(0xFF00FF00), BlendModeMultiply) }, OpDrawColor{Color: NewColor(0xFF00FF00), BlendMode: BlendModeMultiply}},
		{"drawColor default blend", func(c *Canvas) { c.DrawColor(NewColor(0xFF00FF00), BlendModeUnspecified) }, OpDrawColor{Color: NewColor(0xFF00FF00), BlendMode: BlendModeSrcOver}},
		{"drawPaint", func(c *Canvas) { c.DrawPaint(&paint) }, OpDrawPaint{Paint: paint}},
		{"drawLine", func(c *Canvas) { c.DrawLine(Offset{1, 2}, Offset{3, 4}, &stroke) }, OpDrawLine{P1: Offset{1, 2}, P2: Offset{3, 4}, Paint: stroke}},
		{"drawRect", func(c *Canvas) { c.DrawRect(Rect{1, 2, 3, 4}, &paint) }, OpDrawRect{Rect: Rect{1, 2, 3, 4}, Paint: paint}},
		{"drawRRect", func(c *Canvas) { c.DrawRRect(rr, &paint) }, OpDrawRRect{RRect: rr, Paint: paint}},
		{"drawDRRect", func(c *Canvas) { c.DrawDRRect(rr, rr.Deflate(1), &paint) }, OpDrawDRRect{Outer: rr, Inner: rr.Deflate(1), Paint: paint}},
		{"drawRSuperellipse", func(c *Canvas) { c.DrawRSuperellipse(rs, &paint) }, OpDrawRSuperellipse{RSuperellipse: rs, Paint: paint}},
		{"drawOval", func(c *Canvas) { c.DrawOval(Rect{0, 0, 4, 2}, &paint) }, OpDrawOval{Rect: Rect{0, 0, 4, 2}, Paint: paint}},
		{"drawCircle", func(c *Canvas) { c.DrawCircle(Offset{5, 5}, 3, &paint) }, OpDrawCircle{Center: Offset{5, 5}, Radius: 3, Paint: paint}},
		{"drawArc", func(c *Canvas) { c.DrawArc(Rect{0, 0, 4, 4}, 0.5, 1, true, &paint) }, OpDrawArc{Rect: Rect{0, 0, 4, 4}, StartAngle: 0.5, SweepAngle: 1, UseCenter: true, Paint: paint}},
		{"drawPath", func(c *Canvas) { c.DrawPath(path, &paint) }, OpDrawPath{Path: path, Paint: paint}},
		{"drawImage", func(c *Canvas) { c.DrawImage(img, Offset{1, 1}, &paint) }, OpDrawImage{Image: img, Offset: Offset{1, 1}, Paint: paint}},
		{"drawImageRect", func(c *Canvas) { c.DrawImageRect(img, Rect{0, 0, 2, 2}, Rect{0, 0, 4, 4}, &paint) }, OpDrawImageRect{Image: img, Src: Rect{0, 0, 2, 2}, Dst: Rect{0, 0, 4, 4}, Paint: paint}},
		{"drawImageNine", func(c *Canvas) { c.DrawImageNine(img, Rect{1, 1, 2, 2}, Rect{0, 0, 9, 9}, &paint) }, OpDrawImageNine{Image: img, Center: Rect{1, 1, 2, 2}, Dst: Rect{0, 0, 9, 9}, Paint: paint}},
		{"drawPicture", func(c *Canvas) { c.DrawPicture(inner) }, OpDrawPicture{Picture: inner}},
		{"drawParagraph", func(c *Canvas) { c.DrawParagraph(para, Offset{3, 3}) }, OpDrawParagraph{Paragraph: para, Offset: Offset{3, 3}}},
		{"drawPoints", func(c *Canvas) { c.DrawPoints(PointModeLines, []Offset{{0, 0}, {1, 1}}, &stroke) }, OpDrawPoints{Mode: PointModeLines, Points: []Offset{{0, 0}, {1, 1}}, Paint: stroke}},
		{"drawPoints default mode", func(c *Canvas) { c.DrawPoints(PointModeUnspecified, nil, &stroke) }, OpDrawPoints{Mode: PointModePoints, Paint: stroke}},
		{"drawVertices", func(c *Canvas) { c.DrawVertices(verts, BlendModeDst, &paint) }, OpDrawVertices{Vertices: verts, BlendMode: BlendModeDst, Paint: paint}},
		{"drawShadow", func(c *Canvas) { c.DrawShadow(path, NewColor(0xFF000000), 4, true) }, OpDrawShadow{Path: path, Color: NewColor(0xFF000000), Elevation: 4, TransparentOccluder: true}},
		{"clipRect", func(c *Canvas) { c.ClipRect(Rect{0, 0, 5, 5}) }, OpClipRect{Rect: Rect{0, 0, 5, 5}, Op: ClipOpIntersect, AntiAlias: true}},
		{"clipRect difference without antialias", func(c *Canvas) {
			c.ClipRect(Rect{0, 0, 5, 5}, ClipRectOpts{ClipOp: ClipOpDifference, DoAntiAlias: Ptr(false)})
		}, OpClipRect{Rect: Rect{0, 0, 5, 5}, Op: ClipOpDifference, AntiAlias: false}},
		{"clipRRect", func(c *Canvas) { c.ClipRRect(rr) }, OpClipRRect{RRect: rr, AntiAlias: true}},
		{"clipRSuperellipse", func(c *Canvas) { c.ClipRSuperellipse(rs, ClipOpts{DoAntiAlias: Ptr(false)}) }, OpClipRSuperellipse{RSuperellipse: rs}},
		{"clipPath", func(c *Canvas) { c.ClipPath(path) }, OpClipPath{Path: path, AntiAlias: true}},
		{"translate", func(c *Canvas) { c.Translate(3, 4) }, OpTransform{Matrix: *NewMatrix4TranslationValues(3, 4, 0)}},
		{"scale", func(c *Canvas) { c.Scale(2, 3) }, OpTransform{Matrix: *NewMatrix4Diagonal3Values(2, 3, 1)}},
		{"scale uniform", func(c *Canvas) { c.ScaleUniform(2) }, OpTransform{Matrix: *NewMatrix4Diagonal3Values(2, 2, 1)}},
		{"rotate", func(c *Canvas) { c.Rotate(0.5) }, OpTransform{Matrix: *NewMatrix4RotationZ(0.5)}},
		{"skew", func(c *Canvas) { c.Skew(0.25, 0.5) }, OpTransform{Matrix: Matrix4{[16]float32{1, 0.5, 0, 0, 0.25, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}}}},
		{"transform", func(c *Canvas) { c.Transform(NewMatrix4RotationX(1)) }, OpTransform{Matrix: *NewMatrix4RotationX(1)}},
	}
	for _, tt := range tests {
		pic := record(t, nil, tt.call)
		ops := opsOf(pic)
		if len(ops) != 1 {
			t.Errorf("%s: %d ops %v", tt.name, len(ops), ops)
			continue
		}
		if !reflect.DeepEqual(ops[0], tt.want) {
			t.Errorf("%s:\n got  %#v\n want %#v", tt.name, ops[0], tt.want)
		}
	}
}

func TestCanvasSavesCopies(t *testing.T) {
	path := NewPath()
	path.AddRect(Rect{0, 0, 5, 5})
	var paint Paint
	paint.SetColor(NewColor(0xFF0000FF))
	pts := []Offset{{0, 0}, {1, 1}}
	pic := record(t, nil, func(c *Canvas) {
		c.DrawPath(path, &paint)
		c.ClipPath(path)
		c.DrawShadow(path, Color{}, 1, false)
		c.DrawPoints(PointModePolygon, pts, &paint)
		// Changing what was passed afterwards must not change the picture.
		path.LineTo(99, 99)
		paint.SetColor(NewColor(0xFF00FF00))
		pts[0] = Offset{50, 50}
	})
	ops := opsOf(pic)
	if got := ops[0].(OpDrawPath); verbString(got.Path) != "MLLLZ" || got.Paint.Color() != NewColor(0xFF0000FF) {
		t.Errorf("DrawPath recorded %s %v", verbString(got.Path), got.Paint.Color())
	}
	if verbString(ops[1].(OpClipPath).Path) != "MLLLZ" || verbString(ops[2].(OpDrawShadow).Path) != "MLLLZ" {
		t.Error("the clip and shadow paths must be copies")
	}
	if ops[3].(OpDrawPoints).Points[0] != (Offset{0, 0}) {
		t.Error("the points must be copied")
	}
}

func TestCanvasEmptyShapesAreNotRecordedWhenFilled(t *testing.T) {
	var fill, stroke Paint
	stroke.SetStyle(PaintingStyleStroke)
	tests := []struct {
		name string
		call func(c *Canvas)
		want int
	}{
		{"an empty filled rect", func(c *Canvas) { c.DrawRect(Rect{5, 5, 5, 10}, &fill) }, 0},
		{"an inverted empty rect is sorted but still empty", func(c *Canvas) { c.DrawRect(Rect{5, 10, 5, 5}, &fill) }, 0},
		{"an empty stroked rect", func(c *Canvas) { c.DrawRect(Rect{5, 5, 5, 10}, &stroke) }, 1},
		{"an empty filled oval", func(c *Canvas) { c.DrawOval(Rect{0, 0, 0, 0}, &fill) }, 0},
		{"an empty stroked oval", func(c *Canvas) { c.DrawOval(Rect{0, 0, 0, 0}, &stroke) }, 1},
		{"a normal rect", func(c *Canvas) { c.DrawRect(Rect{0, 0, 1, 1}, &fill) }, 1},
		{"an empty circle is kept", func(c *Canvas) { c.DrawCircle(Offset{}, 0, &fill) }, 1},
	}
	for _, tt := range tests {
		if got := len(opsOf(record(t, nil, tt.call))); got != tt.want {
			t.Errorf("%s: %d ops, want %d", tt.name, got, tt.want)
		}
	}
	// Rects are recorded with their edges in order.
	pic := record(t, nil, func(c *Canvas) { c.DrawRect(Rect{10, 10, 10, 5}, &stroke) })
	if got := opsOf(pic)[0].(OpDrawRect).Rect; got != (Rect{10, 5, 10, 10}) {
		t.Errorf("sorted rect = %v", got)
	}
	// A clip with an empty rectangle is still recorded, sorted.
	clip := record(t, nil, func(c *Canvas) { c.ClipRect(Rect{5, 8, 5, 2}) })
	if got := opsOf(clip)[0].(OpClipRect).Rect; got != (Rect{5, 2, 5, 8}) {
		t.Errorf("clip rect = %v", got)
	}
}

func TestCanvasIdentityTransformsAreNotRecorded(t *testing.T) {
	pic := record(t, nil, func(c *Canvas) {
		c.Translate(0, 0)
		c.Scale(1, 1)
		c.Rotate(0)
		c.Rotate(2 * math.Pi)
		c.Skew(0, 0)
		c.Transform(NewMatrix4Identity())
		c.Translate(math.NaN(), 1)
		c.Scale(math.Inf(1), 1)
		c.Rotate(math.NaN())
	})
	if ops := opsOf(pic); len(ops) != 0 {
		t.Errorf("%v", ops)
	}
}

func TestCanvasSaveCount(t *testing.T) {
	rec := NewPictureRecorder()
	c := NewCanvas(rec)
	steps := []struct {
		name string
		do   func()
		want int
	}{
		{"new", func() {}, 1},
		{"save", c.Save, 2},
		{"save", c.Save, 3},
		{"saveLayer", func() { c.SaveLayer(nil, nil) }, 4},
		{"saveLayer with bounds", func() { c.SaveLayer(&Rect{0, 0, 10, 10}, &Paint{}) }, 5},
		{"restore", c.Restore, 4},
		{"restoreToCount 2", func() { c.RestoreToCount(2) }, 2},
		{"restoreToCount to more does nothing", func() { c.RestoreToCount(9) }, 2},
		{"restoreToCount 0 is 1", func() { c.RestoreToCount(0) }, 1},
		{"restore at the bottom does nothing", c.Restore, 1},
		{"restoreToCount negative", func() { c.RestoreToCount(-5) }, 1},
	}
	for _, s := range steps {
		s.do()
		if got := c.GetSaveCount(); got != s.want {
			t.Errorf("after %s: save count %d, want %d", s.name, got, s.want)
		}
	}
	pic := rec.EndRecording()
	// 5 saves, then restores that did something: 1 + 2 + 1 + ... count the ops.
	saves, restores := 0, 0
	for _, op := range opsOf(pic) {
		switch op.(type) {
		case OpSave, OpSaveLayer:
			saves++
		case OpRestore:
			restores++
		}
	}
	if saves != 4 || restores != 4 {
		t.Errorf("recorded %d saves and %d restores, want 4 and 4 (restores that do nothing are not recorded)", saves, restores)
	}
}

func TestCanvasTransformStack(t *testing.T) {
	rec := NewPictureRecorder()
	c := NewCanvas(rec)
	if !c.GetTransform().IsIdentity() {
		t.Error("a new canvas has the identity transform")
	}
	c.Translate(10, 20)
	c.Save()
	c.Scale(2, 3)
	if got := c.GetTransform().TransformPoint(Offset{1, 1}); got != (Offset{12, 23}) {
		t.Errorf("scaled then translated: %v", got)
	}
	// The returned matrix is a copy.
	c.GetTransform().storage[12] = 999
	if c.GetTransform().storage[12] != 10 {
		t.Error("GetTransform must return a copy")
	}
	c.Restore()
	if got := c.GetTransform().TransformPoint(Offset{1, 1}); got != (Offset{11, 21}) {
		t.Errorf("after restore: %v", got)
	}
	// Order: the last transform applies first to a point.
	d := NewCanvas(NewPictureRecorder())
	d.Translate(100, 0)
	d.Rotate(math.Pi / 2)
	if got := d.GetTransform().TransformPoint(Offset{1, 0}); !near(got.Dx, 100, 1e-5) || !near(got.Dy, 1, 1e-5) {
		t.Errorf("rotate then translate: %v", got)
	}
	e := NewCanvas(NewPictureRecorder())
	e.Skew(1, 0) // x' = x + y
	if got := e.GetTransform().TransformPoint(Offset{0, 2}); got != (Offset{2, 2}) {
		t.Errorf("skew: %v", got)
	}
	f := NewCanvas(NewPictureRecorder())
	f.Skew(0, 1) // y' = y + x
	if got := f.GetTransform().TransformPoint(Offset{3, 0}); got != (Offset{3, 3}) {
		t.Errorf("skew y: %v", got)
	}
}

func TestCanvasClipBounds(t *testing.T) {
	cull := Rect{0, 0, 200, 100}
	tests := []struct {
		name       string
		do         func(c *Canvas)
		dest       Rect
		local      Rect
		localEps   float64
		wantErrors bool
	}{
		{"nothing clipped", func(*Canvas) {}, Rect{0, 0, 200, 100}, Rect{0, 0, 200, 100}, 0, false},
		{"a rect", func(c *Canvas) { c.ClipRect(Rect{10, 20, 60, 70}) }, Rect{10, 20, 60, 70}, Rect{10, 20, 60, 70}, 0, false},
		{"two rects", func(c *Canvas) { c.ClipRect(Rect{10, 20, 60, 70}); c.ClipRect(Rect{40, 0, 100, 50}) }, Rect{40, 20, 60, 50}, Rect{40, 20, 60, 50}, 0, false},
		{"a rect past the cull rect", func(c *Canvas) { c.ClipRect(Rect{-50, -50, 300, 300}) }, Rect{0, 0, 200, 100}, Rect{0, 0, 200, 100}, 0, false},
		{"disjoint rects clip everything", func(c *Canvas) { c.ClipRect(Rect{0, 0, 10, 10}); c.ClipRect(Rect{50, 50, 60, 60}) }, Rect{}, Rect{}, 0, false},
		{"an empty rect clips everything", func(c *Canvas) { c.ClipRect(Rect{5, 5, 5, 50}) }, Rect{}, Rect{}, 0, false},
		{"translated", func(c *Canvas) { c.Translate(30, 10); c.ClipRect(Rect{0, 0, 20, 20}) }, Rect{30, 10, 50, 30}, Rect{0, 0, 20, 20}, 1e-6, false},
		{"scaled", func(c *Canvas) { c.Scale(2, 2); c.ClipRect(Rect{5, 5, 15, 15}) }, Rect{10, 10, 30, 30}, Rect{5, 5, 15, 15}, 1e-6, false},
		{"translate after the clip moves the local bounds", func(c *Canvas) { c.ClipRect(Rect{10, 10, 50, 50}); c.Translate(5, 5) }, Rect{10, 10, 50, 50}, Rect{5, 5, 45, 45}, 1e-6, false},
		{"scale after the clip", func(c *Canvas) { c.ClipRect(Rect{10, 10, 50, 50}); c.Scale(2, 2) }, Rect{10, 10, 50, 50}, Rect{5, 5, 25, 25}, 1e-6, false},
		{"a rotated clip is bounded by its box, and by the cull rect", func(c *Canvas) { c.Rotate(math.Pi / 4); c.ClipRect(Rect{0, 0, 10, 10}) }, Rect{0, 0, 7.0710678, 14.1421356}, Rect{0, -5, 15, 10}, 1e-4, true},
		{"an rrect", func(c *Canvas) { c.ClipRRect(NewRRectFromRectXY(Rect{10, 10, 50, 50}, 5, 5)) }, Rect{10, 10, 50, 50}, Rect{10, 10, 50, 50}, 0, false},
		{"a superellipse", func(c *Canvas) { c.ClipRSuperellipse(NewRSuperellipseFromRectXY(Rect{10, 10, 50, 50}, 5, 5)) }, Rect{10, 10, 50, 50}, Rect{10, 10, 50, 50}, 0, false},
		{"a path", func(c *Canvas) {
			p := NewPath()
			p.MoveTo(20, 20)
			p.LineTo(80, 30)
			p.LineTo(40, 90)
			c.ClipPath(p)
		}, Rect{20, 20, 80, 90}, Rect{20, 20, 80, 90}, 0, false},
		{"an empty path clips everything", func(c *Canvas) { c.ClipPath(NewPath()) }, Rect{}, Rect{}, 0, false},
		{"difference with a rect that takes a side", func(c *Canvas) {
			c.ClipRect(Rect{0, 0, 80, 100}, ClipRectOpts{ClipOp: ClipOpDifference})
		}, Rect{80, 0, 200, 100}, Rect{80, 0, 200, 100}, 0, false},
		{"difference with a rect that takes the bottom", func(c *Canvas) {
			c.ClipRect(Rect{-10, 60, 300, 200}, ClipRectOpts{ClipOp: ClipOpDifference})
		}, Rect{0, 0, 200, 60}, Rect{0, 0, 200, 60}, 0, false},
		{"difference with a rect in the middle changes nothing", func(c *Canvas) {
			c.ClipRect(Rect{50, 20, 80, 60}, ClipRectOpts{ClipOp: ClipOpDifference})
		}, Rect{0, 0, 200, 100}, Rect{0, 0, 200, 100}, 0, false},
		{"difference with a rect that covers everything", func(c *Canvas) {
			c.ClipRect(Rect{-1, -1, 500, 500}, ClipRectOpts{ClipOp: ClipOpDifference})
		}, Rect{}, Rect{}, 0, false},
		{"difference with a rotated rect changes nothing", func(c *Canvas) {
			c.Rotate(0.3)
			c.ClipRect(Rect{-500, -500, 500, 500}, ClipRectOpts{ClipOp: ClipOpDifference})
		}, Rect{0, 0, 200, 100}, Rect{}, -1, false},
		{"a layer with bounds clips to them", func(c *Canvas) { c.SaveLayer(&Rect{10, 10, 40, 40}, nil) }, Rect{10, 10, 40, 40}, Rect{10, 10, 40, 40}, 0, false},
		{"a layer without bounds does not", func(c *Canvas) { c.SaveLayer(nil, nil) }, Rect{0, 0, 200, 100}, Rect{0, 0, 200, 100}, 0, false},
	}
	for _, tt := range tests {
		c := NewCanvas(NewPictureRecorder(), CanvasOpts{CullRect: &cull})
		tt.do(c)
		if got := c.GetDestinationClipBounds(); !rectClose(got, tt.dest, math.Max(tt.localEps, 1e-9)) {
			t.Errorf("%s: destination %v, want %v", tt.name, got, tt.dest)
		}
		if tt.localEps >= 0 {
			if got := c.GetLocalClipBounds(); !rectClose(got, tt.local, math.Max(tt.localEps, 1e-9)) {
				t.Errorf("%s: local %v, want %v", tt.name, got, tt.local)
			}
		}
	}
}

func rectClose(a, b Rect, eps float64) bool {
	return near(a.Left, b.Left, eps) && near(a.Top, b.Top, eps) && near(a.Right, b.Right, eps) && near(a.Bottom, b.Bottom, eps)
}

func TestCanvasClipIsRestored(t *testing.T) {
	c := NewCanvas(NewPictureRecorder(), CanvasOpts{CullRect: &Rect{0, 0, 100, 100}})
	c.ClipRect(Rect{0, 0, 50, 50})
	c.Save()
	c.Translate(10, 10)
	c.ClipRect(Rect{0, 0, 10, 10})
	if got := c.GetDestinationClipBounds(); got != (Rect{10, 10, 20, 20}) {
		t.Errorf("inside: %v", got)
	}
	c.Restore()
	if got := c.GetDestinationClipBounds(); got != (Rect{0, 0, 50, 50}) {
		t.Errorf("after restore: %v", got)
	}
	if !c.GetTransform().IsIdentity() {
		t.Error("the transform must be restored too")
	}
	// A layer's bounds are undone by the restore.
	c.SaveLayer(&Rect{0, 0, 5, 5}, nil)
	c.Restore()
	if got := c.GetDestinationClipBounds(); got != (Rect{0, 0, 50, 50}) {
		t.Errorf("after restoring a layer: %v", got)
	}
}

func TestCanvasDefaultCullRectIsLargest(t *testing.T) {
	c := NewCanvas(NewPictureRecorder())
	if got := c.GetDestinationClipBounds(); got != RectLargest {
		t.Errorf("got %v, want %v", got, RectLargest)
	}
}

func TestCanvasNonInvertibleTransformHasNoLocalBounds(t *testing.T) {
	c := NewCanvas(NewPictureRecorder(), CanvasOpts{CullRect: &Rect{0, 0, 10, 10}})
	c.Scale(0, 1)
	if got := c.GetLocalClipBounds(); got != (Rect{}) {
		t.Errorf("got %v", got)
	}
}

func TestCanvasAfterEndRecording(t *testing.T) {
	rec := NewPictureRecorder()
	if rec.IsRecording() {
		t.Error("a new recorder is not recording")
	}
	c := NewCanvas(rec)
	if !rec.IsRecording() || c.String() != "Canvas(recording: true)" || rec.String() != "PictureRecorder(recording: true)" {
		t.Errorf("recording state: %v %s %s", rec.IsRecording(), c, rec)
	}
	c.DrawColor(Color{A: 1}, BlendModeSrc)
	pic := rec.EndRecording()
	if rec.IsRecording() || c.String() != "Canvas(recording: false)" {
		t.Error("recording ended")
	}
	// The canvas no longer records, but still tracks its state.
	c.DrawColor(Color{A: 1}, BlendModeSrc)
	c.Save()
	if len(opsOf(pic)) != 1 {
		t.Errorf("the picture must not change: %v", opsOf(pic))
	}
	if c.GetSaveCount() != 2 {
		t.Errorf("save count %d", c.GetSaveCount())
	}
	// The recorder can be used again.
	NewCanvas(rec).DrawColor(Color{A: 1}, BlendModeDst)
	if second := rec.EndRecording(); len(opsOf(second)) != 1 {
		t.Errorf("second picture: %v", opsOf(second))
	}
}

func TestCanvasPanics(t *testing.T) {
	tests := []struct {
		name  string
		do    func()
		tag   string
		route string
	}{
		{"a busy recorder", func() {
			rec := NewPictureRecorder()
			NewCanvas(rec)
			NewCanvas(rec)
		}, "CanvasRecorderBusy", "NewCanvas.CanvasRecorderBusy"},
		{"ending without recording", func() { NewPictureRecorder().EndRecording() }, "PictureRecorderNotRecording", "PictureRecorder.EndRecording.PictureRecorderNotRecording"},
		{"ending twice", func() {
			rec := NewPictureRecorder()
			NewCanvas(rec)
			rec.EndRecording()
			rec.EndRecording()
		}, "PictureRecorderNotRecording", "PictureRecorder.EndRecording.PictureRecorderNotRecording"},
		{"atlas transforms and rects", func() {
			NewCanvas(NewPictureRecorder()).DrawAtlas(&Image{}, []RSTransform{{}}, nil, nil, BlendModeUnspecified, nil, nil)
		}, "AtlasTransformCountMismatch", "Canvas.DrawAtlas.AtlasTransformCountMismatch"},
		{"atlas colors", func() {
			NewCanvas(NewPictureRecorder()).DrawAtlas(&Image{}, []RSTransform{{}}, []Rect{{}}, []Color{{}, {}}, BlendModeUnspecified, nil, nil)
		}, "AtlasColorCountMismatch", "Canvas.DrawAtlas.AtlasColorCountMismatch"},
		{"two canvas options", func() {
			NewCanvas(NewPictureRecorder(), CanvasOpts{}, CanvasOpts{})
		}, "MultipleOptions", "NewCanvas->oneOpts[...].MultipleOptions"},
	}
	for _, tt := range tests {
		func() {
			defer func() {
				r := recover()
				if r == nil {
					t.Errorf("%s: no panic", tt.name)
					return
				}
				appErr, ok := errors.FromError(recoverError(t, r))
				if !ok || appErr.Tag != tt.tag || !appErr.HasRoute(tt.route) {
					t.Errorf("%s: got %v, want %s via %s", tt.name, appErr, tt.tag, tt.route)
				}
			}()
			tt.do()
		}()
	}
}

func TestCanvasDrawAtlas(t *testing.T) {
	cull := Rect{0, 0, 9, 9}
	pic := record(t, nil, func(c *Canvas) {
		c.DrawAtlas(&Image{}, []RSTransform{{1, 0, 0, 0}, {1, 0, 5, 5}}, []Rect{{0, 0, 2, 2}, {2, 2, 4, 4}}, []Color{NewColor(0xFFFF0000), NewColor(0xFF00FF00)}, BlendModeModulate, &cull, nil)
		c.DrawAtlas(&Image{}, nil, nil, nil, BlendModeUnspecified, nil, nil)
	})
	ops := opsOf(pic)
	a := ops[0].(OpDrawAtlas)
	if len(a.Transforms) != 2 || len(a.Colors) != 2 || a.BlendMode != BlendModeModulate || !a.HasCullRect || a.CullRect != cull {
		t.Errorf("atlas op = %+v", a)
	}
	b := ops[1].(OpDrawAtlas)
	if b.BlendMode != BlendModeSrc || b.HasCullRect || b.Colors != nil {
		t.Errorf("empty atlas op = %+v", b)
	}
}

func TestPictureBasics(t *testing.T) {
	cull := Rect{0, 0, 40, 30}
	pic := record(t, &cull, func(c *Canvas) {
		path := NewPath()
		path.AddOval(Rect{0, 0, 10, 10})
		c.DrawPath(path, &Paint{})
		c.DrawPoints(PointModePoints, make([]Offset, 100), &Paint{})
	})
	if pic.CullRect() != cull || pic.DisplayList().CullRect() != cull {
		t.Errorf("cull rect = %v", pic.CullRect())
	}
	small := record(t, nil, func(c *Canvas) { c.DrawColor(Color{}, BlendModeSrc) })
	if pic.ApproximateBytesUsed() <= small.ApproximateBytesUsed() || small.ApproximateBytesUsed() <= 0 {
		t.Errorf("sizes: %d, %d", pic.ApproximateBytesUsed(), small.ApproximateBytesUsed())
	}
	nested := record(t, nil, func(c *Canvas) { c.DrawPicture(pic) })
	if nested.ApproximateBytesUsed() < pic.ApproximateBytesUsed() {
		t.Error("a picture that draws another is at least as big")
	}
	pic.Dispose()
	if pic.String() != "Picture" {
		t.Error("String wrong")
	}
	for _, f := range []func(int, int, ...ToImageOpts) (*Image, error){pic.ToImage, pic.ToImageSync} {
		if _, err := f(10, 10); err == nil {
			t.Error("ToImage without a rasterizer is an error")
		} else if appErr, ok := errors.FromError(err); !ok || appErr.Tag != "RasterizerMissing" {
			t.Errorf("error = %v", err)
		}
		if _, err := f(0, 10); err == nil {
			t.Error("a zero width is an error")
		} else if appErr, ok := errors.FromError(err); !ok || appErr.Tag != "ImageSizeInvalid" {
			t.Errorf("error = %v", err)
		}
		fake := &fakeRasterizer{}
		img, err := f(30, 20, ToImageOpts{Rasterizer: fake})
		if err != nil || img == nil || img.Width() != 30 || img.Height() != 20 {
			t.Fatalf("ToImage with a rasterizer = %v, %v", img, err)
		}
		layers := fake.scene.Root().Children()
		if len(layers) != 1 {
			t.Fatalf("the rasterizer got %d layers", len(layers))
		}
		if pl, ok := layers[0].(*PictureLayer); !ok || pl.Picture != pic || pl.Offset != (Offset{}) {
			t.Errorf("the rasterizer got %+v, want the picture at the origin", layers[0])
		}
	}
	// The rasterizer's own error comes back unchanged.
	if _, err := pic.ToImage(5, 5, ToImageOpts{Rasterizer: &fakeRasterizer{err: errors.New("RasterFailed", "no")}}); err == nil {
		t.Error("the rasterizer's error was lost")
	} else if appErr, ok := errors.FromError(err); !ok || appErr.Tag != "RasterFailed" {
		t.Errorf("error = %v", err)
	}
}

// fakeRasterizer records the scene it is asked to draw and returns a blank
// image of the size asked for.
type fakeRasterizer struct {
	scene *Scene
	err   error
}

func (f *fakeRasterizer) RasterizeScene(scene *Scene, width, height int) (*Image, error) {
	f.scene = scene
	if f.err != nil {
		return nil, f.err
	}
	return NewImageFromRGBA(make([]byte, 4*width*height), width, height), nil
}
