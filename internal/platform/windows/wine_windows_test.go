//go:build windows && glutter_wine

package windows

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// These tests open real windows under Wine on a headless X server (SPEC §25.13):
//
//	GOOS=windows go test -tags glutter_wine -exec scripts/wine-exec.sh -p 1 ./internal/platform/windows/...
//
// Each test is on one OS thread from New to Close, as the UI thread of an app is.

// frameRecord is a frame the handler was asked for: the size of the client area at that moment.
type frameRecord struct {
	view          ui.ViewID
	width, height int32
}

// rec is the handler of the tests: it records what the platform reports.
type rec struct {
	platform.NopHandler
	p       *Platform
	metrics []platform.ViewMetrics
	frames  []frameRecord
	focus   []bool
	kfocus  []bool
	closes  []ui.ViewID
	// closeAnswer is what OnCloseRequest says.
	closeAnswer bool
	lost        int
	displays    int
	onFrame     func(view ui.ViewID)
	// settings are the settings OnSettings got, and lifecycle the states OnLifecycle got.
	settings  []platform.Settings
	lifecycle []ui.AppLifecycleState
	// batches are the batches OnPointer got, and pointers the same data in one list.
	batches  [][]ui.PointerData
	pointers []ui.PointerData
	// drops are the events OnDrop got, answered by dropAnswer (none when it is nil); dragEnds are the
	// results OnDragEnd got; focused is OnViewFocus with its view (dnd_wine_test.go).
	drops      []ui.DropEvent
	dropAnswer func(ui.DropEvent) ui.DropActions
	dragEnds   []ui.DragResult
	focused    []focusChange
}

// focusChange is one call of OnViewFocus.
type focusChange struct {
	view    ui.ViewID
	focused bool
}

func (r *rec) OnMetrics(_ ui.ViewID, m platform.ViewMetrics) { r.metrics = append(r.metrics, m) }
func (r *rec) OnFrame(view ui.ViewID, _ time.Duration) {
	w := r.p.windowByView(view)
	r.frames = append(r.frames, frameRecord{view, w.clientW, w.clientH})
	if r.onFrame != nil {
		r.onFrame(view)
	}
}
func (r *rec) OnViewFocus(v ui.ViewID, f bool) {
	r.focus = append(r.focus, f)
	r.focused = append(r.focused, focusChange{v, f})
}
func (r *rec) OnKeyboardFocus(_ ui.ViewID, f bool) { r.kfocus = append(r.kfocus, f) }
func (r *rec) OnCloseRequest(v ui.ViewID) bool     { r.closes = append(r.closes, v); return r.closeAnswer }
func (r *rec) OnContentLost(ui.ViewID)             { r.lost++ }
func (r *rec) OnDisplaysChanged()                  { r.displays++ }
func (r *rec) OnSettings(s platform.Settings)      { r.settings = append(r.settings, s) }
func (r *rec) OnLifecycle(s ui.AppLifecycleState)  { r.lifecycle = append(r.lifecycle, s) }

// newTestPlatform locks the thread, starts the platform and closes it when the test ends.
func newTestPlatform(t *testing.T, opts ...Options) (*Platform, *rec) {
	t.Helper()
	runtime.LockOSThread()
	t.Cleanup(runtime.UnlockOSThread)
	var o Options
	if len(opts) > 0 {
		o = opts[0]
	}
	p, err := New(o)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := p.Close(); err != nil {
			t.Error(err)
		}
	})
	return p, &rec{p: p, closeAnswer: true}
}

// runFor runs the platform until the test ends it with the cancel it is given, or fails after
// the limit. drive is posted, so it runs after the first window was shown.
func runFor(t *testing.T, p *Platform, r *rec, limit time.Duration, drive func(cancel context.CancelFunc)) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	if drive != nil {
		p.Post(func() { drive(cancel) })
	}
	if err := p.Run(ctx, r); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("the test did not finish in %v", limit)
	}
}

// after runs fn on the UI thread d later, from the timer heap.
func after(p *Platform, d time.Duration, fn func()) { p.timers.add(time.Now().Add(d), fn) }

