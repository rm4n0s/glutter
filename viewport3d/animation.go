package viewport3d

import (
	"math"
	"sort"

	"github.com/rm4n0s/glutter/model"
	"github.com/rm4n0s/glutter/vecmath"
)

// poseNode is what a pose says of a node: the parts of its transform that an animation changed.
type poseNode struct {
	trs model.TRS
	// has says which parts are set: 1 translation, 2 rotation, 4 scale.
	has uint8
}

// Pose is the local transforms and the morph weights that animations gave to nodes of a model at one time (SPEC §28.5). A part that
// no animation changed is not in it: the node keeps its own. Sample writes into a pose, and Uploaded.SetPose shows it. A pose is made
// once and reused: sampling again allocates nothing.
//
// Flutter: no counterpart
type Pose struct {
	nodes   map[*model.Node]*poseNode
	weights map[*model.Node][]float32
}

// NewPose makes an empty pose: every node as the model has it.
//
// Flutter: no counterpart
func NewPose() *Pose {
	return &Pose{nodes: map[*model.Node]*poseNode{}, weights: map[*model.Node][]float32{}}
}

// Clear forgets what the pose says, keeping its memory.
//
// Flutter: no counterpart
func (p *Pose) Clear() {
	for _, n := range p.nodes {
		n.has = 0
	}
	for k, w := range p.weights {
		p.weights[k] = w[:0]
	}
}

// Local is the local transform of a node in the pose: the translation, the rotation and the scale of the animations, and for what
// they did not change those of the node (the identity for a node that the file gave as a matrix).
//
// Flutter: no counterpart
func (p *Pose) Local(n *model.Node) model.TRS {
	t := model.IdentityTRS()
	if n.TRS != nil {
		t = *n.TRS
	}
	if p == nil {
		return t
	}
	if pn := p.nodes[n]; pn != nil {
		if pn.has&1 != 0 {
			t.Translation = pn.trs.Translation
		}
		if pn.has&2 != 0 {
			t.Rotation = pn.trs.Rotation
		}
		if pn.has&4 != 0 {
			t.Scale = pn.trs.Scale
		}
	}
	return t
}

// Matrix is the local matrix of a node in the pose: the node's own when no animation changed it.
//
// Flutter: no counterpart
func (p *Pose) Matrix(n *model.Node) vecmath.Mat4 {
	if p != nil {
		if pn := p.nodes[n]; pn != nil && pn.has != 0 {
			return p.Local(n).Matrix()
		}
	}
	return n.Transform
}

// Weights are the morph weights that an animation gave a node, or nil.
//
// Flutter: no counterpart
func (p *Pose) Weights(n *model.Node) []float32 {
	if p == nil {
		return nil
	}
	if w := p.weights[n]; len(w) > 0 {
		return w
	}
	return nil
}

func (p *Pose) node(n *model.Node) *poseNode {
	pn := p.nodes[n]
	if pn == nil {
		pn = &poseNode{}
		p.nodes[n] = pn
	}
	return pn
}

// Animation plays a model.Animation: it gives the pose at a time (SPEC §28.5). It has no blending and no state: the app chooses
// the time (looping it as it likes) and the animation to sample.
//
// Flutter: no counterpart
type Animation struct {
	a *model.Animation
}

// NewAnimation makes the player of an animation of a model.
//
// Flutter: no counterpart
func NewAnimation(a *model.Animation) *Animation { return &Animation{a: a} }

// Name is the name of the animation.
//
// Flutter: no counterpart
func (a *Animation) Name() string { return a.a.Name }

// Duration is the time of the last keyframe, in seconds.
//
// Flutter: no counterpart
func (a *Animation) Duration() float32 { return a.a.Duration() }

// Sample writes the value of every channel at the time t (in seconds) into the pose, over what it had for those nodes and parts.
// Before the first keyframe a channel has the first value and after the last one the last value: looping is the caller's. The
// interpolation of a channel is its own: STEP keeps a value until the next time, LINEAR blends (rotations on the shortest arc), and
// CUBICSPLINE is the cubic Hermite spline of the specification (glTF 2.0, appendix C), whose rotations are normalized.
//
// Flutter: no counterpart
func (a *Animation) Sample(t float32, p *Pose) {
	for i := range a.a.Channels {
		c := &a.a.Channels[i]
		if c.Node == nil || len(c.Times) == 0 {
			continue
		}
		switch c.Path {
		case model.PathTranslation:
			var v [3]float32
			sampleChannel(c, t, 3, v[:])
			pn := p.node(c.Node)
			pn.trs.Translation, pn.has = vecmath.Vec3{X: float64(v[0]), Y: float64(v[1]), Z: float64(v[2])}, pn.has|1
		case model.PathScale:
			var v [3]float32
			sampleChannel(c, t, 3, v[:])
			pn := p.node(c.Node)
			pn.trs.Scale, pn.has = vecmath.Vec3{X: float64(v[0]), Y: float64(v[1]), Z: float64(v[2])}, pn.has|4
		case model.PathRotation:
			var v [4]float32
			sampleRotation(c, t, &v)
			pn := p.node(c.Node)
			pn.trs.Rotation, pn.has = vecmath.Quat{X: float64(v[0]), Y: float64(v[1]), Z: float64(v[2]), W: float64(v[3])}, pn.has|2
		case model.PathWeights:
			n := c.Width()
			w := p.weights[c.Node]
			if cap(w) < n {
				w = make([]float32, n)
			}
			w = w[:n]
			sampleChannel(c, t, n, w)
			p.weights[c.Node] = w
		}
	}
}

