package engine

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/internal/dbus/fakebus"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/fake"
	"github.com/rm4n0s/glutter/ui"
)

// recorder collects what the dispatcher's callbacks saw, from the fake UI thread.
type recorder struct {
	mu   sync.Mutex
	log  []string
	disp *ui.PlatformDispatcher
}

func (r *recorder) add(s string) { r.mu.Lock(); r.log = append(r.log, s); r.mu.Unlock() }

func (r *recorder) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.log...)
}

func (r *recorder) dispatcher() *ui.PlatformDispatcher {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.disp
}

// start is Config.Start: it keeps the dispatcher for the test.
func (r *recorder) start(d *ui.PlatformDispatcher) error {
	r.mu.Lock()
	r.disp = d
	r.mu.Unlock()
	return nil
}

// onUI runs fn on the fake UI thread and waits for it, so that a test can read
// the dispatcher, which belongs to that thread, while Run is going.
func onUI(t *testing.T, h *harness, fn func()) {
	t.Helper()
	done := make(chan struct{})
	h.c.Post(func() { fn(); close(done) })
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the UI thread did not run the function")
	}
}

func newDispatcherHarness(t *testing.T) (*harness, *recorder) {
	t.Helper()
	h := newHarness(t, map[string]string{"DISPLAY": ":0"})
	r := &recorder{}
	h.cfg.Start = r.start
	return h, r
}

// finish stops the engine and checks that Run returned without an error.
func finish(t *testing.T, h *harness, done <-chan error) {
	t.Helper()
	h.stopEngine(t)
	if err := result(t, done); err != nil {
		t.Fatal(err)
	}
}

func TestStartGetsTheDispatcherWithItsViewSettingsAndDisplays(t *testing.T) {
	h, r := newDispatcherHarness(t)
	h.c.SetDisplays([]platform.Display{
		{ID: 1, Bounds: ui.NewRectFromLTRB(0, 0, 960, 540), DevicePixelRatio: 2, RefreshRate: 144},
		{ID: 2, Bounds: ui.NewRectFromLTRB(960, 0, 2880, 1080), DevicePixelRatio: 1, RefreshRate: 60, Primary: true},
	})
	h.c.SetSettings(platform.Settings{
		PlatformBrightness: ui.BrightnessDark,
		TextScaleFactor:    1.5,
		HasAccentColor:     true,
		AccentColor:        ui.NewColor(0xFF336699),
		DisableAnimations:  true,
		HighContrast:       true,
		Locales:            []ui.Locale{ui.NewLocale("de", "DE"), ui.NewLocale("en")},
	})
	var seen struct {
		views    int
		implicit bool
		started  bool
	}
	h.cfg.Start = func(d *ui.PlatformDispatcher) error {
		seen.views, seen.implicit, seen.started = len(d.Views()), d.ImplicitView() != nil, true
		return r.start(d)
	}
	done := h.run(t)
	finish(t, h, done)
	d := r.dispatcher()
	if !seen.started || seen.views != 1 || !seen.implicit {
		t.Errorf("at Start: %+v, want one implicit view", seen)
	}
	if got := d.Displays(); len(got) != 2 || got[0].Size != (ui.Size{Width: 1920, Height: 1080}) || got[0].RefreshRate != 144 ||
		got[1].Size != (ui.Size{Width: 1920, Height: 1080}) || got[1].ID != 2 {
		t.Errorf("displays = %v", got)
	}
	if d.PlatformBrightness() != ui.BrightnessDark || d.TextScaleFactor() != 1.5 || d.Locale() != ui.NewLocale("de", "DE") ||
		!d.AccessibilityFeatures().DisableAnimations || !d.AccessibilityFeatures().HighContrast ||
		d.AccentColor() == nil || *d.AccentColor() != ui.NewColor(0xFF336699) {
		t.Errorf("platform configuration: %v %v %v %v", d.PlatformBrightness(), d.TextScaleFactor(), d.Locales(), d.AccessibilityFeatures())
	}
}

func TestStartErrorEndsRun(t *testing.T) {
	h, _ := newDispatcherHarness(t)
	h.cfg.Start = func(*ui.PlatformDispatcher) error { return errors.New(tagStartFailed, "no") }
	err := Run(t.Context(), h.cfg)
	assertRoute(t, err, "StartFailed", "StartFailed")
	if !h.c.closed {
		t.Error("the connection was not closed")
	}
}

