//go:build js

package engine

import (
	"context"
	"testing"

	"github.com/rm4n0s/errors"
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
	if len(cands) != 1 || cands[0].Name != "soft" {
		t.Errorf("the web renderers are %v, want only soft for now", cands)
	}
	// Names of renderers that exist on other OSes or come later are RendererUnavailable here.
	for _, name := range []string{"vulkan", "gles", "webgpu", "webgl"} {
		_, err := newChooser(&Config{}, quiet(), name, cands)
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != "RendererUnavailable" {
			t.Errorf("renderer %q: got %v, want RendererUnavailable", name, err)
		}
	}
	if _, err := newChooser(&Config{}, quiet(), "soft", cands); err != nil {
		t.Errorf("renderer soft: %v", err)
	}
}

func TestWebFontsAreBundledOnly(t *testing.T) {
	m := SystemFontMatcher(context.Background(), quiet())
	files, err := m.Match(context.Background(), "sans-serif", 400, false, "")
	if err != nil || len(files) != 0 {
		t.Errorf("Match = %v, %v; want no system fonts and no error", files, err)
	}
}
