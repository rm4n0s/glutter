//go:build js

package engine

import (
	"context"
	"reflect"
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
