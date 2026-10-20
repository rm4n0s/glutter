// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets_test

import (
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/widgets"
	"slices"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// Ported from packages/flutter/test/widgets/app_lifecycle_listener_test.dart:
// 'Default Diagnostics', 'Diagnostics', 'listens to AppLifecycleState',
// 'Triggers correct state transition callbacks' and 'Receives exit requests'.
// The memory event test waits for the services channels.

const (
	stDetached = ui.AppLifecycleStateDetached
	stResumed  = ui.AppLifecycleStateResumed
	stInactive = ui.AppLifecycleStateInactive
	stHidden   = ui.AppLifecycleStateHidden
	stPaused   = ui.AppLifecycleStatePaused
)

// newLifecycleTree is the setUp of the Dart test: the state starts detached.
func newLifecycleTree(t *testing.T) *gluttertest.Tester {
	t.Helper()
	ts := gluttertest.New(t)
	ts.Dispatcher().SetAppLifecycleState(stPaused)
	ts.Dispatcher().SetAppLifecycleState(stDetached)
	return ts
}

func TestAppLifecycleListenerListensToTheState(t *testing.T) {
	ts := newLifecycleTree(t)
	states := []ui.AppLifecycleState{ts.Binding.Scheduler().LifecycleState()}
	l := NewAppLifecycleListener(ts.Binding, AppLifecycleListenerOpts{OnStateChange: func(s ui.AppLifecycleState) { states = append(states, s) }})
	defer l.Dispose()
	if !slices.Equal(states, []ui.AppLifecycleState{stDetached}) {
		t.Fatalf("states = %v, want detached", states)
	}
	ts.Dispatcher().SetAppLifecycleState(stInactive)
	// "resumed" is generated.
	if want := []ui.AppLifecycleState{stDetached, stResumed, stInactive}; !slices.Equal(states, want) {
		t.Errorf("states = %v, want %v", states, want)
	}
	ts.Dispatcher().SetAppLifecycleState(stResumed)
	if want := []ui.AppLifecycleState{stDetached, stResumed, stInactive, stResumed}; !slices.Equal(states, want) {
		t.Errorf("states = %v, want %v", states, want)
	}
}

func TestAppLifecycleListenerTriggersTheTransitionCallbacks(t *testing.T) {
	ts := newLifecycleTree(t)
	var got []string
	add := func(name string) func() { return func() { got = append(got, name) } }
	l := NewAppLifecycleListener(ts.Binding, AppLifecycleListenerOpts{
		OnDetach: add("detach"), OnHide: add("hide"), OnInactive: add("inactive"), OnPause: add("pause"),
		OnRestart: add("restart"), OnResume: add("resume"), OnShow: add("show"),
	})
	defer l.Dispose()
	steps := []struct {
		clear bool
		to    ui.AppLifecycleState
		want  []string
	}{
		{false, stResumed, []string{"resume"}},
		{false, stInactive, []string{"resume", "inactive"}},
		{false, stHidden, []string{"resume", "inactive", "hide"}},
		{false, stPaused, []string{"resume", "inactive", "hide", "pause"}},
		// Go back to resume.
		{true, stHidden, []string{"restart"}},
		{false, stInactive, []string{"restart", "show"}},
		{false, stResumed, []string{"restart", "show", "resume"}},
		// Generates intermediate states from lower to higher lifecycle states.
		{true, stPaused, []string{"inactive", "hide", "pause"}},
		// Wraps around from pause to detach.
		{false, stDetached, []string{"inactive", "hide", "pause", "detach"}},
		{false, stResumed, []string{"inactive", "hide", "pause", "detach", "resume"}},
		{false, stPaused, []string{"inactive", "hide", "pause", "detach", "resume", "inactive", "hide", "pause"}},
		// Generates intermediate states from higher to lower lifecycle states.
		{true, stResumed, []string{"restart", "show", "resume"}},
		// Go to detached.
		{true, stDetached, []string{"inactive", "hide", "pause", "detach"}},
	}
	for i, step := range steps {
		if step.clear {
			got = nil
		}
		ts.Dispatcher().SetAppLifecycleState(step.to)
		if !slices.Equal(got, step.want) {
			t.Fatalf("step %d (to %v): transitions = %v, want %v", i, step.to, got, step.want)
		}
	}
}

func TestAppLifecycleListenerReceivesExitRequests(t *testing.T) {
	ts := newLifecycleTree(t)
	requested := false
	l := NewAppLifecycleListener(ts.Binding, AppLifecycleListenerOpts{OnExitRequested: func() ui.AppExitResponse {
		requested = true
		return ui.AppExitResponseCancel
	}})
	defer l.Dispose()
	if got := ts.Binding.HandleRequestAppExit(); !requested || got != ui.AppExitResponseCancel {
		t.Errorf("requested = %v, response = %v; want true and cancel", requested, got)
	}
	if ts.Dispatcher().RequestClose(ui.ImplicitViewID) {
		t.Error("closing the window was allowed although OnExitRequested canceled")
	}
	plain := NewAppLifecycleListener(ts.Binding)
	defer plain.Dispose()
	if plain.DidRequestAppExit() != ui.AppExitResponseExit {
		t.Error("a listener without OnExitRequested should let the app exit")
	}
}

func TestAppLifecycleListenerDiagnostics(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	l := NewAppLifecycleListener(ts.Binding)
	defer l.Dispose()
	empty := foundation.DiagnosticableString(listenerDiagnostics{l})
	l2 := NewAppLifecycleListener(ts.Binding, AppLifecycleListenerOpts{
		OnExitRequested: func() ui.AppExitResponse { return ui.AppExitResponseCancel },
		OnStateChange:   func(ui.AppLifecycleState) {},
	})
	defer l2.Dispose()
	full := foundation.DiagnosticableString(listenerDiagnostics{l2})
	if strings.Contains(empty, "onStateChange") || strings.Contains(empty, "onExitRequested") {
		t.Errorf("default diagnostics = %q, want no flags", empty)
	}
	if !strings.Contains(full, "onStateChange") || !strings.Contains(full, "onExitRequested") {
		t.Errorf("diagnostics = %q, want onStateChange and onExitRequested", full)
	}
	if strings.Index(full, "onStateChange") > strings.Index(full, "onExitRequested") {
		t.Errorf("diagnostics = %q, want the flags in the order of Flutter", full)
	}
}

// listenerDiagnostics makes the listener a Diagnosticable.
type listenerDiagnostics struct{ *AppLifecycleListener }

func TestAppLifecycleListenerUseAfterDispose(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	l := NewAppLifecycleListener(ts.Binding)
	l.Dispose()
	if ts.Binding.RemoveObserver(l) {
		t.Error("Dispose did not remove the observer")
	}
	requireTag(t, "AppLifecycleListenerDisposed", func() { l.Dispose() })
	requireTag(t, "AppLifecycleListenerDisposed", func() { l.DidChangeAppLifecycleState(stResumed) })
}

func TestAppLifecycleListenerInvalidTransition(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	l := NewAppLifecycleListener(ts.Binding)
	defer l.Dispose()
	l.DidChangeAppLifecycleState(stResumed)
	v := panicOf(func() { l.DidChangeAppLifecycleState(stPaused) })
	_ = requireErrorTagFromPanic(t, v, "AppLifecycleInvalidTransition")
}
