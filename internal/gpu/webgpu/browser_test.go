//go:build glutter_browser

package webgpu_test

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/proof"
	"github.com/rm4n0s/glutter/internal/webtest"
	"github.com/rm4n0s/glutter/internal/webtool"
)

// These tests run the explicit renderer on WebGPU in real browsers (SPEC §17 "Browser tests"). The
// program is internal/gpu/webgpu/gpuwebtest: its mode says what it does, and it posts what it saw.
// Headless Chromium gets WebGPU from the flag in internal/webtest/browser.go (SwiftShader's
// adapter here); Firefox gets it from a preference, where its build has it. Chromium without
// WebGPU fails every test; Firefox without it skips them, saying why in the log and in the skip.

var (
	gpuwebOnce sync.Once
	gpuwebDir  string
	gpuwebErr  error
)

func gpuwebSite(t *testing.T) string {
	t.Helper()
	gpuwebOnce.Do(func() {
		if gpuwebDir, gpuwebErr = os.MkdirTemp("", "glutter-web-gpuwebtest-"); gpuwebErr != nil {
			return
		}
		_, gpuwebErr = webtool.Build(context.Background(), webtool.BuildOptions{
			Package: "./internal/gpu/webgpu/gpuwebtest", Dir: webtest.ModuleRoot(t), OutDir: gpuwebDir,
		})
	})
	if gpuwebErr != nil {
		t.Fatalf("building the WebGPU test program for the web: %v", gpuwebErr)
	}
	return gpuwebDir
}

func TestMain(m *testing.M) {
	code := m.Run()
	if gpuwebDir != "" {
		_ = os.RemoveAll(gpuwebDir) // a temporary directory: a leftover is harmless
	}
	os.Exit(code)
}

// await runs the program in the browser with the query and returns the result it posts.
func await(t *testing.T, b webtest.Browser, query url.Values, timeout time.Duration) json.RawMessage {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout+30*time.Second)
	defer cancel()
	res, err := webtest.Await(ctx, webtest.ShotOptions{Dir: gpuwebSite(t), Browser: b, Query: query, Width: 480, Height: 640, Timeout: timeout}, "gpuweb")
	if err != nil {
		t.Fatalf("Await in %s: %v", b, err)
	}
	return res.Data
}

// eachBrowserWithWebGPU runs fn for each browser that has WebGPU. Chromium must: the flag that
// turns it on is kept in internal/webtest, and an adapter that is missing there is a failure. A
// browser build without WebGPU (Firefox on Linux, unless its build has the implementation) skips
// its subtest, with the reason that the program found.
func eachBrowserWithWebGPU(t *testing.T, fn func(t *testing.T, b webtest.Browser)) {
	t.Helper()
	webtest.EachBrowser(t, func(t *testing.T, b webtest.Browser) {
		var p struct {
			Available bool   `json:"available"`
			Tag       string `json:"tag"`
			Message   string `json:"message"`
		}
		if err := json.Unmarshal(await(t, b, url.Values{"mode": {"probe"}}, time.Minute), &p); err != nil {
			t.Fatal(err)
		}
		if !p.Available {
			if b == webtest.Chromium {
				t.Fatalf("Chromium has no WebGPU (%s: %s): the flags in internal/webtest/browser.go should give it", p.Tag, p.Message)
			}
			t.Skipf("%s has no WebGPU in this environment (%s: %s)", b, p.Tag, p.Message)
		}
		fn(t, b)
	})
}

type diffStats struct{ Pixels, Over8, Over32, Over96, Max int }

func (d diffStats) gputest() gputest.Diff {
	return gputest.Diff{Pixels: d.Pixels, Over8: d.Over8, Over32: d.Over32, Over96: d.Over96, Max: d.Max}
}

type sceneResult struct {
	Name        string    `json:"name"`
	Width       int       `json:"width"`
	Height      int       `json:"height"`
	Frame       diffStats `json:"frame"`
	Snapshot    diffStats `json:"snapshot"`
	Repeat      diffStats `json:"repeat"`
	Skipped     int       `json:"skipped"`
	Unsupported []string  `json:"unsupported"`
	Error       string    `json:"error"`
}

type parityResult struct {
	Scenes     []sceneResult `json:"scenes"`
	Renderer   string        `json:"renderer"`
	Validation []string      `json:"validation"`
}

