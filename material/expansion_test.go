// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func TestCheckboxListTile(t *testing.T) {
	var got []foundation.Opt[bool]
	onChanged := func(v foundation.Opt[bool]) { got = append(got, v) }
	tests := []struct {
		name  string
		value foundation.Opt[bool]
		opts  CheckboxListTileOpts
		want  foundation.Opt[bool]
	}{
		{"unchecked", foundation.Some(false), CheckboxListTileOpts{}, foundation.Some(true)},
		{"checked", foundation.Some(true), CheckboxListTileOpts{}, foundation.Some(false)},
		{"checked tristate", foundation.Some(true), CheckboxListTileOpts{Tristate: true}, foundation.None[bool]()},
		{"none tristate", foundation.None[bool](), CheckboxListTileOpts{Tristate: true}, foundation.Some(false)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got = nil
			tc.opts.Title = widgets.NewText("Title")
			ts := pumpTile(t, standardTheme(true), widgets.NewSizedBox(NewCheckboxListTile(tc.value, onChanged, tc.opts), widgets.SizedBoxOpts{Width: foundation.Some(400.0)}))
			ts.Tap(gluttertest.ByText("Title"))
			ts.PumpAndSettle()
			if len(got) != 1 || got[0] != tc.want {
				t.Errorf("onChanged got %v, want %v", got, tc.want)
			}
		})
	}
	t.Run("disabled", func(t *testing.T) {
		got = nil
		ts := pumpTile(t, standardTheme(true), widgets.NewSizedBox(NewCheckboxListTile(foundation.Some(false), nil, CheckboxListTileOpts{Title: widgets.NewText("Title")}),
			widgets.SizedBoxOpts{Width: foundation.Some(400.0)}))
		ts.Tap(gluttertest.ByText("Title"))
		if len(got) != 0 {
			t.Errorf("a disabled tile changed: %v", got)
		}
	})
	t.Run("control affinity", func(t *testing.T) {
		for _, affinity := range []ListTileControlAffinity{ListTileControlAffinityLeading, ListTileControlAffinityTrailing, ListTileControlAffinityPlatform} {
			ts := pumpTile(t, standardTheme(true), widgets.NewSizedBox(NewCheckboxListTile(foundation.Some(false), onChanged,
				CheckboxListTileOpts{Title: widgets.NewText("Title"), ControlAffinity: affinity}), widgets.SizedBoxOpts{Width: foundation.Some(400.0)}))
			box := ts.Center(gluttertest.ByType[*Checkbox]()).Dx
			title := ts.Center(gluttertest.ByText("Title")).Dx
			if wantLeading := affinity == ListTileControlAffinityLeading; (box < title) != wantLeading {
				t.Errorf("affinity %v: the checkbox is at %v, the title at %v", affinity, box, title)
			}
		}
	})
	t.Run("secondary and selected", func(t *testing.T) {
		ts := pumpTile(t, standardTheme(true), widgets.NewSizedBox(NewCheckboxListTile(foundation.Some(true), onChanged,
			CheckboxListTileOpts{Title: widgets.NewText("Title"), Secondary: NewIcon(Icons.Add()), Selected: true}), widgets.SizedBoxOpts{Width: foundation.Some(400.0)}))
		if !ts.Exists(gluttertest.ByIcon(Icons.Add())) {
			t.Error("no secondary widget")
		}
	})
}

