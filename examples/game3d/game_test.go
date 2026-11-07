package main

import (
	"math"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/gamepad"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
)

const frame = 16 * time.Millisecond

// newGame starts the game on a tester with a recording GPU device and returns it with the state of the page.
func newGame(t testing.TB, opts ...gluttertest.Opts) (*gluttertest.Tester, *gameState) {
	t.Helper()
	o := gluttertest.Opts{GPU: true}
	if len(opts) > 0 {
		o = opts[0]
	}
	ts := gluttertest.New(t, o)
	ts.SetViewSize(ui.Size{Width: 1280, Height: 720})
	ts.PumpWidget(NewGameApp())
	ts.Pump(frame)
	ts.Pump(frame)
	return ts, gluttertest.StateOf[*gameState](ts, gluttertest.ByType[*GamePage]())
}

func pumpFrames(ts *gluttertest.Tester, n int) {
	for i := 0; i < n; i++ {
		ts.Pump(frame)
	}
}

// energy is the mean square of the samples.
func energy(s []float32) float64 {
	sum := 0.0
	for _, v := range s {
		sum += float64(v) * float64(v)
	}
	return sum / float64(len(s))
}

func TestTheGameStartsWithItsSceneItsHUDAndItsMusic(t *testing.T) {
	ts, s := newGame(t)
	// The level has 17 solid boxes, 4 lava tiles and 12 coins, each a node with a mesh; the character has five primitives.
	if got := len(s.stage.scene.Items()); got != 17+4+12+5 {
		t.Errorf("the scene has %d items, want %d", got, 17+4+12+5)
	}
	if len(s.stage.coins) != 12 || s.stage.character == nil || s.stage.character.Parts[0].Item.Skin() == nil {
		t.Errorf("coins %d, character %v", len(s.stage.coins), s.stage.character)
	}
	// The camera is behind the character, looking along +z and down at it.
	if eye := s.stage.camera.Eye(); eye.Z > s.world.pos.Z-3 || eye.Y < 2 {
		t.Errorf("the camera is at %v, the character at %v", eye, s.world.pos)
	}
	if f := s.stage.camera.Forward(); f.Z < 0.7 || f.Y > -0.1 {
		t.Errorf("the camera looks along %v", f)
	}
	for _, text := range []string{"0 / 12"} {
		if len(ts.Elements(gluttertest.ByText(text))) == 0 {
			t.Errorf("no %q on the screen", text)
		}
	}
	if len(ts.Elements(gluttertest.ByText("Paused"))) != 0 {
		t.Error("the menu is open at the start")
	}
	// The music plays on the device that the game opened.
	dev := ts.Audio().Playback()
	if dev == nil {
		t.Fatal("the game opened no playback device")
	}
	if e := energy(dev.Pull(4410, 441)); e < 1e-5 {
		t.Errorf("the music is silent (energy %g)", e)
	}
}

func TestTheStickRunsAndAJumpLeaves(t *testing.T) {
	ts, s := newGame(t)
	pad := ts.ConnectPad(gluttertest.PadSpec{Name: "Pad", Rumble: true})
	start := s.world.pos
	pad.SetAxis(gamepad.LeftY, -1) // up: forward
	pumpFrames(ts, 60)
	if s.world.pos.Z < start.Z+3 || math.Abs(s.world.pos.X-start.X) > 0.1 || s.world.anim != "run" {
		t.Errorf("after a second of forward the character is at %v (from %v), animation %q", s.world.pos, start, s.world.anim)
	}
	// The right stick turns the camera, and the character runs towards where it looks.
	yaw := s.stage.yaw
	pad.SetAxis(gamepad.RightX, 1)
	pumpFrames(ts, 30)
	pad.SetAxis(gamepad.RightX, 0)
	if s.stage.yaw >= yaw-0.5 {
		t.Errorf("the camera's yaw went from %v to %v: the right stick turns it", yaw, s.stage.yaw)
	}
	pad.SetAxis(gamepad.LeftY, 0)
	pumpFrames(ts, 40)
	pad.Press(gamepad.South)
	pumpFrames(ts, 4)
	if s.world.onGround || s.world.pos.Y < 0.2 || s.world.anim != "jump" {
		t.Errorf("after A the character is at %v, on the ground %v, animation %q", s.world.pos, s.world.onGround, s.world.anim)
	}
	// Holding A is one jump: the character comes down and stays down.
	pumpFrames(ts, 90)
	if !s.world.onGround && s.world.vel.Y > 0 {
		t.Errorf("the character jumped again: %v", s.world.pos)
	}
	pad.Release(gamepad.South)
}

