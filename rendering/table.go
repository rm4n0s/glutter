// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package rendering

import (
	"math"
	"sort"
	"strconv"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
)

// TableCellVerticalAlignment is how a cell is aligned vertically within its row.
// The zero value is TableCellVerticalAlignmentUnspecified: the default of the
// table.
//
// Flutter: rendering/table.dart TableCellVerticalAlignment
type TableCellVerticalAlignment int

// The values of TableCellVerticalAlignment follow Flutter's declaration order.
const (
	// TableCellVerticalAlignmentUnspecified is the zero value: the default of
	// the table (null in Flutter's parent data).
	//
	// Flutter: no counterpart
	TableCellVerticalAlignmentUnspecified TableCellVerticalAlignment = iota
	// TableCellVerticalAlignmentTop places the cell at the top of the row.
	//
	// Flutter: rendering/table.dart TableCellVerticalAlignment.top
	TableCellVerticalAlignmentTop
	// TableCellVerticalAlignmentMiddle centers the cell in the row.
	//
	// Flutter: rendering/table.dart TableCellVerticalAlignment.middle
	TableCellVerticalAlignmentMiddle
	// TableCellVerticalAlignmentBottom places the cell at the bottom of the row.
	//
	// Flutter: rendering/table.dart TableCellVerticalAlignment.bottom
	TableCellVerticalAlignmentBottom
	// TableCellVerticalAlignmentBaseline aligns the baseline of the cell with
	// the baseline of the row.
	//
	// Flutter: rendering/table.dart TableCellVerticalAlignment.baseline
	TableCellVerticalAlignmentBaseline
	// TableCellVerticalAlignmentFill sizes the cell to the height of the row.
	//
	// Flutter: rendering/table.dart TableCellVerticalAlignment.fill
	TableCellVerticalAlignmentFill
	// TableCellVerticalAlignmentIntrinsicHeight sizes the row to the cells'
	// intrinsic height, then makes the cell fill the row.
	//
	// Flutter: rendering/table.dart TableCellVerticalAlignment.intrinsicHeight
	TableCellVerticalAlignmentIntrinsicHeight
)

// Name returns the Dart enum name, such as "middle".
//
// Flutter: rendering/table.dart TableCellVerticalAlignment.name
func (a TableCellVerticalAlignment) Name() string {
	return enumName(int(a), "top", "middle", "bottom", "baseline", "fill", "intrinsicHeight")
}

// String returns the name Dart prints, such as "TableCellVerticalAlignment.top".
//
// Flutter: rendering/table.dart TableCellVerticalAlignment.toString
func (a TableCellVerticalAlignment) String() string { return "TableCellVerticalAlignment." + a.Name() }

// TableCellParentData is the parent data of the cells of a RenderTable.
//
// Flutter: rendering/table.dart TableCellParentData
type TableCellParentData struct {
	BoxParentData
	// VerticalAlignment is how the cell is aligned vertically; unspecified is
	// the table's default.
	//
	// Flutter: rendering/table.dart TableCellParentData.verticalAlignment
	VerticalAlignment TableCellVerticalAlignment
	// X is the column that the child was in the last time the table was laid
	// out.
	//
	// Flutter: rendering/table.dart TableCellParentData.x
	X foundation.Opt[int]
	// Y is the row that the child was in the last time the table was laid out.
	//
	// Flutter: rendering/table.dart TableCellParentData.y
	Y foundation.Opt[int]
}

// String is the offset and the vertical alignment.
//
// Flutter: rendering/table.dart TableCellParentData.toString
func (d *TableCellParentData) String() string {
	if d.VerticalAlignment == TableCellVerticalAlignmentUnspecified {
		return d.BoxParentData.String() + "; default vertical alignment"
	}
	return d.BoxParentData.String() + "; " + d.VerticalAlignment.String()
}

// TableColumnWidth defines the width of a column of a table.
//
// Flutter: rendering/table.dart TableColumnWidth
type TableColumnWidth interface {
	// MinIntrinsicWidth is the smallest width that the column can have, given
	// its cells and the width of the table (infinite if unbounded).
	//
	// Flutter: rendering/table.dart TableColumnWidth.minIntrinsicWidth
	MinIntrinsicWidth(cells []Box, containerWidth float64) float64
	// MaxIntrinsicWidth is the ideal width of the column.
	//
	// Flutter: rendering/table.dart TableColumnWidth.maxIntrinsicWidth
	MaxIntrinsicWidth(cells []Box, containerWidth float64) float64
	// Flex is the flex factor of the column, or None if it takes no part in the
	// distribution of the free space.
	//
	// Flutter: rendering/table.dart TableColumnWidth.flex
	Flex(cells []Box) foundation.Opt[float64]
	// String describes the column width.
	String() string
}

// IntrinsicColumnWidth sizes the column by the intrinsic width of its widest
// cell.
//
// Flutter: rendering/table.dart IntrinsicColumnWidth
type IntrinsicColumnWidth struct {
	flex foundation.Opt[float64]
}

// IntrinsicColumnWidthOpts are the optional parameters of NewIntrinsicColumnWidth.
//
// Flutter: rendering/table.dart IntrinsicColumnWidth.new
type IntrinsicColumnWidthOpts struct {
	// Flex, if set, makes the column take a share of the free space.
	Flex foundation.Opt[float64]
}

// NewIntrinsicColumnWidth creates the column width.
//
// Flutter: rendering/table.dart IntrinsicColumnWidth.new
func NewIntrinsicColumnWidth(opts ...IntrinsicColumnWidthOpts) IntrinsicColumnWidth {
	return IntrinsicColumnWidth{flex: firstOpts(opts).Flex}
}

// MinIntrinsicWidth is the largest minimum intrinsic width among the cells.
//
// Flutter: rendering/table.dart IntrinsicColumnWidth.minIntrinsicWidth
func (IntrinsicColumnWidth) MinIntrinsicWidth(cells []Box, _ float64) float64 {
	result := 0.0
	for _, cell := range cells {
		result = math.Max(result, cell.GetMinIntrinsicWidth(math.Inf(1)))
	}
	return result
}

// MaxIntrinsicWidth is the largest maximum intrinsic width among the cells.
//
// Flutter: rendering/table.dart IntrinsicColumnWidth.maxIntrinsicWidth
func (IntrinsicColumnWidth) MaxIntrinsicWidth(cells []Box, _ float64) float64 {
	result := 0.0
	for _, cell := range cells {
		result = math.Max(result, cell.GetMaxIntrinsicWidth(math.Inf(1)))
	}
	return result
}

// Flex is the flex the column was created with.
//
// Flutter: rendering/table.dart IntrinsicColumnWidth.flex
func (c IntrinsicColumnWidth) Flex([]Box) foundation.Opt[float64] { return c.flex }

// String is "IntrinsicColumnWidth(flex: 1.0)".
//
// Flutter: rendering/table.dart IntrinsicColumnWidth.toString
func (c IntrinsicColumnWidth) String() string {
	flex := "null"
	if f, ok := c.flex.Get(); ok {
		flex = strconv.FormatFloat(f, 'f', 1, 64)
	}
	return "IntrinsicColumnWidth(flex: " + flex + ")"
}