func TestSwitchListTile(t *testing.T) {
	var got []bool
	onChanged := func(v bool) { got = append(got, v) }
	for _, value := range []bool{false, true} {
		got = nil
		ts := pumpTile(t, standardTheme(true), widgets.NewSizedBox(NewSwitchListTile(value, onChanged, SwitchListTileOpts{Title: widgets.NewText("Title")}),
			widgets.SizedBoxOpts{Width: foundation.Some(400.0)}))
		ts.Tap(gluttertest.ByText("Title"))
		ts.PumpAndSettle()
		if len(got) != 1 || got[0] != !value {
			t.Errorf("value %v: onChanged got %v", value, got)
		}
	}
	ts := pumpTile(t, standardTheme(true), widgets.NewSizedBox(NewSwitchListTile(false, onChanged, SwitchListTileOpts{Title: widgets.NewText("Title"), ControlAffinity: ListTileControlAffinityLeading}),
		widgets.SizedBoxOpts{Width: foundation.Some(400.0)}))
	if ts.Center(gluttertest.ByType[*Switch]()).Dx > ts.Center(gluttertest.ByText("Title")).Dx {
		t.Error("the switch is not at the start")
	}
	ts = pumpTile(t, standardTheme(true), widgets.NewSizedBox(NewSwitchListTile(false, nil, SwitchListTileOpts{Title: widgets.NewText("Title")}),
		widgets.SizedBoxOpts{Width: foundation.Some(400.0)}))
	got = nil
	ts.Tap(gluttertest.ByText("Title"))
	if len(got) != 0 {
		t.Error("a disabled switch tile changed")
	}
}

func TestRadioListTile(t *testing.T) {
	t.Run("with a group", func(t *testing.T) {
		var picked []foundation.Opt[int]
		group := foundation.Some(1)
		ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
		var rebuild func()
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: standardTheme(true), Home: NewScaffold(ScaffoldOpts{Body: widgets.NewStatefulBuilder(func(bc widgets.BuildContext, set widgets.StateSetter) widgets.Widget {
			rebuild = func() { set(func() {}) }
			return widgets.NewSizedBox(widgets.NewRadioGroup(func(v foundation.Opt[int]) {
				picked = append(picked, v)
				group = v
				rebuild()
			}, widgets.NewColumn([]widgets.Widget{
				NewRadioListTile(1, RadioListTileOpts[int]{Title: widgets.NewText("One")}),
				NewRadioListTile(2, RadioListTileOpts[int]{Title: widgets.NewText("Two")}),
			}), widgets.RadioGroupOpts[int]{GroupValue: group}), widgets.SizedBoxOpts{Width: foundation.Some(400.0)})
		})})}))
		ts.PumpAndSettle()
		ts.Tap(gluttertest.ByText("Two"))
		ts.PumpAndSettle()
		if len(picked) != 1 || picked[0] != foundation.Some(2) {
			t.Errorf("picked %v", picked)
		}
		ts.Tap(gluttertest.ByText("Two"))
		ts.PumpAndSettle()
		if len(picked) != 1 {
			t.Errorf("a selected radio that is not toggleable fires again: %v", picked)
		}
	})
	t.Run("with callbacks", func(t *testing.T) {
		var picked []foundation.Opt[string]
		for _, tc := range []struct {
			name       string
			group      foundation.Opt[string]
			toggleable bool
			want       []foundation.Opt[string]
		}{
			{"select", foundation.Some("b"), false, []foundation.Opt[string]{foundation.Some("a")}},
			{"selected", foundation.Some("a"), false, nil},
			{"toggled off", foundation.Some("a"), true, []foundation.Opt[string]{foundation.None[string]()}},
		} {
			picked = nil
			ts := pumpTile(t, standardTheme(true), widgets.NewSizedBox(NewRadioListTile("a", RadioListTileOpts[string]{
				Title: widgets.NewText("A"), GroupValue: tc.group, Toggleable: tc.toggleable, OnChanged: func(v foundation.Opt[string]) { picked = append(picked, v) },
			}), widgets.SizedBoxOpts{Width: foundation.Some(400.0)}))
			ts.Tap(gluttertest.ByText("A"))
			ts.PumpAndSettle()
			if len(picked) != len(tc.want) || len(picked) > 0 && picked[0] != tc.want[0] {
				t.Errorf("%s: picked %v, want %v", tc.name, picked, tc.want)
			}
		}
	})
	t.Run("disabled without callbacks", func(t *testing.T) {
		ts := pumpTile(t, standardTheme(true), widgets.NewSizedBox(NewRadioListTile(1, RadioListTileOpts[int]{Title: widgets.NewText("A")}), widgets.SizedBoxOpts{Width: foundation.Some(400.0)}))
		ts.Tap(gluttertest.ByText("A"))
		ts.PumpAndSettle()
	})
	t.Run("affinity", func(t *testing.T) {
		for _, affinity := range []ListTileControlAffinity{ListTileControlAffinityLeading, ListTileControlAffinityTrailing, ListTileControlAffinityPlatform} {
			ts := pumpTile(t, standardTheme(true), widgets.NewSizedBox(NewRadioListTile(1, RadioListTileOpts[int]{Title: widgets.NewText("A"), ControlAffinity: affinity,
				OnChanged: func(foundation.Opt[int]) {}}), widgets.SizedBoxOpts{Width: foundation.Some(400.0)}))
			radio := ts.Center(gluttertest.ByType[*Radio[int]]()).Dx
			title := ts.Center(gluttertest.ByText("A")).Dx
			if wantLeading := affinity != ListTileControlAffinityTrailing; (radio < title) != wantLeading {
				t.Errorf("affinity %v: radio at %v, title at %v", affinity, radio, title)
			}
		}
	})
}

