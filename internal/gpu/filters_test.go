package gpu

import (
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
