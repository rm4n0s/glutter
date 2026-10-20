package fake

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

const testTimeout = 10 * time.Second

// recordingHandler is a platform.Handler that records each call as a string.
// It is only used from Run's goroutine, so tests read calls after Run returns.
type recordingHandler struct {
	calls      []string
	allowClose bool
}

func (h *recordingHandler) add(format string, args ...any) {
	h.calls = append(h.calls, fmt.Sprintf(format, args...))
}
func (h *recordingHandler) OnPointer(data []ui.PointerData) { h.add("OnPointer %d", len(data)) }
func (h *recordingHandler) OnKey(data ui.KeyData)           { h.add("OnKey %d", data.Physical) }
func (h *recordingHandler) OnTextInput(ui.TextInputEvent) {
	h.add("OnTextInput")
}
func (h *recordingHandler) OnFrame(view ui.ViewID, t time.Duration) {
	h.add("OnFrame %d %v", view, t)
}
func (h *recordingHandler) OnMetrics(view ui.ViewID, m platform.ViewMetrics) {
	h.add("OnMetrics %d %vx%v@%v", view, m.PhysicalSize.Width, m.PhysicalSize.Height, m.DevicePixelRatio)
}
func (h *recordingHandler) OnLifecycle(s ui.AppLifecycleState) { h.add("OnLifecycle %v", s) }
func (h *recordingHandler) OnDrop(view ui.ViewID, _ platform.DropEvent) {
	h.add("OnDrop %d", view)
}
func (h *recordingHandler) OnSettings(s platform.Settings) {
	h.add("OnSettings %v", s.PlatformBrightness)
}
func (h *recordingHandler) OnViewFocus(view ui.ViewID, focused bool) {
	h.add("OnViewFocus %d %v", view, focused)
}
func (h *recordingHandler) OnCloseRequest(view ui.ViewID) bool {
	h.add("OnCloseRequest %d", view)
	return h.allowClose
}

// runQueued runs p until everything queued so far has been delivered.
func runQueued(t *testing.T, p *Platform, h platform.Handler) {
	t.Helper()
	timeout, cancelTimeout := context.WithTimeout(context.Background(), testTimeout)
	defer cancelTimeout()
	ctx, stop := context.WithCancel(timeout)
	defer stop()
	p.Post(stop) // FIFO: runs after everything queued before it
	if err := p.Run(ctx, h); err != nil {
		t.Fatalf("Run returned %v", err)
	}
	if timeout.Err() != nil {
		t.Fatal("Run did not finish before the timeout")
	}
}

func TestPostFromManyGoroutinesKeepsOrderPerGoroutine(t *testing.T) {
	const goroutines, perGoroutine = 100, 50

	p := New()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	runCtx, stop := context.WithCancel(ctx)
	defer stop()

	got := make([][]int, goroutines) // only touched by Run's goroutine
	var remaining atomic.Int64
	remaining.Store(goroutines * perGoroutine)

	runErr := make(chan error, 1)
	go func() { runErr <- p.Run(runCtx, &recordingHandler{}) }()

	var wg sync.WaitGroup
	for g := range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range perGoroutine {
				p.Post(func() {
					got[g] = append(got[g], i)
					if remaining.Add(-1) == 0 {
						stop()
					}
				})
			}
		}()
	}
	wg.Wait()

	select {
	case err := <-runErr:
		if err != nil {
			t.Fatalf("Run returned %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("Run did not deliver every posted function; %d are missing", remaining.Load())
	}
	for g, seq := range got {
		if len(seq) != perGoroutine {
			t.Errorf("goroutine %d: got %d functions, want %d", g, len(seq), perGoroutine)
			continue
		}
		for i, v := range seq {
			if v != i {
				t.Errorf("goroutine %d: position %d ran post %d", g, i, v)
				break
			}
		}
	}
}

func TestRunStopsWhenContextIsCancelled(t *testing.T) {
	tests := []struct {
		name   string
		cancel func(cancel context.CancelFunc, p *Platform)
	}{
		{"cancelled before Run", func(cancel context.CancelFunc, _ *Platform) { cancel() }},
		{"cancelled while idle", func(cancel context.CancelFunc, _ *Platform) {
			time.AfterFunc(20*time.Millisecond, cancel)
		}},
		{"cancelled by a posted function", func(cancel context.CancelFunc, p *Platform) { p.Post(cancel) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			tt.cancel(cancel, p)

			done := make(chan error, 1)
			go func() { done <- p.Run(ctx, &recordingHandler{}) }()
			select {
			case err := <-done:
				if err != nil {
					t.Errorf("Run returned %v, want nil", err)
				}
			case <-time.After(testTimeout):
				t.Fatal("Run did not stop after the context was cancelled")
			}
		})
	}
}

func TestRunTwiceFailsWithPlatformAlreadyRunning(t *testing.T) {
	p := New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	started := make(chan struct{})
	p.Post(func() { close(started) })
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx, &recordingHandler{}) }()
	<-started

	err := p.Run(ctx, &recordingHandler{})
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T", err)
	}
	if !appErr.HasRoute("Platform.Run.PlatformAlreadyRunning") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}

	cancel()
	if err := <-done; err != nil {
		t.Errorf("first Run returned %v, want nil", err)
	}
	// After Run returned, the platform can run again.
	ctx2, cancel2 := context.WithCancel(context.Background())
	cancel2()
	if err := p.Run(ctx2, &recordingHandler{}); err != nil {
		t.Errorf("Run after the first one finished returned %v, want nil", err)
	}
}

