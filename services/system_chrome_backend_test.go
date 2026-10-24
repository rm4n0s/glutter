// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package services

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// fakeChromeBackend records the calls that SystemChrome, HapticFeedback and SystemSound make,
// like the mock method call handler of SystemChannels.platform in the Dart tests. Post queues the
// function, as scheduleMicrotask does, until flush runs it.
type fakeChromeBackend struct {
	log      []string
	orient   [][]ui.DeviceOrientation
	modes    []ui.SystemUIMode
	overlays [][]ui.SystemUIOverlay
	styles   []ui.SystemUIOverlayStyle
	callback func(bool)
	haptics  []ui.HapticFeedbackKind
	sounds   []ui.SystemSoundKind
	queue    []func()
}

func (f *fakeChromeBackend) SetPreferredOrientations(o []ui.DeviceOrientation) {
	f.log = append(f.log, "orientations")
	f.orient = append(f.orient, o)
}

func (f *fakeChromeBackend) SetEnabledSystemUIMode(m ui.SystemUIMode, o []ui.SystemUIOverlay) {
	f.log = append(f.log, "mode")
	f.modes = append(f.modes, m)
	f.overlays = append(f.overlays, o)
}

func (f *fakeChromeBackend) SetSystemUIOverlayStyle(s ui.SystemUIOverlayStyle) {
	f.log = append(f.log, "style")
	f.styles = append(f.styles, s)
}

func (f *fakeChromeBackend) SetSystemUIChangeCallback(fn func(bool)) {
	f.log = append(f.log, "callback")
	f.callback = fn
}

func (f *fakeChromeBackend) PerformHapticFeedback(k ui.HapticFeedbackKind) {
	f.haptics = append(f.haptics, k)
}

func (f *fakeChromeBackend) PlaySystemSound(k ui.SystemSoundKind) { f.sounds = append(f.sounds, k) }
func (f *fakeChromeBackend) Post(fn func())                       { f.queue = append(f.queue, fn) }

func (f *fakeChromeBackend) flush() {
	for len(f.queue) > 0 {
		fn := f.queue[0]
		f.queue = f.queue[1:]
		fn()
	}
}

// Ported from packages/flutter/test/services/system_chrome_test.dart ('SystemChrome overlay
// style test').
func TestSystemChromeOverlayStyleIsDebounced(t *testing.T) {
	b := &fakeChromeBackend{}
	c := NewSystemChrome(b)

	// Start with the overlay style set to dark mode.
	c.SetSystemUIOverlayStyle(SystemUiOverlayStyleDark)
	b.flush()
	b.styles, b.log = nil, nil

	// The first call is a cache miss and will queue a microtask.
	c.SetSystemUIOverlayStyle(SystemUiOverlayStyleLight)
	if len(b.queue) != 1 {
		t.Fatalf("queued %d functions, want 1", len(b.queue))
	}
	b.flush()
	light := ui.NewColor(0xFF000000)
	want := ui.SystemUIOverlayStyle{
		SystemNavigationBarColor:          &light,
		StatusBarBrightness:               ui.BrightnessDark,
		StatusBarIconBrightness:           ui.BrightnessLight,
		SystemNavigationBarIconBrightness: ui.BrightnessLight,
	}
	if !reflect.DeepEqual(b.styles, []ui.SystemUIOverlayStyle{want}) {
		t.Fatalf("sent %+v, want [%+v]", b.styles, want)
	}
	if len(b.queue) != 0 {
		t.Errorf("%d functions still queued", len(b.queue))
	}

	// The second call with the same value is a no-op.
	b.styles = nil
	c.SetSystemUIOverlayStyle(SystemUiOverlayStyleLight)
	if len(b.queue) != 0 {
		t.Fatalf("the same style queued %d functions, want 0", len(b.queue))
	}

	enforcedOff, enforcedOn := false, true
	c.SetSystemUIOverlayStyle(SystemUiOverlayStyle{
		SystemStatusBarContrastEnforced:     foundation.Some(false),
		SystemNavigationBarContrastEnforced: foundation.Some(true),
	})
	if len(b.queue) != 1 {
		t.Fatalf("a new style queued %d functions, want 1", len(b.queue))
	}
	b.flush()
	want = ui.SystemUIOverlayStyle{SystemStatusBarContrastEnforced: &enforcedOff, SystemNavigationBarContrastEnforced: &enforcedOn}
	if !reflect.DeepEqual(b.styles, []ui.SystemUIOverlayStyle{want}) {
		t.Fatalf("sent %+v, want [%+v]", b.styles, want)
	}
}

