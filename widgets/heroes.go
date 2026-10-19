package widgets

import (
	"fmt"
	"slices"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// CreateRectTween creates the tween that a hero flight follows from one rect to
// another.
//
// Flutter: widgets/heroes.dart CreateRectTween
type CreateRectTween func(begin, end foundation.Opt[ui.Rect]) *animation.Tween[foundation.Opt[ui.Rect]]

// HeroPlaceholderBuilder builds the widget that is in the place of a hero while
// it is in flight: heroSize is the size of the hero and child is its child.
//
// Flutter: widgets/heroes.dart HeroPlaceholderBuilder
type HeroPlaceholderBuilder func(bc BuildContext, heroSize ui.Size, child Widget) Widget

// HeroFlightShuttleBuilder builds the widget that flies between two heroes.
//
// Flutter: widgets/heroes.dart HeroFlightShuttleBuilder
type HeroFlightShuttleBuilder func(flightContext BuildContext, animation animation.Animation[float64], flightDirection HeroFlightDirection, fromHeroContext, toHeroContext BuildContext) Widget

// HeroFlightDirection is the direction of a hero flight: push when a route is
// pushed onto the navigator, and pop when it is popped.
//
// Flutter: widgets/heroes.dart HeroFlightDirection
type HeroFlightDirection int

const (
	// HeroFlightDirectionUnspecified is no flight.
	HeroFlightDirectionUnspecified HeroFlightDirection = iota
	// HeroFlightDirectionPush is the flight of a route that is pushed.
	//
	// Flutter: widgets/heroes.dart HeroFlightDirection.push
	HeroFlightDirectionPush
	// HeroFlightDirectionPop is the flight of a route that is popped.
	//
	// Flutter: widgets/heroes.dart HeroFlightDirection.pop
	HeroFlightDirectionPop
)

// String names the direction.
func (d HeroFlightDirection) String() string {
	switch d {
	case HeroFlightDirectionPush:
		return "HeroFlightDirection.push"
	case HeroFlightDirectionPop:
		return "HeroFlightDirection.pop"
	}
	return "HeroFlightDirection.unspecified"
}

// HeroOpts are the optional parameters of NewHero.
//
// Flutter: widgets/heroes.dart Hero.new
type HeroOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// CreateRectTween creates the tween of the flight of the hero; the default
	// is a RectTween.
	CreateRectTween CreateRectTween
	// FlightShuttleBuilder builds the widget that flies; the default shows the
	// child of the destination hero.
	FlightShuttleBuilder HeroFlightShuttleBuilder
	// PlaceholderBuilder builds the widget that stays in the place of the hero
	// during the flight; the default is an empty box of the size of the hero.
	PlaceholderBuilder HeroPlaceholderBuilder
	// TransitionOnUserGestures makes the hero fly during a user gesture that pops
	// the route, such as the iOS back swipe.
	TransitionOnUserGestures bool
	// Curve is the curve of the flight; the default is Curves.FastOutSlowIn.
	Curve animation.Curve
	// ReverseCurve is the curve of the flight on pop; the default is the flipped
	// Curve.
	ReverseCurve animation.Curve
}

// Hero marks its child as a candidate for hero animations: when a route is pushed
// or popped, the heroes with the same tag in the two routes fly from one to the
// other.
//
// Flutter: widgets/heroes.dart Hero
type Hero struct {
	StatefulWidget
	tag   any
	child Widget
	o     HeroOpts
}

// NewHero creates a hero with the tag; heroes with equal tags are paired. The tag
// must be comparable.
//
// Flutter: widgets/heroes.dart Hero.new
func NewHero(tag any, child Widget, opts ...HeroOpts) *Hero {
	o := firstOpts(opts)
	if o.Curve == nil {
		o.Curve = animation.Curves.FastOutSlowIn
	}
	return &Hero{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), tag, child, o}
}

// Tag identifies the hero.
//
// Flutter: widgets/heroes.dart Hero.tag
func (w *Hero) Tag() any { return w.tag }

// Child is the widget below.
//
// Flutter: widgets/heroes.dart Hero.child
func (w *Hero) Child() Widget { return w.child }

// CreateRectTween creates the tween of the flight, or nil.
//
// Flutter: widgets/heroes.dart Hero.createRectTween
func (w *Hero) CreateRectTween() CreateRectTween { return w.o.CreateRectTween }

// FlightShuttleBuilder builds the widget that flies, or nil.
//
// Flutter: widgets/heroes.dart Hero.flightShuttleBuilder
func (w *Hero) FlightShuttleBuilder() HeroFlightShuttleBuilder { return w.o.FlightShuttleBuilder }

// PlaceholderBuilder builds the placeholder of the hero, or nil.
//
// Flutter: widgets/heroes.dart Hero.placeholderBuilder
func (w *Hero) PlaceholderBuilder() HeroPlaceholderBuilder { return w.o.PlaceholderBuilder }

// TransitionOnUserGestures is whether the hero flies during user gestures.
//
// Flutter: widgets/heroes.dart Hero.transitionOnUserGestures
func (w *Hero) TransitionOnUserGestures() bool { return w.o.TransitionOnUserGestures }

