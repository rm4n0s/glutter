package gestures

import (
	"math"
	"slices"
	"time"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// DefaultMouseScrollToScaleFactor is how many pixels of mouse scroll make a
// scale by a factor of e, when the scroll causes a scale.
//
// Flutter: gestures/scale.dart kDefaultMouseScrollToScaleFactor
const DefaultMouseScrollToScaleFactor = 200.0

// DefaultTrackpadScrollToScaleFactor is how the trackpad scroll turns into a
// scale, when the scroll causes a scale.
//
// Flutter: gestures/scale.dart kDefaultTrackpadScrollToScaleFactor
var DefaultTrackpadScrollToScaleFactor = ui.Offset{Dx: 0, Dy: -1 / DefaultMouseScrollToScaleFactor}

type scaleState int

const (
	scaleReady scaleState = iota
	scalePossible
	scaleAccepted
	scaleStarted
)

type pointerPanZoomData struct {
	parent   *ScaleGestureRecognizer
	position ui.Offset
	pan      ui.Offset
	scale    float64
	rotation float64
}

func (d pointerPanZoomData) focalPoint() ui.Offset {
	if d.parent.TrackpadScrollCausesScale {
		return d.position
	}
	return d.position.Add(d.pan)
}

func (d pointerPanZoomData) scaleFactor() float64 {
	if d.parent.TrackpadScrollCausesScale {
		f := d.parent.TrackpadScrollToScaleFactor
		return d.scale * math.Exp(d.pan.Dx*f.Dx+d.pan.Dy*f.Dy)
	}
	return d.scale
}

// ScaleStartDetails is the data of a scale gesture that started.
//
// Flutter: gestures/scale.dart ScaleStartDetails
type ScaleStartDetails struct {
	// FocalPoint is the initial focal point of the pointers, in global
	// coordinates.
	FocalPoint ui.Offset
	// LocalFocalPoint is FocalPoint in the coordinates of the target.
	LocalFocalPoint ui.Offset
	// PointerCount is the number of pointers.
	PointerCount int
	// SourceTimeStamp is the time of the event that started the gesture.
	SourceTimeStamp foundation.Opt[time.Duration]
	// Kind is the kind of device of the first pointer.
	Kind foundation.Opt[ui.PointerDeviceKind]
}

// ScaleStartOpts are the optional parameters of NewScaleStartDetails.
//
// Flutter: gestures/scale.dart ScaleStartDetails.new
type ScaleStartOpts struct {
	// FocalPoint is the focal point of the pointers.
	FocalPoint ui.Offset
	// LocalFocalPoint is the focal point in local coordinates; the default is
	// FocalPoint.
	LocalFocalPoint foundation.Opt[ui.Offset]
	// PointerCount is the number of pointers.
	PointerCount int
	// SourceTimeStamp is the time of the event.
	SourceTimeStamp foundation.Opt[time.Duration]
	// Kind is the kind of device.
	Kind foundation.Opt[ui.PointerDeviceKind]
}

// NewScaleStartDetails creates the details.
//
// Flutter: gestures/scale.dart ScaleStartDetails.new
func NewScaleStartDetails(opts ...ScaleStartOpts) ScaleStartDetails {
	o := oneOpts(opts)
	return ScaleStartDetails{
		FocalPoint: o.FocalPoint, LocalFocalPoint: localOrGlobal(o.LocalFocalPoint, o.FocalPoint),
		PointerCount: o.PointerCount, SourceTimeStamp: o.SourceTimeStamp, Kind: o.Kind,
	}
}

// DebugFillProperties adds the properties of the details.
//
// Flutter: gestures/scale.dart ScaleStartDetails.debugFillProperties
func (d ScaleStartDetails) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("focalPoint", d.FocalPoint))
	p.Add(foundation.NewDiagnosticsProperty("localFocalPoint", d.LocalFocalPoint))
	p.Add(foundation.NewIntProperty("pointerCount", foundation.Some(d.PointerCount)))
	p.Add(foundation.NewDiagnosticsPropertyOpt("sourceTimeStamp", d.SourceTimeStamp))
}

