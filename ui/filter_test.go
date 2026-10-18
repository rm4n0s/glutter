package ui

import "testing"

func TestMaskFilter(t *testing.T) {
	a := NewMaskFilterBlur(BlurStyleSolid, 4.25)
	if a.Style() != BlurStyleSolid || a.Sigma() != 4.25 {
		t.Errorf("Style/Sigma = %v %v", a.Style(), a.Sigma())
	}
	if NewMaskFilterBlur(BlurStyleUnspecified, 1).Style() != BlurStyleNormal {
		t.Error("an unspecified style is normal")
	}
	tests := []struct {
		name string
		x, y *MaskFilter
		want bool
	}{
		{"same values", a, NewMaskFilterBlur(BlurStyleSolid, 4.25), true},
		{"other style", a, NewMaskFilterBlur(BlurStyleInner, 4.25), false},
		{"other sigma", a, NewMaskFilterBlur(BlurStyleSolid, 4), false},
		{"nil and nil", nil, nil, true},
		{"nil and a filter", nil, a, false},
		{"a filter and nil", a, nil, false},
	}
	for _, tt := range tests {
		if got := tt.x.Equal(tt.y); got != tt.want {
			t.Errorf("%s: Equal = %v, want %v", tt.name, got, tt.want)
		}
	}
	if got, want := a.String(), "MaskFilter.blur(BlurStyle.solid, 4.3)"; got != want {
		t.Errorf("String = %q, want %q", got, want)
	}
}

