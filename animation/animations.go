package animation

import (
	"fmt"
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// constAnimation is an animation whose value and status never change, so its
// listeners are never called.
type constAnimation[T any] struct {
	value  T
	status AnimationStatus
	name   string
}

func (a *constAnimation[T]) AddListener(foundation.VoidCallback) foundation.ListenerHandle {
	return foundation.ListenerHandle{}
}
func (a *constAnimation[T]) RemoveListener(foundation.ListenerHandle) {}
func (a *constAnimation[T]) AddStatusListener(AnimationStatusListener) StatusListenerHandle {
	return StatusListenerHandle{}
}
func (a *constAnimation[T]) RemoveStatusListener(StatusListenerHandle) {}
func (a *constAnimation[T]) Status() AnimationStatus                   { return a.status }
func (a *constAnimation[T]) Value() T                                  { return a.value }
func (a *constAnimation[T]) String() string {
	if a.name != "" {
		return a.name
	}
	return fmt.Sprintf("%s %v; paused", DescribeAnimation[T](a), a.value)
}

// AlwaysComplete is an animation that is always complete (value 1, status
// completed). Using it involves less overhead than an AnimationController with
// an initial value of 1. It is for APIs that expect an animation when nothing
// should animate.
//
// Flutter: animation/animations.dart kAlwaysCompleteAnimation
var AlwaysComplete Animation[float64] = &constAnimation[float64]{1, AnimationStatusCompleted, "kAlwaysCompleteAnimation"}

// AlwaysDismissed is an animation that is always dismissed (value 0, status
// dismissed).
//
// Flutter: animation/animations.dart kAlwaysDismissedAnimation
var AlwaysDismissed Animation[float64] = &constAnimation[float64]{0, AnimationStatusDismissed, "kAlwaysDismissedAnimation"}

// NewAlwaysStoppedAnimation creates an animation that is always stopped at the
// value; its status is always AnimationStatusForward. Its value and status can
// never change, so its listeners are never called and it is safe to reuse it in
// many places.
//
// Flutter: animation/animations.dart AlwaysStoppedAnimation.new
func NewAlwaysStoppedAnimation[T any](value T) Animation[T] {
	return &constAnimation[T]{value: value, status: AnimationStatusForward}
}

// ProxyAnimation is an animation that is a proxy for another animation: the
// parent can change after the proxy was handed to someone else.
//
// Flutter: animation/animations.dart ProxyAnimation
type ProxyAnimation struct {
	LazyListeners
	LocalListeners
	LocalStatusListeners

	parent         Animation[float64]
	status         AnimationStatus
	value          float64
	valueHandle    foundation.ListenerHandle
	statusHandle   StatusListenerHandle
	parentAttached bool
}

// NewProxyAnimation creates a proxy animation. Without a parent (nil) the proxy
// has the status AnimationStatusDismissed and the value 0.
//
// Flutter: animation/animations.dart ProxyAnimation.new
func NewProxyAnimation(parent Animation[float64]) *ProxyAnimation {
	p := &ProxyAnimation{parent: parent}
	if parent == nil {
		p.status = AnimationStatusDismissed
	}
	p.wire()
	return p
}

func (p *ProxyAnimation) wire() {
	p.OnStartListening = p.didStartListening
	p.OnStopListening = p.didStopListening
	p.LocalListeners.OnRegister = p.DidRegisterListener
	p.LocalListeners.OnUnregister = p.DidUnregisterListener
	p.LocalStatusListeners.OnRegister = p.DidRegisterListener
	p.LocalStatusListeners.OnUnregister = p.DidUnregisterListener
}

func (p *ProxyAnimation) didStartListening() {
	if p.parent != nil {
		p.valueHandle = p.parent.AddListener(p.NotifyListeners)
		p.statusHandle = p.parent.AddStatusListener(p.NotifyStatusListeners)
		p.parentAttached = true
	}
}

func (p *ProxyAnimation) didStopListening() {
	if p.parent != nil && p.parentAttached {
		p.parent.RemoveListener(p.valueHandle)
		p.parent.RemoveStatusListener(p.statusHandle)
		p.parentAttached = false
	}
}

// Parent is the animation whose value this animation proxies.
//
// Flutter: animation/animations.dart ProxyAnimation.parent
func (p *ProxyAnimation) Parent() Animation[float64] { return p.parent }

// SetParent changes the animation to proxy. The listeners of the proxy are moved
// to the new parent, and called when the value or the status differ.
//
// Flutter: animation/animations.dart ProxyAnimation.parent
func (p *ProxyAnimation) SetParent(value Animation[float64]) {
	if value == p.parent {
		return
	}
	if p.parent != nil {
		p.status = p.parent.Status()
		p.value = p.parent.Value()
		if p.IsListening() {
			p.didStopListening()
		}
	}
	p.parent = value
	if p.parent != nil {
		if p.IsListening() {
			p.didStartListening()
		}
		if p.value != p.parent.Value() {
			p.NotifyListeners()
		}
		if p.status != p.parent.Status() {
			p.NotifyStatusListeners(p.parent.Status())
		}
		p.status = AnimationStatusUnspecified
		p.value = 0
	}
}

// Status is the status of the parent.
//
// Flutter: animation/animations.dart ProxyAnimation.status
func (p *ProxyAnimation) Status() AnimationStatus {
	if p.parent != nil {
		return p.parent.Status()
	}
	return p.status
}

// Value is the value of the parent.
//
// Flutter: animation/animations.dart ProxyAnimation.value
func (p *ProxyAnimation) Value() float64 {
	if p.parent != nil {
		return p.parent.Value()
	}
	return p.value
}

// String describes the animation.
//
// Flutter: animation/animations.dart ProxyAnimation.toString
func (p *ProxyAnimation) String() string {
	if p.parent == nil {
		return fmt.Sprintf("ProxyAnimation(null; %s %.3f)", DescribeAnimation[float64](p), p.Value())
	}
	return fmt.Sprintf("%v➩ProxyAnimation", p.parent)
}

// ReverseAnimation is an animation that is the reverse of another: while the
// parent runs forward from 0 to 1, this one runs in reverse from 1 to 0.
//
// Flutter: animation/animations.dart ReverseAnimation
type ReverseAnimation struct {
	LazyListeners
	LocalStatusListeners

	parent       Animation[float64]
	statusHandle StatusListenerHandle
}

// NewReverseAnimation creates the animation that reverses the parent.
//
// Flutter: animation/animations.dart ReverseAnimation.new
func NewReverseAnimation(parent Animation[float64]) *ReverseAnimation {
	r := &ReverseAnimation{parent: parent}
	r.OnStartListening = func() {
		r.statusHandle = r.parent.AddStatusListener(func(s AnimationStatus) { r.NotifyStatusListeners(reverseStatus(s)) })
	}
	r.OnStopListening = func() { r.parent.RemoveStatusListener(r.statusHandle) }
	r.OnRegister = r.DidRegisterListener
	r.OnUnregister = r.DidUnregisterListener
	return r
}

// Parent is the animation whose value and direction this one reverses.
//
// Flutter: animation/animations.dart ReverseAnimation.parent
func (r *ReverseAnimation) Parent() Animation[float64] { return r.parent }

// AddListener adds the listener to the parent.
//
// Flutter: animation/animations.dart ReverseAnimation.addListener
func (r *ReverseAnimation) AddListener(listener foundation.VoidCallback) foundation.ListenerHandle {
	r.DidRegisterListener()
	return r.parent.AddListener(listener)
}

// RemoveListener removes the listener from the parent.
//
// Flutter: animation/animations.dart ReverseAnimation.removeListener
func (r *ReverseAnimation) RemoveListener(handle foundation.ListenerHandle) {
	r.parent.RemoveListener(handle)
	r.DidUnregisterListener()
}

// Status is the reverse of the status of the parent.
//
// Flutter: animation/animations.dart ReverseAnimation.status
func (r *ReverseAnimation) Status() AnimationStatus { return reverseStatus(r.parent.Status()) }

// Value is 1 minus the value of the parent.
//
// Flutter: animation/animations.dart ReverseAnimation.value
func (r *ReverseAnimation) Value() float64 { return 1.0 - r.parent.Value() }

func reverseStatus(s AnimationStatus) AnimationStatus {
	switch s {
	case AnimationStatusForward:
		return AnimationStatusReverse
	case AnimationStatusReverse:
		return AnimationStatusForward
	case AnimationStatusCompleted:
		return AnimationStatusDismissed
	case AnimationStatusDismissed:
		return AnimationStatusCompleted
	}
	return s
}

// String describes the animation.
//
// Flutter: animation/animations.dart ReverseAnimation.toString
func (r *ReverseAnimation) String() string { return fmt.Sprintf("%v➪ReverseAnimation", r.parent) }

// CurvedAnimation is an animation that applies a curve to another animation,
// with a different curve for the reverse direction if you like. Depending on the
// curve, its output can have a wider range than its input (elastic curves
// overshoot). It adds a status listener to its parent, so call Dispose when it
// is no longer needed, and do not create one in a build method.
//
// Flutter: animation/animations.dart CurvedAnimation
type CurvedAnimation struct {
	AnimationWithParent[float64]

	// Curve is the curve used in the forward direction.
	Curve Curve
	// ReverseCurve is the curve used in the reverse direction; nil uses Curve in
	// both directions. If the parent changes direction without first reaching
	// completed or dismissed, the animation stays on the same curve, to avoid
	// jumps.
	ReverseCurve Curve
	// IsDisposed is whether Dispose was called.
	IsDisposed bool

	curveDirection AnimationStatus
	statusHandle   StatusListenerHandle
}

// NewCurvedAnimation creates the animation. The reverse curve (nil for none)
// is used when the parent runs in reverse.
//
// Flutter: animation/animations.dart CurvedAnimation.new
func NewCurvedAnimation(parent Animation[float64], curve Curve, reverseCurve ...Curve) *CurvedAnimation {
	a := &CurvedAnimation{AnimationWithParent: AnimationWithParent[float64]{Parent: parent}, Curve: curve}
	if len(reverseCurve) > 0 {
		a.ReverseCurve = reverseCurve[0]
	}
	a.updateCurveDirection(parent.Status())
	a.statusHandle = parent.AddStatusListener(a.updateCurveDirection)
	return a
}

func (a *CurvedAnimation) updateCurveDirection(status AnimationStatus) {
	if status.IsAnimating() {
		if a.curveDirection == AnimationStatusUnspecified {
			a.curveDirection = status
		}
		return
	}
	a.curveDirection = AnimationStatusUnspecified
}

func (a *CurvedAnimation) useForwardCurve() bool {
	dir := a.curveDirection
	if dir == AnimationStatusUnspecified {
		dir = a.Parent.Status()
	}
	return a.ReverseCurve == nil || dir != AnimationStatusReverse
}

// Dispose removes the listener that the animation added to its parent.
//
// Flutter: animation/animations.dart CurvedAnimation.dispose
func (a *CurvedAnimation) Dispose() {
	a.IsDisposed = true
	a.Parent.RemoveStatusListener(a.statusHandle)
}

// Value is the value of the parent passed through the active curve. In debug
// builds it panics with the tag CurveEndpointInvalid when the curve does not map
// 0 to near 0 and 1 to near 1.
//
// Flutter: animation/animations.dart CurvedAnimation.value
func (a *CurvedAnimation) Value() float64 {
	active := a.Curve
	if !a.useForwardCurve() {
		active = a.ReverseCurve
	}
	t := a.Parent.Value()
	if active == nil {
		return t
	}
	if t == 0 || t == 1 {
		if foundation.DebugMode {
			transformed := active.Transform(t)
			if math.Round(transformed) != t {
				panic(errors.New("CurveEndpointInvalid", fmt.Sprintf("curves must map 0.0 to near zero and 1.0 to near one but %T mapped %v to %v, which is near %v", active, t, transformed, math.Round(transformed)),
					"curve", fmt.Sprint(active), "t", t))
			}
		}
		return t
	}
	return active.Transform(t)
}

// String describes the animation.
//
// Flutter: animation/animations.dart CurvedAnimation.toString
func (a *CurvedAnimation) String() string {
	switch {
	case a.ReverseCurve == nil:
		return fmt.Sprintf("%v➩%v", a.Parent, a.Curve)
	case a.useForwardCurve():
		return fmt.Sprintf("%v➩%vₒₙ/%v", a.Parent, a.Curve, a.ReverseCurve)
	}
	return fmt.Sprintf("%v➩%v/%vₒₙ", a.Parent, a.Curve, a.ReverseCurve)
}

type trainHoppingMode int

const (
	trainHoppingNone trainHoppingMode = iota
	trainHoppingMinimize
	trainHoppingMaximize
)

// TrainHoppingAnimation starts by proxying one animation, and when the value of
// that animation crosses the value of the second (because the second goes the
// other way, or one overtakes the other), it hops to proxying the second. It
// tracks both animations even without listeners of its own, so call Dispose to
// shut it down.
//
// Flutter: animation/animations.dart TrainHoppingAnimation
type TrainHoppingAnimation struct {
	LocalListeners
	LocalStatusListeners

	// OnSwitchedTrain is called when the animation switches to be driven by the
	// second animation. It is not called when both started at the same value; the
	// second is then used from the start.
	OnSwitchedTrain func()

	currentTrain Animation[float64]
	nextTrain    Animation[float64]
	mode         trainHoppingMode

	lastStatus    AnimationStatus
	lastValue     float64
	hasLastValue  bool
	currentValueH foundation.ListenerHandle
	nextValueH    foundation.ListenerHandle
	currentStatH  StatusListenerHandle
}

// NewTrainHoppingAnimation creates the animation. The next train (nil for none)
// is the animation to hop to; without it, this animation only proxies the
// current one.
//
// Flutter: animation/animations.dart TrainHoppingAnimation.new
func NewTrainHoppingAnimation(currentTrain, nextTrain Animation[float64], onSwitchedTrain func()) *TrainHoppingAnimation {
	t := &TrainHoppingAnimation{currentTrain: currentTrain, nextTrain: nextTrain, OnSwitchedTrain: onSwitchedTrain}
	if t.nextTrain != nil {
		switch cv, nv := t.currentTrain.Value(), t.nextTrain.Value(); {
		case cv == nv:
			t.currentTrain = t.nextTrain
			t.nextTrain = nil
		case cv > nv:
			t.mode = trainHoppingMaximize
		default:
			t.mode = trainHoppingMinimize
		}
	}
	t.currentStatH = t.currentTrain.AddStatusListener(t.statusChangeHandler)
	t.currentValueH = t.currentTrain.AddListener(t.valueChangeHandler)
	if t.nextTrain != nil {
		t.nextValueH = t.nextTrain.AddListener(t.valueChangeHandler)
	}
	return t
}

// CurrentTrain is the animation that currently drives this one; it changes from
// the first animation to the second when OnSwitchedTrain is called.
//
// Flutter: animation/animations.dart TrainHoppingAnimation.currentTrain
func (t *TrainHoppingAnimation) CurrentTrain() Animation[float64] { return t.currentTrain }

func (t *TrainHoppingAnimation) statusChangeHandler(status AnimationStatus) {
	if status != t.lastStatus {
		t.NotifyStatusListeners(status)
		t.lastStatus = status
	}
}

// Status is the status of the current train.
//
// Flutter: animation/animations.dart TrainHoppingAnimation.status
func (t *TrainHoppingAnimation) Status() AnimationStatus { return t.currentTrain.Status() }

func (t *TrainHoppingAnimation) valueChangeHandler() {
	hop := false
	if t.nextTrain != nil {
		switch t.mode {
		case trainHoppingMinimize:
			hop = t.nextTrain.Value() <= t.currentTrain.Value()
		case trainHoppingMaximize:
			hop = t.nextTrain.Value() >= t.currentTrain.Value()
		}
		if hop {
			t.currentTrain.RemoveStatusListener(t.currentStatH)
			t.currentTrain.RemoveListener(t.currentValueH)
			t.currentTrain = t.nextTrain
			t.currentValueH = t.nextValueH
			t.nextTrain = nil
			t.currentStatH = t.currentTrain.AddStatusListener(t.statusChangeHandler)
			t.statusChangeHandler(t.currentTrain.Status())
		}
	}
	if newValue := t.Value(); !t.hasLastValue || newValue != t.lastValue {
		t.NotifyListeners()
		t.lastValue, t.hasLastValue = newValue, true
	}
	if hop && t.OnSwitchedTrain != nil {
		t.OnSwitchedTrain()
	}
}

// Value is the value of the current train.
//
// Flutter: animation/animations.dart TrainHoppingAnimation.value
func (t *TrainHoppingAnimation) Value() float64 { return t.currentTrain.Value() }

// Dispose frees the resources of the animation; it can't be used afterwards.
//
// Flutter: animation/animations.dart TrainHoppingAnimation.dispose
func (t *TrainHoppingAnimation) Dispose() {
	t.currentTrain.RemoveStatusListener(t.currentStatH)
	t.currentTrain.RemoveListener(t.currentValueH)
	t.currentTrain = nil
	if t.nextTrain != nil {
		t.nextTrain.RemoveListener(t.nextValueH)
	}
	t.nextTrain = nil
	t.ClearListeners()
	t.ClearStatusListeners()
}

// String describes the animation.
//
// Flutter: animation/animations.dart TrainHoppingAnimation.toString
func (t *TrainHoppingAnimation) String() string {
	if t.nextTrain != nil {
		return fmt.Sprintf("%v➩TrainHoppingAnimation(next: %v)", t.currentTrain, t.nextTrain)
	}
	return fmt.Sprintf("%v➩TrainHoppingAnimation(no next)", t.currentTrain)
}

// CompoundAnimation combines two animations: the value comes from a function of
// them, and the status is the status of Next if it is moving, otherwise the
// status of First. Either can be a CompoundAnimation itself.
//
// Flutter: animation/animations.dart CompoundAnimation
type CompoundAnimation[T comparable] struct {
	LazyListeners
	LocalListeners
	LocalStatusListeners

	// First is the first animation; its status takes precedence if neither moves.
	First Animation[T]
	// Next is the second animation.
	Next Animation[T]

	name   string
	valueF func(first, next T) T

	handles      [2]foundation.ListenerHandle
	statHandles  [2]StatusListenerHandle
	lastStatus   AnimationStatus
	lastValue    T
	hasLastValue bool
}

// NewCompoundAnimation creates an animation of the two, whose value is
// combine(first.Value(), next.Value()). The name is used in String.
//
// Flutter: animation/animations.dart CompoundAnimation.new
func NewCompoundAnimation[T comparable](name string, first, next Animation[T], combine func(first, next T) T) *CompoundAnimation[T] {
	c := &CompoundAnimation[T]{First: first, Next: next, name: name, valueF: combine}
	c.OnStartListening = func() {
		c.handles[0] = first.AddListener(c.maybeNotifyListeners)
		c.statHandles[0] = first.AddStatusListener(c.maybeNotifyStatusListeners)
		c.handles[1] = next.AddListener(c.maybeNotifyListeners)
		c.statHandles[1] = next.AddStatusListener(c.maybeNotifyStatusListeners)
	}
	c.OnStopListening = func() {
		first.RemoveListener(c.handles[0])
		first.RemoveStatusListener(c.statHandles[0])
		next.RemoveListener(c.handles[1])
		next.RemoveStatusListener(c.statHandles[1])
	}
	c.LocalListeners.OnRegister = c.DidRegisterListener
	c.LocalListeners.OnUnregister = c.DidUnregisterListener
	c.LocalStatusListeners.OnRegister = c.DidRegisterListener
	c.LocalStatusListeners.OnUnregister = c.DidUnregisterListener
	return c
}

// Status is the status of Next if it is moving, otherwise the status of First.
//
// Flutter: animation/animations.dart CompoundAnimation.status
func (c *CompoundAnimation[T]) Status() AnimationStatus {
	if c.Next.Status().IsAnimating() {
		return c.Next.Status()
	}
	return c.First.Status()
}

// Value is the combination of the values of the two animations.
//
// Flutter: animation/animations.dart CompoundAnimation.value
func (c *CompoundAnimation[T]) Value() T { return c.valueF(c.First.Value(), c.Next.Value()) }

func (c *CompoundAnimation[T]) maybeNotifyStatusListeners(AnimationStatus) {
	if s := c.Status(); s != c.lastStatus {
		c.lastStatus = s
		c.NotifyStatusListeners(s)
	}
}

func (c *CompoundAnimation[T]) maybeNotifyListeners() {
	if v := c.Value(); !c.hasLastValue || v != c.lastValue {
		c.lastValue, c.hasLastValue = v, true
		c.NotifyListeners()
	}
}

// String describes the animation.
//
// Flutter: animation/animations.dart CompoundAnimation.toString
func (c *CompoundAnimation[T]) String() string {
	return fmt.Sprintf("%s(%v, %v)", c.name, c.First, c.Next)
}

// NewAnimationMean creates an animation of the mean of two others. Its status is
// the status of the right animation if it moves, otherwise the status of the
// left one.
//
// Flutter: animation/animations.dart AnimationMean
func NewAnimationMean(left, right Animation[float64]) *CompoundAnimation[float64] {
	return NewCompoundAnimation("AnimationMean", left, right, func(a, b float64) float64 { return (a + b) / 2 })
}

// NewAnimationMax creates an animation of the maximum of two others. Either can
// be an AnimationMax itself, to combine more.
//
// Flutter: animation/animations.dart AnimationMax
func NewAnimationMax[T int | float64](first, next Animation[T]) *CompoundAnimation[T] {
	return NewCompoundAnimation("AnimationMax", first, next, func(a, b T) T { return max(a, b) })
}

// NewAnimationMin creates an animation of the minimum of two others. Either can
// be an AnimationMin itself, to combine more.
//
// Flutter: animation/animations.dart AnimationMin
func NewAnimationMin[T int | float64](first, next Animation[T]) *CompoundAnimation[T] {
	return NewCompoundAnimation("AnimationMin", first, next, func(a, b T) T { return min(a, b) })
}
