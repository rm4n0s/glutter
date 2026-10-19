package gestures

import "github.com/rm4n0s/glutter/internal/opts"

// oneOpts is opts.First; the wrapper lets constructors name a parameter opts.
func oneOpts[T any](o []T) T { return opts.First(o) }
