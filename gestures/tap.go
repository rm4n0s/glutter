package gestures

import (
	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// TapDownDetails is the data of a pointer that touched the screen, which might
// be the start of a tap.
//
// Flutter: gestures/tap.dart TapDownDetails
type TapDownDetails struct {
	// GlobalPosition is where the pointer touched the screen, in global
	// coordinates.
	GlobalPosition ui.Offset
	// LocalPosition is the position in the coordinates of the target.
	LocalPosition ui.Offset
	// Kind is the kind of device that touched the screen.
	Kind foundation.Opt[ui.PointerDeviceKind]
}

// TapDownOpts are the optional parameters of NewTapDownDetails.
//
// Flutter: gestures/tap.dart TapDownDetails.new
type TapDownOpts struct {
	// GlobalPosition is where the pointer touched the screen.
	GlobalPosition ui.Offset
	// LocalPosition is the position in the coordinates of the target; the
	// default is GlobalPosition.
	LocalPosition foundation.Opt[ui.Offset]
	// Kind is the kind of device.
	Kind foundation.Opt[ui.PointerDeviceKind]
}

// NewTapDownDetails creates the details.
//
// Flutter: gestures/tap.dart TapDownDetails.new
func NewTapDownDetails(opts ...TapDownOpts) TapDownDetails {
	o := oneOpts(opts)
	return TapDownDetails{GlobalPosition: o.GlobalPosition, LocalPosition: localOrGlobal(o.LocalPosition, o.GlobalPosition), Kind: o.Kind}
}

// GetGlobalPosition is GlobalPosition.
//
// Flutter: gestures/tap.dart TapDownDetails.globalPosition
func (d TapDownDetails) GetGlobalPosition() ui.Offset { return d.GlobalPosition }

// GetLocalPosition is LocalPosition.
//
// Flutter: gestures/tap.dart TapDownDetails.localPosition
func (d TapDownDetails) GetLocalPosition() ui.Offset { return d.LocalPosition }

// DebugFillProperties adds the properties of the details.
//
// Flutter: gestures/tap.dart TapDownDetails.debugFillProperties
func (d TapDownDetails) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("globalPosition", d.GlobalPosition))
	p.Add(foundation.NewDiagnosticsProperty("localPosition", d.LocalPosition))
	p.Add(foundation.NewEnumProperty("kind", d.Kind))
}

// String describes the details.
//
// Flutter: gestures/tap.dart TapDownDetails.toString
func (d TapDownDetails) String() string { return foundation.DiagnosticableString(d) }

// GestureTapDownCallback is called when a pointer that might tap touches the
// screen.
//
// Flutter: gestures/tap.dart GestureTapDownCallback
type GestureTapDownCallback func(details TapDownDetails)

// TapUpDetails is the data of a pointer that stopped touching the screen to end a
// tap.
//
// Flutter: gestures/tap.dart TapUpDetails
type TapUpDetails struct {
	// GlobalPosition is where the pointer was lifted, in global coordinates.
	GlobalPosition ui.Offset
	// LocalPosition is the position in the coordinates of the target.
	LocalPosition ui.Offset
	// Kind is the kind of device that tapped.
	Kind ui.PointerDeviceKind
}

// TapUpOpts are the optional parameters of NewTapUpDetails.
//
// Flutter: gestures/tap.dart TapUpDetails.new
type TapUpOpts struct {
	// GlobalPosition is where the pointer was lifted.
	GlobalPosition ui.Offset
	// LocalPosition is the position in the coordinates of the target; the
	// default is GlobalPosition.
	LocalPosition foundation.Opt[ui.Offset]
}

// NewTapUpDetails creates the details for a tap of a device of the kind.
//
// Flutter: gestures/tap.dart TapUpDetails.new
func NewTapUpDetails(kind ui.PointerDeviceKind, opts ...TapUpOpts) TapUpDetails {
	o := oneOpts(opts)
	return TapUpDetails{GlobalPosition: o.GlobalPosition, LocalPosition: localOrGlobal(o.LocalPosition, o.GlobalPosition), Kind: kind}
}

