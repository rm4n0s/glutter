//go:build glutter_browser

package gles_test

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/proof"
	"github.com/rm4n0s/glutter/internal/webtest"
	"github.com/rm4n0s/glutter/internal/webtool"
)

// These tests run the gles renderer on WebGL 2 in real browsers (SPEC §17 "Browser tests"). The
// program is internal/gpu/gles/glwebtest: its mode says what it does, and it posts what it saw.
// Headless Chromium gets its WebGL 2 from SwiftShader (the flag is in internal/webtest/browser.go
// with its reason); Firefox from whatever its headless mode finds. Nothing is skipped when a
// browser has no WebGL 2: the test fails, saying so.

var (
	glwebOnce sync.Once
	glwebDir  string
	glwebErr  error
)

func glwebSite(t *testing.T) string {
	t.Helper()
	glwebOnce.Do(func() {
		if glwebDir, glwebErr = os.MkdirTemp("", "glutter-web-gltest-"); glwebErr != nil {
			return
		}
		_, glwebErr = webtool.Build(context.Background(), webtool.BuildOptions{
			Package: "./internal/gpu/gles/glwebtest", Dir: webtest.ModuleRoot(t), OutDir: glwebDir,
		})
	})
	if glwebErr != nil {
		t.Fatalf("building the WebGL test program for the web: %v", glwebErr)
	}
	return glwebDir
}

func TestMain(m *testing.M) {
	code := m.Run()
	if glwebDir != "" {
		_ = os.RemoveAll(glwebDir) // a temporary directory: a leftover is harmless
	}
	os.Exit(code)
}

// await runs the program in the browser with the query and returns the result it posts.
func await(t *testing.T, b webtest.Browser, query url.Values, timeout time.Duration) json.RawMessage {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout+30*time.Second)
	defer cancel()
	res, err := webtest.Await(ctx, webtest.ShotOptions{Dir: glwebSite(t), Browser: b, Query: query, Width: 480, Height: 640, Timeout: timeout}, "glweb")
	if err != nil {
		t.Fatalf("Await in %s: %v", b, err)
	}
	return res.Data
}

type diffStats struct{ Pixels, Over8, Over32, Over96, Max int }

type sceneResult struct {
	Name            string    `json:"name"`
	Width           int       `json:"width"`
	Height          int       `json:"height"`
	Frame           diffStats `json:"frame"`
	Snapshot        diffStats `json:"snapshot"`
	SnapshotVsFrame diffStats `json:"snapshotVsFrame"`
	Repeat          diffStats `json:"repeat"`
	Skipped         int       `json:"skipped"`
	Unsupported     []string  `json:"unsupported"`
	Error           string    `json:"error"`
}

type parityResult struct {
	Scenes   []sceneResult `json:"scenes"`
	Renderer string        `json:"renderer"`
}

func (d diffStats) gputest() gputest.Diff {
	return gputest.Diff{Pixels: d.Pixels, Over8: d.Over8, Over32: d.Over32, Over96: d.Over96, Max: d.Max}
}

// TestBrowserEveryProofSceneMatchesSoftware is the parity test of step 3.6 on WebGL 2: every
// proof scene (the goldens of the proof tool) is drawn by the gles renderer into a canvas and
// read back, and is within the tolerance that the display tests give the GPU (gputest.ProofTolerances)
// of what the software renderer draws. The renderer skips nothing; its snapshot, an offscreen
// framebuffer, is within the same tolerance, and a frame drawn twice is the same twice.
func TestBrowserEveryProofSceneMatchesSoftware(t *testing.T) {
	webtest.EachBrowser(t, func(t *testing.T, b webtest.Browser) {
		var res parityResult
		if err := json.Unmarshal(await(t, b, url.Values{"mode": {"parity"}}, 8*time.Minute), &res); err != nil {
			t.Fatal(err)
		}
		t.Logf("%s draws with %q", b, res.Renderer)
		got := map[string]sceneResult{}
		for _, s := range res.Scenes {
			got[s.Name] = s
		}
		for _, name := range proof.Names() {
			t.Run(name, func(t *testing.T) {
				s, ok := got[name]
				if !ok {
					t.Fatalf("the program did not draw the scene %q", name)
				}
				if s.Error != "" {
					t.Fatalf("the scene failed: %s", s.Error)
				}
				tol, ok := gputest.ProofTolerances[name]
				if !ok {
					t.Fatalf("the proof scene %q has no tolerance in gputest.ProofTolerances", name)
				}
				if s.Skipped != 0 {
					t.Errorf("the renderer skipped %d commands: %v", s.Skipped, s.Unsupported)
				}
				tol.Check(t, name+" frame", s.Frame.gputest())
				tol.Check(t, name+" snapshot", s.Snapshot.gputest())
				if s.Repeat.Max != 0 {
					t.Errorf("the frame drawn a second time differs from the first by up to %d levels", s.Repeat.Max)
				}
				t.Logf("snapshot against frame: %+v", s.SnapshotVsFrame)
				if s.SnapshotVsFrame.Max != 0 {
					t.Errorf("the snapshot differs from the frame by up to %d levels", s.SnapshotVsFrame.Max)
				}
			})
		}
		if len(got) != len(proof.Names()) {
			t.Errorf("the program drew %d scenes, there are %d", len(got), len(proof.Names()))
		}
	})
}

