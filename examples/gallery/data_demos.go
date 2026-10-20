package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	m "github.com/rm4n0s/glutter/material"
)

// fullWidth is the width of a section that takes both columns.
const fullWidth = 2*columnWidth + 24

func buildDataPage(bc m.BuildContext) m.Widget {
	top := func(children ...m.Widget) m.Widget {
		return m.NewRow(children, m.RowOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
	}
	wide := func(title string, child m.Widget) m.Widget {
		theme := m.ThemeOf(bc)
		return m.NewPadding(m.NewEdgeInsetsFromLTRB(24, 16, 0, 8), m.NewSizedBox(m.NewColumn([]m.Widget{
			m.NewPadding(m.NewEdgeInsetsFromLTRB(0, 0, 0, 12), m.NewText(title, m.TextOpts{Style: theme.TextTheme.TitleMedium})),
			child,
		}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart}), m.SizedBoxOpts{Width: m.Some(fullWidth)}))
	}
	return m.NewListView([]m.Widget{
		wide("DataTable", NewDataTableDemo()),
		top(section(bc, "Progress indicators", NewProgressDemos()), section(bc, "RefreshIndicator and Scrollbar", NewRefreshScrollbarDemos())),
		top(section(bc, "Stepper", NewStepperDemo()), section(bc, "CarouselView", NewCarouselDemos())),
		wide("PaginatedDataTable", NewPaginatedDemo()),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(24.0)}),
	})
}

type dessert struct {
	name     string
	calories int
	fat      float64
	selected bool
}

func dessertList() []*dessert {
	return []*dessert{
		{name: "Frozen yogurt", calories: 159, fat: 6.0},
		{name: "Ice cream sandwich", calories: 237, fat: 9.0},
		{name: "Eclair", calories: 262, fat: 16.0},
		{name: "Cupcake", calories: 305, fat: 3.7},
		{name: "Gingerbread", calories: 356, fat: 16.0},
	}
}

// DataTableDemo is a table with sortable columns and selectable rows.
type DataTableDemo struct{ m.StatefulWidget }

// NewDataTableDemo creates the demo.
func NewDataTableDemo() *DataTableDemo { return &DataTableDemo{} }

// CreateState creates the state.
func (w *DataTableDemo) CreateState() m.State { return &dataTableDemoState{} }

type dataTableDemoState struct {
	m.StateOf[*DataTableDemo]
	desserts  []*dessert
	sortIndex int
	ascending bool
}

func (s *dataTableDemoState) InitState() {
	s.desserts = dessertList()
	s.sortIndex, s.ascending = -1, true
}

func (s *dataTableDemoState) sortBy(column int, ascending bool) {
	less := map[int]func(a, b *dessert) bool{
		0: func(a, b *dessert) bool { return a.name < b.name },
		1: func(a, b *dessert) bool { return a.calories < b.calories },
		2: func(a, b *dessert) bool { return a.fat < b.fat },
	}[column]
	s.SetState(func() {
		s.sortIndex, s.ascending = column, ascending
		sort.SliceStable(s.desserts, func(i, j int) bool {
			if ascending {
				return less(s.desserts[i], s.desserts[j])
			}
			return less(s.desserts[j], s.desserts[i])
		})
	})
}

