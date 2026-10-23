//go:build linux && !android && glutter_display

package wayland

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// These tests need a Wayland compositor. Run them under the headless one:
//
//	scripts/headless-wayland.sh -- go test -race -tags glutter_display ./internal/platform/linux/wayland/

func connectToCompositor(t *testing.T) *Platform {
	t.Helper()
	// These tests check the shared-memory path; the EGL path is tested in
	// internal/gpu/gles.
	p, err := New("", Options{ShmFill: true})
	if err != nil {
		if appErr, ok := errors.FromError(err); ok {
			t.Fatalf("could not connect to the compositor: %s %v (is WAYLAND_DISPLAY set? run under scripts/headless-wayland.sh)", appErr.Tag, appErr.Metadata)
		}
		t.Fatalf("could not connect to the compositor: %v", err)
	}
	t.Cleanup(func() {
		if err := p.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	return p
}

func TestDisplayConnectListsRequiredGlobals(t *testing.T) {
	p := connectToCompositor(t)

	var all []string
	for iface := range p.sess.advertised {
		all = append(all, iface)
	}
	sort.Strings(all)
	t.Logf("the compositor offers: %v", all)

	for _, r := range required {
		globals, ok := p.Globals(r.iface.Name)
		if !ok {
			t.Errorf("required global %s is not offered", r.iface.Name)
			continue
		}
		t.Logf("%s: offered up to version %d, bound at %d", r.iface.Name, globals[0].Version, bindVersion(globals[0], r.max))
	}
	for name, g := range p.sess.optionalPresent {
		t.Logf("optional %s: version %d", name, g.Version)
	}

	s := p.sess
	if s.compositor == nil || s.subcompositor == nil || s.shm == nil || s.seat == nil || s.wmBase == nil || len(s.outputs) == 0 {
		t.Errorf("not every required global is bound: %+v", s)
	}
	if v := s.compositor.ProxyVersion(); v < 4 {
		t.Errorf("wl_compositor was bound at version %d, want at least 4", v)
	}
}

func TestDisplayRunAndPost(t *testing.T) {
	p := connectToCompositor(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	posted := make(chan struct{})
	go func() {
		time.Sleep(50 * time.Millisecond)
		p.Post(func() { close(posted) })
	}()
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx, nopHandler{}) }()

	select {
	case <-posted:
	case <-time.After(5 * time.Second):
		t.Fatal("a posted function did not run")
	}
	cancel()
	if err := <-done; err != nil {
		t.Errorf("Run returned %v, want nil", err)
	}
}

// TestDisplayWindowMapsAndCloses opens a window on the real compositor and
// checks the whole path: the first configure arrives, the window is mapped
// with a buffer of the asked size, and a close request ends Run.
func TestDisplayWindowMapsAndCloses(t *testing.T) {
	p := connectToCompositor(t)

	cfg := platform.DefaultWindowConfig()
	cfg.Title = "Glutter display test"
	cfg.AppID = "dev.glutter.displaytest"
	cfg.Size = ui.Size{Width: 640, Height: 400}
	if _, err := p.NewWindow(cfg); err != nil {
		t.Fatal(err)
	}

	h := &recHandler{allowClose: true}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx, h) }()

	waitUntil(t, func() bool { return len(h.metricsSeen()) > 0 }, "no configure arrived, so the window never got a size")
	var configures int
	var states WindowStates
	var w *window
	finished := make(chan struct{})
	p.Post(func() {
		w = p.windows[0]
		configures, states = w.configures, w.states
		if !w.mapped {
			t.Error("the window is not mapped after the first configure")
		}
		close(finished)
	})
	<-finished
	if configures < 1 {
		t.Errorf("configures = %d, want at least 1", configures)
	}
	m := h.metricsSeen()[0]
	if m.PhysicalSize.Width <= 0 || m.PhysicalSize.Height <= 0 || m.DevicePixelRatio < 1 {
		t.Errorf("metrics = %+v, want a positive size at a ratio of at least 1", m)
	}
	t.Logf("first configure: %vx%v, states %+v", m.PhysicalSize.Width, m.PhysicalSize.Height, states)

	// The compositor owns the close button; ask the window to close the way the
	// toolkit would on a close request.
	closed := make(chan struct{})
	p.Post(func() {
		w.onClose()
		close(closed)
	})
	<-closed
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned %v after the window closed", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return after the last window closed")
	}
	if got := h.closeRequests(); len(got) != 1 || got[0] != 0 {
		t.Errorf("close requests = %v, want [0]", got)
	}
}

