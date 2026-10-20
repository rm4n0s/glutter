package main

import (
	m "github.com/rm4n0s/glutter/material"
)

// buildHomePage is the first page: the choice of the look of the gallery, and an
// index of the components with the page of each group.
func buildHomePage(bc m.BuildContext) m.Widget {
	theme := m.ThemeOf(bc)
	scope := GalleryScopeOf(bc)
	cards := make([]m.Widget, 0, len(scope.pages)-1)
	for _, p := range scope.pages {
		if p.Slug == "home" {
			continue
		}
		cards = append(cards, NewIndexCard(p, func() { scope.onPage(p.Slug) }))
	}
	return m.NewListView([]m.Widget{
		m.NewPadding(m.NewEdgeInsetsFromLTRB(24, 24, 24, 0), m.NewColumn([]m.Widget{
			m.NewText("Material 3 in Glutter", m.TextOpts{Style: theme.TextTheme.HeadlineMedium}),
			m.NewPadding(m.NewEdgeInsetsFromLTRB(0, 8, 0, 0), m.NewText(
				"Every component of the Material library, on its own page. Choose a page below or in the list at the left.",
				m.TextOpts{Style: theme.TextTheme.BodyLarge})),
		}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})),
		section(bc, "Mode", NewModePicker()),
		section(bc, "Seed color", NewSeedPicker()),
		m.NewPadding(m.NewEdgeInsetsFromLTRB(24, 24, 24, 8), m.NewText("Components", m.TextOpts{Style: theme.TextTheme.TitleMedium})),
		m.NewPadding(m.NewEdgeInsetsFromLTRB(24, 0, 24, 24), m.NewWrap(cards, m.WrapOpts{Spacing: 16, RunSpacing: 16})),
	})
}

// IndexCard is a card of the index: the name and the components of a page. A tap
// opens the page.
type IndexCard struct {
	m.StatelessWidget
	page  Page
	onTap func()
}

// NewIndexCard creates the card for the page.
func NewIndexCard(page Page, onTap func()) *IndexCard { return &IndexCard{page: page, onTap: onTap} }

// Build builds the card.
func (w *IndexCard) Build(bc m.BuildContext) m.Widget {
	theme := m.ThemeOf(bc)
	return m.NewSizedBox(m.NewCardFilled(m.NewInkWell(m.NewPadding(m.NewEdgeInsetsAll(16), m.NewColumn([]m.Widget{
		m.NewRow([]m.Widget{
			m.NewIcon(w.page.Icon, m.IconOpts{Color: m.Some(theme.ColorScheme.Primary())}),
			m.NewSizedBox(nil, m.SizedBoxOpts{Width: m.Some(12.0)}),
			m.NewExpanded(m.NewText(w.page.Name, m.TextOpts{Style: theme.TextTheme.TitleMedium})),
		}),
		m.NewPadding(m.NewEdgeInsetsFromLTRB(0, 8, 0, 0), m.NewText(w.page.Components, m.TextOpts{Style: theme.TextTheme.BodyMedium})),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})), m.InkWellOpts{OnTap: w.onTap})), m.SizedBoxOpts{Width: m.Some(300.0)})
}

// ModePicker chooses between the light, the system and the dark theme.
type ModePicker struct{ m.StatelessWidget }

// NewModePicker creates the picker.
func NewModePicker() *ModePicker { return &ModePicker{} }

// Build builds the segmented button.
func (w *ModePicker) Build(bc m.BuildContext) m.Widget {
	scope := GalleryScopeOf(bc)
	return m.NewSegmentedButton([]m.ButtonSegment[m.ThemeMode]{
		{Value: m.ThemeModeLight, Label: m.NewText("Light"), Icon: m.NewIcon(m.Icons.LightMode())},
		{Value: m.ThemeModeSystem, Label: m.NewText("System"), Icon: m.NewIcon(m.Icons.BrightnessAuto())},
		{Value: m.ThemeModeDark, Label: m.NewText("Dark"), Icon: m.NewIcon(m.Icons.DarkMode())},
	}, []m.ThemeMode{scope.mode}, func(selected []m.ThemeMode) {
		if len(selected) > 0 {
			scope.onMode(selected[0])
		}
	})
}

// SeedPicker is a row of colors; the theme is made from the one that is chosen.
type SeedPicker struct{ m.StatelessWidget }

// NewSeedPicker creates the picker.
func NewSeedPicker() *SeedPicker { return &SeedPicker{} }

// Build builds the swatches.
func (w *SeedPicker) Build(bc m.BuildContext) m.Widget {
	scope := GalleryScopeOf(bc)
	swatches := make([]m.Widget, len(seedOptions))
	for i, option := range seedOptions {
		var mark m.Widget
		if i == scope.seed {
			mark = m.NewIcon(m.Icons.Check(), m.IconOpts{Color: m.Some(m.Colors.White)})
		}
		swatches[i] = m.NewTooltip(m.NewInk(m.NewInkWell(m.NewCenter(mark), m.InkWellOpts{OnTap: func() { scope.onSeed(i) }, CustomBorder: m.NewCircleBorder()}),
			m.InkOpts{Decoration: m.NewBoxDecoration(m.BoxDecorationOpts{Color: m.Some(option.color), Shape: m.BoxShapeCircle}), Width: m.Some(40.0), Height: m.Some(40.0)}),
			m.TooltipOpts{Message: option.name})
	}
	return m.NewWrap(swatches, m.WrapOpts{Spacing: 12, RunSpacing: 12})
}