// TestBrowserEveryProofSceneMatchesSoftware is the parity test of step 3.6 on WebGPU: every proof
// scene (the goldens of the proof tool) is drawn by the explicit renderer into a canvas, read back
// in a later task, and is within the tolerance that the Vulkan and GLES renderers have
// (gputest.ProofTolerances) of what the software renderer draws. The renderer skips nothing; its
// snapshot is the software renderer's, so it is the same pixels; a frame drawn twice is the same
// twice; and the browser reports no validation error for any call.
func TestBrowserEveryProofSceneMatchesSoftware(t *testing.T) {
	eachBrowserWithWebGPU(t, func(t *testing.T, b webtest.Browser) {
		var res parityResult
		if err := json.Unmarshal(await(t, b, url.Values{"mode": {"parity"}}, 10*time.Minute), &res); err != nil {
			t.Fatal(err)
		}
		t.Logf("%s draws with %q", b, res.Renderer)
		if len(res.Validation) != 0 {
			t.Errorf("the browser reported %d validation errors, the first: %s", len(res.Validation), res.Validation[0])
		}
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
				if s.Snapshot.Max != 0 {
					t.Errorf("the snapshot (drawn by soft) differs from soft by up to %d levels", s.Snapshot.Max)
				}
				if s.Repeat.Max != 0 {
					t.Errorf("the frame drawn a second time differs from the first by up to %d levels", s.Repeat.Max)
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

// TestBrowserUnavailable: every way WebGPU can fail has its tag, and its route starts at the call
// that the program made. A browser without navigator.gpu or without an adapter is WebGPUUnavailable,
// an adapter that gives no device or an Acquire that is not answered in time is WebGPUDeviceFailed,
// a WGSL module that the browser rejects is ShaderModuleInvalid; then the swapchains of canvases
// that cannot be used.
func TestBrowserUnavailable(t *testing.T) {
	eachBrowserWithWebGPU(t, func(t *testing.T, b webtest.Browser) {
		var res struct {
			NoGPU      tagResult `json:"noGPU"`
			NoAdapter  tagResult `json:"noAdapter"`
			NoDevice   tagResult `json:"noDevice"`
			BadShader  tagResult `json:"badShader"`
			Timeout    tagResult `json:"timeout"`
			OpenOfNil  tagResult `json:"openOfNil"`
			Taken      tagResult `json:"taken"`
			NoCanvas   tagResult `json:"noCanvas"`
			WrongKind  tagResult `json:"wrongKind"`
			Fresh      bool      `json:"fresh"`
			Validation []string  `json:"validation"`
		}
		if err := json.Unmarshal(await(t, b, url.Values{"mode": {"unavailable"}}, time.Minute), &res); err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct {
			name, tag, route string
			got              tagResult
		}{
			{"no navigator.gpu", "WebGPUUnavailable", "Acquire.", res.NoGPU},
			{"no adapter", "WebGPUUnavailable", "Acquire.", res.NoAdapter},
			{"no device", "WebGPUDeviceFailed", "Acquire.", res.NoDevice},
			{"a shader the browser rejects", "ShaderModuleInvalid", "Acquire.", res.BadShader},
			{"no answer in time", "WebGPUDeviceFailed", "Acquire.", res.Timeout},
			{"no provider", "WebGPUUnavailable", "Open", res.OpenOfNil},
			{"canvas with a 2D context", "WebGPUContextUnavailable", "newSwapchain", res.Taken},
			{"view without a canvas", "WebGPUCanvasNotFound", "newSwapchain", res.NoCanvas},
			{"handle of another kind", "WebGPUSurfaceKindUnsupported", "newSwapchain", res.WrongKind},
		} {
			if tc.got.Tag != tc.tag || !strings.Contains(tc.got.Route, tc.route) || !strings.Contains(tc.got.Route, tc.tag) {
				t.Errorf("%s: got %s via %s, want %s via a route with %s", tc.name, tc.got.Tag, tc.got.Route, tc.tag, tc.route)
			}
		}
		if !res.Fresh {
			t.Error("a new canvas gave no swapchain")
		}
		if len(res.Validation) != 0 {
			t.Errorf("the browser reported validation errors: %v", res.Validation)
		}
	})
}

// TestBrowserDeviceLost: GPUDevice.destroy() takes the device away under a running renderer, as a
// driver reset would. The frame after it finds the device lost (GPUDeviceLost inside the renderer,
// the path Vulkan's loss takes), makes every resource again, and, because the browser's new device
// is not there yet and the UI goroutine does not wait for a promise, draws nothing and fails with
// nothing; the owner is told when the device has come, and the frame after that is the first frame
// again, pixel for pixel. A second loss less than ten seconds after the first ends with
// GPUDeviceLost.
func TestBrowserDeviceLost(t *testing.T) {
	eachBrowserWithWebGPU(t, func(t *testing.T, b webtest.Browser) {
		var res struct {
			First          bool      `json:"first"`
			DuringLoss     string    `json:"duringLoss"`
			Ready          int       `json:"ready"`
			DrewDuringLoss bool      `json:"drewDuringLoss"`
			AfterError     string    `json:"afterError"`
			AfterVsFirst   diffStats `json:"afterVsFirst"`
			SecondLoss     string    `json:"secondLoss"`
			Validation     []string  `json:"validation"`
		}
		if err := json.Unmarshal(await(t, b, url.Values{"mode": {"loss"}}, 2*time.Minute), &res); err != nil {
			t.Fatal(err)
		}
		t.Logf("%+v", res)
		if !res.First {
			t.Fatalf("the first frame failed: %s", res.AfterError)
		}
		if res.DuringLoss != "pending" {
			t.Errorf("the frame after the loss gave %q, want pending: nothing drawn, no error", res.DuringLoss)
		}
		if res.DrewDuringLoss {
			t.Error("the frame after the loss drew something: it should have found the device lost")
		}
		if res.Ready != 1 {
			t.Errorf("the owner was told %d times that the new device came, want 1", res.Ready)
		}
		if res.AfterError != "" {
			t.Fatalf("after the new device came: %s", res.AfterError)
		}
		if res.AfterVsFirst.Max != 0 {
			t.Errorf("the frame on the new device differs from the first by up to %d levels", res.AfterVsFirst.Max)
		}
		if res.SecondLoss != "GPUDeviceLost" {
			t.Errorf("the second loss within ten seconds gave %q, want GPUDeviceLost", res.SecondLoss)
		}
	})
}

type chooseResult struct {
	Error     string `json:"error"`
	Canvases  int    `json:"canvases"`
	WebGPU    bool   `json:"webgpu"`
	WebGL2    bool   `json:"webgl2"`
	Context2D bool   `json:"context2d"`
}

// TestBrowserRendererChoice: the engine's web renderer list is webgpu, webgl, then soft (SPEC
// §9.6). The canvas of the app tells which one was chosen, because a canvas has the one kind of
// context it was first asked for. Auto and webgpu choose WebGPU; webgl and soft are asked for by
// name; when the browser has no WebGPU (the test hides navigator.gpu) auto falls back to WebGL 2
// on a new canvas, leaving one canvas in the page, and to soft when WebGL 2 is gone too, while
// webgpu by name ends RunApp with WebGPUUnavailable.
func TestBrowserRendererChoice(t *testing.T) {
	tests := []struct {
		name  string
		query url.Values
		want  chooseResult
	}{
		{"auto", url.Values{}, chooseResult{Canvases: 1, WebGPU: true}},
		{"webgpu by name", url.Values{"GLUTTER_RENDERER": {"webgpu"}}, chooseResult{Canvases: 1, WebGPU: true}},
		{"webgl by name", url.Values{"GLUTTER_RENDERER": {"webgl"}}, chooseResult{Canvases: 1, WebGL2: true}},
		{"soft by name", url.Values{"GLUTTER_RENDERER": {"soft"}}, chooseResult{Canvases: 1, Context2D: true}},
		{"auto without WebGPU", url.Values{"poison": {"1"}}, chooseResult{Canvases: 1, WebGL2: true}},
		{"auto without WebGPU or WebGL 2", url.Values{"poison": {"1"}, "poisongl": {"1"}}, chooseResult{Canvases: 1, Context2D: true}},
		{"webgpu by name without WebGPU", url.Values{"GLUTTER_RENDERER": {"webgpu"}, "poison": {"1"}}, chooseResult{Error: "WebGPUUnavailable"}},
		{"gles on the web", url.Values{"GLUTTER_RENDERER": {"gles"}}, chooseResult{Error: "RendererUnavailable"}},
	}
	eachBrowserWithWebGPU(t, func(t *testing.T, b webtest.Browser) {
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
				if got != tc.want {
					t.Errorf("got %+v, want %+v", got, tc.want)
				}
			})
		}
	})
}

type switchResult struct {
	StartedOnWebGPU bool   `json:"startedOnWebGPU"`
	Canvases        int    `json:"canvases"`
	WebGPU          bool   `json:"webgpu"`
	WebGL2          bool   `json:"webgl2"`
	Context2D       bool   `json:"context2d"`
	Devices         int    `json:"devices"`
	Log             string `json:"log"`
}

// TestBrowserLastResortSwitchesToWebGL: WebGPU's last resort (SPEC §9.5, §9.6) in the real engine. The
// app starts on WebGPU; then its device is taken away twice within ten seconds (GPUDevice.destroy(), as in
// TestBrowserDeviceLost), or once with the browser giving no new device. The chooser closes the WebGPU
// renderer, puts a new canvas in place and draws the frame with WebGL 2: the page ends with one canvas
// whose context is WebGL 2, and the log says so.
func TestBrowserLastResortSwitchesToWebGL(t *testing.T) {
	tests := []struct {
		name  string
		query url.Values
	}{
		{"a second loss within ten seconds", url.Values{"mode": {"switch"}}},
		{"a new device that the browser will not give", url.Values{"mode": {"switch"}, "refuse": {"1"}}},
	}
	eachBrowserWithWebGPU(t, func(t *testing.T, b webtest.Browser) {
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				var res switchResult
				if err := json.Unmarshal(await(t, b, tc.query, 2*time.Minute), &res); err != nil {
					t.Fatal(err)
				}
				t.Logf("%+v", res)
				if !res.StartedOnWebGPU {
					t.Fatal("the app did not start on WebGPU")
				}
				if res.Canvases != 1 || !res.WebGL2 || res.WebGPU || res.Context2D {
					t.Errorf("the page ends with %d canvases, webgpu %v, webgl2 %v, 2d %v; want one canvas with WebGL 2",
						res.Canvases, res.WebGPU, res.WebGL2, res.Context2D)
				}
				for _, want := range []string{"renderer given up, switching to the next", "renderer=webgpu"} {
					if !strings.Contains(res.Log, want) {
						t.Errorf("the log does not say %q:\n%s", want, res.Log)
					}
				}
				if strings.Count(res.Log, "renderer chosen") != 2 {
					t.Errorf("the log should have two choices (WebGPU, then WebGL 2):\n%s", res.Log)
				}
			})
		}
	})
}

