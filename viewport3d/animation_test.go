package viewport3d_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/model"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-5 }

// TestAnimationSample checks the three interpolations, and the times before the first and after the last keyframe, against values
// worked out by hand.
func TestAnimationSample(t *testing.T) {
	node := model.NewNode("n")
	quarter := float32(math.Sqrt2 / 2)
	tests := []struct {
		name    string
		channel model.AnimationChannel
		t       float32
		want    []float64
	}{
		{"linear translation, a quarter of the way", model.AnimationChannel{Path: model.PathTranslation, Times: []float32{1, 3}, Values: []float32{0, 0, 0, 4, 8, 12}}, 1.5, []float64{1, 2, 3}},
		{"linear translation, the middle", model.AnimationChannel{Path: model.PathTranslation, Times: []float32{1, 3}, Values: []float32{0, 0, 0, 4, 8, 12}}, 2, []float64{2, 4, 6}},
		{"before the first keyframe", model.AnimationChannel{Path: model.PathTranslation, Times: []float32{1, 3}, Values: []float32{5, 6, 7, 4, 8, 12}}, 0, []float64{5, 6, 7}},
		{"after the last keyframe", model.AnimationChannel{Path: model.PathTranslation, Times: []float32{1, 3}, Values: []float32{5, 6, 7, 4, 8, 12}}, 9, []float64{4, 8, 12}},
		{"at the last keyframe", model.AnimationChannel{Path: model.PathTranslation, Times: []float32{1, 3}, Values: []float32{5, 6, 7, 4, 8, 12}}, 3, []float64{4, 8, 12}},
		{"at a middle keyframe", model.AnimationChannel{Path: model.PathTranslation, Times: []float32{0, 1, 2}, Values: []float32{0, 0, 0, 1, 1, 1, 5, 5, 5}}, 1, []float64{1, 1, 1}},
		{"in the second interval", model.AnimationChannel{Path: model.PathTranslation, Times: []float32{0, 1, 2}, Values: []float32{0, 0, 0, 1, 1, 1, 5, 5, 5}}, 1.5, []float64{3, 3, 3}},
		{"step keeps the value", model.AnimationChannel{Path: model.PathScale, Interpolation: model.InterpolationStep, Times: []float32{0, 1, 2}, Values: []float32{1, 1, 1, 2, 2, 2, 3, 3, 3}}, 1.99, []float64{2, 2, 2}},
		{"step changes at the time", model.AnimationChannel{Path: model.PathScale, Interpolation: model.InterpolationStep, Times: []float32{0, 1, 2}, Values: []float32{1, 1, 1, 2, 2, 2, 3, 3, 3}}, 2, []float64{3, 3, 3}},
		{"slerp of a quarter turn about z, halfway", model.AnimationChannel{Path: model.PathRotation, Times: []float32{0, 1}, Values: []float32{0, 0, 0, 1, 0, 0, quarter, quarter}}, 0.5,
			[]float64{0, 0, math.Sin(math.Pi / 8), math.Cos(math.Pi / 8)}},
		{"slerp takes the short way", model.AnimationChannel{Path: model.PathRotation, Times: []float32{0, 1}, Values: []float32{0, 0, 0, 1, 0, 0, -quarter, -quarter}}, 0.5,
			[]float64{0, 0, math.Sin(math.Pi / 8), math.Cos(math.Pi / 8)}},
		// Hermite: v0 = 0, out-tangent 2, v1 = 1, in-tangent -1, times 0 and 2, s = 1/2: 0.125*2*2 + 0.5 + (-0.125)*2*(-1) = 1.25.
		{"cubic spline translation", model.AnimationChannel{Path: model.PathTranslation, Interpolation: model.InterpolationCubicSpline, Times: []float32{0, 2},
			Values: []float32{0, 0, 0, 0, 0, 0, 2, 2, 2, -1, -1, -1, 1, 1, 1, 0, 0, 0}}, 1, []float64{1.25, 1.25, 1.25}},
		{"cubic spline at the first keyframe", model.AnimationChannel{Path: model.PathTranslation, Interpolation: model.InterpolationCubicSpline, Times: []float32{0, 2},
			Values: []float32{0, 0, 0, 7, 8, 9, 2, 2, 2, -1, -1, -1, 1, 1, 1, 0, 0, 0}}, 0, []float64{7, 8, 9}},
		{"cubic spline after the last keyframe", model.AnimationChannel{Path: model.PathTranslation, Interpolation: model.InterpolationCubicSpline, Times: []float32{0, 2},
			Values: []float32{0, 0, 0, 7, 8, 9, 2, 2, 2, -1, -1, -1, 1, 1, 1, 0, 0, 0}}, 5, []float64{1, 1, 1}},
		{"cubic spline of rotations is normalized", model.AnimationChannel{Path: model.PathRotation, Interpolation: model.InterpolationCubicSpline, Times: []float32{0, 1},
			Values: []float32{0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0}}, 0.5, nil},
		{"weights, two targets", model.AnimationChannel{Path: model.PathWeights, Times: []float32{0, 2}, Values: []float32{0, 1, 1, 0}}, 1, []float64{0.5, 0.5}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.channel
			c.Node = node
			pose := viewport3d.NewPose()
			viewport3d.NewAnimation(&model.Animation{Channels: []model.AnimationChannel{c}}).Sample(tt.t, pose)
			var got []float64
			local := pose.Local(node)
			switch c.Path {
			case model.PathTranslation:
				got = []float64{local.Translation.X, local.Translation.Y, local.Translation.Z}
			case model.PathScale:
				got = []float64{local.Scale.X, local.Scale.Y, local.Scale.Z}
			case model.PathRotation:
				q := local.Rotation
				if l := q.Length(); math.Abs(l-1) > 1e-6 {
					t.Errorf("the rotation has length %v", l)
				}
				// q and -q are the same rotation: compare with the sign that has w >= 0.
				if q.W < 0 {
					q = vecmath.Quat{X: -q.X, Y: -q.Y, Z: -q.Z, W: -q.W}
				}
				got = []float64{q.X, q.Y, q.Z, q.W}
			case model.PathWeights:
				for _, w := range pose.Weights(node) {
					got = append(got, float64(w))
				}
			}
			if tt.want == nil {
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if !near(got[i], tt.want[i]) {
					t.Errorf("got %v, want %v", got, tt.want)
					break
				}
			}
		})
	}
}

