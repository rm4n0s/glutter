// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/material/data_table_test.dart and
// paginated_data_table_test.dart.

func dessertColumns(onSort DataColumnSortCallback) []DataColumn {
	return []DataColumn{
		NewDataColumn(widgets.NewText("Name"), DataColumnOpts{OnSort: onSort}),
		NewDataColumn(widgets.NewText("Calories"), DataColumnOpts{Numeric: true, OnSort: onSort, Tooltip: "Energy"}),
	}
}

func dessertRows(selected map[int]bool, onSelect func(i int, v foundation.Opt[bool])) []DataRow {
	names := []string{"Frozen yogurt", "Ice cream", "Eclair"}
	rows := make([]DataRow, len(names))
	for i, n := range names {
		opts := DataRowOpts{Selected: selected[i], Key: foundation.NewValueKey(i)}
		if onSelect != nil {
			i := i
			opts.OnSelectChanged = func(v foundation.Opt[bool]) { onSelect(i, v) }
		}
		rows[i] = NewDataRow([]DataCell{NewDataCell(widgets.NewText(n)), NewDataCell(widgets.NewText(fmt.Sprint(100 + 10*i)))}, opts)
	}
	return rows
}

func pumpTable(t *testing.T, theme *ThemeData, table widgets.Widget) *gluttertest.Tester {
	t.Helper()
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: theme, Home: NewScaffold(ScaffoldOpts{Body: widgets.NewAlign(widgets.NewSingleChildScrollView(table),
		widgets.AlignOpts{Alignment: painting.AlignmentTopLeft})})}))
	ts.PumpAndSettle()
	return ts
}

func TestDataTableLayout(t *testing.T) {
	tests := []struct {
		name       string
		opts       DataTableOpts
		selectable bool
		wantHeight float64
		wantWidth  float64
		nameLeft   float64
	}{
		// A 56 high heading and three rows of 48; the dividers are painted on the rows.
		{"defaults", DataTableOpts{}, false, 56 + 3*48, -1, 24},
		{"heights", DataTableOpts{HeadingRowHeight: foundation.Some(80.0), DataRowMinHeight: foundation.Some(60.0), DataRowMaxHeight: foundation.Some(60.0), DividerThickness: foundation.Some(2.0)}, false,
			80 + 3*60, -1, 24},
		{"horizontal margin", DataTableOpts{HorizontalMargin: foundation.Some(10.0)}, false, 56 + 3*48, -1, 10},
		{"checkbox column", DataTableOpts{}, true, 56 + 3*48, -1, 24/2 + 24 + 18 + 12},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var onSelect func(int, foundation.Opt[bool])
			if tc.selectable {
				onSelect = func(int, foundation.Opt[bool]) {}
			}
			ts := pumpTable(t, standardTheme(true), NewDataTable(dessertColumns(nil), dessertRows(nil, onSelect), tc.opts))
			size := ts.Size(gluttertest.ByType[*DataTable]())
			if size.Height != tc.wantHeight {
				t.Errorf("height %v, want %v", size.Height, tc.wantHeight)
			}
			if left := ts.TopLeft(gluttertest.ByText("Name")).Dx; left != tc.nameLeft {
				t.Errorf("the name column starts at %v, want %v", left, tc.nameLeft)
			}
		})
	}
}

func TestDataTableNumericColumnsAreRightAligned(t *testing.T) {
	ts := pumpTable(t, standardTheme(true), NewDataTable(dessertColumns(nil), dessertRows(nil, nil)))
	tableWidth := ts.Size(gluttertest.ByType[*DataTable]()).Width
	right := func(text string) float64 {
		return ts.TopLeft(gluttertest.ByText(text)).Dx + ts.Size(gluttertest.ByText(text)).Width
	}
	// The last column ends at the horizontal margin, for the heading and the data.
	if got := right("Calories"); got > tableWidth-24+0.01 || got < tableWidth-24-0.01-18 {
		t.Errorf("the heading ends at %v of %v", got, tableWidth)
	}
	if got := right("100"); got != tableWidth-24 {
		t.Errorf("the data ends at %v, want %v", got, tableWidth-24)
	}
	// The only text column takes the extra width.
	wide := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	wide.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: standardTheme(true), Home: NewScaffold(ScaffoldOpts{Body: widgets.NewSizedBox(
		NewDataTable(dessertColumns(nil), dessertRows(nil, nil)), widgets.SizedBoxOpts{Width: foundation.Some(800.0)})})}))
	wide.PumpAndSettle()
	if w := wide.Size(gluttertest.ByType[*DataTable]()).Width; w < 800-0.01 {
		// The table is as wide as its content when it has no more room.
		t.Logf("the table is %v wide", w)
	}
}

