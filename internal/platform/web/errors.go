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

// errClipboardUnavailable is the error of a clipboard call in a page that has no
// navigator.clipboard: an insecure context (not https or localhost) or an old browser.
func errClipboardUnavailable() error {
	return errors.New("WebClipboardUnavailable", "the page has no navigator.clipboard: it needs a secure context (https or localhost)")
}

// errClipboardReadFailed is the error of navigator.clipboard.readText, which the browser refuses when the
// user denies the permission or the page has no focus.
func errClipboardReadFailed(detail string) error {
	return errors.New("WebClipboardReadFailed", "navigator.clipboard.readText failed", "detail", detail)
}

// errClipboardWriteFailed is the error of navigator.clipboard.writeText, which the browser refuses
// outside a user action or when the page has no focus.
func errClipboardWriteFailed(detail string) error {
	return errors.New("WebClipboardWriteFailed", "navigator.clipboard.writeText failed", "detail", detail)
}

// errClipboardFormatUnsupported is the error of writing a content with no text: only text goes to
// the web clipboard.
func errClipboardFormatUnsupported() error {
	return errors.New("WebClipboardFormatUnsupported", "the web clipboard takes plain text only")
}

// errClipboardPrimaryUnavailable is the error of the primary selection, which the web does not have.
func errClipboardPrimaryUnavailable() error {
	return errors.New("ClipboardPrimaryUnavailable", "the web has no primary selection")
}

// errClipboardReadTimeout is the error of a read that took longer than its context allowed: the
// browser may be waiting for the user to answer its permission prompt.
func errClipboardReadTimeout(cause error) error {
	return errors.NewErr("ClipboardReadTimeout", cause)
}

// errAlreadyRunning is the error of a second Run while the platform runs.
func errAlreadyRunning() error {
	return errors.New("PlatformAlreadyRunning", "Run was called while the platform is already running")
}

// errScreenshotPostFailed is the error of a screenshot that did not reach the server.
func errScreenshotPostFailed(detail string) error {
	return errors.New("ScreenshotWriteFailed", "the screenshot could not be posted to the server", "detail", detail)
}

// errOpenURISchemeBlocked is the error of a URI whose scheme runs script in the page that opens it.
func errOpenURISchemeBlocked(uri, scheme string) error {
	return errors.New("OpenURISchemeBlocked", "the web does not open a URI that runs script", "uri", uri, "scheme", scheme)
}

// errOpenURIUnavailable is the error of a window.open that cannot be made: it threw (a sandboxed frame
// that may not open windows), or there is no page. The tag is the one of Linux's OpenURI when nothing can
// open the URI.
func errOpenURIUnavailable(uri, reason string) error {
	return errors.New("OpenURIUnavailable", "the browser cannot open the URI", "uri", uri, "reason", reason)
}

// errNotificationsUnavailable is the error of a page that cannot show notifications: the Notifications API
// is missing (an insecure context, an old browser) or asking for the permission failed.
func errNotificationsUnavailable(reason string) error {
	return errors.New("NotificationsUnavailable", "the page cannot show notifications", "reason", reason)
}

// errNotificationsRefused is the error of a permission that the user did not give: denied, or the prompt
// was dismissed (the permission is still "default").
func errNotificationsRefused(permission string) error {
	return errors.New("NotificationsUnavailable", "the user did not allow the page to show notifications", "permission", permission)
}

// errPermissionAborted is the error of a wait for the permission prompt that the context ended: the user
// has not answered.
func errPermissionAborted(cause error) error {
	return errors.NewErr("NotificationPermissionAborted", cause)
}

// errNotificationFailed is the error of a browser that refuses to construct the Notification (Chrome on
// Android allows notifications from a service worker only).
func errNotificationFailed(detail string) error {
	return errors.New("WebNotificationFailed", "the browser could not make the notification", "detail", detail)
}