// FixedColumnWidth sizes the column to a constant width.
//
// Flutter: rendering/table.dart FixedColumnWidth
type FixedColumnWidth struct{ value float64 }

// NewFixedColumnWidth creates the column width.
//
// Flutter: rendering/table.dart FixedColumnWidth.new
func NewFixedColumnWidth(value float64) FixedColumnWidth { return FixedColumnWidth{value} }

// Value is the width in logical pixels.
//
// Flutter: rendering/table.dart FixedColumnWidth.value
func (c FixedColumnWidth) Value() float64 { return c.value }

// MinIntrinsicWidth is the width.
//
// Flutter: rendering/table.dart FixedColumnWidth.minIntrinsicWidth
func (c FixedColumnWidth) MinIntrinsicWidth([]Box, float64) float64 { return c.value }

// MaxIntrinsicWidth is the width.
//
// Flutter: rendering/table.dart FixedColumnWidth.maxIntrinsicWidth
func (c FixedColumnWidth) MaxIntrinsicWidth([]Box, float64) float64 { return c.value }

// Flex is none.
//
// Flutter: rendering/table.dart TableColumnWidth.flex
func (FixedColumnWidth) Flex([]Box) foundation.Opt[float64] { return foundation.None[float64]() }

// String is "FixedColumnWidth(5.0)".
//
// Flutter: rendering/table.dart FixedColumnWidth.toString
func (c FixedColumnWidth) String() string {
	return "FixedColumnWidth(" + foundation.DebugFormatDouble(foundation.Some(c.value)) + ")"
}

// FractionColumnWidth sizes the column to a fraction of the table's width.
//
// Flutter: rendering/table.dart FractionColumnWidth
type FractionColumnWidth struct{ value float64 }

// NewFractionColumnWidth creates the column width.
//
// Flutter: rendering/table.dart FractionColumnWidth.new
func NewFractionColumnWidth(value float64) FractionColumnWidth { return FractionColumnWidth{value} }

// Value is the fraction of the table's width.
//
// Flutter: rendering/table.dart FractionColumnWidth.value
func (c FractionColumnWidth) Value() float64 { return c.value }

// MinIntrinsicWidth is the fraction of the container's width, or 0 if it is
// unbounded.
//
// Flutter: rendering/table.dart FractionColumnWidth.minIntrinsicWidth
func (c FractionColumnWidth) MinIntrinsicWidth(_ []Box, containerWidth float64) float64 {
	if math.IsInf(containerWidth, 0) || math.IsNaN(containerWidth) {
		return 0
	}
	return c.value * containerWidth
}

// MaxIntrinsicWidth is the fraction of the container's width, or 0 if it is
// unbounded.
//
// Flutter: rendering/table.dart FractionColumnWidth.maxIntrinsicWidth
func (c FractionColumnWidth) MaxIntrinsicWidth(cells []Box, containerWidth float64) float64 {
	return c.MinIntrinsicWidth(cells, containerWidth)
}

// Flex is none.
//
// Flutter: rendering/table.dart TableColumnWidth.flex
func (FractionColumnWidth) Flex([]Box) foundation.Opt[float64] { return foundation.None[float64]() }

// String is "FractionColumnWidth(0.5)".
//
// Flutter: rendering/table.dart FractionColumnWidth.toString
func (c FractionColumnWidth) String() string {
	return "FractionColumnWidth(" + dartDouble(c.value) + ")"
}

// FlexColumnWidth sizes the column by a share of the space left over by the
// other columns.
//
// Flutter: rendering/table.dart FlexColumnWidth
type FlexColumnWidth struct{ value float64 }

// FlexColumnWidthOpts are the optional parameters of NewFlexColumnWidth.
//
// Flutter: rendering/table.dart FlexColumnWidth.new
type FlexColumnWidthOpts struct {
	// Value is the flex factor; the default is 1.
	Value foundation.Opt[float64]
}

// NewFlexColumnWidth creates the column width.
//
// Flutter: rendering/table.dart FlexColumnWidth.new
func NewFlexColumnWidth(opts ...FlexColumnWidthOpts) FlexColumnWidth {
	return FlexColumnWidth{firstOpts(opts).Value.OrElse(1)}
}

// Value is the flex factor.
//
// Flutter: rendering/table.dart FlexColumnWidth.value
func (c FlexColumnWidth) Value() float64 { return c.value }

// MinIntrinsicWidth is 0.
//
// Flutter: rendering/table.dart FlexColumnWidth.minIntrinsicWidth
func (FlexColumnWidth) MinIntrinsicWidth([]Box, float64) float64 { return 0 }

// MaxIntrinsicWidth is 0.
//
// Flutter: rendering/table.dart FlexColumnWidth.maxIntrinsicWidth
func (FlexColumnWidth) MaxIntrinsicWidth([]Box, float64) float64 { return 0 }

// Flex is the flex factor.
//
// Flutter: rendering/table.dart FlexColumnWidth.flex
func (c FlexColumnWidth) Flex([]Box) foundation.Opt[float64] { return foundation.Some(c.value) }

// String is "FlexColumnWidth(1.0)".
//
// Flutter: rendering/table.dart FlexColumnWidth.toString
func (c FlexColumnWidth) String() string {
	return "FlexColumnWidth(" + foundation.DebugFormatDouble(foundation.Some(c.value)) + ")"
}

// MaxColumnWidth sizes the column to the larger of two column widths.
//
// Flutter: rendering/table.dart MaxColumnWidth
type MaxColumnWidth struct{ a, b TableColumnWidth }

// NewMaxColumnWidth creates the column width.
//
// Flutter: rendering/table.dart MaxColumnWidth.new
func NewMaxColumnWidth(a, b TableColumnWidth) MaxColumnWidth { return MaxColumnWidth{a, b} }

// MinIntrinsicWidth is the larger of the two.
//
// Flutter: rendering/table.dart MaxColumnWidth.minIntrinsicWidth
func (c MaxColumnWidth) MinIntrinsicWidth(cells []Box, w float64) float64 {
	return math.Max(c.a.MinIntrinsicWidth(cells, w), c.b.MinIntrinsicWidth(cells, w))
}

// MaxIntrinsicWidth is the larger of the two.
//
// Flutter: rendering/table.dart MaxColumnWidth.maxIntrinsicWidth
func (c MaxColumnWidth) MaxIntrinsicWidth(cells []Box, w float64) float64 {
	return math.Max(c.a.MaxIntrinsicWidth(cells, w), c.b.MaxIntrinsicWidth(cells, w))
}

// Flex is the larger of the two flex factors, or the one that is set.
//
// Flutter: rendering/table.dart MaxColumnWidth.flex
func (c MaxColumnWidth) Flex(cells []Box) foundation.Opt[float64] {
	return combineFlex(c.a.Flex(cells), c.b.Flex(cells), math.Max)
}

