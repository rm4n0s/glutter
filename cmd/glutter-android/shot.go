package main

import (
	"bytes"
	"context"
	"flag"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
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
	rotation := fs.Int("rotation", 0, "turn the device before the app starts: 0 portrait, 1 and 3 landscape, 2 upside down; it is turned back after")
	out := fs.String("o", "", "the PNG to write")
	pkg, err := parse(fs, args)
	if err != nil {
		return err
	}
	if *out == "" {
		return errors.New(tagAndroidUsage, "shot needs -o file.png")
	}
	if *rotation < 0 || *rotation > 3 {
		return errors.New(tagAndroidUsage, "-rotation is 0, 1, 2 or 3")
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
	if *rotation != 0 {
		if err := turnDevice(ctx, adb, *rotation); err != nil {
			return err
		}
		// Back to portrait, even when the shot failed (a fresh context: ctx may be done).
		defer func() { _ = turnDevice(context.Background(), adb, 0) }() // best effort: the emulator is stopped next
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

// turnDevice sets the rotation of the device (0 portrait, 1 and 3 landscape, 2 upside down) and
// switches the automatic rotation off, as `settings put system user_rotation` does.
func turnDevice(ctx context.Context, adb androidtool.ADB, rotation int) error {
	if _, err := adb.Output(ctx, "shell", "settings", "put", "system", "accelerometer_rotation", "0"); err != nil {
		return err
	}
	if _, err := adb.Output(ctx, "shell", "settings", "put", "system", "user_rotation", strconv.Itoa(rotation)); err != nil {
		return err
	}
	time.Sleep(time.Second) // the window system turns the display
	return nil
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

// screenOnceDrawn waits until the app has presented its first frame, waits
// the settle time, takes the screen and stops the app.
func screenOnceDrawn(ctx context.Context, adb androidtool.ADB, appID string, settle, timeout time.Duration) ([]byte, error) {
	deadline := time.Now().Add(timeout)
	for !strings.Contains(logOf(ctx, adb, appID), "first frame presented") {
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
	// The emulator shows a presented frame some time after the app presented it, on Vulkan
	// sometimes seconds: an all-black screen is taken again for a while.
	for end := time.Now().Add(blankWait); err == nil && blankScreen(png) && time.Now().Before(end); {
		time.Sleep(500 * time.Millisecond)
		png, err = adb.Screencap(ctx)
	}
	if err != nil {
		return nil, err
	}
	if _, err := adb.Output(ctx, "shell", "am", "force-stop", appID); err != nil {
		return nil, err
	}
	return png, nil
}

// blankWait is how long a black screen is taken again.
const blankWait = 10 * time.Second

// blankScreen reports whether a screenshot is black but for the status bar: fewer than 1% of its
// pixels are lighter than a dark gray. An image that does not decode is not blank.
func blankScreen(data []byte) bool {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return false
	}
	b := img.Bounds()
	light := 0
	for y := b.Min.Y; y < b.Max.Y; y += 4 {
		for x := b.Min.X; x < b.Max.X; x += 4 {
			r, g, bl, _ := img.At(x, y).RGBA()
			if (r+g+bl)/3>>8 > 32 {
				light++
			}
		}
	}
	return light*16*100 < b.Dx()*b.Dy()
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
