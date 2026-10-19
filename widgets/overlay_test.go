package widgets_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/overlay_test.dart.
// Not ported: 'OverlayEntry dispatches memory events' and 'of method calls
// getElementForInheritedWidgetOfExactType' (they need a fake context and the
// leak tracker), 'Overlay is not visible from sub-views' (it needs ViewAnchor, step
// 14.2), 'Overlay always applies clip' (a painting matcher), and the render tree
// dump of 'Offstage overlay' (the dump format is not Flutter's: its entries are
// checked through the tree).

type overlayLog struct{ order []string }

func (l *overlayLog) entry(name string, opts ...OverlayEntryOpts) *OverlayEntry {
	return NewOverlayEntry(func(BuildContext) Widget {
		l.order = append(l.order, name)
		return NewContainer(nil)
	}, opts...)
}

func pumpOverlay(ts *gluttertest.Tester, key *GlobalKey[*OverlayState], entries ...*OverlayEntry) *OverlayState {
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{Key: key, InitialEntries: entries})))
	state, _ := key.CurrentState()
	return state
}

func TestOverlayEntryContextContainsTheOverlay(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	key := NewGlobalKey[*OverlayState]()
	built := false
	entry1 := NewOverlayEntry(func(bc BuildContext) Widget {
		built = true
		if state, ok := OverlayMaybeOf(bc); !ok || state != mustState(key) {
			t.Error("the entry does not find its overlay")
		}
		return NewContainer(nil)
	})
	entry2 := NewOverlayEntry(func(BuildContext) Widget { return NewContainer(nil) })
	pumpOverlay(ts, key, entry1, entry2)
	if !built {
		t.Error("the entry was not built")
	}
	entry1.Remove()
	entry1.Dispose()
	entry2.Remove()
	entry2.Dispose()
}

func mustState(key *GlobalKey[*OverlayState]) *OverlayState {
	s, _ := key.CurrentState()
	return s
}

func TestOverlayInsert(t *testing.T) {
	tests := []struct {
		name   string
		insert func(overlay *OverlayState, base, entry *OverlayEntry)
		want   []string
	}{
		{"top", func(o *OverlayState, _, e *OverlayEntry) { o.Insert(e) }, []string{"Base", "New"}},
		{"below", func(o *OverlayState, b, e *OverlayEntry) { o.Insert(e, OverlayInsertOpts{Below: b}) }, []string{"New", "Base"}},
		{"above", func(o *OverlayState, b, e *OverlayEntry) { o.Insert(e, OverlayInsertOpts{Above: b}) }, []string{"Base", "New"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			var log overlayLog
			key := NewGlobalKey[*OverlayState]()
			base := log.entry("Base")
			overlay := pumpOverlay(ts, key, base)
			if !slices.Equal(log.order, []string{"Base"}) {
				t.Fatalf("build order = %v", log.order)
			}
			log.order = nil
			newEntry := log.entry("New")
			tc.insert(overlay, base, newEntry)
			ts.Pump()
			if !slices.Equal(log.order, tc.want) {
				t.Errorf("build order = %v, want %v", log.order, tc.want)
			}
			base.Remove()
			base.Dispose()
			newEntry.Remove()
			newEntry.Dispose()
		})
	}
}

func TestOverlayInsertAll(t *testing.T) {
	tests := []struct {
		name   string
		insert func(overlay *OverlayState, base *OverlayEntry, entries []*OverlayEntry)
		want   []string
	}{
		{"top", func(o *OverlayState, _ *OverlayEntry, e []*OverlayEntry) { o.InsertAll(e) }, []string{"Base", "New0", "New1"}},
		{"below", func(o *OverlayState, b *OverlayEntry, e []*OverlayEntry) { o.InsertAll(e, OverlayInsertOpts{Below: b}) }, []string{"New0", "New1", "Base"}},
		{"above", func(o *OverlayState, b *OverlayEntry, e []*OverlayEntry) { o.InsertAll(e, OverlayInsertOpts{Above: b}) }, []string{"Base", "New0", "New1"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			var log overlayLog
			key := NewGlobalKey[*OverlayState]()
			base := log.entry("Base")
			overlay := pumpOverlay(ts, key, base)
			log.order = nil
			entries := []*OverlayEntry{log.entry("New0"), log.entry("New1")}
			tc.insert(overlay, base, entries)
			ts.Pump()
			if !slices.Equal(log.order, tc.want) {
				t.Errorf("build order = %v, want %v", log.order, tc.want)
			}
		})
	}
}

