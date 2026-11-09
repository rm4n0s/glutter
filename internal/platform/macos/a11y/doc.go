// Package a11y is the accessibility bridge of macOS (SPEC §29.10): a plain-Go model of the semantics trees
// of the views of the app and the mapping of a node to what NSAccessibility says about an element (its role, label,
// value, help, state, actions and text range), and, in the files for macOS, the elements that VoiceOver and the
// Accessibility Inspector read and the notifications that tell them of a change.
//
// The model and the tables carry no OS tag, so that their rules are tested on every host; the elements are
// NSAccessibilityElement instances that a Go-made subclass gives the actions to (the properties are set on them,
// since a Go callback cannot return an NSRect). The tables follow Chromium's ax_platform_node_mac.mm, through
// Flutter's copy of it; the model follows Flutter's AccessibilityBridge.
package a11y
