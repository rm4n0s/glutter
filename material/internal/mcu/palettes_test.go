package mcu

import (
	"slices"
	"testing"
)

func closeToColor(t *testing.T, name string, got, want uint32) {
	t.Helper()
	if d := Cam16FromInt(got).Distance(Cam16FromInt(want)); d > 5 {
		t.Errorf("%s = %s, want close to %s (distance %v)", name, HexFromArgb(got, true), HexFromArgb(want, true), d)
	}
}

func constantList(c uint32) []uint32 {
	out := make([]uint32, CommonSize)
	for i := range out {
		out[i] = c
	}
	return out
}

// Ported from palettes_test.dart.
func TestTonalPaletteEquality(t *testing.T) {
	// Regression test for https://github.com/material-foundation/material-color-utilities/issues/56
	a1, a2 := TonalPaletteOf(1, 1), TonalPaletteOf(1, 1)
	b1, b2 := TonalPaletteFromList(constantList(0xDEADBEEF)), TonalPaletteFromList(constantList(0xDEADBEEF))
	if a1.Equal(b1) || b1.Equal(a1) {
		t.Error("a palette of hue and chroma equals one of a list")
	}
	if !a1.Equal(a2) || !b1.Equal(b2) {
		t.Error("the same palettes are not equal")
	}
	c1 := TonalPaletteFromList(constantList(123))
	list := constantList(123)
	for i, tone := range CommonTones {
		if tone < 15 {
			list[i] = 456
		}
	}
	c2 := TonalPaletteFromList(list)
	if c1.Get(50) != c2.Get(50) {
		t.Error("the colors at tone 50 differ")
	}
	if c1.Equal(c2) {
		t.Error("palettes that differ at tone 0 are equal")
	}
}

func TestTonalPaletteOfBlue(t *testing.T) {
	hct := HctFromInt(0xff0000ff)
	tones := TonalPaletteOf(hct.Hue(), hct.Chroma())
	want := map[int]uint32{
		0: 0xff000000, 10: 0xff00006e, 20: 0xff0001ac, 30: 0xff0000ef, 40: 0xff343dff, 50: 0xff5a64ff, 60: 0xff7c84ff,
		70: 0xff9da3ff, 80: 0xffbec2ff, 90: 0xffe0e0ff, 95: 0xfff1efff, 99: 0xfffffbff, 100: 0xffffffff,
		3: 0xff00003c, // a tone that is not a common one
	}
	for tone, c := range want {
		requireColor(t, "tone "+string(rune('0'+tone/100))+string(rune('0'+tone/10%10))+string(rune('0'+tone%10)), tones.Get(tone), c)
	}
	asList := []uint32{0xff000000, 0xff00006e, 0xff0001ac, 0xff0000ef, 0xff343dff, 0xff5a64ff, 0xff7c84ff, 0xff9da3ff, 0xffbec2ff, 0xffe0e0ff, 0xfff1efff, 0xfffffbff, 0xffffffff}
	if got := tones.AsList(); !slices.Equal(got, asList) {
		t.Errorf("asList = %x, want %x", got, asList)
	}
	a, b := TonalPaletteOf(hct.Hue(), hct.Chroma()), TonalPaletteOf(hct.Hue(), hct.Chroma())
	hctC := HctFromInt(0xff123456)
	c := TonalPaletteOf(hctC.Hue(), hctC.Chroma())
	if !a.Equal(b) || b.Equal(c) {
		t.Error("equality of palettes of hue and chroma")
	}
}