func newWin(t *testing.T, p *Platform, cfg platform.WindowConfig) *window {
	t.Helper()
	w, err := p.NewWindow(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return w.(*window)
}

func TestWindowOpensAndReportsItsMetrics(t *testing.T) {
	p, r := newTestPlatform(t)
	w := newWin(t, p, platform.WindowConfig{Title: "Opens", Size: ui.Size{Width: 320, Height: 240}})
	runFor(t, p, r, 20*time.Second, func(cancel context.CancelFunc) {
		defer cancel()
		scale := scaleOfDPI(win32.GetDpiForWindow(w.hwnd))
		if len(r.metrics) == 0 {
			t.Fatal("the handler got no metrics")
		}
		m := r.metrics[0]
		wantW, wantH := physical(320, scale), physical(240, scale)
		if int32(m.PhysicalSize.Width) != wantW || int32(m.PhysicalSize.Height) != wantH || m.DevicePixelRatio != scale {
			t.Errorf("first metrics = %vx%v at %v, want %dx%d at %v", m.PhysicalSize.Width, m.PhysicalSize.Height, m.DevicePixelRatio, wantW, wantH, scale)
		}
		if rc, err := win32.GetClientRect(w.hwnd); err != nil || rc.Right != wantW || rc.Bottom != wantH {
			t.Errorf("client rect = %+v, %v; want %dx%d", rc, err, wantW, wantH)
		}
		if len(r.frames) == 0 || r.frames[0].width != wantW || r.frames[0].height != wantH {
			t.Errorf("frames = %+v: the first frame is drawn at the size of the window before it is shown", r.frames)
		}
		if !win32.IsWindowVisible(w.hwnd) {
			t.Error("the window is not shown after its first frame")
		}
		if m.Window.Title != "Opens" || m.Window.Decoration != ui.DecorationServerSide {
			t.Errorf("window info = %+v", m.Window)
		}
		if _, ok := m.Display.Get(); !ok {
			t.Error("the metrics name no display")
		}
		if len(r.focus) == 0 {
			t.Error("the handler was not told whether the window has the focus")
		}
		if got := w.NativeSurface(); got.Window != w.hwnd || got.Display != p.instance || got.Kind != "win32" {
			t.Errorf("NativeSurface = %+v", got)
		}
	})
}

func TestSetSizeAndSetWindowPosReportAndDrawBeforeTheyReturn(t *testing.T) {
	p, r := newTestPlatform(t)
	w := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 320, Height: 240}})
	runFor(t, p, r, 20*time.Second, func(cancel context.CancelFunc) {
		defer cancel()
		scale := scaleOfDPI(win32.GetDpiForWindow(w.hwnd))
		check := func(name string, wantW, wantH int32, do func()) {
			nm, nf := len(r.metrics), len(r.frames)
			do()
			if len(r.metrics) <= nm {
				t.Errorf("%s: no new metrics before it returned", name)
				return
			}
			m := r.metrics[len(r.metrics)-1]
			if int32(m.PhysicalSize.Width) != wantW || int32(m.PhysicalSize.Height) != wantH {
				t.Errorf("%s: metrics %vx%v, want %dx%d", name, m.PhysicalSize.Width, m.PhysicalSize.Height, wantW, wantH)
			}
			if len(r.frames) <= nf {
				t.Errorf("%s: no frame before it returned", name)
				return
			}
			if f := r.frames[len(r.frames)-1]; f.width != wantW || f.height != wantH {
				t.Errorf("%s: the frame is %dx%d, want %dx%d", name, f.width, f.height, wantW, wantH)
			}
		}
		check("SetSize", physical(400, scale), physical(300, scale), func() { w.SetSize(ui.Size{Width: 400, Height: 300}) })
		in := w.insets()
		ow, oh := in.outer(physical(250, scale), physical(180, scale))
		check("SetWindowPos", physical(250, scale), physical(180, scale), func() {
			if err := win32.SetWindowPos(w.hwnd, 0, 0, 0, ow, oh, win32.SWPNoMove|win32.SWPNoZOrder|win32.SWPNoActivate); err != nil {
				t.Error(err)
			}
		})
		// The same size again reports nothing.
		nm := len(r.metrics)
		w.SetSize(ui.Size{Width: 250, Height: 180})
		if len(r.metrics) != nm {
			t.Error("a SetSize to the size it has reported again")
		}
	})
}

