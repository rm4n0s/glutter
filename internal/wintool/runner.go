package wintool

import (
	"context"
	"os/exec"
	"strings"

	"github.com/rm4n0s/errors"
)

// Command is one program the builder runs.
//
// Flutter: no counterpart
type Command struct {
	// Dir is the working directory; the current one when empty.
	Dir string
	// Env is the complete environment; the current one when nil.
	Env []string
	// Name is the program, found on the PATH when it has no directory.
	Name string
	// Args are the arguments.
	Args []string
}

// String is the command line, for messages.
func (c Command) String() string { return strings.Join(append([]string{c.Name}, c.Args...), " ") }

// Runner runs the programs of a build (the go command). A test gives the builder a fake one and checks
// the command lines.
//
// Flutter: no counterpart
type Runner interface {
	// Run runs the command and returns its combined output. A command that fails is an error tagged
	// WindowsCommandFailed, with the command line and its output in the metadata.
	Run(ctx context.Context, c Command) (string, error)
}

// ExecRunner is the Runner that runs the programs with os/exec.
//
// Flutter: no counterpart
type ExecRunner struct{}

// Run implements Runner.
func (ExecRunner) Run(ctx context.Context, c Command) (string, error) {
	cmd := exec.CommandContext(ctx, c.Name, c.Args...)
	cmd.Dir, cmd.Env = c.Dir, c.Env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), errors.NewErr(TagWindowsCommandFailed, err).SetMetadata("command", c.String(), "output", string(out))
	}
	return string(out), nil
}

// TagWindowsCommandFailed is the tag of the error for a program of the build that cannot be started or
// exits with a failure; the metadata "command" and "output" say which and why.
const TagWindowsCommandFailed = "WindowsCommandFailed"
