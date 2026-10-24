package main

import (
	"os"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	m "github.com/rm4n0s/glutter/material"
)

// AppBarDemos shows three app bars: a standard one, one with a centered title
// and a bottom widget, and one with a flexible space.
type AppBarDemos struct{ m.StatelessWidget }

// NewAppBarDemos creates the demos.
func NewAppBarDemos() *AppBarDemos { return &AppBarDemos{} }

func (w *AppBarDemos) Build(bc m.BuildContext) m.Widget {
	colors := m.ThemeOf(bc).ColorScheme
	bottom := m.NewPreferredSize(m.Size{Width: 0, Height: 48},
		m.NewSizedBox(m.NewCenter(m.NewText("A bottom widget")), m.SizedBoxOpts{Height: m.Some(48.0)}))
	bars := []*m.AppBar{
		m.NewAppBar(m.AppBarOpts{
			NoPrimary: true,
			Leading:   NewIconAction(m.Icons.Menu(), "Menu", func() {}),
			Title:     m.NewText("Standard"),
			Actions:   []m.Widget{NewIconAction(m.Icons.Search(), "Search", func() {}), NewIconAction(m.Icons.Favorite(), "Favorite", func() {})},
		}),
		m.NewAppBar(m.AppBarOpts{
			NoPrimary:       true,
			CenterTitle:     m.Some(true),
			Title:           m.NewText("Centered, with a bottom"),
			Bottom:          bottom,
			BackgroundColor: m.Some(colors.SurfaceContainer()),
		}),
		m.NewAppBar(m.AppBarOpts{
			NoPrimary:       true,
			Title:           m.NewText("Flexible space"),
			ToolbarHeight:   m.Some(72.0),
			FlexibleSpace:   m.NewColoredBox(colors.TertiaryContainer(), m.NewSizedBoxExpand(nil)),
			ForegroundColor: m.Some(colors.OnTertiaryContainer()),
			Actions:         []m.Widget{NewIconAction(m.Icons.Info(), "Info", func() {})},
		}),
	}
	children := make([]m.Widget, 0, 2*len(bars))
	for i, b := range bars {
		if i > 0 {
			children = append(children, m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(12.0)}))
		}
		// An app bar gets its height from the scaffold; out of one, it needs the height.
		children = append(children, framed(bc, m.NewSizedBox(b, m.SizedBoxOpts{Height: m.Some(b.PreferredSize().Height)})))
	}
	return m.NewColumn(children)
}

// SliverAppBarDemos shows a medium and a large SliverAppBar, each in a scroll view
// that can be scrolled.
type SliverAppBarDemos struct{ m.StatelessWidget }

// NewSliverAppBarDemos creates the demos.
func NewSliverAppBarDemos() *SliverAppBarDemos { return &SliverAppBarDemos{} }

func (w *SliverAppBarDemos) Build(bc m.BuildContext) m.Widget {
	list := func(bar *m.SliverAppBar) m.Widget {
		return m.NewSizedBox(framed(bc, m.NewMaterial(m.NewCustomScrollView([]m.Widget{
			bar,
			m.NewSliverFixedExtentListBuilder(func(_ m.BuildContext, i int) m.Widget {
				if i >= 30 {
					return nil
				}
				return m.NewPadding(m.NewEdgeInsetsSymmetric(0, 16), m.NewAlign(m.NewText("Item "+itoa(i+1)), m.AlignOpts{Alignment: m.AlignmentCenterLeft}))
			}, 48),
		}, m.CustomScrollViewOpts{Primary: m.Some(false)}), m.MaterialOpts{Type: m.MaterialTypeTransparency})),
			m.SizedBoxOpts{Width: m.Some((sectionWidth(bc) - 12) / 2), Height: m.Some(280.0)})
	}
	return m.NewRow([]m.Widget{
		list(m.NewSliverAppBarMedium(m.SliverAppBarOpts{NoPrimary: true, Title: m.NewText("Medium")})),
		m.NewSizedBox(nil, m.SizedBoxOpts{Width: m.Some(12.0)}),
		list(m.NewSliverAppBarLarge(m.SliverAppBarOpts{NoPrimary: true, Title: m.NewText("Large")})),
	})
}

// MessageDemos has buttons that show snack bars and a banner.
type MessageDemos struct{ m.StatefulWidget }

// NewMessageDemos creates the demos.
func NewMessageDemos() *MessageDemos { return &MessageDemos{} }

// CreateState creates the state.
func (w *MessageDemos) CreateState() m.State { return &messageDemosState{} }

