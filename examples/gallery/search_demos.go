package main

import (
	"os"
	"strings"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/services"
)

var countries = []string{
	"Argentina", "Australia", "Austria", "Belgium", "Brazil", "Canada", "Chile", "Denmark", "Egypt", "Finland", "France",
	"Germany", "Ghana", "Greece", "Grenada", "Guatemala", "Guinea", "India", "Ireland", "Italy", "Japan", "Kenya", "Mexico",
	"Netherlands", "Norway", "Peru", "Poland", "Portugal", "Spain", "Sweden", "Switzerland", "Turkey",
}

// matchingCountries are the countries that have the query in their name, ignoring
// the case; they are all of them for an empty query.
func matchingCountries(query string) []string {
	var out []string
	for _, c := range countries {
		if strings.Contains(strings.ToLower(c), strings.ToLower(query)) {
			out = append(out, c)
		}
	}
	return out
}

func buildSearchPage(bc m.BuildContext) m.Widget {
	top := func(children ...m.Widget) m.Widget {
		return m.NewRow(children, m.RowOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
	}
	return m.NewListView([]m.Widget{
		top(section(bc, "SearchAnchor.bar: a search bar that opens a view", NewCountrySearchDemo()), section(bc, "SearchBar", NewSearchBarDemo())),
		top(section(bc, "Autocomplete", NewAutocompleteDemo())),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(300.0)}),
	})
}

// devSearchDelay is how long GLUTTER_DEV_QUERY waits before it opens the search view.
const devSearchDelay = 400 * time.Millisecond

// devQuery is the text that GLUTTER_DEV_QUERY asks to search for, in a debug
// build: the search view of the Search page opens with it.
func devQuery(getenv func(string) string) string {
	if !foundation.DebugMode {
		return ""
	}
	return getenv("GLUTTER_DEV_QUERY")
}

// CountrySearchDemo is a search bar that opens a view with the countries that
// match the text; choosing one closes the view and puts it in the bar.
type CountrySearchDemo struct{ m.StatefulWidget }

// NewCountrySearchDemo creates the demo.
func NewCountrySearchDemo() *CountrySearchDemo { return &CountrySearchDemo{} }

// CreateState creates the state.
func (w *CountrySearchDemo) CreateState() m.State { return &countrySearchDemoState{} }

type countrySearchDemoState struct {
	m.StateOf[*CountrySearchDemo]
	controller *m.SearchController
}

// InitState opens the view when GLUTTER_DEV_QUERY asks for it.
func (s *countrySearchDemoState) InitState() {
	s.controller = m.NewSearchController()
	query := devQuery(os.Getenv)
	if query == "" {
		return
	}
	// A view that is not full screen closes when the size of the window changes,
	// and a window gets its size after the first frame: it opens a little later.
	m.WidgetsBindingOf(s.Context()).Gestures().NewTimer(devSearchDelay, func() {
		if s.Mounted() {
			s.controller.SetText(query)
			s.controller.OpenView()
		}
	})
}

func (s *countrySearchDemoState) Dispose() { s.controller.Dispose() }

