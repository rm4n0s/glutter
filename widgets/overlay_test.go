package widgets_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/overlay_test.dart.
// Overlay arrives with the scroll views (reorderable lists drag in it); the rest
// of the Dart tests (OverlayPortal, LookupBoundary, sub-views, semantics, the
// render tree dump and the clip behavior) belong to step 10.1.

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
