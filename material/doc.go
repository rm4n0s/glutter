// Package material implements Material 3. It re-exports everything in
// package widgets (and the identifiers of animation, painting, foundation,
// rendering, gestures, services, physics, scheduler and ui that Flutter's
// material.dart exports) through generated wrappers, so an app imports one
// package.
//
// Flutter: package:flutter/material.dart.
package material

//go:generate go run ../cmd/glutter-gen icons -reference ../reference/flutter -out icons_gen.go
//go:generate go run ../cmd/glutter-gen reexport -target . -out widgets_reexport_gen.go