func TestConstraintsHold(t *testing.T) {
	p, r := newTestPlatform(t)
	w := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 320, Height: 240}, MinSize: ui.Size{Width: 200, Height: 150}, MaxSize: ui.Size{Width: 500, Height: 400}})
	runFor(t, p, r, 20*time.Second, func(cancel context.CancelFunc) {
		defer cancel()
		scale := scaleOfDPI(win32.GetDpiForWindow(w.hwnd))
		in := w.insets()
		try := func(name string, outerW, outerH int32, wantClientW, wantClientH int32) {
			if err := win32.SetWindowPos(w.hwnd, 0, 0, 0, outerW, outerH, win32.SWPNoMove|win32.SWPNoZOrder|win32.SWPNoActivate); err != nil {
				t.Fatal(err)
			}
			rc, err := win32.GetClientRect(w.hwnd)
			if err != nil {
				t.Fatal(err)
			}
			if rc.Right != wantClientW || rc.Bottom != wantClientH {
				t.Errorf("%s: client %dx%d, want %dx%d", name, rc.Right, rc.Bottom, wantClientW, wantClientH)
			}
		}
		try("too small", 50, 50, physical(200, scale), physical(150, scale))
		try("too large", 5000, 5000, physical(500, scale), physical(400, scale))
		w.SetSizeConstraints(ui.Size{Width: 300, Height: 100}, ui.Size{Width: 450, Height: 350})
		ow, oh := in.outer(180, 120)
		try("a new minimum (the height is above it)", ow, oh, physical(300, scale), 120)
		w.SetSizeConstraints(ui.Size{}, ui.Size{})
		ow, oh = in.outer(180, 120) // Windows keeps a smallest size of its own, larger than 10 pixels
		try("no limits", ow, oh, 180, 120)
	})
}

func TestStatesAreReported(t *testing.T) {
	p, r := newTestPlatform(t)
	w := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 320, Height: 240}})
	runFor(t, p, r, 30*time.Second, func(cancel context.CancelFunc) {
		last := func() ui.WindowInfo { return r.metrics[len(r.metrics)-1].Window }
		normalRect, _ := win32.GetWindowRect(w.hwnd)
		normalClient := [2]int32{w.clientW, w.clientH}

		w.SetState(platform.WindowStateMaximized)
		if !win32.IsZoomed(w.hwnd) || !last().Maximized {
			t.Errorf("maximized: zoomed %v, info %+v", win32.IsZoomed(w.hwnd), last())
		}
		if f := r.frames[len(r.frames)-1]; f.width != w.clientW || f.height != w.clientH || (f.width == normalClient[0] && f.height == normalClient[1]) {
			t.Errorf("the frame after maximizing is %dx%d, the client is %dx%d", f.width, f.height, w.clientW, w.clientH)
		}
		w.SetState(platform.WindowStateNormal)
		if win32.IsZoomed(w.hwnd) || last().Maximized || w.clientW != normalClient[0] || w.clientH != normalClient[1] {
			t.Errorf("restored: zoomed %v, info %+v, client %dx%d", win32.IsZoomed(w.hwnd), last(), w.clientW, w.clientH)
		}

		w.SetState(platform.WindowStateFullscreen)
		mi, err := win32.GetMonitorInfo(win32.MonitorFromWindow(w.hwnd, win32.MonitorDefaultToNearest))
		if err != nil {
			t.Fatal(err)
		}
		rc, _ := win32.GetWindowRect(w.hwnd)
		if rc != mi.Monitor || !last().Fullscreen {
			t.Errorf("full screen: window %+v, monitor %+v, info %+v", rc, mi.Monitor, last())
		}
		if style := uint32(win32.GetWindowLongPtr(w.hwnd, win32.GWLStyle)); style&win32.WSCaption != 0 || style&win32.WSPopup == 0 {
			t.Errorf("the style in full screen = %#x", style)
		}
		w.SetState(platform.WindowStateNormal)
		rc, _ = win32.GetWindowRect(w.hwnd)
		if rc != normalRect || last().Fullscreen {
			t.Errorf("after full screen: window %+v, want %+v, info %+v", rc, normalRect, last())
		}

		w.SetState(platform.WindowStateMinimized)
		if !win32.IsIconic(w.hwnd) || !last().Minimized {
			t.Errorf("minimized: iconic %v, info %+v", win32.IsIconic(w.hwnd), last())
		}
		if last().Fullscreen || r.metrics[len(r.metrics)-1].PhysicalSize.Width == 0 {
			t.Errorf("a minimized window reports %+v", r.metrics[len(r.metrics)-1])
		}
		// A minimized window draws no frame, and keeps the request until it is shown again.
		nf := len(r.frames)
		w.RequestFrame()
		after(p, 200*time.Millisecond, func() {
			if len(r.frames) != nf {
				t.Errorf("%d frames were drawn while the window was minimized", len(r.frames)-nf)
			}
			w.SetState(platform.WindowStateNormal)
			if win32.IsIconic(w.hwnd) || last().Minimized {
				t.Errorf("restored from minimized: iconic %v, info %+v", win32.IsIconic(w.hwnd), last())
			}
			after(p, 200*time.Millisecond, func() {
				defer cancel()
				if len(r.frames) <= nf {
					t.Error("the frame that was asked for while minimized was never drawn")
				}
			})
		})
	})
}

