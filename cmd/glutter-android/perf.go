package main

import (
	"bytes"
	"context"
	"flag"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/androidtest"
	"github.com/rm4n0s/glutter/internal/androidtool"
)

// runPerf is `glutter-android perf` (SPEC §15 "Android", step 26.5): it builds the package for
// each ABI, installs it and prints a report with the APK size, the cold start, the idle CPU, the
// frame times of a scenario and the memory. Without -device it starts the emulator of the
// harness and stops it afterwards.
func runPerf(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("perf", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var bf buildFlags
	var df deviceFlags
	var envs listFlag
	bf.add(fs, false)
	df.add(fs)
	// perf measures release builds of both ABIs unless told otherwise.
	bf.release, bf.abi = true, "arm64,amd64"
	fs.Lookup("release").DefValue, fs.Lookup("abi").DefValue = "true", "arm64,amd64"
	fs.Var(&envs, "env", "an environment variable for the app, GLUTTER_X=v (repeatable); GLUTTER_DEV_EVENTS=script drives the frame scenario and needs -release=false")
	avd := fs.String("avd", "glutter-37", "the virtual device to start when no -device is given")
	renderer := fs.String("renderer", "", "the renderer: auto, vulkan, gles or soft (GLUTTER_RENDERER)")
	starts := fs.Int("starts", 5, "how many times the app is stopped and started to time the cold start")
	settle := fs.Duration("settle", 8*time.Second, "how long the app runs after its first frame before the idle window")
	seconds := fs.Int("seconds", 10, "the length of the idle window and of the frame window, in seconds")
	frames := fs.Bool("frames", true, "run the frame window (GLUTTER_FRAME_STATS=1 with the -env scenario)")
	format := fs.String("format", "text", "the report: text or markdown")
	out := fs.String("o", "", "write the report to this file (default: standard output)")
	pkg, err := parse(fs, args)
	if err != nil {
		return err
	}
	if *format != "text" && *format != "markdown" {
		return errors.New(tagAndroidPerfBadFormat, "-format is text or markdown", "format", *format)
	}
	if *starts < 1 || *seconds < 1 || *settle < 0 {
		return errors.New(tagAndroidPerfBadCount, "-starts and -seconds are at least 1, and -settle is not negative", "starts", *starts, "seconds", *seconds, "settle", settle.String())
	}
	extras, err := parseEnvs(envs)
	if err != nil {
		return err
	}
	if *renderer != "" {
		extras["GLUTTER_RENDERER"] = *renderer
	}
	if _, ok := extras[devEventsKey]; ok && bf.release {
		return errors.New(tagAndroidPerfDevEventsRelease, "GLUTTER_DEV_EVENTS is read only by debug builds: add -release=false (its frame times are those of a debug build)")
	}
	abis, err := parseABIs(bf.abi)
	if err != nil {
		return err
	}

	var sdk *androidtool.SDK
	var adb androidtool.ADB
	if df.device == "" {
		emu, err := androidtest.Start(ctx, androidtest.Options{AVD: *avd})
		if err != nil {
			return err
		}
		defer emu.Stop()
		if sdk, err = androidtool.FindSDK(os.Getenv); err != nil {
			return err
		}
		adb = emu.ADB
	} else if sdk, adb, err = df.connect(ctx); err != nil {
		return err
	}
	opts := bf.options(pkg)
	if opts.Release && (opts.Key == "" || opts.Cert == "") {
		// A measurement build is signed with the debug key, so that it installs over the debug
		// build of the same app; it is not a build to publish.
		dir, err := debugKeyDir()
		if err != nil {
			return err
		}
		key, err := androidtool.EnsureDebugKey(dir)
		if err != nil {
			return err
		}
		opts.Key, opts.Cert = key.Key, key.Cert
	}
	dir, err := os.MkdirTemp("", "glutter-android-perf-")
	if err != nil {
		return errors.NewErr(tagAndroidPerfTempDir, err)
	}
	defer func() { _ = os.RemoveAll(dir) }() // best effort: a temporary directory

	deps := perfDeps{
		adb: adb, serial: adb.Serial, dir: dir, progress: stderr, now: time.Now, sleep: sleepContext,
		build: func(ctx context.Context, o androidtool.BuildOptions) (*androidtool.BuildResult, error) {
			return androidtool.Build(ctx, sdk, androidtool.ExecRunner{}, o)
		},
	}
	if deps.serial == "" {
		deps.serial = "device"
	}
	rep, err := measurePerf(ctx, deps, perfOptions{pkg: pkg, build: opts, abis: abis, extras: extras,
		starts: *starts, settle: *settle, seconds: *seconds, frames: *frames})
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := rep.render(&buf, *format); err != nil {
		return err
	}
	if *out == "" {
		_, err := stdout.Write(buf.Bytes())
		if err != nil {
			return errors.NewErr(tagAndroidPerfWriteFailed, err)
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		return errors.NewErr(tagAndroidPerfWriteFailed, err).SetMetadata("path", *out)
	}
	if err := os.WriteFile(*out, buf.Bytes(), 0o644); err != nil {
		return errors.NewErr(tagAndroidPerfWriteFailed, err).SetMetadata("path", *out)
	}
	return say(stderr, "wrote %s\n", *out)
}

// debugKeyDir is where androidtool.Build keeps the debug key by default.
func debugKeyDir() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", errors.NewErr(tagAndroidPerfKeyDir, err)
	}
	return filepath.Join(cfg, "glutter", "android-debug"), nil
}

