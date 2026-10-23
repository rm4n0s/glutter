package android

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

func TestRunStartsWithTheStateTheJavaSideHadSent(t *testing.T) {
	h := newHarness(t)
	// The events come before Run: the activity is quicker than main.
	h.b.surfaceCreated(0x1000)
	h.b.surfaceChanged(1080, 2400, 2.625, 90, 1080, 2400)
	h.b.lifecycleChanged(lifecycleResumed)
	h.b.focusChanged(true)
	h.run()
	want := []string{"lifecycle " + ui.AppLifecycleStateResumed.String(), "metrics 1080x2400@2.625", "focus true"}
	waitFor(t, "the start-up reports", func() bool { return len(h.h.log()) >= len(want) })
	for i, w := range want {
		if got := h.h.log()[i]; got != w {
			t.Errorf("call %d = %q, want %q", i, got, w)
		}
	}
	// The surface is blank, so a frame is asked for, and drawn when the choreographer says.
	waitFor(t, "the first vsync request", func() bool { return h.host.vsyncRequests() == 1 })
	h.host.fire <- 16 * time.Millisecond
	waitFor(t, "the first frame", func() bool { return h.h.count("frame") == 1 })
	if got := h.win.NativeSurface(); got != (gpu.SurfaceHandle{Kind: gpu.SurfaceKindAndroid, Window: 0x1000}) {
		t.Errorf("NativeSurface = %+v", got)
	}
}

func TestFramesAreMergedAndNeedASurface(t *testing.T) {
	h := newHarness(t)
	h.run()
	h.request()
	if n := h.host.vsyncRequests(); n != 0 {
		t.Fatalf("a frame was asked for without a surface (%d requests)", n)
	}
	h.b.surfaceCreated(0x2000)
	h.b.surfaceChanged(100, 200, 1, 60, 100, 200)
	waitFor(t, "the vsync request", func() bool { return h.host.vsyncRequests() == 1 })
	// Requests during the frame are one more frame.
	h.h.onFrame = func(time.Duration) {
		h.win.RequestFrame() // on the UI thread
		h.win.RequestFrame()
	}
	h.host.fire <- time.Millisecond
	waitFor(t, "the second vsync request", func() bool { return h.host.vsyncRequests() == 2 })
	h.h.onFrame = nil
	h.host.fire <- 2 * time.Millisecond
	h.sync(func() {})
	if h.h.count("frame") != 2 {
		t.Errorf("frames = %d, want 2: %v", h.h.count("frame"), h.h.log())
	}
	h.sync(func() {})
	if n := h.host.vsyncRequests(); n != 2 {
		t.Errorf("vsync requests = %d, want 2", n)
	}
}

func TestNoFramesWhileTheAppIsStopped(t *testing.T) {
	h := newHarness(t)
	h.b.surfaceCreated(0x3000)
	h.b.surfaceChanged(100, 200, 1, 60, 100, 200)
	h.b.lifecycleChanged(lifecycleStopped)
	h.run()
	h.request()
	if n := h.host.vsyncRequests(); n != 0 {
		t.Fatalf("a frame was asked for while stopped (%d)", n)
	}
	h.b.lifecycleChanged(lifecycleResumed)
	waitFor(t, "the frame after the app is visible again", func() bool { return h.host.vsyncRequests() == 1 })
	if h.h.count("lifecycle") < 2 {
		t.Errorf("lifecycle reports: %v", h.h.log())
	}
}

