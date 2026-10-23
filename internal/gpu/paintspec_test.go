package gpu

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

func twoStops() []ui.Color {
	return []ui.Color{ui.NewColor(0xFFFF0000), ui.NewColor(0xFF0000FF)}
}

func TestRampStops(t *testing.T) {
	tests := []struct {
		name   string
		stops  []float64
		colors int
		want   []float64
	}{
		{"evenly spaced", nil, 3, []float64{0, 0.5, 1}},
		{"stops that do not reach the ends", []float64{0.2, 0.5, 0.6}, 3, []float64{0, 0.2, 0.5, 0.6, 1}},
		{"stops that go back are held", []float64{0.5, 0.2, 0.9}, 3, []float64{0, 0.5, 0.5, 0.9, 1}},
		{"one color", nil, 1, []float64{0, 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cols := make([]ui.Color, tt.colors)
			for i := range cols {
				cols[i] = ui.NewColor(0xFF000000 | uint32(i*40))
			}
			r := newRamp(cols, tt.stops)
			if len(r.pos) != len(tt.want) {
				t.Fatalf("positions %v, want %v", r.pos, tt.want)
			}
			for i, p := range tt.want {
				if r.pos[i] != p {
					t.Errorf("positions %v, want %v", r.pos, tt.want)
					break
				}
			}
		})
	}
}

func TestRampPremultipliedAt(t *testing.T) {
	r := newRamp([]ui.Color{ui.NewColor(0xFFFF0000), ui.NewColor(0x00FF0000)}, nil)
	tests := []struct {
		t    float64
		want [4]float32
	}{
		{-1, [4]float32{1, 0, 0, 1}},
		{0, [4]float32{1, 0, 0, 1}},
		{0.5, [4]float32{0.5, 0, 0, 0.5}}, // interpolated unpremultiplied, then premultiplied
		{1, [4]float32{0, 0, 0, 0}},
		{2, [4]float32{0, 0, 0, 0}},
	}
	for _, tt := range tests {
		if got := r.premultipliedAt(tt.t); got != tt.want {
			t.Errorf("premultipliedAt(%v) = %v, want %v", tt.t, got, tt.want)
		}
	}
}

func TestSetGradient(t *testing.T) {
	cols := twoStops()
	many := make([]ui.Color, MaxStops+1)
	manyStops := make([]float64, len(many))
	for i := range many {
		many[i] = cols[0]
		manyStops[i] = float64(i) / float64(len(many)-1)
	}
	tests := []struct {
		name   string
		g      *ui.Gradient
		result PaintResult
		feat   string
		check  func(t *testing.T, sp *PaintSpec)
	}{
		{"linear", ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 100}, cols), PaintDraw, "", func(t *testing.T, sp *PaintSpec) {
			if sp.Shader != ShaderLinear || sp.P0 != [4]float32{0, 0, 100, 0} || sp.P1[0] != 1.0/10000 || sp.RampN != 2 || !sp.Dither {
				t.Errorf("spec %+v", sp)
			}
		}},
		{"radial", ui.NewGradientRadial(ui.Offset{Dx: 5, Dy: 6}, 7, cols), PaintDraw, "", func(t *testing.T, sp *PaintSpec) {
			if sp.Shader != ShaderRadial || sp.P0 != [4]float32{5, 6, 7, 0} {
				t.Errorf("spec %+v", sp)
			}
		}},
		{"sweep", ui.NewGradientSweep(ui.Offset{Dx: 3, Dy: 4}, cols, ui.GradientSweepOpts{StartAngle: 1, EndAngle: 3, TileMode: ui.TileModeMirror}), PaintDraw, "", func(t *testing.T, sp *PaintSpec) {
			if sp.Shader != ShaderSweep || sp.P0 != [4]float32{3, 4, 1, 2} || sp.Tile[0] != 2 {
				t.Errorf("spec %+v", sp)
			}
		}},
		{"degenerate line is the last color", ui.NewGradientLinear(ui.Offset{Dx: 5}, ui.Offset{Dx: 5}, cols), PaintDraw, "", func(t *testing.T, sp *PaintSpec) {
			if sp.Shader != ShaderConst || sp.ConstCol != [4]float32{0, 0, 1, 1} || sp.RampN != 0 {
				t.Errorf("spec %+v", sp)
			}
		}},
		{"too many colors", ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 5}, many, ui.GradientLinearOpts{ColorStops: manyStops}), PaintUnsupported, "gradient with more than 32 colors", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sp PaintSpec
			res, feat := sp.SetGradient(tt.g, Affine2D{A: 1, D: 1})
			if res != tt.result || feat != tt.feat {
				t.Fatalf("SetGradient = %v, %q; want %v, %q", res, feat, tt.result, tt.feat)
			}
			if tt.check != nil {
				tt.check(t, &sp)
			}
		})
	}
	// The shader's own matrix and the transform to the surface are both inverted into XF.
	var sp PaintSpec
	g := ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 10}, cols, ui.GradientLinearOpts{Matrix4: ui.NewMatrix4Diagonal3Values(2, 2, 1)})
	sp.SetGradient(g, Affine2D{A: 1, D: 1, E: 10, F: 20})
	if sp.XF != [6]float32{0.5, 0, 0, 0.5, -5, -10} {
		t.Errorf("xf = %v", sp.XF)
	}
}

