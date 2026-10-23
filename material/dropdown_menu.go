// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	// dropdownMenuMinimumWidth is the minimum width of the menu.
	//
	// Flutter: material/dropdown_menu.dart _kMinimumWidth
	dropdownMenuMinimumWidth = 112.0
	// dropdownMenuDefaultHorizontalPadding is the horizontal padding of an entry.
	//
	// Flutter: material/dropdown_menu.dart _kDefaultHorizontalPadding
	dropdownMenuDefaultHorizontalPadding = 12.0
	// dropdownMenuInputStartGap is the gap before the label of an entry with
	// Material 3.
	//
	// Flutter: material/dropdown_menu.dart _kInputStartGap
	dropdownMenuInputStartGap = 4.0
)

// DropdownMenuFilterCallback is the type of the function that filters the
// entries of a DropdownMenu for the text of its field.
//
// Flutter: material/dropdown_menu.dart FilterCallback
type DropdownMenuFilterCallback[T comparable] func(entries []DropdownMenuEntry[T], filter string) []DropdownMenuEntry[T]

// DropdownMenuSearchCallback is the type of the function that finds the index of
// the entry that matches the text of the field, or none.
//
// Flutter: material/dropdown_menu.dart SearchCallback
type DropdownMenuSearchCallback[T comparable] func(entries []DropdownMenuEntry[T], query string) foundation.Opt[int]

// DropdownMenuDecorationBuilder is the type of the function that builds the
// decoration of the text field of a DropdownMenu.
//
// Flutter: material/dropdown_menu.dart DropdownMenuDecorationBuilder
type DropdownMenuDecorationBuilder func(bc widgets.BuildContext, controller *widgets.MenuController) InputDecoration

// DropdownMenuEntry describes an entry of a DropdownMenu: a value and the label
// that stands for it.
//
// Flutter: material/dropdown_menu.dart DropdownMenuEntry
type DropdownMenuEntry[T comparable] struct {
	// Value is the value that the entry stands for.
	Value T
	// Label is the text of the entry, which is the text of the field when the
	// entry is selected.
	Label string
	// LabelWidget is a widget to show instead of the label.
	LabelWidget widgets.Widget
	// LeadingIcon is an icon before the label.
	LeadingIcon widgets.Widget
	// TrailingIcon is an icon after the label.
	TrailingIcon widgets.Widget
	// Disabled makes the entry not selectable (Flutter's enabled: false).
	Disabled bool
	// Style is the style of the button of the entry.
	Style *ButtonStyle
}

// DropdownMenuEntryOpts are the optional parameters of NewDropdownMenuEntry.
//
// Flutter: material/dropdown_menu.dart DropdownMenuEntry.new
type DropdownMenuEntryOpts struct {
	// LabelWidget is a widget to show instead of the label.
	LabelWidget widgets.Widget
	// LeadingIcon is an icon before the label.
	LeadingIcon widgets.Widget
	// TrailingIcon is an icon after the label.
	TrailingIcon widgets.Widget
	// Disabled makes the entry not selectable.
	Disabled bool
	// Style is the style of the button of the entry.
	Style *ButtonStyle
}

// NewDropdownMenuEntry creates an entry.
//
// Flutter: material/dropdown_menu.dart DropdownMenuEntry.new
func NewDropdownMenuEntry[T comparable](value T, label string, opts ...DropdownMenuEntryOpts) DropdownMenuEntry[T] {
	o := firstOpts(opts)
	return DropdownMenuEntry[T]{Value: value, Label: label, LabelWidget: o.LabelWidget, LeadingIcon: o.LeadingIcon,
		TrailingIcon: o.TrailingIcon, Disabled: o.Disabled, Style: o.Style}
}

// DropdownMenuCloseBehavior defines the behavior of the menu when an entry is
// selected.
//
// Flutter: material/dropdown_menu.dart DropdownMenuCloseBehavior
type DropdownMenuCloseBehavior int

// The values of DropdownMenuCloseBehavior follow Flutter's declaration order.
const (
	// DropdownMenuCloseBehaviorUnspecified is the zero value: the default, all.
	//
	// Flutter: no counterpart
	DropdownMenuCloseBehaviorUnspecified DropdownMenuCloseBehavior = iota
	// DropdownMenuCloseBehaviorAll closes all the menus that are open, including
	// the ones of a parent menu.
	//
	// Flutter: material/dropdown_menu.dart DropdownMenuCloseBehavior.all
	DropdownMenuCloseBehaviorAll
	// DropdownMenuCloseBehaviorSelf closes only this menu.
	//
	// Flutter: material/dropdown_menu.dart DropdownMenuCloseBehavior.self
	DropdownMenuCloseBehaviorSelf
	// DropdownMenuCloseBehaviorNone leaves the menus open.
	//
	// Flutter: material/dropdown_menu.dart DropdownMenuCloseBehavior.none
	DropdownMenuCloseBehaviorNone
)

// Name returns the Dart enum name, such as "all".
//
// Flutter: material/dropdown_menu.dart DropdownMenuCloseBehavior.name
func (b DropdownMenuCloseBehavior) Name() string { return enumName(int(b), "all", "self", "none") }

// String is the name with the type, like Dart's enum toString.
func (b DropdownMenuCloseBehavior) String() string { return "DropdownMenuCloseBehavior." + b.Name() }

