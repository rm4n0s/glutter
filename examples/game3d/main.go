// Command game3d is a small 3D game that uses the games work of Glutter 1.10 and 1.11 together (SPEC §27, §28): Coin Run. A
// character (a skinned glTF model, made with its skeleton and four animations by internal/genassets) runs and jumps through a
// small level (a glTF file too) in a viewport3d.View with PBR materials and a shadow, collects its twelve coins and keeps out of the
// lava. A pad (package gamepad: the left stick moves, the right stick looks, A jumps, Start pauses, and the D-pad and A drive the
// pause menu through the key events that pads make), the keyboard (WASD or the arrows, Space, Esc) and a locked mouse (a click in
// the view locks it; package widgets' PointerLock) all play it, and on a phone the touch does: drag on the left half to move, on
// the right half to look, and the round button jumps. The music and the sound effects are package audio streams, a hit makes the
// pad rumble, and the pause menu is Material. The view draws at every frame (Continuous).
//
// Run it with: go run ./examples/game3d
//
// It needs a GPU renderer (Vulkan, Direct3D 12 or WebGPU); with another one the view shows its fallback, a line of text.
//
// On Windows: go run ./cmd/glutter-windows run ./examples/game3d. On the web: go run ./cmd/glutter-web serve ./examples/game3d. On
// Android: go run ./cmd/glutter-android run ./examples/game3d.
//
// Environment: GLUTTER_BACKEND=x11|wayland (Linux) or win32 (Windows), GLUTTER_RENDERER=vulkan|d3d12|webgpu,
// GLUTTER_SCREENSHOT=<file.png>, GLUTTER_SCREENSHOT_AFTER=<duration>. In debug builds GLUTTER_DEV_GAME3D=proof puts the character on the
// second stepping stone with four coins picked up and the camera turned towards the level (the state of the proof pictures), and
// =paused shows the pause menu.
package main

import (
	"context"
	"log"
	"log/slog"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/rm4n0s/glutter/audio"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gamepad"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gpu"
	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
)

func main() {
	err := m.RunApp(NewGameApp(), m.RunAppOpts{
		AppID:  "dev.glutter.game3d",
		Window: m.WindowConfig{Title: "Coin Run", Size: m.Size{Width: 1280, Height: 720}},
	})
	if err != nil {
		log.Fatal(err)
	}
}

// GameApp is the Material app.
type GameApp struct{ m.StatelessWidget }

// NewGameApp creates the app.
func NewGameApp() *GameApp { return &GameApp{} }

// Build makes the app with the page of the game.
func (a *GameApp) Build(bc m.BuildContext) m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "Coin Run",
		Theme:                      m.NewThemeData(m.ThemeDataOpts{ColorSchemeSeed: m.Some(m.Colors.Amber.Primary), Brightness: m.BrightnessDark}),
		DebugShowCheckedModeBanner: m.Some(false),
		Home:                       NewGamePage(),
	})
}

// GamePage is the page: the view of the game, its heads-up display and the menu.
type GamePage struct{ m.StatefulWidget }

// NewGamePage creates the page.
func NewGamePage(opts ...m.WidgetOpts) *GamePage {
	return &GamePage{StatefulWidget: m.NewStatefulWidgetBase(opts...)}
}

// CreateState creates the state.
func (p *GamePage) CreateState() m.State { return &gameState{} }

// The sensitivity of the camera: radians for a pixel of the mouse or of a touch, and radians a second at the full push of a stick.
const (
	mouseLook = 0.0025
	touchLook = 0.006
	stickLook = 2.6
	keyLook   = 2.0
	deadZone  = 0.18
	// touchReach is how far a touch is dragged to run at full speed, in logical pixels.
	touchReach = 70.0
)

// A frame is never longer than this for the game: a window that was dragged or a long pause does not throw the character.
const maxStep = 1.0 / 20

