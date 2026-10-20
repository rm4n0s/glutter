package widgets

import (
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
)

// AnimatedSwitcherTransitionBuilder builds the transition of a child of an
// AnimatedSwitcher from the child and the animation of its switch.
//
// Flutter: widgets/animated_switcher.dart AnimatedSwitcherTransitionBuilder
type AnimatedSwitcherTransitionBuilder func(child Widget, animation animation.Animation[float64]) Widget

// AnimatedSwitcherLayoutBuilder lays out the current child and the children that
// are on their way out (the previous children), in the order they should be
// painted. The current child is nil when there is none.
//
// Flutter: widgets/animated_switcher.dart AnimatedSwitcherLayoutBuilder
type AnimatedSwitcherLayoutBuilder func(currentChild Widget, previousChildren []Widget) Widget

// AnimatedSwitcherDefaultTransitionBuilder is the transition builder of an
// AnimatedSwitcher that has none: a FadeTransition keyed with the key of the
// child.
//
// Flutter: widgets/animated_switcher.dart AnimatedSwitcher.defaultTransitionBuilder
func AnimatedSwitcherDefaultTransitionBuilder(child Widget, animation animation.Animation[float64]) Widget {
	var key foundation.Key
	if k := child.Key(); k != nil {
		key = foundation.NewValueKey[foundation.Key](k)
	} else {
		key = foundation.NewValueKey[foundation.Key](nil)
	}
	return NewFadeTransition(animation, child, FadeTransitionOpts{Key: key})
}

// AnimatedSwitcherDefaultLayoutBuilder is the layout builder of an
// AnimatedSwitcher that has none: a Stack with the previous children below the
// current one, centered.
//
// Flutter: widgets/animated_switcher.dart AnimatedSwitcher.defaultLayoutBuilder
func AnimatedSwitcherDefaultLayoutBuilder(currentChild Widget, previousChildren []Widget) Widget {
	children := append([]Widget(nil), previousChildren...)
	if currentChild != nil {
		children = append(children, currentChild)
	}
	return NewStack(children, StackOpts{Alignment: painting.AlignmentCenter})
}

// AnimatedSwitcherOpts are the optional parameters of NewAnimatedSwitcher.
//
// Flutter: widgets/animated_switcher.dart AnimatedSwitcher.new
type AnimatedSwitcherOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// ReverseDuration is the duration of the transition of a child that is
	// replaced; it defaults to the duration.
	ReverseDuration foundation.Opt[time.Duration]
	// SwitchInCurve is the curve of the transition of the new child; the default
	// is Curves.Linear.
	SwitchInCurve animation.Curve
	// SwitchOutCurve is the curve of the transition of the old child; the
	// default is Curves.Linear.
	SwitchOutCurve animation.Curve
	// TransitionBuilder builds the transition of each child; the default is
	// AnimatedSwitcherDefaultTransitionBuilder.
	TransitionBuilder AnimatedSwitcherTransitionBuilder
	// LayoutBuilder lays out the children; the default is
	// AnimatedSwitcherDefaultLayoutBuilder.
	LayoutBuilder AnimatedSwitcherLayoutBuilder
}

// AnimatedSwitcher is a widget that by default does a cross-fade between a new
// widget and the widget previously set on the AnimatedSwitcher as a child. If
// the child changes type or key, the old one fades out while the new one fades
// in; a child of the same type and key only updates.
//
// Flutter: widgets/animated_switcher.dart AnimatedSwitcher
type AnimatedSwitcher struct {
	StatefulWidget
	child             Widget
	duration          time.Duration
	reverseDuration   foundation.Opt[time.Duration]
	switchInCurve     animation.Curve
	switchOutCurve    animation.Curve
	transitionBuilder AnimatedSwitcherTransitionBuilder
	layoutBuilder     AnimatedSwitcherLayoutBuilder
}

