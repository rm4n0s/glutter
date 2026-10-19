package services

import (
	"context"
	"testing"

	"github.com/rm4n0s/errors"
)

// memoryClipboard is a MockClipboard: the text lives in the struct and the
// results arrive when run is called, like the engine's.
type memoryClipboard struct {
	text    string
	has     bool
	err     error
	pending []func()
}

func (m *memoryClipboard) SetClipboardText(_ context.Context, text string, done func(error)) {
	m.text, m.has = text, true
	if done != nil {
		err := m.err
		m.pending = append(m.pending, func() { done(err) })
	}
}

func (m *memoryClipboard) GetClipboardText(_ context.Context, done func(string, bool, error)) {
	text, ok, err := m.text, m.has, m.err
	m.pending = append(m.pending, func() { done(text, ok, err) })
}

func (m *memoryClipboard) run() {
	for len(m.pending) > 0 {
		fn := m.pending[0]
		m.pending = m.pending[1:]
		fn()
	}
}

// Ported from clipboard_test.dart. The two cases that expect a TypeError for a
// platform answer without text have no counterpart: the backend answers with
// the text and a flag, so there is no malformed answer.
func TestClipboardGetData(t *testing.T) {
	tests := []struct {
		name     string
		has      bool
		text     string
		format   string
		wantText string
		wantNil  bool
	}{
		{name: "returns text", has: true, text: "Hello world", format: ClipboardKTextPlain, wantText: "Hello world"},
		{name: "returns nil", format: ClipboardKTextPlain, wantNil: true},
		{name: "returns empty text", has: true, text: "", format: ClipboardKTextPlain, wantText: ""},
		{name: "unknown format", has: true, text: "Hello", format: "image/png", wantNil: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &memoryClipboard{text: tt.text, has: tt.has}
			c := NewClipboard(m)
			called := false
			c.GetData(context.Background(), tt.format, func(data *ClipboardData, err error) {
				called = true
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				switch {
				case tt.wantNil && data != nil:
					t.Errorf("data = %+v, want nil", data)
				case !tt.wantNil && (data == nil || data.Text != tt.wantText):
					t.Errorf("data = %+v, want %q", data, tt.wantText)
				}
			})
			m.run()
			if !called {
				t.Error("done was not called")
			}
		})
	}
}

func TestClipboardSetData(t *testing.T) {
	m := &memoryClipboard{}
	c := NewClipboard(m)
	var done bool
	c.SetData(context.Background(), NewClipboardData("Hello world"), func(err error) {
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		done = true
	})
	if !m.has || m.text != "Hello world" {
		t.Errorf("clipboard = %q (has %v)", m.text, m.has)
	}
	if done {
		t.Error("done was called before the method returned")
	}
	m.run()
	if !done {
		t.Error("done was not called")
	}
	// A nil done is allowed.
	c.SetData(context.Background(), NewClipboardData("again"), nil)
	m.run()
	if m.text != "again" {
		t.Errorf("clipboard = %q", m.text)
	}
}

func TestClipboardHasStrings(t *testing.T) {
	tests := []struct {
		name string
		has  bool
		text string
		want bool
	}{
		{"text", true, "x", true},
		{"empty text", true, "", false},
		{"nothing", false, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &memoryClipboard{text: tt.text, has: tt.has}
			var got, called bool
			NewClipboard(m).HasStrings(context.Background(), func(has bool, err error) {
				got, called = has, true
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			})
			m.run()
			if !called || got != tt.want {
				t.Errorf("HasStrings = %v (called %v), want %v", got, called, tt.want)
			}
		})
	}
}

func TestClipboardPassesBackendErrors(t *testing.T) {
	boom := errors.New("BackendBoom", "the backend failed")
	m := &memoryClipboard{err: boom}
	c := NewClipboard(m)
	var setErr, getErr, hasErr error
	c.SetData(context.Background(), NewClipboardData("x"), func(err error) { setErr = err })
	c.GetData(context.Background(), ClipboardKTextPlain, func(_ *ClipboardData, err error) { getErr = err })
	c.HasStrings(context.Background(), func(_ bool, err error) { hasErr = err })
	m.run()
	for name, err := range map[string]error{"set": setErr, "get": getErr, "has": hasErr} {
		if err != boom {
			t.Errorf("%s: error = %v, want the backend's", name, err)
		}
	}
}

func TestClipboardWithoutBackend(t *testing.T) {
	var c *Clipboard
	var errs []error
	c.SetData(context.Background(), NewClipboardData("x"), func(err error) { errs = append(errs, err) })
	c.GetData(context.Background(), ClipboardKTextPlain, func(_ *ClipboardData, err error) { errs = append(errs, err) })
	NewClipboard(nil).HasStrings(context.Background(), func(_ bool, err error) { errs = append(errs, err) })
	c.SetData(context.Background(), NewClipboardData("x"), nil) // no done: nothing to call
	if len(errs) != 3 {
		t.Fatalf("got %d results, want 3", len(errs))
	}
	for i, err := range errs {
		appErr, ok := errors.FromError(err)
		if !ok {
			t.Fatalf("%d: expected *errors.Error, got %T", i, err)
		}
		if !appErr.HasRoute("errClipboardUnavailable.ClipboardUnavailable") {
			t.Errorf("%d: unexpected failure path: %s", i, appErr.Route())
		}
	}
}