// DropdownMenuOpts are the optional parameters of NewDropdownMenu.
//
// Flutter: material/dropdown_menu.dart DropdownMenu.new
type DropdownMenuOpts[T comparable] struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Disabled makes the menu not respond to the user (Flutter's enabled: false).
	Disabled bool
	// Width is the width of the menu and the field; the default is that of the
	// widest entry.
	Width foundation.Opt[float64]
	// MenuHeight is the maximum height of the menu; the default is that of the
	// menu style.
	MenuHeight foundation.Opt[float64]
	// LeadingIcon is an icon before the text of the field.
	LeadingIcon widgets.Widget
	// TrailingIcon is the icon after the text of the field that shows the menu is
	// closed; the default is Icons.arrow_drop_down.
	TrailingIcon widgets.Widget
	// NoShowTrailingIcon leaves the trailing icon out (Flutter's
	// showTrailingIcon is true by default).
	NoShowTrailingIcon bool
	// TrailingIconFocusNode is the focus node of the trailing icon button.
	TrailingIconFocusNode *widgets.FocusNode
	// Label is the widget that labels the field.
	Label widgets.Widget
	// HintText is the hint of the field.
	HintText foundation.Opt[string]
	// HelperText is the text below the field.
	HelperText foundation.Opt[string]
	// ErrorText is the error text below the field.
	ErrorText foundation.Opt[string]
	// SelectedTrailingIcon is the icon after the text of the field that shows the
	// menu is open; the default is Icons.arrow_drop_up.
	SelectedTrailingIcon widgets.Widget
	// EnableFilter shows only the entries that match the text of the field.
	EnableFilter bool
	// NoEnableSearch keeps the entry that matches the text from being
	// highlighted (Flutter's enableSearch is true by default).
	NoEnableSearch bool
	// KeyboardType is the kind of text the input method offers.
	KeyboardType foundation.Opt[services.TextInputType]
	// TextStyle is the style of the text of the field and the entries.
	TextStyle *painting.TextStyle
	// TextAlign is how the text of the field is aligned horizontally.
	TextAlign ui.TextAlign
	// InputDecorationTheme is the decoration theme of the field.
	InputDecorationTheme *InputDecorationThemeData
	// DecorationBuilder builds the decoration of the field; it can't be given
	// with the Label, HintText, HelperText or ErrorText.
	DecorationBuilder DropdownMenuDecorationBuilder
	// MenuStyle is the style of the menu.
	MenuStyle *MenuStyle
	// Controller controls the text of the field.
	Controller *widgets.TextEditingController
	// InitialSelection is the value of the entry that is selected at first.
	InitialSelection foundation.Opt[T]
	// OnSelected is called with the value of the entry that the user selects.
	OnSelected func(value foundation.Opt[T])
	// FocusNode is the focus node of the field.
	FocusNode *widgets.FocusNode
	// RequestFocusOnTap says whether a tap on the field takes the focus and shows
	// the keyboard; the default is true on desktop.
	RequestFocusOnTap foundation.Opt[bool]
	// SelectOnly makes the field read-only, like a button.
	SelectOnly bool
	// ExpandedInsets makes the menu as wide as its parent, less the insets.
	ExpandedInsets painting.EdgeInsetsGeometry
	// FilterCallback filters the entries for the text; it needs EnableFilter.
	FilterCallback DropdownMenuFilterCallback[T]
	// SearchCallback finds the entry to highlight for the text.
	SearchCallback DropdownMenuSearchCallback[T]
	// InputFormatters change the text the user enters.
	InputFormatters []services.TextInputFormatter
	// AlignmentOffset is the offset of the menu from the field.
	AlignmentOffset foundation.Opt[ui.Offset]
	// CloseBehavior defines what happens to the menus when an entry is selected;
	// the default is all.
	CloseBehavior DropdownMenuCloseBehavior
	// MaxLines is the maximum number of lines of the field; the default is 1.
	MaxLines foundation.Opt[int]
	// TextInputAction is the action of the Enter key.
	TextInputAction services.TextInputAction
	// CursorHeight is the height of the caret.
	CursorHeight foundation.Opt[float64]
	// MenuController controls the menu from outside.
	MenuController *widgets.MenuController
	// ScrollPadding is the space kept around the caret when the field scrolls to
	// show it; the default is 20 on all sides.
	ScrollPadding foundation.Opt[painting.EdgeInsets]
}

// DropdownMenu is a Material 3 text field with a menu of entries: the user
// picks one, or types to filter and search them.
//
// Deviation: the restoration of the state is not ported, and the
// inputDecorationTheme is the data, not the InputDecorationTheme widget (SPEC
// §12.10).
//
// Flutter: material/dropdown_menu.dart DropdownMenu
type DropdownMenu[T comparable] struct {
	widgets.StatefulWidget
	entries []DropdownMenuEntry[T]
	o       DropdownMenuOpts[T]
}

// NewDropdownMenu creates a menu of the entries. In debug builds it panics with
// the tag DropdownMenuFilterCallbackWithoutFilter when it has a filter callback
// and the filter is not enabled, DropdownMenuTrailingIconFocusNodeWithoutIcon
// when it has a focus node for the trailing icon and no icon, and
// DropdownMenuDecorationBuilderWithTexts when it has a decoration builder and
// a label, hint, helper or error.
//
// Flutter: material/dropdown_menu.dart DropdownMenu.new
func NewDropdownMenu[T comparable](dropdownMenuEntries []DropdownMenuEntry[T], opts ...DropdownMenuOpts[T]) *DropdownMenu[T] {
	o := firstOpts(opts)
	if foundation.DebugMode {
		if o.FilterCallback != nil && !o.EnableFilter {
			panic(errors.New(tagDropdownMenuFilterCallbackWithoutFilter, "a filterCallback needs enableFilter"))
		}
		if o.TrailingIconFocusNode != nil && o.NoShowTrailingIcon {
			panic(errors.New(tagDropdownMenuTrailingIconFocusNodeWithoutIcon, "a trailingIconFocusNode needs showTrailingIcon"))
		}
		if o.DecorationBuilder != nil && (o.Label != nil || o.HintText.IsSome() || o.HelperText.IsSome() || o.ErrorText.IsSome()) {
			panic(errors.New(tagDropdownMenuDecorationBuilderWithTexts, "a decorationBuilder can't be given with a label, hintText, helperText or errorText"))
		}
	}
	if o.CloseBehavior == DropdownMenuCloseBehaviorUnspecified {
		o.CloseBehavior = DropdownMenuCloseBehaviorAll
	}
	if o.MaxLines.IsNone() {
		o.MaxLines = foundation.Some(1)
	}
	return &DropdownMenu[T]{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), entries: dropdownMenuEntries, o: o}
}

// Entries are the entries of the menu.
//
// Flutter: material/dropdown_menu.dart DropdownMenu.dropdownMenuEntries
func (w *DropdownMenu[T]) Entries() []DropdownMenuEntry[T] { return w.entries }

// Enabled is whether the menu responds to the user.
//
// Flutter: material/dropdown_menu.dart DropdownMenu.enabled
func (w *DropdownMenu[T]) Enabled() bool { return !w.o.Disabled }