func TestTonalPaletteFromList(t *testing.T) {
	hueChroma := TonalPaletteOf(270, 36)
	cached := hueChroma.AsList()
	broken := slices.Clone(cached)
	broken[2] = HctFrom(180, 24, 20).ToInt()
	broken[9] = HctFrom(0, 12, 90).ToInt()
	rebuilt := TonalPaletteFromList(broken)

	t.Run("correctly deduces original hue and chroma", func(t *testing.T) {
		closeTo(t, "hue", rebuilt.Hue(), 270, 1)
		closeTo(t, "chroma", rebuilt.Chroma(), 36, 1)
	})
	t.Run("low-chroma noise does not affect the hue and chroma deduced", func(t *testing.T) {
		clean := TonalPaletteFromList(cached)
		if rebuilt.Hue() != clean.Hue() || rebuilt.Chroma() != clean.Chroma() {
			t.Errorf("rebuilt (%v, %v) differs from clean (%v, %v)", rebuilt.Hue(), rebuilt.Chroma(), clean.Hue(), clean.Chroma())
		}
	})
	t.Run("returns cached colors when possible", func(t *testing.T) {
		requireColor(t, "20", rebuilt.Get(20), broken[2])
		requireColor(t, "50", rebuilt.Get(50), broken[5])
		requireColor(t, "90", rebuilt.Get(90), broken[9])
		requireColor(t, "99", rebuilt.Get(99), broken[11])
	})
	t.Run("correctly deduces colors at other tones", func(t *testing.T) {
		for _, tone := range []int{15, 53, 78} {
			closeToColor(t, "tone", rebuilt.Get(tone), hueChroma.Get(tone))
			closeTo(t, "deduced tone", rebuilt.GetHct(float64(tone)).Tone(), float64(tone), 1)
		}
	})
	t.Run("asList", func(t *testing.T) {
		ints := make([]uint32, CommonSize)
		for i := range ints {
			ints[i] = uint32(i)
		}
		if got := TonalPaletteFromList(ints).AsList(); !slices.Equal(got, ints) {
			t.Errorf("asList = %v, want %v", got, ints)
		}
	})
	t.Run("operator ==", func(t *testing.T) {
		palette1, palette2, palette3 := TonalPaletteOf(270, 36), TonalPaletteOf(180, 36), TonalPaletteOf(270, 12)
		palette4 := TonalPaletteFromList(palette1.AsList())
		brokenList := palette1.AsList()
		brokenList[2] = HctFrom(180, 24, 20).ToInt()
		brokenList[9] = HctFrom(0, 12, 90).ToInt()
		palette5 := TonalPaletteFromList(brokenList)
		palettes := []*TonalPalette{palette1, palette2, palette3, palette4, palette5}
		// The grid of the Dart test: 1 equals 4, and every palette equals itself.
		for i, a := range palettes {
			for j, b := range palettes {
				want := i == j || (i == 0 && j == 3) || (i == 3 && j == 0)
				if got := a.Equal(b); got != want {
					t.Errorf("palette %d == palette %d is %v, want %v", i+1, j+1, got, want)
				}
			}
		}
	})
	t.Run("String", func(t *testing.T) {
		if got := TonalPaletteOf(270, 36).String(); got != "TonalPalette.of(270, 36)" {
			t.Errorf("String = %q", got)
		}
		if got := TonalPaletteFromList(constantList(1)).String(); got == "" || got[:24] != "TonalPalette.fromList([1" {
			t.Errorf("String = %q", got)
		}
	})
}

func TestTonalPaletteFromListSize(t *testing.T) {
	skipUnlessDebug(t)
	defer func() {
		appErr, ok := asError(recover())
		if !ok || !appErr.HasRoute("TonalPaletteFromList.TonalPaletteListSize") {
			t.Errorf("panic = %v", appErr)
		}
	}()
	TonalPaletteFromList([]uint32{1, 2, 3})
}

func TestTonalPaletteFromHctKeepsTheColorAsTheKeyColor(t *testing.T) {
	hct := HctFromInt(0xff123456)
	p := TonalPaletteFromHct(hct)
	if p.KeyColor() != hct || p.Hue() != hct.Hue() || p.Chroma() != hct.Chroma() {
		t.Error("the palette of a color is not made from it")
	}
}

