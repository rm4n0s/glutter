package material

import intopts "github.com/rm4n0s/glutter/internal/opts"

func firstOpts[T any](o []T) T { return intopts.First(o) }
