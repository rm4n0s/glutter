//go:build js

package engine

import (
	"context"
	"log/slog"
	"math"
	"reflect"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
)

// TestRunOutsideAPageOnJS runs under Node, which has no page: the web platform says so with
// BackendUnavailable. The browser tests (internal/webtest) run the real thing.
func TestRunOutsideAPageOnJS(t *testing.T) {
	err := Run(context.Background(), Config{Logger: quiet()})
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if appErr.Tag != "BackendUnavailable" || !appErr.HasRoute("Run->connectDisplay->New->errNoPage.BackendUnavailable") {
		t.Errorf("got %s via %s, want BackendUnavailable via Run->connectDisplay->New->errNoPage", appErr.Tag, appErr.Route())
	}
}

func TestWebRendererCandidates(t *testing.T) {
	cands := rendererCandidates()
	var names []string
	for _, c := range cands {
		names = append(names, c.Name)
	}
	if !reflect.DeepEqual(names, []string{"webgpu", "webgl", "soft"}) {
		t.Errorf("the web renderers are %v, want webgpu, webgl then soft", names)
	}
	// Only WebGPU has a last resort (SPEC §9.5): a lost device that cannot be had again switches the
	// page to WebGL 2, while a lost WebGL context is restored by its own renderer.
	for i, c := range cands {
		if hasLastResort := c.SwitchOn != nil; hasLastResort != (i == 0) {
			t.Errorf("renderer %s: last resort %v", c.Name, hasLastResort)
		}
	}
	// Names of renderers that exist on other OSes are RendererUnavailable here.
	for _, name := range []string{"vulkan", "gles"} {
		_, err := newChooser(&Config{}, quiet(), name, cands)
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != "RendererUnavailable" {
			t.Errorf("renderer %q: got %v, want RendererUnavailable", name, err)
		}
	}
	for _, name := range []string{"webgpu", "webgl", "soft", "auto", ""} {
		if _, err := newChooser(&Config{}, quiet(), name, cands); err != nil {
			t.Errorf("renderer %q: %v", name, err)
		}
	}
	if _, err := newChooser(&Config{}, quiet(), "bogus", cands); err == nil {
		t.Error("an unknown renderer was accepted")
	}
}

func TestWebFontsAreBundledOnly(t *testing.T) {
	m := SystemFontMatcher(context.Background(), quiet())
	files, err := m.Match(context.Background(), "sans-serif", 400, false, "")
	if err != nil || len(files) != 0 {
		t.Errorf("Match = %v, %v; want no system fonts and no error", files, err)
	}
}

// TestConfigPostsWhereTheUIThreadIs: before Run has made the platform there is no other thread, so
// a function posted by the startup runs at once; after, it goes through the platform's Post.
func TestConfigPostsWhereTheUIThreadIs(t *testing.T) {
	var c Config
	ran := 0
	c.postToUI(func() { ran++ })
	if ran != 1 {
		t.Fatalf("posted before Run: ran %d times, want 1 at once", ran)
	}
	var queued []func()
	c.post = func(fn func()) { queued = append(queued, fn) }
	c.postToUI(func() { ran++ })
	if ran != 1 || len(queued) != 1 {
		t.Fatalf("posted after Run: ran %d, queued %d; want it queued", ran, len(queued))
	}
	c.redrawWindow(gpu.SurfaceHandle{}) // no hook yet: nothing happens
	var redrawn gpu.SurfaceHandle
	c.redraw = func(h gpu.SurfaceHandle) { redrawn = h }
	c.redrawWindow(gpu.SurfaceHandle{Kind: gpu.SurfaceKindWeb, Window: 7})
	if redrawn.Window != 7 {
		t.Errorf("redraw got %v", redrawn)
	}
}

// webFakes are the web's three candidates, with their names and their last resort, over fake
// renderers; fail makes the named ones fail their first frame with the tag.
func webFakes(fail map[string]string) ([]rendererCandidate, *[]*namedRenderer) {
	made := &[]*namedRenderer{}
	cands := rendererCandidates()
	for i := range cands {
		name := cands[i].Name
		cands[i].New = func(*Config, *slog.Logger) renderer {
			r := &namedRenderer{fakeRenderer: &fakeRenderer{}, name: name}
			if tag := fail[name]; tag != "" {
				r.err = errors.New(tag, "this renderer does not work")
			}
			*made = append(*made, r)
			return r
		}
	}
	return cands, made
}