// Curve is the curve of the flight.
//
// Flutter: widgets/heroes.dart Hero.curve
func (w *Hero) Curve() animation.Curve { return w.o.Curve }

// ReverseCurve is the curve of the flight on pop, or nil.
//
// Flutter: widgets/heroes.dart Hero.reverseCurve
func (w *Hero) ReverseCurve() animation.Curve { return w.o.ReverseCurve }

// CreateState creates the state.
//
// Flutter: widgets/heroes.dart Hero.createState
func (w *Hero) CreateState() State { return &heroState{} }

// heroMap is a map from tag to hero state that remembers the insertion order.
type heroMap struct {
	tags   []any
	states map[any]*heroState
}

func newHeroMap() *heroMap { return &heroMap{states: map[any]*heroState{}} }

func (m *heroMap) set(tag any, s *heroState) {
	if _, ok := m.states[tag]; !ok {
		m.tags = append(m.tags, tag)
	}
	m.states[tag] = s
}

func (m *heroMap) remove(tag any) {
	if _, ok := m.states[tag]; ok {
		delete(m.states, tag)
		m.tags = slices.DeleteFunc(m.tags, func(t any) bool { return t == tag })
	}
}

// allHeroesFor returns all of the heroes in the context, indexed by tag, that
// should be considered for animation when the navigator transitions from one
// PageRoute to another.
//
// Flutter: widgets/heroes.dart Hero._allHeroesFor
func allHeroesFor(bc BuildContext, isUserGestureTransition bool, navigator *NavigatorState) *heroMap {
	result := newHeroMap()
	inviteHero := func(hero *StatefulElement, tag any) {
		if foundation.DebugMode {
			if _, ok := result.states[tag]; ok {
				panic(errors.New("HeroTagDuplicated",
					"there are multiple heroes that share the same tag within a subtree; within each subtree for which heroes are to be animated (i.e. a PageRoute subtree), each Hero must have a unique non-null tag",
					"tag", tag, "hero", hero.String()))
			}
		}
		heroWidget := hero.widget.(*Hero)
		heroState := hero.state.(*heroState)
		if !isUserGestureTransition || heroWidget.o.TransitionOnUserGestures {
			result.set(tag, heroState)
		} else {
			// If the transition is not allowed, the hero must not be hidden. A hero
			// can be hidden previously due to a hero transition.
			heroState.endFlight(false)
		}
	}
	var visitor func(Element)
	visitor = func(element Element) {
		widget := element.AsElement().widget
		if hero, ok := widget.(*Hero); ok {
			heroElement := element.(*StatefulElement)
			if NavigatorOf(heroElement) == navigator {
				inviteHero(heroElement, hero.tag)
			} else {
				// The nearest navigator to the Hero is not the Navigator that is
				// currently transitioning from one route to another. This means the
				// Hero is inside a nested Navigator and should only be considered
				// for animation if it is part of the top-most route in that nested
				// Navigator and if that route is also a PageRoute.
				if heroRoute, ok := ModalRouteOf(heroElement).(PageRoute); ok && heroRoute.IsCurrent() {
					inviteHero(heroElement, hero.tag)
				}
			}
		} else if mode, ok := widget.(*HeroMode); ok && !mode.enabled {
			return
		}
		element.VisitChildren(visitor)
	}
	bc.VisitChildElements(visitor)
	return result
}

type heroState struct {
	StateOf[*Hero]
	key             *GlobalKey[State]
	placeholderSize foundation.Opt[ui.Size]
	// shouldIncludeChild is whether the placeholder widget should wrap the child
	// of the hero as its own child, when the placeholder size is set (the hero is
	// in its flight animation). See startFlight.
	shouldIncludeChild bool
}

func (s *heroState) InitState() {
	s.key = NewGlobalKey[State]()
	s.shouldIncludeChild = true
}

// startFlight is called when a new hero flight animation takes place: a
// placeholder widget is built to replace the original hero widget.
//
// When shouldIncludeChildInPlaceholder is true and the PlaceholderBuilder is nil,
// the placeholder includes the child of the original hero as a descendant, which
// allows the original element tree to be preserved. It is typically true for the
// from hero in a push transition, and false otherwise.
func (s *heroState) startFlight(shouldIncludeChildInPlaceholder bool) {
	s.shouldIncludeChild = shouldIncludeChildInPlaceholder
	if foundation.DebugMode && !s.Mounted() {
		panic(errors.New("HeroNotMounted", "a hero flight started for a hero that is not mounted"))
	}
	box, _ := s.Context().FindRenderObject().(rendering.Box)
	if foundation.DebugMode && (box == nil || !box.HasSize()) {
		panic(errors.New("HeroSizeMissing", "a hero flight started for a hero whose render object has no size"))
	}
	s.SetState(func() { s.placeholderSize = foundation.Some(box.Size()) })
}

