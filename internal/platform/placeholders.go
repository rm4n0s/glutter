package platform

// Services groups the optional capabilities of a platform that the engine uses. Each
// accessor returns nil when the capability is absent. (The desktop services of
// package desktop, such as the tray and the notifications, are made by the app.)
//
// Flutter: no counterpart
type Services interface {
	// Accessibility returns the bridge to the assistive technologies, or nil.
	Accessibility() Accessibility
}
