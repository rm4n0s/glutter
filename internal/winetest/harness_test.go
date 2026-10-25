//go:build linux

package winetest

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
)

// fakeHarness is a Harness whose programs are all found, whose files are reported by exists, and
// whose commands are recorded instead of run.
type fakeHarness struct {
	*Harness
	commands []Command
	// onRun runs when a command is "run"; it can make the prefix appear.
	onRun func(c Command) (Result, error)
}

func newFake(t *testing.T, prefixExists bool) *fakeHarness {
	t.Helper()
	root := t.TempDir()
	f := &fakeHarness{Harness: &Harness{
		Root:     root,
		Prefix:   filepath.Join(root, ".wine-test", "prefix"),
		Environ:  []string{"PATH=/bin", "HOME=/home/x", "WAYLAND_DISPLAY=wayland-0", "WINEDEBUG=+all"},
		LookPath: func(name string) (string, error) { return "/usr/bin/" + name, nil },
	}}
	f.Stat = func(path string) (os.FileInfo, error) {
		switch {
		case strings.HasSuffix(path, "system.reg"):
			if prefixExists {
				return os.Stat(root)
			}
		case path == lavapipeICD:
			return os.Stat(root)
		}
		return nil, os.ErrNotExist
	}
	f.Exec = func(_ context.Context, c Command) (Result, error) {
		f.commands = append(f.commands, c)
		if f.onRun != nil {
			return f.onRun(c)
		}
		return Result{}, nil
	}
	return f
}

func (f *fakeHarness) lines() []string {
	var out []string
	for _, c := range f.commands {
		out = append(out, strings.Join(c.Argv, " "))
	}
	return out
}

func TestEnvironment(t *testing.T) {
	f := newFake(t, true)
	got := f.Environment("GLUTTER_SCREENSHOT=/x.png")
	want := []string{
		"PATH=/bin", "HOME=/home/x",
		"WINEPREFIX=" + f.Prefix, "WINEARCH=win64", "WINEDLLOVERRIDES=mscoree,mshtml=", "WINEDEBUG=-all",
		"VK_DRIVER_FILES=" + lavapipeICD, "GLUTTER_SCREENSHOT=/x.png",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("Environment =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	tests := []struct {
		name    string
		environ []string
		stat    bool
		want    string // a variable that must be there; WINEPREFIX= alone means the prefix of the harness
		absent  string // a variable name that must not be
	}{
		{"GLUTTER_WINEDEBUG sets WINEDEBUG", []string{"GLUTTER_WINEDEBUG=err+all"}, true, "WINEDEBUG=err+all", ""},
		{"a VK_DRIVER_FILES that is set stays", []string{"VK_DRIVER_FILES=/mine.json"}, true, "VK_DRIVER_FILES=/mine.json", ""},
		{"no lavapipe, no VK_DRIVER_FILES", nil, false, "WINEARCH=win64", "VK_DRIVER_FILES"},
		{"WAYLAND_DISPLAY and WAYLAND_SOCKET are removed", []string{"WAYLAND_DISPLAY=w", "WAYLAND_SOCKET=3"}, true, "WINEARCH=win64", "WAYLAND"},
		{"the environment of the caller does not set the prefix", []string{"WINEPREFIX=/home/user/.wine"}, true, "WINEPREFIX=", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newFake(t, true)
			h.Environ = tt.environ
			if !tt.stat {
				h.Stat = func(string) (os.FileInfo, error) { return nil, os.ErrNotExist }
			}
			env := h.Environment()
			if tt.want == "WINEPREFIX=" {
				tt.want += h.Prefix
			}
			if !contains(env, tt.want) {
				t.Errorf("%v does not have %q", env, tt.want)
			}
			if tt.absent != "" {
				for _, kv := range env {
					if strings.HasPrefix(kv, tt.absent) {
						t.Errorf("%v has %q", env, kv)
					}
				}
			}
			if strings.Contains(strings.Join(env, "\n"), "/home/user/.wine") {
				t.Errorf("the user's own prefix is in the environment: %v", env)
			}
		})
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func TestEnsurePrefix(t *testing.T) {
	t.Run("an existing prefix runs nothing", func(t *testing.T) {
		f := newFake(t, true)
		if err := f.EnsurePrefix(context.Background()); err != nil {
			t.Fatal(err)
		}
		if len(f.commands) != 0 {
			t.Errorf("commands = %v, want none", f.lines())
		}
	})
	t.Run("a new prefix is booted under a time limit, then the server is stopped", func(t *testing.T) {
		f := newFake(t, false)
		f.onRun = func(c Command) (Result, error) {
			if strings.Contains(strings.Join(c.Argv, " "), "wineboot") {
				f.Stat = func(string) (os.FileInfo, error) { return os.Stat(f.Root) } // system.reg appears
			}
			return Result{}, nil
		}
		if err := f.EnsurePrefix(context.Background()); err != nil {
			t.Fatal(err)
		}
		want := []string{
			`sh -c ulimit -c 1; exec "$@" sh timeout -k 10s 90s ` + f.Root + `/scripts/headless-x11.sh -- sh -c timeout 20 "$1" wineboot --init; true sh /usr/bin/wine64`,
			"/usr/bin/wineserver -k",
		}
		if got := f.lines(); strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Errorf("commands =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
		}
		if !contains(f.commands[0].Env, "WINEPREFIX="+f.Prefix) {
			t.Errorf("the boot does not run in the prefix: %v", f.commands[0].Env)
		}
		if _, err := os.Stat(f.Prefix + ".lock"); err != nil {
			t.Errorf("no lock file: %v", err)
		}
	})
	tests := []struct {
		name      string
		setup     func(f *fakeHarness)
		wantTag   string
		wantRoute string
		wantMeta  string
	}{
		{"the prefix is still not there after the boot", func(*fakeHarness) {}, "WinePrefixFailed", "Harness.EnsurePrefix.WinePrefixFailed", "output"},
		{"Wine is not installed", func(f *fakeHarness) {
			f.LookPath = func(name string) (string, error) { return "", exec.ErrNotFound }
		}, "WineNotFound", "Harness.EnsurePrefix->Harness.wineProgram.WineNotFound", "install"},
		{"wineserver is not installed", func(f *fakeHarness) {
			f.LookPath = func(name string) (string, error) {
				if name == "wineserver" {
					return "", exec.ErrNotFound
				}
				return "/usr/bin/" + name, nil
			}
		}, "WineNotFound", "Harness.EnsurePrefix->Harness.wineProgram.WineNotFound", "install"},
		{"Xvfb is not installed", func(f *fakeHarness) {
			f.LookPath = func(name string) (string, error) {
				if name == "Xvfb" {
					return "", exec.ErrNotFound
				}
				return "/usr/bin/" + name, nil
			}
		}, "WineStartFailed", "Harness.EnsurePrefix.WineStartFailed", "install"},
		{"the boot cannot be started", func(f *fakeHarness) {
			f.onRun = func(Command) (Result, error) { return Result{}, errors.New(tagRunnerFailed, "no process") }
		}, tagRunnerFailed, tagRunnerFailed, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFake(t, false)
			tt.setup(f)
			err := f.EnsurePrefix(context.Background())
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T: %v", err, err)
			}
			if appErr.Tag != tt.wantTag || !appErr.HasRoute(tt.wantRoute) {
				t.Errorf("got %s via %s, want %s via %s", appErr.Tag, appErr.Route(), tt.wantTag, tt.wantRoute)
			}
			if tt.wantMeta != "" {
				if _, ok := appErr.Metadata[tt.wantMeta]; !ok {
					t.Errorf("metadata = %v, want %q", appErr.Metadata, tt.wantMeta)
				}
			}
		})
	}
}

