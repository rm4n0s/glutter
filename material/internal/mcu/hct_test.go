package mcu

import (
	"fmt"
	"math"
	"os"
	"testing"
)

const (
	black   uint32 = 0xff000000
	white   uint32 = 0xffffffff
	red     uint32 = 0xffff0000
	green   uint32 = 0xff00ff00
	blue    uint32 = 0xff0000ff
	midgray uint32 = 0xff777777
)

func closeTo(t *testing.T, name string, got, want, delta float64) {
	t.Helper()
	if math.IsNaN(got) || math.Abs(got-want) > delta {
		t.Errorf("%s = %v, want %v ± %v", name, got, want, delta)
	}
}

func requireColor(t *testing.T, name string, got, want uint32) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %s, want %s", name, HexFromArgb(got, true), HexFromArgb(want, true))
	}
}

func colorIsOnBoundary(argb uint32) bool {
	r, g, b := RedFromArgb(argb), GreenFromArgb(argb), BlueFromArgb(argb)
	return r == 0 || r == 255 || g == 0 || g == 255 || b == 0 || b == 255
}

// Ported from hct_test.dart.
func TestHctEquality(t *testing.T) {
	if !HctFromInt(123).Equal(HctFromInt(123)) {
		t.Error("the colors of the same ARGB are not equal")
	}
	if HctFromInt(123).Equal(HctFromInt(124)) || HctFromInt(123).Equal(nil) {
		t.Error("different colors are equal")
	}
}

func TestConversionsAreReflexive(t *testing.T) {
	requireColor(t, "red", Cam16FromInt(red).Viewed(ViewingConditionsStandard), red)
}

func TestYFromLstar(t *testing.T) {
	closeTo(t, "y of midgray", YFromLstar(50.0), 18.418, 0.001)
	closeTo(t, "y of black", YFromLstar(0.0), 0.0, 0.001)
	closeTo(t, "y of white", YFromLstar(100.0), 100.0, 0.001)
}

func TestCam16OfPrimaries(t *testing.T) {
	tests := []struct {
		name              string
		argb              uint32
		j, chroma, hue, m float64
		s, q              float64
	}{
		{"red", red, 46.445, 113.357, 27.408, 89.494, 91.889, 105.988},
		{"green", green, 79.331, 108.410, 142.139, 85.587, 78.604, 138.520},
		{"blue", blue, 25.465, 87.230, 282.788, 68.867, 93.674, 78.481},
		{"black", black, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0},
		{"white", white, 100.0, 2.869, 209.492, 2.265, 12.068, 155.521},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cam := Cam16FromInt(tt.argb)
			closeTo(t, "j", cam.J, tt.j, 0.001)
			closeTo(t, "chroma", cam.Chroma, tt.chroma, 0.001)
			closeTo(t, "hue", cam.Hue, tt.hue, 0.001)
			closeTo(t, "m", cam.M, tt.m, 0.001)
			closeTo(t, "s", cam.S, tt.s, 0.001)
			closeTo(t, "q", cam.Q, tt.q, 0.001)
		})
	}
}

func TestGamutMap(t *testing.T) {
	// The Dart test of midgray tests green, a copy and paste of the original;
	// midgray is added here.
	for name, argb := range map[string]uint32{"red": red, "green": green, "blue": blue, "white": white, "midgray": midgray} {
		cam := Cam16FromInt(argb)
		requireColor(t, name, HctFrom(cam.Hue, cam.Chroma, LstarFromArgb(argb)).ToInt(), argb)
	}
}

func TestHctReturnsASufficientlyCloseColor(t *testing.T) {
	for hue := 15; hue < 360; hue += 30 {
		for chroma := 0; chroma <= 100; chroma += 10 {
			for tone := 20; tone <= 80; tone += 10 {
				desc := fmt.Sprintf("H%d C%d T%d", hue, chroma, tone)
				c := HctFrom(float64(hue), float64(chroma), float64(tone))
				if chroma > 0 {
					closeTo(t, "hue of "+desc, c.Hue(), float64(hue), 4.0)
				}
				if c.Chroma() < 0 || c.Chroma() > float64(chroma)+2.5 {
					t.Errorf("chroma of %s = %v, want close or less", desc, c.Chroma())
				}
				if c.Chroma() < float64(chroma)-2.5 && !colorIsOnBoundary(c.ToInt()) {
					t.Errorf("%s: a request for a non-sRGB color returned %s, which is not on the boundary of the sRGB cube", desc, HexFromArgb(c.ToInt(), true))
				}
				closeTo(t, "tone of "+desc, c.Tone(), float64(tone), 0.5)
			}
		}
	}
}

