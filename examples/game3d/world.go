package main

import (
	"math"
	"strings"

	"github.com/rm4n0s/glutter/model"
	"github.com/rm4n0s/glutter/vecmath"
)

// The rules of the game, with no window, no GPU and no sound: a world of boxes that the character walks on and runs into, coins
// to pick up and lava to avoid. The page feeds it the input of a frame (step) and shows what it holds.

// The size of the character and how it moves.
const (
	charRadius  = 0.35 // the character is a box of this half width and depth,
	charHeight  = 1.75 // and this tall
	runSpeed    = 5.5  // metres a second
	runAccel    = 38.0 // metres a second, each second, on the ground
	airAccel    = 14.0 // and in the air
	gravity     = 22.0
	jumpSpeed   = 8.0 // a jump 1.45 m high
	fallLimit   = -12.0
	maxHealth   = 3
	hurtTime    = 1.6 // seconds in which the character cannot be hurt again
	stunTime    = 0.4 // seconds in which the character is thrown and cannot steer
	coinReach   = 0.8 // how near the middle of the character a coin is picked up
	turnSpeed   = 14.0
	terminalFal = -30.0
)

// box is an axis-aligned box of the level.
type box struct{ min, max vecmath.Vec3 }

// overlaps says whether two boxes share some volume.
func (a box) overlaps(b box) bool {
	return a.min.X < b.max.X && a.max.X > b.min.X && a.min.Y < b.max.Y && a.max.Y > b.min.Y && a.min.Z < b.max.Z && a.max.Z > b.min.Z
}

func (a box) centre() vecmath.Vec3 { return a.min.Add(a.max).Scale(0.5) }

// coin is a coin of the level: where it is and whether it was picked up.
type coin struct {
	at    vecmath.Vec3
	taken bool
}

// state is where a game is: playing, won (every coin picked up) or lost (no health left).
type state int

const (
	statePlaying state = iota
	stateWon
	stateLost
)

// input is what the player asks for in one frame, from whichever device: the move in the world's plane (x and z, of length at most
// 1: the direction in which to run and how fast), and the press of the jump button in this frame (a held button is not a press).
type input struct {
	move vecmath.Vec3
	jump bool
}

// events are the things that happened in a step, for the sound and the rumble.
type events struct {
	jumped, coin, hit, won, lost bool
}

// world is the whole game.
type world struct {
	solids  []box
	hazards []box
	coins   []coin
	spawn   vecmath.Vec3

	// The character: its feet, its velocity, the direction it faces (an angle about y: 0 faces +z), and whether it stands.
	pos, vel vecmath.Vec3
	yaw      float64
	onGround bool

	health int
	hurt   float64 // seconds left in which the character cannot be hurt
	stun   float64 // seconds left in which it is thrown about
	picked int
	state  state

	// The animation that plays and for how long it has.
	anim     string
	animTime float64
}

// newWorld makes the world of a level model: the nodes whose names say what they are (Ground, Platform, Wall and Pillar are solid,
// Hazard hurts, Coin is picked up, Spawn is where the character starts), with the boxes of their meshes at the place of their nodes.
func newWorld(level *model.Model) *world {
	w := &world{}
	var walk func(n *model.Node, parent vecmath.Mat4)
	walk = func(n *model.Node, parent vecmath.Mat4) {
		at := parent.Mul(n.Transform)
		name := n.Name
		switch {
		case strings.HasPrefix(name, "Coin"):
			w.coins = append(w.coins, coin{at: at.Translation()})
		case name == "Spawn":
			w.spawn = at.Translation()
		case strings.HasPrefix(name, "Hazard"), strings.HasPrefix(name, "Ground"), strings.HasPrefix(name, "Platform"),
			strings.HasPrefix(name, "Wall"), strings.HasPrefix(name, "Pillar"):
			for _, mesh := range n.Meshes {
				b := meshBox(mesh, at)
				if strings.HasPrefix(name, "Hazard") {
					w.hazards = append(w.hazards, b)
				} else {
					w.solids = append(w.solids, b)
				}
			}
		}
		for _, c := range n.Children {
			walk(c, at)
		}
	}
	for _, n := range level.Nodes {
		walk(n, vecmath.Mat4Identity())
	}
	w.reset()
	return w
}

