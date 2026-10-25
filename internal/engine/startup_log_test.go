package engine

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
)

// reportingRenderer is a fake renderer that says how its start went.
type reportingRenderer struct {
	*fakeRenderer
	report gpu.StartupReport
}

func (r *reportingRenderer) StartupReport() gpu.StartupReport { return r.report }

func TestChooserLogsTheParts(t *testing.T) {
	handle := gpu.SurfaceHandle{Kind: gpu.SurfaceKindFake, Window: 1}
	report := gpu.StartupReport{Context: 30 * time.Millisecond, Shaders: 20 * time.Millisecond, Built: 3, Cached: 5, Refused: 1, CacheBytes: 99}
	tests := []struct {
		name     string
		env      string
		firstErr bool
		reports  bool
		want     []string
		unwanted []string
	}{
		{"without the statistics nothing is added", "", false, true, nil, []string{"stage="}},
		{"every part of a renderer that reports", "1", true, true, []string{
			`stage="first frame of the renderer" renderer=second`,
			`stage="GPU context or device" renderer=second took=30ms`,
			`stage=shaders renderer=second took=20ms built=3 from_cache=5 refused=1 pipeline_cache_bytes=99`,
			`stage="frame without the above" renderer=second`,
		}, nil},
		{"a renderer that does not report has the whole first frame", "1", true, false, []string{`stage="first frame of the renderer" renderer=second`}, []string{"GPU context", "stage=shaders"}},
		{"a renderer that failed is logged with its error and time", "1", true, true, []string{
			`stage="renderer tried" renderer=first`, "err.Tag=" + tagFirstRendererFailed, `stage="first frame of the renderer" renderer=second`,
		}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log, buf := recordingLog()
			cfg := &Config{Getenv: func(k string) string {
				if k == FrameStatsEnv {
					return tt.env
				}
				return ""
			}}
			mk := func(fail bool, reports bool) func(*Config, *slog.Logger) renderer {
				return func(*Config, *slog.Logger) renderer {
					f := &fakeRenderer{}
					if fail {
						f.err = errors.New(tagFirstRendererFailed, "this renderer does not work")
					}
					if reports {
						return &reportingRenderer{fakeRenderer: f, report: report}
					}
					return f
				}
			}
			cands := []rendererCandidate{{Name: "first", New: mk(tt.firstErr, false)}, {Name: "second", New: mk(false, tt.reports)}}
			c, err := newChooser(cfg, log, "auto", cands)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := c.Draw(handle, 10, 10, nil, false); err != nil {
				t.Fatal(err)
			}
			out := buf.String()
			for _, w := range tt.want {
				if !strings.Contains(out, w) {
					t.Errorf("the log lacks %q:\n%s", w, out)
				}
			}
			for _, u := range tt.unwanted {
				if strings.Contains(out, u) {
					t.Errorf("the log has %q:\n%s", u, out)
				}
			}
		})
	}
}

func TestNewShaderCache(t *testing.T) {
	tmp := t.TempDir()
	tests := []struct {
		name    string
		cfgDir  string
		env     map[string]string
		wantDir string // "" for no cache
	}{
		{"the config's directory", filepath.Join(tmp, "cfg"), map[string]string{"XDG_CACHE_HOME": "/x", "LocalAppData": "/x"}, filepath.Join(tmp, "cfg")},
		// XDG_CACHE_HOME is the base on Linux and LocalAppData on Windows (defaultShaderCacheDir).
		{"the environment's", "", map[string]string{"XDG_CACHE_HOME": tmp, "LocalAppData": tmp}, filepath.Join(tmp, "glutter", "shaders")},
		{"turned off beats the config", filepath.Join(tmp, "cfg"), map[string]string{"GLUTTER_SHADER_CACHE": "0"}, ""},
		{"no place for it", "", map[string]string{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log, _ := recordingLog()
			c := newShaderCache(&Config{ShaderCacheDir: tt.cfgDir}, func(k string) string { return tt.env[k] }, log)
			switch {
			case tt.wantDir == "" && c != nil:
				t.Errorf("a cache in %q, want none", c.Dir())
			case tt.wantDir != "" && (c == nil || c.Dir() != tt.wantDir):
				t.Errorf("cache = %v, want %q", c, tt.wantDir)
			}
		})
	}
}

func TestShaderCacheFailureIsLoggedOnce(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	log, buf := recordingLog()
	c := newShaderCache(&Config{ShaderCacheDir: filepath.Join(blocker, "shaders")}, func(string) string { return "" }, log)
	c.Store("a", []byte("1"))
	c.Store("b", []byte("2"))
	out := buf.String()
	if n := strings.Count(out, "the shader cache cannot be used"); n != 1 {
		t.Errorf("logged %d times, want once:\n%s", n, out)
	}
	if !strings.Contains(out, "tag=ShaderCacheWriteFailed") {
		t.Errorf("the log lacks the tag:\n%s", out)
	}
}

const tagFirstRendererFailed = "FirstRendererFailed"
