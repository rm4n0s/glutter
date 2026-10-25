package engine

import (
	"context"
	"log/slog"
	"strconv"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/fake"
	"github.com/rm4n0s/glutter/internal/platform/mobile"
	"github.com/rm4n0s/glutter/ui"
)

// windowsEngine is a handler with the first window on the fake platform, driven by
// the test directly, as the platform would drive it, without an event loop.
type windowsEngine struct {
	h   *handler
	p   *fake.Platform
	r   *fakeRenderer
	d   *ui.PlatformDispatcher
	log *drawLog
}

// drawLog counts the frames the dispatcher ran.
type drawLog struct{ begin, draw int }

func newWindowsEngine(t *testing.T) *windowsEngine {
	t.Helper()
	p := fake.New()
	e := &windowsEngine{p: p, r: &fakeRenderer{}, log: &drawLog{}}
	e.h = &handler{
		cfg: &Config{Logger: quiet()}, log: quiet(), conn: &conn{Platform: p}, cancel: func() {},
		newRenderer: func(*Config, *slog.Logger) renderer { return e.r },
		appID:       "org.example.App", decorations: platform.DecorationsApp,
	}
	win, err := p.NewWindow(platform.WindowConfig{Title: "Main", AppID: "org.example.App"})
	if err != nil {
		t.Fatal(err)
	}
	e.h.first = e.h.addWindow(win)
	e.d = ui.NewPlatformDispatcher(e.h)
	e.h.disp = e.d
	e.d.AddView(win.ID(), ui.ViewConfiguration{})
	e.d.OnBeginFrame = func(time.Duration) { e.log.begin++ }
	return e
}

// size gives the window of the view a size, as the platform does at its first
// configure.
func (e *windowsEngine) size(view ui.ViewID, w, h float64) {
	e.h.OnMetrics(view, platform.ViewMetrics{PhysicalSize: ui.Size{Width: w, Height: h}, DevicePixelRatio: 1})
}

// renderAll makes OnDrawFrame render a scene into every view, as the framework does.
func (e *windowsEngine) renderAll(scenes map[ui.ViewID]*ui.Scene) {
	e.d.OnDrawFrame = func() {
		e.log.draw++
		for _, v := range e.d.Views() {
			if err := v.Render(scenes[v.ViewID()]); err != nil {
				panic(err)
			}
		}
	}
}

// handles are the surface handles that the renderer drew into, in order.
func (e *windowsEngine) drawn() []gpu.SurfaceHandle {
	calls, _ := e.r.snapshot()
	out := make([]gpu.SurfaceHandle, len(calls))
	for i, c := range calls {
		out[i] = c.handle
	}
	return out
}

func handleOf(id ui.ViewID) gpu.SurfaceHandle {
	return gpu.SurfaceHandle{Kind: gpu.SurfaceKindFake, Window: uintptr(id)}
}