func TestExpansionTile(t *testing.T) {
	var changes []bool
	tile := func(opts ExpansionTileOpts) widgets.Widget {
		opts.Children = []widgets.Widget{widgets.NewText("Child A"), widgets.NewText("Child B")}
		opts.OnExpansionChanged = func(v bool) { changes = append(changes, v) }
		return widgets.NewSizedBox(NewExpansionTile(widgets.NewText("Header"), opts), widgets.SizedBoxOpts{Width: foundation.Some(400.0)})
	}
	t.Run("expand and collapse", func(t *testing.T) {
		changes = nil
		ts := pumpTile(t, standardTheme(true), tile(ExpansionTileOpts{}))
		if ts.Exists(gluttertest.ByText("Child A")) && ts.Size(gluttertest.ByType[*ExpansionTile]()).Height != 58 {
			t.Fatalf("a collapsed tile is %v high (want 58)", ts.Size(gluttertest.ByType[*ExpansionTile]()).Height)
		}
		ts.Tap(gluttertest.ByText("Header"))
		ts.PumpAndSettle()
		if !ts.Exists(gluttertest.ByText("Child A")) || ts.Size(gluttertest.ByType[*ExpansionTile]()).Height <= 58 {
			t.Fatal("the tile does not expand")
		}
		ts.Tap(gluttertest.ByText("Header"))
		ts.PumpAndSettle()
		if ts.Size(gluttertest.ByType[*ExpansionTile]()).Height != 58 {
			t.Error("the tile does not collapse")
		}
		if len(changes) != 2 || !changes[0] || changes[1] {
			t.Errorf("changes %v", changes)
		}
	})
	t.Run("initially expanded", func(t *testing.T) {
		ts := pumpTile(t, standardTheme(true), tile(ExpansionTileOpts{InitiallyExpanded: true}))
		if ts.Size(gluttertest.ByType[*ExpansionTile]()).Height <= 58 {
			t.Error("the tile is not expanded")
		}
	})
	t.Run("maintain state", func(t *testing.T) {
		for _, maintain := range []bool{false, true} {
			ts := pumpTile(t, standardTheme(true), tile(ExpansionTileOpts{MaintainState: maintain}))
			if got := ts.Exists(gluttertest.ByText("Child A", gluttertest.FindOpts{IncludeOffstage: true})); got != maintain {
				t.Errorf("maintainState=%v: the children are in the tree = %v", maintain, got)
			}
		}
	})
	t.Run("controller", func(t *testing.T) {
		controller := widgets.NewExpansibleController()
		ts := pumpTile(t, standardTheme(true), tile(ExpansionTileOpts{Controller: controller}))
		controller.Expand()
		ts.PumpAndSettle()
		if !ts.Exists(gluttertest.ByText("Child A")) || ts.Size(gluttertest.ByType[*ExpansionTile]()).Height <= 58 {
			t.Error("the controller does not expand the tile")
		}
		controller.Collapse()
		ts.PumpAndSettle()
		if ts.Size(gluttertest.ByType[*ExpansionTile]()).Height != 58 {
			t.Error("the controller does not collapse the tile")
		}
	})
	t.Run("disabled", func(t *testing.T) {
		changes = nil
		ts := pumpTile(t, standardTheme(true), tile(ExpansionTileOpts{Disabled: true}))
		ts.Tap(gluttertest.ByText("Header"))
		ts.PumpAndSettle()
		if len(changes) != 0 {
			t.Errorf("a disabled tile changed: %v", changes)
		}
	})
	t.Run("icon placement", func(t *testing.T) {
		for _, tc := range []struct {
			affinity ListTileControlAffinity
			leading  bool
		}{{ListTileControlAffinityLeading, true}, {ListTileControlAffinityTrailing, false}, {ListTileControlAffinityPlatform, false}} {
			ts := pumpTile(t, standardTheme(true), tile(ExpansionTileOpts{ControlAffinity: tc.affinity}))
			if (ts.Center(gluttertest.ByIcon(Icons.ExpandMore())).Dx < ts.Center(gluttertest.ByText("Header")).Dx) != tc.leading {
				t.Errorf("affinity %v: wrong side", tc.affinity)
			}
		}
		ts := pumpTile(t, standardTheme(true), tile(ExpansionTileOpts{NoShowTrailingIcon: true}))
		if ts.Exists(gluttertest.ByIcon(Icons.ExpandMore())) {
			t.Error("the trailing icon is shown")
		}
		ts = pumpTile(t, standardTheme(true), tile(ExpansionTileOpts{Trailing: NewIcon(Icons.Add())}))
		if !ts.Exists(gluttertest.ByIcon(Icons.Add())) || ts.Exists(gluttertest.ByIcon(Icons.ExpandMore())) {
			t.Error("the custom trailing widget is not used")
		}
	})
	t.Run("colors", func(t *testing.T) {
		th := standardTheme(true)
		collapsed, expanded := ui.NewColor(0xFF112233), ui.NewColor(0xFF445566)
		var headerColor ui.Color
		header := widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
			headerColor = widgets.DefaultTextStyleOf(bc).Style().Color.Value()
			return widgets.NewText("Header")
		})
		ts := pumpTile(t, th, widgets.NewSizedBox(NewExpansionTile(header, ExpansionTileOpts{CollapsedTextColor: foundation.Some(collapsed), TextColor: foundation.Some(expanded),
			Children: []widgets.Widget{widgets.NewText("c")}}), widgets.SizedBoxOpts{Width: foundation.Some(400.0)}))
		if headerColor != collapsed {
			t.Errorf("collapsed header color %v, want %v", headerColor, collapsed)
		}
		ts.Tap(gluttertest.ByType[*ListTile]())
		ts.PumpAndSettle()
		if headerColor != expanded {
			t.Errorf("expanded header color %v, want %v", headerColor, expanded)
		}
	})
	t.Run("theme", func(t *testing.T) {
		data := ExpansionTileThemeData{ChildrenPadding: painting.NewEdgeInsetsAll(20)}
		ts := pumpTile(t, standardTheme(true), NewExpansionTileTheme(data, tile(ExpansionTileOpts{InitiallyExpanded: true})))
		plain := pumpTile(t, standardTheme(true), tile(ExpansionTileOpts{InitiallyExpanded: true}))
		if ts.Size(gluttertest.ByType[*ExpansionTile]()).Height != plain.Size(gluttertest.ByType[*ExpansionTile]()).Height+40 {
			t.Error("the children padding of the theme is not used")
		}
	})
	t.Run("shape", func(t *testing.T) {
		ts := pumpTile(t, standardTheme(true), tile(ExpansionTileOpts{Shape: painting.NewRoundedRectangleBorder(), CollapsedShape: painting.NewRoundedRectangleBorder()}))
		if !ts.Exists(gluttertest.ByType[*Material]()) {
			t.Error("a tile with a shape is in a Material")
		}
	})
}

