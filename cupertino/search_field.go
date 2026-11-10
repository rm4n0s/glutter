package cupertino

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	// searchFieldMinHeightBeforeTotalTransparency is the part of the height under which the icons of the field
	// are invisible when the list under it scrolls.
	//
	// Flutter: cupertino/search_field.dart _kMinHeightBeforeTotalTransparency
	searchFieldMinHeightBeforeTotalTransparency = 4.0 / 5.0
	// searchFieldMaxPrefixIconSize is the size from which the prefix icon is hidden while the field has the focus.
	//
	// Flutter: cupertino/search_field.dart _kMaxPrefixIconSize
	searchFieldMaxPrefixIconSize = 30.0
)

// CupertinoSearchTextFieldOpts are the optional parameters of NewCupertinoSearchTextField.
//
// Deviation: Flutter's restorationId is not ported (SPEC §12.10).
//
// Flutter: cupertino/search_field.dart CupertinoSearchTextField.new
type CupertinoSearchTextFieldOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Controller is the controller of the text; the default is one of the field's own.
	Controller *widgets.TextEditingController
	// OnChanged is called when the text changes.
	OnChanged func(text string)
	// OnSubmitted is called when the user submits the text.
	OnSubmitted func(text string)
	// Style is the style of the text.
	Style *painting.TextStyle
	// Placeholder is the text shown while the field is empty; the default is the localized "Search".
	Placeholder foundation.Opt[string]
	// PlaceholderStyle is the style of the placeholder; the default is the secondary label color.
	PlaceholderStyle *painting.TextStyle
	// Decoration is painted behind the field; it excludes BackgroundColor and BorderRadius.
	Decoration *painting.BoxDecoration
	// BackgroundColor is the color of the default decoration; the default is the tertiary system fill.
	BackgroundColor ui.ColorValue
	// BorderRadius is the corners of the default decoration; the default is 9.
	BorderRadius painting.BorderRadiusGeometry
	// KeyboardType is the type of keyboard; the default is text.
	KeyboardType foundation.Opt[services.TextInputType]
	// Padding is the padding of the text; the default is 5.5, 8, 5.5, 8 (start, top, end, bottom).
	Padding painting.EdgeInsetsGeometry
	// ItemColor is the color of the icons; the default is the secondary label.
	ItemColor ui.ColorValue
	// ItemSize is the size of the icons before the text scale; the default is 20.
	ItemSize foundation.Opt[float64]
	// PrefixInsets is the padding around the prefix icon; the default is 6, 8, 0, 8.
	PrefixInsets painting.EdgeInsetsGeometry
	// PrefixIcon is the icon before the text; the default is the search icon.
	PrefixIcon widgets.Widget
	// SuffixInsets is the padding around the suffix icon; the default is 0, 8, 5, 8.
	SuffixInsets painting.EdgeInsetsGeometry
	// SuffixIcon is the icon of the clear button; the default is the filled circle with a cross.
	SuffixIcon *widgets.Icon
	// SuffixMode is when the suffix is shown; the default is editing.
	SuffixMode OverlayVisibilityMode
	// OnSuffixTap is called when the suffix is tapped; the default clears the text.
	OnSuffixTap func()
	// FocusNode is the focus node of the field; the default is one of the field's own.
	FocusNode *widgets.FocusNode
	// SmartQuotesType is how quotes are made smart.
	SmartQuotesType services.SmartQuotesType
	// SmartDashesType is how dashes are made smart.
	SmartDashesType services.SmartDashesType
	// DisableIMEPersonalizedLearning turns off the personalized learning of the keyboard (Flutter's
	// enableIMEPersonalizedLearning is true by default).
	DisableIMEPersonalizedLearning bool
	// Autofocus asks for the focus when the field is built.
	Autofocus bool
	// OnTap is called when the field is tapped.
	OnTap func()
	// Autocorrect is whether the keyboard corrects the text; the default is true.
	Autocorrect foundation.Opt[bool]
	// Enabled is whether the field answers input; the default is true.
	Enabled foundation.Opt[bool]
	// CursorWidth is the width of the cursor; the default is 2.
	CursorWidth foundation.Opt[float64]
	// CursorHeight is the height of the cursor; the default is the height of the font.
	CursorHeight foundation.Opt[float64]
	// CursorRadius is the radius of the corners of the cursor; the default is 2.
	CursorRadius foundation.Opt[ui.Radius]
	// CursorOpacityAnimates is whether the cursor fades in and out; the default is true.
	CursorOpacityAnimates foundation.Opt[bool]
	// CursorColor is the color of the cursor.
	CursorColor ui.ColorValue
}