func TestTheFrameworkCreatesWindows(t *testing.T) {
	tests := []struct {
		name string
		spec ui.WindowSpec
		want func(c platform.WindowConfig) bool
	}{
		{"regular", ui.WindowSpec{Title: "Second", Size: ui.Size{Width: 400, Height: 300}, MinSize: ui.Size{Width: 200, Height: 100}},
			func(c platform.WindowConfig) bool {
				return c.Kind == platform.WindowKindUnspecified && !c.HasParent && !c.Modal && c.Title == "Second" &&
					c.Size == (ui.Size{Width: 400, Height: 300}) && c.MinSize == (ui.Size{Width: 200, Height: 100}) &&
					c.AppID == "org.example.App" && c.Decorations == platform.DecorationsApp
			}},
		{"modal dialog", ui.WindowSpec{Kind: ui.WindowKindDialog, Parent: ui.ImplicitViewID, HasParent: true},
			func(c platform.WindowConfig) bool {
				return c.Kind == platform.WindowKindDialog && c.HasParent && c.Parent == ui.ImplicitViewID && c.Modal
			}},
		{"modeless dialog", ui.WindowSpec{Kind: ui.WindowKindDialog},
			func(c platform.WindowConfig) bool {
				return c.Kind == platform.WindowKindDialog && !c.HasParent && !c.Modal
			}},
		{"a regular window with a parent is not modal", ui.WindowSpec{Parent: ui.ImplicitViewID, HasParent: true},
			func(c platform.WindowConfig) bool { return c.HasParent && !c.Modal }},
		{"drawn by the app", ui.WindowSpec{AppDecorations: true},
			func(c platform.WindowConfig) bool { return c.Decorations == platform.DecorationsApp }},
		{"not resizable", ui.WindowSpec{NotResizable: true},
			func(c platform.WindowConfig) bool { return !c.IsResizable() }},
		{"with an icon and a content type", ui.WindowSpec{Icon: ui.WindowIcon{Name: "game"}, ContentType: ui.ContentTypeGame},
			func(c platform.WindowConfig) bool {
				return c.Icon.Name == "game" && c.ContentType == ui.ContentTypeGame
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newWindowsEngine(t)
			v, err := e.d.CreateWindow(tt.spec)
			if err != nil {
				t.Fatal(err)
			}
			if v.ViewID() == ui.ImplicitViewID || e.d.View(v.ViewID()) != v {
				t.Fatalf("view = %v", v.ViewID())
			}
			ws := e.p.Windows()
			if len(ws) != 2 {
				t.Fatalf("%d platform windows, want 2", len(ws))
			}
			if !tt.want(ws[1].Config()) {
				t.Errorf("config = %+v", ws[1].Config())
			}
		})
	}
}

func TestWindowPlatformFailureComesBack(t *testing.T) {
	e := newWindowsEngine(t)
	e.h.conn = &failingWindow{conn: &conn{Platform: e.p}, err: errors.New(tagWaylandProtocolMissing, "dialog protocol missing")}
	_, err := e.d.CreateWindow(ui.WindowSpec{})
	assertRoute(t, err, "WaylandProtocolMissing", "WaylandProtocolMissing")
	if len(e.d.Views()) != 1 {
		t.Error("a view was made for a window that does not exist")
	}
}

func TestTheFrameworkChangesAndDestroysAWindow(t *testing.T) {
	e := newWindowsEngine(t)
	v, err := e.d.CreateWindow(ui.WindowSpec{Title: "Second", Size: ui.Size{Width: 400, Height: 300}})
	if err != nil {
		t.Fatal(err)
	}
	id := v.ViewID()
	win := e.p.Windows()[1]

	e.d.SetWindowTitle(id, "Renamed")
	e.d.SetWindowSize(id, ui.Size{Width: 500, Height: 350})
	e.d.SetWindowConstraints(id, ui.Size{Width: 300, Height: 200}, ui.Size{Width: 900, Height: 700})
	if win.Title() != "Renamed" || win.Size() != (ui.Size{Width: 500, Height: 350}) ||
		win.SizeConstraints() != (fake.SizeConstraints{Min: ui.Size{Width: 300, Height: 200}, Max: ui.Size{Width: 900, Height: 700}}) {
		t.Errorf("title %q size %v constraints %+v", win.Title(), win.Size(), win.SizeConstraints())
	}
	// The other window is not touched.
	if first := e.p.Windows()[0]; first.Title() != "Main" {
		t.Errorf("the first window's title = %q", first.Title())
	}

	// Draw into it so that it has a surface, then destroy it.
	e.size(id, 500, 350)
	e.renderAll(map[ui.ViewID]*ui.Scene{id: {}, ui.ImplicitViewID: {}})
	e.h.OnFrame(id, time.Millisecond)
	e.d.DestroyWindow(id)
	e.d.DestroyWindow(id) // twice is fine
	if !win.Closed() {
		t.Error("the platform window is open")
	}
	if len(e.d.Views()) != 1 || !v.Destroyed() {
		t.Errorf("views = %d, destroyed = %v", len(e.d.Views()), v.Destroyed())
	}
	if len(e.r.released) != 1 || e.r.released[0] != handleOf(id) {
		t.Errorf("released surfaces = %v, want the window's", e.r.released)
	}
	// Rendering into a destroyed view does nothing.
	n := len(e.drawn())
	if err := v.Render(&ui.Scene{}); err != nil || len(e.drawn()) != n {
		t.Errorf("render after destroy: %v, %d more draws", err, len(e.drawn())-n)
	}
	// Asking for a window that is gone does nothing.
	e.d.SetWindowTitle(id, "late")
	if win.Title() == "late" {
		t.Error("a destroyed window got a title")
	}
}

