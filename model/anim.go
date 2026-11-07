package model

import "github.com/rm4n0s/glutter/vecmath"

// MorphTarget is a shape that a mesh can be moved towards: the change of each vertex at a weight of 1, added to the vertex with the
// weight of the target.
//
// Flutter: no counterpart
type MorphTarget struct {
	// Name is the name in the file (mesh extras "targetNames"), or empty.
	Name string
	// Positions are the changes of x, y and z of each vertex, or nil for none.
	Positions []float32
	// Normals are the changes of the normals, x, y and z of each vertex, or nil for none.
	Normals []float32
	// Tangents are the changes of the x, y and z of the tangents, or nil for none.
	Tangents []float32
}

// TRS is the transform of a node as a translation, a rotation and a scale, which is what an animation changes.
//
// Flutter: no counterpart
type TRS struct {
	Translation vecmath.Vec3
	Rotation    vecmath.Quat
	Scale       vecmath.Vec3
}

// IdentityTRS is the transform that does nothing.
//
// Flutter: no counterpart
func IdentityTRS() TRS {
	return TRS{Rotation: vecmath.QuatIdentity(), Scale: vecmath.Vec3{X: 1, Y: 1, Z: 1}}
}

// Matrix is the matrix of the transform: scale, then rotation, then translation.
//
// Flutter: no counterpart
func (t TRS) Matrix() vecmath.Mat4 {
	return vecmath.NewMat4TRS(t.Translation, t.Rotation, t.Scale)
}

// Skin is the skeleton of the meshes of the nodes that use it.
//
// Flutter: no counterpart
type Skin struct {
	// Name is the name in the file, or empty.
	Name string
	// Joints are the nodes that move the vertices; the JOINTS attribute of a mesh counts in this list.
	Joints []*Node
	// InverseBindMatrices take a vertex from the coordinates of the mesh to those of each joint when the skeleton is at rest; one
	// for each joint (the identity when the file has none).
	InverseBindMatrices []vecmath.Mat4
	// Skeleton is the root of the joints, or nil.
	Skeleton *Node
}

// AnimationPath is what an animation channel changes of its node.
//
// Flutter: no counterpart
type AnimationPath uint8

const (
	// PathTranslation changes the translation: three values at each time (x, y, z).
	PathTranslation AnimationPath = iota
	// PathRotation changes the rotation: four values at each time, a quaternion (x, y, z, w).
	PathRotation
	// PathScale changes the scale: three values at each time.
	PathScale
	// PathWeights changes the weights of the morph targets: as many values at each time as the node's mesh has targets.
	PathWeights
)

// String is the name of the path, as glTF writes it.
//
// Flutter: no counterpart
func (p AnimationPath) String() string {
	switch p {
	case PathTranslation:
		return "translation"
	case PathRotation:
		return "rotation"
	case PathScale:
		return "scale"
	case PathWeights:
		return "weights"
	}
	return "unknown"
}

// Interpolation is how an animation goes from a value to the next.
//
// Flutter: no counterpart
type Interpolation uint8

const (
	// InterpolationLinear blends the two values (a spherical blend for a rotation); the zero value.
	InterpolationLinear Interpolation = iota
	// InterpolationStep keeps a value until the next time.
	InterpolationStep
	// InterpolationCubicSpline is a cubic Hermite spline: each time has three values, an in-tangent, the value and an out-tangent.
	InterpolationCubicSpline
)

// String is the name of the interpolation, as glTF writes it.
//
// Flutter: no counterpart
func (i Interpolation) String() string {
	switch i {
	case InterpolationLinear:
		return "LINEAR"
	case InterpolationStep:
		return "STEP"
	case InterpolationCubicSpline:
		return "CUBICSPLINE"
	}
	return "unknown"
}

// AnimationChannel changes one thing of one node over time.
//
// Flutter: no counterpart
type AnimationChannel struct {
	// Node is the node that is changed.
	Node *Node
	// Path is what is changed.
	Path AnimationPath
	// Interpolation is how the values are blended.
	Interpolation Interpolation
	// Times are the seconds of the keyframes, strictly increasing.
	Times []float32
	// Values are the keyframes: Width values for each time, or three times as many for a cubic spline (in-tangent, value,
	// out-tangent of each time). The channels of one sampler of the file share the same Times.
	Values []float32
}

// Width is how many values a keyframe has: 3 for a translation or a scale, 4 for a rotation, and for the weights the number of
// morph targets, which is told by the length of Values and Times.
//
// Flutter: no counterpart
func (c *AnimationChannel) Width() int {
	switch c.Path {
	case PathTranslation, PathScale:
		return 3
	case PathRotation:
		return 4
	}
	if len(c.Times) == 0 {
		return 0
	}
	n := len(c.Values) / len(c.Times)
	if c.Interpolation == InterpolationCubicSpline {
		n /= 3
	}
	return n
}

// Animation is a set of channels that play together.
//
// Flutter: no counterpart
type Animation struct {
	// Name is the name in the file, or empty.
	Name string
	// Channels are the changes.
	Channels []AnimationChannel
}

// Duration is the time of the last keyframe of the channels, in seconds; 0 for an animation with no keyframes.
//
// Flutter: no counterpart
func (a *Animation) Duration() float32 {
	var d float32
	for i := range a.Channels {
		if t := a.Channels[i].Times; len(t) > 0 {
			d = max(d, t[len(t)-1])
		}
	}
	return d
}

// Scene is a set of root nodes.
//
// Flutter: no counterpart
type Scene struct {
	// Name is the name in the file, or empty.
	Name string
	// Nodes are the roots of the scene.
	Nodes []*Node
}