func TestExpansibleController(t *testing.T) {
	c := widgets.NewExpansibleController()
	notified := 0
	c.AddListener(func() { notified++ })
	c.Expand()
	c.Expand()
	c.Collapse()
	c.Toggle()
	if notified != 3 || !c.IsExpanded() {
		t.Errorf("notified %d times, expanded %v", notified, c.IsExpanded())
	}
	skipUnlessDebug(t)
	ts := pumpTile(t, standardTheme(true), widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		wantRoute(t, recoverError(func() { widgets.ExpansibleControllerOf(bc) }), "ExpansibleControllerNoAncestor")
		if widgets.ExpansibleControllerMaybeOf(bc) != nil {
			t.Error("a controller without an Expansible")
		}
		return widgets.NewSizedBox(nil)
	}))
	_ = ts
}

func TestExpandIcon(t *testing.T) {
	var pressed []bool
	for _, expanded := range []bool{false, true} {
		pressed = nil
		ts := pumpButton(t, standardTheme(true), NewExpandIcon(func(v bool) { pressed = append(pressed, v) }, ExpandIconOpts{IsExpanded: expanded}))
		ts.Tap(gluttertest.ByType[*ExpandIcon]())
		ts.PumpAndSettle()
		if len(pressed) != 1 || pressed[0] != expanded {
			t.Errorf("expanded=%v: pressed %v", expanded, pressed)
		}
	}
	// An icon without a callback ignores taps.
	ts := pumpButton(t, standardTheme(true), NewExpandIcon(nil))
	ts.Tap(gluttertest.ByType[*ExpandIcon]())
	ts.PumpAndSettle()
}

