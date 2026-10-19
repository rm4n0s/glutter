package widgets

import (
	"context"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/renderhost"
	"github.com/rm4n0s/glutter/internal/renderingtest"
	"github.com/rm4n0s/glutter/ui"
)

// bindingTree is a widgets binding on a fake engine with one view: what
// gluttertest's tester will be (step 7.5), without its API.
type bindingTree struct {
	t    *testing.T
	host *renderhost.Host
	d    *ui.PlatformDispatcher
	view *ui.View
	b    *WidgetsBinding
	raw  time.Duration
	errs []error
	cfg  ui.ViewConfiguration
	plat ui.PlatformConfiguration
}

type bindingTreeOpts struct {
	size ui.Size
	// ratio is the device pixel ratio; the default is 1.
	ratio float64
	// collectPanics collects recovered panics in errs instead of panicking again.
	collectPanics bool
	debug         *DebugOptions
}

func newBindingTree(t *testing.T, opts ...bindingTreeOpts) *bindingTree {
	t.Helper()
	o := firstOpts(opts)
	if o.size == (ui.Size{}) {
		o.size = ui.Size{Width: 800, Height: 600}
	}
	if o.ratio == 0 {
		o.ratio = 1
	}
	bt := &bindingTree{t: t, host: &renderhost.Host{}}
	bt.d = ui.NewPlatformDispatcher(bt.host)
	bt.d.SetTextEngine(renderingtest.BundledTextEngine())
	bt.cfg = ui.ViewConfiguration{PhysicalSize: o.size, DevicePixelRatio: o.ratio}
	bt.view = bt.d.AddView(ui.ImplicitViewID, bt.cfg)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	bt.b = NewWidgetsBinding(ctx, bt.d, WidgetsBindingOpts{
		Debug: o.debug,
		ErrorReporter: func(err error) {
			if !o.collectPanics && foundation.IsRecoveredPanic(err) {
				panic(err)
			}
			bt.errs = append(bt.errs, err)
		},
	})
	return bt
}

// attach wraps the widget in the default View and attaches it, as RunApp does.
func (bt *bindingTree) attach(w Widget) {
	bt.t.Helper()
	bt.b.AttachRootWidget(bt.b.WrapWithDefaultView(w))
}

// frame runs one frame, and what the engine posted.
func (bt *bindingTree) frame() {
	bt.t.Helper()
	bt.raw += 16 * time.Millisecond
	bt.b.Scheduler().HandleBeginFrame(foundation.Some(bt.raw))
	bt.b.Scheduler().HandleDrawFrame()
	bt.host.RunPosted()
}

// pump attaches the widget if nothing is attached, else replaces the widget,
// and runs a frame.
func (bt *bindingTree) pump(w Widget) {
	bt.t.Helper()
	if !bt.b.IsRootWidgetAttached() {
		bt.attach(w)
	} else {
		bt.b.AttachRootWidget(bt.b.WrapWithDefaultView(w))
	}
	bt.frame()
}

// setView changes the metrics of the view.
func (bt *bindingTree) setView(change func(c *ui.ViewConfiguration)) {
	change(&bt.cfg)
	bt.d.UpdateViewConfiguration(ui.ImplicitViewID, bt.cfg)
}

// setPlatform changes the system settings.
func (bt *bindingTree) setPlatform(change func(c *ui.PlatformConfiguration)) {
	change(&bt.plat)
	bt.d.UpdatePlatformConfiguration(bt.plat)
}

// find returns the elements below the root, in depth-first order, whose widget
// satisfies pred.
func (bt *bindingTree) find(pred func(Widget) bool) []Element {
	var found []Element
	var visit ElementVisitor
	visit = func(e Element) {
		if pred(e.AsElement().widget) {
			found = append(found, e)
		}
		e.VisitChildren(visit)
	}
	if root := bt.b.RootElement(); root != nil {
		root.VisitChildren(visit)
	}
	return found
}

// attachUnwrapped attaches the widget without the default View, as RunWidget
// does.
func (bt *bindingTree) attachUnwrapped(w Widget) { bt.b.AttachRootWidget(w) }