// ScaleUpdateDetails is the data of a scale gesture that changed.
//
// Flutter: gestures/scale.dart ScaleUpdateDetails
type ScaleUpdateDetails struct {
	// FocalPointDelta is how far the focal point moved since the last update.
	FocalPointDelta ui.Offset
	// FocalPoint is the focal point of the pointers, in global coordinates.
	FocalPoint ui.Offset
	// LocalFocalPoint is FocalPoint in the coordinates of the target.
	LocalFocalPoint ui.Offset
	// Scale is the scale since the gesture started.
	Scale float64
	// HorizontalScale is the horizontal scale since the gesture started.
	HorizontalScale float64
	// VerticalScale is the vertical scale since the gesture started.
	VerticalScale float64
	// Rotation is the rotation since the gesture started, in radians.
	Rotation float64
	// PointerCount is the number of pointers.
	PointerCount int
	// SourceTimeStamp is the time of the event.
	SourceTimeStamp foundation.Opt[time.Duration]
}

// ScaleUpdateOpts are the optional parameters of NewScaleUpdateDetails.
//
// Flutter: gestures/scale.dart ScaleUpdateDetails.new
type ScaleUpdateOpts struct {
	// FocalPoint is the focal point of the pointers.
	FocalPoint ui.Offset
	// LocalFocalPoint is the focal point in local coordinates; the default is
	// FocalPoint.
	LocalFocalPoint foundation.Opt[ui.Offset]
	// Scale is the scale; the default is 1.
	Scale foundation.Opt[float64]
	// HorizontalScale is the horizontal scale; the default is 1.
	HorizontalScale foundation.Opt[float64]
	// VerticalScale is the vertical scale; the default is 1.
	VerticalScale foundation.Opt[float64]
	// Rotation is the rotation in radians.
	Rotation float64
	// PointerCount is the number of pointers.
	PointerCount int
	// FocalPointDelta is how far the focal point moved.
	FocalPointDelta ui.Offset
	// SourceTimeStamp is the time of the event.
	SourceTimeStamp foundation.Opt[time.Duration]
}

// NewScaleUpdateDetails creates the details. A negative scale panics with the
// tag ScaleNegative in debug builds.
//
// Flutter: gestures/scale.dart ScaleUpdateDetails.new
func NewScaleUpdateDetails(opts ...ScaleUpdateOpts) ScaleUpdateDetails {
	o := oneOpts(opts)
	scale, h, v := o.Scale.OrElse(1), o.HorizontalScale.OrElse(1), o.VerticalScale.OrElse(1)
	if foundation.DebugMode && (scale < 0 || h < 0 || v < 0) {
		panic(errors.New("ScaleNegative", "the scales must not be negative", "scale", scale, "horizontal", h, "vertical", v))
	}
	return ScaleUpdateDetails{
		FocalPointDelta: o.FocalPointDelta, FocalPoint: o.FocalPoint,
		LocalFocalPoint: localOrGlobal(o.LocalFocalPoint, o.FocalPoint),
		Scale:           scale, HorizontalScale: h, VerticalScale: v,
		Rotation: o.Rotation, PointerCount: o.PointerCount, SourceTimeStamp: o.SourceTimeStamp,
	}
}

// DebugFillProperties adds the properties of the details.
//
// Flutter: gestures/scale.dart ScaleUpdateDetails.debugFillProperties
func (d ScaleUpdateDetails) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("focalPointDelta", d.FocalPointDelta))
	p.Add(foundation.NewDiagnosticsProperty("focalPoint", d.FocalPoint))
	p.Add(foundation.NewDiagnosticsProperty("localFocalPoint", d.LocalFocalPoint))
	p.Add(foundation.NewDoubleProperty("scale", foundation.Some(d.Scale)))
	p.Add(foundation.NewDoubleProperty("horizontalScale", foundation.Some(d.HorizontalScale)))
	p.Add(foundation.NewDoubleProperty("verticalScale", foundation.Some(d.VerticalScale)))
	p.Add(foundation.NewDoubleProperty("rotation", foundation.Some(d.Rotation)))
	p.Add(foundation.NewIntProperty("pointerCount", foundation.Some(d.PointerCount)))
	p.Add(foundation.NewDiagnosticsPropertyOpt("sourceTimeStamp", d.SourceTimeStamp))
}