// String is "MaxColumnWidth(a, b)".
//
// Flutter: rendering/table.dart MaxColumnWidth.toString
func (c MaxColumnWidth) String() string {
	return "MaxColumnWidth(" + c.a.String() + ", " + c.b.String() + ")"
}

// MinColumnWidth sizes the column to the smaller of two column widths.
//
// Flutter: rendering/table.dart MinColumnWidth
type MinColumnWidth struct{ a, b TableColumnWidth }

// NewMinColumnWidth creates the column width.
//
// Flutter: rendering/table.dart MinColumnWidth.new
func NewMinColumnWidth(a, b TableColumnWidth) MinColumnWidth { return MinColumnWidth{a, b} }

// MinIntrinsicWidth is the smaller of the two.
//
// Flutter: rendering/table.dart MinColumnWidth.minIntrinsicWidth
func (c MinColumnWidth) MinIntrinsicWidth(cells []Box, w float64) float64 {
	return math.Min(c.a.MinIntrinsicWidth(cells, w), c.b.MinIntrinsicWidth(cells, w))
}

// MaxIntrinsicWidth is the smaller of the two.
//
// Flutter: rendering/table.dart MinColumnWidth.maxIntrinsicWidth
func (c MinColumnWidth) MaxIntrinsicWidth(cells []Box, w float64) float64 {
	return math.Min(c.a.MaxIntrinsicWidth(cells, w), c.b.MaxIntrinsicWidth(cells, w))
}

// Flex is the smaller of the two flex factors, or the one that is set.
//
// Flutter: rendering/table.dart MinColumnWidth.flex
func (c MinColumnWidth) Flex(cells []Box) foundation.Opt[float64] {
	return combineFlex(c.a.Flex(cells), c.b.Flex(cells), math.Min)
}

// String is "MinColumnWidth(a, b)".
//
// Flutter: rendering/table.dart MinColumnWidth.toString
func (c MinColumnWidth) String() string {
	return "MinColumnWidth(" + c.a.String() + ", " + c.b.String() + ")"
}

func combineFlex(a, b foundation.Opt[float64], pick func(x, y float64) float64) foundation.Opt[float64] {
	av, aok := a.Get()
	bv, bok := b.Get()
	switch {
	case !aok:
		return b
	case !bok:
		return a
	}
	return foundation.Some(pick(av, bv))
}

// dartDouble prints a float64 the way Dart prints a double.
func dartDouble(v float64) string {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	for _, c := range s {
		if c == '.' {
			return s
		}
	}
	return s + ".0"
}

// RenderTable implements the table layout algorithm: the cells are in rows and
// columns, a column is as wide as its TableColumnWidth says, and a row is as
// tall as its tallest cell.
//
// Flutter: rendering/table.dart RenderTable
type RenderTable struct {
	RenderBox
	columns               int
	rows                  int
	columnWidths          map[int]TableColumnWidth
	defaultColumnWidth    TableColumnWidth
	textDirection         ui.TextDirection
	border                *TableBorder
	rowDecorations        []painting.Decoration
	hasRowDecorations     bool
	rowDecorationPainters []painting.BoxPainter
	configuration         painting.ImageConfiguration
	defaultVerticalAlign  TableCellVerticalAlignment
	textBaseline          ui.TextBaseline
	children              []Box
	baselineDistance      foundation.Opt[float64]
	rowTops               []float64
	columnLefts           []float64
	tableWidth            float64

	idToIndexMap map[int]tableIndex
	cachedRows   map[int]*semantics.SemanticsNode
	cachedCells  map[tableIndex]*semantics.SemanticsNode
}

// RenderTableOpts are the optional parameters of NewRenderTable.
//
// Flutter: rendering/table.dart RenderTable.new
type RenderTableOpts struct {
	// Columns is the number of columns; the default is the length of the first
	// row of Children, or 0.
	Columns foundation.Opt[int]
	// Rows is the number of rows; it can not be set together with Children.
	Rows foundation.Opt[int]
	// ColumnWidths are the widths of the columns that have their own.
	ColumnWidths map[int]TableColumnWidth
	// DefaultColumnWidth defaults to NewFlexColumnWidth().
	DefaultColumnWidth TableColumnWidth
	// Border is the border of the table; nil for none.
	Border *TableBorder
	// RowDecorations are the decorations painted behind the rows; entries may be
	// nil.
	RowDecorations []painting.Decoration
	// Configuration is the configuration for the row decorations.
	Configuration painting.ImageConfiguration
	// DefaultVerticalAlignment defaults to TableCellVerticalAlignmentTop.
	DefaultVerticalAlignment TableCellVerticalAlignment
	// TextBaseline is the baseline for baseline-aligned cells; unspecified is
	// null.
	TextBaseline ui.TextBaseline
	// Children are the cells, row by row.
	Children [][]Box
}

// NewRenderTable creates a table.
//
// Flutter: rendering/table.dart RenderTable.new
func NewRenderTable(textDirection ui.TextDirection, opts ...RenderTableOpts) *RenderTable {
	o := firstOpts(opts)
	if foundation.DebugMode {
		if c, ok := o.Columns.Get(); ok && c < 0 {
			panic(errors.New("TableNegativeCount", "the number of columns of a table must not be negative", "columns", c))
		}
		if r, ok := o.Rows.Get(); ok && r < 0 {
			panic(errors.New("TableNegativeCount", "the number of rows of a table must not be negative", "rows", r))
		}
		if o.Rows.IsSome() && o.Children != nil {
			panic(errors.New("TableRowsAndChildren", "a table can not be given both a number of rows and children"))
		}
	}
	if o.DefaultColumnWidth == nil {
		o.DefaultColumnWidth = NewFlexColumnWidth()
	}
	if o.DefaultVerticalAlignment == TableCellVerticalAlignmentUnspecified {
		o.DefaultVerticalAlignment = TableCellVerticalAlignmentTop
	}
	if o.ColumnWidths == nil {
		o.ColumnWidths = map[int]TableColumnWidth{}
	}
	columns := 0
	if c, ok := o.Columns.Get(); ok {
		columns = c
	} else if len(o.Children) > 0 {
		columns = len(o.Children[0])
	}
	r := &RenderTable{textDirection: textDirection, columns: columns, rows: o.Rows.OrElse(0), columnWidths: o.ColumnWidths,
		defaultColumnWidth: o.DefaultColumnWidth, border: o.Border, textBaseline: o.TextBaseline,
		defaultVerticalAlign: o.DefaultVerticalAlignment, configuration: o.Configuration}
	InitRenderBox(&r.RenderBox, r)
	r.AsRenderObject().setChildModel(r)
	r.children = make([]Box, r.columns*r.rows)
	r.SetRowDecorations(o.RowDecorations)
	for _, row := range o.Children {
		cells := make([]Box, len(row))
		copy(cells, row)
		r.AddRow(cells)
	}
	return r
}

func (r *RenderTable) checkCells() {
	if foundation.DebugMode && len(r.children) != r.rows*r.columns {
		panic(errors.New("TableCellCountMismatch", "the cells of a table do not match its rows and columns",
			"cells", len(r.children), "rows", r.rows, "columns", r.columns))
	}
}

