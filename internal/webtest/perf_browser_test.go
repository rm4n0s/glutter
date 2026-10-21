//go:build glutter_browser

package webtest

import (
	"context"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/webtool"
)

// TestBrowserPerfMeasuresTheCounter runs the measurement that scripts/perf-web.sh makes, on the
// counter in each browser: the starts are timed (the later ones with app.wasm in the cache), the
// idle counter asks for no frame and arms no timer while nothing happens, the CPU time of the
// browser is read, and a click on the floating action button is answered by a frame.
func TestBrowserPerfMeasuresTheCounter(t *testing.T) {
	dir := t.TempDir()
	if _, err := webtool.Build(context.Background(), webtool.BuildOptions{Package: "./examples/counter", Dir: ModuleRoot(t), OutDir: dir, Release: true}); err != nil {
		t.Fatal(err)
	}
	EachBrowser(t, func(t *testing.T, b Browser) {
		t.Run("idle", func(t *testing.T) {
			res, err := Perf(context.Background(), PerfOptions{
				Dir: dir, Browser: b, Width: 480, Height: 640, Loads: 2, Settle: time.Second, Measure: 3 * time.Second, Timeout: 90 * time.Second,
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(res.FirstFrames) != 2 || res.FirstFrames[0] <= 0 || res.FirstFrames[1] <= 0 {
				t.Errorf("FirstFrames = %v, want two times", res.FirstFrames)
			}
			if res.Renderer == "" {
				t.Error("the renderer of the first frame was not read")
			}
			if res.RAFCalls != 0 || res.TimerCalls != 0 {
				t.Errorf("the idle counter asked for %d animation frames and %d timers, want none", res.RAFCalls, res.TimerCalls)
			}
			if res.Window < 2*time.Second || res.CPU < 0 {
				t.Errorf("window %v, CPU %v", res.Window, res.CPU)
			}
		})
		t.Run("click", func(t *testing.T) {
			res, err := Perf(context.Background(), PerfOptions{
				Dir: dir, Browser: b, Width: 480, Height: 640, Loads: 1, Timeout: 90 * time.Second,
				Clicks: [][2]float64{{436, 596}}, ClickGap: 600 * time.Millisecond,
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(res.Clicks) != 1 {
				t.Fatalf("Clicks = %v, want one", res.Clicks)
			}
			if c := res.Clicks[0]; c.Latency <= 0 || len(c.Frames) == 0 || c.Max <= 0 {
				t.Errorf("the click was not answered by a frame: %+v", c)
			}
		})
		t.Run("blank", func(t *testing.T) {
			res, err := Perf(context.Background(), PerfOptions{Browser: b, Blank: true, Settle: time.Second, Measure: 2 * time.Second, Timeout: 60 * time.Second})
			if err != nil {
				t.Fatal(err)
			}
			if res.Window < time.Second || len(res.FirstFrames) != 0 {
				t.Errorf("window %v, first frames %v", res.Window, res.FirstFrames)
			}
		})
	})
}