// NewAnimatedSwitcher creates the widget; the child may be nil.
//
// Flutter: widgets/animated_switcher.dart AnimatedSwitcher.new
func NewAnimatedSwitcher(duration time.Duration, child Widget, opts ...AnimatedSwitcherOpts) *AnimatedSwitcher {
	o := firstOpts(opts)
	if o.SwitchInCurve == nil {
		o.SwitchInCurve = animation.Curves.Linear
	}
	if o.SwitchOutCurve == nil {
		o.SwitchOutCurve = animation.Curves.Linear
	}
	return &AnimatedSwitcher{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), child, duration, o.ReverseDuration,
		o.SwitchInCurve, o.SwitchOutCurve, o.TransitionBuilder, o.LayoutBuilder}
}

// Child is the current child, or nil.
//
// Flutter: widgets/animated_switcher.dart AnimatedSwitcher.child
func (w *AnimatedSwitcher) Child() Widget { return w.child }

// Duration is the duration of the transition from the old child to the new one.
//
// Flutter: widgets/animated_switcher.dart AnimatedSwitcher.duration
func (w *AnimatedSwitcher) Duration() time.Duration { return w.duration }

// ReverseDuration is the duration of the transition when a child is replaced.
//
// Flutter: widgets/animated_switcher.dart AnimatedSwitcher.reverseDuration
func (w *AnimatedSwitcher) ReverseDuration() foundation.Opt[time.Duration] { return w.reverseDuration }

// SwitchInCurve is the curve of the transition of the new child.
//
// Flutter: widgets/animated_switcher.dart AnimatedSwitcher.switchInCurve
func (w *AnimatedSwitcher) SwitchInCurve() animation.Curve { return w.switchInCurve }

// SwitchOutCurve is the curve of the transition of the old child.
//
// Flutter: widgets/animated_switcher.dart AnimatedSwitcher.switchOutCurve
func (w *AnimatedSwitcher) SwitchOutCurve() animation.Curve { return w.switchOutCurve }

// CreateState creates the state.
//
// Flutter: widgets/animated_switcher.dart AnimatedSwitcher.createState
func (w *AnimatedSwitcher) CreateState() State { return &animatedSwitcherState{} }

func (w *AnimatedSwitcher) transition() AnimatedSwitcherTransitionBuilder {
	if w.transitionBuilder == nil {
		return AnimatedSwitcherDefaultTransitionBuilder
	}
	return w.transitionBuilder
}

func (w *AnimatedSwitcher) layout() AnimatedSwitcherLayoutBuilder {
	if w.layoutBuilder == nil {
		return AnimatedSwitcherDefaultLayoutBuilder
	}
	return w.layoutBuilder
}

// switcherEntry is the internal representation of a child that, now or in the
// past, was set on the AnimatedSwitcher and is transitioning.
type switcherEntry struct {
	controller *animation.AnimationController
	animation  *animation.CurvedAnimation
	// transition is the currently built transition of the child.
	transition Widget
	// widgetChild is the child at the time the entry was created or updated.
	widgetChild Widget
}

type animatedSwitcherState struct {
	StateOf[*AnimatedSwitcher]
	TickerProvider
	currentEntry    *switcherEntry
	outgoingEntries []*switcherEntry
	// outgoingWidgets caches the transitions of the outgoing entries; nil when
	// it needs to be rebuilt.
	outgoingWidgets []Widget
	outgoingValid   bool
	childNumber     int
}

func (s *animatedSwitcherState) InitState() {
	s.addEntryForNewChild(false)
}

func (s *animatedSwitcherState) DidUpdateWidget(old *AnimatedSwitcher) {
	// Deviation: Dart compares the transition builders with !=; Go functions
	// can't be compared, so the transitions are updated unless both widgets use
	// the default builder (SPEC §12.10).
	if s.Widget().transitionBuilder != nil || old.transitionBuilder != nil {
		for _, e := range s.outgoingEntries {
			s.updateTransitionForEntry(e)
		}
		if s.currentEntry != nil {
			s.updateTransitionForEntry(s.currentEntry)
		}
		s.markChildWidgetCacheAsDirty()
	}

	hasNewChild := s.Widget().child != nil
	hasOldChild := s.currentEntry != nil
	if hasNewChild != hasOldChild || hasNewChild && !CanUpdate(s.currentEntry.widgetChild, s.Widget().child) {
		// The child has changed: fade the current entry out and add a new one.
		s.childNumber++
		s.addEntryForNewChild(true)
	} else if s.currentEntry != nil {
		// The child has been updated. Update the widget and the transition of
		// the current entry even though no new animation starts, and keep the
		// key of the previous transition so that the transition is updated
		// instead of replaced.
		s.currentEntry.widgetChild = s.Widget().child
		s.updateTransitionForEntry(s.currentEntry)
		s.markChildWidgetCacheAsDirty()
	}
}