// Columns is the number of columns.
//
// Flutter: rendering/table.dart RenderTable.columns
func (r *RenderTable) Columns() int { return r.columns }

// SetColumns changes the number of columns, dropping the cells of the removed
// columns.
//
// Flutter: rendering/table.dart RenderTable.columns
func (r *RenderTable) SetColumns(value int) {
	if foundation.DebugMode && value < 0 {
		panic(errors.New("TableNegativeCount", "the number of columns of a table must not be negative", "columns", value))
	}
	if value == r.columns {
		return
	}
	oldColumns, oldChildren := r.columns, r.children
	r.columns = value
	r.children = make([]Box, r.columns*r.rows)
	toCopy := min(r.columns, oldColumns)
	for y := 0; y < r.rows; y++ {
		for x := 0; x < toCopy; x++ {
			r.children[x+y*r.columns] = oldChildren[x+y*oldColumns]
		}
	}
	if oldColumns > r.columns {
		for y := 0; y < r.rows; y++ {
			for x := r.columns; x < oldColumns; x++ {
				if old := oldChildren[x+y*oldColumns]; !isNil(old) {
					r.DropChild(old)
				}
			}
		}
	}
	r.MarkNeedsLayout()
}

// Rows is the number of rows.
//
// Flutter: rendering/table.dart RenderTable.rows
func (r *RenderTable) Rows() int { return r.rows }

// SetRows changes the number of rows, dropping the cells of the removed rows.
//
// Flutter: rendering/table.dart RenderTable.rows
func (r *RenderTable) SetRows(value int) {
	if foundation.DebugMode && value < 0 {
		panic(errors.New("TableNegativeCount", "the number of rows of a table must not be negative", "rows", value))
	}
	if value == r.rows {
		return
	}
	if r.rows > value {
		for xy := r.columns * value; xy < len(r.children); xy++ {
			if c := r.children[xy]; !isNil(c) {
				r.DropChild(c)
			}
		}
	}
	r.rows = value
	newLen := r.columns * r.rows
	if newLen <= len(r.children) {
		r.children = r.children[:newLen]
	} else {
		r.children = append(r.children, make([]Box, newLen-len(r.children))...)
	}
	r.MarkNeedsLayout()
}

// ColumnWidths is a copy of the widths of the columns that have their own.
//
// Flutter: rendering/table.dart RenderTable.columnWidths
func (r *RenderTable) ColumnWidths() map[int]TableColumnWidth {
	out := make(map[int]TableColumnWidth, len(r.columnWidths))
	for k, v := range r.columnWidths {
		out[k] = v
	}
	return out
}

// SetColumnWidths replaces the widths of the columns that have their own; nil
// clears them.
//
// Flutter: rendering/table.dart RenderTable.columnWidths
func (r *RenderTable) SetColumnWidths(value map[int]TableColumnWidth) {
	if len(r.columnWidths) == 0 && value == nil {
		return
	}
	if value == nil {
		value = map[int]TableColumnWidth{}
	}
	r.columnWidths = value
	r.MarkNeedsLayout()
}

// SetColumnWidth sets the width of a column.
//
// Flutter: rendering/table.dart RenderTable.setColumnWidth
func (r *RenderTable) SetColumnWidth(column int, value TableColumnWidth) {
	if old, ok := r.columnWidths[column]; ok && old == value {
		return
	}
	r.columnWidths[column] = value
	r.MarkNeedsLayout()
}

// DefaultColumnWidth is the width of the columns that have none of their own.
//
// Flutter: rendering/table.dart RenderTable.defaultColumnWidth
func (r *RenderTable) DefaultColumnWidth() TableColumnWidth { return r.defaultColumnWidth }

// SetDefaultColumnWidth marks the table as needing layout when the value changes.
//
// Flutter: rendering/table.dart RenderTable.defaultColumnWidth
func (r *RenderTable) SetDefaultColumnWidth(value TableColumnWidth) {
	if r.defaultColumnWidth == value {
		return
	}
	r.defaultColumnWidth = value
	r.MarkNeedsLayout()
}

// TextDirection is the order of the columns.
//
// Flutter: rendering/table.dart RenderTable.textDirection
func (r *RenderTable) TextDirection() ui.TextDirection { return r.textDirection }

// SetTextDirection marks the table as needing layout when the value changes.
//
// Flutter: rendering/table.dart RenderTable.textDirection
func (r *RenderTable) SetTextDirection(value ui.TextDirection) {
	if r.textDirection == value {
		return
	}
	r.textDirection = value
	r.MarkNeedsLayout()
}

// Border is the border of the table, or nil.
//
// Flutter: rendering/table.dart RenderTable.border
func (r *RenderTable) Border() *TableBorder { return r.border }

// SetBorder marks the table as needing paint when the value changes.
//
// Flutter: rendering/table.dart RenderTable.border
func (r *RenderTable) SetBorder(value *TableBorder) {
	if (r.border == nil) == (value == nil) && (r.border == nil || *r.border == *value) {
		return
	}
	r.border = value
	r.MarkNeedsPaint()
}

// RowDecorations is a copy of the decorations painted behind the rows.
//
// Flutter: rendering/table.dart RenderTable.rowDecorations
func (r *RenderTable) RowDecorations() []painting.Decoration {
	return append([]painting.Decoration(nil), r.rowDecorations...)
}

// SetRowDecorations replaces the decorations painted behind the rows (nil for
// none) and disposes the painters of the old ones.
//
// Deviation: Dart skips the work when the same list is set again; slices can not
// be compared, so the painters are always recreated.
//
// Flutter: rendering/table.dart RenderTable.rowDecorations
func (r *RenderTable) SetRowDecorations(value []painting.Decoration) {
	r.rowDecorations = value
	r.hasRowDecorations = value != nil
	r.disposeRowDecorationPainters()
	if value != nil {
		r.rowDecorationPainters = make([]painting.BoxPainter, len(value))
	} else {
		r.rowDecorationPainters = nil
	}
}

func (r *RenderTable) disposeRowDecorationPainters() {
	for _, p := range r.rowDecorationPainters {
		if !isNil(p) {
			p.Dispose()
		}
	}
}

// Configuration is the configuration passed to the row decorations.
//
// Flutter: rendering/table.dart RenderTable.configuration
func (r *RenderTable) Configuration() painting.ImageConfiguration { return r.configuration }

// SetConfiguration marks the table as needing paint when the value changes.
//
// Flutter: rendering/table.dart RenderTable.configuration
func (r *RenderTable) SetConfiguration(value painting.ImageConfiguration) {
	if value == r.configuration {
		return
	}
	r.configuration = value
	r.MarkNeedsPaint()
}

// DefaultVerticalAlignment is how the cells without their own alignment are
// aligned vertically.
//
// Flutter: rendering/table.dart RenderTable.defaultVerticalAlignment
func (r *RenderTable) DefaultVerticalAlignment() TableCellVerticalAlignment {
	return r.defaultVerticalAlign
}

