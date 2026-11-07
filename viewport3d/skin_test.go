package viewport3d_test

import (
	"testing"

	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
)

func v3(x, y, z float64) vecmath.Vec3 { return vecmath.Vec3{X: x, Y: y, Z: z} }

// TestJointMatricesOfATwoBoneChain checks the matrices that the shader blends, against a vertex worked out by hand: a chain of two
// bones, the second a child of the first and bent a quarter turn about z at its base.
func TestJointMatricesOfATwoBoneChain(t *testing.T) {
	// The bind pose: joint 0 at the origin, joint 1 at x = 1. The inverse bind matrices take the world of the bind pose into the
	// joints' own.
	skin := viewport3d.NewSkin([]vecmath.Mat4{vecmath.Mat4Identity(), vecmath.NewMat4Translation(v3(-1, 0, 0))})
	if skin.JointCount() != 2 {
		t.Fatalf("%d joints", skin.JointCount())
	}
	// Joint 0 is lifted by 2 along y; joint 1 is its child at x = 1 (in the parent's frame) turned a quarter turn about z.
	root := vecmath.NewMat4Translation(v3(0, 2, 0))
	child := root.Mul(vecmath.NewMat4Translation(v3(1, 0, 0))).Mul(vecmath.NewMat4RotationZ(3.141592653589793 / 2))
	skin.SetJoints([]vecmath.Mat4{root, child})
	dst := make([]vecmath.Mat4, 2)
	skin.JointMatrices(vecmath.Mat4Identity(), vecmath.Vec3{}, dst)
	tests := []struct {
		name  string
		joint int
		from  vecmath.Vec3
		want  vecmath.Vec3
	}{
		// A vertex of the first bone is lifted by 2.
		{"joint 0 lifts a vertex", 0, v3(0.5, 0, 0), v3(0.5, 2, 0)},
		// A vertex 0.5 beyond the second joint is, in that joint's frame, 0.5 along x; after the quarter turn it points along y.
		{"joint 1 bends a vertex", 1, v3(1.5, 0, 0), v3(1, 2.5, 0)},
		{"joint 1 keeps its own position", 1, v3(1, 0, 0), v3(1, 2, 0)},
	}
	for _, tt := range tests {
		if got := dst[tt.joint].TransformPoint(tt.from); got.Sub(tt.want).Length() > 1e-9 {
			t.Errorf("%s: %v goes to %v, want %v", tt.name, tt.from, got, tt.want)
		}
	}
}

// TestJointMatricesAtRestAreTheIdentity checks that a skin that has not been posed leaves the mesh as it is, whatever the transform of
// the item and the origin of the mesh.
func TestJointMatricesAtRestAreTheIdentity(t *testing.T) {
	skin := viewport3d.NewSkin([]vecmath.Mat4{vecmath.NewMat4Translation(v3(-2, 0, 0)), vecmath.NewMat4Translation(v3(-5, 0, 0)).Mul(vecmath.NewMat4Scale(v3(2, 2, 2)))})
	dst := make([]vecmath.Mat4, 2)
	skin.JointMatrices(vecmath.Mat4Identity(), vecmath.Vec3{}, dst)
	for i, m := range dst {
		if !m.NearlyEqual(vecmath.Mat4Identity(), 1e-12) {
			t.Errorf("joint %d at rest has the matrix %v", i, m)
		}
	}
}

// TestJointMatricesIgnoreTheItemsTransformAndKeepFloat64 checks the rule of glTF 2.0 (3.7.3.3) that a skinned mesh follows its joints, not its node, with
// the matrices made in float64 relative to the mesh's origin: the position that the model matrix and a joint matrix give a vertex is
// the one that the joint world and the inverse bind matrix give it, however far away the origin of the mesh is.
func TestJointMatricesIgnoreTheItemsTransformAndKeepFloat64(t *testing.T) {
	origin := v3(5e7, -3e7, 2e7)
	ibm := vecmath.NewMat4Translation(v3(-1, 0, 0))
	skin := viewport3d.NewSkin([]vecmath.Mat4{ibm, ibm})
	world := vecmath.NewMat4Translation(origin).Mul(vecmath.NewMat4RotationY(0.7)).Mul(vecmath.NewMat4Translation(v3(1, 0, 0)))
	skin.SetJointWorld(1, world)
	item := vecmath.NewMat4Translation(v3(100, 200, 300)).Mul(vecmath.NewMat4RotationX(0.3))
	dst := make([]vecmath.Mat4, 2)
	skin.JointMatrices(item, origin, dst)
	// model is what the instance carries: the item's transform after the translation to the mesh's origin.
	model := item.Mul(vecmath.NewMat4Translation(origin))
	rel := v3(0.25, -0.5, 0.75) // a vertex relative to the origin of the mesh, in the bind pose
	got := model.TransformPoint(dst[1].TransformPoint(rel))
	want := world.Mul(ibm).TransformPoint(origin.Add(rel))
	if got.Sub(want).Length() > 1e-6 {
		t.Errorf("the vertex goes to %v, want %v", got, want)
	}
}

// TestJointMatricesOfASingularItemAreTheIdentity checks the matrices of an item whose transform cannot be inverted.
func TestJointMatricesOfASingularItemAreTheIdentity(t *testing.T) {
	skin := viewport3d.NewSkin([]vecmath.Mat4{vecmath.Mat4Identity()})
	dst := []vecmath.Mat4{{}}
	skin.JointMatrices(vecmath.NewMat4Scale(v3(0, 1, 1)), vecmath.Vec3{}, dst)
	if dst[0] != vecmath.Mat4Identity() {
		t.Errorf("got %v", dst[0])
	}
}

// TestSkinNotifiesItsListenersOnChange checks SetJointWorld and SetJoints.
func TestSkinNotifiesItsListenersOnChange(t *testing.T) {
	skin := viewport3d.NewSkin([]vecmath.Mat4{vecmath.Mat4Identity(), vecmath.Mat4Identity()})
	calls := 0
	skin.AddListener(func() { calls++ })
	moved := vecmath.NewMat4Translation(v3(1, 0, 0))
	tests := []struct {
		name string
		do   func()
		want int
	}{
		{"a joint moves", func() { skin.SetJointWorld(0, moved) }, 1},
		{"it stays", func() { skin.SetJointWorld(0, moved) }, 0},
		{"two joints move: one notice", func() { skin.SetJoints([]vecmath.Mat4{vecmath.Mat4Identity(), moved}) }, 1},
		{"the same joints", func() { skin.SetJoints([]vecmath.Mat4{vecmath.Mat4Identity(), moved}) }, 0},
		{"more matrices than joints: the extra are left out", func() { skin.SetJoints([]vecmath.Mat4{vecmath.Mat4Identity(), moved, moved}) }, 0},
	}
	for _, tt := range tests {
		calls = 0
		tt.do()
		if calls != tt.want {
			t.Errorf("%s: %d notices, want %d", tt.name, calls, tt.want)
		}
	}
	if got := skin.JointWorld(1); got != moved {
		t.Errorf("JointWorld = %v", got)
	}
	if got := skin.InverseBindMatrix(1); got != vecmath.Mat4Identity() {
		t.Errorf("InverseBindMatrix = %v", got)
	}
}
