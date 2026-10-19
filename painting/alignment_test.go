package painting

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

var textDirections = []ui.TextDirection{ui.TextDirectionRTL, ui.TextDirectionLTR}

func approxAlignment(t *testing.T, got, want Alignment) {
	t.Helper()
	if math.Abs(got.X-want.X) > 1e-10 || math.Abs(got.Y-want.Y) > 1e-10 {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestAlignmentControl(t *testing.T) {
	a := Alignment{0.5, 0.25}
	tests := []struct {
		name      string
		got, want Alignment
	}{
		{"div", a.Div(2), Alignment{0.25, 0.125}},
		{"truncDiv", a.TruncDiv(2), AlignmentCenter},
		{"mod", a.Mod(5), Alignment{0.5, 0.25}},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}

func TestAlignmentLerp(t *testing.T) {
	a, b := AlignmentTopLeft, AlignmentTopCenter
	if got := AlignmentLerp(&a, &b, 0.25); *got != (Alignment{-0.75, -1.0}) {
		t.Errorf("lerp(a, b) = %v", *got)
	}
	if AlignmentLerp(nil, nil, 0.25) != nil {
		t.Error("lerp(nil, nil) should be nil")
	}
	if got := AlignmentLerp(nil, &b, 0.25); *got != (Alignment{0.0, -0.25}) {
		t.Errorf("lerp(nil, b) = %v", *got)
	}
	if got := AlignmentLerp(&a, nil, 0.25); *got != (Alignment{-0.75, -0.75}) {
		t.Errorf("lerp(a, nil) = %v", *got)
	}
	if AlignmentLerp(&a, &a, 0.5) != &a {
		t.Error("lerp of identical values should return the same pointer")
	}
	d := AlignmentDirectionalTopStart
	if AlignmentDirectionalLerp(&d, &d, 0.5) != &d {
		t.Error("directional lerp of identical values should return the same pointer")
	}
	var g AlignmentGeometry = AlignmentTopLeft
	if AlignmentGeometryLerp(g, g, 0.5) != g {
		t.Error("geometry lerp of identical values should return the same value")
	}
	if AlignmentGeometryLerp(nil, nil, 0) != nil {
		t.Error("geometry lerp of nils should be nil")
	}
}

func TestAlignmentGeometryInvariants(t *testing.T) {
	topStart, topEnd := AlignmentDirectionalTopStart, AlignmentDirectionalTopEnd
	center, topLeft, topRight := AlignmentCenter, AlignmentTopLeft, AlignmentTopRight
	numbers := []float64{0.0, 1.0, -1.0, 2.0, 0.25, 0.5, 100.0, -999.75}

	eq := func(name string, a, b AlignmentGeometry) {
		t.Helper()
		if !a.Equal(b) {
			t.Errorf("%s: %v != %v", name, a, b)
		}
	}
	eq("topEnd*0 + topRight*0", topEnd.Mul(0).Add(topRight.Mul(0)), center)
	eq("(topEnd+topRight)*0", AlignmentGeometryMul(topEnd.Add(topRight), 0), topEnd.Mul(0).Add(topRight.Mul(0)))
	eq("commutes", topStart.Add(topLeft), topLeft.Add(topStart))
	for _, d := range textDirections {
		if got, want := topStart.Add(topLeft).Resolve(d), topStart.Resolve(d).Plus(topLeft); got != want {
			t.Errorf("resolve(%v): %v != %v", d, got, want)
		}
	}
	if topStart.Resolve(ui.TextDirectionLTR) != topLeft || topStart.Resolve(ui.TextDirectionRTL) != topRight {
		t.Error("topStart should resolve to topLeft in LTR and topRight in RTL")
	}
	eq("topEnd*0", topEnd.Mul(0), center)
	eq("topLeft*0", topLeft.Mul(0), center)
	eq("topStart*1", topStart.Mul(1), topStart)
	for _, n := range numbers {
		eq("n", topStart.Mul(n).Add(topStart), topStart.Mul(n+1))
		eq("end n", topEnd.Mul(n).Add(topEnd), topEnd.Mul(n+1))
		for _, m := range numbers {
			eq("n+m", topStart.Mul(n).Add(topStart.Mul(m)), topStart.Mul(n+m))
		}
	}
	eq("topStart*3", topStart.Plus(topStart).Plus(topStart), topStart.Mul(3))
	for _, d := range textDirections {
		if got, want := topEnd.Mul(0).Add(topRight.Mul(0)).Resolve(d), center.Add(center).Resolve(d); got != want {
			t.Errorf("zero add resolve(%v): %v != %v", d, got, want)
		}
		if got, want := topEnd.Mul(0).Add(topLeft).Resolve(d), center.Add(topLeft).Resolve(d); got != want {
			t.Errorf("zero add topLeft resolve(%v): %v != %v", d, got, want)
		}
	}
	for name, ne := range map[string][2]AlignmentGeometry{
		"start/left": {topStart, topLeft}, "end/left": {topEnd, topLeft}, "start/right": {topStart, topRight}, "end/right": {topEnd, topRight},
		"mixed/left": {topStart.Add(topLeft), topLeft}, "mixed/start": {topStart.Add(topLeft), topStart},
	} {
		if ne[0].Equal(ne[1]) {
			t.Errorf("%s: should not be equal", name)
		}
	}
}

func TestAlignmentResolve(t *testing.T) {
	tests := []struct {
		name     string
		in       AlignmentDirectional
		ltr, rtl Alignment
	}{
		{"positive", AlignmentDirectional{0.25, 0.3}, Alignment{0.25, 0.3}, Alignment{-0.25, 0.3}},
		{"negative", AlignmentDirectional{-0.25, 0.3}, Alignment{-0.25, 0.3}, Alignment{0.25, 0.3}},
		{"overflow", AlignmentDirectional{1.25, 0.3}, Alignment{1.25, 0.3}, Alignment{-1.25, 0.3}},
		{"negative y", AlignmentDirectional{0.5, -0.3}, Alignment{0.5, -0.3}, Alignment{-0.5, -0.3}},
		{"center", AlignmentDirectionalCenter, AlignmentCenter, AlignmentCenter},
		{"bottomEnd", AlignmentDirectionalBottomEnd, AlignmentBottomRight, AlignmentBottomLeft},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.Resolve(ui.TextDirectionLTR); got != tt.ltr {
				t.Errorf("ltr = %v, want %v", got, tt.ltr)
			}
			if got := tt.in.Resolve(ui.TextDirectionRTL); got != tt.rtl {
				t.Errorf("rtl = %v, want %v", got, tt.rtl)
			}
		})
	}
	if AlignmentDirectionalCenterStart.Resolve(ui.TextDirectionLTR) != AlignmentDirectionalCenterEnd.Resolve(ui.TextDirectionRTL) {
		t.Error("centerStart in LTR should equal centerEnd in RTL")
	}
}

func TestAlignmentGeometryLerpAdHoc(t *testing.T) {
	mixed1 := Alignment{10, 20}.Add(AlignmentDirectional{30, 50})
	mixed2 := Alignment{70, 110}.Add(AlignmentDirectional{130, 170})
	mixed3 := Alignment{25, 42.5}.Add(AlignmentDirectional{55, 80})
	for _, d := range textDirections {
		if got := AlignmentGeometryLerp(mixed1, mixed2, 0.0).Resolve(d); got != mixed1.Resolve(d) {
			t.Errorf("t=0 %v: %v", d, got)
		}
		if got := AlignmentGeometryLerp(mixed1, mixed2, 1.0).Resolve(d); got != mixed2.Resolve(d) {
			t.Errorf("t=1 %v: %v", d, got)
		}
		if got := AlignmentGeometryLerp(mixed1, mixed2, 0.25).Resolve(d); got != mixed3.Resolve(d) {
			t.Errorf("t=.25 %v: %v want %v", d, got, mixed3.Resolve(d))
		}
	}
}

func TestAlignmentLerpCommutesWithResolve(t *testing.T) {
	offsets := []AlignmentGeometry{
		AlignmentTopLeft, AlignmentTopCenter, AlignmentTopRight, AlignmentDirectionalTopStart, AlignmentDirectionalTopCenter,
		AlignmentDirectionalTopEnd, AlignmentCenterLeft, AlignmentCenter, AlignmentCenterRight, AlignmentDirectionalCenterStart,
		AlignmentDirectionalCenter, AlignmentDirectionalCenterEnd, AlignmentBottomLeft, AlignmentBottomCenter, AlignmentBottomRight,
		AlignmentDirectionalBottomStart, AlignmentDirectionalBottomCenter, AlignmentDirectionalBottomEnd,
		Alignment{-1.0, 0.65}, AlignmentDirectional{-1.0, 0.45}, AlignmentDirectional{0.125, 0.625}, Alignment{0.25, 0.875},
		Alignment{0.0625, 0.5625}.Add(AlignmentDirectional{0.1875, 0.6875}), AlignmentDirectional{2.0, 3.0}, Alignment{2.0, 3.0},
		Alignment{2.0, 3.0}.Add(AlignmentDirectional{5.0, 3.0}), Alignment{10.0, 20.0}.Add(AlignmentDirectional{30.0, 50.0}),
		Alignment{70.0, 110.0}.Add(AlignmentDirectional{130.0, 170.0}), Alignment{25.0, 42.5}.Add(AlignmentDirectional{55.0, 80.0}),
		nil,
	}
	times := []float64{0.25, 0.5, 0.75}
	for _, d := range textDirections {
		defaultValue := AlignmentDirectionalCenter.Resolve(d)
		resolve := func(g AlignmentGeometry) Alignment {
			if g == nil {
				return defaultValue
			}
			return g.Resolve(d)
		}
		for _, a := range offsets {
			ra := resolve(a)
			for _, b := range offsets {
				rb := resolve(b)
				approxAlignment(t, *AlignmentLerp(&ra, &rb, 0), ra)
				approxAlignment(t, *AlignmentLerp(&ra, &rb, 1), rb)
				approxAlignment(t, resolve(AlignmentGeometryLerp(a, b, 0)), ra)
				approxAlignment(t, resolve(AlignmentGeometryLerp(a, b, 1)), rb)
				for _, tm := range times {
					v := resolve(AlignmentGeometryLerp(a, b, tm))
					approxAlignment(t, v, *AlignmentLerp(&ra, &rb, tm))
					if v.X < math.Min(ra.X, rb.X)-1e-9 || v.X > math.Max(ra.X, rb.X)+1e-9 ||
						v.Y < math.Min(ra.Y, rb.Y)-1e-9 || v.Y > math.Max(ra.Y, rb.Y)+1e-9 {
						t.Fatalf("lerp(%v, %v, %v) = %v is outside the range", a, b, tm, v)
					}
				}
			}
		}
	}
}

func TestAlignmentOperators(t *testing.T) {
	directional := AlignmentDirectional{1, 2}
	normal := Alignment{3, 5}
	if got := directional.Add(normal).Resolve(ui.TextDirectionLTR); got != (Alignment{4, 7}) {
		t.Errorf("add ltr = %v", got)
	}
	if got := directional.Add(normal).Resolve(ui.TextDirectionRTL); got != (Alignment{2, 7}) {
		t.Errorf("add rtl = %v", got)
	}
	if normal.Add(normal) != AlignmentGeometry(normal.Mul(2)) {
		t.Errorf("normal+normal = %v", normal.Add(normal))
	}
	tests := []struct {
		name      string
		got, want AlignmentGeometry
	}{
		{"directional mul", directional.Mul(2), AlignmentDirectional{2, 4}},
		{"directional div", directional.Div(2), AlignmentDirectional{0.5, 1}},
		{"directional mod", directional.Mod(2), AlignmentDirectionalCenterEnd},
		{"directional truncDiv", directional.TruncDiv(2), AlignmentDirectionalBottomCenter},
		{"mul", Alignment{1, 2}.Mul(2), Alignment{2, 4}},
		{"div", Alignment{1, 2}.Div(2), Alignment{0.5, 1}},
		{"mod", Alignment{1, 2}.Mod(2), AlignmentCenterRight},
		{"truncDiv", Alignment{1, 2}.TruncDiv(2), AlignmentBottomCenter},
		{"plus", Alignment{1, 2}.Plus(Alignment{3, 5}), Alignment{4, 7}},
		{"minus", Alignment{1, 2}.Minus(Alignment{3, 5}), Alignment{-2, -3}},
		{"directional plus", AlignmentDirectional{1, 2}.Plus(AlignmentDirectional{3, 5}), AlignmentDirectional{4, 7}},
		{"directional minus", AlignmentDirectional{1, 2}.Minus(AlignmentDirectional{3, 5}), AlignmentDirectional{-2, -3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.got.Equal(tt.want) {
				t.Errorf("got %v, want %v", tt.got, tt.want)
			}
		})
	}
	for _, d := range textDirections {
		if got, want := AlignmentCenter.Add(directional.Mul(2)).Resolve(d), (AlignmentDirectional{2, 4}).Resolve(d); got != want {
			t.Errorf("center + directional*2 (%v): %v != %v", d, got, want)
		}
	}
}

func TestAlignmentString(t *testing.T) {
	tests := []struct {
		got  AlignmentGeometry
		want string
	}{
		{Alignment{1.0001, 2.0001}, "Alignment(1.0, 2.0)"},
		{AlignmentCenter, "Alignment.center"},
		{AlignmentBottomLeft.Add(AlignmentDirectionalCenterEnd), "Alignment.bottomLeft + AlignmentDirectional.centerEnd"},
		{Alignment{0.0001, 0.0001}, "Alignment(0.0, 0.0)"},
		{AlignmentDirectionalCenter, "AlignmentDirectional.center"},
	}
	for _, tt := range tests {
		if got := tt.got.String(); got != tt.want {
			t.Errorf("String = %q, want %q", got, tt.want)
		}
	}
}

func TestAlignmentGeometryHelpers(t *testing.T) {
	size := ui.Size{Width: 100, Height: 200}
	rect := ui.NewRectFromLTWH(10, 20, 100, 200)
	tests := []struct {
		name string
		got  any
		want any
	}{
		{"alongSize", Alignment{0.5, -0.5}.AlongSize(size), ui.Offset{Dx: 75, Dy: 50}},
		{"alongOffset", Alignment{0.5, -0.5}.AlongOffset(ui.Offset{Dx: 20, Dy: 40}), ui.Offset{Dx: 15, Dy: 10}},
		{"withinRect", Alignment{0.5, -0.5}.WithinRect(rect), ui.Offset{Dx: 85, Dy: 70}},
		{"inscribe", AlignmentTopLeft.Inscribe(ui.Size{Width: 10, Height: 20}, rect), ui.NewRectFromLTWH(10, 20, 10, 20)},
		{"inscribe center", AlignmentCenter.Inscribe(ui.Size{Width: 10, Height: 20}, rect), ui.NewRectFromLTWH(55, 110, 10, 20)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %v, want %v", tt.got, tt.want)
			}
		})
	}
}

func TestTextAlignVertical(t *testing.T) {
	for _, y := range []float64{-1, 0, 1} {
		if got := NewTextAlignVertical(y); got.Y != y {
			t.Errorf("NewTextAlignVertical(%v) = %v", y, got)
		}
	}
}
