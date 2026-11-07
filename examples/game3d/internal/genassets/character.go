package main

import "math"

// The joints of the character, in the order of the skin: the number of a joint is its index here.
const (
	jHips = iota
	jSpine
	jHead
	jArmL
	jArmR
	jLegL
	jLegR
	jointCount
)

var jointNames = [jointCount]string{"Hips", "Spine", "Head", "ArmL", "ArmR", "LegL", "LegR"}

// jointParent is the parent of each joint (-1 for the top one) and jointAt the place of its pivot in the bind pose, in the
// coordinates of the character: the feet at y = 0, facing +z, the left side at +x.
var (
	jointParent = [jointCount]int{-1, jHips, jSpine, jSpine, jSpine, jHips, jHips}
	jointAt     = [jointCount][3]float64{
		jHips:  {0, 0.95, 0},
		jSpine: {0, 1.05, 0},
		jHead:  {0, 1.50, 0},
		jArmL:  {0.30, 1.45, 0},
		jArmR:  {-0.30, 1.45, 0},
		jLegL:  {0.11, 0.90, 0},
		jLegR:  {-0.11, 0.90, 0},
	}
)

func local(j int) [3]float64 {
	at := jointAt[j]
	if p := jointParent[j]; p >= 0 {
		for i := range at {
			at[i] -= jointAt[p][i]
		}
	}
	return at
}

// makeCharacter writes the character: a figure of boxes with a skeleton of seven joints and four animations (idle, run, jump,
// hit), as a skinned glTF file.
func makeCharacter() []byte {
	g := &glb{}
	shirt := g.material("shirt", [4]float64{0.16, 0.38, 0.85, 1}, 0, 0.7, [3]float64{})
	pants := g.material("pants", [4]float64{0.12, 0.13, 0.20, 1}, 0, 0.8, [3]float64{})
	skin := g.material("skin", [4]float64{0.93, 0.72, 0.58, 1}, 0, 0.6, [3]float64{})
	shoes := g.material("shoes", [4]float64{0.92, 0.92, 0.90, 1}, 0, 0.5, [3]float64{})
	eyes := g.material("eyes", [4]float64{0.02, 0.02, 0.03, 1}, 0, 0.4, [3]float64{})

	var gs, gp, gk, gh, ge geometry
	// The boxes follow the bind pose: each belongs to the joint that moves it.
	gp.box(0, 0.95, 0, 0.20, 0.10, 0.11, jHips, true)
	gs.box(0, 1.275, 0, 0.22, 0.225, 0.12, jSpine, true)
	gk.box(0, 1.64, 0, 0.13, 0.14, 0.13, jHead, true)
	ge.box(0.06, 1.68, 0.131, 0.025, 0.03, 0.01, jHead, true)
	ge.box(-0.06, 1.68, 0.131, 0.025, 0.03, 0.01, jHead, true)
	for _, side := range []struct {
		x     float64
		joint uint16
	}{{0.30, jArmL}, {-0.30, jArmR}} {
		gs.box(side.x, 1.17, 0, 0.07, 0.28, 0.07, side.joint, true)
		gk.box(side.x, 0.84, 0, 0.075, 0.07, 0.075, side.joint, true)
	}
	for _, side := range []struct {
		x     float64
		joint uint16
	}{{0.11, jLegL}, {-0.11, jLegR}} {
		gp.box(side.x, 0.50, 0, 0.09, 0.40, 0.09, side.joint, true)
		gh.box(side.x, 0.05, 0.04, 0.095, 0.05, 0.14, side.joint, true)
	}
	prim := []any{g.primitive(&gs, shirt), g.primitive(&gp, pants), g.primitive(&gk, skin), g.primitive(&gh, shoes), g.primitive(&ge, eyes)}
	g.Meshes = append(g.Meshes, map[string]any{"name": "Body", "primitives": prim})

	// The nodes: 0 is the character, then the joints 1..7, then the mesh node 8.
	nodeOf := func(j int) int { return 1 + j }
	children := []int{nodeOf(jHips), 1 + jointCount}
	g.Nodes = append(g.Nodes, map[string]any{"name": "Character", "children": children})
	kids := map[int][]int{}
	for j := 1; j < jointCount; j++ {
		kids[jointParent[j]] = append(kids[jointParent[j]], nodeOf(j))
	}
	for j := 0; j < jointCount; j++ {
		n := map[string]any{"name": jointNames[j], "translation": local(j)}
		if c := kids[j]; len(c) > 0 {
			n["children"] = c
		}
		g.Nodes = append(g.Nodes, n)
	}
	joints := make([]int, jointCount)
	var ibm []float32
	for j := range joints {
		joints[j] = nodeOf(j)
		at := jointAt[j]
		// The inverse of the bind pose of a joint with no rotation is the translation by minus its place (column-major).
		ibm = append(ibm, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, float32(-at[0]), float32(-at[1]), float32(-at[2]), 1)
	}
	g.Skins = append(g.Skins, map[string]any{"name": "Skeleton", "joints": joints, "skeleton": nodeOf(jHips), "inverseBindMatrices": g.floats(ibm, "MAT4", 0, false)})
	g.Nodes = append(g.Nodes, map[string]any{"name": "Body", "mesh": 0, "skin": 0})

	for _, a := range characterAnimations() {
		g.Animations = append(g.Animations, a.write(g, nodeOf))
	}
	return g.bytes([]int{0})
}

// track is the keys of one joint's property over an animation.
type track struct {
	joint int
	path  string // "rotation" or "translation"
	vals  []float32
}

type animation struct {
	name     string
	duration float64
	keys     int
	tracks   []track
}

func (a animation) times() []float32 {
	t := make([]float32, a.keys)
	for i := range t {
		t[i] = float32(a.duration * float64(i) / float64(a.keys-1))
	}
	return t
}

