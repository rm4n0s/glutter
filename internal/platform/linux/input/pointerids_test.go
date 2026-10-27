//go:build linux && !android

package input

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

// idsOf returns the pointer identifiers of the data, in order.
func idsOf(data []ui.PointerData) []int {
	out := make([]int, len(data))
	for i, d := range data {
		out[i] = d.PointerIdentifier
	}
	return out
}

// changesOf returns the changes of the data, in order.
func changesOf(data []ui.PointerData) []ui.PointerChange {
	out := make([]ui.PointerChange, len(data))
	for i, d := range data {
		out[i] = d.Change
	}
	return out
}

const (
	btnLeft  = 0x110
	btnRight = 0x111
)

func TestMousePointerIdentifiers(t *testing.T) {
	tests := []struct {
		name   string
		script func(b *PointerBatcher)
		// changes and ids are the changes and the identifiers of everything the script made.
		changes []ui.PointerChange
		ids     []int
	}{
		{
			name: "two presses in a row get two identifiers; the moves and the up carry their press's",
			script: func(b *PointerBatcher) {
				b.Enter(0, 1, 1)
				b.Motion(2, 2)
				b.Button(btnLeft, true)
				b.Motion(3, 3)
				b.Button(btnLeft, false)
				b.Button(btnLeft, true)
				b.Motion(4, 4)
				b.Button(btnLeft, false)
			},
			changes: []ui.PointerChange{
				ui.PointerChangeAdd, ui.PointerChangeHover,
				ui.PointerChangeDown, ui.PointerChangeMove, ui.PointerChangeUp,
				ui.PointerChangeDown, ui.PointerChangeMove, ui.PointerChangeUp,
			},
			ids: []int{0, 0, 1, 1, 1, 2, 2, 2},
		},
		{
			name: "a hover between presses carries the identifier of the last one",
			script: func(b *PointerBatcher) {
				b.Enter(0, 1, 1)
				b.Button(btnLeft, true)
				b.Button(btnLeft, false)
				b.Motion(5, 5)
				b.Button(btnLeft, true)
				b.Button(btnLeft, false)
				b.Motion(6, 6)
			},
			changes: []ui.PointerChange{
				ui.PointerChangeAdd, ui.PointerChangeDown, ui.PointerChangeUp, ui.PointerChangeHover,
				ui.PointerChangeDown, ui.PointerChangeUp, ui.PointerChangeHover,
			},
			ids: []int{0, 1, 1, 1, 2, 2, 2},
		},
		{
			name: "a second button of the same press changes the buttons with a move and keeps the identifier",
			script: func(b *PointerBatcher) {
				b.Enter(0, 1, 1)
				b.Button(btnLeft, true)
				b.Button(btnRight, true)
				b.Button(btnRight, false)
				b.Button(btnLeft, false)
			},
			changes: []ui.PointerChange{
				ui.PointerChangeAdd, ui.PointerChangeDown, ui.PointerChangeMove, ui.PointerChangeMove, ui.PointerChangeUp,
			},
			ids: []int{0, 1, 1, 1, 1},
		},
		{
			name: "leaving with a button down releases it and removes the pointer with the same identifier",
			script: func(b *PointerBatcher) {
				b.Enter(0, 1, 1)
				b.Button(btnLeft, true)
				b.Leave()
			},
			changes: []ui.PointerChange{
				ui.PointerChangeAdd, ui.PointerChangeDown, ui.PointerChangeUp, ui.PointerChangeRemove,
			},
			ids: []int{0, 1, 1, 1},
		},
		{
			name: "a pointer that is added again has had no contact yet, and its next press is new",
			script: func(b *PointerBatcher) {
				b.Enter(0, 1, 1)
				b.Button(btnLeft, true)
				b.Button(btnLeft, false)
				b.Leave()
				b.Enter(0, 2, 2)
				b.Motion(3, 3)
				b.Button(btnLeft, true)
			},
			changes: []ui.PointerChange{
				ui.PointerChangeAdd, ui.PointerChangeDown, ui.PointerChangeUp, ui.PointerChangeRemove,
				ui.PointerChangeAdd, ui.PointerChangeHover, ui.PointerChangeDown,
			},
			ids: []int{0, 1, 1, 1, 0, 0, 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewPointerBatcher(tickingClock(), nil)
			tt.script(b)
			got := b.Take()
			if c := changesOf(got); !reflect.DeepEqual(c, tt.changes) {
				t.Fatalf("changes %v, want %v", c, tt.changes)
			}
			if ids := idsOf(got); !reflect.DeepEqual(ids, tt.ids) {
				t.Errorf("identifiers %v, want %v", ids, tt.ids)
			}
		})
	}
}

