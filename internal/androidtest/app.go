package androidtest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/androidtool"
)

// BuildAPK builds a package of the module for x86_64 (the emulator's ABI) into dir and returns
// the APK and its application ID. The debug key is the harness's own, in root/user, so that the
// tests never touch the developer's config directory. Extra options (a permission, an app ID) go
// in opts.
//
// Flutter: no counterpart
func (e *Emulator) BuildAPK(ctx context.Context, pkg, dir string, opts androidtool.BuildOptions) (apk, appID string, err error) {
	sdk, err := androidtool.FindSDK(os.Getenv)
	if err != nil {
		return "", "", err
	}
	mod, err := ModuleRoot()
	if err != nil {
		return "", "", err
	}
	opts.Package = pkg
	opts.Dir = mod
	opts.ABIs = []string{"amd64"}
	opts.Out = filepath.Join(dir, filepath.Base(pkg)+".apk")
	opts.ConfigDir = filepath.Join(e.Root, "debug-key")
	res, err := androidtool.Build(ctx, sdk, androidtool.ExecRunner{}, opts)
	if err != nil {
		return "", "", err
	}
	return res.APK, res.AppID, nil
}

// Install installs the APK over any earlier version.
func (e *Emulator) Install(ctx context.Context, apk string) error { return e.ADB.Install(ctx, apk) }

// Launch clears the log, stops the app if it runs and starts it with the given extras (the
// GLUTTER_ environment of the program).
//
// Right after the boot of an old image (API 26) the log cannot be cleared for a while, so clearing it
// is tried again for up to two minutes.
func (e *Emulator) Launch(ctx context.Context, appID string, extras map[string]string) error {
	var err error
	for end := time.Now().Add(2 * time.Minute); ; time.Sleep(2 * time.Second) {
		if _, err = e.ADB.Output(ctx, "logcat", "-c"); err == nil {
			break
		}
		if time.Now().After(end) || ctx.Err() != nil {
			return err
		}
	}
	return e.ADB.Start(ctx, appID, extras)
}

// WaitExit waits until the app has no process, for at most timeout. It fails with
// AndroidTestTimeout when the app still runs.
func (e *Emulator) WaitExit(ctx context.Context, appID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		pid, err := e.ADB.Pid(ctx, appID)
		if err != nil {
			return err
		}
		if pid == 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return errors.New(TagAndroidTestTimeout, "the app still runs", "appID", appID, "pid", pid, "timeout", timeout.String())
		}
		select {
		case <-ctx.Done():
			return errors.NewErr(TagAndroidTestTimeout, ctx.Err()).SetMetadata("appID", appID)
		case <-time.After(300 * time.Millisecond):
		}
	}
}

// WaitPid waits until the app has a process and returns its ID.
func (e *Emulator) WaitPid(ctx context.Context, appID string, timeout time.Duration) (int, error) {
	deadline := time.Now().Add(timeout)
	for {
		pid, err := e.ADB.Pid(ctx, appID)
		if err != nil {
			return 0, err
		}
		if pid != 0 {
			return pid, nil
		}
		if time.Now().After(deadline) {
			return 0, errors.New(TagAndroidTestTimeout, "the app has no process", "appID", appID, "timeout", timeout.String())
		}
		select {
		case <-ctx.Done():
			return 0, errors.NewErr(TagAndroidTestTimeout, ctx.Err()).SetMetadata("appID", appID)
		case <-time.After(300 * time.Millisecond):
		}
	}
}

// Log is what the program of the app printed since Launch: the lines logcat has under the app's
// tag (the application ID, SPEC §24.9).
func (e *Emulator) Log(ctx context.Context, appID string) (string, error) {
	return e.ADB.Output(ctx, "logcat", "-d", "-v", "brief", "-s", appID+":V")
}

// FullLog is the whole log since Launch, every tag: crashes of the runtime and the system's
// lines about the app included.
func (e *Emulator) FullLog(ctx context.Context) (string, error) {
	return e.ADB.Output(ctx, "logcat", "-d", "-v", "brief")
}

// WaitLog waits until the app's log has a line that contains want, for at most timeout, and
// returns the log.
func (e *Emulator) WaitLog(ctx context.Context, appID, want string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	for {
		log, err := e.Log(ctx, appID)
		if err != nil {
			return "", err
		}
		if strings.Contains(log, want) {
			return log, nil
		}
		if time.Now().After(deadline) {
			return log, errors.New(TagAndroidTestTimeout, "the log never had the line", "appID", appID, "want", want, "log", log)
		}
		select {
		case <-ctx.Done():
			return log, errors.NewErr(TagAndroidTestTimeout, ctx.Err()).SetMetadata("appID", appID, "want", want)
		case <-time.After(300 * time.Millisecond):
		}
	}
}

