package widgets

import (
	"fmt"
	"slices"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/scheduler"
)

// TickerModeData is the effective ticker mode at a place in the tree: whether
// tickers tick and whether they force frames.
//
// Flutter: widgets/ticker_provider.dart TickerModeData
type TickerModeData struct {
	// Enabled is whether tickers below tick; a disabled ticker is muted.
	//
	// Flutter: widgets/ticker_provider.dart TickerModeData.enabled
	Enabled bool
	// ForceFrames is whether tickers below schedule frames even when frames are
	// disabled (see scheduler.Ticker.ForceFrames).
	//
	// Flutter: widgets/ticker_provider.dart TickerModeData.forceFrames
	ForceFrames bool
}

// TickerModeDataFallback is the mode where there is no TickerMode above: enabled,
// without forced frames.
//
// Flutter: widgets/ticker_provider.dart TickerModeData.fallback
var TickerModeDataFallback = TickerModeData{Enabled: true}

// constantTickerMode is a ValueListenable that never changes.
type constantTickerMode struct{ value TickerModeData }

func (constantTickerMode) AddListener(foundation.VoidCallback) foundation.ListenerHandle {
	return foundation.ListenerHandle{}
}
func (constantTickerMode) RemoveListener(foundation.ListenerHandle) {}
func (c constantTickerMode) Value() TickerModeData                  { return c.value }

var fallbackTickerMode foundation.ValueListenable[TickerModeData] = constantTickerMode{TickerModeDataFallback}

// TickerModeOpts are the optional parameters of NewTickerMode.
//
// Flutter: widgets/ticker_provider.dart TickerMode.new
type TickerModeOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// ForceFrames is whether the tickers below schedule frames even when frames
	// are disabled.
	ForceFrames bool
}

// TickerMode enables or disables the tickers (and with them the animations)
// of the widgets below it. A disabled ticker is muted: it keeps its time but
// does not tick, so the animations of the routes that are not visible don't
// run.
//
// Flutter: widgets/ticker_provider.dart TickerMode
type TickerMode struct {
	StatefulWidget
	enabled     bool
	forceFrames bool
	child       Widget
}

// NewTickerMode creates the widget. The mode it makes is the combination of its
// own and the ones above: enabled only if every one is, and forcing frames if
// any does.
//
// Flutter: widgets/ticker_provider.dart TickerMode.new
func NewTickerMode(enabled bool, child Widget, opts ...TickerModeOpts) *TickerMode {
	o := firstOpts(opts)
	return &TickerMode{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), enabled, o.ForceFrames, child}
}

// Enabled is whether the tickers below are enabled (if the modes above allow).
//
// Flutter: widgets/ticker_provider.dart TickerMode.enabled
func (w *TickerMode) Enabled() bool { return w.enabled }

// ForceFrames is whether the tickers below force frames.
//
// Flutter: widgets/ticker_provider.dart TickerMode.forceFrames
func (w *TickerMode) ForceFrames() bool { return w.forceFrames }

// Child is the widget below this one.
//
// Flutter: widgets/ticker_provider.dart TickerMode.child
func (w *TickerMode) Child() Widget { return w.child }

// CreateState creates the state.
//
// Flutter: widgets/ticker_provider.dart TickerMode.createState
func (w *TickerMode) CreateState() State {
	return &tickerModeState{
		effectiveMode:   foundation.NewValueNotifier(TickerModeDataFallback.Enabled),
		effectiveValues: foundation.NewValueNotifier(TickerModeDataFallback),
		ancestorEnabled: TickerModeDataFallback.Enabled,
	}
}

// TickerModeValuesOf is the ticker mode at the context, or the fallback. The
// context depends on the mode.
//
// Flutter: widgets/ticker_provider.dart TickerMode.valuesOf
func TickerModeValuesOf(bc BuildContext) TickerModeData {
	if w, ok := DependOn[*effectiveTickerMode](bc); ok {
		return w.values()
	}
	return TickerModeDataFallback
}

