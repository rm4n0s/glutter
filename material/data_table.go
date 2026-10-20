// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"fmt"
	"math"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	dataTableHeadingRowHeight          = 56.0
	dataTableHorizontalMargin          = 24.0
	dataTableColumnSpacing             = 56.0
	dataTableSortArrowPadding          = 2.0
	dataTableDividerThickness          = 1.0
	dataTableSortArrowAnimationDuraton = 150 * time.Millisecond
	sortArrowIconBaselineOffset        = -1.5
	sortArrowIconSize                  = 16.0
)

// DataTableTheme is an inherited widget that defines the visual properties of the
// DataTables below it.
//
// Flutter: material/data_table_theme.dart DataTableTheme
type DataTableTheme struct {
	widgets.InheritedWidget
	data DataTableThemeData
}

// NewDataTableTheme creates the theme.
//
// Flutter: material/data_table_theme.dart DataTableTheme.new
func NewDataTableTheme(data DataTableThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *DataTableTheme {
	return &DataTableTheme{InheritedWidget: widgets.NewInheritedWidgetBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/data_table_theme.dart DataTableTheme.data
func (w *DataTableTheme) Data() DataTableThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/data_table_theme.dart DataTableTheme.updateShouldNotify
func (w *DataTableTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*DataTableTheme).data)
}

// DataTableThemeOf returns the data of the closest DataTableTheme above the
// context, or the dataTableTheme of the closest Theme. The context depends on it.
//
// Flutter: material/data_table_theme.dart DataTableTheme.of
func DataTableThemeOf(bc widgets.BuildContext) DataTableThemeData {
	if t, ok := widgets.DependOn[*DataTableTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).DataTableTheme
}

// DataColumnSortCallback is called when the user asks to sort a column.
//
// Flutter: material/data_table.dart DataColumnSortCallback
type DataColumnSortCallback func(columnIndex int, ascending bool)

// DataColumnOpts are the optional parameters of NewDataColumn.
//
// Flutter: material/data_table.dart DataColumn
type DataColumnOpts struct {
	// ColumnWidth is the width of the column; the default is an intrinsic width,
	// with a flex for the only text column.
	ColumnWidth rendering.TableColumnWidth
	// Tooltip is the tooltip of the heading cell.
	Tooltip string
	// Numeric is whether the column holds numbers: they are right aligned.
	Numeric bool
	// OnSort is called when the user taps the heading to sort the column.
	OnSort DataColumnSortCallback
	// MouseCursor is the cursor over the heading cell, for each state.
	MouseCursor widgets.WidgetStateProperty[services.MouseCursor]
	// HeadingRowAlignment is the alignment of the heading; the theme or start by
	// default.
	HeadingRowAlignment rendering.MainAxisAlignment
}

// DataColumn describes a column of a DataTable.
//
// Flutter: material/data_table.dart DataColumn
type DataColumn struct {
	// Label is the heading of the column.
	Label widgets.Widget
	o     DataColumnOpts
}

// NewDataColumn creates the description of a column.
//
// Flutter: material/data_table.dart DataColumn.new
func NewDataColumn(label widgets.Widget, opts ...DataColumnOpts) DataColumn {
	return DataColumn{Label: label, o: firstOpts(opts)}
}

// Numeric is whether the column holds numbers.
//
// Flutter: material/data_table.dart DataColumn.numeric
func (c DataColumn) Numeric() bool { return c.o.Numeric }

// OnSort is the callback to sort the column.
//
// Flutter: material/data_table.dart DataColumn.onSort
func (c DataColumn) OnSort() DataColumnSortCallback { return c.o.OnSort }

// DataRowOpts are the optional parameters of NewDataRow.
//
// Flutter: material/data_table.dart DataRow
type DataRowOpts struct {
	// Key is the key of the row, which lets the table keep the state of the cells
	// when the rows are reordered.
	Key foundation.LocalKey
	// Selected is whether the row is selected.
	Selected bool
	// OnSelectChanged is called when the user selects or unselects the row; a
	// row without it cannot be selected.
	OnSelectChanged func(selected foundation.Opt[bool])
	// OnLongPress is called when the user long presses the row.
	OnLongPress gestures.GestureLongPressCallback
	// OnHover is called when a pointer enters or leaves the row.
	OnHover func(hovering bool)
	// Color is the color of the row, for each state.
	Color widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// MouseCursor is the cursor over the row, for each state.
	MouseCursor widgets.WidgetStateProperty[services.MouseCursor]
}

// DataRow describes a row of a DataTable.
//
// Flutter: material/data_table.dart DataRow
type DataRow struct {
	// Cells are the data of the row, one for each column.
	Cells []DataCell
	o     DataRowOpts
}

// NewDataRow creates the description of a row.
//
// Flutter: material/data_table.dart DataRow.new
func NewDataRow(cells []DataCell, opts ...DataRowOpts) DataRow {
	return DataRow{Cells: cells, o: firstOpts(opts)}
}

// NewDataRowByIndex creates the description of a row whose key is the index.
//
// Flutter: material/data_table.dart DataRow.byIndex
func NewDataRowByIndex(index foundation.Opt[int], cells []DataCell, opts ...DataRowOpts) DataRow {
	o := firstOpts(opts)
	o.Key = foundation.NewValueKey(optPtr(index))
	return DataRow{Cells: cells, o: o}
}

// Selected is whether the row is selected.
//
// Flutter: material/data_table.dart DataRow.selected
func (r DataRow) Selected() bool { return r.o.Selected }

func (r DataRow) debugInteractive() bool {
	if r.o.OnSelectChanged != nil {
		return true
	}
	for _, c := range r.Cells {
		if c.debugInteractive() {
			return true
		}
	}
	return false
}

// DataCellOpts are the optional parameters of NewDataCell.
//
// Flutter: material/data_table.dart DataCell
type DataCellOpts struct {
	// Placeholder is whether the child is a placeholder: it is dimmed.
	Placeholder bool
	// ShowEditIcon is whether to show an edit icon after the child.
	ShowEditIcon bool
	// OnTap is called when the user taps the cell.
	OnTap gestures.GestureTapCallback
	// OnLongPress is called when the user long presses the cell.
	OnLongPress gestures.GestureLongPressCallback
	// OnTapDown is called when the user touches the cell.
	OnTapDown gestures.GestureTapDownCallback
	// OnDoubleTap is called when the user double taps the cell.
	OnDoubleTap gestures.GestureTapCallback
	// OnTapCancel is called when the tap on the cell is cancelled.
	OnTapCancel gestures.GestureTapCancelCallback
}

// DataCell describes a cell of a DataRow.
//
// Flutter: material/data_table.dart DataCell
type DataCell struct {
	// Child is the content of the cell.
	Child widgets.Widget
	o     DataCellOpts
}

// NewDataCell creates the description of a cell.
//
// Flutter: material/data_table.dart DataCell.new
func NewDataCell(child widgets.Widget, opts ...DataCellOpts) DataCell {
	return DataCell{Child: child, o: firstOpts(opts)}
}

// DataCellEmpty is a cell with no content.
//
// Flutter: material/data_table.dart DataCell.empty
func DataCellEmpty() DataCell { return NewDataCell(widgets.NewSizedBoxShrink(nil)) }

func (c DataCell) debugInteractive() bool {
	o := c.o
	return o.OnTap != nil || o.OnDoubleTap != nil || o.OnLongPress != nil || o.OnTapDown != nil || o.OnTapCancel != nil
}

// DataTableOpts are the optional parameters of NewDataTable.
//
// Flutter: material/data_table.dart DataTable
type DataTableOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// SortColumnIndex is the column that the rows are sorted by.
	SortColumnIndex foundation.Opt[int]
	// SortDescending says that the sort is descending: Flutter's sortAscending is
	// true by default.
	SortDescending bool
	// OnSelectAll is called when the user selects or unselects all the rows with
	// the checkbox of the heading; by default the table calls the OnSelectChanged
	// of the rows.
	OnSelectAll func(selected foundation.Opt[bool])
	// Decoration is the background of the table.
	Decoration painting.Decoration
	// DataRowColor is the color of the data rows, for each state.
	DataRowColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// DataRowMinHeight is the minimum height of a data row; the default is
	// MinInteractiveDimension.
	DataRowMinHeight foundation.Opt[float64]
	// DataRowMaxHeight is the maximum height of a data row; the default is
	// MinInteractiveDimension.
	DataRowMaxHeight foundation.Opt[float64]
	// DataTextStyle is the style of the text of the data cells.
	DataTextStyle *painting.TextStyle
	// HeadingRowColor is the color of the heading row, for each state.
	HeadingRowColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// HeadingRowHeight is the height of the heading row; the default is 56.
	HeadingRowHeight foundation.Opt[float64]
	// HeadingTextStyle is the style of the text of the heading cells.
	HeadingTextStyle *painting.TextStyle
	// HorizontalMargin is the margin at the two sides of the table; the default
	// is 24.
	HorizontalMargin foundation.Opt[float64]
	// ColumnSpacing is the space between the columns; the default is 56.
	ColumnSpacing foundation.Opt[float64]
	// HideCheckboxColumn hides the column of checkboxes of the selectable rows:
	// Flutter's showCheckboxColumn is true by default.
	HideCheckboxColumn bool
	// ShowBottomBorder is whether to draw a line at the bottom of every row
	// (the default draws it at the top of every row but the heading).
	ShowBottomBorder bool
	// DividerThickness is the thickness of the lines between the rows.
	DividerThickness foundation.Opt[float64]
	// CheckboxHorizontalMargin is the horizontal margin around the checkboxes.
	CheckboxHorizontalMargin foundation.Opt[float64]
	// Border is the border of the table.
	Border *rendering.TableBorder
	// ClipBehavior is how the table is clipped by the border radius.
	ClipBehavior ui.Clip
}