type tagResult struct {
	Tag   string `json:"tag"`
	Route string `json:"route"`
}

// TestBrowserUnavailable: a canvas that has a 2D context gives no WebGL 2 context (a canvas keeps
// the first kind it was asked for), and every failure of making the context has its tag; a new
// canvas gives one.
func TestBrowserUnavailable(t *testing.T) {
	webtest.EachBrowser(t, func(t *testing.T, b webtest.Browser) {
		var res struct {
			Taken     tagResult `json:"taken"`
			NoCanvas  tagResult `json:"noCanvas"`
			WrongKind tagResult `json:"wrongKind"`
			Fresh     bool      `json:"fresh"`
			Second    tagResult `json:"second"`
		}
		if err := json.Unmarshal(await(t, b, url.Values{"mode": {"unavailable"}}, time.Minute), &res); err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct {
			name, tag, route string
			got              tagResult
		}{
			{"canvas with a 2D context", "WebGL2Unavailable", "Attach->", res.Taken},
			{"view without a canvas", "WebGLCanvasNotFound", "Attach->", res.NoCanvas},
			{"handle of another kind", "WebGLSurfaceKindUnsupported", "Attach->", res.WrongKind},
			{"second view on one context", "EGLSurfaceFailed", "Attach->", res.Second},
		} {
			if tc.got.Tag != tc.tag || !strings.Contains(tc.got.Route, tc.route) || !strings.Contains(tc.got.Route, tc.tag) {
				t.Errorf("%s: got %s via %s, want %s via a route with %s", tc.name, tc.got.Tag, tc.got.Route, tc.tag, tc.route)
			}
		}
		if !res.Fresh {
			t.Error("a new canvas gave no WebGL 2 context")
		}
	})
}

// TestBrowserContextLoss: WEBGL_lose_context takes the context away. While it is lost, Render
// draws nothing and fails nothing, and a read-back and a snapshot are WebGLContextLost. When the
// browser restores it, the callback is told once, and the frame after that is the first frame
// again, pixel for pixel: every resource was made again, and the frame is drawn whole.
func TestBrowserContextLoss(t *testing.T) {
	webtest.EachBrowser(t, func(t *testing.T, b webtest.Browser) {
		var res struct {
			First            bool      `json:"first"`
			LostCapture      string    `json:"lostCapture"`
			LostRenderOK     bool      `json:"lostRenderOK"`
			LostSnapshot     string    `json:"lostSnapshot"`
			Restored         int       `json:"restored"`
			AfterError       string    `json:"afterError"`
			AfterVsFirst     diffStats `json:"afterVsFirst"`
			SnapshotVsFirst  diffStats `json:"snapshotVsFirst"`
			AfterRenderError string    `json:"afterRenderError"`
		}
		if err := json.Unmarshal(await(t, b, url.Values{"mode": {"loss"}}, time.Minute), &res); err != nil {
			t.Fatal(err)
		}
		t.Logf("%+v", res)
		if !res.First {
			t.Fatalf("the first frame failed: %s", res.AfterError)
		}
		if res.LostCapture != "WebGLContextLost" {
			t.Errorf("the read-back while lost: %q, want WebGLContextLost", res.LostCapture)
		}
		if !res.LostRenderOK {
			t.Error("Render failed while the context was lost")
		}
		if res.LostSnapshot != "WebGLContextLost" {
			t.Errorf("the snapshot while lost: %q, want WebGLContextLost", res.LostSnapshot)
		}
		if res.Restored != 1 {
			t.Errorf("the restoration reached the renderer's callback %d times, want 1", res.Restored)
		}
		if res.AfterError != "" || res.AfterRenderError != "" {
			t.Fatalf("after the restoration: %q, %q", res.AfterError, res.AfterRenderError)
		}
		if res.AfterVsFirst.Max != 0 || res.SnapshotVsFirst.Max != 0 {
			t.Errorf("the frames after the restoration differ from the first: frame %+v, snapshot %+v", res.AfterVsFirst, res.SnapshotVsFirst)
		}
	})
}