func TestInjectDeliversEventsInOrder(t *testing.T) {
	tests := []struct {
		name  string
		event Event
		want  string
	}{
		{"pointer", PointerEvent{Data: []ui.PointerData{{}, {}}}, "OnPointer 2"},
		{"key", KeyEvent{Data: ui.KeyData{Physical: 7}}, "OnKey 7"},
		{"text input", TextInputEvent{}, "OnTextInput"},
		{"frame", FrameEvent{View: 1, FrameTime: 16 * time.Millisecond}, "OnFrame 1 16ms"},
		{"metrics", MetricsEvent{View: 2, Metrics: platform.ViewMetrics{PhysicalSize: ui.Size{Width: 640, Height: 400}, DevicePixelRatio: 1.5}}, "OnMetrics 2 640x400@1.5"},
		{"lifecycle", LifecycleEvent{State: ui.AppLifecycleStateResumed}, "OnLifecycle AppLifecycleStateResumed"},
		{"drop", DropEvent{View: 3}, "OnDrop 3"},
		{"settings", SettingsEvent{Settings: platform.Settings{PlatformBrightness: ui.BrightnessDark}}, "OnSettings BrightnessDark"},
		{"close request", CloseRequestEvent{View: 4}, "OnCloseRequest 4"},
		{"view focus", ViewFocusEvent{View: 5, Focused: true}, "OnViewFocus 5 true"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			h := &recordingHandler{}
			p.Inject(tt.event) // queued before Run starts
			runQueued(t, p, h)
			if want := []string{tt.want}; !reflect.DeepEqual(h.calls, want) {
				t.Errorf("handler calls = %q, want %q", h.calls, want)
			}
			log := p.Log()
			if len(log) == 0 || log[0].Kind != EntryEvent || !reflect.DeepEqual(log[0].Value, tt.event) {
				t.Errorf("log = %v, want it to start with the injected event", log)
			}
		})
	}

	t.Run("posts and events interleave in queue order", func(t *testing.T) {
		p := New()
		h := &recordingHandler{}
		p.Post(func() { h.add("post 1") })
		p.Inject(KeyEvent{Data: ui.KeyData{Physical: 1}})
		p.Post(func() { h.add("post 2") })
		p.Inject(KeyEvent{Data: ui.KeyData{Physical: 2}})
		runQueued(t, p, h)
		want := []string{"post 1", "OnKey 1", "post 2", "OnKey 2"}
		if !reflect.DeepEqual(h.calls, want) {
			t.Errorf("order = %q, want %q", h.calls, want)
		}
	})
}

func TestSettingsEventUpdatesSettings(t *testing.T) {
	p := New()
	want := platform.Settings{PlatformBrightness: ui.BrightnessDark, TextScaleFactor: 1.25}
	p.Inject(SettingsEvent{Settings: want})
	runQueued(t, p, &recordingHandler{})
	if got := p.Settings(); !reflect.DeepEqual(got, want) {
		t.Errorf("Settings() = %+v, want %+v", got, want)
	}
}

func TestCloseRequest(t *testing.T) {
	tests := []struct {
		name       string
		allowClose bool
		wantClosed bool
	}{
		{"handler allows closing", true, true},
		{"handler keeps the window", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			w, err := p.NewWindow(platform.DefaultWindowConfig())
			if err != nil {
				t.Fatal(err)
			}
			p.Inject(CloseRequestEvent{View: w.ID()})
			runQueued(t, p, &recordingHandler{allowClose: tt.allowClose})
			if got := p.Windows()[0].Closed(); got != tt.wantClosed {
				t.Errorf("Closed() = %v, want %v", got, tt.wantClosed)
			}
			var decision any = "missing"
			for _, e := range p.Log() {
				if e.Kind == EntryCloseDecision {
					decision = e.Value
				}
			}
			if decision != tt.wantClosed {
				t.Errorf("close decision in the log = %v, want %v", decision, tt.wantClosed)
			}
		})
	}
}