// DataTable is a Material Design data table: a table of rows with a heading, with
// optional sorting and selection.
//
// Flutter: material/data_table.dart DataTable
type DataTable struct {
	widgets.StatelessWidget
	columns        []DataColumn
	rows           []DataRow
	o              DataTableOpts
	onlyTextColumn foundation.Opt[int]
}

var dataTableHeadingRowKey = foundation.NewUniqueKey()

// NewDataTable creates a data table. In debug builds it panics with the tags
// DataTableNoColumns, DataTableSortColumnIndexRange, DataTableRowCellCount,
// DataTableDividerThicknessNegative and DataTableRowHeightRange for the
// assertions of Flutter.
//
// Flutter: material/data_table.dart DataTable.new
func NewDataTable(columns []DataColumn, rows []DataRow, opts ...DataTableOpts) *DataTable {
	o := firstOpts(opts)
	if foundation.DebugMode {
		if len(columns) == 0 {
			panic(errors.New("DataTableNoColumns", "a DataTable needs at least one column"))
		}
		if i, ok := o.SortColumnIndex.Get(); ok && (i < 0 || i >= len(columns)) {
			panic(errors.New("DataTableSortColumnIndexRange", "the sortColumnIndex of a DataTable must be a column", "sortColumnIndex", i, "columns", len(columns)))
		}
		for i, row := range rows {
			if len(row.Cells) != len(columns) {
				panic(errors.New("DataTableRowCellCount", fmt.Sprintf("All rows must have the same number of cells as there are header cells (%d)", len(columns)),
					"row", i, "cells", len(row.Cells)))
			}
		}
		if v, ok := o.DividerThickness.Get(); ok && v < 0 {
			panic(errors.New("DataTableDividerThicknessNegative", "the dividerThickness of a DataTable must not be negative", "dividerThickness", v))
		}
		if lo, ok := o.DataRowMinHeight.Get(); ok {
			if hi, ok := o.DataRowMaxHeight.Get(); ok && hi < lo {
				panic(errors.New("DataTableRowHeightRange", "the dataRowMaxHeight of a DataTable must not be less than its dataRowMinHeight", "dataRowMinHeight", lo, "dataRowMaxHeight", hi))
			}
		}
	}
	// The only column that is not numeric gets the flex of the extra width.
	result := foundation.None[int]()
	for i, c := range columns {
		if c.o.Numeric {
			continue
		}
		if result.IsSome() {
			result = foundation.None[int]()
			break
		}
		result = foundation.Some(i)
	}
	return &DataTable{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), columns: columns, rows: rows, o: o, onlyTextColumn: result}
}