func TestOverlayRearrange(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	var log overlayLog
	key := NewGlobalKey[*OverlayState]()
	initial := []*OverlayEntry{log.entry("0"), log.entry("1"), log.entry("2"), log.entry("3")}
	overlay := pumpOverlay(ts, key, initial...)
	if !slices.Equal(log.order, []string{"0", "1", "2", "3"}) {
		t.Fatalf("build order = %v", log.order)
	}
	newEntry := log.entry("4")
	log.order = nil
	// Entry 1 is missing: it ends up on top.
	overlay.Rearrange([]*OverlayEntry{initial[3], newEntry, initial[2], initial[0]})
	ts.Pump()
	if !slices.Equal(log.order, []string{"3", "4", "2", "0", "1"}) {
		t.Errorf("build order = %v", log.order)
	}
}

func TestOverlayInsertPositionIsVerified(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	var log overlayLog
	key := NewGlobalKey[*OverlayState]()
	base := log.entry("base")
	overlay := pumpOverlay(ts, key, base)
	requireTag(t, "OverlayInsertPositionInvalid", func() {
		overlay.Insert(log.entry("a"), OverlayInsertOpts{Above: log.entry("b"), Below: log.entry("c")})
	})
	requireTag(t, "OverlayInsertPositionInvalid", func() { overlay.Insert(log.entry("a"), OverlayInsertOpts{Above: log.entry("b")}) })
	requireTag(t, "OverlayInsertPositionInvalid", func() {
		overlay.Rearrange([]*OverlayEntry{base}, OverlayInsertOpts{Above: log.entry("b")})
	})
	overlay.Insert(log.entry("ok"), OverlayInsertOpts{Above: base})
	ts.Pump()
}

func TestOverlayOfThrowsWithoutAnOverlay(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	var found *OverlayState
	var foundOK bool
	ts.PumpWidget(ltr(NewBuilder(func(bc BuildContext) Widget {
		found, foundOK = OverlayMaybeOf(bc)
		requireTag(t, "OverlayMissing", func() { OverlayOf(bc, OverlayOfOpts{DebugRequiredFor: NewContainer(nil)}) })
		return NewContainer(nil)
	})))
	if found != nil || foundOK {
		t.Error("an overlay was found")
	}
	key := NewGlobalKey[*OverlayState]()
	var log overlayLog
	entry := NewOverlayEntry(func(bc BuildContext) Widget {
		found, foundOK = OverlayMaybeOf(bc)
		return NewContainer(nil)
	})
	pumpOverlay(ts, key, entry)
	if !foundOK || found != mustState(key) {
		t.Error("the overlay was not found")
	}
	_ = log
}

func TestOverlayEntriesBelowOpaqueEntriesAreIgnoredForHitTesting(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	key := NewGlobalKey[*OverlayState]()
	bottom, top := 0, 0
	base := NewOverlayEntry(func(BuildContext) Widget {
		return NewGestureDetector(nil, GestureDetectorOpts{OnTap: func() { bottom++ }})
	}, OverlayEntryOpts{MaintainState: true})
	overlay := pumpOverlay(ts, key, base)
	ts.Tap(gluttertest.ByKey(key))
	if bottom != 1 {
		t.Fatalf("bottom taps = %d, want 1", bottom)
	}
	overlay.Insert(NewOverlayEntry(func(BuildContext) Widget { return NewContainer(nil) }, OverlayEntryOpts{MaintainState: true, Opaque: true}))
	ts.Pump()
	// The bottom entry is offstage and gets no taps.
	if ts.Count(gluttertest.ByType[*GestureDetector]()) != 0 || ts.Count(gluttertest.ByType[*GestureDetector](gluttertest.FindOpts{IncludeOffstage: true})) != 1 {
		t.Error("the bottom entry is not offstage")
	}
	ts.Tap(gluttertest.ByKey(key))
	if bottom != 1 {
		t.Errorf("bottom taps = %d, want 1", bottom)
	}
	overlay.Insert(NewOverlayEntry(func(BuildContext) Widget {
		return NewGestureDetector(nil, GestureDetectorOpts{OnTap: func() { top++ }})
	}, OverlayEntryOpts{MaintainState: true, Opaque: true}))
	ts.Pump()
	ts.Tap(gluttertest.ByKey(key))
	if top != 1 || bottom != 1 {
		t.Errorf("top %d bottom %d", top, bottom)
	}
}

