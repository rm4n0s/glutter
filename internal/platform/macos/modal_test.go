package macos

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestModalStack(t *testing.T) {
	const parent, m1, m2, other ui.ViewID = 1, 2, 3, 4
	type step struct {
		do      func(s *modalStack)
		blocked map[ui.ViewID]bool
		target  map[ui.ViewID]ui.ViewID
	}
	tests := []struct {
		name  string
		steps []step
	}{
		{"open and close", []step{
			{func(s *modalStack) { s.open(m1, parent) },
				map[ui.ViewID]bool{parent: true, m1: false, other: false},
				map[ui.ViewID]ui.ViewID{parent: m1, m1: m1, other: other}},
			{func(s *modalStack) { s.close(m1) },
				map[ui.ViewID]bool{parent: false},
				map[ui.ViewID]ui.ViewID{parent: parent}},
		}},
		{"nested: the parent and the first modal are both blocked", []step{
			{func(s *modalStack) { s.open(m1, parent); s.open(m2, m1) },
				map[ui.ViewID]bool{parent: true, m1: true, m2: false},
				map[ui.ViewID]ui.ViewID{parent: m2, m1: m2, m2: m2}},
			{func(s *modalStack) { s.close(m2) },
				map[ui.ViewID]bool{parent: true, m1: false},
				map[ui.ViewID]ui.ViewID{parent: m1, m1: m1}},
			{func(s *modalStack) { s.close(m1) },
				map[ui.ViewID]bool{parent: false},
				map[ui.ViewID]ui.ViewID{parent: parent}},
		}},
		{"nested, closed in the other order: the parent is free, the inner modal is orphaned", []step{
			{func(s *modalStack) { s.open(m1, parent); s.open(m2, m1) }, nil, nil},
			{func(s *modalStack) { s.close(m1) },
				map[ui.ViewID]bool{parent: false, m1: true, m2: false},
				map[ui.ViewID]ui.ViewID{parent: parent, m2: m2}},
			{func(s *modalStack) { s.close(m2) },
				map[ui.ViewID]bool{parent: false, m1: false},
				map[ui.ViewID]ui.ViewID{parent: parent}},
		}},
		{"two modals over one window: the last opened one gets the input", []step{
			{func(s *modalStack) { s.open(m1, parent); s.open(m2, parent) },
				map[ui.ViewID]bool{parent: true},
				map[ui.ViewID]ui.ViewID{parent: m2}},
			{func(s *modalStack) { s.close(m2) },
				map[ui.ViewID]bool{parent: true},
				map[ui.ViewID]ui.ViewID{parent: m1}},
		}},
		{"opening a modal twice adds it once", []step{
			{func(s *modalStack) { s.open(m1, parent); s.open(m1, parent); s.close(m1) },
				map[ui.ViewID]bool{parent: false}, nil},
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var s modalStack
			for i, st := range tc.steps {
				st.do(&s)
				for v, want := range st.blocked {
					if got := s.blocked(v); got != want {
						t.Errorf("step %d: blocked(%d) = %v, want %v", i, v, got, want)
					}
				}
				for v, want := range st.target {
					if got := s.target(v); got != want {
						t.Errorf("step %d: target(%d) = %d, want %d", i, v, got, want)
					}
				}
			}
		})
	}
}

func TestModalStackClose(t *testing.T) {
	var s modalStack
	if _, ok := s.close(7); ok {
		t.Error("closing a window that is not modal found a parent")
	}
	s.open(7, 1)
	if !s.any() {
		t.Error("any is false with a modal window open")
	}
	if parent, ok := s.close(7); !ok || parent != 1 {
		t.Errorf("close = %d, %v, want 1, true", parent, ok)
	}
	if s.any() {
		t.Error("any is true after the modal window closed")
	}
	if _, ok := s.close(7); ok {
		t.Error("a modal window closed twice")
	}
}
