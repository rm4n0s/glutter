//go:build linux && glutter_wine

package winetest

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// newHarnessAndProgram makes the harness, builds the program of testdata/hello for Windows, and
// registers the check that Wine is stopped and nothing is left running when the test ends.
func newHarnessAndProgram(t *testing.T) (*Harness, string) {
	t.Helper()
	h, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(t.TempDir(), "hello.exe")
	build := exec.Command("go", "build", "-o", exe, "./internal/winetest/testdata/hello")
	build.Dir = h.Root
	build.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if err := h.Stop(ctx); err != nil {
			t.Errorf("Stop: %v", err)
		}
		if left := Leftovers(h.Root); len(left) > 0 {
			t.Errorf("processes left running:\n%s", strings.Join(left, "\n"))
		}
	})
	return h, exe
}

// TestRunStartsAWindowsProgram builds the tiny program of testdata/hello for Windows, starts it
// under Wine with the harness, reads its exit code and output, and checks that nothing is left
// running (SPEC §25.13). Run it with `go test -tags glutter_wine -p 1 ./internal/winetest`.
func TestRunStartsAWindowsProgram(t *testing.T) {
	h, exe := newHarnessAndProgram(t)

	tests := []struct {
		name     string
		args     []string
		env      []string
		wantCode int
		wantOut  []string
	}{
		{"the output and a status of 0", nil, nil, 0, []string{"hello from Windows", "args: []", "under wine: true"}},
		{"a status of 7, the arguments and the environment", []string{"7", "two words"}, []string{"GLUTTER_HELLO=yes"}, 7,
			[]string{"args: [7 two words]", "GLUTTER_HELLO: yes"}},
		{"no Wayland display reaches the program", nil, []string{"WAYLAND_DISPLAY=wayland-0"}, 0, []string{"wayland: false"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()
			res, err := h.Run(ctx, exe, RunOptions{Args: tt.args, Env: tt.env, Timeout: time.Minute})
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if res.ExitCode != tt.wantCode {
				t.Errorf("exit code = %d, want %d\n%s", res.ExitCode, tt.wantCode, res.Output)
			}
			for _, want := range tt.wantOut {
				if !strings.Contains(res.Output, want) {
					t.Errorf("output does not contain %q:\n%s", want, res.Output)
				}
			}
		})
	}
}

// TestRunEndsAProgramThatTakesTooLong checks that `timeout` ends a program that does not stop, with
// status 124 and well before the program would have ended, and that nothing stays behind.
func TestRunEndsAProgramThatTakesTooLong(t *testing.T) {
	h, exe := newHarnessAndProgram(t)
	start := time.Now()
	res, err := h.Run(context.Background(), exe, RunOptions{Args: []string{"sleep"}, Timeout: 3 * time.Second})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.ExitCode != 124 {
		t.Errorf("exit code = %d, want 124 (timeout)\n%s", res.ExitCode, res.Output)
	}
	if d := time.Since(start); d > 40*time.Second {
		t.Errorf("the run took %v", d)
	}
}

// TestRunStopsWhenTheContextEnds checks that a cancelled context ends the run with WineStartFailed
// and leaves nothing behind.
func TestRunStopsWhenTheContextEnds(t *testing.T) {
	h, exe := newHarnessAndProgram(t)
	if err := h.EnsurePrefix(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := h.Run(ctx, exe, RunOptions{Args: []string{"sleep"}, Timeout: time.Minute}); err == nil {
		t.Error("Run did not report that the context ended")
	}
}

// TestLeftoversSeesWhatIsRunning starts a program that sleeps, checks that Leftovers lists the
// processes of the prefix while it runs, stops Wine, and checks that the list is empty afterwards.
func TestLeftoversSeesWhatIsRunning(t *testing.T) {
	h, exe := newHarnessAndProgram(t)
	if err := h.EnsurePrefix(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := h.Run(ctx, exe, RunOptions{Args: []string{"sleep"}, Timeout: 30 * time.Second})
		done <- err
	}()
	deadline := time.Now().Add(25 * time.Second)
	for len(Leftovers(h.Root)) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("Leftovers did not see the running program")
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err := h.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatalf("Run: %v", err)
	}
	if left := Leftovers(h.Root); len(left) > 0 {
		t.Errorf("processes left after Stop:\n%s", strings.Join(left, "\n"))
	}
}