func (s *countrySearchDemoState) Build(bc m.BuildContext) m.Widget {
	return m.NewColumn([]m.Widget{
		m.NewSizedBox(m.NewSearchAnchorBar(func(bc m.BuildContext, c *m.SearchController, deliver func([]m.Widget)) {
			var tiles []m.Widget
			for _, country := range matchingCountries(c.Text()) {
				country := country
				tiles = append(tiles, m.NewInkWell(m.NewPadding(m.NewEdgeInsetsSymmetric(12, 16), m.NewRow([]m.Widget{
					m.NewPadding(m.NewEdgeInsetsFromLTRB(0, 0, 16, 0), m.NewIcon(m.Icons.Public())), m.NewText(country)})),
					m.InkWellOpts{OnTap: func() { c.CloseView(m.Some(country)) }}))
			}
			deliver(tiles)
		}, m.SearchAnchorBarOpts{SearchController: s.controller, BarHintText: m.Some("Search countries")}), m.SizedBoxOpts{Width: m.Some(columnWidth - 24)}),
		m.NewPadding(m.NewEdgeInsetsFromLTRB(0, 12, 0, 0), m.NewText("Choose a country from the view")),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}

// SearchBarDemo has search bars on their own: with a leading icon and trailing
// buttons, one that is disabled, and one that has the text selected.
type SearchBarDemo struct{ m.StatefulWidget }

// NewSearchBarDemo creates the demo.
func NewSearchBarDemo() *SearchBarDemo { return &SearchBarDemo{} }

// CreateState creates the state.
func (w *SearchBarDemo) CreateState() m.State { return &searchBarDemoState{} }

type searchBarDemoState struct {
	m.StateOf[*SearchBarDemo]
	submitted string
}

func (s *searchBarDemoState) Build(bc m.BuildContext) m.Widget {
	bar := func(opts m.SearchBarOpts) m.Widget {
		return m.NewPadding(m.NewEdgeInsetsFromLTRB(0, 0, 0, 12), m.NewSizedBox(m.NewSearchBar(opts), m.SizedBoxOpts{Width: m.Some(columnWidth - 24)}))
	}
	submitted := "Nothing submitted"
	if s.submitted != "" {
		submitted = "Submitted: " + s.submitted
	}
	return m.NewColumn([]m.Widget{
		bar(m.SearchBarOpts{HintText: m.Some("Search"), Leading: m.NewIcon(m.Icons.Search()),
			Trailing:    []m.Widget{m.NewIconButton(func() {}, m.NewIcon(m.Icons.Mic()), m.IconButtonOpts{Tooltip: "Voice search"})},
			OnSubmitted: func(text string) { s.SetState(func() { s.submitted = text }) }}),
		bar(m.SearchBarOpts{HintText: m.Some("Disabled"), Leading: m.NewIcon(m.Icons.Search()), Disabled: true}),
		m.NewText(submitted),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}

// AutocompleteDemo is a text field that suggests countries as the user types.
type AutocompleteDemo struct{ m.StatefulWidget }

// NewAutocompleteDemo creates the demo.
func NewAutocompleteDemo() *AutocompleteDemo { return &AutocompleteDemo{} }

// CreateState creates the state.
func (w *AutocompleteDemo) CreateState() m.State { return &autocompleteDemoState{} }

type autocompleteDemoState struct {
	m.StateOf[*AutocompleteDemo]
	selected string
}

func (s *autocompleteDemoState) Build(bc m.BuildContext) m.Widget {
	selected := "No country selected"
	if s.selected != "" {
		selected = "Selected: " + s.selected
	}
	return m.NewColumn([]m.Widget{
		m.NewSizedBox(m.NewAutocomplete(func(value services.TextEditingValue, deliver func([]string)) {
			if value.Text == "" {
				deliver(nil)
				return
			}
			deliver(matchingCountries(value.Text))
		}, m.AutocompleteOpts[string]{
			OnSelected: func(c string) { s.SetState(func() { s.selected = c }) },
			FieldViewBuilder: func(_ m.BuildContext, controller *m.TextEditingController, focusNode *m.FocusNode, onSubmitted foundation.VoidCallback) m.Widget {
				return m.NewTextFormField(m.TextFormFieldOpts{Controller: controller, FocusNode: focusNode, OnFieldSubmitted: func(string) { onSubmitted() },
					Decoration: &m.InputDecoration{LabelText: m.Some("Country"), HintText: m.Some("Type to see suggestions"), Border: m.NewOutlineInputBorder()}})
			}}), m.SizedBoxOpts{Width: m.Some(columnWidth - 24)}),
		m.NewPadding(m.NewEdgeInsetsFromLTRB(0, 12, 0, 0), m.NewText(selected)),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}
