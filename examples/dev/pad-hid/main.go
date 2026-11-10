// Command pad-hid is a development example (step 31.3) for the HID drivers of the gamepads: for each pad it shows the name, the
// kind, the battery, the touchpad fingers and the gyroscope and accelerometer, and it has buttons for the light bar, the motors
// and the motors of the triggers. A square turns with the gyroscope, so that the sensors can be seen working. It uses package
// internal/pads for the sensors and the touchpad, which 31.7 and 31.9 make public, so it is a development example.
//
// Run it with a pad connected (PlayStation 4 or 5, Switch Pro, Xbox over Bluetooth); on Linux the user must be allowed to open
// /dev/hidraw* (docs/GUIDE.md has the udev rule), and GLUTTER_GAMEPAD_HID=0 reads the pads without the drivers:
//
//	go run ./examples/dev/pad-hid
package main

import (
	"fmt"
	"log"
	"log/slog"
	"strings"
	"time"

	"github.com/rm4n0s/glutter/internal/pads"
	"github.com/rm4n0s/glutter/internal/platform"
	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/ui"
)

func main() {
	err := m.RunApp(newApp(), m.RunAppOpts{
		AppID:  "dev.glutter.padhid",
		Window: m.WindowConfig{Title: "Pad HID drivers", Size: m.Size{Width: 560, Height: 720}},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func newApp() m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "Pad HID drivers",
		DebugShowCheckedModeBanner: m.Some(false),
		Home:                       &page{StatefulWidget: m.NewStatefulWidgetBase()},
	})
}

type page struct{ m.StatefulWidget }

func (p *page) CreateState() m.State { return &pageState{} }

type pageState struct {
	m.StateOf[*page]
	mgr *pads.Manager
	// stop ends the listening to the set of pads, and listening the pads that are listened to.
	stop      func()
	listening map[*pads.Pad][]func()
	// angle is the rotation that the gyroscope has turned the square to, in radians.
	angle  float64
	lastAt time.Duration
	status string
}

func (s *pageState) InitState() { s.listening = map[*pads.Pad][]func(){} }

func (s *pageState) DidChangeDependencies() {
	if s.stop != nil {
		return
	}
	var host pads.Host
	if !m.WidgetsBindingOf(s.Context()).PlatformDispatcher().HostAs(&host) {
		s.status = "this platform has no gamepads"
		return
	}
	host.UsePads()
	s.mgr = host.PadManager()
	attach := func() {
		for _, p := range s.mgr.Pads() {
			if _, ok := s.listening[p]; ok {
				continue
			}
			slog.Info("gamepad", "name", p.Name(), "kind", p.Kind().String(), "rumble", p.CanRumble(), "triggers", p.CanRumbleTriggers(),
				"led", p.HasLED(), "touchpads", p.Touchpads(), "sensors", p.HasSensors())
			redraw := func() { s.SetState(func() {}) }
			s.listening[p] = []func(){
				p.AddTouchListener(func(pads.Touch) { redraw() }),
				p.AddSensorListener(func(r pads.SensorReading) {
					s.SetState(func() {
						if r.Kind == platform.PadGyroscope {
							if s.lastAt != 0 {
								s.angle += float64(r.Values[2]) * (r.Time - s.lastAt).Seconds()
							}
							s.lastAt = r.Time
						}
					})
				}),
				p.AddListener(func(pads.Event) { redraw() }),
			}
		}
	}
	attach()
	s.stop = s.mgr.AddListener(func() { s.SetState(attach) })
}

func (s *pageState) Dispose() {
	if s.stop != nil {
		s.stop()
	}
	for _, stops := range s.listening {
		for _, stop := range stops {
			stop()
		}
	}
}

func (s *pageState) check(what string, err error) {
	if err != nil {
		s.SetState(func() { s.status = what + ": " + err.Error() })
		slog.Info(what, "err", err)
	}
}

func (s *pageState) padCard(p *pads.Pad) m.Widget {
	power := p.Power()
	lines := []string{
		fmt.Sprintf("%s (%s)", p.Name(), p.Kind().String()),
		fmt.Sprintf("battery: %d%% (state %d)", power.Percent, power.State),
		fmt.Sprintf("touchpads: %d, fingers down: %d", p.Touchpads(), len(p.Touches())),
	}
	for _, t := range p.Touches() {
		lines = append(lines, fmt.Sprintf("  finger %d at %.2f, %.2f pressure %.2f", t.Finger, t.X, t.Y, t.Pressure))
	}
	if g, ok := p.LatestSensor(platform.PadGyroscope); ok {
		lines = append(lines, fmt.Sprintf("gyro rad/s: %+.2f %+.2f %+.2f", g.Values[0], g.Values[1], g.Values[2]))
	}
	if a, ok := p.LatestSensor(platform.PadAccelerometer); ok {
		lines = append(lines, fmt.Sprintf("accel m/s2: %+.2f %+.2f %+.2f", a.Values[0], a.Values[1], a.Values[2]))
	}
	colour := func(r, g, b uint8) func() {
		return func() { s.check("set the light", p.SetLED(r, g, b)) }
	}
	return m.NewCard(m.NewPadding(m.NewEdgeInsetsAll(12), m.NewColumn([]m.Widget{
		m.NewText(strings.Join(lines, "\n")),
		m.NewWrap([]m.Widget{
			m.NewElevatedButton(colour(255, 0, 0), m.NewText("red")),
			m.NewElevatedButton(colour(0, 255, 0), m.NewText("green")),
			m.NewElevatedButton(colour(0, 0, 255), m.NewText("blue")),
			m.NewElevatedButton(func() { s.check("rumble", p.Rumble(0xC000, 0x6000, time.Second)) }, m.NewText("rumble")),
			m.NewElevatedButton(func() { s.check("trigger rumble", p.RumbleTriggers(0xFFFF, 0xFFFF, time.Second)) }, m.NewText("triggers")),
			m.NewElevatedButton(func() {
				s.check("sensors", p.EnableSensors(!p.SensorsEnabled()))
				s.SetState(func() { s.angle, s.lastAt = 0, 0 })
			}, m.NewText("sensors on/off")),
		}, m.WrapOpts{Spacing: 8, RunSpacing: 8}),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})))
}

func (s *pageState) Build(bc m.BuildContext) m.Widget {
	var cards []m.Widget
	if s.mgr != nil {
		for _, p := range s.mgr.Pads() {
			cards = append(cards, s.padCard(p))
		}
	}
	if len(cards) == 0 {
		cards = append(cards, m.NewText("no pad is connected"))
	}
	if s.status != "" {
		cards = append(cards, m.NewText(s.status, m.TextOpts{Style: &m.TextStyle{Color: m.Some(ui.NewColor(0xFFB3261E))}}))
	}
	cards = append(cards, m.NewSizedBox(m.NewTransformRotate(s.angle,
		m.NewContainer(nil, m.ContainerOpts{Color: m.Some(ui.NewColor(0xFF6750A4))})), m.SizedBoxOpts{Width: m.Some(120.0), Height: m.Some(120.0)}))
	return m.NewScaffold(m.ScaffoldOpts{
		AppBar: m.NewAppBar(m.AppBarOpts{Title: m.NewText("Pad HID drivers")}),
		Body:   m.NewListView(cards),
	})
}
