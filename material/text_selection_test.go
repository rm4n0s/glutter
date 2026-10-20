// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func TestMaterialTextSelectionControls(t *testing.T) {
	c := MaterialTextSelectionHandleControls
	if got := c.GetHandleSize(14); got != (ui.Size{Width: 22, Height: 22}) {
		t.Errorf("handle size = %v, want 22x22", got)
	}
	tests := []struct {
		handle rendering.TextSelectionHandleType
		want   ui.Offset
	}{
		{rendering.TextSelectionHandleTypeCollapsed, ui.Offset{Dx: 11, Dy: -4}},
		{rendering.TextSelectionHandleTypeLeft, ui.Offset{Dx: 22}},
		{rendering.TextSelectionHandleTypeRight, ui.Offset{}},
	}
	for _, tc := range tests {
		if got := c.GetHandleAnchor(tc.handle, 14); got != tc.want {
			t.Errorf("anchor of %v = %v, want %v", tc.handle, got, tc.want)
		}
	}
}

func TestMaterialTextSelectionHandleTaps(t *testing.T) {
	for _, handleType := range []rendering.TextSelectionHandleType{
		rendering.TextSelectionHandleTypeLeft, rendering.TextSelectionHandleTypeRight, rendering.TextSelectionHandleTypeCollapsed} {
		t.Run(handleType.Name(), func(t *testing.T) {
			taps := 0
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 200, Height: 200}})
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				return widgets.NewAlign(widgets.NewKeyedSubtree(
					MaterialTextSelectionHandleControls.BuildHandle(bc, handleType, 14, func() { taps++ }), widgets.WidgetOpts{Key: foundation.NewValueKey("handle")}),
					widgets.AlignOpts{Alignment: painting.AlignmentCenter})
			})})}))
			ts.Tap(gluttertest.ByKey(foundation.NewValueKey("handle")))
			if taps != 1 {
				t.Errorf("taps = %d, want 1", taps)
			}
		})
	}
}

func TestDesktopTextSelectionControlsHaveNoHandles(t *testing.T) {
	c := DesktopTextSelectionHandleControls
	if c.GetHandleSize(14) != ui.SizeZero || c.GetHandleAnchor(rendering.TextSelectionHandleTypeLeft, 14) != ui.OffsetZero {
		t.Error("the desktop controls have handles")
	}
	ts := gluttertest.New(t)
	var handle widgets.Widget
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		handle = c.BuildHandle(bc, rendering.TextSelectionHandleTypeCollapsed, 14, nil)
		return handle
	})}))
	if sb, ok := handle.(*widgets.SizedBox); !ok || sb.Width().OrElse(1) != 0 {
		t.Errorf("handle = %T, want an empty box", handle)
	}
}

func TestTextSelectionThemeDataBehavior(t *testing.T) {
	a := NewTextSelectionThemeData(TextSelectionThemeDataOpts{CursorColor: foundation.Some(ui.NewColor(0xFF000000)), SelectionColor: foundation.Some(ui.NewColor(0xFF000000))})
	b := NewTextSelectionThemeData(TextSelectionThemeDataOpts{CursorColor: foundation.Some(ui.NewColor(0xFFFFFFFF)), SelectionHandleColor: foundation.Some(ui.NewColor(0xFFFFFFFF))})
	if !a.Equal(a) || a.Equal(b) {
		t.Error("equality is wrong")
	}
	mid := TextSelectionThemeDataLerp(a, b, 0.5)
	if mid.CursorColor.Value() == a.CursorColor.Value() || mid.CursorColor.Value() == b.CursorColor.Value() {
		t.Errorf("lerp cursor color = %v", mid.CursorColor)
	}
	if got := a.CopyWith(TextSelectionThemeDataCopy{SelectionHandleColor: foundation.Some(ui.NewColor(0xFF010203))}); got.SelectionHandleColor != foundation.Some(ui.NewColor(0xFF010203)) || got.CursorColor != a.CursorColor {
		t.Errorf("CopyWith = %+v", got)
	}
	for _, want := range []string{"cursorColor", "selectionColor"} {
		if s := a.String(); foundation.DebugMode && !containsStr(s, want) {
			t.Errorf("description %q lacks %s", s, want)
		}
	}
	if a.ToStringShort() != "TextSelectionThemeData" {
		t.Errorf("short = %q", a.ToStringShort())
	}
}

