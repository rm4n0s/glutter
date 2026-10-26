package main

import (
	"bytes"
	"context"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/wintool"
)

// runPerf is `glutter-windows perf` (SPEC §15 "Windows", step 28.15): it builds the package for each
// architecture and runs the amd64 .exe under Wine, and prints a report with the size of the .exe files, the
// cold start with an empty and with a warm shader cache, the idle CPU and the memory of the Wine
// processes of the prefix (/proc), and the frame times of a scenario. The numbers are Wine's: Direct3D 12
// runs on vkd3d on the host's Vulkan (lavapipe), so they are for noticing regressions, not what a GPU on
// Windows does (SPEC §15).
func runPerf(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("perf", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var bf buildFlags
	var envs listFlag
	bf.add(fs, false)
	// perf measures release builds unless told otherwise.
	bf.release = true
	fs.Lookup("release").DefValue = "true"
	arch := fs.String("arch", "amd64,arm64", "the architectures to build and size, comma separated; amd64 is the one that runs under Wine")
	fs.Var(&envs, "env", "an environment variable for the app, GLUTTER_X=v (repeatable); GLUTTER_DEV_EVENTS=script drives the frame scenario and needs -release=false")
	renderer := fs.String("renderer", "d3d12", "the renderer: auto, d3d12 or soft (GLUTTER_RENDERER)")
	starts := fs.Int("starts", 5, "how many times the app is started to time the cold start, with an empty and with a warm shader cache")
	settle := fs.Duration("settle", 8*time.Second, "how long the app runs after its start before the idle window")
	seconds := fs.Int("seconds", 10, "the length of the idle window and of the frame window, in seconds")
	frames := fs.Bool("frames", true, "run the frame window (GLUTTER_FRAME_STATS=1 with the -env scenario)")
	format := fs.String("format", "text", "the report: text or markdown")
	out := fs.String("o", "", "write the report to this file (default: standard output)")
	pkg, err := parse(fs, args)
	if err != nil {
		return err
	}
	if *format != "text" && *format != "markdown" {
		return errors.New(tagWindowsPerfBadFormat, "-format is text or markdown", "format", *format)
	}
	if *starts < 1 || *seconds < 1 || *settle < 0 {
		return errors.New(tagWindowsPerfBadCount, "-starts and -seconds are at least 1, and -settle is not negative", "starts", *starts, "seconds", *seconds, "settle", settle.String())
	}
	extras, err := parseEnvs(envs)
	if err != nil {
		return err
	}
	if *renderer != "" {
		extras = append(extras, "GLUTTER_RENDERER="+*renderer)
	}
	if hasEnv(extras, devEventsKey) && bf.release {
		return errors.New(tagWindowsPerfDevEventsRelease, "GLUTTER_DEV_EVENTS is read only by debug builds: add -release=false (its frame times are those of a debug build)")
	}
	archs := splitList(*arch)
	if !contains(archs, "amd64") {
		return errors.New(tagWindowsPerfBadArch, "-arch must include amd64, the architecture that runs under Wine", "arch", *arch)
	}
	launcher, err := newLauncher(true)
	if err != nil {
		return err
	}
	wine, err := wineOf(launcher)
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "glutter-windows-perf-")
	if err != nil {
		return errors.NewErr(tagWindowsPerfTempDir, err)
	}
	defer func() { _ = os.RemoveAll(dir) }() // best effort: a temporary directory

	prefix := wine.Harness.Prefix
	deps := perfDeps{
		launcher: wine, procs: newProcFS(prefix), dir: dir, progress: stderr, now: time.Now, sleep: sleepContext, loc: time.Local,
		build: func(ctx context.Context, o wintool.BuildOptions) (*wintool.BuildResult, error) {
			return wintool.Build(ctx, wintool.ExecRunner{}, o)
		},
		clearCache: func() error { return clearShaderCache(prefix) },
	}
	rep, err := measurePerf(ctx, deps, perfOptions{pkg: pkg, build: bf.options(pkg), archs: archs, extras: extras,
		starts: *starts, settle: *settle, seconds: *seconds, frames: *frames})
	if err != nil {
		return err
	}
	return writeReport(rep, *format, *out, stdout, stderr)
}

