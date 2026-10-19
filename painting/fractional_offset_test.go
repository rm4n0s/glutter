package painting

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

func TestFractionalOffset(t *testing.T) {
	tests := []struct {
		name   string
		got    FractionalOffset
		wantDX float64
		wantDY float64
		wantX  float64
		wantY  float64
		str    string
	}{
		{"topLeft", FractionalOffsetTopLeft, 0, 0, -1, -1, "FractionalOffset(0.0, 0.0)"},
		{"center", FractionalOffsetCenter, 0.5, 0.5, 0, 0, "FractionalOffset(0.5, 0.5)"},
		{"bottomRight", FractionalOffsetBottomRight, 1, 1, 1, 1, "FractionalOffset(1.0, 1.0)"},
		{"from offset and size", NewFractionalOffsetFromOffsetAndSize(ui.Offset{Dx: 25, Dy: 10}, ui.Size{Width: 100, Height: 40}), 0.25, 0.25, -0.5, -0.5, "FractionalOffset(0.3, 0.3)"},
		{"from offset and rect", NewFractionalOffsetFromOffsetAndRect(ui.Offset{Dx: 35, Dy: 20}, ui.NewRectFromLTWH(10, 10, 100, 40)), 0.25, 0.25, -0.5, -0.5, "FractionalOffset(0.3, 0.3)"},
		{"plus", FractionalOffsetTopLeft.Plus(NewFractionalOffset(0.25, 0.5)), 0.25, 0.5, -0.5, 0, "FractionalOffset(0.3, 0.5)"},
		{"minus", NewFractionalOffset(0.75, 0.5).Minus(NewFractionalOffset(0.25, 0.5)), 0.5, 0, 0, -1, "FractionalOffset(0.5, 0.0)"},
		{"neg", NewFractionalOffset(0.25, 0.5).Neg(), -0.25, -0.5, -1.5, -2, "FractionalOffset(-0.3, -0.5)"},
		{"mul", NewFractionalOffset(0.25, 0.5).Mul(2), 0.5, 1, 0, 1, "FractionalOffset(0.5, 1.0)"},
		{"div", NewFractionalOffset(0.5, 1).Div(2), 0.25, 0.5, -0.5, 0, "FractionalOffset(0.3, 0.5)"},
		{"truncDiv", NewFractionalOffset(0.75, 1.5).TruncDiv(0.5), 1, 3, 1, 5, "FractionalOffset(1.0, 3.0)"},
		{"mod", NewFractionalOffset(0.75, 1.5).Mod(0.5), 0.25, 0, -0.5, -1, "FractionalOffset(0.3, 0.0)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !near(tt.got.DX(), tt.wantDX, 1e-12) || !near(tt.got.DY(), tt.wantDY, 1e-12) {
				t.Errorf("dx, dy = %v, %v, want %v, %v", tt.got.DX(), tt.got.DY(), tt.wantDX, tt.wantDY)
			}
			if !near(tt.got.X, tt.wantX, 1e-12) || !near(tt.got.Y, tt.wantY, 1e-12) {
				t.Errorf("x, y = %v, %v, want %v, %v", tt.got.X, tt.got.Y, tt.wantX, tt.wantY)
			}
			if got := tt.got.String(); got != tt.str {
				t.Errorf("String = %q, want %q", got, tt.str)
			}
		})
	}
	// A fractional offset is an alignment.
	var g AlignmentGeometry = FractionalOffsetCenter
	if !g.Equal(AlignmentCenter) {
		t.Error("FractionalOffset.center equals Alignment.center")
	}
	if got, ok := FractionalOffsetTopLeft.Add(FractionalOffsetBottomRight).(FractionalOffset); !ok || got.DX() != 1 {
		t.Errorf("Add of fractional offsets = %v", FractionalOffsetTopLeft.Add(FractionalOffsetBottomRight))
	}
	if _, ok := FractionalOffsetTopLeft.Add(AlignmentBottomRight).(Alignment); !ok {
		t.Error("adding a plain alignment gives an alignment")
	}
}

func TestFractionalOffsetLerp(t *testing.T) {
	a, b := NewFractionalOffset(0, 0), NewFractionalOffset(1, 0.5)
	tests := []struct {
		name           string
		a, b           *FractionalOffset
		t              float64
		wantDX, wantDY float64
	}{
		{"both", &a, &b, 0.5, 0.5, 0.25},
		{"from null is from the center", nil, &b, 0.5, 0.75, 0.5},
		{"to null is to the center", &a, nil, 0.5, 0.25, 0.25},
	}
	for _, tt := range tests {
		got := FractionalOffsetLerp(tt.a, tt.b, tt.t)
		if !near(got.DX(), tt.wantDX, 1e-12) || !near(got.DY(), tt.wantDY, 1e-12) {
			t.Errorf("%s: got %v, want (%v, %v)", tt.name, got, tt.wantDX, tt.wantDY)
		}
	}
	if FractionalOffsetLerp(nil, nil, 0.5) != nil || FractionalOffsetLerp(&a, &a, 0.5) != &a {
		t.Error("lerp of equal pointers returns the first")
	}
}

