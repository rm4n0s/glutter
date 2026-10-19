package painting

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func near(a, b, d float64) bool { return math.Abs(a-b) <= d }

func hsvNear(a, b HSVColor, d float64) bool {
	return near(a.Alpha, b.Alpha, d) && near(a.Hue, b.Hue, d*360) && near(a.Saturation, b.Saturation, d) && near(a.Value, b.Value, d)
}

func hslNear(a, b HSLColor, d float64) bool {
	return near(a.Alpha, b.Alpha, d) && near(a.Hue, b.Hue, d*360) && near(a.Saturation, b.Saturation, d) && near(a.Lightness, b.Lightness, d)
}

func equalColors(t *testing.T, got []ui.Color, want []uint32) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d colors, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != ui.NewColor(want[i]) {
			t.Errorf("color %d = %v, want %v", i, got[i], ui.NewColor(want[i]))
		}
	}
}

func TestHSVColorControl(t *testing.T) {
	color := NewHSVColorFromAHSV(0.7, 28, 0.3, 0.6)
	tests := []struct {
		name      string
		got, want HSVColor
	}{
		{"withAlpha", color.WithAlpha(0.8), NewHSVColorFromAHSV(0.8, 28, 0.3, 0.6)},
		{"withHue", color.WithHue(123), NewHSVColorFromAHSV(0.7, 123, 0.3, 0.6)},
		{"withSaturation", color.WithSaturation(0.9), NewHSVColorFromAHSV(0.7, 28, 0.9, 0.6)},
		{"withValue", color.WithValue(0.1), NewHSVColorFromAHSV(0.7, 28, 0.3, 0.1)},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, tt.got, tt.want)
		}
	}
	if got := color.ToColor(); got != ui.NewColor(0xb399816b) {
		t.Errorf("ToColor = %v", got)
	}
	other := NewHSVColorFromAHSV(0.3, 128, 0.7, 0.2)
	r := HSVColorLerp(&color, &other, 0.25)
	if !near(r.Alpha, 0.6, 1e-9) || !near(r.Hue, 53, 1e-9) || !near(r.Saturation, 0.4, 1e-4) || !near(r.Value, 0.5, 1e-9) {
		t.Errorf("lerp = %v", r)
	}
	if got := color.String(); got != "HSVColor(0.7, 28.0, 0.3, 0.6)" {
		t.Errorf("String = %q", got)
	}
}

func TestHSVColorSweeps(t *testing.T) {
	var out []ui.Color
	for hue := 0.0; hue <= 360.0; hue += 36.0 {
		hsv := NewHSVColorFromAHSV(1, hue, 1, 1)
		c := hsv.ToColor()
		out = append(out, c)
		if hue != 360.0 && !hsvNear(NewHSVColorFromColor(c), hsv, 0.01) {
			t.Errorf("hue %v does not round-trip: %v", hue, NewHSVColorFromColor(c))
		}
	}
	equalColors(t, out, []uint32{0xffff0000, 0xffff9900, 0xffccff00, 0xff33ff00, 0xff00ff66, 0xff00ffff, 0xff0066ff, 0xff3300ff, 0xffcc00ff, 0xffff0099, 0xffff0000})

	out = nil
	for s := 0.0; s < 1.0; s += 0.1 {
		hsv := NewHSVColorFromAHSV(1, 0, s, 1)
		c := hsv.ToColor()
		out = append(out, c)
		if !hsvNear(NewHSVColorFromColor(c), hsv, 0.01) {
			t.Errorf("saturation %v does not round-trip", s)
		}
	}
	equalColors(t, out, []uint32{0xffffffff, 0xffffe6e6, 0xffffcccc, 0xffffb3b3, 0xffff9999, 0xffff8080, 0xffff6666, 0xffff4d4d, 0xffff3333, 0xffff1a1a, 0xffff0000})

	out = nil
	for v := 0.0; v < 1.0; v += 0.1 {
		out = append(out, NewHSVColorFromAHSV(1, 0, 1, v).ToColor())
	}
	equalColors(t, out, []uint32{0xff000000, 0xff1a0000, 0xff330000, 0xff4d0000, 0xff660000, 0xff800000, 0xff990000, 0xffb30000, 0xffcc0000, 0xffe50000, 0xffff0000})
}

