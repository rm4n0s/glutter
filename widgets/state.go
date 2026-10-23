package widgets

import (
	"context"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// StateLifecycle is the stage of a State's life, kept for debug checks.
//
// Flutter: widgets/framework.dart _StateLifecycle
type StateLifecycle int

const (
	// StateLifecycleCreated is a state that has been created and not yet
	// initialized.
	//
	// Flutter: widgets/framework.dart _StateLifecycle.created
	StateLifecycleCreated StateLifecycle = iota
	// StateLifecycleInitialized is a state whose InitState has run.
	//
	// Flutter: widgets/framework.dart _StateLifecycle.initialized
	StateLifecycleInitialized
	// StateLifecycleReady is a state that is built and in the tree.
	//
	// Flutter: widgets/framework.dart _StateLifecycle.ready
	StateLifecycleReady
	// StateLifecycleDefunct is a state that has been disposed.
	//
	// Flutter: widgets/framework.dart _StateLifecycle.defunct
	StateLifecycleDefunct
)

var stateLifecycleNames = [...]string{"created", "initialized", "ready", "defunct"}

// String is the name of the stage ("ready").
func (l StateLifecycle) String() string { return stateLifecycleNames[l] }

// Name is the name of the stage, for diagnostics.
func (l StateLifecycle) Name() string { return l.String() }

// State is the logic and internal state for a StatefulWidget. A state type
// embeds StateOf[W], implements Build, and implements any of the optional
// lifecycle interfaces it needs:
//
//   - InitState()
//   - DidChangeDependencies()
//   - DidUpdateWidget(old W)
//   - Deactivate()
//   - Activate()
//   - Dispose()
//   - Reassemble()
//
// The framework does its own bookkeeping, so there is nothing like Flutter's
// super.initState() to call.
//
// Flutter: widgets/framework.dart State
type State interface {
	// Build describes the part of the user interface represented by the
	// widget.
	//
	// Flutter: widgets/framework.dart State.build
	Build(bc BuildContext) Widget
	stateInternals() stateInternals
}

// stateInternals is what StateOf[W] gives the framework, which cannot name W.
type stateInternals interface {
	data() *stateData
	// bind sets the widget; it reports false when the widget is not a W.
	bind(w Widget) bool
	// swapWidget sets the widget and returns the old one.
	swapWidget(w Widget) Widget
	// widgetAny is the current widget, or nil.
	widgetAny() Widget
	// didUpdateWidget calls the DidUpdateWidget method of the outer state, if it
	// has one, with the old widget.
	didUpdateWidget(old Widget)
}

// Optional lifecycle interfaces of a State.
type (
	initStater                  interface{ InitState() }
	didChangeDependenciesStater interface{ DidChangeDependencies() }
	deactivateStater            interface{ Deactivate() }
	activateStater              interface{ Activate() }
	disposeStater               interface{ Dispose() }
	reassembleStater            interface{ Reassemble() }
)

// stateData is the non-generic part of StateOf.
type stateData struct {
	self      State
	element   *StatefulElement
	lifecycle StateLifecycle
	lifetime  context.Context
	cancel    context.CancelFunc
	// owner is the build owner of the element. It is set once, on the UI thread,
	// when the state is first built, and never cleared, so that other goroutines
	// can read it.
	owner *BuildOwner
	// mixins are the structs embedded in the state that hook into its
	// lifecycle.
	mixins []stateMixin
}

// StateOf is the embedded base of a State for the widget type W. It gives the
// state its widget, its context and SetState.
//
//	type counterState struct {
//		widgets.StateOf[*Counter]
//		count int
//	}
//
// Flutter: widgets/framework.dart State
type StateOf[W Widget] struct {
	stateData
	widget W
	// hasWidget is whether widget holds a widget: it is false before the state
	// is mounted and after it is unmounted.
	hasWidget bool
}

func (s *StateOf[W]) data() *stateData { return &s.stateData }

func (s *StateOf[W]) stateInternals() stateInternals { return s }

func (s *StateOf[W]) bind(w Widget) bool {
	typed, ok := w.(W)
	if !ok {
		return false
	}
	s.widget, s.hasWidget = typed, true
	return true
}

func (s *StateOf[W]) swapWidget(w Widget) Widget {
	old := s.widgetAny()
	s.widget, s.hasWidget = w.(W), true
	return old
}

func (s *StateOf[W]) widgetAny() Widget {
	if !s.hasWidget {
		return nil
	}
	return s.widget
}

func (s *StateOf[W]) didUpdateWidget(old Widget) {
	if h, ok := s.self.(interface{ DidUpdateWidget(old W) }); ok {
		h.DidUpdateWidget(old.(W))
	}
}

// Widget is the current configuration of the state. A state is created before
// its widget is known; the framework sets it before InitState.
//
// Flutter: widgets/framework.dart State.widget
func (s *StateOf[W]) Widget() W { return s.widget }

// Context is the location in the tree where this widget builds. It panics with
// the tag StateContextUnmounted in debug builds once the state is unmounted.
//
// Flutter: widgets/framework.dart State.context
func (s *StateOf[W]) Context() BuildContext {
	if s.element == nil {
		if foundation.DebugMode {
			panic(errors.New(tagStateContextUnmounted,
				"this widget has been unmounted, so the State no longer has a context (and should be considered defunct); consider canceling any active work during Dispose or using Mounted to determine if the State is still active"))
		}
		return nil
	}
	return s.element
}

// Mounted is whether this state is currently in the tree: true from the moment
// the framework attaches it until Dispose. Check it before SetState after any
// asynchronous gap.
//
// Flutter: widgets/framework.dart State.mounted
func (s *StateOf[W]) Mounted() bool { return s.element != nil }

// Lifetime is a context.Context that is cancelled when the state is disposed.
// Goroutines that outlive a single call should stop when it is done.
//
// Flutter: no counterpart (SPEC §12.6)
func (s *StateOf[W]) Lifetime() context.Context {
	if s.lifetime == nil {
		s.lifetime, s.cancel = context.WithCancel(context.Background())
		if s.lifecycle == StateLifecycleDefunct {
			s.cancel()
		}
	}
	return s.lifetime
}

// DebugLifecycleState is the stage of the state's life. Flutter keeps it only
// in debug builds.
//
// Flutter: widgets/framework.dart State._debugLifecycleState
func (s *StateOf[W]) DebugLifecycleState() StateLifecycle { return s.lifecycle }

// SetState notifies the framework that the internal state of this object has
// changed in a way that may affect the user interface: it runs fn and marks the
// element dirty, so that Build runs again.
//
// SetState is safe to call from any goroutine (SPEC §13). On another goroutine
// it posts fn and the rebuild to the UI thread, where they run in order; if the
// state has been unmounted by then, both are dropped. Without a Post on the
// build owner it must be called on the UI thread.
//
// Calling it after Dispose panics with the tag SetStateAfterDispose, and
// calling it before the state is in the tree panics with SetStateBeforeMount
// (debug builds).
//
// Flutter: widgets/framework.dart State.setState
func (s *StateOf[W]) SetState(fn func()) {
	if o := s.owner; o != nil && o.Post != nil && !o.UIThread.IsCurrent() {
		o.Post(func() {
			if s.element != nil {
				s.SetState(fn)
			}
		})
		return
	}
	if foundation.DebugMode {
		if s.lifecycle == StateLifecycleDefunct {
			panic(errors.New(tagSetStateAfterDispose,
				"SetState was called after Dispose on a State whose widget no longer appears in the widget tree; this can happen when code calls SetState from a timer, from an animation callback, or after an asynchronous operation (such as a request or other goroutine) completes after the widget has been removed from the tree; the preferred solution is to cancel the timer or stop listening to the animation in Dispose, another solution is to check Mounted before calling SetState (for example after an await-like gap), and a State that is retained after it left the tree might indicate a memory leak",
				"state", foundation.RuntimeTypeName(s.self)))
		}
		if s.lifecycle == StateLifecycleCreated && !s.Mounted() {
			panic(errors.New(tagSetStateBeforeMount,
				"SetState was called on a State whose widget has not been inserted into the widget tree yet; it is not necessary to call SetState in the constructor, since the state is already assumed to be dirty when it is initially created",
				"state", foundation.RuntimeTypeName(s.self)))
		}
	}
	if fn != nil {
		fn()
	}
	if s.element != nil {
		s.element.MarkNeedsBuild()
	}
}

// Go runs fn on a new goroutine with the Lifetime of the state as its context,
// which is cancelled when the state is disposed. A panic in fn does not crash
// the program: it is recovered and reported as a CallbackPanicked error through
// the framework's error reporter, on the UI thread, like the panics in the
// other callbacks (SPEC §14.2). Call Go on the UI thread, after InitState began;
// in debug builds calling it before the state is built panics with the tag
// StateGoBeforeMount. Inside fn, use SetState to touch the UI.
//
// Flutter: no counterpart (SPEC §12.6)
func (s *StateOf[W]) Go(fn func(ctx context.Context)) {
	o := s.owner
	if o == nil {
		panic(errors.New(tagStateGoBeforeMount,
			"Go was called on a State that has not been built yet; call it from InitState or later",
			"state", foundation.RuntimeTypeName(s.self)))
	}
	o.UIThread.AssertCurrent("State.Go")
	ctx := s.Lifetime()
	var report foundation.ErrorReporter
	if o.OnError != nil {
		report = o.reportFromAnyGoroutine
	}
	state := foundation.RuntimeTypeName(s.self)
	go foundation.Guard(report, foundation.CallbackPanicked, "in a goroutine started by State.Go of "+state,
		func() []any { return []any{"library", "widgets library", "state", state} },
		func() { fn(ctx) })
}

// ToStringShort describes the state by its type and short hash.
func (s *StateOf[W]) ToStringShort() string { return foundation.DescribeIdentity(s.self) }

// String describes the state, such as "_CounterState#1a2b3(lifecycle state:
// created, no widget, not mounted)". A state that is mounted and ready has
// nothing to add to its short description.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (s *StateOf[W]) String() string {
	var d foundation.Diagnosticable = s
	short := foundation.DescribeIdentity(s)
	if s.self != nil {
		short = foundation.DescribeIdentity(s.self)
		if outer, ok := s.self.(foundation.Diagnosticable); ok {
			d = outer
		}
	}
	return foundation.DiagnosticableString(stateDescription{d: d, short: short})
}