func TestKeysRunLookAndJump(t *testing.T) {
	ts, s := newGame(t)
	start := s.world.pos
	ts.SendKeyDownEvent(services.LogicalKeyboardKeyKeyW)
	pumpFrames(ts, 45)
	ts.SendKeyUpEvent(services.LogicalKeyboardKeyKeyW)
	if s.world.pos.Z < start.Z+2 {
		t.Errorf("W moved the character to %v", s.world.pos)
	}
	// D runs to the camera's right: the camera looks along +z, whose right is -x.
	at := s.world.pos
	ts.SendKeyDownEvent(services.LogicalKeyboardKeyKeyD)
	pumpFrames(ts, 45)
	ts.SendKeyUpEvent(services.LogicalKeyboardKeyKeyD)
	if s.world.pos.X > at.X-2 {
		t.Errorf("D moved the character from %v to %v", at, s.world.pos)
	}
	yaw := s.stage.yaw
	ts.SendKeyDownEvent(services.LogicalKeyboardKeyArrowLeft)
	pumpFrames(ts, 20)
	ts.SendKeyUpEvent(services.LogicalKeyboardKeyArrowLeft)
	if s.stage.yaw <= yaw+0.3 {
		t.Errorf("the left arrow turned the camera from %v to %v", yaw, s.stage.yaw)
	}
	ts.SendKeyDownEvent(services.LogicalKeyboardKeySpace)
	pumpFrames(ts, 5)
	if s.world.onGround {
		t.Error("Space did not jump")
	}
	ts.SendKeyUpEvent(services.LogicalKeyboardKeySpace)
}

func TestPauseFreezesTheGameAndThePadDrivesTheMenu(t *testing.T) {
	ts, s := newGame(t)
	pad := ts.ConnectPad(gluttertest.PadSpec{Name: "Pad"})
	pad.Tap(gamepad.Start)
	pumpFrames(ts, 2)
	if !s.paused || len(ts.Elements(gluttertest.ByText("Paused"))) == 0 {
		t.Fatalf("paused %v after Start", s.paused)
	}
	if !s.snd.paused {
		t.Error("the music is not quieter while paused")
	}
	// A stick that is pushed does nothing in a pause.
	at := s.world.pos
	pad.SetAxis(gamepad.LeftY, -1)
	pumpFrames(ts, 30)
	pad.SetAxis(gamepad.LeftY, 0)
	if s.world.pos != at {
		t.Errorf("the character moved while the game was paused: %v", s.world.pos)
	}
	// The D-pad moves the focus in the menu, and A presses the button that has it: Resume has it at first.
	pad.Tap(gamepad.DPadDown)
	pumpFrames(ts, 2)
	pad.Tap(gamepad.DPadUp)
	pumpFrames(ts, 2)
	pad.Tap(gamepad.South)
	pumpFrames(ts, 2)
	if s.paused || len(ts.Elements(gluttertest.ByText("Paused"))) != 0 || s.snd.paused {
		t.Errorf("A on Resume left the game paused: %v", s.paused)
	}
	// B closes the menu too.
	pad.Tap(gamepad.Start)
	pumpFrames(ts, 2)
	pad.Tap(gamepad.East)
	pumpFrames(ts, 2)
	if s.paused {
		t.Error("B did not close the menu")
	}
	// Esc pauses and resumes; the music switch of the menu turns the music off and on.
	ts.SendKeyEvent(services.LogicalKeyboardKeyEscape)
	pumpFrames(ts, 2)
	if !s.paused {
		t.Fatal("Esc did not pause")
	}
	ts.Tap(gluttertest.ByText("Music"))
	pumpFrames(ts, 2)
	if s.musicOn || s.snd.music.Gain() != 0 {
		t.Errorf("music on %v, gain %v after the switch", s.musicOn, s.snd.music.Gain())
	}
	ts.Tap(gluttertest.ByText("Music"))
	pumpFrames(ts, 2)
	if !s.musicOn || s.snd.music.Gain() == 0 {
		t.Errorf("music on %v, gain %v after the second tap", s.musicOn, s.snd.music.Gain())
	}
	ts.Tap(gluttertest.ByText("Resume"))
	pumpFrames(ts, 2)
	if s.paused {
		t.Error("Resume did not resume")
	}
}