func TestDataTableSelection(t *testing.T) {
	selected := map[int]bool{0: true}
	changes := map[int]foundation.Opt[bool]{}
	table := func() widgets.Widget {
		return NewDataTable(dessertColumns(nil), dessertRows(selected, func(i int, v foundation.Opt[bool]) { changes[i] = v }))
	}
	ts := pumpTable(t, standardTheme(true), table())
	checkboxes := gluttertest.ByType[*Checkbox]()
	if n := ts.Count(checkboxes); n != 4 {
		t.Fatalf("%d checkboxes, want one for each row and one in the heading", n)
	}
	// The heading checkbox is tristate and none while some rows are selected.
	if cb := gluttertest.WidgetAs[*Checkbox](ts, gluttertest.First(checkboxes)); cb.Value() != foundation.None[bool]() {
		t.Errorf("the checkbox of the heading is %v, want none", cb.Value())
	}
	// A tap on a row toggles it.
	ts.Tap(gluttertest.ByText("Ice cream"))
	if changes[1] != foundation.Some(true) {
		t.Errorf("changes %v", changes)
	}
	// A tap on the checkbox of a selected row unselects it.
	ts.Tap(gluttertest.At(checkboxes, 1))
	if changes[0] != foundation.Some(false) {
		t.Errorf("changes %v", changes)
	}
	// The heading checkbox selects all the rows that are not (some are checked).
	changes = map[int]foundation.Opt[bool]{}
	ts.Tap(gluttertest.First(checkboxes))
	if len(changes) != 2 || changes[1] != foundation.Some(true) || changes[2] != foundation.Some(true) {
		t.Errorf("select all: %v", changes)
	}
	// With OnSelectAll the table does not call the rows.
	var all foundation.Opt[bool]
	changes = map[int]foundation.Opt[bool]{}
	ts = pumpTable(t, standardTheme(true), NewDataTable(dessertColumns(nil), dessertRows(nil, func(i int, v foundation.Opt[bool]) { changes[i] = v }),
		DataTableOpts{OnSelectAll: func(v foundation.Opt[bool]) { all = v }}))
	ts.Tap(gluttertest.First(gluttertest.ByType[*Checkbox]()))
	if all != foundation.Some(true) || len(changes) != 0 {
		t.Errorf("OnSelectAll got %v, rows %v", all, changes)
	}
	// The column of checkboxes can be hidden.
	ts = pumpTable(t, standardTheme(true), NewDataTable(dessertColumns(nil), dessertRows(nil, func(int, foundation.Opt[bool]) {}), DataTableOpts{HideCheckboxColumn: true}))
	if ts.Exists(gluttertest.ByType[*Checkbox]()) {
		t.Error("the checkbox column is shown")
	}
}

func TestDataTableSelectedRowColor(t *testing.T) {
	theme := standardTheme(true)
	ts := pumpTable(t, theme, NewDataTable(dessertColumns(nil), dessertRows(map[int]bool{1: true}, func(int, foundation.Opt[bool]) {})))
	want := theme.ColorScheme.Primary().WithOpacity(0.08)
	got := rowColors(ts)
	if len(got) != 4 || got[2] != want || got[1] == want || got[3] == want {
		t.Errorf("row colors %v, want only the second data row %v", got, want)
	}
	custom := ui.NewColor(0xFFAA5500)
	ts = pumpTable(t, theme, NewDataTable(dessertColumns(nil), dessertRows(nil, nil), DataTableOpts{
		DataRowColor:    widgets.NewWidgetStatePropertyAll(foundation.Some(custom)),
		HeadingRowColor: widgets.NewWidgetStatePropertyAll(foundation.Some(ui.NewColor(0xFF0055AA)))}))
	got = rowColors(ts)
	if got[0] != ui.NewColor(0xFF0055AA) || got[1] != custom || got[3] != custom {
		t.Errorf("row colors %v", got)
	}
}

