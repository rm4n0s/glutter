// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

func sameTextHeightBehavior(a, b *ui.TextHeightBehavior) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// DefaultTextStyle is the text style, and the other text options, that
// descendant Text widgets use when they don't set their own.
//
// Flutter: widgets/text.dart DefaultTextStyle
type DefaultTextStyle struct {
	InheritedTheme
	style              *painting.TextStyle
	textAlign          ui.TextAlign
	softWrap           bool
	overflow           painting.TextOverflow
	maxLines           foundation.Opt[int]
	textWidthBasis     painting.TextWidthBasis
	textHeightBehavior *ui.TextHeightBehavior
}

// DefaultTextStyleOpts are the optional parameters of NewDefaultTextStyle.
//
// Flutter: widgets/text.dart DefaultTextStyle.new
type DefaultTextStyleOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// TextAlign is how the text is aligned horizontally; unspecified means the
	// Text widget's default (start).
	TextAlign ui.TextAlign
	// SoftWrap is whether the text wraps at the box width; the default is true.
	SoftWrap foundation.Opt[bool]
	// Overflow is how visual overflow is handled; the default is
	// TextOverflowClip.
	Overflow painting.TextOverflow
	// MaxLines is the maximum number of lines; it must be greater than zero.
	MaxLines foundation.Opt[int]
	// TextWidthBasis is how the width of the text is measured; the default is
	// TextWidthBasisParent.
	TextWidthBasis painting.TextWidthBasis
	// TextHeightBehavior says how the line height applies to the first ascent and
	// the last descent.
	TextHeightBehavior *ui.TextHeightBehavior
}

// NewDefaultTextStyle creates the widget. In debug builds it panics with the
// tag TextMaxLinesInvalid when MaxLines is not greater than zero.
//
// Flutter: widgets/text.dart DefaultTextStyle.new
func NewDefaultTextStyle(style *painting.TextStyle, child Widget, opts ...DefaultTextStyleOpts) *DefaultTextStyle {
	o := firstOpts(opts)
	assertMaxLines(o.MaxLines)
	if o.Overflow == painting.TextOverflowUnspecified {
		o.Overflow = painting.TextOverflowClip
	}
	if o.TextWidthBasis == painting.TextWidthBasisUnspecified {
		o.TextWidthBasis = painting.TextWidthBasisParent
	}
	return &DefaultTextStyle{NewInheritedThemeBase(child, WidgetOpts{Key: o.Key}), style, o.TextAlign, o.SoftWrap.OrElse(true),
		o.Overflow, o.MaxLines, o.TextWidthBasis, o.TextHeightBehavior}
}

func assertMaxLines(maxLines foundation.Opt[int]) {
	if m, ok := maxLines.Get(); ok && m <= 0 && foundation.DebugMode {
		panic(errors.New("TextMaxLinesInvalid", "maxLines must be greater than zero", "maxLines", m))
	}
}

// NewDefaultTextStyleFallback creates the default text style that is used when
// there is none above: an empty style, wrapping, clipping.
//
// Flutter: widgets/text.dart DefaultTextStyle.fallback
func NewDefaultTextStyleFallback(opts ...WidgetOpts) *DefaultTextStyle {
	return &DefaultTextStyle{NewInheritedThemeBase(NewSizedBoxShrink(nil), opts...), painting.NewTextStyle(), ui.TextAlignUnspecified, true,
		painting.TextOverflowClip, foundation.None[int](), painting.TextWidthBasisParent, nil}
}

