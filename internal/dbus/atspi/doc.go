// Package atspi is the accessibility bridge for Linux desktops (SPEC §8.10): it
// mirrors the framework's semantics tree as AT-SPI2 objects on the accessibility bus,
// so that screen readers such as Orca can read the app and act on it.
//
// The bridge is enabled only while a screen reader is on (the property
// org.a11y.Status.ScreenReaderEnabled of the session bus), so the semantics tree
// costs nothing otherwise. The model of the tree (model.go) and the mapping of roles
// and states (roles.go) do not touch the bus; the bridge (bridge.go) puts them on it.
//
// Flutter: no counterpart (Flutter's accessibility bridge for Linux is the GTK
// embedder's)
package atspi
