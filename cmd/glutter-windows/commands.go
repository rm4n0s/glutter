package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/winetest"
	"github.com/rm4n0s/glutter/internal/wintool"
)

// listFlag is a flag that can be given more than once.
type listFlag []string

func (l *listFlag) String() string     { return strings.Join(*l, ",") }
func (l *listFlag) Set(v string) error { *l = append(*l, v); return nil }

// buildFlags are the flags of the commands that build an .exe.
type buildFlags struct {
	out, arch, name, appID, icon, version string
	release, console                      bool
}

func (b *buildFlags) add(fs *flag.FlagSet, withOutAndArch bool) {
	if withOutAndArch {
		fs.StringVar(&b.out, "o", "", "the .exe to write (default: <package>.exe; with several architectures <package>_<arch>.exe)")
		fs.StringVar(&b.arch, "arch", "", "the architectures to build, comma separated: amd64, arm64 (default: amd64)")
	}
	fs.BoolVar(&b.release, "release", false, "build with -tags glutter_release and -ldflags=-s -w")
	fs.BoolVar(&b.console, "console", false, "build a console program (no -H=windowsgui)")
	fs.StringVar(&b.name, "name", "", "the name of the application in the manifest and the version information (default: the name of the package)")
	fs.StringVar(&b.appID, "app-id", "", "the name of the assembly identity (default: org.glutter.<name of the package>)")
	fs.StringVar(&b.icon, "icon", "", "a square PNG of at least 16 pixels for the icon (default: Glutter's)")
	fs.StringVar(&b.version, "version", "", "the version, 1.2.3.4 (default: 1.0.0.0)")
}

func (b *buildFlags) options(pkg string) wintool.BuildOptions {
	var archs []string
	for _, a := range strings.Split(b.arch, ",") {
		if a = strings.TrimSpace(a); a != "" {
			archs = append(archs, a)
		}
	}
	return wintool.BuildOptions{Package: pkg, Out: b.out, Archs: archs, Release: b.release, Console: b.console,
		Name: b.name, AppID: b.appID, Icon: b.icon, Version: b.version}
}

// parse parses the flags and returns the one package that is left.
func parse(fs *flag.FlagSet, args []string) (string, error) {
	if err := fs.Parse(args); err != nil {
		return "", errors.NewErr(tagWindowsUsage, err)
	}
	if fs.NArg() != 1 {
		return "", errors.New(tagWindowsUsage, "one package is needed, such as ./examples/counter\n"+usage)
	}
	return fs.Arg(0), nil
}

// say prints to the command's output.
func say(w io.Writer, format string, a ...any) error {
	if _, err := fmt.Fprintf(w, format, a...); err != nil {
		return errors.NewErr(tagWindowsUsage, err)
	}
	return nil
}

// runBuild is `glutter-windows build`.
func runBuild(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var bf buildFlags
	bf.add(fs, true)
	pkg, err := parse(fs, args)
	if err != nil {
		return err
	}
	res, err := wintool.Build(ctx, wintool.ExecRunner{}, bf.options(pkg))
	if err != nil {
		return err
	}
	return sayBuilt(stdout, pkg, res)
}

func sayBuilt(w io.Writer, pkg string, res *wintool.BuildResult) error {
	for _, e := range res.Exes {
		if err := say(w, "built %s into %s (%s, %s; %s)\n", pkg, e.Path, e.Arch, size(e.Size), res.AppID); err != nil {
			return err
		}
	}
	return nil
}

func size(n int64) string { return fmt.Sprintf("%.1f MB", float64(n)/(1<<20)) }

// parseEnvs checks the values of -env: GLUTTER_NAME=value.
func parseEnvs(envs []string) ([]string, error) {
	for _, kv := range envs {
		k, _, ok := strings.Cut(kv, "=")
		if !ok || !strings.HasPrefix(k, "GLUTTER_") {
			return nil, errors.New(tagWindowsUsage, "-env takes GLUTTER_NAME=value, got "+kv)
		}
	}
	return envs, nil
}