// endFlight ends the flight. When keepPlaceholder is true, the placeholder
// continues to be shown after the flight ends. Otherwise the child of the hero
// becomes visible and its TickerMode is enabled again. It can be called when the
// hero is not in a flight.
func (s *heroState) endFlight(keepPlaceholder bool) {
	if keepPlaceholder || !s.placeholderSize.IsSome() {
		return
	}
	s.placeholderSize = foundation.None[ui.Size]()
	if s.Mounted() {
		// Tells the widget to rebuild if it is mounted; the placeholder size has
		// been updated already.
		s.SetState(func() {})
	}
}

func (s *heroState) Build(bc BuildContext) Widget {
	if foundation.DebugMode {
		if _, ok := FindAncestorWidgetOfExactType[*Hero](bc); ok {
			panic(errors.New("HeroDescendantOfHero", "a Hero widget cannot be the descendant of another Hero widget"))
		}
	}
	w := s.Widget()
	size, showPlaceholder := s.placeholderSize.Get()
	if showPlaceholder && w.o.PlaceholderBuilder != nil {
		return w.o.PlaceholderBuilder(bc, size, w.child)
	}
	if showPlaceholder && !s.shouldIncludeChild {
		return NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(size.Width), Height: foundation.Some(size.Height)})
	}
	box := SizedBoxOpts{}
	if showPlaceholder {
		box.Width, box.Height = foundation.Some(size.Width), foundation.Some(size.Height)
	}
	return NewSizedBox(
		NewOffstage(
			NewTickerMode(!showPlaceholder, NewKeyedSubtree(w.child, WidgetOpts{Key: s.key})),
			OffstageOpts{Offstage: foundation.Some(showPlaceholder)},
		),
		box,
	)
}

// heroFlightManifest is everything known about a hero flight that is to be
// started or diverted.
//
// Flutter: widgets/heroes.dart _HeroFlightManifest
type heroFlightManifest struct {
	flightType              HeroFlightDirection
	overlay                 *OverlayState
	navigatorSize           ui.Size
	fromRoute, toRoute      PageRoute
	fromHero, toHero        *heroState
	createRectTween         CreateRectTween
	shuttleBuilder          HeroFlightShuttleBuilder
	isUserGestureTransition bool
	isDiverted              bool

	curved           *animation.CurvedAnimation
	fromLocation     *ui.Rect
	toLocation       *ui.Rect
	valid, validDone bool
}

func (m *heroFlightManifest) tag() any { return m.fromHero.Widget().tag }

func (m *heroFlightManifest) animation() animation.Animation[float64] {
	if m.curved != nil {
		return m.curved
	}
	var parent animation.Animation[float64]
	var curve, reverseCurve animation.Curve
	switch m.flightType {
	case HeroFlightDirectionPush:
		parent = m.toRoute.Animation()
		curve = m.toHero.Widget().o.Curve
		reverseCurve = m.toHero.Widget().o.ReverseCurve
	default:
		parent = m.fromRoute.Animation()
		curve = m.fromHero.Widget().o.Curve
		reverseCurve = m.fromHero.Widget().o.ReverseCurve
	}
	if reverseCurve == nil {
		reverseCurve = curve.Flipped()
	}
	if m.isDiverted {
		reverseCurve = nil
	}
	if reverseCurve == nil {
		m.curved = animation.NewCurvedAnimation(parent, curve)
	} else {
		m.curved = animation.NewCurvedAnimation(parent, curve, reverseCurve)
	}
	return m.curved
}

func (m *heroFlightManifest) createHeroRectTween(begin, end foundation.Opt[ui.Rect]) *animation.Tween[foundation.Opt[ui.Rect]] {
	create := m.toHero.Widget().o.CreateRectTween
	if create == nil {
		create = m.createRectTween
	}
	if create != nil {
		if t := create(begin, end); t != nil {
			return t
		}
	}
	return animation.NewRectTween(begin, end)
}

// boundingBoxFor is the bounding box for the render object of the context, in
// the coordinate space of the render object of the ancestor context.
func heroBoundingBoxFor(bc, ancestorContext BuildContext) ui.Rect {
	if foundation.DebugMode && ancestorContext == nil {
		panic(errors.New("HeroSubtreeMissing", "a hero flight needs the subtree context of its route"))
	}
	box := bc.FindRenderObject().(rendering.Box)
	if foundation.DebugMode && (!box.HasSize() || !box.Size().IsFinite()) {
		panic(errors.New("HeroSizeMissing", "a hero flight started for a hero whose render object has no finite size"))
	}
	return box.GetTransformTo(ancestorContext.FindRenderObject()).TransformRect(ui.OffsetZero.And(box.Size()))
}

func (m *heroFlightManifest) fromHeroLocation() ui.Rect {
	if m.fromLocation == nil {
		r := heroBoundingBoxFor(m.fromHero.Context(), m.fromRoute.SubtreeContext())
		m.fromLocation = &r
	}
	return *m.fromLocation
}

func (m *heroFlightManifest) toHeroLocation() ui.Rect {
	if m.toLocation == nil {
		r := heroBoundingBoxFor(m.toHero.Context(), m.toRoute.SubtreeContext())
		m.toLocation = &r
	}
	return *m.toLocation
}

func (m *heroFlightManifest) isValid() bool {
	if !m.validDone {
		m.valid = m.toHeroLocation().IsFinite() && (m.isDiverted || m.fromHeroLocation().IsFinite())
		m.validDone = true
	}
	return m.valid
}