func (w *DataTable) debugInteractive() bool {
	for _, c := range w.columns {
		if c.o.OnSort != nil {
			return true
		}
	}
	for _, r := range w.rows {
		if r.debugInteractive() {
			return true
		}
	}
	return false
}

func (w *DataTable) handleSelectAll(checked foundation.Opt[bool], someChecked bool) {
	effective := someChecked || checked.OrElse(false)
	if w.o.OnSelectAll != nil {
		w.o.OnSelectAll(foundation.Some(effective))
		return
	}
	for _, row := range w.rows {
		if row.o.OnSelectChanged != nil && row.o.Selected != effective {
			row.o.OnSelectChanged(foundation.Some(effective))
		}
	}
}

func (w *DataTable) buildCheckbox(checked foundation.Opt[bool], onRowTap func(), onCheckboxChanged func(foundation.Opt[bool]),
	overlayColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]], tristate bool, rowMouseCursor services.MouseCursor, theme DataTableThemeData, bc widgets.BuildContext) widgets.Widget {
	horizontalMargin := firstOpt(w.o.HorizontalMargin, theme.HorizontalMargin, foundation.Some(dataTableHorizontalMargin)).Value()
	start := firstOpt(w.o.CheckboxHorizontalMargin, theme.CheckboxHorizontalMargin, foundation.Some(horizontalMargin)).Value()
	end := firstOpt(w.o.CheckboxHorizontalMargin, theme.CheckboxHorizontalMargin, foundation.Some(horizontalMargin/2.0)).Value()
	var contents widgets.Widget = widgets.NewSemantics(
		widgets.NewPadding(painting.NewEdgeInsetsDirectionalFromSTEB(start, 0, end, 0),
			widgets.NewCenter(NewCheckbox(checked, onCheckboxChanged, CheckboxOpts{Tristate: tristate}))),
		widgets.SemanticsOpts{Container: true})
	if onRowTap != nil {
		contents = NewTableRowInkWell(contents, TableRowInkWellOpts{OnTap: onRowTap, OverlayColor: overlayColor, MouseCursor: rowMouseCursor})
	}
	return widgets.NewTableCell(contents, widgets.TableCellOpts{VerticalAlignment: rendering.TableCellVerticalAlignmentFill})
}

