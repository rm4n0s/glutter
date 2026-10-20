// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"slices"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

// Ported from packages/flutter/test/widgets/app_lifecycle_listener_test.dart
// ('Triggers correct state transition callbacks'): the table of the generated
// transitions. It calls an unexported function, so it stays in the package.

const (
	stDetached = ui.AppLifecycleStateDetached
	stResumed  = ui.AppLifecycleStateResumed
	stInactive = ui.AppLifecycleStateInactive
	stHidden   = ui.AppLifecycleStateHidden
	stPaused   = ui.AppLifecycleStatePaused
)

func TestGenerateStateTransitions(t *testing.T) {
	none := ui.AppLifecycleStateUnspecified
	tests := []struct {
		from, to ui.AppLifecycleState
		want     []ui.AppLifecycleState
	}{
		{stResumed, stResumed, nil},
		{none, stHidden, []ui.AppLifecycleState{stHidden}},
		{stDetached, stInactive, []ui.AppLifecycleState{stResumed, stInactive}},
		{stResumed, stPaused, []ui.AppLifecycleState{stInactive, stHidden, stPaused}},
		{stPaused, stResumed, []ui.AppLifecycleState{stHidden, stInactive, stResumed}},
		{stPaused, stDetached, []ui.AppLifecycleState{stDetached}},
		{stResumed, stDetached, []ui.AppLifecycleState{stInactive, stHidden, stPaused, stDetached}},
		{stDetached, stPaused, []ui.AppLifecycleState{stResumed, stInactive, stHidden, stPaused}},
		{stHidden, stInactive, []ui.AppLifecycleState{stInactive}},
	}
	for _, tc := range tests {
		got := generateStateTransitions(tc.from, tc.to)
		if !slices.Equal(got, tc.want) {
			t.Errorf("%v -> %v = %v, want %v", tc.from, tc.to, got, tc.want)
		}
	}
}