// sleepContext waits for d, or until ctx is done (AndroidPerfInterrupted).
func sleepContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return errors.NewErr(tagAndroidPerfInterrupted, ctx.Err())
	case <-t.C:
		return nil
	}
}

// The tags of the errors that perf makes (rm4n0s/errors, R5).
const (
	// tagAndroidPerfBadFormat is the tag of the error for a -format that is not text or markdown.
	tagAndroidPerfBadFormat = "AndroidPerfBadFormat"
	// tagAndroidPerfBadCount is the tag of the error for a -starts, -seconds or -settle out of range.
	tagAndroidPerfBadCount = "AndroidPerfBadCount"
	// tagAndroidPerfDevEventsRelease is the tag of the error for dev events given to a release build.
	tagAndroidPerfDevEventsRelease = "AndroidPerfDevEventsRelease"
	// tagAndroidPerfBadABI is the tag of the error for an -abi that names no known ABI.
	tagAndroidPerfBadABI = "AndroidPerfBadABI"
	// tagAndroidPerfNoABI is the tag of the error for a device that runs none of the built ABIs.
	tagAndroidPerfNoABI = "AndroidPerfNoABI"
	// tagAndroidPerfStartRejected is the tag of the error when am start -W reports a failure.
	tagAndroidPerfStartRejected = "AndroidPerfStartRejected"
	// tagAndroidPerfAmStartParse is the tag of the error for an am start -W output without times.
	tagAndroidPerfAmStartParse = "AndroidPerfAmStartParse"
	// tagAndroidPerfProcStatParse is the tag of the error for a /proc/<pid>/stat that is not read.
	tagAndroidPerfProcStatParse = "AndroidPerfProcStatParse"
	// tagAndroidPerfProcStatusParse is the tag of the error for a /proc/<pid>/status without VmRSS.
	tagAndroidPerfProcStatusParse = "AndroidPerfProcStatusParse"
	// tagAndroidPerfMeminfoParse is the tag of the error for a dumpsys meminfo without a total.
	tagAndroidPerfMeminfoParse = "AndroidPerfMeminfoParse"
	// tagAndroidPerfAPKRead is the tag of the error when the zip of an APK cannot be read.
	tagAndroidPerfAPKRead = "AndroidPerfAPKRead"
	// tagAndroidPerfNoFrame is the tag of the error when the app logs no first frame in time.
	tagAndroidPerfNoFrame = "AndroidPerfNoFrame"
	// tagAndroidPerfAppDied is the tag of the error when the app's process ends or changes.
	tagAndroidPerfAppDied = "AndroidPerfAppDied"
	// tagAndroidPerfInterrupted is the tag of the error when the context ends during a wait.
	tagAndroidPerfInterrupted = "AndroidPerfInterrupted"
	// tagAndroidPerfWriteFailed is the tag of the error when the report cannot be written.
	tagAndroidPerfWriteFailed = "AndroidPerfWriteFailed"
	// tagAndroidPerfTempDir is the tag of the error when the temporary directory for the APKs cannot be made.
	tagAndroidPerfTempDir = "AndroidPerfTempDir"
	// tagAndroidPerfKeyDir is the tag of the error when the config directory is unknown.
	tagAndroidPerfKeyDir = "AndroidPerfKeyDir"
)