// buildForHost builds the package for this machine into a temporary directory and returns the .exe
// and the function that removes the directory.
func buildForHost(ctx context.Context, bf *buildFlags, pkg string, stdout io.Writer) (exe string, cleanup func(), err error) {
	cleanup = func() {}
	dir, err := os.MkdirTemp("", "glutter-windows-run-")
	if err != nil {
		return "", cleanup, errors.NewErr(tagWindowsUsage, err)
	}
	cleanup = func() { _ = os.RemoveAll(dir) } // best effort: a temporary directory
	opts := bf.options(pkg)
	opts.Archs = []string{hostArch}
	opts.Out = filepath.Join(dir, "app.exe")
	res, err := wintool.Build(ctx, wintool.ExecRunner{}, opts)
	if err != nil {
		cleanup()
		return "", func() {}, err
	}
	if err := sayBuilt(stdout, pkg, res); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return res.Exes[0].Path, cleanup, nil
}

// runRun is `glutter-windows run`.
func runRun(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var bf buildFlags
	var envs listFlag
	bf.add(fs, false)
	headless := fs.Bool("headless", false, "under Wine, run on a headless X server instead of the display of this session")
	timeout := fs.Duration("timeout", time.Hour, "how long the app may run before it is ended")
	fs.Var(&envs, "env", "an environment variable for the app, GLUTTER_X=v (repeatable)")
	pkg, err := parse(fs, args)
	if err != nil {
		return err
	}
	if _, err := parseEnvs(envs); err != nil {
		return err
	}
	launcher, err := newLauncher(*headless)
	if err != nil {
		return err
	}
	exe, cleanup, err := buildForHost(ctx, &bf, pkg, stdout)
	if err != nil {
		return err
	}
	defer cleanup()
	if err := say(stdout, "running %s; its output is shown when it ends (Ctrl+C stops it)\n", pkg); err != nil {
		return err
	}
	res, err := wintool.Run(ctx, launcher, exe, winetest.RunOptions{Env: envs, Timeout: *timeout})
	if res.Output != "" {
		if serr := say(stdout, "%s", res.Output); serr != nil && err == nil {
			err = serr
		}
	}
	return err
}

// runShot is `glutter-windows shot` (proof method F, SPEC §25.13): it builds the app, runs it with
// GLUTTER_SCREENSHOT and saves the PNG the app saved. With -screen it saves the whole screen of the
// headless X server instead.
func runShot(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("shot", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var bf buildFlags
	var envs listFlag
	bf.add(fs, false)
	fs.Var(&envs, "env", "an environment variable for the app, GLUTTER_X=v (repeatable)")
	screen := fs.Bool("screen", false, "capture the whole screen of the X server (other windows, dialogs) instead of the app's frame")
	settle := fs.Duration("settle", 3*time.Second, "with -screen, how long to wait before the capture")
	timeout := fs.Duration("timeout", 2*time.Minute, "how long to wait for the app")
	out := fs.String("o", "", "the PNG to write")
	pkg, err := parse(fs, args)
	if err != nil {
		return err
	}
	if *out == "" {
		return errors.New(tagWindowsUsage, "shot needs -o file.png")
	}
	if _, err := parseEnvs(envs); err != nil {
		return err
	}
	launcher, err := newLauncher(true)
	if err != nil {
		return err
	}
	exe, cleanup, err := buildForHost(ctx, &bf, pkg, stdout)
	if err != nil {
		return err
	}
	defer cleanup()
	png, output, err := wintool.Shot(ctx, launcher, exe, wintool.ShotOptions{Env: envs, Screen: *screen, Settle: *settle, Timeout: *timeout})
	if err != nil {
		if output != "" {
			if werr := say(stderr, "the output of the app:\n%s\n", output); werr != nil {
				return werr
			}
		}
		return err
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		return errors.NewErr(tagWindowsUsage, err).SetMetadata("path", *out)
	}
	if err := os.WriteFile(*out, png, 0o644); err != nil {
		return errors.NewErr(tagWindowsUsage, err).SetMetadata("path", *out)
	}
	return say(stdout, "saved %s (%d bytes)\n", *out, len(png))
}
