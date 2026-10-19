package gestures

import "testing"

// Ported from packages/flutter/test/gestures/arena_test.dart.

const primaryKey = 4

type testGestureArenaMember struct {
	t                    *testing.T
	key                  int // the pointer the member expects to be called for
	acceptRan, rejectRan bool
}

func (m *testGestureArenaMember) AcceptGesture(key int) {
	m.t.Helper()
	if key != m.key {
		m.t.Errorf("AcceptGesture(%d), want %d", key, m.key)
	}
	m.acceptRan = true
}

func (m *testGestureArenaMember) RejectGesture(key int) {
	m.t.Helper()
	if key != m.key {
		m.t.Errorf("RejectGesture(%d), want %d", key, m.key)
	}
	m.rejectRan = true
}

type arenaTester struct {
	t                       *testing.T
	micro                   *microtasks
	arena                   *GestureArenaManager
	first, second           *testGestureArenaMember
	firstEntry, secondEntry GestureArenaEntry
}

func newArenaTester(t *testing.T) *arenaTester {
	micro := &microtasks{}
	return &arenaTester{
		t: t, micro: micro, arena: NewGestureArenaManager(micro.schedule),
		first: &testGestureArenaMember{t: t, key: primaryKey}, second: &testGestureArenaMember{t: t, key: primaryKey},
	}
}

func (a *arenaTester) addFirst()  { a.firstEntry = a.arena.Add(primaryKey, a.first) }
func (a *arenaTester) addSecond() { a.secondEntry = a.arena.Add(primaryKey, a.second) }

func (a *arenaTester) expect(firstAccept, firstReject, secondAccept, secondReject bool) {
	a.t.Helper()
	got := [4]bool{a.first.acceptRan, a.first.rejectRan, a.second.acceptRan, a.second.rejectRan}
	want := [4]bool{firstAccept, firstReject, secondAccept, secondReject}
	if got != want {
		a.t.Errorf("first accept/reject, second accept/reject = %v, want %v", got, want)
	}
}
func (a *arenaTester) expectNothing()   { a.t.Helper(); a.expect(false, false, false, false) }
func (a *arenaTester) expectFirstWin()  { a.t.Helper(); a.expect(true, false, false, true) }
func (a *arenaTester) expectSecondWin() { a.t.Helper(); a.expect(false, true, true, false) }

func TestArena(t *testing.T) {
	tests := []struct {
		name string
		run  func(a *arenaTester)
	}{
		{"Should win by accepting", func(a *arenaTester) {
			a.addFirst()
			a.addSecond()
			a.arena.Close(primaryKey)
			a.expectNothing()
			a.firstEntry.Resolve(GestureAccepted)
			a.expectFirstWin()
		}},
		{"Should win by sweep", func(a *arenaTester) {
			a.addFirst()
			a.addSecond()
			a.arena.Close(primaryKey)
			a.expectNothing()
			a.arena.Sweep(primaryKey)
			a.expectFirstWin()
		}},
		{"Should win on release after hold sweep release", func(a *arenaTester) {
			a.addFirst()
			a.addSecond()
			a.arena.Close(primaryKey)
			a.expectNothing()
			a.arena.Hold(primaryKey)
			a.expectNothing()
			a.arena.Sweep(primaryKey)
			a.expectNothing()
			a.arena.Release(primaryKey)
			a.expectFirstWin()
		}},
		{"Should win on sweep after hold release sweep", func(a *arenaTester) {
			a.addFirst()
			a.addSecond()
			a.arena.Close(primaryKey)
			a.expectNothing()
			a.arena.Hold(primaryKey)
			a.expectNothing()
			a.arena.Release(primaryKey)
			a.expectNothing()
			a.arena.Sweep(primaryKey)
			a.expectFirstWin()
		}},
		{"Only first winner should win", func(a *arenaTester) {
			a.addFirst()
			a.addSecond()
			a.arena.Close(primaryKey)
			a.expectNothing()
			a.firstEntry.Resolve(GestureAccepted)
			a.secondEntry.Resolve(GestureAccepted)
			a.expectFirstWin()
		}},
		{"Only first winner should win, regardless of order", func(a *arenaTester) {
			a.addFirst()
			a.addSecond()
			a.arena.Close(primaryKey)
			a.expectNothing()
			a.secondEntry.Resolve(GestureAccepted)
			a.firstEntry.Resolve(GestureAccepted)
			a.expectSecondWin()
		}},
		{"Win before close is delayed to close", func(a *arenaTester) {
			a.addFirst()
			a.addSecond()
			a.expectNothing()
			a.firstEntry.Resolve(GestureAccepted)
			a.expectNothing()
			a.arena.Close(primaryKey)
			a.expectFirstWin()
		}},
		{"Win before close is delayed to close, and only first winner should win", func(a *arenaTester) {
			a.addFirst()
			a.addSecond()
			a.expectNothing()
			a.firstEntry.Resolve(GestureAccepted)
			a.secondEntry.Resolve(GestureAccepted)
			a.expectNothing()
			a.arena.Close(primaryKey)
			a.expectFirstWin()
		}},
		{"Win before close is delayed to close, and only first winner should win, regardless of order", func(a *arenaTester) {
			a.addFirst()
			a.addSecond()
			a.expectNothing()
			a.secondEntry.Resolve(GestureAccepted)
			a.firstEntry.Resolve(GestureAccepted)
			a.expectNothing()
			a.arena.Close(primaryKey)
			a.expectSecondWin()
		}},
		// Cases of arena.dart that its own test file does not cover.
		{"A lone member wins by default, in a microtask", func(a *arenaTester) {
			a.addFirst()
			a.arena.Close(primaryKey)
			a.expectNothing()
			a.micro.flush()
			a.expect(true, false, false, false)
		}},
		{"The last member wins when the other rejects after close", func(a *arenaTester) {
			a.addFirst()
			a.addSecond()
			a.arena.Close(primaryKey)
			a.firstEntry.Resolve(GestureRejected)
			a.micro.flush()
			a.expect(false, true, true, false)
		}},
		{"An arena with no members is dropped", func(a *arenaTester) {
			a.addFirst()
			a.firstEntry.Resolve(GestureRejected)
			a.arena.Close(primaryKey)
			a.arena.Sweep(primaryKey) // no panic: the arena is gone
			a.expect(false, true, false, false)
		}},
		{"Resolving after the arena resolved does nothing", func(a *arenaTester) {
			a.addFirst()
			a.addSecond()
			a.arena.Close(primaryKey)
			a.arena.Sweep(primaryKey)
			a.firstEntry.Resolve(GestureRejected)
			a.expectFirstWin()
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) { tc.run(newArenaTester(t)) })
	}
}