// GetGlobalPosition is GlobalPosition.
//
// Flutter: gestures/tap.dart TapUpDetails.globalPosition
func (d TapUpDetails) GetGlobalPosition() ui.Offset { return d.GlobalPosition }

// GetLocalPosition is LocalPosition.
//
// Flutter: gestures/tap.dart TapUpDetails.localPosition
func (d TapUpDetails) GetLocalPosition() ui.Offset { return d.LocalPosition }

// DebugFillProperties adds the properties of the details.
//
// Flutter: gestures/tap.dart TapUpDetails.debugFillProperties
func (d TapUpDetails) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("globalPosition", d.GlobalPosition))
	p.Add(foundation.NewDiagnosticsProperty("localPosition", d.LocalPosition))
	p.Add(foundation.NewEnumProperty("kind", foundation.Some(d.Kind)))
}

// String describes the details.
//
// Flutter: gestures/tap.dart TapUpDetails.toString
func (d TapUpDetails) String() string { return foundation.DiagnosticableString(d) }

// TapMoveDetails is the data of a pointer that moved while a tap was possible.
//
// Flutter: gestures/tap.dart TapMoveDetails
type TapMoveDetails struct {
	// GlobalPosition is the global position of the pointer.
	GlobalPosition ui.Offset
	// LocalPosition is the position in the coordinates of the target.
	LocalPosition ui.Offset
	// Kind is the kind of device.
	Kind ui.PointerDeviceKind
	// Delta is how far the pointer moved since the last event.
	Delta ui.Offset
}

// TapMoveOpts are the optional parameters of NewTapMoveDetails.
//
// Flutter: gestures/tap.dart TapMoveDetails.new
type TapMoveOpts struct {
	// GlobalPosition is the global position of the pointer.
	GlobalPosition ui.Offset
	// Delta is how far the pointer moved.
	Delta ui.Offset
	// LocalPosition is the position in the coordinates of the target; the
	// default is GlobalPosition.
	LocalPosition foundation.Opt[ui.Offset]
}

// NewTapMoveDetails creates the details for a device of the kind.
//
// Flutter: gestures/tap.dart TapMoveDetails.new
func NewTapMoveDetails(kind ui.PointerDeviceKind, opts ...TapMoveOpts) TapMoveDetails {
	o := oneOpts(opts)
	return TapMoveDetails{
		GlobalPosition: o.GlobalPosition, LocalPosition: localOrGlobal(o.LocalPosition, o.GlobalPosition),
		Kind: kind, Delta: o.Delta,
	}
}

// GestureTapUpCallback is called when a pointer that tapped stops touching the
// screen.
//
// Flutter: gestures/tap.dart GestureTapUpCallback
type GestureTapUpCallback func(details TapUpDetails)

// GestureTapCallback is called when a tap happens.
//
// Flutter: gestures/tap.dart GestureTapCallback
type GestureTapCallback func()

// GestureTapMoveCallback is called when a pointer that might tap moves.
//
// Flutter: gestures/tap.dart GestureTapMoveCallback
type GestureTapMoveCallback func(details TapMoveDetails)

// GestureTapCancelCallback is called when the pointer of a tap is cancelled.
//
// Flutter: gestures/tap.dart GestureTapCancelCallback
type GestureTapCancelCallback func()

// BaseTapHooks are the methods of a BaseTapGestureRecognizer that the subclass
// supplies.
//
// Flutter: gestures/tap.dart BaseTapGestureRecognizer
type BaseTapHooks interface {
	PrimaryPointerHooks
	// HandleTapDown is called when the pointer has been down for PressTimeout
	// or the recognizer wins the arena.
	//
	// Flutter: gestures/tap.dart BaseTapGestureRecognizer.handleTapDown
	HandleTapDown(down *PointerDownEvent)
	// HandleTapUp is called when the tap is recognized: the pointer is up and
	// the recognizer won.
	//
	// Flutter: gestures/tap.dart BaseTapGestureRecognizer.handleTapUp
	HandleTapUp(down *PointerDownEvent, up *PointerUpEvent)
	// HandleTapMove is called when the pointer moves. The default does
	// nothing.
	//
	// Flutter: gestures/tap.dart BaseTapGestureRecognizer.handleTapMove
	HandleTapMove(move *PointerMoveEvent)
	// HandleTapCancel is called when the tap is cancelled after HandleTapDown.
	// The cancel event is nil if the cancellation was not a cancel event; the
	// reason says why.
	//
	// Flutter: gestures/tap.dart BaseTapGestureRecognizer.handleTapCancel
	HandleTapCancel(down *PointerDownEvent, cancel *PointerCancelEvent, reason string)
}