// CreateState creates the state.
//
// Flutter: material/dropdown_menu.dart DropdownMenu.createState
func (w *DropdownMenu[T]) CreateState() widgets.State { return &dropdownMenuFieldState[T]{} }

// The intents of the keys of the menu.
//
// Flutter: material/dropdown_menu.dart _ArrowUpIntent, _ArrowDownIntent, _EnterIntent
type (
	dropdownArrowUpIntent   struct{ widgets.IntentBase }
	dropdownArrowDownIntent struct{ widgets.IntentBase }
	dropdownEnterIntent     struct{ widgets.IntentBase }
)

func dropdownEditableShortcuts() widgets.ShortcutMap {
	return widgets.ShortcutMap{
		{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyArrowLeft), Intent: widgets.NewExtendSelectionByCharacterIntent(false, true)},
		{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyArrowRight), Intent: widgets.NewExtendSelectionByCharacterIntent(true, true)},
		{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyArrowUp), Intent: dropdownArrowUpIntent{}},
		{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyArrowDown), Intent: dropdownArrowDownIntent{}},
	}
}

func dropdownSelectOnlyShortcuts() widgets.ShortcutMap {
	return widgets.ShortcutMap{
		{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyArrowUp), Intent: dropdownArrowUpIntent{}},
		{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyArrowDown), Intent: dropdownArrowDownIntent{}},
		{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyEnter), Intent: dropdownEnterIntent{}},
	}
}

// dropdownMenuFieldState is the state of a DropdownMenu.
//
// Flutter: material/dropdown_menu.dart _DropdownMenuState
type dropdownMenuFieldState[T comparable] struct {
	widgets.StateOf[*DropdownMenu[T]]
	anchorKey                    *widgets.GlobalKey[widgets.State]
	leadingKey                   *widgets.GlobalKey[widgets.State]
	buttonItemKeys               []*widgets.GlobalKey[widgets.State]
	controller                   *widgets.MenuController
	enableFilter                 bool
	enableSearch                 bool
	filteredEntries              []DropdownMenuEntry[T]
	initialMenu                  []widgets.Widget
	currentHighlight             foundation.Opt[int]
	leadingPadding               foundation.Opt[float64]
	menuHasEnabledItem           bool
	localTextEditingController   *widgets.TextEditingController
	internalFocusNode            *widgets.FocusNode
	highlightedItemStatesControl *widgets.WidgetStatesController
	localTrailingIconFocusNode   *widgets.FocusNode
}

func (s *dropdownMenuFieldState[T]) effectiveTextEditingController() *widgets.TextEditingController {
	if c := s.Widget().o.Controller; c != nil {
		return c
	}
	if s.localTextEditingController == nil {
		s.localTextEditingController = widgets.NewTextEditingController()
	}
	return s.localTextEditingController
}

func (s *dropdownMenuFieldState[T]) trailingIconButtonFocusNode() *widgets.FocusNode {
	if f := s.Widget().o.TrailingIconFocusNode; f != nil {
		return f
	}
	if s.localTrailingIconFocusNode == nil {
		s.localTrailingIconFocusNode = widgets.NewFocusNode(widgets.FocusNodeOpts{})
	}
	return s.localTrailingIconFocusNode
}

func (s *dropdownMenuFieldState[T]) newButtonKeys(n int) []*widgets.GlobalKey[widgets.State] {
	keys := make([]*widgets.GlobalKey[widgets.State], n)
	for i := range keys {
		keys[i] = widgets.NewGlobalKey[widgets.State]()
	}
	return keys
}

func (s *dropdownMenuFieldState[T]) anyEnabled(entries []DropdownMenuEntry[T]) bool {
	for _, e := range entries {
		if !e.Disabled {
			return true
		}
	}
	return false
}

func (s *dropdownMenuFieldState[T]) setText(label string) {
	s.effectiveTextEditingController().SetValue(services.NewTextEditingValue(services.TextEditingValueOpts{
		Text: label, Selection: foundation.Some(services.NewTextSelectionCollapsed(len(label)))}))
}

func (s *dropdownMenuFieldState[T]) selectInitial() {
	w := s.Widget()
	init, ok := w.o.InitialSelection.Get()
	if !ok {
		return
	}
	for _, e := range s.filteredEntries {
		if e.Value == init {
			s.setText(e.Label)
			return
		}
	}
}

// InitState selects the initial entry and makes the controllers.
//
// Flutter: material/dropdown_menu.dart _DropdownMenuState.initState
func (s *dropdownMenuFieldState[T]) InitState() {
	w := s.Widget()
	s.anchorKey = widgets.NewGlobalKey[widgets.State]()
	s.leadingKey = widgets.NewGlobalKey[widgets.State]()
	s.internalFocusNode = widgets.NewFocusNode(widgets.FocusNodeOpts{})
	s.enableSearch = !w.o.NoEnableSearch
	s.filteredEntries = w.entries
	s.buttonItemKeys = s.newButtonKeys(len(s.filteredEntries))
	s.menuHasEnabledItem = s.anyEnabled(s.filteredEntries)
	s.selectInitial()
	s.refreshLeadingPadding()
	s.controller = w.o.MenuController
	if s.controller == nil {
		s.controller = widgets.NewMenuController()
	}
}

// Dispose releases the controllers and the focus nodes.
//
// Flutter: material/dropdown_menu.dart _DropdownMenuState.dispose
func (s *dropdownMenuFieldState[T]) Dispose() {
	if s.localTextEditingController != nil {
		s.localTextEditingController.Dispose()
		s.localTextEditingController = nil
	}
	s.internalFocusNode.Dispose()
	if s.localTrailingIconFocusNode != nil {
		s.localTrailingIconFocusNode.Dispose()
		s.localTrailingIconFocusNode = nil
	}
	if s.highlightedItemStatesControl != nil {
		s.highlightedItemStatesControl.Dispose()
	}
}

