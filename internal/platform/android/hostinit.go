package android

import "github.com/rm4n0s/errors"

// makeHost prepares the looper of the calling thread with initLooper (0 is success) and returns
// the host. It fails with LooperUnavailable when the system gives no looper, eventfd or
// choreographer.
//
// Flutter: no counterpart
func makeHost(initLooper func() int, h host) (host, error) {
	if status := initLooper(); status != 0 {
		return nil, errors.New(tagLooperUnavailable, "the UI thread's ALooper, eventfd or AChoreographer could not be made", "status", status)
	}
	return h, nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagLooperUnavailable = "LooperUnavailable"
)