// ScaleEndDetails is the data of a scale gesture that ended.
//
// Flutter: gestures/scale.dart ScaleEndDetails
type ScaleEndDetails struct {
	// Velocity is the velocity of the last pointers.
	Velocity Velocity
	// ScaleVelocity is how fast the scale changed, in scale per second; -1 if
	// it is not known.
	ScaleVelocity float64
	// PointerCount is the number of pointers.
	PointerCount int
}

// DebugFillProperties adds the properties of the details.
//
// Flutter: gestures/scale.dart ScaleEndDetails.debugFillProperties
func (d ScaleEndDetails) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("velocity", d.Velocity))
	p.Add(foundation.NewDoubleProperty("scaleVelocity", foundation.Some(d.ScaleVelocity)))
	p.Add(foundation.NewIntProperty("pointerCount", foundation.Some(d.PointerCount)))
}

// GestureScaleStartCallback is called when a scale gesture starts.
//
// Flutter: gestures/scale.dart GestureScaleStartCallback
type GestureScaleStartCallback func(details ScaleStartDetails)

// GestureScaleUpdateCallback is called when a scale gesture changes.
//
// Flutter: gestures/scale.dart GestureScaleUpdateCallback
type GestureScaleUpdateCallback func(details ScaleUpdateDetails)

// GestureScaleEndCallback is called when a scale gesture ends.
//
// Flutter: gestures/scale.dart GestureScaleEndCallback
type GestureScaleEndCallback func(details ScaleEndDetails)

func isFlingGesture(velocity Velocity) bool {
	return velocity.PixelsPerSecond.DistanceSquared() > MinFlingVelocity*MinFlingVelocity
}

type lineBetweenPointers struct {
	pointerStartLocation ui.Offset
	pointerStartID       int
	pointerEndLocation   ui.Offset
	pointerEndID         int
}

// ScaleGestureRecognizerOpts are the optional parameters of
// NewScaleGestureRecognizer.
//
// Flutter: gestures/scale.dart ScaleGestureRecognizer.new
type ScaleGestureRecognizerOpts struct {
	// DebugOwner is the object that owns the recognizer.
	DebugOwner any
	// SupportedDevices are the kinds of pointer the recognizer accepts; nil for
	// all.
	SupportedDevices []ui.PointerDeviceKind
	// AllowedButtonsFilter says whether the buttons of a pointer are allowed.
	AllowedButtonsFilter AllowedButtonsFilter
	// DragStartBehavior is where the gesture starts; the default is
	// DragStartBehaviorDown.
	DragStartBehavior DragStartBehavior
	// TrackpadScrollCausesScale makes a trackpad scroll scale.
	TrackpadScrollCausesScale bool
	// TrackpadScrollToScaleFactor is how a trackpad scroll turns into a scale;
	// the default is DefaultTrackpadScrollToScaleFactor.
	TrackpadScrollToScaleFactor foundation.Opt[ui.Offset]
}