func TestTextSelectionThemeOfAndWidget(t *testing.T) {
	data := NewTextSelectionThemeData(TextSelectionThemeDataOpts{CursorColor: foundation.Some(ui.NewColor(0xFF112233))})
	var got TextSelectionThemeData
	ts := gluttertest.New(t)
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewTextSelectionTheme(data, widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		got = TextSelectionThemeOf(bc)
		return widgets.NewSizedBox(nil)
	}))}))
	if got != data {
		t.Errorf("TextSelectionThemeOf = %+v, want %+v", got, data)
	}
	w := NewTextSelectionTheme(data, widgets.NewSizedBox(nil))
	if !w.UpdateShouldNotify(NewTextSelectionTheme(TextSelectionThemeData{}, widgets.NewSizedBox(nil))) || w.UpdateShouldNotify(NewTextSelectionTheme(data, widgets.NewSizedBox(nil))) {
		t.Error("UpdateShouldNotify is wrong")
	}
}

func TestTextSelectionThemeStylesTheField(t *testing.T) {
	cursor, selection := ui.NewColor(0xFF00CC00), ui.NewColor(0x6600CC00)
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 200}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{
		Theme: NewThemeData(ThemeDataOpts{TextSelectionTheme: NewTextSelectionThemeData(TextSelectionThemeDataOpts{
			CursorColor: foundation.Some(cursor), SelectionColor: foundation.Some(selection)})}),
		Home: NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(widgets.NewSizedBox(NewTextField(TextFieldOpts{Controller: widgets.NewTextEditingController("hello")}),
			widgets.SizedBoxOpts{Width: foundation.Some(300.0)}))}),
	}))
	ts.Tap(gluttertest.ByType[*TextField]())
	ts.Pump()
	r := ts.RenderObject(gluttertest.Descendant(gluttertest.ByType[*TextField](), gluttertest.ByElementPredicate("render editable", func(e widgets.Element) bool {
		_, ok := e.AsElement().RenderObject().(*rendering.RenderEditable)
		return ok
	}))).(*rendering.RenderEditable)
	if got := r.CursorColor().OrElse(ui.Color{}).WithOpacity(1); got != cursor.WithOpacity(1) {
		t.Errorf("cursor color = %v, want %v", got, cursor)
	}
	if got := r.SelectionColor(); got != foundation.Some(selection) {
		t.Errorf("selection color = %v, want %v", got, selection)
	}
}

func TestAdaptiveTextSelectionToolbarLabels(t *testing.T) {
	ts := gluttertest.New(t)
	var labels map[string]string
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		labels = map[string]string{}
		for _, item := range []widgets.ContextMenuButtonItem{
			{Type: widgets.ContextMenuButtonTypeCut},
			{Type: widgets.ContextMenuButtonTypeCopy},
			{Type: widgets.ContextMenuButtonTypePaste},
			{Type: widgets.ContextMenuButtonTypeSelectAll},
			{Type: widgets.ContextMenuButtonTypeDelete},
			{Type: widgets.ContextMenuButtonTypeLookUp},
			{Type: widgets.ContextMenuButtonTypeSearchWeb},
			{Type: widgets.ContextMenuButtonTypeShare},
			{Type: widgets.ContextMenuButtonTypeLiveTextInput},
			{Type: widgets.ContextMenuButtonTypeCustom},
			{Type: widgets.ContextMenuButtonTypeCopy, Label: str("Duplicate")},
		} {
			key := item.Type.Name()
			if item.Label.IsSome() {
				key = "label"
			}
			labels[key] = AdaptiveTextSelectionToolbarGetButtonLabel(bc, item)
		}
		return widgets.NewSizedBox(nil)
	})}))
	want := map[string]string{
		"cut": "Cut", "copy": "Copy", "paste": "Paste", "selectAll": "Select all", "delete": "DELETE",
		"lookUp": "Look Up", "searchWeb": "Search Web", "share": "Share", "liveTextInput": "Scan text",
		"custom": "", "label": "Duplicate",
	}
	for k, w := range want {
		if labels[k] != w {
			t.Errorf("label of %s = %q, want %q", k, labels[k], w)
		}
	}
}

