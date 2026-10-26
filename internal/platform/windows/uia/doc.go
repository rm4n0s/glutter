// Package uia is the accessibility bridge of Windows (SPEC §25.11): it mirrors the framework's semantics
// tree as UI Automation providers, so that Narrator, NVDA, JAWS and Accessibility Insights can read the
// app and act on it. The bridge is off until the first WM_GETOBJECT that asks a window for its UI
// Automation provider.
//
// The model of the trees, the mapping of roles, flags and actions to control types, properties and
// patterns, the Text pattern's ranges, the on/off state and the translation of changes into events and of
// requests into actions (Core) are plain Go, tested on any host. The COM objects that Go makes for the
// providers, the thunks for the methods that take a double, and the Bridge that is a
// platform.Accessibility are *_windows.go. Sub-packages: callshim (calls a method with doubles, for
// tests), semtree (builds semantics updates for tests) and uiatest (the UI Automation client of the
// tests).
//
// Flutter: shell/platform/windows/accessibility_bridge_windows.cc AccessibilityBridgeWindows
package uia
