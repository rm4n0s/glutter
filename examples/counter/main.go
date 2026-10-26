// Command counter is the reference example of the SPEC (§12.12): a Material 3
// app with an app bar, a counter and a floating action button that adds one. It
// uses only package material, imported as m.
//
// Run it with: go run ./examples/counter
//
// On Windows: go run ./cmd/glutter-windows run ./examples/counter (under Wine on Linux, headless with
// -headless; GLUTTER_BACKEND is win32 there). Nothing of this example is missing on Windows.
//
// Environment: GLUTTER_BACKEND=x11|wayland (Linux) or win32 (Windows), GLUTTER_SCREENSHOT=<file.png>,
// GLUTTER_SCREENSHOT_AFTER=<duration>. In debug builds: GLUTTER_DEV_TAPS=<n>
// starts the counter at n, as if the button had been tapped n times.
package main

import (
	"log"
	"os"
	"strconv"

	"github.com/rm4n0s/glutter/foundation"
	m "github.com/rm4n0s/glutter/material"
)

func main() {
	err := m.RunApp(NewMyApp(), m.RunAppOpts{
		AppID:  "dev.glutter.counter",
		Window: m.WindowConfig{Title: "Counter", Size: m.Size{Width: 480, Height: 640}},
	})
	if err != nil {
		log.Fatal(err)
	}
}

type MyApp struct{ m.StatelessWidget }

func NewMyApp() *MyApp { return &MyApp{} }

func (a *MyApp) Build(bc m.BuildContext) m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "Counter",
		Theme:                      m.NewThemeData(m.ThemeDataOpts{ColorSchemeSeed: m.Some(m.Colors.Blue.Primary)}),
		DebugShowCheckedModeBanner: m.Some(false),
		Home:                       NewCounterPage("Counter"),
	})
}

type CounterPage struct {
	m.StatefulWidget
	title string
}

func NewCounterPage(title string, opts ...m.WidgetOpts) *CounterPage {
	return &CounterPage{StatefulWidget: m.NewStatefulWidgetBase(opts...), title: title}
}

func (p *CounterPage) Title() string        { return p.title }
func (p *CounterPage) CreateState() m.State { return &counterPageState{} }

type counterPageState struct {
	m.StateOf[*CounterPage]
	count int
}

func (s *counterPageState) InitState() { s.count = devTaps(os.Getenv) }

func (s *counterPageState) increment() { s.SetState(func() { s.count++ }) }

func (s *counterPageState) Build(bc m.BuildContext) m.Widget {
	return m.NewScaffold(m.ScaffoldOpts{
		AppBar: m.NewAppBar(m.AppBarOpts{Title: m.NewText(s.Widget().Title())}),
		Body: m.NewCenter(
			m.NewColumn([]m.Widget{
				m.NewText("You have pushed the button this many times:"),
				m.NewText(strconv.Itoa(s.count), m.TextOpts{Style: m.ThemeOf(bc).TextTheme.HeadlineMedium}),
			}, m.ColumnOpts{MainAxisAlignment: m.MainAxisAlignmentCenter}),
		),
		FloatingActionButton: m.NewFloatingActionButton(s.increment, m.NewIcon(m.Icons.Add()),
			m.FloatingActionButtonOpts{Tooltip: "Increment"}),
	})
}

// devTaps is the number of taps that GLUTTER_DEV_TAPS asks for; it is zero in
// release and profile builds and when the value is not a non-negative number.
func devTaps(getenv func(string) string) int {
	if !foundation.DebugMode {
		return 0
	}
	n, err := strconv.Atoi(getenv("GLUTTER_DEV_TAPS"))
	if err != nil || n < 0 {
		return 0
	}
	return n
}