// DidUpdateWidget follows the changes of the entries, the selection and the
// controllers.
//
// Flutter: material/dropdown_menu.dart _DropdownMenuState.didUpdateWidget
func (s *dropdownMenuFieldState[T]) DidUpdateWidget(old *DropdownMenu[T]) {
	w := s.Widget()
	if old.o.Controller != w.o.Controller && s.localTextEditingController != nil {
		s.localTextEditingController.Dispose()
		s.localTextEditingController = nil
	}
	if old.o.EnableFilter != w.o.EnableFilter && !w.o.EnableFilter {
		s.enableFilter = false
	}
	if old.o.NoEnableSearch != w.o.NoEnableSearch && w.o.NoEnableSearch {
		s.enableSearch = false
		s.currentHighlight = foundation.None[int]()
	}
	if !sameDropdownEntries(old.entries, w.entries) {
		s.currentHighlight = foundation.None[int]()
		s.filteredEntries = w.entries
		s.buttonItemKeys = s.newButtonKeys(len(s.filteredEntries))
		s.menuHasEnabledItem = s.anyEnabled(s.filteredEntries)
	}
	if old.o.LeadingIcon != w.o.LeadingIcon {
		s.refreshLeadingPadding()
	}
	if old.o.InitialSelection != w.o.InitialSelection {
		s.selectInitial()
	}
	if old.o.MenuController != w.o.MenuController {
		s.controller = w.o.MenuController
		if s.controller == nil {
			s.controller = widgets.NewMenuController()
		}
	}
}

// sameDropdownEntries is whether the two lists are the same slice, which stands
// for Dart's identical on the two lists.
func sameDropdownEntries[T comparable](a, b []DropdownMenuEntry[T]) bool {
	return len(a) == len(b) && (len(a) == 0 || &a[0] == &b[0])
}

func (s *dropdownMenuFieldState[T]) canRequestFocus() bool {
	w := s.Widget()
	if f := w.o.FocusNode; f != nil {
		return f.CanRequestFocus()
	}
	if r, ok := w.o.RequestFocusOnTap.Get(); ok {
		return r
	}
	switch ThemeOf(s.Context()).Platform {
	case foundation.TargetPlatformIOS, foundation.TargetPlatformAndroid, foundation.TargetPlatformFuchsia:
		return false
	}
	return true
}

func (s *dropdownMenuFieldState[T]) selectOnly() bool { return s.Widget().o.SelectOnly }

func (s *dropdownMenuFieldState[T]) isButton() bool { return !s.canRequestFocus() || s.selectOnly() }

// refreshLeadingPadding measures the leading icon after the frame.
//
// Flutter: material/dropdown_menu.dart _DropdownMenuState.refreshLeadingPadding
func (s *dropdownMenuFieldState[T]) refreshLeadingPadding() {
	widgets.WidgetsBindingOf(s.Context()).RunPostFrame(func(time.Duration) {
		if !s.Mounted() {
			return
		}
		s.SetState(func() { s.leadingPadding = widthOf(s.leadingKey) })
	})
}

// scrollToHighlight scrolls the highlighted entry into view after the frame.
//
// Flutter: material/dropdown_menu.dart _DropdownMenuState.scrollToHighlight
func (s *dropdownMenuFieldState[T]) scrollToHighlight() {
	widgets.WidgetsBindingOf(s.Context()).RunPostFrame(func(time.Duration) {
		i, ok := s.currentHighlight.Get()
		if !ok || i >= len(s.buttonItemKeys) {
			return
		}
		if highlightContext := s.buttonItemKeys[i].CurrentContext(); highlightContext != nil {
			if scrollable := widgets.ScrollableMaybeOf(highlightContext); scrollable != nil {
				scrollable.Position().EnsureVisible(highlightContext.FindRenderObject())
			}
		}
	})
}

// widthOf is the width of the box of the key, if it has one.
//
// Flutter: material/dropdown_menu.dart _DropdownMenuState.getWidth
func widthOf(key *widgets.GlobalKey[widgets.State]) foundation.Opt[float64] {
	if c := key.CurrentContext(); c != nil {
		if box, ok := c.FindRenderObject().(rendering.Box); ok && box.HasSize() {
			return foundation.Some(box.Size().Width)
		}
	}
	return foundation.None[float64]()
}

// filter keeps the entries whose label contains the text of the field.
//
// Flutter: material/dropdown_menu.dart _DropdownMenuState.filter
func (s *dropdownMenuFieldState[T]) filter(entries []DropdownMenuEntry[T]) []DropdownMenuEntry[T] {
	filterText := strings.ToLower(s.effectiveTextEditingController().Text())
	var out []DropdownMenuEntry[T]
	for _, e := range entries {
		if strings.Contains(strings.ToLower(e.Label), filterText) {
			out = append(out, e)
		}
	}
	return out
}

// shouldUpdateCurrentHighlight is whether the highlight has to move to the entry
// that matches the text.
//
// Flutter: material/dropdown_menu.dart _DropdownMenuState._shouldUpdateCurrentHighlight
func (s *dropdownMenuFieldState[T]) shouldUpdateCurrentHighlight(entries []DropdownMenuEntry[T]) bool {
	searchText := strings.ToLower(s.effectiveTextEditingController().Value().Text)
	if searchText == "" {
		return true
	}
	h, ok := s.currentHighlight.Get()
	if !ok || h >= len(entries) {
		return true
	}
	return !strings.Contains(strings.ToLower(entries[h].Label), searchText)
}

// search finds the first entry whose label contains the text of the field.
//
// Flutter: material/dropdown_menu.dart _DropdownMenuState.search
func (s *dropdownMenuFieldState[T]) search(entries []DropdownMenuEntry[T]) foundation.Opt[int] {
	searchText := strings.ToLower(s.effectiveTextEditingController().Value().Text)
	if searchText == "" {
		return foundation.None[int]()
	}
	for i, e := range entries {
		if strings.Contains(strings.ToLower(e.Label), searchText) {
			return foundation.Some(i)
		}
	}
	return foundation.None[int]()
}