func TestHSVColorLerps(t *testing.T) {
	var out []ui.Color
	start, end := NewHSVColorFromAHSV(1, 0, 1, 1), NewHSVColorFromAHSV(1, 360, 1, 1)
	for tm := -0.5; tm < 1.5; tm += 0.1 {
		out = append(out, HSVColorLerp(&start, &end, tm).ToColor())
	}
	equalColors(t, out[:12], []uint32{0xff00ffff, 0xff0066ff, 0xff3300ff, 0xffcc00ff, 0xffff0099, 0xffff0000, 0xffff9900, 0xffccff00, 0xff33ff00, 0xff00ff66, 0xff00ffff, 0xff0066ff})

	out = nil
	s0, s1 := NewHSVColorFromAHSV(1, 0, 0, 1), NewHSVColorFromAHSV(1, 0, 1, 1)
	for tm := -0.1; tm < 1.1; tm += 0.1 {
		out = append(out, HSVColorLerp(&s0, &s1, tm).ToColor())
	}
	equalColors(t, out, []uint32{0xffffffff, 0xffffffff, 0xffffe6e6, 0xffffcccc, 0xffffb3b3, 0xffff9999, 0xffff8080, 0xffff6666, 0xffff4d4d, 0xffff3333, 0xffff1a1a, 0xffff0000, 0xffff0000})

	out = nil
	v0, v1 := NewHSVColorFromAHSV(1, 0, 1, 0), NewHSVColorFromAHSV(1, 0, 1, 1)
	for tm := -0.1; tm < 1.1; tm += 0.1 {
		out = append(out, HSVColorLerp(&v0, &v1, tm).ToColor())
	}
	equalColors(t, out, []uint32{0xff000000, 0xff000000, 0xff1a0000, 0xff330000, 0xff4d0000, 0xff660000, 0xff800000, 0xff990000, 0xffb30000, 0xffcc0000, 0xffe50000, 0xffff0000, 0xffff0000})

	if HSVColorLerp(nil, nil, 0) != nil || HSVColorLerp(&start, &start, 0.5) != &start {
		t.Error("lerp of nils or of identical colors is wrong")
	}
	if got := HSVColorLerp(nil, &end, 0.5); got.Alpha != 0.5 {
		t.Errorf("lerp from nil alpha = %v", got.Alpha)
	}
}

func TestHSLColorControl(t *testing.T) {
	color := NewHSLColorFromAHSL(0.7, 28, 0.3, 0.6)
	tests := []struct {
		name      string
		got, want HSLColor
	}{
		{"withAlpha", color.WithAlpha(0.8), NewHSLColorFromAHSL(0.8, 28, 0.3, 0.6)},
		{"withHue", color.WithHue(123), NewHSLColorFromAHSL(0.7, 123, 0.3, 0.6)},
		{"withSaturation", color.WithSaturation(0.9), NewHSLColorFromAHSL(0.7, 28, 0.9, 0.6)},
		{"withLightness", color.WithLightness(0.1), NewHSLColorFromAHSL(0.7, 28, 0.3, 0.1)},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, tt.got, tt.want)
		}
	}
	if got := color.ToColor(); got != ui.NewColor(0xb3b8977a) {
		t.Errorf("ToColor = %v", got)
	}
	other := NewHSLColorFromAHSL(0.3, 128, 0.7, 0.2)
	r := HSLColorLerp(&color, &other, 0.25)
	if !near(r.Alpha, 0.6, 1e-9) || !near(r.Hue, 53, 1e-9) || !near(r.Saturation, 0.4, 1e-4) || !near(r.Lightness, 0.5, 1e-9) {
		t.Errorf("lerp = %v", r)
	}
}