// SetDefaultVerticalAlignment marks the table as needing layout when the value
// changes.
//
// Flutter: rendering/table.dart RenderTable.defaultVerticalAlignment
func (r *RenderTable) SetDefaultVerticalAlignment(value TableCellVerticalAlignment) {
	if value == TableCellVerticalAlignmentUnspecified {
		value = TableCellVerticalAlignmentTop
	}
	if r.defaultVerticalAlign == value {
		return
	}
	r.defaultVerticalAlign = value
	r.MarkNeedsLayout()
}

// TextBaseline is the baseline for baseline-aligned cells. Unspecified is null.
//
// Flutter: rendering/table.dart RenderTable.textBaseline
func (r *RenderTable) TextBaseline() ui.TextBaseline { return r.textBaseline }

// SetTextBaseline marks the table as needing layout when the value changes.
//
// Flutter: rendering/table.dart RenderTable.textBaseline
func (r *RenderTable) SetTextBaseline(value ui.TextBaseline) {
	if r.textBaseline == value {
		return
	}
	r.textBaseline = value
	r.MarkNeedsLayout()
}

// SetupParentData gives the child a TableCellParentData if it has none.
//
// Flutter: rendering/table.dart RenderTable.setupParentData
func (r *RenderTable) SetupParentData(child RenderObject) {
	if _, ok := child.ParentData().(*TableCellParentData); !ok {
		child.AsRenderObject().SetParentData(&TableCellParentData{})
	}
}

// SetFlatChildren replaces the cells with a flat list of rows * columns cells,
// keeping the cells that stay where they are.
//
// Flutter: rendering/table.dart RenderTable.setFlatChildren
func (r *RenderTable) SetFlatChildren(columns int, cells []Box) {
	if columns == r.columns && len(cells) == len(r.children) && (len(cells) == 0 || &cells[0] == &r.children[0]) {
		return
	}
	if foundation.DebugMode && columns < 0 {
		panic(errors.New("TableNegativeCount", "the number of columns of a table must not be negative", "columns", columns))
	}
	if columns == 0 || len(cells) == 0 {
		if foundation.DebugMode && len(cells) != 0 {
			panic(errors.New("TableCellCountMismatch", "a table with no columns can not have cells", "cells", len(cells)))
		}
		r.columns = columns
		if len(r.children) == 0 {
			return
		}
		for _, old := range r.children {
			if !isNil(old) {
				r.DropChild(old)
			}
		}
		r.rows = 0
		r.children = nil
		r.MarkNeedsLayout()
		return
	}
	if foundation.DebugMode && len(cells)%columns != 0 {
		panic(errors.New("TableCellCountMismatch", "the number of cells of a table must be a multiple of its columns",
			"cells", len(cells), "columns", columns))
	}
	lost := map[Box]bool{}
	var lostOrder []Box
	for y := 0; y < r.rows; y++ {
		for x := 0; x < r.columns; x++ {
			old := r.children[x+y*r.columns]
			xyNew := x + y*columns
			if !isNil(old) && (x >= columns || xyNew >= len(cells) || old != cells[xyNew]) {
				lost[old] = true
				lostOrder = append(lostOrder, old)
			}
		}
	}
	for y := 0; y*columns < len(cells); y++ {
		for x := 0; x < columns; x++ {
			xyNew, xyOld := x+y*columns, x+y*r.columns
			cell := cells[xyNew]
			if !isNil(cell) && (x >= r.columns || y >= r.rows || r.children[xyOld] != cell) {
				if lost[cell] {
					delete(lost, cell)
				} else {
					r.AdoptChild(cell)
				}
			}
		}
	}
	for _, c := range lostOrder {
		if lost[c] {
			r.DropChild(c)
		}
	}
	r.columns = columns
	r.rows = len(cells) / columns
	r.children = append([]Box(nil), cells...)
	r.MarkNeedsLayout()
}

// SetChildren replaces the cells with the rows; nil removes all of them.
//
// Flutter: rendering/table.dart RenderTable.setChildren
func (r *RenderTable) SetChildren(cells [][]Box) {
	if cells == nil {
		r.SetFlatChildren(0, nil)
		return
	}
	for _, old := range r.children {
		if !isNil(old) {
			r.DropChild(old)
		}
	}
	r.children = nil
	r.columns = 0
	if len(cells) > 0 {
		r.columns = len(cells[0])
	}
	r.rows = 0
	for _, row := range cells {
		r.AddRow(append([]Box(nil), row...))
	}
	r.checkCells()
}

// AddRow adds a row of cells (entries can be nil). In debug builds it panics
// with TableCellCountMismatch if the row does not have as many cells as the
// table has columns.
//
// Flutter: rendering/table.dart RenderTable.addRow
func (r *RenderTable) AddRow(cells []Box) {
	if foundation.DebugMode && len(cells) != r.columns {
		panic(errors.New("TableCellCountMismatch", "a row must have as many cells as the table has columns",
			"cells", len(cells), "columns", r.columns))
	}
	r.checkCells()
	r.rows++
	r.children = append(r.children, cells...)
	for _, cell := range cells {
		if !isNil(cell) {
			r.AdoptChild(cell)
		}
	}
	r.MarkNeedsLayout()
}

// SetChild puts the cell at the column and row (nil removes it).
//
// Flutter: rendering/table.dart RenderTable.setChild
func (r *RenderTable) SetChild(x, y int, value Box) {
	if foundation.DebugMode && (x < 0 || x >= r.columns || y < 0 || y >= r.rows) {
		panic(errors.New("TableCellOutOfRange", "the cell is outside the table", "x", x, "y", y, "columns", r.columns, "rows", r.rows))
	}
	r.checkCells()
	xy := x + y*r.columns
	old := r.children[xy]
	if old == value {
		return
	}
	if !isNil(old) {
		r.DropChild(old)
	}
	r.children[xy] = value
	if !isNil(value) {
		r.AdoptChild(value)
	}
}

// Detach disposes the painters of the row decorations and detaches the cells.
//
// Flutter: rendering/table.dart RenderTable.detach
func (r *RenderTable) Detach() {
	r.RenderBox.Detach()
	if r.hasRowDecorations {
		r.disposeRowDecorationPainters()
		r.rowDecorationPainters = make([]painting.BoxPainter, len(r.rowDecorations))
	}
}

func (r *RenderTable) visitChildren(visitor RenderObjectVisitor) {
	r.checkCells()
	for _, c := range r.children {
		if !isNil(c) {
			visitor(c)
		}
	}
}

func (r *RenderTable) attachChildren(owner *PipelineOwner) {
	for _, c := range r.children {
		if !isNil(c) {
			c.Attach(owner)
		}
	}
}

func (r *RenderTable) detachChildren() {
	for _, c := range r.children {
		if !isNil(c) {
			c.Detach()
		}
	}
}

func (r *RenderTable) redepthChildren() {
	r.visitChildren(r.RedepthChild)
}