// DefaultTextStyleMergeOpts are the optional parameters of
// NewDefaultTextStyleMerge. Zero values keep what the ancestor has.
//
// Flutter: widgets/text.dart DefaultTextStyle.merge
type DefaultTextStyleMergeOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Style is merged into the style of the ancestor.
	Style *painting.TextStyle
	// TextAlign replaces the alignment of the ancestor when specified.
	TextAlign ui.TextAlign
	// SoftWrap replaces the soft wrap of the ancestor when set.
	SoftWrap foundation.Opt[bool]
	// Overflow replaces the overflow of the ancestor when specified.
	Overflow painting.TextOverflow
	// MaxLines replaces the maximum lines of the ancestor when set.
	MaxLines foundation.Opt[int]
	// TextWidthBasis replaces the width basis of the ancestor when specified.
	TextWidthBasis painting.TextWidthBasis
	// TextHeightBehavior replaces the height behavior of the ancestor when set.
	TextHeightBehavior *ui.TextHeightBehavior
}

// NewDefaultTextStyleMerge creates a widget that merges the options with the
// default text style of the closest ancestor.
//
// Flutter: widgets/text.dart DefaultTextStyle.merge
func NewDefaultTextStyleMerge(child Widget, opts ...DefaultTextStyleMergeOpts) Widget {
	o := firstOpts(opts)
	return NewBuilder(func(bc BuildContext) Widget {
		parent := DefaultTextStyleOf(bc)
		style := parent.style
		if o.Style != nil {
			style = parent.style.Merge(o.Style)
		}
		d := &DefaultTextStyle{NewInheritedThemeBase(child, WidgetOpts{Key: o.Key}), style, parent.textAlign, parent.softWrap,
			parent.overflow, parent.maxLines, parent.textWidthBasis, parent.textHeightBehavior}
		if o.TextAlign != ui.TextAlignUnspecified {
			d.textAlign = o.TextAlign
		}
		if sw, ok := o.SoftWrap.Get(); ok {
			d.softWrap = sw
		}
		if o.Overflow != painting.TextOverflowUnspecified {
			d.overflow = o.Overflow
		}
		if o.MaxLines.IsSome() {
			d.maxLines = o.MaxLines
		}
		if o.TextWidthBasis != painting.TextWidthBasisUnspecified {
			d.textWidthBasis = o.TextWidthBasis
		}
		if o.TextHeightBehavior != nil {
			d.textHeightBehavior = o.TextHeightBehavior
		}
		return d
	})
}

// Style is the text style to apply to descendant Text widgets without explicit
// style.
//
// Flutter: widgets/text.dart DefaultTextStyle.style
func (w *DefaultTextStyle) Style() *painting.TextStyle { return w.style }

// TextAlign is how the text is aligned horizontally; unspecified is null.
//
// Flutter: widgets/text.dart DefaultTextStyle.textAlign
func (w *DefaultTextStyle) TextAlign() ui.TextAlign { return w.textAlign }

// SoftWrap is whether the text wraps at the box width.
//
// Flutter: widgets/text.dart DefaultTextStyle.softWrap
func (w *DefaultTextStyle) SoftWrap() bool { return w.softWrap }

// Overflow is how visual overflow is handled.
//
// Flutter: widgets/text.dart DefaultTextStyle.overflow
func (w *DefaultTextStyle) Overflow() painting.TextOverflow { return w.overflow }

// MaxLines is the maximum number of lines, if any.
//
// Flutter: widgets/text.dart DefaultTextStyle.maxLines
func (w *DefaultTextStyle) MaxLines() foundation.Opt[int] { return w.maxLines }

// TextWidthBasis is how the width of the text is measured.
//
// Flutter: widgets/text.dart DefaultTextStyle.textWidthBasis
func (w *DefaultTextStyle) TextWidthBasis() painting.TextWidthBasis { return w.textWidthBasis }

// TextHeightBehavior is how the line height applies to the first ascent and the
// last descent, or nil.
//
// Flutter: widgets/text.dart DefaultTextStyle.textHeightBehavior
func (w *DefaultTextStyle) TextHeightBehavior() *ui.TextHeightBehavior { return w.textHeightBehavior }

