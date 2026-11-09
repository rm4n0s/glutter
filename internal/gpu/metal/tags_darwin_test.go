//go:build darwin && !ios

package metal

import "github.com/rm4n0s/errors"

// asTagged returns the tag of an error of this module.
func asTagged(err error) (string, bool) {
	appErr, ok := errors.FromError(err)
	if !ok {
		return "", false
	}
	return appErr.Tag, true
}