// BaseTapOpts are the parameters of InitBaseTap.
//
// Flutter: gestures/tap.dart BaseTapGestureRecognizer.new
type BaseTapOpts struct {
	GestureRecognizerOpts
	// PreAcceptSlopTolerance is how far the pointer may move before the
	// recognizer wins the arena without it giving up.
	PreAcceptSlopTolerance SlopTolerance
	// PostAcceptSlopTolerance is how far the pointer may move after the
	// recognizer won.
	PostAcceptSlopTolerance SlopTolerance
}

// BaseTapGestureRecognizer is the base of the recognizers that recognize taps:
// a pointer that goes down, and up again without moving much.
//
// Flutter: gestures/tap.dart BaseTapGestureRecognizer
type BaseTapGestureRecognizer struct {
	PrimaryPointerGestureRecognizer
	baseTap BaseTapHooks

	sentTapDown               bool
	wonArenaForPrimaryPointer bool
	down                      *PointerDownEvent
	up                        *PointerUpEvent
}

// InitBaseTap sets up the recognizer; self is the recognizer that embeds it.
//
// Flutter: gestures/tap.dart BaseTapGestureRecognizer.new
func (r *BaseTapGestureRecognizer) InitBaseTap(binding *GestureBinding, self BaseTapHooks, opts BaseTapOpts) {
	r.InitPrimaryPointer(binding, self, PrimaryPointerOpts{
		GestureRecognizerOpts:   opts.GestureRecognizerOpts,
		Deadline:                foundation.Some(PressTimeout),
		PreAcceptSlopTolerance:  opts.PreAcceptSlopTolerance,
		PostAcceptSlopTolerance: opts.PostAcceptSlopTolerance,
	})
	r.baseTap = self
}

// HandleTapMove does nothing.
//
// Flutter: gestures/tap.dart BaseTapGestureRecognizer.handleTapMove
func (r *BaseTapGestureRecognizer) HandleTapMove(*PointerMoveEvent) {}

// AddAllowedPointer remembers the down event of a new tap.
//
// Flutter: gestures/tap.dart BaseTapGestureRecognizer.addAllowedPointer
func (r *BaseTapGestureRecognizer) AddAllowedPointer(event *PointerDownEvent) {
	if r.State() == GestureRecognizerStateReady {
		// If there is no result in the previous gesture arena, we ignore them
		// and prepare to accept a new pointer.
		if r.down != nil && r.up != nil {
			r.reset()
		}
		// down must be assigned here instead of in HandlePrimaryPointer,
		// because AcceptGesture might be called before HandlePrimaryPointer,
		// which relies on down to call HandleTapDown.
		r.down = event
	}
	if r.down != nil {
		// This is false when this tap gesture has been rejected while the
		// pointer is down (for example by movement), and another allowed
		// pointer is added, in which case all pointers are ignored.
		r.PrimaryPointerGestureRecognizer.AddAllowedPointer(event)
	}
}

// StartTrackingPointer tracks the pointer; the recognizer must have a down
// event.
//
// Flutter: gestures/tap.dart BaseTapGestureRecognizer.startTrackingPointer
func (r *BaseTapGestureRecognizer) StartTrackingPointer(pointer int, transform *ui.Matrix4) {
	// The recognizer should never track any pointers when down is nil.
	if foundation.DebugMode && r.down == nil {
		panic(errors.New(tagTapTrackingWithoutDown, "the tap recognizer tracks a pointer without a down event"))
	}
	r.PrimaryPointerGestureRecognizer.StartTrackingPointer(pointer, transform)
}