func TestCam16ToXyz(t *testing.T) {
	xyz := Cam16FromInt(red).XyzInViewingConditions(ViewingConditionsSRGB)
	closeTo(t, "x", xyz[0], 41.23, 0.01)
	closeTo(t, "y", xyz[1], 21.26, 0.01)
	closeTo(t, "z", xyz[2], 1.93, 0.01)
}

func TestColorRelativity(t *testing.T) {
	tests := []struct {
		color            string
		argb             uint32
		inBlack, inWhite uint32
	}{
		{"red", red, 0xff9F5C51, 0xffFF5D48},
		{"green", green, 0xffACD69D, 0xff8EFF77},
		{"blue", blue, 0xff343654, 0xff3F49FF},
		{"white", white, 0xffFFFFFF, 0xffFFFFFF},
		{"midgray", midgray, 0xff605F5F, 0xff8E8E8E},
		{"black", black, 0xff000000, 0xff000000},
	}
	for _, tt := range tests {
		t.Run(tt.color, func(t *testing.T) {
			hct := HctFromInt(tt.argb)
			requireColor(t, "in black", hct.InViewingConditions(NewViewingConditions(ViewingConditionsOpts{BackgroundLstar: some(0.0)})).ToInt(), tt.inBlack)
			requireColor(t, "in white", hct.InViewingConditions(NewViewingConditions(ViewingConditionsOpts{BackgroundLstar: some(100.0)})).ToInt(), tt.inWhite)
		})
	}
}

func TestHctSettersKeepTheOthersAsFarAsTheGamutAllows(t *testing.T) {
	c := HctFrom(270, 40, 50)
	c.SetTone(80)
	closeTo(t, "tone", c.Tone(), 80, 0.5)
	closeTo(t, "hue", c.Hue(), 270, 4)
	c.SetChroma(10)
	closeTo(t, "chroma", c.Chroma(), 10, 2.5)
	c.SetHue(30)
	closeTo(t, "hue", c.Hue(), 30, 4)
	if c.String() == "" {
		t.Error("no string")
	}
	if got, want := HctFrom(27.4, 113.4, 53.2).String(), "H27 C113 T53"; got != want {
		t.Errorf("String = %q, want %q", got, want)
	}
}

// Ported from hct_round_trip_test.dart. The Dart test walks all 16.7 million
// colors and takes minutes; by default every 997th color is tried, and
// GLUTTER_MCU_FULL_ROUNDTRIP=1 walks them all.
func TestHctPreservesTheOriginalColor(t *testing.T) {
	step := uint64(997)
	if os.Getenv("GLUTTER_MCU_FULL_ROUNDTRIP") == "1" {
		step = 1
	}
	for n := uint64(0xFF000000); n <= 0xFFFFFFFF; n += step {
		argb := uint32(n)
		hct := HctFromInt(argb)
		got := HctFrom(hct.Hue(), hct.Chroma(), hct.Tone()).ToInt()
		if got != argb {
			t.Fatalf("round trip of %s = %s", HexFromArgb(argb, true), HexFromArgb(got, true))
		}
	}
}

func TestViewingConditionsSurroundMustBeInRange(t *testing.T) {
	skipUnlessDebug(t)
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("no panic")
		}
		appErr, ok := asError(r)
		if !ok || !appErr.HasRoute("NewViewingConditions.ViewingConditionsSurroundInvalid") {
			t.Errorf("panic = %v", r)
		}
	}()
	NewViewingConditions(ViewingConditionsOpts{Surround: some(3.0)})
}