func TestStartInTheStateOfTheConfig(t *testing.T) {
	for _, tt := range []struct {
		name  string
		state platform.WindowState
		check func(t *testing.T, w *window)
	}{
		{"maximized", platform.WindowStateMaximized, func(t *testing.T, w *window) {
			if !win32.IsZoomed(w.hwnd) || !w.state.maximized {
				t.Errorf("zoomed %v, state %+v", win32.IsZoomed(w.hwnd), w.state)
			}
		}},
		{"full screen", platform.WindowStateFullscreen, func(t *testing.T, w *window) {
			mi, _ := win32.GetMonitorInfo(win32.MonitorFromWindow(w.hwnd, win32.MonitorDefaultToNearest))
			if rc, _ := win32.GetWindowRect(w.hwnd); rc != mi.Monitor || !w.state.fullscreen {
				t.Errorf("window %+v, monitor %+v, state %+v", rc, mi.Monitor, w.state)
			}
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p, r := newTestPlatform(t)
			w := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 320, Height: 240}, State: tt.state})
			runFor(t, p, r, 20*time.Second, func(cancel context.CancelFunc) {
				defer cancel()
				tt.check(t, w)
				// The first frame was drawn at the size the window had before it was shown (the normal one):
				// the window never shows at its normal size first, and the first frame after the show is the
				// size of its state.
				last := r.frames[len(r.frames)-1]
				if last.width != w.clientW || last.height != w.clientH {
					t.Errorf("the last frame is %dx%d, the client is %dx%d", last.width, last.height, w.clientW, w.clientH)
				}
			})
		})
	}
}

func TestSetHiddenRemovesTheWindowAndAsksForNoFrame(t *testing.T) {
	p, r := newTestPlatform(t)
	w := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 320, Height: 240}})
	runFor(t, p, r, 20*time.Second, func(cancel context.CancelFunc) {
		if !w.SetHidden(true) || win32.IsWindowVisible(w.hwnd) {
			t.Errorf("hidden: visible %v", win32.IsWindowVisible(w.hwnd))
		}
		nf := len(r.frames)
		w.RequestFrame()
		after(p, 200*time.Millisecond, func() {
			if len(r.frames) != nf {
				t.Errorf("%d frames while hidden", len(r.frames)-nf)
			}
			lost := r.lost
			w.SetHidden(false)
			if !win32.IsWindowVisible(w.hwnd) {
				t.Error("the window is not visible after it was shown")
			}
			if r.lost != lost+1 {
				t.Error("the handler was not told that the next frame must be whole")
			}
			after(p, 200*time.Millisecond, func() {
				defer cancel()
				if len(r.frames) <= nf {
					t.Error("the frame asked for while hidden never came after the show")
				}
			})
		})
	})
}