// outputScale is the scale of the headless compositor's output: HEADLESS_SCALE, as
// scripts/headless-wayland.sh reads it, or 1.
func outputScale(t *testing.T) float64 {
	t.Helper()
	v := os.Getenv("HEADLESS_SCALE")
	if v == "" {
		return 1
	}
	scale, err := strconv.ParseFloat(v, 64)
	if err != nil || scale < 1 {
		t.Fatalf("HEADLESS_SCALE = %q", v)
	}
	return scale
}

// TestDisplayScaleSizesTheBuffer checks, against the real compositor, that a window's
// buffer is ceil(logical size x scale) and that the metrics say so. Run it with
// HEADLESS_SCALE=1.5 for a fractional scale (SPEC §8.11).
func TestDisplayScaleSizesTheBuffer(t *testing.T) {
	scale := outputScale(t)
	p := connectToCompositor(t)
	cfg := platform.DefaultWindowConfig()
	cfg.Title = "Glutter display test"
	cfg.Size = ui.Size{Width: 333, Height: 201} // sizes that do not scale to whole pixels
	if _, err := p.NewWindow(cfg); err != nil {
		t.Fatal(err)
	}
	h := &recHandler{allowClose: true}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx, h) }()

	wantW, wantH := math.Ceil(333*scale), math.Ceil(201*scale)
	waitUntil(t, func() bool {
		m := h.metricsSeen()
		return len(m) > 0 && m[len(m)-1].PhysicalSize == (ui.Size{Width: wantW, Height: wantH})
	}, "the metrics never reached the size of a buffer at the output's scale")
	if m := h.metricsSeen(); m[len(m)-1].DevicePixelRatio != scale {
		t.Errorf("device pixel ratio = %v, want %v", m[len(m)-1].DevicePixelRatio, scale)
	}
	var bufW, bufH int32
	var viewport bool
	finished := make(chan struct{})
	p.Post(func() {
		w := p.windows[0]
		bufW, bufH, viewport = w.buffer.width, w.buffer.height, w.viewportUsed
		close(finished)
	})
	<-finished
	if float64(bufW) != wantW || float64(bufH) != wantH {
		t.Errorf("the buffer is %dx%d pixels, want %vx%v", bufW, bufH, wantW, wantH)
	}
	// A fractional scale goes through a viewport; an integer one through the buffer scale.
	if _, ok := p.Globals("wp_fractional_scale_manager_v1"); ok && scale != math.Trunc(scale) && !viewport {
		t.Error("a fractional scale did not use a viewport")
	}
	cancel()
	if err := <-done; err != nil {
		t.Errorf("Run returned %v", err)
	}
}