// buildButtons builds the button of each entry.
//
// Flutter: material/dropdown_menu.dart _DropdownMenuState._buildButtons
func (s *dropdownMenuFieldState[T]) buildButtons(bc widgets.BuildContext, entries []DropdownMenuEntry[T], focusedIndex foundation.Opt[int], enableScrollToHighlight, excludeSemantics, useMaterial3 bool) []widgets.Widget {
	w := s.Widget()
	effectiveInputStartGap := 0.0
	if useMaterial3 {
		effectiveInputStartGap = dropdownMenuInputStartGap
	}
	themeStyle := MenuButtonThemeOf(bc).Style
	focused := widgets.NewWidgetStates(widgets.WidgetStateFocused)
	pick := func(entry, theme widgets.WidgetStateProperty[foundation.Opt[ui.Color]]) widgets.WidgetStateProperty[foundation.Opt[ui.Color]] {
		if entry != nil {
			return entry
		}
		return theme
	}
	result := make([]widgets.Widget, 0, len(entries))
	for i, entry := range entries {
		padding := dropdownMenuDefaultHorizontalPadding
		if entry.LeadingIcon == nil {
			padding = s.leadingPadding.OrElse(dropdownMenuDefaultHorizontalPadding)
		}
		effectiveStyle := entry.Style
		if effectiveStyle == nil {
			effectiveStyle = MenuItemButtonStyleFrom(ButtonStyleFromOpts{
				Padding: painting.EdgeInsetsDirectional{Start: padding, End: dropdownMenuDefaultHorizontalPadding}})
		}
		var entryStyle, theme ButtonStyle
		if entry.Style != nil {
			entryStyle = *entry.Style
		}
		if themeStyle != nil {
			theme = *themeStyle
		}
		effectiveForegroundColor := pick(entryStyle.ForegroundColor, theme.ForegroundColor)
		effectiveIconColor := pick(entryStyle.IconColor, theme.IconColor)
		effectiveOverlayColor := pick(entryStyle.OverlayColor, theme.OverlayColor)
		effectiveBackgroundColor := pick(entryStyle.BackgroundColor, theme.BackgroundColor)
		entryIsSelected := !entry.Disabled && focusedIndex == foundation.Some(i)
		if entryIsSelected {
			if s.highlightedItemStatesControl != nil {
				s.highlightedItemStatesControl.Dispose()
			}
			s.highlightedItemStatesControl = widgets.NewWidgetStatesController(widgets.WidgetStateFocused)
			defaultStyle := NewMenuItemButton().DefaultStyleOf(bc)
			resolveFocusedColor := func(p widgets.WidgetStateProperty[foundation.Opt[ui.Color]]) foundation.Opt[ui.Color] {
				if p == nil {
					return foundation.None[ui.Color]()
				}
				return p.Resolve(focused)
			}
			orDefault := func(p, def widgets.WidgetStateProperty[foundation.Opt[ui.Color]]) widgets.WidgetStateProperty[foundation.Opt[ui.Color]] {
				if p != nil {
					return p
				}
				return def
			}
			focusedForegroundColor := resolveFocusedColor(orDefault(effectiveForegroundColor, defaultStyle.ForegroundColor)).Value()
			focusedIconColor := resolveFocusedColor(orDefault(effectiveIconColor, defaultStyle.IconColor)).Value()
			focusedOverlayColor := resolveFocusedColor(orDefault(effectiveOverlayColor, defaultStyle.OverlayColor)).Value()
			focusedBackgroundColor := resolveFocusedColor(effectiveBackgroundColor).OrElse(ThemeOf(bc).ColorScheme.OnSurface().WithOpacity(0.12))
			effectiveStyle = effectiveStyle.CopyWith(ButtonStyleOpts{
				BackgroundColor: allOpt(focusedBackgroundColor),
				ForegroundColor: allOpt(focusedForegroundColor),
				IconColor:       allOpt(focusedIconColor),
				OverlayColor:    allOpt(focusedOverlayColor),
			})
		} else {
			effectiveStyle = effectiveStyle.CopyWith(ButtonStyleOpts{
				BackgroundColor: effectiveBackgroundColor,
				ForegroundColor: effectiveForegroundColor,
				IconColor:       effectiveIconColor,
				OverlayColor:    effectiveOverlayColor,
			})
		}
		label := entry.LabelWidget
		if label == nil {
			label = widgets.NewText(entry.Label)
		}
		if width, ok := w.o.Width.Get(); ok {
			horizontalPadding := padding + dropdownMenuDefaultHorizontalPadding + effectiveInputStartGap
			label = widgets.NewConstrainedBox(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MaxWidth: foundation.Some(width - horizontalPadding)}), label)
		}
		var key foundation.Key
		if enableScrollToHighlight {
			key = s.buttonItemKeys[i]
		}
		var statesController *widgets.WidgetStatesController
		if entryIsSelected {
			statesController = s.highlightedItemStatesControl
		}
		var onPressed func()
		if !entry.Disabled && w.Enabled() {
			entry, i := entry, i
			onPressed = func() { s.selectEntry(entry, i) }
		}
		result = append(result, widgets.NewExcludeFocus(
			widgets.NewExcludeSemantics(
				NewMenuItemButton(MenuItemButtonOpts{
					Key:                   key,
					StatesController:      statesController,
					Style:                 effectiveStyle,
					LeadingIcon:           entry.LeadingIcon,
					TrailingIcon:          entry.TrailingIcon,
					NoCloseOnActivate:     w.o.CloseBehavior != DropdownMenuCloseBehaviorAll,
					OnPressed:             onPressed,
					NoRequestFocusOnHover: true,
					Child:                 widgets.NewPadding(painting.EdgeInsetsDirectional{Start: effectiveInputStartGap}, label),
				}),
				widgets.ExcludeSemanticsOpts{Excluding: foundation.Some(excludeSemantics)})))
	}
	return result
}

// selectEntry is the action of the button of an entry.
func (s *dropdownMenuFieldState[T]) selectEntry(entry DropdownMenuEntry[T], i int) {
	w := s.Widget()
	if !s.Mounted() {
		if w.o.Controller != nil {
			w.o.Controller.SetValue(services.NewTextEditingValue(services.TextEditingValueOpts{
				Text: entry.Label, Selection: foundation.Some(services.NewTextSelectionCollapsed(len(entry.Label)))}))
		}
		if w.o.OnSelected != nil {
			w.o.OnSelected(foundation.Some(entry.Value))
		}
		return
	}
	s.setText(entry.Label)
	s.currentHighlight = foundation.None[int]()
	if !w.o.NoEnableSearch {
		s.currentHighlight = foundation.Some(i)
	}
	if w.o.OnSelected != nil {
		w.o.OnSelected(foundation.Some(entry.Value))
	}
	s.enableFilter = false
	if w.o.CloseBehavior == DropdownMenuCloseBehaviorSelf {
		s.controller.Close()
	}
}

