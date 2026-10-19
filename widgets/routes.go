package widgets

import (
	"fmt"
	"reflect"
	"slices"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/physics"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
)

// RoutePageBuilder builds the primary content of a route.
//
// Flutter: widgets/routes.dart RoutePageBuilder
type RoutePageBuilder func(bc BuildContext, animation, secondaryAnimation animation.Animation[float64]) Widget

// RouteTransitionsBuilder builds the transition of the page of a route.
//
// Flutter: widgets/routes.dart RouteTransitionsBuilder
type RouteTransitionsBuilder func(bc BuildContext, animation, secondaryAnimation animation.Animation[float64], child Widget) Widget

// OverlayRoute is a route that displays widgets in the Overlay of the
// navigator.
//
// Flutter: widgets/routes.dart OverlayRoute
type OverlayRoute interface {
	Route
	// CreateOverlayEntries creates the entries that the route puts in the
	// overlay; they are created in Install.
	//
	// Flutter: widgets/routes.dart OverlayRoute.createOverlayEntries
	CreateOverlayEntries() []*OverlayEntry
	// FinishedWhenPopped is whether the navigator can finalize the route as soon
	// as it is popped, or the route calls FinalizeRoute itself later.
	//
	// Flutter: widgets/routes.dart OverlayRoute.finishedWhenPopped
	FinishedWhenPopped() bool
}

// OverlayRouteBase implements the parts of OverlayRoute that are shared. Embed
// it and call InitOverlayRouteBase; implement CreateOverlayEntries.
//
// Flutter: widgets/routes.dart OverlayRoute
type OverlayRouteBase struct {
	RouteBase
	oself          OverlayRoute
	overlayEntries []*OverlayEntry
}

// InitOverlayRouteBase sets the outer value of the route.
//
// Flutter: widgets/routes.dart OverlayRoute.new
func InitOverlayRouteBase(b *OverlayRouteBase, self OverlayRoute, opts RouteOpts) {
	InitRouteBase(&b.RouteBase, self, opts)
	b.oself = self
}

// OverlayEntries are the entries of the route in the overlay.
//
// Flutter: widgets/routes.dart OverlayRoute.overlayEntries
func (b *OverlayRouteBase) OverlayEntries() []*OverlayEntry { return b.overlayEntries }

// Install creates the overlay entries.
//
// Flutter: widgets/routes.dart OverlayRoute.install
func (b *OverlayRouteBase) Install() {
	if foundation.DebugMode && len(b.overlayEntries) > 0 {
		panic(errors.New("OverlayRouteInstalledTwice", "the overlay entries of the route exist already; a route can be installed only once",
			"route", b.self.String()))
	}
	b.overlayEntries = append(b.overlayEntries, b.oself.CreateOverlayEntries()...)
	b.RouteBase.Install()
}

// FinishedWhenPopped is true.
//
// Flutter: widgets/routes.dart OverlayRoute.finishedWhenPopped
func (b *OverlayRouteBase) FinishedWhenPopped() bool { return true }

// DidPop finalizes the route when FinishedWhenPopped.
//
// Flutter: widgets/routes.dart OverlayRoute.didPop
func (b *OverlayRouteBase) DidPop(result any) bool {
	returnValue := b.RouteBase.DidPop(result)
	if b.oself.FinishedWhenPopped() {
		b.navigator.FinalizeRoute(b.oself)
	}
	return returnValue
}

// Dispose disposes the overlay entries.
//
// Flutter: widgets/routes.dart OverlayRoute.dispose
func (b *OverlayRouteBase) Dispose() {
	for _, entry := range b.overlayEntries {
		entry.Dispose()
	}
	b.overlayEntries = nil
	b.RouteBase.Dispose()
}

// TransitionRoute is an overlay route with an animation: it animates in when it
// is pushed and out when it is popped.
//
// Flutter: widgets/routes.dart TransitionRoute
type TransitionRoute interface {
	OverlayRoute
	// Completed returns a channel that gets the result of the route when its
	// transition is done and it is disposed.
	//
	// Flutter: widgets/routes.dart TransitionRoute.completed
	Completed() <-chan any
	// TransitionDuration is the duration of the transition in.
	//
	// Flutter: widgets/routes.dart TransitionRoute.transitionDuration
	TransitionDuration() time.Duration
	// ReverseTransitionDuration is the duration of the transition out; it
	// defaults to TransitionDuration.
	//
	// Flutter: widgets/routes.dart TransitionRoute.reverseTransitionDuration
	ReverseTransitionDuration() time.Duration
	// Opaque is whether the route obscures the previous one when the transition
	// is complete.
	//
	// Flutter: widgets/routes.dart TransitionRoute.opaque
	Opaque() bool
	// AllowSnapshotting is whether the route's transitions may be snapshotted.
	//
	// Flutter: widgets/routes.dart TransitionRoute.allowSnapshotting
	AllowSnapshotting() bool
	// Animation drives the transition of the route.
	//
	// Flutter: widgets/routes.dart TransitionRoute.animation
	Animation() animation.Animation[float64]
	// SecondaryAnimation drives the transition of the route when another route
	// is pushed on top of it.
	//
	// Flutter: widgets/routes.dart TransitionRoute.secondaryAnimation
	SecondaryAnimation() animation.Animation[float64]
	// Controller is the controller of the animation.
	//
	// Flutter: widgets/routes.dart TransitionRoute.controller
	Controller() *animation.AnimationController
	// CreateAnimationController creates the controller for the animation.
	//
	// Flutter: widgets/routes.dart TransitionRoute.createAnimationController
	CreateAnimationController() *animation.AnimationController
	// CreateAnimation creates the animation of the route from the controller.
	//
	// Flutter: widgets/routes.dart TransitionRoute.createAnimation
	CreateAnimation() animation.Animation[float64]
	// CreateSimulation creates a simulation to drive the animation instead of a
	// duration; nil means the controller runs for the duration.
	//
	// Flutter: widgets/routes.dart TransitionRoute.createSimulation
	CreateSimulation(forward bool) physics.Simulation
	// CanTransitionTo is whether this route can transition to the next one,
	// animating its secondary animation.
	//
	// Flutter: widgets/routes.dart TransitionRoute.canTransitionTo
	CanTransitionTo(nextRoute TransitionRoute) bool
	// CanTransitionFrom is whether this route can transition from the previous
	// one.
	//
	// Flutter: widgets/routes.dart TransitionRoute.canTransitionFrom
	CanTransitionFrom(previousRoute TransitionRoute) bool
	// DebugLabel names the route for diagnostics.
	//
	// Flutter: widgets/routes.dart TransitionRoute.debugLabel
	DebugLabel() string
	// PopGestureEnabled is whether a back gesture may pop the route.
	//
	// Flutter: widgets/routes.dart PredictiveBackRoute.popGestureEnabled
	PopGestureEnabled() bool
	// HandleStartBackGesture starts a back gesture at the progress.
	//
	// Flutter: widgets/routes.dart PredictiveBackRoute.handleStartBackGesture
	HandleStartBackGesture(progress float64)
	// HandleUpdateBackGestureProgress updates the progress of a back gesture.
	//
	// Flutter: widgets/routes.dart PredictiveBackRoute.handleUpdateBackGestureProgress
	HandleUpdateBackGestureProgress(progress float64)
	// HandleCommitBackGesture pops the route at the end of a back gesture.
	//
	// Flutter: widgets/routes.dart PredictiveBackRoute.handleCommitBackGesture
	HandleCommitBackGesture()
	// HandleCancelBackGesture brings the route back at the end of a back gesture.
	//
	// Flutter: widgets/routes.dart PredictiveBackRoute.handleCancelBackGesture
	HandleCancelBackGesture()
	transitionBase() *TransitionRouteBase
}

// TransitionRouteBase implements the parts of TransitionRoute that are shared.
// Embed it and call InitTransitionRouteBase; implement TransitionDuration, Opaque
// and CreateOverlayEntries.
//
// Flutter: widgets/routes.dart TransitionRoute
type TransitionRouteBase struct {
	OverlayRouteBase
	tself TransitionRoute

	transitionCompleter completer
	popFinalized        bool
	animation           animation.Animation[float64]
	statusHandle        animation.StatusListenerHandle
	controller          *animation.AnimationController
	secondaryAnimation  *animation.ProxyAnimation
	simulation          physics.Simulation
	result              any

	// WillDisposeAnimationController is whether the route disposes the controller
	// when it is disposed; the default is true.
	willDisposeAnimationController bool
	// trainHoppingListenerRemover disposes the existing train hopping animation
	// and removes its listener; it is non-nil while a train hopping is in
	// progress.
	trainHoppingListenerRemover func()
}

// InitTransitionRouteBase sets the outer value of the route.
//
// Flutter: widgets/routes.dart TransitionRoute.new
func InitTransitionRouteBase(b *TransitionRouteBase, self TransitionRoute, opts RouteOpts) {
	InitOverlayRouteBase(&b.OverlayRouteBase, self, opts)
	b.tself = self
	b.secondaryAnimation = animation.NewProxyAnimation(animation.AlwaysDismissed)
	b.willDisposeAnimationController = true
}

func (b *TransitionRouteBase) transitionBase() *TransitionRouteBase { return b }

// Completed returns a channel that gets the result of the route once its
// transition is done and the route is disposed.
//
// Flutter: widgets/routes.dart TransitionRoute.completed
func (b *TransitionRouteBase) Completed() <-chan any { return b.transitionCompleter.Chan() }

// ReverseTransitionDuration is the TransitionDuration.
//
// Flutter: widgets/routes.dart TransitionRoute.reverseTransitionDuration
func (b *TransitionRouteBase) ReverseTransitionDuration() time.Duration {
	return b.tself.TransitionDuration()
}

// AllowSnapshotting is true.
//
// Flutter: widgets/routes.dart TransitionRoute.allowSnapshotting
func (b *TransitionRouteBase) AllowSnapshotting() bool { return true }

// FinishedWhenPopped is whether the controller is dismissed already and the pop
// was not finalized yet. This keeps a route that reached the dismissed state
// while still current disposable when it is eventually popped (this happens with
// the iOS dismiss gesture).
//
// Flutter: widgets/routes.dart TransitionRoute.finishedWhenPopped
func (b *TransitionRouteBase) FinishedWhenPopped() bool {
	return b.controller.Status().IsDismissed() && !b.popFinalized
}

// Animation drives the transition of the route.
//
// Flutter: widgets/routes.dart TransitionRoute.animation
func (b *TransitionRouteBase) Animation() animation.Animation[float64] { return b.animation }

// Controller is the controller of the animation.
//
// Flutter: widgets/routes.dart TransitionRoute.controller
func (b *TransitionRouteBase) Controller() *animation.AnimationController { return b.controller }

