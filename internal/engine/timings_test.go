package engine

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/ui"
)

func TestFirstFrameAndFrameStatsAreLogged(t *testing.T) {
	var buf bytes.Buffer
	e := newWindowsEngine(t)
	e.h.log = slog.New(slog.NewTextHandler(&buf, nil))
	e.h.runStart = time.Now().Add(-120 * time.Millisecond)
	e.h.frameStats = true
	e.size(0, 400, 300)
	e.renderAll(map[ui.ViewID]*ui.Scene{0: {}})

	e.h.OnFrame(0, time.Millisecond)
	e.h.OnFrame(0, 2*time.Millisecond)
	out := buf.String()
	if n := strings.Count(out, "first frame presented"); n != 1 {
		t.Fatalf("the first frame was logged %d times:\n%s", n, out)
	}
	if !strings.Contains(out, "since_run=") {
		t.Errorf("the log has no since_run:\n%s", out)
	}
	if strings.Contains(out, "msg=frames") {
		t.Errorf("the statistics were logged before a second passed:\n%s", out)
	}

	// A second later the frames are summed and logged, and the sums start again.
	e.h.stats.since = time.Now().Add(-1100 * time.Millisecond)
	e.h.OnFrame(0, 3*time.Millisecond)
	out = buf.String()
	if !strings.Contains(out, "msg=frames") || !strings.Contains(out, "fps=") || !strings.Contains(out, "build_avg=") || !strings.Contains(out, "over_16ms=0") {
		t.Errorf("no statistics line:\n%s", out)
	}
	if e.h.stats.frames != 0 {
		t.Errorf("the sums were not reset: %+v", e.h.stats)
	}
}

func TestFrameStatsAreOffByDefault(t *testing.T) {
	var buf bytes.Buffer
	e := newWindowsEngine(t)
	e.h.log = slog.New(slog.NewTextHandler(&buf, nil))
	e.size(0, 400, 300)
	e.renderAll(map[ui.ViewID]*ui.Scene{0: {}})
	e.h.OnFrame(0, time.Millisecond)
	if strings.Contains(buf.String(), "msg=frames") {
		t.Errorf("statistics without the variable:\n%s", buf.String())
	}
}

func TestFrameLogsNameTheRendererAndItsDevice(t *testing.T) {
	var buf bytes.Buffer
	e := newWindowsEngine(t)
	e.h.log = slog.New(slog.NewTextHandler(&buf, nil))
	e.h.frameStats = true
	e.h.r = &namedRenderer{fakeRenderer: e.r, name: "gles: llvmpipe (LLVM 19.1.7, 256 bits)"}
	e.size(0, 400, 300)
	e.renderAll(map[ui.ViewID]*ui.Scene{0: {}})
	e.h.OnFrame(0, time.Millisecond)
	e.h.stats.since = time.Now().Add(-1100 * time.Millisecond)
	e.h.OnFrame(0, 2*time.Millisecond)
	out := buf.String()
	if n := strings.Count(out, `renderer="gles: llvmpipe (LLVM 19.1.7, 256 bits)"`); n != 2 {
		t.Errorf("the renderer is named %d times, want in the first frame and in the statistics:\n%s", n, out)
	}
}

// TestSlowFramesAreLoggedOnTheirOwn: with the statistics on, a frame over a 60th of a second is
// logged by itself with its build and raster times, so that what made it slow (a page built for the
// first time, a glyph atlas, a pipeline) can be told from the average of its second (step 21.7).
func TestSlowFramesAreLoggedOnTheirOwn(t *testing.T) {
	tests := []struct {
		name          string
		build, raster time.Duration
		want          bool
	}{
		{"a fast frame", 2 * time.Millisecond, 3 * time.Millisecond, false},
		{"a slow build", 40 * time.Millisecond, 3 * time.Millisecond, true},
		{"a slow raster", 2 * time.Millisecond, 30 * time.Millisecond, true},
		{"both a little slow", 10 * time.Millisecond, 10 * time.Millisecond, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			e := newWindowsEngine(t)
			e.h.log = slog.New(slog.NewTextHandler(&buf, nil))
			e.h.frameStats = true
			e.h.firstLogged = true
			e.h.clock.buildStart = 0
			e.h.clock.rasterStart = tc.build
			e.h.clock.rasterFinish = tc.build + tc.raster
			e.h.noteFrame()
			out := buf.String()
			if got := strings.Contains(out, `msg="slow frame"`); got != tc.want {
				t.Fatalf("slow frame logged = %v, want %v:\n%s", got, tc.want, out)
			}
			if tc.want && (!strings.Contains(out, "build="+tc.build.String()) || !strings.Contains(out, "raster="+tc.raster.String())) {
				t.Errorf("the line does not carry the times %v and %v:\n%s", tc.build, tc.raster, out)
			}
		})
	}
}
