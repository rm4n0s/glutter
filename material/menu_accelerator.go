// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"strings"
	"unicode"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// isCupertino is whether the target platform is one of Apple's.
//
// Flutter: material/menu_anchor.dart _isCupertino
func isCupertino() bool {
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS:
		return true
	}
	return false
}

// usesSymbolicModifiers is whether the labels of the shortcuts have the
// modifiers as symbols ("⌃ ⇧ A") instead of names ("Ctrl+Shift+A").
//
// Flutter: material/menu_anchor.dart _usesSymbolicModifiers
func usesSymbolicModifiers() bool { return isCupertino() }

// platformSupportsAccelerators is whether the platform has menu accelerators. On
// iOS and macOS, pressing the Option key (the Alt key) causes a different set of
// characters to be generated, and the native menus don't support accelerators
// anyhow, so accelerators are disabled there.
//
// Flutter: material/menu_anchor.dart _platformSupportsAccelerators
func platformSupportsAccelerators() bool { return !isCupertino() }

// MenuAcceleratorCallbackBindingOpts are the optional parameters of
// NewMenuAcceleratorCallbackBinding.
//
// Flutter: material/menu_anchor.dart MenuAcceleratorCallbackBinding.new
type MenuAcceleratorCallbackBindingOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// OnInvoke is the function that is called when the accelerator of a
	// descendant label is pressed; nil makes the item have no accelerator.
	OnInvoke func()
	// HasSubmenu says that the item opens a submenu: its accelerator is only
	// active while that submenu is closed.
	HasSubmenu bool
}

// MenuAcceleratorCallbackBinding is an inherited widget that gives the
// MenuAcceleratorLabels below it the function to call when their accelerator is
// pressed.
//
// Flutter: material/menu_anchor.dart MenuAcceleratorCallbackBinding
type MenuAcceleratorCallbackBinding struct {
	widgets.InheritedWidget
	o MenuAcceleratorCallbackBindingOpts
}

// NewMenuAcceleratorCallbackBinding creates the widget.
//
// Flutter: material/menu_anchor.dart MenuAcceleratorCallbackBinding.new
func NewMenuAcceleratorCallbackBinding(child widgets.Widget, opts ...MenuAcceleratorCallbackBindingOpts) *MenuAcceleratorCallbackBinding {
	o := firstOpts(opts)
	return &MenuAcceleratorCallbackBinding{InheritedWidget: widgets.NewInheritedWidgetBase(child, widgets.WidgetOpts{Key: o.Key}), o: o}
}

// OnInvoke is the function that is called when the accelerator is pressed.
//
// Flutter: material/menu_anchor.dart MenuAcceleratorCallbackBinding.onInvoke
func (w *MenuAcceleratorCallbackBinding) OnInvoke() func() { return w.o.OnInvoke }

// HasSubmenu is whether the item opens a submenu.
//
// Flutter: material/menu_anchor.dart MenuAcceleratorCallbackBinding.hasSubmenu
func (w *MenuAcceleratorCallbackBinding) HasSubmenu() bool { return w.o.HasSubmenu }

// UpdateShouldNotify is whether the callback or the submenu flag changed.
//
// Flutter: material/menu_anchor.dart MenuAcceleratorCallbackBinding.updateShouldNotify
func (w *MenuAcceleratorCallbackBinding) UpdateShouldNotify(old widgets.Widget) bool {
	o := old.(*MenuAcceleratorCallbackBinding)
	return (w.o.OnInvoke == nil) != (o.o.OnInvoke == nil) || w.o.HasSubmenu != o.o.HasSubmenu
}

// MenuAcceleratorCallbackBindingMaybeOf is the closest binding above the
// context, or nil. The context depends on it.
//
// Flutter: material/menu_anchor.dart MenuAcceleratorCallbackBinding.maybeOf
func MenuAcceleratorCallbackBindingMaybeOf(bc widgets.BuildContext) *MenuAcceleratorCallbackBinding {
	b, _ := widgets.DependOn[*MenuAcceleratorCallbackBinding](bc)
	return b
}