// ScaleGestureRecognizer recognizes a scale gesture: one or more pointers that
// move, spread, close or rotate.
//
// Flutter: gestures/scale.dart ScaleGestureRecognizer
type ScaleGestureRecognizer struct {
	OneSequenceGestureRecognizer

	// DragStartBehavior is where the gesture starts.
	//
	// Flutter: gestures/scale.dart ScaleGestureRecognizer.dragStartBehavior
	DragStartBehavior DragStartBehavior
	// OnStart is called when the pointers start a scale.
	//
	// Flutter: gestures/scale.dart ScaleGestureRecognizer.onStart
	OnStart GestureScaleStartCallback
	// OnUpdate is called when the scale changes.
	//
	// Flutter: gestures/scale.dart ScaleGestureRecognizer.onUpdate
	OnUpdate GestureScaleUpdateCallback
	// OnEnd is called when the pointers end the scale, or their number changes.
	//
	// Flutter: gestures/scale.dart ScaleGestureRecognizer.onEnd
	OnEnd GestureScaleEndCallback
	// TrackpadScrollCausesScale makes a trackpad scroll scale.
	//
	// Flutter: gestures/scale.dart ScaleGestureRecognizer.trackpadScrollCausesScale
	TrackpadScrollCausesScale bool
	// TrackpadScrollToScaleFactor is how a trackpad scroll turns into a scale.
	//
	// Flutter: gestures/scale.dart ScaleGestureRecognizer.trackpadScrollToScaleFactor
	TrackpadScrollToScaleFactor ui.Offset

	state         scaleState
	lastTransform *ui.Matrix4

	initialFocalPoint     ui.Offset
	currentFocalPoint     foundation.Opt[ui.Offset]
	initialSpan           float64
	currentSpan           float64
	initialHorizontalSpan float64
	currentHorizontalSpan float64
	initialVerticalSpan   float64
	currentVerticalSpan   float64
	localFocalPoint       ui.Offset
	initialLine           *lineBetweenPointers
	currentLine           *lineBetweenPointers

	pointerLocations orderedMap[ui.Offset]
	// pointerQueue sorts the pointers in order of entrance.
	pointerQueue         []int
	velocityTrackers     map[int]*VelocityTracker
	scaleVelocityTracker *VelocityTracker
	delta                ui.Offset

	pointerPanZooms            orderedMap[pointerPanZoomData]
	initialPanZoomScaleFactor  float64
	initialPanZoomRotationFact float64
	initialEventTimestamp      foundation.Opt[time.Duration]
}

// NewScaleGestureRecognizer creates a recognizer that works with the binding.
//
// Flutter: gestures/scale.dart ScaleGestureRecognizer.new
func NewScaleGestureRecognizer(binding *GestureBinding, opts ...ScaleGestureRecognizerOpts) *ScaleGestureRecognizer {
	o := oneOpts(opts)
	r := &ScaleGestureRecognizer{
		DragStartBehavior:           o.DragStartBehavior,
		TrackpadScrollCausesScale:   o.TrackpadScrollCausesScale,
		TrackpadScrollToScaleFactor: o.TrackpadScrollToScaleFactor.OrElse(DefaultTrackpadScrollToScaleFactor),
		velocityTrackers:            map[int]*VelocityTracker{},
		initialPanZoomScaleFactor:   1,
	}
	if r.DragStartBehavior == DragStartBehaviorUnspecified {
		r.DragStartBehavior = DragStartBehaviorDown
	}
	r.InitOneSequence(binding, r, GestureRecognizerOpts{DebugOwner: o.DebugOwner, SupportedDevices: o.SupportedDevices, AllowedButtonsFilter: o.AllowedButtonsFilter})
	return r
}

// PointerCount is the number of pointers: two for each trackpad gesture and one
// for each other pointer.
//
// Flutter: gestures/scale.dart ScaleGestureRecognizer.pointerCount
func (r *ScaleGestureRecognizer) PointerCount() int {
	return 2*r.pointerPanZooms.len() + len(r.pointerQueue)
}

func (r *ScaleGestureRecognizer) pointerScaleFactor() float64 {
	if r.initialSpan > 0 {
		return r.currentSpan / r.initialSpan
	}
	return 1
}

func (r *ScaleGestureRecognizer) pointerHorizontalScaleFactor() float64 {
	if r.initialHorizontalSpan > 0 {
		return r.currentHorizontalSpan / r.initialHorizontalSpan
	}
	return 1
}

func (r *ScaleGestureRecognizer) pointerVerticalScaleFactor() float64 {
	if r.initialVerticalSpan > 0 {
		return r.currentVerticalSpan / r.initialVerticalSpan
	}
	return 1
}

func (r *ScaleGestureRecognizer) withPanZoomScale(scale float64) float64 {
	for _, p := range r.pointerPanZooms.values() {
		scale *= p.scaleFactor() / r.initialPanZoomScaleFactor
	}
	return scale
}

func (r *ScaleGestureRecognizer) scaleFactor() float64 {
	return r.withPanZoomScale(r.pointerScaleFactor())
}

func (r *ScaleGestureRecognizer) horizontalScaleFactor() float64 {
	return r.withPanZoomScale(r.pointerHorizontalScaleFactor())
}