func TestExpansionPanelList(t *testing.T) {
	headers := func(label string) ExpansionPanelHeaderBuilder {
		return func(bc widgets.BuildContext, isExpanded bool) widgets.Widget { return widgets.NewText(label) }
	}
	// AnimatedCrossFade keeps both of its children in the tree, so the open
	// bodies are the ones with a height.
	open := func(ts *gluttertest.Tester, i int) bool {
		return ts.Size(gluttertest.At(gluttertest.ByType[*widgets.AnimatedCrossFade](), i)).Height > 0
	}
	t.Run("independent panels", func(t *testing.T) {
		var calls [][2]any
		expanded := []bool{false, true}
		var rebuild func()
		ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 800}})
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: standardTheme(true), Home: NewScaffold(ScaffoldOpts{Body: widgets.NewSingleChildScrollView(widgets.NewStatefulBuilder(
			func(bc widgets.BuildContext, set widgets.StateSetter) widgets.Widget {
				rebuild = func() { set(func() {}) }
				return NewExpansionPanelList([]*ExpansionPanel{
					NewExpansionPanel(headers("One"), widgets.NewText("Body one"), ExpansionPanelOpts{IsExpanded: expanded[0]}),
					NewExpansionPanel(headers("Two"), widgets.NewText("Body two"), ExpansionPanelOpts{IsExpanded: expanded[1], CanTapOnHeader: true}),
				}, ExpansionPanelListOpts{ExpansionCallback: func(i int, e bool) {
					calls = append(calls, [2]any{i, e})
					expanded[i] = e
					rebuild()
				}})
			}))})}))
		ts.PumpAndSettle()
		if open(ts, 0) || !open(ts, 1) {
			t.Fatal("the bodies are not shown as the panels say")
		}
		ts.Tap(gluttertest.First(gluttertest.ByIcon(Icons.ExpandMore())))
		ts.PumpAndSettle()
		ts.Tap(gluttertest.ByText("Two"))
		ts.PumpAndSettle()
		if len(calls) != 2 || calls[0] != [2]any{0, true} || calls[1] != [2]any{1, false} {
			t.Errorf("calls %v", calls)
		}
		if !open(ts, 0) || open(ts, 1) {
			t.Error("the panels did not follow")
		}
	})
	t.Run("radio", func(t *testing.T) {
		var calls [][2]any
		ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 800}})
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: standardTheme(true), Home: NewScaffold(ScaffoldOpts{Body: widgets.NewSingleChildScrollView(
			NewExpansionPanelListRadio([]*ExpansionPanelRadio{
				NewExpansionPanelRadio("a", headers("One"), widgets.NewText("Body one"), ExpansionPanelOpts{CanTapOnHeader: true}),
				NewExpansionPanelRadio("b", headers("Two"), widgets.NewText("Body two"), ExpansionPanelOpts{CanTapOnHeader: true}),
			}, ExpansionPanelListOpts{InitialOpenPanelValue: "a", ExpansionCallback: func(i int, e bool) { calls = append(calls, [2]any{i, e}) }}))})}))
		ts.PumpAndSettle()
		if !open(ts, 0) || open(ts, 1) {
			t.Fatal("the initial panel is not the only open one")
		}
		ts.Tap(gluttertest.ByText("Two"))
		ts.PumpAndSettle()
		if open(ts, 0) || !open(ts, 1) {
			t.Error("opening a panel does not close the other")
		}
		if len(calls) != 2 || calls[0] != [2]any{0, false} || calls[1] != [2]any{1, true} {
			t.Errorf("calls %v", calls)
		}
	})
	t.Run("errors", func(t *testing.T) {
		skipUnlessDebug(t)
		ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 800}})
		duplicate := NewExpansionPanelListRadio([]*ExpansionPanelRadio{
			NewExpansionPanelRadio("a", headers("One"), widgets.NewText("b")), NewExpansionPanelRadio("a", headers("Two"), widgets.NewText("c")),
		})
		wantRoute(t, recoverError(func() {
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: duplicate})}))
			takeAndPanic(ts)
		}), "ExpansionPanelListDuplicateValues")
		wantRoute(t, recoverError(func() {
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: NewExpansionPanelList(nil, ExpansionPanelListOpts{Elevation: foundation.Some(5.0)})})}))
			takeAndPanic(ts)
		}), "ExpansionPanelListInvalidElevation")
	})
}