func TestMetricsReachTheView(t *testing.T) {
	h, r := newDispatcherHarness(t)
	h.c.SetDisplays([]platform.Display{
		{ID: 1, Bounds: ui.NewRectFromLTRB(0, 0, 100, 100), DevicePixelRatio: 1, RefreshRate: 50, Primary: true},
		{ID: 2, Bounds: ui.NewRectFromLTRB(100, 0, 300, 100), DevicePixelRatio: 2, RefreshRate: 90},
	})
	type snap struct {
		size    ui.Size
		dpr     float64
		padding ui.ViewPadding
		insets  ui.ViewPadding
		display ui.Display
	}
	var mu sync.Mutex
	var snaps []snap
	h.cfg.Start = func(d *ui.PlatformDispatcher) error {
		d.OnMetricsChanged = func() {
			v := d.ImplicitView()
			mu.Lock()
			snaps = append(snaps, snap{v.PhysicalSize(), v.DevicePixelRatio(), v.Padding(), v.ViewInsets(), v.Display()})
			mu.Unlock()
		}
		return r.start(d)
	}
	h.c.Inject(metrics(640, 400))
	h.c.Inject(fake.MetricsEvent{View: 0, Metrics: platform.ViewMetrics{
		PhysicalSize: ui.Size{Width: 800, Height: 600}, DevicePixelRatio: 1.5,
		ViewPadding: ui.ViewPadding{Top: 24, Bottom: 10}, ViewInsets: ui.ViewPadding{Bottom: 30},
		Display: foundation.Some(2),
	}})
	done := h.run(t)
	waitFor(t, func() bool { mu.Lock(); defer mu.Unlock(); return len(snaps) == 2 })
	finish(t, h, done)
	mu.Lock()
	defer mu.Unlock()
	if s := snaps[0]; s.size.Width != 640 || s.size.Height != 400 || s.dpr != 1 || s.display.ID != 1 || s.display.RefreshRate != 50 {
		t.Errorf("first metrics = %+v: the primary display is the default", s)
	}
	// The padding is the view padding the insets do not already cover.
	if s := snaps[1]; s.size.Width != 800 || s.dpr != 1.5 || s.padding != (ui.ViewPadding{Top: 24}) ||
		s.insets != (ui.ViewPadding{Bottom: 30}) || s.display.ID != 2 || s.display.Size.Width != 400 {
		t.Errorf("second metrics = %+v", s)
	}
}

func TestMetricsWithoutDisplaysGetADefaultDisplay(t *testing.T) {
	h, r := newDispatcherHarness(t)
	h.c.Inject(metrics(320, 200))
	done := h.run(t)
	var disp ui.Display
	waitFor(t, func() bool {
		ok := false
		onUI(t, h, func() {
			d := r.dispatcher()
			ok = d != nil && d.ImplicitView().PhysicalSize().Width == 320
			if ok {
				disp = d.ImplicitView().Display()
			}
		})
		return ok
	})
	finish(t, h, done)
	if disp.RefreshRate != 60 || disp.Size.Width != 320 || disp.DevicePixelRatio != 1 {
		t.Errorf("display = %v", disp)
	}
}