// MenuAcceleratorCallbackBindingOf is the closest binding above the context. In
// debug builds it panics with the tag MenuAcceleratorCallbackBindingMissing when
// there is none.
//
// Flutter: material/menu_anchor.dart MenuAcceleratorCallbackBinding.of
func MenuAcceleratorCallbackBindingOf(bc widgets.BuildContext) *MenuAcceleratorCallbackBinding {
	b := MenuAcceleratorCallbackBindingMaybeOf(bc)
	if foundation.DebugMode && b == nil {
		panic(errors.New("MenuAcceleratorCallbackBindingMissing",
			"MenuAcceleratorCallbackBindingOf was called with a context that does not contain a MenuAcceleratorCallbackBinding; this can happen because a widget that looks for one is used without one above it"))
	}
	return b
}

// MenuAcceleratorChildBuilder is the type of the function that builds the label
// of a MenuAcceleratorLabel from the text without the accelerator markers. The
// index is the grapheme of the label that is the accelerator, or -1 when the
// accelerator is not shown.
//
// Flutter: material/menu_anchor.dart MenuAcceleratorChildBuilder
type MenuAcceleratorChildBuilder func(bc widgets.BuildContext, label string, index int) widgets.Widget

// MenuAcceleratorLabelOpts are the optional parameters of
// NewMenuAcceleratorLabel.
//
// Flutter: material/menu_anchor.dart MenuAcceleratorLabel.new
type MenuAcceleratorLabelOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Builder builds the label; the default is MenuAcceleratorLabelDefaultLabelBuilder,
	// which underlines the accelerator.
	Builder MenuAcceleratorChildBuilder
}

// MenuAcceleratorLabel is a widget that draws the label text of a menu item
// with an accelerator character that can be pressed with Alt to activate the
// item. An ampersand marks the accelerator: "&Open" has the accelerator O, and
// "&&" is an ampersand. The accelerator is shown while Alt is held.
//
// Flutter: material/menu_anchor.dart MenuAcceleratorLabel
type MenuAcceleratorLabel struct {
	widgets.StatefulWidget
	label   string
	builder MenuAcceleratorChildBuilder
}

// NewMenuAcceleratorLabel creates the widget.
//
// Flutter: material/menu_anchor.dart MenuAcceleratorLabel.new
func NewMenuAcceleratorLabel(label string, opts ...MenuAcceleratorLabelOpts) *MenuAcceleratorLabel {
	o := firstOpts(opts)
	if o.Builder == nil {
		o.Builder = MenuAcceleratorLabelDefaultLabelBuilder
	}
	return &MenuAcceleratorLabel{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), label: label, builder: o.Builder}
}

// Label is the label with the accelerator markers.
//
// Flutter: material/menu_anchor.dart MenuAcceleratorLabel.label
func (w *MenuAcceleratorLabel) Label() string { return w.label }

// DisplayLabel is the label without the accelerator markers.
//
// Flutter: material/menu_anchor.dart MenuAcceleratorLabel.displayLabel
func (w *MenuAcceleratorLabel) DisplayLabel() string {
	return MenuAcceleratorLabelStripAcceleratorMarkers(w.label, nil)
}

// HasAccelerator is whether the label has an accelerator marker: an ampersand
// that is followed by a character that is not an ampersand or a space.
//
// Flutter: material/menu_anchor.dart MenuAcceleratorLabel.hasAccelerator
func (w *MenuAcceleratorLabel) HasAccelerator() bool {
	runes := []rune(w.label)
	for i, r := range runes {
		if r != '&' || i+1 >= len(runes) {
			continue
		}
		if n := runes[i+1]; n != '&' && !unicode.IsSpace(n) {
			return true
		}
	}
	return false
}

