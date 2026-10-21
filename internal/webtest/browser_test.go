//go:build linux

package webtest

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
)

// fakeBrowser writes a program that stands in for a browser: it records its arguments and
// environment, starts a child that sleeps (as browsers start helpers), and sleeps. It returns
// the path of the program and of the file it records to.
func fakeBrowser(t *testing.T) (program, record string) {
	t.Helper()
	dir := t.TempDir()
	program, record = filepath.Join(dir, "browser"), filepath.Join(dir, "record")
	script := `#!/bin/sh
{
  for a in "$@"; do echo "arg:$a"; done
  echo "env:HOME:$HOME"
  echo "env:DISPLAY:${DISPLAY-unset}"
  echo "env:WAYLAND_DISPLAY:${WAYLAND_DISPLAY-unset}"
  echo "pgid:$(ps -o pgid= -p $$ | tr -d ' ')"
} > "` + record + `"
sleep 300 &
echo "child:$!" >> "` + record + `"
sleep 300
`
	if err := os.WriteFile(program, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return program, record
}

// waitForFile waits for the file to hold what ready accepts, and returns its content.
func waitForFile(t *testing.T, path string, ready func(string) bool) string {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if data, err := os.ReadFile(path); err == nil && ready(string(data)) {
			return string(data)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("%s never held what the test waits for", path)
	return ""
}

func alive(pid int) bool { return syscall.Kill(pid, 0) == nil }

func recordValue(record, key string) string {
	for _, line := range strings.Split(record, "\n") {
		if v, ok := strings.CutPrefix(line, key+":"); ok {
			return v
		}
	}
	return ""
}

func TestLaunchChromiumFlagsAndCleanup(t *testing.T) {
	t.Setenv("DISPLAY", ":99")
	t.Setenv("WAYLAND_DISPLAY", "wayland-99")
	t.Setenv(ExtraFlagsEnv, "--extra-one --extra-two")
	program, record := fakeBrowser(t)
	p, err := Launch(Chromium, "http://127.0.0.1:1/page", LaunchOptions{Path: program, Width: 700, Height: 500})
	if err != nil {
		t.Fatal(err)
	}
	stopped := false
	defer func() {
		if !stopped {
			p.Stop()
		}
	}()
	rec := waitForFile(t, record, func(s string) bool { return strings.Contains(s, "child:") })
	for _, want := range []string{"arg:--headless=new", "arg:--window-size=700,500", "arg:--force-device-scale-factor=1", "arg:--extra-one", "arg:--extra-two", "arg:http://127.0.0.1:1/page"} {
		if !strings.Contains(rec, want+"\n") {
			t.Errorf("the browser was not given %s; it got:\n%s", want, rec)
		}
	}
	if strings.Contains(rec, "--no-sandbox") {
		t.Error("--no-sandbox is on without being asked for")
	}
	var profile string
	for _, line := range strings.Split(rec, "\n") {
		if v, ok := strings.CutPrefix(line, "arg:--user-data-dir="); ok {
			profile = v
		}
	}
	if profile == "" || !strings.Contains(profile, "glutter-browser-") {
		t.Fatalf("the profile is %q, want a temporary directory of the harness", profile)
	}
	if _, err := os.Stat(profile); err != nil {
		t.Errorf("the profile does not exist while the browser runs: %v", err)
	}
	if home := recordValue(rec, "env:HOME"); !strings.HasPrefix(home, filepath.Dir(profile)) {
		t.Errorf("HOME is %q, want a directory inside the temporary profile %q", home, filepath.Dir(profile))
	}
	if d, w := recordValue(rec, "env:DISPLAY"), recordValue(rec, "env:WAYLAND_DISPLAY"); d != "unset" || w != "unset" {
		t.Errorf("the browser can see the user's display (DISPLAY=%s WAYLAND_DISPLAY=%s)", d, w)
	}
	child, err := strconv.Atoi(recordValue(rec, "child"))
	if err != nil || !alive(child) {
		t.Fatalf("the helper process of the browser is not running (child %q)", recordValue(rec, "child"))
	}
	pgid, _ := strconv.Atoi(recordValue(rec, "pgid"))
	if pgid == syscall.Getpgrp() || pgid == 0 {
		t.Errorf("the browser is in process group %d, the test's own is %d: it must have a group of its own", pgid, syscall.Getpgrp())
	}

	p.Stop()
	stopped = true
	select {
	case <-p.Done():
	default:
		t.Error("Done is not closed after Stop")
	}
	deadline := time.Now().Add(5 * time.Second)
	for alive(child) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if alive(child) {
		_ = syscall.Kill(child, syscall.SIGKILL) // cleaning up after the failure below
		t.Error("the helper process of the browser survived Stop")
	}
	if _, err := os.Stat(profile); !os.IsNotExist(err) {
		t.Errorf("the profile is still there after Stop (err %v)", err)
	}
	p.Stop() // twice is fine
}

func TestLaunchFirefoxProfile(t *testing.T) {
	program, record := fakeBrowser(t)
	p, err := Launch(Firefox, "http://127.0.0.1:1/", LaunchOptions{Path: program})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Stop()
	rec := waitForFile(t, record, func(s string) bool { return strings.Contains(s, "child:") })
	var profile string
	for _, line := range strings.Split(rec, "\n") {
		if strings.HasPrefix(line, "arg:") && strings.Contains(line, "glutter-browser-") {
			profile = strings.TrimPrefix(line, "arg:")
		}
	}
	for _, want := range []string{"arg:--headless\n", "arg:--no-remote\n", "arg:--new-instance\n", "arg:--window-size=800,600\n"} {
		if !strings.Contains(rec, want) {
			t.Errorf("Firefox was not given %q; it got:\n%s", strings.TrimSpace(want), rec)
		}
	}
	prefs, err := os.ReadFile(filepath.Join(profile, "user.js"))
	if err != nil {
		t.Fatalf("no user.js in the profile %q: %v", profile, err)
	}
	for _, want := range []string{`user_pref("browser.shell.checkDefaultBrowser", false);`, `user_pref("datareporting.policy.dataSubmissionEnabled", false);`, `user_pref("layout.css.devPixelsPerPx", "1");`} {
		if !strings.Contains(string(prefs), want) {
			t.Errorf("user.js lacks %s", want)
		}
	}
}

func TestLaunchScaleFactor(t *testing.T) {
	for _, b := range []Browser{Chromium, Firefox} {
		t.Run(string(b), func(t *testing.T) {
			program, record := fakeBrowser(t)
			p, err := Launch(b, "http://127.0.0.1:1/", LaunchOptions{Path: program, ScaleFactor: 1.5})
			if err != nil {
				t.Fatal(err)
			}
			defer p.Stop()
			rec := waitForFile(t, record, func(s string) bool { return strings.Contains(s, "child:") })
			if b == Chromium {
				if !strings.Contains(rec, "arg:--force-device-scale-factor=1.5\n") {
					t.Errorf("Chromium was not given the scale factor:\n%s", rec)
				}
				return
			}
			for _, line := range strings.Split(rec, "\n") {
				if profile, ok := strings.CutPrefix(line, "arg:"); ok && strings.Contains(profile, "glutter-browser-") {
					prefs, err := os.ReadFile(filepath.Join(profile, "user.js"))
					if err != nil || !strings.Contains(string(prefs), `user_pref("layout.css.devPixelsPerPx", "1.5");`) {
						t.Errorf("user.js does not set the scale factor (err %v):\n%s", err, prefs)
					}
				}
			}
		})
	}
}

func TestLaunchEndsAtTheLifetime(t *testing.T) {
	program, _ := fakeBrowser(t)
	start := time.Now()
	p, err := Launch(Chromium, "http://127.0.0.1:1/", LaunchOptions{Path: program, Lifetime: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Stop()
	select {
	case <-p.Done():
	case <-time.After(15 * time.Second):
		t.Fatal("the browser outlived its lifetime of one second")
	}
	if time.Since(start) < 900*time.Millisecond {
		t.Errorf("the browser ended after %v, before its lifetime", time.Since(start))
	}
}

func TestLocate(t *testing.T) {
	tests := []struct {
		name    string
		browser Browser
		have    map[string]string
		want    string
		failed  bool
	}{
		{"first name wins", Chromium, map[string]string{"chromium-browser": "/a/chromium-browser", "chromium": "/b/chromium"}, "/a/chromium-browser", false},
		{"second name", Chromium, map[string]string{"chromium": "/b/chromium"}, "/b/chromium", false},
		{"chrome", Chromium, map[string]string{"google-chrome": "/c/chrome"}, "/c/chrome", false},
		{"firefox", Firefox, map[string]string{"firefox": "/f/firefox"}, "/f/firefox", false},
		{"none", Chromium, map[string]string{"firefox": "/f/firefox"}, "", true},
		{"not a browser", Browser("lynx"), map[string]string{"lynx": "/l/lynx"}, "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Locate(tc.browser, func(name string) (string, error) {
				if p, ok := tc.have[name]; ok {
					return p, nil
				}
				return "", os.ErrNotExist
			})
			if tc.failed {
				appErr, ok := errors.FromError(err)
				if !ok || appErr.Tag != "BrowserNotFound" || !appErr.HasRoute("Locate.BrowserNotFound") {
					t.Fatalf("Locate = %v, want BrowserNotFound via Locate", err)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Errorf("Locate = %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}

func TestParseBrowser(t *testing.T) {
	for _, name := range []string{"chromium", "firefox"} {
		if b, err := ParseBrowser(name); err != nil || string(b) != name {
			t.Errorf("ParseBrowser(%q) = %q, %v", name, b, err)
		}
	}
	_, err := ParseBrowser("safari")
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "BrowserUnknown" || !appErr.HasRoute("ParseBrowser.BrowserUnknown") {
		t.Errorf("ParseBrowser(safari) = %v, want BrowserUnknown via ParseBrowser", err)
	}
}

func TestLaunchFailures(t *testing.T) {
	t.Run("no timeout program", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir()) // nothing in it: timeout is not found
		program, _ := fakeBrowser(t)
		_, err := Launch(Chromium, "http://127.0.0.1:1/", LaunchOptions{Path: program})
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != "BrowserLaunchFailed" || !appErr.HasRoute("Launch.BrowserLaunchFailed") {
			t.Fatalf("Launch = %v, want BrowserLaunchFailed via Launch", err)
		}
	})
	t.Run("browser not installed", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		_, err := Launch(Chromium, "http://127.0.0.1:1/", LaunchOptions{})
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != "BrowserNotFound" || !appErr.HasRoute("Launch->Locate.BrowserNotFound") {
			t.Fatalf("Launch = %v, want BrowserNotFound via Launch->Locate", err)
		}
	})
	t.Run("unknown browser", func(t *testing.T) {
		program, _ := fakeBrowser(t)
		_, err := Launch(Browser("lynx"), "http://127.0.0.1:1/", LaunchOptions{Path: program})
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != "BrowserUnknown" || !appErr.HasRoute("Launch.BrowserUnknown") {
			t.Fatalf("Launch = %v, want BrowserUnknown via Launch", err)
		}
	})
}

func TestShotWithAFakeBrowser(t *testing.T) {
	if _, err := os.Stat("/usr/bin/curl"); err != nil {
		t.Skip("curl is not installed")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("app"), 0o644); err != nil {
		t.Fatal(err)
	}
	png := filepath.Join(dir, "shot.png")
	if err := os.WriteFile(png, tinyPNG, 0o644); err != nil {
		t.Fatal(err)
	}
	program := filepath.Join(dir, "browser")
	// The "browser" opens the frame page, then posts a PNG to the server like the engine does.
	script := `#!/bin/sh
for last; do :; done
base="${last%/__glutter/*}"
curl -s "$last" -o "` + dir + `/frame.html"
curl -s -X POST --data-binary @"` + png + `" "$base/__glutter/screenshot"
sleep 300
`
	if err := os.WriteFile(program, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	got, err := Shot(ctx, ShotOptions{
		Dir: dir, Browser: Chromium, BrowserPath: program, Width: 321, Height: 123,
		Query:   map[string][]string{"GLUTTER_DEV_TAPS": {"3"}},
		Resizes: []Resize{{After: time.Second, Width: 10, Height: 20}},
	})
	if err != nil {
		t.Fatalf("Shot: %v", err)
	}
	if string(got) != string(tinyPNG) {
		t.Errorf("Shot returned %q, want the PNG the browser posted", got)
	}
	frame, err := os.ReadFile(filepath.Join(dir, "frame.html"))
	if err != nil {
		t.Fatalf("the browser did not get the frame page: %v", err)
	}
	for _, want := range []string{"width:321px;height:123px", "GLUTTER_DEV_TAPS=3", "GLUTTER_SCREENSHOT=1", "glutter_report=", `f.style.width="10px"`} {
		if !strings.Contains(string(frame), want) {
			t.Errorf("the frame page lacks %q", want)
		}
	}
}

func TestShotTimesOutAndStopsTheBrowser(t *testing.T) {
	dir := t.TempDir()
	program, record := fakeBrowser(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := Shot(ctx, ShotOptions{Dir: dir, Browser: Chromium, BrowserPath: program, Timeout: 500 * time.Millisecond})
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "BrowserTimeout" || !appErr.HasRoute("Shot->awaitScreenshot.BrowserTimeout") {
		t.Fatalf("Shot = %v, want BrowserTimeout via Shot->awaitScreenshot", err)
	}
	rec, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	child, err := strconv.Atoi(recordValue(string(rec), "child"))
	if err != nil {
		t.Fatalf("no helper process was recorded: %v", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for alive(child) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if alive(child) {
		_ = syscall.Kill(child, syscall.SIGKILL) // cleaning up after the failure below
		t.Error("the browser's helper process survived a timed-out Shot")
	}
}