func TestFramesCallBeginAndDrawWithMonotonicTimes(t *testing.T) {
	h, r := newDispatcherHarness(t)
	var times []time.Duration
	var mu sync.Mutex
	h.cfg.Start = func(d *ui.PlatformDispatcher) error {
		d.OnBeginFrame = func(ft time.Duration) { r.add("begin"); mu.Lock(); times = append(times, ft); mu.Unlock() }
		d.OnDrawFrame = func() {
			r.add("draw")
			if err := d.ImplicitView().Render(&ui.Scene{}); err != nil {
				t.Error(err)
			}
		}
		return r.start(d)
	}
	h.c.Inject(metrics(100, 50))
	for _, ft := range []time.Duration{16 * time.Millisecond, 33 * time.Millisecond, 20 * time.Millisecond, 66 * time.Millisecond} {
		h.c.Inject(fake.FrameEvent{View: 0, FrameTime: ft})
	}
	done := h.run(t)
	waitFor(t, func() bool { calls, _ := h.r.snapshot(); return len(calls) == 4 })
	finish(t, h, done)
	want := []string{"begin", "draw", "begin", "draw", "begin", "draw", "begin", "draw"}
	if got := r.snapshot(); !reflect.DeepEqual(got, want) {
		t.Errorf("callbacks = %v", got)
	}
	mu.Lock()
	defer mu.Unlock()
	wantTimes := []time.Duration{16 * time.Millisecond, 33 * time.Millisecond, 33 * time.Millisecond, 66 * time.Millisecond}
	if !reflect.DeepEqual(times, wantTimes) {
		t.Errorf("frame times = %v, want %v", times, wantTimes)
	}
	calls, _ := h.r.snapshot()
	for i, c := range calls {
		if c.width != 100 || c.height != 50 || c.scene == nil {
			t.Errorf("draw %d = %+v, want the scene at 100x50", i, c)
		}
	}
}

func TestScheduleFrameAsksTheWindowForAFrame(t *testing.T) {
	h, r := newDispatcherHarness(t)
	done := h.run(t)
	waitFor(t, func() bool { return r.dispatcher() != nil })
	count := func() int {
		n := 0
		for _, e := range h.c.Log() {
			if e.Kind == fake.EntryRequestFrame {
				n++
			}
		}
		return n
	}
	before := count()
	onUI(t, h, func() { r.dispatcher().ScheduleFrame(); r.dispatcher().ScheduleFrame() })
	if got := count() - before; got != 2 {
		t.Errorf("RequestFrame entries = %d, want 2 (the platform merges the requests)", got)
	}
	finish(t, h, done)
}

func TestAFrameThatRendersNothingDrawsTheLastSceneAgain(t *testing.T) {
	h, r := newDispatcherHarness(t)
	scene := &ui.Scene{}
	renderOnce := true
	h.cfg.Start = func(d *ui.PlatformDispatcher) error {
		d.OnDrawFrame = func() {
			if renderOnce {
				renderOnce = false
				if err := d.ImplicitView().Render(scene); err != nil {
					t.Error(err)
				}
			}
		}
		return r.start(d)
	}
	h.c.Inject(metrics(64, 64))
	h.c.Inject(fake.FrameEvent{View: 0, FrameTime: time.Millisecond})
	h.c.Inject(fake.FrameEvent{View: 0, FrameTime: 2 * time.Millisecond})
	h.c.Inject(fake.FrameEvent{View: 0, FrameTime: 3 * time.Millisecond})
	done := h.run(t)
	waitFor(t, func() bool { calls, _ := h.r.snapshot(); return len(calls) == 3 })
	finish(t, h, done)
	calls, _ := h.r.snapshot()
	for i, c := range calls {
		if c.scene != scene {
			t.Errorf("draw %d got scene %p, want the one the framework rendered (%p)", i, c.scene, scene)
		}
	}
}

func TestAFrameWithNoFrameworkDrawsTheDevelopmentColor(t *testing.T) {
	h := newHarness(t, map[string]string{"DISPLAY": ":0"})
	h.c.Inject(metrics(10, 10))
	h.c.Inject(fake.FrameEvent{View: 0})
	done := h.run(t)
	waitFor(t, func() bool { calls, _ := h.r.snapshot(); return len(calls) == 1 })
	finish(t, h, done)
	if calls, _ := h.r.snapshot(); calls[0].scene != nil {
		t.Errorf("scene = %v, want nil: the background only", calls[0].scene)
	}
}

func TestSceneFuncStillWorksOnTheDispatcher(t *testing.T) {
	h := newHarness(t, map[string]string{"DISPLAY": ":0"})
	scene := &ui.Scene{}
	var mu sync.Mutex
	var sizes []ui.Size
	var dprs []float64
	h.cfg.Scene = func(size ui.Size, dpr float64) *ui.Scene {
		mu.Lock()
		sizes, dprs = append(sizes, size), append(dprs, dpr)
		mu.Unlock()
		return scene
	}
	h.c.Inject(fake.MetricsEvent{View: 0, Metrics: platform.ViewMetrics{PhysicalSize: ui.Size{Width: 90, Height: 30}, DevicePixelRatio: 2}})
	h.c.Inject(fake.FrameEvent{View: 0})
	done := h.run(t)
	waitFor(t, func() bool { calls, _ := h.r.snapshot(); return len(calls) == 1 })
	finish(t, h, done)
	mu.Lock()
	defer mu.Unlock()
	if calls, _ := h.r.snapshot(); calls[0].scene != scene || len(sizes) != 1 || sizes[0].Width != 90 || dprs[0] != 2 {
		t.Errorf("scene func: sizes %v dprs %v", sizes, dprs)
	}
}

