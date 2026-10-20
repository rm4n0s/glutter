// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"fmt"
	"math"
	"slices"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// PaginatedDataTableDefaultRowsPerPage is the number of rows shown on a page by
// default.
//
// Flutter: material/paginated_data_table.dart PaginatedDataTable.defaultRowsPerPage
const PaginatedDataTableDefaultRowsPerPage = 10

// DataTableSource is the data model of a PaginatedDataTable: it gives the rows
// on request and notifies its listeners when the rows change.
//
// Flutter: material/data_table_source.dart DataTableSource
type DataTableSource interface {
	foundation.Listenable
	// GetRow returns the row at the index, or false when the data of the row has
	// not arrived yet (the table shows a progress indicator in its place).
	//
	// Flutter: material/data_table_source.dart DataTableSource.getRow
	GetRow(index int) (DataRow, bool)
	// RowCount is the number of rows, or the number of rows known so far when
	// IsRowCountApproximate.
	//
	// Flutter: material/data_table_source.dart DataTableSource.rowCount
	RowCount() int
	// IsRowCountApproximate is whether there are more rows than RowCount says.
	//
	// Flutter: material/data_table_source.dart DataTableSource.isRowCountApproximate
	IsRowCountApproximate() bool
	// SelectedRowCount is the number of rows that are selected.
	//
	// Flutter: material/data_table_source.dart DataTableSource.selectedRowCount
	SelectedRowCount() int
}

// PaginatedDataTableOpts are the optional parameters of NewPaginatedDataTable.
//
// Flutter: material/paginated_data_table.dart PaginatedDataTable
type PaginatedDataTableOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Header is the title of the table; it needs to be given for Actions.
	Header widgets.Widget
	// Actions are the widgets after the header, such as icon buttons.
	Actions []widgets.Widget
	// SortColumnIndex is the column that the rows are sorted by.
	SortColumnIndex foundation.Opt[int]
	// SortDescending says that the sort is descending.
	SortDescending bool
	// OnSelectAll is called when the user selects or unselects all the rows.
	OnSelectAll func(selected foundation.Opt[bool])
	// DataRowMinHeight is the minimum height of a data row.
	DataRowMinHeight foundation.Opt[float64]
	// DataRowMaxHeight is the maximum height of a data row.
	DataRowMaxHeight foundation.Opt[float64]
	// HeadingRowHeight is the height of the heading row; the default is 56.
	HeadingRowHeight foundation.Opt[float64]
	// HorizontalMargin is the margin at the two sides of the table; the default
	// is 24.
	HorizontalMargin foundation.Opt[float64]
	// ColumnSpacing is the space between the columns; the default is 56.
	ColumnSpacing foundation.Opt[float64]
	// HideCheckboxColumn hides the checkboxes of the selectable rows.
	HideCheckboxColumn bool
	// ShowFirstLastButtons adds the buttons that go to the first and to the last
	// page.
	ShowFirstLastButtons bool
	// InitialFirstRowIndex is the index of the first row on the first page; the
	// default is 0.
	InitialFirstRowIndex foundation.Opt[int]
	// OnPageChanged is called with the index of the first row of the new page.
	OnPageChanged func(firstRowIndex int)
	// RowsPerPage is the number of rows on a page; the default is
	// PaginatedDataTableDefaultRowsPerPage.
	RowsPerPage foundation.Opt[int]
	// AvailableRowsPerPage are the choices of the "rows per page" dropdown; the
	// default is 10, 20, 50 and 100.
	AvailableRowsPerPage []int
	// OnRowsPerPageChanged is called when the user picks another number of rows per
	// page; the dropdown is shown when it is given.
	OnRowsPerPageChanged func(rowsPerPage foundation.Opt[int])
	// DragStartBehavior is where a drag of the horizontal scroll starts.
	DragStartBehavior gestures.DragStartBehavior
	// ArrowHeadColor is the color of the buttons of the footer.
	ArrowHeadColor foundation.Opt[ui.Color]
	// CheckboxHorizontalMargin is the horizontal margin around the checkboxes.
	CheckboxHorizontalMargin foundation.Opt[float64]
	// Controller is the controller of the horizontal scroll.
	Controller widgets.ScrollController
	// Primary is whether the horizontal scroll uses the PrimaryScrollController.
	Primary foundation.Opt[bool]
	// HeadingRowColor is the color of the heading row, for each state.
	HeadingRowColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// DividerThickness is the thickness of the lines between the rows.
	DividerThickness foundation.Opt[float64]
	// HideEmptyRows hides the blank rows that fill the last page.
	HideEmptyRows bool
}

