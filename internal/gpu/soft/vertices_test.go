package soft

import (
	"math"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/ui"
)

func pts(v ...float64) []ui.Offset { return poly(v...) }

func TestVerticesCoverage(t *testing.T) {
	square := pts(2, 2, 10, 2, 10, 10, 2, 10)
	tests := []struct {
		name string
		v    *ui.Vertices
		want float64
	}{
		{"one triangle", ui.NewVertices(ui.VertexModeTriangles, pts(0, 0, 8, 0, 0, 8)), 32},
		{"triangle is the same either way round", ui.NewVertices(ui.VertexModeTriangles, pts(0, 0, 0, 8, 8, 0)), 32},
		{"two triangles of a square", ui.NewVertices(ui.VertexModeTriangles, square, ui.VerticesOpts{Indices: []int{0, 1, 2, 0, 2, 3}}), 64},
		{"strip", ui.NewVertices(ui.VertexModeTriangleStrip, pts(2, 2, 10, 2, 2, 10, 10, 10)), 64},
		{"fan", ui.NewVertices(ui.VertexModeTriangleFan, square), 64},
		{"extra points are ignored", ui.NewVertices(ui.VertexModeTriangles, pts(0, 0, 8, 0, 0, 8, 20, 20, 21, 20)), 32},
		{"fewer than three", ui.NewVertices(ui.VertexModeTriangles, pts(0, 0, 8, 0)), 0},
		{"degenerate", ui.NewVertices(ui.VertexModeTriangles, pts(0, 0, 4, 4, 8, 8)), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := renderOf(t, 16, 16, func(c *ui.Canvas) { c.DrawVertices(tt.v, ui.BlendModeSrcOver, paint(0xFF000000)) })
			// Pixels are covered or not; the count is the area up to the edge pixels.
			if got := coverageSum(img); math.Abs(got-tt.want) > tt.want*0.15+1 {
				t.Errorf("covered %v pixels, want about %v", got, tt.want)
			}
		})
	}
}

func TestVerticesSharedEdgeIsCoveredOnce(t *testing.T) {
	// Two half-opaque triangles share a diagonal. Every pixel of the square,
	// including those on the diagonal, must be painted exactly once.
	square := pts(0, 0, 8, 0, 8, 8, 0, 8)
	v := ui.NewVertices(ui.VertexModeTriangles, square, ui.VerticesOpts{Indices: []int{0, 1, 2, 0, 2, 3}})
	img := renderOf(t, 8, 8, func(c *ui.Canvas) { c.DrawVertices(v, ui.BlendModeSrcOver, paint(0x80000000)) })
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			if got := px(img, x, y)[3]; got != 128 {
				t.Fatalf("alpha at %d,%d = %d, want 128 (covered once)", x, y, got)
			}
		}
	}
}

func TestVerticesColors(t *testing.T) {
	colors := []ui.Color{ui.NewColor(0xFFFF0000), ui.NewColor(0xFF00FF00), ui.NewColor(0xFF0000FF)}
	v := ui.NewVertices(ui.VertexModeTriangles, pts(0, 0, 32, 0, 0, 32), ui.VerticesOpts{Colors: colors})
	img := renderOf(t, 32, 32, func(c *ui.Canvas) { c.DrawVertices(v, ui.BlendModeDst, paint(0xFFFFFFFF)) })
	// Near each corner the color is nearly that corner's.
	for _, tt := range []struct {
		x, y int
		dom  int // the channel that must dominate
	}{{1, 1, 0}, {28, 1, 1}, {1, 28, 2}} {
		p := px(img, tt.x, tt.y)
		for k := 0; k < 3; k++ {
			if k != tt.dom && p[tt.dom] <= p[k]+100 {
				t.Errorf("pixel %d,%d = %v, channel %d should dominate", tt.x, tt.y, p, tt.dom)
			}
		}
	}
	// At the middle of the hypotenuse the green and blue are equal.
	if p := px(img, 16, 15); absInt(int(p[1])-int(p[2])) > 20 || p[0] > 30 {
		t.Errorf("middle of the hypotenuse = %v", p)
	}
	// BlendModeSrc ignores the vertex colors and paints the paint's color.
	img = renderOf(t, 32, 32, func(c *ui.Canvas) { c.DrawVertices(v, ui.BlendModeSrc, paint(0xFF123456)) })
	if got := px(img, 3, 3); got != [4]uint8{0x12, 0x34, 0x56, 255} {
		t.Errorf("BlendModeSrc = %v", got)
	}
	// Without colors the paint's color is used, whatever the blend mode.
	plain := ui.NewVertices(ui.VertexModeTriangles, pts(0, 0, 8, 0, 0, 8))
	img = renderOf(t, 8, 8, func(c *ui.Canvas) { c.DrawVertices(plain, ui.BlendModeSrcOver, paint(0xFF123456)) })
	if got := px(img, 1, 1); got != [4]uint8{0x12, 0x34, 0x56, 255} {
		t.Errorf("paint color = %v", got)
	}
}

