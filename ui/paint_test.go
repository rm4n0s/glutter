package ui

import "testing"

func TestPaintEnumStrings(t *testing.T) {
	tests := []struct {
		got, want string
	}{
		{BlendModeSrcOver.String(), "BlendMode.srcOver"},
		{BlendModeSrcATop.String(), "BlendMode.srcATop"},
		{BlendModeLuminosity.String(), "BlendMode.luminosity"},
		{BlendModeUnspecified.String(), "BlendMode.unspecified"},
		{BlendMode(99).String(), "BlendMode(99)"},
		{FilterQualityMedium.String(), "FilterQuality.medium"},
		{StrokeCapRound.String(), "StrokeCap.round"},
		{StrokeJoinBevel.String(), "StrokeJoin.bevel"},
		{PaintingStyleStroke.String(), "PaintingStyle.stroke"},
		{ClipAntiAliasWithSaveLayer.String(), "Clip.antiAliasWithSaveLayer"},
		{TileModeRepeated.String(), "TileMode.repeated"},
		{BlurStyleInner.String(), "BlurStyle.inner"},
		{ColorSpaceSRGB.String(), "ColorSpace.sRGB"},
		{ColorSpaceExtendedSRGB.String(), "ColorSpace.extendedSRGB"},
		{ColorSpaceDisplayP3.String(), "ColorSpace.displayP3"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("got %q, want %q", tt.got, tt.want)
		}
	}
}

// TestEnumOrder checks the values against Flutter's declaration order, which the
// renderers rely on.
func TestEnumOrder(t *testing.T) {
	blend := []BlendMode{
		BlendModeClear, BlendModeSrc, BlendModeDst, BlendModeSrcOver, BlendModeDstOver, BlendModeSrcIn,
		BlendModeDstIn, BlendModeSrcOut, BlendModeDstOut, BlendModeSrcATop, BlendModeDstATop, BlendModeXor,
		BlendModePlus, BlendModeModulate, BlendModeScreen, BlendModeOverlay, BlendModeDarken, BlendModeLighten,
		BlendModeColorDodge, BlendModeColorBurn, BlendModeHardLight, BlendModeSoftLight, BlendModeDifference,
		BlendModeExclusion, BlendModeMultiply, BlendModeHue, BlendModeSaturation, BlendModeColor, BlendModeLuminosity,
	}
	if len(blend) != 29 {
		t.Fatalf("%d blend modes, want 29", len(blend))
	}
	for i, m := range blend {
		// Unspecified is 0, so Flutter's index i is i+1.
		if int(m) != i+1 {
			t.Errorf("%v = %d, want %d", m, m, i+1)
		}
	}
	if ColorSpaceSRGB != 0 {
		t.Error("the zero ColorSpace must be sRGB")
	}
	if TileModeClamp != 1 || TileModeDecal != 4 || BlurStyleNormal != 1 || BlurStyleInner != 4 {
		t.Error("TileMode and BlurStyle order wrong")
	}
}

func TestPaintDefaults(t *testing.T) {
	var p Paint
	if p.Color() != (Color{A: 1}) || !p.IsAntiAlias() || p.BlendMode() != BlendModeSrcOver ||
		p.Style() != PaintingStyleFill || p.StrokeWidth() != 0 || p.StrokeCap() != StrokeCapButt ||
		p.StrokeJoin() != StrokeJoinMiter || p.StrokeMiterLimit() != 4 || p.FilterQuality() != FilterQualityNone ||
		p.MaskFilter() != nil || p.Shader() != nil || p.ColorFilter() != nil || p.ImageFilter() != nil || p.InvertColors() {
		t.Errorf("the zero Paint is not Flutter's default: %v", p)
	}
	if *NewPaint() != p {
		t.Error("NewPaint must be the zero value")
	}
	if p.String() != "Paint()" {
		t.Errorf("String = %q, want Paint()", p.String())
	}
}

func TestPaintSetters(t *testing.T) {
	blur := NewMaskFilterBlur(BlurStyleOuter, 2)
	cf := NewColorFilterMode(NewColor(0xFF00FF00), BlendModeMultiply)
	imf := NewImageFilterBlur(ImageFilterBlurOpts{SigmaX: 3, SigmaY: 3})
	grad := NewGradientLinear(Offset{}, Offset{10, 0}, []Color{{A: 1}, {A: 1, R: 1}})
	var p Paint
	p.SetColor(NewColor(0xFF6750A4))
	p.SetIsAntiAlias(false)
	p.SetBlendMode(BlendModeMultiply)
	p.SetStyle(PaintingStyleStroke)
	p.SetStrokeWidth(2.5)
	p.SetStrokeCap(StrokeCapRound)
	p.SetStrokeJoin(StrokeJoinBevel)
	p.SetStrokeMiterLimit(7)
	p.SetFilterQuality(FilterQualityHigh)
	p.SetMaskFilter(blur)
	p.SetShader(grad)
	p.SetColorFilter(cf)
	p.SetImageFilter(imf)
	p.SetInvertColors(true)

	if p.Color() != NewColor(0xFF6750A4) || p.IsAntiAlias() || p.BlendMode() != BlendModeMultiply ||
		p.Style() != PaintingStyleStroke || p.StrokeWidth() != 2.5 || p.StrokeCap() != StrokeCapRound ||
		p.StrokeJoin() != StrokeJoinBevel || p.StrokeMiterLimit() != 7 || p.FilterQuality() != FilterQualityHigh ||
		p.MaskFilter() != blur || p.Shader() != grad || p.ColorFilter() != cf || p.ImageFilter() != imf || !p.InvertColors() {
		t.Errorf("getters do not return what the setters were given: %v", p)
	}

	// Copies are independent.
	c := NewPaintFrom(&p)
	c.SetColor(NewColor(0xFF000000))
	c.SetStrokeWidth(1)
	c.SetMaskFilter(nil)
	if p.Color() != NewColor(0xFF6750A4) || p.StrokeWidth() != 2.5 || p.MaskFilter() != blur {
		t.Error("changing a copy changed the original")
	}
	if c.Color() != (Color{A: 1}) || c.MaskFilter() != nil || c.Shader() != grad {
		t.Error("the copy lost what it should have kept")
	}
}

