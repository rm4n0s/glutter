// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// ActionButtonOpts are the optional parameters of the action buttons.
//
// Flutter: material/action_buttons.dart BackButton.new
type ActionButtonOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Color is the color of the icon; the default is the one of the icon theme.
	Color foundation.Opt[ui.Color]
	// Style customizes the appearance of the IconButton of the button.
	Style *ButtonStyle
	// OnPressed is called when the button is pressed; the default does what the
	// button is for (pops the route, opens the drawer).
	OnPressed func()
}

// actionIcon is the icon of an action button, with the label that Android reads.
//
// Flutter: material/action_buttons.dart _ActionIcon
func actionIcon(bc widgets.BuildContext, builder func(ActionIconThemeData) widgets.WidgetBuilder, data widgets.IconData, androidLabel func(MaterialLocalizations) string) widgets.Widget {
	if b := builder(ActionIconThemeOf(bc)); b != nil {
		return b(bc)
	}
	opts := widgets.IconOpts{}
	// This can't use the platform from Theme because it is the Android OS that
	// expects the duplicated tooltip and label.
	if foundation.DefaultTargetPlatform() == foundation.TargetPlatformAndroid {
		opts.SemanticLabel = foundation.Some(androidLabel(MaterialLocalizationsOf(bc)))
	}
	return widgets.NewIcon(data, opts)
}

// BackButtonIcon is the icon of a BackButton: an arrow back that depends on the
// platform.
//
// Flutter: material/action_buttons.dart BackButtonIcon
type BackButtonIcon struct{ widgets.StatelessWidget }

// NewBackButtonIcon creates the icon.
//
// Flutter: material/action_buttons.dart BackButtonIcon.new
func NewBackButtonIcon(opts ...widgets.WidgetOpts) *BackButtonIcon {
	return &BackButtonIcon{widgets.NewStatelessWidgetBase(opts...)}
}

// Build builds the icon.
//
// Flutter: material/action_buttons.dart BackButtonIcon.build
func (w *BackButtonIcon) Build(bc widgets.BuildContext) widgets.Widget {
	data := Icons.ArrowBack()
	switch ThemeOf(bc).Platform {
	case foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS:
		data = Icons.ArrowBackIosNewRounded()
	}
	return actionIcon(bc, func(t ActionIconThemeData) widgets.WidgetBuilder { return t.BackButtonIconBuilder }, data, MaterialLocalizations.BackButtonTooltip)
}

// BackButton is an icon button with a platform specific back icon that pops the
// route.
//
// Flutter: material/action_buttons.dart BackButton
type BackButton struct {
	widgets.StatelessWidget
	o ActionButtonOpts
}

// NewBackButton creates the button.
//
// Flutter: material/action_buttons.dart BackButton.new
func NewBackButton(opts ...ActionButtonOpts) *BackButton {
	o := firstOpts(opts)
	return &BackButton{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), o: o}
}

// Build builds the button.
//
// Flutter: material/action_buttons.dart _ActionButton.build
func (w *BackButton) Build(bc widgets.BuildContext) widgets.Widget {
	DebugCheckHasMaterialLocalizations(bc)
	return NewIconButton(func() {
		if w.o.OnPressed != nil {
			w.o.OnPressed()
		} else {
			widgets.NavigatorMaybePop(bc)
		}
	}, NewBackButtonIcon(), IconButtonOpts{Style: w.o.Style, Color: w.o.Color, Tooltip: MaterialLocalizationsOf(bc).BackButtonTooltip()})
}

// CloseButtonIcon is the icon of a CloseButton.
//
// Flutter: material/action_buttons.dart CloseButtonIcon
type CloseButtonIcon struct{ widgets.StatelessWidget }

// NewCloseButtonIcon creates the icon.
//
// Flutter: material/action_buttons.dart CloseButtonIcon.new
func NewCloseButtonIcon(opts ...widgets.WidgetOpts) *CloseButtonIcon {
	return &CloseButtonIcon{widgets.NewStatelessWidgetBase(opts...)}
}

// Build builds the icon.
//
// Flutter: material/action_buttons.dart CloseButtonIcon.build
func (w *CloseButtonIcon) Build(bc widgets.BuildContext) widgets.Widget {
	return actionIcon(bc, func(t ActionIconThemeData) widgets.WidgetBuilder { return t.CloseButtonIconBuilder }, Icons.Close(), MaterialLocalizations.CloseButtonTooltip)
}

// CloseButton is an icon button with a close icon that pops the route.
//
// Flutter: material/action_buttons.dart CloseButton
type CloseButton struct {
	widgets.StatelessWidget
	o ActionButtonOpts
}

// NewCloseButton creates the button.
//
// Flutter: material/action_buttons.dart CloseButton.new
func NewCloseButton(opts ...ActionButtonOpts) *CloseButton {
	o := firstOpts(opts)
	return &CloseButton{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), o: o}
}

