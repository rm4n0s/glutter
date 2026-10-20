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
	// Build makes the content of the page.
	Build func(bc m.BuildContext) m.Widget
}

// galleryPages are the pages, in the order of the navigation rail. Each step of
// the Material components adds its own.
var galleryPages = []Page{
	{Slug: "app-bars", Name: "App bars & snack bars", Icon: m.Icons.WebAsset(), Build: buildAppBarsPage},
	{Slug: "navigation", Name: "Navigation & tabs", Icon: m.Icons.Explore(), Build: buildNavigationPage},
}

// columnWidth is the width of a section, for two of them side by side in the page.
const columnWidth = 392.0

// section is a titled part of a page, columnWidth wide.
func section(bc m.BuildContext, title string, child m.Widget) m.Widget {
	theme := m.ThemeOf(bc)
	return m.NewPadding(m.NewEdgeInsetsFromLTRB(24, 16, 0, 8), m.NewSizedBox(m.NewColumn([]m.Widget{
		m.NewPadding(m.NewEdgeInsetsFromLTRB(0, 0, 0, 12), m.NewText(title, m.TextOpts{Style: theme.TextTheme.TitleMedium})),
		child,
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart}), m.SizedBoxOpts{Width: m.Some(columnWidth)}))
}

// demoButton is a button with the Material 3 look of a filled tonal button.
//
// TODO(step 13.3): the real buttons replace it.
func demoButton(bc m.BuildContext, label string, onTap func()) m.Widget {
	colors := m.ThemeOf(bc).ColorScheme
	shape := m.NewStadiumBorder()
	return m.NewMaterial(
		m.NewInkWell(
			m.NewSizedBox(
				m.NewPadding(m.NewEdgeInsetsSymmetric(0, 24), m.NewCenter(m.NewText(label, m.TextOpts{
					Style: m.ThemeOf(bc).TextTheme.LabelLarge.CopyWith(m.TextStyleCopy{Color: m.Some(colors.OnSecondaryContainer())})}),
					m.CenterOpts{WidthFactor: m.Some(1.0)})),
				m.SizedBoxOpts{Height: m.Some(40.0)}),
			m.InkWellOpts{OnTap: onTap, CustomBorder: shape}),
		m.MaterialOpts{Color: m.Some(colors.SecondaryContainer()), Shape: shape})
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
		return m.NewRow(children, m.RowOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
	}
	return m.NewListView([]m.Widget{
		top(section(bc, "AppBar", NewAppBarDemos()), section(bc, "SliverAppBar: medium and large", NewSliverAppBarDemos())),
		top(section(bc, "SnackBar and MaterialBanner", NewMessageDemos()), section(bc, "BottomAppBar", NewBottomAppBarDemo())),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(24.0)}),
	})
}

// itoa is for the labels of the lists.
func itoa(i int) string { return strconv.Itoa(i) }