func (w *DataTable) buildHeadingCell(bc widgets.BuildContext, padding painting.EdgeInsetsGeometry, label widgets.Widget, tooltip string, numeric bool,
	onSort func(), sorted, ascending bool, overlayColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]], mouseCursor services.MouseCursor,
	headingRowAlignment rendering.MainAxisAlignment) widgets.Widget {
	themeData := ThemeOf(bc)
	dataTableTheme := DataTableThemeOf(bc)
	row := []widgets.Widget{}
	if headingRowAlignment == rendering.MainAxisAlignmentCenter && onSort != nil {
		row = append(row, widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(sortArrowIconSize + dataTableSortArrowPadding)}))
	}
	row = append(row, label)
	if onSort != nil {
		up := foundation.None[bool]()
		if sorted {
			up = foundation.Some(ascending)
		}
		row = append(row, newSortArrow(sorted, up, dataTableSortArrowAnimationDuraton), widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(dataTableSortArrowPadding)}))
	}
	rowOpts := widgets.RowOpts{MainAxisAlignment: headingRowAlignment}
	if numeric {
		rowOpts.TextDirection = ui.TextDirectionRTL
	}
	label = widgets.NewSemantics(widgets.NewRow(row, rowOpts), widgets.SemanticsOpts{Role: foundation.Some(ui.SemanticsRoleColumnHeader)})
	style := firstNonNil(w.o.HeadingTextStyle, dataTableTheme.HeadingTextStyle, themeData.DataTableTheme.HeadingTextStyle, themeData.TextTheme.TitleSmall)
	height := firstOpt(w.o.HeadingRowHeight, dataTableTheme.HeadingRowHeight, themeData.DataTableTheme.HeadingRowHeight, foundation.Some(dataTableHeadingRowHeight)).Value()
	var alignment painting.AlignmentGeometry = painting.AlignmentDirectionalCenterStart
	if numeric {
		alignment = painting.AlignmentCenterRight
	}
	label = widgets.NewContainer(widgets.NewAnimatedDefaultTextStyle(widgets.DefaultTextStyleOf(bc).Style().Merge(style), dataTableSortArrowAnimationDuraton, label,
		widgets.AnimatedDefaultTextStyleOpts{SoftWrap: foundation.Some(false)}),
		widgets.ContainerOpts{Padding: padding, Height: foundation.Some(height), Alignment: alignment})
	if tooltip != "" {
		label = NewTooltip(label, TooltipOpts{Message: tooltip})
	}
	return NewInkWell(label, InkWellOpts{OnTap: onSort, OverlayColor: overlayColor, MouseCursor: mouseCursor})
}

func (w *DataTable) buildDataCell(bc widgets.BuildContext, padding painting.EdgeInsetsGeometry, label widgets.Widget, numeric bool, cell DataCell, row DataRow,
	overlayColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]], mouseCursor services.MouseCursor) widgets.Widget {
	themeData := ThemeOf(bc)
	dataTableTheme := DataTableThemeOf(bc)
	co := cell.o
	if co.ShowEditIcon {
		rowOpts := widgets.RowOpts{}
		if numeric {
			rowOpts.TextDirection = ui.TextDirectionRTL
		}
		label = widgets.NewRow([]widgets.Widget{widgets.NewExpanded(label), widgets.NewIcon(Icons.Edit(), widgets.IconOpts{Size: foundation.Some(18.0)})}, rowOpts)
	}
	style := firstNonNil(w.o.DataTextStyle, dataTableTheme.DataTextStyle, themeData.DataTableTheme.DataTextStyle, themeData.TextTheme.BodyMedium)
	minHeight := firstOpt(w.o.DataRowMinHeight, dataTableTheme.DataRowMinHeight, themeData.DataTableTheme.DataRowMinHeight, foundation.Some(MinInteractiveDimension)).Value()
	maxHeight := firstOpt(w.o.DataRowMaxHeight, dataTableTheme.DataRowMaxHeight, themeData.DataTableTheme.DataRowMaxHeight, foundation.Some(MinInteractiveDimension)).Value()
	merged := widgets.DefaultTextStyleOf(bc).Style().Merge(style)
	if co.Placeholder {
		merged = merged.CopyWith(painting.TextStyleCopy{Color: foundation.Some(style.Color.Value().WithOpacity(0.6))})
	}
	var alignment painting.AlignmentGeometry = painting.AlignmentDirectionalCenterStart
	if numeric {
		alignment = painting.AlignmentCenterRight
	}
	label = widgets.NewContainer(widgets.NewDefaultTextStyle(merged, NewDropdownButtonHideUnderline(label)), widgets.ContainerOpts{
		Padding: padding, Alignment: alignment,
		Constraints: foundation.Some(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MinHeight: foundation.Some(minHeight), MaxHeight: foundation.Some(maxHeight)}))})
	var onSelectChanged func()
	if row.o.OnSelectChanged != nil {
		onSelectChanged = func() { row.o.OnSelectChanged(foundation.Some(!row.o.Selected)) }
	}
	switch {
	case cell.debugInteractive():
		label = NewInkWell(label, InkWellOpts{OnTap: co.OnTap, OnDoubleTap: co.OnDoubleTap, OnLongPress: co.OnLongPress, OnTapCancel: co.OnTapCancel,
			OnTapDown: co.OnTapDown, OverlayColor: overlayColor})
	case onSelectChanged != nil || row.o.OnLongPress != nil || row.o.OnHover != nil:
		label = NewTableRowInkWell(label, TableRowInkWellOpts{OnTap: onSelectChanged, OnLongPress: row.o.OnLongPress, OnHover: row.o.OnHover,
			OverlayColor: overlayColor, MouseCursor: mouseCursor})
	}
	return widgets.NewTableCell(label)
}

