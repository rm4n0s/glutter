package painting

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

var (
	colorR = ui.NewColor(0xffff0000)
	colorG = ui.NewColor(0xff00ff00)
)

func paintDecoration(t *testing.T, d Decoration, offset ui.Offset, cfg ImageConfiguration) []ui.DisplayOp {
	t.Helper()
	painter := d.CreateBoxPainter(func() {})
	return recordOps(func(c *ui.Canvas) { painter.Paint(c, offset, cfg) })
}

func cfgOf(w, h float64, td ui.TextDirection) ImageConfiguration {
	return ImageConfiguration{Size: foundation.Some(ui.Size{Width: w, Height: h}), TextDirection: td}
}

// decorationsEqual is Dart's == on decorations.
func decorationsEqual(a, b Decoration) bool {
	switch x := a.(type) {
	case *BoxDecoration:
		y, ok := b.(*BoxDecoration)
		return ok && x.Equal(y)
	case *ShapeDecoration:
		y, ok := b.(*ShapeDecoration)
		return ok && x.Equal(y)
	case *FlutterLogoDecoration:
		y, ok := b.(*FlutterLogoDecoration)
		return ok && x.Equal(y)
	}
	return a == nil && b == nil
}

func TestBoxDecorationLerpIdentical(t *testing.T) {
	if BoxDecorationLerp(nil, nil, 0) != nil {
		t.Error("lerp of nils should be nil")
	}
	d := NewBoxDecoration()
	if BoxDecorationLerp(d, d, 0.5) != d {
		t.Error("lerp of identical decorations should return the same decoration")
	}
}

func TestBoxDecorationWithBorderRadiusDirectional(t *testing.T) {
	d := NewBoxDecoration(BoxDecorationOpts{
		Color:        foundation.Some(ui.NewColor(0xFF000000)),
		BorderRadius: BorderRadiusDirectional{TopStart: ui.NewRadiusCircular(100)},
	})
	size := ui.Size{Width: 1000, Height: 1000}
	rect := (ui.Offset{}).And(size)
	tests := []struct {
		td   ui.TextDirection
		want ui.RRect
	}{
		{ui.TextDirectionRTL, ui.NewRRectFromRectAndCorners(rect, ui.Corners{TopRight: ui.NewRadiusCircular(100)})},
		{ui.TextDirectionLTR, ui.NewRRectFromRectAndCorners(rect, ui.Corners{TopLeft: ui.NewRadiusCircular(100)})},
	}
	for _, tt := range tests {
		ops := paintDecoration(t, d, ui.Offset{}, cfgOf(1000, 1000, tt.td))
		if len(ops) != 1 {
			t.Fatalf("%v: ops = %v", tt.td, ops)
		}
		if got, ok := ops[0].(ui.OpDrawRRect); !ok || got.RRect != tt.want {
			t.Errorf("%v: paint = %v, want %v", tt.td, ops[0], tt.want)
		}
	}
	hits := []struct {
		td   ui.TextDirection
		pos  ui.Offset
		want bool
	}{
		{ui.TextDirectionRTL, ui.Offset{Dx: 10, Dy: 10}, true}, {ui.TextDirectionRTL, ui.Offset{Dx: 990, Dy: 10}, false},
		{ui.TextDirectionLTR, ui.Offset{Dx: 10, Dy: 10}, false}, {ui.TextDirectionLTR, ui.Offset{Dx: 990, Dy: 10}, true},
	}
	for _, tt := range hits {
		if got := d.HitTest(size, tt.pos, tt.td); got != tt.want {
			t.Errorf("HitTest(%v, %v) = %v, want %v", tt.pos, tt.td, got, tt.want)
		}
	}
}