// wineOf is the Wine launcher behind l. It fails with WindowsPerfUnsupported on a machine that runs the
// .exe itself (Windows), where there are no Wine processes to read from /proc.
func wineOf(l wintool.Launcher) (wintool.WineLauncher, error) {
	wine, ok := l.(wintool.WineLauncher)
	if !ok {
		return wintool.WineLauncher{}, errors.New(tagWindowsPerfUnsupported, "perf measures the Wine processes through /proc: run it on Linux under Wine; on a Windows PC use Task Manager (docs/MANUAL-CHECKS.md, 28.15)")
	}
	return wine, nil
}

// clearShaderCache empties the shader cache of the Wine prefix (glutter\shaders under %LOCALAPPDATA% of
// each user of the prefix, SPEC §9.7), so that the next start compiles the shaders. It fails with
// WindowsPerfCacheClear.
func clearShaderCache(prefix string) error {
	dirs, err := filepath.Glob(filepath.Join(prefix, "drive_c", "users", "*", "AppData", "Local", "glutter", "shaders"))
	if err != nil {
		return errors.NewErr(tagWindowsPerfCacheClear, err).SetMetadata("prefix", prefix)
	}
	for _, d := range dirs {
		if err := os.RemoveAll(d); err != nil {
			return errors.NewErr(tagWindowsPerfCacheClear, err).SetMetadata("dir", d)
		}
	}
	return nil
}

// writeReport renders the report and writes it to the file out, or to stdout when out is empty. It
// fails with WindowsPerfWriteFailed.
func writeReport(rep *perfReport, format, out string, stdout, stderr io.Writer) error {
	var buf bytes.Buffer
	if err := rep.render(&buf, format); err != nil {
		return err
	}
	if out == "" {
		if _, err := stdout.Write(buf.Bytes()); err != nil {
			return errors.NewErr(tagWindowsPerfWriteFailed, err)
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return errors.NewErr(tagWindowsPerfWriteFailed, err).SetMetadata("path", out)
	}
	if err := os.WriteFile(out, buf.Bytes(), 0o644); err != nil {
		return errors.NewErr(tagWindowsPerfWriteFailed, err).SetMetadata("path", out)
	}
	return say(stderr, "wrote %s\n", out)
}

// splitList splits a comma separated list, leaving out empty entries.
func splitList(s string) []string {
	var out []string
	for _, a := range strings.Split(s, ",") {
		if a = strings.TrimSpace(a); a != "" {
			out = append(out, a)
		}
	}
	return out
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// hasEnv says whether a KEY=value list has the key.
func hasEnv(list []string, key string) bool {
	for _, kv := range list {
		if k, _, _ := strings.Cut(kv, "="); k == key {
			return true
		}
	}
	return false
}

// sleepContext waits for d, or until ctx is done (WindowsPerfInterrupted).
func sleepContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return errors.NewErr(tagWindowsPerfInterrupted, ctx.Err())
	case <-t.C:
		return nil
	}
}

// The tags of the errors that perf makes (rm4n0s/errors, R5); more are in perf_measure.go and perf_proc.go.
const (
	// tagWindowsPerfBadFormat is the tag of the error for a -format that is not text or markdown.
	tagWindowsPerfBadFormat = "WindowsPerfBadFormat"
	// tagWindowsPerfBadCount is the tag of the error for a -starts, -seconds or -settle out of range.
	tagWindowsPerfBadCount = "WindowsPerfBadCount"
	// tagWindowsPerfDevEventsRelease is the tag of the error for dev events given to a release build.
	tagWindowsPerfDevEventsRelease = "WindowsPerfDevEventsRelease"
	// tagWindowsPerfBadArch is the tag of the error for an -arch without amd64.
	tagWindowsPerfBadArch = "WindowsPerfBadArch"
	// tagWindowsPerfUnsupported is the tag of the error for perf on a machine that does not run Wine.
	tagWindowsPerfUnsupported = "WindowsPerfUnsupported"
	// tagWindowsPerfTempDir is the tag of the error when the temporary directory cannot be made.
	tagWindowsPerfTempDir = "WindowsPerfTempDir"
	// tagWindowsPerfCacheClear is the tag of the error when the shader cache of the prefix cannot be emptied.
	tagWindowsPerfCacheClear = "WindowsPerfCacheClear"
	// tagWindowsPerfWriteFailed is the tag of the error when the report cannot be written to its place.
	tagWindowsPerfWriteFailed = "WindowsPerfWriteFailed"
	// tagWindowsPerfInterrupted is the tag of the error when the context ends during a wait.
	tagWindowsPerfInterrupted = "WindowsPerfInterrupted"
)