// HandlePrimaryPointer checks for the end of the tap.
//
// Flutter: gestures/tap.dart BaseTapGestureRecognizer.handlePrimaryPointer
func (r *BaseTapGestureRecognizer) HandlePrimaryPointer(event PointerEvent) {
	switch e := event.(type) {
	case *PointerUpEvent:
		r.up = e
		r.checkUp()
	case *PointerCancelEvent:
		r.baseTap.Resolve(GestureRejected)
		if r.sentTapDown {
			r.checkCancel(e, "")
		}
		r.reset()
	default:
		if event.Buttons() != r.down.Buttons() {
			r.baseTap.Resolve(GestureRejected)
			r.StopTrackingPointer(r.PrimaryPointer().Value())
		} else if move, ok := event.(*PointerMoveEvent); ok {
			r.checkMove(move)
		}
	}
}

// Resolve cancels the tap if the recognizer won and is now rejected, then
// resolves.
//
// Flutter: gestures/tap.dart BaseTapGestureRecognizer.resolve
func (r *BaseTapGestureRecognizer) Resolve(disposition GestureDisposition) {
	if r.wonArenaForPrimaryPointer && disposition == GestureRejected {
		// This can happen if the gesture has been cancelled. For example, when
		// the pointer has exceeded the touch slop, the buttons have been
		// changed, or if the recognizer is disposed.
		if foundation.DebugMode && !r.sentTapDown {
			panic(errors.New(tagTapCancelWithoutDown, "the tap won the arena without sending a tap down"))
		}
		r.checkCancel(nil, "spontaneous")
		r.reset()
	}
	r.PrimaryPointerGestureRecognizer.Resolve(disposition)
}

// DidExceedDeadline sends the tap down.
//
// Flutter: gestures/tap.dart BaseTapGestureRecognizer.didExceedDeadline
func (r *BaseTapGestureRecognizer) DidExceedDeadline() { r.checkDown() }

// AcceptGesture sends the tap down and, if the pointer is up, the tap up.
//
// Flutter: gestures/tap.dart BaseTapGestureRecognizer.acceptGesture
func (r *BaseTapGestureRecognizer) AcceptGesture(pointer int) {
	r.PrimaryPointerGestureRecognizer.AcceptGesture(pointer)
	if p, _ := r.PrimaryPointer().Get(); pointer == p {
		r.checkDown()
		r.wonArenaForPrimaryPointer = true
		r.checkUp()
	}
}

// RejectGesture cancels the tap if it sent a tap down.
//
// Flutter: gestures/tap.dart BaseTapGestureRecognizer.rejectGesture
func (r *BaseTapGestureRecognizer) RejectGesture(pointer int) {
	r.PrimaryPointerGestureRecognizer.RejectGesture(pointer)
	if p, _ := r.PrimaryPointer().Get(); pointer == p {
		// Another gesture won the arena. The base has already left the possible
		// state (Flutter asserts it).
		if r.sentTapDown {
			r.checkCancel(nil, "forced")
		}
		r.reset()
	}
}

func (r *BaseTapGestureRecognizer) checkDown() {
	if r.sentTapDown {
		return
	}
	r.baseTap.HandleTapDown(r.down)
	r.sentTapDown = true
}

func (r *BaseTapGestureRecognizer) checkUp() {
	if !r.wonArenaForPrimaryPointer || r.up == nil {
		return
	}
	r.baseTap.HandleTapUp(r.down, r.up)
	r.reset()
}

func (r *BaseTapGestureRecognizer) checkCancel(event *PointerCancelEvent, note string) {
	r.baseTap.HandleTapCancel(r.down, event, note)
}

func (r *BaseTapGestureRecognizer) checkMove(event *PointerMoveEvent) {
	r.baseTap.HandleTapMove(event)
}

func (r *BaseTapGestureRecognizer) reset() {
	r.sentTapDown = false
	r.wonArenaForPrimaryPointer = false
	r.up = nil
	r.down = nil
}