// DebugFillProperties adds the lifecycle state, the widget and the element.
//
// Flutter: widgets/framework.dart State.debugFillProperties
func (s *StateOf[W]) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	if foundation.DebugMode {
		properties.Add(foundation.NewEnumProperty("lifecycle state", foundation.Some(s.lifecycle),
			foundation.EnumPropertyOpts{DefaultValue: StateLifecycleReady}))
	}
	properties.Add(foundation.NewObjectFlagProperty("_widget", s.widgetAny(), foundation.ObjectFlagPropertyOpts{IfNull: "no widget"}))
	var el Element
	if s.element != nil {
		el = s.element
	}
	properties.Add(foundation.NewObjectFlagProperty("_element", el, foundation.ObjectFlagPropertyOpts{IfNull: "not mounted"}))
}

// stateDescription lets a state print itself under the name of its outer type.
type stateDescription struct {
	d     foundation.Diagnosticable
	short string
}

func (d stateDescription) ToStringShort() string { return d.short }

func (d stateDescription) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	d.d.DebugFillProperties(p)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagSetStateAfterDispose  = "SetStateAfterDispose"
	tagSetStateBeforeMount   = "SetStateBeforeMount"
	tagStateContextUnmounted = "StateContextUnmounted"
	tagStateGoBeforeMount    = "StateGoBeforeMount"
)