func TestSetImageShader(t *testing.T) {
	img := ui.NewImageFromRGBA(make([]byte, 16*8*4), 16, 8)
	shader := func(q ui.FilterQuality, m *ui.Matrix4) *ui.ImageShader {
		return ui.NewImageShader(img, ui.TileModeRepeated, ui.TileModeMirror, m, ui.ImageShaderOpts{FilterQuality: q})
	}
	found := func(*ui.Image) (any, int, error) { return "tex", 4, nil }
	var sp PaintSpec
	res, err := sp.SetImageShader(shader(ui.FilterQualityHigh, ui.NewMatrix4Diagonal3Values(0.25, 0.25, 1)), Affine2D{A: 1, D: 1}, found)
	// Four image pixels per surface pixel: the level two mip, with bicubic filtering.
	if err != nil || res != PaintDraw || sp.Shader != ShaderImage || sp.Tile != [2]int32{1, 2} || sp.Image != "tex" ||
		sp.ImgInfo != [4]float32{3, 2, 0, 0} || sp.XF != [6]float32{4, 0, 0, 4, 0, 0} {
		t.Errorf("spec %+v, %v, %v", sp, res, err)
	}
	// The level never goes past the last mip.
	sp = PaintSpec{}
	if _, err := sp.SetImageShader(shader(ui.FilterQualityMedium, ui.NewMatrix4Diagonal3Values(0.01, 0.01, 1)), Affine2D{A: 1, D: 1}, found); err != nil || sp.ImgInfo != [4]float32{2, 3, 0, 0} {
		t.Errorf("img info %v, %v", sp.ImgInfo, err)
	}
	// Without a mip filter the level is zero and none is quality 0.
	sp = PaintSpec{}
	if _, err := sp.SetImageShader(shader(ui.FilterQualityNone, ui.NewMatrix4Diagonal3Values(0.25, 0.25, 1)), Affine2D{A: 1, D: 1}, found); err != nil || sp.ImgInfo != [4]float32{0, 0, 0, 0} {
		t.Errorf("img info %v, %v", sp.ImgInfo, err)
	}
	// A matrix that cannot be undone is a degenerate shader.
	sp = PaintSpec{}
	if res, err := sp.SetImageShader(shader(ui.FilterQualityLow, ui.NewMatrix4Diagonal3Values(0, 1, 1)), Affine2D{A: 1, D: 1}, found); err != nil || res != PaintDraw || sp.Shader != ShaderConst {
		t.Errorf("spec %+v, %v, %v", sp, res, err)
	}
	// An error from the texture lookup comes back as it is.
	failing := func(*ui.Image) (any, int, error) { return nil, 0, errors.New(tagTestNoTexture, "no texture") }
	sp = PaintSpec{}
	res, err = sp.SetImageShader(shader(ui.FilterQualityLow, ui.NewMatrix4Identity()), Affine2D{A: 1, D: 1}, failing)
	if appErr, ok := errors.FromError(err); res != PaintUnsupported || !ok || appErr.Tag != tagTestNoTexture {
		t.Errorf("got %v, %v", res, err)
	}
}

func TestSetColorFilter(t *testing.T) {
	var m [20]float64
	for i := range m {
		m[i] = float64(i)
	}
	tests := []struct {
		name   string
		filter *ui.ColorFilter
		invert bool
		check  func(t *testing.T, sp *PaintSpec)
	}{
		{"mode", ui.NewColorFilterMode(ui.NewColor(0x80FF0000), ui.BlendModeMultiply), false, func(t *testing.T, sp *PaintSpec) {
			a := float32(0x80) / 255
			if sp.Filter != FilterMode || sp.FilterBlend != 24 || sp.FilterCol != [4]float32{a, 0, 0, a} {
				t.Errorf("spec %+v", sp)
			}
		}},
		{"matrix by columns", ui.NewColorFilterMatrix(m), false, func(t *testing.T, sp *PaintSpec) {
			// Column 1 holds the weights of green in the four outputs: m[1], m[6], m[11], m[16].
			if sp.Filter != FilterMatrix || sp.FilterMat[4] != 1 || sp.FilterMat[5] != 6 || sp.FilterMat[6] != 11 || sp.FilterMat[7] != 16 || sp.FilterMat[16] != 4 {
				t.Errorf("matrix %v", sp.FilterMat)
			}
		}},
		{"to sRGB", ui.NewColorFilterLinearToSrgbGamma(), false, func(t *testing.T, sp *PaintSpec) {
			if sp.Filter != FilterToSrgb {
				t.Errorf("filter %d", sp.Filter)
			}
		}},
		{"from sRGB", ui.NewColorFilterSrgbToLinearGamma(), false, func(t *testing.T, sp *PaintSpec) {
			if sp.Filter != FilterFromSrgb {
				t.Errorf("filter %d", sp.Filter)
			}
		}},
		{"inverted colors only", nil, true, func(t *testing.T, sp *PaintSpec) {
			if !sp.Invert || sp.Filter != FilterNone {
				t.Errorf("spec %+v", sp)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sp PaintSpec
			sp.SetColorFilter(tt.filter, tt.invert)
			tt.check(t, &sp)
		})
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagTestNoTexture = "TestNoTexture"
)
