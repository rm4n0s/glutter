package semantics

import (
	"slices"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
)

// SemanticsHandle is a request for semantics. While at least one handle is open
// the framework produces a semantics tree. Dispose the handle when done.
//
// Flutter: semantics/binding.dart SemanticsHandle
type SemanticsHandle struct {
	onDispose func()
	disposed  bool
}

// Dispose releases the request. In debug builds disposing a handle twice panics
// with the tag SemanticsHandleDisposedTwice.
//
// Flutter: semantics/binding.dart SemanticsHandle.dispose
func (h *SemanticsHandle) Dispose() {
	if h.disposed {
		if foundation.DebugMode {
			panic(errors.New("SemanticsHandleDisposedTwice", "a semantics handle was disposed twice"))
		}
		return
	}
	h.disposed = true
	h.onDispose()
}

// SemanticsBinding is the glue between the semantics tree and the engine: it
// knows whether semantics are wanted, passes the actions of assistive
// technologies to the framework, and creates the update builders. Create one per
// app with NewSemanticsBinding.
//
// Flutter: semantics/binding.dart SemanticsBinding
type SemanticsBinding struct {
	dispatcher            *ui.PlatformDispatcher
	scheduler             *scheduler.SchedulerBinding
	semanticsEnabled      *foundation.ValueNotifier[bool]
	outstandingHandles    int
	semanticsHandle       *SemanticsHandle
	actionListeners       []*semanticsActionListener
	accessibilityFeatures ui.AccessibilityFeatures
	performAction         func(ui.SemanticsActionEvent)
	// DebugSemanticsDisableAnimations overrides DisableAnimations in debug
	// builds when it is set.
	//
	// Flutter: semantics/debug.dart debugSemanticsDisableAnimations
	DebugSemanticsDisableAnimations foundation.Opt[bool]
}

type semanticsActionListener struct{ fn func(ui.SemanticsActionEvent) }

// SemanticsBindingOpts are the optional parameters of NewSemanticsBinding.
//
// Flutter: semantics/binding.dart SemanticsBinding.initInstances
type SemanticsBindingOpts struct {
	// PerformSemanticsAction performs an action of an assistive technology; the
	// renderer binding passes the action to the semantics owners.
	PerformSemanticsAction func(action ui.SemanticsActionEvent)
}

// NewSemanticsBinding creates the binding and connects it to the dispatcher.
// The scheduler delays the accessibility features callback until after a frame
// that is being built.
//
// Flutter: semantics/binding.dart SemanticsBinding.initInstances
func NewSemanticsBinding(dispatcher *ui.PlatformDispatcher, sched *scheduler.SchedulerBinding, opts ...SemanticsBindingOpts) *SemanticsBinding {
	var o SemanticsBindingOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	b := &SemanticsBinding{
		dispatcher: dispatcher, scheduler: sched, performAction: o.PerformSemanticsAction,
		semanticsEnabled:      foundation.NewValueNotifier(dispatcher.SemanticsEnabled()),
		accessibilityFeatures: dispatcher.AccessibilityFeatures(),
	}
	dispatcher.OnSemanticsEnabledChanged = b.handleSemanticsEnabledChanged
	dispatcher.OnSemanticsActionEvent = b.handleSemanticsActionEvent
	dispatcher.OnAccessibilityFeaturesChanged = func() {
		if sched != nil && sched.SchedulerPhase() == scheduler.SchedulerPhasePersistentCallbacks {
			sched.AddPostFrameCallback(func(time.Duration) { b.HandleAccessibilityFeaturesChanged() })
		} else {
			b.HandleAccessibilityFeaturesChanged()
		}
	}
	b.handleSemanticsEnabledChanged()
	b.AddSemanticsEnabledListener(b.handleFrameworkSemanticsEnabledChanged)
	if b.SemanticsEnabled() {
		b.handleFrameworkSemanticsEnabledChanged()
	}
	return b
}

// SetPerformSemanticsAction sets the function that performs the actions of
// assistive technologies.
//
// Flutter: semantics/binding.dart SemanticsBinding.performSemanticsAction
func (b *SemanticsBinding) SetPerformSemanticsAction(fn func(ui.SemanticsActionEvent)) {
	b.performAction = fn
}

// SemanticsEnabled is whether semantics are wanted: an assistive technology
// asks for them or a handle is open.
//
// Flutter: semantics/binding.dart SemanticsBinding.semanticsEnabled
func (b *SemanticsBinding) SemanticsEnabled() bool { return b.semanticsEnabled.Value() }

