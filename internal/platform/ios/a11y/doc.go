// Package a11y is the plain-Go part of the accessibility bridge of iOS (SPEC §30.7): what a node of the semantics tree is to
// UIAccessibility: whether it is an element, its label, value and hint, its traits, the actions that VoiceOver can do on it and the
// scroll directions that VoiceOver sends. The model of the trees is internal/platform/darwin/a11ymodel (shared with macOS); the elements,
// the container (the view) and the notifications are in package ios (a11y_ios.go), which UIKit needs.
//
// The rules are ported from Flutter's SemanticsObject.mm. Nothing here has an OS tag, so the tables are tested on every host.
package a11y