func (r *ScaleGestureRecognizer) verticalScaleFactor() float64 {
	return r.withPanZoomScale(r.pointerVerticalScaleFactor())
}

func (r *ScaleGestureRecognizer) computeRotationFactor() float64 {
	factor := 0.0
	if r.initialLine != nil && r.currentLine != nil {
		fx, fy := r.initialLine.pointerStartLocation.Dx, r.initialLine.pointerStartLocation.Dy
		sx, sy := r.initialLine.pointerEndLocation.Dx, r.initialLine.pointerEndLocation.Dy
		nfx, nfy := r.currentLine.pointerStartLocation.Dx, r.currentLine.pointerStartLocation.Dy
		nsx, nsy := r.currentLine.pointerEndLocation.Dx, r.currentLine.pointerEndLocation.Dy
		angle1 := math.Atan2(fy-sy, fx-sx)
		angle2 := math.Atan2(nfy-nsy, nfx-nsx)
		factor = angle2 - angle1
	}
	for _, p := range r.pointerPanZooms.values() {
		factor += p.rotation
	}
	factor -= r.initialPanZoomRotationFact
	return factor
}

// AddAllowedPointer tracks the pointer.
//
// Flutter: gestures/scale.dart ScaleGestureRecognizer.addAllowedPointer
func (r *ScaleGestureRecognizer) AddAllowedPointer(event *PointerDownEvent) {
	r.OneSequenceGestureRecognizer.AddAllowedPointer(event)
	r.velocityTrackers[event.Pointer()] = NewVelocityTrackerWithKind(r.binding.Clock(), event.Kind())
	r.initialEventTimestamp = foundation.Some(event.TimeStamp())
	if r.state == scaleReady {
		r.state = scalePossible
		r.initialSpan, r.currentSpan = 0, 0
		r.initialHorizontalSpan, r.currentHorizontalSpan = 0, 0
		r.initialVerticalSpan, r.currentVerticalSpan = 0, 0
	}
}

// IsPointerPanZoomAllowed allows every trackpad gesture.
//
// Flutter: gestures/scale.dart ScaleGestureRecognizer.isPointerPanZoomAllowed
func (r *ScaleGestureRecognizer) IsPointerPanZoomAllowed(*PointerPanZoomStartEvent) bool { return true }

// AddAllowedPointerPanZoom tracks the trackpad gesture.
//
// Flutter: gestures/scale.dart ScaleGestureRecognizer.addAllowedPointerPanZoom
func (r *ScaleGestureRecognizer) AddAllowedPointerPanZoom(event *PointerPanZoomStartEvent) {
	r.OneSequenceGestureRecognizer.AddAllowedPointerPanZoom(event)
	r.oneSeq.StartTrackingPointer(event.Pointer(), event.Transform())
	r.velocityTrackers[event.Pointer()] = NewVelocityTrackerWithKind(r.binding.Clock(), event.Kind())
	r.initialEventTimestamp = foundation.Some(event.TimeStamp())
	if r.state == scaleReady {
		r.state = scalePossible
		r.initialPanZoomScaleFactor = 1
		r.initialPanZoomRotationFact = 0
	}
}

