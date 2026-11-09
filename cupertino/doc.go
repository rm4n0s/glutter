// Package cupertino implements the Cupertino widgets: Flutter's cupertino library,
// the look of iOS and macOS. It re-exports everything in package widgets (and the
// identifiers of animation, painting, foundation, rendering, gestures, services,
// physics, scheduler and ui that Flutter's cupertino.dart exports) through generated
// wrappers, so an app imports one package, as with package material.
//
// Colors that depend on the context are CupertinoDynamicColor; the parameters that
// take "a Color, or a dynamic color" are of the interface type ui.ColorValue, where
// nil is Dart's null (SPEC §12.10).
//
// Flutter: package:flutter/cupertino.dart.
package cupertino

//go:generate go run ../cmd/glutter-gen icons -cupertino -reference ../reference/flutter -out icons_gen.go
//go:generate go run ../cmd/glutter-gen reexport -target . -out widgets_reexport_gen.go
