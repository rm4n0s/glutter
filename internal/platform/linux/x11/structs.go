//go:build linux && !android

package x11

import "unsafe"

// Go mirrors of the Xlib and XInput2 structures this backend reads or writes
// (LP64). Each has a layout test against the C headers. XEvent itself is a
// union of 192 bytes that lives in C memory; the types below are views of it.

// xEventSize is sizeof(XEvent).
const xEventSize = 192

// xAnyEvent is XAnyEvent: the start of every event.
type xAnyEvent struct {
	typ       int32
	_         int32
	serial    uint64
	sendEvent int32
	_         int32
	display   uintptr
	window    uint64
}

// xConfigureEvent is XConfigureEvent: a window was moved, resized or restacked.
type xConfigureEvent struct {
	typ              int32
	_                int32
	serial           uint64
	sendEvent        int32
	_                int32
	display          uintptr
	event            uint64
	window           uint64
	x, y             int32
	width, height    int32
	borderWidth      int32
	_                int32
	above            uint64
	overrideRedirect int32
	_                int32
}

// xClientMessageEvent is XClientMessageEvent. The data is five longs for format
// 32.
type xClientMessageEvent struct {
	typ         int32
	_           int32
	serial      uint64
	sendEvent   int32
	_           int32
	display     uintptr
	window      uint64
	messageType uint64
	format      int32
	_           int32
	data        [5]int64
}

// xFocusChangeEvent is XFocusChangeEvent.
type xFocusChangeEvent struct {
	typ       int32
	_         int32
	serial    uint64
	sendEvent int32
	_         int32
	display   uintptr
	window    uint64
	mode      int32
	detail    int32
}

// xExposeEvent is XExposeEvent.
type xExposeEvent struct {
	typ           int32
	_             int32
	serial        uint64
	sendEvent     int32
	_             int32
	display       uintptr
	window        uint64
	x, y          int32
	width, height int32
	count         int32
	_             int32
}

// xMapEvent is XMapEvent (and XUnmapEvent, which has the same start).
type xMapEvent struct {
	typ              int32
	_                int32
	serial           uint64
	sendEvent        int32
	_                int32
	display          uintptr
	event            uint64
	window           uint64
	overrideRedirect int32
	_                int32
}

// xDestroyWindowEvent is XDestroyWindowEvent.
type xDestroyWindowEvent struct {
	typ       int32
	_         int32
	serial    uint64
	sendEvent int32
	_         int32
	display   uintptr
	event     uint64
	window    uint64
}

// xSelectionRequestEvent is XSelectionRequestEvent: another client asks the owner
// of a selection for its content.
type xSelectionRequestEvent struct {
	typ       int32
	_         int32
	serial    uint64
	sendEvent int32
	_         int32
	display   uintptr
	owner     uint64
	requestor uint64
	selection uint64
	target    uint64
	property  uint64
	time      uint64
}

// xSelectionEvent is XSelectionEvent: the owner answers a request for a
// selection.
type xSelectionEvent struct {
	typ       int32
	_         int32
	serial    uint64
	sendEvent int32
	_         int32
	display   uintptr
	requestor uint64
	selection uint64
	target    uint64
	property  uint64
	time      uint64
}

// xSelectionClearEvent is XSelectionClearEvent: another client took a selection
// that this one owned.
type xSelectionClearEvent struct {
	typ       int32
	_         int32
	serial    uint64
	sendEvent int32
	_         int32
	display   uintptr
	window    uint64
	selection uint64
	time      uint64
}

// xPropertyEvent is XPropertyEvent: a property of a window changed.
type xPropertyEvent struct {
	typ       int32
	_         int32
	serial    uint64
	sendEvent int32
	_         int32
	display   uintptr
	window    uint64
	atom      uint64
	time      uint64
	state     int32
	_         int32
}

// xGenericEventCookie is XGenericEventCookie: the event of an extension, with its
// data fetched by XGetEventData.
type xGenericEventCookie struct {
	typ       int32
	_         int32
	serial    uint64
	sendEvent int32
	_         int32
	display   uintptr
	extension int32
	evtype    int32
	cookie    uint32
	_         uint32
	data      uintptr
}

// xErrorEvent is XErrorEvent, given to the error handler.
type xErrorEvent struct {
	typ         int32
	_           int32
	display     uintptr
	resourceID  uint64
	serial      uint64
	errorCode   uint8
	requestCode uint8
	minorCode   uint8
	_           [5]byte
}

// xSetWindowAttributes is XSetWindowAttributes.
type xSetWindowAttributes struct {
	backgroundPixmap   uint64
	backgroundPixel    uint64
	borderPixmap       uint64
	borderPixel        uint64
	bitGravity         int32
	winGravity         int32
	backingStore       int32
	_                  int32
	backingPlanes      uint64
	backingPixel       uint64
	saveUnder          int32
	_                  int32
	eventMask          int64
	doNotPropagateMask int64
	overrideRedirect   int32
	_                  int32
	colormap           uint64
	cursor             uint64
}

