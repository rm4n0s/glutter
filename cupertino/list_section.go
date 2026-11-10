package cupertino

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	listSectionMarginTop                          = 22.0
	listSectionBaseDividerMargin                  = 20.0
	listSectionBaseAdditionalDividerMargin        = 44.0
	listSectionInsetDividerMargin                 = 14.0
	listSectionInsetAdditionalDividerMargin       = 42.0
	listSectionInsetAdditionalDividerMarginNoLead = 14.0
	listSectionInsetGroupedRadius                 = 10.0
)

var (
	listSectionDefaultHeaderMargin          = painting.NewEdgeInsetsDirectionalFromSTEB(20, 0, 20, 6)
	listSectionInsetGroupedHeaderMargin     = painting.NewEdgeInsetsDirectionalFromSTEB(20, 16, 20, 6)
	listSectionDefaultFooterMargin          = painting.NewEdgeInsetsDirectionalFromSTEB(20, 0, 20, 0)
	listSectionInsetGroupedFooterMargin     = painting.NewEdgeInsetsDirectionalFromSTEB(20, 0, 20, 10)
	listSectionDefaultRowsMargin            = painting.EdgeInsets{Bottom: 8}
	listSectionInsetGroupedRowsMargin       = painting.NewEdgeInsetsDirectionalFromSTEB(20, 20, 20, 10)
	listSectionInsetGroupedRowsMarginHeader = painting.NewEdgeInsetsDirectionalFromSTEB(20, 0, 20, 10)
)

// listSectionHeaderFooterColor is the color of the header and the footer of a base section.
//
// Flutter: cupertino/list_section.dart _kHeaderFooterColor
func listSectionHeaderFooterColor() CupertinoDynamicColor {
	rgb := func(r, g, b uint32) ui.Color { return ui.NewColor(0xFF000000 | r<<16 | g<<8 | b) }
	return NewCupertinoDynamicColor(rgb(108, 108, 108), rgb(142, 142, 146), rgb(74, 74, 77), rgb(176, 176, 183),
		rgb(108, 108, 108), rgb(142, 142, 146), rgb(108, 108, 108), rgb(142, 142, 146))
}

// CupertinoListSectionType is the type of a CupertinoListSection.
//
// Flutter: cupertino/list_section.dart CupertinoListSectionType
type CupertinoListSectionType int

const (
	// CupertinoListSectionTypeUnspecified is the zero value.
	//
	// Flutter: no counterpart
	CupertinoListSectionTypeUnspecified CupertinoListSectionType = iota
	// CupertinoListSectionTypeBase is a section with long dividers on its top and bottom edges.
	//
	// Flutter: cupertino/list_section.dart CupertinoListSectionType.base
	CupertinoListSectionTypeBase
	// CupertinoListSectionTypeInsetGrouped is a section whose rows are in a rounded card inset from the screen's edges.
	//
	// Flutter: cupertino/list_section.dart CupertinoListSectionType.insetGrouped
	CupertinoListSectionTypeInsetGrouped
)

// CupertinoListSectionOpts are the optional parameters of NewCupertinoListSection and NewCupertinoListSectionInsetGrouped.
//
// Flutter: cupertino/list_section.dart CupertinoListSection.new
type CupertinoListSectionOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Children are the rows of the section.
	Children []widgets.Widget
	// Header is shown above the rows.
	Header widgets.Widget
	// Footer is shown below the rows.
	Footer widgets.Widget
	// Margin is the space around the rows; the default is that of the type of the section.
	Margin painting.EdgeInsetsGeometry
	// BackgroundColor is the background of the section; the default is CupertinoColors.SystemGroupedBackground.
	BackgroundColor ui.ColorValue
	// Decoration is the decoration of the group of rows; the default is a card of CupertinoColors.SecondarySystemGroupedBackground.
	Decoration painting.Decoration
	// ClipBehavior is how the rows are clipped; the default is none for a base section and hard edge for an inset grouped one.
	ClipBehavior ui.Clip
	// DividerMargin is the margin of the dividers at the start; the default is that of the type of the section.
	DividerMargin foundation.Opt[float64]
	// AdditionalDividerMargin is added to DividerMargin for the dividers between the rows; the default follows NotHasLeading.
	AdditionalDividerMargin foundation.Opt[float64]
	// TopMargin is the space above a base section; the default is 22.
	TopMargin foundation.Opt[float64]
	// NotHasLeading says that the rows have no leading widget, so the dividers have less margin.
	NotHasLeading bool
	// SeparatorColor is the color of the dividers; the default is CupertinoColors.Separator.
	SeparatorColor ui.ColorValue
}

