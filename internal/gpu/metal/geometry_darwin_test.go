//go:build darwin && !ios

package metal

import (
	"image"

	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/ui"
)

func imageRect(x0, y0, x1, y1 int) image.Rectangle { return image.Rect(x0, y0, x1, y1) }
func imagePoint(x, y int) image.Point              { return image.Pt(x, y) }

// sceneOfRect is a scene with a blue rectangle on the white background, at 2x.
func sceneOfRect() *ui.Scene {
	return gputest.CanvasScene(func(c *ui.Canvas) {
		c.DrawRect(ui.NewRectFromLTWH(20, 20, 200, 120), gputest.FillPaint(0xFF3366CC))
	})
}
