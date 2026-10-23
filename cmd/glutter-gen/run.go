package main

import (
	"bytes"
	"fmt"
	"log/slog"
	"sort"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/cmd/glutter-gen/internal/androiddex"
	"github.com/rm4n0s/glutter/cmd/glutter-gen/internal/apicheck"
	"github.com/rm4n0s/glutter/cmd/glutter-gen/internal/copywith"
	"github.com/rm4n0s/glutter/cmd/glutter-gen/internal/errcheck"
	"github.com/rm4n0s/glutter/cmd/glutter-gen/internal/icons"
	"github.com/rm4n0s/glutter/cmd/glutter-gen/internal/keys"
	"github.com/rm4n0s/glutter/cmd/glutter-gen/internal/reexport"
	"github.com/rm4n0s/glutter/cmd/glutter-gen/internal/shaders"
	"github.com/rm4n0s/glutter/cmd/glutter-gen/internal/wayland"
)

// commands maps each subcommand name to its implementation. It is read-only.
var commands = map[string]func(args []string) error{
	"android-dex": androiddex.Run,
	"apicheck":    apicheck.Run,
	"wayland":     wayland.Run,
	"keys":        keys.Run,
	"reexport":    reexport.Run,
	"copywith":    copywith.Run,
	"errcheck":    errcheck.Run,
	"icons":       icons.Run,
	"shaders":     shaders.Run,
}

// commandNames returns the subcommand names in alphabetical order.
func commandNames() []string {
	names := make([]string, 0, len(commands))
	for name := range commands {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// dispatch runs the subcommand named by args[0] with the remaining arguments.
// Errors from subcommands are returned untouched.
func dispatch(args []string) error {
	if len(args) == 0 {
		return errors.New(tagMissingCommand, "no command given", "commands", commandNames())
	}
	run, ok := commands[args[0]]
	if !ok {
		return errors.New(tagUnknownCommand, "unknown command", "command", args[0], "commands", commandNames())
	}
	return run(args[1:])
}

// formatError renders err for the terminal. It is logged with slog as "err", so an error
// from the errors package becomes JSON with its tag, route, metadata and stack, which a test
// can replay with errors.FromJson. Any other error becomes its message.
func formatError(err error) string {
	if _, ok := errors.FromError(err); !ok {
		return fmt.Sprintf("glutter-gen: %v\n", err)
	}
	var out bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&out, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if len(groups) == 0 && a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))
	logger.Error("glutter-gen failed", "err", err)
	return out.String()
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagMissingCommand = "MissingCommand"
	tagUnknownCommand = "UnknownCommand"
)