// CupertinoListSection is an iOS style list section: a group of rows, with a header and a footer, separated by dividers.
//
// Flutter: cupertino/list_section.dart CupertinoListSection
type CupertinoListSection struct {
	widgets.StatelessWidget
	sectionType             CupertinoListSectionType
	margin                  painting.EdgeInsetsGeometry
	dividerMargin           float64
	additionalDividerMargin float64
	topMargin               foundation.Opt[float64]
	clip                    ui.Clip
	o                       CupertinoListSectionOpts
}

func newListSection(sectionType CupertinoListSectionType, o CupertinoListSectionOpts) *CupertinoListSection {
	if foundation.DebugMode && len(o.Children) == 0 && o.Header == nil {
		panic(errors.New(tagCupertinoListSectionEmpty, "a CupertinoListSection needs children or a header"))
	}
	s := &CupertinoListSection{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), sectionType: sectionType, o: o, topMargin: o.TopMargin}
	if o.BackgroundColor == nil {
		s.o.BackgroundColor = CupertinoColors.SystemGroupedBackground
	}
	if sectionType == CupertinoListSectionTypeBase {
		s.margin = o.Margin
		if s.margin == nil {
			s.margin = listSectionDefaultRowsMargin
		}
		s.clip = o.ClipBehavior
		if s.clip == ui.ClipUnspecified {
			s.clip = ui.ClipNone
		}
		s.dividerMargin = o.DividerMargin.OrElse(listSectionBaseDividerMargin)
		additional := 0.0
		if !o.NotHasLeading {
			additional = listSectionBaseAdditionalDividerMargin
		}
		s.additionalDividerMargin = o.AdditionalDividerMargin.OrElse(additional)
		if !s.topMargin.IsSome() {
			s.topMargin = foundation.Some(listSectionMarginTop)
		}
		return s
	}
	s.margin = o.Margin
	if s.margin == nil {
		if o.Header == nil {
			s.margin = listSectionInsetGroupedRowsMargin
		} else {
			s.margin = listSectionInsetGroupedRowsMarginHeader
		}
	}
	s.clip = o.ClipBehavior
	if s.clip == ui.ClipUnspecified {
		s.clip = ui.ClipHardEdge
	}
	s.dividerMargin = o.DividerMargin.OrElse(listSectionInsetDividerMargin)
	additional := listSectionInsetAdditionalDividerMargin
	if o.NotHasLeading {
		additional = listSectionInsetAdditionalDividerMarginNoLead
	}
	s.additionalDividerMargin = o.AdditionalDividerMargin.OrElse(additional)
	return s
}

// NewCupertinoListSection creates a base list section. In debug builds it panics with the tag CupertinoListSectionEmpty
// when it has no children and no header.
//
// Flutter: cupertino/list_section.dart CupertinoListSection.new
func NewCupertinoListSection(opts ...CupertinoListSectionOpts) *CupertinoListSection {
	return newListSection(CupertinoListSectionTypeBase, firstOf(opts))
}

// NewCupertinoListSectionInsetGrouped creates a list section whose rows are in a rounded card inset from the screen's
// edges. It panics like NewCupertinoListSection.
//
// Flutter: cupertino/list_section.dart CupertinoListSection.insetGrouped
func NewCupertinoListSectionInsetGrouped(opts ...CupertinoListSectionOpts) *CupertinoListSection {
	return newListSection(CupertinoListSectionTypeInsetGrouped, firstOf(opts))
}

// Type is the type of the section.
//
// Flutter: cupertino/list_section.dart CupertinoListSection.type
func (w *CupertinoListSection) Type() CupertinoListSectionType { return w.sectionType }

// Margin is the space around the rows.
//
// Flutter: cupertino/list_section.dart CupertinoListSection.margin
func (w *CupertinoListSection) Margin() painting.EdgeInsetsGeometry { return w.margin }

// ClipBehavior is how the rows are clipped.
//
// Flutter: cupertino/list_section.dart CupertinoListSection.clipBehavior
func (w *CupertinoListSection) ClipBehavior() ui.Clip { return w.clip }

// DividerMargin is the margin of the dividers at the start.
//
// Flutter: cupertino/list_section.dart CupertinoListSection.dividerMargin
func (w *CupertinoListSection) DividerMargin() float64 { return w.dividerMargin }

// AdditionalDividerMargin is added to DividerMargin for the dividers between the rows.
//
// Flutter: cupertino/list_section.dart CupertinoListSection.additionalDividerMargin
func (w *CupertinoListSection) AdditionalDividerMargin() float64 { return w.additionalDividerMargin }