// Build builds the table.
//
// Flutter: material/data_table.dart DataTable.build
func (w *DataTable) Build(bc widgets.BuildContext) widgets.Widget {
	if foundation.DebugMode && w.debugInteractive() {
		DebugCheckHasMaterial(bc)
	}
	theme := ThemeOf(bc)
	dataTableTheme := DataTableThemeOf(bc)
	headingRowColor := firstNonNilState(w.o.HeadingRowColor, dataTableTheme.HeadingRowColor, theme.DataTableTheme.HeadingRowColor)
	dataRowColor := firstNonNilState(w.o.DataRowColor, dataTableTheme.DataRowColor, theme.DataTableTheme.DataRowColor)
	defaultRowColor := widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
		if states.Has(widgets.WidgetStateSelected) {
			return foundation.Some(theme.ColorScheme.Primary().WithOpacity(0.08))
		}
		return foundation.None[ui.Color]()
	})
	anyRowSelectable := false
	for _, row := range w.rows {
		if row.o.OnSelectChanged != nil {
			anyRowSelectable = true
			break
		}
	}
	displayCheckboxColumn := !w.o.HideCheckboxColumn && anyRowSelectable
	rowsWithCheckbox, rowsChecked := 0, 0
	if displayCheckboxColumn {
		for _, row := range w.rows {
			if row.o.OnSelectChanged != nil {
				rowsWithCheckbox++
				if row.o.Selected {
					rowsChecked++
				}
			}
		}
	}
	allChecked := displayCheckboxColumn && rowsChecked == rowsWithCheckbox
	anyChecked := displayCheckboxColumn && rowsChecked > 0
	someChecked := anyChecked && !allChecked
	horizontalMargin := firstOpt(w.o.HorizontalMargin, dataTableTheme.HorizontalMargin, theme.DataTableTheme.HorizontalMargin, foundation.Some(dataTableHorizontalMargin)).Value()
	checkboxStart := firstOpt(w.o.CheckboxHorizontalMargin, dataTableTheme.CheckboxHorizontalMargin, theme.DataTableTheme.CheckboxHorizontalMargin, foundation.Some(horizontalMargin)).Value()
	checkboxEnd := firstOpt(w.o.CheckboxHorizontalMargin, dataTableTheme.CheckboxHorizontalMargin, theme.DataTableTheme.CheckboxHorizontalMargin, foundation.Some(horizontalMargin/2.0)).Value()
	columnSpacing := firstOpt(w.o.ColumnSpacing, dataTableTheme.ColumnSpacing, theme.DataTableTheme.ColumnSpacing, foundation.Some(dataTableColumnSpacing)).Value()

	columnCount := len(w.columns)
	if displayCheckboxColumn {
		columnCount++
	}
	tableColumns := make([]rendering.TableColumnWidth, columnCount)
	tableRows := make([]widgets.TableRow, len(w.rows)+1) // the +1 is for the header row
	thickness := firstOpt(w.o.DividerThickness, dataTableTheme.DividerThickness, theme.DataTableTheme.DividerThickness, foundation.Some(dataTableDividerThickness)).Value()
	for index := range tableRows {
		isSelected := index > 0 && w.rows[index-1].o.Selected
		isDisabled := index > 0 && anyRowSelectable && w.rows[index-1].o.OnSelectChanged == nil
		var states widgets.WidgetStates
		if isSelected {
			states = states.With(widgets.WidgetStateSelected)
		}
		if isDisabled {
			states = states.With(widgets.WidgetStateDisabled)
		}
		var rowColor foundation.Opt[ui.Color]
		if index > 0 {
			property := firstNonNilState(w.rows[index-1].o.Color, dataRowColor)
			if property != nil {
				rowColor = property.Resolve(states)
			}
		} else if headingRowColor != nil {
			rowColor = headingRowColor.Resolve(0)
		}
		borderSide := DividerCreateBorderSide(bc, DividerBorderSideOpts{Width: foundation.Some(thickness)})
		var border painting.BoxBorder
		switch {
		case w.o.ShowBottomBorder:
			border = painting.Border{Bottom: borderSide}
		case index == 0:
		default:
			border = painting.Border{Top: borderSide}
		}
		var key foundation.Key = dataTableHeadingRowKey
		if index > 0 {
			key = w.rows[index-1].o.Key
		}
		tableRows[index] = widgets.TableRow{Key: key, Decoration: painting.NewBoxDecoration(painting.BoxDecorationOpts{
			Border: border, Color: firstOpt(rowColor, defaultRowColor.Resolve(states))}),
			Children: make([]widgets.Widget, columnCount)}
	}

	displayColumnIndex := 0
	if displayCheckboxColumn {
		tableColumns[0] = rendering.NewFixedColumnWidth(checkboxStart + CheckboxWidth + checkboxEnd)
		checked := foundation.Some(allChecked)
		if someChecked {
			checked = foundation.None[bool]()
		}
		tableRows[0].Children[0] = w.buildCheckbox(checked, nil, func(c foundation.Opt[bool]) { w.handleSelectAll(c, someChecked) }, nil, true, nil, dataTableTheme, bc)
		for i, row := range w.rows {
			var states widgets.WidgetStates
			if row.o.Selected {
				states = states.With(widgets.WidgetStateSelected)
			}
			var onRowTap func()
			var onChanged func(foundation.Opt[bool])
			if row.o.OnSelectChanged != nil {
				r := row
				onRowTap = func() { r.o.OnSelectChanged(foundation.Some(!r.o.Selected)) }
				onChanged = r.o.OnSelectChanged
			}
			tableRows[i+1].Children[0] = w.buildCheckbox(foundation.Some(row.o.Selected), onRowTap, onChanged, firstNonNilState(row.o.Color, dataRowColor), false,
				rowCursor(row, dataTableTheme, states), dataTableTheme, bc)
		}
		displayColumnIndex++
	}
	for dataColumnIndex, column := range w.columns {
		var paddingStart float64
		switch {
		case dataColumnIndex == 0 && displayCheckboxColumn && w.o.CheckboxHorizontalMargin.IsNone():
			paddingStart = horizontalMargin / 2.0
		case dataColumnIndex == 0:
			paddingStart = horizontalMargin
		default:
			paddingStart = columnSpacing / 2.0
		}
		paddingEnd := columnSpacing / 2.0
		if dataColumnIndex == len(w.columns)-1 {
			paddingEnd = horizontalMargin
		}
		padding := painting.NewEdgeInsetsDirectionalFromSTEB(paddingStart, 0, paddingEnd, 0)
		switch {
		case column.o.ColumnWidth != nil:
			tableColumns[displayColumnIndex] = column.o.ColumnWidth
		case w.onlyTextColumn == foundation.Some(dataColumnIndex):
			tableColumns[displayColumnIndex] = rendering.NewIntrinsicColumnWidth(rendering.IntrinsicColumnWidthOpts{Flex: foundation.Some(1.0)})
		default:
			tableColumns[displayColumnIndex] = rendering.NewIntrinsicColumnWidth()
		}
		var headerStates widgets.WidgetStates
		if column.o.OnSort == nil {
			headerStates = headerStates.With(widgets.WidgetStateDisabled)
		}
		var onSort func()
		if column.o.OnSort != nil {
			c, idx := column, dataColumnIndex
			onSort = func() {
				c.o.OnSort(idx, w.o.SortColumnIndex != foundation.Some(idx) || w.o.SortDescending)
			}
		}
		var headerCursor services.MouseCursor
		if column.o.MouseCursor != nil {
			headerCursor = column.o.MouseCursor.Resolve(headerStates)
		}
		if headerCursor == nil && dataTableTheme.HeadingCellCursor != nil {
			headerCursor = dataTableTheme.HeadingCellCursor.Resolve(headerStates)
		}
		alignment := column.o.HeadingRowAlignment
		if alignment == rendering.MainAxisAlignmentUnspecified {
			alignment = dataTableTheme.HeadingRowAlignment
		}
		tableRows[0].Children[displayColumnIndex] = w.buildHeadingCell(bc, padding, column.Label, column.o.Tooltip, column.o.Numeric, onSort,
			w.o.SortColumnIndex == foundation.Some(dataColumnIndex), !w.o.SortDescending, headingRowColor, headerCursor, alignment)
		for i, row := range w.rows {
			var states widgets.WidgetStates
			if row.o.Selected {
				states = states.With(widgets.WidgetStateSelected)
			}
			tableRows[i+1].Children[displayColumnIndex] = w.buildDataCell(bc, padding, row.Cells[dataColumnIndex].Child, column.o.Numeric, row.Cells[dataColumnIndex], row,
				firstNonNilState(row.o.Color, dataRowColor), rowCursor(row, dataTableTheme, states))
		}
		displayColumnIndex++
	}

	widths := make(map[int]rendering.TableColumnWidth, len(tableColumns))
	for i, c := range tableColumns {
		widths[i] = c
	}
	var materialBorderRadius painting.BorderRadiusGeometry
	if w.o.Border != nil {
		materialBorderRadius = w.o.Border.BorderRadius
	}
	decoration := firstNonNilDecoration(w.o.Decoration, dataTableTheme.Decoration, theme.DataTableTheme.Decoration)
	return widgets.NewContainer(NewMaterial(widgets.NewTable(tableRows, widgets.TableOpts{ColumnWidths: widths,
		DefaultVerticalAlignment: rendering.TableCellVerticalAlignmentMiddle, Border: w.o.Border}),
		MaterialOpts{Type: MaterialTypeTransparency, BorderRadius: materialBorderRadius, ClipBehavior: w.o.ClipBehavior}),
		widgets.ContainerOpts{Decoration: decoration})
}

