package services

import (
	"net/url"
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
)

// These tests are ported from packages/flutter/test/services/system_navigator_test.dart
// and system_sound_test.dart; the mock method handlers are the hooks.

func TestSystemNavigatorPop(t *testing.T) {
	var got []foundation.Opt[bool]
	n := &SystemNavigator{OnPop: func(animated foundation.Opt[bool]) { got = append(got, animated) }}
	n.Pop()
	n.Pop(true)
	n.Pop(false)
	want := []foundation.Opt[bool]{foundation.None[bool](), foundation.Some(true), foundation.Some(false)}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("OnPop calls = %v, want %v", got, want)
	}
}

func TestSystemNavigatorNavigationCalls(t *testing.T) {
	type update struct {
		uri     string
		state   any
		replace bool
	}
	var log []string
	var updates []update
	n := &SystemNavigator{
		OnSelectSingleEntryHistory: func() { log = append(log, "selectSingleEntryHistory") },
		OnSelectMultiEntryHistory:  func() { log = append(log, "selectMultiEntryHistory") },
		OnRouteInformationUpdated: func(u *url.URL, state any, replace bool) {
			log = append(log, "routeInformationUpdated")
			updates = append(updates, update{u.String(), state, replace})
		},
	}
	n.SelectSingleEntryHistory()
	n.SelectMultiEntryHistory()
	a := &url.URL{Path: "a"}
	n.RouteInformationUpdated(a, nil, false)
	n.RouteInformationUpdated(a, true, false)
	n.RouteInformationUpdated(a, true, true)
	wantLog := []string{
		"selectSingleEntryHistory", "selectMultiEntryHistory",
		"routeInformationUpdated", "routeInformationUpdated", "routeInformationUpdated",
	}
	if !reflect.DeepEqual(log, wantLog) {
		t.Errorf("log = %v", log)
	}
	wantUpdates := []update{{"a", nil, false}, {"a", true, false}, {"a", true, true}}
	if !reflect.DeepEqual(updates, wantUpdates) {
		t.Errorf("updates = %v, want %v", updates, wantUpdates)
	}
}

func TestSystemNavigatorSetFrameworkHandlesBack(t *testing.T) {
	tests := []struct {
		platform foundation.TargetPlatform
		want     []bool
	}{
		{foundation.TargetPlatformAndroid, []bool{true, false}},
		{foundation.TargetPlatformIOS, nil},
		{foundation.TargetPlatformMacOS, nil},
		{foundation.TargetPlatformFuchsia, nil},
		{foundation.TargetPlatformLinux, nil},
		{foundation.TargetPlatformWindows, nil},
	}
	for _, tc := range tests {
		t.Run(tc.platform.Name(), func(t *testing.T) {
			if foundation.IsWeb {
				// system_navigator.dart: the call is for Android and not for the web.
				tc.want = nil
			}
			foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(tc.platform))
			t.Cleanup(func() { foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]()) })
			var got []bool
			n := &SystemNavigator{OnSetFrameworkHandlesBack: func(v bool) { got = append(got, v) }}
			n.SetFrameworkHandlesBack(true)
			n.SetFrameworkHandlesBack(false)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("OnSetFrameworkHandlesBack calls = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSystemNavigatorWithoutHooksDoesNothing(t *testing.T) {
	var n SystemNavigator
	n.Pop()
	n.SelectSingleEntryHistory()
	n.SelectMultiEntryHistory()
	n.RouteInformationUpdated(&url.URL{Path: "a"}, nil, false)
	n.SetFrameworkHandlesBack(true)
	var nilNavigator *SystemNavigator
	nilNavigator.Pop()
	nilNavigator.SelectSingleEntryHistory()
	nilNavigator.SelectMultiEntryHistory()
	nilNavigator.RouteInformationUpdated(nil, nil, false)
	nilNavigator.SetFrameworkHandlesBack(true)
}

func TestSystemSound(t *testing.T) {
	var played []string
	s := &SystemSound{OnPlay: func(t SystemSoundType) { played = append(played, t.String()) }}
	s.Play(SystemSoundTypeClick)
	s.Play(SystemSoundTypeAlert)
	s.Play(SystemSoundTypeTick)
	want := []string{"SystemSoundType.click", "SystemSoundType.alert", "SystemSoundType.tick"}
	if !reflect.DeepEqual(played, want) {
		t.Errorf("played = %v, want %v", played, want)
	}
	if SystemSoundTypeUnspecified.String() != "SystemSoundType.unspecified" {
		t.Error("the zero value has the wrong name")
	}
	// Without a hook, or without a sound, nothing happens.
	(&SystemSound{}).Play(SystemSoundTypeClick)
	var nilSound *SystemSound
	nilSound.Play(SystemSoundTypeClick)
}