func absInt(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func TestVerticesTransformAndClip(t *testing.T) {
	v := ui.NewVertices(ui.VertexModeTriangles, pts(0, 0, 4, 0, 0, 4))
	img := renderOf(t, 16, 16, func(c *ui.Canvas) {
		c.Translate(6, 6)
		c.Scale(2, 2)
		c.ClipRect(ui.NewRectFromLTWH(0, 0, 3, 3)) // in the scaled space: 6..12
		c.DrawVertices(v, ui.BlendModeSrcOver, paint(0xFF000000))
	})
	if got := px(img, 7, 7)[3]; got != 255 {
		t.Errorf("inside = %d", got)
	}
	if got := px(img, 13, 6)[3]; got != 0 {
		t.Errorf("outside the clip = %d", got)
	}
	if got := px(img, 5, 5)[3]; got != 0 {
		t.Errorf("outside the shape = %d", got)
	}
}

func TestVerticesNotImplemented(t *testing.T) {
	colors := []ui.Color{ui.NewColor(0xFFFF0000), ui.NewColor(0xFF00FF00), ui.NewColor(0xFF0000FF)}
	v := ui.NewVertices(ui.VertexModeTriangles, pts(0, 0, 8, 0, 0, 8), ui.VerticesOpts{Colors: colors})
	for name, draw := range map[string]func(c *ui.Canvas){
		"colors with srcOver":  func(c *ui.Canvas) { c.DrawVertices(v, ui.BlendModeSrcOver, paint(0xFF000000)) },
		"colors with modulate": func(c *ui.Canvas) { c.DrawVertices(v, ui.BlendModeModulate, paint(0xFF000000)) },
		"a shader": func(c *ui.Canvas) {
			p := ui.NewPaint()
			p.SetShader(ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 1}, colors[:2]))
			c.DrawVertices(v, ui.BlendModeDst, p)
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Render(sceneOf(ui.Offset{}, draw), 8, 8)
			if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("SoftNotImplemented") {
				t.Errorf("expected SoftNotImplemented, got %v", err)
			}
		})
	}
}

func TestGoldenVertices(t *testing.T) {
	rgb := []ui.Color{ui.NewColor(0xFFFF0000), ui.NewColor(0xFF00FF00), ui.NewColor(0xFF0000FF)}
	tests := []struct {
		name string
		v    *ui.Vertices
		mode ui.BlendMode
	}{
		{"triangles", ui.NewVertices(ui.VertexModeTriangles, pts(6, 6, 58, 12, 20, 58), ui.VerticesOpts{Colors: rgb}), ui.BlendModeDst},
		{"strip", ui.NewVertices(ui.VertexModeTriangleStrip, pts(6, 50, 14, 10, 26, 54, 34, 8, 46, 56, 58, 12),
			ui.VerticesOpts{Colors: []ui.Color{rgb[0], rgb[1], rgb[2], rgb[0], rgb[1], rgb[2]}}), ui.BlendModeDst},
		{"fan", ui.NewVertices(ui.VertexModeTriangleFan, pts(32, 32, 32, 4, 58, 20, 58, 44, 32, 60, 6, 44, 6, 20),
			ui.VerticesOpts{Colors: []ui.Color{ui.NewColor(0xFFFFFFFF), rgb[0], rgb[1], rgb[2], rgb[0], rgb[1], rgb[2]}}), ui.BlendModeDst},
		{"indexed", ui.NewVertices(ui.VertexModeTriangles, pts(8, 8, 56, 8, 56, 56, 8, 56),
			ui.VerticesOpts{Colors: []ui.Color{rgb[0], rgb[1], rgb[2], ui.NewColor(0xFFFFFF00)}, Indices: []int{0, 1, 2, 0, 2, 3}}), ui.BlendModeDst},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := renderOf(t, 64, 64, func(c *ui.Canvas) { white(c); c.DrawVertices(tt.v, tt.mode, paint(0xFF000000)) })
			goldens.Check(t, "vertices-"+tt.name, img)
		})
	}
}