// TickerModeGetValuesNotifier is a ValueListenable of the ticker mode at the
// context. The context does not depend on it, so it can be used in InitState and
// Dispose; a context that is not mounted gets the fallback.
//
// Flutter: widgets/ticker_provider.dart TickerMode.getValuesNotifier
func TickerModeGetValuesNotifier(bc BuildContext) foundation.ValueListenable[TickerModeData] {
	if bc == nil || !bc.Mounted() {
		return fallbackTickerMode
	}
	if w, ok := GetInherited[*effectiveTickerMode](bc); ok {
		return w.valuesNotifier
	}
	return fallbackTickerMode
}

// TickerModeMergeOpts are the optional parameters of NewTickerModeMerge.
//
// Flutter: widgets/ticker_provider.dart TickerMode.merge
type TickerModeMergeOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Enabled replaces the enabled flag; the default is the one above.
	Enabled foundation.Opt[bool]
	// ForceFrames replaces the forceFrames flag; the default is the one above.
	ForceFrames foundation.Opt[bool]
}

// NewTickerModeMerge creates a TickerMode that takes the flags it doesn't set
// from the TickerMode above it.
//
// Flutter: widgets/ticker_provider.dart TickerMode.merge
func NewTickerModeMerge(child Widget, opts ...TickerModeMergeOpts) Widget {
	o := firstOpts(opts)
	return NewBuilder(func(bc BuildContext) Widget {
		parent, _ := DependOn[*effectiveTickerMode](bc)
		parentEnabled, parentForce := TickerModeDataFallback.Enabled, TickerModeDataFallback.ForceFrames
		if parent != nil {
			parentEnabled, parentForce = parent.enabled, parent.forceFrames
		}
		return NewTickerMode(o.Enabled.OrElse(parentEnabled), child, TickerModeOpts{Key: o.Key, ForceFrames: o.ForceFrames.OrElse(parentForce)})
	})
}

type tickerModeState struct {
	StateOf[*TickerMode]
	ancestorEnabled     bool
	ancestorForceFrames bool
	effectiveMode       *foundation.ValueNotifier[bool]
	effectiveValues     *foundation.ValueNotifier[TickerModeData]
}

func (s *tickerModeState) DidChangeDependencies() {
	parent, _ := DependOn[*effectiveTickerMode](s.Context())
	s.ancestorEnabled, s.ancestorForceFrames = TickerModeDataFallback.Enabled, TickerModeDataFallback.ForceFrames
	if parent != nil {
		s.ancestorEnabled, s.ancestorForceFrames = parent.enabled, parent.forceFrames
	}
	s.updateEffectiveMode()
}

func (s *tickerModeState) DidUpdateWidget(*TickerMode) { s.updateEffectiveMode() }

func (s *tickerModeState) Dispose() {
	s.effectiveMode.Dispose()
	s.effectiveValues.Dispose()
}

func (s *tickerModeState) updateEffectiveMode() {
	enabled := s.ancestorEnabled && s.Widget().enabled
	force := s.ancestorForceFrames || s.Widget().forceFrames
	s.effectiveMode.SetValue(enabled)
	s.effectiveValues.SetValue(TickerModeData{Enabled: enabled, ForceFrames: force})
}

func (s *tickerModeState) Build(BuildContext) Widget {
	return &effectiveTickerMode{
		InheritedWidget: NewInheritedWidgetBase(s.Widget().child),
		enabled:         s.effectiveMode.Value(), forceFrames: s.effectiveValues.Value().ForceFrames,
		notifier: s.effectiveMode, valuesNotifier: s.effectiveValues,
	}
}

// effectiveTickerMode is the inherited widget that TickerMode builds.
type effectiveTickerMode struct {
	InheritedWidget
	enabled        bool
	forceFrames    bool
	notifier       *foundation.ValueNotifier[bool]
	valuesNotifier *foundation.ValueNotifier[TickerModeData]
}

func (w *effectiveTickerMode) values() TickerModeData { return w.valuesNotifier.Value() }

func (w *effectiveTickerMode) UpdateShouldNotify(old Widget) bool {
	o := old.(*effectiveTickerMode)
	return w.enabled != o.enabled || w.forceFrames != o.forceFrames
}

// tickerHost is what SingleTickerProvider and TickerProvider share: the state
// they are embedded in, and the tickers' view of the app.
type tickerHost struct {
	state State
}

func (h *tickerHost) attach(s State) { h.state = s }

func (h *tickerHost) context() BuildContext {
	if h.state == nil {
		return nil
	}
	if el := h.state.stateInternals().data().element; el != nil {
		return el
	}
	return nil
}