// PaginatedDataTable is a table of data that is shown a page at a time, with a
// header, actions and a footer that moves between the pages.
//
// Flutter: material/paginated_data_table.dart PaginatedDataTable
type PaginatedDataTable struct {
	widgets.StatefulWidget
	columns []DataColumn
	source  DataTableSource
	o       PaginatedDataTableOpts
}

// NewPaginatedDataTable creates a paginated table of the columns that gets its
// rows from the source. In debug builds it panics with the tags
// PaginatedDataTableActionsWithoutHeader, PaginatedDataTableNoColumns,
// PaginatedDataTableSortColumnIndexRange, PaginatedDataTableRowHeightRange,
// PaginatedDataTableRowsPerPageNotPositive, PaginatedDataTableDividerThicknessNegative,
// PaginatedDataTableRowsPerPageNotAvailable and
// PaginatedDataTablePrimaryAndController for the assertions of Flutter.
//
// Flutter: material/paginated_data_table.dart PaginatedDataTable.new
func NewPaginatedDataTable(columns []DataColumn, source DataTableSource, opts ...PaginatedDataTableOpts) *PaginatedDataTable {
	o := firstOpts(opts)
	if o.AvailableRowsPerPage == nil {
		d := PaginatedDataTableDefaultRowsPerPage
		o.AvailableRowsPerPage = []int{d, d * 2, d * 5, d * 10}
	}
	if foundation.DebugMode {
		rows := o.RowsPerPage.OrElse(PaginatedDataTableDefaultRowsPerPage)
		switch {
		case len(o.Actions) > 0 && o.Header == nil:
			panic(errors.New("PaginatedDataTableActionsWithoutHeader", "the actions of a PaginatedDataTable need a header"))
		case len(columns) == 0:
			panic(errors.New("PaginatedDataTableNoColumns", "a PaginatedDataTable needs at least one column"))
		case o.SortColumnIndex.IsSome() && (o.SortColumnIndex.Value() < 0 || o.SortColumnIndex.Value() >= len(columns)):
			panic(errors.New("PaginatedDataTableSortColumnIndexRange", "the sortColumnIndex of a PaginatedDataTable must be a column", "sortColumnIndex", o.SortColumnIndex.Value()))
		case o.DataRowMinHeight.IsSome() && o.DataRowMaxHeight.IsSome() && o.DataRowMaxHeight.Value() < o.DataRowMinHeight.Value():
			panic(errors.New("PaginatedDataTableRowHeightRange", "the dataRowMaxHeight of a PaginatedDataTable must not be less than its dataRowMinHeight"))
		case rows <= 0:
			panic(errors.New("PaginatedDataTableRowsPerPageNotPositive", "the rowsPerPage of a PaginatedDataTable must be greater than zero", "rowsPerPage", rows))
		case o.DividerThickness.IsSome() && o.DividerThickness.Value() < 0:
			panic(errors.New("PaginatedDataTableDividerThicknessNegative", "the dividerThickness of a PaginatedDataTable must not be negative"))
		case o.OnRowsPerPageChanged != nil && !slices.Contains(o.AvailableRowsPerPage, rows):
			panic(errors.New("PaginatedDataTableRowsPerPageNotAvailable", "the rowsPerPage of a PaginatedDataTable must be one of its availableRowsPerPage", "rowsPerPage", rows))
		case o.Controller != nil && o.Primary.OrElse(false):
			panic(errors.New("PaginatedDataTablePrimaryAndController", "Primary ScrollViews obtain their ScrollController via inheritance from a PrimaryScrollController widget. You cannot both set primary to true and pass an explicit controller."))
		}
	}
	return &PaginatedDataTable{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), columns: columns, source: source, o: o}
}

func (w *PaginatedDataTable) rowsPerPage() int {
	return w.o.RowsPerPage.OrElse(PaginatedDataTableDefaultRowsPerPage)
}