func TestAHitRumblesThePadAndTakesAHeart(t *testing.T) {
	ts, s := newGame(t)
	pad := ts.ConnectPad(gluttertest.PadSpec{Name: "Pad", Rumble: true})
	// Hazard.4 is a tile of lava at (2, -1).
	s.world.pos = vecmath.Vec3{X: 2, Z: -1}
	pumpFrames(ts, 3)
	if s.world.health != maxHealth-1 {
		t.Fatalf("health %d after standing on lava", s.world.health)
	}
	rumbles := pad.Rumbles()
	if len(rumbles) == 0 || rumbles[0].Low == 0 || rumbles[0].Duration < 100*time.Millisecond {
		t.Errorf("the pad's rumbles are %v", rumbles)
	}
	if s.world.anim != "hit" {
		t.Errorf("animation %q", s.world.anim)
	}
	// The sound of the hit goes to the device.
	if e := energy(ts.Audio().Playback().Pull(2205, 441)); e < 1e-4 {
		t.Errorf("silence after the hit (energy %g)", e)
	}
}

func TestACoinSoundsAndTheHUDCounts(t *testing.T) {
	ts, s := newGame(t)
	s.world.pos = s.world.coins[0].at.Sub(vecmath.Vec3{Y: 0.8})
	pumpFrames(ts, 3)
	if s.world.picked != 1 || len(ts.Elements(gluttertest.ByText("1 / 12"))) == 0 {
		t.Errorf("picked %d", s.world.picked)
	}
	if s.stage.coins[0].Visible() {
		t.Error("the coin is still there")
	}
	// The coin's sound plays: the device's frames have more in them than the music of the first moment alone.
	if e := energy(ts.Audio().Playback().Pull(3000, 300)); e < 1e-4 {
		t.Errorf("energy %g", e)
	}
}

func TestTheMouseLooksWhenLockedAndAnEndedLockPauses(t *testing.T) {
	ts, s := newGame(t)
	ts.TapAt(ui.Offset{Dx: 640, Dy: 360}, gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse})
	pumpFrames(ts, 1)
	if !ts.PointerLocked(ui.ImplicitViewID) {
		t.Fatal("a click did not lock the pointer")
	}
	yaw, pitch := s.stage.yaw, s.stage.pitch
	ts.MoveLockedMouse(ui.Offset{Dx: 100, Dy: 40})
	pumpFrames(ts, 2)
	if s.stage.yaw >= yaw-0.1 || s.stage.pitch <= pitch+0.03 {
		t.Errorf("the mouse turned the camera from (%v, %v) to (%v, %v)", yaw, pitch, s.stage.yaw, s.stage.pitch)
	}
	ts.EndPointerLock(ui.ImplicitViewID)
	pumpFrames(ts, 2)
	if !s.paused {
		t.Error("the game goes on after the platform took the pointer back")
	}
	// A pause that the player asked for gives the pointer back.
	ts.Tap(gluttertest.ByText("Resume"))
	pumpFrames(ts, 1)
	ts.TapAt(ui.Offset{Dx: 640, Dy: 360}, gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse})
	ts.SendKeyEvent(services.LogicalKeyboardKeyEscape)
	pumpFrames(ts, 2)
	if ts.PointerLocked(ui.ImplicitViewID) || !s.paused {
		t.Errorf("Esc left the pointer locked: %v", ts.PointerLocked(ui.ImplicitViewID))
	}
}