func (h *tickerHost) binding() *WidgetsBinding { return MaybeWidgetsBindingOf(h.context()) }

// DisableAnimations is whether the system asks for animations to be off. A
// provider is the animation.Environment of the controllers that use it.
//
// Flutter: semantics/binding.dart SemanticsBinding.disableAnimations
func (h *tickerHost) DisableAnimations() bool {
	if b := h.binding(); b != nil {
		return b.Renderer().SemanticsBinding().DisableAnimations()
	}
	return false
}

// ErrorReporter is where the errors of the listeners of the controllers go: the
// error reporter of the app's binding.
//
// Flutter: foundation/assertions.dart FlutterError.onError
func (h *tickerHost) ErrorReporter() foundation.ErrorReporter {
	if b := h.binding(); b != nil {
		return b.ErrorReporter()
	}
	return nil
}

func (h *tickerHost) newTicker(onTick scheduler.TickerCallback) *scheduler.Ticker {
	b := h.binding()
	if b == nil {
		panic(errors.New(tagWidgetsBindingMissing,
			"a ticker was created by a State whose build owner does not belong to a WidgetsBinding; widgets run by RunApp and by gluttertest always have one"))
	}
	t := scheduler.NewTicker(b.Scheduler(), onTick)
	if foundation.DebugMode {
		t.DebugLabel = "created by " + foundation.DescribeIdentity(h.state)
	}
	return t
}

// SingleTickerProvider is embedded in a State that makes one Ticker, usually
// through one AnimationController: it is the TickerProvider of the controller
// (pass the state as the vsync), and mutes the ticker when a TickerMode above
// is disabled.
//
//	type spinnerState struct {
//		widgets.StateOf[*Spinner]
//		widgets.SingleTickerProvider
//		controller *animation.AnimationController
//	}
//
//	func (s *spinnerState) InitState() {
//		s.controller = animation.NewAnimationController(s, ...)
//	}
//
// A second CreateTicker panics with the tag SingleTickerProviderMultipleTickers
// in debug builds; use TickerProvider for a state that makes several. The
// ticker must be disposed (by disposing its controller) before the state is
// disposed, or Dispose panics with TickerProviderActiveTicker in debug builds.
//
// Flutter: widgets/ticker_provider.dart SingleTickerProviderStateMixin
type SingleTickerProvider struct {
	tickerHost
	ticker   *scheduler.Ticker
	notifier foundation.ValueListenable[TickerModeData]
	handle   foundation.ListenerHandle
}

func (p *SingleTickerProvider) attachStateMixin(s State) { p.attach(s) }

// CreateTicker makes the state's ticker.
//
// Flutter: widgets/ticker_provider.dart SingleTickerProviderStateMixin.createTicker
func (p *SingleTickerProvider) CreateTicker(onTick scheduler.TickerCallback) *scheduler.Ticker {
	if foundation.DebugMode && p.ticker != nil {
		panic(errors.New(tagSingleTickerProviderMultipleTickers,
			fmt.Sprintf("%s is a SingleTickerProvider but multiple tickers were created; a SingleTickerProvider can only be used as a TickerProvider once; if a State is used for multiple AnimationController objects, or if it is passed to other objects and those objects might use it more than one time in total, then instead of embedding a SingleTickerProvider, use a regular TickerProvider", foundation.RuntimeTypeName(p.state)),
			"state", foundation.RuntimeTypeName(p.state)))
	}
	p.ticker = p.newTicker(onTick)
	p.updateNotifier()
	p.updateTicker() // sets the ticker's muted flag
	return p.ticker
}

func (p *SingleTickerProvider) updateTicker() {
	values := p.notifier.Value()
	if p.ticker != nil {
		p.ticker.SetMuted(!values.Enabled)
		p.ticker.ForceFrames = values.ForceFrames
	}
}

func (p *SingleTickerProvider) updateNotifier() {
	n := TickerModeGetValuesNotifier(p.context())
	if p.notifier == n {
		return
	}
	if p.notifier != nil {
		p.notifier.RemoveListener(p.handle)
	}
	p.handle = n.AddListener(p.updateTicker)
	p.notifier = n
}