// TestAnimationDuration is the time of the last keyframe of the channels.
func TestAnimationDuration(t *testing.T) {
	n := model.NewNode("n")
	a := viewport3d.NewAnimation(&model.Animation{Name: "walk", Channels: []model.AnimationChannel{
		{Node: n, Path: model.PathTranslation, Times: []float32{0, 1.5}, Values: []float32{0, 0, 0, 1, 1, 1}},
		{Node: n, Path: model.PathScale, Times: []float32{0, 2.5}, Values: []float32{1, 1, 1, 2, 2, 2}},
	}})
	if a.Duration() != 2.5 || a.Name() != "walk" {
		t.Errorf("Duration %v, Name %q", a.Duration(), a.Name())
	}
}

// TestPoseKeepsWhatTheNodeHas checks that a pose changes the parts that an animation sets and leaves the others as the node has them,
// that Clear forgets them, and that a node with no TRS (a matrix) keeps its matrix until a part is set.
func TestPoseKeepsWhatTheNodeHas(t *testing.T) {
	n := model.NewNode("n")
	n.TRS = &model.TRS{Translation: vecmath.Vec3{X: 1, Y: 2, Z: 3}, Rotation: vecmath.QuatIdentity(), Scale: vecmath.Vec3{X: 2, Y: 2, Z: 2}}
	n.Transform = n.TRS.Matrix()
	rot := model.AnimationChannel{Node: n, Path: model.PathRotation, Times: []float32{0}, Values: []float32{0, 0, 1, 0}}
	pose := viewport3d.NewPose()
	if m := pose.Matrix(n); m != n.Transform {
		t.Errorf("an empty pose changes the node's matrix: %v", m)
	}
	viewport3d.NewAnimation(&model.Animation{Channels: []model.AnimationChannel{rot}}).Sample(0, pose)
	l := pose.Local(n)
	if l.Translation != n.TRS.Translation || l.Scale != n.TRS.Scale {
		t.Errorf("the translation and the scale of the node were lost: %+v", l)
	}
	if !near(l.Rotation.Z, 1) {
		t.Errorf("the rotation is %+v, want a half turn about z", l.Rotation)
	}
	pose.Clear()
	if m := pose.Matrix(n); m != n.Transform {
		t.Errorf("a cleared pose changes the node's matrix: %v", m)
	}
	var nilPose *viewport3d.Pose
	if m := nilPose.Matrix(n); m != n.Transform || nilPose.Weights(n) != nil {
		t.Errorf("a nil pose is not the model at rest")
	}
	// A node the file gave as a matrix has the identity for the parts that no animation set.
	matrix := model.NewNode("matrix")
	if l := pose.Local(matrix); l.Scale != (vecmath.Vec3{X: 1, Y: 1, Z: 1}) || l.Rotation != vecmath.QuatIdentity() {
		t.Errorf("the local transform of a node with a matrix: %+v", l)
	}
}
