package painting

import (
	"strings"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// The matrix helpers of Flutter's MatrixUtils live on ui.Matrix4, next to the
// type: AsTranslation (MatrixUtils.getAsTranslation), AsScale
// (getAsScale), Equals (matrixEquals), IsIdentity, TransformPoint,
// TransformRect, InverseTransformRect, NewMatrix4CylindricalProjection
// (createCylindricalProjectionTransform), NewMatrix4ForceToPoint (forceToPoint)
// and MatrixMultiplyInPlace (multiplyInPlace).

// DebugDescribeTransform describes a transform as four lines of four numbers;
// a nil transform is "null".
//
// Flutter: painting/matrix_utils.dart debugDescribeTransform
func DebugDescribeTransform(transform *ui.Matrix4) []string {
	if transform == nil {
		return []string{"null"}
	}
	rows := make([]string, 4)
	for r := 0; r < 4; r++ {
		cells := make([]string, 4)
		for c := 0; c < 4; c++ {
			cells[c] = foundation.DebugFormatDouble(foundation.Some(float64(transform.Entry(r, c))))
		}
		rows[r] = "[" + string(rune('0'+r)) + "] " + strings.Join(cells, ",")
	}
	return rows
}

// NewTransformProperty creates a diagnostics property for a transform; a nil
// transform is Dart's null. It is drawn in one line, as rows split by
// semicolons, where the parent does not break lines between properties, and
// in four lines otherwise.
//
// Flutter: painting/matrix_utils.dart TransformProperty
func NewTransformProperty(name string, value *ui.Matrix4, opts ...foundation.DiagnosticsPropertyOpts) *foundation.DiagnosticsProperty {
	o := firstOpts(opts)
	o.ValueToString = func(parent *foundation.TextTreeConfiguration) string {
		if parent != nil && !parent.LineBreakProperties {
			rows := make([]string, 4)
			for r := 0; r < 4; r++ {
				cells := make([]string, 4)
				for c := 0; c < 4; c++ {
					cells[c] = foundation.DebugFormatDouble(foundation.Some(float64(value.Entry(r, c))))
				}
				rows[r] = strings.Join(cells, ",")
			}
			return "[" + strings.Join(rows, "; ") + "]"
		}
		return strings.Join(DebugDescribeTransform(value), "\n")
	}
	return foundation.NewDiagnosticsProperty(name, value, o)
}