func (r *RenderTable) describeChildren() []foundation.DiagnosticsNode {
	if len(r.children) == 0 {
		return []foundation.DiagnosticsNode{foundation.NewDiagnosticsNodeMessage("table is empty")}
	}
	var out []foundation.DiagnosticsNode
	for y := 0; y < r.rows; y++ {
		for x := 0; x < r.columns; x++ {
			name := "child (" + strconv.Itoa(x) + ", " + strconv.Itoa(y) + ")"
			if child := r.children[x+y*r.columns]; !isNil(child) {
				out = append(out, foundation.ToDiagnosticsNode(child, foundation.DiagnosticsNodeOpts{Name: foundation.Some(name)}))
			} else {
				out = append(out, foundation.NewDiagnosticsProperty[any](name, nil,
					foundation.DiagnosticsPropertyOpts{IfNull: "is null", ShowSeparator: foundation.Some(false)}))
			}
		}
	}
	return out
}

// DebugDescribeChildren describes the cells, row by row.
//
// Flutter: rendering/table.dart RenderTable.debugDescribeChildren
func (r *RenderTable) DebugDescribeChildren() []foundation.DiagnosticsNode {
	return r.describeChildren()
}

// Column returns the cells of the column that are not nil.
//
// Flutter: rendering/table.dart RenderTable.column
func (r *RenderTable) Column(x int) []Box {
	var out []Box
	for y := 0; y < r.rows; y++ {
		if c := r.children[x+y*r.columns]; !isNil(c) {
			out = append(out, c)
		}
	}
	return out
}

// Row returns the cells of the row that are not nil.
//
// Flutter: rendering/table.dart RenderTable.row
func (r *RenderTable) Row(y int) []Box {
	var out []Box
	for xy := y * r.columns; xy < (y+1)*r.columns; xy++ {
		if c := r.children[xy]; !isNil(c) {
			out = append(out, c)
		}
	}
	return out
}

func (r *RenderTable) columnWidth(x int) TableColumnWidth {
	if w, ok := r.columnWidths[x]; ok && w != nil {
		return w
	}
	return r.defaultColumnWidth
}

func (r *RenderTable) alignmentOf(cell Box) TableCellVerticalAlignment {
	if a := cell.ParentData().(*TableCellParentData).VerticalAlignment; a != TableCellVerticalAlignmentUnspecified {
		return a
	}
	return r.defaultVerticalAlign
}

// ComputeMinIntrinsicWidth is the sum of the columns' minimum widths.
//
// Flutter: rendering/table.dart RenderTable.computeMinIntrinsicWidth
func (r *RenderTable) ComputeMinIntrinsicWidth(float64) float64 {
	r.checkCells()
	if r.rows*r.columns == 0 {
		return 0
	}
	total := 0.0
	for x := 0; x < r.columns; x++ {
		total += r.columnWidth(x).MinIntrinsicWidth(r.Column(x), math.Inf(1))
	}
	return total
}

// ComputeMaxIntrinsicWidth is the sum of the columns' maximum widths.
//
// Flutter: rendering/table.dart RenderTable.computeMaxIntrinsicWidth
func (r *RenderTable) ComputeMaxIntrinsicWidth(float64) float64 {
	r.checkCells()
	if r.rows*r.columns == 0 {
		return 0
	}
	total := 0.0
	for x := 0; x < r.columns; x++ {
		total += r.columnWidth(x).MaxIntrinsicWidth(r.Column(x), math.Inf(1))
	}
	return total
}

// ComputeMinIntrinsicHeight is the sum of the heights of the rows at the width.
//
// Flutter: rendering/table.dart RenderTable.computeMinIntrinsicHeight
func (r *RenderTable) ComputeMinIntrinsicHeight(width float64) float64 {
	r.checkCells()
	if r.rows*r.columns == 0 {
		return 0
	}
	widths := r.computeColumnWidths(NewBoxConstraintsTightForFinite(TightForFiniteOpts{Width: foundation.Some(width)}))
	rowTop := 0.0
	for y := 0; y < r.rows; y++ {
		rowHeight := 0.0
		for x := 0; x < r.columns; x++ {
			if child := r.children[x+y*r.columns]; !isNil(child) {
				rowHeight = math.Max(rowHeight, child.GetMaxIntrinsicHeight(widths[x]))
			}
		}
		rowTop += rowHeight
	}
	return rowTop
}

// ComputeMaxIntrinsicHeight is the minimum intrinsic height.
//
// Flutter: rendering/table.dart RenderTable.computeMaxIntrinsicHeight
func (r *RenderTable) ComputeMaxIntrinsicHeight(width float64) float64 {
	return r.GetMinIntrinsicHeight(width)
}

// ComputeDistanceToActualBaseline is the baseline of the first row, if a cell in
// it is aligned to the baseline.
//
// Flutter: rendering/table.dart RenderTable.computeDistanceToActualBaseline
func (r *RenderTable) ComputeDistanceToActualBaseline(ui.TextBaseline) foundation.Opt[float64] {
	if foundation.DebugMode && r.DebugNeedsLayout() {
		panic(errors.New("BaselineNeedsLayout", "the baseline was read from a table that needs layout",
			"object", foundation.DescribeIdentity(r)))
	}
	return r.baselineDistance
}

func checkColumnValue(v float64, what string, x int, positive bool) {
	if !foundation.DebugMode {
		return
	}
	if math.IsInf(v, 0) || math.IsNaN(v) || v < 0 || (positive && v == 0) {
		panic(errors.New("TableColumnWidthInvalid", "a TableColumnWidth gave an invalid "+what, "column", x, "value", v))
	}
}

