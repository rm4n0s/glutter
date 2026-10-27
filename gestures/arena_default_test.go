package gestures

import "testing"

// Tests for the default resolution of the arena when it runs after the arena
// it was scheduled for is gone (step 29.11). Glutter's microtask is a post to
// the UI thread, so other pointer events can run before it, and a platform can
// give a new pointer the id of one that is finished.

// countingMember counts how many times it was told that it won or lost.
type countingMember struct {
	name      string
	accepts   int
	rejects   int
	onReject  func()
	onAccept  func()
	lastEntry GestureArenaEntry
}

func (m *countingMember) AcceptGesture(int) {
	m.accepts++
	if m.onAccept != nil {
		m.onAccept()
	}
}

func (m *countingMember) RejectGesture(int) {
	m.rejects++
	if m.onReject != nil {
		m.onReject()
	}
}

func (m *countingMember) String() string { return m.name }

// join adds the member to the arena of the pointer and keeps its entry.
func (m *countingMember) join(a *GestureArenaManager, pointer int) GestureArenaEntry {
	m.lastEntry = a.Add(pointer, m)
	return m.lastEntry
}

func TestArenaDefaultWinAfterTheArenaIsGone(t *testing.T) {
	const p = primaryKey
	type fixture struct {
		arena *GestureArenaManager
		micro *microtasks
		a, b  *countingMember // the members of the first arena
		c, d  *countingMember // the members of the second arena, for the same pointer
	}
	// want is {accepts, rejects} of a, b, c, d.
	type counts [4][2]int
	tests := []struct {
		name       string
		run        func(f *fixture)
		want       counts
		wantArenas int // the arenas that are still open at the end
	}{
		{"the member rejects, a new down for the same pointer, then the post runs", func(f *fixture) {
			entry := f.a.join(f.arena, p)
			f.arena.Close(p)
			entry.Resolve(GestureRejected)
			f.c.join(f.arena, p)
			f.micro.flush()
			f.arena.Close(p)
			f.micro.flush()
		}, counts{{0, 1}, {}, {1, 0}, {}}, 0},
		{"both members reject, a new down, then the post runs", func(f *fixture) {
			ea, eb := f.a.join(f.arena, p), f.b.join(f.arena, p)
			f.arena.Close(p)
			ea.Resolve(GestureRejected)
			eb.Resolve(GestureRejected)
			f.c.join(f.arena, p)
			f.micro.flush()
			f.arena.Close(p)
			f.micro.flush()
		}, counts{{0, 1}, {0, 1}, {1, 0}, {}}, 0},
		{"a sweep before the post, a new down, then the post runs", func(f *fixture) {
			f.a.join(f.arena, p)
			f.arena.Close(p)
			f.arena.Sweep(p)
			f.c.join(f.arena, p)
			f.micro.flush()
			f.arena.Close(p)
			f.micro.flush()
		}, counts{{1, 0}, {}, {1, 0}, {}}, 0},
		{"a sweep before the post, a new down that is not closed, then the post runs", func(f *fixture) {
			f.a.join(f.arena, p)
			f.arena.Close(p)
			f.arena.Sweep(p)
			f.c.join(f.arena, p)
			f.micro.flush()
			// The new arena is still open: nothing is decided for it.
		}, counts{{1, 0}, {}, {}, {}}, 1},
		{"a hold and a release, a sweep before the post", func(f *fixture) {
			f.a.join(f.arena, p)
			f.arena.Close(p)
			f.arena.Hold(p)
			f.arena.Sweep(p) // waits for the release
			f.arena.Release(p)
			f.c.join(f.arena, p)
			f.micro.flush()
			f.arena.Close(p)
			f.micro.flush()
		}, counts{{1, 0}, {}, {1, 0}, {}}, 0},
		{"the other member rejects and the last one is accepted by the post of an arena that a favor resolved", func(f *fixture) {
			ea, eb := f.a.join(f.arena, p), f.b.join(f.arena, p)
			f.arena.Close(p)
			ea.Resolve(GestureRejected) // one member left: the default win is posted
			eb.Resolve(GestureAccepted) // and it wins at once
			f.c.join(f.arena, p)
			f.micro.flush()
			f.arena.Close(p)
			f.micro.flush()
		}, counts{{0, 1}, {1, 0}, {1, 0}, {}}, 0},
		{"a pointer cancel (a sweep) of a closed arena with two members, a new down", func(f *fixture) {
			ea, eb := f.a.join(f.arena, p), f.b.join(f.arena, p)
			f.arena.Close(p)
			ea.Resolve(GestureRejected)
			f.arena.Sweep(p) // b is the only member: the cancel sweeps it
			_ = eb
			f.c.join(f.arena, p)
			f.micro.flush()
			f.arena.Close(p)
			f.micro.flush()
		}, counts{{0, 1}, {1, 0}, {1, 0}, {}}, 0},
		{"an entry of the old arena resolves in the new arena of the same pointer", func(f *fixture) {
			ea := f.a.join(f.arena, p)
			f.arena.Close(p)
			ea.Resolve(GestureRejected)
			f.c.join(f.arena, p)
			f.d.join(f.arena, p)
			f.arena.Close(p)
			ea.Resolve(GestureAccepted) // a is gone: this changes nothing
			ea.Resolve(GestureRejected)
			f.micro.flush()
			f.arena.Sweep(p)
		}, counts{{0, 1}, {}, {1, 0}, {0, 1}}, 0},
		{"a callback ends the arena and opens another while the first is resolving", func(f *fixture) {
			ea, eb := f.a.join(f.arena, p), f.b.join(f.arena, p)
			f.arena.Close(p)
			f.a.onReject = func() {
				eb.Resolve(GestureRejected) // the arena is empty now and goes away
				f.c.join(f.arena, p)        // a new arena for the same pointer
			}
			ea.Resolve(GestureRejected)
			f.micro.flush()
			f.arena.Close(p)
			f.micro.flush()
		}, counts{{0, 1}, {0, 1}, {1, 0}, {}}, 0},
		{"a callback sweeps the arena while a member is being rejected", func(f *fixture) {
			ea := f.a.join(f.arena, p)
			f.b.join(f.arena, p)
			f.arena.Close(p)
			f.a.onReject = func() {
				f.arena.Sweep(p) // b wins
				f.c.join(f.arena, p)
			}
			ea.Resolve(GestureRejected)
			f.micro.flush()
			f.arena.Close(p)
			f.micro.flush()
		}, counts{{0, 1}, {1, 0}, {1, 0}, {}}, 0},
		{"the default win still decides the arena it was posted for", func(f *fixture) {
			f.a.join(f.arena, p)
			f.arena.Close(p)
			f.micro.flush()
			f.micro.flush()
		}, counts{{1, 0}, {}, {}, {}}, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			micro := &microtasks{}
			f := &fixture{
				arena: NewGestureArenaManager(micro.schedule), micro: micro,
				a: &countingMember{name: "a"}, b: &countingMember{name: "b"},
				c: &countingMember{name: "c"}, d: &countingMember{name: "d"},
			}
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("panic: %v", r)
					}
				}()
				tc.run(f)
			}()
			var got counts
			for i, m := range []*countingMember{f.a, f.b, f.c, f.d} {
				got[i] = [2]int{m.accepts, m.rejects}
			}
			if got != tc.want {
				t.Errorf("{accepts, rejects} of a, b, c, d = %v, want %v", got, tc.want)
			}
			if n := len(f.arena.arenas); n != tc.wantArenas {
				t.Errorf("%d arenas are left, want %d", n, tc.wantArenas)
			}
		})
	}
}