func (s *animatedSwitcherState) addEntryForNewChild(animate bool) {
	if s.currentEntry != nil {
		s.outgoingEntries = append(s.outgoingEntries, s.currentEntry)
		s.currentEntry.controller.Reverse()
		s.markChildWidgetCacheAsDirty()
		s.currentEntry = nil
	}
	w := s.Widget()
	if w.child == nil {
		return
	}
	controller := animation.NewAnimationController(s, animation.AnimationControllerOpts{
		Duration:        foundation.Some(w.duration),
		ReverseDuration: w.reverseDuration,
	})
	curved := animation.NewCurvedAnimation(controller, w.switchInCurve, w.switchOutCurve)
	s.currentEntry = s.newEntry(w.child, w.transition(), controller, curved)
	if animate {
		controller.Forward()
	} else {
		controller.SetValue(1)
	}
}

func (s *animatedSwitcherState) newEntry(child Widget, builder AnimatedSwitcherTransitionBuilder,
	controller *animation.AnimationController, curved *animation.CurvedAnimation) *switcherEntry {
	entry := &switcherEntry{
		widgetChild: child,
		transition:  NewKeyedSubtreeWrap(s.childNumber, builder(child, curved)),
		animation:   curved,
		controller:  controller,
	}
	curved.AddStatusListener(func(status animation.AnimationStatus) {
		if status.IsDismissed() {
			s.SetState(func() {
				s.outgoingEntries = removeSwitcherEntry(s.outgoingEntries, entry)
				s.markChildWidgetCacheAsDirty()
			})
			controller.Dispose()
			curved.Dispose()
		}
	})
	return entry
}

func removeSwitcherEntry(entries []*switcherEntry, entry *switcherEntry) []*switcherEntry {
	for i, e := range entries {
		if e == entry {
			return append(entries[:i:i], entries[i+1:]...)
		}
	}
	return entries
}

func (s *animatedSwitcherState) markChildWidgetCacheAsDirty() { s.outgoingValid = false }

func (s *animatedSwitcherState) updateTransitionForEntry(entry *switcherEntry) {
	entry.transition = NewKeyedSubtree(s.Widget().transition()(entry.widgetChild, entry.animation), WidgetOpts{Key: entry.transition.Key()})
}

func (s *animatedSwitcherState) rebuildOutgoingWidgetsIfNeeded() {
	if s.outgoingValid {
		return
	}
	s.outgoingWidgets = make([]Widget, len(s.outgoingEntries))
	for i, e := range s.outgoingEntries {
		s.outgoingWidgets[i] = e.transition
	}
	s.outgoingValid = true
}

func (s *animatedSwitcherState) Dispose() {
	if s.currentEntry != nil {
		s.currentEntry.controller.Dispose()
		s.currentEntry.animation.Dispose()
	}
	for _, e := range s.outgoingEntries {
		e.controller.Dispose()
		e.animation.Dispose()
	}
}

func (s *animatedSwitcherState) Build(BuildContext) Widget {
	s.rebuildOutgoingWidgetsIfNeeded()
	var current Widget
	var currentKey foundation.Key
	if s.currentEntry != nil {
		current = s.currentEntry.transition
		currentKey = current.Key()
	}
	// The previous children, without the one that has the key of the current
	// child, and without duplicates.
	seen := map[Widget]struct{}{}
	var previous []Widget
	for _, w := range s.outgoingWidgets {
		if s.currentEntry != nil && w.Key() == currentKey {
			continue
		}
		if _, ok := seen[w]; ok {
			continue
		}
		seen[w] = struct{}{}
		previous = append(previous, w)
	}
	return s.Widget().layout()(current, previous)
}