func TestCloseRequests(t *testing.T) {
	p, r := newTestPlatform(t)
	w := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 320, Height: 240}})
	r.closeAnswer = false
	start := time.Now()
	runFor(t, p, r, 20*time.Second, func(cancel context.CancelFunc) {
		win32.SendMessage(w.hwnd, win32.WMClose, 0, 0)
		if len(r.closes) != 1 || !win32.IsWindow(w.hwnd) || w.closed {
			t.Errorf("a refused close request: requests %v, window exists %v", r.closes, win32.IsWindow(w.hwnd))
		}
		r.closeAnswer = true
		p.Post(func() { win32.SendMessage(w.hwnd, win32.WMClose, 0, 0) }) // the last window: Run ends by itself
	})
	if time.Since(start) > 15*time.Second {
		t.Error("Run did not end when the last window was destroyed")
	}
	if win32.IsWindow(w.hwnd) || len(r.closes) != 2 {
		t.Errorf("window exists %v, requests %v", win32.IsWindow(w.hwnd), r.closes)
	}
	if len(p.windows) != 0 {
		t.Errorf("%d windows are left in the table", len(p.windows))
	}
}

func TestOnlyTheLastWindowEndsRun(t *testing.T) {
	p, r := newTestPlatform(t)
	a := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 200, Height: 150}})
	b := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 200, Height: 150}})
	runFor(t, p, r, 20*time.Second, func(cancel context.CancelFunc) {
		defer cancel()
		a.Close()
		if p.quit {
			t.Error("the first window ended Run")
		}
		if !win32.IsWindow(b.hwnd) || len(p.windows) != 1 {
			t.Error("the second window was destroyed with the first")
		}
		if a.id == b.id {
			t.Error("two windows have one view")
		}
	})
}

func TestAppDrawnTitleBar(t *testing.T) {
	p, r := newTestPlatform(t)
	w := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 320, Height: 240}, Decorations: platform.DecorationsApp})
	runFor(t, p, r, 20*time.Second, func(cancel context.CancelFunc) {
		defer cancel()
		scale := scaleOfDPI(win32.GetDpiForWindow(w.hwnd))
		wr, _ := win32.GetWindowRect(w.hwnd)
		cr, _ := win32.GetClientRect(w.hwnd)
		if cr.Right != wr.Right-wr.Left || cr.Bottom != wr.Bottom-wr.Top {
			t.Errorf("client %dx%d, window %dx%d: the client area must cover the caption", cr.Right, cr.Bottom, wr.Right-wr.Left, wr.Bottom-wr.Top)
		}
		if cr.Right != physical(320, scale) || cr.Bottom != physical(240, scale) {
			t.Errorf("client %dx%d, want the size that was asked for", cr.Right, cr.Bottom)
		}
		if r.metrics[0].Window.Decoration != ui.DecorationApp {
			t.Errorf("decoration = %v", r.metrics[0].Window.Decoration)
		}
		hit := func(x, y int32) uintptr {
			return win32.SendMessage(w.hwnd, win32.WMNCHitTest, 0, uintptr(uint32(uint16(x))|uint32(uint16(y))<<16))
		}
		tests := []struct {
			name string
			x, y int32
			want uintptr
		}{
			{"the middle", wr.Left + (wr.Right-wr.Left)/2, wr.Top + (wr.Bottom-wr.Top)/2, win32.HTClient},
			{"the top left corner", wr.Left + 1, wr.Top + 1, win32.HTTopLeft},
			{"the bottom right corner", wr.Right - 2, wr.Bottom - 2, win32.HTBottomRight},
			{"the left edge", wr.Left + 1, wr.Top + 100, win32.HTLeft},
			{"the top edge", wr.Left + 100, wr.Top + 1, win32.HTTop},
		}
		for _, tt := range tests {
			if got := hit(tt.x, tt.y); got != tt.want {
				t.Errorf("%s: hit test = %d, want %d", tt.name, got, tt.want)
			}
		}
	})
}

func TestPostFromManyGoroutinesRunsEveryFunctionInOrder(t *testing.T) {
	p, r := newTestPlatform(t)
	newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 100, Height: 100}})
	const goroutines, each = 100, 100
	next := make([]int, goroutines)
	ran := 0
	done := make(chan struct{})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	p.Post(func() {
		for g := 0; g < goroutines; g++ {
			go func() {
				for i := 0; i < each; i++ {
					p.Post(func() {
						if next[g] != i {
							t.Errorf("goroutine %d: function %d ran when %d was next", g, i, next[g])
						}
						next[g] = i + 1
						ran++
						if ran == goroutines*each {
							close(done)
						}
					})
				}
			}()
		}
	})
	go func() {
		select {
		case <-done:
		case <-ctx.Done():
		}
		cancel()
		p.wake()
	}()
	if err := p.Run(ctx, r); err != nil {
		t.Fatal(err)
	}
	if ran != goroutines*each {
		t.Errorf("%d of %d functions ran", ran, goroutines*each)
	}
}