func TestMouseScrollSignalCarriesTheIdentifierOfTheLastPress(t *testing.T) {
	b := NewPointerBatcher(tickingClock(), nil)
	b.Enter(0, 1, 1)
	b.Button(btnLeft, true)
	b.Button(btnLeft, false)
	b.Take()
	b.AxisSource(AxisSourceWheel)
	b.AxisValue120(AxisVertical, 120)
	got := b.Frame()
	if len(got) != 1 || got[0].SignalKind != ui.PointerSignalKindScroll || got[0].PointerIdentifier != 1 {
		t.Errorf("scroll: %+v", got)
	}
}

func TestTrackpadGesturesGetIdentifiersOfTheirOwn(t *testing.T) {
	b := NewPointerBatcher(tickingClock(), nil)
	b.Enter(0, 1, 1)
	b.Button(btnLeft, true)
	b.Button(btnLeft, false)
	b.PinchBegin()
	b.PinchUpdate(1, 1, 1.5, 0)
	b.PinchEnd()
	b.SwipeBegin()
	b.SwipeEnd()
	got := b.Take()
	wantChanges := []ui.PointerChange{
		ui.PointerChangeAdd, ui.PointerChangeDown, ui.PointerChangeUp,
		ui.PointerChangePanZoomStart, ui.PointerChangePanZoomUpdate, ui.PointerChangePanZoomEnd,
		ui.PointerChangePanZoomStart, ui.PointerChangePanZoomEnd,
	}
	if c := changesOf(got); !reflect.DeepEqual(c, wantChanges) {
		t.Fatalf("changes %v, want %v", c, wantChanges)
	}
	if ids, want := idsOf(got), []int{0, 1, 1, 2, 2, 2, 3, 3}; !reflect.DeepEqual(ids, want) {
		t.Errorf("identifiers %v, want %v", ids, want)
	}
}

