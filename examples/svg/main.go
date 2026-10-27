// Command svg shows the SVG pictures of package svg (the widgets of flutter_svg). On the left, a
// grid of eight icons whose currentColor comes from a DefaultSvgTheme, the same icons tinted with a
// ColorFilter, and one icon at three sizes; on the right, a logo with gradients and a card with SVG
// text and a drop-shadow filter. The pictures are SVG files in assets/, embedded in the program
// and loaded as assets. It uses only the public packages: material, imported as m, and svg.
//
// Run it with: go run ./examples/svg
//
// On the web: go run ./cmd/glutter-web serve ./examples/svg. On Android: go run ./cmd/glutter-android
// run ./examples/svg. On Windows: go run ./cmd/glutter-windows run ./examples/svg (under Wine on
// Linux, headless with -headless; GLUTTER_BACKEND is win32 there). Nothing of this example is
// missing on any of them: the pictures are parsed and drawn by plain Go.
//
// Environment: GLUTTER_BACKEND=x11|wayland (Linux) or win32 (Windows), GLUTTER_SCREENSHOT=<file.png>,
// GLUTTER_SCREENSHOT_AFTER=<duration> (the pictures load on goroutines: a shot of the finished
// page asks for a short delay).
package main

import (
	"embed"
	"io/fs"
	"log"
	"strconv"

	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/svg"
)

//go:embed assets
var assetFiles embed.FS

// Assets are the SVG files of the example, with the keys "assets/icons/home.svg", "assets/logo.svg" and
// "assets/card.svg", as an asset has in a Flutter app.
func Assets() fs.FS { return assetFiles }

func main() {
	err := m.RunApp(NewSvgApp(), m.RunAppOpts{
		AppID:  "dev.glutter.svg",
		Assets: Assets(),
		Window: m.WindowConfig{Title: "SVG", Size: m.Size{Width: 760, Height: 560}},
	})
	if err != nil {
		log.Fatal(err)
	}
}

// SvgApp is the Material app around the page.
type SvgApp struct{ m.StatelessWidget }

// NewSvgApp creates the app.
func NewSvgApp() *SvgApp { return &SvgApp{} }

// Build builds the app.
func (a *SvgApp) Build(m.BuildContext) m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "SVG",
		Theme:                      m.NewThemeData(m.ThemeDataOpts{ColorSchemeSeed: m.Some(m.Colors.Indigo.Primary)}),
		DebugShowCheckedModeBanner: m.Some(false),
		Home:                       NewSvgPage(),
	})
}

// SvgPage shows the pictures.
type SvgPage struct{ m.StatelessWidget }

// NewSvgPage creates the page.
func NewSvgPage() *SvgPage { return &SvgPage{} }

// iconNames are the files of assets/icons, without the extension.
var iconNames = []string{"home", "search", "settings", "heart", "bell", "mail", "star", "user"}

func gap(width, height float64) m.Widget {
	return m.NewSizedBox(nil, m.SizedBoxOpts{Width: m.Some(width), Height: m.Some(height)})
}

// Build builds the page.
func (p *SvgPage) Build(bc m.BuildContext) m.Widget {
	theme := m.ThemeOf(bc)
	scheme := theme.ColorScheme
	heading := func(s string) m.Widget {
		return m.NewPadding(m.NewEdgeInsetsSymmetric(8, 0), m.NewText(s, m.TextOpts{Style: theme.TextTheme.TitleSmall}))
	}

	// The icons say currentColor; a DefaultSvgTheme above them gives its value.
	var plain []m.Widget
	for _, name := range iconNames {
		plain = append(plain, svg.NewSvgPictureAsset("assets/icons/"+name+".svg", svg.SvgPictureAssetOpts{
			SvgPictureOpts: svg.SvgPictureOpts{Width: m.Some(36.0), Height: m.Some(36.0)},
		}))
	}
	themed := svg.NewDefaultSvgTheme(svg.SvgTheme{CurrentColor: scheme.Primary()}, m.NewWrap(plain, m.WrapOpts{Spacing: 12, RunSpacing: 12}))

	// The same files, black by default, tinted by a ColorFilter on the whole picture.
	tints := []m.Color{scheme.Primary(), scheme.Secondary(), scheme.Tertiary(), scheme.Error()}
	var tinted []m.Widget
	for i, name := range iconNames {
		tinted = append(tinted, svg.NewSvgPictureAsset("assets/icons/"+name+".svg", svg.SvgPictureAssetOpts{
			SvgPictureOpts: svg.SvgPictureOpts{
				Width: m.Some(36.0), Height: m.Some(36.0),
				ColorFilter: m.NewColorFilterMode(tints[i%len(tints)], m.BlendModeSrcIn),
			},
		}))
	}

	// One file at three sizes: the picture is recorded once and scaled.
	var sizes []m.Widget
	for i, size := range []float64{24, 48, 96} {
		if i > 0 {
			sizes = append(sizes, gap(20, 0))
		}
		sizes = append(sizes, m.NewColumn([]m.Widget{
			svg.NewSvgPictureAsset("assets/icons/star.svg", svg.SvgPictureAssetOpts{SvgPictureOpts: svg.SvgPictureOpts{
				Width: m.Some(size), Theme: m.Some(svg.SvgTheme{CurrentColor: scheme.Tertiary()}),
			}}),
			gap(0, 4),
			m.NewText(strconv.Itoa(int(size))+" px", m.TextOpts{Style: theme.TextTheme.BodySmall}),
		}))
	}

	left := m.NewColumn([]m.Widget{
		heading("Icons: currentColor from the theme"),
		themed,
		gap(0, 18),
		heading("The same icons with a ColorFilter"),
		m.NewWrap(tinted, m.WrapOpts{Spacing: 12, RunSpacing: 12}),
		gap(0, 18),
		heading("One icon, three sizes"),
		m.NewRow(sizes, m.RowOpts{CrossAxisAlignment: m.CrossAxisAlignmentEnd}),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})

	right := m.NewColumn([]m.Widget{
		heading("A logo with gradients"),
		svg.NewSvgPictureAsset("assets/logo.svg", svg.SvgPictureAssetOpts{SvgPictureOpts: svg.SvgPictureOpts{Width: m.Some(150.0)}}),
		gap(0, 18),
		heading("SVG text and a drop-shadow filter"),
		svg.NewSvgPictureAsset("assets/card.svg", svg.SvgPictureAssetOpts{SvgPictureOpts: svg.SvgPictureOpts{Width: m.Some(300.0)}}),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})

	return m.NewScaffold(m.ScaffoldOpts{
		AppBar: m.NewAppBar(m.AppBarOpts{Title: m.NewText("SVG pictures")}),
		Body: m.NewPadding(m.NewEdgeInsetsAll(16), m.NewSingleChildScrollView(m.NewRow([]m.Widget{
			m.NewExpanded(left, m.ExpandedOpts{Flex: m.Some(5)}),
			gap(24, 0),
			m.NewExpanded(right, m.ExpandedOpts{Flex: m.Some(4)}),
		}, m.RowOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart}))),
	})
}
