// Command android-a11y is a development example (step 25.11): a Material page with a button, a text
// field, a list and a live region, which logs what happens to each (the button's count, the list's scroll
// offset, the text field's text), so that the device tests can drive it with accessibility actions as
// TalkBack does (the dev events script's a11y: events), read the nodes the system sees (uiautomator) and
// check the events that are sent. It uses the public API only and runs on every platform.
//
// Run it on a device or the emulator with the bridge forced on (the images without TalkBack):
//
//	go run ./cmd/glutter-android run -env GLUTTER_A11Y=1 ./examples/dev/android-a11y
package main

import (
	"fmt"
	"log"
	"log/slog"

	m "github.com/rm4n0s/glutter/material"
)

func main() {
	err := m.RunApp(newApp(), m.RunAppOpts{
		AppID:  "dev.glutter.androida11y",
		Window: m.WindowConfig{Title: "Android a11y", Size: m.Size{Width: 480, Height: 640}},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func newApp() m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "Android a11y",
		DebugShowCheckedModeBanner: m.Some(false),
		Home:                       &page{StatefulWidget: m.NewStatefulWidgetBase()},
	})
}

type page struct{ m.StatefulWidget }

func (p *page) CreateState() m.State { return &pageState{} }

type pageState struct {
	m.StateOf[*page]
	count      int
	controller *m.ScrollControllerBase
	logged     int
}

func (s *pageState) InitState() {
	s.controller = m.NewScrollController()
	s.controller.AddListener(func() {
		// One line for each 50 logical pixels, so that a scroll is seen but the log stays short.
		if step := int(s.controller.Offset() / 50); step != s.logged {
			s.logged = step
			slog.Info("list scrolled", "offset", fmt.Sprintf("%.0f", s.controller.Offset()))
		}
	})
}

func (s *pageState) Dispose() { s.controller.Dispose() }

func (s *pageState) Build(bc m.BuildContext) m.Widget {
	slog.Info("build", "count", s.count)
	return m.NewScaffold(m.ScaffoldOpts{
		AppBar: m.NewAppBar(m.AppBarOpts{Title: m.NewText("Android a11y")}),
		Body: m.NewColumn([]m.Widget{
			m.NewPadding(m.NewEdgeInsetsAll(8), m.NewElevatedButton(func() {
				s.SetState(func() { s.count++ })
				slog.Info("button pressed", "count", s.count)
			}, m.NewText("Increment"))),
			m.NewSemantics(m.NewText(fmt.Sprintf("Count: %d", s.count)), m.SemanticsOpts{LiveRegion: m.Some(true)}),
			m.NewPadding(m.NewEdgeInsetsAll(8), m.NewTextField(m.TextFieldOpts{
				Decoration: &m.InputDecoration{LabelText: m.Some("Name")},
				OnChanged:  func(text string) { slog.Info("field changed", "text", text) },
			})),
			m.NewExpanded(m.NewSemantics(m.NewListViewBuilder(func(bc m.BuildContext, i int) m.Widget {
				return m.NewListTile(m.ListTileOpts{Title: m.NewText(fmt.Sprintf("Item %d", i))})
			}, m.ListViewOpts{ItemCount: m.Some(100), Controller: s.controller}), m.SemanticsOpts{
				Container: true, Label: m.Some("Items"),
			})),
		}),
	})
}
