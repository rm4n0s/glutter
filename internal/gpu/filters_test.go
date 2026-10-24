package gpu

import (
	"image"
	"math"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func blurFilter(sx, sy float64, tm ui.TileMode) ui.ImageFilter {
	return ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: sx, SigmaY: sy, TileMode: tm})
}

func TestBlurPasses(t *testing.T) {
	tests := []struct {
		name   string
		sx, sy float64
		tile   ui.TileMode
		want   []PassKind
		// sizes are the output sizes of the passes.
		sizes [][2]int
	}{
		{"none", 0, 0.001, ui.TileModeDecal, nil, nil},
		{"small", 2, 3, ui.TileModeClamp, []PassKind{PassBlur, PassBlur}, [][2]int{{100, 60}, {100, 60}}},
		{"x only", 3, 0, ui.TileModeClamp, []PassKind{PassBlur}, [][2]int{{100, 60}}},
		{"y only", 0, 3, ui.TileModeClamp, []PassKind{PassBlur}, [][2]int{{100, 60}}},
		{"large x is made on a narrower picture", 20, 2, ui.TileModeMirror,
			[]PassKind{PassDown, PassBlur, PassBlur, PassUp}, [][2]int{{25, 60}, {25, 60}, {25, 60}, {100, 60}}},
		{"large both", 40, 40, ui.TileModeDecal,
			[]PassKind{PassDown, PassBlur, PassBlur, PassUp}, [][2]int{{13, 8}, {13, 8}, {13, 8}, {100, 60}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BlurPasses(tt.sx, tt.sy, tt.tile, 100, 60)
			if len(got) != len(tt.want) {
				t.Fatalf("passes = %d, want %d", len(got), len(tt.want))
			}
			for i, p := range got {
				if p.Kind != tt.want[i] || [2]int{p.OutW, p.OutH} != tt.sizes[i] {
					t.Errorf("pass %d = kind %d size %dx%d, want kind %d size %v", i, p.Kind, p.OutW, p.OutH, tt.want[i], tt.sizes[i])
				}
				if p.Kind == PassBlur {
					// The kernel is 3 sigma wide on each side and sums to 1.
					var sum float64
					for i := -int(p.Radius); i <= int(p.Radius); i++ {
						sum += math.Exp(-float64(i*i)/(2*float64(p.Sigma)*float64(p.Sigma))) * float64(p.Norm)
					}
					if math.Abs(sum-1) > 1e-5 || p.Radius != int32(math.Ceil(3*float64(p.Sigma))) || p.Sigma > MaxLowSigma {
						t.Errorf("kernel of pass %d: sigma %v radius %d sums to %v", i, p.Sigma, p.Radius, sum)
					}
					if p.Tile != TileIndex(tt.tile) {
						t.Errorf("tile = %d", p.Tile)
					}
				}
			}
		})
	}
	// The second axis of a blur goes along y.
	if p := BlurPasses(2, 2, ui.TileModeClamp, 10, 10); p[0].Axis != 0 || p[1].Axis != 1 {
		t.Errorf("axes = %d, %d", p[0].Axis, p[1].Axis)
	}
}

func TestFilterPassesOfEveryFilter(t *testing.T) {
	ctm := Affine2D{A: 2, D: 2}
	rot := ui.NewMatrix4Identity()
	rot.RotateZ(0.5)
	singular := ui.NewMatrix4Diagonal3Values(0, 1, 1)
	tests := []struct {
		name   string
		filter ui.ImageFilter
		want   []PassKind
	}{
		{"blur", blurFilter(2, 3, ui.TileModeClamp), []PassKind{PassBlur, PassBlur}},
		{"dilate", ui.NewImageFilterDilate(ui.ImageFilterMorphologyOpts{RadiusX: 3, RadiusY: 0}), []PassKind{PassMorph}},
		{"erode", ui.NewImageFilterErode(ui.ImageFilterMorphologyOpts{RadiusX: 1, RadiusY: 2}), []PassKind{PassMorph, PassMorph}},
		{"matrix", ui.NewImageFilterMatrix(rot), []PassKind{PassMatrix}},
		{"compose", ui.NewImageFilterCompose(blurFilter(1, 1, ui.TileModeClamp), ui.NewImageFilterDilate(ui.ImageFilterMorphologyOpts{RadiusX: 1, RadiusY: 1})),
			[]PassKind{PassMorph, PassMorph, PassBlur, PassBlur}},
		{"color", ui.NewColorFilterSaturation(0.5), []PassKind{PassColor}},
		{"singular matrix", ui.NewImageFilterMatrix(singular), []PassKind{PassMatrix}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := FilterPasses(tt.filter, ctm, ui.TileModeDecal, 40, 30)
			if !ok || len(got) != len(tt.want) {
				t.Fatalf("passes = %d (ok %v), want %d", len(got), ok, len(tt.want))
			}
			for i, p := range got {
				if p.Kind != tt.want[i] {
					t.Errorf("pass %d kind = %d, want %d", i, p.Kind, tt.want[i])
				}
			}
		})
	}
	if _, ok := FilterPasses(nil, ctm, ui.TileModeDecal, 40, 30); ok {
		t.Error("a nil filter was accepted")
	}
	// The radii are in pixels: the transform's scale applies, and a blur without a
	// tile mode gets the mode it is given.
	got, _ := FilterPasses(blurFilter(2, 2, ui.TileModeUnspecified), ctm, ui.TileModeMirror, 40, 30)
	if got[0].Sigma != 4 || got[0].Tile != TileIndex(ui.TileModeMirror) {
		t.Errorf("blur pass = sigma %v tile %d", got[0].Sigma, got[0].Tile)
	}
	if got, _ := FilterPasses(ui.NewImageFilterDilate(ui.ImageFilterMorphologyOpts{RadiusX: 1.4, RadiusY: 0}), ctm, ui.TileModeDecal, 40, 30); got[0].Radius != 3 {
		t.Errorf("dilate radius = %d, want 1.4 * 2 rounded", got[0].Radius)
	}
	// A matrix filter that cannot be inverted draws nothing; a nearest one is nearest.
	if got, _ := FilterPasses(ui.NewImageFilterMatrix(singular), ctm, ui.TileModeDecal, 4, 4); got[0].Quality != 2 {
		t.Errorf("singular quality = %d", got[0].Quality)
	}
	if got, _ := FilterPasses(ui.NewImageFilterMatrix(rot, ui.ImageFilterMatrixOpts{FilterQuality: ui.FilterQualityNone}), ctm, ui.TileModeDecal, 4, 4); got[0].Quality != 1 {
		t.Errorf("nearest quality = %d", got[0].Quality)
	}
}

