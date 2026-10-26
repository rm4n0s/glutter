package winetest

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
)

// PrefixEnv names the environment variable that moves the Wine prefix, as scripts/wine-exec.sh
// reads it.
//
// Flutter: no counterpart
const PrefixEnv = "GLUTTER_WINEPREFIX"

// DebugEnv names the environment variable that sets WINEDEBUG (the default is -all).
//
// Flutter: no counterpart
const DebugEnv = "GLUTTER_WINEDEBUG"

// WineInstall and XvfbInstall are the commands that install what the harness needs.
const (
	// WineInstall installs Wine on Fedora.
	//
	// Flutter: no counterpart
	WineInstall = "sudo dnf install wine-core"
	// XvfbInstall installs the headless X server on Fedora.
	//
	// Flutter: no counterpart
	XvfbInstall = "sudo dnf install xorg-x11-server-Xvfb"
)

// Command is a command line for the Runner: the program and its arguments, the environment, and
// the directory.
//
// Flutter: no counterpart
type Command struct {
	// Argv is the program and its arguments.
	Argv []string
	// Env is the whole environment of the process (KEY=value).
	Env []string
	// Dir is the working directory; empty means the current one.
	Dir string
}

// Result is what a command left.
//
// Flutter: no counterpart
type Result struct {
	// ExitCode is the exit status of the command; 124 means that `timeout` ended it.
	ExitCode int
	// Output is the standard output and the standard error together.
	Output string
}

// Runner runs a command to its end, in a process group of its own, and kills the group when ctx is
// done. A test gives the Harness a fake one to see the command lines.
//
// Flutter: no counterpart
type Runner func(ctx context.Context, c Command) (Result, error)

// Harness starts Windows programs under Wine. The zero value is not usable: make it with New.
//
// Flutter: no counterpart
type Harness struct {
	// Root is the module root; the prefix and the scripts are below it.
	Root string
	// Prefix is the Wine prefix (WINEPREFIX): Root/.wine-test/prefix, or the value of GLUTTER_WINEPREFIX.
	Prefix string
	// Environ is the environment the commands start from; New sets it to os.Environ().
	Environ []string
	// LookPath finds a program; New sets it to exec.LookPath.
	LookPath func(string) (string, error)
	// Stat tells whether a file exists; New sets it to os.Stat.
	Stat func(string) (os.FileInfo, error)
	// Exec runs the commands; New sets it to the real runner.
	Exec Runner
}

// New makes a Harness for the module at root (found from the working directory when empty). The
// prefix is root/.wine-test/prefix unless GLUTTER_WINEPREFIX is set. It fails with
// ModuleRootNotFound when no go.mod is above the working directory.
//
// Flutter: no counterpart
func New(root string) (*Harness, error) {
	if root == "" {
		dir, err := os.Getwd()
		if err != nil {
			return nil, errors.NewErr(tagModuleRootNotFound, err)
		}
		for {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				root = dir
				break
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				return nil, errors.New(tagModuleRootNotFound, "there is no go.mod above the working directory", "dir", dir)
			}
			dir = parent
		}
	}
	h := &Harness{Root: root, Environ: os.Environ(), LookPath: exec.LookPath, Stat: os.Stat, Exec: runCommand}
	h.Prefix = PrefixDir(root, h.Environ)
	return h, nil
}

// PrefixDir is the prefix for a module root: the value of GLUTTER_WINEPREFIX in environ, else
// root/.wine-test/prefix.
//
// Flutter: no counterpart
func PrefixDir(root string, environ []string) string {
	if v := envValue(environ, PrefixEnv); v != "" {
		return v
	}
	return filepath.Join(root, ".wine-test", "prefix")
}

// envValue is the value of key in a KEY=value list, or "".
func envValue(environ []string, key string) string {
	for i := len(environ) - 1; i >= 0; i-- {
		if v, ok := strings.CutPrefix(environ[i], key+"="); ok {
			return v
		}
	}
	return ""
}

// lavapipeICD is the ICD file of Mesa's software Vulkan driver, the only one that presents on Xvfb
// (SPEC §25.13).
const lavapipeICD = "/usr/share/vulkan/icd.d/lvp_icd.x86_64.json"