// meshBox is the box of a mesh at a transform that only moves it (a level has no rotated or scaled box).
func meshBox(mesh *model.Mesh, at vecmath.Mat4) box {
	lo, hi := vecmath.Vec3{X: math.Inf(1), Y: math.Inf(1), Z: math.Inf(1)}, vecmath.Vec3{X: math.Inf(-1), Y: math.Inf(-1), Z: math.Inf(-1)}
	for i := 0; i < mesh.VertexCount(); i++ {
		p := mesh.Position(i)
		lo, hi = lo.Min(p), hi.Max(p)
	}
	move := at.Translation()
	return box{min: lo.Add(move), max: hi.Add(move)}
}

// reset starts the game again: the character at the spawn with all its health, every coin back.
func (w *world) reset() {
	w.pos, w.vel, w.yaw, w.onGround = w.spawn, vecmath.Vec3{}, 0, true
	w.health, w.hurt, w.stun, w.picked, w.state = maxHealth, 0, 0, 0, statePlaying
	for i := range w.coins {
		w.coins[i].taken = false
	}
	w.setAnim("idle")
}

func (w *world) setAnim(name string) {
	if w.anim != name {
		w.anim, w.animTime = name, 0
	}
}

// bounds is the box of the character at its place.
func (w *world) bounds() box {
	return box{
		min: vecmath.Vec3{X: w.pos.X - charRadius, Y: w.pos.Y, Z: w.pos.Z - charRadius},
		max: vecmath.Vec3{X: w.pos.X + charRadius, Y: w.pos.Y + charHeight, Z: w.pos.Z + charRadius},
	}
}

func approach(v, target, by float64) float64 {
	switch {
	case v < target:
		return math.Min(v+by, target)
	case v > target:
		return math.Max(v-by, target)
	}
	return v
}

// turnTo turns the angle a towards b by at most step, the short way round.
func turnTo(a, b, step float64) float64 {
	d := math.Mod(b-a, 2*math.Pi)
	if d > math.Pi {
		d -= 2 * math.Pi
	} else if d < -math.Pi {
		d += 2 * math.Pi
	}
	if math.Abs(d) <= step {
		return b
	}
	return a + math.Copysign(step, d)
}

// step plays dt seconds of the game with the input of the frame and says what happened.
func (w *world) step(dt float64, in input) events {
	var ev events
	w.animTime += dt
	if w.state != statePlaying {
		w.vel.X, w.vel.Z = 0, 0
		w.setAnim("idle")
		return ev
	}
	if w.hurt > 0 {
		w.hurt = math.Max(0, w.hurt-dt)
	}
	if w.stun > 0 {
		w.stun = math.Max(0, w.stun-dt)
	}

	// Steer: the velocity on the ground plane goes towards what the player asks, unless the character is being thrown.
	if w.stun == 0 {
		accel := airAccel
		if w.onGround {
			accel = runAccel
		}
		want := vecmath.Vec3{X: in.move.X, Z: in.move.Z}
		if l := want.Length(); l > 1 {
			want = want.Scale(1 / l)
		}
		w.vel.X = approach(w.vel.X, want.X*runSpeed, accel*dt)
		w.vel.Z = approach(w.vel.Z, want.Z*runSpeed, accel*dt)
		if want.Length() > 0.1 {
			w.yaw = turnTo(w.yaw, math.Atan2(want.X, want.Z), turnSpeed*dt)
		}
		if in.jump && w.onGround {
			w.vel.Y = jumpSpeed
			w.onGround = false
			ev.jumped = true
		}
	}
	w.vel.Y = math.Max(w.vel.Y-gravity*dt, terminalFal)

	w.move(dt)
	if w.pos.Y < fallLimit {
		// Fell off the world: back to the start, with a hurt.
		w.pos, w.vel = w.spawn, vecmath.Vec3{}
		w.onGround = true
		w.hurtBy(vecmath.Vec3{}, &ev, false)
	}
	w.pick(&ev)
	if w.hurt == 0 {
		me := w.bounds()
		me.min = me.min.Add(vecmath.Vec3{X: 0.06, Y: 0.04, Z: 0.06})
		me.max = me.max.Sub(vecmath.Vec3{X: 0.06, Y: 0.06, Z: 0.06})
		for _, h := range w.hazards {
			if me.overlaps(h) {
				w.hurtBy(h.centre(), &ev, true)
				break
			}
		}
	}

	speed := math.Hypot(w.vel.X, w.vel.Z)
	switch {
	case w.state == stateLost:
		w.setAnim("hit")
	case w.stun > 0:
		w.setAnim("hit")
	case !w.onGround:
		w.setAnim("jump")
	case speed > 0.5:
		w.setAnim("run")
	default:
		w.setAnim("idle")
	}
	return ev
}