func TestSettingsAndLifecycleReachTheDispatcher(t *testing.T) {
	h, r := newDispatcherHarness(t)
	h.cfg.Start = func(d *ui.PlatformDispatcher) error {
		d.OnPlatformBrightnessChanged = func() { r.add("brightness " + d.PlatformBrightness().String()) }
		d.OnTextScaleFactorChanged = func() { r.add("scale") }
		d.OnLocaleChanged = func() { r.add("locale " + d.Locale().String()) }
		d.OnAccessibilityFeaturesChanged = func() { r.add("accessibility " + d.AccessibilityFeatures().String()) }
		d.OnAppLifecycleStateChanged = func(s ui.AppLifecycleState) { r.add("lifecycle " + s.String()) }
		return r.start(d)
	}
	h.c.Inject(fake.LifecycleEvent{State: ui.AppLifecycleStateResumed})
	h.c.Inject(fake.SettingsEvent{Settings: platform.Settings{PlatformBrightness: ui.BrightnessDark, Locales: []ui.Locale{ui.NewLocale("fr")}}})
	h.c.Inject(fake.SettingsEvent{Settings: platform.Settings{PlatformBrightness: ui.BrightnessDark, Locales: []ui.Locale{ui.NewLocale("fr")}, TextScaleFactor: 2, DisableAnimations: true}})
	h.c.Inject(fake.LifecycleEvent{State: ui.AppLifecycleStateInactive})
	done := h.run(t)
	waitFor(t, func() bool { return len(r.snapshot()) >= 6 })
	finish(t, h, done)
	want := []string{
		"lifecycle AppLifecycleStateResumed",
		"locale fr", "brightness BrightnessDark",
		"scale", "accessibility AccessibilityFeatures[disableAnimations, reduceMotion]",
		"lifecycle AppLifecycleStateInactive",
	}
	if got := r.snapshot(); !reflect.DeepEqual(got, want) {
		t.Errorf("callbacks = %v\nwant %v", got, want)
	}
}

func TestDesktopFontsReachTheDispatcher(t *testing.T) {
	h, r := newDispatcherHarness(t)
	h.cfg.Start = func(d *ui.PlatformDispatcher) error {
		d.OnPlatformConfigurationChanged = func() {
			f, s := d.MonospaceFont()
			u, us := d.UIFont()
			r.add(fmt.Sprintf("config mono %s %v ui %s %v", f, s, u, us))
		}
		return r.start(d)
	}
	h.c.Inject(fake.SettingsEvent{Settings: platform.Settings{MonospaceFontFamily: "Adwaita Mono", MonospaceFontSize: 11, UIFontFamily: "Cantarell", UIFontSize: 11}})
	done := h.run(t)
	waitFor(t, func() bool { return len(r.snapshot()) >= 1 })
	finish(t, h, done)
	if got, want := r.snapshot(), []string{"config mono Adwaita Mono 11 ui Cantarell 11"}; !reflect.DeepEqual(got, want) {
		t.Errorf("callbacks = %v, want %v", got, want)
	}
}

