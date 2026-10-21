package engine

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/platform/fake"
)

// namedRenderer is a fakeRenderer that names itself and its device.
type namedRenderer struct {
	*fakeRenderer
	name string
}

func (r *namedRenderer) Describe() string { return r.name }

// fakeNames are the names of the fake candidates, by the letter the tests use for them:
// a is "vulkan", b is "gles" and c is "soft".
var fakeNames = map[string]string{"a": "vulkan", "b": "gles", "c": "soft"}

// fakeCandidates are the renderers a, b and c, in that order. Each letter in failing
// makes its renderer fail the first frame with its own tag, "AFailed", "BFailed" or
// "CFailed". The renderers that were made are in made, in order of creation; the
// first letter of a renderer's name is its letter in the tests.
func fakeCandidates(failing ...string) (cands []rendererCandidate, made *[]*namedRenderer) {
	made = &[]*namedRenderer{}
	for _, key := range []string{"a", "b", "c"} {
		fails := false
		for _, f := range failing {
			fails = fails || f == key
		}
		name := fakeNames[key]
		cands = append(cands, rendererCandidate{Name: name, New: func(*Config, *slog.Logger) renderer {
			r := &namedRenderer{fakeRenderer: &fakeRenderer{}, name: key + ": device of " + name}
			if fails {
				r.err = errors.New(strings.ToUpper(key)+"Failed", "this renderer does not work")
			}
			*made = append(*made, r)
			return r
		}})
	}
	return cands, made
}

// logBuffer is a bytes.Buffer that the engine's goroutine can write while the test reads.
type logBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *logBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *logBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func recordingLog() (*slog.Logger, *logBuffer) {
	buf := &logBuffer{}
	return slog.New(slog.NewTextHandler(buf, nil)), buf
}

func TestChooser(t *testing.T) {
	handle := gpu.SurfaceHandle{Kind: gpu.SurfaceKindFake, Window: 1}
	tests := []struct {
		name    string
		want    string
		failing []string
		// wantUsed is the name of the renderer that drew the frame ("" when none did),
		// wantMade the names of the renderers that were made, wantTag the tag of the
		// error of Draw, wantLogs how many "not usable" lines the log has.
		wantUsed string
		wantMade string
		wantTag  string
		wantLogs int
	}{
		{name: "auto uses the first candidate", want: "auto", wantUsed: "a", wantMade: "a"},
		{name: "an empty value is auto", want: "", wantUsed: "a", wantMade: "a"},
		{name: "asked for by name", want: "b", wantUsed: "b", wantMade: "b"},
		{name: "auto falls back once and logs once", want: "auto", failing: []string{"a"}, wantUsed: "b", wantMade: "ab", wantLogs: 1},
		{name: "auto falls back twice and logs twice", want: "auto", failing: []string{"a", "b"}, wantUsed: "c", wantMade: "abc", wantLogs: 2},
		{name: "every candidate failing returns the last error", want: "auto", failing: []string{"a", "b", "c"}, wantMade: "abc", wantTag: "CFailed", wantLogs: 3},
		{name: "a name never falls back", want: "a", failing: []string{"a"}, wantMade: "a", wantTag: "AFailed"},
		{name: "a name skips the candidates before it", want: "c", failing: []string{"a"}, wantUsed: "c", wantMade: "c"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cands, made := fakeCandidates(tt.failing...)
			log, buf := recordingLog()
			c, err := newChooser(&Config{}, log, fakeNames[tt.want], cands)
			if err != nil {
				t.Fatalf("newChooser: %v", err)
			}
			for frame := 0; frame < 3; frame++ { // the choice is made once
				_, err = c.Draw(handle, 10, 10, nil, false)
				if tt.wantTag != "" {
					if appErr, ok := errors.FromError(err); !ok || appErr.Tag != tt.wantTag {
						t.Fatalf("frame %d: Draw = %v, want the tag %s", frame, err, tt.wantTag)
					}
					break
				}
				if err != nil {
					t.Fatalf("frame %d: %v", frame, err)
				}
			}
			var got string
			used := ""
			for _, r := range *made {
				got += r.name[:1]
				if calls, _ := r.snapshot(); len(calls) > 0 && r.err == nil {
					used = r.name[:1]
					if len(calls) != 3 {
						t.Errorf("renderer %s drew %d frames, want 3", used, len(calls))
					}
				}
			}
			if got != tt.wantMade || used != tt.wantUsed {
				t.Errorf("made %q and used %q, want %q and %q", got, used, tt.wantMade, tt.wantUsed)
			}
			if n := strings.Count(buf.String(), "renderer not usable"); n != tt.wantLogs {
				t.Errorf("the log has %d failed tries, want %d:\n%s", n, tt.wantLogs, buf)
			}
			if tt.wantLogs > 0 && !strings.Contains(buf.String(), "level=INFO") {
				t.Errorf("a failed try is not logged at Info level:\n%s", buf)
			}
			for _, r := range *made {
				_, closed := r.snapshot()
				if r.err != nil && closed != 1 {
					t.Errorf("the failed renderer %s was closed %d times, want once", r.name, closed)
				}
			}
			if tt.wantUsed != "" {
				if want := tt.wantUsed + ": device of " + fakeNames[tt.wantUsed]; c.Describe() != want {
					t.Errorf("Describe = %q, want %q", c.Describe(), want)
				}
				c.Close()
				for _, r := range *made {
					if _, closed := r.snapshot(); r.name[:1] == tt.wantUsed && closed != 1 {
						t.Errorf("Close closed the renderer %d times, want once", closed)
					}
				}
			}
		})
	}
}