func TestBoxDecorationWithGradientAndAlignmentDirectional(t *testing.T) {
	d := NewBoxDecoration(BoxDecorationOpts{
		Color: foundation.Some(ui.NewColor(0xFF000000)),
		Gradient: NewLinearGradient(colorsOf(0xFF000000, 0xFFFFFFFF),
			LinearGradientOpts{Begin: AlignmentDirectionalCenterStart, End: AlignmentDirectionalBottomEnd}),
	})
	ops := paintDecoration(t, d, ui.Offset{}, cfgOf(1000, 1000, ui.TextDirectionRTL))
	if len(ops) != 1 {
		t.Fatalf("ops = %v", ops)
	}
	got, ok := ops[0].(ui.OpDrawRect)
	if !ok || got.Rect != ui.NewRectFromLTWH(0, 0, 1000, 1000) {
		t.Errorf("paint = %v", ops[0])
	}
}

func TestBoxDecorationGetClipPath(t *testing.T) {
	rect := ui.NewRectFromLTWH(0, 0, 100, 20)
	rounded := NewBoxDecoration(BoxDecorationOpts{BorderRadius: NewBorderRadiusCircular(10)})
	if !looksLike(rounded.GetClipPath(rect, ui.TextDirectionLTR), []ui.Offset{{Dx: 30, Dy: 10}, {Dx: 50, Dy: 10}}, []ui.Offset{{Dx: 1, Dy: 1}, {Dx: 99, Dy: 19}}) {
		t.Error("the rounded clip path is wrong")
	}
	circle := NewBoxDecoration(BoxDecorationOpts{Shape: BoxShapeCircle})
	if !looksLike(circle.GetClipPath(rect, ui.TextDirectionLTR), []ui.Offset{{Dx: 50, Dy: 1}, {Dx: 40, Dy: 10}}, []ui.Offset{{Dx: 40, Dy: 0}, {Dx: 10, Dy: 10}}) {
		t.Error("the circle clip path is wrong")
	}
	plain := NewBoxDecoration().GetClipPath(rect, ui.TextDirectionLTR)
	if b := plain.GetBounds(); b != rect {
		t.Errorf("rectangle clip path = %v", b)
	}
}

func TestBoxDecorationEquality(t *testing.T) {
	c := foundation.Some(ui.NewColor(0))
	one := NewBoxDecoration(BoxDecorationOpts{Color: c, BackgroundBlendMode: ui.BlendModeColor})
	two := NewBoxDecoration(BoxDecorationOpts{Color: c, BackgroundBlendMode: ui.BlendModeDifference})
	if one.Equal(two) {
		t.Error("decorations with different blend modes should differ")
	}
	if !one.Equal(NewBoxDecoration(BoxDecorationOpts{Color: c, BackgroundBlendMode: ui.BlendModeColor})) {
		t.Error("equal decorations should be equal")
	}
	shadow := NewBoxShadow(BoxShadowOpts{BlurRadius: 2})
	if NewBoxDecoration(BoxDecorationOpts{BoxShadow: []BoxShadow{shadow}}).Equal(NewBoxDecoration()) {
		t.Error("shadows should matter")
	}
}

func TestBoxDecorationCopyWith(t *testing.T) {
	d := NewBoxDecoration(BoxDecorationOpts{Color: foundation.Some(colorR), Shape: BoxShapeCircle})
	got := d.CopyWith(BoxDecorationCopy{Color: foundation.Some(colorG), BorderRadius: nil})
	if !got.Equal(NewBoxDecoration(BoxDecorationOpts{Color: foundation.Some(colorG), Shape: BoxShapeCircle})) {
		t.Errorf("CopyWith = %v", got)
	}
}

