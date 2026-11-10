package cupertino

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

var (
	formRowDefaultPadding             = painting.NewEdgeInsetsDirectionalFromSTEB(20, 6, 6, 6)
	formSectionInsetGroupedRowsMargin = painting.NewEdgeInsetsDirectionalFromSTEB(20, 0, 20, 10)
)

// CupertinoFormRowOpts are the optional parameters of NewCupertinoFormRow.
//
// Flutter: cupertino/form_row.dart CupertinoFormRow.new
type CupertinoFormRowOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Prefix is shown at the start, before the child; it is usually a label.
	Prefix widgets.Widget
	// Padding is the padding around the content; the default is 20, 6, 6, 6 (start, top, end, bottom).
	Padding painting.EdgeInsetsGeometry
	// Helper is shown under the row, at the start.
	Helper widgets.Widget
	// Error is shown under the row, in red; it is usually the message of the validator.
	Error widgets.Widget
}

// CupertinoFormRow is an iOS style form row: a prefix and a child with an optional helper and error below.
//
// Flutter: cupertino/form_row.dart CupertinoFormRow
type CupertinoFormRow struct {
	widgets.StatelessWidget
	child widgets.Widget
	o     CupertinoFormRowOpts
}

// NewCupertinoFormRow creates a form row of the child.
//
// Flutter: cupertino/form_row.dart CupertinoFormRow.new
func NewCupertinoFormRow(child widgets.Widget, opts ...CupertinoFormRowOpts) *CupertinoFormRow {
	o := firstOf(opts)
	return &CupertinoFormRow{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), child: child, o: o}
}

// Child is the widget of the row.
//
// Flutter: cupertino/form_row.dart CupertinoFormRow.child
func (w *CupertinoFormRow) Child() widgets.Widget { return w.child }

// Build builds the row.
//
// Flutter: cupertino/form_row.dart CupertinoFormRow.build
func (w *CupertinoFormRow) Build(bc widgets.BuildContext) widgets.Widget {
	textStyle := CupertinoThemeOf(bc).TextTheme().TextStyle()
	var first []widgets.Widget
	if w.o.Prefix != nil {
		first = append(first, widgets.NewDefaultTextStyle(textStyle, w.o.Prefix))
	}
	first = append(first, widgets.NewFlexible(widgets.NewAlign(w.child, widgets.AlignOpts{Alignment: painting.AlignmentDirectionalCenterEnd})))
	rows := []widgets.Widget{widgets.NewRow(first, widgets.RowOpts{MainAxisAlignment: rendering.MainAxisAlignmentSpaceBetween})}
	if w.o.Helper != nil {
		rows = append(rows, widgets.NewAlign(widgets.NewDefaultTextStyle(textStyle, w.o.Helper), widgets.AlignOpts{Alignment: painting.AlignmentDirectionalCenterStart}))
	}
	if w.o.Error != nil {
		rows = append(rows, widgets.NewAlign(widgets.NewDefaultTextStyle(painting.NewTextStyle(painting.TextStyleOpts{
			Color: foundation.Some(CupertinoColors.DestructiveRed.Color), FontWeight: ui.FontWeightW500}), w.o.Error),
			widgets.AlignOpts{Alignment: painting.AlignmentDirectionalCenterStart}))
	}
	padding := w.o.Padding
	if padding == nil {
		padding = formRowDefaultPadding
	}
	return widgets.NewPadding(padding, widgets.NewColumn(rows))
}

// CupertinoFormSectionOpts are the optional parameters of NewCupertinoFormSection and NewCupertinoFormSectionInsetGrouped.
//
// Flutter: cupertino/form_section.dart CupertinoFormSection.new
type CupertinoFormSectionOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Header is shown above the rows.
	Header widgets.Widget
	// Footer is shown below the rows.
	Footer widgets.Widget
	// Margin is the space around the rows; the default is none for a base section and 20, 0, 20, 10 for an inset grouped one.
	Margin painting.EdgeInsetsGeometry
	// BackgroundColor is the background of the section; the default is CupertinoColors.SystemGroupedBackground.
	BackgroundColor ui.ColorValue
	// Decoration is the decoration of the group of rows.
	Decoration painting.Decoration
	// ClipBehavior is how the rows are clipped; the default is none.
	ClipBehavior ui.Clip
}

// CupertinoFormSection is an iOS style form section: a CupertinoListSection of form rows.
//
// Flutter: cupertino/form_section.dart CupertinoFormSection
type CupertinoFormSection struct {
	widgets.StatelessWidget
	sectionType CupertinoListSectionType
	children    []widgets.Widget
	o           CupertinoFormSectionOpts
}

// NewCupertinoFormSection creates a base form section of the children. In debug builds it panics with the tag
// CupertinoFormSectionEmpty when there are no children.
//
// Flutter: cupertino/form_section.dart CupertinoFormSection.new
func NewCupertinoFormSection(children []widgets.Widget, opts ...CupertinoFormSectionOpts) *CupertinoFormSection {
	return newFormSection(CupertinoListSectionTypeBase, children, firstOf(opts))
}

// NewCupertinoFormSectionInsetGrouped creates an inset grouped form section of the children. It panics like NewCupertinoFormSection.
//
// Flutter: cupertino/form_section.dart CupertinoFormSection.insetGrouped
func NewCupertinoFormSectionInsetGrouped(children []widgets.Widget, opts ...CupertinoFormSectionOpts) *CupertinoFormSection {
	return newFormSection(CupertinoListSectionTypeInsetGrouped, children, firstOf(opts))
}

func newFormSection(t CupertinoListSectionType, children []widgets.Widget, o CupertinoFormSectionOpts) *CupertinoFormSection {
	if foundation.DebugMode && len(children) == 0 {
		panic(errors.New(tagCupertinoFormSectionEmpty, "a CupertinoFormSection needs at least one child"))
	}
	if o.Margin == nil {
		if t == CupertinoListSectionTypeBase {
			o.Margin = painting.EdgeInsetsZero
		} else {
			o.Margin = formSectionInsetGroupedRowsMargin
		}
	}
	if o.BackgroundColor == nil {
		o.BackgroundColor = CupertinoColors.SystemGroupedBackground
	}
	return &CupertinoFormSection{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), sectionType: t, children: children, o: o}
}

// Type is the type of the section.
//
// Flutter: cupertino/form_section.dart CupertinoFormSection._type
func (w *CupertinoFormSection) Type() CupertinoListSectionType { return w.sectionType }

// Build builds the section.
//
// Flutter: cupertino/form_section.dart CupertinoFormSection.build
func (w *CupertinoFormSection) Build(bc widgets.BuildContext) widgets.Widget {
	style := painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(13.0), Color: foundation.Some(CupertinoDynamicColorResolve(CupertinoColors.SecondaryLabel, bc))})
	var header, footer widgets.Widget
	if w.o.Header != nil {
		header = widgets.NewDefaultTextStyle(style, w.o.Header)
	}
	if w.o.Footer != nil {
		footer = widgets.NewDefaultTextStyle(style, w.o.Footer)
	}
	o := CupertinoListSectionOpts{Header: header, Footer: footer, Margin: w.o.Margin, BackgroundColor: w.o.BackgroundColor, Decoration: w.o.Decoration,
		ClipBehavior: w.o.ClipBehavior, NotHasLeading: true, Children: w.children}
	if w.o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipNone
	}
	if w.sectionType == CupertinoListSectionTypeBase {
		return NewCupertinoListSection(o)
	}
	return NewCupertinoListSectionInsetGrouped(o)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCupertinoFormSectionEmpty = "CupertinoFormSectionEmpty"
)
