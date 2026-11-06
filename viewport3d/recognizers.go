package viewport3d

import (
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/widgets"
)

// The recognizers of the navigation (SPEC §27.8). Each is a gestures.ScaleGestureRecognizer (which already follows one or
// more pointers, their focal point, spread and twist) made for one kind of input, and each is a member of the gesture
// arena like any recognizer of Flutter: a button above the view, a drag of a parent or a Scrollable around it wins or
// loses the arena by the same rules. They have a type of their own because a RawGestureDetector keeps its recognizers by
// type.

// The roles of the recognizers: the types that tell them apart.
type (
	// leftRole follows the left (primary) button of a mouse.
	leftRole struct{}
	// middleRole follows the middle (tertiary) button of a mouse.
	middleRole struct{}
	// rightRole follows the right (secondary) button of a mouse.
	rightRole struct{}
	// touchRole follows fingers and a stylus: one pointer orbits, two or more pan, pinch and twist.
	touchRole struct{}
)

// scaleRecognizer is the recognizer of one role. session is the navigation's state for the gesture that is running.
type scaleRecognizer[R any] struct {
	*gestures.ScaleGestureRecognizer
	session dragSession
}

// doubleClickRecognizer is the recognizer of the double click (and the double tap) that fits the scene.
type doubleClickRecognizer struct {
	*gestures.DoubleTapGestureRecognizer
}

// scaleFactory makes the factory of the recognizers of a role: the binding makes each with its devices and buttons, and
// the navigation sets its callbacks each time the widget is built.
func scaleFactory[R any](binding *gestures.GestureBinding, devices []ui.PointerDeviceKind, buttons gestures.AllowedButtonsFilter,
	configure func(r *scaleRecognizer[R]),
) widgets.GestureRecognizerFactory {
	return widgets.NewGestureRecognizerFactoryWithHandlers(
		func() *scaleRecognizer[R] {
			return &scaleRecognizer[R]{ScaleGestureRecognizer: gestures.NewScaleGestureRecognizer(binding, gestures.ScaleGestureRecognizerOpts{
				SupportedDevices: devices, AllowedButtonsFilter: buttons,
			})}
		},
		configure,
	)
}

// doubleClickFactory makes the factory of the double click recognizer.
func doubleClickFactory(binding *gestures.GestureBinding, configure func(r *doubleClickRecognizer)) widgets.GestureRecognizerFactory {
	return widgets.NewGestureRecognizerFactoryWithHandlers(
		func() *doubleClickRecognizer {
			return &doubleClickRecognizer{gestures.NewDoubleTapGestureRecognizer(binding, gestures.DoubleTapGestureRecognizerOpts{
				SupportedDevices: []ui.PointerDeviceKind{ui.PointerDeviceKindMouse, ui.PointerDeviceKindTouch, ui.PointerDeviceKindStylus},
			})}
		},
		configure,
	)
}

// buttonOnly allows a pointer that has exactly the button pressed.
func buttonOnly(button int) gestures.AllowedButtonsFilter {
	return func(buttons int) bool { return buttons == button }
}

var (
	mouseDevices = []ui.PointerDeviceKind{ui.PointerDeviceKindMouse}
	touchDevices = []ui.PointerDeviceKind{ui.PointerDeviceKindTouch, ui.PointerDeviceKindStylus}
)

// dragSession is what the navigation keeps while a recognizer's gesture runs.
type dragSession struct {
	// action is what the drag does; None when the gesture is not for the navigation.
	action NavigationAction
	// trackpad is whether the gesture is the pan/zoom of a trackpad.
	trackpad bool
	// pivot is the point of the world that an orbit turns around, when the app can tell it.
	pivot    vecmath.Vec3
	hasPivot bool
	// locked is whether the navigation locked the pointer: the motion then comes from the pointer events, whose positions
	// do not change (SPEC §27.6).
	locked bool
	// skip makes the next update ignore its motion: the pointer lock ended during the drag and the pointer jumped back.
	skip bool
	// lastFocal is the focal point of the previous update, in the navigation's coordinates. The recognizer reports the
	// motion since its previous event only, which leaves out what the pointer moved before the gesture was accepted; the
	// navigation takes its own differences, from where the pointer went down.
	lastFocal ui.Offset
	// lastScale and lastRotation are the scale and the twist of the previous update, which the recognizer reports as
	// totals since the gesture (or the last change of the number of pointers) began.
	lastScale, lastRotation float64
}

// reset starts a gesture.
func (d *dragSession) reset() { *d = dragSession{lastScale: 1} }

// moved returns how far the focal point went since the previous call, and remembers it.
func (d *dragSession) moved(focal ui.Offset) ui.Offset {
	delta := focal.Sub(d.lastFocal)
	d.lastFocal = focal
	return delta
}