type messageDemosState struct{ m.StateOf[*MessageDemos] }

// InitState shows a snack bar when GLUTTER_DEV_SNACKBAR is set in a debug build,
// for the screenshots of the proof.
func (s *messageDemosState) InitState() {
	if !devSnackBar(os.Getenv) {
		return
	}
	m.WidgetsBindingOf(s.Context()).Scheduler().AddPostFrameCallback(func(time.Duration) {
		m.ScaffoldMessengerOf(s.Context()).ShowSnackBar(m.NewSnackBar(m.NewText("Item deleted"), m.SnackBarOpts{
			Action: m.NewSnackBarAction("UNDO", func() {}),
		}))
	})
}

func (s *messageDemosState) Build(bc m.BuildContext) m.Widget {
	messenger := func() *m.ScaffoldMessengerState { return m.ScaffoldMessengerOf(bc) }
	snack := func(label string, make func() *m.SnackBar) m.Widget {
		return demoButton(bc, label, func() { messenger().ShowSnackBar(make()) })
	}
	return m.NewWrap([]m.Widget{
		snack("Show SnackBar", func() *m.SnackBar { return m.NewSnackBar(m.NewText("A fixed snack bar")) }),
		snack("With an action", func() *m.SnackBar {
			return m.NewSnackBar(m.NewText("Item deleted"), m.SnackBarOpts{
				Action: m.NewSnackBarAction("UNDO", func() {}),
			})
		}),
		snack("Floating", func() *m.SnackBar {
			return m.NewSnackBar(m.NewText("A floating snack bar"), m.SnackBarOpts{Behavior: m.SnackBarBehaviorFloating, Width: m.Some(360.0)})
		}),
		snack("With a close icon", func() *m.SnackBar {
			return m.NewSnackBar(m.NewText("It can be closed"), m.SnackBarOpts{ShowCloseIcon: m.Some(true)})
		}),
		demoButton(bc, "Show MaterialBanner", func() {
			var banner *m.ScaffoldFeatureController[*m.MaterialBanner, m.MaterialBannerClosedReason]
			banner = messenger().ShowMaterialBanner(m.NewMaterialBanner(
				m.NewText("A banner stays until it is dismissed."),
				[]m.Widget{demoButton(bc, "DISMISS", func() { banner.Close() })},
				m.MaterialBannerOpts{Leading: m.NewIcon(m.Icons.Info())}))
		}),
		demoButton(bc, "Clear all", func() {
			messenger().ClearSnackBars()
			messenger().ClearMaterialBanners()
		}),
	}, m.WrapOpts{Spacing: 12, RunSpacing: 12})
}

// BottomAppBarDemo is a small Scaffold with a BottomAppBar that has a notch for a
// floating action button that docks in it.
type BottomAppBarDemo struct{ m.StatelessWidget }

// NewBottomAppBarDemo creates the demo.
func NewBottomAppBarDemo() *BottomAppBarDemo { return &BottomAppBarDemo{} }

func (w *BottomAppBarDemo) Build(bc m.BuildContext) m.Widget {
	return m.NewSizedBox(framed(bc, m.NewScaffold(m.ScaffoldOpts{
		AppBar: m.NewAppBar(m.AppBarOpts{NoPrimary: true, NoAutomaticallyImplyLeading: true, Title: m.NewText("A small scaffold")}),
		Body:   m.NewCenter(m.NewText("The button docks in the notch of the bar")),
		BottomNavigationBar: m.NewBottomAppBar(m.BottomAppBarOpts{
			Shape:  m.CircularNotchedRectangle{},
			Height: m.Some(64.0),
			Child: m.NewRow([]m.Widget{
				NewIconAction(m.Icons.Menu(), "Menu", func() {}),
				NewIconAction(m.Icons.Search(), "Search", func() {}),
			}),
		}),
		FloatingActionButton:         m.NewFloatingActionButton(func() {}, m.NewIcon(m.Icons.Add()), m.FloatingActionButtonOpts{Tooltip: "Add"}),
		FloatingActionButtonLocation: m.FloatingActionButtonLocationCenterDocked,
	})), m.SizedBoxOpts{Width: m.Some(columnWidth), Height: m.Some(230.0)})
}

// devSnackBar is whether GLUTTER_DEV_SNACKBAR is "1" in a debug build.
func devSnackBar(getenv func(string) string) bool {
	return foundation.DebugMode && getenv("GLUTTER_DEV_SNACKBAR") == "1"
}