func TestSystemChromeSendsTheLastStyleOfManyCalls(t *testing.T) {
	b := &fakeChromeBackend{}
	c := NewSystemChrome(b)
	c.SetSystemUIOverlayStyle(SystemUiOverlayStyleDark)
	c.SetSystemUIOverlayStyle(SystemUiOverlayStyleLight)
	if len(b.queue) != 1 {
		t.Fatalf("two calls queued %d functions, want 1", len(b.queue))
	}
	b.flush()
	if len(b.styles) != 1 || b.styles[0].StatusBarIconBrightness != ui.BrightnessLight {
		t.Errorf("sent %+v, want the light style once", b.styles)
	}
	if got, ok := c.LatestStyle(); !ok || got != SystemUiOverlayStyleLight {
		t.Errorf("LatestStyle = %v, %v", got, ok)
	}
	// Back to the style that was sent: nothing changes in the same turn.
	c.SetSystemUIOverlayStyle(SystemUiOverlayStyleDark)
	c.SetSystemUIOverlayStyle(SystemUiOverlayStyleLight)
	b.flush()
	if len(b.styles) != 1 {
		t.Errorf("a style that was replaced before the flush was sent: %+v", b.styles)
	}
}

// Ported from system_chrome_test.dart ('SystemChrome handles detached lifecycle state').
func TestSystemChromeHandlesDetachedLifecycleState(t *testing.T) {
	b := &fakeChromeBackend{}
	c := NewSystemChrome(b)
	style := SystemUiOverlayStyle{}
	c.SetSystemUIOverlayStyle(style)
	b.flush()
	if len(b.styles) != 1 {
		t.Fatalf("sent %d styles, want 1", len(b.styles))
	}
	// Setting the same state should not send another message to the host.
	c.SetSystemUIOverlayStyle(style)
	b.flush()
	if len(b.styles) != 1 {
		t.Fatalf("sent %d styles, want 1", len(b.styles))
	}
	// The state should be sent again if the app was detached.
	c.HandleAppLifecycleStateChanged(ui.AppLifecycleStateDetached)
	b.flush()
	if _, ok := c.LatestStyle(); ok {
		t.Error("the latest style survived the detach")
	}
	c.HandleAppLifecycleStateChanged(ui.AppLifecycleStateResumed)
	c.SetSystemUIOverlayStyle(style)
	b.flush()
	if len(b.styles) != 2 {
		t.Errorf("sent %d styles, want 2", len(b.styles))
	}
}

// Ported from system_chrome_test.dart ('setPreferredOrientations control test',
// 'setEnabledSystemUIMode control test', 'setEnabledSystemUIMode passes correct overlays for
// manual configuration' and 'setSystemUIChangeCallback control test').
func TestSystemChromeCalls(t *testing.T) {
	t.Run("preferred orientations", func(t *testing.T) {
		b := &fakeChromeBackend{}
		NewSystemChrome(b).SetPreferredOrientations([]DeviceOrientation{DeviceOrientationPortraitUp, DeviceOrientationLandscapeRight})
		want := [][]ui.DeviceOrientation{{ui.DeviceOrientationPortraitUp, ui.DeviceOrientationLandscapeRight}}
		if !reflect.DeepEqual(b.orient, want) {
			t.Errorf("orientations = %v, want %v", b.orient, want)
		}
	})
	t.Run("lean back", func(t *testing.T) {
		b := &fakeChromeBackend{}
		NewSystemChrome(b).SetEnabledSystemUIMode(SystemUiModeLeanBack, nil)
		if !reflect.DeepEqual(b.modes, []ui.SystemUIMode{ui.SystemUIModeLeanBack}) || len(b.overlays[0]) != 0 {
			t.Errorf("modes = %v, overlays = %v", b.modes, b.overlays)
		}
	})
	t.Run("manual passes the overlays", func(t *testing.T) {
		b := &fakeChromeBackend{}
		NewSystemChrome(b).SetEnabledSystemUIMode(SystemUiModeManual, []SystemUiOverlay{SystemUiOverlayTop})
		if !reflect.DeepEqual(b.modes, []ui.SystemUIMode{ui.SystemUIModeManual}) ||
			!reflect.DeepEqual(b.overlays, [][]ui.SystemUIOverlay{{ui.SystemUIOverlayTop}}) {
			t.Errorf("modes = %v, overlays = %v", b.modes, b.overlays)
		}
	})
	t.Run("change callback", func(t *testing.T) {
		b := &fakeChromeBackend{}
		c := NewSystemChrome(b)
		var seen []bool
		c.SetSystemUIChangeCallback(func(visible bool) { seen = append(seen, visible) })
		if b.callback == nil {
			t.Fatal("the backend has no callback")
		}
		b.callback(false)
		b.callback(true)
		if !reflect.DeepEqual(seen, []bool{false, true}) {
			t.Errorf("callback saw %v", seen)
		}
		c.SetSystemUIChangeCallback(nil)
		if b.callback != nil {
			t.Error("a nil callback did not remove the callback")
		}
	})
}

func TestSystemChromeRestoreSystemUIOverlays(t *testing.T) {
	b := &fakeChromeBackend{}
	c := NewSystemChrome(b)
	c.RestoreSystemUIOverlays()
	if len(b.log) != 0 {
		t.Fatalf("restoring before any mode sent %v", b.log)
	}
	c.SetEnabledSystemUIMode(SystemUiModeManual, []SystemUiOverlay{SystemUiOverlayBottom})
	c.RestoreSystemUIOverlays()
	wantOverlays := [][]ui.SystemUIOverlay{{ui.SystemUIOverlayBottom}, {ui.SystemUIOverlayBottom}}
	if !reflect.DeepEqual(b.modes, []ui.SystemUIMode{ui.SystemUIModeManual, ui.SystemUIModeManual}) || !reflect.DeepEqual(b.overlays, wantOverlays) {
		t.Errorf("modes = %v, overlays = %v", b.modes, b.overlays)
	}
}