func TestAnIdleAppSleeps(t *testing.T) {
	p, r := newTestPlatform(t)
	newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 100, Height: 100}})
	var turns0, turns1 uint64
	var cpu0, cpu1 time.Duration
	runFor(t, p, r, 20*time.Second, func(cancel context.CancelFunc) {
		// Let what the show posts settle, then look at a second in which the platform has no timer and
		// nothing is posted: the test wakes it from its own goroutine, once, at the end.
		after(p, 300*time.Millisecond, func() {
			turns0, cpu0 = p.turns.Load(), cpuTime(t)
			time.AfterFunc(time.Second, func() {
				p.Post(func() {
					turns1, cpu1 = p.turns.Load(), cpuTime(t)
					cancel()
				})
			})
		})
	})
	if d := turns1 - turns0; d > 2 { // the turn of the post that ends it, and the one that began the second
		t.Errorf("%d loop turns in a second of nothing", d)
	}
	if d := cpu1 - cpu0; d > 60*time.Millisecond {
		t.Errorf("the process used %v of CPU in a second of nothing", d)
	}
}

func TestOnlyOneRunAtATime(t *testing.T) {
	p, r := newTestPlatform(t)
	newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 100, Height: 100}})
	runFor(t, p, r, 20*time.Second, func(cancel context.CancelFunc) {
		defer cancel()
		err := p.Run(context.Background(), r)
		assertRoute(t, err, "PlatformAlreadyRunning", "Platform.Run.PlatformAlreadyRunning")
	})
}

func TestFrameRequestsAreMerged(t *testing.T) {
	p, r := newTestPlatform(t)
	w := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 100, Height: 100}})
	runFor(t, p, r, 20*time.Second, func(cancel context.CancelFunc) {
		n0 := len(r.frames)
		for i := 0; i < 5; i++ {
			w.RequestFrame()
		}
		after(p, 300*time.Millisecond, func() {
			if got := len(r.frames) - n0; got != 1 {
				t.Errorf("5 requests made %d frames, want 1", got)
			}
			// A request made during a frame asks for the next one.
			n1 := len(r.frames)
			requested := false
			r.onFrame = func(ui.ViewID) {
				if !requested {
					requested = true
					w.RequestFrame()
				}
			}
			w.RequestFrame()
			after(p, 300*time.Millisecond, func() {
				defer cancel()
				if got := len(r.frames) - n1; got != 2 {
					t.Errorf("a request during a frame made %d frames in all, want 2", got)
				}
			})
		})
	})
}

func TestADialogOfAModalParentDisablesIt(t *testing.T) {
	p, r := newTestPlatform(t)
	owner := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}})
	dialog := newWin(t, p, platform.WindowConfig{Kind: platform.WindowKindDialog, Parent: owner.id, HasParent: true, Modal: true, Size: ui.Size{Width: 200, Height: 100}})
	runFor(t, p, r, 20*time.Second, func(cancel context.CancelFunc) {
		defer cancel()
		disabled := func(w *window) bool {
			return uint32(win32.GetWindowLongPtr(w.hwnd, win32.GWLStyle))&win32.WSDisabled != 0
		}
		if !disabled(owner) {
			t.Error("a modal dialog did not disable its owner")
		}
		if dialog.owner != owner || uint32(win32.GetWindowLongPtr(dialog.hwnd, win32.GWLExStyle))&win32.WSExDlgModalFrame == 0 {
			t.Error("the dialog has no owner or no dialog frame")
		}
		dialog.Close()
		if disabled(owner) {
			t.Error("the owner is still disabled after the dialog closed")
		}
	})
}