func TestEachWindowPresentsOneFramePerFrameRequest(t *testing.T) {
	e := newWindowsEngine(t)
	v, err := e.d.CreateWindow(ui.WindowSpec{Size: ui.Size{Width: 200, Height: 100}})
	if err != nil {
		t.Fatal(err)
	}
	second := v.ViewID()
	e.size(0, 800, 600)
	e.size(second, 200, 100)
	e.renderAll(map[ui.ViewID]*ui.Scene{0: {}, second: {}})

	// The framework asks for a frame: every window is asked for one.
	e.h.ScheduleFrame()
	requested := map[ui.ViewID]int{}
	for _, entry := range e.p.Log() {
		if entry.Kind == fake.EntryRequestFrame {
			requested[entry.View]++
		}
	}
	if requested[0] != 1 || requested[second] != 1 {
		t.Fatalf("frame requests = %v, want one for each window", requested)
	}

	// The frame of the second window runs the framework once and draws that window.
	e.h.OnFrame(second, time.Millisecond)
	if e.log.begin != 1 || e.log.draw != 1 {
		t.Errorf("frames run = %d begin, %d draw", e.log.begin, e.log.draw)
	}
	if got := e.drawn(); len(got) != 1 || got[0] != handleOf(second) {
		t.Fatalf("drawn after the second frame = %v, want only the second window", got)
	}
	// The frame of the first window presents what the framework rendered for it,
	// without running the framework again.
	e.h.OnFrame(0, 2*time.Millisecond)
	if e.log.begin != 1 || e.log.draw != 1 {
		t.Errorf("the first window's frame ran the framework again: %d begin, %d draw", e.log.begin, e.log.draw)
	}
	if got := e.drawn(); len(got) != 2 || got[1] != handleOf(0) {
		t.Errorf("drawn = %v, want the first window next", got)
	}
}

func TestAWindowThatAskedForNoFrameIsNotDrawn(t *testing.T) {
	e := newWindowsEngine(t)
	v, err := e.d.CreateWindow(ui.WindowSpec{Size: ui.Size{Width: 200, Height: 100}})
	if err != nil {
		t.Fatal(err)
	}
	second := v.ViewID()
	e.size(0, 800, 600)
	e.size(second, 200, 100)
	e.renderAll(map[ui.ViewID]*ui.Scene{0: {}, second: {}})

	// The platform's own frame of the second window (its first one, say).
	e.h.OnFrame(second, time.Millisecond)
	if got := e.drawn(); len(got) != 1 || got[0] != handleOf(second) {
		t.Errorf("drawn = %v, want only the window whose frame it was", got)
	}
	if e.h.windows[0].pending {
		t.Error("the first window waits to present a frame it did not ask for")
	}
}

func TestAWindowThePlatformAsksAFrameOfDrawsItsLastSceneAgain(t *testing.T) {
	e := newWindowsEngine(t)
	e.size(0, 800, 600)
	scene := &ui.Scene{}
	rendered := false
	e.d.OnDrawFrame = func() {
		if !rendered {
			rendered = true
			_ = e.d.ImplicitView().Render(scene)
		}
	}
	e.h.OnFrame(0, time.Millisecond)
	e.h.OnFrame(0, 2*time.Millisecond) // the framework renders nothing now
	calls, _ := e.r.snapshot()
	if len(calls) != 2 || calls[0].scene != scene || calls[1].scene != scene {
		t.Errorf("draw calls = %+v, want the same scene twice", calls)
	}
}