// AddSemanticsEnabledListener calls the listener when SemanticsEnabled changes.
//
// Flutter: semantics/binding.dart SemanticsBinding.addSemanticsEnabledListener
func (b *SemanticsBinding) AddSemanticsEnabledListener(listener foundation.VoidCallback) foundation.ListenerHandle {
	return b.semanticsEnabled.AddListener(listener)
}

// RemoveSemanticsEnabledListener removes a listener.
//
// Flutter: semantics/binding.dart SemanticsBinding.removeSemanticsEnabledListener
func (b *SemanticsBinding) RemoveSemanticsEnabledListener(handle foundation.ListenerHandle) {
	b.semanticsEnabled.RemoveListener(handle)
}

// AddSemanticsActionListener calls the listener for each action of an assistive
// technology, before it is performed. It returns a function that removes it.
//
// Flutter: semantics/binding.dart SemanticsBinding.addSemanticsActionListener
func (b *SemanticsBinding) AddSemanticsActionListener(listener func(ui.SemanticsActionEvent)) (remove func()) {
	l := &semanticsActionListener{listener}
	b.actionListeners = append(b.actionListeners, l)
	return func() {
		b.actionListeners = slices.DeleteFunc(b.actionListeners, func(x *semanticsActionListener) bool { return x == l })
	}
}

// DebugOutstandingSemanticsHandles is the number of handles that are open.
//
// Flutter: semantics/binding.dart SemanticsBinding.debugOutstandingSemanticsHandles
func (b *SemanticsBinding) DebugOutstandingSemanticsHandles() int { return b.outstandingHandles }

// EnsureSemantics asks for semantics until the handle is disposed.
//
// Flutter: semantics/binding.dart SemanticsBinding.ensureSemantics
func (b *SemanticsBinding) EnsureSemantics() *SemanticsHandle {
	b.outstandingHandles++
	b.semanticsEnabled.SetValue(true)
	return &SemanticsHandle{onDispose: b.didDisposeSemanticsHandle}
}

func (b *SemanticsBinding) didDisposeSemanticsHandle() {
	b.outstandingHandles--
	b.semanticsEnabled.SetValue(b.outstandingHandles > 0)
}

func (b *SemanticsBinding) handleSemanticsEnabledChanged() {
	if b.dispatcher.SemanticsEnabled() {
		if b.semanticsHandle == nil {
			b.semanticsHandle = b.EnsureSemantics()
		}
	} else if b.semanticsHandle != nil {
		b.semanticsHandle.Dispose()
		b.semanticsHandle = nil
	}
}

func (b *SemanticsBinding) handleSemanticsActionEvent(action ui.SemanticsActionEvent) {
	local := slices.Clone(b.actionListeners)
	for _, l := range local {
		if slices.Contains(b.actionListeners, l) {
			l.fn(action)
		}
	}
	if b.performAction != nil {
		b.performAction(action)
	}
}

func (b *SemanticsBinding) handleFrameworkSemanticsEnabledChanged() {
	b.dispatcher.SetSemanticsTreeEnabled(b.SemanticsEnabled())
}

// AccessibilityFeatures are the accessibility settings of the system.
//
// Flutter: semantics/binding.dart SemanticsBinding.accessibilityFeatures
func (b *SemanticsBinding) AccessibilityFeatures() ui.AccessibilityFeatures {
	return b.accessibilityFeatures
}

// HandleAccessibilityFeaturesChanged reads the accessibility features again.
// An owner of a binding that needs to react to the change wraps it.
//
// Flutter: semantics/binding.dart SemanticsBinding.handleAccessibilityFeaturesChanged
func (b *SemanticsBinding) HandleAccessibilityFeaturesChanged() {
	b.accessibilityFeatures = b.dispatcher.AccessibilityFeatures()
}

// CreateSemanticsUpdateBuilder creates a builder for an update.
//
// Flutter: semantics/binding.dart SemanticsBinding.createSemanticsUpdateBuilder
func (b *SemanticsBinding) CreateSemanticsUpdateBuilder() *ui.SemanticsUpdateBuilder {
	return ui.NewSemanticsUpdateBuilder()
}

// DisableAnimations is whether animations should be off: the system asks for
// it, or in debug builds DebugSemanticsDisableAnimations says so.
//
// Flutter: semantics/binding.dart SemanticsBinding.disableAnimations
func (b *SemanticsBinding) DisableAnimations() bool {
	value := b.accessibilityFeatures.DisableAnimations
	if foundation.DebugMode {
		if v, ok := b.DebugSemanticsDisableAnimations.Get(); ok {
			value = v
		}
	}
	return value
}