func TestNewWindowRecordsConfigAndAssignsViewIDs(t *testing.T) {
	p := New()
	cfgs := []platform.WindowConfig{
		{Title: "main", AppID: "dev.glutter.test", Size: ui.Size{Width: 640, Height: 400}, Resizable: foundation.Some(true), Decorated: foundation.Some(true)},
		{Kind: platform.WindowKindDialog, HasParent: true, Parent: 0, Modal: true, Title: "dialog"},
	}
	for i, cfg := range cfgs {
		w, err := p.NewWindow(cfg)
		if err != nil {
			t.Fatalf("NewWindow(%d): %v", i, err)
		}
		if w.ID() != ui.ViewID(i) {
			t.Errorf("window %d: ID() = %d, want %d", i, w.ID(), i)
		}
		if surface := w.NativeSurface(); surface.Kind != gpu.SurfaceKindFake || surface.Window != uintptr(w.ID()) {
			t.Errorf("window %d: NativeSurface() = %#v, want a fake surface for view %d", i, surface, w.ID())
		}
	}
	windows := p.Windows()
	if len(windows) != len(cfgs) {
		t.Fatalf("Windows() has %d windows, want %d", len(windows), len(cfgs))
	}
	for i, w := range windows {
		if !reflect.DeepEqual(w.Config(), cfgs[i]) {
			t.Errorf("window %d: Config() = %+v, want %+v", i, w.Config(), cfgs[i])
		}
		if w.Title() != cfgs[i].Title {
			t.Errorf("window %d: Title() = %q, want %q", i, w.Title(), cfgs[i].Title)
		}
	}
	log := p.Log()
	if len(log) != 2 || log[0].Kind != EntryNewWindow || !reflect.DeepEqual(log[1].Value, cfgs[1]) {
		t.Errorf("log = %v, want one NewWindow entry per window", log)
	}
}

func TestWindowRecordsCalls(t *testing.T) {
	p := New()
	w, err := p.NewWindow(platform.DefaultWindowConfig())
	if err != nil {
		t.Fatal(err)
	}
	fw := p.Windows()[0]

	w.SetTitle("hello")
	w.SetSize(ui.Size{Width: 800, Height: 600})
	w.SetState(platform.WindowStateMaximized)
	w.SetSizeConstraints(ui.Size{Width: 100, Height: 50}, ui.Size{Width: 900, Height: 700})
	w.SetCursor(platform.MouseCursor(3))
	w.SetTextInput(ui.TextInputState{Enabled: true})
	w.RequestFrame()
	if err := w.StartDrag(platform.DragData{}); err != nil {
		t.Errorf("StartDrag: %v", err)
	}
	w.Close()
	w.Close() // the second Close records nothing

	if fw.Title() != "hello" || fw.Size() != (ui.Size{Width: 800, Height: 600}) ||
		fw.State() != platform.WindowStateMaximized || fw.Cursor() != 3 ||
		!fw.TextInput().Enabled || !fw.Closed() ||
		fw.SizeConstraints() != (SizeConstraints{Min: ui.Size{Width: 100, Height: 50}, Max: ui.Size{Width: 900, Height: 700}}) {
		t.Errorf("window state was not recorded: title=%q size=%v state=%v cursor=%v input=%v closed=%v",
			fw.Title(), fw.Size(), fw.State(), fw.Cursor(), fw.TextInput(), fw.Closed())
	}
	var kinds []EntryKind
	for _, e := range p.Log() {
		kinds = append(kinds, e.Kind)
	}
	want := []EntryKind{EntryNewWindow, EntrySetTitle, EntrySetSize, EntrySetState, EntrySetSizeConstraints, EntrySetCursor, EntrySetTextInput, EntryRequestFrame, EntryStartDrag, EntryClose}
	if !reflect.DeepEqual(kinds, want) {
		t.Errorf("log kinds = %v, want %v", kinds, want)
	}
}

func TestDisplaysSettingsAndServices(t *testing.T) {
	p := New()
	if got := p.Displays(); len(got) != 0 {
		t.Errorf("Displays() = %v, want none", got)
	}
	displays := []platform.Display{{ID: 1, Name: "HEADLESS-1", DevicePixelRatio: 1.5, RefreshRate: 60, Primary: true}}
	p.SetDisplays(displays)
	if got := p.Displays(); !reflect.DeepEqual(got, displays) {
		t.Errorf("Displays() = %v, want %v", got, displays)
	}
	if p.Services() != nil {
		t.Errorf("Services() = %v, want nil", p.Services())
	}
	if got := p.Settings(); !reflect.DeepEqual(got, platform.Settings{}) {
		t.Errorf("Settings() = %+v, want the zero value", got)
	}
}
