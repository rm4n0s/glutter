// Command gallery shows the Material components of Glutter, one page for each
// group of them, with a side list to choose the page and a toggle between the
// light and the dark theme. It uses only package material, imported as m.
//
// The first page is the home page, with an index of the components and the choice
// of the seed color and of the light, dark or system mode.
//
// Run it with: go run ./examples/gallery
//
// Environment: GLUTTER_BACKEND=x11|wayland, GLUTTER_SCREENSHOT=<file.png>,
// GLUTTER_SCREENSHOT_AFTER=<duration>, GLUTTER_GALLERY_PAGE=<slug> opens the page with that
// slug (in every build; scripts/perf.sh uses it), GLUTTER_GALLERY_ANIMATE=1 starts its progress indicators animating, and
// GLUTTER_GALLERY_REBUILD=1 builds the whole app again in every frame (scripts/perf-web.sh measures the
// frames of a full rebuild with it). On the web they are query parameters (?GLUTTER_GALLERY_PAGE=data).
// Serve it with: go run ./cmd/glutter-web serve ./examples/gallery. In debug builds: GLUTTER_DEV_PAGE=<slug>
// does the same, GLUTTER_DEV_THEME=dark starts in dark mode, and
// GLUTTER_DEV_SNACKBAR=1 shows a snack bar when the first page opens, and
// GLUTTER_DEV_OPEN_MENU=file/recent opens those menus of the Menus page, and
// GLUTTER_DEV_TEXT_FIELDS=focus,validate focuses a field and validates the form
// of the Text fields page, and GLUTTER_DEV_QUERY=gr opens the search view of the
// Search page with that text, and GLUTTER_DEV_PERFORMANCE_OVERLAY=1 shows the
// performance overlay at the top of the window.
package main

import (
	"log"
	"os"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	m "github.com/rm4n0s/glutter/material"
)

func main() {
	err := m.RunApp(NewGalleryApp(), m.RunAppOpts{
		AppID:  "dev.glutter.gallery",
		Window: m.WindowConfig{Title: appTitle, Size: m.Size{Width: 1100, Height: 760}},
	})
	if err != nil {
		log.Fatal(err)
	}
}

// appTitle is the title of the window and of the app.
const appTitle = "Glutter \u2013 Material 3 gallery"

// seedOption is a color that the light and the dark theme can be made from.
type seedOption struct {
	name  string
	color m.Color
}

// seedOptions are the colors of the seed picker; the first is the default.
var seedOptions = []seedOption{
	{"Blue", m.Colors.Blue.Primary},
	{"Teal", m.Colors.Teal.Primary},
	{"Green", m.Colors.Green.Primary},
	{"Orange", m.Colors.Orange.Primary},
	{"Pink", m.Colors.Pink.Primary},
	{"Purple", m.Colors.Purple.Primary},
	{"Red", m.Colors.Red.Primary},
}

// GalleryScope gives the pages the state of the app: the seed color and the theme
// mode, and the ways to change them and to open another page.
type GalleryScope struct {
	m.InheritedWidget
	seed   int
	mode   m.ThemeMode
	onSeed func(index int)
	onMode func(mode m.ThemeMode)
	onPage func(slug string)
	// pages are the pages of the gallery, for the index of the home page.
	pages []Page
}

// NewGalleryScope creates the scope around the shell.
func NewGalleryScope(pages []Page, seed int, mode m.ThemeMode, onSeed func(int), onMode func(m.ThemeMode), onPage func(string), child m.Widget) *GalleryScope {
	return &GalleryScope{InheritedWidget: m.NewInheritedWidgetBase(child), seed: seed, mode: mode, onSeed: onSeed, onMode: onMode, onPage: onPage, pages: pages}
}

// UpdateShouldNotify is whether the seed or the mode changed.
func (w *GalleryScope) UpdateShouldNotify(old m.Widget) bool {
	o := old.(*GalleryScope)
	return o.seed != w.seed || o.mode != w.mode
}

// GalleryScopeOf is the scope above the context.
func GalleryScopeOf(bc m.BuildContext) *GalleryScope {
	scope, ok := m.DependOn[*GalleryScope](bc)
	if !ok {
		panic("the gallery pages need a GalleryScope above them")
	}
	return scope
}

// GalleryApp is the app: it owns the page that is open and the theme mode.
type GalleryApp struct{ m.StatefulWidget }

// NewGalleryApp creates the app.
func NewGalleryApp() *GalleryApp { return &GalleryApp{} }

// CreateState creates the state.
func (a *GalleryApp) CreateState() m.State { return &galleryAppState{} }

type galleryAppState struct {
	m.StateOf[*GalleryApp]
	page int
	seed int
	mode m.ThemeMode
	// light and dark are the themes of the seed in lightFor: they are made again only when the
	// seed changes, not in every build. A theme is a big value to make (a color scheme is solved from
	// the seed) and to compare (a new ThemeData is compared field by field with the old one, an old one
	// that is the same pointer is not), and the gallery rebuilds the whole app when the mode changes.
	light, dark *m.ThemeData
	lightFor    int
}