func TestAWindowWithNoSizeIsNotDrawn(t *testing.T) {
	e := newWindowsEngine(t)
	v, err := e.d.CreateWindow(ui.WindowSpec{})
	if err != nil {
		t.Fatal(err)
	}
	e.renderAll(map[ui.ViewID]*ui.Scene{0: {}, v.ViewID(): {}})
	e.h.OnFrame(v.ViewID(), time.Millisecond)
	if got := e.drawn(); len(got) != 0 {
		t.Errorf("drawn = %v before the first configure", got)
	}
}

func TestClosingAWindow(t *testing.T) {
	tests := []struct {
		name       string
		takeOver   bool // the framework's window owner takes the request
		exitCancel bool // the app cancels the exit
		wantClose  bool
		wantGone   bool
	}{
		{"nobody objects", false, false, true, true},
		{"the app cancels the exit", false, true, false, false},
		{"the window owner takes the request", true, false, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newWindowsEngine(t)
			v, err := e.d.CreateWindow(ui.WindowSpec{Size: ui.Size{Width: 200, Height: 100}})
			if err != nil {
				t.Fatal(err)
			}
			id := v.ViewID()
			e.size(id, 200, 100)
			e.renderAll(map[ui.ViewID]*ui.Scene{0: {}, id: {}})
			e.h.OnFrame(id, time.Millisecond) // gives the window a surface
			e.d.OnWindowCloseRequested = func(view ui.ViewID) bool { return tt.takeOver && view == id }
			e.d.OnExitRequested = func(ui.ViewID) ui.AppExitResponse {
				if tt.exitCancel {
					return ui.AppExitResponseCancel
				}
				return ui.AppExitResponseExit
			}
			if got := e.h.OnCloseRequest(id); got != tt.wantClose {
				t.Errorf("OnCloseRequest = %v, want %v", got, tt.wantClose)
			}
			if gone := e.d.View(id) == nil; gone != tt.wantGone {
				t.Errorf("view removed = %v, want %v", gone, tt.wantGone)
			}
			if released := len(e.r.released) == 1; released != tt.wantGone {
				t.Errorf("surface released = %v, want %v", released, tt.wantGone)
			}
			// The other window and the renderer stay.
			if e.d.ImplicitView() == nil || e.r.closed != 0 {
				t.Error("closing a window took the others with it")
			}
		})
	}
}

func TestClosingTheFirstWindowLeavesTheOthers(t *testing.T) {
	e := newWindowsEngine(t)
	v, err := e.d.CreateWindow(ui.WindowSpec{Size: ui.Size{Width: 200, Height: 100}})
	if err != nil {
		t.Fatal(err)
	}
	second := v.ViewID()
	e.size(0, 800, 600)
	e.size(second, 200, 100)
	e.renderAll(map[ui.ViewID]*ui.Scene{0: {}, second: {}})
	e.h.OnFrame(0, time.Millisecond)

	if !e.h.OnCloseRequest(0) {
		t.Fatal("the first window was not allowed to close")
	}
	// Input and the cursor now go to the window that is left.
	e.h.OnKey(ui.KeyData{})
	e.h.ActivateSystemCursor(0, "click")
	e.h.SetTextInput(ui.TextInputState{Enabled: true})
	if e.p.Windows()[1].TextInput().Enabled != true {
		t.Error("the input method of the window that is left was not told")
	}
	e.h.OnFrame(second, 2*time.Millisecond)
	if got := e.drawn(); got[len(got)-1] != handleOf(second) {
		t.Errorf("drawn = %v, want the second window last", got)
	}
}

