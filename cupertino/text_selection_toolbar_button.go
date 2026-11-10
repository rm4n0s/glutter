package cupertino

import (
	"math"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// toolbarButtonFontStyle is the style of the text of a button of the toolbar.
var toolbarButtonFontStyle = painting.NewTextStyle(painting.TextStyleOpts{
	Inherit: foundation.Some(false), FontSize: foundation.Some(15.0), LetterSpacing: foundation.Some(-0.15), FontWeight: ui.FontWeightW400})

var (
	toolbarTextColor    = NewCupertinoDynamicColorWithBrightness(CupertinoColors.Black, CupertinoColors.White)
	toolbarPressedColor = NewCupertinoDynamicColorWithBrightness(ui.NewColor(0x10000000), ui.NewColor(0x10FFFFFF))
)

// toolbarButtonPadding is the padding of a button of the toolbar.
var toolbarButtonPadding = painting.NewEdgeInsetsSymmetric(18.0, 16.0)

// CupertinoTextSelectionToolbarButtonOpts are the optional parameters of the constructors of
// CupertinoTextSelectionToolbarButton.
//
// Flutter: cupertino/text_selection_toolbar_button.dart CupertinoTextSelectionToolbarButton.new
type CupertinoTextSelectionToolbarButtonOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// OnPressed is called when the button is pressed; nil disables it.
	OnPressed func()
}

// CupertinoTextSelectionToolbarButton is a button in the style of the toolbar of the text selection of iOS.
//
// Flutter: cupertino/text_selection_toolbar_button.dart CupertinoTextSelectionToolbarButton
type CupertinoTextSelectionToolbarButton struct {
	widgets.StatefulWidget
	child      widgets.Widget
	onPressed  func()
	buttonItem *widgets.ContextMenuButtonItem
	text       foundation.Opt[string]
}

// NewCupertinoTextSelectionToolbarButton creates a button that shows the child.
//
// Flutter: cupertino/text_selection_toolbar_button.dart CupertinoTextSelectionToolbarButton.new
func NewCupertinoTextSelectionToolbarButton(child widgets.Widget, opts ...CupertinoTextSelectionToolbarButtonOpts) *CupertinoTextSelectionToolbarButton {
	o := firstOf(opts)
	return &CupertinoTextSelectionToolbarButton{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), child: child, onPressed: o.OnPressed}
}

// NewCupertinoTextSelectionToolbarButtonText creates a button that shows the text.
//
// Flutter: cupertino/text_selection_toolbar_button.dart CupertinoTextSelectionToolbarButton.text
func NewCupertinoTextSelectionToolbarButtonText(text string, opts ...CupertinoTextSelectionToolbarButtonOpts) *CupertinoTextSelectionToolbarButton {
	o := firstOf(opts)
	return &CupertinoTextSelectionToolbarButton{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), text: foundation.Some(text), onPressed: o.OnPressed}
}

// NewCupertinoTextSelectionToolbarButtonButtonItem creates a button from the item of a context menu: its label,
// or the default of its type, and its callback.
//
// Flutter: cupertino/text_selection_toolbar_button.dart CupertinoTextSelectionToolbarButton.buttonItem
func NewCupertinoTextSelectionToolbarButtonButtonItem(buttonItem widgets.ContextMenuButtonItem, opts ...CupertinoTextSelectionToolbarButtonOpts) *CupertinoTextSelectionToolbarButton {
	o := firstOf(opts)
	return &CupertinoTextSelectionToolbarButton{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), buttonItem: &buttonItem, onPressed: buttonItem.OnPressed}
}

// Child is the widget that the button shows, if it was made with one.
//
// Flutter: cupertino/text_selection_toolbar_button.dart CupertinoTextSelectionToolbarButton.child
func (w *CupertinoTextSelectionToolbarButton) Child() widgets.Widget { return w.child }

// OnPressed is called when the button is pressed.
//
// Flutter: cupertino/text_selection_toolbar_button.dart CupertinoTextSelectionToolbarButton.onPressed
func (w *CupertinoTextSelectionToolbarButton) OnPressed() func() { return w.onPressed }

// Text is the text that the button shows, if it was made with one.
//
// Flutter: cupertino/text_selection_toolbar_button.dart CupertinoTextSelectionToolbarButton.text
func (w *CupertinoTextSelectionToolbarButton) Text() foundation.Opt[string] { return w.text }

// CupertinoTextSelectionToolbarButtonGetButtonLabel is the label of the item: its own, or the localized one of its
// type (empty for the types that have none).
//
// Flutter: cupertino/text_selection_toolbar_button.dart CupertinoTextSelectionToolbarButton.getButtonLabel
func CupertinoTextSelectionToolbarButtonGetButtonLabel(bc widgets.BuildContext, buttonItem widgets.ContextMenuButtonItem) string {
	if l, ok := buttonItem.Label.Get(); ok {
		return l
	}
	DebugCheckHasCupertinoLocalizations(bc)
	localizations := CupertinoLocalizationsOf(bc)
	switch buttonItem.Type {
	case widgets.ContextMenuButtonTypeCut:
		return localizations.CutButtonLabel()
	case widgets.ContextMenuButtonTypeCopy:
		return localizations.CopyButtonLabel()
	case widgets.ContextMenuButtonTypePaste:
		return localizations.PasteButtonLabel()
	case widgets.ContextMenuButtonTypeSelectAll:
		return localizations.SelectAllButtonLabel()
	case widgets.ContextMenuButtonTypeLookUp:
		return localizations.LookUpButtonLabel()
	case widgets.ContextMenuButtonTypeSearchWeb:
		return localizations.SearchWebButtonLabel()
	case widgets.ContextMenuButtonTypeShare:
		return localizations.ShareButtonLabel()
	}
	return ""
}