func TestPositionedWithinAnOverlayEntry(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	key := NewGlobalKey[*OverlayState]()
	child := foundation.NewValueKey("child")
	pumpOverlay(ts, key, NewOverlayEntry(func(BuildContext) Widget {
		return NewPositioned(NewSizedBox(nil, SizedBoxOpts{Key: child, Width: foundation.Some(10.0), Height: foundation.Some(20.0)}),
			PositionedOpts{Left: foundation.Some(30.0), Top: foundation.Some(40.0)})
	}))
	if got := ts.TopLeft(gluttertest.ByKey(child)); got != (ui.Offset{Dx: 30, Dy: 40}) {
		t.Errorf("child at %v", got)
	}
}

func TestOverlayEntryInsertionErrors(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(ltr(NewOverlay()))
	overlay := gluttertest.StateOf[*OverlayState](ts, gluttertest.ByType[*Overlay]())
	entry := NewOverlayEntry(func(BuildContext) Widget { return NewSizedBox(nil) })
	overlay.Insert(entry)
	// Inserting in the same overlay again.
	requireTag(t, "OverlayEntryAlreadyInserted", func() { overlay.Insert(entry) })
	ts.PumpWidget(ltr(NewSizedBox(NewOverlay())))
	// Inserting in an overlay that was disposed.
	requireTag(t, "OverlayInsertDisposed", func() { overlay.Insert(entry) })
	// Inserting in another overlay without removing the entry.
	newOverlay := gluttertest.StateOf[*OverlayState](ts, gluttertest.ByType[*Overlay]())
	requireTag(t, "OverlayEntryInOtherOverlay", func() { newOverlay.Insert(entry) })
}

func TestOverlayEntryListenable(t *testing.T) {
	setup := func(t *testing.T) (*gluttertest.Tester, *OverlayState, *OverlayEntry, *[]bool) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		overlay := pumpOverlay(ts, NewGlobalKey[*OverlayState]())
		entry := NewOverlayEntry(func(BuildContext) Widget { return NewContainer(nil) })
		log := &[]bool{}
		entry.AddListener(func() { *log = append(*log, entry.Mounted()) })
		return ts, overlay, entry, log
	}
	t.Run("the mounted state can be listened to", func(t *testing.T) {
		ts, overlay, entry, log := setup(t)
		overlay.Insert(entry)
		if len(*log) != 0 {
			t.Fatalf("log = %v", *log)
		}
		ts.Pump() // The entry is mounted.
		if fmt.Sprint(*log) != "[true]" {
			t.Fatalf("log = %v", *log)
		}
		entry.Remove()
		if fmt.Sprint(*log) != "[true]" {
			t.Fatalf("log = %v", *log)
		}
		ts.Pump()
		overlay.Insert(entry)
		ts.Pump()
		entry.Remove()
		ts.Pump()
		if fmt.Sprint(*log) != "[true false true false]" {
			t.Errorf("log = %v", *log)
		}
		entry.Dispose()
	})
	t.Run("it throws if disposed before removal", func(t *testing.T) {
		skipUnlessDebug(t)
		_, overlay, entry, _ := setup(t)
		overlay.Insert(entry)
		requireTag(t, "OverlayEntryStillInserted", func() { entry.Dispose() })
		entry.Remove()
		entry.Dispose()
	})
	t.Run("dispose works", func(t *testing.T) {
		skipUnlessDebug(t)
		entry := NewOverlayEntry(func(BuildContext) Widget { return NewContainer(nil) })
		entry.Dispose()
		requireTag(t, "OverlayEntryDisposed", func() { entry.AddListener(func() {}) })
	})
	t.Run("delayed dispose", func(t *testing.T) {
		skipUnlessDebug(t)
		ts, overlay, entry, log := setup(t)
		overlay.Insert(entry)
		ts.Pump()
		entry.Remove()
		// The listeners are notified one last time after the dispose.
		entry.Dispose()
		if fmt.Sprint(*log) != "[true]" {
			t.Fatalf("log = %v", *log)
		}
		ts.Pump()
		if fmt.Sprint(*log) != "[true false]" {
			t.Errorf("log = %v", *log)
		}
		if err := ts.TakeError(); err != nil {
			t.Errorf("error: %v", err)
		}
		requireTag(t, "OverlayEntryDisposed", func() { entry.AddListener(func() {}) })
	})
	t.Run("it asserts when remove is called twice", func(t *testing.T) {
		skipUnlessDebug(t)
		_, overlay, entry, _ := setup(t)
		overlay.Insert(entry)
		requireTag(t, "OverlayEntryNotInserted", func() {
			entry.Remove()
			entry.Remove()
		})
		entry.Dispose()
	})
}

