package mcu

import "testing"

// Ported from contrast_test.dart.
func TestContrast(t *testing.T) {
	closeTo(t, "ratioOfTones out of bounds", RatioOfTones(-10.0, 110.0), 21.0, 0.001)
	closeTo(t, "lighter impossible ratio", Lighter(90.0, 10.0), -1.0, 0.001)
	closeTo(t, "lighter above", Lighter(110.0, 2.0), -1.0, 0.001)
	closeTo(t, "lighter below", Lighter(-10.0, 2.0), -1.0, 0.001)
	closeTo(t, "lighterUnsafe", LighterUnsafe(100.0, 2.0), 100, 0.001)
	closeTo(t, "darker impossible ratio", Darker(10.0, 20.0), -1.0, 0.001)
	closeTo(t, "darker above", Darker(110.0, 2.0), -1.0, 0.001)
	closeTo(t, "darker below", Darker(-10.0, 2.0), -1.0, 0.001)
	closeTo(t, "darkerUnsafe", DarkerUnsafe(0.0, 2.0), 0.0, 0.001)
	// The values that the functions find reach the ratio.
	for _, tone := range []float64{0, 20, 40, 60, 80, 100} {
		for _, ratio := range []float64{1.5, 3, 4.5, 7} {
			if l := Lighter(tone, ratio); l >= 0 && RatioOfTones(l, tone) < ratio-0.01 {
				t.Errorf("lighter(%v, %v) = %v has the ratio %v", tone, ratio, l, RatioOfTones(l, tone))
			}
			if d := Darker(tone, ratio); d >= 0 && RatioOfTones(d, tone) < ratio-0.01 {
				t.Errorf("darker(%v, %v) = %v has the ratio %v", tone, ratio, d, RatioOfTones(d, tone))
			}
		}
	}
}

// Ported from blend_test.dart.
func TestHarmonize(t *testing.T) {
	const yellow uint32 = 0xffffff00
	tests := []struct {
		name         string
		design, from uint32
		want         uint32
	}{
		{"redToBlue", red, blue, 0xffFB0057}, {"redToGreen", red, green, 0xffD85600}, {"redToYellow", red, yellow, 0xffD85600},
		{"blueToGreen", blue, green, 0xff0047A3}, {"blueToRed", blue, red, 0xff5700DC}, {"blueToYellow", blue, yellow, 0xff0047A3},
		{"greenToBlue", green, blue, 0xff00FC94}, {"greenToRed", green, red, 0xffB1F000}, {"greenToYellow", green, yellow, 0xffB1F000},
		{"yellowToBlue", yellow, blue, 0xffEBFFBA}, {"yellowToGreen", yellow, green, 0xffEBFFBA}, {"yellowToRed", yellow, red, 0xffFFF6E3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireColor(t, tt.name, Harmonize(tt.design, tt.from), tt.want)
		})
	}
}

func TestBlendHctHueAndUcs(t *testing.T) {
	// At amount 0 the color is unchanged by the hue blend (to within the gamut).
	got := BlendHctHue(red, blue, 0)
	if d := Cam16FromInt(got).Distance(Cam16FromInt(red)); d > 1 {
		t.Errorf("blend at 0 = %s, %v away from red", HexFromArgb(got, true), d)
	}
	// At amount 1 the CAM16-UCS blend is the second color.
	if d := Cam16FromInt(BlendCam16Ucs(red, blue, 1)).Distance(Cam16FromInt(blue)); d > 1 {
		t.Errorf("ucs blend at 1 is %v away from blue", d)
	}
	// Halfway the hue of the result is on the short way from red to blue, through
	// magenta.
	mid := HctFromInt(BlendHctHue(red, blue, 0.5))
	if mid.Hue() <= 283 && mid.Hue() >= 27 {
		t.Errorf("hue at 0.5 = %v, want between blue and red through magenta", mid.Hue())
	}
}

// Ported from dislike_analyzer_test.dart.
func TestDislikeAnalyzer(t *testing.T) {
	t.Run("Monk Skin Tone Scale colors liked", func(t *testing.T) {
		// From https://skintone.google#/get-started
		for _, c := range []uint32{0xfff6ede4, 0xfff3e7db, 0xfff7ead0, 0xffeadaba, 0xffd7bd96, 0xffa07e56, 0xff825c43, 0xff604134, 0xff3a312a, 0xff292420} {
			if IsDisliked(HctFromInt(c)) {
				t.Errorf("%s is disliked", HexFromArgb(c, true))
			}
		}
	})
	bile := []uint32{0xff95884B, 0xff716B40, 0xffB08E00, 0xff4C4308, 0xff464521}
	t.Run("bile colors disliked and then fixed", func(t *testing.T) {
		for _, c := range bile {
			hct := HctFromInt(c)
			if !IsDisliked(hct) {
				t.Errorf("%s was likable", HexFromArgb(c, true))
			}
			if IsDisliked(FixIfDisliked(hct)) {
				t.Errorf("%s is still disliked after the fix", HexFromArgb(c, true))
			}
		}
	})
	t.Run("tone 67 not disliked", func(t *testing.T) {
		c := HctFrom(100.0, 50.0, 67.0)
		if IsDisliked(c) || FixIfDisliked(c).ToInt() != c.ToInt() {
			t.Error("tone 67 was changed")
		}
	})
}