func TestRunCommandLines(t *testing.T) {
	f := newFake(t, true)
	f.onRun = func(c Command) (Result, error) {
		if strings.Contains(c.Argv[len(c.Argv)-1], "prog.exe") || strings.Contains(strings.Join(c.Argv, " "), "prog.exe") {
			return Result{ExitCode: 3, Output: "out"}, nil
		}
		return Result{}, nil
	}
	res, err := f.Run(context.Background(), "/tmp/prog.exe", RunOptions{Args: []string{"-x", "1"}, Env: []string{"GLUTTER_SCREENSHOT=/s.png"}, Timeout: 45 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 3 || res.Output != "out" {
		t.Errorf("Result = %+v", res)
	}
	want := []string{
		`sh -c ulimit -c 1; exec "$@" sh timeout -k 10s 45s ` + f.Root + `/scripts/headless-x11.sh -- /usr/bin/wine64 /tmp/prog.exe -x 1`,
		"/usr/bin/wineserver -k",
	}
	if got := f.lines(); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("commands =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if !contains(f.commands[0].Env, "GLUTTER_SCREENSHOT=/s.png") || f.commands[0].Dir != f.Root {
		t.Errorf("env %v dir %q", f.commands[0].Env, f.commands[0].Dir)
	}
}

func TestRunStopsWineWhenTheRunFails(t *testing.T) {
	f := newFake(t, true)
	f.onRun = func(c Command) (Result, error) {
		if c.Argv[len(c.Argv)-1] == "-k" {
			return Result{}, nil
		}
		return Result{}, errors.New(tagRunnerFailed, "the run failed")
	}
	if _, err := f.Run(context.Background(), "prog.exe", RunOptions{}); err == nil {
		t.Fatal("the failure of the run was lost")
	}
	if got := f.lines(); len(got) != 2 || got[1] != "/usr/bin/wineserver -k" {
		t.Errorf("commands = %v, want the run, then wineserver -k", got)
	}
}

func TestRunWithoutWine(t *testing.T) {
	f := newFake(t, true)
	f.LookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	_, err := f.Run(context.Background(), "prog.exe", RunOptions{})
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "WineNotFound" || !appErr.HasRoute("Harness.Run->Harness.wineProgram.WineNotFound") {
		t.Errorf("got %v, want WineNotFound via Harness.Run->Harness.wineProgram", err)
	}
	if appErr.Metadata["install"] != WineInstall {
		t.Errorf("metadata = %v", appErr.Metadata)
	}
}

func TestNewFindsTheModuleRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)
	t.Setenv(PrefixEnv, "")
	h, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	if h.Root != root || h.Prefix != filepath.Join(root, ".wine-test", "prefix") {
		t.Errorf("Root %q Prefix %q", h.Root, h.Prefix)
	}
	t.Setenv(PrefixEnv, "/elsewhere/prefix")
	if h, err = New(root); err != nil || h.Prefix != "/elsewhere/prefix" {
		t.Errorf("with %s: %v, %v", PrefixEnv, h, err)
	}
	t.Chdir(t.TempDir()) // no go.mod above the temporary directory
	_, err = New("")
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "ModuleRootNotFound" || !appErr.HasRoute("New.ModuleRootNotFound") {
		t.Errorf("got %v, want ModuleRootNotFound via New", err)
	}
}