// SecondaryAnimation drives the transition when another route is pushed on top.
//
// Flutter: widgets/routes.dart TransitionRoute.secondaryAnimation
func (b *TransitionRouteBase) SecondaryAnimation() animation.Animation[float64] {
	return b.secondaryAnimation
}

// WillDisposeAnimationController is whether the route disposes the controller of
// the animation when it is disposed; the default is true.
//
// Flutter: widgets/routes.dart TransitionRoute.willDisposeAnimationController
func (b *TransitionRouteBase) WillDisposeAnimationController() bool {
	return b.willDisposeAnimationController
}

// SetWillDisposeAnimationController changes whether the route disposes the
// controller of the animation.
//
// Flutter: widgets/routes.dart TransitionRoute.willDisposeAnimationController
func (b *TransitionRouteBase) SetWillDisposeAnimationController(value bool) {
	b.willDisposeAnimationController = value
}

// debugTransitionCompleted is whether the route was disposed already; in debug
// builds only.
//
// Flutter: widgets/routes.dart TransitionRoute.debugTransitionCompleted
func (b *TransitionRouteBase) debugTransitionCompleted() bool {
	return foundation.DebugMode && b.transitionCompleter.IsCompleted()
}

func (b *TransitionRouteBase) debugRequireUsable(method string) {
	if !foundation.DebugMode {
		return
	}
	if b.debugTransitionCompleted() {
		panic(errors.New("TransitionRouteDisposed", "cannot reuse a "+foundation.RuntimeTypeName(b.self)+" after disposing it",
			"method", method))
	}
}

func (b *TransitionRouteBase) debugRequireInstalled(method string) {
	if foundation.DebugMode && b.controller == nil {
		panic(errors.New("TransitionRouteNotInstalled", foundation.RuntimeTypeName(b.self)+"."+method+" was called before calling Install or after calling Dispose",
			"method", method))
	}
}

// CreateAnimationController creates the controller of the route, with the
// navigator as its vsync.
//
// Flutter: widgets/routes.dart TransitionRoute.createAnimationController
func (b *TransitionRouteBase) CreateAnimationController() *animation.AnimationController {
	b.debugRequireUsable("CreateAnimationController")
	return animation.NewAnimationController(b.navigator, animation.AnimationControllerOpts{
		Duration:        foundation.Some(b.tself.TransitionDuration()),
		ReverseDuration: foundation.Some(b.tself.ReverseTransitionDuration()),
		DebugLabel:      b.tself.DebugLabel(),
	})
}

// CreateAnimation is the view of the controller.
//
// Flutter: widgets/routes.dart TransitionRoute.createAnimation
func (b *TransitionRouteBase) CreateAnimation() animation.Animation[float64] {
	b.debugRequireUsable("CreateAnimation")
	return b.controller.View()
}

// CreateSimulation is nil: the controller runs for the duration.
//
// Flutter: widgets/routes.dart TransitionRoute.createSimulation
func (b *TransitionRouteBase) CreateSimulation(forward bool) physics.Simulation {
	if foundation.DebugMode && b.tself.TransitionDuration() < 0 {
		panic(errors.New("TransitionRouteDurationNegative", "the duration must be positive for a non-simulation animation",
			"duration", b.tself.TransitionDuration()))
	}
	return nil
}

func (b *TransitionRouteBase) createSimulationAndVerify(forward bool) physics.Simulation {
	simulation := b.tself.CreateSimulation(forward)
	if foundation.DebugMode && b.tself.TransitionDuration() < 0 {
		panic(errors.New("TransitionRouteDurationNegative", "the duration must be positive for an animation that doesn't use a simulation; either set TransitionDuration or set CreateSimulation",
			"duration", b.tself.TransitionDuration()))
	}
	return simulation
}

func (b *TransitionRouteBase) handleStatusChanged(status animation.AnimationStatus) {
	switch status {
	case animation.AnimationStatusCompleted:
		if len(b.overlayEntries) > 0 {
			b.overlayEntries[0].SetOpaque(b.tself.Opaque())
		}
	case animation.AnimationStatusForward, animation.AnimationStatusReverse:
		if len(b.overlayEntries) > 0 {
			b.overlayEntries[0].SetOpaque(false)
		}
	case animation.AnimationStatusDismissed:
		// The route may still be active if a subclass controls the transition and
		// reaches the dismissed status. For example, the iOS back gesture drives
		// this animation to the dismissed status before removing the route and
		// disposing it.
		if !b.self.IsActive() {
			b.navigator.FinalizeRoute(b.tself)
			b.popFinalized = true
		}
	}
}

// Install creates the controller and the animation, and the overlay entries.
//
// Flutter: widgets/routes.dart TransitionRoute.install
func (b *TransitionRouteBase) Install() {
	if foundation.DebugMode && b.debugTransitionCompleted() {
		panic(errors.New("TransitionRouteDisposed", "cannot install a "+foundation.RuntimeTypeName(b.self)+" after disposing it"))
	}
	b.controller = b.tself.CreateAnimationController()
	if foundation.DebugMode && b.controller == nil {
		panic(errors.New("TransitionRouteControllerMissing", foundation.RuntimeTypeName(b.self)+".CreateAnimationController() returned nil"))
	}
	b.animation = b.tself.CreateAnimation()
	if foundation.DebugMode && b.animation == nil {
		panic(errors.New("TransitionRouteAnimationMissing", foundation.RuntimeTypeName(b.self)+".CreateAnimation() returned nil"))
	}
	b.statusHandle = b.animation.AddStatusListener(b.handleStatusChanged)
	b.OverlayRouteBase.Install()
	if b.animation.Status().IsCompleted() && len(b.overlayEntries) > 0 {
		b.overlayEntries[0].SetOpaque(b.tself.Opaque())
	}
}

// DidPush starts the transition in.
//
// Flutter: widgets/routes.dart TransitionRoute.didPush
func (b *TransitionRouteBase) DidPush() *scheduler.TickerFuture {
	b.debugRequireInstalled("DidPush")
	b.debugRequireUsable("DidPush")
	b.RouteBase.DidPush()
	b.simulation = b.createSimulationAndVerify(true)
	if b.simulation == nil {
		return b.controller.Forward()
	}
	return b.controller.AnimateWith(b.simulation)
}

// DidAdd completes the transition at once.
//
// Flutter: widgets/routes.dart TransitionRoute.didAdd
func (b *TransitionRouteBase) DidAdd() {
	b.debugRequireInstalled("DidAdd")
	b.debugRequireUsable("DidAdd")
	b.RouteBase.DidAdd()
	b.controller.SetValue(b.controller.UpperBound)
}

// DidReplace takes the value of the controller of the route it replaces.
//
// Flutter: widgets/routes.dart TransitionRoute.didReplace
func (b *TransitionRouteBase) DidReplace(oldRoute Route) {
	b.debugRequireInstalled("DidReplace")
	b.debugRequireUsable("DidReplace")
	if old, ok := oldRoute.(TransitionRoute); ok {
		b.controller.SetValue(old.transitionBase().controller.Value())
	}
	b.RouteBase.DidReplace(oldRoute)
}

// DidPop starts the transition out.
//
// Flutter: widgets/routes.dart TransitionRoute.didPop
func (b *TransitionRouteBase) DidPop(result any) bool {
	b.debugRequireInstalled("DidPop")
	if foundation.DebugMode && b.transitionCompleter.IsCompleted() {
		panic(errors.New("TransitionRouteDisposed", "cannot reuse a "+foundation.RuntimeTypeName(b.self)+" after disposing it"))
	}
	b.result = result
	b.simulation = b.createSimulationAndVerify(false)
	if b.simulation == nil {
		b.controller.Reverse()
	} else {
		b.controller.AnimateBackWith(b.simulation)
	}
	return b.OverlayRouteBase.DidPop(result)
}

// DidPopNext updates the secondary animation.
//
// Flutter: widgets/routes.dart TransitionRoute.didPopNext
func (b *TransitionRouteBase) DidPopNext(nextRoute Route) {
	b.debugRequireInstalled("DidPopNext")
	b.debugRequireUsable("DidPopNext")
	b.updateSecondaryAnimation(nextRoute)
	b.RouteBase.DidPopNext(nextRoute)
}

// DidChangeNext updates the secondary animation.
//
// Flutter: widgets/routes.dart TransitionRoute.didChangeNext
func (b *TransitionRouteBase) DidChangeNext(nextRoute Route) {
	b.debugRequireInstalled("DidChangeNext")
	b.debugRequireUsable("DidChangeNext")
	b.updateSecondaryAnimation(nextRoute)
	b.RouteBase.DidChangeNext(nextRoute)
}

func (b *TransitionRouteBase) updateSecondaryAnimation(nextRoute Route) {
	// There may be a train hopping in progress. It can't be disposed until it is
	// replaced with a new animation.
	previousRemover := b.trainHoppingListenerRemover
	b.trainHoppingListenerRemover = nil

	next, isTransition := nextRoute.(TransitionRoute)
	if isTransition && b.tself.CanTransitionTo(next) && next.CanTransitionFrom(b.tself) {
		nextBase := next.transitionBase()
		current := b.secondaryAnimation.Parent()
		if current != nil {
			currentTrain := current
			if hopping, ok := current.(*animation.TrainHoppingAnimation); ok {
				currentTrain = hopping.CurrentTrain()
			}
			nextTrain := nextBase.animation
			if currentTrain.Value() == nextTrain.Value() || !nextTrain.Status().IsAnimating() {
				b.setSecondaryAnimation(nextTrain, &nextBase.transitionCompleter)
			} else {
				// The two trains animate at different values, so a train hopping is
				// needed. There are three possibilities:
				//  1. The next train is hopped on when the two trains meet in the
				//     middle, using TrainHoppingAnimation.
				//  2. The trains never cross each other, so the animation is set to
				//     the next train once it stops animating.
				//  3. updateSecondaryAnimation is called again before the hopping
				//     finishes; the remover left for that call cleans up the
				//     existing hopping.
				var newAnimation *animation.TrainHoppingAnimation
				var jumpHandle animation.StatusListenerHandle
				jumpOnAnimationEnd := func(status animation.AnimationStatus) {
					if !status.IsAnimating() {
						// The next train stopped animating without a hop: sets the
						// secondary animation directly and disposes the hopping.
						b.setSecondaryAnimation(nextTrain, &nextBase.transitionCompleter)
						if b.trainHoppingListenerRemover != nil {
							b.trainHoppingListenerRemover()
							b.trainHoppingListenerRemover = nil
						}
					}
				}
				b.trainHoppingListenerRemover = func() {
					nextTrain.RemoveStatusListener(jumpHandle)
					if newAnimation != nil {
						newAnimation.Dispose()
					}
				}
				jumpHandle = nextTrain.AddStatusListener(jumpOnAnimationEnd)
				newAnimation = animation.NewTrainHoppingAnimation(currentTrain, nextTrain, func() {
					// The next train can be hopped on, so there is no need to listen
					// to whether it stopped.
					b.setSecondaryAnimation(newAnimation.CurrentTrain(), &nextBase.transitionCompleter)
					if b.trainHoppingListenerRemover != nil {
						b.trainHoppingListenerRemover()
						b.trainHoppingListenerRemover = nil
					}
				})
				b.setSecondaryAnimation(newAnimation, &nextBase.transitionCompleter)
			}
		} else {
			b.setSecondaryAnimation(nextBase.animation, &nextBase.transitionCompleter)
		}
	} else {
		b.setSecondaryAnimation(animation.AlwaysDismissed, nil)
	}
	// Finally, disposes the previous train hopping animation, which has been
	// updated successfully by now.
	if previousRemover != nil {
		previousRemover()
	}
}