func TestChooserDelegatesReleaseAndInvalidate(t *testing.T) {
	cands, made := fakeCandidates()
	c, err := newChooser(&Config{}, quiet(), "", cands)
	if err != nil {
		t.Fatal(err)
	}
	h := gpu.SurfaceHandle{Kind: gpu.SurfaceKindFake, Window: 7}
	c.Release(h)    // before the choice: nothing to release
	c.Invalidate(h) // the same
	if len(*made) != 0 {
		t.Fatalf("Release and Invalidate made a renderer")
	}
	if _, err := c.Draw(h, 4, 4, nil, false); err != nil {
		t.Fatal(err)
	}
	c.Invalidate(h)
	c.Release(h)
	r := (*made)[0]
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.invalidated) != 1 || len(r.released) != 1 {
		t.Errorf("the renderer was told %d invalidations and %d releases, want one each", len(r.invalidated), len(r.released))
	}
}

func TestNewChooserErrors(t *testing.T) {
	cands, _ := fakeCandidates()
	tests := []struct {
		name  string
		want  string
		cands []rendererCandidate
		tag   string
		route string
	}{
		{"an unknown value", "metal", cands, "RendererUnknown", "newChooser.RendererUnknown"},
		{"a value with another case", "GLES", cands, "RendererUnknown", "newChooser.RendererUnknown"},
		{"a known renderer this OS does not have", "webgpu", cands, "RendererUnavailable", "newChooser.RendererUnavailable"},
		{"a known renderer that is not in the list", "webgl", cands, "RendererUnavailable", "newChooser.RendererUnavailable"},
		{"a known renderer when the list is empty", "gles", nil, "RendererUnavailable", "newChooser.RendererUnavailable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newChooser(&Config{}, quiet(), tt.want, tt.cands)
			assertRoute(t, err, tt.tag, tt.route)
		})
	}
}

func TestDrawWithNoCandidates(t *testing.T) {
	c, err := newChooser(&Config{}, quiet(), "auto", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Draw(gpu.SurfaceHandle{}, 4, 4, nil, false)
	assertRoute(t, err, "RendererUnavailable", "chooser.Draw.RendererUnavailable")
}

func TestRunChoosesTheRenderer(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		opt  string
		// wantMade is the renderers that were made, in order.
		wantMade string
	}{
		{"nothing asked for", nil, "", "a"},
		{"the environment variable", map[string]string{RendererEnv: "gles"}, "", "b"},
		{"the option overrides the environment variable", map[string]string{RendererEnv: "gles"}, "soft", "c"},
		{"auto in the option overrides a name in the environment", map[string]string{RendererEnv: "gles"}, "auto", "a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t, tt.env)
			cands, made := fakeCandidates()
			h.cfg.newRenderer = nil
			h.cfg.candidates = func() []rendererCandidate { return cands }
			h.cfg.Renderer = tt.opt
			log, buf := recordingLog()
			h.cfg.Logger = log
			h.c.Inject(metrics(10, 10))
			h.c.Inject(fake.FrameEvent{View: 0, FrameTime: time.Millisecond})
			done := h.run(t)
			waitFor(t, func() bool { return strings.Contains(buf.String(), "first frame presented") })
			h.stopEngine(t)
			if err := result(t, done); err != nil {
				t.Fatal(err)
			}
			var got string
			for _, r := range *made {
				got += r.name[:1]
			}
			if got != tt.wantMade {
				t.Errorf("made %q, want %q", got, tt.wantMade)
			}
			// The log names the renderer and its device (SPEC §9.6).
			want := `renderer="` + tt.wantMade + `: device of ` + fakeNames[tt.wantMade] + `"`
			if n := strings.Count(buf.String(), want); n < 2 { // "renderer chosen" and "first frame presented"
				t.Errorf("the log names the renderer %d times, want it in the choice and in the first frame:\n%s", n, buf)
			}
		})
	}
}

func TestRunRendererErrors(t *testing.T) {
	tests := []struct {
		name  string
		env   map[string]string
		opt   string
		tag   string
		route string
	}{
		{"an unknown value in the option", nil, "metal", "RendererUnknown", "Run->newChooser.RendererUnknown"},
		{"an unknown value in the environment", map[string]string{RendererEnv: "metal"}, "", "RendererUnknown", "Run->newChooser.RendererUnknown"},
		{"a known renderer the OS does not have", nil, "webgpu", "RendererUnavailable", "Run->newChooser.RendererUnavailable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t, tt.env)
			cands, made := fakeCandidates()
			h.cfg.newRenderer = nil
			h.cfg.candidates = func() []rendererCandidate { return cands }
			h.cfg.Renderer = tt.opt
			assertRoute(t, Run(context.Background(), h.cfg), tt.tag, tt.route)
			if len(*made) != 0 || h.c.closed {
				t.Error("the error must come before the display and the renderers are touched")
			}
		})
	}
}

func TestRunWithEveryRendererFailing(t *testing.T) {
	h := newHarness(t, nil)
	cands, made := fakeCandidates("a", "b", "c")
	h.cfg.newRenderer = nil
	h.cfg.candidates = func() []rendererCandidate { return cands }
	h.c.Inject(metrics(10, 10))
	h.c.Inject(fake.FrameEvent{View: 0})
	assertRoute(t, result(t, h.run(t)), "CFailed", "CFailed")
	if len(*made) != 3 {
		t.Errorf("%d renderers were tried, want 3", len(*made))
	}
}
