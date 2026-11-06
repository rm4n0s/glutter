// Package viewport3d is the 3D view of Glutter for CAD and games (SPEC §27.8): a Camera with CAD navigation, and, in
// later steps, a Scene of meshes, materials, lines, points and edges, section planes, picking and overlays.
//
// The camera orbits a target at a distance, as a CAD viewer's does. It works in float64 (D30, package vecmath) and gives
// the GPU float32 matrices relative to the eye, so that a scene 50 km wide with a detail of 0.1 mm draws without jitter:
// Camera.ModelView subtracts the eye from the item's translation in float64 before the matrices go to float32. The
// projections are reversed-Z with an infinite far plane (perspective) and WebGPU's clip space (SPEC §27.1).
//
// Navigation is a widget, NewNavigation, whose gesture recognizers take part in Glutter's gesture arena, so a recognizer
// above the view (a button over it, a drag of a parent) still wins or loses the arena as in Flutter. Its defaults are
// the table of SPEC §27.8, and NavigationOpts changes them.
//
// Everything here runs on the UI thread (SPEC §13).
//
// Flutter: no counterpart
package viewport3d