func TestSurfaceDestroyedWaitsForTheFrame(t *testing.T) {
	h := newHarness(t)
	h.b.surfaceCreated(0x4000)
	h.b.surfaceChanged(100, 200, 1, 60, 100, 200)
	h.run()
	inFrame, release := make(chan struct{}), make(chan struct{})
	h.h.onFrame = func(time.Duration) {
		close(inFrame)
		<-release
	}
	waitFor(t, "the vsync request", func() bool { return h.host.vsyncRequests() == 1 })
	h.host.fire <- time.Millisecond
	<-inFrame

	returned := make(chan struct{})
	go func() {
		h.b.surfaceDestroyed()
		close(returned)
	}()
	select {
	case <-returned:
		t.Fatal("surfaceDestroyed returned while the renderer drew into the surface")
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	select {
	case <-returned:
	case <-time.After(5 * time.Second):
		t.Fatal("surfaceDestroyed did not return after the frame")
	}
	if got := h.h.count("lost android:16384"); got != 1 {
		t.Errorf("OnSurfaceLost calls = %d: %v", got, h.h.log())
	}
	if rel := h.host.releasedWindows(); len(rel) != 1 || rel[0] != 0x4000 {
		t.Errorf("released windows %v", rel)
	}
	if got := h.win.NativeSurface(); got != (gpu.SurfaceHandle{}) {
		t.Errorf("NativeSurface after destroy = %+v", got)
	}
	// No frame is asked for until there is a surface again.
	h.request()
	if n := h.host.vsyncRequests(); n != 1 {
		t.Errorf("vsync requests = %d, want 1 (none after the destroy)", n)
	}
}

func TestSurfaceDestroyedGivesUpAfterTheWait(t *testing.T) {
	h := newHarness(t)
	h.b.wait = 50 * time.Millisecond
	h.b.surfaceCreated(0x5000)
	h.b.surfaceChanged(100, 200, 1, 60, 100, 200)
	h.run()
	inFrame, release := make(chan struct{}), make(chan struct{})
	h.h.onFrame = func(time.Duration) {
		close(inFrame)
		<-release
	}
	waitFor(t, "the vsync request", func() bool { return h.host.vsyncRequests() == 1 })
	h.host.fire <- time.Millisecond
	<-inFrame
	start := time.Now()
	h.b.surfaceDestroyed() // the frame never ends within the wait
	if d := time.Since(start); d < 40*time.Millisecond || d > 2*time.Second {
		t.Errorf("surfaceDestroyed took %v, want about the wait of 50ms", d)
	}
	close(release)
	waitFor(t, "the late release", func() bool { return len(h.host.releasedWindows()) == 1 })
}

func TestSurfaceDestroyedWithNoPlatformRunning(t *testing.T) {
	b := newBridge()
	b.surfaceCreated(0x6000)
	b.surfaceChanged(10, 20, 1, 60, 10, 20)
	start := time.Now()
	b.surfaceDestroyed()
	if time.Since(start) > time.Second {
		t.Error("surfaceDestroyed waited for a UI thread that is not there")
	}
	if s := b.snapshot(); s.window != 0 || s.ready() {
		t.Errorf("state after destroy: %+v", s)
	}
}

func TestANewSurfaceReplacesTheOldOne(t *testing.T) {
	h := newHarness(t)
	h.b.surfaceCreated(0x7000)
	h.b.surfaceChanged(100, 200, 2, 60, 100, 200)
	h.run()
	// A destroy that was missed: the new window comes first.
	h.b.surfaceCreated(0x7100)
	waitFor(t, "the old window to be released", func() bool { return len(h.host.releasedWindows()) == 1 })
	if rel := h.host.releasedWindows(); rel[0] != 0x7000 {
		t.Errorf("released %v", rel)
	}
	if h.h.count("lost android:28672") != 1 {
		t.Errorf("log %v", h.h.log())
	}
	if h.h.count("metrics 100x200@2.000") < 2 {
		t.Errorf("the new surface did not report its metrics: %v", h.h.log())
	}
}

func TestMetricsChangeReportsTheViewAndTheDisplays(t *testing.T) {
	h := newHarness(t)
	h.b.surfaceCreated(0x8000)
	h.b.surfaceChanged(1080, 2400, 2.625, 60, 1080, 2400)
	h.run()
	h.b.surfaceChanged(2400, 1080, 2.625, 60, 2400, 1080)
	waitFor(t, "the rotation", func() bool { return h.h.count("metrics 2400x1080") == 1 })
	waitFor(t, "the displays", func() bool { return h.h.count("displays") == 1 })
	// The same values again change nothing.
	h.b.surfaceChanged(2400, 1080, 2.625, 60, 2400, 1080)
	h.sync(func() {})
	if h.h.count("metrics 2400x1080") != 1 || h.h.count("displays") != 1 {
		t.Errorf("an unchanged surface reported again: %v", h.h.log())
	}
	var d []platform.Display
	h.sync(func() { d = h.p.Displays() })
	if len(d) != 1 || d[0].Bounds.Width() != 2400/2.625 || d[0].DevicePixelRatio != 2.625 {
		t.Errorf("displays %+v", d)
	}
}

func TestPostRunsInOrderOnTheUIThread(t *testing.T) {
	h := newHarness(t)
	h.run()
	var mu sync.Mutex
	var got []int
	done := make(chan struct{})
	for i := 0; i < 20; i++ {
		h.p.Post(func() {
			mu.Lock()
			got = append(got, i)
			mu.Unlock()
			if i == 19 {
				close(done)
			}
		})
	}
	<-done
	for i, v := range got {
		if v != i {
			t.Fatalf("order %v", got)
		}
	}
}

func TestRunTwiceAndTheEndOfRun(t *testing.T) {
	h := newHarness(t)
	h.run()
	err := h.p.Run(context.Background(), &recorder{})
	appErr, ok := errors.FromError(err)
	if !ok || !appErr.HasRoute("Platform.Run."+tagPlatformAlreadyRunning) {
		t.Errorf("second Run: %v", err)
	}
	h.stop()
	h.stop = nil
	// After Run ended, the platform can run again.
	h2 := &recorder{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- h.p.Run(ctx, h2) }()
	waitFor(t, "the second Run", func() bool { return h2.count("lifecycle") == 1 })
	cancel()
	if err := <-done; err != nil {
		t.Errorf("second Run returned %v", err)
	}
}

func TestEventsOfTheJavaSideAreInOrderWithPost(t *testing.T) {
	h := newHarness(t)
	h.run()
	var seen []string
	var mu sync.Mutex
	h.p.Post(func() { mu.Lock(); seen = append(seen, "post1"); mu.Unlock() })
	h.b.lifecycleChanged(lifecyclePaused)
	h.p.Post(func() {
		mu.Lock()
		seen = append(seen, "post2 after "+h.h.log()[len(h.h.log())-1])
		mu.Unlock()
	})
	h.sync(func() {})
	mu.Lock()
	defer mu.Unlock()
	if len(seen) != 2 || seen[0] != "post1" || !strings.HasPrefix(seen[1], "post2 after lifecycle") {
		t.Errorf("seen %v", seen)
	}
}

func TestNewWindowAndWindowRoutes(t *testing.T) {
	p := newPlatform(newBridge(), newFakeHost(), nil)
	tests := []struct {
		name string
		cfg  platform.WindowConfig
		ok   bool
	}{
		{name: "regular", cfg: platform.WindowConfig{Kind: platform.WindowKindRegular}, ok: true},
		{name: "a second window", cfg: platform.WindowConfig{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.NewWindow(tt.cfg)
			if tt.ok {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			appErr, ok := errors.FromError(err)
			if !ok || !appErr.HasRoute("Platform.NewWindow."+tagWindowUnsupported) {
				t.Errorf("got %v", err)
			}
		})
	}
	for name, cfg := range map[string]platform.WindowConfig{
		"a dialog":      {Kind: platform.WindowKindDialog},
		"borderless":    {Kind: platform.WindowKindBorderless},
		"with a parent": {HasParent: true},
	} {
		q := newPlatform(newBridge(), newFakeHost(), nil)
		_, err := q.NewWindow(cfg)
		if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("Platform.NewWindow."+tagWindowUnsupported) {
			t.Errorf("%s: %v", name, err)
		}
	}
	w := p.win
	if err := w.StartDrag(ui.DragRequest{}); !hasRoute(err, "window.StartDrag."+tagDragUnsupported) {
		t.Errorf("StartDrag: %v", err)
	}
	if err := w.KeepScreenOn(true); !hasRoute(err, "window.KeepScreenOn."+tagIdleInhibitUnavailable) {
		t.Errorf("KeepScreenOn: %v", err)
	}
	if w.SetHidden(true) {
		t.Error("SetHidden reported success")
	}
	if p.Services() != nil || len(p.Displays()) != 1 || p.Settings().TextScaleFactor != 1 {
		t.Error("services, displays or settings")
	}
}

func hasRoute(err error, route string) bool {
	appErr, ok := errors.FromError(err)
	return ok && appErr.HasRoute(route)
}

func TestClipboardFailsWithClipboardUnavailable(t *testing.T) {
	h := newHarness(t)
	h.run()
	clip := h.p.Clipboard()
	type result struct {
		name string
		err  error
	}
	results := make(chan result, 4)
	ctx := context.Background()
	clip.SetText(ctx, "x", func(err error) { results <- result{"SetText", err} })
	clip.GetText(ctx, func(_ string, ok bool, err error) {
		if ok {
			t.Error("GetText found text")
		}
		results <- result{"GetText", err}
	})
	clip.SetContent(ctx, ui.ClipboardKindUnspecified, ui.ClipboardContent{}, func(err error) { results <- result{"SetContent", err} })
	clip.GetContent(ctx, ui.ClipboardKindUnspecified, "text/plain", func(_ ui.ClipboardContent, err error) { results <- result{"GetContent", err} })
	for i := 0; i < 4; i++ {
		select {
		case r := <-results:
			if !hasRoute(r.err, "unavailableClipboard."+r.name+"."+tagClipboardUnavailable) {
				t.Errorf("%s: %v", r.name, r.err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("a clipboard call never answered")
		}
	}
	// A nil done function is fine.
	clip.SetText(ctx, "x", nil)
	h.sync(func() {})
}

func TestFrameRequests(t *testing.T) {
	var f frameRequests
	if f.shouldSchedule(true) {
		t.Error("scheduled with nothing wanted")
	}
	f.request()
	f.request()
	if f.shouldSchedule(false) {
		t.Error("scheduled while a frame cannot be drawn")
	}
	if !f.shouldSchedule(true) {
		t.Fatal("did not schedule")
	}
	f.scheduledNow()
	if f.shouldSchedule(true) {
		t.Error("scheduled twice")
	}
	if f.fired(false) {
		t.Error("drew while a frame cannot be drawn")
	}
	if !f.wanted {
		t.Error("the request was lost when the frame could not be drawn")
	}
	f.scheduledNow()
	if !f.fired(true) || f.wanted {
		t.Error("the frame was not drawn, or the request stayed")
	}
	f.request() // made during the frame
	if !f.shouldSchedule(true) {
		t.Error("the request made during the frame was not kept")
	}
	if f.scheduledNow(); f.fired(true) != true {
		t.Error("the next frame was not drawn")
	}
	if f.fired(true) {
		t.Error("a callback with nothing wanted drew")
	}
}

func TestSurfaceState(t *testing.T) {
	var s surfaceState
	if s.ready() || s.handle() != (gpu.SurfaceHandle{}) {
		t.Error("an empty state is ready")
	}
	if old := s.created(0x10); old != 0 {
		t.Errorf("old = %x", old)
	}
	if s.ready() {
		t.Error("ready without a size")
	}
	if !s.changed(100, 200, 2, 60, 100, 200) {
		t.Error("the first change did not differ")
	}
	if s.changed(100, 200, 2, 60, 100, 200) {
		t.Error("the same change differed")
	}
	if !s.changed(100, 200, 3, 60, 100, 200) {
		t.Error("a new density did not differ")
	}
	if !s.ready() || s.handle() != (gpu.SurfaceHandle{Kind: gpu.SurfaceKindAndroid, Window: 0x10}) {
		t.Errorf("state %+v", s)
	}
	if old := s.created(0x20); old != 0x10 {
		t.Errorf("created returned %x, want the replaced window", old)
	}
	if old := s.destroyed(); old != 0x20 || s.window != 0 || s.width != 100 {
		t.Errorf("destroyed returned %x; state %+v", old, s)
	}
	if old := s.destroyed(); old != 0 {
		t.Errorf("a second destroy returned %x", old)
	}
}

func TestMetricsAndDisplay(t *testing.T) {
	tests := []struct {
		name         string
		s            surfaceState
		wantDPR      float64
		wantW, wantH float64
		wantRefresh  float64
	}{
		{"a phone", surfaceState{width: 1080, height: 2400, density: 2.625, refresh: 120, displayWidth: 1080, displayHeight: 2400}, 2.625, 1080 / 2.625, 2400 / 2.625, 120},
		{"nothing known", surfaceState{width: 100, height: 50}, 1, 100, 50, 60},
		{"a window smaller than the display", surfaceState{width: 500, height: 500, density: 2, refresh: 60, displayWidth: 1000, displayHeight: 2000}, 2, 500, 1000, 60},
		{"a bad density", surfaceState{width: 10, height: 10, density: -1}, 1, 10, 10, 60},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.s.viewMetrics()
			if m.DevicePixelRatio != tt.wantDPR || m.PhysicalSize.Width != float64(tt.s.width) || m.PhysicalSize.Height != float64(tt.s.height) {
				t.Errorf("metrics %+v", m)
			}
			d := tt.s.display()
			if d.Bounds.Width() != tt.wantW || d.Bounds.Height() != tt.wantH || d.RefreshRate != tt.wantRefresh || !d.Primary {
				t.Errorf("display %+v", d)
			}
		})
	}
}

func TestLifecycleMapping(t *testing.T) {
	tests := []struct {
		in      lifecycleState
		want    ui.AppLifecycleState
		visible bool
	}{
		{lifecycleUnknown, ui.AppLifecycleStateResumed, true},
		{lifecycleResumed, ui.AppLifecycleStateResumed, true},
		{lifecyclePaused, ui.AppLifecycleStateInactive, true},
		{lifecycleStopped, ui.AppLifecycleStateHidden, false},
	}
	for _, tt := range tests {
		if tt.in.appState() != tt.want || tt.in.visible() != tt.visible {
			t.Errorf("%d: %v visible=%v, want %v visible=%v", tt.in, tt.in.appState(), tt.in.visible(), tt.want, tt.visible)
		}
	}
}

func TestMakeHost(t *testing.T) {
	fh := newFakeHost()
	h, err := makeHost(func() int { return 0 }, fh)
	if err != nil || h != host(fh) {
		t.Fatalf("makeHost = %v, %v", h, err)
	}
	_, err = makeHost(func() int { return -1 }, fh)
	appErr, ok := errors.FromError(err)
	if !ok || !appErr.HasRoute("makeHost."+tagLooperUnavailable) || appErr.Metadata["status"] != -1 {
		t.Errorf("got %v", err)
	}
}