func TestPortalSettingsReachTheDispatcher(t *testing.T) {
	h, r := newDispatcherHarness(t)
	h.cfg.Start = func(d *ui.PlatformDispatcher) error {
		d.OnPlatformBrightnessChanged = func() { r.add("brightness " + d.PlatformBrightness().String()) }
		d.OnTextScaleFactorChanged = func() { r.add("scale " + strconv.FormatFloat(d.TextScaleFactor(), 'f', -1, 64)) }
		d.OnAccessibilityFeaturesChanged = func() { r.add("accessibility " + d.AccessibilityFeatures().String()) }
		return r.start(d)
	}
	bus := fakebus.New(":1.5")
	bus.Handle(dbus.PortalDest, dbus.PortalPath, "org.freedesktop.portal.Settings.ReadAll", func(fakebus.Call) ([]any, error) {
		return []any{map[string]map[string]godbus.Variant{}}, nil
	})
	// The platform posts to its UI thread; the fake one delivers an injected
	// event to the handler there, which is what a real platform does with
	// the settings it reads.
	dbus.WatchSettings(context.Background(), bus, func(string) string { return "" }, h.c.Post, func(s platform.Settings) {
		h.c.Inject(fake.SettingsEvent{Settings: s})
	})
	changed := func(key string, value any) {
		bus.Emit(&godbus.Signal{Path: dbus.PortalPath, Name: "org.freedesktop.portal.Settings.SettingChanged",
			Body: []any{"org.gnome.desktop.interface", key, godbus.MakeVariant(value)}})
	}
	bus.Emit(&godbus.Signal{Path: dbus.PortalPath, Name: "org.freedesktop.portal.Settings.SettingChanged",
		Body: []any{"org.freedesktop.appearance", "color-scheme", godbus.MakeVariant(uint32(1))}})
	changed("text-scaling-factor", 1.5)
	changed("enable-animations", false)
	done := h.run(t)
	waitFor(t, func() bool { return len(r.snapshot()) >= 3 })
	finish(t, h, done)
	want := []string{
		"brightness BrightnessDark",
		"scale 1.5",
		"accessibility AccessibilityFeatures[disableAnimations, reduceMotion]",
	}
	if got := r.snapshot(); !reflect.DeepEqual(got, want) {
		t.Errorf("callbacks = %v\nwant %v", got, want)
	}
}

func TestInputReachesTheDispatcherAtTheStartOfTheFrame(t *testing.T) {
	h, r := newDispatcherHarness(t)
	h.cfg.Start = func(d *ui.PlatformDispatcher) error {
		d.OnPointerDataPacket = func(p ui.PointerDataPacket) { r.add("pointer " + itoa(len(p.Data))) }
		d.OnKeyData = func(k ui.KeyData) bool { r.add("key " + string(rune(k.Logical))); return true }
		d.OnBeginFrame = func(time.Duration) { r.add("begin") }
		d.OnDrawFrame = func() {
			r.add("draw")
			_ = d.ImplicitView().Render(&ui.Scene{})
		}
		return r.start(d)
	}
	h.c.Inject(metrics(10, 10))
	h.c.Inject(fake.KeyEvent{Data: ui.KeyData{Type: ui.KeyEventTypeDown, Logical: 'q'}})
	h.c.Inject(fake.PointerEvent{Data: []ui.PointerData{{Change: ui.PointerChangeAdd}, {Change: ui.PointerChangeHover}}})
	h.c.Inject(fake.FrameEvent{View: 0})
	done := h.run(t)
	waitFor(t, func() bool { return len(r.snapshot()) == 4 })
	finish(t, h, done)
	want := []string{"key q", "pointer 2", "begin", "draw"}
	if got := r.snapshot(); !reflect.DeepEqual(got, want) {
		t.Errorf("callbacks = %v, want %v", got, want)
	}
	// Each event asked for a frame.
	n := 0
	for _, e := range h.c.Log() {
		if e.Kind == fake.EntryRequestFrame {
			n++
		}
	}
	if n < 2 {
		t.Errorf("RequestFrame entries = %d, want one for each event", n)
	}
}

func itoa(i int) string { return string(rune('0' + i)) }

func TestRequestAppExitEndsRun(t *testing.T) {
	h, r := newDispatcherHarness(t)
	done := h.run(t)
	waitFor(t, func() bool { return r.dispatcher() != nil })
	h.c.Post(func() { r.dispatcher().RequestAppExit() })
	if err := result(t, done); err != nil {
		t.Fatal(err)
	}
}