func TestBoxDecorationPaintsShadowsBackgroundAndBorder(t *testing.T) {
	shadow := NewBoxShadow(BoxShadowOpts{Color: foundation.Some(ui.NewColor(0x80000000)), Offset: ui.Offset{Dx: 2, Dy: 3}, BlurRadius: 4, SpreadRadius: 1})
	d := NewBoxDecoration(BoxDecorationOpts{
		Color:     foundation.Some(colorR),
		Border:    NewBorderAll(BorderSideOpts{Color: foundation.Some(colorG), Width: foundation.Some(2.0)}),
		BoxShadow: []BoxShadow{shadow},
	})
	ops := paintDecoration(t, d, ui.Offset{Dx: 10, Dy: 20}, cfgOf(100, 50, ui.TextDirectionLTR))
	if len(ops) < 3 {
		t.Fatalf("ops = %v", ops)
	}
	shadowOp, ok := ops[0].(ui.OpDrawRect)
	if !ok || shadowOp.Rect != ui.NewRectFromLTWH(10, 20, 100, 50).Shift(ui.Offset{Dx: 2, Dy: 3}).Inflate(1) || shadowOp.Paint.MaskFilter() == nil {
		t.Errorf("shadow op = %v", ops[0])
	}
	// The background is inset by half of the opaque border, so that it doesn't show through.
	bg, ok := ops[1].(ui.OpDrawRect)
	if !ok || bg.Rect != ui.NewRectFromLTWH(10, 20, 100, 50).Deflate(1) || bg.Paint.Color() != colorR {
		t.Errorf("background op = %v", ops[1])
	}
}

func TestBoxDecorationPadding(t *testing.T) {
	if !NewBoxDecoration().Padding().Equal(EdgeInsetsZero) {
		t.Error("padding without a border should be zero")
	}
	d := NewBoxDecoration(BoxDecorationOpts{Border: NewBorderAll(BorderSideOpts{Width: foundation.Some(3.0)})})
	if !d.Padding().Equal(NewEdgeInsetsAll(3)) {
		t.Errorf("padding = %v", d.Padding())
	}
}

func TestBoxDecorationLerp(t *testing.T) {
	a := NewBoxDecoration(BoxDecorationOpts{Color: foundation.Some(ui.NewColor(0xFF000000)), BorderRadius: NewBorderRadiusCircular(10)})
	b := NewBoxDecoration(BoxDecorationOpts{Color: foundation.Some(ui.NewColor(0xFFFFFFFF)), BorderRadius: NewBorderRadiusCircular(20), Shape: BoxShapeCircle})
	_ = b
	c := NewBoxDecoration(BoxDecorationOpts{Color: foundation.Some(ui.NewColor(0xFFFFFFFF)), BorderRadius: NewBorderRadiusCircular(20)})
	mid := BoxDecorationLerp(a, c, 0.5)
	if got := mid.Color.Value(); got != ui.NewColor(0xFF000000).Lerp(ui.NewColor(0xFFFFFFFF), 0.5) {
		t.Errorf("mid color = %v", got)
	}
	if !mid.BorderRadius.Equal(NewBorderRadiusCircular(15)) {
		t.Errorf("mid radius = %v", mid.BorderRadius)
	}
	if BoxDecorationLerp(a, c, 0) != a || BoxDecorationLerp(a, c, 1) != c {
		t.Error("endpoints are wrong")
	}
	if got := BoxDecorationLerp(nil, c, 0.5); got.Color.Value().Alpha() != 128 {
		t.Errorf("from nil alpha = %d", got.Color.Value().Alpha())
	}
	if d := DecorationLerp(a, nil, 0.5).(*BoxDecoration); d.Color.Value().Alpha() != 128 {
		t.Errorf("to nil alpha = %d", d.Color.Value().Alpha())
	}
	// Different shapes switch at half.
	shapeMid := BoxDecorationLerp(a, b, 0.4)
	if shapeMid.Shape != BoxShapeRectangle || BoxDecorationLerp(a, b, 0.6).Shape != BoxShapeCircle {
		t.Error("shape should change at 0.5")
	}
}