func (b *TransitionRouteBase) setSecondaryAnimation(a animation.Animation[float64], disposed *completer) {
	b.secondaryAnimation.SetParent(a)
	// Releases the reference to the animation of the next route when that route
	// is disposed.
	if disposed != nil {
		var post func(func())
		if sched := b.binding(); sched != nil {
			post = sched.Post
		}
		disposed.Then(func(any) {
			release := func() {
				if b.secondaryAnimation.Parent() == a {
					b.secondaryAnimation.SetParent(animation.AlwaysDismissed)
					if hopping, ok := a.(*animation.TrainHoppingAnimation); ok {
						hopping.Dispose()
					}
				}
			}
			// Dart runs the callback of a future in a microtask.
			if post != nil {
				post(release)
			} else {
				release()
			}
		})
	}
}

// CanTransitionTo is true.
//
// Flutter: widgets/routes.dart TransitionRoute.canTransitionTo
func (b *TransitionRouteBase) CanTransitionTo(nextRoute TransitionRoute) bool { return true }

// CanTransitionFrom is true.
//
// Flutter: widgets/routes.dart TransitionRoute.canTransitionFrom
func (b *TransitionRouteBase) CanTransitionFrom(previousRoute TransitionRoute) bool { return true }

// PopGestureEnabled is false.
//
// Flutter: widgets/routes.dart PredictiveBackRoute.popGestureEnabled
func (b *TransitionRouteBase) PopGestureEnabled() bool { return false }

// HandleStartBackGesture sets the controller to the progress and tells the
// navigator that a user gesture started.
//
// Flutter: widgets/routes.dart TransitionRoute.handleStartBackGesture
func (b *TransitionRouteBase) HandleStartBackGesture(progress float64) {
	if foundation.DebugMode && !b.self.IsCurrent() {
		panic(errors.New("RouteNotCurrent", "a back gesture started for a route that is not the current one", "route", b.self.String()))
	}
	if b.controller != nil {
		b.controller.SetValue(progress)
	}
	if b.navigator != nil {
		b.navigator.DidStartUserGesture()
	}
}

// HandleUpdateBackGestureProgress sets the controller to the progress, unless
// another navigation happened during the gesture.
//
// Flutter: widgets/routes.dart TransitionRoute.handleUpdateBackGestureProgress
func (b *TransitionRouteBase) HandleUpdateBackGestureProgress(progress float64) {
	if !b.self.IsCurrent() {
		return
	}
	if b.controller != nil {
		b.controller.SetValue(progress)
	}
}

// HandleCancelBackGesture animates the route back in.
//
// Flutter: widgets/routes.dart TransitionRoute.handleCancelBackGesture
func (b *TransitionRouteBase) HandleCancelBackGesture() { b.handleDragEnd(true) }

// HandleCommitBackGesture pops the route.
//
// Flutter: widgets/routes.dart TransitionRoute.handleCommitBackGesture
func (b *TransitionRouteBase) HandleCommitBackGesture() { b.handleDragEnd(false) }

func (b *TransitionRouteBase) handleDragEnd(animateForward bool) {
	if b.self.IsCurrent() {
		if animateForward {
			// Typically HandleUpdateBackGestureProgress completed the animation
			// already. If not, animates to completion.
			if !b.controller.Status().IsCompleted() {
				b.controller.Forward()
			}
		} else {
			// This route is destined to pop at this point, so the navigator's pop
			// is reused.
			if b.navigator != nil {
				b.navigator.Pop()
			}
			// The pop may have finished inline if the route was at the target
			// destination already.
			if b.controller != nil && b.controller.IsAnimating() {
				b.controller.Reverse(b.controller.UpperBound)
			}
		}
	}
	if b.controller != nil && b.controller.IsAnimating() {
		// Keeps the user gesture in progress until the animation is done.
		var handle animation.StatusListenerHandle
		controller := b.controller
		navigator := b.navigator
		handle = controller.AddStatusListener(func(animation.AnimationStatus) {
			if navigator != nil {
				navigator.DidStopUserGesture()
			}
			controller.RemoveStatusListener(handle)
		})
	} else if b.navigator != nil {
		b.navigator.DidStopUserGesture()
	}
}

// Dispose disposes the controller and completes the channel of Completed.
//
// Flutter: widgets/routes.dart TransitionRoute.dispose
func (b *TransitionRouteBase) Dispose() {
	if foundation.DebugMode && b.transitionCompleter.IsCompleted() {
		panic(errors.New("TransitionRouteDisposed", "cannot dispose a "+foundation.RuntimeTypeName(b.self)+" twice"))
	}
	if b.animation != nil {
		b.animation.RemoveStatusListener(b.statusHandle)
	}
	if b.willDisposeAnimationController && b.controller != nil {
		b.controller.Dispose()
	}
	b.transitionCompleter.Complete(b.result)
	b.OverlayRouteBase.Dispose()
}

// DebugLabel names the route for diagnostics.
//
// Flutter: widgets/routes.dart TransitionRoute.debugLabel
func (b *TransitionRouteBase) DebugLabel() string { return foundation.RuntimeTypeName(b.self) }

// String describes the route.
//
// Flutter: widgets/routes.dart TransitionRoute.toString
func (b *TransitionRouteBase) String() string {
	return fmt.Sprintf("%s(animation: %v)", foundation.RuntimeTypeName(b.self), b.controller)
}

// LocalHistoryEntry is an entry in the local history of a route. When the user
// pops the route, the entries are removed one at a time before the route itself.
//
// Flutter: widgets/routes.dart LocalHistoryEntry
type LocalHistoryEntry struct {
	onRemove               func()
	owner                  LocalHistoryRoute
	impliesAppBarDismissal bool
}

// LocalHistoryEntryOpts are the optional parameters of NewLocalHistoryEntry.
//
// Flutter: widgets/routes.dart LocalHistoryEntry.new
type LocalHistoryEntryOpts struct {
	// OnRemove is called when the entry is removed from the history of its route.
	OnRemove func()
	// ImpliesAppBarDismissal is whether an app bar shows a close button while the
	// entry is there; the default is true.
	ImpliesAppBarDismissal foundation.Opt[bool]
}

// NewLocalHistoryEntry creates an entry.
//
// Flutter: widgets/routes.dart LocalHistoryEntry.new
func NewLocalHistoryEntry(opts ...LocalHistoryEntryOpts) *LocalHistoryEntry {
	o := firstOpts(opts)
	return &LocalHistoryEntry{onRemove: o.OnRemove, impliesAppBarDismissal: o.ImpliesAppBarDismissal.OrElse(true)}
}

// ImpliesAppBarDismissal is whether an app bar shows a close button while the
// entry is in the history.
//
// Flutter: widgets/routes.dart LocalHistoryEntry.impliesAppBarDismissal
func (e *LocalHistoryEntry) ImpliesAppBarDismissal() bool { return e.impliesAppBarDismissal }

// Remove removes the entry from the history of its route.
//
// Flutter: widgets/routes.dart LocalHistoryEntry.remove
func (e *LocalHistoryEntry) Remove() {
	if e.owner != nil {
		e.owner.RemoveLocalHistoryEntry(e)
	}
}

func (e *LocalHistoryEntry) notifyRemoved() {
	if e.onRemove != nil {
		e.onRemove()
	}
}

// LocalHistoryRoute is a route that can handle pops itself, with a stack of
// local history entries.
//
// Flutter: widgets/routes.dart LocalHistoryRoute
type LocalHistoryRoute interface {
	Route
	// AddLocalHistoryEntry adds a local history entry; popping the route removes
	// the entries first.
	//
	// Flutter: widgets/routes.dart LocalHistoryRoute.addLocalHistoryEntry
	AddLocalHistoryEntry(entry *LocalHistoryEntry)
	// RemoveLocalHistoryEntry removes a local history entry.
	//
	// Flutter: widgets/routes.dart LocalHistoryRoute.removeLocalHistoryEntry
	RemoveLocalHistoryEntry(entry *LocalHistoryEntry)
}

// localHistory is the local history of a route: the logic of LocalHistoryRoute,
// shared by the routes that have it.
type localHistory struct {
	route                       LocalHistoryRoute
	entries                     []*LocalHistoryEntry
	impliesAppBarDismissalCount int
}

func (h *localHistory) add(entry *LocalHistoryEntry) {
	if foundation.DebugMode && entry.owner != nil {
		panic(errors.New("LocalHistoryEntryOwned", "the local history entry is in the history of a route already"))
	}
	entry.owner = h.route
	wasEmpty := len(h.entries) == 0
	h.entries = append(h.entries, entry)
	internalStateChanged := false
	if entry.impliesAppBarDismissal {
		internalStateChanged = h.impliesAppBarDismissalCount == 0
		h.impliesAppBarDismissalCount++
	}
	if wasEmpty || internalStateChanged {
		h.route.ChangedInternalState()
	}
}

func (h *localHistory) remove(entry *LocalHistoryEntry) {
	i := slices.Index(h.entries, entry)
	if foundation.DebugMode && (entry.owner != h.route || i < 0) {
		panic(errors.New("LocalHistoryEntryNotOwned", "the local history entry is not in the history of this route"))
	}
	internalStateChanged := false
	if i >= 0 {
		h.entries = slices.Delete(h.entries, i, i+1)
		if entry.impliesAppBarDismissal {
			h.impliesAppBarDismissalCount--
			internalStateChanged = h.impliesAppBarDismissalCount == 0
		}
	}
	entry.owner = nil
	entry.notifyRemoved()
	if len(h.entries) == 0 || internalStateChanged {
		var sched *scheduler.SchedulerBinding
		if r, ok := h.route.(interface{ AsRoute() *RouteBase }); ok {
			if binding := r.AsRoute().binding(); binding != nil {
				sched = binding.Scheduler()
			}
		}
		if sched != nil && sched.SchedulerPhase() == scheduler.SchedulerPhasePersistentCallbacks {
			// The local history might be removed as a result of disposing the
			// inactive elements during finalizeTree. The state is locked at this
			// moment, and the change can only be notified in the next frame.
			sched.AddPostFrameCallback(func(time.Duration) {
				if h.route.IsActive() {
					h.route.ChangedInternalState()
				}
			})
		} else {
			h.route.ChangedInternalState()
		}
	}
}