func TestRenderOfUnknownViewFails(t *testing.T) {
	h, r := newDispatcherHarness(t)
	var got error
	var mu sync.Mutex
	h.cfg.Start = func(d *ui.PlatformDispatcher) error {
		d.OnDrawFrame = func() {
			v := d.AddView(99, ui.ViewConfiguration{})
			mu.Lock()
			got = v.Render(&ui.Scene{})
			mu.Unlock()
			_ = d.ImplicitView().Render(nil)
		}
		return r.start(d)
	}
	h.c.Inject(metrics(10, 10))
	h.c.Inject(fake.FrameEvent{View: 0})
	done := h.run(t)
	waitFor(t, func() bool { mu.Lock(); defer mu.Unlock(); return got != nil })
	finish(t, h, done)
	mu.Lock()
	defer mu.Unlock()
	assertRoute(t, got, "ViewNotFound", "handler.RenderView.ViewNotFound")
}

func TestRendererFailureEndsRunWithTheError(t *testing.T) {
	h, r := newDispatcherHarness(t)
	h.r.err = errors.New(tagEGLContextLost, "lost")
	var renderErr error
	var mu sync.Mutex
	h.cfg.Start = func(d *ui.PlatformDispatcher) error {
		d.OnDrawFrame = func() {
			err := d.ImplicitView().Render(&ui.Scene{})
			mu.Lock()
			renderErr = err
			mu.Unlock()
		}
		return r.start(d)
	}
	h.c.Inject(metrics(10, 10))
	h.c.Inject(fake.FrameEvent{View: 0})
	done := h.run(t)
	err := result(t, done)
	assertRoute(t, err, "EGLContextLost", "EGLContextLost")
	mu.Lock()
	defer mu.Unlock()
	assertRoute(t, renderErr, "EGLContextLost", "EGLContextLost")
}

func TestClosingTheWindowRemovesTheView(t *testing.T) {
	h, r := newDispatcherHarness(t)
	h.c.Inject(fake.CloseRequestEvent{View: 0})
	done := h.run(t)
	waitFor(t, func() bool {
		gone := false
		onUI(t, h, func() { d := r.dispatcher(); gone = d != nil && d.ImplicitView() == nil })
		return gone
	})
	finish(t, h, done)
}