// HasActivity says whether the system still has an activity of the app (`dumpsys activity
// activities`).
func (e *Emulator) HasActivity(ctx context.Context, appID string) (bool, error) {
	out, err := e.ADB.Output(ctx, "shell", "dumpsys", "activity", "activities")
	if err != nil {
		return false, err
	}
	return strings.Contains(out, appID+"/"+androidtool.ActivityClass), nil
}

// RunAs runs a command as the app (`run-as`, which a debuggable app allows) and returns its
// output: the way to read the files an app wrote.
func (e *Emulator) RunAs(ctx context.Context, appID string, command ...string) (string, error) {
	args := append([]string{"exec-out", "run-as", appID}, command...)
	return e.ADB.Output(ctx, args...)
}

// TagAndroidTestTimeout is the tag of the error for something a device test waited for that did
// not happen in time.
const TagAndroidTestTimeout = "AndroidTestTimeout"

// ScreenshotFile is where the self-capture of an app writes, below the app's directory.
const ScreenshotFile = "files/glutter-screenshot.png"

// Screenshot starts the app with GLUTTER_SCREENSHOT=1 (and the given extras), waits for it to save
// the frame of its first idle moment and end, and returns the PNG (proof method E, SPEC §24.13).
// A test that gets no file reads the app's log with Log.
func (e *Emulator) Screenshot(ctx context.Context, appID string, extras map[string]string, timeout time.Duration) ([]byte, error) {
	if err := e.ADB.RemoveAppFile(ctx, appID, ScreenshotFile); err != nil {
		return nil, err
	}
	all := map[string]string{"GLUTTER_SCREENSHOT": "1"}
	for k, v := range extras {
		all[k] = v
	}
	if err := e.Launch(ctx, appID, all); err != nil {
		return nil, err
	}
	if err := e.WaitExit(ctx, appID, timeout); err != nil {
		return nil, err
	}
	return e.ADB.PullAppFile(ctx, appID, ScreenshotFile)
}

// Describe is the text of an error for a failed test: its tag, route and metadata (the command
// and its output), which err.Error() leaves out.
func Describe(err error) string {
	appErr, ok := errors.FromError(err)
	if !ok {
		return err.Error()
	}
	return fmt.Sprintf("%s: %s\nroute: %s\nmetadata: %v", appErr.Tag, appErr.Message, appErr.Route(), appErr.Metadata)
}

// PullAppDir reads every file of a directory of the app's private directory (`run-as ls`, then
// `run-as cat` of each), keyed by name. The directory has no subdirectories.
func (e *Emulator) PullAppDir(ctx context.Context, appID, dir string) (map[string][]byte, error) {
	out, err := e.ADB.Output(ctx, "exec-out", "run-as", appID, "ls", dir)
	if err != nil {
		return nil, err
	}
	files := map[string][]byte{}
	for _, name := range strings.Fields(out) {
		b, err := e.ADB.PullAppFile(ctx, appID, dir+"/"+name)
		if err != nil {
			return nil, err
		}
		files[name] = b
	}
	return files, nil
}

// ValidationLayerDir is the directory with the Khronos validation layer of Android, where the
// device tests of Vulkan expect it: <module root>/.android-test/vvl, with
// x86_64/libVkLayer_khronos_validation.so in it (the android-binaries zip of a
// Vulkan-ValidationLayers release, unzipped there; SPEC §24.13). It fails with
// AndroidTestValidationLayerMissing, with that instruction, when the library is not there.
//
// Flutter: no counterpart
func ValidationLayerDir() (string, error) {
	root, err := ModuleRoot()
	if err != nil {
		return "", err
	}
	return validationLayerDir(root)
}

func validationLayerDir(root string) (string, error) {
	dir := filepath.Join(root, ".android-test", "vvl")
	layer := filepath.Join(dir, "x86_64", androidtool.ValidationLayerFile)
	if _, err := os.Stat(layer); err != nil {
		return "", errors.NewErr(TagAndroidTestValidationLayerMissing, err).SetMetadata("path", layer,
			"install", "unzip the android-binaries-*.zip of the newest release at github.com/KhronosGroup/Vulkan-ValidationLayers/releases into "+dir)
	}
	return dir, nil
}

// TagAndroidTestValidationLayerMissing is the tag of the error for a device test of Vulkan that
// needs the validation layer for Android and does not find it; the metadata "install" says how to
// get it.
const TagAndroidTestValidationLayerMissing = "AndroidTestValidationLayerMissing"
