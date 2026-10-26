package wintool

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/winetest"
)

// Launcher starts a Windows program and waits for it: under Wine on Linux (WineLauncher), or by
// itself on Windows (LocalLauncher).
//
// Flutter: no counterpart
type Launcher interface {
	// Launch runs the .exe and returns its exit code and output. A program that exits with a failure
	// is not an error: the status is in the Result.
	Launch(ctx context.Context, exe string, o winetest.RunOptions) (winetest.Result, error)
	// ProgramPath is a file of the host as the program sees it: Z:\tmp\x.png under Wine, the path
	// itself on Windows.
	ProgramPath(hostPath string) string
}

// ScreenLauncher is a Launcher that can also save the whole screen the program runs on.
//
// Flutter: no counterpart
type ScreenLauncher interface {
	Launcher
	// Capture runs the .exe and saves the root window of its display to the PNG out after
	// GLUTTER_SHOT_DELAY seconds (the environment of o can set it).
	Capture(ctx context.Context, exe, out string, o winetest.RunOptions) (winetest.Result, error)
}

// WineLauncher runs a program under Wine through the harness of internal/winetest: on a headless X
// server, or, without Headless, on the display of the environment (the window shows on the
// screen).
//
// Flutter: no counterpart
type WineLauncher struct {
	// Harness starts Wine.
	Harness *winetest.Harness
	// Headless runs the program on Xvfb, not on the user's display.
	Headless bool
}

// Launch implements Launcher.
func (w WineLauncher) Launch(ctx context.Context, exe string, o winetest.RunOptions) (winetest.Result, error) {
	o.OnDisplay = !w.Headless
	return w.Harness.Run(ctx, exe, o)
}

// Capture implements ScreenLauncher: it runs the program on a headless X server whatever Headless
// says, because the capture takes the X server's root window.
func (w WineLauncher) Capture(ctx context.Context, exe, out string, o winetest.RunOptions) (winetest.Result, error) {
	return w.Harness.Capture(ctx, exe, out, o)
}