// move moves the character by its velocity, one axis at a time, so that it slides along what it meets, and lands on what it falls on.
func (w *world) move(dt float64) {
	const eps = 1e-4
	w.pos.X += w.vel.X * dt
	for _, s := range w.solids {
		if me := w.bounds(); me.overlaps(s) {
			if w.vel.X > 0 {
				w.pos.X = s.min.X - charRadius - eps
			} else {
				w.pos.X = s.max.X + charRadius + eps
			}
			w.vel.X = 0
		}
	}
	w.pos.Z += w.vel.Z * dt
	for _, s := range w.solids {
		if me := w.bounds(); me.overlaps(s) {
			if w.vel.Z > 0 {
				w.pos.Z = s.min.Z - charRadius - eps
			} else {
				w.pos.Z = s.max.Z + charRadius + eps
			}
			w.vel.Z = 0
		}
	}
	w.pos.Y += w.vel.Y * dt
	w.onGround = false
	for _, s := range w.solids {
		if me := w.bounds(); me.overlaps(s) {
			if w.vel.Y <= 0 {
				w.pos.Y = s.max.Y
				w.onGround = true
			} else {
				w.pos.Y = s.min.Y - charHeight - eps
			}
			w.vel.Y = 0
		}
	}
	// Standing on a box leaves the character a hair above it, so that the next step's fall touches it again.
	if !w.onGround && w.vel.Y == 0 {
		w.vel.Y = -0.01
	}
}

// pick picks up the coins that are near the middle of the character.
func (w *world) pick(ev *events) {
	mid := w.pos.Add(vecmath.Vec3{Y: charHeight / 2})
	for i := range w.coins {
		c := &w.coins[i]
		if !c.taken && c.at.DistanceTo(mid) < coinReach+charHeight/4 {
			c.taken = true
			w.picked++
			ev.coin = true
		}
	}
	if w.picked == len(w.coins) && len(w.coins) > 0 {
		w.state = stateWon
		ev.won = true
	}
}

// hurtBy takes a point of health, throws the character away from a place (when it is a hit and not a fall) and makes it
// untouchable for a moment. A character with no health left has lost.
func (w *world) hurtBy(from vecmath.Vec3, ev *events, knock bool) {
	w.health--
	w.hurt = hurtTime
	ev.hit = true
	if knock {
		away := vecmath.Vec3{X: w.pos.X - from.X, Z: w.pos.Z - from.Z}
		if away.Length() < 1e-6 {
			away = vecmath.Vec3{Z: -1}
		}
		away = away.Normalize()
		w.vel = vecmath.Vec3{X: away.X * 6, Y: 6, Z: away.Z * 6}
		w.onGround = false
		w.stun = stunTime
	}
	if w.health <= 0 {
		w.health = 0
		w.state = stateLost
		ev.lost = true
	}
}