// TestBrowserLastResortNeverSwitchesAName: WebGPU asked for by name (GLUTTER_RENDERER=webgpu) is never
// replaced (SPEC §9.6): the second loss within ten seconds ends RunApp with GPUDeviceLost, as Vulkan's
// does, and the loader shows that in the page.
func TestBrowserLastResortNeverSwitchesAName(t *testing.T) {
	eachBrowserWithWebGPU(t, func(t *testing.T, b webtest.Browser) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		_, err := webtest.Await(ctx, webtest.ShotOptions{
			Dir: gpuwebSite(t), Browser: b, Width: 480, Height: 640, Timeout: 2 * time.Minute,
			Query: url.Values{"mode": {"switch"}, "GLUTTER_RENDERER": {"webgpu"}},
		}, "gpuweb")
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != "WebAppFailed" {
			t.Fatalf("got %v, want WebAppFailed: the app should end with the loss", err)
		}
		for _, field := range []string{"log", "shown"} {
			if text, _ := appErr.Metadata[field].(string); !strings.Contains(text, "GPUDeviceLost") {
				t.Errorf("%s of the failure is %q, want it to name GPUDeviceLost", field, text)
			}
		}
	})
}

// TestBrowserLoaderShowsAStartFailure: GLUTTER_RENDERER=webgpu on a browser without WebGPU (the test hides
// navigator.gpu) ends RunApp with WebGPUUnavailable (SPEC §9.6), and the loader puts the error into the
// page for the user, not only into the console (SPEC §23.1). The loader reports the text of the element it
// made.
func TestBrowserLoaderShowsAStartFailure(t *testing.T) {
	webtest.EachBrowser(t, func(t *testing.T, b webtest.Browser) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		_, err := webtest.Await(ctx, webtest.ShotOptions{
			Dir: gpuwebSite(t), Browser: b, Width: 480, Height: 640, Timeout: 2 * time.Minute,
			Query: url.Values{"mode": {"failstart"}, "GLUTTER_RENDERER": {"webgpu"}},
		}, "gpuweb")
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != "WebAppFailed" {
			t.Fatalf("got %v, want WebAppFailed", err)
		}
		shown, _ := appErr.Metadata["shown"].(string)
		for _, want := range []string{"The app stopped with an error", "exit code 1", "WebGPUUnavailable"} {
			if !strings.Contains(shown, want) {
				t.Errorf("the page shows %q, want it to say %q", shown, want)
			}
		}
	})
}
