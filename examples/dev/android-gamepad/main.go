// Command android-gamepad is a development example (step 31.2): a Material page with two buttons, the first one focused, that log
// each time they are activated, and a listener for the gamepads that logs each pad's buttons. The device tests press the A button
// and the D-pad with `adb shell input gamepad keyevent` and check that a press arrives once (a pad's key goes to the gamepad
// service only, never to the keyboard as well) and that the D-pad moves the focus. It uses the public API only and runs on every
// platform.
//
// Run it on a device or the emulator with:
//
//	go run ./cmd/glutter-android run ./examples/dev/android-gamepad
package main

import (
	"log"
	"log/slog"
	"strconv"

	"github.com/rm4n0s/glutter/gamepad"
	m "github.com/rm4n0s/glutter/material"
)

func main() {
	err := m.RunApp(newApp(), m.RunAppOpts{
		AppID:  "dev.glutter.androidgamepad",
		Window: m.WindowConfig{Title: "Android gamepad", Size: m.Size{Width: 480, Height: 640}},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func newApp() m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "Android gamepad",
		DebugShowCheckedModeBanner: m.Some(false),
		Home:                       &page{StatefulWidget: m.NewStatefulWidgetBase()},
	})
}

type page struct{ m.StatefulWidget }

func (p *page) CreateState() m.State { return &pageState{} }

type pageState struct {
	m.StateOf[*page]
	counts [2]int
	// unlisten stops listening to the set of pads, and listening the pads that are listened to.
	unlisten  func()
	listening map[*gamepad.Pad]func()
}

func (s *pageState) InitState() { s.listening = map[*gamepad.Pad]func(){} }

func (s *pageState) DidChangeDependencies() {
	if s.unlisten != nil {
		return
	}
	pads := gamepad.Of(s.Context())
	attach := func() {
		for _, p := range pads.Pads() {
			if _, ok := s.listening[p]; ok {
				continue
			}
			slog.Info("gamepad", "name", p.Name(), "kind", p.Kind().String(), "rumble", p.CanRumble())
			s.listening[p] = p.AddListener(func(e gamepad.Event) {
				slog.Info("pad event", "kind", int(e.Kind), "button", int(e.Button), "axis", int(e.Axis), "value", e.Value)
			})
		}
	}
	attach()
	handle := pads.AddListener(attach)
	s.unlisten = func() { pads.RemoveListener(handle) }
}

func (s *pageState) Dispose() {
	if s.unlisten != nil {
		s.unlisten()
	}
	for _, stop := range s.listening {
		stop()
	}
}

func (s *pageState) button(i int) m.Widget {
	return m.NewElevatedButton(func() {
		s.SetState(func() { s.counts[i]++ })
		slog.Info("button pressed", "index", i, "count", s.counts[i])
	}, m.NewText("button "+strconv.Itoa(i)+": "+strconv.Itoa(s.counts[i])), m.ElevatedButtonOpts{Autofocus: i == 0})
}

func (s *pageState) Build(bc m.BuildContext) m.Widget {
	return m.NewScaffold(m.ScaffoldOpts{
		AppBar: m.NewAppBar(m.AppBarOpts{Title: m.NewText("Android gamepad")}),
		Body: m.NewColumn([]m.Widget{
			m.NewSizedBox(s.button(0), m.SizedBoxOpts{Height: m.Some(80.0)}),
			m.NewSizedBox(s.button(1), m.SizedBoxOpts{Height: m.Some(80.0)}),
		}),
	})
}
