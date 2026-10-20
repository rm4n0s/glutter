// Command gallery shows the Material components of Glutter, one page for each
// group of them, with a side list to choose the page and a toggle between the
// light and the dark theme. It uses only package material, imported as m.
//
// Run it with: go run ./examples/gallery
//
// Environment: GLUTTER_BACKEND=x11|wayland, GLUTTER_SCREENSHOT=<file.png>,
// GLUTTER_SCREENSHOT_AFTER=<duration>. In debug builds: GLUTTER_DEV_PAGE=<slug>
// opens the page with that slug, GLUTTER_DEV_THEME=dark starts in dark mode, and
// GLUTTER_DEV_SNACKBAR=1 shows a snack bar when the first page opens.
package main

import (
	"log"
	"os"

	"github.com/rm4n0s/glutter/foundation"
	m "github.com/rm4n0s/glutter/material"
)

func main() {
	err := m.RunApp(NewGalleryApp(), m.RunAppOpts{
		AppID:  "dev.glutter.gallery",
		Window: m.WindowConfig{Title: "Glutter gallery", Size: m.Size{Width: 1100, Height: 760}},
	})
	if err != nil {
		log.Fatal(err)
	}
}

// seed is the color that the light and the dark theme are made from.
var seed = m.Colors.Blue.Primary

// GalleryApp is the app: it owns the page that is open and the theme mode.
type GalleryApp struct{ m.StatefulWidget }

// NewGalleryApp creates the app.
func NewGalleryApp() *GalleryApp { return &GalleryApp{} }

// CreateState creates the state.
func (a *GalleryApp) CreateState() m.State { return &galleryAppState{} }

type galleryAppState struct {
	m.StateOf[*GalleryApp]
	page int
	mode m.ThemeMode
}

func (s *galleryAppState) InitState() {
	s.page = devPage(os.Getenv, galleryPages)
	s.mode = devThemeMode(os.Getenv)
}

func (s *galleryAppState) Build(bc m.BuildContext) m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "Glutter gallery",
		Theme:                      m.NewThemeData(m.ThemeDataOpts{ColorSchemeSeed: m.Some(seed)}),
		DarkTheme:                  m.NewThemeData(m.ThemeDataOpts{ColorSchemeSeed: m.Some(seed), Brightness: m.BrightnessDark}),
		ThemeMode:                  s.mode,
		DebugShowCheckedModeBanner: m.Some(false),
		Home: NewGalleryShell(s.page, s.mode == m.ThemeModeDark,
			func(page int) { s.SetState(func() { s.page = page }) },
			func() {
				s.SetState(func() {
					if s.mode == m.ThemeModeDark {
						s.mode = m.ThemeModeLight
					} else {
						s.mode = m.ThemeModeDark
					}
				})
			}),
	})
}

// devPage is the index of the page that GLUTTER_DEV_PAGE asks for: the page with
// that slug, or the first one in release and profile builds and when no page has
// the slug.
func devPage(getenv func(string) string, pages []Page) int {
	if !foundation.DebugMode {
		return 0
	}
	slug := getenv("GLUTTER_DEV_PAGE")
	for i, p := range pages {
		if p.Slug == slug {
			return i
		}
	}
	return 0
}

// devThemeMode is dark when GLUTTER_DEV_THEME is "dark" in a debug build, and
// light otherwise.
func devThemeMode(getenv func(string) string) m.ThemeMode {
	if foundation.DebugMode && getenv("GLUTTER_DEV_THEME") == "dark" {
		return m.ThemeModeDark
	}
	return m.ThemeModeLight
}

// GalleryShell is the frame of every page: the app bar with the theme toggle,
// the side list of the pages, and the page that is open.
type GalleryShell struct {
	m.StatelessWidget
	page         int
	dark         bool
	onPage       func(int)
	onToggleMode func()
}

// NewGalleryShell creates the shell showing the page with the index.
func NewGalleryShell(page int, dark bool, onPage func(int), onToggleMode func()) *GalleryShell {
	return &GalleryShell{page: page, dark: dark, onPage: onPage, onToggleMode: onToggleMode}
}

func (w *GalleryShell) Build(bc m.BuildContext) m.Widget {
	colors := m.ThemeOf(bc).ColorScheme
	icon, tooltip := m.Icons.DarkMode(), "Switch to the dark theme"
	if w.dark {
		icon, tooltip = m.Icons.LightMode(), "Switch to the light theme"
	}
	return m.NewScaffold(m.ScaffoldOpts{
		AppBar: m.NewAppBar(m.AppBarOpts{
			Title:   m.NewText("Glutter gallery"),
			Actions: []m.Widget{NewIconAction(icon, tooltip, w.onToggleMode), m.NewSizedBox(nil, m.SizedBoxOpts{Width: m.Some(8.0)})},
		}),
		Body: m.NewRow([]m.Widget{
			NewPageRail(w.page, w.onPage),
			m.NewColoredBox(colors.OutlineVariant(), m.NewSizedBox(nil, m.SizedBoxOpts{Width: m.Some(1.0)})),
			m.NewExpanded(galleryPages[w.page].Build(bc)),
		}, m.RowOpts{CrossAxisAlignment: m.CrossAxisAlignmentStretch}),
	})
}

// NewIconAction is an icon button with a tooltip.
func NewIconAction(icon m.IconData, tooltip string, onTap func()) m.Widget {
	return m.NewIconButton(onTap, m.NewIcon(icon), m.IconButtonOpts{Tooltip: tooltip})
}

// NewPageRail is the navigation rail on the side with a destination for each
// page, extended so that the names of the pages show next to the icons.
func NewPageRail(selected int, onSelect func(int)) m.Widget {
	destinations := make([]m.NavigationRailDestination, len(galleryPages))
	for i, p := range galleryPages {
		destinations[i] = m.NewNavigationRailDestination(m.NewIcon(p.Icon), m.NewText(p.Name))
	}
	return m.NewNavigationRail(destinations, m.Some(selected), m.NavigationRailOpts{
		Extended: true, MinExtendedWidth: m.Some(240.0), OnDestinationSelected: onSelect})
}