// CreateState creates the state.
//
// Flutter: material/paginated_data_table.dart PaginatedDataTable.createState
func (w *PaginatedDataTable) CreateState() widgets.State {
	return &PaginatedDataTableState{rows: map[int]*DataRow{}, tableKey: widgets.NewGlobalKey[widgets.State]()}
}

// PaginatedDataTableState is the state of a PaginatedDataTable; PageTo moves it
// to the page of a row.
//
// Flutter: material/paginated_data_table.dart PaginatedDataTableState
type PaginatedDataTableState struct {
	widgets.StateOf[*PaginatedDataTable]
	firstRowIndex       int
	rowCount            int
	rowCountApproximate bool
	selectedRowCount    int
	rows                map[int]*DataRow
	listener            foundation.ListenerHandle
	tableKey            *widgets.GlobalKey[widgets.State]
}

// InitState reads the first row from the page storage and listens to the source.
//
// Flutter: material/paginated_data_table.dart PaginatedDataTableState.initState
func (s *PaginatedDataTableState) InitState() {
	s.firstRowIndex = s.Widget().o.InitialFirstRowIndex.OrElse(0)
	if bucket := widgets.MaybePageStorageOf(s.Context()); bucket != nil {
		if saved, ok := bucket.ReadState(s.Context()).(int); ok {
			s.firstRowIndex = saved
		}
	}
	s.listener = s.Widget().source.AddListener(s.handleDataSourceChanged)
	s.updateCaches()
}

// DidUpdateWidget listens to the new source.
//
// Flutter: material/paginated_data_table.dart PaginatedDataTableState.didUpdateWidget
func (s *PaginatedDataTableState) DidUpdateWidget(old *PaginatedDataTable) {
	if old.source != s.Widget().source {
		old.source.RemoveListener(s.listener)
		s.listener = s.Widget().source.AddListener(s.handleDataSourceChanged)
		s.updateCaches()
	}
}

// Dispose stops listening to the source.
//
// Flutter: material/paginated_data_table.dart PaginatedDataTableState.dispose
func (s *PaginatedDataTableState) Dispose() { s.Widget().source.RemoveListener(s.listener) }

func (s *PaginatedDataTableState) handleDataSourceChanged() {
	s.SetState(s.updateCaches)
}

func (s *PaginatedDataTableState) updateCaches() {
	source := s.Widget().source
	s.rowCount = source.RowCount()
	s.rowCountApproximate = source.IsRowCountApproximate()
	s.selectedRowCount = source.SelectedRowCount()
	clear(s.rows)
}

// PageTo shows the page that has the row, and calls onPageChanged when the page
// changed.
//
// Flutter: material/paginated_data_table.dart PaginatedDataTableState.pageTo
func (s *PaginatedDataTableState) PageTo(rowIndex int) {
	old := s.firstRowIndex
	s.SetState(func() {
		rowsPerPage := s.Widget().rowsPerPage()
		s.firstRowIndex = (rowIndex / rowsPerPage) * rowsPerPage
	})
	if f := s.Widget().o.OnPageChanged; f != nil && old != s.firstRowIndex {
		f(s.firstRowIndex)
	}
}

func (s *PaginatedDataTableState) blankRowFor(index int) DataRow {
	cells := make([]DataCell, len(s.Widget().columns))
	for i := range cells {
		cells[i] = DataCellEmpty()
	}
	return NewDataRowByIndex(foundation.Some(index), cells)
}

func (s *PaginatedDataTableState) progressIndicatorRowFor(index int) DataRow {
	columns := s.Widget().columns
	cells := make([]DataCell, len(columns))
	have := false
	for i, c := range columns {
		if !c.Numeric() {
			have = true
			cells[i] = NewDataCell(NewCircularProgressIndicator())
		} else {
			cells[i] = DataCellEmpty()
		}
	}
	if !have {
		cells[0] = NewDataCell(NewCircularProgressIndicator())
	}
	return NewDataRowByIndex(foundation.Some(index), cells)
}