// rowCursor is the cursor of the row for the states, or of the theme.
func rowCursor(row DataRow, theme DataTableThemeData, states widgets.WidgetStates) services.MouseCursor {
	if row.o.MouseCursor != nil {
		if c := row.o.MouseCursor.Resolve(states); c != nil {
			return c
		}
	}
	if theme.DataRowCursor != nil {
		return theme.DataRowCursor.Resolve(states)
	}
	return nil
}

func firstNonNilState[T any](values ...widgets.WidgetStateProperty[T]) widgets.WidgetStateProperty[T] {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}

func firstNonNilDecoration(values ...painting.Decoration) painting.Decoration {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}

// TableRowInkWellOpts are the optional parameters of NewTableRowInkWell.
//
// Flutter: material/data_table.dart TableRowInkWell
type TableRowInkWellOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// OnTap is called when the user taps the row.
	OnTap gestures.GestureTapCallback
	// OnDoubleTap is called when the user double taps the row.
	OnDoubleTap gestures.GestureTapCallback
	// OnLongPress is called when the user long presses the row.
	OnLongPress gestures.GestureLongPressCallback
	// OnHighlightChanged is called when the highlight of the row changes.
	OnHighlightChanged func(value bool)
	// OnHover is called when a pointer enters or leaves the row.
	OnHover func(value bool)
	// OnSecondaryTap is called when the user taps the row with a secondary button.
	OnSecondaryTap gestures.GestureTapCallback
	// OnSecondaryTapDown is called when the user touches the row with a secondary
	// button.
	OnSecondaryTapDown gestures.GestureTapDownCallback
	// OverlayColor is the ink color, for each state.
	OverlayColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// MouseCursor is the cursor over the row.
	MouseCursor services.MouseCursor
}