func TestAdaptiveTextSelectionToolbarBuilds(t *testing.T) {
	var pressed []string
	press := func(s string) func() { return func() { pressed = append(pressed, s) } }
	tests := []struct {
		name    string
		toolbar widgets.Widget
		want    []string
		none    bool
	}{
		{"button items", NewAdaptiveTextSelectionToolbarButtonItems([]widgets.ContextMenuButtonItem{
			{OnPressed: press("copy"), Type: widgets.ContextMenuButtonTypeCopy},
			{OnPressed: press("mine"), Label: str("Mine")}},
			widgets.TextSelectionToolbarAnchors{PrimaryAnchor: ui.Offset{Dx: 100, Dy: 100}}), []string{"Copy", "Mine"}, false},
		{"children", NewAdaptiveTextSelectionToolbar(widgets.TextSelectionToolbarAnchors{PrimaryAnchor: ui.Offset{Dx: 100, Dy: 100}},
			[]widgets.Widget{widgets.NewText("child one")}), []string{"child one"}, false},
		{"no items builds nothing", NewAdaptiveTextSelectionToolbarButtonItems(nil, widgets.TextSelectionToolbarAnchors{}), nil, true},
		{"editable items", NewAdaptiveTextSelectionToolbarEditable(widgets.ClipboardStatusPasteable,
			widgets.EditableButtonItemsCallbacks{OnCopy: press("copy"), OnPaste: press("paste"), OnSelectAll: press("all")},
			widgets.TextSelectionToolbarAnchors{PrimaryAnchor: ui.Offset{Dx: 100, Dy: 100}}), []string{"Copy", "Paste", "Select all"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 400}})
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: tc.toolbar})}))
			if got := ts.Exists(gluttertest.ByType[*DesktopTextSelectionToolbar]()); got == tc.none {
				t.Errorf("toolbar built = %v, want %v", got, !tc.none)
			}
			for _, label := range tc.want {
				if !ts.Exists(gluttertest.ByText(label)) {
					t.Errorf("no %q", label)
				}
			}
		})
	}
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 400}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: tests[0].toolbar})}))
	ts.Tap(gluttertest.ByText("Mine"))
	if len(pressed) != 1 || pressed[0] != "mine" {
		t.Errorf("pressed = %q, want [mine]", pressed)
	}
}

func TestDesktopTextSelectionToolbarLayout(t *testing.T) {
	tests := []struct {
		name   string
		anchor ui.Offset
		want   ui.Offset // the top left of the card
	}{
		{"at the anchor", ui.Offset{Dx: 50, Dy: 60}, ui.Offset{Dx: 50, Dy: 60}},
		{"moved back from the right edge", ui.Offset{Dx: 390, Dy: 60}, ui.Offset{Dx: 400 - 8 - 222}},
		{"moved up from the bottom edge", ui.Offset{Dx: 50, Dy: 395}, ui.Offset{Dx: 50}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 400}})
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: NewDesktopTextSelectionToolbar(tc.anchor,
				[]widgets.Widget{NewDesktopTextSelectionToolbarButton(func() {}, widgets.NewText("one"))})})}))
			// The first box below the toolbar is the one of the card.
			cardFinder := gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*DesktopTextSelectionToolbar](), gluttertest.ByType[*widgets.SizedBox]()))
			card := ts.RenderObject(cardFinder).(rendering.Box)
			if w := card.Size().Width; w != 222 {
				t.Errorf("width = %v, want 222", w)
			}
			pos := ts.TopLeft(cardFinder)
			if tc.name == "at the anchor" && pos != tc.want {
				t.Errorf("position = %v, want %v", pos, tc.want)
			}
			if pos.Dx+222 > 400-8+0.001 {
				t.Errorf("the toolbar goes over the right edge: x = %v", pos.Dx)
			}
			if pos.Dy+card.Size().Height > 400+0.001 {
				t.Errorf("the toolbar goes over the bottom edge: y = %v", pos.Dy)
			}
		})
	}
}

