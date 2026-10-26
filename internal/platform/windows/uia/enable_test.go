package uia

import "testing"

func TestEnableStateMachine(t *testing.T) {
	var e Enable
	steps := []struct {
		name       string
		do         func() (answer, turnedOn bool)
		wantAnswer bool
		wantOn     bool
		wantState  bool
	}{
		{"a client asks before the bridge has a host", func() (bool, bool) { return e.Ask(RootObjectID) }, false, false, false},
		{"the host arrives", func() (bool, bool) { e.Started(); return false, false }, false, false, false},
		{"a request for another object (MSAA) is not ours", func() (bool, bool) { return e.Ask(-4) }, false, false, false},
		{"the first ask turns it on", func() (bool, bool) { return e.Ask(RootObjectID) }, true, true, true},
		{"the second is answered and turns nothing on", func() (bool, bool) { return e.Ask(RootObjectID) }, true, false, true},
		{"another object id still is not ours", func() (bool, bool) { return e.Ask(0) }, false, false, true},
	}
	for _, s := range steps {
		answer, on := s.do()
		if answer != s.wantAnswer || on != s.wantOn || e.On() != s.wantState {
			t.Errorf("%s: answer %v, turned on %v, state %v; want %v, %v, %v", s.name, answer, on, e.On(), s.wantAnswer, s.wantOn, s.wantState)
		}
	}
	if !e.Close() {
		t.Error("Close of a bridge that was on says it was off")
	}
	if answer, on := e.Ask(RootObjectID); answer || on || e.On() {
		t.Errorf("after Close: answer %v, turned on %v, on %v; want none", answer, on, e.On())
	}
	if e.Close() {
		t.Error("the second Close says the bridge was on")
	}
	var fresh Enable
	fresh.Started()
	if fresh.Close() {
		t.Error("Close of a bridge that never was on says it was on")
	}
}