func TestOverlayWrap(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(ltr(NewOverlayWrap(NewCenter(NewText("Hello World")))))
	overlayState := gluttertest.StateOf[*OverlayState](ts, gluttertest.ByType[*Overlay]())
	if ts.Count(gluttertest.ByText("Hello World")) != 1 || ts.Count(gluttertest.ByText("Bye, bye")) != 0 {
		t.Fatal("the first child")
	}
	ts.PumpWidget(ltr(NewOverlayWrap(NewCenter(NewText("Bye, bye")))))
	if ts.Count(gluttertest.ByText("Hello World")) != 0 || ts.Count(gluttertest.ByText("Bye, bye")) != 1 ||
		gluttertest.StateOf[*OverlayState](ts, gluttertest.ByType[*Overlay]()) != overlayState {
		t.Error("the child was not replaced in the same overlay")
	}
}

func TestOverlayInAnUnconstrainedEnvironment(t *testing.T) {
	t.Run("Overlay.wrap is sized by its child", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		ts.PumpWidget(ltr(NewUnconstrainedBox(NewOverlayWrap(NewCenter(NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(123.0), Height: foundation.Some(456.0)}))))))
		if got := ts.Size(gluttertest.ByType[*Overlay]()); got != (ui.Size{Width: 123, Height: 456}) {
			t.Errorf("size = %v", got)
		}
	})
	errorTests := []struct {
		name  string
		entry func() *OverlayEntry
	}{
		{"has no child", func() *OverlayEntry { return nil }},
		{"only has a positioned child", func() *OverlayEntry {
			return NewOverlayEntry(func(BuildContext) Widget {
				return NewPositioned(NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(600.0), Height: foundation.Some(600.0)}), PositionedOpts{Top: foundation.Some(100.0)})
			}, OverlayEntryOpts{CanSizeOverlay: true})
		}},
		{"has no canSizeOverlay child", func() *OverlayEntry {
			return NewOverlayEntry(func(BuildContext) Widget {
				return NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(600.0), Height: foundation.Some(600.0)})
			})
		}},
	}
	for _, tc := range errorTests {
		t.Run("it throws if unconstrained and "+tc.name, func(t *testing.T) {
			skipUnlessDebug(t)
			ts := gluttertest.New(t, gluttertest.Opts{})
			var entries []*OverlayEntry
			if e := tc.entry(); e != nil {
				entries = append(entries, e)
			}
			ts.PumpWidget(ltr(NewUnconstrainedBox(NewOverlay(OverlayOpts{InitialEntries: entries}))))
			_ = requireErrorTag(t, takeFirst(t, ts), "OverlayUnbounded")
		})
	}
}