func TestDesktopTextSelectionToolbarButtonColors(t *testing.T) {
	tests := []struct {
		name  string
		theme *ThemeData
		want  ui.Color
	}{
		{"light", NewThemeData(ThemeDataOpts{Brightness: ui.BrightnessLight}), Colors.Black87},
		{"dark", NewThemeData(ThemeDataOpts{Brightness: ui.BrightnessDark}), Colors.White},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 400}})
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: tc.theme, DarkTheme: tc.theme, Home: NewScaffold(ScaffoldOpts{Body: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				return widgets.NewAlign(NewDesktopTextSelectionToolbarButtonText(bc, func() {}, "label"), widgets.AlignOpts{Alignment: painting.AlignmentTopLeft})
			})})}))
			text := ts.Widget(gluttertest.ByText("label")).(*widgets.Text)
			if text.Style().Color != foundation.Some(tc.want) {
				t.Errorf("text color = %v, want %v", text.Style().Color, tc.want)
			}
		})
	}
}

func TestDesktopTextSelectionToolbarNeedsAChild(t *testing.T) {
	skipUnlessDebug(t)
	wantRoute(t, recoverError(func() { NewDesktopTextSelectionToolbar(ui.Offset{}, nil) }), "DesktopTextSelectionToolbarNoChildren")
}

func TestSelectionAreaSelectsAndCopies(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 300}})
	var changed []string
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewAlign(NewSelectionArea(
		widgets.NewText("hello selectable world"), SelectionAreaOpts{OnSelectionChanged: func(c *rendering.SelectedContent) {
			if c != nil {
				changed = append(changed, c.PlainText)
			}
		}}), widgets.AlignOpts{Alignment: painting.AlignmentTopLeft})})}))
	state := gluttertest.StateOf[*SelectionAreaState](ts, gluttertest.ByType[*SelectionArea]())
	state.SelectableRegion().SelectAll()
	ts.Pump()
	if len(changed) == 0 || changed[len(changed)-1] != "hello selectable world" {
		t.Errorf("selected = %q", changed)
	}
	state.SelectableRegion().CopySelection()
	ts.Pump()
	if text, _ := ts.ClipboardText(); text != "hello selectable world" {
		t.Errorf("clipboard = %q", text)
	}
}

func TestSelectionAreaContextMenu(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 300}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewAlign(NewSelectionArea(
		widgets.NewText("hello selectable world")), widgets.AlignOpts{Alignment: painting.AlignmentTopLeft})})}))
	rightClick(ts, ts.TopLeft(gluttertest.ByText("hello selectable world")).Add(ui.Offset{Dx: 20, Dy: 6}))
	if !ts.Exists(gluttertest.ByText("Select all")) {
		t.Error("the context menu has no Select all button")
	}
	ts.Tap(gluttertest.ByText("Select all"))
	ts.Pump()
	state := gluttertest.StateOf[*SelectionAreaState](ts, gluttertest.ByType[*SelectionArea]())
	if state.SelectableRegion() == nil {
		t.Fatal("no selectable region")
	}
}

func TestSelectionAreaNoContextMenu(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 300}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewAlign(NewSelectionArea(
		widgets.NewText("hello selectable world"), SelectionAreaOpts{NoContextMenu: true}), widgets.AlignOpts{Alignment: painting.AlignmentTopLeft})})}))
	rightClick(ts, ts.TopLeft(gluttertest.ByText("hello selectable world")).Add(ui.Offset{Dx: 20, Dy: 6}))
	if ts.Exists(gluttertest.ByText("Select all")) {
		t.Error("a context menu was built")
	}
}

var _ = services.LogicalKeyboardKeyEnter
