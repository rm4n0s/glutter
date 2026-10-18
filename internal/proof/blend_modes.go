package proof

import (
	"github.com/rm4n0s/glutter/ui"
)

func init() {
	register(Scene{
		Name:        "blend-modes",
		Description: "all 29 blend modes in a 6x5 grid, in the order of BlendMode; each cell draws a gradient circle and blends a gradient square into it",
		Width:       636,
		Height:      480,
		Build:       func() *ui.Scene { return singlePicture(636, 480, drawBlendModes) },
	})
}

// blendModeOrder lists the 29 modes in the order of Flutter's BlendMode enum.
func blendModeOrder() []ui.BlendMode {
	var modes []ui.BlendMode
	for m := ui.BlendModeClear; m <= ui.BlendModeLuminosity; m++ {
		modes = append(modes, m)
	}
	return modes
}

func drawBlendModes(c *ui.Canvas) {
	const cw, ch = 106.0, 96.0
	for i, mode := range blendModeOrder() {
		r := ui.NewRectFromLTWH(float64(i%6)*cw, float64(i/6)*ch, cw, ch)
		// A checkerboard shows where each result is transparent.
		c.Save()
		c.ClipRect(r.Deflate(3))
		for y := 0; y < 12; y++ {
			for x := 0; x < 14; x++ {
				if (x+y)%2 == 0 {
					c.DrawRect(ui.NewRectFromLTWH(r.Left+float64(x)*8, r.Top+float64(y)*8, 8, 8), fillPaint(0xFFE4E4E4))
				}
			}
		}
		c.SaveLayer(nil, nil)
		// The destination: a circle fading from yellow to red, left to right.
		dst := ui.NewPaint()
		dst.SetShader(ui.NewGradientLinear(ui.Offset{Dx: r.Left + 12, Dy: 0}, ui.Offset{Dx: r.Left + 62, Dy: 0}, colorList(0xFFFFD600, 0xFFD50000)))
		c.DrawCircle(ui.Offset{Dx: r.Left + 38, Dy: r.Top + 42}, 28, dst)
		// The source: a square from blue to transparent blue, top to bottom.
		src := ui.NewPaint()
		src.SetShader(ui.NewGradientLinear(ui.Offset{Dx: 0, Dy: r.Top + 30}, ui.Offset{Dx: 0, Dy: r.Top + 82}, colorList(0xFF1565C0, 0x331565C0)))
		src.SetBlendMode(mode)
		c.DrawRect(ui.NewRectFromLTWH(r.Left+44, r.Top+30, 52, 52), src)
		c.Restore()
		c.Restore()
	}
}