func TestShapeDecorationConstructor(t *testing.T) {
	same := NewShapeDecoration(Border{})
	if !same.Equal(NewShapeDecoration(Border{})) {
		t.Error("equal decorations should be equal")
	}
	tests := []struct {
		name string
		got  *ShapeDecoration
		want *ShapeDecoration
	}{
		{"circle", NewShapeDecorationFromBoxDecoration(NewBoxDecoration(BoxDecorationOpts{Shape: BoxShapeCircle})), NewShapeDecoration(NewCircleBorder())},
		{
			"directional radius", NewShapeDecorationFromBoxDecoration(NewBoxDecoration(BoxDecorationOpts{BorderRadius: NewBorderRadiusDirectionalCircular(100)})),
			NewShapeDecoration(NewRoundedRectangleBorder(RoundedRectangleBorderOpts{BorderRadius: NewBorderRadiusDirectionalCircular(100)})),
		},
		{
			"circle with border", NewShapeDecorationFromBoxDecoration(NewBoxDecoration(BoxDecorationOpts{
				Shape: BoxShapeCircle, Border: NewBorderAll(BorderSideOpts{Color: foundation.Some(colorG)})})),
			NewShapeDecoration(NewCircleBorder(CircleBorderOpts{Side: side(BorderSideOpts{Color: foundation.Some(colorG)})})),
		},
		{
			"border", NewShapeDecorationFromBoxDecoration(NewBoxDecoration(BoxDecorationOpts{Border: NewBorderAll(BorderSideOpts{Color: foundation.Some(colorR)})})),
			NewShapeDecoration(NewBorderAll(BorderSideOpts{Color: foundation.Some(colorR)})),
		},
		{
			"directional border", NewShapeDecorationFromBoxDecoration(NewBoxDecoration(BoxDecorationOpts{Border: BorderDirectional{Start: side(BorderSideOpts{})}})),
			NewShapeDecoration(BorderDirectional{Start: side(BorderSideOpts{})}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.got.Equal(tt.want) {
				t.Errorf("got %v, want %v", tt.got, tt.want)
			}
		})
	}
}