// TestDisplayResizeNeverCommitsTheOldSize resizes a floating window through sway's IPC
// and checks that after each acknowledged configure the next buffer committed has the
// size that was acknowledged, and that the buffer follows the logical size and the
// scale. The sizes are the container's, which sway makes the window smaller than by its
// borders and title bar, so the test follows what the compositor decides.
func TestDisplayResizeNeverCommitsTheOldSize(t *testing.T) {
	if _, err := exec.LookPath("swaymsg"); err != nil || os.Getenv("SWAYSOCK") == "" {
		t.Skip("this test resizes the window through sway (run under scripts/headless-wayland.sh)")
	}
	scale := outputScale(t)
	p := connectToCompositor(t)
	cfg := platform.DefaultWindowConfig()
	cfg.Title = "Glutter resize test"
	cfg.AppID = "dev.glutter.resizetest"
	cfg.Size = ui.Size{Width: 400, Height: 300}
	win, err := p.NewWindow(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var trace []string
	win.(*window).trace = func(e string) { mu.Lock(); trace = append(trace, e); mu.Unlock() }

	h := &recHandler{allowClose: true}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx, h) }()
	// Whatever happens, Run ends before the platform is closed.
	defer func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("Run returned %v", err)
		}
	}()

	waitUntil(t, func() bool { return len(h.metricsSeen()) > 0 }, "no configure arrived")
	last := h.metricsSeen()[len(h.metricsSeen())-1].PhysicalSize
	for _, size := range [][2]int{{500, 350}, {321, 201}, {640, 480}, {400, 300}} {
		out, err := exec.Command("swaymsg", "[app_id=dev.glutter.resizetest]", "resize", "set", "width", strconv.Itoa(size[0]), "px", "height", strconv.Itoa(size[1]), "px").CombinedOutput()
		if err != nil {
			t.Fatalf("swaymsg: %v: %s", err, out)
		}
		waitUntil(t, func() bool {
			m := h.metricsSeen()
			return m[len(m)-1].PhysicalSize != last
		}, fmt.Sprintf("the window did not change size after resizing to %v (metrics: %v)", size, h.metricsSeen()))
		m := h.metricsSeen()
		last = m[len(m)-1].PhysicalSize
		// The buffer is the logical size times the scale, rounded up.
		var width, height int32
		finished := make(chan struct{})
		p.Post(func() { w := p.windows[0]; width, height = w.width, w.height; close(finished) })
		<-finished
		if want := (ui.Size{Width: math.Ceil(float64(width) * scale), Height: math.Ceil(float64(height) * scale)}); last != want {
			t.Errorf("after resizing to %v: the buffer is %v for a window of %dx%d at scale %v, want %v", size, last, width, height, scale, want)
		}
	}
	cancel()

	mu.Lock()
	defer mu.Unlock()
	acks := 0
	for i, e := range trace {
		if !strings.HasPrefix(e, "ack ") {
			continue
		}
		acks++
		if i+1 >= len(trace) || trace[i+1] != "commit "+strings.TrimPrefix(e, "ack ") {
			t.Errorf("the buffer committed after %q is not that size: %v", e, trace[i:min(i+2, len(trace))])
		}
	}
	if acks < 5 {
		t.Errorf("only %d configures were acknowledged: %v", acks, trace)
	}
}

// TestDisplayDecorationModeFollowsTheCompositor checks that the way the window is
// decorated is the first of SPEC §8.2 that the compositor allows: server-side
// where it has zxdg_decoration_manager_v1 (sway), libdecor where it has not
// (GNOME), and the title bar of the app when the app asks for it.
func TestDisplayDecorationModeFollowsTheCompositor(t *testing.T) {
	p := connectToCompositor(t)
	_, serverSide := p.Globals("zxdg_decoration_manager_v1")
	want := ui.DecorationLibdecor
	switch {
	case serverSide:
		want = ui.DecorationServerSide
	case p.libdecorContext() == nil:
		want = ui.DecorationApp
	}
	for _, tt := range []struct {
		name string
		set  func(*platform.WindowConfig)
		want ui.DecorationMode
	}{
		{"auto", func(*platform.WindowConfig) {}, want},
		{"the app draws it", func(c *platform.WindowConfig) { c.Decorations = platform.DecorationsApp }, ui.DecorationApp},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := platform.DefaultWindowConfig()
			cfg.Title = "Glutter display test"
			tt.set(&cfg)
			w, err := p.NewWindow(cfg)
			if err != nil {
				t.Fatal(err)
			}
			defer w.Close()
			if got := w.(*window).deco; got != tt.want {
				t.Errorf("decoration = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDisplayLibdecorWindowMaps makes a window with libdecor on whatever
// compositor there is (sway answers with its own title bar through
// xdg-decoration, which libdecor follows) and checks that its configure is
// answered, so that the window is mapped and reports its size.
func TestDisplayLibdecorWindowMaps(t *testing.T) {
	p := connectToCompositor(t)
	if p.libdecorContext() == nil {
		t.Skip("libdecor is not available")
	}
	cfg := platform.DefaultWindowConfig()
	cfg.Title = "Glutter libdecor test"
	cfg.Size = ui.Size{Width: 300, Height: 200}
	cfg.Decorations = platform.DecorationsLibdecor
	if _, err := p.NewWindow(cfg); err != nil {
		t.Fatal(err)
	}
	h := &recHandler{allowClose: true}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx, h) }()
	waitUntil(t, func() bool { return len(h.metricsSeen()) > 0 }, "the window never reported its metrics: libdecor's configure was not answered")
	info, _ := h.lastWindowInfo()
	if info.Decoration != ui.DecorationLibdecor || info.Title != cfg.Title {
		t.Errorf("window info = %+v", info)
	}
	cancel()
	if err := <-done; err != nil {
		t.Errorf("Run returned %v", err)
	}
}

// viewsHandler records the views that reported their metrics and the close requests.
type viewsHandler struct {
	platform.NopHandler
	mu      sync.Mutex
	metrics map[ui.ViewID]bool
	closed  []ui.ViewID
}

func (h *viewsHandler) OnMetrics(view ui.ViewID, _ platform.ViewMetrics) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.metrics == nil {
		h.metrics = map[ui.ViewID]bool{}
	}
	h.metrics[view] = true
}

func (h *viewsHandler) OnCloseRequest(view ui.ViewID) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closed = append(h.closed, view)
	return true
}