func TestWinningAndPlayingAgain(t *testing.T) {
	ts, s := newGame(t)
	for i := range s.world.coins {
		s.world.coins[i].taken = true
	}
	s.world.picked = 11
	s.world.coins[3].taken = false
	s.world.pos = s.world.coins[3].at.Sub(vecmath.Vec3{Y: 0.8})
	pumpFrames(ts, 3)
	if s.world.state != stateWon || len(ts.Elements(gluttertest.ByText("You win!"))) == 0 {
		t.Fatalf("state %v after the last coin", s.world.state)
	}
	ts.Tap(gluttertest.ByText("Play again"))
	pumpFrames(ts, 3)
	if s.world.state != statePlaying || s.world.picked != 0 || s.world.pos.Z > -15 || !s.stage.coins[3].Visible() {
		t.Errorf("state %v, picked %d, at %v", s.world.state, s.world.picked, s.world.pos)
	}
	// Three hurts end it.
	for i := 0; i < maxHealth; i++ {
		s.world.hurt = 0
		s.world.pos = vecmath.Vec3{X: 2, Z: -1}
		pumpFrames(ts, 3)
	}
	if s.world.state != stateLost || len(ts.Elements(gluttertest.ByText("Game over"))) == 0 {
		t.Errorf("state %v after three hits", s.world.state)
	}
}

func TestTheSceneIsMadeAgainAfterTheDeviceIsLost(t *testing.T) {
	ts, s := newGame(t)
	before := len(s.stage.scene.Items())
	old := s.stage.level
	ts.GPUDevice().Lose()
	pumpFrames(ts, 4)
	if got := len(s.stage.scene.Items()); got != before || s.stage.level == old || s.stage.level == nil {
		t.Errorf("the scene has %d items, had %d; the level was made again: %v", got, before, s.stage.level != old)
	}
}

func TestWithoutAGPUThePageShowsItsFallbackAndThenAllTheSame(t *testing.T) {
	ts, _ := newGame(t, gluttertest.Opts{})
	if len(ts.Elements(gluttertest.ByText("This app needs a GPU renderer: Vulkan, Direct3D 12 or WebGPU."))) == 0 {
		t.Error("no fallback text")
	}
}

func TestWithoutAudioTheGameIsSilentButPlays(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{GPU: true})
	ts.Audio().SetUnavailable(true)
	ts.SetViewSize(ui.Size{Width: 1280, Height: 720})
	ts.PumpWidget(NewGameApp())
	pumpFrames(ts, 3)
	s := gluttertest.StateOf[*gameState](ts, gluttertest.ByType[*GamePage]())
	if s.snd != nil {
		t.Fatal("sounds with no audio")
	}
	s.world.pos = s.world.coins[0].at.Sub(vecmath.Vec3{Y: 0.8})
	pumpFrames(ts, 3)
	if s.world.picked != 1 {
		t.Error("the game did not go on without sound")
	}
	// Pausing and the menu's music switch are no-ops without sounds.
	ts.SendKeyEvent(services.LogicalKeyboardKeyEscape)
	pumpFrames(ts, 2)
	ts.Tap(gluttertest.ByText("Music"))
	pumpFrames(ts, 2)
	if !s.paused || s.musicOn {
		t.Errorf("paused %v, music %v", s.paused, s.musicOn)
	}
}

