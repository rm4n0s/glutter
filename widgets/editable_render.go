package widgets

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// editableRender is the widget that creates the RenderEditable for an
// EditableText.
//
// Flutter: widgets/editable_text.dart _Editable
type editableRender struct {
	MultiChildRenderObjectWidget
	inlineSpan             painting.InlineSpan
	value                  services.TextEditingValue
	cursorColor            foundation.Opt[ui.Color]
	startHandleLayerLink   *rendering.LayerLink
	endHandleLayerLink     *rendering.LayerLink
	backgroundCursorColor  foundation.Opt[ui.Color]
	showCursor             *foundation.ValueNotifier[bool]
	forceLine              bool
	readOnly               bool
	hasFocus               bool
	maxLines               foundation.Opt[int]
	minLines               foundation.Opt[int]
	expands                bool
	strutStyle             *painting.StrutStyle
	selectionColor         foundation.Opt[ui.Color]
	textScaler             painting.TextScaler
	textAlign              ui.TextAlign
	textDirection          ui.TextDirection
	locale                 foundation.Opt[ui.Locale]
	obscuringCharacter     string
	obscureText            bool
	textHeightBehavior     *ui.TextHeightBehavior
	textWidthBasis         painting.TextWidthBasis
	offset                 rendering.ViewportOffset
	rendererIgnoresPointer bool
	cursorWidth            float64
	cursorHeight           foundation.Opt[float64]
	cursorRadius           foundation.Opt[ui.Radius]
	cursorOffset           ui.Offset
	paintCursorAboveText   bool
	selectionHeightStyle   ui.BoxHeightStyle
	selectionWidthStyle    ui.BoxWidthStyle
	enableInteractiveSel   bool
	textSelectionDelegate  services.TextSelectionDelegate
	devicePixelRatio       float64
	promptRectRange        foundation.Opt[ui.TextRange]
	promptRectColor        foundation.Opt[ui.Color]
	clipBehavior           ui.Clip
}

func newEditableRender(key foundation.Key, e *editableRender) *editableRender {
	e.MultiChildRenderObjectWidget = NewMultiChildRenderObjectWidgetBase(ExtractWidgetsFromInlineSpan(e.inlineSpan, e.textScaler), WidgetOpts{Key: key})
	return e
}

func (w *editableRender) effectiveLocale(bc BuildContext) foundation.Opt[ui.Locale] {
	if w.locale.IsSome() {
		return w.locale
	}
	if l, ok := LocalizationsMaybeLocaleOf(bc); ok {
		return foundation.Some(l)
	}
	return foundation.None[ui.Locale]()
}

// CreateRenderObject creates the render editable.
//
// Flutter: widgets/editable_text.dart _Editable.createRenderObject
func (w *editableRender) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	var engine ui.ParagraphFactory
	var g *gestures.GestureBinding
	if b := MaybeWidgetsBindingOf(bc); b != nil {
		engine = b.PlatformDispatcher().TextEngine()
		g = b.Gestures()
	}
	maxLines := w.maxLines
	if maxLines.IsNone() {
		maxLines = foundation.Some(rendering.MaxLinesUnlimited)
	}
	return rendering.NewRenderEditable(w.textDirection, w.startHandleLayerLink, w.endHandleLayerLink, w.offset, w.textSelectionDelegate, rendering.RenderEditableOpts{
		TextEngine: engine, Gestures: g, Text: w.inlineSpan, CursorColor: w.cursorColor, BackgroundCursorColor: w.backgroundCursorColor,
		ShowCursor: w.showCursor, DisableForceLine: !w.forceLine, ReadOnly: w.readOnly, HasFocus: w.hasFocus, MaxLines: maxLines,
		MinLines: w.minLines, Expands: w.expands, StrutStyle: w.strutStyle, SelectionColor: w.selectionColor, TextScaler: w.textScaler,
		TextAlign: w.textAlign, Locale: w.effectiveLocale(bc), Selection: foundation.Some(w.value.Selection), IgnorePointer: w.rendererIgnoresPointer,
		ObscuringCharacter: w.obscuringCharacter, ObscureText: w.obscureText, TextHeightBehavior: w.textHeightBehavior,
		TextWidthBasis: w.textWidthBasis, CursorWidth: foundation.Some(w.cursorWidth), CursorHeight: w.cursorHeight, CursorRadius: w.cursorRadius,
		PaintCursorAboveText: w.paintCursorAboveText, CursorOffset: w.cursorOffset, DevicePixelRatio: foundation.Some(w.devicePixelRatio),
		SelectionHeightStyle: w.selectionHeightStyle, SelectionWidthStyle: w.selectionWidthStyle,
		EnableInteractiveSelection: foundation.Some(w.enableInteractiveSel), PromptRectRange: w.promptRectRange, PromptRectColor: w.promptRectColor,
		ClipBehavior: w.clipBehavior,
	})
}