// didPop removes the last entry; it reports whether there was one, in which case
// the pop was handled and the route stays.
func (h *localHistory) didPop() bool {
	if len(h.entries) == 0 {
		return false
	}
	entry := h.entries[len(h.entries)-1]
	h.entries = h.entries[:len(h.entries)-1]
	entry.owner = nil
	entry.notifyRemoved()
	internalStateChanged := false
	if entry.impliesAppBarDismissal {
		h.impliesAppBarDismissalCount--
		internalStateChanged = h.impliesAppBarDismissalCount == 0
	}
	if len(h.entries) == 0 || internalStateChanged {
		h.route.ChangedInternalState()
	}
	return true
}

func (h *localHistory) willHandlePopInternally() bool { return len(h.entries) > 0 }

// LocalHistoryRouteBase is a RouteBase with a local history: embed it, and call
// InitLocalHistoryRouteBase, in a route that is not a ModalRoute but can handle
// pops itself (the modal routes have a local history already).
//
// Flutter: widgets/routes.dart LocalHistoryRoute
type LocalHistoryRouteBase struct {
	RouteBase
	history localHistory
}

// InitLocalHistoryRouteBase sets the outer value of the route.
//
// Flutter: widgets/routes.dart LocalHistoryRoute
func InitLocalHistoryRouteBase(b *LocalHistoryRouteBase, self LocalHistoryRoute, opts RouteOpts) {
	InitRouteBase(&b.RouteBase, self, opts)
	b.history = localHistory{route: self}
}

// AddLocalHistoryEntry adds a local history entry; popping the route removes
// the entries first.
//
// Flutter: widgets/routes.dart LocalHistoryRoute.addLocalHistoryEntry
func (b *LocalHistoryRouteBase) AddLocalHistoryEntry(entry *LocalHistoryEntry) { b.history.add(entry) }

// RemoveLocalHistoryEntry removes a local history entry.
//
// Flutter: widgets/routes.dart LocalHistoryRoute.removeLocalHistoryEntry
func (b *LocalHistoryRouteBase) RemoveLocalHistoryEntry(entry *LocalHistoryEntry) {
	b.history.remove(entry)
}

// PopDisposition is pop while there are local history entries, else the
// disposition of the route.
//
// Flutter: widgets/routes.dart LocalHistoryRoute.popDisposition
func (b *LocalHistoryRouteBase) PopDisposition() RoutePopDisposition {
	if b.history.willHandlePopInternally() {
		return RoutePopDispositionPop
	}
	return b.RouteBase.PopDisposition()
}

// DidPop removes the last local history entry, and does not pop the route,
// while there are some.
//
// Flutter: widgets/routes.dart LocalHistoryRoute.didPop
func (b *LocalHistoryRouteBase) DidPop(result any) bool {
	if b.history.didPop() {
		return false
	}
	return b.RouteBase.DidPop(result)
}

// WillHandlePopInternally is whether the route has local history entries.
//
// Flutter: widgets/routes.dart LocalHistoryRoute.willHandlePopInternally
func (b *LocalHistoryRouteBase) WillHandlePopInternally() bool {
	return b.history.willHandlePopInternally()
}

type modalRouteAspect int

const (
	modalRouteAspectIsCurrent modalRouteAspect = iota
	modalRouteAspectCanPop
	modalRouteAspectSettings
	modalRouteAspectIsActive
	modalRouteAspectIsFirst
	modalRouteAspectOpaque
	modalRouteAspectPopDisposition
)

// modalScopeStatus is the inherited model that tells the widgets of a modal route
// about the state of the route.
//
// Flutter: widgets/routes.dart _ModalScopeStatus
type modalScopeStatus struct {
	InheritedModel
	isCurrent              bool
	canPop                 bool
	impliesAppBarDismissal bool
	opaque                 bool
	route                  ModalRoute
}

func (w *modalScopeStatus) UpdateShouldNotify(old Widget) bool {
	o := old.(*modalScopeStatus)
	return w.isCurrent != o.isCurrent || w.canPop != o.canPop ||
		w.impliesAppBarDismissal != o.impliesAppBarDismissal || w.route != o.route || w.opaque != o.opaque
}

func (w *modalScopeStatus) UpdateShouldNotifyDependent(old Widget, dependencies *AspectSet) bool {
	o := old.(*modalScopeStatus)
	return slices.ContainsFunc(dependencies.Items(), func(dependency any) bool {
		aspect, ok := dependency.(modalRouteAspect)
		if !ok {
			return false
		}
		switch aspect {
		case modalRouteAspectIsCurrent:
			return w.isCurrent != o.isCurrent
		case modalRouteAspectCanPop:
			return w.canPop != o.canPop
		case modalRouteAspectSettings:
			return w.route.Settings() != o.route.Settings()
		case modalRouteAspectIsActive:
			return w.route.IsActive() != o.route.IsActive()
		case modalRouteAspectIsFirst:
			return w.route.IsFirst() != o.route.IsFirst()
		case modalRouteAspectOpaque:
			return w.opaque != o.opaque
		case modalRouteAspectPopDisposition:
			return w.route.PopDisposition() != o.route.PopDisposition()
		}
		return false
	})
}

// modalScope is the widget that builds the page of a modal route.
//
// Flutter: widgets/routes.dart _ModalScope
type modalScope struct {
	StatefulWidget
	route ModalRoute
}

func newModalScope(key foundation.Key, route ModalRoute) *modalScope {
	return &modalScope{NewStatefulWidgetBase(WidgetOpts{Key: key}), route}
}

func (w *modalScope) CreateState() State { return &modalScopeState{} }

type modalScopeState struct {
	StateOf[*modalScope]
	// page caches the result of calling the route's BuildPage, and is cleared
	// whenever the dependencies change. This implements the contract of
	// BuildPage: it is called once, unless something like a ModalRoute.of
	// dependency triggers an update.
	page Widget
	// listenable is the combination of the two animations of the route.
	listenable              foundation.Listenable
	primaryScrollController ScrollController
}

func (s *modalScopeState) InitState() {
	route := s.Widget().route
	var animations []foundation.Listenable
	if a := route.Animation(); a != nil {
		animations = append(animations, a)
	}
	if a := route.SecondaryAnimation(); a != nil {
		animations = append(animations, a)
	}
	s.listenable = foundation.MergeListenables(animations...)
	s.primaryScrollController = NewScrollController()
}

func (s *modalScopeState) DidUpdateWidget(old *modalScope) {
	if foundation.DebugMode && s.Widget().route != old.route {
		panic(errors.New("ModalScopeRouteChanged", "the route of a modal scope can not change"))
	}
	// TODO(step 10.5): update the traversal edge behavior of the focus scope node.
}

func (s *modalScopeState) DidChangeDependencies() {
	s.page = nil
	// TODO(step 10.5): update the focus scope node, and set it as the first focus
	// of the navigator when the route is current and requests focus.
}

func (s *modalScopeState) forceRebuildPage() {
	s.SetState(func() { s.page = nil })
}

func (s *modalScopeState) Dispose() {
	s.primaryScrollController.Dispose()
}

func (s *modalScopeState) shouldIgnoreFocusRequest() bool {
	route := s.Widget().route
	if a := route.Animation(); a != nil && a.Status() == animation.AnimationStatusReverse {
		return true
	}
	return route.Navigator() != nil && route.Navigator().UserGestureInProgress()
}

// routeSetState should wrap any change to route.IsCurrent, route.CanPop and
// route.Offstage.
func (s *modalScopeState) routeSetState(fn func()) {
	// TODO(step 10.5): set the first focus of the navigator when the route is
	// current, requests focus and is not ignoring focus requests.
	s.SetState(fn)
}

func (s *modalScopeState) Build(bc BuildContext) Widget {
	route := s.Widget().route
	base := route.modalBase()
	// TODO(step 10.5): only the top most route takes part in the focus traversal.
	// TODO(step 10.6): Actions with the DismissIntent map the Escape key to
	// MaybePop.
	return &modalScopeStatus{
		InheritedModel: NewInheritedModelBase(
			NewOffstage(
				NewPageStorage(base.storageBucket,
					NewBuilder(func(bc BuildContext) Widget {
						return NewPrimaryScrollController(s.primaryScrollController,
							NewRepaintBoundary(
								NewListenableBuilder(s.listenable, func(bc BuildContext, child Widget) Widget {
									return route.modalBase().buildFlexibleTransitions(bc, route.Animation(), route.SecondaryAnimation(),
										// This additional ListenableBuilder is included because if the
										// value of the userGestureInProgressNotifier changes, only the
										// IgnorePointer needs to be rebuilt.
										NewListenableBuilder(s.userGestureNotifier(route), func(bc BuildContext, child Widget) Widget {
											ignoreEvents := s.shouldIgnoreFocusRequest()
											return NewIgnorePointer(child, IgnorePointerOpts{Ignoring: foundation.Some(ignoreEvents)})
										}, child))
								}, s.cachedPage(route)),
							))
					}),
				),
				OffstageOpts{Offstage: foundation.Some(route.Offstage())},
			),
		),
		isCurrent:              route.IsCurrent(),
		canPop:                 route.CanPop(),
		opaque:                 route.Opaque(),
		impliesAppBarDismissal: route.ImpliesAppBarDismissal(),
		route:                  route,
	}
}

func (s *modalScopeState) userGestureNotifier(route ModalRoute) foundation.Listenable {
	if navigator := route.Navigator(); navigator != nil {
		return navigator.UserGestureInProgressNotifier()
	}
	return foundation.NewValueNotifier(false)
}

func (s *modalScopeState) cachedPage(route ModalRoute) Widget {
	if s.page == nil {
		s.page = NewRepaintBoundary(
			NewBuilder(func(bc BuildContext) Widget {
				return route.BuildPage(bc, route.Animation(), route.SecondaryAnimation())
			}),
			RepaintBoundaryOpts{Key: route.modalBase().subtreeKey},
		)
	}
	return s.page
}

// PopEntry is a pop scope that a route consults when it is asked to pop; PopScope
// registers itself as one.
//
// Flutter: widgets/routes.dart PopEntry
type PopEntry interface {
	// OnPopInvokedWithResult is called after a pop was handled (didPop true) or
	// blocked.
	//
	// Flutter: widgets/routes.dart PopEntry.onPopInvokedWithResult
	OnPopInvokedWithResult(didPop bool, result any)
	// CanPopNotifier tells whether a pop is allowed.
	//
	// Flutter: widgets/routes.dart PopEntry.canPopNotifier
	CanPopNotifier() foundation.ValueListenable[bool]
}