type gameState struct {
	m.StateOf[*GamePage]
	m.SingleTickerProvider
	ticker *scheduler.Ticker
	last   time.Duration

	assets *assets
	world  *world
	stage  *stage
	snd    *sounds

	// binding, pads and keyboard are what a frame reads; padList is the slice that the frame's pads are appended to, so that a frame
	// allocates nothing.
	binding  *m.WidgetsBinding
	pads     *gamepad.Gamepads
	keyboard *services.HardwareKeyboard
	padList  []*gamepad.Pad

	// cancelDevice stops the calls of gpu.OnDevice; the stage is made on each device the view gets.
	cancelDevice func()
	loadErr      string

	paused  bool
	musicOn bool
	// jumpHeld is whether a jump button was down in the last frame, so that holding it is not another press.
	jumpHeld bool
	// mouseDX and mouseDY are the motion of a locked mouse since the last frame.
	mouseDX, mouseDY float64
	lockHandle       foundation.ListenerHandle
	// The touches: the one that moves and the one that looks, with where the first began and the move it gives.
	moveTouch, lookTouch int
	moveOrigin           ui.Offset
	touchMove            vecmath.Vec3
	touchJump            bool

	// What the heads-up display shows, to rebuild it when it changes and not at every frame.
	shownCoins, shownHealth int
	shownState              state
}

// InitState reads the assets, makes the world and the stage, opens the audio and starts the frames.
func (s *gameState) InitState() {
	ctx := context.Background()
	bc := s.Context()
	binding := m.WidgetsBindingOf(bc)
	s.binding, s.pads, s.keyboard = binding, gamepad.ForBinding(binding), binding.Keyboard()
	s.musicOn, s.moveTouch, s.lookTouch = true, -1, -1
	a, err := loadAssets(ctx)
	if err != nil {
		s.loadErr = "The game could not be loaded: " + err.Error()
		return
	}
	s.assets = a
	s.world = newWorld(a.level)
	s.stage = newStage()
	s.shownCoins, s.shownHealth, s.shownState = -1, -1, statePlaying
	if snd, err := newSounds(ctx, audio.ForBinding(binding), a); err != nil {
		slog.Info("the game plays without sound", "err", err)
	} else {
		s.snd = snd
	}
	// The view has its device a moment after its first frame (and a new one after a device is lost): the stage is made on each.
	s.cancelDevice = gpu.OnDevice(binding, func(dev *gpu.Device) {
		if err := s.stage.load(ctx, dev, a); err != nil {
			s.SetState(func() { s.loadErr = "The scene could not be made: " + err.Error() })
			return
		}
		s.stage.sync(s.world, 0.016)
	})
	s.lockHandle = m.PointerLockOf(bc).AddListener(s.lockEnded)
	s.ticker = s.CreateTicker(s.tick)
	s.ticker.Start()
	if foundation.DebugMode {
		s.devState(os.Getenv("GLUTTER_DEV_GAME3D"))
	}
}

// devState puts the game in the state of the proof pictures (debug builds).
func (s *gameState) devState(what string) {
	switch what {
	case "proof", "paused":
		s.world.pos = vecmath.Vec3{X: -7, Y: 1.4, Z: -7}
		s.world.yaw = math.Pi * 0.15
		for i := 0; i < 4; i++ {
			s.world.coins[i].taken = true
			s.world.picked++
		}
		s.stage.yaw, s.stage.pitch = math.Pi*1.15, 0.36
		s.paused = what == "paused"
	}
}

// Dispose stops the frames, the sound and the device's calls, and frees the stage.
func (s *gameState) Dispose() {
	if s.ticker != nil {
		s.ticker.Dispose()
	}
	if s.cancelDevice != nil {
		s.cancelDevice()
	}
	m.PointerLockOf(s.Context()).RemoveListener(s.lockHandle)
	if s.stage != nil {
		s.stage.dispose()
	}
	if err := s.snd.close(); err != nil {
		slog.Info("the sound did not close well", "err", err)
	}
}

// lockEnded pauses the game when the lock of the mouse ended by itself (the window lost the focus, or the platform took the pointer
// back); the lock's listeners are not told of an Unlock, which the game asks for when it pauses.
func (s *gameState) lockEnded() {
	if !s.paused {
		s.SetState(func() { s.setPaused(true) })
	}
}