func TestBackdropMatrix(t *testing.T) {
	// A magnifier: scale 2 around the point (300, 300) of a lens at (200, 200) of 100 x 100.
	magnify := func(scale, cx, cy, fx, fy float64) ui.ImageFilter {
		m := ui.NewMatrix4Identity()
		m.Translate(float32(scale*(-fx-cx)+cx), float32(scale*(-fy-cy)+cy), 0)
		m.Scale(float32(scale), float32(scale), float32(scale))
		return ui.NewImageFilterMatrix(m, ui.ImageFilterMatrixOpts{FilterQuality: ui.FilterQualityHigh})
	}
	screen := image.Rect(0, 0, 800, 600)
	lens := image.Rect(200, 200, 300, 300)
	tests := []struct {
		name       string
		filter     ui.ImageFilter
		ctm        Affine2D
		bounds     image.Rectangle
		wantOK     bool
		wantSrc    image.Rectangle
		wantNoRead bool
	}{
		{"a magnifier reads what is under its focal point", magnify(2, 250, 250, 50, 50), Affine2D{A: 1, D: 1}, screen, true,
			// The lens maps back to (275, 275) to (325, 325), plus the pixel of the bilinear taps.
			image.Rect(274, 274, 326, 326), false},
		{"under a scale of the canvas", magnify(2, 125, 125, 25, 25), Scaling(2, 2), screen, true, image.Rect(274, 274, 326, 326), false},
		{"nothing under the surface it reads", magnify(2, 250, 250, 50, 400), Affine2D{A: 1, D: 1}, image.Rect(0, 0, 800, 560), true,
			image.Rectangle{}, true},
		{"a blur reads the pixels of the layer", blurFilter(2, 2, ui.TileModeMirror), Affine2D{A: 1, D: 1}, screen, false, lens, false},
		{"a singular matrix reads nothing special", ui.NewImageFilterMatrix(ui.NewMatrix4Zero()), Affine2D{A: 1, D: 1}, screen, false, lens, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, pass, ok := BackdropMatrix(tt.filter, tt.ctm, lens, tt.bounds)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				if src != lens {
					t.Errorf("src = %v, want the layer's rectangle", src)
				}
				return
			}
			if tt.wantNoRead {
				if pass.Quality != 2 || src.Dx() != 1 || src.Dy() != 1 {
					t.Errorf("src = %v, quality %d, want one pixel and no drawing", src, pass.Quality)
				}
				return
			}
			if src != tt.wantSrc {
				t.Errorf("src = %v, want %v", src, tt.wantSrc)
			}
			if pass.Kind != PassMatrix || pass.OutW != 100 || pass.OutH != 100 || pass.Quality != 0 {
				t.Fatalf("pass = %+v", pass)
			}
			// What the shader of the pass computes, over a copy of src, must be the pixel of the
			// surface that the filter reads, relative to the corner of src.
			inv, _ := tt.ctm.Invert()
			filterBack, _ := inv.Then(AffineOf(tt.filter.(*ui.MatrixImageFilter).Matrix())).Then(tt.ctm).Invert()
			for _, q := range [][2]int{{0, 0}, {99, 0}, {0, 99}, {50, 50}, {99, 99}} {
				devX, devY := float64(src.Min.X+q[0])+0.5, float64(src.Min.Y+q[1])+0.5
				gotX := float64(pass.Inv[0])*devX + float64(pass.Inv[2])*devY + float64(pass.Inv[4]) - float64(src.Min.X)
				gotY := float64(pass.Inv[1])*devX + float64(pass.Inv[3])*devY + float64(pass.Inv[5]) - float64(src.Min.Y)
				want := filterBack.Apply(float64(lens.Min.X+q[0])+0.5, float64(lens.Min.Y+q[1])+0.5)
				if math.Abs(gotX-(want.Dx-float64(src.Min.X))) > 1e-3 || math.Abs(gotY-(want.Dy-float64(src.Min.Y))) > 1e-3 {
					t.Errorf("output pixel %v reads (%.3f, %.3f) of the copy, want (%.3f, %.3f)", q, gotX, gotY, want.Dx-float64(src.Min.X), want.Dy-float64(src.Min.Y))
				}
			}
		})
	}
}