// ModalRoute is a route that blocks interaction with previous routes: it has a
// barrier below its page, can be dismissed by tapping the barrier, and can hold
// the state of PopScopes.
//
// Flutter: widgets/routes.dart ModalRoute
type ModalRoute interface {
	TransitionRoute
	LocalHistoryRoute
	// BuildPage builds the primary contents of the route, once.
	//
	// Flutter: widgets/routes.dart ModalRoute.buildPage
	BuildPage(bc BuildContext, animation, secondaryAnimation animation.Animation[float64]) Widget
	// BuildTransitions builds the transition of the page; it is called every
	// time the animation changes.
	//
	// Flutter: widgets/routes.dart ModalRoute.buildTransitions
	BuildTransitions(bc BuildContext, animation, secondaryAnimation animation.Animation[float64], child Widget) Widget
	// DelegatedTransition is the transition that this route builds for the route
	// below it, or nil.
	//
	// Flutter: widgets/routes.dart ModalRoute.delegatedTransition
	DelegatedTransition() DelegatedTransitionBuilder
	// BarrierDismissible is whether tapping the barrier pops the route.
	//
	// Flutter: widgets/routes.dart ModalRoute.barrierDismissible
	BarrierDismissible() bool
	// SemanticsDismissible is whether the barrier can be dismissed by assistive
	// technology.
	//
	// Flutter: widgets/routes.dart ModalRoute.semanticsDismissible
	SemanticsDismissible() bool
	// BarrierColor is the color of the barrier, if any.
	//
	// Flutter: widgets/routes.dart ModalRoute.barrierColor
	BarrierColor() foundation.Opt[ui.Color]
	// BarrierLabel is the semantic label of the barrier.
	//
	// Flutter: widgets/routes.dart ModalRoute.barrierLabel
	BarrierLabel() string
	// BarrierCurve is the curve of the barrier color's fade.
	//
	// Flutter: widgets/routes.dart ModalRoute.barrierCurve
	BarrierCurve() animation.Curve
	// MaintainState is whether the route keeps its state while it is not on top.
	//
	// Flutter: widgets/routes.dart ModalRoute.maintainState
	MaintainState() bool
	// BuildModalBarrier builds the barrier of the route.
	//
	// Flutter: widgets/routes.dart ModalRoute.buildModalBarrier
	BuildModalBarrier() Widget
	// Offstage is whether the route is built but not shown.
	//
	// Flutter: widgets/routes.dart ModalRoute.offstage
	Offstage() bool
	// SetOffstage changes whether the route is built but not shown.
	//
	// Flutter: widgets/routes.dart ModalRoute.offstage
	SetOffstage(value bool)
	// SubtreeContext is the context of the subtree of the page, or nil.
	//
	// Flutter: widgets/routes.dart ModalRoute.subtreeContext
	SubtreeContext() BuildContext
	// CanPop is whether the route can be popped: there is an active route below
	// or it handles the pop itself.
	//
	// Flutter: widgets/routes.dart ModalRoute.canPop
	CanPop() bool
	// ImpliesAppBarDismissal is whether an app bar shows a close button.
	//
	// Flutter: widgets/routes.dart ModalRoute.impliesAppBarDismissal
	ImpliesAppBarDismissal() bool
	// FullscreenDialog is whether the route is a fullscreen dialog.
	//
	// Flutter: widgets/routes.dart ModalRoute.fullscreenDialog
	FullscreenDialog() bool
	// RegisterPopEntry adds a pop scope to the route.
	//
	// Flutter: widgets/routes.dart ModalRoute.registerPopEntry
	RegisterPopEntry(popEntry PopEntry)
	// UnregisterPopEntry removes a pop scope from the route.
	//
	// Flutter: widgets/routes.dart ModalRoute.unregisterPopEntry
	UnregisterPopEntry(popEntry PopEntry)
	modalBase() *ModalRouteBase
}

// ModalRouteOpts are the optional parameters of the modal route bases.
//
// Flutter: widgets/routes.dart ModalRoute.new
type ModalRouteOpts struct {
	RouteOpts
	// Filter is applied to everything below the route, behind the barrier.
	Filter ui.ImageFilter
}

// ModalRouteBase implements the parts of ModalRoute that are shared. Embed it and
// call InitModalRouteBase; implement BuildPage, BarrierDismissible, BarrierColor,
// BarrierLabel, MaintainState, Opaque and TransitionDuration.
//
// Flutter: widgets/routes.dart ModalRoute
type ModalRouteBase struct {
	TransitionRouteBase
	mself  ModalRoute
	filter ui.ImageFilter

	receivedTransition DelegatedTransitionBuilder
	offstage           bool
	animationProxy     *animation.ProxyAnimation
	secondaryProxy     *animation.ProxyAnimation
	popEntries         []PopEntry
	popEntryHandles    map[PopEntry]foundation.ListenerHandle

	history localHistory

	scopeKey      *GlobalKey[*modalScopeState]
	subtreeKey    *GlobalKey[State]
	storageBucket *PageStorageBucket

	modalBarrier    *OverlayEntry
	modalScopeEntry *OverlayEntry
	modalScopeCache Widget
}

// InitModalRouteBase sets the outer value of the route.
//
// Flutter: widgets/routes.dart ModalRoute.new
func InitModalRouteBase(b *ModalRouteBase, self ModalRoute, opts ModalRouteOpts) {
	InitTransitionRouteBase(&b.TransitionRouteBase, self, opts.RouteOpts)
	b.mself = self
	b.filter = opts.Filter
	b.scopeKey = NewGlobalKey[*modalScopeState]()
	b.subtreeKey = NewGlobalKey[State]()
	b.storageBucket = NewPageStorageBucket()
	b.popEntryHandles = map[PopEntry]foundation.ListenerHandle{}
	b.history = localHistory{route: self}
}

func (b *ModalRouteBase) modalBase() *ModalRouteBase { return b }

func modalRouteOf(bc BuildContext, aspect any) ModalRoute {
	status, ok := DependOnModel[*modalScopeStatus](bc, aspect)
	if !ok {
		return nil
	}
	return status.route
}

// ModalRouteOf returns the modal route that the context is in, or nil. The
// context depends on the route, so it is rebuilt whenever the route changes.
//
// Flutter: widgets/routes.dart ModalRoute.of
func ModalRouteOf(bc BuildContext) ModalRoute { return modalRouteOf(bc, nil) }

// ModalRouteIsCurrentOf is whether the route that the context is in is the
// current one; none without a route. The context is rebuilt when it changes.
//
// Flutter: widgets/routes.dart ModalRoute.isCurrentOf
func ModalRouteIsCurrentOf(bc BuildContext) foundation.Opt[bool] {
	if r := modalRouteOf(bc, modalRouteAspectIsCurrent); r != nil {
		return foundation.Some(r.IsCurrent())
	}
	return foundation.None[bool]()
}

// ModalRouteCanPopOf is whether the route that the context is in can pop; none
// without a route.
//
// Flutter: widgets/routes.dart ModalRoute.canPopOf
func ModalRouteCanPopOf(bc BuildContext) foundation.Opt[bool] {
	if r := modalRouteOf(bc, modalRouteAspectCanPop); r != nil {
		return foundation.Some(r.CanPop())
	}
	return foundation.None[bool]()
}

// ModalRouteSettingsOf returns the settings of the route that the context is in,
// or nil.
//
// Flutter: widgets/routes.dart ModalRoute.settingsOf
func ModalRouteSettingsOf(bc BuildContext) RouteSettings {
	if r := modalRouteOf(bc, modalRouteAspectSettings); r != nil {
		return r.Settings()
	}
	return nil
}

// ModalRouteIsActiveOf is whether the route that the context is in is active;
// none without a route.
//
// Flutter: widgets/routes.dart ModalRoute.isActiveOf
func ModalRouteIsActiveOf(bc BuildContext) foundation.Opt[bool] {
	if r := modalRouteOf(bc, modalRouteAspectIsActive); r != nil {
		return foundation.Some(r.IsActive())
	}
	return foundation.None[bool]()
}

// ModalRouteIsFirstOf is whether the route that the context is in is the first
// one; none without a route.
//
// Flutter: widgets/routes.dart ModalRoute.isFirstOf
func ModalRouteIsFirstOf(bc BuildContext) foundation.Opt[bool] {
	if r := modalRouteOf(bc, modalRouteAspectIsFirst); r != nil {
		return foundation.Some(r.IsFirst())
	}
	return foundation.None[bool]()
}

// ModalRouteOpaqueOf is whether the route that the context is in is opaque; none
// without a route.
//
// Flutter: widgets/routes.dart ModalRoute.opaqueOf
func ModalRouteOpaqueOf(bc BuildContext) foundation.Opt[bool] {
	if r := modalRouteOf(bc, modalRouteAspectOpaque); r != nil {
		return foundation.Some(r.Opaque())
	}
	return foundation.None[bool]()
}

// ModalRoutePopDispositionOf is the pop disposition of the route that the
// context is in; none without a route.
//
// Flutter: widgets/routes.dart ModalRoute.popDispositionOf
func ModalRoutePopDispositionOf(bc BuildContext) foundation.Opt[RoutePopDisposition] {
	if r := modalRouteOf(bc, modalRoutePopDispositionAspect()); r != nil {
		return foundation.Some(r.PopDisposition())
	}
	return foundation.None[RoutePopDisposition]()
}

func modalRoutePopDispositionAspect() modalRouteAspect { return modalRouteAspectPopDisposition }

// ModalRouteWithName returns a predicate that is true for the modal routes that
// have the name and do not handle pops internally.
//
// Flutter: widgets/routes.dart ModalRoute.withName
func ModalRouteWithName(name string) RoutePredicate {
	return func(route Route) bool {
		_, isModal := route.(ModalRoute)
		return !route.WillHandlePopInternally() && isModal && route.Settings().Name() == name
	}
}

// SetState runs fn and rebuilds the scope of the route, if it is in the tree.
//
// Flutter: widgets/routes.dart ModalRoute.setState
func (b *ModalRouteBase) SetState(fn func()) {
	if state, ok := b.scopeKey.CurrentState(); ok {
		state.routeSetState(fn)
	} else {
		// The route isn't visible, so its setState method isn't called, but the
		// callback still must run, otherwise the state of the route is not updated.
		fn()
	}
}

// Filter is applied to everything below the route, or nil.
//
// Flutter: widgets/routes.dart ModalRoute.filter
func (b *ModalRouteBase) Filter() ui.ImageFilter { return b.filter }