// UpdateShouldNotify is whether any of the options changed.
//
// Flutter: widgets/text.dart DefaultTextStyle.updateShouldNotify
func (w *DefaultTextStyle) UpdateShouldNotify(old Widget) bool {
	o := old.(*DefaultTextStyle)
	return !w.style.Equal(o.style) || w.textAlign != o.textAlign || w.softWrap != o.softWrap || w.overflow != o.overflow ||
		w.maxLines != o.maxLines || w.textWidthBasis != o.textWidthBasis || !sameTextHeightBehavior(w.textHeightBehavior, o.textHeightBehavior)
}

// Wrap makes a DefaultTextStyle with these options around the child.
//
// Flutter: widgets/text.dart DefaultTextStyle.wrap
func (w *DefaultTextStyle) Wrap(_ BuildContext, child Widget) Widget {
	return &DefaultTextStyle{NewInheritedThemeBase(child), w.style, w.textAlign, w.softWrap, w.overflow, w.maxLines, w.textWidthBasis, w.textHeightBehavior}
}

// DebugFillProperties adds the style.
//
// Flutter: widgets/text.dart DefaultTextStyle.debugFillProperties
func (w *DefaultTextStyle) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	if w.style != nil {
		w.style.DebugFillProperties(p)
	}
}

// DefaultTextStyleOf is the default text style of the closest DefaultTextStyle
// above the context, or the fallback. The context depends on it.
//
// Flutter: widgets/text.dart DefaultTextStyle.of
func DefaultTextStyleOf(bc BuildContext) *DefaultTextStyle {
	if d, ok := DependOn[*DefaultTextStyle](bc); ok {
		return d
	}
	return NewDefaultTextStyleFallback()
}

// DefaultTextHeightBehavior is the text height behavior that descendant Text
// widgets use when they don't set their own.
//
// Flutter: widgets/text.dart DefaultTextHeightBehavior
type DefaultTextHeightBehavior struct {
	InheritedTheme
	textHeightBehavior ui.TextHeightBehavior
}

// NewDefaultTextHeightBehavior creates the widget.
//
// Flutter: widgets/text.dart DefaultTextHeightBehavior.new
func NewDefaultTextHeightBehavior(textHeightBehavior ui.TextHeightBehavior, child Widget, opts ...WidgetOpts) *DefaultTextHeightBehavior {
	return &DefaultTextHeightBehavior{NewInheritedThemeBase(child, opts...), textHeightBehavior}
}

// TextHeightBehavior is the behavior the widget gives.
//
// Flutter: widgets/text.dart DefaultTextHeightBehavior.textHeightBehavior
func (w *DefaultTextHeightBehavior) TextHeightBehavior() ui.TextHeightBehavior {
	return w.textHeightBehavior
}

// UpdateShouldNotify is whether the behavior changed.
//
// Flutter: widgets/text.dart DefaultTextHeightBehavior.updateShouldNotify
func (w *DefaultTextHeightBehavior) UpdateShouldNotify(old Widget) bool {
	return w.textHeightBehavior != old.(*DefaultTextHeightBehavior).textHeightBehavior
}

// Wrap makes a DefaultTextHeightBehavior with this behavior around the child.
//
// Flutter: widgets/text.dart DefaultTextHeightBehavior.wrap
func (w *DefaultTextHeightBehavior) Wrap(_ BuildContext, child Widget) Widget {
	return NewDefaultTextHeightBehavior(w.textHeightBehavior, child)
}

// DefaultTextHeightBehaviorMaybeOf is the behavior of the closest
// DefaultTextHeightBehavior above the context, if there is one.
//
// Flutter: widgets/text.dart DefaultTextHeightBehavior.maybeOf
func DefaultTextHeightBehaviorMaybeOf(bc BuildContext) *ui.TextHeightBehavior {
	if d, ok := DependOn[*DefaultTextHeightBehavior](bc); ok {
		b := d.textHeightBehavior
		return &b
	}
	return nil
}