// Build builds the button.
//
// Flutter: material/action_buttons.dart _ActionButton.build
func (w *CloseButton) Build(bc widgets.BuildContext) widgets.Widget {
	DebugCheckHasMaterialLocalizations(bc)
	return NewIconButton(func() {
		if w.o.OnPressed != nil {
			w.o.OnPressed()
		} else {
			widgets.NavigatorMaybePop(bc)
		}
	}, NewCloseButtonIcon(), IconButtonOpts{Style: w.o.Style, Color: w.o.Color, Tooltip: MaterialLocalizationsOf(bc).CloseButtonTooltip()})
}

// DrawerButtonIcon is the icon of a DrawerButton: a menu.
//
// Flutter: material/action_buttons.dart DrawerButtonIcon
type DrawerButtonIcon struct{ widgets.StatelessWidget }

// NewDrawerButtonIcon creates the icon.
//
// Flutter: material/action_buttons.dart DrawerButtonIcon.new
func NewDrawerButtonIcon(opts ...widgets.WidgetOpts) *DrawerButtonIcon {
	return &DrawerButtonIcon{widgets.NewStatelessWidgetBase(opts...)}
}

// Build builds the icon.
//
// Flutter: material/action_buttons.dart DrawerButtonIcon.build
func (w *DrawerButtonIcon) Build(bc widgets.BuildContext) widgets.Widget {
	return actionIcon(bc, func(t ActionIconThemeData) widgets.WidgetBuilder { return t.DrawerButtonIconBuilder }, Icons.Menu(), MaterialLocalizations.OpenAppDrawerTooltip)
}

// DrawerButton is an icon button that opens the drawer of the Scaffold.
//
// Flutter: material/action_buttons.dart DrawerButton
type DrawerButton struct {
	widgets.StatelessWidget
	o ActionButtonOpts
}

// NewDrawerButton creates the button.
//
// Flutter: material/action_buttons.dart DrawerButton.new
func NewDrawerButton(opts ...ActionButtonOpts) *DrawerButton {
	o := firstOpts(opts)
	return &DrawerButton{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), o: o}
}

// Build builds the button.
//
// Flutter: material/action_buttons.dart _ActionButton.build
func (w *DrawerButton) Build(bc widgets.BuildContext) widgets.Widget {
	DebugCheckHasMaterialLocalizations(bc)
	return NewIconButton(func() {
		if w.o.OnPressed != nil {
			w.o.OnPressed()
		} else {
			ScaffoldOf(bc).OpenDrawer()
		}
	}, NewDrawerButtonIcon(), IconButtonOpts{Style: w.o.Style, Color: w.o.Color, Tooltip: MaterialLocalizationsOf(bc).OpenAppDrawerTooltip()})
}

// EndDrawerButtonIcon is the icon of an EndDrawerButton: a menu.
//
// Flutter: material/action_buttons.dart EndDrawerButtonIcon
type EndDrawerButtonIcon struct{ widgets.StatelessWidget }

// NewEndDrawerButtonIcon creates the icon.
//
// Flutter: material/action_buttons.dart EndDrawerButtonIcon.new
func NewEndDrawerButtonIcon(opts ...widgets.WidgetOpts) *EndDrawerButtonIcon {
	return &EndDrawerButtonIcon{widgets.NewStatelessWidgetBase(opts...)}
}

// Build builds the icon.
//
// Flutter: material/action_buttons.dart EndDrawerButtonIcon.build
func (w *EndDrawerButtonIcon) Build(bc widgets.BuildContext) widgets.Widget {
	return actionIcon(bc, func(t ActionIconThemeData) widgets.WidgetBuilder { return t.EndDrawerButtonIconBuilder }, Icons.Menu(), MaterialLocalizations.OpenAppDrawerTooltip)
}

// EndDrawerButton is an icon button that opens the end drawer of the Scaffold.
//
// Flutter: material/action_buttons.dart EndDrawerButton
type EndDrawerButton struct {
	widgets.StatelessWidget
	o ActionButtonOpts
}

// NewEndDrawerButton creates the button.
//
// Flutter: material/action_buttons.dart EndDrawerButton.new
func NewEndDrawerButton(opts ...ActionButtonOpts) *EndDrawerButton {
	o := firstOpts(opts)
	return &EndDrawerButton{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), o: o}
}

// Build builds the button.
//
// Flutter: material/action_buttons.dart _ActionButton.build
func (w *EndDrawerButton) Build(bc widgets.BuildContext) widgets.Widget {
	DebugCheckHasMaterialLocalizations(bc)
	return NewIconButton(func() {
		if w.o.OnPressed != nil {
			w.o.OnPressed()
		} else {
			ScaffoldOf(bc).OpenEndDrawer()
		}
	}, NewEndDrawerButtonIcon(), IconButtonOpts{Style: w.o.Style, Color: w.o.Color, Tooltip: MaterialLocalizationsOf(bc).OpenAppDrawerTooltip()})
}
