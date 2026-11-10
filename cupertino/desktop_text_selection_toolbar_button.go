package cupertino

import (
	"math"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// desktopToolbarButtonFontStyle is the style of the text of a button of the desktop toolbar.
var desktopToolbarButtonFontStyle = painting.NewTextStyle(painting.TextStyleOpts{
	Inherit: foundation.Some(false), FontSize: foundation.Some(14.0), LetterSpacing: foundation.Some(-0.15), FontWeight: ui.FontWeightW400})

// desktopToolbarButtonPadding is the padding of a button of the desktop toolbar.
var desktopToolbarButtonPadding = painting.NewEdgeInsetsFromLTRB(8.0, 2.0, 8.0, 5.0)

// CupertinoDesktopTextSelectionToolbarButtonOpts are the optional parameters of the constructors of
// CupertinoDesktopTextSelectionToolbarButton.
//
// Flutter: cupertino/desktop_text_selection_toolbar_button.dart CupertinoDesktopTextSelectionToolbarButton.new
type CupertinoDesktopTextSelectionToolbarButtonOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// CupertinoDesktopTextSelectionToolbarButton is a button in the style of the context menu of the text of macOS.
//
// Flutter: cupertino/desktop_text_selection_toolbar_button.dart CupertinoDesktopTextSelectionToolbarButton
type CupertinoDesktopTextSelectionToolbarButton struct {
	widgets.StatefulWidget
	onPressed  func()
	child      widgets.Widget
	buttonItem *widgets.ContextMenuButtonItem
	text       foundation.Opt[string]
}

// NewCupertinoDesktopTextSelectionToolbarButton creates a button that shows the child.
//
// Flutter: cupertino/desktop_text_selection_toolbar_button.dart CupertinoDesktopTextSelectionToolbarButton.new
func NewCupertinoDesktopTextSelectionToolbarButton(onPressed func(), child widgets.Widget, opts ...CupertinoDesktopTextSelectionToolbarButtonOpts) *CupertinoDesktopTextSelectionToolbarButton {
	o := firstOf(opts)
	return &CupertinoDesktopTextSelectionToolbarButton{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), onPressed: onPressed, child: child}
}

// NewCupertinoDesktopTextSelectionToolbarButtonText creates a button that shows the text.
//
// Flutter: cupertino/desktop_text_selection_toolbar_button.dart CupertinoDesktopTextSelectionToolbarButton.text
func NewCupertinoDesktopTextSelectionToolbarButtonText(onPressed func(), text string, opts ...CupertinoDesktopTextSelectionToolbarButtonOpts) *CupertinoDesktopTextSelectionToolbarButton {
	o := firstOf(opts)
	return &CupertinoDesktopTextSelectionToolbarButton{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), onPressed: onPressed, text: foundation.Some(text)}
}

// NewCupertinoDesktopTextSelectionToolbarButtonButtonItem creates a button from the item of a context menu.
//
// Flutter: cupertino/desktop_text_selection_toolbar_button.dart CupertinoDesktopTextSelectionToolbarButton.buttonItem
func NewCupertinoDesktopTextSelectionToolbarButtonButtonItem(buttonItem widgets.ContextMenuButtonItem, opts ...CupertinoDesktopTextSelectionToolbarButtonOpts) *CupertinoDesktopTextSelectionToolbarButton {
	o := firstOf(opts)
	return &CupertinoDesktopTextSelectionToolbarButton{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), onPressed: buttonItem.OnPressed, buttonItem: &buttonItem}
}

// OnPressed is called when the button is pressed.
//
// Flutter: cupertino/desktop_text_selection_toolbar_button.dart CupertinoDesktopTextSelectionToolbarButton.onPressed
func (w *CupertinoDesktopTextSelectionToolbarButton) OnPressed() func() { return w.onPressed }

// Child is the widget that the button shows, if it was made with one.
//
// Flutter: cupertino/desktop_text_selection_toolbar_button.dart CupertinoDesktopTextSelectionToolbarButton.child
func (w *CupertinoDesktopTextSelectionToolbarButton) Child() widgets.Widget { return w.child }

// Text is the text that the button shows, if it was made with one.
//
// Flutter: cupertino/desktop_text_selection_toolbar_button.dart CupertinoDesktopTextSelectionToolbarButton.text
func (w *CupertinoDesktopTextSelectionToolbarButton) Text() foundation.Opt[string] { return w.text }

// CreateState creates the state.
//
// Flutter: cupertino/desktop_text_selection_toolbar_button.dart CupertinoDesktopTextSelectionToolbarButton.createState
func (w *CupertinoDesktopTextSelectionToolbarButton) CreateState() widgets.State {
	return &cupertinoDesktopTextSelectionToolbarButtonState{}
}

type cupertinoDesktopTextSelectionToolbarButtonState struct {
	widgets.StateOf[*CupertinoDesktopTextSelectionToolbarButton]
	isHovered bool
}

func (s *cupertinoDesktopTextSelectionToolbarButtonState) onEnter(*gestures.PointerEnterEvent) {
	s.SetState(func() { s.isHovered = true })
}

func (s *cupertinoDesktopTextSelectionToolbarButtonState) onExit(*gestures.PointerExitEvent) {
	s.SetState(func() { s.isHovered = false })
}

func (s *cupertinoDesktopTextSelectionToolbarButtonState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	child := w.child
	if child == nil {
		var label string
		if t, ok := w.text.Get(); ok {
			label = t
		} else {
			label = CupertinoTextSelectionToolbarButtonGetButtonLabel(bc, *w.buttonItem)
		}
		color := CupertinoDynamicColorResolve(NewCupertinoDynamicColorWithBrightness(CupertinoColors.Black, CupertinoColors.White), bc)
		if s.isHovered {
			color = CupertinoDynamicColorResolve(CupertinoThemeOf(bc).PrimaryContrastingColor(), bc)
		}
		child = widgets.NewText(label, widgets.TextOpts{Overflow: painting.TextOverflowEllipsis,
			Style: desktopToolbarButtonFontStyle.CopyWith(painting.TextStyleCopy{Color: foundation.Some(color)})})
	}
	var color ui.ColorValue
	if s.isHovered {
		color = CupertinoThemeOf(bc).PrimaryColor()
	}
	return widgets.NewSizedBox(widgets.NewMouseRegion(NewCupertinoButton(w.onPressed, child, CupertinoButtonOpts{
		Alignment:      painting.AlignmentCenterLeft,
		BorderRadius:   foundation.Some(painting.NewBorderRadiusAll(ui.NewRadiusCircular(4.0))),
		Color:          color,
		MinimumSize:    foundation.Some(ui.Size{}),
		Padding:        desktopToolbarButtonPadding,
		PressedOpacity: foundation.Some(0.7),
	}), widgets.MouseRegionOpts{OnEnter: s.onEnter, OnExit: s.onExit}), widgets.SizedBoxOpts{Width: foundation.Some(math.Inf(1))})
}