// DebugFillProperties adds the state of the tap.
//
// Flutter: gestures/tap.dart BaseTapGestureRecognizer.debugFillProperties
func (r *BaseTapGestureRecognizer) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	r.PrimaryPointerGestureRecognizer.DebugFillProperties(p)
	p.Add(foundation.NewFlagProperty("wonArenaForPrimaryPointer", foundation.Some(r.wonArenaForPrimaryPointer),
		foundation.FlagPropertyOpts{IfTrue: "won arena"}))
	if r.up != nil {
		p.Add(foundation.NewDiagnosticsProperty("finalPosition", r.up.Position(),
			foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull}))
		p.Add(foundation.NewDiagnosticsProperty("finalLocalPosition", r.up.LocalPosition(),
			foundation.DiagnosticsPropertyOpts{DefaultValue: r.up.Position()}))
	} else {
		p.Add(foundation.NewDiagnosticsProperty[any]("finalPosition", nil,
			foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull}))
		p.Add(foundation.NewDiagnosticsProperty[any]("finalLocalPosition", nil,
			foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull}))
	}
	if r.down != nil {
		p.Add(foundation.NewIntProperty("button", foundation.Some(r.down.Buttons()),
			foundation.NumberPropertyOpts{DefaultValue: foundation.DefaultNull}))
	} else {
		p.Add(foundation.NewIntProperty("button", foundation.None[int](),
			foundation.NumberPropertyOpts{DefaultValue: foundation.DefaultNull}))
	}
	p.Add(foundation.NewFlagProperty("sentTapDown", foundation.Some(r.sentTapDown),
		foundation.FlagPropertyOpts{IfTrue: "sent tap down"}))
}

// DebugDescription is "base tap".
//
// Flutter: gestures/tap.dart BaseTapGestureRecognizer.debugDescription
func (r *BaseTapGestureRecognizer) DebugDescription() string { return "base tap" }

// TapGestureRecognizerOpts are the optional parameters of
// NewTapGestureRecognizer.
//
// Flutter: gestures/tap.dart TapGestureRecognizer.new
type TapGestureRecognizerOpts struct {
	// DebugOwner is the object that owns the recognizer.
	DebugOwner any
	// SupportedDevices are the kinds of pointer the recognizer accepts; nil for
	// all.
	SupportedDevices []ui.PointerDeviceKind
	// AllowedButtonsFilter says whether the buttons of a pointer are allowed.
	AllowedButtonsFilter AllowedButtonsFilter
	// PreAcceptSlopTolerance is how far the pointer may move before the
	// recognizer wins the arena without it giving up.
	PreAcceptSlopTolerance SlopTolerance
	// PostAcceptSlopTolerance is how far the pointer may move after the
	// recognizer won.
	PostAcceptSlopTolerance SlopTolerance
}

func (o TapGestureRecognizerOpts) baseTap() BaseTapOpts {
	return BaseTapOpts{
		GestureRecognizerOpts: GestureRecognizerOpts{
			DebugOwner: o.DebugOwner, SupportedDevices: o.SupportedDevices, AllowedButtonsFilter: o.AllowedButtonsFilter,
		},
		PreAcceptSlopTolerance:  o.PreAcceptSlopTolerance,
		PostAcceptSlopTolerance: o.PostAcceptSlopTolerance,
	}
}