func TestViewFocusAndInputFollowTheFocusedWindow(t *testing.T) {
	e := newWindowsEngine(t)
	v, err := e.d.CreateWindow(ui.WindowSpec{Size: ui.Size{Width: 200, Height: 100}})
	if err != nil {
		t.Fatal(err)
	}
	second := v.ViewID()
	e.size(0, 800, 600)
	e.size(second, 200, 100)
	e.renderAll(map[ui.ViewID]*ui.Scene{0: {}, second: {}})
	var events []ui.ViewFocusEvent
	e.d.OnViewFocusChange = func(ev ui.ViewFocusEvent) { events = append(events, ev) }
	var keys int
	e.d.OnKeyData = func(ui.KeyData) bool { keys++; return true }

	// The platform moves the focus from the first window to the second: the old
	// one is told first, and the events are handled at the start of the next frame.
	e.h.OnViewFocus(0, true)
	e.h.OnViewFocus(0, false)
	e.h.OnViewFocus(second, true)
	e.h.OnKey(ui.KeyData{Logical: 'a'})
	if len(events) != 0 || keys != 0 {
		t.Fatalf("the input was handled before the frame: %d events, %d keys", len(events), keys)
	}
	e.h.OnFrame(second, time.Millisecond)
	want := []ui.ViewFocusEvent{
		{ViewID: 0, State: ui.ViewFocusStateFocused},
		{ViewID: 0, State: ui.ViewFocusStateUnfocused},
		{ViewID: second, State: ui.ViewFocusStateFocused},
	}
	if len(events) != len(want) {
		t.Fatalf("focus events = %+v, want %+v", events, want)
	}
	for i := range want {
		if events[i] != want[i] {
			t.Errorf("event %d = %+v, want %+v", i, events[i], want[i])
		}
	}
	if keys != 1 {
		t.Errorf("%d keys, want 1", keys)
	}

	// The input method and the cursor go to the windows they belong to.
	e.h.SetTextInput(ui.TextInputState{Enabled: true})
	if !e.p.Windows()[1].TextInput().Enabled || e.p.Windows()[0].TextInput().Enabled {
		t.Error("the input method is not on the focused window")
	}
	e.h.OnPointer([]ui.PointerData{{ViewID: 0}})
	e.h.ActivateSystemCursor(0, "click")
	if e.p.Windows()[0].Cursor() != platform.MouseCursorClick || e.p.Windows()[1].Cursor() != platform.MouseCursorUnspecified {
		t.Errorf("cursors = %v %v, want the click on the window under the pointer", e.p.Windows()[0].Cursor(), e.p.Windows()[1].Cursor())
	}
}

func TestAViewFocusRequestActivatesTheWindow(t *testing.T) {
	e := newWindowsEngine(t)
	v, err := e.d.CreateWindow(ui.WindowSpec{})
	if err != nil {
		t.Fatal(err)
	}
	second := v.ViewID()
	e.h.OnViewFocus(0, true)
	e.d.RequestViewFocusChange(second, ui.ViewFocusStateFocused, ui.ViewFocusDirectionForward)
	e.d.RequestViewFocusChange(0, ui.ViewFocusStateFocused, ui.ViewFocusDirectionForward) // it has the focus already
	e.d.RequestViewFocusChange(second, ui.ViewFocusStateUnfocused, ui.ViewFocusDirectionUndefined)
	var activations []ui.ViewID
	for _, entry := range e.p.Log() {
		if a, ok := entry.Value.(ui.WindowAction); ok && entry.Kind == fake.EntryWindowAction && a.Kind == ui.WindowActionActivate {
			activations = append(activations, entry.View)
		}
	}
	if len(activations) != 1 || activations[0] != second {
		t.Errorf("activated windows = %v, want only the second", activations)
	}
}

func TestWindowActionsGoToTheirWindow(t *testing.T) {
	e := newWindowsEngine(t)
	v, err := e.d.CreateWindow(ui.WindowSpec{})
	if err != nil {
		t.Fatal(err)
	}
	v.PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionMaximize})
	e.d.ImplicitView().PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionClose})
	got := map[ui.ViewID]ui.WindowActionKind{}
	for _, entry := range e.p.Log() {
		if a, ok := entry.Value.(ui.WindowAction); ok && entry.Kind == fake.EntryWindowAction {
			got[entry.View] = a.Kind
		}
	}
	if got[v.ViewID()] != ui.WindowActionMaximize || got[0] != ui.WindowActionClose {
		t.Errorf("actions = %v", got)
	}
}