// write adds the animation to the file: one sampler for each track, sharing the times.
func (a animation) write(g *glb, nodeOf func(int) int) map[string]any {
	in := g.floats(a.times(), "SCALAR", 0, true)
	var samplers, channels []any
	for i, t := range a.tracks {
		kind := "VEC4"
		if t.path == "translation" {
			kind = "VEC3"
		}
		out := g.floats(t.vals, kind, 0, false)
		samplers = append(samplers, map[string]any{"input": in, "output": out, "interpolation": "LINEAR"})
		channels = append(channels, map[string]any{"sampler": i, "target": map[string]any{"node": nodeOf(t.joint), "path": t.path}})
	}
	return map[string]any{"name": a.name, "samplers": samplers, "channels": channels}
}

// quat is the rotation by an angle in radians about an axis (0 x, 1 y, 2 z), as the four numbers of a glTF quaternion.
func quat(axis int, angle float64) [4]float32 {
	s, c := math.Sin(angle/2), math.Cos(angle/2)
	q := [4]float32{0, 0, 0, float32(c)}
	q[axis] = float32(s)
	return q
}

// rotations makes the track of a joint that turns about an axis by f(phase) radians, for the phase 0 to 1 over the keys.
func rotations(joint, axis, keys int, f func(phase float64) float64) track {
	t := track{joint: joint, path: "rotation"}
	for i := 0; i < keys; i++ {
		q := quat(axis, f(float64(i)/float64(keys-1)))
		t.vals = append(t.vals, q[:]...)
	}
	return t
}

// rotationsXZ turns about the x axis and then, if it is nonzero, the z axis: for the arms of the hit.
func rotationsXZ(joint, keys int, fx, fz func(phase float64) float64) track {
	t := track{joint: joint, path: "rotation"}
	for i := 0; i < keys; i++ {
		p := float64(i) / float64(keys-1)
		ax, az := fx(p), fz(p)
		// The product of the two rotations, x after z.
		sx, cx, sz, cz := math.Sin(ax/2), math.Cos(ax/2), math.Sin(az/2), math.Cos(az/2)
		t.vals = append(t.vals, float32(sx*cz), float32(-sx*sz), float32(cx*sz), float32(cx*cz))
	}
	return t
}

func bob(keys int, f func(phase float64) float64) track {
	t := track{joint: jHips, path: "translation"}
	for i := 0; i < keys; i++ {
		at := jointAt[jHips]
		y := at[1] + f(float64(i)/float64(keys-1))
		t.vals = append(t.vals, float32(at[0]), float32(y), float32(at[2]))
	}
	return t
}

const rad = math.Pi / 180

func characterAnimations() []animation {
	const keys = 13
	sine := func(amp float64) func(float64) float64 {
		return func(p float64) float64 { return amp * math.Sin(2*math.Pi*p) }
	}
	neg := func(f func(float64) float64) func(float64) float64 { return func(p float64) float64 { return -f(p) } }
	zero := func(float64) float64 { return 0 }
	constant := func(v float64) func(float64) float64 { return func(float64) float64 { return v } }
	// The ease of a pose that comes and goes: 0, then 1, then 0.
	pulse := func(p float64) float64 { return math.Sin(math.Pi * p) }
	return []animation{
		{"idle", 2.4, keys, []track{
			rotations(jSpine, 0, keys, sine(2*rad)),
			rotations(jHead, 0, keys, sine(-1.5*rad)),
			rotations(jArmL, 0, keys, sine(4*rad)),
			rotations(jArmR, 0, keys, sine(-4*rad)),
			bob(keys, sine(0.006)),
		}},
		{"run", 0.66, keys, []track{
			rotations(jLegL, 0, keys, sine(50*rad)),
			rotations(jLegR, 0, keys, neg(sine(50*rad))),
			rotations(jArmL, 0, keys, neg(sine(55*rad))),
			rotations(jArmR, 0, keys, sine(55*rad)),
			rotations(jSpine, 0, keys, func(p float64) float64 { return 12*rad + 3*rad*math.Sin(4*math.Pi*p) }),
			rotations(jHead, 0, keys, constant(-8*rad)),
			bob(keys, func(p float64) float64 { return 0.05 * math.Abs(math.Sin(2*math.Pi*p)) }),
		}},
		{"jump", 0.8, keys, []track{
			rotations(jLegL, 0, keys, func(p float64) float64 { return -45 * rad * pulse(p) }),
			rotations(jLegR, 0, keys, func(p float64) float64 { return 10 * rad * pulse(p) }),
			rotationsXZ(jArmL, keys, func(p float64) float64 { return -150 * rad * pulse(p) }, func(p float64) float64 { return -10 * rad * pulse(p) }),
			rotationsXZ(jArmR, keys, func(p float64) float64 { return -150 * rad * pulse(p) }, func(p float64) float64 { return 10 * rad * pulse(p) }),
			rotations(jSpine, 0, keys, func(p float64) float64 { return 6 * rad * pulse(p) }),
		}},
		{"hit", 0.5, keys, []track{
			rotations(jSpine, 0, keys, func(p float64) float64 { return -28 * rad * pulse(p) }),
			rotations(jHead, 0, keys, func(p float64) float64 { return -14 * rad * pulse(p) }),
			rotationsXZ(jArmL, keys, zero, func(p float64) float64 { return 75 * rad * pulse(p) }),
			rotationsXZ(jArmR, keys, zero, func(p float64) float64 { return -75 * rad * pulse(p) }),
			rotations(jLegL, 0, keys, func(p float64) float64 { return 12 * rad * pulse(p) }),
			rotations(jLegR, 0, keys, func(p float64) float64 { return -12 * rad * pulse(p) }),
		}},
	}
}