// BuildTransitions returns the child.
//
// Flutter: widgets/routes.dart ModalRoute.buildTransitions
func (b *ModalRouteBase) BuildTransitions(bc BuildContext, animation, secondaryAnimation animation.Animation[float64], child Widget) Widget {
	return child
}

// DelegatedTransition is nil.
//
// Flutter: widgets/routes.dart ModalRoute.delegatedTransition
func (b *ModalRouteBase) DelegatedTransition() DelegatedTransitionBuilder { return nil }

// ReceivedTransition is the transition that the route above delegated to this
// one, or nil.
//
// Flutter: widgets/routes.dart ModalRoute.receivedTransition
func (b *ModalRouteBase) ReceivedTransition() DelegatedTransitionBuilder { return b.receivedTransition }

// SetReceivedTransition sets the transition that the route above delegated to
// this one. It is for tests.
//
// Flutter: widgets/routes.dart ModalRoute.receivedTransition
func (b *ModalRouteBase) SetReceivedTransition(t DelegatedTransitionBuilder) {
	b.receivedTransition = t
}

// buildFlexibleTransitions wraps the transitions of this route with the
// delegated transition of the route above, when there is one.
func (b *ModalRouteBase) buildFlexibleTransitions(bc BuildContext, a, secondary animation.Animation[float64], child Widget) Widget {
	if b.receivedTransition == nil || secondary.Status().IsDismissed() {
		return b.mself.BuildTransitions(bc, a, secondary, child)
	}
	// A static proxy animation suppresses the original secondary transition.
	proxy := animation.NewProxyAnimation(nil)
	proxiedOriginalTransitions := b.mself.BuildTransitions(bc, a, proxy, child)
	// If the received transition returns nil, the original transitions are
	// returned with the secondary animation still proxied. This keeps a
	// desynchronized animation from playing.
	if w := b.receivedTransition(bc, a, secondary, b.mself.AllowSnapshotting(), proxiedOriginalTransitions); w != nil {
		return w
	}
	return proxiedOriginalTransitions
}

// Install creates the proxies of the animations.
//
// Flutter: widgets/routes.dart ModalRoute.install
func (b *ModalRouteBase) Install() {
	b.TransitionRouteBase.Install()
	b.animationProxy = animation.NewProxyAnimation(b.TransitionRouteBase.Animation())
	b.secondaryProxy = animation.NewProxyAnimation(b.TransitionRouteBase.SecondaryAnimation())
}

// DidPush pushes the route; focus is requested for the route here in step 10.5.
//
// Flutter: widgets/routes.dart ModalRoute.didPush
func (b *ModalRouteBase) DidPush() *scheduler.TickerFuture {
	// TODO(step 10.5): when the scope exists and the navigator requests focus, set
	// the focus scope node of the route as the first focus of the navigator.
	return b.TransitionRouteBase.DidPush()
}

// DidAdd adds the route; focus is requested for the route here in step 10.5.
//
// Flutter: widgets/routes.dart ModalRoute.didAdd
func (b *ModalRouteBase) DidAdd() {
	// TODO(step 10.5): set the first focus, as DidPush does.
	b.TransitionRouteBase.DidAdd()
}

// BarrierDismissible is false by default through the page route; modal routes
// implement it.

// SemanticsDismissible is true.
//
// Flutter: widgets/routes.dart ModalRoute.semanticsDismissible
func (b *ModalRouteBase) SemanticsDismissible() bool { return true }

// BarrierCurve is Curves.ease.
//
// Flutter: widgets/routes.dart ModalRoute.barrierCurve
func (b *ModalRouteBase) BarrierCurve() animation.Curve { return animation.Curves.Ease }

// PopGestureInProgress is whether a user gesture that may pop the route is going
// on.
//
// Flutter: widgets/routes.dart ModalRoute.popGestureInProgress
func (b *ModalRouteBase) PopGestureInProgress() bool { return b.navigator.UserGestureInProgress() }

// PopGestureEnabled is whether a back gesture may pop the route: it is not the
// first, does not handle pops itself, cannot be vetoed, and is not animating.
//
// Flutter: widgets/routes.dart ModalRoute.popGestureEnabled
func (b *ModalRouteBase) PopGestureEnabled() bool {
	// If there is nothing to go back to, the gesture is not supported.
	if b.self.IsFirst() {
		return false
	}
	// If the route would not pop when popped, the gesture would be confusing, or
	// would skip internal routes.
	if b.self.WillHandlePopInternally() {
		return false
	}
	// If attempts to dismiss the route might be vetoed, such as in a page with
	// forms, the route can't be dismissed with a swipe.
	if b.self.PopDisposition() == RoutePopDispositionDoNotPop {
		return false
	}
	// A route that is animating already can't be swiped.
	if !b.mself.Animation().Status().IsCompleted() {
		return false
	}
	// A back gesture would be welcome.
	return true
}

// Offstage is whether the route is built but not shown.
//
// Flutter: widgets/routes.dart ModalRoute.offstage
func (b *ModalRouteBase) Offstage() bool { return b.offstage }

// SetOffstage builds the route without showing it, with its animations at the end
// (animation complete, secondary animation dismissed).
//
// Flutter: widgets/routes.dart ModalRoute.offstage
func (b *ModalRouteBase) SetOffstage(value bool) {
	if b.offstage == value {
		return
	}
	b.SetState(func() { b.offstage = value })
	if b.offstage {
		b.animationProxy.SetParent(animation.AlwaysComplete)
		b.secondaryProxy.SetParent(animation.AlwaysDismissed)
	} else {
		b.animationProxy.SetParent(b.TransitionRouteBase.Animation())
		b.secondaryProxy.SetParent(b.TransitionRouteBase.SecondaryAnimation())
	}
	b.self.ChangedInternalState()
}

// SubtreeContext is the context of the subtree of the page, or nil.
//
// Flutter: widgets/routes.dart ModalRoute.subtreeContext
func (b *ModalRouteBase) SubtreeContext() BuildContext {
	return b.subtreeKey.CurrentContext()
}

// Animation is the proxy of the animation of the route; it is the same animation,
// except while the route is offstage.
//
// Flutter: widgets/routes.dart ModalRoute.animation
func (b *ModalRouteBase) Animation() animation.Animation[float64] {
	if b.animationProxy == nil {
		return nil
	}
	return b.animationProxy
}

// SecondaryAnimation is the proxy of the secondary animation of the route.
//
// Flutter: widgets/routes.dart ModalRoute.secondaryAnimation
func (b *ModalRouteBase) SecondaryAnimation() animation.Animation[float64] {
	if b.secondaryProxy == nil {
		return nil
	}
	return b.secondaryProxy
}

// PopDisposition is do-not-pop when a pop scope blocks pops, else the
// disposition of the local history route.
//
// Flutter: widgets/routes.dart ModalRoute.popDisposition
func (b *ModalRouteBase) PopDisposition() RoutePopDisposition {
	for _, popEntry := range b.popEntries {
		if !popEntry.CanPopNotifier().Value() {
			return RoutePopDispositionDoNotPop
		}
	}
	// LocalHistoryRoute.popDisposition
	if b.mself.WillHandlePopInternally() {
		return RoutePopDispositionPop
	}
	return b.RouteBase.PopDisposition()
}

// OnPopInvokedWithResult tells the pop scopes.
//
// Flutter: widgets/routes.dart ModalRoute.onPopInvokedWithResult
func (b *ModalRouteBase) OnPopInvokedWithResult(didPop bool, result any) {
	for _, popEntry := range slices.Clone(b.popEntries) {
		popEntry.OnPopInvokedWithResult(didPop, result)
	}
	b.RouteBase.OnPopInvokedWithResult(didPop, result)
}

// RegisterPopEntry adds a pop scope to the route.
//
// Flutter: widgets/routes.dart ModalRoute.registerPopEntry
func (b *ModalRouteBase) RegisterPopEntry(popEntry PopEntry) {
	if !slices.Contains(b.popEntries, popEntry) {
		b.popEntries = append(b.popEntries, popEntry)
	}
	b.popEntryHandles[popEntry] = popEntry.CanPopNotifier().AddListener(b.maybeDispatchNavigationNotification)
	b.maybeDispatchNavigationNotification()
}

// UnregisterPopEntry removes a pop scope from the route.
//
// Flutter: widgets/routes.dart ModalRoute.unregisterPopEntry
func (b *ModalRouteBase) UnregisterPopEntry(popEntry PopEntry) {
	if i := slices.Index(b.popEntries, popEntry); i >= 0 {
		b.popEntries = slices.Delete(b.popEntries, i, i+1)
	}
	if handle, ok := b.popEntryHandles[popEntry]; ok {
		popEntry.CanPopNotifier().RemoveListener(handle)
		delete(b.popEntryHandles, popEntry)
	}
	b.maybeDispatchNavigationNotification()
}

func (b *ModalRouteBase) schedulerBinding() *scheduler.SchedulerBinding {
	if binding := b.binding(); binding != nil {
		return binding.Scheduler()
	}
	return nil
}

func (b *ModalRouteBase) maybeDispatchNavigationNotification() {
	if !b.self.IsCurrent() {
		return
	}
	// canPop indicates that the originator of the notification can handle a pop.
	// In the case of PopScope, it handles pops when canPop is false, hence the
	// seemingly backward logic.
	notification := NavigationNotification{CanHandlePop: b.self.PopDisposition() == RoutePopDispositionDoNotPop}
	sched := b.schedulerBinding()
	if sched == nil {
		return
	}
	// Avoid dispatching a notification in the middle of a build.
	if sched.SchedulerPhase() == scheduler.SchedulerPhasePostFrameCallbacks {
		if ctx := b.mself.SubtreeContext(); ctx != nil {
			DispatchNotification(ctx, notification)
		}
		return
	}
	sched.AddPostFrameCallback(func(time.Duration) {
		ctx := b.mself.SubtreeContext()
		if ctx == nil || !ctx.Mounted() {
			return
		}
		DispatchNotification(ctx, notification)
	})
}

func sameDelegatedTransition(a, b DelegatedTransitionBuilder) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer()
}

func (b *ModalRouteBase) updateReceivedTransition(nextRoute Route) {
	if next, ok := nextRoute.(ModalRoute); ok && b.mself.CanTransitionTo(next) &&
		!sameDelegatedTransition(next.DelegatedTransition(), b.mself.DelegatedTransition()) {
		b.receivedTransition = next.DelegatedTransition()
	} else {
		b.receivedTransition = nil
	}
}

// DidChangePrevious tells the scope that the state changed.
//
// Flutter: widgets/routes.dart ModalRoute.didChangePrevious
func (b *ModalRouteBase) DidChangePrevious(previousRoute Route) {
	b.TransitionRouteBase.DidChangePrevious(previousRoute)
	b.self.ChangedInternalState()
}