// DefaultTextHeightBehaviorOf is the behavior of the closest
// DefaultTextHeightBehavior above the context. It panics with the tag
// DefaultTextHeightBehaviorMissing when there is none.
//
// Flutter: widgets/text.dart DefaultTextHeightBehavior.of
func DefaultTextHeightBehaviorOf(bc BuildContext) ui.TextHeightBehavior {
	b := DefaultTextHeightBehaviorMaybeOf(bc)
	if b == nil {
		panic(errors.New("DefaultTextHeightBehaviorMissing", "no DefaultTextHeightBehavior widget found above the context"))
	}
	return *b
}

// TextOpts are the optional parameters of NewText and NewTextRich.
//
// Flutter: widgets/text.dart Text.new
type TextOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Style is the style to use; when its Inherit is true, it is merged into the
	// default text style.
	Style *painting.TextStyle
	// StrutStyle is the strut style to use.
	StrutStyle *painting.StrutStyle
	// TextAlign is how the text is aligned horizontally; unspecified means the
	// default text style's, or start.
	TextAlign ui.TextAlign
	// TextDirection is the direction of the text; the default is the one of the
	// closest Directionality.
	TextDirection ui.TextDirection
	// Locale is the locale used to select region specific glyphs.
	Locale foundation.Opt[ui.Locale]
	// SoftWrap is whether the text wraps at the box width; the default comes
	// from the default text style.
	SoftWrap foundation.Opt[bool]
	// Overflow is how visual overflow is handled.
	Overflow painting.TextOverflow
	// TextScaler is how the font size is scaled; the default is the one of the
	// MediaQuery.
	TextScaler painting.TextScaler
	// MaxLines is the maximum number of lines.
	MaxLines foundation.Opt[int]
	// SemanticsLabel replaces the text in the semantics tree.
	SemanticsLabel foundation.Opt[string]
	// SemanticsIdentifier is the identifier of the semantics node.
	SemanticsIdentifier foundation.Opt[string]
	// TextWidthBasis is how the width of the text is measured.
	TextWidthBasis painting.TextWidthBasis
	// TextHeightBehavior says how the line height applies to the first ascent
	// and the last descent.
	TextHeightBehavior *ui.TextHeightBehavior
	// SelectionColor is the color of the selection highlight.
	SelectionColor foundation.Opt[ui.Color]
}

// Text displays a string of text with a single style.
//
// Deviation: the MediaQuery text spacing overrides (line height, letter and
// word spacing) are not applied (SPEC §12.10).
//
// Flutter: widgets/text.dart Text
type Text struct {
	StatelessWidget
	data     foundation.Opt[string]
	textSpan painting.InlineSpan
	opts     TextOpts
}

// NewText creates a text widget showing the string.
//
// Flutter: widgets/text.dart Text.new
func NewText(data string, opts ...TextOpts) *Text {
	o := firstOpts(opts)
	assertMaxLines(o.MaxLines)
	return &Text{NewStatelessWidgetBase(WidgetOpts{Key: o.Key}), foundation.Some(data), nil, o}
}

// NewTextRich creates a text widget showing the span, which can have several
// styles.
//
// Flutter: widgets/text.dart Text.rich
func NewTextRich(textSpan painting.InlineSpan, opts ...TextOpts) *Text {
	o := firstOpts(opts)
	assertMaxLines(o.MaxLines)
	return &Text{NewStatelessWidgetBase(WidgetOpts{Key: o.Key}), foundation.None[string](), textSpan, o}
}

// Data is the text to display, if the widget was made by NewText.
//
// Flutter: widgets/text.dart Text.data
func (w *Text) Data() foundation.Opt[string] { return w.data }

// TextSpan is the span to display, if the widget was made by NewTextRich.
//
// Flutter: widgets/text.dart Text.textSpan
func (w *Text) TextSpan() painting.InlineSpan { return w.textSpan }

// Style is the style to use for the text.
//
// Flutter: widgets/text.dart Text.style
func (w *Text) Style() *painting.TextStyle { return w.opts.Style }

// PlainText is the text of the widget without styles.
//
// Flutter: no counterpart (used by finders)
func (w *Text) PlainText() string {
	if d, ok := w.data.Get(); ok {
		return d
	}
	return painting.InlineSpanToPlainText(w.textSpan)
}