func TestOverlayDoesNotCrashAtZeroArea(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(ltr(NewCenter(NewSizedBoxShrink(NewOverlay()))))
	if ts.Count(gluttertest.ByType[*Overlay]()) != 1 {
		t.Error("no Overlay")
	}
}

// statefulProbe is StatefulTestWidget: it counts its builds.
type statefulProbe struct{ StatefulWidget }

func newStatefulProbe(key foundation.Key) *statefulProbe {
	return &statefulProbe{NewStatefulWidgetBase(WidgetOpts{Key: key})}
}

func (w *statefulProbe) CreateState() State { return &statefulProbeState{} }

type statefulProbeState struct {
	StateOf[*statefulProbe]
	rebuildCount int
}

func (s *statefulProbeState) Build(BuildContext) Widget {
	s.rebuildCount++
	return NewContainer(nil)
}

func TestOverlayOffstageEntries(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	keys := []foundation.Key{foundation.NewUniqueKey(), foundation.NewUniqueKey(), foundation.NewUniqueKey()}
	var entries []*OverlayEntry
	for _, key := range keys {
		entries = append(entries, NewOverlayEntry(func(BuildContext) Widget { return NewContainer(nil, ContainerOpts{Key: key}) },
			OverlayEntryOpts{Opaque: true, MaintainState: true}))
	}
	defer func() {
		for _, e := range entries {
			removeEntry(e)
		}
	}()
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: entries})))
	off := gluttertest.FindOpts{IncludeOffstage: true}
	for i, key := range keys {
		onstage := ts.Count(gluttertest.ByKey(key))
		all := ts.Count(gluttertest.ByKey(key, off))
		if wantOnstage := map[bool]int{true: 1, false: 0}[i == 2]; onstage != wantOnstage || all != 1 {
			t.Errorf("entry %d: %d onstage, %d built; want %d, 1", i, onstage, all, wantOnstage)
		}
	}
}

func TestOverlayMaybeOfWithAndWithoutAnOverlay(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	var found *OverlayState
	entry := NewOverlayEntry(func(bc BuildContext) Widget {
		found, _ = OverlayMaybeOf(bc)
		return NewContainer(nil)
	})
	defer removeEntry(entry)
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
	if found == nil {
		t.Fatal("no overlay is found inside one")
	}
	found = nil
	ts.PumpWidget(ltr(NewBuilder(func(bc BuildContext) Widget {
		found, _ = OverlayMaybeOf(bc)
		return NewSizedBox(nil)
	})))
	if found != nil {
		t.Error("an overlay is found outside of one")
	}
	if errs := ts.TakeErrors(); len(errs) != 0 {
		t.Errorf("errors: %v", errs)
	}
}

func TestOverlayEntryOpaqueCanBeChangedBeforeItIsInserted(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	overlayKey := NewGlobalKey[*OverlayState]()
	root, top := foundation.NewUniqueKey(), foundation.NewUniqueKey()
	rootEntry := NewOverlayEntry(func(BuildContext) Widget { return NewContainer(nil, ContainerOpts{Key: root}) })
	defer removeEntry(rootEntry)
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{Key: overlayKey, InitialEntries: []*OverlayEntry{rootEntry}})))
	if ts.Count(gluttertest.ByKey(root)) != 1 {
		t.Fatal("the root entry is not shown")
	}
	newEntry := NewOverlayEntry(func(BuildContext) Widget { return NewContainer(nil, ContainerOpts{Key: top}) })
	defer removeEntry(newEntry)
	if newEntry.Opaque() {
		t.Fatal("a new entry is opaque")
	}
	newEntry.SetOpaque(true) // Neither fails nor panics.
	if !newEntry.Opaque() {
		t.Fatal("SetOpaque did not change the entry")
	}
	// The new opaqueness is honored when the entry is inserted into an overlay.
	mustState(overlayKey).Insert(newEntry)
	ts.PumpAndSettle()
	if ts.Count(gluttertest.ByKey(root)) != 0 || ts.Count(gluttertest.ByKey(top)) != 1 {
		t.Errorf("root found %d times, top %d times; want 0, 1", ts.Count(gluttertest.ByKey(root)), ts.Count(gluttertest.ByKey(top)))
	}
}