// moveHighlight moves the highlight by one entry, skipping the disabled ones.
func (s *dropdownMenuFieldState[T]) moveHighlight(step int) {
	s.SetState(func() {
		w := s.Widget()
		if !w.Enabled() || !s.menuHasEnabledItem || !s.controller.IsOpen() {
			return
		}
		s.enableFilter = false
		s.enableSearch = false
		n := len(s.filteredEntries)
		wrap := func(i int) int { return ((i % n) + n) % n }
		h := s.currentHighlight.OrElse(map[int]int{-1: 0, 1: -1}[step])
		h = wrap(h + step)
		for s.filteredEntries[h].Disabled {
			h = wrap(h + step)
		}
		s.currentHighlight = foundation.Some(h)
		s.setText(s.filteredEntries[h].Label)
	})
}

func (s *dropdownMenuFieldState[T]) handleEnterKey() {
	if s.selectOnly() && !s.controller.IsOpen() {
		s.controller.Open()
		return
	}
	s.handleSubmitted()
}

// handlePressed opens the menu, or closes it.
//
// Flutter: material/dropdown_menu.dart _DropdownMenuState.handlePressed
func (s *dropdownMenuFieldState[T]) handlePressed(controller *widgets.MenuController, focusForKeyboard bool) {
	if controller.IsOpen() {
		s.currentHighlight = foundation.None[int]()
		controller.Close()
	} else {
		s.filteredEntries = s.Widget().entries
		if s.effectiveTextEditingController().Text() != "" {
			s.enableFilter = false
		}
		controller.Open()
		if focusForKeyboard {
			s.internalFocusNode.RequestFocus()
		}
	}
	s.SetState(func() {})
}

// handleSubmitted selects the highlighted entry and closes the menu.
//
// Flutter: material/dropdown_menu.dart _DropdownMenuState._handleSubmitted
func (s *dropdownMenuFieldState[T]) handleSubmitted() {
	w := s.Widget()
	if h, ok := s.currentHighlight.Get(); ok {
		entry := s.filteredEntries[h]
		if !entry.Disabled {
			s.setText(entry.Label)
			if w.o.OnSelected != nil {
				w.o.OnSelected(foundation.Some(entry.Value))
			}
		}
	} else if s.controller.IsOpen() && w.o.OnSelected != nil {
		w.o.OnSelected(foundation.None[T]())
	}
	if w.o.NoEnableSearch {
		s.currentHighlight = foundation.None[int]()
	}
	s.controller.Close()
}

// Build builds the menu anchor with the field.
//
// Flutter: material/dropdown_menu.dart _DropdownMenuState.build
func (s *dropdownMenuFieldState[T]) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	useMaterial3 := ThemeOf(bc).UseMaterial3
	if s.initialMenu == nil {
		s.initialMenu = s.buildButtons(bc, w.entries, foundation.None[int](), false, true, useMaterial3)
	}
	theme := DropdownMenuThemeOf(bc)
	defaults := dropdownMenuDefaultsM3(bc)
	if s.enableFilter {
		if w.o.FilterCallback != nil {
			s.filteredEntries = w.o.FilterCallback(s.filteredEntries, s.effectiveTextEditingController().Text())
		} else {
			s.filteredEntries = s.filter(w.entries)
		}
	}
	s.menuHasEnabledItem = s.anyEnabled(s.filteredEntries)
	if s.enableSearch {
		if w.o.SearchCallback != nil {
			s.currentHighlight = w.o.SearchCallback(s.filteredEntries, s.effectiveTextEditingController().Text())
		} else if s.shouldUpdateCurrentHighlight(s.filteredEntries) {
			s.currentHighlight = s.search(s.filteredEntries)
		}
		if s.currentHighlight.IsSome() {
			s.scrollToHighlight()
		}
	}
	menu := s.buildButtons(bc, s.filteredEntries, s.currentHighlight, true, false, useMaterial3)
	baseTextStyle := w.o.TextStyle
	if baseTextStyle == nil {
		baseTextStyle = theme.TextStyle
	}
	if baseTextStyle == nil {
		baseTextStyle = defaults.TextStyle
	}
	disabledColor := theme.DisabledColor.OrElse(defaults.DisabledColor.Value())
	effectiveTextStyle := baseTextStyle
	if !w.Enabled() {
		if baseTextStyle != nil {
			effectiveTextStyle = baseTextStyle.CopyWith(painting.TextStyleCopy{Color: foundation.Some(disabledColor)})
		} else {
			effectiveTextStyle = painting.NewTextStyle(painting.TextStyleOpts{Color: foundation.Some(disabledColor)})
		}
	}
	effectiveMenuStyle := w.o.MenuStyle
	if effectiveMenuStyle == nil {
		effectiveMenuStyle = theme.MenuStyle
	}
	if effectiveMenuStyle == nil {
		effectiveMenuStyle = defaults.MenuStyle
	}
	minimumWidth := func(width float64) widgets.WidgetStateProperty[foundation.Opt[ui.Size]] {
		style := effectiveMenuStyle
		return widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) foundation.Opt[ui.Size] {
			maximumWidth := width
			if style.MaximumSize != nil {
				if size, ok := style.MaximumSize.Resolve(states).Get(); ok {
					maximumWidth = size.Width
				}
			}
			return foundation.Some(ui.Size{Width: math.Min(width, maximumWidth)})
		})
	}
	anchorWidth := widthOf(s.anchorKey)
	if width, ok := w.o.Width.Get(); ok {
		effectiveMenuStyle = effectiveMenuStyle.CopyWith(MenuStyleOpts{MinimumSize: minimumWidth(width)})
	} else if aw, ok := anchorWidth.Get(); ok {
		effectiveMenuStyle = effectiveMenuStyle.CopyWith(MenuStyleOpts{MinimumSize: minimumWidth(aw)})
	}
	if h, ok := w.o.MenuHeight.Get(); ok {
		effectiveMenuStyle = effectiveMenuStyle.CopyWith(MenuStyleOpts{
			MaximumSize: widgets.NewWidgetStatePropertyAll(foundation.Some(ui.Size{Width: math.Inf(1), Height: h}))})
	}
	effectiveInputDecorationTheme := w.o.InputDecorationTheme
	if effectiveInputDecorationTheme == nil {
		effectiveInputDecorationTheme = theme.InputDecorationTheme
	}
	if effectiveInputDecorationTheme == nil {
		effectiveInputDecorationTheme = defaults.InputDecorationTheme
	}
	var effectiveMouseCursor services.MouseCursor
	if w.Enabled() {
		effectiveMouseCursor = services.SystemMouseCursors.Text
		if s.isButton() {
			effectiveMouseCursor = services.SystemMouseCursors.Click
		}
	}
	var menuAnchor widgets.Widget = NewMenuAnchor(menu, MenuAnchorOpts{
		Style:                    effectiveMenuStyle,
		AlignmentOffset:          w.o.AlignmentOffset,
		ReservedPadding:          painting.EdgeInsetsZero,
		Controller:               s.controller,
		NoCrossAxisUnconstrained: true,
		Builder: func(bc widgets.BuildContext, controller *widgets.MenuController, _ widgets.Widget) widgets.Widget {
			return s.buildAnchor(bc, controller, effectiveTextStyle, effectiveMouseCursor, *effectiveInputDecorationTheme)
		},
	})
	if padding := w.o.ExpandedInsets; padding != nil {
		menuAnchor = widgets.NewPadding(padding.Clamp(painting.EdgeInsetsZero,
			painting.EdgeInsets{Left: math.Inf(1), Right: math.Inf(1)}.Add(painting.EdgeInsetsDirectional{Start: math.Inf(1), End: math.Inf(1)})), menuAnchor)
	}
	menuAnchor = widgets.NewAlign(menuAnchor, widgets.AlignOpts{
		Alignment:    painting.AlignmentDirectionalTopStart,
		WidthFactor:  foundation.Some(1.0),
		HeightFactor: foundation.Some(1.0),
	})
	up := widgets.NewCallbackAction(func(dropdownArrowUpIntent) any { s.moveHighlight(-1); return nil })
	down := widgets.NewCallbackAction(func(dropdownArrowDownIntent) any { s.moveHighlight(1); return nil })
	enter := widgets.NewCallbackAction(func(dropdownEnterIntent) any { s.handleEnterKey(); return nil })
	dismiss := widgets.NewDismissMenuAction(s.controller)
	return widgets.NewActions(widgets.ActionMap{
		reflect.TypeFor[dropdownArrowUpIntent]():   up,
		reflect.TypeFor[dropdownArrowDownIntent](): down,
		reflect.TypeFor[dropdownEnterIntent]():     enter,
		dismiss.IntentType():                       dismiss,
	}, widgets.NewStack([]widgets.Widget{
		widgets.NewShortcuts(widgets.ShortcutMap{
			{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyArrowUp), Intent: dropdownArrowUpIntent{}},
			{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyArrowDown), Intent: dropdownArrowDownIntent{}},
			{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyEnter), Intent: dropdownEnterIntent{}},
			{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyEscape), Intent: widgets.DismissIntent{}},
		}, widgets.NewFocus(widgets.NewSizedBoxShrink(nil), widgets.FocusOpts{
			FocusNode: s.internalFocusNode, SkipTraversal: foundation.Some(true)})),
		menuAnchor,
	}))
}