// Build makes the RichText with the default text style merged in.
//
// Flutter: widgets/text.dart Text.build
func (w *Text) Build(bc BuildContext) Widget {
	o := w.opts
	defaults := DefaultTextStyleOf(bc)
	effectiveStyle := o.Style
	if o.Style == nil || o.Style.Inherit {
		effectiveStyle = defaults.style.Merge(o.Style)
	}
	if MediaQueryBoldTextOf(bc) {
		effectiveStyle = effectiveStyle.Merge(painting.NewTextStyle(painting.TextStyleOpts{FontWeight: ui.FontWeightBold}))
	}
	var children []painting.InlineSpan
	if w.textSpan != nil {
		children = []painting.InlineSpan{w.textSpan}
	}
	effectiveSpan := painting.NewTextSpan(painting.TextSpanOpts{Style: effectiveStyle, Text: w.data, Locale: o.Locale, Children: children})

	textScaler := o.TextScaler
	if textScaler == nil {
		textScaler = MediaQueryTextScalerOf(bc)
	}
	textAlign := o.TextAlign
	if textAlign == ui.TextAlignUnspecified {
		textAlign = defaults.textAlign
	}
	softWrap := o.SoftWrap.OrElse(defaults.softWrap)
	overflow := o.Overflow
	if overflow == painting.TextOverflowUnspecified {
		overflow = effectiveStyle.Overflow
	}
	if overflow == painting.TextOverflowUnspecified {
		overflow = defaults.overflow
	}
	maxLines := o.MaxLines
	if maxLines.IsNone() {
		maxLines = defaults.maxLines
	}
	widthBasis := o.TextWidthBasis
	if widthBasis == painting.TextWidthBasisUnspecified {
		widthBasis = defaults.textWidthBasis
	}
	heightBehavior := o.TextHeightBehavior
	if heightBehavior == nil {
		heightBehavior = defaults.textHeightBehavior
	}
	if heightBehavior == nil {
		heightBehavior = DefaultTextHeightBehaviorMaybeOf(bc)
	}
	registrar := SelectionContainerMaybeOf(bc)
	style := DefaultSelectionStyleOf(bc)
	selectionColor := o.SelectionColor
	if selectionColor.IsNone() {
		selectionColor = style.selectionColor
	}
	if selectionColor.IsNone() {
		selectionColor = foundation.Some(DefaultSelectionColor)
	}
	richOpts := RichTextOpts{
		TextAlign: textAlign, TextDirection: o.TextDirection, Locale: o.Locale, SoftWrap: foundation.Some(softWrap), Overflow: overflow,
		TextScaler: textScaler, MaxLines: maxLines, StrutStyle: o.StrutStyle, TextWidthBasis: widthBasis,
		TextHeightBehavior: heightBehavior, SelectionColor: selectionColor,
	}
	var result Widget
	if registrar != nil {
		cursor := style.mouseCursor
		if cursor == nil {
			cursor = services.SystemMouseCursors.Text
		}
		result = NewMouseRegion(&selectableTextContainer{StatefulWidget: NewStatefulWidgetBase(), text: NewRichText(effectiveSpan, richOpts)},
			MouseRegionOpts{Cursor: cursor})
	} else {
		result = NewRichText(effectiveSpan, richOpts)
	}
	if o.SemanticsLabel.IsSome() || o.SemanticsIdentifier.IsSome() {
		result = NewSemantics(NewExcludeSemantics(result, ExcludeSemanticsOpts{Excluding: foundation.Some(o.SemanticsLabel.IsSome())}),
			SemanticsOpts{TextDirection: o.TextDirection, Label: o.SemanticsLabel, Identifier: o.SemanticsIdentifier})
	}
	return result
}

