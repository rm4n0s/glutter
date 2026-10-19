package animation

import (
	"github.com/rm4n0s/glutter/foundation"
)

// AnimationStatus is the status of an animation.
//
// Flutter: animation/animation.dart AnimationStatus
type AnimationStatus int

// The values of AnimationStatus follow Flutter's declaration order.
const (
	// AnimationStatusUnspecified is the zero value; it is not a real status.
	//
	// Flutter: no counterpart
	AnimationStatusUnspecified AnimationStatus = iota
	// AnimationStatusDismissed means the animation is stopped at the beginning.
	//
	// Flutter: animation/animation.dart AnimationStatus.dismissed
	AnimationStatusDismissed
	// AnimationStatusForward means the animation is running from beginning to
	// end.
	//
	// Flutter: animation/animation.dart AnimationStatus.forward
	AnimationStatusForward
	// AnimationStatusReverse means the animation is running backwards, from end to
	// beginning.
	//
	// Flutter: animation/animation.dart AnimationStatus.reverse
	AnimationStatusReverse
	// AnimationStatusCompleted means the animation is stopped at the end.
	//
	// Flutter: animation/animation.dart AnimationStatus.completed
	AnimationStatusCompleted
)

// IsDismissed reports whether the status is AnimationStatusDismissed.
//
// Flutter: animation/animation.dart AnimationStatus.isDismissed
func (s AnimationStatus) IsDismissed() bool { return s == AnimationStatusDismissed }

// IsCompleted reports whether the status is AnimationStatusCompleted.
//
// Flutter: animation/animation.dart AnimationStatus.isCompleted
func (s AnimationStatus) IsCompleted() bool { return s == AnimationStatusCompleted }

// IsAnimating reports whether the animation is running: forward or reverse.
//
// Flutter: animation/animation.dart AnimationStatus.isAnimating
func (s AnimationStatus) IsAnimating() bool {
	return s == AnimationStatusForward || s == AnimationStatusReverse
}

// IsForwardOrCompleted reports whether the animation is running forward, or is
// at the end.
//
// Flutter: animation/animation.dart AnimationStatus.isForwardOrCompleted
func (s AnimationStatus) IsForwardOrCompleted() bool {
	return s == AnimationStatusForward || s == AnimationStatusCompleted
}

// Name is the name of the status as Dart writes it, such as "forward".
//
// Flutter: animation/animation.dart AnimationStatus.name
func (s AnimationStatus) Name() string {
	switch s {
	case AnimationStatusDismissed:
		return "dismissed"
	case AnimationStatusForward:
		return "forward"
	case AnimationStatusReverse:
		return "reverse"
	case AnimationStatusCompleted:
		return "completed"
	}
	return "unspecified"
}

// String is the name with the enum, such as "AnimationStatus.forward".
//
// Flutter: animation/animation.dart AnimationStatus
func (s AnimationStatus) String() string { return "AnimationStatus." + s.Name() }

// AnimationStatusListener is called when the status of an animation changes.
//
// Flutter: animation/animation.dart AnimationStatusListener
type AnimationStatusListener func(status AnimationStatus)

// StatusListenerHandle identifies a status listener that was added to an
// animation, to remove it. The zero value identifies none.
//
// Flutter: no counterpart (Dart removes a listener by its function)
type StatusListenerHandle struct{ id uint64 }

// Animation is a value of type T that changes over time, with a status. The
// listeners are called when the value changes; the status listeners when the
// status changes. An Animation[float64] is a foundation.ValueListenable[float64],
// so it can drive a widget that takes one.
//
// Flutter: animation/animation.dart Animation
type Animation[T any] interface {
	foundation.ValueListenable[T]
	// AddStatusListener calls the listener every time the status of the
	// animation changes.
	//
	// Flutter: animation/animation.dart Animation.addStatusListener
	AddStatusListener(listener AnimationStatusListener) StatusListenerHandle
	// RemoveStatusListener stops calling the listener.
	//
	// Flutter: animation/animation.dart Animation.removeStatusListener
	RemoveStatusListener(handle StatusListenerHandle)
	// Status is the current status of the animation.
	//
	// Flutter: animation/animation.dart Animation.status
	Status() AnimationStatus
}

// IsDismissed reports whether the animation is stopped at the beginning.
//
// Flutter: animation/animation.dart Animation.isDismissed
func IsDismissed[T any](a Animation[T]) bool { return a.Status().IsDismissed() }

// IsCompleted reports whether the animation is stopped at the end.
//
// Flutter: animation/animation.dart Animation.isCompleted
func IsCompleted[T any](a Animation[T]) bool { return a.Status().IsCompleted() }

// IsAnimating reports whether the animation is running, forward or reverse.
//
// Flutter: animation/animation.dart Animation.isAnimating
func IsAnimating[T any](a Animation[T]) bool { return a.Status().IsAnimating() }

// IsForwardOrCompleted reports whether the animation is running forward or is at
// the end.
//
// Flutter: animation/animation.dart Animation.isForwardOrCompleted
func IsForwardOrCompleted[T any](a Animation[T]) bool { return a.Status().IsForwardOrCompleted() }

// Drive chains an Animatable to the animation: the result is an animation of U
// that follows the parent.
//
// Flutter: animation/animation.dart Animation.drive
func Drive[U any](parent Animation[float64], child Animatable[U]) Animation[U] {
	return child.Animate(parent)
}

// DescribeAnimation is the text that Flutter's Animation.toString gives for the
// animation: its identity and a symbol for its status.
//
// Flutter: animation/animation.dart Animation.toString
func DescribeAnimation[T any](a Animation[T]) string {
	return foundation.DescribeIdentity(a) + "(" + statusSymbol(a.Status()) + ")"
}

func statusSymbol(s AnimationStatus) string {
	switch s {
	case AnimationStatusForward:
		return "▶"
	case AnimationStatusReverse:
		return "◀"
	case AnimationStatusCompleted:
		return "⏭"
	}
	return "⏮"
}

// valueListenableAnimation is an Animation that follows a ValueListenable.
type valueListenableAnimation[T any] struct {
	listenable  foundation.ValueListenable[T]
	transformer func(T) T
}

// NewAnimationFromValueListenable creates an animation that has the value of the
// listenable, passed through the transformer if there is one (nil for none). It
// has no status of its own: it is always AnimationStatusForward.
//
// Flutter: animation/animation.dart Animation.fromValueListenable
func NewAnimationFromValueListenable[T any](listenable foundation.ValueListenable[T], transformer func(T) T) Animation[T] {
	return &valueListenableAnimation[T]{listenable, transformer}
}

func (a *valueListenableAnimation[T]) AddListener(listener foundation.VoidCallback) foundation.ListenerHandle {
	return a.listenable.AddListener(listener)
}

func (a *valueListenableAnimation[T]) RemoveListener(handle foundation.ListenerHandle) {
	a.listenable.RemoveListener(handle)
}

func (a *valueListenableAnimation[T]) AddStatusListener(AnimationStatusListener) StatusListenerHandle {
	return StatusListenerHandle{}
}

func (a *valueListenableAnimation[T]) RemoveStatusListener(StatusListenerHandle) {}

func (a *valueListenableAnimation[T]) Status() AnimationStatus { return AnimationStatusForward }

func (a *valueListenableAnimation[T]) Value() T {
	if a.transformer != nil {
		return a.transformer(a.listenable.Value())
	}
	return a.listenable.Value()
}
