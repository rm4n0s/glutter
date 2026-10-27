package gpu

import (
	"fmt"
	"image"
	"math"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func cullFill() ui.Paint {
	p := ui.NewPaint()
	p.SetColor(ui.NewColor(0xFF336699))
	return *p
}

func cullStroke(width float64) ui.Paint {
	p := ui.NewPaint()
	p.SetColor(ui.NewColor(0xFF336699))
	p.SetStyle(ui.PaintingStyleStroke)
	p.SetStrokeWidth(width)
	return *p
}

// cullShapes makes one shape of each kind that fits the rectangle r, with the paint.
func cullShapes(r ui.Rect, paint ui.Paint) []struct {
	name string
	op   ui.DisplayOp
} {
	path := ui.NewPath()
	path.MoveTo(r.Left, r.Top)
	path.CubicTo(r.Right, r.Top, r.Right, r.Bottom, r.Left, r.Bottom)
	path.Close()
	rr := ui.NewRRectFromRectXY(r, 4, 4)
	hole := ui.NewRRectFromRectXY(r.Deflate(5), 2, 2)
	return []struct {
		name string
		op   ui.DisplayOp
	}{
		{"rect", ui.OpDrawRect{Rect: r, Paint: paint}},
		{"rrect", ui.OpDrawRRect{RRect: rr, Paint: paint}},
		{"drrect", ui.OpDrawDRRect{Outer: rr, Inner: hole, Paint: paint}},
		{"rsuperellipse", ui.OpDrawRSuperellipse{RSuperellipse: ui.NewRSuperellipseFromRectXY(r, 6, 6), Paint: paint}},
		{"oval", ui.OpDrawOval{Rect: r, Paint: paint}},
		{"circle", ui.OpDrawCircle{Center: ui.Offset{Dx: (r.Left + r.Right) / 2, Dy: (r.Top + r.Bottom) / 2}, Radius: (r.Right - r.Left) / 2, Paint: paint}},
		{"arc", ui.OpDrawArc{Rect: r, StartAngle: 0.3, SweepAngle: 3, Paint: paint}},
		{"path", ui.OpDrawPath{Path: path, Paint: paint}},
		{"line", ui.OpDrawLine{P1: ui.Offset{Dx: r.Left, Dy: r.Top}, P2: ui.Offset{Dx: r.Right, Dy: r.Bottom}, Paint: paint}},
		{"points", ui.OpDrawPoints{Mode: ui.PointModePolygon, Points: []ui.Offset{{Dx: r.Left, Dy: r.Top}, {Dx: r.Right, Dy: r.Bottom}, {Dx: r.Left, Dy: r.Bottom}}, Paint: paint}},
		{"shadow", ui.OpDrawShadow{Path: path, Color: ui.NewColor(0xFF000000), Elevation: 0}},
	}
}

// cullTransforms are the transforms the shapes are drawn under; the shapes are placed so that,
// under each, they are far outside the clip when they are meant to be.
var cullTransforms = []struct {
	name string
	m    func() *ui.Matrix4
}{
	{"identity", func() *ui.Matrix4 { return ui.NewMatrix4Identity() }},
	{"translation", func() *ui.Matrix4 { return ui.NewMatrix4TranslationValues(7.25, -3.5, 0) }},
	{"scale", func() *ui.Matrix4 { return ui.NewMatrix4Diagonal3Values(2, 3, 1) }},
	{"rotation", func() *ui.Matrix4 { return ui.NewMatrix4RotationZ(math.Pi / 6) }},
}

func TestOutsideClipLeavesOutShapesFarFromTheClip(t *testing.T) {
	visible := image.Rect(0, 0, 100, 100)
	// Far to the right and below: outside under each transform.
	far := ui.NewRectFromLTWH(500, 300, 40, 30)
	for _, paint := range []struct {
		name string
		p    ui.Paint
	}{{"fill", cullFill()}, {"stroke", cullStroke(6)}, {"hairline", cullStroke(0)}} {
		for _, tr := range cullTransforms {
			for _, sh := range cullShapes(far, paint.p) {
				t.Run(fmt.Sprintf("%s/%s/%s", paint.name, tr.name, sh.name), func(t *testing.T) {
					if !OutsideClip(sh.op, tr.m(), visible) {
						t.Error("a shape far from the clip was not culled")
					}
				})
			}
		}
	}
}

func TestOutsideClipKeepsShapesThatTouchTheClip(t *testing.T) {
	visible := image.Rect(0, 0, 100, 100)
	tests := []struct {
		name  string
		rect  ui.Rect
		paint ui.Paint
		m     *ui.Matrix4
	}{
		{"inside", ui.NewRectFromLTWH(10, 10, 20, 20), cullFill(), ui.NewMatrix4Identity()},
		{"the edge pixel of a fill", ui.NewRectFromLTRB(99.5, 10, 140, 50), cullFill(), ui.NewMatrix4Identity()},
		{"reaching in by its stroke", ui.NewRectFromLTRB(102, 10, 140, 50), cullStroke(8), ui.NewMatrix4Identity()},
		{"reaching in by a hairline", ui.NewRectFromLTRB(100.2, 10, 140, 50), cullStroke(0), ui.NewMatrix4Identity()},
		{"reaching in by a thin stroke", ui.NewRectFromLTRB(100.4, 10, 140, 50), cullStroke(0.2), ui.NewMatrix4Identity()},
		{"under a translation", ui.NewRectFromLTRB(120, 10, 150, 50), cullFill(), ui.NewMatrix4TranslationValues(-30, 0, 0)},
		{"under a scale", ui.NewRectFromLTRB(150, 10, 300, 50), cullFill(), ui.NewMatrix4Diagonal3Values(0.5, 1, 1)},
		{"stroke under a scale", ui.NewRectFromLTRB(52, 10, 80, 50), cullStroke(8), ui.NewMatrix4Diagonal3Values(2, 1, 1)},
		{"under a rotation", ui.NewRectFromLTRB(60, -90, 140, -60), cullFill(), ui.NewMatrix4RotationZ(math.Pi / 2)},
	}
	for _, tt := range tests {
		for _, sh := range cullShapes(tt.rect, tt.paint) {
			if sh.name == "shadow" || sh.name == "line" || sh.name == "points" || sh.name == "arc" {
				continue // their geometry is not the rectangle's edge
			}
			t.Run(tt.name+"/"+sh.name, func(t *testing.T) {
				if OutsideClip(sh.op, tt.m, visible) {
					t.Error("a shape that touches the clip was culled")
				}
			})
		}
	}
}

func TestOutsideClipDoesNotGuess(t *testing.T) {
	visible := image.Rect(0, 0, 100, 100)
	far := ui.NewRectFromLTWH(500, 300, 40, 30)
	blur := ui.NewPaint()
	blur.SetMaskFilter(ui.NewMaskFilterBlur(ui.BlurStyleNormal, 200))
	filtered := ui.NewPaint()
	filtered.SetImageFilter(ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 50, SigmaY: 50}))
	persp := ui.NewMatrix4Identity()
	persp.SetEntry(3, 0, 0.001)
	tests := []struct {
		name string
		op   ui.DisplayOp
		m    *ui.Matrix4
		want bool
	}{
		{"a mask filter blurs past the bounds", ui.OpDrawRect{Rect: far, Paint: *blur}, ui.NewMatrix4Identity(), false},
		{"an image filter blurs past the bounds", ui.OpDrawRect{Rect: far, Paint: *filtered}, ui.NewMatrix4Identity(), false},
		{"a perspective transform", ui.OpDrawRect{Rect: far, Paint: cullFill()}, persp, false},
		{"a color covers the clip", ui.OpDrawColor{Color: ui.NewColor(0xFF000000)}, ui.NewMatrix4Identity(), false},
		{"a paint covers the clip", ui.OpDrawPaint{Paint: cullFill()}, ui.NewMatrix4Identity(), false},
		{"a rectangle that is not finite", ui.OpDrawRect{Rect: ui.Rect{Left: math.NaN(), Right: 5, Bottom: 5}, Paint: cullFill()}, ui.NewMatrix4Identity(), false},
		{"a miter that is infinite", ui.OpDrawRect{Rect: far, Paint: func() ui.Paint {
			p := cullStroke(2)
			p.SetStrokeMiterLimit(math.Inf(1))
			return p
		}()}, ui.NewMatrix4Identity(), false},
		{"no points", ui.OpDrawPoints{Paint: cullStroke(2)}, ui.NewMatrix4Identity(), false},
		{"a plain rectangle far away", ui.OpDrawRect{Rect: far, Paint: cullFill()}, ui.NewMatrix4Identity(), true},
	}
	for _, tt := range tests {
		if got := OutsideClip(tt.op, tt.m, visible); got != tt.want {
			t.Errorf("%s: OutsideClip = %v, want %v", tt.name, got, tt.want)
		}
	}
	if !OutsideClip(ui.OpDrawRect{Rect: far, Paint: cullFill()}, ui.NewMatrix4Identity(), image.Rectangle{}) {
		t.Error("nothing is visible in an empty rectangle")
	}
}

