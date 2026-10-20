// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

// DismissDirection is the direction in which a Dismissible can be dismissed.
//
// TODO: the Dismissible widget (Flutter's widgets/dismissible.dart), which is
// not part of the plan yet; only the enum exists, for SnackBar's dismissDirection.
//
// Flutter: widgets/dismissible.dart DismissDirection
type DismissDirection int

const (
	// DismissDirectionUnspecified is the zero value: the default of the widget.
	//
	// Flutter: no counterpart
	DismissDirectionUnspecified DismissDirection = iota
	// DismissDirectionVertical can be dismissed by dragging either up or down.
	//
	// Flutter: widgets/dismissible.dart DismissDirection.vertical
	DismissDirectionVertical
	// DismissDirectionHorizontal can be dismissed by dragging either left or right.
	//
	// Flutter: widgets/dismissible.dart DismissDirection.horizontal
	DismissDirectionHorizontal
	// DismissDirectionEndToStart can be dismissed by dragging in the reading
	// direction's opposite direction (to the start).
	//
	// Flutter: widgets/dismissible.dart DismissDirection.endToStart
	DismissDirectionEndToStart
	// DismissDirectionStartToEnd can be dismissed by dragging in the reading
	// direction (to the end).
	//
	// Flutter: widgets/dismissible.dart DismissDirection.startToEnd
	DismissDirectionStartToEnd
	// DismissDirectionUp can be dismissed by dragging up only.
	//
	// Flutter: widgets/dismissible.dart DismissDirection.up
	DismissDirectionUp
	// DismissDirectionDown can be dismissed by dragging down only.
	//
	// Flutter: widgets/dismissible.dart DismissDirection.down
	DismissDirectionDown
	// DismissDirectionNone cannot be dismissed by dragging.
	//
	// Flutter: widgets/dismissible.dart DismissDirection.none
	DismissDirectionNone
)

// Name returns the Dart enum name, such as "endToStart".
//
// Flutter: widgets/dismissible.dart DismissDirection.name
func (d DismissDirection) Name() string {
	return enumName(int(d), "vertical", "horizontal", "endToStart", "startToEnd", "up", "down", "none")
}

// String returns the name Dart prints, such as "DismissDirection.up".
//
// Flutter: widgets/dismissible.dart DismissDirection.toString
func (d DismissDirection) String() string { return "DismissDirection." + d.Name() }