// HandleEvent follows the pointers.
//
// Flutter: gestures/scale.dart ScaleGestureRecognizer.handleEvent
func (r *ScaleGestureRecognizer) HandleEvent(event PointerEvent) {
	if foundation.DebugMode && r.state == scaleReady {
		panic(errors.New("ScaleEventWhileReady", "an event arrived while the scale recognizer is ready"))
	}
	didChangeConfiguration := false
	shouldStartIfAccepted := false
	switch e := event.(type) {
	case *PointerMoveEvent:
		tracker := r.velocityTrackers[e.Pointer()]
		if !e.Synthesized() {
			tracker.AddPosition(e.TimeStamp(), e.Position())
		}
		r.pointerLocations.set(e.Pointer(), e.Position())
		shouldStartIfAccepted = true
		r.lastTransform = e.Transform()
	case *PointerDownEvent:
		r.pointerLocations.set(e.Pointer(), e.Position())
		r.pointerQueue = append(r.pointerQueue, e.Pointer())
		didChangeConfiguration = true
		shouldStartIfAccepted = true
		r.lastTransform = e.Transform()
	case *PointerUpEvent, *PointerCancelEvent:
		r.pointerLocations.remove(event.Pointer())
		r.pointerQueue = slices.DeleteFunc(r.pointerQueue, func(p int) bool { return p == event.Pointer() })
		didChangeConfiguration = true
		r.lastTransform = event.Transform()
	case *PointerPanZoomStartEvent:
		r.pointerPanZooms.set(e.Pointer(), pointerPanZoomData{parent: r, position: e.Position(), scale: 1})
		didChangeConfiguration = true
		shouldStartIfAccepted = true
		r.lastTransform = e.Transform()
	case *PointerPanZoomUpdateEvent:
		if !e.Synthesized() && !r.TrackpadScrollCausesScale {
			r.velocityTrackers[e.Pointer()].AddPosition(e.TimeStamp(), e.Pan())
		}
		r.pointerPanZooms.set(e.Pointer(), pointerPanZoomData{parent: r, position: e.Position(), pan: e.Pan(), scale: e.Scale(), rotation: e.Rotation()})
		r.lastTransform = e.Transform()
		shouldStartIfAccepted = true
	case *PointerPanZoomEndEvent:
		r.pointerPanZooms.remove(e.Pointer())
		didChangeConfiguration = true
	}
	r.updateLines()
	r.update()
	if !didChangeConfiguration || r.reconfigure(event.Pointer()) {
		r.advanceStateMachine(shouldStartIfAccepted, event)
	}
	r.StopTrackingIfPointerNoLongerDown(event)
}

func (r *ScaleGestureRecognizer) update() {
	previousFocalPoint, hadPrevious := r.currentFocalPoint.Get()
	focalPoint := ui.Offset{}
	for _, k := range r.pointerLocations.keys {
		focalPoint = focalPoint.Add(r.pointerLocations.m[k])
	}
	for _, p := range r.pointerPanZooms.values() {
		focalPoint = focalPoint.Add(p.focalPoint())
	}
	current := focalPoint.Div(math.Max(1, float64(r.pointerLocations.len()+r.pointerPanZooms.len())))
	r.currentFocalPoint = foundation.Some(current)
	_ = previousFocalPoint
	if !hadPrevious {
		r.localFocalPoint = TransformPosition(r.lastTransform, current)
		r.delta = ui.Offset{}
	} else {
		localPrevious := r.localFocalPoint
		r.localFocalPoint = TransformPosition(r.lastTransform, current)
		r.delta = r.localFocalPoint.Sub(localPrevious)
	}

	count := r.pointerLocations.len()
	pointerFocalPoint := ui.Offset{}
	for _, k := range r.pointerLocations.keys {
		pointerFocalPoint = pointerFocalPoint.Add(r.pointerLocations.m[k])
	}
	if count > 0 {
		pointerFocalPoint = pointerFocalPoint.Div(float64(count))
	}
	totalDeviation, totalHorizontalDeviation, totalVerticalDeviation := 0.0, 0.0, 0.0
	for _, k := range r.pointerLocations.keys {
		loc := r.pointerLocations.m[k]
		totalDeviation += pointerFocalPoint.Sub(loc).Distance()
		totalHorizontalDeviation += math.Abs(pointerFocalPoint.Dx - loc.Dx)
		totalVerticalDeviation += math.Abs(pointerFocalPoint.Dy - loc.Dy)
	}
	r.currentSpan, r.currentHorizontalSpan, r.currentVerticalSpan = 0, 0, 0
	if count > 0 {
		r.currentSpan = totalDeviation / float64(count)
		r.currentHorizontalSpan = totalHorizontalDeviation / float64(count)
		r.currentVerticalSpan = totalVerticalDeviation / float64(count)
	}
}

