package main

import (
	"context"
	"math"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/vecmath"
)

const tick = 1.0 / 60

func loadWorld(t testing.TB) (*world, *assets) {
	t.Helper()
	a, err := loadAssets(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return newWorld(a.level), a
}

// run plays n frames with the input and returns what happened over them.
func run(w *world, n int, in input) events {
	var all events
	for i := 0; i < n; i++ {
		ev := w.step(tick, in)
		all.jumped = all.jumped || ev.jumped
		all.coin = all.coin || ev.coin
		all.hit = all.hit || ev.hit
		all.won = all.won || ev.won
		all.lost = all.lost || ev.lost
		in.jump = false
	}
	return all
}

func near(a, b, eps float64) bool { return math.Abs(a-b) <= eps }

func TestTheLevelIsMadeOfItsNodes(t *testing.T) {
	w, a := loadWorld(t)
	if len(w.coins) != 12 || len(w.hazards) != 4 || len(w.solids) != 17 {
		t.Errorf("%d coins, %d hazards, %d solids; want 12, 4 and 17", len(w.coins), len(w.hazards), len(w.solids))
	}
	if w.spawn != (vecmath.Vec3{Z: -16}) {
		t.Errorf("the spawn is %v", w.spawn)
	}
	// The ground is the box of the mesh at its node: 42 by 42, its top at 0.
	var ground *box
	for i := range w.solids {
		if s := w.solids[i]; near(s.max.Y, 0, 1e-6) && near(s.max.X-s.min.X, 42, 1e-6) {
			ground = &w.solids[i]
		}
	}
	if ground == nil || !near(ground.min.Y, -1, 1e-6) {
		t.Errorf("no ground in %v", w.solids)
	}
	if got := len(a.character.Animations); got != 4 {
		t.Errorf("the character has %d animations, want idle, run, jump and hit", got)
	}
	names := map[string]bool{}
	for _, an := range a.character.Animations {
		names[an.Name] = an.Duration() > 0.3
	}
	for _, n := range []string{"idle", "run", "jump", "hit"} {
		if !names[n] {
			t.Errorf("no animation %q (or it is too short): %v", n, names)
		}
	}
	if len(a.character.Skins) != 1 || len(a.character.Skins[0].Joints) != 7 {
		t.Errorf("skins %v", a.character.Skins)
	}
}

func TestTheCharacterStandsOnTheGround(t *testing.T) {
	w, _ := loadWorld(t)
	w.pos.Y = 3
	run(w, 120, input{})
	if !w.onGround || !near(w.pos.Y, 0, 1e-3) || w.anim != "idle" {
		t.Errorf("after the fall the character is at %v, on the ground %v, animation %q", w.pos, w.onGround, w.anim)
	}
}

func TestMovingAndTurning(t *testing.T) {
	tests := []struct {
		name string
		move vecmath.Vec3
		want func(w *world) bool
	}{
		{"forward along +z", vecmath.Vec3{Z: 1}, func(w *world) bool { return w.pos.Z > -16+3 && near(w.pos.X, 0, 1e-6) && near(w.yaw, 0, 1e-6) }},
		{"to the side along +x", vecmath.Vec3{X: 1}, func(w *world) bool { return w.pos.X > 3 && near(w.yaw, math.Pi/2, 0.05) }},
		{"half a push is slower", vecmath.Vec3{Z: 0.5}, func(w *world) bool { return w.pos.Z > -16+1 && w.pos.Z < -16+3 }},
		{"more than a push counts as one", vecmath.Vec3{Z: 5}, func(w *world) bool { return w.pos.Z < -16+6 }},
		{"a diagonal is not faster", vecmath.Vec3{X: 1, Z: 1}, func(w *world) bool { return math.Hypot(w.pos.X, w.pos.Z+16) < 5.5*1.05 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, _ := loadWorld(t)
			run(w, 60, input{move: tt.move})
			if !tt.want(w) {
				t.Errorf("after a second the character is at %v facing %v", w.pos, w.yaw)
			}
			if w.anim != "run" {
				t.Errorf("animation %q, want run", w.anim)
			}
		})
	}
}

func TestAJumpIsAsHighAsItShouldBe(t *testing.T) {
	w, _ := loadWorld(t)
	run(w, 5, input{}) // settle
	ev := w.step(tick, input{jump: true})
	if !ev.jumped || w.onGround || w.anim != "jump" {
		t.Fatalf("jumped %v, on the ground %v, animation %q", ev.jumped, w.onGround, w.anim)
	}
	top := 0.0
	for i := 0; i < 120 && !w.onGround; i++ {
		w.step(tick, input{})
		top = math.Max(top, w.pos.Y)
	}
	want := jumpSpeed * jumpSpeed / (2 * gravity)
	if !near(top, want, 0.08) || !w.onGround || !near(w.pos.Y, 0, 1e-3) {
		t.Errorf("the jump reached %.3f (want %.3f) and the character is at %v on the ground %v", top, want, w.pos, w.onGround)
	}
	// A jump in the air is not one.
	run(w, 5, input{})
	w.step(tick, input{jump: true})
	if ev := w.step(tick, input{jump: true}); ev.jumped {
		t.Error("a second jump in the air")
	}
}

func TestPlatformsAreClimbedAndWallsStop(t *testing.T) {
	w, _ := loadWorld(t)
	// Platform.1 is 3 by 3 at (-3, -10), 0.7 high: jump from the ground in front of it, run onto it, stand.
	w.pos = vecmath.Vec3{X: -3, Y: 0, Z: -13}
	run(w, 3, input{})
	run(w, 1, input{move: vecmath.Vec3{Z: 1}, jump: true})
	run(w, 40, input{move: vecmath.Vec3{Z: 1}})
	run(w, 60, input{})
	if !w.onGround || !near(w.pos.Y, 0.7, 1e-3) || w.pos.Z < -11.5 || w.pos.Z > -8.5 {
		t.Errorf("the character is at %v, on the ground %v: it should stand on the platform (0.7 high, z from -11.5 to -8.5)", w.pos, w.onGround)
	}
	// It does not walk through a wall: the north wall's face is at z = 21.
	w.pos = vecmath.Vec3{X: 4, Y: 0, Z: 17}
	run(w, 200, input{move: vecmath.Vec3{Z: 1}})
	if !near(w.pos.Z, 21-charRadius, 0.01) {
		t.Errorf("the character stopped at z = %v, want %v", w.pos.Z, 21-charRadius)
	}
	// And it slides along it.
	x := w.pos.X
	run(w, 30, input{move: vecmath.Vec3{X: 1, Z: 1}})
	if w.pos.X < x+1.5 || !near(w.pos.Z, 21-charRadius, 0.01) {
		t.Errorf("sliding along the wall the character is at %v", w.pos)
	}
}

func TestACeilingStopsTheJump(t *testing.T) {
	w := &world{solids: []box{
		{vecmath.Vec3{X: -10, Y: -1, Z: -10}, vecmath.Vec3{X: 10, Y: 0, Z: 10}},
		{vecmath.Vec3{X: -10, Y: 2, Z: -10}, vecmath.Vec3{X: 10, Y: 3, Z: 10}},
	}}
	w.reset()
	run(w, 5, input{})
	w.step(tick, input{jump: true})
	top := 0.0
	for i := 0; i < 90; i++ {
		w.step(tick, input{})
		top = math.Max(top, w.pos.Y+charHeight)
	}
	if top > 2+1e-3 || !w.onGround {
		t.Errorf("the head reached %v under a ceiling at 2; on the ground %v", top, w.onGround)
	}
}

func TestCoinsArePickedUpAndTheGameIsWon(t *testing.T) {
	w, _ := loadWorld(t)
	w.pos = vecmath.Vec3{X: 0, Y: 0, Z: -12.2} // the first coin is at (0, 0.8, -12)
	ev := run(w, 2, input{})
	if !ev.coin || w.picked != 1 || !w.coins[0].taken || w.state != statePlaying {
		t.Errorf("coin %v, picked %d", ev.coin, w.picked)
	}
	if ev := run(w, 5, input{}); ev.coin || w.picked != 1 {
		t.Error("a coin was picked twice")
	}
	for i := range w.coins {
		w.coins[i].taken = true
	}
	w.picked = len(w.coins) - 1
	w.coins[5].taken = false
	w.pos = w.coins[5].at.Sub(vecmath.Vec3{Y: 0.8})
	w.vel = vecmath.Vec3{}
	ev = run(w, 2, input{})
	if !ev.won || w.state != stateWon {
		t.Errorf("won %v, state %v after the last coin", ev.won, w.state)
	}
	// A game that is won does not go on.
	at := w.pos
	run(w, 30, input{move: vecmath.Vec3{X: 1}})
	if w.pos.X != at.X {
		t.Error("the character moved after the game was won")
	}
	w.reset()
	if w.state != statePlaying || w.picked != 0 || w.health != maxHealth || w.pos != w.spawn || w.coins[0].taken {
		t.Errorf("reset left %+v", w)
	}
}

func TestLavaHurtsOnceForAWhileAndThreeHurtsLose(t *testing.T) {
	w, _ := loadWorld(t)
	// Hazard.4 is a tile of 2 by 2 at (2, -1).
	w.pos = vecmath.Vec3{X: 2, Y: 0, Z: -1}
	ev := run(w, 2, input{})
	if !ev.hit || w.health != maxHealth-1 || w.hurt <= 0 || w.stun <= 0 {
		t.Fatalf("hit %v, health %d, hurt %v, stun %v", ev.hit, w.health, w.hurt, w.stun)
	}
	if w.vel.Y <= 0 && w.onGround {
		t.Errorf("the character was not thrown: %v", w.vel)
	}
	if w.anim != "hit" {
		t.Errorf("animation %q, want hit", w.anim)
	}
	// While it cannot be hurt, lava does nothing.
	w.pos = vecmath.Vec3{X: 2, Y: 0, Z: -1}
	if ev := run(w, 10, input{}); ev.hit || w.health != maxHealth-1 {
		t.Errorf("hurt again too soon: health %d", w.health)
	}
	// After the wait it hurts again, and with no health left the game is lost.
	for w.state == statePlaying {
		w.hurt = 0
		w.pos = vecmath.Vec3{X: 2, Y: 0, Z: -1}
		run(w, 2, input{})
	}
	if w.state != stateLost || w.health != 0 {
		t.Errorf("state %v, health %d", w.state, w.health)
	}
}

func TestFallingOffTheWorldIsAHurtAndAReturn(t *testing.T) {
	w := &world{solids: []box{{vecmath.Vec3{X: -2, Y: -1, Z: -2}, vecmath.Vec3{X: 2, Y: 0, Z: 2}}}, spawn: vecmath.Vec3{Y: 0}}
	w.reset()
	w.pos = vecmath.Vec3{X: 5, Y: 0, Z: 0} // off the edge
	var ev events
	for i := 0; i < 200 && !ev.hit; i++ {
		ev = w.step(tick, input{})
	}
	if !ev.hit || w.health != maxHealth-1 || w.pos.Y > 0.5 || !near(w.pos.X, 0, 1e-6) {
		t.Errorf("hit %v, health %d, at %v", ev.hit, w.health, w.pos)
	}
}

func TestApproachAndTurnTo(t *testing.T) {
	approaches := []struct{ v, target, by, want float64 }{
		{0, 5, 1, 1}, {0, 0.5, 1, 0.5}, {5, 0, 2, 3}, {5, 4.5, 2, 4.5}, {3, 3, 1, 3},
	}
	for _, tt := range approaches {
		if got := approach(tt.v, tt.target, tt.by); got != tt.want {
			t.Errorf("approach(%v, %v, %v) = %v, want %v", tt.v, tt.target, tt.by, got, tt.want)
		}
	}
	turns := []struct{ a, b, step, want float64 }{
		{0, 1, 0.5, 0.5}, {0, 1, 2, 1}, {0, -1, 0.5, -0.5},
		{3, -3, 0.2, 3.2},                  // across the half turn the short way
		{-3, 3, 0.2, -3.2},                 //
		{0, 2 * math.Pi, 0.1, 2 * math.Pi}, // a whole turn is no turn: the angle is the same
		{0, math.Pi - 0.1, 5, math.Pi - 0.1},
	}
	for _, tt := range turns {
		if got := turnTo(tt.a, tt.b, tt.step); !near(got, tt.want, 1e-9) {
			t.Errorf("turnTo(%v, %v, %v) = %v, want %v", tt.a, tt.b, tt.step, got, tt.want)
		}
	}
}

func TestAnAssetThatIsNotThere(t *testing.T) {
	_, err := readAsset("assets/nothing.wav")
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T", err)
	}
	if !appErr.HasRoute("readAsset." + tagAssetMissing) {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
	if f, _ := appErr.Metadata["file"].(string); !strings.HasSuffix(f, "nothing.wav") {
		t.Errorf("metadata %v", appErr.Metadata)
	}
}