func TestKeepScreenOnAndTheIcon(t *testing.T) {
	p, r := newTestPlatform(t)
	w := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 100, Height: 100}})
	runFor(t, p, r, 20*time.Second, func(cancel context.CancelFunc) {
		defer cancel()
		if err := w.KeepScreenOn(true); err != nil || p.keepOnCount() != 1 {
			t.Errorf("KeepScreenOn(true) = %v, count %d", err, p.keepOnCount())
		}
		if err := w.KeepScreenOn(false); err != nil || p.keepOnCount() != 0 {
			t.Errorf("KeepScreenOn(false) = %v, count %d", err, p.keepOnCount())
		}
		pix := make([]byte, 32*32*4)
		for i := 0; i < len(pix); i += 4 {
			pix[i], pix[i+3] = 255, 255
		}
		icon := ui.WindowIcon{Images: []ui.WindowIconImage{{Size: 32, RGBA: pix}, {Size: 16, RGBA: pix[:16*16*4]}}}
		if err := w.SetIcon(icon); err != nil {
			t.Fatal(err)
		}
		if big := win32.SendMessage(w.hwnd, win32.WMGetIcon, win32.IconBig, 0); big == 0 || big != w.icons[0] {
			t.Errorf("the big icon is %#x, want %#x", big, w.icons[0])
		}
		if small := win32.SendMessage(w.hwnd, win32.WMGetIcon, win32.IconSmall, 0); small == 0 || small != w.icons[1] {
			t.Errorf("the small icon is %#x, want %#x", small, w.icons[1])
		}
		if err := w.SetIcon(ui.WindowIcon{}); err != nil || w.icons != [2]uintptr{} {
			t.Errorf("removing the icon = %v, icons %v", err, w.icons)
		}
		w.SetTitle("A new title")
		if w.title != "A new title" {
			t.Error("the title is not kept")
		}
		w.SetContentType(ui.ContentTypeUnspecified)
		w.SetTextInput(ui.TextInputState{})
		w.SetCursor(platform.MouseCursorUnspecified)
	})
}

func TestDisplays(t *testing.T) {
	p, _ := newTestPlatform(t)
	ds := p.Displays()
	if len(ds) == 0 {
		t.Fatal("no displays")
	}
	primary := 0
	for _, d := range ds {
		if d.Primary {
			primary++
		}
		if d.ID == 0 || d.Bounds.Width() <= 0 || d.Bounds.Height() <= 0 || d.DevicePixelRatio < 1 || d.Name == "" {
			t.Errorf("display %+v", d)
		}
		if d.RefreshRate != 0 && d.RefreshRate <= 1 {
			t.Errorf("refresh rate %v: 0 and 1 mean none, and are reported as 0", d.RefreshRate)
		}
	}
	if primary != 1 {
		t.Errorf("%d primary displays", primary)
	}
}

func TestDevEventsScriptRunsAfterTheFirstFrame(t *testing.T) {
	p, r := newTestPlatform(t, Options{Getenv: func(k string) string {
		if k == DevEventsEnv {
			return "wait:150ms;wait:150ms"
		}
		return ""
	}})
	newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 100, Height: 100}})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var started time.Time
	r.onFrame = func(ui.ViewID) {
		if started.IsZero() {
			started = time.Now()
		}
	}
	go func() {
		<-p.devDone
		cancel()
		p.wake()
	}()
	if err := p.Run(ctx, r); err != nil {
		t.Fatal(err)
	}
	select {
	case <-p.devDone:
	default:
		t.Fatal("the script did not finish")
	}
	if d := time.Since(started); d < 280*time.Millisecond {
		t.Errorf("the script of two 150 ms waits finished %v after the first frame", d)
	}
}

func TestDevEventsScriptWithAnErrorFailsNew(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	_, err := New(Options{Getenv: func(k string) string {
		if k == DevEventsEnv {
			return "wait:soon"
		}
		return ""
	}})
	assertRoute(t, err, TagWindowsDevEventsInvalid, "New->ParseDevEvents.WindowsDevEventsInvalid")
}

func TestAPanicOfTheHandlerIsRaisedAgainByRun(t *testing.T) {
	p, r := newTestPlatform(t)
	w := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 100, Height: 100}})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	p.Post(func() {
		r.onFrame = func(ui.ViewID) { panic("the handler broke") }
		w.RequestFrame()
	})
	defer func() {
		if v := recover(); v != "the handler broke" {
			t.Errorf("recovered %v, want the panic of the handler", v)
		}
		if p.panicked != nil {
			t.Error("the record of the panic was kept")
		}
	}()
	_ = p.Run(ctx, r)
	t.Fatal("Run returned")
}