// Regression tests for https://github.com/flutter/flutter/issues/45797.
func TestOverlayEntriesDoNotRebuildWhenOpaquenessChanges(t *testing.T) {
	for _, tc := range []struct {
		name string
		// run builds the overlay, makes the middle entry opaque and returns the
		// states of the three entries.
		run func(ts *gluttertest.Tester, overlayKey *GlobalKey[*OverlayState], bottom, middle, top *OverlayEntry)
	}{
		{"when an entry becomes opaque", func(ts *gluttertest.Tester, overlayKey *GlobalKey[*OverlayState], bottom, middle, top *OverlayEntry) {
			ts.PumpWidget(ltr(NewOverlay(OverlayOpts{Key: overlayKey, InitialEntries: []*OverlayEntry{bottom, middle, top}})))
			middle.SetOpaque(true)
			ts.Pump()
		}},
		{"when an opaque entry is added", func(ts *gluttertest.Tester, overlayKey *GlobalKey[*OverlayState], bottom, middle, top *OverlayEntry) {
			middle.SetOpaque(true)
			ts.PumpWidget(ltr(NewOverlay(OverlayOpts{Key: overlayKey, InitialEntries: []*OverlayEntry{bottom, top}})))
			mustState(overlayKey).Rearrange([]*OverlayEntry{bottom, middle, top})
			ts.Pump()
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			overlayKey := NewGlobalKey[*OverlayState]()
			bottomKey, middleKey, topKey := foundation.NewUniqueKey(), foundation.NewUniqueKey(), foundation.NewUniqueKey()
			bottomWidget, middleWidget, topWidget := newStatefulProbe(bottomKey), newStatefulProbe(middleKey), newStatefulProbe(topKey)
			keep := OverlayEntryOpts{MaintainState: true}
			bottom := NewOverlayEntry(func(BuildContext) Widget { return bottomWidget }, keep)
			middle := NewOverlayEntry(func(BuildContext) Widget { return middleWidget }, keep)
			top := NewOverlayEntry(func(BuildContext) Widget { return topWidget }, keep)
			defer func() {
				for _, e := range []*OverlayEntry{bottom, middle, top} {
					if e.Mounted() {
						removeEntry(e)
					}
				}
			}()
			tc.run(ts, overlayKey, bottom, middle, top)
			off := gluttertest.FindOpts{IncludeOffstage: true}
			if ts.Count(gluttertest.ByKey(bottomKey)) != 0 {
				t.Error("the bottom widget is onstage")
			}
			count := func(key foundation.Key, opts ...gluttertest.FindOpts) int {
				return gluttertest.StateOf[*statefulProbeState](ts, gluttertest.ByKey(key, opts...)).rebuildCount
			}
			if got := count(bottomKey, off); got != 1 {
				t.Errorf("the bottom widget was built %d times, want 1", got)
			}
			if got := count(middleKey); got != 1 {
				t.Errorf("the middle widget was built %d times, want 1", got)
			}
			if got := count(topKey); got != 1 {
				t.Errorf("the top widget was built %d times, want 1", got)
			}
		})
	}
}

func TestOverlaySemanticsOfEntriesBelowOpaqueEntriesAreIgnored(t *testing.T) {
	skipUnlessDebug(t) // the semantics nodes are read from the render objects
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.EnsureSemantics()
	bottom := NewOverlayEntry(func(BuildContext) Widget {
		return NewSemantics(NewSizedBox(nil), SemanticsOpts{Label: foundation.Some("bottom")})
	},
		OverlayEntryOpts{MaintainState: true})
	top := NewOverlayEntry(func(BuildContext) Widget {
		return NewSemantics(NewSizedBox(nil), SemanticsOpts{Label: foundation.Some("top")})
	},
		OverlayEntryOpts{MaintainState: true, Opaque: true})
	defer removeEntry(bottom)
	defer removeEntry(top)
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{bottom, top}})))
	if len(semanticNodesWith(ts, "top")) != 1 {
		t.Error("the top entry has no semantics node")
	}
	if len(semanticNodesWith(ts, "bottom")) != 0 {
		t.Error("the bottom entry has a semantics node")
	}
}

