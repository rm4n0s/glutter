package platform

// ScreenshotPoster is implemented by a Platform that cannot write files, so that the
// self-capture of the engine (GLUTTER_SCREENSHOT) hands the PNG over to it instead: the web
// posts it to the server that serves the page (SPEC §23.8).
//
// Flutter: no counterpart
type ScreenshotPoster interface {
	// PostScreenshot sends the PNG to where the host collects it. It returns at once;
	// done is called later, on the UI thread, with the error if the PNG did not arrive.
	PostScreenshot(png []byte, done func(err error))
}

// ExitController is implemented by a Platform whose app cannot end itself: a web page cannot
// close itself, so the engine does nothing when the app asks to exit (SPEC §23.3).
//
// Flutter: no counterpart
type ExitController interface {
	// CanExit says whether the app may end Run when it asks to exit.
	CanExit() bool
}
