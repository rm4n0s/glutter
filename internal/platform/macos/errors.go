package macos

// The tags of the errors that package macos makes (SPEC §29.14; rm4n0s/errors, R5).
const (
	// TagMacOSNotMainThread is the tag of the error of a platform made on a thread that is not the main
	// thread of the process: AppKit works only there (SPEC §29.3).
	TagMacOSNotMainThread = "MacOSNotMainThread"
	// TagPlatformAlreadyRunning is the tag of the error of a second Run while the first is active.
	TagPlatformAlreadyRunning = "PlatformAlreadyRunning"
	// TagMacOSWindowCreateFailed is the tag of the error of a window that AppKit did not make.
	TagMacOSWindowCreateFailed = "MacOSWindowCreateFailed"
	// TagMacOSDevEventsInvalid is the tag of the error of a GLUTTER_DEV_EVENTS script that cannot be read.
	TagMacOSDevEventsInvalid = "MacOSDevEventsInvalid"
	// TagIdleInhibitUnavailable is the tag of the error of a power assertion that the system refused.
	TagIdleInhibitUnavailable = "IdleInhibitUnavailable"
)