// themes are the light and the dark theme of the seed color.
func (s *galleryAppState) themes() (light, dark *m.ThemeData) {
	if s.light == nil || s.lightFor != s.seed {
		seed := seedOptions[s.seed].color
		s.light = m.NewThemeData(m.ThemeDataOpts{ColorSchemeSeed: m.Some(seed)})
		s.dark = m.NewThemeData(m.ThemeDataOpts{ColorSchemeSeed: m.Some(seed), Brightness: m.BrightnessDark})
		s.lightFor = s.seed
	}
	return s.light, s.dark
}

func (s *galleryAppState) InitState() {
	s.page = devPage(os.Getenv, galleryPages)
	s.mode = devThemeMode(os.Getenv)
	if os.Getenv("GLUTTER_GALLERY_REBUILD") == "1" {
		s.rebuildEveryFrame()
	}
}

// rebuildEveryFrame makes the app build again after every frame, so that each frame is a full
// rebuild of the page, as when the theme changes (GLUTTER_GALLERY_REBUILD, for the measurements of
// SPEC §15). It runs from the post-frame callbacks, which also ask for the next frame.
func (s *galleryAppState) rebuildEveryFrame() {
	var again func(time.Duration)
	again = func(time.Duration) {
		if !s.Mounted() {
			return
		}
		s.SetState(func() {})
		m.WidgetsBindingOf(s.Context()).RunPostFrame(again)
	}
	m.WidgetsBindingOf(s.Context()).RunPostFrame(again)
}

func (s *galleryAppState) Build(bc m.BuildContext) m.Widget {
	light, dark := s.themes()
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      appTitle,
		Theme:                      light,
		DarkTheme:                  dark,
		ThemeMode:                  s.mode,
		DebugShowCheckedModeBanner: m.Some(false),
		ShowPerformanceOverlay:     devPerformanceOverlay(os.Getenv),
		Home: NewGalleryScope(galleryPages, s.seed, s.mode,
			func(seed int) { s.SetState(func() { s.seed = seed }) },
			func(mode m.ThemeMode) { s.SetState(func() { s.mode = mode }) },
			func(slug string) {
				for i, p := range galleryPages {
					if p.Slug == slug {
						s.SetState(func() { s.page = i })
					}
				}
			},
			NewGalleryShell(s.page, s.mode == m.ThemeModeDark,
				func(page int) { s.SetState(func() { s.page = page }) },
				func() {
					s.SetState(func() {
						if s.mode == m.ThemeModeDark {
							s.mode = m.ThemeModeLight
						} else {
							s.mode = m.ThemeModeDark
						}
					})
				})),
	})
}

// devPage is the index of the page that GLUTTER_GALLERY_PAGE asks for, in every build, or
// GLUTTER_DEV_PAGE, in debug builds: the page with that slug, or the first one when
// neither is set and when no page has the slug.
func devPage(getenv func(string) string, pages []Page) int {
	slug := getenv("GLUTTER_GALLERY_PAGE")
	if slug == "" && foundation.DebugMode {
		slug = getenv("GLUTTER_DEV_PAGE")
	}
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

// devPerformanceOverlay is true when GLUTTER_DEV_PERFORMANCE_OVERLAY is "1" in a debug
// build.
func devPerformanceOverlay(getenv func(string) string) bool {
	return foundation.DebugMode && getenv("GLUTTER_DEV_PERFORMANCE_OVERLAY") == "1"
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

// narrowWidth is the width in logical pixels under which the shell, as on a phone, keeps the list of the
// pages in a drawer instead of beside the page (Material's compact window size class).
const narrowWidth = 600.0

func (w *GalleryShell) Build(bc m.BuildContext) m.Widget {
	colors := m.ThemeOf(bc).ColorScheme
	icon, tooltip := m.Icons.DarkMode(), "Switch to the dark theme"
	if w.dark {
		icon, tooltip = m.Icons.LightMode(), "Switch to the light theme"
	}
	appBar := m.NewAppBar(m.AppBarOpts{
		Title:   m.NewText("Glutter gallery"),
		Actions: []m.Widget{NewIconAction(icon, tooltip, w.onToggleMode), m.NewSizedBox(nil, m.SizedBoxOpts{Width: m.Some(8.0)})},
	})
	if m.MediaQueryWidthOf(bc) < narrowWidth {
		return m.NewScaffold(m.ScaffoldOpts{
			AppBar: appBar,
			Drawer: m.NewBuilder(func(dc m.BuildContext) m.Widget {
				return NewPageDrawer(w.page, func(i int) {
					w.onPage(i)
					m.ScaffoldOf(dc).CloseDrawer()
				})
			}),
			Body: galleryPages[w.page].Build(bc),
		})
	}
	return m.NewScaffold(m.ScaffoldOpts{
		AppBar: appBar,
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

// pageDrawerKey is the key of the drawer of the pages.
var pageDrawerKey = m.NewKey("page-drawer")

// NewPageDrawer is the navigation drawer of a narrow window (a phone) with a destination for each page.
func NewPageDrawer(selected int, onSelect func(int)) m.Widget {
	children := make([]m.Widget, len(galleryPages))
	for i, p := range galleryPages {
		children[i] = m.NewNavigationDrawerDestination(m.NewIcon(p.Icon), m.NewText(p.Name))
	}
	return m.NewNavigationDrawer(children, m.NavigationDrawerOpts{
		Key: pageDrawerKey, SelectedIndex: m.Some(selected), OnDestinationSelected: onSelect})
}