// Environment is the environment of a Wine command (SPEC §25.13): the environment of the harness
// without WAYLAND_DISPLAY and WAYLAND_SOCKET, plus WINEPREFIX, WINEARCH=win64, WINEDLLOVERRIDES
// ("mscoree,mshtml=", so that Wine does not offer to download Mono or Gecko), WINEDEBUG (-all unless
// GLUTTER_WINEDEBUG is set) and VK_DRIVER_FILES pointing at lavapipe's ICD when that file exists and
// the variable is not set. extra are added last (KEY=value).
//
// Flutter: no counterpart
func (h *Harness) Environment(extra ...string) []string {
	var env []string
	for _, kv := range h.Environ {
		key, _, _ := strings.Cut(kv, "=")
		switch key {
		case "WAYLAND_DISPLAY", "WAYLAND_SOCKET", "WINEPREFIX", "WINEARCH", "WINEDLLOVERRIDES", "WINEDEBUG":
			continue
		}
		env = append(env, kv)
	}
	debug := envValue(h.Environ, DebugEnv)
	if debug == "" {
		debug = "-all"
	}
	env = append(env, "WINEPREFIX="+h.Prefix, "WINEARCH=win64", "WINEDLLOVERRIDES=mscoree,mshtml=", "WINEDEBUG="+debug)
	if envValue(h.Environ, "VK_DRIVER_FILES") == "" {
		if _, err := h.Stat(lavapipeICD); err == nil {
			env = append(env, "VK_DRIVER_FILES="+lavapipeICD)
		}
	}
	return append(env, extra...)
}

// shellPrefix starts a command in a shell that lowers the core-file limit to one byte, so that a
// crash of Wine never writes a core file or reaches the system's crash reporter (SPEC §25.13).
var shellPrefix = []string{"sh", "-c", `ulimit -c 1; exec "$@"`, "sh"}

// wineProgram finds the Wine launcher (wine64, else wine) and wineserver. It fails with WineNotFound.
func (h *Harness) wineProgram() (wine, wineserver string, err error) {
	for _, name := range []string{"wine64", "wine"} {
		if wine, err = h.LookPath(name); err == nil {
			break
		}
	}
	if err != nil {
		return "", "", errors.NewErr(tagWineNotFound, err).SetMetadata("install", WineInstall)
	}
	if wineserver, err = h.LookPath("wineserver"); err != nil {
		return "", "", errors.NewErr(tagWineNotFound, err).SetMetadata("install", WineInstall)
	}
	return wine, wineserver, nil
}

// headless is the command line that runs argv under a headless X server: `timeout -k 10s <n>s
// scripts/headless-x11.sh -- argv...`, inside the shell that limits core files.
func (h *Harness) headless(timeout time.Duration, argv ...string) []string {
	secs := strconv.Itoa(max(1, int(timeout.Seconds())))
	cmd := append([]string{}, shellPrefix...)
	cmd = append(cmd, "timeout", "-k", "10s", secs+"s", filepath.Join(h.Root, "scripts", "headless-x11.sh"), "--")
	return append(cmd, argv...)
}

// Stop stops the Wine server of the prefix and everything it started (`wineserver -k`; never
// `wineserver -w`, because winedevice stays alive). It is safe to call when nothing runs.
//
// Flutter: no counterpart
func (h *Harness) Stop(ctx context.Context) error {
	_, wineserver, err := h.wineProgram()
	if err != nil {
		return err
	}
	// wineserver -k exits with a non-zero status when no server runs for the prefix, which is the
	// state we want, so the status is not read.
	_, err = h.Exec(ctx, Command{Argv: []string{wineserver, "-k"}, Env: h.Environment(), Dir: h.Root})
	return err
}

// EnsurePrefix makes the Wine prefix if it is not there (system.reg is its mark), under a lock
// file, so that tests of different packages that start together make it once. The first boot of a
// new prefix hangs when the wine-cms package is not installed (setupapi loops on the missing
// mscms.dll), so the boot is cut after 20 seconds and the server stopped: `timeout 20 wine wineboot
// --init`, then `wineserver -k` (SPEC §25.13). It fails with WineNotFound, and with
// WinePrefixFailed when the prefix is still not there afterwards.
//
// Flutter: no counterpart
func (h *Harness) EnsurePrefix(ctx context.Context) error {
	if h.prefixReady() {
		return nil
	}
	wine, _, err := h.wineProgram()
	if err != nil {
		return err
	}
	if _, err := h.LookPath("Xvfb"); err != nil {
		return errors.NewErr(tagWineStartFailed, err).SetMetadata("install", XvfbInstall)
	}
	if err := os.MkdirAll(filepath.Dir(h.Prefix), 0o755); err != nil {
		return errors.NewErr(tagWinePrefixFailed, err).SetMetadata("prefix", h.Prefix)
	}
	unlock, err := lockFile(h.Prefix + ".lock")
	if err != nil {
		return err
	}
	defer unlock()
	if h.prefixReady() { // another test made it while this one waited
		return nil
	}
	boot := h.headless(90*time.Second, "sh", "-c", `timeout 20 "$1" wineboot --init; true`, "sh", wine)
	res, err := h.Exec(ctx, Command{Argv: boot, Env: h.Environment(), Dir: h.Root})
	stopErr := h.Stop(ctx)
	if err != nil {
		return err
	}
	if stopErr != nil {
		return stopErr
	}
	if !h.prefixReady() {
		return errors.New(tagWinePrefixFailed, "wineboot did not make the prefix", "prefix", h.Prefix, "output", res.Output)
	}
	return nil
}