func (m *heroFlightManifest) String() string {
	invalid := ""
	if !m.isValid() {
		invalid = ", INVALID"
	}
	return fmt.Sprintf("_HeroFlightManifest(%v tag: %v from route: %v to route: %v with hero: %v to %v)%s",
		m.flightType, m.tag(), describeSettings(m.fromRoute.Settings()), describeSettings(m.toRoute.Settings()), m.fromHero, m.toHero, invalid)
}

func (m *heroFlightManifest) dispose() {
	if m.curved != nil {
		m.curved.Dispose()
	}
}

// heroFlight builds the in-flight hero widget.
//
// Flutter: widgets/heroes.dart _HeroFlight
type heroFlight struct {
	onFlightEnded func(*heroFlight)
	heroRectTween *animation.Tween[foundation.Opt[ui.Rect]]
	shuttle       Widget
	heroOpacity   animation.Animation[float64]
	proxy         *animation.ProxyAnimation
	manifest      *heroFlightManifest
	overlayEntry  *OverlayEntry
	aborted       bool

	statusHandle                    animation.StatusListenerHandle
	tickHandle                      foundation.ListenerHandle
	scheduledPerformAnimationUpdate bool
}

var heroReverseTween = animation.NewTween(1.0, 0.0)

func newHeroFlight(onFlightEnded func(*heroFlight)) *heroFlight {
	f := &heroFlight{onFlightEnded: onFlightEnded, heroOpacity: animation.AlwaysComplete}
	f.proxy = animation.NewProxyAnimation(nil)
	f.statusHandle = f.proxy.AddStatusListener(f.handleAnimationUpdate)
	return f
}

func (f *heroFlight) setManifest(m *heroFlightManifest) {
	if f.manifest != nil {
		f.manifest.dispose()
	}
	f.manifest = m
}

// buildOverlay is the OverlayEntry builder of the overlay of the hero.
func (f *heroFlight) buildOverlay(bc BuildContext) Widget {
	if f.shuttle == nil {
		f.shuttle = f.manifest.shuttleBuilder(bc, f.manifest.animation(), f.manifest.flightType,
			f.manifest.fromHero.Context(), f.manifest.toHero.Context())
	}
	return NewAnimatedBuilder(f.proxy, func(bc BuildContext, child Widget) Widget {
		rect, _ := f.heroRectTween.Evaluate(f.proxy).Get()
		offsets := rendering.NewRelativeRectFromSize(rect, f.manifest.navigatorSize)
		return NewPositioned(
			NewIgnorePointer(NewFadeTransition(f.heroOpacity, child)),
			PositionedOpts{
				Top: foundation.Some(offsets.Top), Right: foundation.Some(offsets.Right),
				Bottom: foundation.Some(offsets.Bottom), Left: foundation.Some(offsets.Left),
			},
		)
	}, f.shuttle)
}

func (f *heroFlight) performAnimationUpdate(status animation.AnimationStatus) {
	if !status.IsAnimating() {
		f.proxy.SetParent(nil)
		f.overlayEntry.Remove()
		f.overlayEntry.Dispose()
		f.overlayEntry = nil
		// The hero underneath the current page stays hidden. If the animation
		// completed, the to hero is on top and the from hero stays hidden. If it
		// is dismissed, the animation was triggered but canceled before it
		// finished, so the to hero stays hidden instead.
		f.manifest.fromHero.endFlight(status.IsCompleted())
		f.manifest.toHero.endFlight(status.IsDismissed())
		f.onFlightEnded(f)
		f.proxy.RemoveListener(f.tickHandle)
	}
}

func (f *heroFlight) handleAnimationUpdate(status animation.AnimationStatus) {
	// The animation does not finish until the user lifts the finger, so the
	// status update is suppressed while the gesture is in progress, and delayed
	// until the finger is lifted.
	navigator := f.manifest.fromRoute.Navigator()
	if navigator == nil || !navigator.UserGestureInProgress() {
		f.performAnimationUpdate(status)
		return
	}
	if f.scheduledPerformAnimationUpdate {
		return
	}
	f.scheduledPerformAnimationUpdate = true
	notifier := navigator.UserGestureInProgressNotifier()
	var handle foundation.ListenerHandle
	handle = notifier.AddListener(func() {
		f.scheduledPerformAnimationUpdate = false
		notifier.RemoveListener(handle)
		f.performAnimationUpdate(f.proxy.Status())
	})
}

func (f *heroFlight) dispose() {
	if f.overlayEntry != nil {
		f.overlayEntry.Remove()
		f.overlayEntry.Dispose()
		f.overlayEntry = nil
		f.proxy.SetParent(nil)
		f.proxy.RemoveListener(f.tickHandle)
		f.proxy.RemoveStatusListener(f.statusHandle)
	}
	if f.manifest != nil {
		f.manifest.dispose()
	}
}