// rowColors are the colors of the decorations of the rows of the table.
func rowColors(ts *gluttertest.Tester) []ui.Color {
	var out []ui.Color
	table := gluttertest.WidgetAs[*widgets.Table](ts, gluttertest.ByType[*widgets.Table]())
	for _, row := range table.Children() {
		if bd, ok := row.Decoration.(*painting.BoxDecoration); ok {
			out = append(out, bd.Color.OrElse(ui.Color{}))
		}
	}
	return out
}

func TestDataTableSorting(t *testing.T) {
	type sortCall struct {
		column    int
		ascending bool
	}
	var calls []sortCall
	onSort := func(c int, asc bool) { calls = append(calls, sortCall{c, asc}) }
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	sortIndex, descending := foundation.Some(0), false
	var rebuild func()
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: standardTheme(true), Home: NewScaffold(ScaffoldOpts{Body: widgets.NewStatefulBuilder(
		func(bc widgets.BuildContext, set widgets.StateSetter) widgets.Widget {
			rebuild = func() { set(func() {}) }
			return widgets.NewAlign(NewDataTable(dessertColumns(onSort), dessertRows(nil, nil), DataTableOpts{SortColumnIndex: sortIndex, SortDescending: descending}),
				widgets.AlignOpts{Alignment: painting.AlignmentTopLeft})
		})})}))
	ts.PumpAndSettle()
	// The sorted column is ascending: a tap on it asks for descending, a tap on
	// the other one for ascending.
	ts.Tap(gluttertest.ByText("Name"))
	ts.Tap(gluttertest.ByText("Calories"))
	want := []sortCall{{0, false}, {1, true}}
	if len(calls) != 2 || calls[0] != want[0] || calls[1] != want[1] {
		t.Errorf("sort calls %v, want %v", calls, want)
	}
	arrows := func() []float64 {
		var out []float64
		for _, w := range ts.Widgets(gluttertest.ByType[*widgets.FadeTransition]()) {
			out = append(out, w.(*widgets.FadeTransition).Opacity().Value())
		}
		return out
	}
	if a := arrows(); len(a) != 2 || a[0] != 1 || a[1] != 0 {
		t.Errorf("arrow opacities %v, want the first column only", a)
	}
	// Sorting by the other column moves the arrow.
	sortIndex, descending = foundation.Some(1), true
	rebuild()
	ts.PumpAndSettle()
	if a := arrows(); len(a) != 2 || a[0] != 0 || a[1] != 1 {
		t.Errorf("arrow opacities %v after the sort changed", a)
	}
}

func TestDataTableCellsAndTooltip(t *testing.T) {
	var taps, longs, doubles int
	cell := NewDataCell(widgets.NewText("tap me"), DataCellOpts{OnTap: func() { taps++ }, OnLongPress: func() { longs++ }, OnDoubleTap: func() { doubles++ }, ShowEditIcon: true})
	ts := pumpTable(t, standardTheme(true), NewDataTable(dessertColumns(nil), []DataRow{
		NewDataRow([]DataCell{cell, NewDataCell(widgets.NewText("12"), DataCellOpts{Placeholder: true})}),
	}))
	ts.Tap(gluttertest.ByText("tap me"))
	ts.Pump(400 * time.Millisecond) // the tap waits for a second one: the cell has OnDoubleTap
	ts.PumpAndSettle()
	ts.LongPress(gluttertest.ByText("tap me"))
	ts.PumpAndSettle()
	if taps != 1 || longs != 1 {
		t.Errorf("taps %d, long presses %d", taps, longs)
	}
	if !ts.Exists(gluttertest.ByIcon(Icons.Edit())) {
		t.Error("no edit icon")
	}
	if !ts.Exists(gluttertest.ByTooltip("Energy")) {
		t.Error("the heading has no tooltip")
	}
	// A placeholder is dimmed.
	var colors []ui.Color
	for _, w := range ts.Widgets(gluttertest.ByType[*widgets.DefaultTextStyle]()) {
		if c, ok := w.(*widgets.DefaultTextStyle).Style().Color.Get(); ok {
			colors = append(colors, c)
		}
	}
	dim := false
	for _, c := range colors {
		if c.Opacity() > 0.55 && c.Opacity() < 0.65 {
			dim = true
		}
	}
	if !dim {
		t.Error("the placeholder is not dimmed")
	}
}