// Ported from dynamic_scheme_test.dart.
func TestGetRotatedHue(t *testing.T) {
	src := HctFrom(43, 16, 16)
	tests := []struct {
		name            string
		hues, rotations []float64
		want            float64
	}{
		{"0 length input", nil, nil, 43},
		{"1 length input no rotation", []float64{0}, []float64{0}, 43},
		{"on boundary rotation correct", []float64{0, 42, 360}, []float64{0, 15, 0}, 43 + 15},
		{"rotation result larger than 360 degrees wraps", []float64{0, 42, 360}, []float64{0, 480, 0}, 163},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			closeTo(t, "hue", GetRotatedHue(src, tt.hues, tt.rotations), tt.want, 1.0)
		})
	}
	t.Run("input length mismatch panics", func(t *testing.T) {
		skipUnlessDebug(t)
		defer func() {
			appErr, ok := asError(recover())
			if !ok || !appErr.HasRoute("GetRotatedHue.DynamicSchemeRotations") {
				t.Errorf("panic = %v", appErr)
			}
		}()
		GetRotatedHue(src, []float64{0, 1}, []float64{0})
	})
}

func TestNewDynamicSchemeNeedsItsPalettes(t *testing.T) {
	skipUnlessDebug(t)
	defer func() {
		appErr, ok := asError(recover())
		if !ok || !appErr.HasRoute("NewDynamicScheme.DynamicSchemeIncomplete") {
			t.Errorf("panic = %v", appErr)
		}
	}()
	NewDynamicScheme(DynamicSchemeOpts{SourceColorHct: HctFromInt(red)})
}

// Ported from temperature_cache_test.dart.
func TestTemperatureCache(t *testing.T) {
	t.Run("raw temperature", func(t *testing.T) {
		closeTo(t, "blue", RawTemperature(HctFromInt(0xff0000ff)), -1.393, 0.001)
		closeTo(t, "red", RawTemperature(HctFromInt(0xffff0000)), 2.351, 0.001)
		closeTo(t, "green", RawTemperature(HctFromInt(0xff00ff00)), -0.267, 0.001)
		closeTo(t, "white", RawTemperature(HctFromInt(0xffffffff)), -0.5, 0.001)
		closeTo(t, "black", RawTemperature(HctFromInt(0xff000000)), -0.5, 0.001)
	})
	t.Run("relative temperature", func(t *testing.T) {
		closeTo(t, "blue", NewTemperatureCache(HctFromInt(0xff0000ff)).InputRelativeTemperature(), 0.0, 0.001)
		closeTo(t, "red", NewTemperatureCache(HctFromInt(0xffff0000)).InputRelativeTemperature(), 1.0, 0.001)
		closeTo(t, "green", NewTemperatureCache(HctFromInt(0xff00ff00)).InputRelativeTemperature(), 0.467, 0.001)
		closeTo(t, "white", NewTemperatureCache(HctFromInt(0xffffffff)).InputRelativeTemperature(), 0.5, 0.001)
		closeTo(t, "black", NewTemperatureCache(HctFromInt(0xff000000)).InputRelativeTemperature(), 0.5, 0.001)
	})
	t.Run("complement", func(t *testing.T) {
		for _, tc := range [][2]uint32{
			{0xff0000ff, 0xff9d0002}, {0xffff0000, 0xff007bfc}, {0xff00ff00, 0xffffd2c9}, {0xffffffff, 0xffffffff}, {0xff000000, 0xff000000},
		} {
			requireColor(t, HexFromArgb(tc[0], true), NewTemperatureCache(HctFromInt(tc[0])).Complement().ToInt(), tc[1])
		}
	})
	t.Run("analogous", func(t *testing.T) {
		for _, tc := range []struct {
			input uint32
			want  [5]uint32
		}{
			{0xff0000ff, [5]uint32{0xff00590c, 0xff00564e, 0xff0000ff, 0xff6700cc, 0xff81009f}},
			{0xffff0000, [5]uint32{0xfff60082, 0xfffc004c, 0xffff0000, 0xffd95500, 0xffaf7200}},
			{0xff00ff00, [5]uint32{0xffcee900, 0xff92f500, 0xff00ff00, 0xff00fd6f, 0xff00fab3}},
			{0xff000000, [5]uint32{0xff000000, 0xff000000, 0xff000000, 0xff000000, 0xff000000}},
			{0xffffffff, [5]uint32{0xffffffff, 0xffffffff, 0xffffffff, 0xffffffff, 0xffffffff}},
		} {
			got := NewTemperatureCache(HctFromInt(tc.input)).Analogous(5, 12)
			if len(got) != 5 {
				t.Fatalf("analogous of %s has %d colors", HexFromArgb(tc.input, true), len(got))
			}
			for i, h := range got {
				requireColor(t, HexFromArgb(tc.input, true)+" analogous "+string(rune('0'+i)), h.ToInt(), tc.want[i])
			}
		}
	})
}