// setPaused pauses or resumes the game. A pause gives the mouse back. Call it inside SetState.
func (s *gameState) setPaused(p bool) {
	if s.paused == p {
		return
	}
	s.paused = p
	s.snd.setPaused(p)
	if p {
		m.PointerLockOf(s.Context()).Unlock()
	}
}

// tick is the frame of the game: it reads the input, plays the world and shows it.
func (s *gameState) tick(elapsed time.Duration) {
	dt := (elapsed - s.last).Seconds()
	s.last = elapsed
	if s.world == nil || dt <= 0 {
		return
	}
	dt = math.Min(dt, maxStep)
	in := s.readInput(dt)
	if s.paused {
		return
	}
	ev := s.world.step(dt, in)
	if ev.jumped {
		s.snd.play("jump")
	}
	if ev.coin {
		s.snd.play("coin")
	}
	if ev.hit {
		s.snd.play("hit")
		s.rumble()
	}
	s.stage.sync(s.world, dt)
	if s.world.picked != s.shownCoins || s.world.health != s.shownHealth || s.world.state != s.shownState {
		s.SetState(func() {
			s.shownCoins, s.shownHealth, s.shownState = s.world.picked, s.world.health, s.world.state
		})
	}
}

// rumble shakes every pad that can, and the phone.
func (s *gameState) rumble() {
	s.padList = s.pads.AppendPads(s.padList[:0])
	for _, p := range s.padList {
		if !p.CanRumble() {
			continue
		}
		if err := p.Rumble(0.9, 0.7, 350*time.Millisecond); err != nil {
			slog.Debug("a pad did not rumble", "err", err)
		}
	}
	s.binding.HapticFeedback().HeavyImpact()
}

// readInput gathers what the player asks for in this frame from the keyboard, the pads, the mouse and the touches, and turns the
// camera by what looks.
func (s *gameState) readInput(dt float64) input {
	down := s.keyboard.IsLogicalKeyPressed
	axis := func(neg, pos services.LogicalKeyboardKey) float64 {
		v := 0.0
		if down(neg) {
			v--
		}
		if down(pos) {
			v++
		}
		return v
	}
	right := axis(services.LogicalKeyboardKeyKeyA, services.LogicalKeyboardKeyKeyD)
	forward := axis(services.LogicalKeyboardKeyKeyS, services.LogicalKeyboardKeyKeyW)
	lookX := axis(services.LogicalKeyboardKeyArrowLeft, services.LogicalKeyboardKeyArrowRight) + axis(services.LogicalKeyboardKeyKeyQ, services.LogicalKeyboardKeyKeyE)
	lookY := axis(services.LogicalKeyboardKeyArrowUp, services.LogicalKeyboardKeyArrowDown)
	lookX, lookY = lookX*keyLook, lookY*keyLook
	jump := down(services.LogicalKeyboardKeySpace)
	s.padList = s.pads.AppendPads(s.padList[:0])
	for _, p := range s.padList {
		st := p.State()
		lx, ly := gamepad.ApplyStickDeadZone(st.Axis(gamepad.LeftX), st.Axis(gamepad.LeftY), deadZone)
		rx, ry := gamepad.ApplyStickDeadZone(st.Axis(gamepad.RightX), st.Axis(gamepad.RightY), deadZone)
		right, forward = right+lx, forward-ly
		lookX, lookY = lookX+rx*stickLook, lookY+ry*stickLook
		jump = jump || st.Down(gamepad.South)
	}
	jump = jump || s.touchJump
	press := jump && !s.jumpHeld
	s.jumpHeld = jump
	// Whatever the move came from, it is at most one.
	if l := math.Hypot(right, forward); l > 1 {
		right, forward = right/l, forward/l
	}
	if s.paused {
		s.mouseDX, s.mouseDY = 0, 0
		return input{}
	}
	s.stage.look(lookX*dt+s.mouseDX*mouseLook, lookY*dt+s.mouseDY*mouseLook)
	s.mouseDX, s.mouseDY = 0, 0
	move := s.stage.move(right, forward)
	if s.touchMove != (vecmath.Vec3{}) {
		move = s.stage.move(s.touchMove.X, s.touchMove.Z)
	}
	return input{move: move, jump: press}
}