func (h *viewsHandler) mapped() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.metrics)
}

// swayTitles are the titles of the windows that sway has with the app ID, or nil
// when the compositor is not sway.
func swayTitles(t *testing.T, appID string) []string {
	t.Helper()
	if _, err := exec.LookPath("swaymsg"); err != nil || os.Getenv("SWAYSOCK") == "" {
		return nil
	}
	out, err := exec.Command("swaymsg", "-t", "get_tree").Output()
	if err != nil {
		t.Fatalf("swaymsg get_tree: %v", err)
	}
	var titles []string
	rest := string(out)
	for {
		i := strings.Index(rest, `"app_id": "`+appID+`"`)
		if i < 0 {
			break
		}
		// The name of the window comes before its app ID in the same node.
		before := rest[:i]
		j := strings.LastIndex(before, `"name": "`)
		if j >= 0 {
			name := before[j+len(`"name": "`):]
			titles = append(titles, name[:strings.Index(name, `"`)])
		}
		rest = rest[i+1:]
	}
	sort.Strings(titles)
	return titles
}

// TestDisplayThreeWindowsAndADialog creates two windows and a modal dialog of the
// first one on the real compositor: all of them are configured and mapped, sway
// has all three, and closing them one at a time, the parent first, ends Run with
// the last. sway shows no sign of the parent of a dialog (it centers every
// floating window), so what the compositor accepts is the proof: a parent or a
// dialog object that was wrong would be a protocol error and end the connection.
func TestDisplayThreeWindowsAndADialog(t *testing.T) {
	p := connectToCompositor(t)
	const appID = "dev.glutter.windowstest"
	main := platform.WindowConfig{Title: "Glutter main", AppID: appID, Size: ui.Size{Width: 400, Height: 300}}
	second := platform.WindowConfig{Title: "Glutter second", AppID: appID, Size: ui.Size{Width: 400, Height: 300}}
	dialog := platform.WindowConfig{
		Title: "Glutter dialog", AppID: appID, Size: ui.Size{Width: 300, Height: 200},
		Kind: platform.WindowKindDialog, HasParent: true, Parent: 0, Modal: true,
	}
	for _, cfg := range []platform.WindowConfig{main, second, dialog} {
		if _, err := p.NewWindow(cfg); err != nil {
			t.Fatal(err)
		}
	}
	_, hasDialogs := p.Globals("xdg_wm_dialog_v1")
	t.Logf("the compositor has xdg_wm_dialog_v1: %v", hasDialogs)

	h := &viewsHandler{}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx, h) }()
	waitUntil(t, func() bool { return h.mapped() == 3 }, "the three windows were not all configured")

	if titles := swayTitles(t, appID); titles != nil {
		want := []string{"Glutter dialog", "Glutter main", "Glutter second"}
		if strings.Join(titles, "|") != strings.Join(want, "|") {
			t.Errorf("sway has the windows %q, want %q", titles, want)
		}
	}

	closeView := func(view ui.ViewID) {
		finished := make(chan struct{})
		p.Post(func() {
			if w, ok := p.windows[view]; ok {
				w.onClose()
			}
			close(finished)
		})
		<-finished
	}
	closeView(0) // the parent goes first, the dialog stays
	select {
	case err := <-done:
		t.Fatalf("Run returned %v while two windows are open", err)
	case <-time.After(300 * time.Millisecond):
	}
	if titles := swayTitles(t, appID); titles != nil && len(titles) != 2 {
		t.Errorf("sway has %q after the first window closed, want 2 windows", titles)
	}
	closeView(2)
	closeView(1)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned %v after the last window closed", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return after the last window closed")
	}
	if got := h.closed; len(got) != 3 || got[0] != 0 {
		t.Errorf("close requests = %v, want the three windows, the first one first", got)
	}
}

