//go:build !glutter_release && !glutter_profile

package foundation

import (
	"testing"

	"github.com/rm4n0s/errors"
)

func TestChangeNotifierDisposed(t *testing.T) {
	tests := []struct {
		name  string
		do    func(n *testNotifier)
		route string
	}{
		{"add listener", func(n *testNotifier) { n.AddListener(func() {}) }, "ChangeNotifier.AddListener->ChangeNotifier.DebugAssertNotDisposed.ChangeNotifierUsedAfterDispose"},
		{"dispose", func(n *testNotifier) { n.Dispose() }, "ChangeNotifier.Dispose->ChangeNotifier.DebugAssertNotDisposed.ChangeNotifierUsedAfterDispose"},
		{"notify", func(n *testNotifier) { n.notify() }, "ChangeNotifier.NotifyListeners->ChangeNotifier.DebugAssertNotDisposed.ChangeNotifierUsedAfterDispose"},
		{"assert", func(n *testNotifier) { n.DebugAssertNotDisposed() }, "ChangeNotifier.DebugAssertNotDisposed.ChangeNotifierUsedAfterDispose"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &testNotifier{}
			if !n.DebugAssertNotDisposed() {
				t.Fatal("DebugAssertNotDisposed on a live notifier must return true")
			}
			n.Dispose()
			err := recoverError(t, func() { tt.do(n) })
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T", err)
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
		})
	}
}

func TestChangeNotifierDisposedWhileNotifying(t *testing.T) {
	n := &testNotifier{}
	var got error
	n.AddListener(func() {
		defer func() {
			if r := recover(); r != nil {
				got = r.(error)
			}
		}()
		n.Dispose()
	})
	n.notify()
	appErr, ok := errors.FromError(got)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T", got)
	}
	if !appErr.HasRoute("ChangeNotifier.Dispose.ChangeNotifierDisposedWhileNotifying") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
}