// keyframe finds the keyframe k with Times[k] <= t < Times[k+1], and the fraction s of the way between them; k is the last
// keyframe (and s 0) at or after the last time and before the first one (and s 0) before the first.
func keyframe(times []float32, t float32) (k int, s float32) {
	n := len(times)
	if t <= times[0] {
		return 0, 0
	}
	if t >= times[n-1] {
		return n - 1, 0
	}
	k = sort.Search(n, func(i int) bool { return times[i] > t }) - 1
	return k, (t - times[k]) / (times[k+1] - times[k])
}

// sampleChannel writes the width values of the channel at t.
func sampleChannel(c *model.AnimationChannel, t float32, width int, out []float32) {
	k, s := keyframe(c.Times, t)
	last := len(c.Times) - 1
	switch c.Interpolation {
	case model.InterpolationStep:
		copy(out, c.Values[k*width:(k+1)*width])
	case model.InterpolationCubicSpline:
		// Each keyframe has three groups of values: the in-tangent, the value and the out-tangent.
		if k == last || s == 0 {
			copy(out, c.Values[(3*k+1)*width:(3*k+2)*width])
			return
		}
		hermite(c, k, s, width, out)
	default:
		if k == last || s == 0 {
			copy(out, c.Values[k*width:(k+1)*width])
			return
		}
		a, b := c.Values[k*width:(k+1)*width], c.Values[(k+1)*width:(k+2)*width]
		for i := range out[:width] {
			out[i] = a[i] + (b[i]-a[i])*s
		}
	}
}

// hermite writes the cubic Hermite spline between the keyframes k and k+1 at the fraction s.
func hermite(c *model.AnimationChannel, k int, s float32, width int, out []float32) {
	dt := c.Times[k+1] - c.Times[k]
	v0, b0 := c.Values[(3*k+1)*width:(3*k+2)*width], c.Values[(3*k+2)*width:(3*k+3)*width]
	a1, v1 := c.Values[(3*(k+1))*width:(3*(k+1)+1)*width], c.Values[(3*(k+1)+1)*width:(3*(k+1)+2)*width]
	s2, s3 := s*s, s*s*s
	h00, h10, h01, h11 := 2*s3-3*s2+1, s3-2*s2+s, -2*s3+3*s2, s3-s2
	for i := 0; i < width; i++ {
		out[i] = h00*v0[i] + h10*dt*b0[i] + h01*v1[i] + h11*dt*a1[i]
	}
}

// sampleRotation writes the quaternion of a rotation channel at t, of length 1.
func sampleRotation(c *model.AnimationChannel, t float32, out *[4]float32) {
	k, s := keyframe(c.Times, t)
	last := len(c.Times) - 1
	switch {
	case c.Interpolation == model.InterpolationStep:
		copy(out[:], c.Values[k*4:(k+1)*4])
	case c.Interpolation == model.InterpolationCubicSpline:
		if k == last || s == 0 {
			copy(out[:], c.Values[(3*k+1)*4:(3*k+2)*4])
		} else {
			hermite(c, k, s, 4, out[:])
		}
	case k == last || s == 0:
		copy(out[:], c.Values[k*4:(k+1)*4])
	default:
		a := quatOf(c.Values[k*4 : (k+1)*4])
		b := quatOf(c.Values[(k+1)*4 : (k+2)*4])
		q := a.Slerp(b, float64(s))
		*out = [4]float32{float32(q.X), float32(q.Y), float32(q.Z), float32(q.W)}
		return
	}
	// The spline and the step are normalized, so that a rotation stays one.
	l := math.Sqrt(float64(out[0]*out[0] + out[1]*out[1] + out[2]*out[2] + out[3]*out[3]))
	if l > 0 {
		for i := range out {
			out[i] = float32(float64(out[i]) / l)
		}
	}
}

func quatOf(v []float32) vecmath.Quat {
	return vecmath.Quat{X: float64(v[0]), Y: float64(v[1]), Z: float64(v[2]), W: float64(v[3])}.Normalize()
}