func (f *heroFlight) onTick() {
	var toHeroBox rendering.Box
	if !f.aborted && f.manifest.toHero.Mounted() {
		toHeroBox, _ = f.manifest.toHero.Context().FindRenderObject().(rendering.Box)
	}
	// Tries to find the new origin of the to hero, if the flight isn't aborted.
	toHeroOrigin := foundation.None[ui.Offset]()
	if toHeroBox != nil && toHeroBox.Attached() && toHeroBox.HasSize() {
		var ancestor rendering.RenderObject
		if ctx := f.manifest.toRoute.SubtreeContext(); ctx != nil {
			if box, ok := ctx.FindRenderObject().(rendering.Box); ok {
				ancestor = box
			}
		}
		toHeroOrigin = foundation.Some(toHeroBox.LocalToGlobal(ui.OffsetZero, ancestor))
	}
	if origin, ok := toHeroOrigin.Get(); ok && origin.IsFinite() {
		// If the new origin of the to hero is available and also paintable, the
		// hero rect tween is updated with it.
		end, _ := f.heroRectTween.End.Get()
		if origin != end.TopLeft() {
			heroRectEnd := origin.And(end.Size())
			f.heroRectTween = f.manifest.createHeroRectTween(f.heroRectTween.Begin, foundation.Some(heroRectEnd))
		}
	} else if f.heroOpacity.Status().IsCompleted() {
		// The to hero no longer exists, or it is no longer the destination of the
		// flight. The hero continues to fly while fading out.
		f.heroOpacity = animation.Drive[float64](f.proxy,
			heroReverseTween.Chain(animation.NewCurveTween(animation.NewInterval(f.proxy.Value(), 1.0))))
	}
	// Updates aborted for the next animation tick.
	origin, ok := toHeroOrigin.Get()
	f.aborted = !ok || !origin.IsFinite()
}

// start is the simple case: a push or a pop animation starts.
func (f *heroFlight) start(initial *heroFlightManifest) {
	if foundation.DebugMode {
		if f.aborted {
			panic(errors.New("HeroFlightAborted", "a hero flight can not start after it was aborted"))
		}
		initialAnimation := initial.animation()
		switch initial.flightType {
		case HeroFlightDirectionPop:
			// During user gesture transitions the animation controller isn't
			// driving the reverse transition, so the status is not important.
			if !initial.isUserGestureTransition && initialAnimation.Status() != animation.AnimationStatusReverse {
				panic(errors.New("HeroFlightStatusInvalid", "a pop flight needs an animation that is reversing"))
			}
		case HeroFlightDirectionPush:
			if initialAnimation.Value() != 0.0 || initialAnimation.Status() != animation.AnimationStatusForward {
				panic(errors.New("HeroFlightStatusInvalid", "a push flight needs an animation that starts forward from 0"))
			}
		}
	}
	f.setManifest(initial)
	var shouldIncludeChildInPlaceholder bool
	switch f.manifest.flightType {
	case HeroFlightDirectionPop:
		f.proxy.SetParent(animation.NewReverseAnimation(f.manifest.animation()))
		shouldIncludeChildInPlaceholder = false
	default:
		f.proxy.SetParent(f.manifest.animation())
		shouldIncludeChildInPlaceholder = true
	}
	f.heroRectTween = f.manifest.createHeroRectTween(foundation.Some(f.manifest.fromHeroLocation()), foundation.Some(f.manifest.toHeroLocation()))
	f.manifest.fromHero.startFlight(shouldIncludeChildInPlaceholder)
	f.manifest.toHero.startFlight(false)
	f.overlayEntry = NewOverlayEntry(f.buildOverlay)
	f.manifest.overlay.Insert(f.overlayEntry)
	f.tickHandle = f.proxy.AddListener(f.onTick)
}

// divert is called when a push or a pop occurred for routes with the same hero
// while this flight's hero was in transition: the hero in flight is redirected to
// the new to route.
func (f *heroFlight) divert(newManifest *heroFlightManifest) {
	m := f.manifest
	if foundation.DebugMode && m.tag() != newManifest.tag() {
		panic(errors.New("HeroFlightTagMismatch", "a hero flight can only be diverted to a flight with the same tag"))
	}
	switch {
	case m.flightType == HeroFlightDirectionPush && newManifest.flightType == HeroFlightDirectionPop:
		// A push flight was interrupted by a pop. The same hero rect tween is used
		// in reverse, rather than creating a new one with the swapped begin and
		// end: tweens like MaterialRectArcTween may create a different path for
		// swapped parameters, and the pop flight path must be the same, reversed,
		// as the push flight path.
		f.proxy.SetParent(animation.NewReverseAnimation(newManifest.animation()))
		f.heroRectTween = animation.NewReverseTween(f.heroRectTween)
	case m.flightType == HeroFlightDirectionPop && newManifest.flightType == HeroFlightDirectionPush:
		// A pop flight was interrupted by a push.
		f.proxy.SetParent(animation.Drive[float64](newManifest.animation(), animation.NewTween(m.animation().Value(), 1.0)))
		if m.fromHero != newManifest.toHero {
			m.fromHero.endFlight(true)
			newManifest.toHero.startFlight(false)
			f.heroRectTween = m.createHeroRectTween(f.heroRectTween.End, foundation.Some(newManifest.toHeroLocation()))
		} else {
			f.heroRectTween = m.createHeroRectTween(f.heroRectTween.End, f.heroRectTween.Begin)
		}
	default:
		// A push or a pop flight is heading to a new route: both are pushes or
		// both are pops.
		f.heroRectTween = m.createHeroRectTween(f.heroRectTween.Evaluate(f.proxy), foundation.Some(newManifest.toHeroLocation()))
		f.shuttle = nil
		if newManifest.flightType == HeroFlightDirectionPop {
			f.proxy.SetParent(animation.NewReverseAnimation(newManifest.animation()))
		} else {
			f.proxy.SetParent(newManifest.animation())
		}
		m.fromHero.endFlight(true)
		m.toHero.endFlight(true)
		// Lets the heroes in each of the routes rebuild with their placeholders.
		newManifest.fromHero.startFlight(newManifest.flightType == HeroFlightDirectionPush)
		newManifest.toHero.startFlight(false)
		// Lets the transition overlay on top of the routes rebuild too, since the
		// old shuttle was cleared.
		f.overlayEntry.MarkNeedsBuild()
	}
	f.setManifest(newManifest)
}