// restart starts the game again.
func (s *gameState) restart() {
	s.SetState(func() {
		s.world.reset()
		s.stage.eye = vecmath.Vec3{}
		s.stage.yaw, s.stage.pitch = math.Pi, 0.42
		s.setPaused(false)
		s.shownCoins = -1
	})
	s.stage.sync(s.world, 0.016)
}

// key handles the keys of the game: Esc, P and a pad's Start pause it.
func (s *gameState) key(_ *m.FocusNode, ev services.KeyEvent) m.KeyEventResult {
	if _, ok := ev.(*services.KeyDownEvent); !ok {
		return m.KeyEventResultIgnored
	}
	switch ev.LogicalKey() {
	case services.LogicalKeyboardKeyEscape, services.LogicalKeyboardKeyKeyP, services.LogicalKeyboardKeyGameButtonStart:
		s.SetState(func() { s.setPaused(!s.paused) })
		return m.KeyEventResultHandled
	}
	return m.KeyEventResultIgnored
}

// menuKey handles the keys of the menu: Esc, P, B and Start close it.
func (s *gameState) menuKey(_ *m.FocusNode, ev services.KeyEvent) m.KeyEventResult {
	if _, ok := ev.(*services.KeyDownEvent); !ok {
		return m.KeyEventResultIgnored
	}
	switch ev.LogicalKey() {
	case services.LogicalKeyboardKeyEscape, services.LogicalKeyboardKeyKeyP, services.LogicalKeyboardKeyGameButtonStart, services.LogicalKeyboardKeyGameButtonB:
		if s.world.state == statePlaying {
			s.SetState(func() { s.setPaused(false) })
			return m.KeyEventResultHandled
		}
	}
	return m.KeyEventResultIgnored
}

// pointerDown locks the mouse on a click, and starts a touch that moves (the left half of the view) or looks (the right half).
func (s *gameState) pointerDown(e *gestures.PointerDownEvent, width float64) {
	if s.paused {
		return
	}
	switch e.Kind() {
	case ui.PointerDeviceKindMouse:
		if lock := m.PointerLockOf(s.Context()); !lock.Locked() {
			if err := lock.Lock(m.ViewOf(s.Context())); err != nil {
				slog.Info("the pointer could not be locked", "err", err)
			}
		}
	case ui.PointerDeviceKindTouch:
		switch {
		case e.Position().Dx < width/2 && s.moveTouch < 0:
			s.moveTouch, s.moveOrigin = e.Pointer(), e.Position()
		case e.Position().Dx >= width/2 && s.lookTouch < 0:
			s.lookTouch = e.Pointer()
		}
	}
}

func (s *gameState) pointerMove(e *gestures.PointerMoveEvent) {
	switch {
	case e.Kind() == ui.PointerDeviceKindMouse:
		s.lookMouse(e.Delta())
	case e.Pointer() == s.moveTouch:
		d := e.Position().Sub(s.moveOrigin)
		v := vecmath.Vec3{X: d.Dx / touchReach, Z: -d.Dy / touchReach}
		if l := v.Length(); l > 1 {
			v = v.Scale(1 / l)
		}
		s.touchMove = v
	case e.Pointer() == s.lookTouch:
		s.mouseDX, s.mouseDY = s.mouseDX+e.Delta().Dx*touchLook/mouseLook, s.mouseDY+e.Delta().Dy*touchLook/mouseLook
	}
}

func (s *gameState) pointerEnd(pointer int) {
	switch pointer {
	case s.moveTouch:
		s.moveTouch, s.touchMove = -1, vecmath.Vec3{}
	case s.lookTouch:
		s.lookTouch = -1
	}
}