// buildAnchor builds the field, the entries that size the menu, and the keys.
//
// Flutter: material/dropdown_menu.dart _DropdownMenuState.build
func (s *dropdownMenuFieldState[T]) buildAnchor(bc widgets.BuildContext, controller *widgets.MenuController, effectiveTextStyle *painting.TextStyle, effectiveMouseCursor services.MouseCursor, effectiveInputDecorationTheme InputDecorationThemeData) widgets.Widget {
	w := s.Widget()
	decorationBuilder := w.o.DecorationBuilder
	if decorationBuilder == nil {
		decorationBuilder = s.buildDefaultDecoration
	}
	decoration := decorationBuilder(bc, controller)
	if decoration.SuffixIcon == nil {
		decoration = decoration.CopyWith(InputDecorationCopy{SuffixIcon: foundation.Some(s.buildDefaultSuffixIcon(bc, controller))})
	}
	effectiveDecoration := decoration.ApplyDefaults(effectiveInputDecorationTheme)
	textFieldDecoration := effectiveDecoration
	if effectiveDecoration.PrefixIcon != nil {
		textFieldDecoration = effectiveDecoration.CopyWith(InputDecorationCopy{
			// The key is used to query the width in refreshLeadingPadding.
			PrefixIcon: foundation.Some[widgets.Widget](widgets.NewSizedBox(effectiveDecoration.PrefixIcon, widgets.SizedBoxOpts{Key: s.leadingKey})),
		})
	}
	localizations := MaterialLocalizationsOf(bc)
	isOpen := s.controller.IsOpen()
	var hint foundation.Opt[string]
	if ThemeOf(bc).Platform == foundation.TargetPlatformIOS {
		hint = foundation.Some(localizations.CollapsedHint())
		if isOpen {
			hint = foundation.Some(localizations.ExpandedHint())
		}
	}
	var onExpand, onCollapse func()
	if !isOpen {
		onExpand = func() { s.controller.Open() }
	} else {
		onCollapse = func() { s.controller.Close() }
	}
	var onTap func()
	if w.Enabled() {
		onTap = func() { s.handlePressed(controller, !s.canRequestFocus()) }
	}
	textField := widgets.Widget(widgets.NewSemantics(
		NewTextField(TextFieldOpts{
			Key:                        s.anchorKey,
			Enabled:                    foundation.Some(w.Enabled()),
			MouseCursor:                effectiveMouseCursor,
			FocusNode:                  w.o.FocusNode,
			CannotRequestFocus:         !s.canRequestFocus(),
			EnableInteractiveSelection: foundation.Some(!s.isButton()),
			ReadOnly:                   s.isButton(),
			KeyboardType:               w.o.KeyboardType,
			TextAlign:                  w.o.TextAlign,
			TextAlignVertical:          foundation.Some(painting.TextAlignVerticalCenter),
			MaxLines:                   w.o.MaxLines,
			TextInputAction:            w.o.TextInputAction,
			CursorHeight:               w.o.CursorHeight,
			Style:                      effectiveTextStyle,
			Controller:                 s.effectiveTextEditingController(),
			OnSubmitted:                func(string) { s.handleSubmitted() },
			OnTap:                      onTap,
			OnChanged: func(string) {
				controller.Open()
				s.SetState(func() {
					s.filteredEntries = w.entries
					s.enableFilter = w.o.EnableFilter
					s.enableSearch = !w.o.NoEnableSearch
				})
			},
			InputFormatters: w.o.InputFormatters,
			Decoration:      &textFieldDecoration,
			ScrollPadding:   w.o.ScrollPadding,
		}),
		widgets.SemanticsOpts{
			Button:     foundation.Some(s.isButton()),
			Hint:       hint,
			Expanded:   foundation.Some(isOpen),
			OnExpand:   onExpand,
			OnCollapse: onCollapse,
		}))
	var effectiveLabel widgets.Widget
	switch {
	case effectiveDecoration.Label != nil:
		effectiveLabel = effectiveDecoration.Label
	case effectiveDecoration.LabelText.IsSome():
		effectiveLabel = widgets.NewText(effectiveDecoration.LabelText.Value())
	}
	var body widgets.Widget
	if w.o.ExpandedInsets != nil {
		body = textField
	} else {
		children := []widgets.Widget{textField}
		children = append(children, s.initialMenu...)
		if effectiveLabel != nil {
			children = append(children, widgets.NewExcludeSemantics(
				widgets.NewPadding(painting.NewEdgeInsetsSymmetric(0, 4.0),
					widgets.NewDefaultTextStyle(effectiveTextStyle, effectiveLabel))))
		}
		suffix := effectiveDecoration.SuffixIcon
		if suffix == nil {
			suffix = widgets.NewSizedBoxShrink(nil)
		}
		prefix := effectiveDecoration.PrefixIcon
		if prefix == nil {
			prefix = widgets.NewSizedBoxShrink(nil)
		}
		children = append(children, suffix, widgets.NewPadding(painting.NewEdgeInsetsAll(8.0), prefix))
		body = newDropdownMenuBody(w.o.Width, children)
	}
	shortcuts := dropdownEditableShortcuts()
	if s.selectOnly() {
		shortcuts = dropdownSelectOnlyShortcuts()
	}
	return widgets.NewShortcuts(shortcuts, body)
}