// DefaultSelectionColor is the color of the selection highlight when none is
// given: Flutter's DefaultSelectionStyle.defaultColor, 0x80808080.
//
// Flutter: widgets/default_selection_style.dart DefaultSelectionStyle.defaultColor
var DefaultSelectionColor = ui.NewColor(0x80808080)

// RichTextOpts are the optional parameters of NewRichText.
//
// Flutter: widgets/basic.dart RichText.new
type RichTextOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// TextAlign is how the text is aligned horizontally; the default is
	// TextAlignStart.
	TextAlign ui.TextAlign
	// TextDirection is the direction of the text; the default is the one of the
	// closest Directionality.
	TextDirection ui.TextDirection
	// SoftWrap is whether the text wraps at the box width; the default is true.
	SoftWrap foundation.Opt[bool]
	// Overflow is how visual overflow is handled; the default is
	// TextOverflowClip.
	Overflow painting.TextOverflow
	// TextScaler is how the font size is scaled; the default is no scaling.
	TextScaler painting.TextScaler
	// MaxLines is the maximum number of lines.
	MaxLines foundation.Opt[int]
	// Locale is the locale used to select region specific glyphs; the default is
	// the one of the closest Localizations.
	Locale foundation.Opt[ui.Locale]
	// StrutStyle is the strut style to use.
	StrutStyle *painting.StrutStyle
	// TextWidthBasis is how the width of the text is measured; the default is
	// TextWidthBasisParent.
	TextWidthBasis painting.TextWidthBasis
	// TextHeightBehavior says how the line height applies to the first ascent
	// and the last descent.
	TextHeightBehavior *ui.TextHeightBehavior
	// SelectionRegistrar is where the text registers its selectable fragments.
	SelectionRegistrar rendering.SelectionRegistrar
	// SelectionColor is the color of the selection highlight; it is required
	// when there is a registrar.
	SelectionColor foundation.Opt[ui.Color]
}

// RichText displays a paragraph of text with multiple styles.
//
// Flutter: widgets/basic.dart RichText
type RichText struct {
	MultiChildRenderObjectWidget
	text painting.InlineSpan
	o    RichTextOpts
}

// NewRichText creates the widget; the children are the widgets of the widget
// spans in the text. In debug builds it panics with the tag
// TextMaxLinesInvalid when MaxLines is not greater than zero and with
// RichTextNoSelectionColor when there is a registrar but no color.
//
// Flutter: widgets/basic.dart RichText.new
func NewRichText(text painting.InlineSpan, opts ...RichTextOpts) *RichText {
	o := firstOpts(opts)
	assertMaxLines(o.MaxLines)
	if foundation.DebugMode && o.SelectionRegistrar != nil && o.SelectionColor.IsNone() {
		panic(errors.New("RichTextNoSelectionColor", "a RichText with a selection registrar needs a selection color"))
	}
	if o.TextAlign == ui.TextAlignUnspecified {
		o.TextAlign = ui.TextAlignStart
	}
	if o.Overflow == painting.TextOverflowUnspecified {
		o.Overflow = painting.TextOverflowClip
	}
	if o.TextScaler == nil {
		o.TextScaler = painting.TextScalerNoScaling
	}
	if o.TextWidthBasis == painting.TextWidthBasisUnspecified {
		o.TextWidthBasis = painting.TextWidthBasisParent
	}
	children := ExtractWidgetsFromInlineSpan(text, o.TextScaler)
	return &RichText{NewMultiChildRenderObjectWidgetBase(children, WidgetOpts{Key: o.Key}), text, o}
}

// Text is the text to display.
//
// Flutter: widgets/basic.dart RichText.text
func (w *RichText) Text() painting.InlineSpan { return w.text }

// TextAlign is how the text is aligned horizontally.
//
// Flutter: widgets/basic.dart RichText.textAlign
func (w *RichText) TextAlign() ui.TextAlign { return w.o.TextAlign }

// TextDirection is the direction of the text, or unspecified for the one of the
// Directionality.
//
// Flutter: widgets/basic.dart RichText.textDirection
func (w *RichText) TextDirection() ui.TextDirection { return w.o.TextDirection }