// Build builds the section.
//
// Flutter: cupertino/list_section.dart CupertinoListSection.build
func (w *CupertinoListSection) Build(bc widgets.BuildContext) widgets.Widget {
	dividerColor := w.o.SeparatorColor
	if dividerColor == nil {
		dividerColor = CupertinoColors.Separator
	}
	resolvedDivider := CupertinoDynamicColorResolve(dividerColor, bc)
	dividerHeight := 1.0 / widgets.MediaQueryDevicePixelRatioOf(bc)
	longDivider := func() widgets.Widget {
		return widgets.NewContainer(nil, widgets.ContainerOpts{Color: foundation.Some(resolvedDivider), Height: foundation.Some(dividerHeight)})
	}
	shortDivider := func() widgets.Widget {
		return widgets.NewContainer(nil, widgets.ContainerOpts{Margin: painting.NewEdgeInsetsDirectionalFromSTEB(w.dividerMargin+w.additionalDividerMargin, 0, 0, 0),
			Color: foundation.Some(resolvedDivider), Height: foundation.Some(dividerHeight)})
	}
	style := CupertinoThemeOf(bc).TextTheme().TextStyle()
	var header, footer widgets.Widget
	base := w.sectionType == CupertinoListSectionTypeBase
	if base {
		style = style.Merge(painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(13.0),
			Color: foundation.Some(CupertinoDynamicColorResolve(listSectionHeaderFooterColor(), bc))}))
		if w.o.Header != nil {
			header = widgets.NewDefaultTextStyle(style, w.o.Header)
		}
	} else if w.o.Header != nil {
		header = widgets.NewDefaultTextStyle(style.Merge(painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(20.0), FontWeight: ui.FontWeightBold})), w.o.Header)
	}
	if w.o.Footer != nil {
		footer = widgets.NewDefaultTextStyle(style, w.o.Footer)
	}

	var group widgets.Widget
	if n := len(w.o.Children); n > 0 {
		var withDividers []widgets.Widget
		if base {
			withDividers = append(withDividers, longDivider())
		}
		for _, child := range w.o.Children[:n-1] {
			withDividers = append(withDividers, child, shortDivider())
		}
		withDividers = append(withDividers, w.o.Children[n-1])
		if base {
			withDividers = append(withDividers, longDivider())
		}
		radius := painting.BorderRadiusGeometry(painting.BorderRadiusZero)
		if !base {
			radius = painting.NewBorderRadiusCircular(listSectionInsetGroupedRadius)
		}
		decoration := w.o.Decoration
		if decoration == nil {
			decoration = painting.NewShapeDecoration(painting.NewRoundedSuperellipseBorder(painting.RoundedSuperellipseBorderOpts{BorderRadius: radius}),
				painting.ShapeDecorationOpts{Color: foundation.Some(CupertinoDynamicColorResolve(CupertinoColors.SecondarySystemGroupedBackground, bc))})
		}
		group = widgets.NewDecoratedBox(decoration, widgets.NewColumn(withDividers))
		if w.clip != ui.ClipNone {
			group = widgets.NewClipRSuperellipse(group, widgets.ClipRSuperellipseOpts{BorderRadius: radius, ClipBehavior: w.clip})
		}
		group = widgets.NewPadding(w.margin, group)
	}

	var column []widgets.Widget
	if base {
		column = append(column, widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: w.topMargin}))
	}
	if header != nil {
		margin := painting.EdgeInsetsGeometry(listSectionInsetGroupedHeaderMargin)
		if base {
			margin = listSectionDefaultHeaderMargin
		}
		column = append(column, widgets.NewAlign(widgets.NewPadding(margin, header), widgets.AlignOpts{Alignment: painting.AlignmentDirectionalCenterStart}))
	}
	if group != nil {
		column = append(column, group)
	}
	if footer != nil {
		margin := painting.EdgeInsetsGeometry(listSectionInsetGroupedFooterMargin)
		if base {
			margin = listSectionDefaultFooterMargin
		}
		column = append(column, widgets.NewAlign(widgets.NewPadding(margin, footer), widgets.AlignOpts{Alignment: painting.AlignmentDirectionalCenterStart}))
	}
	return widgets.NewDecoratedBox(painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(CupertinoDynamicColorResolve(w.o.BackgroundColor, bc))}),
		widgets.NewColumn(column))
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCupertinoListSectionEmpty = "CupertinoListSectionEmpty"
)