// TableRowInkWell is a rectangular area of a Material that responds to touch,
// whose splash covers the whole row of the table that its cell is in. It must be
// a descendant of a Table.
//
// Flutter: material/data_table.dart TableRowInkWell
type TableRowInkWell struct {
	*InkResponse
}

// NewTableRowInkWell creates the ink well of a row, around the child.
//
// Flutter: material/data_table.dart TableRowInkWell.new
func NewTableRowInkWell(child widgets.Widget, opts ...TableRowInkWellOpts) *TableRowInkWell {
	o := firstOpts(opts)
	return &TableRowInkWell{NewInkResponse(child, InkResponseOpts{
		Key: o.Key, OnTap: o.OnTap, OnDoubleTap: o.OnDoubleTap, OnLongPress: o.OnLongPress, OnHighlightChanged: o.OnHighlightChanged, OnHover: o.OnHover,
		OnSecondaryTap: o.OnSecondaryTap, OnSecondaryTapDown: o.OnSecondaryTapDown, OverlayColor: o.OverlayColor, MouseCursor: o.MouseCursor,
		ContainedInkWell: true, HighlightShape: painting.BoxShapeRectangle, GetRectCallback: tableRowRectCallback})}
}

// tableRowRectCallback gives the rectangle of the row of the cell that the
// reference box is in.
//
// Flutter: material/data_table.dart TableRowInkWell.getRectCallback
func tableRowRectCallback(referenceBox rendering.Box) RectCallback {
	return func() ui.Rect {
		var cell rendering.RenderObject = referenceBox
		table := cell.Parent()
		transform := ui.NewMatrix4Identity()
		for table != nil {
			if _, isTable := table.(*rendering.RenderTable); isTable {
				break
			}
			table.ApplyPaintTransform(cell, transform)
			cell = table
			table = table.Parent()
		}
		if t, ok := table.(*rendering.RenderTable); ok {
			cellParentData := cell.ParentData().(*rendering.TableCellParentData)
			rect := t.GetRowBox(cellParentData.Y.Value())
			t.ApplyPaintTransform(cell, transform)
			if offset, ok := transform.AsTranslation(); ok {
				return rect.Shift(ui.Offset{Dx: -offset.Dx, Dy: -offset.Dy})
			}
		}
		return ui.Rect{}
	}
}

