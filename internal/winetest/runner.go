package winetest

import (
	"bytes"
	"context"
	"os/exec"
	"strings"

	"github.com/rm4n0s/errors"
)

// runCommand runs c in a process group of its own and waits. When ctx is done it kills the group.
// A command that ends with a non-zero status is not an error: the status is in the Result. It
// fails with WineStartFailed when the process cannot be started or ctx ends the run.
func runCommand(ctx context.Context, c Command) (Result, error) {
	cmd := exec.Command(c.Argv[0], c.Argv[1:]...)
	cmd.Env = c.Env
	cmd.Dir = c.Dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	setProcessGroup(cmd)
	if err := cmd.Start(); err != nil {
		return Result{}, errors.NewErr(tagWineStartFailed, err).SetMetadata("command", strings.Join(c.Argv, " "))
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	var waitErr error
	cancelled := false
	select {
	case waitErr = <-done:
	case <-ctx.Done():
		cancelled = true
		killGroup(cmd.Process.Pid)
		waitErr = <-done
	}
	// Whatever else the command started in its group (a server that stayed) goes with it.
	killGroup(cmd.Process.Pid)
	res := Result{Output: out.String()}
	var exit *exec.ExitError
	switch {
	case waitErr == nil:
	case errors.As(waitErr, &exit):
		res.ExitCode = exit.ExitCode()
	default:
		return res, errors.NewErr(tagWineStartFailed, waitErr).SetMetadata("command", strings.Join(c.Argv, " "))
	}
	if cancelled {
		return res, errors.NewErr(tagWineStartFailed, ctx.Err()).SetMetadata("command", strings.Join(c.Argv, " "))
	}
	return res, nil
}