type chooseResult struct {
	Error          string `json:"error"`
	Canvases       int    `json:"canvases"`
	WebGL2         bool   `json:"webgl2"`
	Context2D      bool   `json:"context2d"`
	FramesBefore   int    `json:"framesBefore"`
	FramesWhileOff int    `json:"framesWhileOff"`
	FramesAfter    int    `json:"framesAfter"`
}

// TestBrowserRendererChoice: the engine's web renderer list is webgl, then soft (SPEC §9.6). The
// canvas of the app tells which one was chosen, because a canvas has the one kind of context it was
// first asked for: auto and webgl choose WebGL 2, soft the 2D canvas, and when the browser has no
// WebGL 2 (the test makes getContext answer null for it) auto falls back to soft on a new canvas,
// leaving one canvas in the page, while webgl by name fails with WebGL2Unavailable.
func TestBrowserRendererChoice(t *testing.T) {
	tests := []struct {
		name  string
		query url.Values
		want  chooseResult
	}{
		{"auto", url.Values{}, chooseResult{Canvases: 1, WebGL2: true}},
		{"webgl by name", url.Values{"GLUTTER_RENDERER": {"webgl"}}, chooseResult{Canvases: 1, WebGL2: true}},
		{"soft by name", url.Values{"GLUTTER_RENDERER": {"soft"}}, chooseResult{Canvases: 1, Context2D: true}},
		{"auto without WebGL 2", url.Values{"poison": {"1"}}, chooseResult{Canvases: 1, Context2D: true}},
		{"webgl by name without WebGL 2", url.Values{"GLUTTER_RENDERER": {"webgl"}, "poison": {"1"}}, chooseResult{Error: "WebGL2Unavailable"}},
		{"gles on the web", url.Values{"GLUTTER_RENDERER": {"gles"}}, chooseResult{Error: "RendererUnavailable"}},
		{"webgpu before step 21.5", url.Values{"GLUTTER_RENDERER": {"webgpu"}}, chooseResult{Error: "RendererUnavailable"}},
	}
	webtest.EachBrowser(t, func(t *testing.T, b webtest.Browser) {
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				q := url.Values{"mode": {"choose"}}
				for k, v := range tc.query {
					q[k] = v
				}
				var got chooseResult
				if err := json.Unmarshal(await(t, b, q, time.Minute), &got); err != nil {
					t.Fatal(err)
				}
				if got.Error != tc.want.Error || got.Canvases != tc.want.Canvases || got.WebGL2 != tc.want.WebGL2 || got.Context2D != tc.want.Context2D {
					t.Errorf("got %+v, want %+v", got, tc.want)
				}
			})
		}
	})
}

// TestBrowserRestoreAsksForAFrame: an idle app asks for no frames; when its WebGL context is lost
// it still asks for none, and when the browser restores the context the renderer makes its
// resources again and asks the engine for a frame, which draws the last scene whole.
func TestBrowserRestoreAsksForAFrame(t *testing.T) {
	webtest.EachBrowser(t, func(t *testing.T, b webtest.Browser) {
		var got chooseResult
		if err := json.Unmarshal(await(t, b, url.Values{"mode": {"restore"}}, time.Minute), &got); err != nil {
			t.Fatal(err)
		}
		t.Logf("%+v", got)
		if !got.WebGL2 {
			t.Fatalf("the app has no WebGL 2 context: %+v", got)
		}
		if got.FramesWhileOff != got.FramesBefore {
			t.Errorf("frames were asked for while the context was lost: %d before, %d after the loss", got.FramesBefore, got.FramesWhileOff)
		}
		if got.FramesAfter <= got.FramesWhileOff {
			t.Errorf("no frame was asked for after the restoration: %d while lost, %d after", got.FramesWhileOff, got.FramesAfter)
		}
	})
}