func (f *heroFlight) abort() { f.aborted = true }

func (f *heroFlight) String() string {
	return fmt.Sprintf("HeroFlight(for: %v, from: %v, to: %v %v)", f.manifest.tag(),
		describeSettings(f.manifest.fromRoute.Settings()), describeSettings(f.manifest.toRoute.Settings()), f.proxy.Parent())
}

// HeroController manages the hero transitions of a navigator: it is a navigator
// observer that starts the flights when the top route changes.
//
// Flutter: widgets/heroes.dart HeroController
type HeroController struct {
	NavigatorObserverBase
	// self is the outer observer when the controller is embedded in another
	// type that overrides the observer methods; nil for a plain controller.
	self            NavigatorObserver
	createRectTween CreateRectTween
	// flights are all of the heroes that are currently in the overlay and in
	// motion, indexed by the hero tag.
	flights     map[any]*heroFlight
	flightOrder []any
}

// NewHeroController creates a controller; createRectTween creates the tween of
// the flights unless the hero says otherwise (the default is a RectTween).
//
// Flutter: widgets/heroes.dart HeroController.new
func NewHeroController(createRectTween ...CreateRectTween) *HeroController {
	c := &HeroController{flights: map[any]*heroFlight{}}
	if len(createRectTween) > 0 {
		c.createRectTween = createRectTween[0]
	}
	return c
}

// InitHeroController sets the outer observer of a controller that is embedded in
// another type, so that the navigators call the methods that the outer type
// overrides (Flutter subclasses HeroController).
//
// Flutter: no counterpart (the self pattern, SPEC §12.5)
func InitHeroController(c *HeroController, self NavigatorObserver) { c.self = self }

// AsObserver is the observer that the navigator notifies: the outer value if the
// controller is embedded in another type, and the controller otherwise.
func (c *HeroController) AsObserver() NavigatorObserver {
	if c.self != nil {
		return c.self
	}
	return c
}

func (c *HeroController) setFlight(tag any, f *heroFlight) {
	if _, ok := c.flights[tag]; !ok {
		c.flightOrder = append(c.flightOrder, tag)
	}
	c.flights[tag] = f
}

func (c *HeroController) removeFlight(tag any) *heroFlight {
	f, ok := c.flights[tag]
	if !ok {
		return nil
	}
	delete(c.flights, tag)
	c.flightOrder = slices.DeleteFunc(c.flightOrder, func(t any) bool { return t == tag })
	return f
}

// DidChangeTop starts a hero transition from the previous top route to the new
// one.
//
// Flutter: widgets/heroes.dart HeroController.didChangeTop
func (c *HeroController) DidChangeTop(topRoute, previousTopRoute Route) {
	if foundation.DebugMode {
		if !topRoute.IsCurrent() {
			panic(errors.New("HeroControllerRouteNotCurrent", "the top route reported to the hero controller is not the current route"))
		}
		if c.Navigator() == nil {
			panic(errors.New("HeroControllerNavigatorMissing", "the hero controller is not attached to a navigator"))
		}
	}
	if previousTopRoute == nil {
		return
	}
	// Doesn't trigger another flight when a pop is committed as a user gesture
	// back swipe is snapped.
	if !c.Navigator().UserGestureInProgress() {
		c.maybeStartHeroTransition(previousTopRoute, topRoute, false)
	}
}

// DidStartUserGesture starts a hero transition for the user gesture.
//
// Flutter: widgets/heroes.dart HeroController.didStartUserGesture
func (c *HeroController) DidStartUserGesture(route, previousRoute Route) {
	c.maybeStartHeroTransition(route, previousRoute, true)
}

