package wintool

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/winetest"
)

// fakeLauncher is a Launcher that records what it was asked and answers like the app would: it
// saves a frame at the GLUTTER_SCREENSHOT path (or, for Capture, at out) unless it is told not to.
type fakeLauncher struct {
	calls    []winetest.RunOptions
	exes     []string
	result   winetest.Result
	err      error
	saves    bool
	frame    []byte
	captured []string // the out paths of Capture
}

func (f *fakeLauncher) Launch(_ context.Context, exe string, o winetest.RunOptions) (winetest.Result, error) {
	f.calls = append(f.calls, o)
	f.exes = append(f.exes, exe)
	if f.saves {
		for _, kv := range o.Env {
			if path, ok := strings.CutPrefix(kv, "GLUTTER_SCREENSHOT="); ok {
				if err := os.WriteFile(path, f.frame, 0o644); err != nil {
					return winetest.Result{}, errors.NewErr(tagFakeFailed, err)
				}
			}
		}
	}
	return f.result, f.err
}

func (f *fakeLauncher) ProgramPath(host string) string { return host }

type fakeScreenLauncher struct{ *fakeLauncher }

func (f fakeScreenLauncher) Capture(_ context.Context, exe, out string, o winetest.RunOptions) (winetest.Result, error) {
	f.calls = append(f.calls, o)
	f.exes = append(f.exes, exe)
	f.captured = append(f.captured, out)
	if f.saves {
		if err := os.WriteFile(out, f.frame, 0o644); err != nil {
			return winetest.Result{}, errors.NewErr(tagFakeFailed, err)
		}
	}
	return f.result, f.err
}

