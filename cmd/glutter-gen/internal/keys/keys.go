// Package keys implements the `glutter-gen keys` subcommand, which generates the key tables (XKB keycode to Flutter key ids).
//
// Flutter: no counterpart.
package keys

import "github.com/rm4n0s/errors"

// Run executes the `glutter-gen keys` subcommand with the arguments that follow
// its name.
//
// Flutter: no counterpart.
func Run(args []string) error {
	// TODO(step 1.7): generate the key tables.
	return errors.New("NotImplemented", "glutter-gen keys is not implemented yet", "command", "keys", "args", args)
}