// TapGestureRecognizer recognizes taps: a primary, secondary or tertiary
// button that goes down and up again without the pointer moving much. Set the
// callbacks of the buttons to listen for; a pointer whose button has no callback
// is not allowed.
//
// Flutter: gestures/tap.dart TapGestureRecognizer
type TapGestureRecognizer struct {
	BaseTapGestureRecognizer

	// OnTapDown is called when a pointer that might tap with the primary button
	// touches the screen.
	//
	// Flutter: gestures/tap.dart TapGestureRecognizer.onTapDown
	OnTapDown GestureTapDownCallback
	// OnTapUp is called when a pointer taps with the primary button and is
	// lifted.
	//
	// Flutter: gestures/tap.dart TapGestureRecognizer.onTapUp
	OnTapUp GestureTapUpCallback
	// OnTap is called after OnTapUp, when the primary button taps.
	//
	// Flutter: gestures/tap.dart TapGestureRecognizer.onTap
	OnTap GestureTapCallback
	// OnTapMove is called when the pointer moves while a tap with the primary
	// button is possible.
	//
	// Flutter: gestures/tap.dart TapGestureRecognizer.onTapMove
	OnTapMove GestureTapMoveCallback
	// OnTapCancel is called when the pointer that sent OnTapDown will not tap.
	//
	// Flutter: gestures/tap.dart TapGestureRecognizer.onTapCancel
	OnTapCancel GestureTapCancelCallback
	// OnSecondaryTap is called when the secondary button taps.
	//
	// Flutter: gestures/tap.dart TapGestureRecognizer.onSecondaryTap
	OnSecondaryTap GestureTapCallback
	// OnSecondaryTapDown is called when a pointer that might tap with the
	// secondary button touches the screen.
	//
	// Flutter: gestures/tap.dart TapGestureRecognizer.onSecondaryTapDown
	OnSecondaryTapDown GestureTapDownCallback
	// OnSecondaryTapUp is called when the secondary button taps and is
	// lifted.
	//
	// Flutter: gestures/tap.dart TapGestureRecognizer.onSecondaryTapUp
	OnSecondaryTapUp GestureTapUpCallback
	// OnSecondaryTapCancel is called when the pointer that sent
	// OnSecondaryTapDown will not tap.
	//
	// Flutter: gestures/tap.dart TapGestureRecognizer.onSecondaryTapCancel
	OnSecondaryTapCancel GestureTapCancelCallback
	// OnTertiaryTapDown is called when a pointer that might tap with the
	// tertiary button touches the screen.
	//
	// Flutter: gestures/tap.dart TapGestureRecognizer.onTertiaryTapDown
	OnTertiaryTapDown GestureTapDownCallback
	// OnTertiaryTapUp is called when the tertiary button taps and is lifted.
	//
	// Flutter: gestures/tap.dart TapGestureRecognizer.onTertiaryTapUp
	OnTertiaryTapUp GestureTapUpCallback
	// OnTertiaryTapCancel is called when the pointer that sent
	// OnTertiaryTapDown will not tap.
	//
	// Flutter: gestures/tap.dart TapGestureRecognizer.onTertiaryTapCancel
	OnTertiaryTapCancel GestureTapCancelCallback
}

// NewTapGestureRecognizer creates a recognizer that works with the binding.
//
// Flutter: gestures/tap.dart TapGestureRecognizer.new
func NewTapGestureRecognizer(binding *GestureBinding, opts ...TapGestureRecognizerOpts) *TapGestureRecognizer {
	r := &TapGestureRecognizer{}
	r.InitBaseTap(binding, r, oneOpts(opts).baseTap())
	return r
}

// IsPointerAllowed allows a pointer whose button has a callback.
//
// Flutter: gestures/tap.dart TapGestureRecognizer.isPointerAllowed
func (r *TapGestureRecognizer) IsPointerAllowed(event *PointerDownEvent) bool {
	switch event.Buttons() {
	case PrimaryButton:
		if r.OnTapDown == nil && r.OnTap == nil && r.OnTapUp == nil && r.OnTapCancel == nil && r.OnTapMove == nil {
			return false
		}
	case SecondaryButton:
		if r.OnSecondaryTap == nil && r.OnSecondaryTapDown == nil && r.OnSecondaryTapUp == nil && r.OnSecondaryTapCancel == nil {
			return false
		}
	case TertiaryButton:
		if r.OnTertiaryTapDown == nil && r.OnTertiaryTapUp == nil && r.OnTertiaryTapCancel == nil {
			return false
		}
	default:
		return false
	}
	return r.BaseTapGestureRecognizer.IsPointerAllowed(event)
}