// DidChangeNext receives the delegated transition of the next route.
//
// Flutter: widgets/routes.dart ModalRoute.didChangeNext
func (b *ModalRouteBase) DidChangeNext(nextRoute Route) {
	b.updateReceivedTransition(nextRoute)
	b.TransitionRouteBase.DidChangeNext(nextRoute)
	b.self.ChangedInternalState()
}

// DidPopNext receives the delegated transition of the next route.
//
// Flutter: widgets/routes.dart ModalRoute.didPopNext
func (b *ModalRouteBase) DidPopNext(nextRoute Route) {
	b.updateReceivedTransition(nextRoute)
	b.TransitionRouteBase.DidPopNext(nextRoute)
	b.self.ChangedInternalState()
	b.maybeDispatchNavigationNotification()
}

// ChangedInternalState rebuilds the scope and the barrier.
//
// Flutter: widgets/routes.dart ModalRoute.changedInternalState
func (b *ModalRouteBase) ChangedInternalState() {
	b.TransitionRouteBase.ChangedInternalState()
	// There is no need to mark dirty if this method is called during the build
	// phase.
	sched := b.schedulerBinding()
	if sched == nil || sched.SchedulerPhase() != scheduler.SchedulerPhasePersistentCallbacks {
		b.SetState(func() { /* the internal state changed already */ })
		if b.modalBarrier != nil {
			b.modalBarrier.MarkNeedsBuild()
		}
	}
	if b.modalScopeEntry != nil {
		b.modalScopeEntry.SetMaintainState(b.mself.MaintainState())
	}
}

// ChangedExternalState rebuilds the barrier and the page.
//
// Flutter: widgets/routes.dart ModalRoute.changedExternalState
func (b *ModalRouteBase) ChangedExternalState() {
	b.TransitionRouteBase.ChangedExternalState()
	if b.modalBarrier != nil {
		b.modalBarrier.MarkNeedsBuild()
	}
	if state, ok := b.scopeKey.CurrentState(); ok {
		state.forceRebuildPage()
	}
}

// CanPop is whether there is an active route below or the route handles the pop
// itself.
//
// Flutter: widgets/routes.dart ModalRoute.canPop
func (b *ModalRouteBase) CanPop() bool {
	return b.self.HasActiveRouteBelow() || b.self.WillHandlePopInternally()
}

// ImpliesAppBarDismissal is whether an app bar shows a close button.
//
// Flutter: widgets/routes.dart ModalRoute.impliesAppBarDismissal
func (b *ModalRouteBase) ImpliesAppBarDismissal() bool {
	return b.self.HasActiveRouteBelow() || b.history.impliesAppBarDismissalCount > 0
}

// FullscreenDialog is false.
//
// Flutter: widgets/routes.dart ModalRoute.fullscreenDialog
func (b *ModalRouteBase) FullscreenDialog() bool { return false }

func (b *ModalRouteBase) buildModalBarrier(bc BuildContext) Widget {
	barrier := b.mself.BuildModalBarrier()
	if b.filter != nil {
		barrier = NewBackdropFilter(b.filter, barrier)
	}
	barrier = NewIgnorePointer(barrier, IgnorePointerOpts{
		// ChangedInternalState is called when the animation status updates.
		// Dismissed is possible during a manual pop gesture.
		Ignoring: foundation.Some(!b.mself.Animation().Status().IsForwardOrCompleted()),
	})
	if b.mself.SemanticsDismissible() && b.mself.BarrierDismissible() {
		// To be sorted after the modal scope.
		barrier = NewSemantics(barrier, SemanticsOpts{SortKey: semantics.NewOrdinalSortKey(1.0)})
	}
	return barrier
}

// BuildModalBarrier builds the barrier: a ModalBarrier, or an AnimatedModalBarrier
// when the route has a barrier color.
//
// Flutter: widgets/routes.dart ModalRoute.buildModalBarrier
func (b *ModalRouteBase) BuildModalBarrier() Widget {
	if color, ok := b.mself.BarrierColor().Get(); ok && color.Alpha() != 0 && !b.offstage {
		// ChangedInternalState is called if the barrier color or offstage updates.
		if foundation.DebugMode && color == color.WithOpacity(0) {
			panic(errors.New("ModalRouteBarrierColorInvisible", "the barrier color of a modal route can not be invisible when its alpha is not zero"))
		}
		colorAnimation := animation.Drive[foundation.Opt[ui.Color]](b.mself.Animation(),
			animation.NewColorTween(foundation.Some(color.WithOpacity(0)), foundation.Some(color)).
				Chain(animation.NewCurveTween(b.mself.BarrierCurve())))
		return NewAnimatedModalBarrier(colorAnimation, AnimatedModalBarrierOpts{
			Dismissible:                 foundation.Some(b.mself.BarrierDismissible()),
			SemanticsLabel:              b.mself.BarrierLabel(),
			BarrierSemanticsDismissible: foundation.Some(b.mself.SemanticsDismissible()),
		})
	}
	return NewModalBarrier(ModalBarrierOpts{
		Dismissible:                 foundation.Some(b.mself.BarrierDismissible()),
		SemanticsLabel:              b.mself.BarrierLabel(),
		BarrierSemanticsDismissible: foundation.Some(b.mself.SemanticsDismissible()),
	})
}

// buildModalScope caches the part of the modal scope that doesn't change from
// frame to frame, to minimize the amount of building.
func (b *ModalRouteBase) buildModalScope(bc BuildContext) Widget {
	if b.modalScopeCache == nil {
		// To be sorted before the modal barrier.
		b.modalScopeCache = NewSemantics(
			newModalScope(b.scopeKey, b.mself),
			SemanticsOpts{SortKey: semantics.NewOrdinalSortKey(0.0)},
		)
	}
	return b.modalScopeCache
}

// CreateOverlayEntries creates the barrier entry and the entry of the page.
//
// Flutter: widgets/routes.dart ModalRoute.createOverlayEntries
func (b *ModalRouteBase) CreateOverlayEntries() []*OverlayEntry {
	b.modalBarrier = NewOverlayEntry(b.buildModalBarrier)
	b.modalScopeEntry = NewOverlayEntry(b.buildModalScope, OverlayEntryOpts{
		MaintainState:  b.mself.MaintainState(),
		CanSizeOverlay: b.mself.Opaque(),
	})
	return []*OverlayEntry{b.modalBarrier, b.modalScopeEntry}
}

// AddLocalHistoryEntry adds a local history entry; popping the route removes
// the entries first.
//
// Flutter: widgets/routes.dart LocalHistoryRoute.addLocalHistoryEntry
func (b *ModalRouteBase) AddLocalHistoryEntry(entry *LocalHistoryEntry) { b.history.add(entry) }

// RemoveLocalHistoryEntry removes a local history entry.
//
// Flutter: widgets/routes.dart LocalHistoryRoute.removeLocalHistoryEntry
func (b *ModalRouteBase) RemoveLocalHistoryEntry(entry *LocalHistoryEntry) { b.history.remove(entry) }

// DidPop removes the last local history entry, and does not pop the route,
// while there are some.
//
// Flutter: widgets/routes.dart LocalHistoryRoute.didPop
func (b *ModalRouteBase) DidPop(result any) bool {
	if b.history.didPop() {
		return false
	}
	return b.TransitionRouteBase.DidPop(result)
}

// WillHandlePopInternally is whether the route has local history entries.
//
// Flutter: widgets/routes.dart LocalHistoryRoute.willHandlePopInternally
func (b *ModalRouteBase) WillHandlePopInternally() bool { return b.history.willHandlePopInternally() }

// String describes the route.
//
// Flutter: widgets/routes.dart ModalRoute.toString
func (b *ModalRouteBase) String() string {
	return fmt.Sprintf("%s(%v, animation: %v)", foundation.RuntimeTypeName(b.self), describeSettings(b.settings), b.animation)
}

func describeSettings(s RouteSettings) string {
	return describeRouteSettings("RouteSettings", s.Name(), s.Arguments())
}

// PopupRoute is a modal route that overlays a widget over the current route.
//
// Flutter: widgets/routes.dart PopupRoute
type PopupRoute interface {
	ModalRoute
}

// PopupRouteBase implements PopupRoute: it is not opaque, keeps its state and
// can't be snapshotted. Embed it and call InitPopupRouteBase.
//
// Flutter: widgets/routes.dart PopupRoute
type PopupRouteBase struct {
	ModalRouteBase
}

// InitPopupRouteBase sets the outer value of the route.
//
// Flutter: widgets/routes.dart PopupRoute.new
func InitPopupRouteBase(b *PopupRouteBase, self PopupRoute, opts ModalRouteOpts) {
	InitModalRouteBase(&b.ModalRouteBase, self, opts)
}

// Opaque is false.
//
// Flutter: widgets/routes.dart PopupRoute.opaque
func (b *PopupRouteBase) Opaque() bool { return false }

// MaintainState is true.
//
// Flutter: widgets/routes.dart PopupRoute.maintainState
func (b *PopupRouteBase) MaintainState() bool { return true }

// AllowSnapshotting is false.
//
// Flutter: widgets/routes.dart PopupRoute.allowSnapshotting
func (b *PopupRouteBase) AllowSnapshotting() bool { return false }

// RouteAware is an interface for objects that are aware of their current Route.
// Subscribe it to a RouteObserver.
//
// Flutter: widgets/routes.dart RouteAware
type RouteAware interface {
	// DidPopNext is called when the top route has been popped off, and the
	// current route shows up.
	//
	// Flutter: widgets/routes.dart RouteAware.didPopNext
	DidPopNext()
	// DidPush is called when the current route has been pushed.
	//
	// Flutter: widgets/routes.dart RouteAware.didPush
	DidPush()
	// DidPop is called when the current route has been popped off.
	//
	// Flutter: widgets/routes.dart RouteAware.didPop
	DidPop()
	// DidPushNext is called when a new route has been pushed, and the current
	// route is no longer visible.
	//
	// Flutter: widgets/routes.dart RouteAware.didPushNext
	DidPushNext()
}

// RouteAwareBase implements RouteAware with methods that do nothing. Embed it.
//
// Flutter: widgets/routes.dart RouteAware
type RouteAwareBase struct{}

// DidPopNext does nothing.
//
// Flutter: widgets/routes.dart RouteAware.didPopNext
func (RouteAwareBase) DidPopNext() {}

// DidPush does nothing.
//
// Flutter: widgets/routes.dart RouteAware.didPush
func (RouteAwareBase) DidPush() {}

// DidPop does nothing.
//
// Flutter: widgets/routes.dart RouteAware.didPop
func (RouteAwareBase) DidPop() {}

// DidPushNext does nothing.
//
// Flutter: widgets/routes.dart RouteAware.didPushNext
func (RouteAwareBase) DidPushNext() {}

