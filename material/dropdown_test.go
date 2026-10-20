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
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func TestDropdownButtonSelects(t *testing.T) {
	var changed []string
	value := foundation.Some("b")
	items := func() []*DropdownMenuItem[string] {
		var out []*DropdownMenuItem[string]
		for _, v := range []string{"a", "b", "c"} {
			out = append(out, NewDropdownMenuItem(widgets.NewText("item "+v), DropdownMenuItemOpts[string]{Value: foundation.Some(v)}))
		}
		return out
	}
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
	build := func() widgets.Widget {
		return NewMaterialApp(MaterialAppOpts{Theme: standardTheme(true), Home: NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(
			NewDropdownButton(items(), func(v foundation.Opt[string]) { changed = append(changed, v.Value()) },
				DropdownButtonOpts[string]{Value: value}))})})
	}
	ts.PumpWidget(build())
	if got := ts.Count(gluttertest.ByText("item b")); got != 1 {
		t.Fatalf("the selected item is shown %d times, want 1", got)
	}
	ts.Tap(gluttertest.ByText("item b"))
	ts.PumpAndSettle()
	if ts.Count(gluttertest.ByText("item a")) == 0 || ts.Count(gluttertest.ByText("item c")) == 0 {
		t.Fatal("the menu is not shown")
	}
	ts.Tap(gluttertest.ByText("item c"))
	ts.PumpAndSettle()
	if len(changed) != 1 || changed[0] != "c" {
		t.Errorf("changed = %v, want [c]", changed)
	}
	if got := ts.Count(gluttertest.ByText("item a")); got != 0 {
		t.Errorf("the menu is still open")
	}
}

func TestDropdownButtonStates(t *testing.T) {
	mk := func(values ...string) []*DropdownMenuItem[string] {
		var out []*DropdownMenuItem[string]
		for _, v := range values {
			out = append(out, NewDropdownMenuItem(widgets.NewText("item "+v), DropdownMenuItemOpts[string]{Value: foundation.Some(v), Disabled: v == "off"}))
		}
		return out
	}
	hint := func() widgets.Widget { return widgets.NewText("hint") }
	tests := []struct {
		name        string
		onChanged   func(foundation.Opt[string])
		opts        DropdownButtonOpts[string]
		items       []*DropdownMenuItem[string]
		wantShown   string
		wantOpens   bool
		tapDisabled bool
	}{
		{"hint without a value", func(foundation.Opt[string]) {}, DropdownButtonOpts[string]{Hint: hint()}, mk("a", "b"), "hint", true, false},
		{"disabled hint", nil, DropdownButtonOpts[string]{Hint: hint(), DisabledHint: widgets.NewText("off hint")}, mk("a", "b"), "off hint", false, false},
		{"disabled does not open", nil, DropdownButtonOpts[string]{Value: foundation.Some("a")}, mk("a", "b"), "item a", false, false},
		{"selected item builder", func(foundation.Opt[string]) {}, DropdownButtonOpts[string]{Value: foundation.Some("b"),
			SelectedItemBuilder: func(widgets.BuildContext) []widgets.Widget {
				return []widgets.Widget{widgets.NewText("short a"), widgets.NewText("short b")}
			}}, mk("a", "b"), "short b", true, false},
		{"disabled item is not selectable", func(foundation.Opt[string]) {}, DropdownButtonOpts[string]{Value: foundation.Some("a")}, mk("a", "off"), "item a", true, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			changed := 0
			onChanged := tc.onChanged
			if onChanged != nil {
				onChanged = func(foundation.Opt[string]) { changed++ }
			}
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: standardTheme(true), Home: NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(
				NewDropdownButton(tc.items, onChanged, tc.opts))})}))
			if !ts.Exists(gluttertest.ByText(tc.wantShown)) {
				t.Fatalf("%q is not shown", tc.wantShown)
			}
			ts.Tap(gluttertest.ByType[*DropdownButton[string]]())
			ts.PumpAndSettle()
			opened := ts.Exists(gluttertest.ByText("item b")) || ts.Exists(gluttertest.ByText("item off"))
			if opened != tc.wantOpens {
				t.Fatalf("menu open = %v, want %v", opened, tc.wantOpens)
			}
			if tc.tapDisabled {
				ts.Tap(gluttertest.ByText("item off"))
				ts.PumpAndSettle()
				if changed != 0 || !ts.Exists(gluttertest.ByText("item off")) {
					t.Errorf("a disabled item was selected: changed %d", changed)
				}
			}
		})
	}
}