func TestOutsideClipImages(t *testing.T) {
	img := ui.NewImageFromRGBA(make([]byte, 16*8*4), 16, 8)
	visible := image.Rect(0, 0, 100, 100)
	disposed := ui.NewImageFromRGBA(make([]byte, 4*4*4), 4, 4)
	disposed.Dispose()
	tests := []struct {
		name string
		op   ui.DisplayOp
		want bool
	}{
		{"far", ui.OpDrawImage{Image: img, Offset: ui.Offset{Dx: 300, Dy: 300}}, true},
		{"inside", ui.OpDrawImage{Image: img, Offset: ui.Offset{Dx: 10, Dy: 10}}, false},
		{"the corner touches", ui.OpDrawImage{Image: img, Offset: ui.Offset{Dx: 99.5, Dy: 99.5}}, false},
		{"rect far", ui.OpDrawImageRect{Image: img, Src: ui.NewRectFromLTWH(0, 0, 16, 8), Dst: ui.NewRectFromLTWH(300, 0, 32, 16)}, true},
		{"rect with the edges in the other order", ui.OpDrawImageRect{Image: img, Src: ui.NewRectFromLTWH(0, 0, 16, 8), Dst: ui.NewRectFromLTRB(332, 16, 300, 0)}, true},
		{"rect inside", ui.OpDrawImageRect{Image: img, Src: ui.NewRectFromLTWH(0, 0, 16, 8), Dst: ui.NewRectFromLTWH(50, 50, 32, 16)}, false},
		{"nine far", ui.OpDrawImageNine{Image: img, Center: ui.NewRectFromLTWH(4, 2, 8, 4), Dst: ui.NewRectFromLTWH(0, 400, 64, 32)}, true},
		{"nine inside", ui.OpDrawImageNine{Image: img, Center: ui.NewRectFromLTWH(4, 2, 8, 4), Dst: ui.NewRectFromLTWH(0, 0, 64, 32)}, false},
		{"a disposed image is left to the draw, which reports it", ui.OpDrawImage{Image: disposed, Offset: ui.Offset{Dx: 300, Dy: 300}}, false},
		{"no image is left to the draw too", ui.OpDrawImage{Offset: ui.Offset{Dx: 300, Dy: 300}}, false},
	}
	for _, tt := range tests {
		if got := OutsideClip(tt.op, ui.NewMatrix4Identity(), visible); got != tt.want {
			t.Errorf("%s: OutsideClip = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestHasherTellsSignsApart(t *testing.T) {
	hash := func(vs ...float64) Hasher {
		h := NewHasher()
		for _, v := range vs {
			h.F64(v)
		}
		return h
	}
	// Two sign flips must not cancel, as they would in a hash that mixes the top bit of a word
	// only into the top bit of the state.
	a, b := hash(1, 2, 3, 4), hash(-1, 2, 3, -4)
	if a == b {
		t.Error("flipping the signs of two coordinates did not change the hash")
	}
	seen := map[Hasher][2]int{}
	for i := 0; i < 300; i++ {
		for j := 0; j < 300; j++ {
			h := hash(float64(i), float64(j))
			if prev, ok := seen[h]; ok {
				t.Fatalf("(%d,%d) and %v have the same hash", i, j, prev)
			}
			seen[h] = [2]int{i, j}
		}
	}
}