// RouteObserver informs RouteAware subscribers about the routes of type R that
// they are subscribed to.
//
// Flutter: widgets/routes.dart RouteObserver
type RouteObserver[R interface {
	Route
	comparable
}] struct {
	NavigatorObserverBase
	listeners map[R][]RouteAware
	order     []R
}

// NewRouteObserver creates an observer.
//
// Flutter: widgets/routes.dart RouteObserver.new
func NewRouteObserver[R interface {
	Route
	comparable
}]() *RouteObserver[R] {
	return &RouteObserver[R]{listeners: map[R][]RouteAware{}}
}

// DebugObservingRoute is whether the route has subscribers; in debug builds
// only.
//
// Flutter: widgets/routes.dart RouteObserver.debugObservingRoute
func (o *RouteObserver[R]) DebugObservingRoute(route R) bool {
	if !foundation.DebugMode {
		return false
	}
	_, ok := o.listeners[route]
	return ok
}

// Subscribe registers the route aware to be informed about changes to the route;
// it is told that the route was pushed.
//
// Flutter: widgets/routes.dart RouteObserver.subscribe
func (o *RouteObserver[R]) Subscribe(routeAware RouteAware, route R) {
	subscribers, ok := o.listeners[route]
	if !ok {
		o.order = append(o.order, route)
	}
	if !slices.Contains(subscribers, routeAware) {
		o.listeners[route] = append(subscribers, routeAware)
		routeAware.DidPush()
	} else {
		o.listeners[route] = subscribers
	}
}

// Unsubscribe removes the route aware from every route.
//
// Flutter: widgets/routes.dart RouteObserver.unsubscribe
func (o *RouteObserver[R]) Unsubscribe(routeAware RouteAware) {
	for _, route := range slices.Clone(o.order) {
		subscribers, ok := o.listeners[route]
		if !ok {
			continue
		}
		if i := slices.Index(subscribers, routeAware); i >= 0 {
			subscribers = slices.Delete(slices.Clone(subscribers), i, i+1)
		}
		if len(subscribers) == 0 {
			delete(o.listeners, route)
			o.order = slices.DeleteFunc(o.order, func(r R) bool { return r == route })
		} else {
			o.listeners[route] = subscribers
		}
	}
}

// DidPop tells the subscribers of the previous route that its next route was
// popped, and the subscribers of the route that it was popped.
//
// Flutter: widgets/routes.dart RouteObserver.didPop
func (o *RouteObserver[R]) DidPop(route, previousRoute Route) {
	r, okRoute := route.(R)
	p, okPrevious := previousRoute.(R)
	if okRoute && okPrevious {
		for _, routeAware := range slices.Clone(o.listeners[p]) {
			routeAware.DidPopNext()
		}
		for _, routeAware := range slices.Clone(o.listeners[r]) {
			routeAware.DidPop()
		}
	}
}

// DidPush tells the subscribers of the previous route that a new route was pushed.
//
// Flutter: widgets/routes.dart RouteObserver.didPush
func (o *RouteObserver[R]) DidPush(route, previousRoute Route) {
	_, okRoute := route.(R)
	p, okPrevious := previousRoute.(R)
	if okRoute && okPrevious {
		for _, routeAware := range slices.Clone(o.listeners[p]) {
			routeAware.DidPushNext()
		}
	}
}

// RawDialogRouteOpts are the optional parameters of NewRawDialogRoute.
//
// Flutter: widgets/routes.dart RawDialogRoute.new
type RawDialogRouteOpts struct {
	ModalRouteOpts
	// BarrierDismissible is whether tapping the barrier pops the route; the
	// default is true.
	BarrierDismissible foundation.Opt[bool]
	// BarrierColor is the color of the barrier; the default is 0x80000000.
	BarrierColor foundation.Opt[ui.Color]
	// NoBarrierColor makes the barrier colorless.
	NoBarrierColor bool
	// BarrierLabel is the semantic label of the barrier.
	BarrierLabel string
	// TransitionDuration is the duration of the transition; the default is 200 ms.
	TransitionDuration foundation.Opt[time.Duration]
	// TransitionBuilder builds the transition of the dialog; the default is a
	// FadeTransition.
	TransitionBuilder RouteTransitionsBuilder
	// FullscreenDialog is whether the dialog is fullscreen.
	FullscreenDialog bool
}

// RawDialogRoute is a general dialog route which allows for customization of the
// dialog popup. It is used by ShowGeneralDialog.
//
// Flutter: widgets/routes.dart RawDialogRoute
type RawDialogRoute struct {
	PopupRouteBase
	pageBuilder        RoutePageBuilder
	barrierDismissible bool
	barrierLabel       string
	barrierColor       foundation.Opt[ui.Color]
	transitionDuration time.Duration
	transitionBuilder  RouteTransitionsBuilder
	fullscreenDialog   bool
}

// NewRawDialogRoute creates a dialog route.
//
// Flutter: widgets/routes.dart RawDialogRoute.new
func NewRawDialogRoute(pageBuilder RoutePageBuilder, opts ...RawDialogRouteOpts) *RawDialogRoute {
	o := firstOpts(opts)
	r := &RawDialogRoute{
		pageBuilder:        pageBuilder,
		barrierDismissible: o.BarrierDismissible.OrElse(true),
		barrierLabel:       o.BarrierLabel,
		barrierColor:       o.BarrierColor,
		transitionDuration: o.TransitionDuration.OrElse(200 * time.Millisecond),
		transitionBuilder:  o.TransitionBuilder,
		fullscreenDialog:   o.FullscreenDialog,
	}
	if !o.BarrierColor.IsSome() && !o.NoBarrierColor {
		r.barrierColor = foundation.Some(ui.NewColor(0x80000000))
	}
	InitPopupRouteBase(&r.PopupRouteBase, r, o.ModalRouteOpts)
	return r
}

// BarrierDismissible is whether tapping the barrier pops the route.
//
// Flutter: widgets/routes.dart RawDialogRoute.barrierDismissible
func (r *RawDialogRoute) BarrierDismissible() bool { return r.barrierDismissible }

// BarrierLabel is the semantic label of the barrier.
//
// Flutter: widgets/routes.dart RawDialogRoute.barrierLabel
func (r *RawDialogRoute) BarrierLabel() string { return r.barrierLabel }

// BarrierColor is the color of the barrier.
//
// Flutter: widgets/routes.dart RawDialogRoute.barrierColor
func (r *RawDialogRoute) BarrierColor() foundation.Opt[ui.Color] { return r.barrierColor }

// TransitionDuration is the duration of the transition.
//
// Flutter: widgets/routes.dart RawDialogRoute.transitionDuration
func (r *RawDialogRoute) TransitionDuration() time.Duration { return r.transitionDuration }

// FullscreenDialog is whether the dialog is fullscreen.
//
// Flutter: widgets/routes.dart RawDialogRoute.fullscreenDialog
func (r *RawDialogRoute) FullscreenDialog() bool { return r.fullscreenDialog }

// BuildPage builds the page with the page builder, in a semantics scope.
//
// Deviation: Flutter wraps the page in a DisplayFeatureSubScreen with the anchor
// point; Glutter has no display features (SPEC §12.10).
//
// Flutter: widgets/routes.dart RawDialogRoute.buildPage
func (r *RawDialogRoute) BuildPage(bc BuildContext, a, secondary animation.Animation[float64]) Widget {
	return NewSemantics(r.pageBuilder(bc, a, secondary), SemanticsOpts{
		ScopesRoute:        foundation.Some(true),
		ExplicitChildNodes: true,
	})
}

// BuildTransitions builds the transition: a fade by default.
//
// Flutter: widgets/routes.dart RawDialogRoute.buildTransitions
func (r *RawDialogRoute) BuildTransitions(bc BuildContext, a, secondary animation.Animation[float64], child Widget) Widget {
	if r.transitionBuilder == nil {
		// Some default transition.
		return NewFadeTransition(a, child)
	}
	return r.transitionBuilder(bc, a, secondary, child)
}

// ShowGeneralDialogOpts are the optional parameters of ShowGeneralDialog.
//
// Flutter: widgets/routes.dart showGeneralDialog
type ShowGeneralDialogOpts struct {
	// BarrierDismissible is whether tapping the barrier pops the dialog; the
	// default is false.
	BarrierDismissible bool
	// BarrierLabel is the semantic label of the barrier; required when the
	// barrier is dismissible.
	BarrierLabel string
	// BarrierColor is the color of the barrier; the default is 0x80000000.
	BarrierColor foundation.Opt[ui.Color]
	// TransitionDuration is the duration of the transition; the default is 200 ms.
	TransitionDuration foundation.Opt[time.Duration]
	// TransitionBuilder builds the transition of the dialog.
	TransitionBuilder RouteTransitionsBuilder
	// UseRootNavigator pushes the dialog onto the root navigator; the default is
	// true.
	UseRootNavigator foundation.Opt[bool]
	// FullscreenDialog is whether the dialog is fullscreen.
	FullscreenDialog bool
	// RouteSettings are the settings of the route.
	RouteSettings RouteSettings
	// RequestFocus is whether the dialog requests the focus.
	RequestFocus foundation.Opt[bool]
}

// ShowGeneralDialog displays a dialog above the current contents of the app. It
// returns a channel that gets the result when the dialog is popped. In debug
// builds it panics with the tag ShowGeneralDialogBarrierLabelMissing when the
// barrier is dismissible and has no label.
//
// Flutter: widgets/routes.dart showGeneralDialog
func ShowGeneralDialog(bc BuildContext, pageBuilder RoutePageBuilder, opts ...ShowGeneralDialogOpts) <-chan any {
	o := firstOpts(opts)
	if foundation.DebugMode && o.BarrierDismissible && o.BarrierLabel == "" {
		panic(errors.New("ShowGeneralDialogBarrierLabelMissing", "a dismissible dialog barrier needs a barrierLabel"))
	}
	route := NewRawDialogRoute(pageBuilder, RawDialogRouteOpts{
		ModalRouteOpts:     ModalRouteOpts{RouteOpts: RouteOpts{Settings: o.RouteSettings, RequestFocus: o.RequestFocus}},
		BarrierDismissible: foundation.Some(o.BarrierDismissible),
		BarrierColor:       foundation.Some(o.BarrierColor.OrElse(ui.NewColor(0x80000000))),
		BarrierLabel:       o.BarrierLabel,
		TransitionDuration: o.TransitionDuration,
		TransitionBuilder:  o.TransitionBuilder,
		FullscreenDialog:   o.FullscreenDialog,
	})
	return NavigatorOf(bc, NavigatorOfOpts{RootNavigator: o.UseRootNavigator.OrElse(true)}).Push(route)
}
