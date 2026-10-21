package gpu

import (
	"image"
	"math"
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestPartsOf(t *testing.T) {
	tests := []struct {
		name  string
		rect  ui.Rect
		radii [4]ui.Radius
		want  RRectParts
		ok    bool
	}{
		{"plain", ui.NewRectFromLTWH(10, 20, 40, 60), [4]ui.Radius{}, RRectParts{CX: 30, CY: 50, HX: 20, HY: 30}, true},
		{"edges swapped", ui.Rect{Left: 50, Top: 80, Right: 10, Bottom: 20}, [4]ui.Radius{}, RRectParts{CX: 30, CY: 50, HX: 20, HY: 30}, true},
		{"a radius with a zero side is square", ui.NewRectFromLTWH(0, 0, 40, 40), [4]ui.Radius{{X: 5, Y: 0}, {X: 0, Y: 5}, {X: -1, Y: 5}, {X: 5, Y: 5}},
			RRectParts{CX: 20, CY: 20, HX: 20, HY: 20, R: [4]ui.Radius{{}, {}, {}, {X: 5, Y: 5}}}, true},
		{"radii scaled to fit", ui.NewRectFromLTWH(0, 0, 40, 100), [4]ui.Radius{{X: 40, Y: 40}, {X: 40, Y: 40}, {X: 40, Y: 40}, {X: 40, Y: 40}},
			RRectParts{CX: 20, CY: 50, HX: 20, HY: 50, R: [4]ui.Radius{{X: 20, Y: 20}, {X: 20, Y: 20}, {X: 20, Y: 20}, {X: 20, Y: 20}}}, true},
		{"scaled by the tightest side", ui.NewRectFromLTWH(0, 0, 100, 30), [4]ui.Radius{{X: 60, Y: 30}, {X: 60, Y: 10}, {X: 10, Y: 10}, {X: 10, Y: 10}},
			RRectParts{CX: 50, CY: 15, HX: 50, HY: 15, R: [4]ui.Radius{{X: 45, Y: 22.5}, {X: 45, Y: 7.5}, {X: 7.5, Y: 7.5}, {X: 7.5, Y: 7.5}}}, true},
		{"infinite", ui.Rect{Left: 0, Top: 0, Right: math.Inf(1), Bottom: 1}, [4]ui.Radius{}, RRectParts{}, false},
		{"NaN", ui.Rect{Left: math.NaN(), Top: 0, Right: 1, Bottom: 1}, [4]ui.Radius{}, RRectParts{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := PartsOf(tt.rect, tt.radii)
			if ok != tt.ok {
				t.Fatalf("ok = %v", ok)
			}
			if !ok {
				return
			}
			near := func(a, b float64) bool { return math.Abs(a-b) < 1e-9 }
			if !near(got.CX, tt.want.CX) || !near(got.CY, tt.want.CY) || !near(got.HX, tt.want.HX) || !near(got.HY, tt.want.HY) {
				t.Errorf("parts = %+v, want %+v", got, tt.want)
			}
			for i := range got.R {
				if !near(got.R[i].X, tt.want.R[i].X) || !near(got.R[i].Y, tt.want.R[i].Y) {
					t.Errorf("radii = %v, want %v", got.R, tt.want.R)
					break
				}
			}
		})
	}
}

func TestAffine(t *testing.T) {
	tests := []struct {
		name       string
		m          Affine2D
		invertible bool
		similarity bool
	}{
		{"identity", Affine2D{A: 1, D: 1}, true, true},
		{"uniform scale", Affine2D{A: 2, D: 2, E: 5}, true, true},
		{"rotation", Affine2D{A: math.Cos(0.5), B: math.Sin(0.5), C: -math.Sin(0.5), D: math.Cos(0.5)}, true, true},
		{"flip", Affine2D{A: -1, D: 1}, true, true},
		{"stretch", Affine2D{A: 2, D: 1}, true, false},
		{"skew", Affine2D{A: 1, C: 0.5, D: 1}, true, false},
		{"singular", Affine2D{A: 1}, false, false},
		{"NaN", Affine2D{A: math.NaN(), D: 1}, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv, ok := tt.m.Invert()
			if ok != tt.invertible {
				t.Fatalf("invertible = %v", ok)
			}
			if ok {
				// The inverse undoes the transform.
				x, y := 3.0, -7.0
				u, v := tt.m.A*x+tt.m.C*y+tt.m.E, tt.m.B*x+tt.m.D*y+tt.m.F
				bx, by := inv.A*u+inv.C*v+inv.E, inv.B*u+inv.D*v+inv.F
				if math.Abs(bx-x) > 1e-9 || math.Abs(by-y) > 1e-9 {
					t.Errorf("inverse maps back to %v, %v", bx, by)
				}
				if got := tt.m.Similarity(); got != tt.similarity {
					t.Errorf("similarity = %v", got)
				}
			}
		})
	}
}

