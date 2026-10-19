package widgets_test

import (
	"sort"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/tap_region_test.dart:
// the tap down and tap up events inside and outside of the regions, the groups,
// consuming the taps outside, and changing the group of a region. Not ported: the
// right and middle clicks (same code path as the primary button here), the
// navigation tests (they need the page route tester), and the semantics tests (the
// semantics actions of the surface are SPEC §12.10).

type regionSpec struct {
	label   string
	group   any
	consume bool
	// which callbacks record: "down-out", "up-out", "down-in", "up-in".
	record map[string]bool
}

type regionLog struct{ got map[string]bool }

func (l *regionLog) take() []string {
	var out []string
	for k := range l.got {
		out = append(out, k)
	}
	sort.Strings(out)
	l.got = map[string]bool{}
	return out
}

func buildRegions(specs []regionSpec, log *regionLog, propagated *int) Widget {
	row := []Widget{}
	if propagated != nil {
		row = append(row, NewGestureDetector(NewText("Outside"), GestureDetectorOpts{OnTap: func() { *propagated++ }}))
	} else {
		row = append(row, NewText("Outside"))
	}
	for _, s := range specs {
		o := TapRegionOpts{GroupID: s.group, ConsumeOutsideTaps: s.consume}
		if s.record["down-out"] {
			o.OnTapOutside = func(*gestures.PointerDownEvent) { log.got[s.label] = true }
		}
		if s.record["up-out"] {
			o.OnTapUpOutside = func(*gestures.PointerUpEvent) { log.got[s.label] = true }
		}
		if s.record["down-in"] {
			o.OnTapInside = func(*gestures.PointerDownEvent) { log.got[s.label] = true }
		}
		if s.record["up-in"] {
			o.OnTapUpInside = func(*gestures.PointerUpEvent) { log.got[s.label] = true }
		}
		row = append(row, NewTapRegion(NewText(s.label), o))
	}
	style := painting.NewTextStyle(painting.TextStyleOpts{FontFamily: "Roboto", FontSize: foundation.Some(14.0)})
	return NewDirectionality(ui.TextDirectionLTR, NewDefaultTextStyle(style,
		NewColumn([]Widget{NewText("Outside Surface"), NewTapRegionSurface(NewRow(row))})))
}

func clickMouse(ts *gluttertest.Tester, text string, up bool) {
	g := ts.StartGesture(ts.Center(gluttertest.ByText(text)), gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse})
	if up {
		g.Up()
	} else {
		g.Cancel()
	}
}

func TestTapRegionSurfaceDetectsTapsInsideAndOutside(t *testing.T) {
	three := func(rec string, consume bool) []regionSpec {
		r := map[string]bool{rec: true}
		return []regionSpec{
			{label: "No Group", consume: consume && false, record: r},
			{label: "Group 1 A", group: 1, record: r},
			{label: "Group 1 B", group: 1, consume: consume, record: r},
		}
	}
	all := []string{"Group 1 A", "Group 1 B", "No Group"}
	tests := []struct {
		name  string
		specs []regionSpec
		up    bool
		// the labels that were recorded after the click on each text
		want map[string][]string
	}{
		{"outside tap down", three("down-out", false), false, map[string][]string{
			"No Group": {"Group 1 A", "Group 1 B"}, "Group 1 A": {"No Group"}, "Group 1 B": {"No Group"},
			"Outside": all, "Outside Surface": nil}},
		{"outside tap up", three("up-out", false), true, map[string][]string{
			"No Group": {"Group 1 A", "Group 1 B"}, "Group 1 A": {"No Group"}, "Group 1 B": {"No Group"}, "Outside": all}},
		{"inside tap down", three("down-in", false), false, map[string][]string{
			"No Group": {"No Group"}, "Group 1 A": {"Group 1 A", "Group 1 B"}, "Group 1 B": {"Group 1 A", "Group 1 B"},
			"Outside": nil, "Outside Surface": nil}},
		{"inside tap up", three("up-in", false), true, map[string][]string{
			"No Group": {"No Group"}, "Group 1 A": {"Group 1 A", "Group 1 B"}, "Group 1 B": {"Group 1 A", "Group 1 B"},
			"Outside": nil}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			log := &regionLog{got: map[string]bool{}}
			ts.PumpWidget(buildRegions(tc.specs, log, nil))
			ts.Pump()
			if got := log.take(); len(got) != 0 {
				t.Fatalf("callbacks before any tap: %v", got)
			}
			for _, text := range []string{"No Group", "Group 1 A", "Group 1 B", "Outside", "Outside Surface"} {
				want, ok := tc.want[text]
				if !ok {
					continue
				}
				clickMouse(ts, text, tc.up)
				got := log.take()
				if len(got) != len(want) {
					t.Errorf("tap on %q: %v, want %v", text, got, want)
					continue
				}
				for i := range got {
					if got[i] != want[i] {
						t.Errorf("tap on %q: %v, want %v", text, got, want)
						break
					}
				}
			}
		})
	}
}

