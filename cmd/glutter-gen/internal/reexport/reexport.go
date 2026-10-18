// Package reexport implements the `glutter-gen reexport` subcommand, which generates the wrappers that re-export package widgets from package material.
//
// Flutter: no counterpart.
package reexport

import "github.com/rm4n0s/errors"

// Run executes the `glutter-gen reexport` subcommand with the arguments that follow
// its name.
//
// Flutter: no counterpart.
func Run(args []string) error {
	// TODO(step 12.4): generate the re-exports.
	return errors.New("NotImplemented", "glutter-gen reexport is not implemented yet", "command", "reexport", "args", args)
}
