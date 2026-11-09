//go:build darwin && !ios

package macos

import (
	"context"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

func TestNewOffTheMainThreadFails(t *testing.T) {
	// The Go test runner runs a test on a goroutine that is not on the main thread (unless the display
	// tests' TestMain moves it there), so AppKit is out of reach.
	if mainThread(t) {
		t.Skip("this test runs on the main thread")
	}
	_, err := New(Options{})
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != TagMacOSNotMainThread {
		t.Fatalf("err = %v, want %s", err, TagMacOSNotMainThread)
	}
	if !appErr.HasRoute("New." + TagMacOSNotMainThread) {
		t.Errorf("route = %s", appErr.Route())
	}
}

// mainThread says whether the calling goroutine's thread is the main thread.
func mainThread(t *testing.T) bool {
	t.Helper()
	s, err := sys()
	if err != nil {
		t.Fatal(err)
	}
	return s.pthreadMainNP() != 0
}

func TestClassesOfThePlatformCountTheirCallbacks(t *testing.T) {
	s, err := sys()
	if err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string]string{
		"app delegate": s.appDelegate.Name, "window": s.window.Name, "window delegate": s.windowDelegate.Name, "view": s.view.Name,
	} {
		if c == "" {
			t.Errorf("the %s class has no name", name)
		}
	}
}

func TestClipboardHasNoPrimarySelection(t *testing.T) {
	var c clipboard
	var setErr, getErr error
	c.SetContent(context.Background(), ui.ClipboardKindPrimary, ui.ClipboardContent{Text: "x", HasText: true}, func(err error) { setErr = err })
	c.GetContent(context.Background(), ui.ClipboardKindPrimary, ui.ClipboardFormatText, func(_ ui.ClipboardContent, err error) { getErr = err })
	for name, tc := range map[string]struct {
		err   error
		route string
	}{
		"SetContent": {setErr, "clipboard.SetContent.ClipboardPrimaryUnavailable"},
		"GetContent": {getErr, "clipboard.GetContent.ClipboardPrimaryUnavailable"},
	} {
		appErr, ok := errors.FromError(tc.err)
		if !ok || appErr.Tag != tagClipboardPrimaryUnavailable || !appErr.HasRoute(tc.route) {
			t.Errorf("%s: got %v (route %v), want %s", name, tc.err, appErr, tc.route)
		}
	}
}
