package widgets

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/services"
)

// FeedbackForTap provides platform-specific feedback for a tap: it sends the tap
// semantics event, and plays the click sound on Android and Fuchsia.
//
// Deviation: haptic feedback is not ported (SPEC §12.10).
//
// Flutter: widgets/feedback.dart Feedback.forTap
func FeedbackForTap(bc BuildContext) {
	if ro := bc.FindRenderObject(); ro != nil {
		ro.SendSemanticsEvent(semantics.NewTapSemanticEvent())
	}
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformAndroid, foundation.TargetPlatformFuchsia:
		WidgetsBindingOf(bc).SystemSound().Play(services.SystemSoundTypeClick)
	}
}

// FeedbackWrapForTap returns a callback that provides the feedback for a tap
// and then calls the callback. It returns nil for a nil callback.
//
// Flutter: widgets/feedback.dart Feedback.wrapForTap
func FeedbackWrapForTap(callback foundation.VoidCallback, bc BuildContext) foundation.VoidCallback {
	if callback == nil {
		return nil
	}
	return func() {
		FeedbackForTap(bc)
		callback()
	}
}

// FeedbackForLongPress provides platform-specific feedback for a long press: it
// sends the long press semantics event, and plays the click sound on iOS.
//
// Deviation: haptic feedback is not ported (SPEC §12.10).
//
// Flutter: widgets/feedback.dart Feedback.forLongPress
func FeedbackForLongPress(bc BuildContext) {
	if ro := bc.FindRenderObject(); ro != nil {
		ro.SendSemanticsEvent(semantics.NewLongPressSemanticsEvent())
	}
	if foundation.DefaultTargetPlatform() == foundation.TargetPlatformIOS {
		WidgetsBindingOf(bc).SystemSound().Play(services.SystemSoundTypeClick)
	}
}

// FeedbackWrapForLongPress returns a callback that provides the feedback for a
// long press and then calls the callback. It returns nil for a nil callback.
//
// Flutter: widgets/feedback.dart Feedback.wrapForLongPress
func FeedbackWrapForLongPress(callback foundation.VoidCallback, bc BuildContext) foundation.VoidCallback {
	if callback == nil {
		return nil
	}
	return func() {
		FeedbackForLongPress(bc)
		callback()
	}
}