func TestColorFilters(t *testing.T) {
	red := NewColor(0xFFFF0000)
	matrix := [20]float64{1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0.5, 0}
	tests := []struct {
		name string
		f    *ColorFilter
		kind ColorFilterKind
		str  string
	}{
		{"mode", NewColorFilterMode(red, BlendModeMultiply), ColorFilterKindMode,
			"ColorFilter.mode(Color(alpha: 1.0000, red: 1.0000, green: 0.0000, blue: 0.0000, colorSpace: ColorSpace.sRGB), BlendMode.multiply)"},
		{"mode, default blend", NewColorFilterMode(red, BlendModeUnspecified), ColorFilterKindMode,
			"ColorFilter.mode(Color(alpha: 1.0000, red: 1.0000, green: 0.0000, blue: 0.0000, colorSpace: ColorSpace.sRGB), BlendMode.srcOver)"},
		{"matrix", NewColorFilterMatrix(matrix), ColorFilterKindMatrix,
			"ColorFilter.matrix([1.0, 0.0, 0.0, 0.0, 0.0, 0.0, 1.0, 0.0, 0.0, 0.0, 0.0, 0.0, 1.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.5, 0.0])"},
		{"linear to sRGB", NewColorFilterLinearToSrgbGamma(), ColorFilterKindLinearToSrgbGamma, "ColorFilter.linearToSrgbGamma()"},
		{"sRGB to linear", NewColorFilterSrgbToLinearGamma(), ColorFilterKindSrgbToLinearGamma, "ColorFilter.srgbToLinearGamma()"},
	}
	for _, tt := range tests {
		if tt.f.Kind() != tt.kind || tt.f.String() != tt.str || tt.f.DebugShortDescription() != tt.str {
			t.Errorf("%s: kind %v string %q", tt.name, tt.f.Kind(), tt.f.String())
		}
	}
	m := tests[0].f
	if m.Color() != red || m.BlendMode() != BlendModeMultiply {
		t.Errorf("mode accessors: %v %v", m.Color(), m.BlendMode())
	}
	if tests[2].f.Matrix() != matrix {
		t.Errorf("matrix accessor: %v", tests[2].f.Matrix())
	}

	equal := []struct {
		name string
		a, b ImageFilter
		want bool
	}{
		{"same mode", NewColorFilterMode(red, BlendModeMultiply), NewColorFilterMode(red, BlendModeMultiply), true},
		{"other color", NewColorFilterMode(red, BlendModeMultiply), NewColorFilterMode(NewColor(0xFF00FF00), BlendModeMultiply), false},
		{"other blend", NewColorFilterMode(red, BlendModeMultiply), NewColorFilterMode(red, BlendModeScreen), false},
		{"same matrix", NewColorFilterMatrix(matrix), NewColorFilterMatrix(matrix), true},
		{"other matrix", NewColorFilterMatrix(matrix), NewColorFilterMatrix([20]float64{}), false},
		{"gamma filters differ", NewColorFilterLinearToSrgbGamma(), NewColorFilterSrgbToLinearGamma(), false},
		{"same gamma", NewColorFilterLinearToSrgbGamma(), NewColorFilterLinearToSrgbGamma(), true},
		{"not a color filter", NewColorFilterLinearToSrgbGamma(), NewImageFilterErode(), false},
	}
	for _, tt := range equal {
		if got := tt.a.Equal(tt.b); got != tt.want {
			t.Errorf("%s: Equal = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestColorFilterSaturation(t *testing.T) {
	// Computed with Dart's formula for saturation 0.5.
	f := NewColorFilterSaturation(0.5)
	m := f.Matrix()
	if f.Kind() != ColorFilterKindMatrix {
		t.Fatalf("kind = %v", f.Kind())
	}
	want := map[int]float64{0: 0.6063, 1: 0.3576, 2: 0.0361, 5: 0.1063, 6: 0.8576, 7: 0.0361, 10: 0.1063, 11: 0.3576, 12: 0.5361, 18: 1}
	for i, w := range want {
		if !near(m[i], w, 1e-12) {
			t.Errorf("matrix[%d] = %v, want %v", i, m[i], w)
		}
	}
	// Saturation 1 is the identity, 0 is gray: every row the luminance weights.
	id := NewColorFilterSaturation(1).Matrix()
	for i, v := range id {
		w := 0.0
		if i%6 == 0 && i < 20 {
			w = 1
		}
		if !near(v, w, 1e-12) {
			t.Errorf("saturation 1: matrix[%d] = %v, want %v", i, v, w)
		}
	}
	gray := NewColorFilterSaturation(0).Matrix()
	for row := 0; row < 3; row++ {
		if gray[row*5] != 0.2126 || gray[row*5+1] != 0.7152 || gray[row*5+2] != 0.0722 {
			t.Errorf("saturation 0, row %d = %v", row, gray[row*5:row*5+5])
		}
	}
}

func TestImageFilters(t *testing.T) {
	bounds := Rect{1, 2, 3, 4}
	m := NewMatrix4Diagonal3Values(2, 3, 1)
	tests := []struct {
		name  string
		f     ImageFilter
		str   string
		short string
	}{
		{"blur", NewImageFilterBlur(ImageFilterBlurOpts{SigmaX: 3, SigmaY: 4.5}),
			"ImageFilter.blur(3.0, 4.5, unspecified)", "blur(3.0, 4.5, unspecified)"},
		{"blur, tile mode and bounds", NewImageFilterBlur(ImageFilterBlurOpts{SigmaX: 1, TileMode: TileModeDecal, Bounds: &bounds}),
			"ImageFilter.blur(1.0, 0.0, decal, bounds: Rect.fromLTRB(1.0, 2.0, 3.0, 4.0))", "blur(1.0, 0.0, decal, bounds: Rect.fromLTRB(1.0, 2.0, 3.0, 4.0))"},
		{"blur, defaults", NewImageFilterBlur(), "ImageFilter.blur(0.0, 0.0, unspecified)", "blur(0.0, 0.0, unspecified)"},
		{"dilate", NewImageFilterDilate(ImageFilterMorphologyOpts{RadiusX: 2, RadiusY: 3}), "ImageFilter.dilate(2.0, 3.0)", "dilate(2.0, 3.0)"},
		{"erode", NewImageFilterErode(ImageFilterMorphologyOpts{RadiusX: 2.5}), "ImageFilter.erode(2.5, 0.0)", "erode(2.5, 0.0)"},
		{"matrix", NewImageFilterMatrix(m),
			"ImageFilter.matrix([2.0, 0.0, 0.0, 0.0, 0.0, 3.0, 0.0, 0.0, 0.0, 0.0, 1.0, 0.0, 0.0, 0.0, 0.0, 1.0], FilterQuality.medium)",
			"matrix([2.0, 0.0, 0.0, 0.0, 0.0, 3.0, 0.0, 0.0, 0.0, 0.0, 1.0, 0.0, 0.0, 0.0, 0.0, 1.0], FilterQuality.medium)"},
		{"matrix, quality", NewImageFilterMatrix(NewMatrix4Identity(), ImageFilterMatrixOpts{FilterQuality: FilterQualityHigh}),
			"ImageFilter.matrix([1.0, 0.0, 0.0, 0.0, 0.0, 1.0, 0.0, 0.0, 0.0, 0.0, 1.0, 0.0, 0.0, 0.0, 0.0, 1.0], FilterQuality.high)",
			"matrix([1.0, 0.0, 0.0, 0.0, 0.0, 1.0, 0.0, 0.0, 0.0, 0.0, 1.0, 0.0, 0.0, 0.0, 0.0, 1.0], FilterQuality.high)"},
		{"compose", NewImageFilterCompose(NewImageFilterErode(), NewImageFilterDilate(ImageFilterMorphologyOpts{RadiusX: 1})),
			"ImageFilter.compose(source -> dilate(1.0, 0.0) -> erode(0.0, 0.0) -> result)", "dilate(1.0, 0.0) -> erode(0.0, 0.0)"},
	}
	for _, tt := range tests {
		if tt.f.String() != tt.str || tt.f.DebugShortDescription() != tt.short {
			t.Errorf("%s:\n String %q\n want   %q\n short  %q\n want   %q", tt.name, tt.f.String(), tt.str, tt.f.DebugShortDescription(), tt.short)
		}
	}
}

func TestImageFilterAccessors(t *testing.T) {
	bounds := Rect{1, 2, 3, 4}
	blur := NewImageFilterBlur(ImageFilterBlurOpts{SigmaX: 3, SigmaY: 4, TileMode: TileModeMirror, Bounds: &bounds}).(*BlurImageFilter)
	if got, ok := blur.Bounds(); blur.SigmaX() != 3 || blur.SigmaY() != 4 || blur.TileMode() != TileModeMirror || !ok || got != bounds {
		t.Errorf("blur accessors: %v %v %v %v", blur.SigmaX(), blur.SigmaY(), blur.TileMode(), got)
	}
	if _, ok := NewImageFilterBlur().(*BlurImageFilter).Bounds(); ok {
		t.Error("no bounds were given")
	}
	bounds.Left = 99
	if got, _ := blur.Bounds(); got.Left != 1 {
		t.Error("the bounds must be copied")
	}
	dilate := NewImageFilterDilate(ImageFilterMorphologyOpts{RadiusX: 1, RadiusY: 2}).(*DilateImageFilter)
	erode := NewImageFilterErode(ImageFilterMorphologyOpts{RadiusX: 3, RadiusY: 4}).(*ErodeImageFilter)
	if dilate.RadiusX() != 1 || dilate.RadiusY() != 2 || erode.RadiusX() != 3 || erode.RadiusY() != 4 {
		t.Error("morphology accessors wrong")
	}
	m := NewMatrix4TranslationValues(1, 2, 3)
	mf := NewImageFilterMatrix(m).(*MatrixImageFilter)
	m.storage[12] = 50
	if mf.Matrix().storage[12] != 1 || mf.FilterQuality() != FilterQualityMedium {
		t.Error("the matrix filter must copy the matrix and default to medium quality")
	}
	mf.Matrix().storage[12] = 77
	if mf.Matrix().storage[12] != 1 {
		t.Error("Matrix must return a copy")
	}
	outer, inner := NewImageFilterErode(), NewImageFilterDilate()
	c := NewImageFilterCompose(outer, inner).(*ComposeImageFilter)
	if c.Outer() != outer || c.Inner() != inner {
		t.Error("compose accessors wrong")
	}
}

func TestImageFilterEquality(t *testing.T) {
	bounds := Rect{1, 2, 3, 4}
	filters := []struct {
		name string
		make func() ImageFilter
	}{
		{"blur", func() ImageFilter { return NewImageFilterBlur(ImageFilterBlurOpts{SigmaX: 1, SigmaY: 2}) }},
		{"blur, tile mode", func() ImageFilter {
			return NewImageFilterBlur(ImageFilterBlurOpts{SigmaX: 1, SigmaY: 2, TileMode: TileModeClamp})
		}},
		{"blur, bounds", func() ImageFilter {
			return NewImageFilterBlur(ImageFilterBlurOpts{SigmaX: 1, SigmaY: 2, Bounds: &bounds})
		}},
		{"dilate", func() ImageFilter { return NewImageFilterDilate(ImageFilterMorphologyOpts{RadiusX: 1}) }},
		{"erode", func() ImageFilter { return NewImageFilterErode(ImageFilterMorphologyOpts{RadiusX: 1}) }},
		{"matrix", func() ImageFilter { return NewImageFilterMatrix(NewMatrix4TranslationValues(1, 2, 0)) }},
		{"matrix, quality", func() ImageFilter {
			return NewImageFilterMatrix(NewMatrix4TranslationValues(1, 2, 0), ImageFilterMatrixOpts{FilterQuality: FilterQualityLow})
		}},
		{"compose", func() ImageFilter {
			return NewImageFilterCompose(NewImageFilterErode(), NewImageFilterDilate())
		}},
		{"compose, swapped", func() ImageFilter {
			return NewImageFilterCompose(NewImageFilterDilate(), NewImageFilterErode())
		}},
		{"color filter", func() ImageFilter { return NewColorFilterSrgbToLinearGamma() }},
	}
	for i, a := range filters {
		for j, b := range filters {
			// Each filter equals a fresh copy of itself and no other.
			if got := a.make().Equal(b.make()); got != (i == j) {
				t.Errorf("%s == %s: %v, want %v", a.name, b.name, got, i == j)
			}
		}
	}
}