func TestMergeableMaterial(t *testing.T) {
	key := func(i int) foundation.LocalKey { return foundation.NewValueKey(i) }
	slice := func(i int, h float64) MergeableMaterialItem {
		return NewMaterialSlice(key(i), sizedBox(100, h))
	}
	// A MergeableMaterial needs an unbounded main axis, like a ListBody.
	scrolled := func(items ...MergeableMaterialItem) widgets.Widget {
		return widgets.NewSingleChildScrollView(NewMergeableMaterial(items, MergeableMaterialOpts{}))
	}
	tests := []struct {
		name  string
		items []MergeableMaterialItem
		want  float64
	}{
		{"two slices", []MergeableMaterialItem{slice(1, 20), slice(2, 30)}, 50},
		{"a gap between", []MergeableMaterialItem{slice(1, 20), NewMaterialGap(key(9), 16), slice(2, 30)}, 66},
		{"a small gap", []MergeableMaterialItem{slice(1, 20), NewMaterialGap(key(9), 4), slice(2, 30)}, 54},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pumpTile(t, standardTheme(true), scrolled(tc.items...))
			if got := ts.Size(gluttertest.ByType[*MergeableMaterial]()).Height; got != tc.want {
				t.Errorf("height %v, want %v", got, tc.want)
			}
		})
	}
	t.Run("a gap opens and closes", func(t *testing.T) {
		withGap := false
		var rebuild func()
		ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: standardTheme(true), Home: NewScaffold(ScaffoldOpts{Body: widgets.NewStatefulBuilder(
			func(bc widgets.BuildContext, setState widgets.StateSetter) widgets.Widget {
				rebuild = func() { setState(func() {}) }
				if withGap {
					return scrolled(slice(1, 20), NewMaterialGap(key(9), 16), slice(2, 30))
				}
				return scrolled(slice(1, 20), slice(2, 30))
			})})}))
		ts.PumpAndSettle()
		height := func() float64 { return ts.Size(gluttertest.ByType[*MergeableMaterial]()).Height }
		if height() != 50 {
			t.Fatalf("height %v", height())
		}
		withGap = true
		rebuild()
		ts.Pump()
		ts.Pump(100 * time.Millisecond)
		if h := height(); h <= 50 || h >= 66 {
			t.Errorf("the gap is not growing: %v", h)
		}
		ts.PumpAndSettle()
		if height() != 66 {
			t.Errorf("height %v after the gap opened", height())
		}
		withGap = false
		rebuild()
		ts.PumpAndSettle()
		if height() != 50 {
			t.Errorf("height %v after the gap closed", height())
		}
	})
	t.Run("errors", func(t *testing.T) {
		skipUnlessDebug(t)
		ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
		wantRoute(t, recoverError(func() {
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: scrolled(NewMaterialGap(key(1)), slice(2, 10))})}))
			takeAndPanic(ts)
		}), "MergeableMaterialInvalidGaps")
	})
}