// prefixReady is whether the prefix has been booted: system.reg exists.
func (h *Harness) prefixReady() bool {
	_, err := h.Stat(filepath.Join(h.Prefix, "system.reg"))
	return err == nil
}

// RunOptions are the optional parts of Run.
//
// Flutter: no counterpart
type RunOptions struct {
	// Args are the arguments of the program.
	Args []string
	// Env are more environment variables (KEY=value), such as GLUTTER_SCREENSHOT=...
	Env []string
	// Dir is the working directory of the program; empty means the module root.
	Dir string
	// Timeout is how long the program may run; zero means 120 seconds. When it passes, `timeout`
	// ends the program and the exit code is 124.
	Timeout time.Duration
	// OnDisplay starts the program on the display of the environment (DISPLAY) instead of a headless
	// X server: the window shows on the user's screen. Tests never set it; `glutter-windows run`
	// does when it is not given -headless.
	OnDisplay bool
}

// Run starts the Windows program exe under Wine on a headless X server and waits for it. The prefix
// is made first if it is not there. The Result has the exit code and the output of the program. A
// non-zero exit code is not an error: the test reads it. Wine and the server are stopped afterwards
// with `wineserver -k`, also when ctx ends the run early. It fails with WineNotFound, with
// WinePrefixFailed, and with WineStartFailed when the process cannot be started or ctx ends.
//
// Flutter: no counterpart
func (h *Harness) Run(ctx context.Context, exe string, opts RunOptions) (Result, error) {
	return h.start(ctx, opts, func(wine string, timeout time.Duration) []string {
		argv := append([]string{wine, exe}, opts.Args...)
		if opts.OnDisplay {
			return h.onDisplay(timeout, argv...)
		}
		return h.headless(timeout, argv...)
	})
}

// Capture starts the Windows program exe under Wine on a headless X server, as Run does, and saves
// the root window of the server to the PNG file out after GLUTTER_SHOT_DELAY seconds (3 unless the
// environment of opts says another value), with scripts/wine-screenshot.sh (proof method F, SPEC
// §25.13): it shows what lies outside the program's own window, such as a dialog. The script ends
// the program afterwards. A script status other than 0 (the program ended before the capture, or the
// capture failed) is in the Result. opts.OnDisplay is ignored. It fails as Run does.
//
// Flutter: no counterpart
func (h *Harness) Capture(ctx context.Context, exe, out string, opts RunOptions) (Result, error) {
	return h.start(ctx, opts, func(wine string, timeout time.Duration) []string {
		secs := strconv.Itoa(max(1, int(timeout.Seconds())))
		argv := append([]string{}, shellPrefix...)
		argv = append(argv, "timeout", "-k", "10s", secs+"s", filepath.Join(h.Root, "scripts", "wine-screenshot.sh"), out, "--", wine, exe)
		return append(argv, opts.Args...)
	})
}

// start makes the prefix, runs the command line that argv builds for the Wine launcher and the
// timeout, and stops Wine whatever happened.
func (h *Harness) start(ctx context.Context, opts RunOptions, argv func(wine string, timeout time.Duration) []string) (Result, error) {
	wine, _, err := h.wineProgram()
	if err != nil {
		return Result{}, err
	}
	if err := h.EnsurePrefix(ctx); err != nil {
		return Result{}, err
	}
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}
	dir := opts.Dir
	if dir == "" {
		dir = h.Root
	}
	res, runErr := h.Exec(ctx, Command{Argv: argv(wine, timeout), Env: h.Environment(opts.Env...), Dir: dir})
	// Stop whatever the program left, whether it ended or not. The context may be done already:
	// stopping must still happen, so it gets a context of its own.
	stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	if err := h.Stop(stopCtx); err != nil && runErr == nil {
		return res, err
	}
	return res, runErr
}

// onDisplay is the command line that runs argv on the display of the environment: `timeout -k 10s
// <n>s argv...`, inside the shell that limits core files.
func (h *Harness) onDisplay(timeout time.Duration, argv ...string) []string {
	secs := strconv.Itoa(max(1, int(timeout.Seconds())))
	cmd := append([]string{}, shellPrefix...)
	cmd = append(cmd, "timeout", "-k", "10s", secs+"s")
	return append(cmd, argv...)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagModuleRootNotFound = "ModuleRootNotFound"
	tagWineNotFound       = TagWineNotFound
	tagWinePrefixFailed   = TagWinePrefixFailed
	tagWineStartFailed    = TagWineStartFailed

	// TagWineNotFound is the tag of the error for a Wine that is not installed; the metadata
	// "install" has the command.
	TagWineNotFound = "WineNotFound"
	// TagWinePrefixFailed is the tag of the error for a prefix that cannot be made.
	TagWinePrefixFailed = "WinePrefixFailed"
	// TagWineStartFailed is the tag of the error for a program, a server or a display that cannot
	// be started.
	TagWineStartFailed = "WineStartFailed"
)
