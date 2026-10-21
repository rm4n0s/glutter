package web

import "github.com/rm4n0s/errors"

// errHostElementNotFound is the error for a selector of glutterConfig.hostElement that
// matches no element (SPEC §23.2).
func errHostElementNotFound(selector string) error {
	return errors.New("HostElementNotFound", "glutterConfig.hostElement matches no element of the page", "selector", selector)
}

// errWindowUnsupported is the error for a window that the web cannot have: one view
// per page (SPEC §23.2).
func errWindowUnsupported(reason string) error {
	return errors.New("WindowUnsupported", "the web has one window per page: "+reason)
}

// errNoPage is the error for a program that is not running in a web page, such as one
// that runs under Node.
func errNoPage() error {
	return errors.New("BackendUnavailable", "there is no web page to draw in: the program is not running in a browser",
		"goos", "js")
}

// errDragUnsupported is the error of Window.StartDrag: drag and drop is not in 1.1
// (SPEC §23.4).
func errDragUnsupported() error {
	return errors.New("DragUnsupported", "dragging from a web page is not supported")
}

// errScreenOnUnsupported is the error of Window.KeepScreenOn on the web.
func errScreenOnUnsupported() error {
	return errors.New("IdleInhibitUnavailable", "the web platform cannot keep the screen on yet")
}

// errClipboardUnavailable is the error of the clipboard until step 21.3 gives the web
// one.
func errClipboardUnavailable() error {
	return errors.New("WebClipboardUnavailable", "the web clipboard is not implemented yet")
}

// errAlreadyRunning is the error of a second Run while the platform runs.
func errAlreadyRunning() error {
	return errors.New("PlatformAlreadyRunning", "Run was called while the platform is already running")
}

// errScreenshotPostFailed is the error of a screenshot that did not reach the server.
func errScreenshotPostFailed(detail string) error {
	return errors.New("ScreenshotWriteFailed", "the screenshot could not be posted to the server", "detail", detail)
}
