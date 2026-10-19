package gestures

import "github.com/rm4n0s/glutter/ui"

// EagerGestureRecognizerOpts are the optional parameters of
// NewEagerGestureRecognizer.
//
// Flutter: gestures/eager.dart EagerGestureRecognizer.new
type EagerGestureRecognizerOpts struct {
	// SupportedDevices are the kinds of pointer the recognizer accepts; nil for
	// all.
	SupportedDevices []ui.PointerDeviceKind
	// AllowedButtonsFilter says whether the buttons of a pointer are allowed.
	AllowedButtonsFilter AllowedButtonsFilter
}

// EagerGestureRecognizer wins the arena as soon as a pointer goes down. It is
// used to claim the pointers of a widget over the other recognizers.
//
// Flutter: gestures/eager.dart EagerGestureRecognizer
type EagerGestureRecognizer struct {
	OneSequenceGestureRecognizer
}

// NewEagerGestureRecognizer creates a recognizer that works with the binding.
//
// Flutter: gestures/eager.dart EagerGestureRecognizer.new
func NewEagerGestureRecognizer(binding *GestureBinding, opts ...EagerGestureRecognizerOpts) *EagerGestureRecognizer {
	o := oneOpts(opts)
	r := &EagerGestureRecognizer{}
	r.InitOneSequence(binding, r, GestureRecognizerOpts{SupportedDevices: o.SupportedDevices, AllowedButtonsFilter: o.AllowedButtonsFilter})
	return r
}

// AddAllowedPointer tracks the pointer, wins the arena and stops tracking.
//
// Flutter: gestures/eager.dart EagerGestureRecognizer.addAllowedPointer
func (r *EagerGestureRecognizer) AddAllowedPointer(event *PointerDownEvent) {
	r.OneSequenceGestureRecognizer.AddAllowedPointer(event)
	r.Resolve(GestureAccepted)
	r.StopTrackingPointer(event.Pointer())
}

// DebugDescription is "eager".
//
// Flutter: gestures/eager.dart EagerGestureRecognizer.debugDescription
func (r *EagerGestureRecognizer) DebugDescription() string { return "eager" }

// DidStopTrackingLastPointer does nothing.
//
// Flutter: gestures/eager.dart EagerGestureRecognizer.didStopTrackingLastPointer
func (r *EagerGestureRecognizer) DidStopTrackingLastPointer(int) {}

// HandleEvent does nothing.
//
// Flutter: gestures/eager.dart EagerGestureRecognizer.handleEvent
func (r *EagerGestureRecognizer) HandleEvent(PointerEvent) {}