func TestInputForAWindowWithAPendingSceneRunsTheFramework(t *testing.T) {
	e := newWindowsEngine(t)
	v, err := e.d.CreateWindow(ui.WindowSpec{Size: ui.Size{Width: 200, Height: 100}})
	if err != nil {
		t.Fatal(err)
	}
	second := v.ViewID()
	e.size(0, 800, 600)
	e.size(second, 200, 100)
	e.renderAll(map[ui.ViewID]*ui.Scene{0: {}, second: {}})
	keys := 0
	e.d.OnKeyData = func(ui.KeyData) bool { keys++; return true }

	e.h.ScheduleFrame()
	e.h.OnFrame(second, time.Millisecond) // the first window now has a scene to present
	e.h.OnKey(ui.KeyData{Logical: 'a'})   // and a key to handle
	e.h.OnFrame(0, 2*time.Millisecond)
	if keys != 1 || e.log.begin != 2 {
		t.Errorf("keys %d, frames run %d: the key waited for another frame", keys, e.log.begin)
	}
	if got := e.drawn(); got[len(got)-1] != handleOf(0) {
		t.Errorf("drawn = %v, want the first window last", got)
	}
}

func TestAWindowDestroyedDuringItsFrameIsNotDrawn(t *testing.T) {
	e := newWindowsEngine(t)
	v, err := e.d.CreateWindow(ui.WindowSpec{Size: ui.Size{Width: 200, Height: 100}})
	if err != nil {
		t.Fatal(err)
	}
	second := v.ViewID()
	e.size(0, 800, 600)
	e.size(second, 200, 100)
	e.renderAll(map[ui.ViewID]*ui.Scene{0: {}, second: {}})
	e.h.OnFrame(second, time.Millisecond) // gives the window a surface
	before := len(e.drawn())

	// A tap in the window destroys it, as the close button of a dialog does.
	e.d.OnKeyData = func(ui.KeyData) bool { e.d.DestroyWindow(second); return true }
	e.h.OnKey(ui.KeyData{Logical: 'x'})
	e.h.OnFrame(second, 2*time.Millisecond)
	if got := e.drawn(); len(got) != before {
		t.Errorf("drawn = %v: the destroyed window was drawn", got[before:])
	}
	if !e.p.Windows()[1].Closed() {
		t.Error("the platform window is open")
	}
}

func TestWindowAppearanceGoesToTheWindow(t *testing.T) {
	e := newWindowsEngine(t)
	v, err := e.d.CreateWindow(ui.WindowSpec{})
	if err != nil {
		t.Fatal(err)
	}
	if err := v.SetIcon(ui.WindowIcon{Name: "applications-graphics"}); err != nil {
		t.Fatal(err)
	}
	v.SetContentType(ui.ContentTypePhoto)
	if err := v.KeepScreenOn(true); err != nil {
		t.Fatal(err)
	}
	w := e.p.Windows()[1]
	if w.Icon().Name != "applications-graphics" || w.ContentType() != ui.ContentTypePhoto || !w.ScreenKeptOn() {
		t.Errorf("window: icon %+v, content type %v, screen on %v", w.Icon(), w.ContentType(), w.ScreenKeptOn())
	}
	if e.p.Windows()[0].ScreenKeptOn() {
		t.Error("the first window was changed")
	}
	// A wrong icon never reaches the platform.
	err = v.SetIcon(ui.WindowIcon{Images: []ui.WindowIconImage{{Size: 2}}})
	assertRoute(t, err, "WindowIconInvalid", "WindowIconInvalid")
	if w.Icon().Name != "applications-graphics" {
		t.Errorf("the icon is %+v", w.Icon())
	}
	// A window with a wrong icon is not made.
	_, err = e.d.CreateWindow(ui.WindowSpec{Icon: ui.WindowIcon{Images: []ui.WindowIconImage{{Size: 2}}}})
	assertRoute(t, err, "WindowIconInvalid", "WindowIconInvalid")
	if len(e.p.Windows()) != 2 {
		t.Errorf("%d windows", len(e.p.Windows()))
	}
}