func TestPlatformConfigurationConversion(t *testing.T) {
	tests := []struct {
		name string
		in   platform.Settings
		want ui.PlatformConfiguration
	}{
		{"empty", platform.Settings{}, ui.PlatformConfiguration{}},
		{"accessibility", platform.Settings{DisableAnimations: true, HighContrast: true},
			ui.PlatformConfiguration{AccessibilityFeatures: ui.AccessibilityFeatures{DisableAnimations: true, ReduceMotion: true, HighContrast: true}}},
		{"a screen reader", platform.Settings{AccessibleNavigation: true},
			ui.PlatformConfiguration{AccessibilityFeatures: ui.AccessibilityFeatures{AccessibleNavigation: true}}},
		{"desktop fonts", platform.Settings{UIFontFamily: "Cantarell", UIFontSize: 11, MonospaceFontFamily: "Adwaita Mono", MonospaceFontSize: 12},
			ui.PlatformConfiguration{UIFontFamily: "Cantarell", UIFontSize: 11, MonospaceFontFamily: "Adwaita Mono", MonospaceFontSize: 12}},
		{"accent without flag is ignored", platform.Settings{AccentColor: ui.NewColor(0xFF000000)}, ui.PlatformConfiguration{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PlatformConfiguration(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
	accent := PlatformConfiguration(platform.Settings{HasAccentColor: true, AccentColor: ui.NewColor(0xFF010203)})
	if accent.AccentColor == nil || *accent.AccentColor != ui.NewColor(0xFF010203) {
		t.Errorf("accent = %v", accent.AccentColor)
	}
}

func TestScheduleFrameDoesNothingInTheSharedMemoryMode(t *testing.T) {
	p := fake.New()
	win, err := p.NewWindow(platform.DefaultWindowConfig())
	if err != nil {
		t.Fatal(err)
	}
	h := &handler{cfg: &Config{ShmFill: true}}
	h.first = h.addWindow(win)
	h.ScheduleFrame()
	for _, e := range p.Log() {
		if e.Kind == fake.EntryRequestFrame {
			t.Error("the shared-memory mode has no frames to ask for")
		}
	}
	h.cfg.ShmFill = false
	h.ScheduleFrame()
	n := 0
	for _, e := range p.Log() {
		if e.Kind == fake.EntryRequestFrame {
			n++
		}
	}
	if n != 1 {
		t.Errorf("RequestFrame entries = %d, want 1", n)
	}
}

// TestWindowInfoAndActionsReachTheFramework checks both ways of the title bar of
// the app: what the platform says about the window reaches the view, and an
// action of the view reaches the window.
func TestWindowInfoAndActionsReachTheFramework(t *testing.T) {
	h, r := newDispatcherHarness(t)
	want := ui.WindowInfo{Decoration: ui.DecorationApp, Capabilities: ui.WindowCapabilities{Maximize: true}, Title: "T", Maximized: true, Activated: true}
	h.c.Inject(fake.MetricsEvent{View: 0, Metrics: platform.ViewMetrics{PhysicalSize: ui.Size{Width: 640, Height: 400}, DevicePixelRatio: 1, Window: want}})
	done := h.run(t)
	waitFor(t, func() bool {
		if r.dispatcher() == nil {
			return false
		}
		var got ui.WindowInfo
		onUI(t, h, func() { got = r.dispatcher().ImplicitView().Window() })
		return got == want
	})
	actions := []ui.WindowAction{
		{Kind: ui.WindowActionMove},
		{Kind: ui.WindowActionResize, Edge: ui.ResizeEdgeBottomLeft},
		{Kind: ui.WindowActionShowMenu, Position: ui.Offset{Dx: 3, Dy: 4}},
		{Kind: ui.WindowActionClose},
	}
	onUI(t, h, func() {
		for _, a := range actions {
			r.dispatcher().ImplicitView().PerformWindowAction(a)
		}
	})
	var got []ui.WindowAction
	for _, e := range h.c.Log() {
		if e.Kind == fake.EntryWindowAction {
			got = append(got, e.Value.(ui.WindowAction))
		}
	}
	if !reflect.DeepEqual(got, actions) {
		t.Errorf("the window got %+v, want %+v", got, actions)
	}
	finish(t, h, done)
}

// TestDecorationsEnvChoosesTheWayToDecorate checks GLUTTER_DECORATIONS and that a
// choice of the app wins over it.
func TestDecorationsEnvChoosesTheWayToDecorate(t *testing.T) {
	tests := []struct {
		name   string
		env    string
		config platform.Decorations
		want   platform.Decorations
	}{
		{"unset", "", platform.DecorationsUnspecified, platform.DecorationsUnspecified},
		{"auto", "auto", platform.DecorationsUnspecified, platform.DecorationsUnspecified},
		{"app", "app", platform.DecorationsUnspecified, platform.DecorationsApp},
		{"libdecor", "libdecor", platform.DecorationsUnspecified, platform.DecorationsLibdecor},
		{"the app's choice wins", "app", platform.DecorationsAuto, platform.DecorationsAuto},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t, map[string]string{"DISPLAY": ":0", DecorationsEnv: tt.env})
			h.cfg.Window = platform.WindowConfig{Title: "T", Decorations: tt.config}
			done := h.run(t)
			waitFor(t, func() bool { _, _, n := h.hooked(); return n == 1 })
			finish(t, h, done)
			var got platform.Decorations
			for _, e := range h.c.Log() {
				if e.Kind == fake.EntryNewWindow {
					got = e.Value.(platform.WindowConfig).Decorations
				}
			}
			if got != tt.want {
				t.Errorf("the window was made with %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDisplayChangesReachTheDispatcherAndTheViews(t *testing.T) {
	h, r := newDispatcherHarness(t)
	h.c.SetDisplays([]platform.Display{
		{ID: 1, Bounds: ui.NewRectFromLTRB(0, 0, 100, 100), DevicePixelRatio: 1, RefreshRate: 60, Primary: true},
		{ID: 2, Bounds: ui.NewRectFromLTRB(100, 0, 300, 100), DevicePixelRatio: 2, RefreshRate: 60},
	})
	h.c.Inject(fake.MetricsEvent{View: 0, Metrics: platform.ViewMetrics{
		PhysicalSize: ui.Size{Width: 400, Height: 200}, DevicePixelRatio: 2, Display: foundation.Some(2),
	}})
	done := h.run(t)
	display := func() (ui.Display, int) {
		var d ui.Display
		var n int
		onUI(t, h, func() {
			if disp := r.dispatcher(); disp != nil && disp.ImplicitView() != nil {
				d, n = disp.ImplicitView().Display(), len(disp.Displays())
			}
		})
		return d, n
	}
	waitFor(t, func() bool { d, _ := display(); return d.ID == 2 })

	// The refresh rate of the window's display changes, and a third display appears.
	h.c.Inject(fake.DisplaysEvent{Displays: []platform.Display{
		{ID: 1, Bounds: ui.NewRectFromLTRB(0, 0, 100, 100), DevicePixelRatio: 1, RefreshRate: 60, Primary: true},
		{ID: 2, Bounds: ui.NewRectFromLTRB(100, 0, 300, 100), DevicePixelRatio: 2, RefreshRate: 144},
		{ID: 3, Bounds: ui.NewRectFromLTRB(300, 0, 400, 100), DevicePixelRatio: 1},
	}})
	waitFor(t, func() bool { d, n := display(); return n == 3 && d.ID == 2 && d.RefreshRate == 144 })
	d, _ := display()
	if d.Size.Width != 400 || d.DevicePixelRatio != 2 {
		t.Errorf("display of the view = %+v", d)
	}
	var third ui.Display
	onUI(t, h, func() { third = r.dispatcher().Displays()[2] })
	if third.RefreshRate != 60 {
		t.Errorf("a display without a refresh rate must get the default 60, got %v", third.RefreshRate)
	}
	finish(t, h, done)
}

func TestDropEventsReachTheDispatcherAndTheAnswerGoesBack(t *testing.T) {
	h, r := newDispatcherHarness(t)
	h.cfg.Start = func(d *ui.PlatformDispatcher) error {
		d.OnDrop = func(e ui.DropEvent) ui.DropActions {
			r.add("drop " + itoa(int(e.Phase)) + " " + e.Text)
			if e.Phase == ui.DropPhaseDrop {
				return ui.DropActionMove
			}
			return ui.DropActionCopy
		}
		d.OnDragEnd = func(res ui.DragResult) { r.add("end " + itoa(int(res.ID)) + " " + itoa(int(res.Action))) }
		return r.start(d)
	}
	h.c.Inject(fake.DropEvent{Event: ui.DropEvent{Phase: ui.DropPhaseEnter}})
	h.c.Inject(fake.DropEvent{Event: ui.DropEvent{Phase: ui.DropPhaseDrop, Text: "x"}})
	h.c.Inject(fake.DragEndEvent{Result: ui.DragResult{ID: 5, Action: ui.DropActionCopy}})
	done := h.run(t)
	waitFor(t, func() bool { return len(r.snapshot()) >= 3 })
	finish(t, h, done)
	want := []string{"drop 0 ", "drop 3 x", "end 5 1"}
	if got := r.snapshot(); !reflect.DeepEqual(got, want) {
		t.Errorf("callbacks = %v, want %v", got, want)
	}
	if got := h.c.DropAnswers(); !reflect.DeepEqual(got, []ui.DropActions{ui.DropActionCopy, ui.DropActionMove}) {
		t.Errorf("answers = %v", got)
	}
}

func TestStartDragReachesTheWindow(t *testing.T) {
	h, r := newDispatcherHarness(t)
	done := h.run(t)
	var err error
	waitFor(t, func() bool {
		ok := false
		onUI(t, h, func() { ok = r.dispatcher() != nil })
		return ok
	})
	onUI(t, h, func() { err = r.dispatcher().StartDrag(0, ui.DragRequest{ID: 3, Text: "hi"}) })
	if err != nil {
		t.Fatal(err)
	}
	onUI(t, h, func() { err = r.dispatcher().StartDrag(9, ui.DragRequest{ID: 4}) })
	assertRoute(t, err, "ViewNotFound", "ViewNotFound")
	finish(t, h, done)
	var drags []ui.DragRequest
	for _, e := range h.c.Log() {
		if e.Kind == fake.EntryStartDrag {
			drags = append(drags, e.Value.(ui.DragRequest))
		}
	}
	if len(drags) != 1 || drags[0].ID != 3 || drags[0].Text != "hi" {
		t.Errorf("drags = %+v", drags)
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagEGLContextLost = "EGLContextLost"
	tagStartFailed    = "StartFailed"
)
