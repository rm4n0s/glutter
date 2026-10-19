package animation

import (
	intopts "github.com/rm4n0s/glutter/internal/opts"
)

// firstOpts returns the one options value of a trailing `opts ...XOpts`
// parameter (SPEC §12.1).
func firstOpts[T any](o []T) T { return intopts.First(o) }
