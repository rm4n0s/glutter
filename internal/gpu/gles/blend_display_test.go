//go:build glutter_display

package gles

import (
	"image"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/ui"
)

// The paint shaders, color filters, blend modes and layers of step 3.4, compared
// with the software renderer.

func TestDisplayGradientsMatchSoftware(t *testing.T) {
	gpuMatchesSoft(t, "gradients", func(t *testing.T, got, want *image.RGBA, r *Renderer) {
		st := r.LastFrame()
		t.Logf("frame: %+v", st)
		if st.Skipped != 0 {
			t.Errorf("the GPU skipped %d commands: %v", st.Skipped, st.Unsupported)
		}
		// Gradients are dithered the same way on both, so only the rounding of the
		// interpolation and of the edges differs.
		tolerance{over8Permille: 10, over32Permille: 3}.check(t, "gradients", diffRect(got, want, got.Bounds()))
	})
}

func TestDisplayBlendModesMatchSoftware(t *testing.T) {
	gpuMatchesSoft(t, "blend-modes", func(t *testing.T, got, want *image.RGBA, r *Renderer) {
		st := r.LastFrame()
		t.Logf("frame: %+v", st)
		if st.Skipped != 0 {
			t.Errorf("the GPU skipped %d commands: %v", st.Skipped, st.Unsupported)
		}
		for i, mode := range blendModeNames {
			cell := image.Rect(i%6*106, i/6*96, i%6*106+106, i/6*96+96)
			tolerance{over8Permille: 15, over32Permille: 5}.check(t, mode, diffRect(got, want, cell))
		}
	})
}

// blendModeNames are the 29 modes in the order of BlendMode, as the cells of the
// blend-modes scene are.
var blendModeNames = []string{
	"clear", "src", "dst", "srcOver", "dstOver", "srcIn", "dstIn", "srcOut", "dstOut", "srcATop",
	"dstATop", "xor", "plus", "modulate", "screen", "overlay", "darken", "lighten", "colorDodge", "colorBurn",
	"hardLight", "softLight", "difference", "exclusion", "multiply", "hue", "saturation", "color", "luminosity",
}

// pictureOf records a picture of drawing.
func pictureOf(fn func(c *ui.Canvas)) *ui.Picture {
	rec := ui.NewPictureRecorder()
	fn(ui.NewCanvas(rec))
	return rec.EndRecording()
}

// stripesAndDisc is something with an edge and a gradient to put layers over.
func stripesAndDisc(c *ui.Canvas) {
	for i := 0; i < 8; i++ {
		c.DrawRect(ui.NewRectFromLTWH(0, float64(i)*16, 128, 8), fillP(0xFF3050C0))
		c.DrawRect(ui.NewRectFromLTWH(0, float64(i)*16+8, 128, 8), fillP(0xFFF0A020))
	}
	g := ui.NewPaint()
	g.SetShader(ui.NewGradientRadial(ui.Offset{Dx: 64, Dy: 64}, 40, []ui.Color{ui.NewColor(0xFFFF0000), ui.NewColor(0x4000FF00)}))
	c.DrawCircle(ui.Offset{Dx: 64, Dy: 64}, 40, g)
}

func TestDisplayPaintsAndLayersMatchSoftware(t *testing.T) {
	tests := gputest.EffectScenes()
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			scene := tt.Build()
			want, err := soft.Render(scene, gputest.EffectSize, gputest.EffectSize)
			if err != nil {
				t.Fatal(err)
			}
			onDisplay(t, func(h gpu.SurfaceHandle, _, _ int) error {
				r := NewRenderer()
				defer func() { _ = r.Close() }()
				if _, err := r.Attach(h); err != nil {
					return err
				}
				got, err := renderGPU(r, scene, gputest.EffectSize, gputest.EffectSize)
				if err != nil {
					return err
				}
				slug := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(tt.Name, " ", "-"), ",", ""), ":", "")
				dumpImage(t, slug+"-gpu", got)
				dumpImage(t, slug+"-soft", want)
				tolerance{tt.Tolerance.Over8Permille, tt.Tolerance.Over32Permille, tt.Tolerance.Over96}.check(t, tt.Name, diffRect(got, want, got.Bounds()))
				return nil
			})
		})
	}
}

func scale4(s float32) *ui.Matrix4 { return ui.NewMatrix4Diagonal3Values(float32(s), float32(s), 1) }
