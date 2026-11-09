//go:build darwin && !ios

package macos

import (
	"testing"

	"github.com/rm4n0s/errors"
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

func TestClipboardFailsUntilItIsConnected(t *testing.T) {
	var c clipboard
	var got error
	c.SetText(nil, "x", func(err error) { got = err }) //nolint:staticcheck // the stub ignores the context
	appErr, ok := errors.FromError(got)
	if !ok || appErr.Tag != TagClipboardUnavailable {
		t.Errorf("SetText: %v", got)
	}
	var text string
	c.GetText(nil, func(s string, ok bool, err error) { text, got = s, err }) //nolint:staticcheck // as above
	if appErr, ok := errors.FromError(got); !ok || appErr.Tag != TagClipboardUnavailable || text != "" {
		t.Errorf("GetText: %v %q", got, text)
	}
}