// SoftWrap is whether the text wraps at the box width.
//
// Flutter: widgets/basic.dart RichText.softWrap
func (w *RichText) SoftWrap() bool { return w.o.SoftWrap.OrElse(true) }

// Overflow is how visual overflow is handled.
//
// Flutter: widgets/basic.dart RichText.overflow
func (w *RichText) Overflow() painting.TextOverflow { return w.o.Overflow }

// TextScaler is how the font size is scaled.
//
// Flutter: widgets/basic.dart RichText.textScaler
func (w *RichText) TextScaler() painting.TextScaler { return w.o.TextScaler }

// MaxLines is the maximum number of lines.
//
// Flutter: widgets/basic.dart RichText.maxLines
func (w *RichText) MaxLines() foundation.Opt[int] { return w.o.MaxLines }

// PlainText is the text of the widget without styles.
//
// Flutter: no counterpart (used by finders)
func (w *RichText) PlainText() string { return painting.InlineSpanToPlainText(w.text) }

// IsRichText is true: the text finders of gluttertest skip RichText widgets
// unless they are asked for them.
//
// Flutter: no counterpart (used by finders)
func (w *RichText) IsRichText() bool { return true }

func richTextDevicePixelRatio(bc BuildContext) float64 {
	if r, ok := MediaQueryMaybeDevicePixelRatioOf(bc); ok {
		return r
	}
	if v := MaybeViewOf(bc); v != nil {
		return v.DevicePixelRatio()
	}
	return 1
}

func (w *RichText) locale(bc BuildContext) foundation.Opt[ui.Locale] {
	if w.o.Locale.IsSome() {
		return w.o.Locale
	}
	if l, ok := LocalizationsMaybeLocaleOf(bc); ok {
		return foundation.Some(l)
	}
	return foundation.None[ui.Locale]()
}

// CreateRenderObject creates the paragraph.
//
// Flutter: widgets/basic.dart RichText.createRenderObject
func (w *RichText) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	o := w.o
	var engine ui.ParagraphFactory
	if b := MaybeWidgetsBindingOf(bc); b != nil {
		engine = b.PlatformDispatcher().TextEngine()
	}
	return rendering.NewRenderParagraph(w.text, requiredDirection(o.TextDirection, bc), rendering.RenderParagraphOpts{
		TextEngine: engine, TextAlign: o.TextAlign, SoftWrap: foundation.Some(w.SoftWrap()), Overflow: o.Overflow, TextScaler: o.TextScaler,
		MaxLines: o.MaxLines, Locale: w.locale(bc), StrutStyle: o.StrutStyle, TextWidthBasis: o.TextWidthBasis,
		TextHeightBehavior: o.TextHeightBehavior, Registrar: o.SelectionRegistrar, SelectionColor: o.SelectionColor,
		DevicePixelRatio: foundation.Some(richTextDevicePixelRatio(bc)),
	})
}

// UpdateRenderObject copies the configuration to the paragraph.
//
// Flutter: widgets/basic.dart RichText.updateRenderObject
func (w *RichText) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	o := w.o
	r := renderObject.(*rendering.RenderParagraph)
	r.SetText(w.text)
	r.SetTextAlign(o.TextAlign)
	r.SetTextDirection(requiredDirection(o.TextDirection, bc))
	r.SetSoftWrap(w.SoftWrap())
	r.SetOverflow(o.Overflow)
	r.SetTextScaler(o.TextScaler)
	r.SetMaxLines(o.MaxLines)
	r.SetStrutStyle(o.StrutStyle)
	r.SetTextWidthBasis(o.TextWidthBasis)
	r.SetTextHeightBehavior(o.TextHeightBehavior)
	r.SetLocale(w.locale(bc))
	r.SetRegistrar(o.SelectionRegistrar)
	r.SetSelectionColor(o.SelectionColor)
	r.SetDevicePixelRatio(richTextDevicePixelRatio(bc))
}