// sortArrow is the arrow in the heading of a sorted column: it fades in and turns
// around when the direction of the sort changes.
//
// Flutter: material/data_table.dart _SortArrow
type sortArrow struct {
	widgets.StatefulWidget
	visible  bool
	up       foundation.Opt[bool]
	duration time.Duration
}

func newSortArrow(visible bool, up foundation.Opt[bool], duration time.Duration) *sortArrow {
	return &sortArrow{StatefulWidget: widgets.NewStatefulWidgetBase(), visible: visible, up: up, duration: duration}
}

func (w *sortArrow) CreateState() widgets.State { return &sortArrowState{} }

type sortArrowState struct {
	widgets.StateOf[*sortArrow]
	widgets.TickerProvider
	opacityController     *animation.AnimationController
	opacityAnimation      *animation.CurvedAnimation
	orientationController *animation.AnimationController
	orientationAnimation  animation.Animation[float64]
	orientationOffset     float64
	up                    foundation.Opt[bool]
}

var sortArrowTurnTween = animation.NewTween(0.0, math.Pi).Chain(animation.NewCurveTween(animation.Curves.EaseIn))

func (s *sortArrowState) InitState() {
	w := s.Widget()
	s.up = w.up
	s.opacityController = animation.NewAnimationController(s, animation.AnimationControllerOpts{Duration: foundation.Some(w.duration)})
	s.opacityAnimation = animation.NewCurvedAnimation(s.opacityController, animation.Curves.FastOutSlowIn)
	s.opacityAnimation.AddListener(s.rebuild)
	if w.visible {
		s.opacityController.SetValue(1.0)
	} else {
		s.opacityController.SetValue(0.0)
	}
	s.orientationController = animation.NewAnimationController(s, animation.AnimationControllerOpts{Duration: foundation.Some(w.duration)})
	s.orientationAnimation = animation.Drive[float64](s.orientationController, sortArrowTurnTween)
	s.orientationAnimation.AddListener(s.rebuild)
	s.orientationAnimation.AddStatusListener(s.resetOrientationAnimation)
	if w.visible {
		s.orientationOffset = math.Pi
		if w.up.Value() {
			s.orientationOffset = 0
		}
	}
}

func (s *sortArrowState) rebuild() { s.SetState(func() {}) }

func (s *sortArrowState) resetOrientationAnimation(status animation.AnimationStatus) {
	if status == animation.AnimationStatusCompleted {
		s.orientationOffset += math.Pi
		s.orientationController.SetValue(0.0) // This triggers a pointless rebuild.
	}
}

func (s *sortArrowState) DidUpdateWidget(old *sortArrow) {
	w := s.Widget()
	skipArrow := false
	newUp := firstOpt(w.up, s.up)
	if old.visible != w.visible {
		if w.visible && s.opacityController.Status() == animation.AnimationStatusDismissed {
			s.orientationController.Stop()
			s.orientationController.SetValue(0.0)
			s.orientationOffset = math.Pi
			if newUp.Value() {
				s.orientationOffset = 0
			}
			skipArrow = true
		}
		if w.visible {
			s.opacityController.Forward()
		} else {
			s.opacityController.Reverse()
		}
	}
	if s.up != newUp && !skipArrow {
		if s.orientationController.Status() == animation.AnimationStatusDismissed {
			s.orientationController.Forward()
		} else {
			s.orientationController.Reverse()
		}
	}
	s.up = newUp
}

func (s *sortArrowState) Dispose() {
	s.opacityController.Dispose()
	s.orientationController.Dispose()
	s.opacityAnimation.Dispose()
}

func (s *sortArrowState) Build(bc widgets.BuildContext) widgets.Widget {
	transform := ui.NewMatrix4RotationZ(s.orientationOffset + s.orientationAnimation.Value())
	transform.SetTranslationRaw(0.0, sortArrowIconBaselineOffset, 0.0)
	return widgets.NewFadeTransition(s.opacityAnimation, widgets.NewTransform(transform,
		widgets.NewIcon(Icons.ArrowUpward(), widgets.IconOpts{Size: foundation.Some(sortArrowIconSize)}),
		widgets.TransformOpts{Alignment: painting.AlignmentCenter}))
}
