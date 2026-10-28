//go:build !amd64 && !arm64

package goid

// Token is a value that tells the calling goroutine from every other goroutine
// that is alive. Here (not amd64 or arm64, so no assembly) it is the goroutine's
// number, [ID], which costs microseconds; the number of a goroutine is never
// reused, so a token of this kind is also never reused.
//
// Flutter: no counterpart
func Token() uint64 { return ID() }
