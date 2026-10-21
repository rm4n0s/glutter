//go:build !(js && wasm)

package web

// NewDesktop makes the web's desktop services. Off the web there is no page: OpenURI is
// OpenURIUnavailable and the notifications are NotificationsUnavailable. Package desktop does not use
// it off the web; it exists so that the package builds everywhere.
//
// Flutter: no counterpart
func NewDesktop() *Desktop { return newDesktop(nil, nil, nil) }
