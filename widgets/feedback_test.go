package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/services"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/feedback_test.dart.
// Haptic feedback is not ported (SPEC §12.10), so only the click sound counts.

func TestFeedback(t *testing.T) {
	tests := []struct {
		name      string
		platform  foundation.TargetPlatform
		longPress bool
		wantClick int
	}{
		{"android forTap", foundation.TargetPlatformAndroid, false, 1},
		{"android forLongPress", foundation.TargetPlatformAndroid, true, 0},
		{"fuchsia forTap", foundation.TargetPlatformFuchsia, false, 1},
		{"iOS forTap", foundation.TargetPlatformIOS, false, 0},
		{"iOS forLongPress", foundation.TargetPlatformIOS, true, 1},
		{"linux forTap", foundation.TargetPlatformLinux, false, 0},
		{"linux forLongPress", foundation.TargetPlatformLinux, true, 0},
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
			wrapped()
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