// MenuAcceleratorLabelDefaultLabelBuilder builds the label as a Text, or as a
// RichText with the accelerator underlined.
//
// Flutter: material/menu_anchor.dart MenuAcceleratorLabel.defaultLabelBuilder
func MenuAcceleratorLabelDefaultLabelBuilder(bc widgets.BuildContext, label string, index int) widgets.Widget {
	if index < 0 {
		return widgets.NewText(label)
	}
	defaultStyle := widgets.DefaultTextStyleOf(bc).Style()
	bounds := foundation.GraphemeBoundaries(label)
	count := len(bounds) - 1
	var children []painting.InlineSpan
	if index > 0 {
		children = append(children, painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some(label[:bounds[index]]), Style: defaultStyle}))
	}
	children = append(children, painting.NewTextSpan(painting.TextSpanOpts{
		Text:  foundation.Some(label[bounds[index]:bounds[index+1]]),
		Style: defaultStyle.CopyWith(painting.TextStyleCopy{Decoration: foundation.Some(ui.TextDecorationUnderline)}),
	}))
	if index < count-1 {
		children = append(children, painting.NewTextSpan(painting.TextSpanOpts{Text: foundation.Some(label[bounds[index+1]:]), Style: defaultStyle}))
	}
	return widgets.NewRichText(painting.NewTextSpan(painting.TextSpanOpts{Children: children}))
}

// MenuAcceleratorLabelStripAcceleratorMarkers is the label without its
// accelerator markers. A single ampersand marks the character after it as the
// accelerator; two ampersands are one ampersand of the text, and a single one
// at the end is dropped. setIndex, if not nil, is called with the index (in
// graphemes of the returned text) of the first accelerator, or -1.
//
// Flutter: material/menu_anchor.dart MenuAcceleratorLabel.stripAcceleratorMarkers
func MenuAcceleratorLabelStripAcceleratorMarkers(label string, setIndex func(index int)) string {
	quotedAmpersands := 0
	var display strings.Builder
	acceleratorIndex := -1
	// Use graphemes so that the characters that are made of several code points
	// are not split up and interpreted incorrectly.
	bounds := foundation.GraphemeBoundaries(label)
	count := len(bounds) - 1
	char := func(i int) string { return label[bounds[i]:bounds[i+1]] }
	lastWasAmpersand := false
	for i := 0; i < count; i++ {
		// Stop looking one before the end, since a single ampersand at the end is
		// just treated as a quoted ampersand.
		c := char(i)
		if lastWasAmpersand {
			lastWasAmpersand = false
			display.WriteString(c)
			continue
		}
		if c != "&" {
			display.WriteString(c)
			continue
		}
		if i == count-1 {
			// Strip bare ampersands at the end of a string.
			break
		}
		lastWasAmpersand = true
		accelerator := char(i + 1)
		if acceleratorIndex == -1 && accelerator != "&" && strings.TrimSpace(accelerator) != "" {
			// Don't set the accelerator index if the character is an ampersand,
			// or whitespace.
			acceleratorIndex = i - quotedAmpersands
		}
		// As '&<character>' pairs are encountered, the following indices must be
		// adjusted so that they correspond with indices in the stripped string.
		quotedAmpersands++
	}
	if setIndex != nil {
		setIndex(acceleratorIndex)
	}
	return display.String()
}

// CreateState creates the state.
//
// Flutter: material/menu_anchor.dart MenuAcceleratorLabel.createState
func (w *MenuAcceleratorLabel) CreateState() widgets.State { return &menuAcceleratorLabelState{} }

// ToString describes the label.
//
// Flutter: material/menu_anchor.dart MenuAcceleratorLabel.toString
func (w *MenuAcceleratorLabel) String() string { return `MenuAcceleratorLabel("` + w.label + `")` }

// DebugFillProperties adds the label.
//
// Flutter: material/menu_anchor.dart MenuAcceleratorLabel.debugFillProperties
func (w *MenuAcceleratorLabel) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewStringProperty("label", foundation.Some(w.label)))
}

type menuAcceleratorLabelState struct {
	widgets.StateOf[*MenuAcceleratorLabel]
	displayLabel         string
	acceleratorIndex     int
	binding              *MenuAcceleratorCallbackBinding
	menuController       *widgets.MenuController
	shortcutRegistry     *widgets.ShortcutRegistry
	shortcutRegistryItem *widgets.ShortcutRegistryEntry
	showAccelerators     bool
	keyHandle            services.KeyHandlerHandle
}