// xVisualInfo is XVisualInfo.
type xVisualInfo struct {
	visual       uintptr
	visualID     uint64
	screen       int32
	depth        int32
	class        int32
	_            int32
	redMask      uint64
	greenMask    uint64
	blueMask     uint64
	colormapSize int32
	bitsPerRGB   int32
}

// xSizeHints is XSizeHints.
type xSizeHints struct {
	flags                 int64
	x, y                  int32
	width, height         int32
	minWidth, minHeight   int32
	maxWidth, maxHeight   int32
	widthInc, heightInc   int32
	minAspect, maxAspect  [2]int32
	baseWidth, baseHeight int32
	winGravity            int32
	_                     int32
}

// xClassHint is XClassHint.
type xClassHint struct {
	resName  uintptr
	resClass uintptr
}

// xiEventMask is XIEventMask.
type xiEventMask struct {
	deviceID int32
	maskLen  int32
	mask     uintptr
}

// xiButtonState is XIButtonState.
type xiButtonState struct {
	maskLen int32
	_       int32
	mask    uintptr
}

// xiValuatorState is XIValuatorState: the axes that changed, as a bit mask, and
// their values packed in the order of the bits.
type xiValuatorState struct {
	maskLen int32
	_       int32
	mask    uintptr
	values  uintptr
}

// xiModifierState is XIModifierState, and XIGroupState has the same shape.
type xiModifierState struct {
	base, latched, locked, effective int32
}

// xiDeviceEvent is XIDeviceEvent: a key, button, motion or touch event.
type xiDeviceEvent struct {
	typ       int32
	_         int32
	serial    uint64
	sendEvent int32
	_         int32
	display   uintptr
	extension int32
	evtype    int32
	time      uint64
	deviceID  int32
	sourceID  int32
	detail    int32
	_         int32
	root      uint64
	event     uint64
	child     uint64
	rootX     float64
	rootY     float64
	eventX    float64
	eventY    float64
	flags     int32
	_         int32
	buttons   xiButtonState
	valuators xiValuatorState
	mods      xiModifierState
	group     xiModifierState
}

// xiEnterEvent is XIEnterEvent: the pointer entered or left, or the focus
// changed.
type xiEnterEvent struct {
	typ        int32
	_          int32
	serial     uint64
	sendEvent  int32
	_          int32
	display    uintptr
	extension  int32
	evtype     int32
	time       uint64
	deviceID   int32
	sourceID   int32
	detail     int32
	_          int32
	root       uint64
	event      uint64
	child      uint64
	rootX      float64
	rootY      float64
	eventX     float64
	eventY     float64
	mode       int32
	focus      int32
	sameScreen int32
	_          int32
	buttons    xiButtonState
	mods       xiModifierState
	group      xiModifierState
}

// xiDeviceInfo is XIDeviceInfo.
type xiDeviceInfo struct {
	deviceID   int32
	_          int32
	name       uintptr
	use        int32
	attachment int32
	enabled    int32
	numClasses int32
	classes    uintptr // XIAnyClassInfo **
}

// xiAnyClassInfo is XIAnyClassInfo: the start of every class of a device.
type xiAnyClassInfo struct {
	typ      int32
	sourceID int32
}

// xiScrollClassInfo is XIScrollClassInfo: an axis that scrolls.
type xiScrollClassInfo struct {
	typ        int32
	sourceID   int32
	number     int32
	scrollType int32
	increment  float64
	flags      int32
	_          int32
}

// xiValuatorClassInfo is XIValuatorClassInfo: an axis and its current value.
type xiValuatorClassInfo struct {
	typ        int32
	sourceID   int32
	number     int32
	_          int32
	label      uint64
	min, max   float64
	value      float64
	resolution int32
	mode       int32
}

// xColor is XColor.
type xColor struct {
	pixel      uint64
	red, green uint16
	blue       uint16
	flags      uint8
	_          uint8
}

// view returns the C memory at addr as a *T. The memory must belong to C (or
// be allocated by dl) and hold a T.
func view[T any](addr uintptr) *T {
	return (*T)(*(*unsafe.Pointer)(unsafe.Pointer(&addr)))
}

// xrrMonitorInfo is XRRMonitorInfo: a monitor, a rectangle of the screen that one or
// more outputs show.
type xrrMonitorInfo struct {
	name      uint64
	primary   int32
	automatic int32
	noutput   int32
	x, y      int32
	width     int32
	height    int32
	mmWidth   int32
	mmHeight  int32
	_         int32
	outputs   uintptr
}