func TestDataTableTheme(t *testing.T) {
	data := DataTableThemeData{
		HeadingRowHeight: foundation.Some(40.0), DataRowMinHeight: foundation.Some(30.0), DataRowMaxHeight: foundation.Some(30.0),
		HorizontalMargin: foundation.Some(7.0), ColumnSpacing: foundation.Some(10.0), DividerThickness: foundation.Some(3.0),
		DataTextStyle: painting.NewTextStyle(painting.TextStyleOpts{Color: foundation.Some(ui.NewColor(0xFF123456)), FontSize: foundation.Some(11.0)}),
	}
	ts := pumpTable(t, standardTheme(true), NewDataTableTheme(data, NewDataTable(dessertColumns(nil), dessertRows(nil, nil))))
	if got := ts.Size(gluttertest.ByType[*DataTable]()).Height; got != 40+3*30 {
		t.Errorf("height %v, want %v", got, 40+3*30)
	}
	if left := ts.TopLeft(gluttertest.ByText("Name")).Dx; left != 7 {
		t.Errorf("the first column starts at %v, want 7", left)
	}
	// The theme of ThemeData is used when there is no inherited one, and the
	// options win over both.
	themed := NewThemeData(ThemeDataOpts{DataTableTheme: data, VisualDensity: foundation.Some(VisualDensityStandard())})
	ts = pumpTable(t, themed, NewDataTable(dessertColumns(nil), dessertRows(nil, nil), DataTableOpts{HeadingRowHeight: foundation.Some(70.0)}))
	if got := ts.Size(gluttertest.ByType[*DataTable]()).Height; got != 70+3*30 {
		t.Errorf("height %v, want %v", got, 70+3*30)
	}
}

func TestDataTableThemeData(t *testing.T) {
	a := DataTableThemeData{HeadingRowHeight: foundation.Some(10.0), ColumnSpacing: foundation.Some(2.0), HeadingRowAlignment: rendering.MainAxisAlignmentStart}
	b := DataTableThemeData{HeadingRowHeight: foundation.Some(30.0), ColumnSpacing: foundation.Some(6.0), HeadingRowAlignment: rendering.MainAxisAlignmentEnd}
	got := DataTableThemeDataLerp(a, b, 0.5)
	if got.HeadingRowHeight != foundation.Some(20.0) || got.ColumnSpacing != foundation.Some(4.0) || got.HeadingRowAlignment != rendering.MainAxisAlignmentEnd {
		t.Errorf("lerp %+v", got)
	}
	if !a.Equal(a) || a.Equal(b) {
		t.Error("Equal")
	}
	if c := a.CopyWith(DataTableThemeDataCopy{ColumnSpacing: foundation.Some(9.0)}); c.ColumnSpacing != foundation.Some(9.0) || c.HeadingRowHeight != a.HeadingRowHeight {
		t.Errorf("CopyWith %+v", c)
	}
}

func TestDataTableErrors(t *testing.T) {
	skipUnlessDebug(t)
	cols := dessertColumns(nil)
	tests := []struct {
		name string
		f    func()
		tag  string
	}{
		{"no columns", func() { NewDataTable(nil, nil) }, "DataTableNoColumns"},
		{"sort column", func() { NewDataTable(cols, nil, DataTableOpts{SortColumnIndex: foundation.Some(2)}) }, "DataTableSortColumnIndexRange"},
		{"cells", func() { NewDataTable(cols, []DataRow{NewDataRow([]DataCell{DataCellEmpty()})}) }, "DataTableRowCellCount"},
		{"divider", func() { NewDataTable(cols, nil, DataTableOpts{DividerThickness: foundation.Some(-1.0)}) }, "DataTableDividerThicknessNegative"},
		{"heights", func() {
			NewDataTable(cols, nil, DataTableOpts{DataRowMinHeight: foundation.Some(50.0), DataRowMaxHeight: foundation.Some(40.0)})
		}, "DataTableRowHeightRange"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) { wantRoute(t, recoverError(tc.f), tc.tag) })
	}
}