func (s *menuAcceleratorLabelState) InitState() {
	s.acceleratorIndex = -1
	if platformSupportsAccelerators() {
		s.showAccelerators = s.altIsPressed()
		s.keyHandle = widgets.WidgetsBindingOf(s.Context()).Keyboard().AddHandler(s.listenToKeyEvent)
	}
	s.updateDisplayLabel()
}

func (s *menuAcceleratorLabelState) Dispose() {
	s.displayLabel = ""
	if platformSupportsAccelerators() {
		if s.shortcutRegistryItem != nil {
			s.shortcutRegistryItem.Dispose()
		}
		s.shortcutRegistryItem = nil
		s.shortcutRegistry = nil
		s.menuController = nil
		widgets.WidgetsBindingOf(s.Context()).Keyboard().RemoveHandler(s.keyHandle)
	}
}

func (s *menuAcceleratorLabelState) DidChangeDependencies() {
	if !platformSupportsAccelerators() {
		return
	}
	s.binding = MenuAcceleratorCallbackBindingMaybeOf(s.Context())
	s.menuController = widgets.MenuControllerMaybeOf(s.Context())
	s.shortcutRegistry = widgets.ShortcutRegistryMaybeOf(s.Context())
	s.updateAcceleratorShortcut()
}

func (s *menuAcceleratorLabelState) DidUpdateWidget(old *MenuAcceleratorLabel) {
	if s.Widget().label != old.label {
		s.updateDisplayLabel()
	}
}

func (s *menuAcceleratorLabelState) altIsPressed() bool {
	pressed := widgets.WidgetsBindingOf(s.Context()).Keyboard().LogicalKeysPressed()
	for _, k := range []services.LogicalKeyboardKey{services.LogicalKeyboardKeyAltLeft, services.LogicalKeyboardKeyAltRight, services.LogicalKeyboardKeyAlt} {
		if _, ok := pressed[k]; ok {
			return true
		}
	}
	return false
}

func (s *menuAcceleratorLabelState) listenToKeyEvent(services.KeyEvent) bool {
	s.SetState(func() {
		s.showAccelerators = s.altIsPressed()
		s.updateAcceleratorShortcut()
	})
	// Just listening, so it doesn't ever handle a key.
	return false
}

func (s *menuAcceleratorLabelState) updateAcceleratorShortcut() {
	if s.shortcutRegistryItem != nil {
		s.shortcutRegistryItem.Dispose()
		s.shortcutRegistryItem = nil
	}
	// Before registering an accelerator as a shortcut it should meet these
	// conditions:
	//
	// 1) Is showing accelerators (i.e. the Alt key is down).
	// 2) Has an accelerator marker in the label.
	// 3) Has an associated action callback for the label (from the
	//    MenuAcceleratorCallbackBinding).
	// 4) Is part of an anchor that either doesn't have a submenu, or doesn't have
	//    any submenus currently open (only the "deepest" open menu should have
	//    accelerator shortcuts registered).
	if s.showAccelerators && s.acceleratorIndex != -1 && s.binding != nil && s.binding.o.OnInvoke != nil &&
		(!s.binding.o.HasSubmenu || s.menuController == nil || !s.menuController.IsOpen()) && s.shortcutRegistry != nil {
		bounds := foundation.GraphemeBoundaries(s.displayLabel)
		character := strings.ToLower(s.displayLabel[bounds[s.acceleratorIndex]:bounds[s.acceleratorIndex+1]])
		s.shortcutRegistryItem = s.shortcutRegistry.AddAll(widgets.ShortcutMap{
			{Activator: widgets.NewCharacterActivator(character, widgets.CharacterActivatorOpts{Alt: true}),
				Intent: widgets.VoidCallbackIntent{Callback: s.binding.o.OnInvoke}},
		})
	}
}

func (s *menuAcceleratorLabelState) updateDisplayLabel() {
	s.displayLabel = MenuAcceleratorLabelStripAcceleratorMarkers(s.Widget().label, func(index int) { s.acceleratorIndex = index })
}

func (s *menuAcceleratorLabelState) Build(bc widgets.BuildContext) widgets.Widget {
	index := -1
	if s.showAccelerators {
		index = s.acceleratorIndex
	}
	return s.Widget().builder(bc, s.displayLabel, index)
}
