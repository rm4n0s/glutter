// Package mobile is the logic of the desktop services of package desktop on a mobile platform
// (Android in 1.6, SPEC §24.11): opening a URI with an intent and showing notifications through the
// system's notification manager. It is plain Go over two small interfaces, IntentOpener and
// NotificationAPI, that the platform implements over its Java side; it has no operating-system code,
// so every build and the normal test run can use and test it, as with the web's internal/platform/web
// Desktop (SPEC §23.7). It is a package of its own, not part of internal/platform/android, so that
// package desktop can use its types on every target without importing an Android backend (SPEC §6.3).
package mobile