func (r *ScaleGestureRecognizer) updateLines() {
	count := r.pointerLocations.len()
	if foundation.DebugMode && len(r.pointerQueue) < count {
		panic(errors.New("ScalePointerQueueShort", "the pointer queue is shorter than the pointers"))
	}
	if count < 2 {
		r.initialLine = r.currentLine
		return
	}
	first, second := r.pointerQueue[0], r.pointerQueue[1]
	line := &lineBetweenPointers{
		pointerStartID: first, pointerStartLocation: r.pointerLocations.m[first],
		pointerEndID: second, pointerEndLocation: r.pointerLocations.m[second],
	}
	if r.initialLine != nil && r.initialLine.pointerStartID == first && r.initialLine.pointerEndID == second {
		r.currentLine = line
	} else {
		r.initialLine = line
		r.currentLine = r.initialLine
	}
}

func (r *ScaleGestureRecognizer) resetInitialPanZoom() {
	if r.pointerPanZooms.len() == 0 {
		r.initialPanZoomScaleFactor = 1
		r.initialPanZoomRotationFact = 0
		return
	}
	r.initialPanZoomScaleFactor = r.scaleFactor() / r.pointerScaleFactor()
	sum := 0.0
	for i, p := range r.pointerPanZooms.values() {
		if i == 0 {
			sum = p.rotation
		} else {
			sum += p.rotation
		}
	}
	r.initialPanZoomRotationFact = sum
}

func (r *ScaleGestureRecognizer) reconfigure(pointer int) bool {
	r.initialFocalPoint = r.currentFocalPoint.Value()
	r.initialSpan = r.currentSpan
	r.initialLine = r.currentLine
	r.initialHorizontalSpan = r.currentHorizontalSpan
	r.initialVerticalSpan = r.currentVerticalSpan
	r.resetInitialPanZoom()
	// arbitrary PointerDeviceKind
	newScaleTracker := func() *VelocityTracker {
		return NewVelocityTrackerWithKind(r.binding.Clock(), ui.PointerDeviceKindTouch)
	}
	if r.state == scaleStarted {
		if r.OnEnd != nil {
			tracker := r.velocityTrackers[pointer]
			velocity := tracker.GetVelocity()
			scaleVelocity := -1.0
			if r.scaleVelocityTracker != nil {
				scaleVelocity = r.scaleVelocityTracker.GetVelocity().PixelsPerSecond.Dx
			}
			count := r.PointerCount()
			if isFlingGesture(velocity) {
				pps := velocity.PixelsPerSecond
				if pps.DistanceSquared() > MaxFlingVelocity*MaxFlingVelocity {
					velocity = Velocity{pps.Div(pps.Distance()).Mul(MaxFlingVelocity)}
				}
				r.InvokeCallback("onEnd", func() {
					r.OnEnd(ScaleEndDetails{Velocity: velocity, ScaleVelocity: scaleVelocity, PointerCount: count})
				})
			} else {
				r.InvokeCallback("onEnd", func() {
					r.OnEnd(ScaleEndDetails{ScaleVelocity: scaleVelocity, PointerCount: count})
				})
			}
		}
		r.state = scaleAccepted
		r.scaleVelocityTracker = newScaleTracker()
		return false
	}
	r.scaleVelocityTracker = newScaleTracker()
	return true
}

func (r *ScaleGestureRecognizer) advanceStateMachine(shouldStartIfAccepted bool, event PointerEvent) {
	if r.state == scaleReady {
		r.state = scalePossible
	}
	if r.state == scalePossible {
		spanDelta := math.Abs(r.currentSpan - r.initialSpan)
		focalPointDelta := r.currentFocalPoint.Value().Sub(r.initialFocalPoint).Distance()
		if spanDelta > ComputeScaleSlop(event.Kind()) ||
			focalPointDelta > ComputePanSlop(event.Kind(), r.GestureSettings) ||
			math.Max(r.scaleFactor()/r.pointerScaleFactor(), r.pointerScaleFactor()/r.scaleFactor()) > 1.05 {
			r.oneSeq.Resolve(GestureAccepted)
		}
	} else if r.state >= scaleAccepted {
		r.oneSeq.Resolve(GestureAccepted)
	}
	if r.state == scaleAccepted && shouldStartIfAccepted {
		r.initialEventTimestamp = foundation.Some(event.TimeStamp())
		r.state = scaleStarted
		r.dispatchOnStartCallbackIfNeeded()
	}
	if r.state == scaleStarted {
		if r.scaleVelocityTracker != nil {
			r.scaleVelocityTracker.AddPosition(event.TimeStamp(), ui.Offset{Dx: r.scaleFactor()})
		}
		if r.OnUpdate != nil {
			r.InvokeCallback("onUpdate", func() {
				r.OnUpdate(NewScaleUpdateDetails(ScaleUpdateOpts{
					Scale: foundation.Some(r.scaleFactor()), HorizontalScale: foundation.Some(r.horizontalScaleFactor()),
					VerticalScale: foundation.Some(r.verticalScaleFactor()), FocalPoint: r.currentFocalPoint.Value(),
					LocalFocalPoint: foundation.Some(r.localFocalPoint), Rotation: r.computeRotationFactor(),
					PointerCount: r.PointerCount(), FocalPointDelta: r.delta, SourceTimeStamp: foundation.Some(event.TimeStamp()),
				}))
			})
		}
	}
}