// buildDefaultDecoration is the decoration made from the parameters of the menu.
//
// Flutter: material/dropdown_menu.dart _DropdownMenuState._buildDefaultDecoration
func (s *dropdownMenuFieldState[T]) buildDefaultDecoration(bc widgets.BuildContext, controller *widgets.MenuController) InputDecoration {
	w := s.Widget()
	return InputDecoration{
		Label:      w.o.Label,
		HintText:   w.o.HintText,
		HelperText: w.o.HelperText,
		ErrorText:  w.o.ErrorText,
		PrefixIcon: w.o.LeadingIcon,
		SuffixIcon: s.buildDefaultSuffixIcon(bc, controller),
	}
}

// buildDefaultSuffixIcon is the button that opens and closes the menu, or nil.
//
// Flutter: material/dropdown_menu.dart _DropdownMenuState._buildDefaultSuffixIcon
func (s *dropdownMenuFieldState[T]) buildDefaultSuffixIcon(_ widgets.BuildContext, controller *widgets.MenuController) widgets.Widget {
	w := s.Widget()
	if w.o.NoShowTrailingIcon {
		return nil
	}
	var inputTheme *InputDecorationThemeData
	if w.o.InputDecorationTheme != nil {
		inputTheme = w.o.InputDecorationTheme
	}
	isCollapsed := inputTheme != nil && inputTheme.IsCollapsed
	icon := w.o.TrailingIcon
	if icon == nil {
		icon = widgets.NewIcon(Icons.ArrowDropDown())
	}
	selectedIcon := w.o.SelectedTrailingIcon
	if selectedIcon == nil {
		selectedIcon = widgets.NewIcon(Icons.ArrowDropUp())
	}
	var constraints foundation.Opt[rendering.BoxConstraints]
	if inputTheme != nil {
		constraints = inputTheme.SuffixIconConstraints
	}
	var buttonPadding painting.EdgeInsetsGeometry
	outerPadding := painting.EdgeInsetsGeometry(painting.NewEdgeInsetsAll(4.0))
	if isCollapsed {
		buttonPadding = painting.EdgeInsetsZero
		outerPadding = painting.EdgeInsetsZero
	}
	var onPressed func()
	if w.Enabled() {
		onPressed = func() { s.handlePressed(controller, true) }
	}
	return widgets.NewPadding(outerPadding,
		widgets.NewExcludeSemantics(
			NewIconButton(onPressed, icon, IconButtonOpts{
				FocusNode:    s.trailingIconButtonFocusNode(),
				IsSelected:   foundation.Some(controller.IsOpen()),
				Constraints:  constraints,
				Padding:      buttonPadding,
				SelectedIcon: selectedIcon,
			}),
			widgets.ExcludeSemanticsOpts{Excluding: foundation.Some(s.isButton())}))
}

// dropdownMenuDefaultsM3 are the defaults of the menu with Material 3.
//
// Flutter: material/dropdown_menu.dart _DropdownMenuDefaultsM3
func dropdownMenuDefaultsM3(bc widgets.BuildContext) DropdownMenuThemeData {
	theme := ThemeOf(bc)
	input := InputDecorationThemeData{Border: NewOutlineInputBorder()}
	return DropdownMenuThemeData{
		DisabledColor: foundation.Some(theme.ColorScheme.OnSurface().WithOpacity(0.38)),
		TextStyle:     theme.TextTheme.BodyLarge,
		MenuStyle: NewMenuStyle(MenuStyleOpts{
			MinimumSize:   widgets.NewWidgetStatePropertyAll(foundation.Some(ui.Size{Width: dropdownMenuMinimumWidth})),
			MaximumSize:   widgets.NewWidgetStatePropertyAll(foundation.Some(ui.SizeInfinite)),
			VisualDensity: foundation.Some(VisualDensityStandard()),
		}),
		InputDecorationTheme: &input,
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagDropdownMenuDecorationBuilderWithTexts       = "DropdownMenuDecorationBuilderWithTexts"
	tagDropdownMenuFilterCallbackWithoutFilter      = "DropdownMenuFilterCallbackWithoutFilter"
	tagDropdownMenuTrailingIconFocusNodeWithoutIcon = "DropdownMenuTrailingIconFocusNodeWithoutIcon"
)