// TestWebAutoChoosesAtTheFirstFrame: the web's list in order, with a new canvas before each try after
// the first (SPEC §9.6): WebGPU when it works; WebGL 2 when WebGPU does not; soft when neither does; a
// name never falls back; and with every renderer failing the last error is returned.
func TestWebAutoChoosesAtTheFirstFrame(t *testing.T) {
	handle := gpu.SurfaceHandle{Kind: gpu.SurfaceKindWeb, Window: 1}
	tests := []struct {
		name      string
		want      string
		fail      map[string]string
		wantMade  string
		wantTag   string
		wantRenew int
	}{
		{name: "WebGPU works", want: "auto", wantMade: "webgpu"},
		{name: "no WebGPU", want: "auto", fail: map[string]string{"webgpu": "WebGPUUnavailable"}, wantMade: "webgpu,webgl", wantRenew: 1},
		{name: "no WebGPU or WebGL 2", want: "auto", fail: map[string]string{"webgpu": "WebGPUUnavailable", "webgl": "WebGL2Unavailable"},
			wantMade: "webgpu,webgl,soft", wantRenew: 2},
		{name: "a device that cannot be had", want: "auto", fail: map[string]string{"webgpu": "WebGPUDeviceFailed"}, wantMade: "webgpu,webgl", wantRenew: 1},
		{name: "everything fails", want: "auto", fail: map[string]string{"webgpu": "A", "webgl": "B", "soft": "C"},
			wantMade: "webgpu,webgl,soft", wantTag: "C", wantRenew: 2},
		{name: "webgpu by name", want: "webgpu", fail: map[string]string{"webgpu": "WebGPUUnavailable"}, wantMade: "webgpu", wantTag: "WebGPUUnavailable"},
		{name: "webgl by name", want: "webgl", fail: map[string]string{"webgl": "WebGL2Unavailable"}, wantMade: "webgl", wantTag: "WebGL2Unavailable"},
		{name: "soft by name", want: "soft", wantMade: "soft"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cands, made := webFakes(tc.fail)
			renewed := 0
			c, err := newChooser(&Config{renewSurface: func(gpu.SurfaceHandle) { renewed++ }}, quiet(), tc.want, cands)
			if err != nil {
				t.Fatal(err)
			}
			_, err = c.Draw(handle, 10, 10, nil, false)
			if appErr, ok := errors.FromError(err); tc.wantTag == "" && err != nil || tc.wantTag != "" && (!ok || appErr.Tag != tc.wantTag) {
				t.Fatalf("Draw = %v, want tag %q", err, tc.wantTag)
			}
			var names []string
			for _, r := range *made {
				names = append(names, r.name)
			}
			if got := strings.Join(names, ","); got != tc.wantMade {
				t.Errorf("renderers made %q, want %q", got, tc.wantMade)
			}
			if renewed != tc.wantRenew {
				t.Errorf("the canvas was renewed %d times, want %d", renewed, tc.wantRenew)
			}
		})
	}
}

// TestWebGPULastResortSwitchesToWebGL: a WebGPU that draws and then loses its device for good (a second
// loss within ten seconds, or a new device that the browser refuses) hands the page to WebGL 2, on a new
// canvas, with the same frame; a name never switches (SPEC §9.5, §9.6).
func TestWebGPULastResortSwitchesToWebGL(t *testing.T) {
	handle := gpu.SurfaceHandle{Kind: gpu.SurfaceKindWeb, Window: 1}
	for _, tag := range []string{"GPUDeviceLost", "WebGPUDeviceFailed"} {
		t.Run(tag, func(t *testing.T) {
			cands, made := webFakes(nil)
			renewed := 0
			c, err := newChooser(&Config{renewSurface: func(gpu.SurfaceHandle) { renewed++ }}, quiet(), "auto", cands)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := c.Draw(handle, 10, 10, nil, false); err != nil {
				t.Fatal(err)
			}
			(*made)[0].mu.Lock()
			(*made)[0].err = errors.New(tag, "the device is gone")
			(*made)[0].mu.Unlock()
			if _, err := c.Draw(handle, 10, 10, nil, false); err != nil {
				t.Fatalf("the frame after the last resort: %v", err)
			}
			if renewed != 1 || len(*made) != 2 || (*made)[1].name != "webgl" || c.Describe() != "webgl" {
				t.Errorf("renewed %d, made %d, in use %q; want one new canvas and webgl", renewed, len(*made), c.Describe())
			}
			// WebGL 2 has no last resort: its errors are the app's.
			(*made)[1].mu.Lock()
			(*made)[1].err = errors.New(tag, "lost again")
			(*made)[1].mu.Unlock()
			if _, err := c.Draw(handle, 10, 10, nil, false); err == nil {
				t.Error("an error of WebGL 2 was swallowed")
			}
		})
	}
	cands, made := webFakes(nil)
	c, _ := newChooser(&Config{}, quiet(), "webgpu", cands)
	if _, err := c.Draw(handle, 10, 10, nil, false); err != nil {
		t.Fatal(err)
	}
	(*made)[0].err = errors.New("GPUDeviceLost", "the device is gone")
	if _, err := c.Draw(handle, 10, 10, nil, false); err == nil || len(*made) != 1 {
		t.Errorf("webgpu by name switched (%d renderers) or hid the error: %v", len(*made), err)
	}
}

// TestTuneGC: the web collects when the heap has grown five times (a screen's build allocates megabytes on
// the one thread that the page has), within a soft memory limit, unless the environment says otherwise.
func TestTuneGC(t *testing.T) {
	oldPercent := debug.SetGCPercent(100)
	oldLimit := debug.SetMemoryLimit(math.MaxInt64)
	t.Cleanup(func() {
		debug.SetGCPercent(oldPercent)
		debug.SetMemoryLimit(oldLimit)
	})
	t.Setenv("GOGC", "")
	t.Setenv("GOMEMLIMIT", "")
	tuneGC()
	if got := debug.SetGCPercent(100); got != webGCPercent {
		t.Errorf("GC percent = %d, want %d", got, webGCPercent)
	}
	if got := debug.SetMemoryLimit(-1); got != webMemoryLimit {
		t.Errorf("memory limit = %d, want %d", got, int64(webMemoryLimit))
	}

	// A value that the program was started with is kept.
	debug.SetGCPercent(150)
	debug.SetMemoryLimit(math.MaxInt64)
	t.Setenv("GOGC", "150")
	t.Setenv("GOMEMLIMIT", "1GiB")
	tuneGC()
	if got := debug.SetGCPercent(100); got != 150 {
		t.Errorf("GC percent = %d, want the 150 that GOGC gave", got)
	}
	if got := debug.SetMemoryLimit(-1); got != math.MaxInt64 {
		t.Errorf("memory limit = %d, want it left alone", got)
	}
}
