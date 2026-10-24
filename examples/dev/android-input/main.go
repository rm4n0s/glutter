// Command android-input is a development example (step 25.6): a Material page whose Shortcuts
// handle the keys Back and K, so that the device tests can tell a key that a widget handled (it is
// taken, and logs "shortcut ran") from one that nothing handled (Android gets it again, and Back
// closes the activity). It uses the public API only and runs on every platform.
//
// Run it on a device or the emulator with:
//
//	go run ./cmd/glutter-android run ./examples/dev/android-input
package main

import (
	"log"
	"log/slog"
	"strconv"

	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/services"
)

func main() {
	err := m.RunApp(newApp(), m.RunAppOpts{
		AppID:  "dev.glutter.androidinput",
		Window: m.WindowConfig{Title: "Android input", Size: m.Size{Width: 480, Height: 640}},
	})
	if err != nil {
		log.Fatal(err)
	}
}

// tickIntent is what the shortcuts ask for: count one more key.
type tickIntent struct{ m.IntentBase }

func newApp() m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "Android input",
		DebugShowCheckedModeBanner: m.Some(false),
		Home:                       &page{StatefulWidget: m.NewStatefulWidgetBase()},
	})
}

type page struct{ m.StatefulWidget }

func (p *page) CreateState() m.State { return &pageState{} }

type pageState struct {
	m.StateOf[*page]
	count int
}

func (s *pageState) Build(bc m.BuildContext) m.Widget {
	body := m.NewScaffold(m.ScaffoldOpts{
		AppBar: m.NewAppBar(m.AppBarOpts{Title: m.NewText("Android input")}),
		Body: m.NewCenter(m.NewText(strconv.Itoa(s.count),
			m.TextOpts{Style: m.ThemeOf(bc).TextTheme.HeadlineLarge})),
	})
	return m.NewShortcuts(m.ShortcutMap{
		{Activator: m.NewSingleActivator(services.LogicalKeyboardKeyGoBack), Intent: tickIntent{}},
		{Activator: m.NewSingleActivator(services.LogicalKeyboardKeyKeyK), Intent: tickIntent{}},
	}, m.NewActions(m.ActionMap{
		m.IntentTypeOf[tickIntent](): m.NewCallbackAction(func(tickIntent) any {
			s.SetState(func() { s.count++ })
			slog.Info("shortcut ran", "count", s.count)
			return nil
		}),
	}, m.NewFocus(body, m.FocusOpts{Autofocus: true})))
}
