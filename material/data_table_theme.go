// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// DataTableThemeData defines default property values for descendant DataTable widgets. Every
// field that is not set leaves the choice to the widget's own defaults.
//
// Flutter: material/data_table_theme.dart DataTableThemeData
//
//glutter:copywith value
type DataTableThemeData struct {
	// Decoration is the background of the table.
	Decoration painting.Decoration
	// DataRowColor is the color of the data rows, for each state.
	DataRowColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// DataRowMinHeight is the minimum height of the data rows.
	DataRowMinHeight foundation.Opt[float64]
	// DataRowMaxHeight is the maximum height of the data rows.
	DataRowMaxHeight foundation.Opt[float64]
	// DataTextStyle is the style of the text of the data cells.
	DataTextStyle *painting.TextStyle
	// HeadingRowColor is the color of the heading row, for each state.
	HeadingRowColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// HeadingRowHeight is the height of the heading row.
	HeadingRowHeight foundation.Opt[float64]
	// HeadingTextStyle is the style of the text of the heading cells.
	HeadingTextStyle *painting.TextStyle
	// HorizontalMargin is the margin at the two sides of the table.
	HorizontalMargin foundation.Opt[float64]
	// ColumnSpacing is the space between the columns.
	ColumnSpacing foundation.Opt[float64]
	// DividerThickness is the thickness of the lines between the rows.
	DividerThickness foundation.Opt[float64]
	// CheckboxHorizontalMargin is the margin around the checkboxes.
	CheckboxHorizontalMargin foundation.Opt[float64]
	// HeadingCellCursor is the mouse cursor over a heading cell, for each state.
	HeadingCellCursor widgets.WidgetStateProperty[services.MouseCursor]
	// DataRowCursor is the mouse cursor over a data row, for each state.
	DataRowCursor widgets.WidgetStateProperty[services.MouseCursor]
	// HeadingRowAlignment is the alignment of the content of the heading cells.
	HeadingRowAlignment rendering.MainAxisAlignment
}

// DataTableThemeDataOpts are the parameters of NewDataTableThemeData: the fields of the data.
//
// Flutter: material/data_table_theme.dart DataTableThemeData.new
type DataTableThemeDataOpts = DataTableThemeData

// NewDataTableThemeData creates the data; every field is optional.
//
// Flutter: material/data_table_theme.dart DataTableThemeData.new
func NewDataTableThemeData(opts ...DataTableThemeDataOpts) DataTableThemeData {
	return firstOpts(opts)
}

// DataTableThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/data_table_theme.dart DataTableThemeData.lerp
func DataTableThemeDataLerp(a, b DataTableThemeData, t float64) DataTableThemeData {
	return DataTableThemeData{
		Decoration:               painting.DecorationLerp(a.Decoration, b.Decoration, t),
		DataRowColor:             lerpStateOptColor(a.DataRowColor, b.DataRowColor, t),
		DataRowMinHeight:         lerpOptDouble(a.DataRowMinHeight, b.DataRowMinHeight, t),
		DataRowMaxHeight:         lerpOptDouble(a.DataRowMaxHeight, b.DataRowMaxHeight, t),
		DataTextStyle:            painting.TextStyleLerp(a.DataTextStyle, b.DataTextStyle, t),
		HeadingRowColor:          lerpStateOptColor(a.HeadingRowColor, b.HeadingRowColor, t),
		HeadingRowHeight:         lerpOptDouble(a.HeadingRowHeight, b.HeadingRowHeight, t),
		HeadingTextStyle:         painting.TextStyleLerp(a.HeadingTextStyle, b.HeadingTextStyle, t),
		HorizontalMargin:         lerpOptDouble(a.HorizontalMargin, b.HorizontalMargin, t),
		ColumnSpacing:            lerpOptDouble(a.ColumnSpacing, b.ColumnSpacing, t),
		DividerThickness:         lerpOptDouble(a.DividerThickness, b.DividerThickness, t),
		CheckboxHorizontalMargin: lerpOptDouble(a.CheckboxHorizontalMargin, b.CheckboxHorizontalMargin, t),
		HeadingCellCursor:        lerpFirstOrSecond(a.HeadingCellCursor, b.HeadingCellCursor, t),
		DataRowCursor:            lerpFirstOrSecond(a.DataRowCursor, b.DataRowCursor, t),
		HeadingRowAlignment:      lerpFirstOrSecond(a.HeadingRowAlignment, b.HeadingRowAlignment, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/data_table_theme.dart DataTableThemeData.==
func (d DataTableThemeData) Equal(o DataTableThemeData) bool {
	return foundation.ValuesEqual(d.Decoration, o.Decoration) &&
		foundation.ValuesEqual(d.DataRowColor, o.DataRowColor) &&
		d.DataRowMinHeight == o.DataRowMinHeight &&
		d.DataRowMaxHeight == o.DataRowMaxHeight &&
		textStyleEqual(d.DataTextStyle, o.DataTextStyle) &&
		foundation.ValuesEqual(d.HeadingRowColor, o.HeadingRowColor) &&
		d.HeadingRowHeight == o.HeadingRowHeight &&
		textStyleEqual(d.HeadingTextStyle, o.HeadingTextStyle) &&
		d.HorizontalMargin == o.HorizontalMargin &&
		d.ColumnSpacing == o.ColumnSpacing &&
		d.DividerThickness == o.DividerThickness &&
		d.CheckboxHorizontalMargin == o.CheckboxHorizontalMargin &&
		foundation.ValuesEqual(d.HeadingCellCursor, o.HeadingCellCursor) &&
		foundation.ValuesEqual(d.DataRowCursor, o.DataRowCursor) &&
		d.HeadingRowAlignment == o.HeadingRowAlignment
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (DataTableThemeData) ToStringShort() string { return "DataTableThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d DataTableThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/data_table_theme.dart DataTableThemeData.debugFillProperties
func (d DataTableThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(objectProp("decoration", d.Decoration))
	p.Add(objectProp("dataRowColor", propOrNil(d.DataRowColor)))
	p.Add(doubleProp("dataRowMinHeight", d.DataRowMinHeight))
	p.Add(doubleProp("dataRowMaxHeight", d.DataRowMaxHeight))
	p.Add(objectProp("dataTextStyle", d.DataTextStyle))
	p.Add(objectProp("headingRowColor", propOrNil(d.HeadingRowColor)))
	p.Add(doubleProp("headingRowHeight", d.HeadingRowHeight))
	p.Add(objectProp("headingTextStyle", d.HeadingTextStyle))
	p.Add(doubleProp("horizontalMargin", d.HorizontalMargin))
	p.Add(doubleProp("columnSpacing", d.ColumnSpacing))
	p.Add(doubleProp("dividerThickness", d.DividerThickness))
	p.Add(doubleProp("checkboxHorizontalMargin", d.CheckboxHorizontalMargin))
	p.Add(objectProp("headingCellCursor", propOrNil(d.HeadingCellCursor)))
	p.Add(objectProp("dataRowCursor", propOrNil(d.DataRowCursor)))
	p.Add(objectProp("headingRowAlignment", nilIfZero(d.HeadingRowAlignment)))
}
