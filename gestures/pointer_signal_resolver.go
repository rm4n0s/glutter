package gestures

import (
	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
)

// PointerSignalResolvedCallback is called with the signal event the resolver
// resolved to.
//
// Flutter: gestures/pointer_signal_resolver.dart PointerSignalResolvedCallback
type PointerSignalResolvedCallback func(event PointerSignalEvent)

func isSameEvent(a, b PointerSignalEvent) bool {
	return originalOrSelf(a) == originalOrSelf(b)
}

func originalOrSelf(e PointerEvent) PointerEvent {
	if o := e.Original(); o != nil {
		return o
	}
	return e
}

// PointerSignalResolver lets the objects that handle a signal event compete for
// it: the first to Register wins, because hit testing visits the front-most
// object first. The binding calls Resolve after it has dispatched the event.
//
// Flutter: gestures/pointer_signal_resolver.dart PointerSignalResolver
type PointerSignalResolver struct {
	firstRegisteredCallback PointerSignalResolvedCallback
	currentEvent            PointerSignalEvent
}

// NewPointerSignalResolver creates a resolver.
//
// Flutter: gestures/pointer_signal_resolver.dart PointerSignalResolver.new
func NewPointerSignalResolver() *PointerSignalResolver { return &PointerSignalResolver{} }

// Register asks to handle the event. If a callback was registered already for
// this event, the new one is ignored. Registering for another event while one
// is pending panics with the tag PointerSignalEventMismatch in debug builds.
//
// Flutter: gestures/pointer_signal_resolver.dart PointerSignalResolver.register
func (r *PointerSignalResolver) Register(event PointerSignalEvent, callback PointerSignalResolvedCallback) {
	if foundation.DebugMode && r.currentEvent != nil && !isSameEvent(r.currentEvent, event) {
		panic(errors.New("PointerSignalEventMismatch", "a signal event was registered while another is pending"))
	}
	if r.firstRegisteredCallback != nil {
		return
	}
	r.currentEvent = event
	r.firstRegisteredCallback = callback
}

// Resolve calls the callback that registered first, or, if none did, tells the
// platform that it may do its default action for the event.
//
// Flutter: gestures/pointer_signal_resolver.dart PointerSignalResolver.resolve
func (r *PointerSignalResolver) Resolve(event PointerSignalEvent) {
	if r.firstRegisteredCallback == nil {
		// Nothing in the framework/app wants to handle the event. Allow the
		// platform to trigger any default native actions.
		event.Respond(true)
		return
	}
	if foundation.DebugMode && !isSameEvent(r.currentEvent, event) {
		panic(errors.New("PointerSignalEventMismatch", "a different signal event was resolved than the one registered"))
	}
	// TODO(step 7.3): recover a panic in the callback and report it, as
	// Flutter does with FlutterError.reportError.
	r.firstRegisteredCallback(r.currentEvent)
	r.firstRegisteredCallback = nil
	r.currentEvent = nil
}