// DidStopUserGesture invalidates the flights of a gesture that did not move
// towards the pop direction at all.
//
// Flutter: widgets/heroes.dart HeroController.didStopUserGesture
func (c *HeroController) DidStopUserGesture() {
	if c.Navigator().UserGestureInProgress() {
		return
	}
	// When the user gesture ends, if the horizontal drag that initiated the flight
	// (the back swipe) did not move towards the pop direction at all, the
	// animation does not play and the status update callback is never called when
	// the gesture finishes. In this case the flight that was initiated needs to be
	// invalidated manually.
	var invalidFlights []*heroFlight
	for _, tag := range c.flightOrder {
		flight := c.flights[tag]
		if flight.manifest.isUserGestureTransition && flight.manifest.flightType == HeroFlightDirectionPop &&
			flight.proxy.Status().IsDismissed() {
			invalidFlights = append(invalidFlights, flight)
		}
	}
	// Treats these invalidated flights as dismissed. Calling handleAnimationUpdate
	// removes the flight from the flights as well.
	for _, flight := range invalidFlights {
		flight.handleAnimationUpdate(animation.AnimationStatusDismissed)
	}
}

// maybeStartHeroTransition starts a hero transition after the to route has been
// laid out with its animation's value at 1.0, if the transition is between
// different page routes.
func (c *HeroController) maybeStartHeroTransition(fromRoute, toRoute Route, isUserGestureTransition bool) {
	to, toIsPage := toRoute.(PageRoute)
	from, fromIsPage := fromRoute.(PageRoute)
	if toRoute == fromRoute || !toIsPage || !fromIsPage {
		return
	}
	newRouteAnimation := to.Animation()
	oldRouteAnimation := from.Animation()
	flightType := HeroFlightDirectionUnspecified
	switch {
	case isUserGestureTransition, oldRouteAnimation.Status() == animation.AnimationStatusReverse:
		flightType = HeroFlightDirectionPop
	case newRouteAnimation.Status() == animation.AnimationStatusForward:
		flightType = HeroFlightDirectionPush
	}
	// A user gesture may have completed the pop already, or this may be the
	// initial route.
	switch flightType {
	case HeroFlightDirectionPop:
		if from.Animation().Value() == 0.0 {
			return
		}
	case HeroFlightDirectionPush:
		if to.Animation().Value() == 1.0 {
			return
		}
	}
	// For pop transitions driven by a user gesture: if the to page has
	// maintainState = true, the final dimensions of the hero can be measured
	// immediately because the layout of its page is still valid. Unless routes
	// were added directly to the pages stack, which makes the route never get laid
	// out.
	hasValidSize := false
	if ctx := to.SubtreeContext(); ctx != nil {
		if box, ok := ctx.FindRenderObject().(rendering.Box); ok {
			hasValidSize = box.HasSize() && box.Size().IsFinite()
		}
	}
	if isUserGestureTransition && flightType == HeroFlightDirectionPop && to.MaintainState() && hasValidSize {
		c.startHeroTransition(from, to, flightType, isUserGestureTransition)
	} else {
		// Otherwise, measuring is delayed until the end of the next frame, to allow
		// the to route to build and lay out. Putting a route offstage changes its
		// animation value to 1.0. Once this frame completes, the location of the
		// heroes in the to route is known, and the to route goes back onstage.
		to.SetOffstage(to.Animation().Value() == 0.0)
		WidgetsBindingOf(c.Navigator().Context()).Scheduler().AddPostFrameCallback(func(time.Duration) {
			if from.Navigator() == nil || to.Navigator() == nil {
				return
			}
			c.startHeroTransition(from, to, flightType, isUserGestureTransition)
		})
	}
}

