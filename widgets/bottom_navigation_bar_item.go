// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// BottomNavigationBarItemOpts are the optional parameters of NewBottomNavigationBarItem.
//
// Flutter: widgets/bottom_navigation_bar_item.dart BottomNavigationBarItem.new
type BottomNavigationBarItemOpts struct {
	// Key is a key to be passed through to the resultant widget.
	Key foundation.Key
	// Label is the text label of the item; the empty string means none.
	Label string
	// ActiveIcon is the icon shown when the item is selected; the default is the icon.
	ActiveIcon Widget
	// BackgroundColor is the color of the background of the Material bar; the Cupertino tab bar does not use it.
	BackgroundColor foundation.Opt[ui.Color]
	// Tooltip is the text of the tooltip of the item; the empty string means none.
	Tooltip string
	// SemanticsLabel is the semantics label of the item, which replaces the label for the accessibility tools.
	SemanticsLabel foundation.Opt[string]
}

// BottomNavigationBarItem is an interactive button within either a Material or a Cupertino bottom navigation
// bar: an icon with an optional label.
//
// Flutter: widgets/bottom_navigation_bar_item.dart BottomNavigationBarItem
type BottomNavigationBarItem struct {
	key             foundation.Key
	icon            Widget
	activeIcon      Widget
	label           string
	backgroundColor foundation.Opt[ui.Color]
	tooltip         string
	semanticsLabel  foundation.Opt[string]
}

// NewBottomNavigationBarItem creates an item with the icon. The active icon defaults to the icon.
//
// Flutter: widgets/bottom_navigation_bar_item.dart BottomNavigationBarItem.new
func NewBottomNavigationBarItem(icon Widget, opts ...BottomNavigationBarItemOpts) BottomNavigationBarItem {
	o := firstOpts(opts)
	active := o.ActiveIcon
	if active == nil {
		active = icon
	}
	return BottomNavigationBarItem{o.Key, icon, active, o.Label, o.BackgroundColor, o.Tooltip, o.SemanticsLabel}
}

// Key is the key that is passed through to the resultant widget.
//
// Flutter: widgets/bottom_navigation_bar_item.dart BottomNavigationBarItem.key
func (i BottomNavigationBarItem) Key() foundation.Key { return i.key }

// Icon is the icon of the item.
//
// Flutter: widgets/bottom_navigation_bar_item.dart BottomNavigationBarItem.icon
func (i BottomNavigationBarItem) Icon() Widget { return i.icon }

// ActiveIcon is the icon that shows when the item is selected.
//
// Flutter: widgets/bottom_navigation_bar_item.dart BottomNavigationBarItem.activeIcon
func (i BottomNavigationBarItem) ActiveIcon() Widget { return i.activeIcon }

// Label is the text label of the item; the empty string means none.
//
// Flutter: widgets/bottom_navigation_bar_item.dart BottomNavigationBarItem.label
func (i BottomNavigationBarItem) Label() string { return i.label }

// BackgroundColor is the color of the background of the Material bar.
//
// Flutter: widgets/bottom_navigation_bar_item.dart BottomNavigationBarItem.backgroundColor
func (i BottomNavigationBarItem) BackgroundColor() foundation.Opt[ui.Color] { return i.backgroundColor }

// Tooltip is the text of the tooltip of the item; the empty string means none.
//
// Flutter: widgets/bottom_navigation_bar_item.dart BottomNavigationBarItem.tooltip
func (i BottomNavigationBarItem) Tooltip() string { return i.tooltip }

// SemanticsLabel is the semantics label of the item, if one was given.
//
// Flutter: widgets/bottom_navigation_bar_item.dart BottomNavigationBarItem.semanticsLabel
func (i BottomNavigationBarItem) SemanticsLabel() foundation.Opt[string] { return i.semanticsLabel }