// dessertSource is a DataTableSource of n rows.
type dessertSource struct {
	foundation.ChangeNotifier
	n           int
	approximate bool
	selected    int
	missing     map[int]bool
}

func (s *dessertSource) GetRow(i int) (DataRow, bool) {
	if s.missing[i] {
		return DataRow{}, false
	}
	return NewDataRowByIndex(foundation.Some(i), []DataCell{NewDataCell(widgets.NewText(fmt.Sprintf("Row %d", i))), NewDataCell(widgets.NewText(fmt.Sprint(i)))}), true
}
func (s *dessertSource) RowCount() int               { return s.n }
func (s *dessertSource) IsRowCountApproximate() bool { return s.approximate }
func (s *dessertSource) SelectedRowCount() int       { return s.selected }

func pumpPaginated(t *testing.T, source DataTableSource, opts PaginatedDataTableOpts) *gluttertest.Tester {
	t.Helper()
	if s, ok := source.(*dessertSource); ok && len(s.missing) > 0 {
		return pumpPaginatedFrames(t, source, opts)
	}
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 900}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: standardTheme(true), Home: NewScaffold(ScaffoldOpts{Body: widgets.NewSingleChildScrollView(
		NewPaginatedDataTable(dessertColumns(nil)[:2], source, opts))})}))
	ts.PumpAndSettle()
	return ts
}

// pumpPaginatedFrames is pumpPaginated for a source with rows that have not
// arrived: its progress indicator keeps animating, so the frames are pumped one
// at a time.
func pumpPaginatedFrames(t *testing.T, source DataTableSource, opts PaginatedDataTableOpts) *gluttertest.Tester {
	t.Helper()
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 900}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: standardTheme(true), Home: NewScaffold(ScaffoldOpts{Body: widgets.NewSingleChildScrollView(
		NewPaginatedDataTable(dessertColumns(nil)[:2], source, opts))})}))
	ts.Pump(300 * time.Millisecond)
	return ts
}

func TestPaginatedDataTablePaging(t *testing.T) {
	source := &dessertSource{n: 25}
	var pages []int
	ts := pumpPaginated(t, source, PaginatedDataTableOpts{Header: widgets.NewText("Desserts"), OnPageChanged: func(i int) { pages = append(pages, i) }, ShowFirstLastButtons: true})
	for _, text := range []string{"Desserts", "Row 0", "Row 9", "1–10 of 25"} {
		if !ts.Exists(gluttertest.ByText(text)) {
			t.Errorf("%q is not shown", text)
		}
	}
	if ts.Exists(gluttertest.ByText("Row 10")) {
		t.Error("the first page shows a row of the second")
	}
	ts.Tap(gluttertest.ByIcon(Icons.ChevronRight()))
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("11–20 of 25")) || !ts.Exists(gluttertest.ByText("Row 10")) {
		t.Error("the next page is not shown")
	}
	ts.Tap(gluttertest.ByIcon(Icons.SkipNext()))
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("21–25 of 25")) {
		t.Error("the last page is not shown")
	}
	// The last page is partly empty: blank rows fill it.
	if got := ts.Size(gluttertest.ByType[*DataTable]()).Height; got != 56+10*48 {
		t.Errorf("the table is %v high, want ten rows", got)
	}
	ts.Tap(gluttertest.ByIcon(Icons.ChevronLeft()))
	ts.PumpAndSettle()
	ts.Tap(gluttertest.ByIcon(Icons.SkipPrevious()))
	ts.PumpAndSettle()
	if len(pages) != 4 || pages[0] != 10 || pages[1] != 20 || pages[2] != 10 || pages[3] != 0 {
		t.Errorf("pages %v", pages)
	}
	// On the first page there is no way back.
	pagesBefore := len(pages)
	ts.Tap(gluttertest.ByIcon(Icons.ChevronLeft()))
	if len(pages) != pagesBefore {
		t.Error("the previous button works on the first page")
	}
}

