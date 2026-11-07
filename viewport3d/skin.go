package viewport3d

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/vecmath"
)

// Skin is the skeleton of the items that use it (SPEC §28.5): for each joint, the inverse bind matrix (the joint's world
// transform when the mesh was bound to it, inverted) and the joint's current world transform, which the app sets each frame
// (Uploaded.SetPose does it for a glTF model). The mesh's joints (MeshData.Joints) number the joints of the skin. A Skin is a
// Listenable: setting a joint draws the views of the scenes whose items use it again.
//
// Flutter: no counterpart
type Skin struct {
	foundation.ChangeNotifier
	inverseBind []vecmath.Mat4
	world       []vecmath.Mat4
}

// NewSkin creates a skin with one joint for each inverse bind matrix. The joints are at rest: their world transforms are the
// inverses of the inverse bind matrices, so the skinned mesh is in its bind pose.
//
// Flutter: no counterpart
func NewSkin(inverseBind []vecmath.Mat4) *Skin {
	s := &Skin{inverseBind: append([]vecmath.Mat4(nil), inverseBind...), world: make([]vecmath.Mat4, len(inverseBind))}
	for i, ibm := range inverseBind {
		if inv, ok := ibm.Inverse(); ok {
			s.world[i] = inv
		} else {
			s.world[i] = vecmath.Mat4Identity()
		}
	}
	return s
}

// JointCount is the number of joints.
//
// Flutter: no counterpart
func (s *Skin) JointCount() int { return len(s.inverseBind) }

// InverseBindMatrix is the inverse bind matrix of a joint.
//
// Flutter: no counterpart
func (s *Skin) InverseBindMatrix(joint int) vecmath.Mat4 { return s.inverseBind[joint] }

// JointWorld is the current world transform of a joint.
//
// Flutter: no counterpart
func (s *Skin) JointWorld(joint int) vecmath.Mat4 { return s.world[joint] }

// SetJointWorld changes the world transform of a joint, in float64.
//
// Flutter: no counterpart
func (s *Skin) SetJointWorld(joint int, m vecmath.Mat4) {
	if s.world[joint] != m {
		s.world[joint] = m
		s.NotifyListeners()
	}
}

// SetJoints changes the world transforms of the first len(world) joints at once, and tells the listeners once.
//
// Flutter: no counterpart
func (s *Skin) SetJoints(world []vecmath.Mat4) {
	changed := false
	for i := 0; i < len(world) && i < len(s.world); i++ {
		if s.world[i] != world[i] {
			s.world[i] = world[i]
			changed = true
		}
	}
	if changed {
		s.NotifyListeners()
	}
}

// JointMatrices writes the matrices that the vertex shader blends, one for each joint, for an item with the transform
// itemTransform (the item's, before the translation to its mesh's origin) and a mesh whose origin is origin: the matrix of a joint
// takes a position relative to the origin, in the bind pose, to the position relative to the origin that the joint puts it at:
//
//	translation(-origin) · inverse(itemTransform) · jointWorld · inverseBind · translation(origin)
//
// (glTF 2.0, 3.7.3.3: the item's own transform is left out of the skin, so a skinned mesh follows its joints, not its node). It is
// made in float64, so the matrices are small however far the item is from the origin of the world (D30). dst needs a place for
// each joint; a transform that cannot be inverted gives the identity matrices.
//
// Flutter: no counterpart
func (s *Skin) JointMatrices(itemTransform vecmath.Mat4, origin vecmath.Vec3, dst []vecmath.Mat4) {
	inv, ok := itemTransform.Inverse()
	if !ok {
		for i := range s.world {
			dst[i] = vecmath.Mat4Identity()
		}
		return
	}
	shiftIn, shiftOut := vecmath.NewMat4Translation(origin), vecmath.NewMat4Translation(origin.Neg())
	for i := range s.world {
		dst[i] = shiftOut.Mul(inv).Mul(s.world[i]).Mul(s.inverseBind[i]).Mul(shiftIn)
	}
}
