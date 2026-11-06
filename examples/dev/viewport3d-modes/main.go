// Command viewport3d-modes is a development example (step 30.9): one part (a box, a cylinder and a sphere) in the four display
// modes of package viewport3d (shaded, shaded with edges, wireframe and hidden line), side by side, each in a viewport3d.View
// that the mouse and the touch orbit, pan and zoom. It uses the part of internal/scenetest, so it is under examples/dev.
//
// Run it with: go run ./examples/dev/viewport3d-modes
//
// It needs a GPU renderer (Vulkan, Direct3D 12 or WebGPU); with another one each view shows its fallback, a line of text.
//
// Environment: GLUTTER_BACKEND=x11|wayland, GLUTTER_RENDERER=vulkan, GLUTTER_SCREENSHOT=<file.png>.
package main

import (
	"log"

	"github.com/rm4n0s/glutter/internal/scenetest"
	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/viewport3d"
)

func main() {
	err := m.RunApp(NewModesApp(), m.RunAppOpts{
		AppID:  "dev.glutter.viewport3d-modes",
		Window: m.WindowConfig{Title: "viewport3d: display modes", Size: m.Size{Width: 1280, Height: 420}},
	})
	if err != nil {
		log.Fatal(err)
	}
}

// ModesApp is the Material app.
type ModesApp struct{ m.StatelessWidget }

// NewModesApp creates the app.
func NewModesApp() *ModesApp { return &ModesApp{} }

// Build makes the app with the page of the four views.
func (a *ModesApp) Build(bc m.BuildContext) m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "viewport3d",
		Theme:                      m.NewThemeData(m.ThemeDataOpts{ColorSchemeSeed: m.Some(m.Colors.Teal.Primary), Brightness: m.BrightnessDark}),
		DebugShowCheckedModeBanner: m.Some(false),
		Home:                       NewModesPage(),
	})
}

// ModesPage is the page: a row of the four modes.
type ModesPage struct{ m.StatefulWidget }

// NewModesPage creates the page.
func NewModesPage(opts ...m.WidgetOpts) *ModesPage {
	return &ModesPage{StatefulWidget: m.NewStatefulWidgetBase(opts...)}
}

// CreateState creates the state.
func (p *ModesPage) CreateState() m.State { return &modesPageState{} }

// panel is a scene and its camera, with the name of its mode.
type panel struct {
	name   string
	scene  *viewport3d.Scene
	camera *viewport3d.Camera
}

type modesPageState struct {
	m.StateOf[*ModesPage]
	panels []panel
}

// InitState makes the four scenes: each has its own items, since an item is in one scene at a time.
func (s *modesPageState) InitState() {
	modes := []struct {
		name string
		mode viewport3d.DisplayMode
	}{
		{"Shaded", viewport3d.DisplayModeShaded},
		{"Shaded with edges", viewport3d.DisplayModeShadedWithEdges},
		{"Wireframe", viewport3d.DisplayModeWireframe},
		{"Hidden line", viewport3d.DisplayModeHiddenLine},
	}
	for _, md := range modes {
		scene := viewport3d.NewScene()
		scene.SetStyle(viewport3d.SceneStyle{DisplayMode: md.mode})
		scene.Add(scenetest.Part()...)
		s.panels = append(s.panels, panel{name: md.name, scene: scene, camera: scenetest.PartCamera(scene, 320, 360)})
	}
}

// Build makes the row of views, each under its name.
func (s *modesPageState) Build(bc m.BuildContext) m.Widget {
	var tiles []m.Widget
	for _, p := range s.panels {
		view := viewport3d.NewView(p.scene, p.camera, viewport3d.ViewOpts{
			SemanticsLabel: p.name,
			Fallback:       m.NewCenter(m.NewText("This app needs a GPU renderer.")),
		})
		tiles = append(tiles, m.NewExpanded(m.NewColumn([]m.Widget{
			m.NewPadding(m.NewEdgeInsetsAll(8), m.NewText(p.name, m.TextOpts{Style: m.ThemeOf(bc).TextTheme.TitleMedium})),
			m.NewExpanded(view),
		})))
	}
	return m.NewScaffold(m.ScaffoldOpts{
		AppBar: m.NewAppBar(m.AppBarOpts{Title: m.NewText("One part, four display modes")}),
		Body:   m.NewRow(tiles),
	})
}
