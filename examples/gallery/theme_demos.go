package main

import (
	"time"

	m "github.com/rm4n0s/glutter/material"
)

func buildThemesPage(bc m.BuildContext) m.Widget {
	return m.NewListView([]m.Widget{
		m.NewPadding(m.NewEdgeInsetsFromLTRB(24, 16, 24, 0), m.NewText(
			"Each component has a theme widget that changes the look of the components below it. Every sample sets one field, so it differs from the default.",
			m.TextOpts{Style: m.ThemeOf(bc).TextTheme.BodyMedium})),
		m.NewPadding(m.NewEdgeInsetsFromLTRB(24, 8, 24, 24), NewComponentThemes()),
	})
}

// ComponentThemes shows the theme widgets of the components, each around the
// component it changes.
type ComponentThemes struct{ m.StatelessWidget }

// NewComponentThemes creates the demo.
func NewComponentThemes() *ComponentThemes { return &ComponentThemes{} }

// sample is a component below a theme widget, with the name of the theme.
type sample struct {
	name  string
	build func(bc m.BuildContext, accent m.Color) m.Widget
}

// Build builds the samples in a wrap.
func (w *ComponentThemes) Build(bc m.BuildContext) m.Widget {
	scheme := m.ThemeOf(bc).ColorScheme
	accent := scheme.TertiaryContainer()
	cells := make([]m.Widget, len(themeSamples))
	for i, s := range themeSamples {
		cells[i] = m.NewSizedBox(m.NewColumn([]m.Widget{
			m.NewPadding(m.NewEdgeInsetsFromLTRB(0, 0, 0, 6), m.NewText(s.name, m.TextOpts{Style: m.ThemeOf(bc).TextTheme.LabelLarge})),
			framed(bc, m.NewPadding(m.NewEdgeInsetsAll(8), m.NewAlign(s.build(bc, accent), m.AlignOpts{Alignment: m.AlignmentCenterLeft}))),
		}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStretch}), m.SizedBoxOpts{Width: m.Some(300.0)})
	}
	return m.NewWrap(cells, m.WrapOpts{Spacing: 16.0, RunSpacing: 16.0})
}

func noop() {}

func color(c m.Color) m.Opt[m.Color] { return m.Some(c) }

func allColor(c m.Color) m.WidgetStateProperty[m.Opt[m.Color]] {
	return m.NewWidgetStatePropertyAll(m.Some(c))
}

func fixed(w, h float64, child m.Widget) m.Widget {
	return m.NewSizedBox(child, m.SizedBoxOpts{Width: m.Some(w), Height: m.Some(h)})
}

func buttonStyle(accent m.Color) *m.ButtonStyle {
	return m.NewButtonStyle(m.ButtonStyleOpts{BackgroundColor: allColor(accent)})
}

