package widgets

import (
	"context"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// ClipboardStatus is the status of the content on the clipboard.
//
// Flutter: widgets/text_selection.dart ClipboardStatus
type ClipboardStatus int

// The values of ClipboardStatus follow Flutter's declaration order.
const (
	// ClipboardStatusUnspecified is the zero value: the default (unknown).
	//
	// Flutter: no counterpart
	ClipboardStatusUnspecified ClipboardStatus = iota
	// ClipboardStatusPasteable means the clipboard content can be pasted, such as
	// a string of nonzero length.
	//
	// Flutter: widgets/text_selection.dart ClipboardStatus.pasteable
	ClipboardStatusPasteable
	// ClipboardStatusUnknown means the status of the clipboard is unknown. Since
	// getting clipboard data is asynchronous, it is not known whether it can be
	// pasted.
	//
	// Flutter: widgets/text_selection.dart ClipboardStatus.unknown
	ClipboardStatusUnknown
	// ClipboardStatusNotPasteable means the clipboard content cannot be pasted,
	// such as when it is empty.
	//
	// Flutter: widgets/text_selection.dart ClipboardStatus.notPasteable
	ClipboardStatusNotPasteable
)

// ClipboardStatusNotifier notifies when the status of the clipboard changes. It
// queries the clipboard when it is asked to (Update), when the first listener
// is added while the status is unknown, and when the app resumes.
//
// Flutter: widgets/text_selection.dart ClipboardStatusNotifier
type ClipboardStatusNotifier struct {
	foundation.ValueNotifier[ClipboardStatus]
	WidgetsBindingObserverBase
	binding  *WidgetsBinding
	disposed bool
	// fixed says that the status is not asked of the clipboard: the notifier of the web.
	fixed bool
}

// NewClipboardStatusNotifier creates a notifier whose status is unknown until
// the first update.
//
// Flutter: widgets/text_selection.dart ClipboardStatusNotifier.new
func NewClipboardStatusNotifier(binding *WidgetsBinding) *ClipboardStatusNotifier {
	return &ClipboardStatusNotifier{ValueNotifier: *foundation.NewValueNotifier(ClipboardStatusUnknown), binding: binding}
}

// NewWebClipboardStatusNotifier creates the notifier that EditableText uses on the web: the status
// is always pasteable and the clipboard is never asked. Browsers show a permission dialog when the
// clipboard is read, which Update would cause before the paste button is even shown.
//
// Flutter: widgets/editable_text.dart _WebClipboardStatusNotifier
func NewWebClipboardStatusNotifier(binding *WidgetsBinding) *ClipboardStatusNotifier {
	return &ClipboardStatusNotifier{ValueNotifier: *foundation.NewValueNotifier(ClipboardStatusPasteable), binding: binding, fixed: true}
}

// Update asks the clipboard whether it has text, and sets the status when the
// answer arrives.
//
// Flutter: widgets/text_selection.dart ClipboardStatusNotifier.update
func (n *ClipboardStatusNotifier) Update() {
	if n.disposed || n.fixed {
		return
	}
	n.binding.Clipboard().HasStrings(context.Background(), func(has bool, err error) {
		if n.disposed {
			return
		}
		if err != nil {
			n.binding.ErrorReporter()(errors.NewErr("ClipboardStatusCheckFailed", err))
			// In the case of an error from the clipboard, set the value to
			// unknown so that it will try to update again later.
			n.SetValue(ClipboardStatusUnknown)
			return
		}
		if has {
			n.SetValue(ClipboardStatusPasteable)
		} else {
			n.SetValue(ClipboardStatusNotPasteable)
		}
	})
}

// AddListener adds a listener; the first one starts to watch the app lifecycle,
// and the status is updated if it is unknown.
//
// Flutter: widgets/text_selection.dart ClipboardStatusNotifier.addListener
func (n *ClipboardStatusNotifier) AddListener(listener foundation.VoidCallback) foundation.ListenerHandle {
	if !n.HasListeners() {
		n.binding.AddObserver(n)
	}
	if n.Value() == ClipboardStatusUnknown {
		n.Update()
	}
	return n.ValueNotifier.AddListener(listener)
}

// RemoveListener removes a listener; without any left, it stops watching the
// app lifecycle.
//
// Flutter: widgets/text_selection.dart ClipboardStatusNotifier.removeListener
func (n *ClipboardStatusNotifier) RemoveListener(handle foundation.ListenerHandle) {
	n.ValueNotifier.RemoveListener(handle)
	if !n.disposed && !n.HasListeners() {
		n.binding.RemoveObserver(n)
	}
}

// DidChangeAppLifecycleState updates the status when the app resumes.
//
// Flutter: widgets/text_selection.dart ClipboardStatusNotifier.didChangeAppLifecycleState
func (n *ClipboardStatusNotifier) DidChangeAppLifecycleState(state ui.AppLifecycleState) {
	if state == ui.AppLifecycleStateResumed {
		n.Update()
	}
}

// Dispose stops watching the app lifecycle.
//
// Flutter: widgets/text_selection.dart ClipboardStatusNotifier.dispose
func (n *ClipboardStatusNotifier) Dispose() {
	n.binding.RemoveObserver(n)
	n.disposed = true
	n.ValueNotifier.Dispose()
}