// Build builds the table.
func (s *dataTableDemoState) Build(bc m.BuildContext) m.Widget {
	onSort := func(i int, ascending bool) { s.sortBy(i, ascending) }
	columns := []m.DataColumn{
		m.NewDataColumn(m.NewText("Dessert"), m.DataColumnOpts{OnSort: onSort}),
		m.NewDataColumn(m.NewText("Calories"), m.DataColumnOpts{Numeric: true, OnSort: onSort}),
		m.NewDataColumn(m.NewText("Fat (g)"), m.DataColumnOpts{Numeric: true, OnSort: onSort}),
	}
	rows := make([]m.DataRow, len(s.desserts))
	for i, d := range s.desserts {
		rows[i] = m.NewDataRow([]m.DataCell{
			m.NewDataCell(m.NewText(d.name)),
			m.NewDataCell(m.NewText(fmt.Sprint(d.calories))),
			m.NewDataCell(m.NewText(fmt.Sprintf("%.1f", d.fat))),
		}, m.DataRowOpts{Selected: d.selected, OnSelectChanged: func(v foundation.Opt[bool]) {
			s.SetState(func() { d.selected = v.OrElse(false) })
		}})
	}
	selected := 0
	for _, d := range s.desserts {
		if d.selected {
			selected++
		}
	}
	index := foundation.None[int]()
	if s.sortIndex >= 0 {
		index = foundation.Some(s.sortIndex)
	}
	return m.NewColumn([]m.Widget{
		framed(bc, m.NewDataTable(columns, rows, m.DataTableOpts{SortColumnIndex: index, SortDescending: !s.ascending,
			OnSelectAll: func(v foundation.Opt[bool]) {
				s.SetState(func() {
					for _, d := range s.desserts {
						d.selected = v.OrElse(false)
					}
				})
			}})),
		m.NewPadding(m.NewEdgeInsetsFromLTRB(0, 8, 0, 0), m.NewText(fmt.Sprintf("Selected rows: %d", selected))),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}

// ProgressDemos has linear and circular indicators, determinate and not.
type ProgressDemos struct{ m.StatefulWidget }

// NewProgressDemos creates the demo.
func NewProgressDemos() *ProgressDemos { return &ProgressDemos{} }

// CreateState creates the state.
func (w *ProgressDemos) CreateState() m.State { return &progressDemosState{} }

type progressDemosState struct {
	m.StateOf[*ProgressDemos]
	indeterminate bool
}

func (s *progressDemosState) Build(bc m.BuildContext) m.Widget {
	value := m.Some(0.6)
	if s.indeterminate {
		value = m.None[float64]()
	}
	gap := m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(16.0)})
	return m.NewColumn([]m.Widget{
		m.NewRow([]m.Widget{
			m.NewText("Indeterminate"),
			m.NewSwitch(s.indeterminate, func(v bool) { s.SetState(func() { s.indeterminate = v }) }),
		}, m.RowOpts{MainAxisAlignment: m.MainAxisAlignmentSpaceBetween, MainAxisSize: m.MainAxisSizeMax}),
		gap,
		m.NewLinearProgressIndicator(m.LinearProgressIndicatorOpts{Value: value}),
		gap,
		m.NewLinearProgressIndicator(m.LinearProgressIndicatorOpts{Value: value, Year2023: m.Some(false)}),
		gap,
		m.NewRow([]m.Widget{
			m.NewCircularProgressIndicator(m.CircularProgressIndicatorOpts{Value: value}),
			m.NewCircularProgressIndicator(m.CircularProgressIndicatorOpts{Value: value, Year2023: m.Some(false)}),
			m.NewRefreshProgressIndicator(m.RefreshProgressIndicatorOpts{Value: value}),
		}, m.RowOpts{MainAxisAlignment: m.MainAxisAlignmentSpaceAround, MainAxisSize: m.MainAxisSizeMax}),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStretch})
}

// RefreshScrollbarDemos has a list that refreshes when it is pulled down and a
// list with a scrollbar that is always shown.
type RefreshScrollbarDemos struct{ m.StatefulWidget }

// NewRefreshScrollbarDemos creates the demo.
func NewRefreshScrollbarDemos() *RefreshScrollbarDemos { return &RefreshScrollbarDemos{} }

// CreateState creates the state.
func (w *RefreshScrollbarDemos) CreateState() m.State { return &refreshScrollbarDemosState{} }

type refreshScrollbarDemosState struct {
	m.StateOf[*RefreshScrollbarDemos]
	refreshes int
}

func (s *refreshScrollbarDemosState) Build(bc m.BuildContext) m.Widget {
	items := func(prefix string, n int) []m.Widget {
		out := make([]m.Widget, n)
		for i := range out {
			out[i] = m.NewListTile(m.ListTileOpts{Title: m.NewText(fmt.Sprintf("%s %d", prefix, i+1))})
		}
		return out
	}
	refresh := m.NewRefreshIndicator(func() <-chan struct{} {
		s.SetState(func() { s.refreshes++ })
		done := make(chan struct{})
		time.AfterFunc(1500*time.Millisecond, func() { close(done) })
		return done
	}, m.NewListView(items("Pull to refresh", 6), m.ListViewOpts{Physics: m.NewAlwaysScrollableScrollPhysics(nil)}))
	controller := m.NewScrollController()
	scroll := m.NewScrollbar(m.NewListView(items("Scrolled", 12), m.ListViewOpts{Controller: controller}),
		m.ScrollbarOpts{Controller: controller, ThumbVisibility: m.Some(true)})
	return m.NewColumn([]m.Widget{
		m.NewText(fmt.Sprintf("Refreshes: %d", s.refreshes)),
		m.NewSizedBox(framed(bc, refresh), m.SizedBoxOpts{Height: m.Some(150.0)}),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(12.0)}),
		m.NewSizedBox(framed(bc, scroll), m.SizedBoxOpts{Height: m.Some(150.0)}),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStretch})
}

// StepperDemo is a vertical stepper with three steps.
type StepperDemo struct{ m.StatefulWidget }

// NewStepperDemo creates the demo.
func NewStepperDemo() *StepperDemo { return &StepperDemo{} }

// CreateState creates the state.
func (w *StepperDemo) CreateState() m.State { return &stepperDemoState{} }

type stepperDemoState struct {
	m.StateOf[*StepperDemo]
	current int
}