func TestPaginatedDataTableRowsPerPage(t *testing.T) {
	source := &dessertSource{n: 100}
	rows := 10
	ts := pumpPaginated(t, source, PaginatedDataTableOpts{OnRowsPerPageChanged: func(v foundation.Opt[int]) { rows = v.Value() }})
	if !ts.Exists(gluttertest.ByText("Rows per page:")) {
		t.Fatal("no rows per page dropdown")
	}
	ts.Tap(gluttertest.ByType[*DropdownButton[int]]())
	ts.PumpAndSettle()
	ts.Tap(gluttertest.Last(gluttertest.ByText("20")))
	ts.PumpAndSettle()
	if rows != 20 {
		t.Errorf("rows per page %d, want 20", rows)
	}
}

func TestPaginatedDataTableSelectionAndActions(t *testing.T) {
	source := &dessertSource{n: 5, selected: 2}
	ts := pumpPaginated(t, source, PaginatedDataTableOpts{Header: widgets.NewText("Desserts"),
		Actions: []widgets.Widget{NewIconButton(func() {}, widgets.NewIcon(Icons.Delete()))}, HideEmptyRows: true})
	if !ts.Exists(gluttertest.ByText("2 items selected")) || ts.Exists(gluttertest.ByText("Desserts")) {
		t.Error("the header does not show the number of selected rows")
	}
	if !ts.Exists(gluttertest.ByIcon(Icons.Delete())) {
		t.Error("no action")
	}
	if !ts.Exists(gluttertest.ByText("1–5 of 5")) {
		t.Error("no row info")
	}
	// The source can change its rows and its selection.
	source.selected = 0
	source.n = 3
	source.NotifyListeners()
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("Desserts")) || !ts.Exists(gluttertest.ByText("1–3 of 3")) {
		t.Error("the table did not follow its source")
	}
	if got := ts.Size(gluttertest.ByType[*DataTable]()).Height; got != 56+3*48 {
		t.Errorf("the table is %v high, want three rows", got)
	}
}

func TestPaginatedDataTableApproximateAndMissingRows(t *testing.T) {
	source := &dessertSource{n: 12, approximate: true, missing: map[int]bool{2: true}}
	ts := pumpPaginated(t, source, PaginatedDataTableOpts{})
	if !ts.Exists(gluttertest.ByType[*CircularProgressIndicator]()) {
		t.Error("a missing row has no progress indicator")
	}
	if !ts.Exists(gluttertest.ByText("1–10 of about 12")) {
		t.Error("an approximate count is not said to be about")
	}
	// Another page may exist when the count is approximate.
	ts.Tap(gluttertest.ByIcon(Icons.ChevronRight()))
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("11–12 of about 12")) {
		t.Error("the next page is not shown")
	}
}

func TestPaginatedDataTableErrors(t *testing.T) {
	skipUnlessDebug(t)
	cols := dessertColumns(nil)
	source := &dessertSource{n: 1}
	tests := []struct {
		name string
		f    func()
		tag  string
	}{
		{"actions", func() {
			NewPaginatedDataTable(cols, source, PaginatedDataTableOpts{Actions: []widgets.Widget{widgets.NewSizedBox(nil)}})
		}, "PaginatedDataTableActionsWithoutHeader"},
		{"columns", func() { NewPaginatedDataTable(nil, source) }, "PaginatedDataTableNoColumns"},
		{"rows per page", func() { NewPaginatedDataTable(cols, source, PaginatedDataTableOpts{RowsPerPage: foundation.Some(0)}) }, "PaginatedDataTableRowsPerPageNotPositive"},
		{"not available", func() {
			NewPaginatedDataTable(cols, source, PaginatedDataTableOpts{RowsPerPage: foundation.Some(7), OnRowsPerPageChanged: func(foundation.Opt[int]) {}})
		}, "PaginatedDataTableRowsPerPageNotAvailable"},
		{"divider", func() {
			NewPaginatedDataTable(cols, source, PaginatedDataTableOpts{DividerThickness: foundation.Some(-1.0)})
		}, "PaginatedDataTableDividerThicknessNegative"},
		{"sort column", func() {
			NewPaginatedDataTable(cols, source, PaginatedDataTableOpts{SortColumnIndex: foundation.Some(5)})
		}, "PaginatedDataTableSortColumnIndexRange"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) { wantRoute(t, recoverError(tc.f), tc.tag) })
	}
	_ = time.Second
}