func TestShapeDecorationLerp(t *testing.T) {
	if ShapeDecorationLerp(nil, nil, 0) != nil {
		t.Error("lerp of nils should be nil")
	}
	s := NewShapeDecoration(NewCircleBorder())
	if ShapeDecorationLerp(s, s, 0.5) != s {
		t.Error("lerp of identical decorations should return the same decoration")
	}
	var a Decoration = NewShapeDecoration(NewCircleBorder())
	var b Decoration = NewShapeDecoration(NewRoundedRectangleBorder())
	var c Decoration = NewShapeDecoration(NewOvalBorder())
	if !decorationsEqual(DecorationLerp(a, nil, 0), a) || !decorationsEqual(DecorationLerp(nil, b, 0), b) || DecorationLerp(nil, nil, 0) != nil {
		t.Error("lerp with nil is wrong")
	}
	if !decorationsEqual(DecorationLerp(a, b, 0), a) || !decorationsEqual(DecorationLerp(a, b, 1), b) ||
		!decorationsEqual(DecorationLerp(a, c, 0), a) || !decorationsEqual(DecorationLerp(a, c, 1), c) {
		t.Error("lerp endpoints are wrong")
	}
	size := ui.Size{Width: 200, Height: 100}
	hit := func(d Decoration, x, y float64) bool {
		return d.HitTest(size, ui.Offset{Dx: x, Dy: y}, ui.TextDirectionLTR)
	}
	tests := []struct {
		name string
		d    Decoration
		x, y float64
		want bool
	}{
		{"circle", a, 20, 50, false}, {"oval top", c, 50, 5, false}, {"oval left", c, 5, 30, false}, {"rounded", b, 20, 50, true},
		{"circle-rounded .1", DecorationLerp(a, b, 0.1), 20, 50, false},
		{"circle-rounded .5", DecorationLerp(a, b, 0.5), 20, 50, false},
		{"circle-rounded .9", DecorationLerp(a, b, 0.9), 20, 50, true},
		{"circle-oval .1", DecorationLerp(a, c, 0.1), 30, 50, false},
		{"circle-oval .5", DecorationLerp(a, c, 0.5), 30, 50, true},
		{"circle-oval .9", DecorationLerp(a, c, 0.9), 30, 50, true},
		{"rounded-oval .1", DecorationLerp(b, c, 0.1), 45, 10, true},
		{"rounded-oval .5", DecorationLerp(b, c, 0.5), 30, 10, true},
		{"rounded-oval .9", DecorationLerp(b, c, 0.9), 10, 30, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hit(tt.d, tt.x, tt.y); got != tt.want {
				t.Errorf("hit(%v, %v) = %v, want %v", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func TestShapeDecorationLerpColorAndGradient(t *testing.T) {
	gradient := NewLinearGradient([]ui.Color{colorR, colorG})
	colorDecoration := NewShapeDecoration(NewCircleBorder(), ShapeDecorationOpts{Color: foundation.Some(colorR)})
	gradientDecoration := NewShapeDecoration(NewRoundedRectangleBorder(), ShapeDecorationOpts{Gradient: gradient})
	if ShapeDecorationLerp(colorDecoration, gradientDecoration, 0) != colorDecoration || ShapeDecorationLerp(colorDecoration, gradientDecoration, 1) != gradientDecoration {
		t.Error("endpoints are wrong")
	}
	for _, tm := range []float64{0.1, 0.25, 0.49, 0.5, 0.51, 0.75, 0.9} {
		forward := ShapeDecorationLerp(colorDecoration, gradientDecoration, tm)
		reverse := ShapeDecorationLerp(gradientDecoration, colorDecoration, tm)
		for _, d := range []*ShapeDecoration{forward, reverse} {
			if d.Color.IsSome() {
				t.Errorf("t=%v: color = %v, want none", tm, d.Color)
			}
			if _, ok := d.Gradient.(*LinearGradient); !ok {
				t.Errorf("t=%v: gradient = %T", tm, d.Gradient)
			}
		}
	}
	near := ShapeDecorationLerp(colorDecoration, gradientDecoration, 0.001).Gradient.(*LinearGradient)
	if len(near.Colors) != 2 {
		t.Fatalf("colors = %v", near.Colors)
	}
	for _, c := range near.Colors {
		if !(c.R > 0.95 && c.G < 0.05 && c.B < 0.05) {
			t.Errorf("color near the start = %v", c)
		}
	}
}

func TestShapeDecorationGetClipPath(t *testing.T) {
	d := NewShapeDecoration(NewCircleBorder())
	rect := ui.NewRectFromLTWH(0, 0, 100, 20)
	if !looksLike(d.GetClipPath(rect, ui.TextDirectionLTR), []ui.Offset{{Dx: 50, Dy: 10}}, []ui.Offset{{Dx: 1, Dy: 1}, {Dx: 30, Dy: 10}, {Dx: 99, Dy: 19}}) {
		t.Error("the clip path is wrong")
	}
	if !d.Padding().Equal(EdgeInsetsZero) || d.IsComplex() {
		t.Error("padding or isComplex is wrong")
	}
	if !NewShapeDecoration(NewCircleBorder(), ShapeDecorationOpts{Shadows: []BoxShadow{{}}}).IsComplex() {
		t.Error("a decoration with shadows is complex")
	}
}

func TestShapeDecorationPaint(t *testing.T) {
	shadow := NewBoxShadow(BoxShadowOpts{Offset: ui.Offset{Dx: 1, Dy: 2}, BlurRadius: 3})
	d := NewShapeDecoration(NewRoundedRectangleBorder(RoundedRectangleBorderOpts{
		Side: widthSide(2), BorderRadius: NewBorderRadiusCircular(5)}), ShapeDecorationOpts{Color: foundation.Some(colorR), Shadows: []BoxShadow{shadow}})
	ops := paintDecoration(t, d, ui.Offset{Dx: 10, Dy: 20}, cfgOf(100, 50, ui.TextDirectionLTR))
	// shadow, interior, border
	if len(ops) != 3 {
		t.Fatalf("ops = %v", ops)
	}
	rect := ui.NewRectFromLTWH(10, 20, 100, 50)
	if got, ok := ops[0].(ui.OpDrawRRect); !ok || got.RRect != ui.NewRRectFromRectAndRadius(rect.Shift(ui.Offset{Dx: 1, Dy: 2}), ui.NewRadiusCircular(5)) {
		t.Errorf("shadow op = %v", ops[0])
	}
	if got, ok := ops[1].(ui.OpDrawRRect); !ok || got.Paint.Color() != colorR {
		t.Errorf("interior op = %v", ops[1])
	}
}

func TestDecorationDefaults(t *testing.T) {
	var d DecorationDefaults
	if !d.DebugAssertIsValid() || !d.Padding().Equal(EdgeInsetsZero) || d.IsComplex() || d.LerpFrom(nil, 0) != nil || d.LerpTo(nil, 0) != nil || !d.HitTest(ui.Size{}, ui.Offset{}, ui.TextDirectionLTR) {
		t.Error("defaults are wrong")
	}
}

func TestBoxShadow(t *testing.T) {
	s := NewBoxShadow(BoxShadowOpts{Color: foundation.Some(colorR), Offset: ui.Offset{Dx: 2, Dy: 4}, BlurRadius: 6, SpreadRadius: 8})
	if got := s.Scale(2); got.Offset != (ui.Offset{Dx: 4, Dy: 8}) || got.BlurRadius != 12 || got.SpreadRadius != 16 || got.Color != colorR {
		t.Errorf("Scale = %v", got)
	}
	p := s.ToPaint()
	if p.Color() != colorR || p.MaskFilter() == nil || p.MaskFilter().Sigma() != ui.ConvertRadiusToSigma(6) || p.MaskFilter().Style() != ui.BlurStyleNormal {
		t.Errorf("ToPaint = %v", p)
	}
	if got := s.CopyWith(BoxShadowCopy{SpreadRadius: foundation.Some(1.0)}); got.SpreadRadius != 1 || got.BlurRadius != 6 {
		t.Errorf("CopyWith = %v", got)
	}
	other := NewBoxShadow(BoxShadowOpts{Color: foundation.Some(colorG), Offset: ui.Offset{Dx: 4, Dy: 8}, BlurRadius: 10, SpreadRadius: 0})
	mid := BoxShadowLerp(&s, &other, 0.5)
	if mid.Offset != (ui.Offset{Dx: 3, Dy: 6}) || mid.BlurRadius != 8 || mid.SpreadRadius != 4 {
		t.Errorf("Lerp = %v", mid)
	}
	if BoxShadowLerp(nil, nil, 0.5) != nil || BoxShadowLerp(&s, &s, 0.5) != &s {
		t.Error("lerp of nils or identical shadows is wrong")
	}
	if got := BoxShadowLerp(nil, &s, 0.5); got.BlurRadius != 3 {
		t.Errorf("lerp from nil = %v", got)
	}
	// The lists are padded with scaled shadows.
	list := BoxShadowLerpList([]BoxShadow{s}, []BoxShadow{s, other}, 0.5)
	if len(list) != 2 || list[1].BlurRadius != 5 {
		t.Errorf("LerpList = %v", list)
	}
	if BoxShadowLerpList(nil, nil, 0.5) != nil {
		t.Error("LerpList of nils should be nil")
	}
	if got := s.String(); got != "BoxShadow("+colorR.String()+", Offset(2.0, 4.0), 6.0, 8.0, BlurStyle.normal)" {
		t.Errorf("String = %q", got)
	}
}

func TestImageConfiguration(t *testing.T) {
	c := ImageConfiguration{
		DevicePixelRatio: foundation.Some(2.0), TextDirection: ui.TextDirectionLTR, Size: foundation.Some(ui.Size{Width: 10, Height: 20}),
		Platform: foundation.Some(foundation.TargetPlatformLinux),
	}
	if got := c.String(); got != "ImageConfiguration(devicePixelRatio: 2.0, textDirection: TextDirection.ltr, size: Size(10.0, 20.0), platform: linux)" {
		t.Errorf("String = %q", got)
	}
	if got := ImageConfigurationEmpty.String(); got != "ImageConfiguration()" {
		t.Errorf("empty String = %q", got)
	}
	copied := c.CopyWith(ImageConfigurationCopy{DevicePixelRatio: foundation.Some(3.0), TextDirection: foundation.Some(ui.TextDirectionRTL)})
	if copied.DevicePixelRatio.Value() != 3 || copied.TextDirection != ui.TextDirectionRTL || copied.Size != c.Size {
		t.Errorf("CopyWith = %v", copied)
	}
}