func TestKeepScreenOnForAWindowThatDoesNotExist(t *testing.T) {
	e := newWindowsEngine(t)
	err := e.h.KeepScreenOn(99, true)
	assertRoute(t, err, "IdleInhibitUnavailable", "IdleInhibitUnavailable")
	e.h.SetWindowContentType(99, ui.ContentTypeGame) // nothing to do, nothing to fail
	if err := e.h.SetWindowIcon(99, ui.WindowIcon{Name: "x"}); err != nil {
		t.Errorf("SetWindowIcon: %v", err)
	}
}

func TestPortalHandlesAndActivationTokens(t *testing.T) {
	e := newWindowsEngine(t)
	v, err := e.d.CreateWindow(ui.WindowSpec{})
	if err != nil {
		t.Fatal(err)
	}
	var handle, token string
	e.d.PortalParentHandle(v.ViewID(), func(h string) { handle = h })
	e.d.ActivationToken(v.ViewID(), func(tk string) { token = tk })
	// The fake platform answers on its thread: run what it posted.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- e.p.Run(ctx, platform.NopHandler{}) }()
	e.p.Post(cancel)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if want := fake.FakePortalHandlePrefix + strconv.Itoa(int(v.ViewID())); handle != want {
		t.Errorf("handle = %q, want %q", handle, want)
	}
	if token != fake.FakeActivationToken {
		t.Errorf("token = %q", token)
	}
	// A view that does not exist has no handle.
	handle = "x"
	e.h.PortalParentHandle(99, func(h string) { handle = h })
	if handle != "" {
		t.Errorf("handle of a missing view = %q", handle)
	}
}

// A window that closes with the mouse pointer inside makes the pointer leave it while its view
// still exists: the platform's own removed event comes after the view is gone, and the framework
// drops events for views that do not exist, so the mouse tracker would never hear of it.
func TestClosingAWindowUnderThePointerRemovesThePointerFirst(t *testing.T) {
	type got struct {
		view   ui.ViewID
		change ui.PointerChange
		synth  bool
	}
	tests := []struct {
		name    string
		buttons int
		inView  bool // whether the pointer is in the window that closes
		want    []got
	}{
		{"a hovering pointer", 0, true, []got{{1, ui.PointerChangeRemove, false}}},
		{"a pressed pointer is released first", 1, true, []got{{1, ui.PointerChangeUp, true}, {1, ui.PointerChangeRemove, false}}},
		{"a pointer in another window is left alone", 0, false, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newWindowsEngine(t)
			v, err := e.d.CreateWindow(ui.WindowSpec{})
			if err != nil {
				t.Fatal(err)
			}
			id := v.ViewID()
			e.size(id, 500, 350)
			e.size(ui.ImplicitViewID, 400, 300)
			e.renderAll(map[ui.ViewID]*ui.Scene{id: {}, ui.ImplicitViewID: {}})
			var delivered []got
			e.d.OnPointerDataPacket = func(p ui.PointerDataPacket) {
				for _, d := range p.Data {
					delivered = append(delivered, got{d.ViewID, d.Change, d.Synthesized})
				}
			}
			under := id
			if !tt.inView {
				under = ui.ImplicitViewID
			}
			e.h.OnPointer([]ui.PointerData{
				{ViewID: under, Change: ui.PointerChangeAdd, Kind: ui.PointerDeviceKindMouse, PhysicalX: 5, PhysicalY: 6},
				{ViewID: under, Change: ui.PointerChangeHover, Kind: ui.PointerDeviceKindMouse, PhysicalX: 7, PhysicalY: 8, Buttons: tt.buttons},
			})
			delivered = nil // the batch is handled by the next frame
			e.h.OnFrame(id, time.Millisecond)
			delivered = nil
			e.d.DestroyWindow(id)
			if len(delivered) != len(tt.want) {
				t.Fatalf("delivered %v, want %v", delivered, tt.want)
			}
			for i := range tt.want {
				if delivered[i] != tt.want[i] {
					t.Errorf("event %d = %+v, want %+v", i, delivered[i], tt.want[i])
				}
			}
			// The removal is not repeated when the platform reports it too.
			if tt.inView && e.h.mouse.inside {
				t.Error("the engine still has the pointer in the closed window")
			}
		})
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagWaylandProtocolMissing = "WaylandProtocolMissing"
)