// CupertinoSearchTextField is the search field of iOS: a CupertinoTextField with a search icon, a clear button and
// a rounded gray background. Its icons fade out when the list under it scrolls and shrinks its bar.
//
// Flutter: cupertino/search_field.dart CupertinoSearchTextField
type CupertinoSearchTextField struct {
	widgets.StatefulWidget
	o CupertinoSearchTextFieldOpts
}

// NewCupertinoSearchTextField creates a search field. In debug builds it panics with the tags
// CupertinoSearchTextFieldDecorationAndBackgroundColor and CupertinoSearchTextFieldDecorationAndBorderRadius when
// a decoration is given with a background color or a border radius.
//
// Flutter: cupertino/search_field.dart CupertinoSearchTextField.new
func NewCupertinoSearchTextField(opts ...CupertinoSearchTextFieldOpts) *CupertinoSearchTextField {
	o := firstOf(opts)
	if foundation.DebugMode {
		if o.Decoration != nil && o.BackgroundColor != nil {
			panic(errors.New(tagCupertinoSearchTextFieldDecorationAndBackgroundColor, "cannot provide both a background color and a decoration; set the color in the BoxDecoration instead"))
		}
		if o.Decoration != nil && o.BorderRadius != nil {
			panic(errors.New(tagCupertinoSearchTextFieldDecorationAndBorderRadius, "cannot provide both a border radius and a decoration; set the radius in the BoxDecoration instead"))
		}
	}
	if o.KeyboardType.IsNone() {
		o.KeyboardType = foundation.Some(services.TextInputTypeText)
	}
	if o.Padding == nil {
		o.Padding = painting.NewEdgeInsetsDirectionalFromSTEB(5.5, 8, 5.5, 8)
	}
	if o.ItemColor == nil {
		o.ItemColor = CupertinoColors.SecondaryLabel
	}
	if o.PrefixInsets == nil {
		o.PrefixInsets = painting.NewEdgeInsetsDirectionalFromSTEB(6, 8, 0, 8)
	}
	if o.PrefixIcon == nil {
		o.PrefixIcon = widgets.NewIcon(CupertinoIcons.Search())
	}
	if o.SuffixInsets == nil {
		o.SuffixInsets = painting.NewEdgeInsetsDirectionalFromSTEB(0, 8, 5, 8)
	}
	if o.SuffixIcon == nil {
		o.SuffixIcon = widgets.NewIcon(CupertinoIcons.XmarkCircleFill())
	}
	if o.SuffixMode == OverlayVisibilityModeUnspecified {
		o.SuffixMode = OverlayVisibilityModeEditing
	}
	return &CupertinoSearchTextField{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), o: o}
}

// Controller is the controller of the text, if one was given.
//
// Flutter: cupertino/search_field.dart CupertinoSearchTextField.controller
func (w *CupertinoSearchTextField) Controller() *widgets.TextEditingController { return w.o.Controller }

// FocusNode is the focus node of the field, if one was given.
//
// Flutter: cupertino/search_field.dart CupertinoSearchTextField.focusNode
func (w *CupertinoSearchTextField) FocusNode() *widgets.FocusNode { return w.o.FocusNode }

// CreateState creates the state.
//
// Flutter: cupertino/search_field.dart CupertinoSearchTextField.createState
func (w *CupertinoSearchTextField) CreateState() widgets.State {
	return &cupertinoSearchTextFieldState{}
}

type cupertinoSearchTextFieldState struct {
	widgets.StateOf[*CupertinoSearchTextField]
	controller                 *widgets.TextEditingController
	focusNode                  *widgets.FocusNode
	scrollNotificationObserver *widgets.ScrollNotificationObserverState
	scrollObserverHandle       widgets.ScrollNotificationListenerHandle
	scaledIconSize             float64
	fadeExtent                 float64
}

func (s *cupertinoSearchTextFieldState) effectiveController() *widgets.TextEditingController {
	if c := s.Widget().o.Controller; c != nil {
		return c
	}
	return s.controller
}

func (s *cupertinoSearchTextFieldState) effectiveFocusNode() *widgets.FocusNode {
	if f := s.Widget().o.FocusNode; f != nil {
		return f
	}
	return s.focusNode
}

