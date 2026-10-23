package web

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

// clipboardError runs a call of the clipboard that fails, delivers what it posted, and returns
// the error it answered with.
func clipboardError(t *testing.T, c clipboard, p *posts, call func(c clipboard, done func(error))) error {
	t.Helper()
	var err error
	answered := false
	call(c, func(e error) { err, answered = e, true })
	p.run()
	if !answered {
		t.Fatal("the call was not answered")
	}
	return err
}

// TestClipboardErrorRoutes: each failure of the clipboard is made by the function that fails, so the
// route ends in that function.
func TestClipboardErrorRoutes(t *testing.T) {
	ended, cancel := context.WithCancel(context.Background())
	cancel()
	text := ui.ClipboardContent{Text: "x", HasText: true}
	tests := []struct {
		name  string
		api   clipboardAPI
		call  func(c clipboard, done func(error))
		route string
	}{
		{"set the primary selection", &fakeAPI{}, func(c clipboard, done func(error)) {
			c.SetContent(context.Background(), ui.ClipboardKindPrimary, text, done)
		}, "clipboard.SetContent.ClipboardPrimaryUnavailable"},
		{"get the primary selection", &fakeAPI{}, func(c clipboard, done func(error)) {
			c.GetContent(context.Background(), ui.ClipboardKindPrimary, ui.ClipboardFormatText, func(_ ui.ClipboardContent, err error) { done(err) })
		}, "clipboard.GetContent.ClipboardPrimaryUnavailable"},
		{"set an image", &fakeAPI{}, func(c clipboard, done func(error)) {
			c.SetContent(context.Background(), ui.ClipboardKindSystem, ui.ClipboardContent{PNG: []byte{1}}, done)
		}, "clipboard.SetContent.WebClipboardFormatUnsupported"},
		{"write without the API", nil, func(c clipboard, done func(error)) {
			c.SetText(context.Background(), "x", done)
		}, "clipboard.SetText->clipboard.write.WebClipboardUnavailable"},
		{"read without the API", nil, func(c clipboard, done func(error)) {
			c.GetText(context.Background(), func(_ string, _ bool, err error) { done(err) })
		}, "clipboard.GetText->clipboard.read.WebClipboardUnavailable"},
		{"write with an ended context", &fakeAPI{}, func(c clipboard, done func(error)) {
			c.SetText(ended, "x", done)
		}, "clipboard.SetText->clipboard.write.WebClipboardWriteFailed"},
		{"read with an ended context", &fakeAPI{}, func(c clipboard, done func(error)) {
			c.GetText(ended, func(_ string, _ bool, err error) { done(err) })
		}, "clipboard.GetText->clipboard.read.ClipboardReadTimeout"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var p posts
			c := clipboard{post: p.post}
			if tc.api != nil {
				c.api = tc.api
			}
			err := clipboardError(t, c, &p, tc.call)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T: %v", err, err)
			}
			if !appErr.HasRoute(tc.route) {
				t.Errorf("unexpected failure path: %s, want %s", appErr.Route(), tc.route)
			}
		})
	}
}

// TestClipboardReadEndedByItsContext: a read whose context ends while the browser waits fails with
// ClipboardReadTimeout, made by the function that the context calls.
func TestClipboardReadEndedByItsContext(t *testing.T) {
	var p posts
	c := clipboard{post: p.post, api: &fakeAPI{}}
	ctx, cancel := context.WithCancel(context.Background())
	got := make(chan error, 1)
	c.GetText(ctx, func(_ string, _ bool, err error) { got <- err })
	cancel()
	for p.count() == 0 {
		// The context posts from a goroutine of its own.
		time.Sleep(time.Millisecond)
	}
	p.run()
	appErr, ok := errors.FromError(<-got)
	if !ok {
		t.Fatal("expected *errors.Error")
	}
	// The error is made in the function that the context calls, on its own goroutine.
	if !strings.Contains(appErr.Route(), "clipboard.read.func") || !strings.HasSuffix(appErr.Route(), ".ClipboardReadTimeout") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
}

// TestDesktopErrorRoutes: the failures of OpenURI and of the notifications are made by the function
// that fails.
func TestDesktopErrorRoutes(t *testing.T) {
	ended, cancel := context.WithCancel(context.Background())
	cancel()
	show := func(d *Desktop) error {
		_, err := d.ShowNotification(context.Background(), NotificationSpec{Title: "Hi"})
		return err
	}
	tests := []struct {
		name  string
		setup func() (*Desktop, func(*Desktop) error)
		route string
		meta  string
	}{
		{"script URI", func() (*Desktop, func(*Desktop) error) {
			d, _, _, _ := newTestDesktop("granted")
			return d, func(d *Desktop) error { return d.OpenURI(context.Background(), "javascript:alert(1)") }
		}, "Desktop.OpenURI.OpenURISchemeBlocked", "scheme"},
		{"URI with an ended context", func() (*Desktop, func(*Desktop) error) {
			d, _, _, _ := newTestDesktop("granted")
			return d, func(d *Desktop) error { return d.OpenURI(ended, "https://example.org") }
		}, "Desktop.OpenURI.OpenURIUnavailable", "reason"},
		{"URI without a page", func() (*Desktop, func(*Desktop) error) {
			return NewDesktop(), func(d *Desktop) error { return d.OpenURI(context.Background(), "https://example.org") }
		}, "Desktop.OpenURI.OpenURIUnavailable", "reason"},
		{"URI refused by window.open", func() (*Desktop, func(*Desktop) error) {
			d, _, o, _ := newTestDesktop("granted")
			o.err = errors.New(tagThrown, "window.open threw")
			return d, func(d *Desktop) error { return d.OpenURI(context.Background(), "https://example.org") }
		}, "Desktop.OpenURI.OpenURIUnavailable", "reason"},
		{"show without the API", func() (*Desktop, func(*Desktop) error) {
			return &Desktop{}, show
		}, "Desktop.ShowNotification.NotificationsUnavailable", "reason"},
		{"close without the API", func() (*Desktop, func(*Desktop) error) {
			return &Desktop{}, func(d *Desktop) error { return d.CloseNotification(context.Background(), 1) }
		}, "Desktop.CloseNotification.NotificationsUnavailable", "reason"},
		{"permission denied", func() (*Desktop, func(*Desktop) error) {
			d, _, _, _ := newTestDesktop("denied")
			return d, show
		}, "Desktop.ShowNotification->Desktop.ensurePermission.NotificationsUnavailable", "permission"},
		{"page without Notification", func() (*Desktop, func(*Desktop) error) {
			d, _, _, _ := newTestDesktop(permissionUnsupported)
			return d, show
		}, "Desktop.ShowNotification->Desktop.ensurePermission.NotificationsUnavailable", "reason"},
		{"permission prompt ended", func() (*Desktop, func(*Desktop) error) {
			d, _, _, _ := newTestDesktop("default")
			return d, func(d *Desktop) error {
				_, err := d.ShowNotification(ended, NotificationSpec{Title: "Hi"})
				return err
			}
		}, "Desktop.ShowNotification->Desktop.ensurePermission.NotificationPermissionAborted", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d, call := tc.setup()
			err := call(d)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T: %v", err, err)
			}
			if !appErr.HasRoute(tc.route) {
				t.Errorf("unexpected failure path: %s, want %s", appErr.Route(), tc.route)
			}
			if tc.meta != "" {
				if _, has := appErr.Metadata[tc.meta]; !has {
					t.Errorf("metadata = %v, want the key %q", appErr.Metadata, tc.meta)
				}
			}
		})
	}
}