func TestOverlayClipBehaviorCanBeUpdated(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	entry := func() *OverlayEntry { return NewOverlayEntry(func(BuildContext) Widget { return NewContainer(nil) }) }
	first := NewOverlayEntry(func(BuildContext) Widget {
		return NewPositioned(NewContainer(nil), PositionedOpts{Left: foundation.Some(2000.0), Right: foundation.Some(2500.0)})
	})
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{first}})))
	theater := ts.RenderObject(gluttertest.ByType[*Overlay]())
	// By default, the clip behavior is Clip.hardEdge.
	if got := theater.DescribeApproximatePaintClip(nil); got != foundation.Some(ui.NewRectFromLTRB(0, 0, 800, 600)) {
		t.Errorf("clip by default = %v, want the overlay's rect", got)
	}
	for _, clip := range []ui.Clip{ui.ClipNone, ui.ClipHardEdge, ui.ClipAntiAlias, ui.ClipAntiAliasWithSaveLayer} {
		e := entry()
		ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{e}, ClipBehavior: clip})))
		visited := false
		theater.VisitChildren(func(child rendering.RenderObject) {
			visited = true
			got := theater.DescribeApproximatePaintClip(child)
			if clip == ui.ClipNone {
				if got.IsSome() {
					t.Errorf("clip %v: approximate clip = %v, want none", clip, got)
				}
			} else if got != foundation.Some(ui.NewRectFromLTRB(0, 0, 800, 600)) {
				t.Errorf("clip %v: approximate clip = %v, want 800x600", clip, got)
			}
		})
		if !visited {
			t.Errorf("clip %v: no child was visited", clip)
		}
	}
}

func TestOverlayLookupBoundary(t *testing.T) {
	boundaryEntry := func(inner func(bc BuildContext)) *OverlayEntry {
		return NewOverlayEntry(func(BuildContext) Widget {
			return NewLookupBoundary(NewBuilder(func(bc BuildContext) Widget {
				inner(bc)
				return NewContainer(nil)
			}))
		})
	}
	t.Run("hides the Overlay from OverlayMaybeOf", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		var overlay *OverlayState
		entry := boundaryEntry(func(bc BuildContext) { overlay, _ = OverlayMaybeOf(bc) })
		defer removeEntry(entry)
		ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
		if overlay != nil {
			t.Error("the overlay is found through the boundary")
		}
	})
	t.Run("hides the Overlay from OverlayOf", func(t *testing.T) {
		skipUnlessDebug(t)
		ts := gluttertest.New(t, gluttertest.Opts{})
		entry := boundaryEntry(func(bc BuildContext) { OverlayOf(bc) })
		defer removeEntry(entry)
		ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
		appErr := requireErrorTag(t, takeFirst(t, ts), "OverlayMissing")
		for _, want := range []string{"No Overlay widget found within the closest LookupBoundary.", "There is an ancestor Overlay widget, but it is hidden by a LookupBoundary."} {
			if !strings.Contains(appErr.Message, want) {
				t.Errorf("message %q does not contain %q", appErr.Message, want)
			}
		}
		ts.TakeErrors()
	})
	t.Run("hides the Overlay from DebugCheckHasOverlay", func(t *testing.T) {
		skipUnlessDebug(t)
		ts := gluttertest.New(t, gluttertest.Opts{})
		entry := boundaryEntry(func(bc BuildContext) { DebugCheckHasOverlay(bc) })
		defer removeEntry(entry)
		ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
		appErr := requireErrorTag(t, takeFirst(t, ts), "OverlayMissing")
		for _, want := range []string{"No Overlay widget found within the closest LookupBoundary.", "hidden by a LookupBoundary", "widgets require an Overlay widget ancestor within the closest LookupBoundary"} {
			if !strings.Contains(appErr.Message, want) {
				t.Errorf("message %q does not contain %q", appErr.Message, want)
			}
		}
		ts.TakeErrors()
	})
}