func (s *cupertinoSearchTextFieldState) InitState() {
	if s.Widget().o.Controller == nil {
		s.controller = widgets.NewTextEditingController()
	}
	if s.Widget().o.FocusNode == nil {
		s.focusNode = widgets.NewFocusNode(widgets.FocusNodeOpts{})
	}
}

func (s *cupertinoSearchTextFieldState) removeScrollListener() {
	if s.scrollNotificationObserver != nil {
		s.scrollNotificationObserver.RemoveListener(s.scrollObserverHandle)
		s.scrollNotificationObserver = nil
	}
}

func (s *cupertinoSearchTextFieldState) DidChangeDependencies() {
	s.removeScrollListener()
	s.scrollNotificationObserver = widgets.ScrollNotificationObserverMaybeOf(s.Context())
	if s.scrollNotificationObserver != nil {
		s.scrollObserverHandle = s.scrollNotificationObserver.AddListener(s.handleScrollNotification)
	}
}

func (s *cupertinoSearchTextFieldState) DidUpdateWidget(old *CupertinoSearchTextField) {
	w := s.Widget()
	if w.o.Controller == nil && old.o.Controller != nil {
		s.controller = widgets.NewTextEditingControllerFromValue(old.o.Controller.Value())
	} else if w.o.Controller != nil && old.o.Controller == nil {
		s.controller.Dispose()
		s.controller = nil
	}
	if w.o.FocusNode == nil && old.o.FocusNode != nil {
		s.focusNode = widgets.NewFocusNode(widgets.FocusNodeOpts{})
	} else if w.o.FocusNode != nil && old.o.FocusNode == nil {
		s.focusNode.Dispose()
		s.focusNode = nil
	}
}

func (s *cupertinoSearchTextFieldState) Dispose() {
	s.removeScrollListener()
	if s.focusNode != nil {
		s.focusNode.Dispose()
	}
	if s.controller != nil {
		s.controller.Dispose()
	}
}

func (s *cupertinoSearchTextFieldState) defaultOnSuffixTap() {
	c := s.effectiveController()
	textChanged := c.Text() != ""
	c.Clear()
	if f := s.Widget().o.OnChanged; f != nil && textChanged {
		f(c.Text())
	}
}

func (s *cupertinoSearchTextFieldState) handleScrollNotification(notification widgets.ScrollNotification) {
	if _, ok := notification.(*widgets.ScrollUpdateNotification); !ok {
		return
	}
	currentHeight := 0.0
	if size, ok := s.Context().Size(); ok {
		currentHeight = size.Height
	}
	w := s.Widget()
	s.SetState(func() {
		s.fadeExtent = searchFieldScrollOpacity(currentHeight,
			s.scaledIconSize+math.Max(w.o.PrefixInsets.Vertical(), w.o.SuffixInsets.Vertical()))
	})
}

// searchFieldScrollOpacity is how much the icons are faded out for the current height of the field.
//
// Flutter: cupertino/search_field.dart _CupertinoSearchTextFieldState._calculateScrollOpacity
func searchFieldScrollOpacity(currentHeight, maxHeight float64) float64 {
	thresholdHeight := maxHeight * searchFieldMinHeightBeforeTotalTransparency
	switch {
	case currentHeight >= maxHeight:
		return 0.0
	case currentHeight <= thresholdHeight:
		return 1.0
	}
	progress := (currentHeight - thresholdHeight) / (maxHeight - thresholdHeight)
	return 1.0 - progress
}

func (s *cupertinoSearchTextFieldState) animatedInsets(bc widgets.BuildContext, insets painting.EdgeInsetsGeometry) painting.EdgeInsetsGeometry {
	current := insets.Resolve(widgets.DirectionalityOf(bc))
	if animated := painting.EdgeInsetsGeometryLerp(insets, current.CopyWith(painting.EdgeInsetsCopy{Top: foundation.Some(current.Top / 2)}), s.fadeExtent); animated != nil {
		return animated
	}
	return insets
}