func TestPositionDependentBox(t *testing.T) {
	size, child := ui.Size{Width: 200, Height: 100}, ui.Size{Width: 40, Height: 20}
	tests := []struct {
		name string
		o    PositionDependentBoxOpts
		want ui.Offset
	}{
		{"below the target", PositionDependentBoxOpts{Target: ui.Offset{Dx: 100, Dy: 30}, PreferBelow: true, VerticalOffset: 5}, ui.Offset{Dx: 80, Dy: 35}},
		{"above the target", PositionDependentBoxOpts{Target: ui.Offset{Dx: 100, Dy: 70}, PreferBelow: false, VerticalOffset: 5}, ui.Offset{Dx: 80, Dy: 45}},
		{"does not fit below, goes above", PositionDependentBoxOpts{Target: ui.Offset{Dx: 100, Dy: 80}, PreferBelow: true, VerticalOffset: 5}, ui.Offset{Dx: 80, Dy: 55}},
		{"does not fit above, goes below", PositionDependentBoxOpts{Target: ui.Offset{Dx: 100, Dy: 10}, PreferBelow: false, VerticalOffset: 5}, ui.Offset{Dx: 80, Dy: 15}},
		{"left margin", PositionDependentBoxOpts{Target: ui.Offset{Dx: 0, Dy: 30}, PreferBelow: true}, ui.Offset{Dx: 10, Dy: 30}},
		{"right margin", PositionDependentBoxOpts{Target: ui.Offset{Dx: 200, Dy: 30}, PreferBelow: true}, ui.Offset{Dx: 150, Dy: 30}},
		{"custom margin", PositionDependentBoxOpts{Target: ui.Offset{Dx: 0, Dy: 30}, PreferBelow: true, Margin: foundation.Some(2.0)}, ui.Offset{Dx: 2, Dy: 30}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.o.Size, tt.o.ChildSize = size, child
			if got := PositionDependentBox(tt.o); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
	// A child nearly as wide as the container is centered.
	if got := PositionDependentBox(PositionDependentBoxOpts{Size: size, ChildSize: ui.Size{Width: 190, Height: 20}, Target: ui.Offset{Dx: 20, Dy: 30}, PreferBelow: true}); got.Dx != 5 {
		t.Errorf("centered x = %v", got.Dx)
	}
}

func TestCircularNotchedRectangle(t *testing.T) {
	host := lrtb(0, 100, 200, 160)
	guest := ui.NewRectFromLTWH(80, 80, 40, 40)
	tests := []struct {
		name      string
		shape     CircularNotchedRectangle
		guest     foundation.Opt[ui.Rect]
		inNotch   bool // whether the point just below the top edge at the guest center is outside the path
		wantBound ui.Rect
	}{
		{"no guest", CircularNotchedRectangle{}, foundation.None[ui.Rect](), false, host},
		{"guest away", CircularNotchedRectangle{}, foundation.Some(ui.NewRectFromLTWH(0, 0, 10, 10)), false, host},
		{"notch", CircularNotchedRectangle{}, foundation.Some(guest), true, host},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.shape.GetOuterPath(host, tt.guest)
			if !rectNear(path.GetBounds(), tt.wantBound) {
				t.Errorf("bounds = %v, want %v", path.GetBounds(), tt.wantBound)
			}
			// A point of the host under the guest center, near the top edge.
			inside := path.Contains(ui.Offset{Dx: 100, Dy: 105})
			if inside == tt.inNotch {
				t.Errorf("contains(100, 105) = %v, notch = %v", inside, tt.inNotch)
			}
			// A point away from the guest is always inside.
			if !path.Contains(ui.Offset{Dx: 10, Dy: 110}) {
				t.Error("the host away from the guest is solid")
			}
		})
	}
	// The inverted notch cuts the bottom edge.
	inv := CircularNotchedRectangle{Inverted: true}
	invGuest := ui.NewRectFromLTWH(80, 140, 40, 40)
	path := inv.GetOuterPath(host, foundation.Some(invGuest))
	if path.Contains(ui.Offset{Dx: 100, Dy: 155}) || !path.Contains(ui.Offset{Dx: 100, Dy: 110}) {
		t.Error("the inverted notch is at the bottom edge")
	}
}

func TestPaintZigZag(t *testing.T) {
	paint := ui.NewPaint()
	ops := recordOps(func(c *ui.Canvas) { PaintZigZag(c, paint, ui.Offset{Dx: 10, Dy: 10}, ui.Offset{Dx: 50, Dy: 10}, 4, 3) })
	var path *ui.Path
	for _, op := range ops {
		if d, ok := op.(ui.OpDrawPath); ok {
			path = d.Path
		}
	}
	if path == nil {
		t.Fatalf("no path in %v", ops)
	}
	// 4 zigs of 5 pixels each on a line 40 long, zigging 3 either side.
	b := path.GetBounds()
	if !near(b.Left, 0, 1e-9) || !near(b.Right, 40, 1e-9) || !near(b.Top, -3, 1e-9) || !near(b.Bottom, 3, 1e-9) {
		t.Errorf("bounds = %v", b)
	}
	if _, ok := ops[0].(ui.OpSave); !ok {
		t.Errorf("first op %T", ops[0])
	}
	if _, ok := ops[len(ops)-1].(ui.OpRestore); !ok {
		t.Errorf("last op %T", ops[len(ops)-1])
	}
}
