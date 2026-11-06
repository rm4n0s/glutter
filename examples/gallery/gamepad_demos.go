package main

import (
	"fmt"
	"strings"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gamepad"
	m "github.com/rm4n0s/glutter/material"
)

// devGamepads is whether GLUTTER_DEV_GAMEPADS is "1" in a debug build: the gallery then shows a panel in the corner that lists
// the pads that are connected, with their names, kinds, GUIDs, the buttons that are down and the values of the axes, and the
// last events (SPEC §28.2). It is the aid of the manual check of a pad: connect one, press its buttons and see them here, and
// move the focus through the gallery with the D-pad (the pads make key events).
func devGamepads(getenv func(string) string) bool {
	return foundation.DebugMode && getenv("GLUTTER_DEV_GAMEPADS") == "1"
}

// withGamepadPanel puts the panel over the corner of the page.
func withGamepadPanel(child m.Widget) m.Widget {
	return m.NewStack([]m.Widget{
		child,
		m.NewAlign(NewGamepadPanel(), m.AlignOpts{Alignment: m.AlignmentBottomRight}),
	}, m.StackOpts{Fit: m.StackFitExpand})
}

// GamepadPanel lists the pads of the app and what they report.
type GamepadPanel struct{ m.StatefulWidget }

// NewGamepadPanel creates the panel.
func NewGamepadPanel() *GamepadPanel { return &GamepadPanel{} }

// CreateState creates the state of the panel.
func (w *GamepadPanel) CreateState() m.State { return &gamepadPanelState{} }

type gamepadPanelState struct {
	m.StateOf[*GamepadPanel]
	pads    *gamepad.Gamepads
	removes map[*gamepad.Pad]func()
	handle  foundation.ListenerHandle
	log     []string
}

func (s *gamepadPanelState) InitState() {
	s.pads = gamepad.Of(s.Context())
	s.removes = map[*gamepad.Pad]func(){}
	s.handle = s.pads.AddListener(s.padsChanged)
	s.padsChanged()
}

func (s *gamepadPanelState) Dispose() {
	s.pads.RemoveListener(s.handle)
	for _, remove := range s.removes {
		remove()
	}
}

// padsChanged follows the pads that come and go.
func (s *gamepadPanelState) padsChanged() {
	live := map[*gamepad.Pad]bool{}
	for _, p := range s.pads.Pads() {
		live[p] = true
		if _, ok := s.removes[p]; !ok {
			s.removes[p] = p.AddListener(s.event)
			s.note(fmt.Sprintf("%s connected (%s)", p.Name(), p.Kind()))
		}
	}
	for p, remove := range s.removes {
		if !live[p] {
			remove()
			delete(s.removes, p)
			s.note(p.Name() + " removed")
		}
	}
	if s.Mounted() {
		s.SetState(func() {})
	}
}

func (s *gamepadPanelState) event(e gamepad.Event) {
	switch e.Kind {
	case gamepad.ButtonDown:
		s.note(fmt.Sprintf("%s: %s down", e.Pad.Name(), e.Button))
	case gamepad.ButtonUp:
		s.note(fmt.Sprintf("%s: %s up", e.Pad.Name(), e.Button))
	default:
		// The axes are shown live in the list, not in the log.
	}
	if s.Mounted() {
		s.SetState(func() {})
	}
}

func (s *gamepadPanelState) note(line string) {
	s.log = append(s.log, line)
	if len(s.log) > 6 {
		s.log = s.log[len(s.log)-6:]
	}
}

func (s *gamepadPanelState) Build(bc m.BuildContext) m.Widget {
	colors := m.ThemeOf(bc).ColorScheme
	small := m.ThemeOf(bc).TextTheme.BodySmall
	lines := []m.Widget{m.NewText("Gamepads (debug)", m.TextOpts{Style: m.ThemeOf(bc).TextTheme.TitleSmall})}
	add := func(text string) { lines = append(lines, m.NewText(text, m.TextOpts{Style: small})) }
	pads := s.pads.Pads()
	if len(pads) == 0 {
		add("No pad. Connect one and press a button.")
	}
	for _, p := range pads {
		st := p.State()
		var down []string
		for b := gamepad.Button(0); int(b) < gamepad.ButtonCount; b++ {
			if st.Down(b) {
				down = append(down, b.String())
			}
		}
		add(fmt.Sprintf("%s [%s, %v faces] %s", p.Name(), p.Kind(), p.FaceStyle(), p.Path()))
		add("GUID " + p.GUID())
		add("down: " + strings.Join(down, " "))
		add(fmt.Sprintf("L %+.2f %+.2f  R %+.2f %+.2f  LT %.2f RT %.2f",
			st.Axis(gamepad.LeftX), st.Axis(gamepad.LeftY), st.Axis(gamepad.RightX), st.Axis(gamepad.RightY),
			st.Axis(gamepad.LeftTrigger), st.Axis(gamepad.RightTrigger)))
	}
	add("")
	for _, line := range s.log {
		add(line)
	}
	return m.NewPadding(m.NewEdgeInsetsAll(12), m.NewMaterial(m.NewPadding(m.NewEdgeInsetsAll(12),
		m.NewColumn(lines, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart, MainAxisSize: m.MainAxisSizeMin}),
	), m.MaterialOpts{Color: m.Some(colors.SurfaceContainerHighest()), Elevation: 6, BorderRadius: m.NewBorderRadiusCircular(12)}))
}