func (s *cupertinoSearchTextFieldState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	placeholder, ok := w.o.Placeholder.Get()
	if !ok {
		placeholder = CupertinoLocalizationsOf(bc).SearchTextFieldPlaceholderLabel()
	}
	placeholderStyle := w.o.PlaceholderStyle
	if placeholderStyle == nil {
		defaultColor := CupertinoDynamicColorResolve(CupertinoColors.SecondaryLabel, bc)
		placeholderStyle = painting.NewTextStyle(painting.TextStyleOpts{
			Color: foundation.Some(defaultColor.WithAlpha(int(math.Round(255 * (defaultColor.Opacity() * (1 - s.fadeExtent)))))),
		})
	}
	itemSize := w.o.ItemSize.OrElse(20.0)
	s.scaledIconSize = widgets.MediaQueryTextScalerOf(bc).Scale(itemSize)
	decoration := w.o.Decoration
	if decoration == nil {
		var radius painting.BorderRadiusGeometry = painting.NewBorderRadiusAll(ui.NewRadiusCircular(9.0))
		if w.o.BorderRadius != nil {
			radius = w.o.BorderRadius
		}
		var fill ui.ColorValue = CupertinoColors.TertiarySystemFill
		if w.o.BackgroundColor != nil {
			fill = w.o.BackgroundColor
		}
		decoration = painting.NewBoxDecoration(painting.BoxDecorationOpts{
			Color: foundation.Some(CupertinoDynamicColorResolve(fill, bc)), BorderRadius: radius,
		})
	}
	iconColor := CupertinoDynamicColorResolve(w.o.ItemColor, bc)
	suffixIconTheme := widgets.NewIconThemeData(widgets.IconThemeDataOpts{Color: foundation.Some(iconColor), Size: foundation.Some(s.scaledIconSize)})
	prefixSize := s.scaledIconSize
	if s.scaledIconSize >= searchFieldMaxPrefixIconSize && s.effectiveFocusNode().HasFocus() {
		prefixSize = 0.0
	}
	prefixIconTheme := widgets.NewIconThemeData(widgets.IconThemeDataOpts{Color: foundation.Some(iconColor), Size: foundation.Some(prefixSize)})
	prefix := widgets.NewOpacity(1.0-s.fadeExtent, widgets.NewPadding(s.animatedInsets(bc, w.o.PrefixInsets),
		widgets.NewIconTheme(prefixIconTheme, w.o.PrefixIcon)))
	onSuffixTap := w.o.OnSuffixTap
	if onSuffixTap == nil {
		onSuffixTap = s.defaultOnSuffixTap
	}
	suffix := widgets.NewOpacity(1.0-s.fadeExtent, widgets.NewPadding(s.animatedInsets(bc, w.o.SuffixInsets),
		NewCupertinoButton(onSuffixTap, widgets.NewIconTheme(suffixIconTheme, w.o.SuffixIcon),
			CupertinoButtonOpts{MinimumSize: foundation.Some(ui.Size{}), Padding: painting.EdgeInsets{}})))
	return NewCupertinoTextField(CupertinoTextFieldOpts{
		Controller:                     s.effectiveController(),
		Decoration:                     decoration,
		Style:                          w.o.Style,
		Prefix:                         prefix,
		Suffix:                         suffix,
		KeyboardType:                   w.o.KeyboardType,
		OnTap:                          w.o.OnTap,
		Enabled:                        foundation.Some(w.o.Enabled.OrElse(true)),
		CursorWidth:                    foundation.Some(w.o.CursorWidth.OrElse(2.0)),
		CursorHeight:                   w.o.CursorHeight,
		CursorRadius:                   foundation.Some(w.o.CursorRadius.OrElse(ui.NewRadiusCircular(2.0))),
		CursorOpacityAnimates:          foundation.Some(w.o.CursorOpacityAnimates.OrElse(true)),
		CursorColor:                    w.o.CursorColor,
		SuffixMode:                     w.o.SuffixMode,
		Placeholder:                    foundation.Some(placeholder),
		PlaceholderStyle:               placeholderStyle,
		Padding:                        s.animatedInsets(bc, w.o.Padding),
		OnChanged:                      w.o.OnChanged,
		OnSubmitted:                    w.o.OnSubmitted,
		FocusNode:                      s.effectiveFocusNode(),
		Autofocus:                      w.o.Autofocus,
		Autocorrect:                    foundation.Some(w.o.Autocorrect.OrElse(true)),
		SmartQuotesType:                w.o.SmartQuotesType,
		SmartDashesType:                w.o.SmartDashesType,
		DisableIMEPersonalizedLearning: w.o.DisableIMEPersonalizedLearning,
		TextInputAction:                services.TextInputActionSearch,
		DragStartBehavior:              gestures.DragStartBehaviorStart,
	})
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCupertinoSearchTextFieldDecorationAndBackgroundColor = "CupertinoSearchTextFieldDecorationAndBackgroundColor"
	tagCupertinoSearchTextFieldDecorationAndBorderRadius    = "CupertinoSearchTextFieldDecorationAndBorderRadius"
)