// ProgramPath implements Launcher: the host's root is the drive Z: of Wine.
func (WineLauncher) ProgramPath(hostPath string) string {
	return `Z:` + strings.ReplaceAll(hostPath, "/", `\`)
}

// LocalLauncher runs the program itself, which is what `run` and `shot` do on Windows.
//
// Flutter: no counterpart
type LocalLauncher struct{}

// Launch implements Launcher. The program ends with status 124 when o.Timeout (120 seconds when
// zero) passes, as it does under Wine.
func (LocalLauncher) Launch(ctx context.Context, exe string, o winetest.RunOptions) (winetest.Result, error) {
	timeout := o.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, exe, o.Args...)
	cmd.Env = append(os.Environ(), o.Env...)
	cmd.Dir = o.Dir
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	res := winetest.Result{Output: out.String()}
	var exit *exec.ExitError
	switch {
	case err == nil:
	case ctx.Err() != nil:
		return res, errors.NewErr(tagWindowsRunFailed, ctx.Err()).SetMetadata("exe", exe)
	case runCtx.Err() != nil:
		res.ExitCode = 124
	case errors.As(err, &exit):
		res.ExitCode = exit.ExitCode()
	default:
		return res, errors.NewErr(tagWindowsRunFailed, err).SetMetadata("exe", exe)
	}
	return res, nil
}

// ProgramPath implements Launcher.
func (LocalLauncher) ProgramPath(hostPath string) string { return hostPath }

// Run starts the .exe with the launcher and waits for it (`glutter-windows run`). The Result is
// returned also when the program fails, so that its output can be shown. It fails with
// WindowsRunFailed when the program cannot be started or does not end with status 0.
//
// Flutter: no counterpart (flutter run -d windows)
func Run(ctx context.Context, l Launcher, exe string, o winetest.RunOptions) (winetest.Result, error) {
	res, err := l.Launch(ctx, exe, o)
	if err != nil {
		return res, err
	}
	if res.ExitCode != 0 {
		return res, errors.New(tagWindowsRunFailed, "the program ended with a failure", "exe", exe, "exitCode", res.ExitCode)
	}
	return res, nil
}

// ShotOptions are the parameters of Shot.
//
// Flutter: no counterpart
type ShotOptions struct {
	// Env are more environment variables for the app (KEY=value, GLUTTER_X=v).
	Env []string
	// Screen captures the whole screen of the X server (title bar, other windows, dialogs) instead of
	// the app's own frame. It needs a ScreenLauncher.
	Screen bool
	// Settle is how long to wait before the capture of the screen (3 seconds when zero); it is the
	// value of GLUTTER_SHOT_DELAY.
	Settle time.Duration
	// Timeout is how long the app may run (2 minutes when zero).
	Timeout time.Duration
}

// Shot is proof method F (SPEC §25.13): it runs the .exe with GLUTTER_SCREENSHOT, so that the app
// saves its own first frame as a PNG and ends, and returns the PNG together with the output of the
// app. With o.Screen it captures the root window of the X server instead. It fails with
// WindowsShotFailed (the app failed, or saved no frame) and WindowsScreenUnsupported (Screen with a
// launcher that cannot).
//
// Flutter: no counterpart (flutter screenshot)
func Shot(ctx context.Context, l Launcher, exe string, o ShotOptions) (png []byte, output string, err error) {
	timeout := o.Timeout
	if timeout == 0 {
		timeout = 2 * time.Minute
	}
	dir, err := os.MkdirTemp("", "glutter-shot-")
	if err != nil {
		return nil, "", errors.NewErr(tagWindowsShotFailed, err).SetMetadata("what", "the temporary directory")
	}
	defer func() { _ = os.RemoveAll(dir) }() // best effort: a temporary directory
	file := filepath.Join(dir, "shot.png")
	opts := winetest.RunOptions{Env: append([]string(nil), o.Env...), Timeout: timeout}
	var res winetest.Result
	if o.Screen {
		sl, ok := l.(ScreenLauncher)
		if !ok {
			return nil, "", errors.New(tagWindowsScreenUnsupported, "this launcher cannot capture the screen: -screen needs Wine on Linux")
		}
		settle := o.Settle
		if settle <= 0 {
			settle = 3 * time.Second
		}
		opts.Env = append(opts.Env, "GLUTTER_SHOT_DELAY="+strconv.Itoa(max(1, int(settle.Round(time.Second).Seconds()))))
		if res, err = sl.Capture(ctx, exe, file, opts); err != nil {
			return nil, res.Output, err
		}
	} else {
		opts.Env = append(opts.Env, "GLUTTER_SCREENSHOT="+l.ProgramPath(file))
		if res, err = l.Launch(ctx, exe, opts); err != nil {
			return nil, res.Output, err
		}
	}
	if res.ExitCode != 0 {
		return nil, res.Output, errors.New(tagWindowsShotFailed, "the app ended with a failure before it saved its frame", "exe", exe, "exitCode", res.ExitCode, "output", res.Output)
	}
	png, err = os.ReadFile(file)
	if err != nil {
		return nil, res.Output, errors.NewErr(tagWindowsShotFailed, err).SetMetadata("exe", exe, "output", res.Output)
	}
	return png, res.Output, nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagWindowsRunFailed         = TagWindowsRunFailed
	tagWindowsShotFailed        = TagWindowsShotFailed
	tagWindowsScreenUnsupported = TagWindowsScreenUnsupported

	// TagWindowsRunFailed is the tag of the error for a program that cannot be started, or that
	// ends with a status other than 0.
	TagWindowsRunFailed = "WindowsRunFailed"
	// TagWindowsShotFailed is the tag of the error for a shot that failed: the app ended with a
	// failure, or it saved no frame.
	TagWindowsShotFailed = "WindowsShotFailed"
	// TagWindowsScreenUnsupported is the tag of the error for a request to capture the screen on
	// a launcher that cannot (Windows runs the program itself).
	TagWindowsScreenUnsupported = "WindowsScreenUnsupported"
)