// UpdateRenderObject copies the configuration to the render editable.
//
// Flutter: widgets/editable_text.dart _Editable.updateRenderObject
func (w *editableRender) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderEditable)
	r.SetText(w.inlineSpan)
	r.SetCursorColor(w.cursorColor)
	r.SetStartHandleLayerLink(w.startHandleLayerLink)
	r.SetEndHandleLayerLink(w.endHandleLayerLink)
	r.SetBackgroundCursorColor(w.backgroundCursorColor)
	r.SetShowCursor(w.showCursor)
	r.SetForceLine(w.forceLine)
	r.SetReadOnly(w.readOnly)
	r.SetHasFocus(w.hasFocus)
	r.SetMaxLines(w.maxLines)
	r.SetMinLines(w.minLines)
	r.SetExpands(w.expands)
	r.SetStrutStyle(w.strutStyle)
	r.SetSelectionColor(w.selectionColor)
	r.SetTextScaler(w.textScaler)
	r.SetTextAlign(w.textAlign)
	r.SetTextDirection(w.textDirection)
	r.SetLocale(w.effectiveLocale(bc))
	r.SetSelection(foundation.Some(w.value.Selection))
	r.SetOffset(w.offset)
	r.SetIgnorePointer(w.rendererIgnoresPointer)
	r.SetTextHeightBehavior(w.textHeightBehavior)
	r.SetTextWidthBasis(w.textWidthBasis)
	r.SetObscuringCharacter(w.obscuringCharacter)
	r.SetObscureText(w.obscureText)
	r.SetCursorWidth(w.cursorWidth)
	r.SetCursorHeight(w.cursorHeight)
	r.SetCursorRadius(w.cursorRadius)
	r.SetCursorOffset(w.cursorOffset)
	r.SetSelectionHeightStyle(w.selectionHeightStyle)
	r.SetSelectionWidthStyle(w.selectionWidthStyle)
	r.SetEnableInteractiveSelection(foundation.Some(w.enableInteractiveSel))
	r.SetTextSelectionDelegate(w.textSelectionDelegate)
	r.SetDevicePixelRatio(w.devicePixelRatio)
	r.SetPaintCursorAboveText(w.paintCursorAboveText)
	r.SetPromptRectColor(w.promptRectColor)
	r.SetClipBehavior(w.clipBehavior)
	r.SetPromptRectRange(w.promptRectRange)
}

// renderCompositionCallback is the render object of compositionCallback.
//
// Flutter: widgets/editable_text.dart _RenderCompositionCallback
type renderCompositionCallback struct {
	rendering.RenderProxyBox
	compositeCallback rendering.CompositionCallback
	cancelCallback    func()
	enabled           bool
}

func newRenderCompositionCallback(cb rendering.CompositionCallback, enabled bool) *renderCompositionCallback {
	r := &renderCompositionCallback{compositeCallback: cb, enabled: enabled}
	rendering.InitRenderProxyBox(&r.RenderProxyBox, r, nil)
	return r
}

func (r *renderCompositionCallback) setEnabled(value bool) {
	r.enabled = value
	if !value {
		if r.cancelCallback != nil {
			r.cancelCallback()
		}
		r.cancelCallback = nil
	} else if r.cancelCallback == nil {
		r.MarkNeedsPaint()
	}
}

// Paint registers the callback on the layer, if enabled, then paints the child.
//
// Flutter: widgets/editable_text.dart _RenderCompositionCallback.paint
func (r *renderCompositionCallback) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	if r.enabled && r.cancelCallback == nil {
		r.cancelCallback = context.AddCompositionCallback(r.compositeCallback)
	}
	r.RenderProxyBox.Paint(context, offset)
}

// compositionCallback is the widget that calls a function whenever the layer it
// is painted into is composited, to tell the input method where the field is.
//
// Flutter: widgets/editable_text.dart _CompositionCallback
type compositionCallback struct {
	SingleChildRenderObjectWidget
	compositeCallback rendering.CompositionCallback
	enabled           bool
}

func newCompositionCallback(cb rendering.CompositionCallback, enabled bool, child Widget) *compositionCallback {
	return &compositionCallback{NewSingleChildRenderObjectWidgetBase(child), cb, enabled}
}

func (w *compositionCallback) CreateRenderObject(BuildContext) rendering.RenderObject {
	return newRenderCompositionCallback(w.compositeCallback, w.enabled)
}

func (w *compositionCallback) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	renderObject.(*renderCompositionCallback).setEnabled(w.enabled)
}
