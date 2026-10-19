package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from the ClipboardStatusNotifier group of
// packages/flutter/test/widgets/text_selection_test.dart ("update sets value based
// on clipboard contents"). The failure of the clipboard is tested with a clipboard
// backend that fails, in editable_text_internal_test.go. The listener and the
// lifecycle behaviour is Flutter's, from the source.

func TestClipboardStatusNotifierUpdateSetsTheValueFromTheClipboard(t *testing.T) {
	ts := gluttertest.New(t)
	n := NewClipboardStatusNotifier(ts.Binding)
	defer n.Dispose()
	if n.Value() != ClipboardStatusUnknown {
		t.Fatalf("value = %v, want unknown", n.Value())
	}
	n.Update()
	ts.Pump()
	if n.Value() != ClipboardStatusNotPasteable {
		t.Errorf("value = %v, want notPasteable with an empty clipboard", n.Value())
	}
	ts.SetClipboardText("pasteablestring")
	n.Update()
	ts.Pump()
	if n.Value() != ClipboardStatusPasteable {
		t.Errorf("value = %v, want pasteable", n.Value())
	}
	ts.SetClipboardText("")
	n.Update()
	ts.Pump()
	if n.Value() != ClipboardStatusNotPasteable {
		t.Errorf("value = %v, want notPasteable with an empty text", n.Value())
	}
}

func TestClipboardStatusNotifierListeners(t *testing.T) {
	ts := gluttertest.New(t)
	ts.SetClipboardText("text")
	n := NewClipboardStatusNotifier(ts.Binding)
	defer n.Dispose()
	var values []ClipboardStatus
	handle := n.AddListener(func() { values = append(values, n.Value()) })
	// The first listener finds the status unknown: the notifier asks the clipboard.
	ts.Pump()
	if len(values) != 1 || values[0] != ClipboardStatusPasteable {
		t.Fatalf("values = %v, want [pasteable]", values)
	}
	// The app resuming asks again.
	ts.ClearClipboard()
	ts.SetLifecycleState(ui.AppLifecycleStateInactive)
	ts.Pump()
	if n.Value() != ClipboardStatusPasteable {
		t.Errorf("value = %v after the app became inactive, want it unchanged", n.Value())
	}
	ts.SetLifecycleState(ui.AppLifecycleStateResumed)
	ts.Pump()
	if n.Value() != ClipboardStatusNotPasteable {
		t.Errorf("value = %v after the app resumed, want notPasteable", n.Value())
	}
	// Without listeners the notifier doesn't watch the lifecycle any more.
	n.RemoveListener(handle)
	ts.SetClipboardText("again")
	ts.SetLifecycleState(ui.AppLifecycleStateInactive)
	ts.SetLifecycleState(ui.AppLifecycleStateResumed)
	ts.Pump()
	if n.Value() != ClipboardStatusNotPasteable {
		t.Errorf("value = %v, a notifier without listeners updated", n.Value())
	}
}

func TestClipboardStatusNotifierDisposedDoesNotUpdate(t *testing.T) {
	ts := gluttertest.New(t)
	ts.SetClipboardText("text")
	n := NewClipboardStatusNotifier(ts.Binding)
	n.Update()
	n.Dispose()
	ts.Pump() // the answer of the clipboard arrives after the dispose: nothing breaks
	n.Update()
	ts.Pump()
}