func TestTouchPointerIdentifiers(t *testing.T) {
	tests := []struct {
		name    string
		script  func(b *TouchBatcher)
		changes []ui.PointerChange
		ids     []int
	}{
		{
			name: "an id that the compositor reuses gets a new identifier at each down",
			script: func(b *TouchBatcher) {
				b.Down(0, 0, 1, 1)
				b.Motion(0, 2, 2)
				b.Up(0)
				b.Down(0, 0, 3, 3)
				b.Up(0)
			},
			changes: []ui.PointerChange{
				ui.PointerChangeAdd, ui.PointerChangeDown, ui.PointerChangeMove, ui.PointerChangeUp, ui.PointerChangeRemove,
				ui.PointerChangeAdd, ui.PointerChangeDown, ui.PointerChangeUp, ui.PointerChangeRemove,
			},
			ids: []int{0, 1, 1, 1, 1, 0, 2, 2, 2},
		},
		{
			name: "two touches at once have different identifiers, and each event carries its own",
			script: func(b *TouchBatcher) {
				b.Down(0, 7, 1, 1)
				b.Down(0, 3, 50, 50)
				b.Motion(7, 2, 2)
				b.Motion(3, 49, 51)
				b.Up(7)
				b.Up(3)
			},
			changes: []ui.PointerChange{
				ui.PointerChangeAdd, ui.PointerChangeDown, ui.PointerChangeAdd, ui.PointerChangeDown,
				ui.PointerChangeMove, ui.PointerChangeMove,
				ui.PointerChangeUp, ui.PointerChangeRemove, ui.PointerChangeUp, ui.PointerChangeRemove,
			},
			ids: []int{0, 1, 0, 2, 1, 2, 1, 1, 2, 2},
		},
		{
			name: "a cancel ends the contact like an up, with the identifier of the contact",
			script: func(b *TouchBatcher) {
				b.Down(0, 4, 1, 1)
				b.Down(0, 5, 2, 2)
				b.Cancel()
				b.Down(0, 4, 3, 3)
			},
			changes: []ui.PointerChange{
				ui.PointerChangeAdd, ui.PointerChangeDown, ui.PointerChangeAdd, ui.PointerChangeDown,
				ui.PointerChangeCancel, ui.PointerChangeRemove, ui.PointerChangeCancel, ui.PointerChangeRemove,
				ui.PointerChangeAdd, ui.PointerChangeDown,
			},
			ids: []int{0, 1, 0, 2, 1, 1, 2, 2, 0, 3},
		},
		{
			name: "a down for an id that is down cancels the first contact and starts a new one",
			script: func(b *TouchBatcher) {
				b.Down(0, 4, 1, 1)
				b.Down(0, 4, 5, 5)
			},
			changes: []ui.PointerChange{
				ui.PointerChangeAdd, ui.PointerChangeDown,
				ui.PointerChangeCancel, ui.PointerChangeRemove, ui.PointerChangeAdd, ui.PointerChangeDown,
			},
			ids: []int{0, 1, 1, 1, 0, 2},
		},
		{
			name: "a window that goes away cancels its contacts with their identifiers",
			script: func(b *TouchBatcher) {
				b.Down(1, 4, 1, 1)
				b.Down(2, 5, 2, 2)
				b.CancelView(1)
			},
			changes: []ui.PointerChange{
				ui.PointerChangeAdd, ui.PointerChangeDown, ui.PointerChangeAdd, ui.PointerChangeDown,
				ui.PointerChangeCancel, ui.PointerChangeRemove,
			},
			ids: []int{0, 1, 0, 2, 1, 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewTouchBatcher(tickingClock(), nil)
			tt.script(b)
			got := b.Take()
			if c := changesOf(got); !reflect.DeepEqual(c, tt.changes) {
				t.Fatalf("changes %v, want %v", c, tt.changes)
			}
			if ids := idsOf(got); !reflect.DeepEqual(ids, tt.ids) {
				t.Errorf("identifiers %v, want %v", ids, tt.ids)
			}
		})
	}
}

