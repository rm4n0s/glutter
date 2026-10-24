package widgets_test

import (
	"slices"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/feedback_test.dart.
// The binding's sound and haptic services report to hooks (SystemSound.OnPlay and
// HapticFeedback.OnPerform), which stand for the mock platform channel of the Dart tests.

func TestFeedback(t *testing.T) {
	tests := []struct {
		name       string
		platform   foundation.TargetPlatform
		longPress  bool
		wantClick  int
		wantHaptic []ui.HapticFeedbackKind
	}{
		{"android forTap", foundation.TargetPlatformAndroid, false, 1, nil},
		{"android forLongPress", foundation.TargetPlatformAndroid, true, 0, []ui.HapticFeedbackKind{ui.HapticFeedbackVibrate}},
		{"fuchsia forTap", foundation.TargetPlatformFuchsia, false, 1, nil},
		{"fuchsia forLongPress", foundation.TargetPlatformFuchsia, true, 0, []ui.HapticFeedbackKind{ui.HapticFeedbackVibrate}},
		{"iOS forTap", foundation.TargetPlatformIOS, false, 0, nil},
		{"iOS forLongPress", foundation.TargetPlatformIOS, true, 1, []ui.HapticFeedbackKind{ui.HapticFeedbackHeavyImpact}},
		{"linux forTap", foundation.TargetPlatformLinux, false, 0, nil},
		{"linux forLongPress", foundation.TargetPlatformLinux, true, 0, nil},
		{"macOS forLongPress", foundation.TargetPlatformMacOS, true, 0, nil},
		{"windows forLongPress", foundation.TargetPlatformWindows, true, 0, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			skipUnlessDebug(t)
			foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(tc.platform))
			t.Cleanup(func() { foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]()) })
			ts := gluttertest.New(t)
			clicks := 0
			var bc BuildContext
			pumpContainer(ts, NewBuilder(func(c BuildContext) Widget { bc = c; return ltrText("X") }))
			WidgetsBindingOf(bc).SystemSound().OnPlay = func(s services.SystemSoundType) {
				if s == services.SystemSoundTypeClick {
					clicks++
				}
			}
			var haptics []ui.HapticFeedbackKind
			WidgetsBindingOf(bc).HapticFeedback().OnPerform = func(k ui.HapticFeedbackKind) { haptics = append(haptics, k) }
			callbacks := 0
			var wrapped foundation.VoidCallback
			if tc.longPress {
				FeedbackForLongPress(bc)
				wrapped = FeedbackWrapForLongPress(func() { callbacks++ }, bc)
			} else {
				FeedbackForTap(bc)
				wrapped = FeedbackWrapForTap(func() { callbacks++ }, bc)
			}
			if clicks != tc.wantClick {
				t.Fatalf("clicks after feedback = %d, want %d", clicks, tc.wantClick)
			}
			if !slices.Equal(haptics, tc.wantHaptic) {
				t.Fatalf("haptic feedback after feedback = %v, want %v", haptics, tc.wantHaptic)
			}
			wrapped()
			if !slices.Equal(haptics, append(slices.Clone(tc.wantHaptic), tc.wantHaptic...)) {
				t.Fatalf("haptic feedback after the wrapper = %v, want it twice", haptics)
			}
			if clicks != 2*tc.wantClick || callbacks != 1 {
				t.Fatalf("after wrapper: clicks = %d (want %d), callbacks = %d (want 1)", clicks, 2*tc.wantClick, callbacks)
			}
		})
	}
}

func TestFeedbackWrappersReturnNilForANilCallback(t *testing.T) {
	var bc BuildContext
	ts := gluttertest.New(t)
	pumpContainer(ts, NewBuilder(func(c BuildContext) Widget { bc = c; return ltrText("X") }))
	if FeedbackWrapForTap(nil, bc) != nil || FeedbackWrapForLongPress(nil, bc) != nil {
		t.Fatal("a wrapper of a nil callback is not nil")
	}
}
