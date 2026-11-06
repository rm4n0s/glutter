// Package vecmath is the 3D math of Glutter (SPEC §27.7): vectors, quaternions, matrices, rays, planes, boxes, spheres
// and frustums in float64, as values that never allocate.
//
// The CPU works in float64 and the GPU in float32 (D30): a CAD scene 50 km wide with a detail of 0.1 mm keeps its
// precision in the matrices that are multiplied here, and only the result, taken relative to the camera, goes to the
// GPU through Mat4.Float32.
//
// Matrices are column-major, as WGSL reads them: the entry in row r and column c is m[c*4+r]. A matrix applies to a
// column vector, so a.Mul(b) applies b first. The clip space is WebGPU's: x and y from -1 to 1, z from 0 to 1, and
// the view looks down -z. The projections are reversed-Z: the near plane maps to 1 and the far one (or infinity) to 0,
// which makes the depth of a float32 buffer even over the distance. The rotations are right-handed: a positive angle
// turns counterclockwise when seen from the positive end of the axis.
//
// Flutter: no counterpart
package vecmath
