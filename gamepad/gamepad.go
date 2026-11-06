// Package gamepad reads game controllers: which pads are connected, the buttons and axes of each in one standard layout
// (SPEC §28.2), their kind (Xbox, PlayStation, Switch, ...), and rumble and lights where the platform has them.
//
// A pad's buttons are the same on every pad, whatever it reports: the mapping of the pad's own numbers to the standard layout
// comes from SDL 3's table of mappings and from the Linux Gamepad Specification, and an app can add mappings of its own in the
// community's format with Gamepads.AddMappings. No dead zone is applied to the axes: that is the app's choice, and
// ApplyDeadZone helps with it.
//
// By default the pads also move the focus of the app and press its buttons, as key events: see Gamepads.SetKeyEvents.
//
// Everything here belongs to the UI thread. The package is part of Glutter's port of SDL 3's gamepad API (SPEC §28.1); the
// mapping format and the table are SDL's (zlib license, third_party/sdl/LICENSE).
//
// Flutter: no counterpart
package gamepad

import (
	"math"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/gamepaddb"
	"github.com/rm4n0s/glutter/internal/pads"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// PadID names a pad while it is connected. The platform makes it, and it is not used again during a run.
//
// Flutter: no counterpart
type PadID int64

// Button is a button of the standard layout.
//
// Flutter: no counterpart
type Button uint8

// The buttons of the standard layout. South, East, West and North are the positions of the four face buttons: A, B, X and Y on
// an Xbox pad, cross, circle, square and triangle on a PlayStation one (see Pad.FaceStyle).
const (
	// South is the bottom face button.
	//
	// Flutter: no counterpart
	South = Button(gamepaddb.South)
	// East is the right face button.
	//
	// Flutter: no counterpart
	East = Button(gamepaddb.East)
	// West is the left face button.
	//
	// Flutter: no counterpart
	West = Button(gamepaddb.West)
	// North is the top face button.
	//
	// Flutter: no counterpart
	North = Button(gamepaddb.North)
	// Back is the button on the left of the middle (View, Select, Share).
	//
	// Flutter: no counterpart
	Back = Button(gamepaddb.Back)
	// Guide is the button in the middle (Xbox, PS, Home). Not every platform gives it to apps.
	//
	// Flutter: no counterpart
	Guide = Button(gamepaddb.Guide)
	// Start is the button on the right of the middle (Menu, Options).
	//
	// Flutter: no counterpart
	Start = Button(gamepaddb.Start)
	// LeftStick is the press of the left stick.
	//
	// Flutter: no counterpart
	LeftStick = Button(gamepaddb.LeftStick)
	// RightStick is the press of the right stick.
	//
	// Flutter: no counterpart
	RightStick = Button(gamepaddb.RightStick)
	// LeftShoulder is the left bumper.
	//
	// Flutter: no counterpart
	LeftShoulder = Button(gamepaddb.LeftShoulder)
	// RightShoulder is the right bumper.
	//
	// Flutter: no counterpart
	RightShoulder = Button(gamepaddb.RightShoulder)
	// DPadUp is up on the D-pad.
	//
	// Flutter: no counterpart
	DPadUp = Button(gamepaddb.DPadUp)
	// DPadDown is down on the D-pad.
	//
	// Flutter: no counterpart
	DPadDown = Button(gamepaddb.DPadDown)
	// DPadLeft is left on the D-pad.
	//
	// Flutter: no counterpart
	DPadLeft = Button(gamepaddb.DPadLeft)
	// DPadRight is right on the D-pad.
	//
	// Flutter: no counterpart
	DPadRight = Button(gamepaddb.DPadRight)
	// Misc1 is an extra button: Share on an Xbox Series pad, the microphone button on a DualSense.
	//
	// Flutter: no counterpart
	Misc1 = Button(gamepaddb.Misc1)
	// Paddle1 is the upper right paddle on the back of the pad.
	//
	// Flutter: no counterpart
	Paddle1 = Button(gamepaddb.Paddle1)
	// Paddle2 is the upper left paddle on the back.
	//
	// Flutter: no counterpart
	Paddle2 = Button(gamepaddb.Paddle2)
	// Paddle3 is the lower right paddle on the back.
	//
	// Flutter: no counterpart
	Paddle3 = Button(gamepaddb.Paddle3)
	// Paddle4 is the lower left paddle on the back.
	//
	// Flutter: no counterpart
	Paddle4 = Button(gamepaddb.Paddle4)
	// Touchpad is the press of the touchpad of a PlayStation pad.
	//
	// Flutter: no counterpart
	Touchpad = Button(gamepaddb.Touchpad)
	// Misc2 is another extra button.
	//
	// Flutter: no counterpart
	Misc2 = Button(gamepaddb.Misc2)
	// Misc3 is another extra button.
	//
	// Flutter: no counterpart
	Misc3 = Button(gamepaddb.Misc3)
	// Misc4 is another extra button.
	//
	// Flutter: no counterpart
	Misc4 = Button(gamepaddb.Misc4)
	// Misc5 is another extra button.
	//
	// Flutter: no counterpart
	Misc5 = Button(gamepaddb.Misc5)
	// Misc6 is another extra button.
	//
	// Flutter: no counterpart
	Misc6 = Button(gamepaddb.Misc6)
)

// ButtonCount is the number of buttons; the buttons are 0 to ButtonCount-1.
//
// Flutter: no counterpart
const ButtonCount = int(gamepaddb.ButtonCount)

// String is the name of the button as a mapping spells it ("leftshoulder").
//
// Flutter: no counterpart
func (b Button) String() string {
	if n := gamepaddb.Button(b).Name(); n != "" {
		return n
	}
	return "unknown"
}

// Axis is an axis of the standard layout.
//
// Flutter: no counterpart
type Axis uint8

// The axes of the standard layout.
const (
	// LeftX is the left stick from left (−1) to right (1).
	//
	// Flutter: no counterpart
	LeftX = Axis(gamepaddb.LeftX)
	// LeftY is the left stick from up (−1) to down (1).
	//
	// Flutter: no counterpart
	LeftY = Axis(gamepaddb.LeftY)
	// RightX is the right stick from left (−1) to right (1).
	//
	// Flutter: no counterpart
	RightX = Axis(gamepaddb.RightX)
	// RightY is the right stick from up (−1) to down (1).
	//
	// Flutter: no counterpart
	RightY = Axis(gamepaddb.RightY)
	// LeftTrigger is the left trigger from 0 to 1.
	//
	// Flutter: no counterpart
	LeftTrigger = Axis(gamepaddb.LeftTrigger)
	// RightTrigger is the right trigger from 0 to 1.
	//
	// Flutter: no counterpart
	RightTrigger = Axis(gamepaddb.RightTrigger)
)

// AxisCount is the number of axes; the axes are 0 to AxisCount-1.
//
// Flutter: no counterpart
const AxisCount = int(gamepaddb.AxisCount)

// String is the name of the axis as a mapping spells it ("lefttrigger").
//
// Flutter: no counterpart
func (a Axis) String() string {
	if n := gamepaddb.Axis(a).Name(); n != "" {
		return n
	}
	return "unknown"
}

// PadKind is the model of a pad, as far as its vendor and product numbers tell.
//
// Flutter: no counterpart
type PadKind uint8

// The kinds of pads.
const (
	// KindGeneric is a pad that is not known by model, or has the standard layout.
	//
	// Flutter: no counterpart
	KindGeneric = PadKind(iota)
	// KindXbox360 is an Xbox 360 pad.
	//
	// Flutter: no counterpart
	KindXbox360
	// KindXboxOne is an Xbox One, Xbox Series or Xbox Elite pad.
	//
	// Flutter: no counterpart
	KindXboxOne
	// KindPlayStation3 is a PlayStation 3 pad.
	//
	// Flutter: no counterpart
	KindPlayStation3
	// KindPlayStation4 is a PlayStation 4 pad (DualShock 4).
	//
	// Flutter: no counterpart
	KindPlayStation4
	// KindPlayStation5 is a PlayStation 5 pad (DualSense).
	//
	// Flutter: no counterpart
	KindPlayStation5
	// KindSwitchPro is a Nintendo Switch Pro pad, or a pad that copies it.
	//
	// Flutter: no counterpart
	KindSwitchPro
	// KindJoyConLeft is a left Joy-Con on its own.
	//
	// Flutter: no counterpart
	KindJoyConLeft
	// KindJoyConRight is a right Joy-Con on its own.
	//
	// Flutter: no counterpart
	KindJoyConRight
	// KindJoyConPair is two Joy-Cons used as one pad.
	//
	// Flutter: no counterpart
	KindJoyConPair
	// KindGameCube is a GameCube pad.
	//
	// Flutter: no counterpart
	KindGameCube
	// KindSteam is a Steam pad.
	//
	// Flutter: no counterpart
	KindSteam
)

// String names the kind in words.
//
// Flutter: no counterpart
func (k PadKind) String() string {
	switch k {
	case KindXbox360:
		return "Xbox 360"
	case KindXboxOne:
		return "Xbox One"
	case KindPlayStation3:
		return "PlayStation 3"
	case KindPlayStation4:
		return "PlayStation 4"
	case KindPlayStation5:
		return "PlayStation 5"
	case KindSwitchPro:
		return "Switch Pro"
	case KindJoyConLeft:
		return "Joy-Con (L)"
	case KindJoyConRight:
		return "Joy-Con (R)"
	case KindJoyConPair:
		return "Joy-Con pair"
	case KindGameCube:
		return "GameCube"
	case KindSteam:
		return "Steam"
	}
	return "Generic"
}

func padKind(k gamepaddb.Kind) PadKind {
	switch k {
	case gamepaddb.KindXbox360:
		return KindXbox360
	case gamepaddb.KindXboxOne:
		return KindXboxOne
	case gamepaddb.KindPS3:
		return KindPlayStation3
	case gamepaddb.KindPS4:
		return KindPlayStation4
	case gamepaddb.KindPS5:
		return KindPlayStation5
	case gamepaddb.KindSwitchPro:
		return KindSwitchPro
	case gamepaddb.KindJoyConLeft:
		return KindJoyConLeft
	case gamepaddb.KindJoyConRight:
		return KindJoyConRight
	case gamepaddb.KindJoyConPair:
		return KindJoyConPair
	case gamepaddb.KindGameCube:
		return KindGameCube
	case gamepaddb.KindSteam:
		return KindSteam
	}
	return KindGeneric
}

// FaceStyle is how the four face buttons of a pad are labelled: South, East, West and North are A, B, X and Y on an Xbox pad
// and cross, circle, square and triangle on a PlayStation one.
//
// Flutter: no counterpart
type FaceStyle uint8

// The styles of face buttons.
const (
	// FaceABXY is A at the bottom, B on the right, X on the left and Y at the top (Xbox).
	//
	// Flutter: no counterpart
	FaceABXY = FaceStyle(iota)
	// FaceAXBY is A at the bottom, X on the right, B on the left and Y at the top (GameCube).
	//
	// Flutter: no counterpart
	FaceAXBY
	// FaceBAYX is B at the bottom, A on the right, Y on the left and X at the top (Nintendo).
	//
	// Flutter: no counterpart
	FaceBAYX
	// FaceSony is cross at the bottom, circle on the right, square on the left and triangle at the top (PlayStation).
	//
	// Flutter: no counterpart
	FaceSony
)

func faceStyle(f gamepaddb.FaceStyle) FaceStyle {
	switch f {
	case gamepaddb.FaceAXBY:
		return FaceAXBY
	case gamepaddb.FaceBAYX:
		return FaceBAYX
	case gamepaddb.FaceSony:
		return FaceSony
	}
	return FaceABXY
}

// State is the buttons and axes of a pad at one moment.
//
// Flutter: no counterpart
type State struct{ s pads.State }

// Down says whether the button is down.
//
// Flutter: no counterpart
func (s State) Down(b Button) bool { return s.s.Down(gamepaddb.Button(b)) }

// Axis returns the value of the axis: from −1 to 1 for a stick, from 0 to 1 for a trigger. No dead zone is applied.
//
// Flutter: no counterpart
func (s State) Axis(a Axis) float64 {
	if int(a) >= AxisCount {
		return 0
	}
	return axisValue(s.s.Axes[a])
}

func axisValue(v int16) float64 { return max(-1, float64(v)/gamepaddb.AxisMax) }

// EventKind says what an Event reports.
//
// Flutter: no counterpart
type EventKind uint8

const (
	// ButtonDown is a button that went down.
	//
	// Flutter: no counterpart
	ButtonDown = EventKind(pads.ButtonDown)
	// ButtonUp is a button that went up.
	//
	// Flutter: no counterpart
	ButtonUp = EventKind(pads.ButtonUp)
	// AxisMotion is an axis that has a new value.
	//
	// Flutter: no counterpart
	AxisMotion = EventKind(pads.AxisMotion)
)

// Event is a change of a button or an axis of a pad. Events reach the listeners on the UI thread, in the order they happened,
// with the time the pad reported them.
//
// Flutter: no counterpart
type Event struct {
	// Pad is the pad that made the event.
	Pad *Pad
	// Time is when the pad reported the change, on the system's monotonic clock.
	Time time.Duration
	Kind EventKind
	// Button is the button of a ButtonDown or ButtonUp.
	Button Button
	// Axis and Value are the axis of an AxisMotion and its new value (see State.Axis).
	Axis  Axis
	Value float64
}

// Battery is the charge of a pad.
//
// Flutter: no counterpart
type Battery struct {
	// Known says whether the platform knows the charge; Level is from 0 to 1 and means nothing when Known is false.
	Known bool
	Level float64
}

// Pad is a connected gamepad. A pad has the same value for as long as it is connected.
//
// Flutter: no counterpart
type Pad struct {
	p *pads.Pad
}

// ID is the platform's number of the pad.
//
// Flutter: no counterpart
func (p *Pad) ID() PadID { return PadID(p.p.ID()) }

// Name is the name of the pad.
//
// Flutter: no counterpart
func (p *Pad) Name() string { return p.p.Name() }

// Kind is the model of the pad.
//
// Flutter: no counterpart
func (p *Pad) Kind() PadKind { return padKind(p.p.Kind()) }

// FaceStyle is how the face buttons of the pad are labelled.
//
// Flutter: no counterpart
func (p *Pad) FaceStyle() FaceStyle { return faceStyle(p.p.FaceStyle()) }

// GUID names the model of the pad as a mapping line does: 32 hexadecimal digits.
//
// Flutter: no counterpart
func (p *Pad) GUID() string { return p.p.GUID().String() }

// Path says where the pad is on the system ("/dev/input/event7" on Linux), or "".
//
// Flutter: no counterpart
func (p *Pad) Path() string { return p.p.Path() }

// Connected says whether the pad is still there.
//
// Flutter: no counterpart
func (p *Pad) Connected() bool { return p.p.Connected() }

// HasButton says whether the pad has the button.
//
// Flutter: no counterpart
func (p *Pad) HasButton(b Button) bool { return p.p.HasButton(gamepaddb.Button(b)) }

// HasAxis says whether the pad has the axis.
//
// Flutter: no counterpart
func (p *Pad) HasAxis(a Axis) bool { return p.p.HasAxis(gamepaddb.Axis(a)) }

// State returns the buttons and axes of the pad now, for reading once a frame.
//
// Flutter: no counterpart
func (p *Pad) State() State { return State{p.p.State()} }

// Battery returns the charge of the pad.
//
// Flutter: no counterpart
func (p *Pad) Battery() Battery {
	if v := p.p.Battery(); v >= 0 {
		return Battery{Known: true, Level: float64(v) / 100}
	}
	return Battery{}
}

// AddListener adds a function that is called, on the UI thread, with every change of a button or axis of the pad, and returns
// the function that removes it. A listener may add and remove listeners.
//
// Flutter: no counterpart
func (p *Pad) AddListener(fn func(Event)) (remove func()) {
	return p.p.AddListener(func(e pads.Event) {
		ev := Event{Pad: p, Time: e.Time, Kind: EventKind(e.Kind), Button: Button(e.Button), Axis: Axis(e.Axis)}
		if e.Kind == pads.AxisMotion {
			ev.Value = axisValue(e.Value)
		}
		fn(ev)
	})
}

// CanRumble says whether the pad has motors that Rumble can turn.
//
// Flutter: no counterpart
func (p *Pad) CanRumble() bool { return p.p.CanRumble() }

// CanRumbleTriggers says whether the pad has motors in its triggers.
//
// Flutter: no counterpart
func (p *Pad) CanRumbleTriggers() bool { return p.p.CanRumbleTriggers() }

// HasLED says whether the pad has a light that SetLED can set.
//
// Flutter: no counterpart
func (p *Pad) HasLED() bool { return p.p.HasLED() }

// Rumble turns the motors of the pad for d: low is the strength of the big, slow motor and high that of the small, fast one,
// each from 0 to 1. A rumble that is running is replaced; 0 and 0 stop it. It fails with GamepadRumbleUnavailable
// (TagRumbleUnavailable) when the pad or the platform cannot rumble.
//
// Flutter: no counterpart
func (p *Pad) Rumble(low, high float64, d time.Duration) error {
	return p.p.Rumble(strength(low), strength(high), d)
}

// RumbleTriggers turns the motors of the triggers of the pad (Xbox One and Series pads) for d, each from 0 to 1. It fails with
// GamepadRumbleUnavailable (TagRumbleUnavailable) when the pad or the platform cannot.
//
// Flutter: no counterpart
func (p *Pad) RumbleTriggers(left, right float64, d time.Duration) error {
	return p.p.RumbleTriggers(strength(left), strength(right), d)
}

// SetLED sets the colour of the light of the pad (the light bar of a PlayStation pad); the alpha is ignored. It fails with
// GamepadLEDUnavailable (TagLEDUnavailable) when the pad or the platform has none.
//
// Flutter: no counterpart
func (p *Pad) SetLED(c ui.Color) error {
	return p.p.SetLED(uint8(c.R*255+0.5), uint8(c.G*255+0.5), uint8(c.B*255+0.5))
}

func strength(v float64) uint16 {
	switch {
	case v <= 0 || v != v:
		return 0
	case v >= 1:
		return 0xFFFF
	}
	return uint16(v*0xFFFF + 0.5)
}

// The tags of the errors of this package (SPEC §28.7).
const (
	// TagMappingInvalid is the tag of the error of a mapping line that cannot be read.
	//
	// Flutter: no counterpart
	TagMappingInvalid = gamepaddb.TagMappingInvalid
	// TagRumbleUnavailable is the tag of the error of a rumble that the pad or the platform cannot make.
	//
	// Flutter: no counterpart
	TagRumbleUnavailable = "GamepadRumbleUnavailable"
	// TagLEDUnavailable is the tag of the error of a light that the pad or the platform cannot set.
	//
	// Flutter: no counterpart
	TagLEDUnavailable = "GamepadLEDUnavailable"
)

// Gamepads is the set of the pads of the app. It is a ChangeNotifier: its listeners are called when a pad is connected or
// removed. Get it with Of.
//
// Flutter: no counterpart
type Gamepads struct {
	foundation.ChangeNotifier
	m      *pads.Manager
	byPad  map[*pads.Pad]*Pad
	remove func()
}

type gamepadsKey struct{}

// Of returns the gamepads of the app (the ones of the binding). An app on a platform that has no gamepad service gets a set
// that never has a pad.
//
// Flutter: no counterpart
func Of(bc widgets.BuildContext) *Gamepads { return ForBinding(widgets.WidgetsBindingOf(bc)) }

// ForBinding returns the gamepads of the binding, for code that has no BuildContext.
//
// Flutter: no counterpart
func ForBinding(b *widgets.WidgetsBinding) *Gamepads {
	return b.Extension(gamepadsKey{}, func() any {
		g := &Gamepads{byPad: map[*pads.Pad]*Pad{}}
		g.SetErrorReporter(b.ErrorReporter())
		g.SetUIThread(b.UIThread())
		var host pads.Host
		if b.PlatformDispatcher().HostAs(&host) {
			g.m = host.PadManager()
		}
		if g.m != nil {
			g.remove = g.m.AddListener(g.changed)
		}
		return g
	}).(*Gamepads)
}

// changed forgets the wrappers of the pads that are gone and tells the listeners.
func (g *Gamepads) changed() {
	live := map[*pads.Pad]bool{}
	for _, p := range g.m.Pads() {
		live[p] = true
	}
	for p := range g.byPad {
		if !live[p] {
			delete(g.byPad, p)
		}
	}
	g.NotifyListeners()
}

// Available says whether the platform has a gamepad service at all.
//
// Flutter: no counterpart
func (g *Gamepads) Available() bool { return g.m.Available() }

// Pads returns the connected pads that have a mapping, in the order they were connected. The slice is a copy.
//
// Flutter: no counterpart
func (g *Gamepads) Pads() []*Pad {
	list := g.m.Pads()
	out := make([]*Pad, len(list))
	for i, p := range list {
		w := g.byPad[p]
		if w == nil {
			w = &Pad{p: p}
			g.byPad[p] = w
		}
		out[i] = w
	}
	return out
}

// AddMappings adds mappings in the community's format (the lines of SDL's gamecontrollerdb.txt: "GUID,name,a:b0,b:b1,...,
// platform:Linux"). The lines for another platform are left out, and a mapping for a pad that has one replaces it. A pad
// that had no mapping becomes a gamepad if one of the lines is for it. It returns the number of mappings that are new, and
// fails with GamepadMappingInvalid (TagMappingInvalid) at the first line that cannot be read; the lines before it are added.
//
// Flutter: no counterpart
func (g *Gamepads) AddMappings(text string) (int, error) {
	if g.m == nil {
		return 0, nil
	}
	return g.m.AddMappings(text)
}

// SetKeyEvents turns on or off that the pads make key events: the buttons as the game-button keys (South as GameButtonA, East
// as GameButtonB, West as GameButtonX, North as GameButtonY, the shoulders as GameButtonLeft1 and GameButtonRight1, the
// triggers as GameButtonLeft2 and GameButtonRight2 from half their travel, Start, Back and Guide as GameButtonStart,
// GameButtonSelect and GameButtonMode, the sticks' presses as GameButtonThumbLeft and GameButtonThumbRight) and the D-pad as
// the arrow keys. They are on at first: they give every app focus traversal and activation with a pad (ActivateIntent on
// GameButtonA). Flutter does this on Android only (SPEC §12.10). Only the window that has the focus gets the keys.
//
// Flutter: no counterpart
func (g *Gamepads) SetKeyEvents(on bool) {
	if g.m != nil {
		g.m.SetKeyEvents(on)
	}
}

// KeyEvents says whether the pads make key events.
//
// Flutter: no counterpart
func (g *Gamepads) KeyEvents() bool { return g.m.KeyEvents() }

// ApplyDeadZone turns the value of one axis into one with a dead zone: a value whose size is under dead is 0, and the rest
// is spread over the full range, so that the value is continuous. dead is from 0 up to, not including, 1.
//
// Flutter: no counterpart
func ApplyDeadZone(v, dead float64) float64 {
	if dead <= 0 {
		return v
	}
	if dead >= 1 {
		return 0
	}
	switch a := math.Abs(v); {
	case a <= dead:
		return 0
	case v < 0:
		return -(a - dead) / (1 - dead)
	default:
		return (a - dead) / (1 - dead)
	}
}

// ApplyStickDeadZone does the same for the two axes of a stick together: the dead zone is a circle, so that a stick that is
// pushed in a diagonal is not cut by a square, and the size of the push is spread over the full range with the direction
// kept. The result is not longer than 1.
//
// Flutter: no counterpart
func ApplyStickDeadZone(x, y, dead float64) (float64, float64) {
	l := math.Hypot(x, y)
	if l <= dead || l == 0 {
		return 0, 0
	}
	if dead >= 1 {
		return 0, 0
	}
	scale := min(1, (l-dead)/(1-dead)) / l
	return x * scale, y * scale
}