// computeColumnWidths works out the width of every column for the constraints.
//
// Flutter: rendering/table.dart RenderTable._computeColumnWidths
func (r *RenderTable) computeColumnWidths(constraints BoxConstraints) []float64 {
	r.checkCells()
	columns := r.columns
	widths := make([]float64, columns)
	minWidths := make([]float64, columns)
	flexes := make([]foundation.Opt[float64], columns)
	tableWidth, unflexedTableWidth, totalFlex := 0.0, 0.0, 0.0
	for x := 0; x < columns; x++ {
		columnWidth := r.columnWidth(x)
		cells := r.Column(x)
		maxIntrinsic := columnWidth.MaxIntrinsicWidth(cells, constraints.MaxWidth)
		checkColumnValue(maxIntrinsic, "maximum intrinsic width", x, false)
		widths[x] = maxIntrinsic
		tableWidth += maxIntrinsic
		minIntrinsic := columnWidth.MinIntrinsicWidth(cells, constraints.MaxWidth)
		checkColumnValue(minIntrinsic, "minimum intrinsic width", x, false)
		minWidths[x] = minIntrinsic
		if flex, ok := columnWidth.Flex(cells).Get(); ok {
			checkColumnValue(flex, "flex", x, true)
			flexes[x] = foundation.Some(flex)
			totalFlex += flex
		} else {
			unflexedTableWidth += maxIntrinsic
		}
	}
	maxWidthConstraint, minWidthConstraint := constraints.MaxWidth, constraints.MinWidth
	if totalFlex > 0 {
		targetWidth := minWidthConstraint
		if !math.IsInf(maxWidthConstraint, 0) {
			targetWidth = maxWidthConstraint
		}
		if tableWidth < targetWidth {
			remainingWidth := targetWidth - unflexedTableWidth
			for x := 0; x < columns; x++ {
				if flex, ok := flexes[x].Get(); ok {
					flexedWidth := remainingWidth * flex / totalFlex
					if widths[x] < flexedWidth {
						tableWidth += flexedWidth - widths[x]
						widths[x] = flexedWidth
					}
				}
			}
		}
	} else if tableWidth < minWidthConstraint {
		delta := (minWidthConstraint - tableWidth) / float64(columns)
		for x := 0; x < columns; x++ {
			widths[x] += delta
		}
		tableWidth = minWidthConstraint
	}
	if tableWidth > maxWidthConstraint {
		deficit := tableWidth - maxWidthConstraint
		availableColumns := columns
		for deficit > foundation.PrecisionErrorTolerance && totalFlex > foundation.PrecisionErrorTolerance {
			newTotalFlex := 0.0
			for x := 0; x < columns; x++ {
				flex, ok := flexes[x].Get()
				if !ok {
					continue
				}
				newWidth := widths[x] - deficit*flex/totalFlex
				if newWidth <= minWidths[x] {
					deficit -= widths[x] - minWidths[x]
					widths[x] = minWidths[x]
					flexes[x] = foundation.None[float64]()
					availableColumns--
				} else {
					deficit -= widths[x] - newWidth
					widths[x] = newWidth
					newTotalFlex += flex
				}
			}
			totalFlex = newTotalFlex
		}
		for deficit > foundation.PrecisionErrorTolerance && availableColumns > 0 {
			delta := deficit / float64(availableColumns)
			newAvailable := 0
			for x := 0; x < columns; x++ {
				availableDelta := widths[x] - minWidths[x]
				if availableDelta > 0 {
					if availableDelta <= delta {
						deficit -= widths[x] - minWidths[x]
						widths[x] = minWidths[x]
					} else {
						deficit -= delta
						widths[x] -= delta
						newAvailable++
					}
				}
			}
			availableColumns = newAvailable
		}
	}
	return widths
}

// GetRowBox is the box of the row, in the table's coordinates. It must be called
// after layout.
//
// Flutter: rendering/table.dart RenderTable.getRowBox
func (r *RenderTable) GetRowBox(row int) ui.Rect {
	if foundation.DebugMode {
		if row < 0 || row >= r.rows {
			panic(errors.New("TableCellOutOfRange", "the row is outside the table", "row", row, "rows", r.rows))
		}
		if r.DebugNeedsLayout() {
			panic(errors.New("TableNeedsLayout", "the row box was read from a table that needs layout",
				"object", foundation.DescribeIdentity(r)))
		}
	}
	return ui.NewRectFromLTRB(0, r.rowTops[row], r.Size().Width, r.rowTops[row+1])
}

// ComputeDryBaseline is the largest baseline among the baseline-aligned cells of
// the first row.
//
// Flutter: rendering/table.dart RenderTable.computeDryBaseline
func (r *RenderTable) ComputeDryBaseline(constraints BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	if r.rows*r.columns == 0 {
		return foundation.None[float64]()
	}
	widths := r.computeColumnWidths(constraints)
	result := foundation.None[float64]()
	for col := 0; col < r.columns; col++ {
		child := r.children[col]
		if isNil(child) || r.alignmentOf(child) != TableCellVerticalAlignmentBaseline {
			continue
		}
		childConstraints := NewBoxConstraintsTightFor(TightForOpts{Width: foundation.Some(widths[col])})
		if b, ok := child.GetDryBaseline(childConstraints, baseline).Get(); ok {
			if cur, has := result.Get(); !has || cur < b {
				result = foundation.Some(b)
			}
		}
	}
	return result
}

// ComputeDryLayout is the size the constraints would give.
//
// Flutter: rendering/table.dart RenderTable.computeDryLayout
func (r *RenderTable) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	if r.rows*r.columns == 0 {
		return constraints.Constrain(ui.Size{})
	}
	widths := r.computeColumnWidths(constraints)
	tableWidth := 0.0
	for _, w := range widths {
		tableWidth += w
	}
	rowTop := 0.0
	for y := 0; y < r.rows; y++ {
		rowHeight := 0.0
		for x := 0; x < r.columns; x++ {
			child := r.children[x+y*r.columns]
			if isNil(child) {
				continue
			}
			switch r.alignmentOf(child) {
			case TableCellVerticalAlignmentBaseline:
				r.DebugCannotComputeDryLayout("TableCellVerticalAlignment.baseline requires a full layout for baseline metrics to be available.")
				return ui.Size{}
			case TableCellVerticalAlignmentFill:
			default:
				childSize := child.GetDryLayout(NewBoxConstraintsTightFor(TightForOpts{Width: foundation.Some(widths[x])}))
				rowHeight = math.Max(rowHeight, childSize.Height)
			}
		}
		rowTop += rowHeight
	}
	return constraints.Constrain(ui.Size{Width: tableWidth, Height: rowTop})
}

