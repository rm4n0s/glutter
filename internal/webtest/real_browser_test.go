//go:build glutter_browser && linux

package webtest

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// processesUsing lists the processes whose command line mentions the directory.
func processesUsing(dir string) []string {
	var found []string
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join("/proc", e.Name(), "cmdline"))
		if err == nil && strings.Contains(string(data), dir) {
			found = append(found, e.Name()+": "+strings.ReplaceAll(string(data), "\x00", " "))
		}
	}
	return found
}

// TestRealBrowsersPostResultsAndAreKilled starts each real browser on a page of the harness's
// server, which posts what the browser is to the results endpoint; then it stops the browser and
// looks for processes it left behind.
func TestRealBrowsersPostResultsAndAreKilled(t *testing.T) {
	EachBrowser(t, func(t *testing.T, b Browser) {
		srv := newTestServer(t)
		srv.AddPage("/probe.html", []byte(`<!doctype html><title>probe</title><script>
fetch("/__glutter/results", {method: "POST", body: JSON.stringify({kind: "probe", message: navigator.userAgent,
  data: {innerWidth: innerWidth, innerHeight: innerHeight, dpr: devicePixelRatio, visibility: document.visibilityState,
         platform: navigator.platform, languages: navigator.languages}})});
</script>`))
		var output strings.Builder
		p, err := Launch(b, srv.URL+"/probe.html", LaunchOptions{Width: 600, Height: 500, ScaleFactor: 2, Output: &output})
		if err != nil {
			t.Fatal(err)
		}
		profile := ""
		for _, a := range p.cmd.Args {
			if v, ok := strings.CutPrefix(a, "--user-data-dir="); ok {
				profile = filepath.Dir(v)
			}
			if strings.Contains(a, "glutter-browser-") && !strings.HasPrefix(a, "--") {
				profile = filepath.Dir(a)
			}
		}
		stopped := false
		defer func() {
			if !stopped {
				p.Stop()
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		res, err := srv.WaitResult(ctx)
		if err != nil {
			t.Fatalf("no result from %s: %v\n%s", b, err, output.String())
		}
		want := map[Browser]string{Chromium: "Chrome", Firefox: "Firefox"}[b]
		if res.Kind != "probe" || !strings.Contains(res.Message, want) {
			t.Errorf("the page says it is %q, want a %s", res.Message, want)
		}
		var data struct {
			DPR        float64  `json:"dpr"`
			Visibility string   `json:"visibility"`
			Platform   string   `json:"platform"`
			Languages  []string `json:"languages"`
		}
		if err := json.Unmarshal(res.Data, &data); err != nil {
			t.Fatal(err)
		}
		if data.DPR != 2 || data.Visibility != "visible" || !strings.HasPrefix(data.Platform, "Linux") {
			t.Errorf("the browser reports %+v, want a device pixel ratio of 2, a visible page on Linux", data)
		}
		p.Stop()
		stopped = true
		deadline := time.Now().Add(5 * time.Second)
		var left []string
		for time.Now().Before(deadline) {
			if left = processesUsing(profile); len(left) == 0 {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		if profile == "" {
			t.Fatal("could not find the profile directory in the command line")
		}
		if len(left) > 0 {
			t.Errorf("processes of %s are still running after Stop:\n%s", b, strings.Join(left, "\n"))
		}
		if _, err := os.Stat(profile); !os.IsNotExist(err) {
			t.Errorf("the profile is still there (err %v)", err)
		}
	})
}
