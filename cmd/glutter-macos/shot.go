package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
)

// envFlags collects the -env GLUTTER_X=v flags.
type envFlags []string

func (e *envFlags) String() string { return strings.Join(*e, " ") }
func (e *envFlags) Set(v string) error {
	if !strings.Contains(v, "=") {
		return errors.New(tagMacOSUsage, "-env takes NAME=value", "value", v)
	}
	*e = append(*e, v)
	return nil
}

// shotOptions are what `shot` was asked for.
type shotOptions struct {
	pkg     string
	out     string
	env     []string
	front   bool
	timeout time.Duration
	// bundle runs the app from its bundle (built with the flags of bf) and not as a bare program.
	bundle bool
	bf     buildFlags
}

// parseShot reads the flags of `shot`.
func parseShot(args []string, stderr io.Writer) (shotOptions, error) {
	fs := flag.NewFlagSet("shot", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var env envFlags
	var o shotOptions
	fs.Var(&env, "env", "an environment variable for the app, NAME=value (repeatable)")
	fs.BoolVar(&o.front, "front", false, "run the app as a regular app, in front, instead of an accessory app that never takes the focus")
	fs.DurationVar(&o.timeout, "timeout", 90*time.Second, "how long the app may take")
	fs.StringVar(&o.out, "o", "", "the PNG file to write (required)")
	fs.BoolVar(&o.bundle, "bundle", false, "run the app from its signed bundle, as `run` does, and not as a bare program")
	o.bf.add(fs, false)
	// shot makes a debug build unless told otherwise: the hooks it drives (GLUTTER_DEV_*) are read only by debug builds.
	o.bf.debug = true
	fs.Lookup("debug").DefValue = "true"
	if err := fs.Parse(args); err != nil {
		return o, errors.NewErr(tagMacOSUsage, err)
	}
	if o.out == "" || fs.NArg() != 1 {
		return o, errors.New(tagMacOSUsage, "shot needs -o file.png and one package\n"+usage)
	}
	o.pkg, o.env = fs.Arg(0), env
	return o, nil
}

// shotEnv is the environment of the app for a screenshot: the self-capture file, the accessory policy
// (unless -front), and the variables of -env last, so that they win.
func shotEnv(base []string, o shotOptions, abs string) []string {
	env := append([]string(nil), base...)
	env = append(env, "GLUTTER_SCREENSHOT="+abs)
	if !o.front {
		env = append(env, "GLUTTER_MACOS_ACCESSORY=1")
	}
	return append(env, o.env...)
}

// runShot builds the package for the host Mac without cgo and runs it with GLUTTER_SCREENSHOT: the app
// saves its own first settled frame and ends. The program runs in a process group of its own, which a
// timeout or an interrupt kills (nothing is left running).
func runShot(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	o, err := parseShot(args, stderr)
	if err != nil {
		return err
	}
	abs, err := filepath.Abs(o.out)
	if err != nil {
		return errors.NewErr(tagMacOSShotFailed, err).SetMetadata("file", o.out)
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return errors.NewErr(tagMacOSShotFailed, err).SetMetadata("file", abs)
	}
	tmp, err := os.MkdirTemp("", "glutter-macos-shot-")
	if err != nil {
		return errors.NewErr(tagMacOSShotFailed, err)
	}
	defer func() { _ = os.RemoveAll(tmp) }() // best effort: a temporary directory
	exe := filepath.Join(tmp, "app")
	if o.bundle {
		res, err := buildBundle(ctx, o.pkg, &o.bf, tmp)
		if err != nil {
			return err
		}
		exe = res.Layout.Executable
	} else {
		var buildOut bytes.Buffer
		args := []string{"build", "-o", exe}
		if o.bf.debug {
			args = append(args, "-tags", "debug_mode")
		} else {
			args = append(args, "-ldflags=-s -w")
		}
		build := exec.CommandContext(ctx, "go", append(args, o.pkg)...)
		build.Env = append(os.Environ(), "CGO_ENABLED=0")
		build.Stdout, build.Stderr = &buildOut, &buildOut
		if err := build.Run(); err != nil {
			return errors.NewErr(tagMacOSShotFailed, err).SetMetadata("package", o.pkg, "output", buildOut.String())
		}
	}
	_ = os.Remove(abs) // a stale file must not pass for the new one

	ctx, cancel := context.WithTimeout(ctx, o.timeout)
	defer cancel()
	cmd := exec.Command(exe)
	cmd.Env = shotEnv(os.Environ(), o, abs)
	cmd.Stdout, cmd.Stderr = stderr, stderr
	ownProcessGroup(cmd)
	if err := cmd.Start(); err != nil {
		return errors.NewErr(tagMacOSShotFailed, err).SetMetadata("package", o.pkg)
	}
	var waitErr error
	done := make(chan struct{}) // closed when the process has ended; waitErr is then its exit status
	go func() {
		waitErr = cmd.Wait()
		close(done)
	}()
	select {
	case <-done:
		if err := waitErr; err != nil {
			return errors.NewErr(tagMacOSShotFailed, err).SetMetadata("package", o.pkg)
		}
	case <-ctx.Done():
		killGroup(cmd)
		<-done // killed: the exit status is the kill's, not the app's
		return errors.New(tagMacOSShotTimeout, "the app did not save its screenshot in time", "package", o.pkg, "timeout", o.timeout.String())
	}
	if _, err := os.Stat(abs); err != nil {
		return errors.NewErr(tagMacOSShotFailed, err).SetMetadata("package", o.pkg, "file", abs)
	}
	_, err = fmt.Fprintln(stdout, abs)
	if err != nil {
		return errors.NewErr(tagMacOSShotFailed, err)
	}
	return nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagMacOSShotFailed  = "MacOSShotFailed"
	tagMacOSShotTimeout = "MacOSShotTimeout"
)