func (p *SingleTickerProvider) activateStateMixin() {
	// There may be a new TickerMode above.
	if p.ticker == nil {
		return
	}
	p.updateNotifier()
	p.updateTicker()
}

func (p *SingleTickerProvider) disposeStateMixin() {
	if foundation.DebugMode && p.ticker != nil && p.ticker.IsActive() {
		panic(errors.New(tagTickerProviderActiveTicker,
			fmt.Sprintf("%s was disposed with an active Ticker; %s created a Ticker via its SingleTickerProvider, but at the time its Dispose was called, that Ticker was still active; the Ticker must be disposed before the State is; tickers used by AnimationControllers should be disposed by calling Dispose on the AnimationController itself, otherwise the ticker will leak",
				foundation.DescribeIdentity(p.state), foundation.RuntimeTypeName(p.state)),
			"state", foundation.RuntimeTypeName(p.state), "ticker", p.ticker.String()))
	}
	if p.notifier != nil {
		p.notifier.RemoveListener(p.handle)
		p.notifier = nil
	}
}

// TickerProvider is embedded in a State that makes several Tickers (several
// controllers, or controllers that come and go). Like SingleTickerProvider it
// mutes the tickers when a TickerMode above is disabled.
//
// A ticker that is still active when the state is disposed makes Dispose panic
// with the tag TickerProviderActiveTicker in debug builds.
//
// Flutter: widgets/ticker_provider.dart TickerProviderStateMixin
type TickerProvider struct {
	tickerHost
	tickers  []*scheduler.Ticker
	notifier foundation.ValueListenable[TickerModeData]
	handle   foundation.ListenerHandle
}

func (p *TickerProvider) attachStateMixin(s State) { p.attach(s) }

// CreateTicker makes a ticker for the state.
//
// Flutter: widgets/ticker_provider.dart TickerProviderStateMixin.createTicker
func (p *TickerProvider) CreateTicker(onTick scheduler.TickerCallback) *scheduler.Ticker {
	if p.notifier == nil {
		p.updateNotifier() // set up the notifier before the first ticker
	}
	values := p.notifier.Value()
	t := p.newTicker(onTick)
	t.SetMuted(!values.Enabled)
	t.ForceFrames = values.ForceFrames
	t.OnDispose = func() { p.removeTicker(t) }
	p.tickers = append(p.tickers, t)
	return t
}

func (p *TickerProvider) removeTicker(t *scheduler.Ticker) {
	if i := slices.Index(p.tickers, t); i >= 0 {
		p.tickers = slices.Delete(p.tickers, i, i+1)
	}
}

func (p *TickerProvider) updateTickers() {
	values := p.notifier.Value()
	for _, t := range p.tickers {
		t.SetMuted(!values.Enabled)
		t.ForceFrames = values.ForceFrames
	}
}

func (p *TickerProvider) updateNotifier() {
	n := TickerModeGetValuesNotifier(p.context())
	if p.notifier == n {
		return
	}
	if p.notifier != nil {
		p.notifier.RemoveListener(p.handle)
	}
	p.handle = n.AddListener(p.updateTickers)
	p.notifier = n
}

func (p *TickerProvider) activateStateMixin() {
	// There may be a new TickerMode above.
	p.updateNotifier()
	p.updateTickers()
}

func (p *TickerProvider) disposeStateMixin() {
	if foundation.DebugMode {
		for _, t := range p.tickers {
			if t.IsActive() {
				panic(errors.New(tagTickerProviderActiveTicker,
					fmt.Sprintf("%s was disposed with an active Ticker; %s created a Ticker via its TickerProvider, but at the time its Dispose was called, that Ticker was still active; all Tickers must be disposed before the State is; tickers used by AnimationControllers should be disposed by calling Dispose on the AnimationController itself, otherwise the ticker will leak",
						foundation.DescribeIdentity(p.state), foundation.RuntimeTypeName(p.state)),
					"state", foundation.RuntimeTypeName(p.state), "ticker", t.String()))
			}
		}
	}
	if p.notifier != nil {
		p.notifier.RemoveListener(p.handle)
		p.notifier = nil
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagSingleTickerProviderMultipleTickers = "SingleTickerProviderMultipleTickers"
	tagTickerProviderActiveTicker          = "TickerProviderActiveTicker"
)