// lookMouse adds the motion of a locked mouse to the turn of the next frame.
func (s *gameState) lookMouse(d ui.Offset) {
	if m.PointerLockOf(s.Context()).Locked() {
		s.mouseDX, s.mouseDY = s.mouseDX+d.Dx, s.mouseDY+d.Dy
	}
}

// Build makes the view with the heads-up display over it, and the menu when the game is paused or over.
func (s *gameState) Build(bc m.BuildContext) m.Widget {
	if s.loadErr != "" || s.world == nil {
		return m.NewScaffold(m.ScaffoldOpts{Body: m.NewCenter(m.NewPadding(m.NewEdgeInsetsAll(24), m.NewText(s.loadErr)))})
	}
	view := viewport3d.NewView(s.stage.scene, s.stage.camera, viewport3d.ViewOpts{
		NoNavigation:   true,
		Continuous:     true,
		SemanticsLabel: "Coin Run: a 3D level with a character and coins",
		Fallback: m.NewCenter(m.NewPadding(m.NewEdgeInsetsAll(24), m.NewText(
			"This app needs a GPU renderer: Vulkan, Direct3D 12 or WebGPU.", m.TextOpts{TextAlign: m.TextAlignCenter}))),
	})
	game := m.NewLayoutBuilder(func(bc m.BuildContext, c m.BoxConstraints) m.Widget {
		return m.NewListener(view, m.ListenerOpts{
			OnPointerDown:   func(e *gestures.PointerDownEvent) { s.pointerDown(e, c.MaxWidth) },
			OnPointerMove:   s.pointerMove,
			OnPointerHover:  func(e *gestures.PointerHoverEvent) { s.lookMouse(e.Delta()) },
			OnPointerUp:     func(e *gestures.PointerUpEvent) { s.pointerEnd(e.Pointer()) },
			OnPointerCancel: func(e *gestures.PointerCancelEvent) { s.pointerEnd(e.Pointer()) },
		})
	})
	children := []m.Widget{
		m.NewFocus(game, m.FocusOpts{Autofocus: !s.paused && s.world.state == statePlaying, OnKeyEvent: s.key}),
		m.NewPositioned(s.hud(bc), m.PositionedOpts{Left: m.Some(16.0), Top: m.Some(16.0)}),
	}
	if foundation.DefaultTargetPlatform() == foundation.TargetPlatformAndroid || foundation.DefaultTargetPlatform() == foundation.TargetPlatformIOS {
		// A phone has the touch controls: a round button for the jump, and no line of keys.
		children = append(children, m.NewPositioned(s.jumpButton(), m.PositionedOpts{Right: m.Some(28.0), Bottom: m.Some(28.0)}))
	} else {
		children = append(children, m.NewPositioned(s.hint(bc), m.PositionedOpts{Left: m.Some(0.0), Right: m.Some(0.0), Bottom: m.Some(12.0)}))
	}
	if menu := s.menu(bc); menu != nil {
		children = append(children, m.NewPositionedFill(menu))
	}
	return m.NewScaffold(m.ScaffoldOpts{Body: m.NewStack(children, m.StackOpts{Fit: m.StackFitExpand})})
}

// jumpButton is the round button of the touch screen.
func (s *gameState) jumpButton() m.Widget {
	return m.NewListener(m.NewFloatingActionButton(func() {}, m.NewIcon(m.Icons.KeyboardArrowUp()), m.FloatingActionButtonOpts{Tooltip: "Jump"}), m.ListenerOpts{
		OnPointerDown:   func(*gestures.PointerDownEvent) { s.touchJump = true },
		OnPointerUp:     func(*gestures.PointerUpEvent) { s.touchJump = false },
		OnPointerCancel: func(*gestures.PointerCancelEvent) { s.touchJump = false },
	})
}