var themeSamples = []sample{
	{"Theme", func(bc m.BuildContext, a m.Color) m.Widget {
		data := m.NewThemeData(m.ThemeDataOpts{ColorSchemeSeed: m.Some(m.Colors.Green.Primary), Brightness: m.ThemeOf(bc).Brightness()})
		return m.NewTheme(data, m.NewFilledButton(noop, m.NewText("Green theme")))
	}},
	{"AnimatedTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		data := m.NewThemeData(m.ThemeDataOpts{ColorSchemeSeed: m.Some(m.Colors.Orange.Primary), Brightness: m.ThemeOf(bc).Brightness()})
		return m.NewAnimatedTheme(data, m.NewFilledButton(noop, m.NewText("Orange theme")))
	}},
	{"ActionIconTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewActionIconTheme(m.NewActionIconThemeData(m.ActionIconThemeDataOpts{
			BackButtonIconBuilder: func(m.BuildContext) m.Widget { return m.NewIcon(m.Icons.Explore()) }}), m.NewBackButton())
	}},
	{"AppBarTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewAppBarTheme(m.NewAppBarThemeData(m.AppBarThemeDataOpts{BackgroundColor: color(a)}),
			fixed(260, 56, m.NewAppBar(m.AppBarOpts{Title: m.NewText("AppBar"), NoAutomaticallyImplyLeading: true})))
	}},
	{"BadgeTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewBadgeTheme(m.NewBadgeThemeData(m.BadgeThemeDataOpts{BackgroundColor: color(a), TextColor: color(m.Colors.Black)}),
			m.NewBadge(m.NewIcon(m.Icons.Mail()), m.BadgeOpts{Label: m.NewText("3")}))
	}},
	{"BottomAppBarTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewBottomAppBarTheme(m.NewBottomAppBarThemeData(m.BottomAppBarThemeDataOpts{Color: color(a)}),
			fixed(260, 80, m.NewBottomAppBar(m.BottomAppBarOpts{Child: m.NewText("BottomAppBar")})))
	}},
	{"CardTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewCardTheme(m.NewCardThemeData(m.CardThemeDataOpts{Color: color(a)}),
			m.NewCard(m.NewPadding(m.NewEdgeInsetsAll(16), m.NewText("Card"))))
	}},
	{"CarouselViewTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewCarouselViewTheme(m.NewCarouselViewThemeData(m.CarouselViewThemeDataOpts{BackgroundColor: color(a)}),
			fixed(260, 90, m.NewCarouselView(120, []m.Widget{m.NewText("One"), m.NewText("Two"), m.NewText("Three")})))
	}},
	{"CheckboxTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewCheckboxTheme(m.NewCheckboxThemeData(m.CheckboxThemeDataOpts{FillColor: allColor(a)}),
			m.NewCheckbox(m.Some(true), func(m.Opt[bool]) {}))
	}},
	{"ChipTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewChipTheme(m.NewChipThemeData(m.ChipThemeDataOpts{BackgroundColor: color(a)}), m.NewChip(m.NewText("Chip")))
	}},
	{"DataTableTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewDataTableTheme(m.NewDataTableThemeData(m.DataTableThemeDataOpts{DataRowColor: allColor(a)}),
			m.NewDataTable([]m.DataColumn{m.NewDataColumn(m.NewText("Name"))}, []m.DataRow{
				m.NewDataRow([]m.DataCell{m.NewDataCell(m.NewText("Row"))})}, m.DataTableOpts{HideCheckboxColumn: true}))
	}},
	{"DatePickerTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewDatePickerTheme(m.NewDatePickerThemeData(m.DatePickerThemeDataOpts{BackgroundColor: color(a)}),
			m.NewBuilder(func(bc m.BuildContext) m.Widget {
				return m.NewTextButton(func() {
					now := time.Now()
					m.ShowDatePicker(bc, now.AddDate(-1, 0, 0), now.AddDate(1, 0, 0))
				}, m.NewText("Pick a date"))
			}))
	}},
	{"DialogTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewDialogTheme(m.NewDialogThemeData(m.DialogThemeDataOpts{BackgroundColor: color(a)}),
			m.NewBuilder(func(bc m.BuildContext) m.Widget {
				return m.NewTextButton(func() {
					m.ShowDialog[string](bc, func(bc m.BuildContext) m.Widget {
						return m.NewAlertDialog(m.AlertDialogOpts{Title: m.NewText("Themed dialog"), Actions: []m.Widget{
							m.NewTextButton(func() { m.NavigatorPop(bc) }, m.NewText("Close"))}})
					})
				}, m.NewText("Open a dialog"))
			}))
	}},
	{"DividerTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewDividerTheme(m.NewDividerThemeData(m.DividerThemeDataOpts{Color: color(m.ThemeOf(bc).ColorScheme.Primary()), Thickness: m.Some(4.0)}),
			fixed(240, 16, m.NewDivider()))
	}},
	{"DrawerTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewDrawerTheme(m.NewDrawerThemeData(m.DrawerThemeDataOpts{BackgroundColor: color(a)}),
			fixed(160, 90, m.NewDrawer(m.DrawerOpts{Child: m.NewText("Drawer"), Width: m.Some(160.0)})))
	}},
	{"DropdownMenuTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewDropdownMenuTheme(m.NewDropdownMenuThemeData(m.DropdownMenuThemeDataOpts{DisabledColor: color(a)}),
			m.NewDropdownMenu([]m.DropdownMenuEntry[string]{m.NewDropdownMenuEntry("a", "Alpha"), m.NewDropdownMenuEntry("b", "Beta")},
				m.DropdownMenuOpts[string]{Width: m.Some(200.0)}))
	}},
	{"ElevatedButtonTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewElevatedButtonTheme(m.NewElevatedButtonThemeData(m.ElevatedButtonThemeDataOpts{Style: buttonStyle(a)}),
			m.NewElevatedButton(noop, m.NewText("Elevated")))
	}},
	{"ExpansionTileTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewExpansionTileTheme(m.NewExpansionTileThemeData(m.ExpansionTileThemeDataOpts{CollapsedBackgroundColor: color(a)}),
			fixed(260, 60, m.NewExpansionTile(m.NewText("ExpansionTile"))))
	}},
	{"FilledButtonTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewFilledButtonTheme(m.NewFilledButtonThemeData(m.FilledButtonThemeDataOpts{Style: buttonStyle(a)}),
			m.NewFilledButton(noop, m.NewText("Filled")))
	}},
	{"FloatingActionButtonTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewFloatingActionButtonTheme(m.NewFloatingActionButtonThemeData(m.FloatingActionButtonThemeDataOpts{BackgroundColor: color(a)}),
			m.NewFloatingActionButton(noop, m.NewIcon(m.Icons.Add())))
	}},
	{"IconButtonTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewIconButtonTheme(m.NewIconButtonThemeData(m.IconButtonThemeDataOpts{Style: buttonStyle(a)}),
			m.NewIconButton(noop, m.NewIcon(m.Icons.Favorite())))
	}},
	{"InputDecorationTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewInputDecorationTheme(m.NewInputDecorationThemeData(m.InputDecorationThemeDataOpts{Filled: true, FillColor: m.NewWidgetStatePropertyAll(a)}),
			fixed(240, 64, m.NewTextField(m.TextFieldOpts{Decoration: &m.InputDecoration{LabelText: text("Label")}})))
	}},
	{"ListTileTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewListTileTheme(m.NewListTileThemeData(m.ListTileThemeDataOpts{TileColor: color(a)}),
			fixed(260, 56, m.NewListTile(m.ListTileOpts{Title: m.NewText("ListTile")})))
	}},
	{"MaterialBannerTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewMaterialBannerTheme(m.NewMaterialBannerThemeData(m.MaterialBannerThemeDataOpts{BackgroundColor: color(a)}),
			fixed(260, 70, m.NewMaterialBanner(m.NewText("A banner"), []m.Widget{m.NewTextButton(noop, m.NewText("OK"))})))
	}},
	{"MenuTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewMenuTheme(m.NewMenuThemeData(m.MenuThemeDataOpts{Style: m.NewMenuStyle(m.MenuStyleOpts{BackgroundColor: allColor(a)})}),
			m.NewMenuAnchor([]m.Widget{m.NewMenuItemButton(m.MenuItemButtonOpts{OnPressed: noop, Child: m.NewText("Item")})}, m.MenuAnchorOpts{
				Builder: func(bc m.BuildContext, c *m.MenuController, _ m.Widget) m.Widget {
					return m.NewTextButton(func() {
						if c.IsOpen() {
							c.Close()
						} else {
							c.Open()
						}
					}, m.NewText("Menu"))
				}}))
	}},
	{"MenuBarTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewMenuBarTheme(m.NewMenuBarThemeData(m.MenuBarThemeDataOpts{Style: m.NewMenuStyle(m.MenuStyleOpts{BackgroundColor: allColor(a)})}),
			m.NewMenuBar([]m.Widget{m.NewMenuItemButton(m.MenuItemButtonOpts{OnPressed: noop, Child: m.NewText("MenuBar")})}))
	}},
	{"MenuButtonTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewMenuButtonTheme(m.NewMenuButtonThemeData(m.MenuButtonThemeDataOpts{Style: buttonStyle(a)}),
			m.NewMenuBar([]m.Widget{m.NewMenuItemButton(m.MenuItemButtonOpts{OnPressed: noop, Child: m.NewText("MenuItemButton")})}))
	}},
	{"NavigationBarTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewNavigationBarTheme(m.NewNavigationBarThemeData(m.NavigationBarThemeDataOpts{BackgroundColor: color(a)}),
			fixed(300, 80, m.NewNavigationBar([]m.Widget{
				m.NewNavigationDestination(m.NewIcon(m.Icons.Home()), "Home"), m.NewNavigationDestination(m.NewIcon(m.Icons.Search()), "Search")})))
	}},
	{"NavigationDrawerTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewNavigationDrawerTheme(m.NewNavigationDrawerThemeData(m.NavigationDrawerThemeDataOpts{BackgroundColor: color(a)}),
			fixed(240, 140, m.NewNavigationDrawer([]m.Widget{
				m.NewNavigationDrawerDestination(m.NewIcon(m.Icons.Home()), m.NewText("Home")),
				m.NewNavigationDrawerDestination(m.NewIcon(m.Icons.Search()), m.NewText("Search"))}, m.NavigationDrawerOpts{})))
	}},
	{"NavigationRailTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewNavigationRailTheme(m.NewNavigationRailThemeData(m.NavigationRailThemeDataOpts{BackgroundColor: color(a)}),
			fixed(80, 150, m.NewNavigationRail([]m.NavigationRailDestination{
				m.NewNavigationRailDestination(m.NewIcon(m.Icons.Home()), m.NewText("Home")),
				m.NewNavigationRailDestination(m.NewIcon(m.Icons.Search()), m.NewText("Search"))}, m.Some(0))))
	}},
	{"OutlinedButtonTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewOutlinedButtonTheme(m.NewOutlinedButtonThemeData(m.OutlinedButtonThemeDataOpts{Style: buttonStyle(a)}),
			m.NewOutlinedButton(noop, m.NewText("Outlined")))
	}},
	{"PopupMenuTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewPopupMenuTheme(m.NewPopupMenuThemeData(m.PopupMenuThemeDataOpts{Color: color(a)}),
			m.NewPopupMenuButton(func(m.BuildContext) []m.PopupMenuEntry {
				return []m.PopupMenuEntry{m.NewPopupMenuItem(m.NewText("Item"), m.PopupMenuItemOpts[string]{Value: m.Some("item")})}
			}, m.PopupMenuButtonOpts[string]{}))
	}},
	{"ProgressIndicatorTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewProgressIndicatorTheme(m.NewProgressIndicatorThemeData(m.ProgressIndicatorThemeDataOpts{Color: color(m.ThemeOf(bc).ColorScheme.Tertiary())}),
			fixed(240, 8, m.NewLinearProgressIndicator(m.LinearProgressIndicatorOpts{Value: m.Some(0.6)})))
	}},
	{"RadioTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewRadioTheme(m.NewRadioThemeData(m.RadioThemeDataOpts{FillColor: allColor(m.ThemeOf(bc).ColorScheme.Tertiary())}),
			m.NewRadio("a", m.RadioOpts[string]{GroupValue: m.Some("a"), OnChanged: func(m.Opt[string]) {}}))
	}},
	{"ScrollbarTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		controller := m.NewScrollController()
		return m.NewScrollbarTheme(m.NewScrollbarThemeData(m.ScrollbarThemeDataOpts{ThumbColor: allColor(m.ThemeOf(bc).ColorScheme.Tertiary())}),
			fixed(240, 80, m.NewScrollbar(m.NewListView([]m.Widget{m.NewText("One"), m.NewText("Two"), m.NewText("Three"), m.NewText("Four"),
				m.NewText("Five"), m.NewText("Six"), m.NewText("Seven")}, m.ListViewOpts{Controller: controller}),
				m.ScrollbarOpts{Controller: controller, ThumbVisibility: m.Some(true)})))
	}},
	{"SearchBarTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewSearchBarTheme(m.NewSearchBarThemeData(m.SearchBarThemeDataOpts{BackgroundColor: allColor(a)}),
			fixed(260, 56, m.NewSearchBar(m.SearchBarOpts{HintText: text("SearchBar")})))
	}},
	{"SearchViewTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewSearchViewTheme(m.NewSearchViewThemeData(m.SearchViewThemeDataOpts{BackgroundColor: color(a)}),
			m.NewSearchAnchor(func(bc m.BuildContext, c *m.SearchController) m.Widget {
				return m.NewTextButton(func() { c.OpenView() }, m.NewText("Open the view"))
			}, func(_ m.BuildContext, _ *m.SearchController, deliver func([]m.Widget)) {
				deliver([]m.Widget{m.NewListTile(m.ListTileOpts{Title: m.NewText("Suggestion")})})
			}))
	}},
	{"SegmentedButtonTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewSegmentedButtonTheme(m.NewSegmentedButtonThemeData(m.SegmentedButtonThemeDataOpts{Style: buttonStyle(a)}),
			m.NewSegmentedButton([]m.ButtonSegment[string]{{Value: "a", Label: m.NewText("A")}, {Value: "b", Label: m.NewText("B")}}, []string{"a"}, func([]string) {}))
	}},
	{"SnackBarTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewSnackBarTheme(m.NewSnackBarThemeData(m.SnackBarThemeDataOpts{BackgroundColor: color(a)}),
			m.NewBuilder(func(bc m.BuildContext) m.Widget {
				return m.NewTextButton(func() {
					m.ScaffoldMessengerOf(bc).ShowSnackBar(m.NewSnackBar(m.NewText("A themed snack bar")))
				}, m.NewText("Show a snack bar"))
			}))
	}},
	{"SwitchTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewSwitchTheme(m.NewSwitchThemeData(m.SwitchThemeDataOpts{TrackColor: allColor(a)}), m.NewSwitch(true, func(bool) {}))
	}},
	{"TabBarTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewTabBarTheme(m.NewTabBarThemeData(m.TabBarThemeDataOpts{IndicatorColor: color(m.ThemeOf(bc).ColorScheme.Tertiary())}),
			fixed(260, 48, m.NewDefaultTabController(2, m.NewTabBar([]m.Widget{m.NewTab(m.TabOpts{Text: "One"}), m.NewTab(m.TabOpts{Text: "Two"})}))))
	}},
	{"TextButtonTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewTextButtonTheme(m.NewTextButtonThemeData(m.TextButtonThemeDataOpts{Style: buttonStyle(a)}),
			m.NewTextButton(noop, m.NewText("Text")))
	}},
	{"TextSelectionTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewTextSelectionTheme(m.NewTextSelectionThemeData(m.TextSelectionThemeDataOpts{CursorColor: color(m.ThemeOf(bc).ColorScheme.Tertiary())}),
			fixed(240, 56, m.NewTextField(m.TextFieldOpts{Decoration: &m.InputDecoration{LabelText: text("Cursor color")}})))
	}},
	{"TimePickerTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewTimePickerTheme(m.NewTimePickerThemeData(m.TimePickerThemeDataOpts{BackgroundColor: color(a)}),
			m.NewBuilder(func(bc m.BuildContext) m.Widget {
				return m.NewTextButton(func() { m.ShowTimePicker(bc, m.TimeOfDay{Hour: 10, Minute: 30}) }, m.NewText("Pick a time"))
			}))
	}},
	{"TooltipTheme", func(bc m.BuildContext, a m.Color) m.Widget {
		return m.NewTooltipTheme(m.NewTooltipThemeData(m.TooltipThemeDataOpts{PreferBelow: m.Some(false)}),
			m.NewTooltip(m.NewText("Hover me"), m.TooltipOpts{Message: "A themed tooltip"}))
	}},
}
