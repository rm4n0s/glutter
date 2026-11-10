//go:build !ios

package ios

import "github.com/rm4n0s/errors"

// Main fails with IOSUnavailable: this build is not for iOS.
//
// Flutter: no counterpart
func Main(MainOptions) error {
	return errors.New(tagIOSUnavailable, "this build is not for iOS: build with GOOS=ios (glutter-ios)")
}

const tagIOSUnavailable = TagIOSUnavailable
