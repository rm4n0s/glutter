// Command viewport3d-section is a development example (step 30.10): a part (a box, a cylinder and a sphere) cut by a section plane,
// with its cap and the outline of the cut, in a viewport3d.View that the mouse and the touch orbit, pan and zoom. A tap picks what
// is under the pointer (ViewController.Pick), selects it (the highlight color and the outline) and shows what the pick found: the
// item, the triangle, the point of the world and the normal. A switch turns the cap on and off. It uses the part of
// internal/scenetest, so it is under examples/dev.
//
// Run it with: go run ./examples/dev/viewport3d-section
//
// It needs a GPU renderer (Vulkan, Direct3D 12 or WebGPU); with another one the view shows its fallback, a line of text.
//
// Environment: GLUTTER_BACKEND=x11|wayland, GLUTTER_RENDERER=vulkan, GLUTTER_SCREENSHOT=<file.png>,
// GLUTTER_SCREENSHOT_AFTER=<duration>.
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/rm4n0s/glutter/internal/scenetest"
	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
)

func main() {
	err := m.RunApp(NewSectionApp(), m.RunAppOpts{
		AppID:  "dev.glutter.viewport3d-section",
		Window: m.WindowConfig{Title: "viewport3d: section, pick and selection", Size: m.Size{Width: 1000, Height: 560}},
	})
	if err != nil {
		log.Fatal(err)
	}
}

// SectionApp is the Material app.
type SectionApp struct{ m.StatelessWidget }

// NewSectionApp creates the app.
func NewSectionApp() *SectionApp { return &SectionApp{} }

// Build makes the app with the page.
func (a *SectionApp) Build(bc m.BuildContext) m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "viewport3d",
		Theme:                      m.NewThemeData(m.ThemeDataOpts{ColorSchemeSeed: m.Some(m.Colors.Teal.Primary), Brightness: m.BrightnessDark}),
		DebugShowCheckedModeBanner: m.Some(false),
		Home:                       NewSectionPage(),
	})
}

// SectionPage is the page: the view, and next to it what the last pick found.
type SectionPage struct{ m.StatefulWidget }

// NewSectionPage creates the page.
func NewSectionPage(opts ...m.WidgetOpts) *SectionPage {
	return &SectionPage{StatefulWidget: m.NewStatefulWidgetBase(opts...)}
}

// CreateState creates the state.
func (p *SectionPage) CreateState() m.State { return &sectionPageState{} }

type sectionPageState struct {
	m.StateOf[*SectionPage]
	scene  *viewport3d.Scene
	camera *viewport3d.Camera
	ctl    *viewport3d.ViewController
	items  []*viewport3d.Item
	names  map[*viewport3d.Item]string
	cap    bool
	// picked is what the last pick found, as text.
	picked string
}

// The plane that cuts the part: what is on the side of its normal is cut away.
var cutPlane = vecmath.NewPlaneFromPointNormal(vecmath.Vec3{X: 0.1, Z: 0.1}, vecmath.Vec3{X: 1, Z: 1}.Normalize())

// InitState makes the scene with the section plane, and asks, after the first frame, for what is under the sphere.
func (s *sectionPageState) InitState() {
	s.cap = true
	s.picked = "Tap the part to pick."
	s.scene = viewport3d.NewScene()
	s.items = scenetest.Part()
	s.names = map[*viewport3d.Item]string{s.items[0]: "box", s.items[1]: "cylinder", s.items[2]: "sphere"}
	s.scene.Add(s.items...)
	s.scene.SetStyle(s.style(true))
	s.scene.SetSectionPlanes(cutPlane)
	s.camera = scenetest.PartCamera(s.scene, 700, 500)
	s.ctl = viewport3d.NewViewController()
	// The view has its GPU device a moment after its first frame: the first pick waits for it.
	binding := m.WidgetsBindingOf(s.Context())
	time.AfterFunc(500*time.Millisecond, func() {
		binding.Post(func() {
			// The sphere, where it is drawn: the picture of the proof starts with it picked.
			if at, ok := s.camera.Project(s.items[2].Bounds().Center()); ok {
				s.pick(at)
			}
		})
	})
}

// pick asks for what is under a point of the view and selects it.
func (s *sectionPageState) pick(at ui.Offset) {
	s.ctl.Pick(at, func(r viewport3d.PickResult, err error) {
		s.SetState(func() {
			switch {
			case err != nil:
				s.picked = "The pick failed: " + err.Error()
			case !r.Hit:
				s.scene.ClearSelection()
				s.picked = "Nothing there."
			default:
				s.scene.Select(r.Item)
				s.picked = fmt.Sprintf("Picked the %s\ntriangle %d\npoint (%.3f, %.3f, %.3f)\nnormal (%.2f, %.2f, %.2f)",
					s.names[r.Item], r.Index, r.Point.X, r.Point.Y, r.Point.Z, r.Normal.X, r.Normal.Y, r.Normal.Z)
			}
		})
	})
}

// style is the style of the scene: the grid under the part, and the cap of the section when it is on.
func (s *sectionPageState) style(cap bool) viewport3d.SceneStyle {
	return viewport3d.SceneStyle{
		Section: viewport3d.SectionStyle{NoCap: !cap},
		Grid:    viewport3d.GridStyle{Show: true, Origin: vecmath.Vec3{Y: -0.5}, Spacing: 0.5, MajorEvery: 4},
	}
}

func (s *sectionPageState) setCap(on bool) {
	s.SetState(func() {
		s.cap = on
		s.scene.SetStyle(s.style(on))
	})
}

// Build makes the view and the panel.
func (s *sectionPageState) Build(bc m.BuildContext) m.Widget {
	view := m.NewGestureDetector(
		viewport3d.NewView(s.scene, s.camera, viewport3d.ViewOpts{
			Controller:     s.ctl,
			SemanticsLabel: "A part cut by a plane",
			// The axes and the view cube turn with the camera; the label follows the sphere, and hides when something is in front.
			Axes:     &viewport3d.AxesOpts{},
			ViewCube: &viewport3d.ViewCubeOpts{},
			Children: []m.Widget{viewport3d.NewAnchored(s.items[2].Bounds().Center().Add(vecmath.Vec3{Y: 0.5}), m.NewText("Sphere"),
				viewport3d.AnchoredOpts{Alignment: m.AlignmentBottomCenter, HideWhenCovered: true})},
			Fallback: m.NewCenter(m.NewText("This app needs a GPU renderer.")),
		}),
		m.GestureDetectorOpts{OnTapUp: func(d m.TapUpDetails) { s.pick(d.LocalPosition) }},
	)
	panel := m.NewSizedBox(m.NewPadding(m.NewEdgeInsetsAll(16), m.NewColumn([]m.Widget{
		m.NewText("Section plane", m.TextOpts{Style: m.ThemeOf(bc).TextTheme.TitleMedium}),
		m.NewSwitchListTile(s.cap, s.setCap, m.SwitchListTileOpts{Title: m.NewText("Cap")}),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(16.0)}),
		m.NewText("Pick", m.TextOpts{Style: m.ThemeOf(bc).TextTheme.TitleMedium}),
		m.NewText(s.picked),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})), m.SizedBoxOpts{Width: m.Some(300.0)})
	return m.NewScaffold(m.ScaffoldOpts{
		AppBar: m.NewAppBar(m.AppBarOpts{Title: m.NewText("A part cut by a plane, one item picked")}),
		Body:   m.NewRow([]m.Widget{m.NewExpanded(view), panel}),
	})
}