// hud shows the coins and the health.
func (s *gameState) hud(bc m.BuildContext) m.Widget {
	text := m.ThemeOf(bc).TextTheme
	hearts := make([]m.Widget, 0, maxHealth)
	for i := 0; i < maxHealth; i++ {
		icon := m.Icons.FavoriteBorder()
		color := m.Colors.White
		if i < s.world.health {
			icon, color = m.Icons.Favorite(), m.Colors.Red.Shade400()
		}
		hearts = append(hearts, m.NewIcon(icon, m.IconOpts{Color: m.Some(color)}))
	}
	return m.NewCard(m.NewPadding(m.NewEdgeInsetsSymmetric(14, 8), m.NewRow([]m.Widget{
		m.NewIcon(m.Icons.MonetizationOn(), m.IconOpts{Color: m.Some(m.Colors.Amber.Shade400())}),
		m.NewSizedBox(nil, m.SizedBoxOpts{Width: m.Some(6.0)}),
		m.NewText(fmtCoins(s.world.picked, len(s.world.coins)), m.TextOpts{Style: text.TitleMedium}),
		m.NewSizedBox(nil, m.SizedBoxOpts{Width: m.Some(18.0)}),
		m.NewRow(hearts, m.RowOpts{MainAxisSize: m.MainAxisSizeMin}),
	}, m.RowOpts{MainAxisSize: m.MainAxisSizeMin})))
}

// hint tells what the controls are, at the bottom of the view.
func (s *gameState) hint(bc m.BuildContext) m.Widget {
	return m.NewIgnorePointer(m.NewCenter(m.NewCard(m.NewPadding(m.NewEdgeInsetsSymmetric(14, 6), m.NewText(
		"WASD or left stick: move   Space or A: jump   mouse, arrows or right stick: look   Esc or Start: pause",
		m.TextOpts{Style: m.ThemeOf(bc).TextTheme.LabelLarge})))))
}

// menu is the pause menu, or the end of the game, or nil while the game is played.
func (s *gameState) menu(bc m.BuildContext) m.Widget {
	var title, subtitle string
	switch {
	case s.world.state == stateWon:
		title, subtitle = "You win!", fmtCoins(s.world.picked, len(s.world.coins))+" collected"
	case s.world.state == stateLost:
		title, subtitle = "Game over", fmtCoins(s.world.picked, len(s.world.coins))+" collected"
	case s.paused:
		title, subtitle = "Paused", "Use the D-pad and A, or the mouse."
	default:
		return nil
	}
	text := m.ThemeOf(bc).TextTheme
	items := []m.Widget{
		m.NewText(title, m.TextOpts{Style: text.HeadlineMedium}),
		m.NewText(subtitle),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(16.0)}),
	}
	if s.world.state == statePlaying {
		items = append(items,
			m.NewFilledButton(func() { s.SetState(func() { s.setPaused(false) }) }, m.NewText("Resume"), m.FilledButtonOpts{Autofocus: true}),
			m.NewSwitchListTile(s.musicOn, func(on bool) { s.SetState(func() { s.musicOn = on; s.snd.setMusic(on) }) }, m.SwitchListTileOpts{Title: m.NewText("Music")}),
			m.NewOutlinedButton(s.restart, m.NewText("Restart")),
		)
	} else {
		items = append(items, m.NewFilledButton(s.restart, m.NewText("Play again"), m.FilledButtonOpts{Autofocus: true}))
	}
	card := m.NewCard(m.NewPadding(m.NewEdgeInsetsAll(24), m.NewSizedBox(
		m.NewColumn(items, m.ColumnOpts{MainAxisSize: m.MainAxisSizeMin, CrossAxisAlignment: m.CrossAxisAlignmentStretch}),
		m.SizedBoxOpts{Width: m.Some(300.0)})))
	return m.NewFocusScope(m.NewColoredBox(ui.NewColorFromARGB(120, 0, 0, 0), m.NewCenter(card)), m.FocusScopeOpts{OnKeyEvent: s.menuKey})
}

// fmtCoins is the text of the coins that were picked up, "3 / 12".
func fmtCoins(picked, total int) string { return strconv.Itoa(picked) + " / " + strconv.Itoa(total) }
