package android

import "log/slog"

// process is the bridge of this process: JNI has one set of entry points, so the calls of the
// Java side reach one queue, which the platform of the process drains (SPEC §24.2). It is the one
// piece of global state of the platform, and the JNI functions are why.
var process = newBridge()

// Options are the optional parameters of New.
//
// Flutter: no counterpart
type Options struct {
	// Logger receives the platform's log; slog.Default() when nil.
	Logger *slog.Logger
}

// New makes the platform of the process. It must be called on the UI thread, the one that runs
// main: that is where the looper is prepared. It fails with LooperUnavailable.
//
// Flutter: shell/platform/android/platform_view_android.cc PlatformViewAndroid.PlatformViewAndroid
func New(opts ...Options) (*Platform, error) {
	var o Options
	if len(opts) > 0 {
		o = opts[0]
	}
	h, err := makeHost(initLooper, nativeHost{})
	if err != nil {
		return nil, err
	}
	if o.Logger != nil {
		process.log = o.Logger
	}
	return newPlatform(process, h, o.Logger), nil
}
