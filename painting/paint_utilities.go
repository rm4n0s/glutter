package painting

import (
	"math"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// PaintZigZag draws a zig-zag line with the given number of zigs from start to
// end, zigging the width to either side. In debug builds zigs below 1 panic with
// the tag PaintZigZagInvalid.
//
// Flutter: painting/paint_utilities.dart paintZigZag
func PaintZigZag(canvas *ui.Canvas, paint *ui.Paint, start, end ui.Offset, zigs int, width float64) {
	if foundation.DebugMode && zigs <= 0 {
		panic(errors.New(tagPaintZigZagInvalid, "a zig-zag needs at least one zig", "zigs", zigs))
	}
	canvas.Save()
	canvas.Translate(start.Dx, start.Dy)
	end = end.Sub(start)
	canvas.Rotate(math.Atan2(end.Dy, end.Dx))
	length := end.Distance()
	spacing := length / (float64(zigs) * 2.0)
	path := ui.NewPath()
	path.MoveTo(0.0, 0.0)
	for index := 0; index < zigs; index++ {
		x := (float64(index)*2.0 + 1.0) * spacing
		y := width * (float64(index%2)*2.0 - 1.0)
		path.LineTo(x, y)
	}
	path.LineTo(length, 0.0)
	canvas.DrawPath(path, paint)
	canvas.Restore()
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagPaintZigZagInvalid = "PaintZigZagInvalid"
)