// HandleTapDown calls the tap down callback of the button.
//
// Flutter: gestures/tap.dart TapGestureRecognizer.handleTapDown
func (r *TapGestureRecognizer) HandleTapDown(down *PointerDownEvent) {
	details := NewTapDownDetails(TapDownOpts{
		GlobalPosition: down.Position(), LocalPosition: foundation.Some(down.LocalPosition()),
		Kind: foundation.Some(r.GetKindForPointer(down.Pointer())),
	})
	switch down.Buttons() {
	case PrimaryButton:
		if r.OnTapDown != nil {
			r.InvokeCallback("onTapDown", func() { r.OnTapDown(details) })
		}
	case SecondaryButton:
		if r.OnSecondaryTapDown != nil {
			r.InvokeCallback("onSecondaryTapDown", func() { r.OnSecondaryTapDown(details) })
		}
	case TertiaryButton:
		if r.OnTertiaryTapDown != nil {
			r.InvokeCallback("onTertiaryTapDown", func() { r.OnTertiaryTapDown(details) })
		}
	}
}

// HandleTapUp calls the tap up and tap callbacks of the button.
//
// Flutter: gestures/tap.dart TapGestureRecognizer.handleTapUp
func (r *TapGestureRecognizer) HandleTapUp(down *PointerDownEvent, up *PointerUpEvent) {
	details := NewTapUpDetails(up.Kind(), TapUpOpts{GlobalPosition: up.Position(), LocalPosition: foundation.Some(up.LocalPosition())})
	switch down.Buttons() {
	case PrimaryButton:
		if r.OnTapUp != nil {
			r.InvokeCallback("onTapUp", func() { r.OnTapUp(details) })
		}
		if r.OnTap != nil {
			r.InvokeCallback("onTap", r.OnTap)
		}
	case SecondaryButton:
		if r.OnSecondaryTapUp != nil {
			r.InvokeCallback("onSecondaryTapUp", func() { r.OnSecondaryTapUp(details) })
		}
		if r.OnSecondaryTap != nil {
			r.InvokeCallback("onSecondaryTap", func() { r.OnSecondaryTap() })
		}
	case TertiaryButton:
		if r.OnTertiaryTapUp != nil {
			r.InvokeCallback("onTertiaryTapUp", func() { r.OnTertiaryTapUp(details) })
		}
	}
}

// HandleTapMove calls OnTapMove for a move with the primary button.
//
// Flutter: gestures/tap.dart TapGestureRecognizer.handleTapMove
func (r *TapGestureRecognizer) HandleTapMove(move *PointerMoveEvent) {
	if r.OnTapMove != nil && move.Buttons() == PrimaryButton {
		details := NewTapMoveDetails(r.GetKindForPointer(move.Pointer()), TapMoveOpts{
			GlobalPosition: move.Position(), LocalPosition: foundation.Some(move.LocalPosition()), Delta: move.Delta(),
		})
		r.InvokeCallback("onTapMove", func() { r.OnTapMove(details) })
	}
}

// HandleTapCancel calls the tap cancel callback of the button.
//
// Flutter: gestures/tap.dart TapGestureRecognizer.handleTapCancel
func (r *TapGestureRecognizer) HandleTapCancel(down *PointerDownEvent, _ *PointerCancelEvent, reason string) {
	note := reason
	if reason != "" {
		note = reason + " "
	}
	switch down.Buttons() {
	case PrimaryButton:
		if r.OnTapCancel != nil {
			r.InvokeCallback(note+"onTapCancel", r.OnTapCancel)
		}
	case SecondaryButton:
		if r.OnSecondaryTapCancel != nil {
			r.InvokeCallback(note+"onSecondaryTapCancel", r.OnSecondaryTapCancel)
		}
	case TertiaryButton:
		if r.OnTertiaryTapCancel != nil {
			r.InvokeCallback(note+"onTertiaryTapCancel", r.OnTertiaryTapCancel)
		}
	}
}

// DebugDescription is "tap".
//
// Flutter: gestures/tap.dart TapGestureRecognizer.debugDescription
func (r *TapGestureRecognizer) DebugDescription() string { return "tap" }

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagTapCancelWithoutDown   = "TapCancelWithoutDown"
	tagTapTrackingWithoutDown = "TapTrackingWithoutDown"
)