func TestShot(t *testing.T) {
	frame := []byte("\x89PNG frame")
	tests := []struct {
		name     string
		l        *fakeLauncher
		opts     ShotOptions
		wantEnv  []string
		wantFail string // the route, when it must fail
	}{
		{"the app saves its frame", &fakeLauncher{saves: true, frame: frame, result: winetest.Result{Output: "log"}},
			ShotOptions{Env: []string{"GLUTTER_X=1"}}, []string{"GLUTTER_X=1"}, ""},
		{"the app fails", &fakeLauncher{saves: true, frame: frame, result: winetest.Result{ExitCode: 2, Output: "panic"}},
			ShotOptions{}, nil, "Shot.WindowsShotFailed"},
		{"the app ends without a frame", &fakeLauncher{result: winetest.Result{Output: "quiet"}},
			ShotOptions{}, nil, "Shot.WindowsShotFailed"},
		{"the launcher fails", &fakeLauncher{err: errors.New(TagWindowsRunFailed, "no wine")},
			ShotOptions{}, nil, "WindowsRunFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			png, output, err := Shot(context.Background(), tt.l, "/tmp/app.exe", tt.opts)
			if tt.wantFail != "" {
				appErr, ok := errors.FromError(err)
				// The failure of the launcher is returned as it is: its route is the fake's.
				if !ok || !appErr.HasRoute(tt.wantFail) {
					t.Fatalf("got %v, want route %s", err, tt.wantFail)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if string(png) != string(frame) || output != "log" {
				t.Errorf("png %q output %q", png, output)
			}
			if len(tt.l.calls) != 1 || tt.l.exes[0] != "/tmp/app.exe" {
				t.Fatalf("calls = %v on %v", tt.l.calls, tt.l.exes)
			}
			c := tt.l.calls[0]
			if c.Timeout != 2*time.Minute {
				t.Errorf("timeout = %v, want 2m", c.Timeout)
			}
			var shot string
			for _, kv := range c.Env {
				if v, ok := strings.CutPrefix(kv, "GLUTTER_SCREENSHOT="); ok {
					shot = v
				}
			}
			if shot == "" || filepath.Base(shot) != "shot.png" {
				t.Errorf("env %v has no GLUTTER_SCREENSHOT=.../shot.png", c.Env)
			}
			if _, err := os.Stat(filepath.Dir(shot)); !os.IsNotExist(err) {
				t.Errorf("the temporary directory %s was not removed (%v)", filepath.Dir(shot), err)
			}
			for _, w := range tt.wantEnv {
				if !slices.Contains(c.Env, w) {
					t.Errorf("env %v has no %s", c.Env, w)
				}
			}
		})
	}
}

func TestShotOfTheScreen(t *testing.T) {
	frame := []byte("\x89PNG screen")
	f := fakeScreenLauncher{&fakeLauncher{saves: true, frame: frame}}
	png, _, err := Shot(context.Background(), f, "/tmp/app.exe", ShotOptions{Screen: true, Settle: 5 * time.Second, Env: []string{"GLUTTER_A=b"}})
	if err != nil {
		t.Fatal(err)
	}
	if string(png) != string(frame) || len(f.calls) != 1 {
		t.Fatalf("png %q, %d calls", png, len(f.calls))
	}
	if !slices.Contains(f.calls[0].Env, "GLUTTER_SHOT_DELAY=5") || !slices.Contains(f.calls[0].Env, "GLUTTER_A=b") {
		t.Errorf("env = %v, want the delay and the app's variable", f.calls[0].Env)
	}
	for _, kv := range f.calls[0].Env {
		if strings.HasPrefix(kv, "GLUTTER_SCREENSHOT=") {
			t.Errorf("a screen capture sets %s: the app would exit before the capture", kv)
		}
	}

	// The default is 3 seconds; a launcher without Capture cannot.
	f = fakeScreenLauncher{&fakeLauncher{saves: true, frame: frame}}
	if _, _, err := Shot(context.Background(), f, "x.exe", ShotOptions{Screen: true}); err != nil || !slices.Contains(f.calls[0].Env, "GLUTTER_SHOT_DELAY=3") {
		t.Errorf("default delay: %v, env %v", err, f.calls[0].Env)
	}
	_, _, err = Shot(context.Background(), &fakeLauncher{}, "x.exe", ShotOptions{Screen: true})
	if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("Shot.WindowsScreenUnsupported") {
		t.Errorf("got %v, want WindowsScreenUnsupported from Shot", err)
	}
}

func TestRun(t *testing.T) {
	l := &fakeLauncher{result: winetest.Result{Output: "hello"}}
	res, err := Run(context.Background(), l, "a.exe", winetest.RunOptions{Env: []string{"GLUTTER_X=1"}})
	if err != nil || res.Output != "hello" {
		t.Errorf("Run = %+v, %v", res, err)
	}
	l = &fakeLauncher{result: winetest.Result{ExitCode: 3, Output: "oops"}}
	res, err = Run(context.Background(), l, "a.exe", winetest.RunOptions{})
	appErr, ok := errors.FromError(err)
	if !ok || !appErr.HasRoute("Run.WindowsRunFailed") || appErr.Metadata["exitCode"] != 3 {
		t.Errorf("got %v, want WindowsRunFailed from Run with exitCode 3", err)
	}
	if res.Output != "oops" {
		t.Errorf("the output %q was lost with the failure", res.Output)
	}
	l = &fakeLauncher{err: errors.New(TagWindowsRunFailed, "no wine")}
	if _, err = Run(context.Background(), l, "a.exe", winetest.RunOptions{}); err == nil {
		t.Error("the failure of the launcher was lost")
	}
}

func TestWineLauncherPaths(t *testing.T) {
	if got := (WineLauncher{}).ProgramPath("/tmp/glutter-shot-1/shot.png"); got != `Z:\tmp\glutter-shot-1\shot.png` {
		t.Errorf("ProgramPath = %q", got)
	}
	if got := (LocalLauncher{}).ProgramPath(`C:\x\y.png`); got != `C:\x\y.png` {
		t.Errorf("ProgramPath = %q", got)
	}
}

func TestWineLauncherRunsThroughTheHarness(t *testing.T) {
	var got []winetest.Command
	h := &winetest.Harness{
		Root: t.TempDir(), Prefix: filepath.Join(t.TempDir(), "prefix"), Environ: []string{"PATH=/bin"},
		LookPath: func(name string) (string, error) { return "/usr/bin/" + name, nil },
		Stat:     func(string) (os.FileInfo, error) { return os.Stat(os.TempDir()) },
		Exec: func(_ context.Context, c winetest.Command) (winetest.Result, error) {
			got = append(got, c)
			return winetest.Result{}, nil
		},
	}
	for _, headless := range []bool{true, false} {
		got = nil
		l := WineLauncher{Harness: h, Headless: headless}
		if _, err := l.Launch(context.Background(), "/a.exe", winetest.RunOptions{Timeout: 30 * time.Second}); err != nil {
			t.Fatal(err)
		}
		line := strings.Join(got[0].Argv, " ")
		if headless != strings.Contains(line, "headless-x11.sh") {
			t.Errorf("headless=%v: %s", headless, line)
		}
		if !strings.HasSuffix(line, "/usr/bin/wine64 /a.exe") {
			t.Errorf("headless=%v: %s", headless, line)
		}
	}
	got = nil
	if _, err := (WineLauncher{Harness: h, Headless: false}).Capture(context.Background(), "/a.exe", "/o.png", winetest.RunOptions{}); err != nil {
		t.Fatal(err)
	}
	if line := strings.Join(got[0].Argv, " "); !strings.Contains(line, "wine-screenshot.sh /o.png -- /usr/bin/wine64 /a.exe") {
		t.Errorf("Capture: %s", line)
	}
}

func TestLocalLauncher(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh")
	}
	tests := []struct {
		name     string
		args     []string
		opts     winetest.RunOptions
		wantCode int
		wantOut  string
	}{
		{"output and environment", []string{"-c", "echo out; echo $GLUTTER_T >&2"}, winetest.RunOptions{Env: []string{"GLUTTER_T=err"}}, 0, "out\nerr\n"},
		{"a failure status", []string{"-c", "exit 7"}, winetest.RunOptions{}, 7, ""},
		{"the timeout ends it with 124", []string{"-c", "sleep 30"}, winetest.RunOptions{Timeout: 200 * time.Millisecond}, 124, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.opts.Args = tt.args
			res, err := LocalLauncher{}.Launch(context.Background(), sh, tt.opts)
			if err != nil {
				t.Fatal(err)
			}
			if res.ExitCode != tt.wantCode || res.Output != tt.wantOut {
				t.Errorf("Result = %+v, want status %d output %q", res, tt.wantCode, tt.wantOut)
			}
		})
	}
	// A program that does not exist, and a context that ends the run.
	_, err = LocalLauncher{}.Launch(context.Background(), "/nonexistent/app.exe", winetest.RunOptions{})
	if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("LocalLauncher.Launch.WindowsRunFailed") {
		t.Errorf("got %v, want WindowsRunFailed from LocalLauncher.Launch", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = LocalLauncher{}.Launch(ctx, sh, winetest.RunOptions{Args: []string{"-c", "sleep 30"}})
	if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("LocalLauncher.Launch.WindowsRunFailed") {
		t.Errorf("got %v, want WindowsRunFailed from LocalLauncher.Launch", err)
	}
}

// tagFakeFailed is the tag of the failure of a fake launcher to save a file.
const tagFakeFailed = "FakeFailed"
