package widgets

import (
	"context"
	"testing"
	"testing/fstest"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/renderhost"
	"github.com/rm4n0s/glutter/ui"
)

// startFake is what RunApp does once the engine has the window, on the fake
// engine.
func startFake(t *testing.T, app Widget, wrap bool, o RunAppOpts, withView bool) (*WidgetsBinding, *renderhost.Host, error) {
	t.Helper()
	host := &renderhost.Host{}
	d := ui.NewPlatformDispatcher(host)
	if withView {
		d.AddView(ui.ImplicitViewID, ui.ViewConfiguration{PhysicalSize: ui.Size{Width: 640, Height: 480}, DevicePixelRatio: 1})
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	if o.ErrorReporter == nil {
		o.ErrorReporter = func(err error) { t.Errorf("unexpected error: %v", err) }
	}
	b, err := startApp(ctx, d, app, wrap, o)
	return b, host, err
}

func TestStartAppShowsTheAppInTheDefaultView(t *testing.T) {
	var view *ui.View
	var size ui.Size
	b, host, err := startFake(t, builderOf(func(bc BuildContext) Widget {
		view = ViewOf(bc)
		size = MediaQuerySizeOf(bc)
		return newBox(10, 10)
	}), true, RunAppOpts{}, true)
	if err != nil {
		t.Fatal(err)
	}
	if !b.IsRootWidgetAttached() || b.IsIdle() {
		t.Errorf("attached = %v, idle = %v; want attached, and not idle with the first frame scheduled", b.IsRootWidgetAttached(), b.IsIdle())
	}
	if host.Scheduled != 1 {
		t.Errorf("%d frames asked for, want the first one", host.Scheduled)
	}
	b.Scheduler().HandleBeginFrame(foundation.Some(time.Duration(0)))
	b.Scheduler().HandleDrawFrame()
	if view == nil || view != b.PlatformDispatcher().ImplicitView() || size != (ui.Size{Width: 640, Height: 480}) {
		t.Errorf("view = %v, size = %v; want the implicit view at 640x480", view, size)
	}
	if len(host.Scenes()) != 1 {
		t.Errorf("%d scenes, want 1", len(host.Scenes()))
	}
	if !b.IsIdle() {
		t.Error("the app is not idle after the first frame")
	}
}

func TestStartAppWithoutAViewForRunApp(t *testing.T) {
	_, _, err := startFake(t, newLeaf(), true, RunAppOpts{}, false)
	if err == nil {
		t.Fatal("RunApp without an implicit view returned no error")
	}
	_ = requireErrorTag(t, err, "ImplicitViewMissing")
}

func TestStartAppRunWidgetDoesNotWrap(t *testing.T) {
	var outside *ui.View
	b, _, err := startFake(t, builderOf(func(bc BuildContext) Widget {
		outside = MaybeViewOf(bc)
		return newLeaf()
	}), false, RunAppOpts{}, false)
	if err != nil {
		t.Fatal(err)
	}
	b.Scheduler().HandleBeginFrame(foundation.Some(time.Duration(0)))
	b.Scheduler().HandleDrawFrame()
	if outside != nil {
		t.Error("RunWidget wrapped the widget in a View")
	}
}

func TestStartAppAssetBundle(t *testing.T) {
	assets := fstest.MapFS{"hello.txt": {Data: []byte("hi")}}
	b, _, err := startFake(t, newLeaf(), false, RunAppOpts{Assets: assets}, false)
	if err != nil {
		t.Fatal(err)
	}
	data, err := b.Painting().AssetBundle().Load(context.Background(), "hello.txt")
	if err != nil || string(data) != "hi" {
		t.Errorf("asset = %q (%v), want hi", data, err)
	}
}

func TestStartAppSetsTheDispatcherHooks(t *testing.T) {
	b, _, err := startFake(t, newLeaf(), false, RunAppOpts{}, false)
	if err != nil {
		t.Fatal(err)
	}
	d := b.PlatformDispatcher()
	if d.IsIdle == nil || d.OnExitRequested == nil || d.OnAppLifecycleStateChanged == nil {
		t.Error("the dispatcher is missing the hooks of the binding")
	}
}

func TestStartAppErrorReporterAndBuilder(t *testing.T) {
	var reported []error
	b, _, err := startFake(t, builderOf(func(BuildContext) Widget { panic("boom") }), false, RunAppOpts{
		ErrorReporter: func(err error) { reported = append(reported, err) },
		ErrorWidgetBuilder: func(error) Widget {
			return newLeaf()
		},
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	b.Scheduler().HandleBeginFrame(foundation.Some(time.Duration(0)))
	b.Scheduler().HandleDrawFrame()
	if len(reported) != 1 {
		t.Fatalf("%d errors reported, want 1: %v", len(reported), reported)
	}
	_ = requireErrorTag(t, reported[0], foundation.BuildPanicked)
}

func TestRunAppWindowConfig(t *testing.T) {
	tests := []struct {
		name string
		opts RunAppOpts
		want platform.WindowConfig
	}{
		{"defaults", RunAppOpts{}, platform.WindowConfig{Title: "Glutter"}},
		{"everything", RunAppOpts{AppID: "dev.example.app", Window: WindowConfig{
			Title: "Counter", Size: ui.Size{Width: 400, Height: 300}, MinSize: ui.Size{Width: 100, Height: 100},
			MaxSize: ui.Size{Width: 900, Height: 900}, Resizable: foundation.Some(false), Decorated: foundation.Some(false),
			State: WindowStateMaximized,
		}}, platform.WindowConfig{Title: "Counter", AppID: "dev.example.app", Size: ui.Size{Width: 400, Height: 300},
			MinSize: ui.Size{Width: 100, Height: 100}, MaxSize: ui.Size{Width: 900, Height: 900},
			Resizable: foundation.Some(false), Decorated: foundation.Some(false), State: platform.WindowStateMaximized}},
		{"fullscreen", RunAppOpts{Window: WindowConfig{State: WindowStateFullscreen}},
			platform.WindowConfig{Title: "Glutter", State: platform.WindowStateFullscreen}},
		{"minimized", RunAppOpts{Window: WindowConfig{State: WindowStateMinimized}},
			platform.WindowConfig{Title: "Glutter", State: platform.WindowStateMinimized}},
		{"normal", RunAppOpts{Window: WindowConfig{State: WindowStateNormal}},
			platform.WindowConfig{Title: "Glutter", State: platform.WindowStateNormal}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.opts.platformWindow(); got != tc.want {
				t.Errorf("window = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestIsIdle(t *testing.T) {
	bt := newBindingTree(t)
	if bt.b.IsIdle() {
		t.Error("a binding with no root is idle")
	}
	bt.pump(newLeaf())
	if !bt.b.IsIdle() {
		t.Error("the app is not idle after its first frame")
	}
	bt.b.Scheduler().ScheduleFrame()
	if bt.b.IsIdle() {
		t.Error("the app is idle with a frame scheduled")
	}
	bt.frame()
	bt.b.Scheduler().ScheduleFrameCallback(func(time.Duration) {})
	if bt.b.IsIdle() {
		t.Error("the app is idle with an animation running")
	}
}
