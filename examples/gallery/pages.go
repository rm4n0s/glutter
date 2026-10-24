package main

import (
	"strconv"

	m "github.com/rm4n0s/glutter/material"
)

// Page is a page of the gallery.
type Page struct {
	// Slug is the name of the page in GLUTTER_DEV_PAGE.
	Slug string
	// Name is the name in the navigation rail.
	Name string
	// Icon is the icon in the navigation rail.
	Icon m.IconData
	// Components names the components that the page shows, for the index of the
	// home page.
	Components string
	// Build makes the content of the page.
	Build func(bc m.BuildContext) m.Widget
}

// galleryPages are the pages, in the order of the navigation rail. Each step of
// the Material components adds its own.
var galleryPages = []Page{
	{Slug: "home", Name: "Home", Icon: m.Icons.Home(), Build: buildHomePage},
	{Slug: "app-bars", Components: "Scaffold, AppBar, SliverAppBar, BottomAppBar, SnackBar, MaterialBanner", Name: "App bars & snack bars", Icon: m.Icons.WebAsset(), Build: buildAppBarsPage},
	{Slug: "navigation", Components: "NavigationBar, NavigationRail, NavigationDrawer, Drawer, TabBar, TabBarView, BottomSheet", Name: "Navigation & tabs", Icon: m.Icons.Explore(), Build: buildNavigationPage},
	{Slug: "buttons", Components: "ElevatedButton, FilledButton, OutlinedButton, TextButton, IconButton, FloatingActionButton, SegmentedButton", Name: "Buttons", Icon: m.Icons.TouchApp(), Build: buildButtonsPage},
	{Slug: "menus", Components: "MenuBar, MenuAnchor, PopupMenuButton, DropdownButton", Name: "Menus", Icon: m.Icons.Menu(), Build: buildMenusPage},
	{Slug: "text-fields", Components: "TextField, TextFormField, InputDecorator, DropdownMenu, SelectionArea", Name: "Text fields", Icon: m.Icons.EditNote(), Build: buildTextFieldsPage},
	{Slug: "selection", Components: "Checkbox, Radio, Switch, Slider, RangeSlider, Chip, FilterChip, InputChip, CircleAvatar", Name: "Selection & chips", Icon: m.Icons.CheckBox(), Build: buildSelectionPage},
	{Slug: "search", Components: "SearchBar, SearchAnchor, Autocomplete", Name: "Search", Icon: m.Icons.Search(), Build: buildSearchPage},
	{Slug: "dialogs", Components: "AlertDialog, SimpleDialog, Dialog.fullscreen, DatePicker, DateRangePicker, TimePicker", Name: "Dialogs & pickers", Icon: m.Icons.Event(), Build: buildDialogsPage},
	{Slug: "lists", Components: "Card, ListTile, ExpansionTile, ExpansionPanelList, Divider, Badge, Tooltip", Name: "Lists & cards", Icon: m.Icons.ViewList(), Build: buildListsPage},
	{Slug: "data", Components: "DataTable, PaginatedDataTable, progress indicators, RefreshIndicator, Scrollbar, Stepper, CarouselView", Name: "Data & progress", Icon: m.Icons.TableChart(), Build: buildDataPage},
	{Slug: "themes", Components: "The theme widget of each component, from CardTheme to TooltipTheme", Name: "Component themes", Icon: m.Icons.Palette(), Build: buildThemesPage},
	{Slug: "more", Components: "Adaptive forms, action buttons, Ink, RawChip, Dialog and BottomSheet in place, SliverAppBar", Name: "More components", Icon: m.Icons.Widgets(), Build: buildMorePage},
}

// columnWidth is the width of a section, for two of them side by side in the page.
const columnWidth = 392.0

// sectionWidth is the width of a section: columnWidth, or the width of a phone's screen less the
// margins when that is narrower.
func sectionWidth(bc m.BuildContext) float64 {
	return min(columnWidth, m.MediaQueryWidthOf(bc)-24-16)
}

// section is a titled part of a page, columnWidth wide (less on a narrow window).
func section(bc m.BuildContext, title string, child m.Widget) m.Widget {
	theme := m.ThemeOf(bc)
	return m.NewPadding(m.NewEdgeInsetsFromLTRB(24, 16, 0, 8), m.NewSizedBox(m.NewColumn([]m.Widget{
		m.NewPadding(m.NewEdgeInsetsFromLTRB(0, 0, 0, 12), m.NewText(title, m.TextOpts{Style: theme.TextTheme.TitleMedium})),
		child,
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart}), m.SizedBoxOpts{Width: m.Some(sectionWidth(bc))}))
}

// demoButton is a filled tonal button.
func demoButton(_ m.BuildContext, label string, onTap func()) m.Widget {
	return m.NewFilledButtonTonal(onTap, m.NewText(label))
}

// framed draws a thin outline around a demo.
func framed(bc m.BuildContext, child m.Widget) m.Widget {
	return m.NewContainer(child, m.ContainerOpts{
		Decoration: m.NewBoxDecoration(m.BoxDecorationOpts{
			Border:       m.NewBorderAll(m.BorderSideOpts{Color: m.Some(m.ThemeOf(bc).ColorScheme.OutlineVariant())}),
			BorderRadius: m.NewBorderRadiusCircular(12)}),
		ClipBehavior: m.ClipAntiAlias,
	})
}

func buildAppBarsPage(bc m.BuildContext) m.Widget {
	top := func(children ...m.Widget) m.Widget {
		return m.NewWrap(children)
	}
	return m.NewListView([]m.Widget{
		top(section(bc, "AppBar", NewAppBarDemos()), section(bc, "SliverAppBar: medium and large", NewSliverAppBarDemos())),
		top(section(bc, "SnackBar and MaterialBanner", NewMessageDemos()), section(bc, "BottomAppBar", NewBottomAppBarDemo())),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(24.0)}),
	})
}

// itoa is for the labels of the lists.
func itoa(i int) string { return strconv.Itoa(i) }