func (s *stepperDemoState) Build(bc m.BuildContext) m.Widget {
	step := func(i int, title, body string) m.Step {
		state := m.StepStateIndexed
		if i < s.current {
			state = m.StepStateComplete
		}
		return m.NewStep(m.NewText(title), m.NewAlign(m.NewText(body), m.AlignOpts{Alignment: m.AlignmentCenterLeft}),
			m.StepOpts{State: state, IsActive: i <= s.current})
	}
	steps := []m.Step{step(0, "Select a campaign", "Pick one of the campaigns."), step(1, "Create an ad group", "Name the group of ads."), step(2, "Create an ad", "Write the text of the ad.")}
	stepper := m.NewSizedBox(framed(bc, m.NewStepper(steps, m.StepperOpts{
		CurrentStep:  s.current,
		OnStepTapped: func(i int) { s.SetState(func() { s.current = i }) },
		OnStepContinue: func() {
			if s.current < len(steps)-1 {
				s.SetState(func() { s.current++ })
			}
		},
		OnStepCancel: func() {
			if s.current > 0 {
				s.SetState(func() { s.current-- })
			}
		},
	})), m.SizedBoxOpts{Height: m.Some(380.0)})
	return m.NewColumn([]m.Widget{
		m.NewPadding(m.NewEdgeInsetsFromLTRB(0, 0, 0, 8), m.NewText(fmt.Sprintf("Current step: %d of %d", s.current+1, len(steps)))),
		stepper,
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStretch})
}

// CarouselDemos has a carousel of items of a fixed extent and a weighted one.
type CarouselDemos struct{ m.StatelessWidget }

// NewCarouselDemos creates the demo.
func NewCarouselDemos() *CarouselDemos { return &CarouselDemos{} }

// Build builds the carousels.
func (w *CarouselDemos) Build(bc m.BuildContext) m.Widget {
	scheme := m.ThemeOf(bc).ColorScheme
	colors := []m.Color{scheme.PrimaryContainer(), scheme.SecondaryContainer(), scheme.TertiaryContainer(), scheme.ErrorContainer()}
	item := func(i int) m.Widget {
		return m.NewColoredBox(colors[i%len(colors)], m.NewCenter(m.NewText(fmt.Sprintf("Item %d", i+1), m.TextOpts{Style: m.ThemeOf(bc).TextTheme.TitleMedium, MaxLines: m.Some(1), Overflow: m.TextOverflowClip, SoftWrap: m.Some(false)})))
	}
	items := make([]m.Widget, 8)
	for i := range items {
		items[i] = item(i)
	}
	return m.NewColumn([]m.Widget{
		m.NewText("Fixed extent, with snapping"),
		m.NewSizedBox(m.NewCarouselView(150, items, m.CarouselViewOpts{ItemSnapping: true}), m.SizedBoxOpts{Height: m.Some(150.0)}),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(16.0)}),
		m.NewText("Weighted: 1, 6, 1"),
		m.NewSizedBox(m.NewCarouselViewWeighted([]int{1, 6, 1}, items, m.CarouselViewOpts{ItemSnapping: true}), m.SizedBoxOpts{Height: m.Some(150.0)}),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStretch})
}

// pagedSource is the rows of a PaginatedDataTable: n numbered desserts.
type pagedSource struct {
	foundation.ChangeNotifier
	n int
}

func (s *pagedSource) GetRow(i int) (m.DataRow, bool) {
	return m.NewDataRowByIndex(m.Some(i), []m.DataCell{
		m.NewDataCell(m.NewText(fmt.Sprintf("Dessert %d", i+1))),
		m.NewDataCell(m.NewText(fmt.Sprint(100 + 17*i%300))),
	}), true
}
func (s *pagedSource) RowCount() int               { return s.n }
func (s *pagedSource) IsRowCountApproximate() bool { return false }
func (s *pagedSource) SelectedRowCount() int       { return 0 }

// PaginatedDemo is a table with pages of five rows.
type PaginatedDemo struct{ m.StatelessWidget }

// NewPaginatedDemo creates the demo.
func NewPaginatedDemo() *PaginatedDemo { return &PaginatedDemo{} }

// Build builds the table.
func (w *PaginatedDemo) Build(bc m.BuildContext) m.Widget {
	columns := []m.DataColumn{
		m.NewDataColumn(m.NewText("Dessert")),
		m.NewDataColumn(m.NewText("Calories"), m.DataColumnOpts{Numeric: true}),
	}
	return m.NewPaginatedDataTable(columns, &pagedSource{n: 23}, m.PaginatedDataTableOpts{
		Header: m.NewText("Desserts"), RowsPerPage: m.Some(5), AvailableRowsPerPage: []int{5, 10}, HideCheckboxColumn: true,
		OnRowsPerPageChanged: func(m.Opt[int]) {}})
}