func TestLeftoversReadsTheEnvironmentOfProcesses(t *testing.T) {
	proc := t.TempDir()
	write := func(pid, env, stat, cmdline string) {
		dir := filepath.Join(proc, pid)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		for name, data := range map[string]string{"environ": env, "stat": stat, "cmdline": cmdline} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	write("100", "A=1\x00WINEPREFIX=/r/.wine-test/prefix\x00B=2\x00", "100 (wineserver) S 1", "wineserver\x00-p\x00")
	write("101", "WINEPREFIX=/r/.wine-test/prefix\x00", "101 (Xvfb) Z 1", "Xvfb\x00")       // a zombie
	write("102", "WINEPREFIX=/home/u/.wine\x00", "102 (wine) S 1", "wine\x00")              // the user's prefix
	write("103", "WINEPREFIX=/r/.wine-test/prefix-other\x00", "103 (wine) S 1", "wine\x00") // another prefix
	write("104", "WINEPREFIX=/r/.wine-test/prefix\x00", "104 (go) S 1", "go\x00test\x00")   // this process
	write("105", "WINEPREFIX=/r/.wine-test/prefix\x00", "105 (explorer) S 1", "explorer.exe\x00/desktop\x00")
	if err := os.MkdirAll(filepath.Join(proc, "self"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := leftovers(proc, []byte("WINEPREFIX=/r/.wine-test/prefix\x00"), 104)
	want := []string{"100 wineserver -p", "105 explorer.exe /desktop"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("leftovers = %q, want %q", got, want)
	}
	if got := leftovers(filepath.Join(proc, "missing"), []byte("x"), 1); got != nil {
		t.Errorf("a missing /proc = %q", got)
	}
}

func TestRunCommand(t *testing.T) {
	tests := []struct {
		name     string
		argv     []string
		wantCode int
		wantOut  string
		wantTag  string
	}{
		{"output and status", []string{"sh", "-c", "echo out; echo err >&2; exit 3"}, 3, "out\nerr\n", ""},
		{"success", []string{"true"}, 0, "", ""},
		{"a program that is not there", []string{"/nonexistent/program"}, 0, "", "WineStartFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := runCommand(context.Background(), Command{Argv: tt.argv})
			if tt.wantTag != "" {
				appErr, ok := errors.FromError(err)
				if !ok || appErr.Tag != tt.wantTag || !appErr.HasRoute("runCommand.WineStartFailed") {
					t.Errorf("got %v, want %s via runCommand", err, tt.wantTag)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if res.ExitCode != tt.wantCode || res.Output != tt.wantOut {
				t.Errorf("Result = %+v, want code %d output %q", res, tt.wantCode, tt.wantOut)
			}
		})
	}
}

// TestRunCommandKillsTheProcessGroup checks that a cancelled run takes the children down with it.
func TestRunCommandKillsTheProcessGroup(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	// The shell starts a child in the background and waits for it.
	_, err := runCommand(ctx, Command{Argv: []string{"sh", "-c", "sleep 60 & echo $! > " + pidFile + "; wait"}})
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "WineStartFailed" {
		t.Fatalf("got %v, want WineStartFailed", err)
	}
	if d := time.Since(start); d > 10*time.Second {
		t.Errorf("the cancelled run took %v", d)
	}
	data, rerr := os.ReadFile(pidFile)
	if rerr != nil {
		t.Fatal(rerr)
	}
	pid := strings.TrimSpace(string(data))
	deadline := time.Now().Add(5 * time.Second)
	for {
		stat, serr := os.ReadFile("/proc/" + pid + "/stat")
		if serr != nil || strings.Contains(string(stat), ") Z ") {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("the child %s of the cancelled run is still running", pid)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// The tag of the error that a fake runner makes.
const tagRunnerFailed = "RunnerFailed"