// desktopConn is a connection whose platform has desktop services of its own, as Android does.
type desktopConn struct {
	*conn
	svc mobile.Services
}

func (c desktopConn) Desktop() mobile.Services { return c.svc }

// TestPlatformServiceHandsOutTheDesktopServices: the services come through the dispatcher only for
// a platform that has them, only under the name package desktop asks for, and never as a non-nil
// interface around nothing.
func TestPlatformServiceHandsOutTheDesktopServices(t *testing.T) {
	svc := mobile.NewDesktop(nil, nil, nil)
	tests := []struct {
		name string
		conn connection
		ask  string
		want any
	}{
		{"a platform with services", desktopConn{&conn{Platform: fake.New()}, svc}, mobile.ServiceDesktop, svc},
		{"another name", desktopConn{&conn{Platform: fake.New()}, svc}, "other", nil},
		{"a platform whose services are nil", desktopConn{&conn{Platform: fake.New()}, nil}, mobile.ServiceDesktop, nil},
		{"a platform without desktop services", &conn{Platform: fake.New()}, mobile.ServiceDesktop, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &handler{conn: tt.conn}
			d := ui.NewPlatformDispatcher(h)
			if got := d.PlatformService(tt.ask); got != tt.want {
				t.Errorf("PlatformService(%q) = %#v, want %#v", tt.ask, got, tt.want)
			}
		})
	}
}

// A platform whose keyboard focus is not its view focus (Wayland: wl_keyboard against
// the activated state) tells both: the keys, the input method and the cursor follow the
// keyboard, and the framework hears of the view focus, the first time as soon as the
// platform knows it, also when the window is not the active one (step 27.3).
func TestKeyboardFocusRoutesTheKeysAndViewFocusIsTold(t *testing.T) {
	e := newWindowsEngine(t)
	v, err := e.d.CreateWindow(ui.WindowSpec{Size: ui.Size{Width: 200, Height: 100}})
	if err != nil {
		t.Fatal(err)
	}
	second := v.ViewID()
	e.size(0, 800, 600)
	e.size(second, 200, 100)
	e.renderAll(map[ui.ViewID]*ui.Scene{0: {}, second: {}})
	var events []ui.ViewFocusEvent
	e.d.OnViewFocusChange = func(ev ui.ViewFocusEvent) { events = append(events, ev) }

	// The first configure of each window says it is not the active one, and the
	// events come with the input of the first frame.
	e.h.OnViewFocus(0, false)
	e.h.OnViewFocus(second, false)
	if len(events) != 0 {
		t.Fatalf("%d events before the frame", len(events))
	}
	e.h.OnFrame(0, time.Millisecond)
	want := []ui.ViewFocusEvent{{ViewID: 0, State: ui.ViewFocusStateUnfocused}, {ViewID: second, State: ui.ViewFocusStateUnfocused}}
	if len(events) != 2 || events[0] != want[0] || events[1] != want[1] {
		t.Fatalf("focus events = %+v, want %+v", events, want)
	}

	// The keyboard enters the second window: no view focus event, the input method follows.
	e.h.OnKeyboardFocus(second, true)
	e.h.SetTextInput(ui.TextInputState{Enabled: true})
	if !e.p.Windows()[1].TextInput().Enabled || e.p.Windows()[0].TextInput().Enabled {
		t.Error("the input method is not on the window with the keyboard")
	}
	e.h.OnFrame(0, 2*time.Millisecond)
	if len(events) != 2 {
		t.Errorf("the keyboard focus made view focus events: %+v", events[2:])
	}
	// The window with the keyboard is activated, and the other still is not.
	e.h.OnViewFocus(second, true)
	e.h.OnFrame(0, 3*time.Millisecond)
	if len(events) != 3 || events[2] != (ui.ViewFocusEvent{ViewID: second, State: ui.ViewFocusStateFocused}) {
		t.Errorf("focus events = %+v, want the activation of the second window", events)
	}
}