func TestADroppedFrameDoesNotThrowTheCharacter(t *testing.T) {
	ts, s := newGame(t)
	pad := ts.ConnectPad(gluttertest.PadSpec{Name: "Pad"})
	pad.SetAxis(gamepad.LeftY, -1)
	at := s.world.pos
	ts.Pump(2 * time.Second) // one frame that took two seconds
	if d := s.world.pos.Z - at.Z; d > runSpeed*maxStep*1.2 {
		t.Errorf("the character moved %v in one long frame", d)
	}
}

// TestASteadyFrameOfTheGameAllocatesNothing is the allocation target of SPEC §28.6 for the program's own part of a frame: the input of the
// keyboard and a pad, the step of the world, the animation and the sync of the stage. (What the framework and the renderer allocate in a
// frame is measured in their packages.)
func TestASteadyFrameOfTheGameAllocatesNothing(t *testing.T) {
	if raceEnabled {
		t.Skip("the race detector changes what a call allocates")
	}
	ts, s := newGame(t)
	pad := ts.ConnectPad(gluttertest.PadSpec{Name: "Pad", Rumble: true})
	pad.SetAxis(gamepad.LeftX, 0.6)
	pad.SetAxis(gamepad.RightX, 0.4)
	ts.SendKeyDownEvent(services.LogicalKeyboardKeyKeyW)
	elapsed := time.Hour
	for _, name := range []string{"running", "standing"} {
		if name == "standing" {
			pad.SetAxis(gamepad.LeftX, 0)
			pad.SetAxis(gamepad.RightX, 0)
			ts.SendKeyUpEvent(services.LogicalKeyboardKeyKeyW)
			s.world.pos = vecmath.Vec3{X: 4, Z: 8}
		}
		for i := 0; i < 90; i++ { // lets the character settle and the scratch grow
			elapsed += frame
			s.tick(elapsed)
		}
		if got := testing.AllocsPerRun(200, func() {
			elapsed += frame
			s.tick(elapsed)
		}); got != 0 {
			t.Errorf("a frame with the character %s allocates %v times", name, got)
		}
	}
}

// BenchmarkTick is the CPU time of the program's own part of a frame (the input, the step of the world, the animation and the sync of
// the stage), with a pad pushed forward.
func BenchmarkTick(b *testing.B) {
	ts, s := newGame(b)
	pad := ts.ConnectPad(gluttertest.PadSpec{Name: "Pad"})
	pad.SetAxis(gamepad.LeftY, -0.3)
	pad.SetAxis(gamepad.RightX, 0.1)
	elapsed := time.Hour
	for i := 0; i < 60; i++ {
		elapsed += frame
		s.tick(elapsed)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		elapsed += frame
		s.tick(elapsed)
	}
}

// TestAPadButtonIsInTheNextFrame is the first target of SPEC §28.6: a button that is pressed is in the listener's hands in the frame
// after it, here the jump that the game makes of it.
func TestAPadButtonIsInTheNextFrame(t *testing.T) {
	ts, s := newGame(t)
	pad := ts.ConnectPad(gluttertest.PadSpec{Name: "Pad"})
	pumpFrames(ts, 5)
	heard := 0
	pad.Pad().AddListener(func(e gamepad.Event) {
		if e.Kind == gamepad.ButtonDown && e.Button == gamepad.South {
			heard++
		}
	})
	if !s.world.onGround {
		t.Fatal("the character is not on the ground")
	}
	pad.Press(gamepad.South)
	if heard != 1 {
		t.Errorf("the listener heard %d presses before the next frame", heard)
	}
	ts.Pump(frame)
	if s.world.onGround || s.world.vel.Y <= 0 {
		t.Errorf("in the next frame the character is at %v with the velocity %v", s.world.pos, s.world.vel)
	}
}