func (s *PaginatedDataTableState) getRows(firstRowIndex, rowsPerPage int) []DataRow {
	var result []DataRow
	next := firstRowIndex + rowsPerPage
	haveProgressIndicator := false
	for index := firstRowIndex; index < next; index++ {
		var row *DataRow
		if index < s.rowCount || s.rowCountApproximate {
			cached, ok := s.rows[index]
			if !ok {
				if r, found := s.Widget().source.GetRow(index); found {
					cached = &r
				}
				s.rows[index] = cached
			}
			row = cached
			if row == nil && !haveProgressIndicator {
				r := s.progressIndicatorRowFor(index)
				row = &r
				haveProgressIndicator = true
			}
		}
		if !s.Widget().o.HideEmptyRows && row == nil {
			r := s.blankRowFor(index)
			row = &r
		}
		if row != nil {
			result = append(result, *row)
		}
	}
	return result
}

func (s *PaginatedDataTableState) isNextPageUnavailable() bool {
	return !s.rowCountApproximate && s.firstRowIndex+s.Widget().rowsPerPage() >= s.rowCount
}

// Build builds the card with the header, the table and the footer.
//
// Flutter: material/paginated_data_table.dart PaginatedDataTableState.build
func (s *PaginatedDataTableState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	themeData := ThemeOf(bc)
	localizations := MaterialLocalizationsOf(bc)
	rowsPerPage := w.rowsPerPage()

	var headerWidgets []widgets.Widget
	if s.selectedRowCount == 0 && w.o.Header != nil {
		headerWidgets = append(headerWidgets, widgets.NewExpanded(w.o.Header))
	} else if w.o.Header != nil {
		headerWidgets = append(headerWidgets, widgets.NewExpanded(widgets.NewText(localizations.SelectedRowCountTitle(s.selectedRowCount))))
	}
	for _, action := range w.o.Actions {
		headerWidgets = append(headerWidgets, widgets.NewPadding(painting.NewEdgeInsetsDirectionalFromSTEB(24.0-8.0*2.0, 0, 0, 0), action))
	}

	footerTextStyle := themeData.TextTheme.BodySmall
	var footerWidgets []widgets.Widget
	if w.o.OnRowsPerPageChanged != nil {
		var items []*DropdownMenuItem[int]
		for _, value := range w.o.AvailableRowsPerPage {
			if value <= s.rowCount || value == rowsPerPage {
				items = append(items, NewDropdownMenuItem(widgets.NewText(fmt.Sprint(value)), DropdownMenuItemOpts[int]{Value: foundation.Some(value)}))
			}
		}
		footerWidgets = append(footerWidgets,
			widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(14.0)}),
			widgets.NewText(localizations.RowsPerPageTitle()),
			widgets.NewConstrainedBox(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MinWidth: foundation.Some(64.0)}), // 40.0 for the text, 24.0 for the icon
				widgets.NewAlign(NewDropdownButtonHideUnderline(NewDropdownButton(items, w.o.OnRowsPerPageChanged,
					DropdownButtonOpts[int]{Value: foundation.Some(rowsPerPage), Style: footerTextStyle})),
					widgets.AlignOpts{Alignment: painting.AlignmentDirectionalCenterEnd})))
	}
	pageButton := func(icon IconData, tooltip string, onPressed func()) widgets.Widget {
		return NewIconButton(onPressed, widgets.NewIcon(icon), IconButtonOpts{Padding: painting.EdgeInsetsZero, Color: w.o.ArrowHeadColor, Tooltip: tooltip})
	}
	var onFirst, onPrevious, onNext, onLast func()
	if s.firstRowIndex > 0 {
		onFirst = func() { s.PageTo(0) }
		onPrevious = func() { s.PageTo(max(s.firstRowIndex-rowsPerPage, 0)) }
	}
	if !s.isNextPageUnavailable() {
		onNext = func() { s.PageTo(s.firstRowIndex + rowsPerPage) }
		onLast = func() { s.PageTo(int(math.Floor(float64(s.rowCount-1)/float64(rowsPerPage))) * rowsPerPage) }
	}
	footerWidgets = append(footerWidgets,
		widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(32.0)}),
		widgets.NewText(localizations.PageRowsInfoTitle(s.firstRowIndex+1, min(s.firstRowIndex+rowsPerPage, s.rowCount), s.rowCount, s.rowCountApproximate)),
		widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(32.0)}))
	if w.o.ShowFirstLastButtons {
		footerWidgets = append(footerWidgets, pageButton(Icons.SkipPrevious(), localizations.FirstPageTooltip(), onFirst))
	}
	footerWidgets = append(footerWidgets,
		pageButton(Icons.ChevronLeft(), localizations.PreviousPageTooltip(), onPrevious),
		widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(24.0)}),
		pageButton(Icons.ChevronRight(), localizations.NextPageTooltip(), onNext))
	if w.o.ShowFirstLastButtons {
		footerWidgets = append(footerWidgets, pageButton(Icons.SkipNext(), localizations.LastPageTooltip(), onLast))
	}
	footerWidgets = append(footerWidgets, widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(14.0)}))

	return NewCard(widgets.NewLayoutBuilder(func(bc widgets.BuildContext, constraints rendering.BoxConstraints) widgets.Widget {
		children := []widgets.Widget{}
		if len(headerWidgets) > 0 {
			style := themeData.TextTheme.TitleLarge.CopyWith(painting.TextStyleCopy{FontWeight: ui.FontWeightW400})
			var color foundation.Opt[ui.Color]
			if s.selectedRowCount > 0 {
				style = themeData.TextTheme.TitleMedium.CopyWith(painting.TextStyleCopy{Color: foundation.Some(themeData.ColorScheme.Secondary())})
				color = foundation.Some(themeData.SecondaryHeaderColor)
			}
			children = append(children, widgets.NewSemantics(widgets.NewDefaultTextStyle(style,
				widgets.NewIconThemeMerge(widgets.NewIconThemeData(widgets.IconThemeDataOpts{Opacity: foundation.Some(0.54)}),
					NewInk(widgets.NewPadding(painting.NewEdgeInsetsDirectionalFromSTEB(24, 0, 14.0, 0),
						widgets.NewRow(headerWidgets, widgets.RowOpts{MainAxisAlignment: rendering.MainAxisAlignmentEnd})),
						InkOpts{Height: foundation.Some(64.0), Color: color}))),
				widgets.SemanticsOpts{Container: true}))
		}
		children = append(children, widgets.NewSingleChildScrollView(
			widgets.NewConstrainedBox(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MinWidth: foundation.Some(constraints.MinWidth)}),
				NewDataTable(w.columns, s.getRows(s.firstRowIndex, rowsPerPage), DataTableOpts{
					Key: s.tableKey, SortColumnIndex: w.o.SortColumnIndex, SortDescending: w.o.SortDescending, OnSelectAll: w.o.OnSelectAll,
					DividerThickness: w.o.DividerThickness, Decoration: painting.NewBoxDecoration(), DataRowMinHeight: w.o.DataRowMinHeight,
					DataRowMaxHeight: w.o.DataRowMaxHeight, HeadingRowHeight: foundation.Some(w.o.HeadingRowHeight.OrElse(56.0)),
					HorizontalMargin: foundation.Some(w.o.HorizontalMargin.OrElse(24.0)), CheckboxHorizontalMargin: w.o.CheckboxHorizontalMargin,
					ColumnSpacing: foundation.Some(w.o.ColumnSpacing.OrElse(56.0)), HideCheckboxColumn: w.o.HideCheckboxColumn, ShowBottomBorder: true,
					HeadingRowColor: w.o.HeadingRowColor})),
			widgets.SingleChildScrollViewOpts{ScrollDirection: painting.AxisHorizontal, Primary: w.o.Primary, Controller: w.o.Controller, DragStartBehavior: w.o.DragStartBehavior}))
		if w.o.HideEmptyRows {
			missing := min(max(rowsPerPage-s.rowCount+s.firstRowIndex, 0), rowsPerPage)
			children = append(children, widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(w.o.DataRowMaxHeight.OrElse(MinInteractiveDimension) * float64(missing))}))
		}
		children = append(children, widgets.NewDefaultTextStyle(footerTextStyle,
			widgets.NewIconThemeMerge(widgets.NewIconThemeData(widgets.IconThemeDataOpts{Opacity: foundation.Some(0.54)}),
				widgets.NewSizedBox(widgets.NewSingleChildScrollView(widgets.NewRow(footerWidgets),
					widgets.SingleChildScrollViewOpts{DragStartBehavior: w.o.DragStartBehavior, ScrollDirection: painting.AxisHorizontal, Reverse: true}),
					widgets.SizedBoxOpts{Height: foundation.Some(56.0)}))))
		return widgets.NewColumn(children, widgets.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStretch})
	}), CardOpts{NoSemanticContainer: true})
}
