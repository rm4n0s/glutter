// Package uiatest is the UI Automation client that tests use to look at the providers of package uia the
// way Narrator, NVDA and Accessibility Insights do (SPEC §25.11, §25.13): it creates CUIAutomation, finds
// the element of a window, walks the tree, reads properties, calls the patterns and listens to events. The
// interface tables (interfaces.go) have no OS tag, so the layout test checks them against the mingw-w64
// headers on the Linux host; the client itself is *_windows.go.
//
// Flutter: no counterpart
package uiatest