func TestFixedBlendOf(t *testing.T) {
	tests := []struct {
		mode ui.BlendMode
		want FixedBlend
		ok   bool
	}{
		{ui.BlendModeUnspecified, FixedBlend{BlendFactorOne, BlendFactorOneMinusSrcAlpha}, true},
		{ui.BlendModeSrcOver, FixedBlend{BlendFactorOne, BlendFactorOneMinusSrcAlpha}, true},
		{ui.BlendModeDstOver, FixedBlend{BlendFactorOneMinusDstAlpha, BlendFactorOne}, true},
		{ui.BlendModePlus, FixedBlend{BlendFactorOne, BlendFactorOne}, true},
		{ui.BlendModeDstOut, FixedBlend{BlendFactorZero, BlendFactorOneMinusSrcAlpha}, true},
		{ui.BlendModeMultiply, FixedBlend{}, false},
		{ui.BlendModeClear, FixedBlend{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.mode.String(), func(t *testing.T) {
			got, ok := FixedBlendOf(tt.mode)
			if got != tt.want || ok != tt.ok {
				t.Errorf("FixedBlendOf = %v, %v; want %v, %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

// encode runs the draw function of an encoder over the surface of 100 x 100 pixels and
// returns the instances it made and the pixels each covers.
func encode(t *testing.T, m *ui.Matrix4, draw func(e *ShapeEncoder) string) (string, []ShapeInstance, []image.Rectangle) {
	t.Helper()
	var ins []ShapeInstance
	var covers []image.Rectangle
	e := &ShapeEncoder{
		Transform: m, T: AffineOf(m), Outer: image.Rect(0, 0, 100, 100), Clip: [4]float32{0, 0, 100, 100},
		ClipBounds: ui.NewRectFromLTWH(0, 0, 100, 100),
		Paint:      ShapePaint{Color: [4]float32{1, 0, 0, 1}, Antialias: true},
		Emit: func(in *ShapeInstance, c image.Rectangle) {
			ins = append(ins, *in)
			covers = append(covers, c)
		},
	}
	return draw(e), ins, covers
}

func TestShapeEncoder(t *testing.T) {
	fill := ui.NewPaint()
	stroke := ui.NewPaint()
	stroke.SetStyle(ui.PaintingStyleStroke)
	stroke.SetStrokeWidth(4)
	bevel := ui.NewPaint()
	bevel.SetStyle(ui.PaintingStyleStroke)
	bevel.SetStrokeJoin(ui.StrokeJoinBevel)
	rect := ui.NewRectFromLTWH(10, 20, 40, 30)
	id := ui.NewMatrix4Identity()
	rot := ui.NewMatrix4Identity()
	rot.RotateZ(0.3)
	tests := []struct {
		name     string
		m        *ui.Matrix4
		draw     func(e *ShapeEncoder) string
		wantMask string
		kinds    []float32
		covers   []image.Rectangle
	}{
		{"a rect is an exact box", id, func(e *ShapeEncoder) string { return e.RRect(rect, [4]ui.Radius{}, fill) }, "",
			[]float32{ShapeKindBox}, []image.Rectangle{image.Rect(9, 19, 51, 51)}},
		{"a rotated rect is a distance field", rot, func(e *ShapeEncoder) string { return e.RRect(rect, [4]ui.Radius{}, fill) }, "",
			[]float32{ShapeKindRRect}, nil},
		{"a stroke with round corners is an outline", id, func(e *ShapeEncoder) string {
			return e.RRect(rect, OvalRadii(rect), stroke)
		}, "", []float32{ShapeKindOutline}, nil},
		{"a bevel join has no analytic form", id, func(e *ShapeEncoder) string { return e.RRect(rect, [4]ui.Radius{}, bevel) }, "bevel stroke join", nil, nil},
		{"an empty fill draws nothing", id, func(e *ShapeEncoder) string {
			return e.RRect(ui.NewRectFromLTWH(5, 5, 0, 10), [4]ui.Radius{}, fill)
		}, "", nil, nil},
		{"a rect outside the surface draws nothing", id, func(e *ShapeEncoder) string {
			return e.RRect(ui.NewRectFromLTWH(200, 200, 10, 10), [4]ui.Radius{}, fill)
		}, "", nil, nil},
		{"polygon points have no analytic form", id, func(e *ShapeEncoder) string {
			return e.Segments([]ui.Offset{{}, {Dx: 5}}, stroke, false, ui.PointModePolygon)
		}, "polygon points", nil, nil},
		{"a line is one segment", id, func(e *ShapeEncoder) string {
			return e.Segments([]ui.Offset{{Dx: 10, Dy: 10}, {Dx: 40, Dy: 10}}, stroke, true, ui.PointModeLines)
		}, "", []float32{ShapeKindSegment}, nil},
		{"every pair of the points is a segment", id, func(e *ShapeEncoder) string {
			return e.Segments([]ui.Offset{{Dx: 10, Dy: 10}, {Dx: 40, Dy: 10}, {Dx: 10, Dy: 20}, {Dx: 40, Dy: 20}}, stroke, false, ui.PointModeLines)
		}, "", []float32{ShapeKindSegment, ShapeKindSegment}, nil},
		{"a fill covers the clip", rot, func(e *ShapeEncoder) string { e.Fill(); return "" }, "",
			[]float32{ShapeKindFill}, []image.Rectangle{image.Rect(0, 0, 100, 100)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mask, ins, covers := encode(t, tt.m, tt.draw)
			if mask != tt.wantMask {
				t.Errorf("needs the mask for %q, want %q", mask, tt.wantMask)
			}
			var kinds []float32
			for _, in := range ins {
				kinds = append(kinds, in[10])
			}
			if !reflect.DeepEqual(kinds, tt.kinds) {
				t.Errorf("kinds %v, want %v", kinds, tt.kinds)
			}
			if tt.covers != nil && !reflect.DeepEqual(covers, tt.covers) {
				t.Errorf("covers %v, want %v", covers, tt.covers)
			}
			for _, in := range ins {
				if int(in[11])&ShapeFlagAntialias == 0 || in[12] != 1 || in[15] != 1 || in[18] != 100 {
					t.Errorf("flags, color or clip not set: %v", in[11:20])
				}
			}
		})
	}
}

func TestStrokeHalfWidth(t *testing.T) {
	tests := []struct {
		name      string
		scale     float64
		width     float64
		hw        float64
		wantAlpha float32
	}{
		{"a wide stroke", 1, 6, 3, 1},
		{"a stroke of one pixel", 1, 1, 0.5, 1},
		{"a thin stroke is one pixel with less alpha", 1, 0.4, 0.5, 0.4},
		{"a hairline is one full pixel", 1, 0, 0.5, 1},
		{"a hairline under a scale", 2, 0, 0.25, 1},
		{"a collapsed transform draws nothing", 0, 4, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := ui.NewPaint()
			p.SetStrokeWidth(tt.width)
			m := ui.NewMatrix4Identity()
			m.Scale(float32(tt.scale), float32(tt.scale), 1)
			e := &ShapeEncoder{Transform: m, T: AffineOf(m)}
			hw, alpha := e.StrokeHalfWidth(p)
			if math.Abs(hw-tt.hw) > 1e-9 || math.Abs(float64(alpha-tt.wantAlpha)) > 1e-6 {
				t.Errorf("StrokeHalfWidth = %v, %v; want %v, %v", hw, alpha, tt.hw, tt.wantAlpha)
			}
		})
	}
}
