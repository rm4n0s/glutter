// Package testhost holds the platform that gluttertest makes the host's for a test that chose none, so that a
// test on a Mac or an iOS device runs as Linux in every build mode (SPEC §12.10, the row of
// defaultTargetPlatform in a test). In a debug build gluttertest uses foundation's override, as Flutter's
// tests do; a release build has no override (foundation.SetDebugDefaultTargetPlatformOverride panics there),
// so gluttertest sets the platform here and foundation.DefaultTargetPlatform reads it in every mode.
//
// Only gluttertest sets it; a program that does not link gluttertest never does, and then
// foundation.DefaultTargetPlatform is the host's. Like foundation's override, it is process-wide state: the
// framework reads the platform from a function with no binding to ask.
package testhost

import "sync/atomic"

// platform holds the platform as a foundation.TargetPlatform; 0 is none.
var platform atomic.Int32

// Platform is the platform that a test made the host's, as a foundation.TargetPlatform; 0 is none.
//
// Flutter: no counterpart
func Platform() int32 { return platform.Load() }

// SetPlatform makes p, a foundation.TargetPlatform, the host's platform; 0 gives the host its own back.
//
// Flutter: no counterpart
func SetPlatform(p int32) { platform.Store(p) }
