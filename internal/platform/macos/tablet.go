package macos

import "math"

// tabletDetail turns what a tablet event says into the detail of a pointer: the pressure from 0 to 1
// (pressureMax is 1), and the lean of the pen. AppKit gives the tilt as x and y from -1 to 1, which are
// angles from -90 to 90 degrees along each axis; Flutter's tilt is the angle from the normal of the
// surface (0 is upright, pi/2 is flat) and its orientation the direction of the lean in radians from 0 to
// 2 pi (SPEC §29.6).
//
// Flutter: no counterpart (Flutter's macOS embedder does not report tablets; this follows
// PointerData.tilt and PointerData.orientation)
func tabletDetail(pressure, tiltX, tiltY float64) pointerDetail {
	d := pointerDetail{pressure: math.Min(1, math.Max(0, pressure)), pressureMax: 1}
	ax, ay := clamp1(tiltX)*math.Pi/2, clamp1(tiltY)*math.Pi/2
	tx, ty := math.Tan(ax), math.Tan(ay)
	d.tilt = math.Min(math.Pi/2, math.Atan(math.Hypot(tx, ty)))
	if math.IsInf(tx, 0) || math.IsInf(ty, 0) || math.Abs(clamp1(tiltX)) == 1 || math.Abs(clamp1(tiltY)) == 1 {
		d.tilt = math.Pi / 2
	}
	if tx != 0 || ty != 0 {
		d.orientation = math.Atan2(ty, tx)
		if d.orientation < 0 {
			d.orientation += 2 * math.Pi
		}
	}
	return d
}

func clamp1(v float64) float64 { return math.Min(1, math.Max(-1, v)) }

// pointingDevice is NSPointingDeviceType: what the tablet says is in proximity.
type pointingDevice int

// The values of NSPointingDeviceType.
const (
	pointingUnknown pointingDevice = 0
	pointingPen     pointingDevice = 1
	pointingCursor  pointingDevice = 2
	pointingEraser  pointingDevice = 3
)