func TestOverlayIsSizedByTheEntryThatCanSizeIt(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	initialEntry := NewOverlayEntry(func(BuildContext) Widget {
		return NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(123.0), Height: foundation.Some(456.0)})
	}, OverlayEntryOpts{Opaque: true, CanSizeOverlay: true})
	defer removeEntry(initialEntry)
	ts.PumpWidget(ltr(NewUnconstrainedBox(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{initialEntry}}))))
	check := func(step string, size ui.Size, hello, world int) {
		t.Helper()
		if got := ts.Size(gluttertest.ByType[*Overlay]()); got != size {
			t.Errorf("%s: size = %v, want %v", step, got, size)
		}
		if got := ts.Count(gluttertest.ByText("Hello")); got != hello {
			t.Errorf("%s: %d Hello texts, want %d", step, got, hello)
		}
		if got := ts.Count(gluttertest.ByText("World")); got != world {
			t.Errorf("%s: %d World texts, want %d", step, got, world)
		}
	}
	check("initial", ui.Size{Width: 123, Height: 456}, 0, 0)
	overlay := gluttertest.StateOf[*OverlayState](ts, gluttertest.ByType[*Overlay]())
	nonSizingEntry := NewOverlayEntry(func(BuildContext) Widget {
		return NewSizedBox(NewCenter(NewText("Hello")), SizedBoxOpts{Width: foundation.Some(600.0), Height: foundation.Some(600.0)})
	})
	defer nonSizingEntry.Dispose()
	overlay.Insert(nonSizingEntry)
	ts.Pump()
	check("non sizing entry", ui.Size{Width: 123, Height: 456}, 1, 0)
	sizingEntry := NewOverlayEntry(func(BuildContext) Widget {
		return NewSizedBox(NewCenter(NewText("World")), SizedBoxOpts{Width: foundation.Some(222.0), Height: foundation.Some(111.0)})
	}, OverlayEntryOpts{CanSizeOverlay: true})
	defer sizingEntry.Dispose()
	overlay.Insert(sizingEntry)
	ts.Pump()
	check("sizing entry", ui.Size{Width: 222, Height: 111}, 1, 1)
	nonSizingEntry.Remove()
	ts.Pump()
	check("non sizing entry removed", ui.Size{Width: 222, Height: 111}, 0, 1)
	sizingEntry.Remove()
	ts.Pump()
	check("sizing entry removed", ui.Size{Width: 123, Height: 456}, 0, 0)
}

func TestOverlayAlwaysSizeToContent(t *testing.T) {
	t.Run("Overlay.wrap is sized by its child in a loose environment", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		ts.PumpWidget(ltr(NewUnconstrainedBox(NewConstrainedBox(rendering.NewBoxConstraintsLoose(ui.Size{Width: 800, Height: 600}),
			NewOverlayWrap(NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(123.0), Height: foundation.Some(456.0)}), OverlayWrapOpts{AlwaysSizeToContent: true})))))
		if got := ts.Size(gluttertest.ByType[*Overlay]()); got != (ui.Size{Width: 123, Height: 456}) {
			t.Errorf("size = %v, want 123x456", got)
		}
	})
	t.Run("throws without a sizing child", func(t *testing.T) {
		skipUnlessDebug(t)
		ts := gluttertest.New(t, gluttertest.Opts{})
		ts.PumpWidget(ltr(NewUnconstrainedBox(NewConstrainedBox(rendering.NewBoxConstraintsLoose(ui.Size{Width: 800, Height: 600}),
			NewOverlay(OverlayOpts{AlwaysSizeToContent: true})))))
		appErr := requireErrorTag(t, takeFirst(t, ts), "OverlaySizeToContentInvalid")
		if !strings.Contains(appErr.Message, "Overlay was asked to size itself to content but does not have a suitable child") {
			t.Errorf("message = %q", appErr.Message)
		}
		ts.TakeErrors()
	})
}