// CreateState creates the state.
//
// Flutter: cupertino/text_selection_toolbar_button.dart CupertinoTextSelectionToolbarButton.createState
func (w *CupertinoTextSelectionToolbarButton) CreateState() widgets.State {
	return &cupertinoTextSelectionToolbarButtonState{}
}

type cupertinoTextSelectionToolbarButtonState struct {
	widgets.StateOf[*CupertinoTextSelectionToolbarButton]
	isPressed bool
}

func (s *cupertinoTextSelectionToolbarButtonState) onTapDown(gestures.TapDownDetails) {
	s.SetState(func() { s.isPressed = true })
}

func (s *cupertinoTextSelectionToolbarButtonState) onTapUp(gestures.TapUpDetails) {
	s.SetState(func() { s.isPressed = false })
	if f := s.Widget().onPressed; f != nil {
		f()
	}
}

func (s *cupertinoTextSelectionToolbarButtonState) onTapCancel() {
	s.SetState(func() { s.isPressed = false })
}

func (s *cupertinoTextSelectionToolbarButtonState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	content := s.contentWidget(bc)
	color := ui.ColorValue(CupertinoColors.Transparent)
	if s.isPressed {
		color = toolbarPressedColor
	}
	var child widgets.Widget = NewCupertinoButton(w.onPressed, content, CupertinoButtonOpts{
		Color:          CupertinoDynamicColorResolve(color, bc),
		DisabledColor:  CupertinoColors.Transparent,
		Padding:        toolbarButtonPadding,
		PressedOpacity: foundation.Some(1.0),
	})
	if w.onPressed != nil {
		return widgets.NewGestureDetector(child, widgets.GestureDetectorOpts{OnTapDown: s.onTapDown, OnTapUp: s.onTapUp, OnTapCancel: s.onTapCancel})
	}
	return child
}

func (s *cupertinoTextSelectionToolbarButtonState) contentWidget(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	if w.child != nil {
		return w.child
	}
	var label string
	if t, ok := w.text.Get(); ok {
		label = t
	} else {
		label = CupertinoTextSelectionToolbarButtonGetButtonLabel(bc, *w.buttonItem)
	}
	color := ui.ColorValue(CupertinoColors.InactiveGray)
	if w.onPressed != nil {
		color = toolbarTextColor
	}
	textWidget := widgets.NewText(label, widgets.TextOpts{Overflow: painting.TextOverflowEllipsis,
		Style: toolbarButtonFontStyle.CopyWith(painting.TextStyleCopy{Color: foundation.Some(CupertinoDynamicColorResolve(color, bc))})})
	if w.buttonItem != nil && w.buttonItem.Type == widgets.ContextMenuButtonTypeLiveTextInput {
		return widgets.NewSizedBoxSquare(foundation.Some(13.0), widgets.NewCustomPaint(nil, widgets.CustomPaintOpts{
			Painter: &liveTextIconPainter{CustomPainterBase: rendering.NewCustomPainterBase(nil), color: CupertinoDynamicColorResolve(toolbarTextColor, bc)}}))
	}
	return textWidget
}

// liveTextIconPainter paints the icon of the Live Text button.
//
// Flutter: cupertino/text_selection_toolbar_button.dart _LiveTextIconPainter
type liveTextIconPainter struct {
	rendering.CustomPainterBase
	color ui.Color
}

func (p *liveTextIconPainter) Paint(canvas *ui.Canvas, size ui.Size) {
	paint := ui.NewPaint()
	paint.SetStrokeCap(ui.StrokeCapRound)
	paint.SetStrokeJoin(ui.StrokeJoinRound)
	paint.SetStrokeWidth(1.0)
	paint.SetStyle(ui.PaintingStyleStroke)
	paint.SetColor(p.color)
	canvas.Save()
	canvas.Translate(size.Width/2.0, size.Height/2.0)
	origin := ui.Offset{Dx: -size.Width / 2.0, Dy: -size.Height / 2.0}
	path := ui.NewPath()
	path.MoveTo(origin.Dx, origin.Dy+3.5)
	path.LineTo(origin.Dx, origin.Dy+1.0)
	path.ArcToPoint(ui.Offset{Dx: origin.Dx + 1.0, Dy: origin.Dy}, ui.ArcToPointOpts{Radius: ui.NewRadiusCircular(1)})
	path.LineTo(origin.Dx+3.5, origin.Dy)
	rotation := ui.NewMatrix4Identity()
	rotation.RotateZ(math.Pi / 2.0)
	for range 4 {
		canvas.DrawPath(path, paint)
		canvas.Transform(rotation)
	}
	canvas.DrawLine(ui.Offset{Dx: -3.0, Dy: -3.0}, ui.Offset{Dx: 3.0, Dy: -3.0}, paint)
	canvas.DrawLine(ui.Offset{Dx: -3.0, Dy: 0.0}, ui.Offset{Dx: 3.0, Dy: 0.0}, paint)
	canvas.DrawLine(ui.Offset{Dx: -3.0, Dy: 3.0}, ui.Offset{Dx: 1.0, Dy: 3.0}, paint)
	canvas.Restore()
}

func (p *liveTextIconPainter) ShouldRepaint(old rendering.CustomPainter) bool {
	o, ok := old.(*liveTextIconPainter)
	return !ok || o.color != p.color
}

func (p *liveTextIconPainter) HitTest(ui.Offset) foundation.Opt[bool] { return foundation.None[bool]() }