func TestStylusPointerIdentifiers(t *testing.T) {
	pen := ui.PointerDeviceKindStylus
	tests := []struct {
		name    string
		script  func(b *StylusBatcher)
		changes []ui.PointerChange
		ids     []int
	}{
		{
			name: "each contact of the same tool gets a new identifier, and a hover carries the last one",
			script: func(b *StylusBatcher) {
				b.AddTool(4, pen)
				b.ProximityIn(4, 0)
				b.Motion(4, 1, 1)
				b.Frame(4)
				b.Down(4)
				b.Frame(4)
				b.Motion(4, 2, 2)
				b.Frame(4)
				b.Up(4)
				b.Frame(4)
				b.Motion(4, 3, 3)
				b.Frame(4)
				b.Down(4)
				b.Frame(4)
				b.Up(4)
				b.Frame(4)
				b.ProximityOut(4)
				b.Frame(4)
			},
			changes: []ui.PointerChange{
				ui.PointerChangeAdd, ui.PointerChangeDown, ui.PointerChangeMove, ui.PointerChangeUp,
				ui.PointerChangeHover, ui.PointerChangeDown, ui.PointerChangeUp, ui.PointerChangeRemove,
			},
			ids: []int{0, 1, 1, 1, 1, 2, 2, 2},
		},
		{
			name: "a tool that leaves while it touches is lifted and removed with the identifier of the contact",
			script: func(b *StylusBatcher) {
				b.AddTool(4, pen)
				b.ProximityIn(4, 0)
				b.Down(4)
				b.Frame(4)
				b.ProximityOut(4)
				b.Frame(4)
			},
			changes: []ui.PointerChange{ui.PointerChangeAdd, ui.PointerChangeDown, ui.PointerChangeUp, ui.PointerChangeRemove},
			ids:     []int{0, 1, 1, 1},
		},
		{
			name: "a window that goes away cancels the contact with its identifier",
			script: func(b *StylusBatcher) {
				b.AddTool(4, pen)
				b.ProximityIn(4, 3)
				b.Down(4)
				b.Frame(4)
				b.CancelView(3)
			},
			changes: []ui.PointerChange{ui.PointerChangeAdd, ui.PointerChangeDown, ui.PointerChangeCancel, ui.PointerChangeRemove},
			ids:     []int{0, 1, 1, 1},
		},
		{
			name: "a tool that comes near again has had no contact yet",
			script: func(b *StylusBatcher) {
				b.AddTool(4, pen)
				b.ProximityIn(4, 0)
				b.Down(4)
				b.Frame(4)
				b.Up(4)
				b.Frame(4)
				b.ProximityOut(4)
				b.Frame(4)
				b.ProximityIn(4, 0)
				b.Frame(4)
				b.Motion(4, 9, 9)
				b.Frame(4)
			},
			changes: []ui.PointerChange{
				ui.PointerChangeAdd, ui.PointerChangeDown, ui.PointerChangeUp, ui.PointerChangeRemove,
				ui.PointerChangeAdd, ui.PointerChangeHover,
			},
			ids: []int{0, 1, 1, 1, 0, 0},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewStylusBatcher(tickingClock(), nil)
			tt.script(b)
			got := b.Take()
			if c := changesOf(got); !reflect.DeepEqual(c, tt.changes) {
				t.Fatalf("changes %v, want %v", c, tt.changes)
			}
			if ids := idsOf(got); !reflect.DeepEqual(ids, tt.ids) {
				t.Errorf("identifiers %v, want %v", ids, tt.ids)
			}
		})
	}
}

// The batchers of one platform share a counter, so that a mouse press, a touch and a pen never
// have the same identifier in a run (Flutter's converter has one counter for all pointers).
func TestBatchersShareOneCounter(t *testing.T) {
	ids := NewPointerIDs()
	mouse := NewPointerBatcher(tickingClock(), ids)
	touch := NewTouchBatcher(tickingClock(), ids)
	pen := NewStylusBatcher(tickingClock(), ids)
	pen.AddTool(1, ui.PointerDeviceKindStylus)
	pen.ProximityIn(1, 0)
	pen.Frame(1)

	mouse.Enter(0, 1, 1)
	mouse.Button(btnLeft, true) // 1
	touch.Down(0, 0, 5, 5)      // 2
	pen.Down(1)
	pen.Frame(1) // 3
	mouse.Button(btnLeft, false)
	mouse.Button(btnLeft, true) // 4
	touch.Up(0)
	touch.Down(0, 0, 6, 6) // 5

	seen := map[int]bool{}
	for _, d := range [][]ui.PointerData{mouse.Take(), touch.Take(), pen.Take()} {
		for _, e := range d {
			if e.Change == ui.PointerChangeDown {
				if seen[e.PointerIdentifier] {
					t.Errorf("identifier %d given twice", e.PointerIdentifier)
				}
				seen[e.PointerIdentifier] = true
			}
		}
	}
	if len(seen) != 5 {
		t.Errorf("%d different identifiers, want 5: %v", len(seen), seen)
	}
	for id := 1; id <= 5; id++ {
		if !seen[id] {
			t.Errorf("identifier %d was not given: %v", id, seen)
		}
	}
}