// startHeroTransition finds the matching pairs of heroes in from and to, and
// either starts a new hero flight or diverts an existing one.
func (c *HeroController) startHeroTransition(from, to PageRoute, flightType HeroFlightDirection, isUserGestureTransition bool) {
	// If the to route was offstage, its animation value is restored implicitly to
	// what it was before it was moved offstage.
	to.SetOffstage(false)

	navigator := c.Navigator()
	var overlay *OverlayState
	if navigator != nil {
		overlay = navigator.Overlay()
	}
	// If the navigator or the overlay was removed before this end-of-frame
	// callback was called, no transition starts, and there is no need to worry
	// about any Hero widget that was hidden in a previous flight, or about
	// ongoing flights.
	if navigator == nil || overlay == nil {
		return
	}
	navigatorRenderObject, ok := navigator.Context().FindRenderObject().(rendering.Box)
	if !ok {
		if foundation.DebugMode {
			panic(errors.New("HeroNavigatorRenderObjectInvalid", "the navigator has an invalid render object type",
				"navigator", navigator.String()))
		}
		return
	}
	if foundation.DebugMode && !navigatorRenderObject.HasSize() {
		panic(errors.New("HeroNavigatorSizeMissing", "the render object of the navigator has no size", "navigator", navigator.String()))
	}

	// At this point the to heroes may have been built and laid out for the first
	// time. If the from subtree context is nil, endFlight is called on all the to
	// heroes, for good measure. If the to subtree context is nil, the existing
	// flights are aborted.
	fromHeroes := newHeroMap()
	if ctx := from.SubtreeContext(); ctx != nil {
		fromHeroes = allHeroesFor(ctx, isUserGestureTransition, navigator)
	}
	toHeroes := newHeroMap()
	if ctx := to.SubtreeContext(); ctx != nil {
		toHeroes = allHeroesFor(ctx, isUserGestureTransition, navigator)
	}

	for _, tag := range slices.Clone(fromHeroes.tags) {
		fromHero := fromHeroes.states[tag]
		toHero := toHeroes.states[tag]
		existingFlight := c.flights[tag]
		var manifest *heroFlightManifest
		if toHero != nil && flightType != HeroFlightDirectionUnspecified {
			shuttleBuilder := toHero.Widget().o.FlightShuttleBuilder
			if shuttleBuilder == nil {
				shuttleBuilder = fromHero.Widget().o.FlightShuttleBuilder
			}
			if shuttleBuilder == nil {
				shuttleBuilder = defaultHeroFlightShuttleBuilder
			}
			manifest = &heroFlightManifest{
				flightType:              flightType,
				overlay:                 overlay,
				navigatorSize:           navigatorRenderObject.Size(),
				fromRoute:               from,
				toRoute:                 to,
				fromHero:                fromHero,
				toHero:                  toHero,
				createRectTween:         c.createRectTween,
				shuttleBuilder:          shuttleBuilder,
				isUserGestureTransition: isUserGestureTransition,
				isDiverted:              existingFlight != nil,
			}
		}
		// Only proceeds with a valid manifest. Otherwise aborts the existing
		// flight, and calls endFlight when this loop finishes.
		if manifest != nil && manifest.isValid() {
			toHeroes.remove(tag)
			if existingFlight != nil {
				existingFlight.divert(manifest)
			} else {
				flight := newHeroFlight(c.handleFlightEnded)
				c.setFlight(tag, flight)
				flight.start(manifest)
			}
		} else if existingFlight != nil {
			existingFlight.abort()
		}
	}

	// The remaining entries in toHeroes are those that failed to participate in a
	// new flight (for not having a valid manifest). This can happen in a route
	// pop transition when a from hero is no longer mounted, or kept alive by the
	// KeepAlive mechanism but no longer visible.
	for _, tag := range toHeroes.tags {
		toHeroes.states[tag].endFlight(false)
	}
}

func (c *HeroController) handleFlightEnded(flight *heroFlight) {
	if f := c.removeFlight(flight.manifest.tag()); f != nil {
		f.dispose()
	}
}

// defaultHeroFlightShuttleBuilder shows the child of the to hero, with the
// padding of the media query animated from the one of the from hero to the one of
// the to hero.
func defaultHeroFlightShuttleBuilder(flightContext BuildContext, a animation.Animation[float64], flightDirection HeroFlightDirection, fromHeroContext, toHeroContext BuildContext) Widget {
	toHero := toHeroContext.Widget().(*Hero)
	toMediaQueryData, toOK := MediaQueryMaybeOf(toHeroContext)
	fromMediaQueryData, fromOK := MediaQueryMaybeOf(fromHeroContext)
	if !toOK || !fromOK {
		return toHero.child
	}
	fromHeroPadding := fromMediaQueryData.Padding
	toHeroPadding := toMediaQueryData.Padding
	return NewAnimatedBuilder(a, func(bc BuildContext, _ Widget) Widget {
		var padding painting.EdgeInsets
		if flightDirection == HeroFlightDirectionPush {
			padding = NewEdgeInsetsTween(fromHeroPadding, toHeroPadding).Evaluate(a)
		} else {
			padding = NewEdgeInsetsTween(toHeroPadding, fromHeroPadding).Evaluate(a)
		}
		return NewMediaQuery(toMediaQueryData.CopyWith(MediaQueryDataCopyOpts{Padding: foundation.Some(padding)}), toHero.child)
	}, nil)
}

// Dispose disposes the flights in progress.
//
// Flutter: widgets/heroes.dart HeroController.dispose
func (c *HeroController) Dispose() {
	for _, tag := range c.flightOrder {
		c.flights[tag].dispose()
	}
}

// HeroMode controls whether Hero widgets in its subtree take part in hero
// transitions.
//
// Flutter: widgets/heroes.dart HeroMode
type HeroMode struct {
	StatelessWidget
	child   Widget
	enabled bool
}

// HeroModeOpts are the optional parameters of NewHeroMode.
//
// Flutter: widgets/heroes.dart HeroMode.new
type HeroModeOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Enabled is whether the heroes below take part in transitions; the default
	// is true.
	Enabled foundation.Opt[bool]
}

// NewHeroMode creates the widget.
//
// Flutter: widgets/heroes.dart HeroMode.new
func NewHeroMode(child Widget, opts ...HeroModeOpts) *HeroMode {
	o := firstOpts(opts)
	return &HeroMode{NewStatelessWidgetBase(WidgetOpts{Key: o.Key}), child, o.Enabled.OrElse(true)}
}

// Child is the widget below.
//
// Flutter: widgets/heroes.dart HeroMode.child
func (w *HeroMode) Child() Widget { return w.child }

// Enabled is whether the heroes below take part in transitions.
//
// Flutter: widgets/heroes.dart HeroMode.enabled
func (w *HeroMode) Enabled() bool { return w.enabled }

// Build returns the child.
//
// Flutter: widgets/heroes.dart HeroMode.build
func (w *HeroMode) Build(bc BuildContext) Widget { return w.child }
