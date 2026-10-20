package main

import (
	m "github.com/rm4n0s/glutter/material"
)

func buildNavigationPage(bc m.BuildContext) m.Widget {
	top := func(children ...m.Widget) m.Widget {
		return m.NewRow(children, m.RowOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
	}
	return m.NewListView([]m.Widget{
		top(section(bc, "NavigationBar", NewNavigationBarDemo()), section(bc, "TabBar and TabBarView", NewTabsDemo())),
		top(section(bc, "NavigationRail", NewNavigationRailDemo()), section(bc, "NavigationDrawer", NewNavigationDrawerDemo())),
		top(section(bc, "Drawer and DrawerHeader", NewDrawerDemo()), section(bc, "Bottom sheets", NewSheetDemos())),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(24.0)}),
	})
}

// destinationLabels are the destinations of the demos.
var destinationLabels = []struct {
	label string
	icon  func() m.IconData
}{
	{"Explore", m.Icons.Explore},
	{"Inbox", m.Icons.Inbox},
	{"Favorites", m.Icons.Favorite},
	{"Settings", m.Icons.Settings},
}

// NavigationBarDemo is a small Scaffold with a NavigationBar that selects the text
// that the body shows.
type NavigationBarDemo struct{ m.StatefulWidget }

// NewNavigationBarDemo creates the demo.
func NewNavigationBarDemo() *NavigationBarDemo { return &NavigationBarDemo{} }

// CreateState creates the state.
func (w *NavigationBarDemo) CreateState() m.State { return &navigationBarDemoState{} }

type navigationBarDemoState struct {
	m.StateOf[*NavigationBarDemo]
	selected int
}

func (s *navigationBarDemoState) Build(bc m.BuildContext) m.Widget {
	destinations := make([]m.Widget, 3)
	for i := range destinations {
		d := destinationLabels[i]
		destinations[i] = m.NewNavigationDestination(m.NewIcon(d.icon()), d.label)
	}
	return m.NewSizedBox(framed(bc, m.NewScaffold(m.ScaffoldOpts{
		Body: m.NewCenter(m.NewText(destinationLabels[s.selected].label + " page")),
		BottomNavigationBar: m.NewNavigationBar(destinations, m.NavigationBarOpts{
			SelectedIndex:         s.selected,
			OnDestinationSelected: func(i int) { s.SetState(func() { s.selected = i }) },
		}),
	})), m.SizedBoxOpts{Width: m.Some(columnWidth), Height: m.Some(230.0)})
}

// NavigationRailDemo is a NavigationRail that shows the labels of all the
// destinations, with a floating action button on top.
type NavigationRailDemo struct{ m.StatefulWidget }

// NewNavigationRailDemo creates the demo.
func NewNavigationRailDemo() *NavigationRailDemo { return &NavigationRailDemo{} }

// CreateState creates the state.
func (w *NavigationRailDemo) CreateState() m.State { return &navigationRailDemoState{} }

type navigationRailDemoState struct {
	m.StateOf[*NavigationRailDemo]
	selected int
}

func (s *navigationRailDemoState) Build(bc m.BuildContext) m.Widget {
	destinations := make([]m.NavigationRailDestination, len(destinationLabels))
	for i, d := range destinationLabels {
		destinations[i] = m.NewNavigationRailDestination(m.NewIcon(d.icon()), m.NewText(d.label))
	}
	return m.NewSizedBox(framed(bc, m.NewScaffold(m.ScaffoldOpts{
		Body: m.NewRow([]m.Widget{
			m.NewNavigationRail(destinations, m.Some(s.selected), m.NavigationRailOpts{
				LabelType:             m.NavigationRailLabelTypeAll,
				OnDestinationSelected: func(i int) { s.SetState(func() { s.selected = i }) },
			}),
			m.NewExpanded(m.NewCenter(m.NewText(destinationLabels[s.selected].label + " page"))),
		}),
	})), m.SizedBoxOpts{Width: m.Some(columnWidth), Height: m.Some(380.0)})
}

// NavigationDrawerDemo shows a NavigationDrawer in place, with a headline above the
// destinations.
type NavigationDrawerDemo struct{ m.StatefulWidget }

// NewNavigationDrawerDemo creates the demo.
func NewNavigationDrawerDemo() *NavigationDrawerDemo { return &NavigationDrawerDemo{} }

// CreateState creates the state.
func (w *NavigationDrawerDemo) CreateState() m.State { return &navigationDrawerDemoState{} }

type navigationDrawerDemoState struct {
	m.StateOf[*NavigationDrawerDemo]
	selected int
}

func (s *navigationDrawerDemoState) Build(bc m.BuildContext) m.Widget {
	theme := m.ThemeOf(bc)
	children := []m.Widget{
		m.NewPadding(m.NewEdgeInsetsFromLTRB(28, 16, 16, 10), m.NewText("Mail", m.TextOpts{Style: theme.TextTheme.TitleSmall})),
	}
	for _, d := range destinationLabels {
		children = append(children, m.NewNavigationDrawerDestination(m.NewIcon(d.icon()), m.NewText(d.label)))
	}
	return m.NewSizedBox(framed(bc, m.NewNavigationDrawer(children, m.NavigationDrawerOpts{
		SelectedIndex:         m.Some(s.selected),
		OnDestinationSelected: func(i int) { s.SetState(func() { s.selected = i }) },
	})), m.SizedBoxOpts{Width: m.Some(304.0), Height: m.Some(380.0)})
}

// TabsDemo has a primary TabBar with a TabBarView that can be dragged, and a
// scrollable secondary TabBar.
type TabsDemo struct{ m.StatelessWidget }

