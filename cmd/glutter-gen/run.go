package main

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/cmd/glutter-gen/internal/apicheck"
	"github.com/rm4n0s/glutter/cmd/glutter-gen/internal/copywith"
	"github.com/rm4n0s/glutter/cmd/glutter-gen/internal/icons"
	"github.com/rm4n0s/glutter/cmd/glutter-gen/internal/keys"
	"github.com/rm4n0s/glutter/cmd/glutter-gen/internal/reexport"
	"github.com/rm4n0s/glutter/cmd/glutter-gen/internal/wayland"
)

// commands maps each subcommand name to its implementation. It is read-only.
var commands = map[string]func(args []string) error{
	"apicheck": apicheck.Run,
	"wayland":  wayland.Run,
	"keys":     keys.Run,
	"reexport": reexport.Run,
	"copywith": copywith.Run,
	"icons":    icons.Run,
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
		return errors.New("MissingCommand", "no command given", "commands", commandNames())
	}
	run, ok := commands[args[0]]
	if !ok {
		return errors.New("UnknownCommand", "unknown command", "command", args[0], "commands", commandNames())
	}
	return run(args[1:])
}

// formatError renders err for the terminal. An error from the errors package
// becomes indented JSON (ToJson), so it can be replayed in a test with
// errors.FromJson. Any other error becomes its message.
func formatError(err error) string {
	appErr, ok := errors.FromError(err)
	if !ok {
		return fmt.Sprintf("glutter-gen: %v\n", err)
	}
	errJSON, jsonErr := appErr.ToJson()
	if jsonErr != nil {
		return fmt.Sprintf("glutter-gen: %v\n", appErr)
	}
	out, marshalErr := json.MarshalIndent(errJSON, "", "  ")
	if marshalErr != nil {
		return fmt.Sprintf("glutter-gen: %v\n", appErr)
	}
	return string(out) + "\n"
}