func TestSystemChromeWithoutABackendIsSafe(t *testing.T) {
	var nilChrome *SystemChrome
	for name, c := range map[string]*SystemChrome{"nil": nilChrome, "no backend": NewSystemChrome(nil)} {
		t.Run(name, func(t *testing.T) {
			c.SetPreferredOrientations([]DeviceOrientation{DeviceOrientationPortraitUp})
			c.SetEnabledSystemUIMode(SystemUiModeImmersive, nil)
			c.SetSystemUIChangeCallback(func(bool) {})
			c.SetSystemUIOverlayStyle(SystemUiOverlayStyleDark)
			c.RestoreSystemUIOverlays()
			c.HandleAppLifecycleStateChanged(ui.AppLifecycleStateDetached)
			if _, ok := c.LatestStyle(); ok {
				t.Error("a SystemChrome with no backend has a latest style")
			}
		})
	}
}

func TestSystemUiEnumNames(t *testing.T) {
	tests := []struct{ got, want string }{
		{DeviceOrientationPortraitUp.String(), "DeviceOrientation.portraitUp"},
		{DeviceOrientationLandscapeRight.String(), "DeviceOrientation.landscapeRight"},
		{SystemUiOverlayTop.String(), "SystemUiOverlay.top"},
		{SystemUiModeImmersiveSticky.String(), "SystemUiMode.immersiveSticky"},
		{SystemUiModeManual.String(), "SystemUiMode.manual"},
	}
	for _, tc := range tests {
		if tc.got != tc.want {
			t.Errorf("got %q, want %q", tc.got, tc.want)
		}
	}
}

// Ported from packages/flutter/test/services/haptic_feedback_test.dart.
func TestHapticFeedback(t *testing.T) {
	tests := []struct {
		name string
		call func(*HapticFeedback)
		want ui.HapticFeedbackKind
	}{
		{"vibrate", (*HapticFeedback).Vibrate, ui.HapticFeedbackVibrate},
		{"lightImpact", (*HapticFeedback).LightImpact, ui.HapticFeedbackLightImpact},
		{"mediumImpact", (*HapticFeedback).MediumImpact, ui.HapticFeedbackMediumImpact},
		{"heavyImpact", (*HapticFeedback).HeavyImpact, ui.HapticFeedbackHeavyImpact},
		{"selectionClick", (*HapticFeedback).SelectionClick, ui.HapticFeedbackSelectionClick},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := &fakeChromeBackend{}
			h := NewHapticFeedback(b)
			var hooked []ui.HapticFeedbackKind
			h.OnPerform = func(k ui.HapticFeedbackKind) { hooked = append(hooked, k) }
			tc.call(h)
			if !reflect.DeepEqual(b.haptics, []ui.HapticFeedbackKind{tc.want}) || !reflect.DeepEqual(hooked, b.haptics) {
				t.Errorf("backend got %v, hook got %v, want [%v]", b.haptics, hooked, tc.want)
			}
		})
	}
	t.Run("no backend", func(t *testing.T) {
		var nilHaptic *HapticFeedback
		nilHaptic.Vibrate()
		NewHapticFeedback(nil).HeavyImpact()
	})
}

// Ported from packages/flutter/test/services/system_sound_test.dart.
func TestSystemSoundReachesTheBackend(t *testing.T) {
	tests := []struct {
		in   SystemSoundType
		want []ui.SystemSoundKind
	}{
		{SystemSoundTypeClick, []ui.SystemSoundKind{ui.SystemSoundClick}},
		{SystemSoundTypeTick, []ui.SystemSoundKind{ui.SystemSoundTick}},
		{SystemSoundTypeAlert, []ui.SystemSoundKind{ui.SystemSoundAlert}},
		{SystemSoundTypeUnspecified, nil},
	}
	for _, tc := range tests {
		t.Run(tc.in.String(), func(t *testing.T) {
			b := &fakeChromeBackend{}
			s := NewSystemSound(b)
			var hooked []SystemSoundType
			s.OnPlay = func(t SystemSoundType) { hooked = append(hooked, t) }
			s.Play(tc.in)
			if !reflect.DeepEqual(b.sounds, tc.want) || !reflect.DeepEqual(hooked, []SystemSoundType{tc.in}) {
				t.Errorf("backend got %v, hook got %v, want %v", b.sounds, hooked, tc.want)
			}
		})
	}
	t.Run("zero value", func(t *testing.T) {
		var s SystemSound
		s.Play(SystemSoundTypeClick)
		var nilSound *SystemSound
		nilSound.Play(SystemSoundTypeClick)
	})
}