// NewTabsDemo creates the demo.
func NewTabsDemo() *TabsDemo { return &TabsDemo{} }

func (w *TabsDemo) Build(bc m.BuildContext) m.Widget {
	primary := m.NewDefaultTabController(3, m.NewColumn([]m.Widget{
		m.NewTabBar([]m.Widget{
			m.NewTab(m.TabOpts{Text: "Flights", Icon: m.NewIcon(m.Icons.Send())}),
			m.NewTab(m.TabOpts{Text: "Trips", Icon: m.NewIcon(m.Icons.Explore())}),
			m.NewTab(m.TabOpts{Text: "Explore", Icon: m.NewIcon(m.Icons.Search())}),
		}),
		m.NewExpanded(m.NewTabBarView([]m.Widget{
			m.NewCenter(m.NewText("Flights")), m.NewCenter(m.NewText("Trips")), m.NewCenter(m.NewText("Explore")),
		})),
	}))
	names := []string{"Overview", "Specs", "Reviews", "Support", "Related", "Accessories"}
	tabs := make([]m.Widget, len(names))
	for i, n := range names {
		tabs[i] = m.NewTab(m.TabOpts{Text: n})
	}
	secondary := m.NewDefaultTabController(len(names), m.NewColumn([]m.Widget{
		m.NewTabBarSecondary(tabs, m.TabBarOpts{IsScrollable: true}),
	}))
	return m.NewColumn([]m.Widget{
		m.NewSizedBox(framed(bc, m.NewMaterial(primary, m.MaterialOpts{Type: m.MaterialTypeTransparency})),
			m.SizedBoxOpts{Width: m.Some(columnWidth), Height: m.Some(200.0)}),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(16.0)}),
		m.NewSizedBox(framed(bc, m.NewMaterial(secondary, m.MaterialOpts{Type: m.MaterialTypeTransparency})),
			m.SizedBoxOpts{Width: m.Some(columnWidth), Height: m.Some(50.0)}),
	})
}

// DrawerDemo is a small Scaffold with a Drawer: the menu button of its app bar
// opens it.
type DrawerDemo struct{ m.StatelessWidget }

// NewDrawerDemo creates the demo.
func NewDrawerDemo() *DrawerDemo { return &DrawerDemo{} }

func (w *DrawerDemo) Build(bc m.BuildContext) m.Widget {
	colors := m.ThemeOf(bc).ColorScheme
	item := func(label string, icon m.IconData) m.Widget {
		return m.NewSizedBox(m.NewRow([]m.Widget{
			m.NewPadding(m.NewEdgeInsetsSymmetric(0, 16), m.NewIcon(icon)),
			m.NewText(label),
		}), m.SizedBoxOpts{Height: m.Some(48.0)})
	}
	return m.NewSizedBox(framed(bc, m.NewScaffold(m.ScaffoldOpts{
		AppBar: m.NewAppBar(m.AppBarOpts{NoPrimary: true, Title: m.NewText("Tap the menu")}),
		Body:   m.NewCenter(m.NewText("Or drag from the edge")),
		Drawer: m.NewDrawer(m.DrawerOpts{Child: m.NewColumn([]m.Widget{
			m.NewDrawerHeader(m.DrawerHeaderOpts{
				Decoration: m.NewBoxDecoration(m.BoxDecorationOpts{Color: m.Some(colors.PrimaryContainer())}),
				Child:      m.NewAlign(m.NewText("Drawer header"), m.AlignOpts{Alignment: m.AlignmentBottomLeft}),
			}),
			item("Inbox", m.Icons.Inbox()),
			item("Sent", m.Icons.Send()),
			item("Trash", m.Icons.Delete()),
		}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStretch})}),
	})), m.SizedBoxOpts{Width: m.Some(columnWidth), Height: m.Some(260.0)})
}

// SheetDemos has the buttons that show a modal bottom sheet and a persistent one.
type SheetDemos struct{ m.StatelessWidget }

// NewSheetDemos creates the demos.
func NewSheetDemos() *SheetDemos { return &SheetDemos{} }

func (w *SheetDemos) Build(bc m.BuildContext) m.Widget {
	sheet := func(title string, closeLabel string, onClose func(bc m.BuildContext)) m.WidgetBuilder {
		return func(bc m.BuildContext) m.Widget {
			return m.NewSizedBox(m.NewPadding(m.NewEdgeInsetsAll(24), m.NewColumn([]m.Widget{
				m.NewText(title, m.TextOpts{Style: m.ThemeOf(bc).TextTheme.TitleLarge}),
				m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(16.0)}),
				demoButton(bc, closeLabel, func() { onClose(bc) }),
			}, m.ColumnOpts{MainAxisSize: m.MainAxisSizeMin, CrossAxisAlignment: m.CrossAxisAlignmentStart})), m.SizedBoxOpts{Width: m.Some(1000.0)})
		}
	}
	return m.NewWrap([]m.Widget{
		demoButton(bc, "Modal bottom sheet", func() {
			m.ShowModalBottomSheet(bc, sheet("A modal bottom sheet", "Close", func(bc m.BuildContext) { m.NavigatorPop(bc) }),
				m.ShowModalBottomSheetOpts{ShowDragHandle: m.Some(true)})
		}),
		demoButton(bc, "Persistent bottom sheet", func() {
			var controller *m.PersistentBottomSheetController
			controller = m.ShowBottomSheet(bc, sheet("A persistent bottom sheet", "Close", func(m.BuildContext) { controller.Close() }),
				m.ShowBottomSheetOpts{ShowDragHandle: m.Some(true)})
		}),
	}, m.WrapOpts{Spacing: 12, RunSpacing: 12})
}
