package main

import (
	"context"
	"flag"
	"io"
	"os"
	"os/exec"

	"github.com/rm4n0s/errors"
)

// runRun is `glutter-macos run`: it builds the bundle of a package in a temporary directory and starts its executable (inside the
// bundle, so the app has its bundle: the identifier, the Info.plist and the icon), with the log on the terminal. The app is in a
// process group of its own that an interrupt kills, and the bundle is removed when it ends.
func runRun(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	var bf buildFlags
	var env envFlags
	bf.add(fs, false)
	fs.Var(&env, "env", "an environment variable for the app, NAME=value (repeatable)")
	pkg, err := parseWith(fs, args, stderr)
	if err != nil {
		return err
	}
	dir, cleanup, err := tempDir("glutter-macos-run-")
	if err != nil {
		return err
	}
	defer cleanup()
	res, err := buildBundle(ctx, pkg, &bf, dir)
	if err != nil {
		return err
	}
	cmd := exec.Command(res.Layout.Executable)
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout, cmd.Stderr = stdout, stderr
	ownProcessGroup(cmd)
	if err := cmd.Start(); err != nil {
		return errors.NewErr(tagMacOSRunFailed, err).SetMetadata("package", pkg)
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
			return errors.NewErr(tagMacOSRunFailed, err).SetMetadata("package", pkg)
		}
	case <-ctx.Done():
		killGroup(cmd)
		<-done // killed: the exit status is the kill's, not the app's
	}
	return nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const tagMacOSRunFailed = "MacOSRunFailed"