func TestTapRegionSurfaceConsumesOutsideTapsWhenAsked(t *testing.T) {
	ts := gluttertest.New(t)
	log := &regionLog{got: map[string]bool{}}
	rec := map[string]bool{"down-out": true}
	propagated := 0
	specs := []regionSpec{
		{label: "No Group", consume: true, record: rec},
		{label: "Group 1 A", group: 1, record: rec},
		{label: "Group 1 B", group: 1, consume: true, record: rec},
	}
	ts.PumpWidget(buildRegions(specs, log, &propagated))
	ts.Pump()
	for _, text := range []string{"No Group", "Group 1 A", "Group 1 B", "Outside"} {
		clickMouse(ts, text, true)
		if propagated != 0 {
			t.Errorf("tap on %q reached the detector that is outside the consuming regions", text)
		}
		log.take()
	}
	clickMouse(ts, "Outside Surface", true)
	if got := log.take(); len(got) != 0 {
		t.Errorf("tap outside of the surface: %v", got)
	}
}

func TestTapRegionSettingTheGroupUpdatesTheRegistration(t *testing.T) {
	ts := gluttertest.New(t)
	log := &regionLog{got: map[string]bool{}}
	rec := map[string]bool{"down-out": true}
	ts.PumpWidget(buildRegions([]regionSpec{{label: "Group 1 A", group: 1, record: rec}, {label: "Group 1 B", group: 1, record: rec}}, log, nil))
	ts.Pump()
	for _, text := range []string{"Group 1 A", "Group 1 B"} {
		clickMouse(ts, text, true)
		if got := log.take(); len(got) != 0 {
			t.Errorf("tap on %q: %v, want none", text, got)
		}
	}
	clickMouse(ts, "Outside", true)
	if got := log.take(); len(got) != 2 {
		t.Errorf("tap on Outside: %v, want both", got)
	}
	// Now change the group of the second region.
	ts.PumpWidget(buildRegions([]regionSpec{{label: "Group 1 A", group: 1, record: rec}, {label: "Group 2 A", group: 2, record: rec}}, log, nil))
	clickMouse(ts, "Group 1 A", true)
	if got := log.take(); len(got) != 1 || got[0] != "Group 2 A" {
		t.Errorf("tap on Group 1 A: %v, want Group 2 A", got)
	}
	clickMouse(ts, "Group 2 A", true)
	if got := log.take(); len(got) != 1 || got[0] != "Group 1 A" {
		t.Errorf("tap on Group 2 A: %v, want Group 1 A", got)
	}
	clickMouse(ts, "Outside", true)
	if got := log.take(); len(got) != 2 {
		t.Errorf("tap on Outside: %v, want both", got)
	}
}

func TestTapRegionRegistryOfWithoutASurface(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	var bc BuildContext
	ts.PumpWidget(NewBuilder(func(c BuildContext) Widget { bc = c; return NewContainer(nil) }))
	if TapRegionRegistryMaybeOf(bc) != nil {
		t.Error("TapRegionRegistryMaybeOf found a registry without a surface")
	}
	requireTag(t, "TapRegionRegistryMissing", func() { TapRegionRegistryOf(bc) })
}
