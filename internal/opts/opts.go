// Package opts holds the helper behind the trailing options parameter of
// every constructor (SPEC §12.1).
package opts

import "github.com/rm4n0s/errors"

// First returns the one options value of a constructor's trailing
// `opts ...XOpts` parameter, or the zero value when there is none. More than
// one panics with the tag MultipleOptions.
//
// Flutter: no counterpart (Dart's named parameters)
func First[T any](opts []T) T {
	var zero T
	switch len(opts) {
	case 0:
		return zero
	case 1:
		return opts[0]
	}
	panic(errors.New(tagMultipleOptions, "more than one options value was passed", "count", len(opts)))
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagMultipleOptions = "MultipleOptions"
)