// TestDisplayMovingBetweenOutputsRescalesTheNextFrame moves a window between two
// outputs of different scales and checks that the metrics follow at once and that the
// next buffer committed has the size of the new scale (no blurry frame, SPEC §8.11).
// Run it with HEADLESS_SECOND=800x600@2.
func TestDisplayMovingBetweenOutputsRescalesTheNextFrame(t *testing.T) {
	if _, err := exec.LookPath("swaymsg"); err != nil || os.Getenv("SWAYSOCK") == "" || os.Getenv("HEADLESS_SECOND") == "" {
		t.Skip("this test moves the window between two outputs through sway (HEADLESS_SECOND=800x600@2 scripts/headless-wayland.sh)")
	}
	p := connectToCompositor(t)
	cfg := platform.DefaultWindowConfig()
	cfg.Title = "Glutter outputs test"
	cfg.AppID = "dev.glutter.outputstest"
	cfg.Size = ui.Size{Width: 300, Height: 200}
	win, err := p.NewWindow(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var trace []string
	win.(*window).trace = func(e string) { mu.Lock(); trace = append(trace, e); mu.Unlock() }
	commits := func() []string {
		mu.Lock()
		defer mu.Unlock()
		var out []string
		for _, e := range trace {
			if strings.HasPrefix(e, "commit ") {
				out = append(out, strings.TrimPrefix(e, "commit "))
			}
		}
		return out
	}

	h := &recHandler{allowClose: true}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx, h) }()
	defer func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("Run returned %v", err)
		}
	}()

	last := func() platform.ViewMetrics { m := h.metricsSeen(); return m[len(m)-1] }
	waitUntil(t, func() bool { return len(h.metricsSeen()) > 0 && last().Display.IsSome() }, "the window never got a size and a display")
	first := last()

	// Two outputs: the first at the scale of HEADLESS_SCALE, the second at 2.
	var displays []platform.Display
	finished := make(chan struct{})
	p.Post(func() { displays = p.Displays(); close(finished) })
	<-finished
	if len(displays) != 2 {
		t.Fatalf("Displays() = %+v, want the two outputs", displays)
	}
	if !displays[0].Primary || displays[1].Primary || displays[1].DevicePixelRatio != 2 {
		t.Errorf("Displays() = %+v", displays)
	}
	t.Logf("displays: %+v", displays)

	move := func(output string) {
		t.Helper()
		out, err := exec.Command("swaymsg", "[app_id=dev.glutter.outputstest]", "move", "container", "to", "output", output).CombinedOutput()
		if err != nil {
			t.Fatalf("swaymsg: %v: %s", err, out)
		}
	}
	scaleOf := func(display int) float64 {
		for _, d := range displays {
			if d.ID == display {
				return d.DevicePixelRatio
			}
		}
		return 0
	}

	for _, step := range []struct {
		output string
		scale  float64
	}{{"HEADLESS-2", 2}, {"HEADLESS-1", scaleOf(first.Display.Value())}} {
		before := len(commits())
		move(step.output)
		waitUntil(t, func() bool { return last().DevicePixelRatio == step.scale },
			fmt.Sprintf("the scale never became %v after moving to %s (metrics: %+v)", step.scale, step.output, last()))
		m := last()
		if m.Display.IsNone() || scaleOf(m.Display.Value()) != step.scale {
			t.Errorf("after moving to %s the metrics name display %v", step.output, m.Display)
		}
		// The next buffer committed has the pixel size the metrics promise.
		want := fmt.Sprintf("%dx%d", int(m.PhysicalSize.Width), int(m.PhysicalSize.Height))
		waitUntil(t, func() bool {
			c := commits()
			return len(c) > before && slices.Contains(c[before:], want)
		}, fmt.Sprintf("no buffer of %s was committed after moving to %s (commits: %v)", want, step.output, commits()[before:]))
		if c := commits(); c[len(c)-1] != want && !slices.Contains(c[before:], want) {
			t.Errorf("the commits after the move were %v, want %s", c[before:], want)
		}
	}
}