func TestCorePalette(t *testing.T) {
	t.Run("asList", func(t *testing.T) {
		ints := make([]uint32, CorePaletteSize*CommonSize)
		for i := range ints {
			ints[i] = uint32(i)
		}
		if got := CorePaletteFromList(ints).AsList(); !slices.Equal(got, ints) {
			t.Errorf("asList = %v", got)
		}
	})
	t.Run("operator ==", func(t *testing.T) {
		a, b, c := CorePaletteOf(0xff0000ff), CorePaletteOf(0xff0000ff), CorePaletteOf(0xff123456)
		if !a.Equal(b) || b.Equal(c) || a.Equal(nil) {
			t.Error("equality of core palettes")
		}
		if a.String() == "" {
			t.Error("no string")
		}
	})
	tones := []int{100, 95, 90, 80, 70, 60, 50, 40, 30, 20, 10, 0}
	t.Run("of blue", func(t *testing.T) {
		core := CorePaletteOf(0xff0000ff)
		primary := []uint32{0xffffffff, 0xfff1efff, 0xffe0e0ff, 0xffbec2ff, 0xff9da3ff, 0xff7c84ff, 0xff5a64ff, 0xff343dff, 0xff0000ef, 0xff0001ac, 0xff00006e, 0xff000000}
		secondary := []uint32{0xffffffff, 0xfff1efff, 0xffe1e0f9, 0xffc5c4dd, 0xffa9a9c1, 0xff8f8fa6, 0xff75758b, 0xff5c5d72, 0xff444559, 0xff2e2f42, 0xff191a2c, 0xff000000}
		for i, tone := range tones {
			requireColor(t, "primary", core.Primary.Get(tone), primary[i])
			requireColor(t, "secondary", core.Secondary.Get(tone), secondary[i])
		}
	})
	t.Run("content of blue", func(t *testing.T) {
		core := CorePaletteContentOf(0xff0000ff)
		primary := []uint32{0xffffffff, 0xfff1efff, 0xffe0e0ff, 0xffbec2ff, 0xff9da3ff, 0xff7c84ff, 0xff5a64ff, 0xff343dff, 0xff0000ef, 0xff0001ac, 0xff00006e, 0xff000000}
		secondary := []uint32{0xffffffff, 0xfff1efff, 0xffe0e0ff, 0xffc1c3f4, 0xffa5a7d7, 0xff8b8dbb, 0xff7173a0, 0xff585b86, 0xff40436d, 0xff2a2d55, 0xff14173f, 0xff000000}
		for i, tone := range tones {
			requireColor(t, "primary", core.Primary.Get(tone), primary[i])
			requireColor(t, "secondary", core.Secondary.Get(tone), secondary[i])
		}
	})
}

func TestCorePaletteFromListSize(t *testing.T) {
	skipUnlessDebug(t)
	defer func() {
		appErr, ok := asError(recover())
		if !ok || !appErr.HasRoute("CorePaletteFromList.CorePaletteListSize") {
			t.Errorf("panic = %v", appErr)
		}
	}()
	CorePaletteFromList([]uint32{1})
}

func TestKeyColor(t *testing.T) {
	t.Run("exact chroma is available", func(t *testing.T) {
		result := TonalPaletteOf(50.0, 60.0).KeyColor()
		closeTo(t, "hue", result.Hue(), 50.0, 10.0)
		closeTo(t, "chroma", result.Chroma(), 60.0, 0.5)
		if result.Tone() <= 0 || result.Tone() >= 100 {
			t.Errorf("tone = %v", result.Tone())
		}
	})
	t.Run("requesting unusually high chroma", func(t *testing.T) {
		// For hue 149, the chroma peak is 89.6 at tone 87.9.
		result := TonalPaletteOf(149.0, 200.0).KeyColor()
		closeTo(t, "hue", result.Hue(), 149.0, 10.0)
		if result.Chroma() <= 89.0 {
			t.Errorf("chroma = %v, want above 89", result.Chroma())
		}
		if result.Tone() <= 0 || result.Tone() >= 100 {
			t.Errorf("tone = %v", result.Tone())
		}
	})
	t.Run("requesting unusually low chroma", func(t *testing.T) {
		// The key color is the first tone from tone 50 that matches the hue and
		// the chroma: with a very low chroma it is close to tone 50.
		result := TonalPaletteOf(50.0, 3.0).KeyColor()
		closeTo(t, "hue", result.Hue(), 50.0, 10.0)
		closeTo(t, "chroma", result.Chroma(), 3.0, 0.5)
		closeTo(t, "tone", result.Tone(), 50.0, 0.5)
	})
}

func TestSchemeFromCorePalette(t *testing.T) {
	light, dark := SchemeLight(0xff6750a4), SchemeDark(0xff6750a4)
	if light.Primary != CorePaletteOf(0xff6750a4).Primary.Get(40) || dark.Primary != CorePaletteOf(0xff6750a4).Primary.Get(80) {
		t.Error("the primary colors are not the tones 40 and 80")
	}
	if light.Background != light.Surface || dark.Shadow != 0xff000000 || light.Scrim != 0xff000000 {
		t.Error("background, shadow or scrim")
	}
	// The dark error container text is tone 80 in the Dart source.
	if dark.OnErrorContainer != CorePaletteOf(0xff6750a4).Error.Get(80) {
		t.Error("onErrorContainer of the dark scheme is not tone 80")
	}
	if SchemeLightContent(0xff6750a4).Primary == 0 || SchemeDarkContent(0xff6750a4).Primary == 0 {
		t.Error("the content schemes")
	}
}