func TestPaintSettingTheDefaultGivesTheZeroValue(t *testing.T) {
	var p Paint
	p.SetColor(NewColor(0xFF000000))
	p.SetBlendMode(BlendModeSrcOver)
	p.SetStyle(PaintingStyleFill)
	p.SetStrokeCap(StrokeCapButt)
	p.SetStrokeJoin(StrokeJoinMiter)
	p.SetStrokeMiterLimit(4)
	p.SetFilterQuality(FilterQualityNone)
	p.SetIsAntiAlias(true)
	if p != (Paint{}) {
		t.Errorf("setting every default must give the zero Paint, got %+v", p)
	}
	// And the other way: the unspecified values read as the defaults.
	p.SetBlendMode(BlendModeUnspecified)
	p.SetStyle(PaintingStyleUnspecified)
	if p.BlendMode() != BlendModeSrcOver || p.Style() != PaintingStyleFill {
		t.Error("Unspecified must read as the default")
	}
}

func TestPaintStrokeMiterLimitKeepsPrecision(t *testing.T) {
	var p Paint
	for _, v := range []float64{0, 0.5, 1, 4, 4.5, 10, 1000} {
		p.SetStrokeMiterLimit(v)
		if got := p.StrokeMiterLimit(); got != v {
			t.Errorf("StrokeMiterLimit after Set(%v) = %v", v, got)
		}
	}
}

func TestPaintString(t *testing.T) {
	stroke := func(f func(*Paint)) *Paint { p := &Paint{}; p.SetStyle(PaintingStyleStroke); f(p); return p }
	tests := []struct {
		name string
		p    *Paint
		want string
	}{
		{"hairline stroke", stroke(func(*Paint) {}), "Paint(PaintingStyle.stroke hairline)"},
		{"stroke width", stroke(func(p *Paint) { p.SetStrokeWidth(3) }), "Paint(PaintingStyle.stroke 3.0)"},
		{"stroke cap and join", stroke(func(p *Paint) {
			p.SetStrokeWidth(3)
			p.SetStrokeCap(StrokeCapRound)
			p.SetStrokeJoin(StrokeJoinRound)
		}), "Paint(PaintingStyle.stroke 3.0 StrokeCap.round StrokeJoin.round)"},
		{"miter limit", stroke(func(p *Paint) { p.SetStrokeWidth(1); p.SetStrokeMiterLimit(8) }), "Paint(PaintingStyle.stroke 1.0 StrokeJoin.miter up to 8.0)"},
		{"no antialias", func() *Paint { p := &Paint{}; p.SetIsAntiAlias(false); return p }(), "Paint(antialias off)"},
		{"color and blend", func() *Paint {
			p := &Paint{}
			p.SetColor(NewColor(0xFF00FF00))
			p.SetBlendMode(BlendModeXor)
			return p
		}(), "Paint(Color(alpha: 1.0000, red: 0.0000, green: 1.0000, blue: 0.0000, colorSpace: ColorSpace.sRGB); BlendMode.xor)"},
		{"filters", func() *Paint {
			p := &Paint{}
			p.SetColorFilter(NewColorFilterLinearToSrgbGamma())
			p.SetMaskFilter(NewMaskFilterBlur(BlurStyleNormal, 2))
			p.SetFilterQuality(FilterQualityLow)
			p.SetImageFilter(NewImageFilterDilate(ImageFilterMorphologyOpts{RadiusX: 1, RadiusY: 2}))
			p.SetInvertColors(true)
			return p
		}(), "Paint(colorFilter: ColorFilter.linearToSrgbGamma(); maskFilter: MaskFilter.blur(BlurStyle.normal, 2.0); filterQuality: FilterQuality.low; imageFilter: ImageFilter.dilate(1.0, 2.0); invert: true)"},
	}
	for _, tt := range tests {
		got := tt.p.String()
		if got != tt.want {
			t.Errorf("%s:\n got  %s\n want %s", tt.name, got, tt.want)
		}
	}
}
