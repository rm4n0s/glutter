package ios

import (
	"time"

	"github.com/rm4n0s/glutter/ui"
)

// This file is the plain-Go part of the iPad pointer (SPEC §30.4): the hover of a mouse or a trackpad, and the scroll, the pinch and the
// rotation of a trackpad, which UIKit gives through gesture recognizers, converted to the pointer data of the framework as Flutter's
// iOS embedder converts them (the trackpad's gestures are pan-zoom sequences).

// The device numbers of the pointer, above those of the touches (touch.go gives the touches 1, 2, 3, ...): the converter keeps a state
// for each.
const (
	deviceMouse  = 1000
	deviceScroll = 1001
	devicePinch  = 1002
	deviceRotate = 1003
)

// gesturePhase is the state of a gesture recognizer (UIGestureRecognizerState): began, changed, and the two ends.
type gesturePhase int

// The phases of a gesture, with UIKit's numbers.
const (
	gestureBegan gesturePhase = iota + 1
	gestureChanged
	gestureEnded
	gestureCancelled
)

// hoverData is the data of the hover of the pointer over the view at (x, y) points: a hover while it is over, and a remove when it
// leaves. A hover that comes without an add gets one from the converter.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterViewController.mm hoverEvent:
func hoverData(view ui.ViewID, phase gesturePhase, x, y, scale float64, at time.Duration) []ui.PointerData {
	if scale <= 0 {
		scale = 1
	}
	d := ui.PointerData{
		ViewID: view, TimeStamp: at, Kind: ui.PointerDeviceKindMouse, Device: deviceMouse,
		PhysicalX: x * scale, PhysicalY: y * scale, Scale: 1,
	}
	switch phase {
	case gestureBegan, gestureChanged:
		d.Change = ui.PointerChangeHover
	case gestureEnded, gestureCancelled:
		d.Change = ui.PointerChangeRemove
	default:
		return nil
	}
	return []ui.PointerData{d}
}

// panZoom is one pan-zoom gesture of the trackpad: the session of a recognizer, which starts, updates and ends, with the pan, the scale
// and the rotation since its start.
type panZoom struct {
	device int
	active bool
}

// data is the data of the recognizer's state at the place of the pointer (x, y points), with the pan in points (dx, dy), the zoom
// (1 is none) and the rotation in radians since the gesture began. A changed state before a began one is dropped, as is a second began.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterViewController.mm discreteScrollEvent:, scrollEvent:, pinchEvent:, rotationEvent:
func (g *panZoom) data(view ui.ViewID, phase gesturePhase, x, y, dx, dy, zoom, rotation, scale float64, at time.Duration) []ui.PointerData {
	if scale <= 0 {
		scale = 1
	}
	if zoom == 0 {
		zoom = 1
	}
	d := ui.PointerData{
		ViewID: view, TimeStamp: at, Kind: ui.PointerDeviceKindTrackpad, Device: g.device,
		PhysicalX: x * scale, PhysicalY: y * scale, PanX: dx * scale, PanY: dy * scale, Scale: zoom, Rotation: rotation,
	}
	switch phase {
	case gestureBegan:
		if g.active {
			return nil
		}
		g.active = true
		d.Change = ui.PointerChangePanZoomStart
		d.PanX, d.PanY, d.Scale, d.Rotation = 0, 0, 1, 0
	case gestureChanged:
		if !g.active {
			return nil
		}
		d.Change = ui.PointerChangePanZoomUpdate
	case gestureEnded, gestureCancelled:
		if !g.active {
			return nil
		}
		g.active = false
		d.Change = ui.PointerChangePanZoomEnd
	default:
		return nil
	}
	return []ui.PointerData{d}
}
