//go:build glutter_browser

package canvas2d_test

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/issues"
	"github.com/rm4n0s/glutter/internal/webtest"
	"github.com/rm4n0s/glutter/internal/webtool"
)

// These tests run the canvas2d renderer, the software renderer into a 2D canvas, in real browsers
// (SPEC §17 "Browser tests"). The program is internal/gpu/canvas2d/canvaswebtest: its mode says
// what it does, and it posts what it saw.

var (
	canvaswebOnce sync.Once
	canvaswebDir  string
	canvaswebErr  error
)

func canvaswebSite(t *testing.T) string {
	t.Helper()
	canvaswebOnce.Do(func() {
		if canvaswebDir, canvaswebErr = os.MkdirTemp("", "glutter-web-canvastest-"); canvaswebErr != nil {
			return
		}
		_, canvaswebErr = webtool.Build(context.Background(), webtool.BuildOptions{
			Debug:   true,
			Package: "./internal/gpu/canvas2d/canvaswebtest", Dir: webtest.ModuleRoot(t), OutDir: canvaswebDir,
		})
	})
	if canvaswebErr != nil {
		t.Fatalf("building the canvas test program for the web: %v", canvaswebErr)
	}
	return canvaswebDir
}

func TestMain(m *testing.M) {
	code := m.Run()
	if canvaswebDir != "" {
		_ = os.RemoveAll(canvaswebDir) // a temporary directory: a leftover is harmless
	}
	os.Exit(code)
}

// TestBrowserKeepsTheMipmapsOfAPicture runs the reproduction of Fishball's issue of step 49.1
// (internal/issues, soft-mipmaps) with the canvas2d renderer in each browser: a spinner turns over a
// picture that is drawn at half its size. The picture's mipmap chain is made once and kept across
// the frames, and the canvas shows each frame. Before the step every frame made the chain again.
func TestBrowserKeepsTheMipmapsOfAPicture(t *testing.T) {
	webtest.EachBrowser(t, func(t *testing.T, b webtest.Browser) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		got, err := webtest.Await(ctx, webtest.ShotOptions{Dir: canvaswebSite(t), Browser: b, Query: url.Values{"mode": {"issue"}, "issue": {issues.SoftMipmapsName}}, Width: 800, Height: 600, Timeout: 2 * time.Minute}, "canvasweb")
		if err != nil {
			t.Fatalf("Await in %s: %v", b, err)
		}
		var res struct {
			Lines []string `json:"lines"`
			Shown []string `json:"shown"`
			Error string   `json:"error"`
		}
		if err := json.Unmarshal(got.Data, &res); err != nil {
			t.Fatal(err)
		}
		if res.Error != "" {
			t.Fatal(res.Error)
		}
		for _, s := range res.Shown {
			t.Error(s)
		}
		rep, err := issues.ParseSoftMipmaps(res.Lines)
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range rep.Lines() {
			t.Log(line)
		}
		for _, problem := range rep.Problems() {
			t.Error(problem)
		}
	})
}