func (r *ScaleGestureRecognizer) dispatchOnStartCallbackIfNeeded() {
	if r.OnStart != nil {
		r.InvokeCallback("onStart", func() {
			kind := foundation.None[ui.PointerDeviceKind]()
			if len(r.pointerQueue) > 0 {
				kind = foundation.Some(r.GetKindForPointer(r.pointerQueue[0]))
			} else if r.pointerPanZooms.len() > 0 {
				kind = foundation.Some(r.GetKindForPointer(r.pointerPanZooms.keys[0]))
			}
			r.OnStart(NewScaleStartDetails(ScaleStartOpts{
				FocalPoint: r.currentFocalPoint.Value(), LocalFocalPoint: foundation.Some(r.localFocalPoint),
				PointerCount: r.PointerCount(), SourceTimeStamp: r.initialEventTimestamp, Kind: kind,
			}))
		})
	}
	r.initialEventTimestamp = foundation.None[time.Duration]()
}

// AcceptGesture starts the scale gesture.
//
// Flutter: gestures/scale.dart ScaleGestureRecognizer.acceptGesture
func (r *ScaleGestureRecognizer) AcceptGesture(int) {
	if r.state == scalePossible {
		r.state = scaleStarted
		r.dispatchOnStartCallbackIfNeeded()
		if r.DragStartBehavior == DragStartBehaviorStart {
			r.initialFocalPoint = r.currentFocalPoint.Value()
			r.initialSpan = r.currentSpan
			r.initialLine = r.currentLine
			r.initialHorizontalSpan = r.currentHorizontalSpan
			r.initialVerticalSpan = r.currentVerticalSpan
			r.resetInitialPanZoom()
		}
	}
}

// RejectGesture forgets the pointer.
//
// Flutter: gestures/scale.dart ScaleGestureRecognizer.rejectGesture
func (r *ScaleGestureRecognizer) RejectGesture(pointer int) {
	r.pointerPanZooms.remove(pointer)
	r.pointerLocations.remove(pointer)
	r.pointerQueue = slices.DeleteFunc(r.pointerQueue, func(p int) bool { return p == pointer })
	r.StopTrackingPointer(pointer)
}

// DidStopTrackingLastPointer makes the recognizer ready again.
//
// Flutter: gestures/scale.dart ScaleGestureRecognizer.didStopTrackingLastPointer
func (r *ScaleGestureRecognizer) DidStopTrackingLastPointer(int) {
	switch r.state {
	case scalePossible:
		r.oneSeq.Resolve(GestureRejected)
	case scaleReady, scaleStarted:
		if foundation.DebugMode {
			panic(errors.New("ScaleStoppedInInvalidState", "the last pointer stopped while the scale was ready or started"))
		}
	}
	r.state = scaleReady
}

// Dispose releases the velocity trackers.
//
// Flutter: gestures/scale.dart ScaleGestureRecognizer.dispose
func (r *ScaleGestureRecognizer) Dispose() {
	r.velocityTrackers = map[int]*VelocityTracker{}
	r.OneSequenceGestureRecognizer.Dispose()
}

// DebugDescription is "scale".
//
// Flutter: gestures/scale.dart ScaleGestureRecognizer.debugDescription
func (r *ScaleGestureRecognizer) DebugDescription() string { return "scale" }
