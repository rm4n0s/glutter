package main

import (
	"context"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/androidtest"
	"github.com/rm4n0s/glutter/internal/androidtool"
)

// shotFile is where the self-capture of the app writes, below the app's directory (the engine
// puts it in HOME for GLUTTER_SCREENSHOT=1).
const shotFile = "files/glutter-screenshot.png"

// runShot is `glutter-android shot` (proof method E, SPEC §24.13): it builds and starts the app
// with GLUTTER_SCREENSHOT=1 and pulls the PNG the app saves after its first idle frame. With
// -screen it takes the device's whole screen instead. Without -device it starts the emulator of
// the harness, and stops it afterwards.
func runShot(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("shot", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var bf buildFlags
	var df deviceFlags
	var envs listFlag
	bf.add(fs, false)
	df.add(fs)
	fs.Var(&envs, "env", "an environment variable for the app, GLUTTER_X=v (repeatable)")
	avd := fs.String("avd", "glutter-37", "the virtual device to start when no -device is given")
	renderer := fs.String("renderer", "", "the renderer: auto, vulkan, gles or soft (GLUTTER_RENDERER)")
	screen := fs.Bool("screen", false, "capture the device's whole screen (status bar, keyboard, notifications) instead of the app's frame")
	settle := fs.Duration("settle", 1500*time.Millisecond, "with -screen, how long to wait after the first frame")
	timeout := fs.Duration("timeout", 2*time.Minute, "how long to wait for the app")
	out := fs.String("o", "", "the PNG to write")
	pkg, err := parse(fs, args)
	if err != nil {
		return err
	}
	if *out == "" {
		return errors.New(tagAndroidUsage, "shot needs -o file.png")
	}
	extras, err := parseEnvs(envs)
	if err != nil {
		return err
	}
	if *renderer != "" {
		extras["GLUTTER_RENDERER"] = *renderer
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
	appID, apk, cleanup, err := buildForDevice(ctx, sdk, adb, &bf, pkg, stdout)
	if err != nil {
		return err
	}
	defer cleanup()
	if err := adb.Install(ctx, apk); err != nil {
		return err
	}
	if !*screen {
		extras["GLUTTER_SCREENSHOT"] = "1"
		if err := adb.RemoveAppFile(ctx, appID, shotFile); err != nil {
			return err
		}
	}
	if _, err := adb.Output(ctx, "logcat", "-c"); err != nil {
		return err
	}
	if err := adb.Start(ctx, appID, extras); err != nil {
		return err
	}
	var png []byte
	if *screen {
		if png, err = screenOnceDrawn(ctx, adb, appID, *settle, *timeout); err != nil {
			return err
		}
	} else {
		if err := waitExit(ctx, adb, appID, *timeout); err != nil {
			return err
		}
		if png, err = adb.PullAppFile(ctx, appID, shotFile); err != nil {
			if _, werr := io.WriteString(stderr, "the log of the device:\n"+fullLogTail(ctx, adb)+"\n"); werr != nil {
				return errors.NewErr(tagAndroidUsage, werr)
			}
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		return errors.NewErr(tagAndroidUsage, err).SetMetadata("path", *out)
	}
	if err := os.WriteFile(*out, png, 0o644); err != nil {
		return errors.NewErr(tagAndroidUsage, err).SetMetadata("path", *out)
	}
	return say(stdout, "saved %s (%d bytes)\n", *out, len(png))
}

// parseEnvs turns the values of -env into the extras of the activity.
func parseEnvs(envs []string) (map[string]string, error) {
	extras := map[string]string{}
	for _, kv := range envs {
		k, v, ok := strings.Cut(kv, "=")
		if !ok || !strings.HasPrefix(k, "GLUTTER_") {
			return nil, errors.New(tagAndroidUsage, "-env takes GLUTTER_NAME=value, got "+kv)
		}
		extras[k] = v
	}
	return extras, nil
}

// waitExit waits until the app has no process.
func waitExit(ctx context.Context, adb androidtool.ADB, appID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		pid, err := adb.Pid(ctx, appID)
		if err != nil {
			return err
		}
		if pid == 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return errors.New(tagAndroidUsage, "the app did not end in time", "appID", appID, "log", logOf(ctx, adb, appID))
		}
		select {
		case <-ctx.Done():
			return errors.NewErr(tagAndroidUsage, ctx.Err())
		case <-time.After(300 * time.Millisecond):
		}
	}
}

// screenOnceDrawn waits until the app has chosen its renderer (its first frame is drawn), waits
// the settle time, takes the screen and stops the app.
func screenOnceDrawn(ctx context.Context, adb androidtool.ADB, appID string, settle, timeout time.Duration) ([]byte, error) {
	deadline := time.Now().Add(timeout)
	for !strings.Contains(logOf(ctx, adb, appID), "renderer chosen") {
		if time.Now().After(deadline) {
			return nil, errors.New(tagAndroidUsage, "the app drew no frame in time", "appID", appID, "log", logOf(ctx, adb, appID))
		}
		select {
		case <-ctx.Done():
			return nil, errors.NewErr(tagAndroidUsage, ctx.Err())
		case <-time.After(300 * time.Millisecond):
		}
	}
	time.Sleep(settle)
	png, err := adb.Screencap(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := adb.Output(ctx, "shell", "am", "force-stop", appID); err != nil {
		return nil, err
	}
	return png, nil
}

// fullLogTail is the last lines of the whole log of the device, for a failure.
func fullLogTail(ctx context.Context, adb androidtool.ADB) string {
	out, _ := adb.Output(ctx, "logcat", "-d", "-v", "brief") // a failing logcat is an empty log
	var lines []string
	for _, l := range strings.Split(out, "\n") {
		// The device is noisy: keep what can explain a dead app.
		for _, key := range []string{"glutter", "AndroidRuntime", "DEBUG", "libc", "Fatal", "FATAL", "ActivityManager", "ActivityTaskManager"} {
			if strings.Contains(l, key) {
				lines = append(lines, l)
				break
			}
		}
	}
	if len(lines) > 80 {
		lines = lines[len(lines)-80:]
	}
	return strings.Join(lines, "\n")
}

// logOf is the log lines of the app so far.
func logOf(ctx context.Context, adb androidtool.ADB, appID string) string {
	out, _ := adb.Output(ctx, "logcat", "-d", "-v", "brief", "-s", appID+":V") // a failing logcat is an empty log
	return out
}
