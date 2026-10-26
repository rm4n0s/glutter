package androidtool

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// TestSweepStaleTempDirs checks which entries of the temporary directory the sweep removes: only the
// directories of the builder (by prefix) that are older than the age.
func TestSweepStaleTempDirs(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name  string
		entry string
		dir   bool
		age   time.Duration // how long ago it was changed
		gone  bool
	}{
		{name: "an old apk work directory", entry: "glutter-apk-123", dir: true, age: 3 * time.Hour, gone: true},
		{name: "an old dex work directory", entry: "glutter-dex-9", dir: true, age: 5 * time.Hour, gone: true},
		{name: "a recent apk work directory (a build that still runs)", entry: "glutter-apk-456", dir: true, age: 10 * time.Minute},
		{name: "just under the age", entry: "glutter-apk-789", dir: true, age: staleTempAge - time.Minute},
		{name: "an old directory of another program", entry: "other-123", dir: true, age: 48 * time.Hour},
		{name: "an old directory of another glutter tool", entry: "glutter-windows-run-1", dir: true, age: 48 * time.Hour},
		{name: "an old file with the prefix", entry: "glutter-apk-file", age: 48 * time.Hour},
	}
	tmp := t.TempDir()
	for _, tt := range tests {
		path := filepath.Join(tmp, tt.entry)
		if tt.dir {
			if err := os.MkdirAll(filepath.Join(path, "lib", "x86_64"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(path, "lib", "x86_64", "libapp.so"), []byte("ELF"), 0o644); err != nil {
				t.Fatal(err)
			}
		} else if err := os.WriteFile(path, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		when := now.Add(-tt.age)
		if err := os.Chtimes(path, when, when); err != nil {
			t.Fatal(err)
		}
	}
	removed := sweepStaleTempDirs(tmp, now, staleTempAge, apkWorkPrefix, dexWorkPrefix)
	var want []string
	for _, tt := range tests {
		_, err := os.Stat(filepath.Join(tmp, tt.entry))
		if gone := os.IsNotExist(err); gone != tt.gone {
			t.Errorf("%s: removed = %v, want %v", tt.name, gone, tt.gone)
		}
		if tt.gone {
			want = append(want, filepath.Join(tmp, tt.entry))
		}
	}
	sort.Strings(removed)
	sort.Strings(want)
	if strings.Join(removed, ",") != strings.Join(want, ",") {
		t.Errorf("sweep returned %v, want %v", removed, want)
	}
}

// TestSweepStaleTempDirsOfAMissingDirectory checks that a temporary directory that cannot be read is no
// failure: the sweep is best effort.
func TestSweepStaleTempDirsOfAMissingDirectory(t *testing.T) {
	if got := sweepStaleTempDirs(filepath.Join(t.TempDir(), "nothing"), time.Now(), time.Hour, apkWorkPrefix); len(got) != 0 {
		t.Errorf("removed %v from a directory that is not there", got)
	}
}

// leftovers lists the work directories of the builder in tmp.
func leftovers(t *testing.T, tmp string) []string {
	t.Helper()
	entries, err := os.ReadDir(tmp)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), apkWorkPrefix) || strings.HasPrefix(e.Name(), dexWorkPrefix) {
			out = append(out, e.Name())
		}
	}
	return out
}

// TestBuildLeavesNoWorkDirectory checks what the leftover of 2026-10-26 asked for: whichever program of the
// build fails (or the build succeeds, or the context ends), the work directory of Build is gone when it
// returns.
func TestBuildLeavesNoWorkDirectory(t *testing.T) {
	tests := []struct {
		name    string
		fail    string // the program that fails; empty for none
		cancel  bool
		wantErr bool
	}{
		{name: "a build that succeeds"},
		{name: "go fails", fail: "go", wantErr: true},
		{name: "aapt2 fails", fail: "aapt2", wantErr: true},
		{name: "zipalign fails", fail: "zipalign", wantErr: true},
		{name: "apksigner fails", fail: "apksigner", wantErr: true},
		{name: "the context is done", cancel: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			t.Setenv("TMPDIR", tmp)
			dir := t.TempDir()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			run := &fakeRunner{fail: tt.fail}
			var r Runner = run
			if tt.cancel {
				r = cancelRunner{Runner: run, cancel: cancel}
			}
			_, err := Build(ctx, testSDK(), r, BuildOptions{
				Package: "./examples/counter", Out: filepath.Join(dir, "app.apk"), ConfigDir: filepath.Join(dir, "key"), Dex: []byte("dex"),
			})
			if (err != nil) != tt.wantErr {
				t.Fatalf("Build: err = %v, want an error: %v", err, tt.wantErr)
			}
			if left := leftovers(t, tmp); len(left) > 0 {
				t.Errorf("the build left %v in the temporary directory", left)
			}
		})
	}
}

// cancelRunner ends the context at the first program and fails it, as a build that is interrupted does.
type cancelRunner struct {
	Runner
	cancel context.CancelFunc
}

func (c cancelRunner) Run(ctx context.Context, cmd Command) (string, error) {
	c.cancel()
	return "", ctx.Err()
}

// TestBuildRemovesAStaleWorkDirectoryOfAKilledBuild checks that a build sweeps the directory that a killed
// build left, and keeps the recent one of a build that still runs.
func TestBuildRemovesAStaleWorkDirectoryOfAKilledBuild(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	old, recent := filepath.Join(tmp, apkWorkPrefix+"killed"), filepath.Join(tmp, apkWorkPrefix+"running")
	for _, d := range []string{old, recent} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	when := time.Now().Add(-3 * time.Hour)
	if err := os.Chtimes(old, when, when); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if _, err := Build(context.Background(), testSDK(), &fakeRunner{}, BuildOptions{
		Package: "./examples/counter", Out: filepath.Join(dir, "app.apk"), ConfigDir: filepath.Join(dir, "key"), Dex: []byte("dex"),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Errorf("the stale directory is still there: %v", err)
	}
	if _, err := os.Stat(recent); err != nil {
		t.Errorf("the recent directory was removed: %v", err)
	}
}