// PerformLayout works out the column widths, lays the cells out row by row and
// positions them.
//
// Flutter: rendering/table.dart RenderTable.performLayout
func (r *RenderTable) PerformLayout() {
	constraints := r.BoxConstraints()
	rows, columns := r.rows, r.columns
	r.checkCells()
	if rows*columns == 0 {
		r.tableWidth = 0
		r.SetSize(constraints.Constrain(ui.Size{}))
		return
	}
	widths := r.computeColumnWidths(constraints)
	positions := make([]float64, columns)
	if r.textDirection == ui.TextDirectionRTL {
		for x := columns - 2; x >= 0; x-- {
			positions[x] = positions[x+1] + widths[x+1]
		}
		r.columnLefts = make([]float64, columns)
		for i, p := range positions {
			r.columnLefts[columns-1-i] = p
		}
		r.tableWidth = positions[0] + widths[0]
	} else {
		for x := 1; x < columns; x++ {
			positions[x] = positions[x-1] + widths[x-1]
		}
		r.columnLefts = positions
		r.tableWidth = positions[columns-1] + widths[columns-1]
	}
	r.rowTops = r.rowTops[:0]
	r.baselineDistance = foundation.None[float64]()
	rowTop := 0.0
	for y := 0; y < rows; y++ {
		r.rowTops = append(r.rowTops, rowTop)
		rowHeight := 0.0
		haveBaseline := false
		beforeBaseline, afterBaseline := 0.0, 0.0
		baselines := make([]float64, columns)
		for x := 0; x < columns; x++ {
			child := r.children[x+y*columns]
			if isNil(child) {
				continue
			}
			data := child.ParentData().(*TableCellParentData)
			data.X, data.Y = foundation.Some(x), foundation.Some(y)
			switch r.alignmentOf(child) {
			case TableCellVerticalAlignmentBaseline:
				if foundation.DebugMode && r.textBaseline == ui.TextBaselineUnspecified {
					panic(errors.New("TableBaselineMissing", "An explicit textBaseline is required when using baseline alignment.",
						"object", foundation.DescribeIdentity(r)))
				}
				child.Layout(NewBoxConstraintsTightFor(TightForOpts{Width: foundation.Some(widths[x])}), LayoutOpts{ParentUsesSize: true})
				if b, ok := child.GetDistanceToBaseline(r.textBaseline, BaselineOpts{OnlyReal: true}).Get(); ok {
					beforeBaseline = math.Max(beforeBaseline, b)
					afterBaseline = math.Max(afterBaseline, child.Size().Height-b)
					baselines[x] = b
					haveBaseline = true
				} else {
					rowHeight = math.Max(rowHeight, child.Size().Height)
					data.Offset = ui.Offset{Dx: positions[x], Dy: rowTop}
				}
			case TableCellVerticalAlignmentFill:
			default:
				child.Layout(NewBoxConstraintsTightFor(TightForOpts{Width: foundation.Some(widths[x])}), LayoutOpts{ParentUsesSize: true})
				rowHeight = math.Max(rowHeight, child.Size().Height)
			}
		}
		if haveBaseline {
			if y == 0 {
				r.baselineDistance = foundation.Some(beforeBaseline)
			}
			rowHeight = math.Max(rowHeight, beforeBaseline+afterBaseline)
		}
		for x := 0; x < columns; x++ {
			child := r.children[x+y*columns]
			if isNil(child) {
				continue
			}
			data := child.ParentData().(*TableCellParentData)
			switch r.alignmentOf(child) {
			case TableCellVerticalAlignmentBaseline:
				data.Offset = ui.Offset{Dx: positions[x], Dy: rowTop + beforeBaseline - baselines[x]}
			case TableCellVerticalAlignmentTop:
				data.Offset = ui.Offset{Dx: positions[x], Dy: rowTop}
			case TableCellVerticalAlignmentMiddle:
				data.Offset = ui.Offset{Dx: positions[x], Dy: rowTop + (rowHeight-child.Size().Height)/2}
			case TableCellVerticalAlignmentBottom:
				data.Offset = ui.Offset{Dx: positions[x], Dy: rowTop + rowHeight - child.Size().Height}
			default: // fill, intrinsicHeight
				child.Layout(NewBoxConstraintsTightFor(TightForOpts{Width: foundation.Some(widths[x]), Height: foundation.Some(rowHeight)}))
				data.Offset = ui.Offset{Dx: positions[x], Dy: rowTop}
			}
		}
		rowTop += rowHeight
	}
	r.rowTops = append(r.rowTops, rowTop)
	r.SetSize(constraints.Constrain(ui.Size{Width: r.tableWidth, Height: rowTop}))
}

// HitTestChildren hit tests the cells from the last to the first.
//
// Flutter: rendering/table.dart RenderTable.hitTestChildren
func (r *RenderTable) HitTestChildren(result *BoxHitTestResult, position ui.Offset) bool {
	r.checkCells()
	for index := len(r.children) - 1; index >= 0; index-- {
		child := r.children[index]
		if isNil(child) {
			continue
		}
		offset := BoxParentDataOf(child).Offset
		if result.AddWithPaintOffset(&offset, position, func(result *BoxHitTestResult, transformed ui.Offset) bool {
			return child.HitTest(result, transformed)
		}) {
			return true
		}
	}
	return false
}

// Paint paints the row decorations, the cells and the border.
//
// Flutter: rendering/table.dart RenderTable.paint
func (r *RenderTable) Paint(context *PaintingContext, offset ui.Offset) {
	r.checkCells()
	if r.rows*r.columns == 0 {
		if r.border != nil {
			r.border.Paint(context.Canvas(), ui.NewRectFromLTWH(offset.Dx, offset.Dy, r.tableWidth, 0), nil, nil)
		}
		return
	}
	if r.hasRowDecorations {
		canvas := context.Canvas()
		for y := 0; y < r.rows && y < len(r.rowDecorations); y++ {
			dec := r.rowDecorations[y]
			if isNil(dec) {
				continue
			}
			if isNil(r.rowDecorationPainters[y]) {
				r.rowDecorationPainters[y] = dec.CreateBoxPainter(r.MarkNeedsPaint)
			}
			r.rowDecorationPainters[y].Paint(canvas, ui.Offset{Dx: offset.Dx, Dy: offset.Dy + r.rowTops[y]},
				r.configuration.CopyWith(painting.ImageConfigurationCopy{
					Size: foundation.Some(ui.Size{Width: r.Size().Width, Height: r.rowTops[y+1] - r.rowTops[y]}),
				}))
		}
	}
	for _, child := range r.children {
		if !isNil(child) {
			context.PaintChild(child, BoxParentDataOf(child).Offset.Add(offset))
		}
	}
	if r.border != nil {
		borderRect := ui.NewRectFromLTWH(offset.Dx, offset.Dy, r.tableWidth, r.rowTops[len(r.rowTops)-1])
		r.border.Paint(context.Canvas(), borderRect, r.rowTops[1:len(r.rowTops)-1], r.columnLefts[1:])
	}
}

// DebugFillProperties adds the border, the column widths, the size and the
// offsets of the rows and columns.
//
// Flutter: rendering/table.dart RenderTable.debugFillProperties
func (r *RenderTable) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderBox.DebugFillProperties(b)
	b.Add(foundation.NewDiagnosticsPropertyOpt("border", optPtr(r.border), foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull}))
	keys := make([]int, 0, len(r.columnWidths))
	for k := range r.columnWidths {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	desc := "{"
	for i, k := range keys {
		if i > 0 {
			desc += ", "
		}
		desc += strconv.Itoa(k) + ": " + r.columnWidths[k].String()
	}
	desc += "}"
	b.Add(foundation.NewDiagnosticsProperty("specified column widths", desc, foundation.DiagnosticsPropertyOpts{Level: levelHiddenIf(len(keys) == 0)}))
	b.Add(foundation.NewDiagnosticsProperty("default column width", r.defaultColumnWidth.String()))
	b.Add(foundation.NewMessageProperty("table size", strconv.Itoa(r.columns)+"×"+strconv.Itoa(r.rows)))
	var lefts []string
	if r.columnLefts != nil {
		lefts = []string{}
		for _, v := range r.columnLefts {
			lefts = append(lefts, foundation.DebugFormatDouble(foundation.Some(v)))
		}
	}
	b.Add(foundation.NewIterableProperty("column offsets", lefts, foundation.IterablePropertyOpts{IfNull: "unknown"}))
	tops := []string{}
	for _, v := range r.rowTops {
		tops = append(tops, foundation.DebugFormatDouble(foundation.Some(v)))
	}
	b.Add(foundation.NewIterableProperty("row offsets", tops, foundation.IterablePropertyOpts{IfNull: "unknown"}))
}

func optPtr[T any](p *T) foundation.Opt[*T] {
	if p == nil {
		return foundation.None[*T]()
	}
	return foundation.Some(p)
}

func levelHiddenIf(hidden bool) foundation.DiagnosticLevel {
	if hidden {
		return foundation.DiagnosticLevelHidden
	}
	return foundation.DiagnosticLevelInfo
}