func TestHSLColorSweeps(t *testing.T) {
	var out []ui.Color
	for hue := 0.0; hue <= 360.0; hue += 36.0 {
		hsl := NewHSLColorFromAHSL(1, hue, 0.5, 0.5)
		c := hsl.ToColor()
		out = append(out, c)
		if hue != 360.0 && !hslNear(NewHSLColorFromColor(c), hsl, 0.01) {
			t.Errorf("hue %v does not round-trip: %v", hue, NewHSLColorFromColor(c))
		}
	}
	equalColors(t, out, []uint32{0xffbf4040, 0xffbf8c40, 0xffa6bf40, 0xff59bf40, 0xff40bf73, 0xff40bfbf, 0xff4073bf, 0xff5940bf, 0xffa640bf, 0xffbf408c, 0xffbf4040})

	out = nil
	for s := 0.0; s < 1.0; s += 0.1 {
		hsl := NewHSLColorFromAHSL(1, 0, s, 0.5)
		c := hsl.ToColor()
		out = append(out, c)
		if !hslNear(NewHSLColorFromColor(c), hsl, 0.01) {
			t.Errorf("saturation %v does not round-trip", s)
		}
	}
	equalColors(t, out, []uint32{0xff808080, 0xff8c7373, 0xff996666, 0xffa65959, 0xffb34d4d, 0xffbf4040, 0xffcc3333, 0xffd92626, 0xffe51a1a, 0xfff20d0d, 0xffff0000})

	out = nil
	for l := 0.0; l < 1.0; l += 0.1 {
		out = append(out, NewHSLColorFromAHSL(1, 0, 0.5, l).ToColor())
	}
	equalColors(t, out, []uint32{0xff000000, 0xff260d0d, 0xff4d1a1a, 0xff732626, 0xff993333, 0xffbf4040, 0xffcc6666, 0xffd98c8c, 0xffe6b3b3, 0xfff2d9d9, 0xffffffff})
}

func TestHSLColorFromColor(t *testing.T) {
	tests := []struct {
		name                       string
		color                      ui.Color
		hue, saturation, lightness float64
		hueDist, lightDist         float64
	}{
		{"pink", ui.NewColorFromARGB(255, 255, 51, 152), 330, 1, 0.6, 0.3, 0.01},
		{"white", ui.NewColor(0xffffffff), 0, 0, 1, 0, 0},
		{"black", ui.NewColor(0xff000000), 0, 0, 0, 0, 0},
		{"gray", ui.NewColor(0xff808080), 0, 0, 0.5, 0, 0.01},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHSLColorFromColor(tt.color)
			if h.Alpha != 1 || !near(h.Hue, tt.hue, tt.hueDist) || h.Saturation != tt.saturation || !near(h.Lightness, tt.lightness, tt.lightDist) {
				t.Errorf("got %v", h)
			}
		})
	}
	if HSLColorLerp(nil, nil, 0) != nil {
		t.Error("lerp(nil, nil) should be nil")
	}
}

func TestColorSwatch(t *testing.T) {
	keys := []int{50, 100}
	a := NewColorSwatch(ui.NewColor(0xFF000000), keys, map[int]ui.Color{50: ui.NewColor(0xFF000000), 100: ui.NewColor(0xFF202020)})
	b := NewColorSwatch(ui.NewColor(0xFFFFFFFF), keys, map[int]ui.Color{50: ui.NewColor(0xFFFFFFFF), 100: ui.NewColor(0xFFA0A0A0)})
	if c, ok := a.At(100); !ok || c != ui.NewColor(0xFF202020) {
		t.Errorf("At(100) = %v, %v", c, ok)
	}
	if _, ok := a.At(200); ok {
		t.Error("At(200) should be missing")
	}
	if got := a.Keys(); len(got) != 2 || got[0] != 50 || got[1] != 100 {
		t.Errorf("Keys = %v", got)
	}
	got := ColorSwatchLerp(a, b, 0.5)
	if got.Primary != a.Primary.Lerp(b.Primary, 0.5) {
		t.Errorf("Primary = %v", got.Primary)
	}
	if c, _ := got.At(100); c != ui.NewColor(0xFF202020).Lerp(ui.NewColor(0xFFA0A0A0), 0.5) {
		t.Errorf("At(100) = %v", c)
	}
	if ColorSwatchLerp(a, a, 0.5) != a {
		t.Error("lerp of identical swatches should return the same swatch")
	}
	fromNil := ColorSwatchLerp(nil, b, 0.5)
	if c, _ := fromNil.At(50); c.Alpha() != 128 {
		t.Errorf("lerp from nil alpha = %d", c.Alpha())
	}
	if !a.Equal(NewColorSwatch(a.Primary, keys, map[int]ui.Color{50: ui.NewColor(0xFF000000), 100: ui.NewColor(0xFF202020)})) || a.Equal(b) {
		t.Error("swatch equality is wrong")
	}
}
