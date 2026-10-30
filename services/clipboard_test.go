package services

import (
	"context"
	"reflect"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
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
		{name: "unknown format", has: true, text: "Hello", format: "application/json", wantNil: true},
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
	boom := errors.New(tagBackendBoom, "the backend failed")
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
	routes := []string{"Clipboard.SetData.ClipboardUnavailable", "Clipboard.GetData.ClipboardUnavailable", "Clipboard.HasStrings->Clipboard.GetData.ClipboardUnavailable"}
	for i, err := range errs {
		appErr, ok := errors.FromError(err)
		if !ok {
			t.Fatalf("%d: expected *errors.Error, got %T", i, err)
		}
		if !appErr.HasRoute(routes[i]) {
			t.Errorf("%d: unexpected failure path: %s", i, appErr.Route())
		}
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagBackendBoom = "BackendBoom"
)

// richMemoryClipboard is a memoryClipboard that is also a RichClipboardBackend: it keeps the content of
// the system clipboard and the primary selection, and gives the format that is asked for.
type richMemoryClipboard struct {
	memoryClipboard
	system, primary ui.ClipboardContent
	setKinds        []ui.ClipboardKind
}

func (r *richMemoryClipboard) SetClipboard(_ context.Context, kind ui.ClipboardKind, content ui.ClipboardContent, done func(error)) {
	r.setKinds = append(r.setKinds, kind)
	if kind == ui.ClipboardKindPrimary {
		r.primary = content
	} else {
		r.system = content
	}
	if done != nil {
		r.pending = append(r.pending, func() { done(nil) })
	}
}

func (r *richMemoryClipboard) GetClipboard(_ context.Context, kind ui.ClipboardKind, format string, done func(ui.ClipboardContent, error)) {
	content := r.system
	if kind == ui.ClipboardKindPrimary {
		content = r.primary
	}
	var out ui.ClipboardContent
	switch format {
	case ui.ClipboardFormatText:
		out.Text, out.HasText = content.Text, content.HasText
	case ui.ClipboardFormatHTML:
		out.HTML = content.HTML
	}
	r.pending = append(r.pending, func() { done(out, nil) })
}

func TestClipboardHTML(t *testing.T) {
	const fragment = "<b>caf\u00e9</b> \u4e16\u754c"
	tests := []struct {
		name string
		data ClipboardData
		// want is what the backend must hold after SetData.
		want ui.ClipboardContent
	}{
		{"text and HTML", ClipboardData{Text: "caf\u00e9", HTML: fragment}, ui.ClipboardContent{Text: "caf\u00e9", HasText: true, HTML: fragment}},
		{"HTML only has no text", ClipboardData{HTML: fragment}, ui.ClipboardContent{HTML: fragment}},
	}
	for _, tt := range tests {
		t.Run("set "+tt.name, func(t *testing.T) {
			m := &richMemoryClipboard{}
			var setErr error
			NewClipboard(m).SetData(context.Background(), tt.data, func(err error) { setErr = err })
			m.run()
			if setErr != nil || !reflect.DeepEqual(m.system, tt.want) {
				t.Errorf("backend holds %+v (error %v), want %+v", m.system, setErr, tt.want)
			}
			if m.text != "" || m.has {
				t.Error("the text went through the text-only call, so the HTML was lost")
			}
		})
	}

	reads := []struct {
		name     string
		system   ui.ClipboardContent
		format   string
		wantNil  bool
		wantHTML string
	}{
		{"read HTML", ui.ClipboardContent{Text: "t", HasText: true, HTML: fragment}, ClipboardKTextHTML, false, fragment},
		{"a clipboard with no HTML gives nil", ui.ClipboardContent{Text: "t", HasText: true}, ClipboardKTextHTML, true, ""},
		{"an empty clipboard gives nil", ui.ClipboardContent{}, ClipboardKTextHTML, true, ""},
		{"the text of a clipboard that has HTML", ui.ClipboardContent{Text: "t", HasText: true, HTML: fragment}, ClipboardKTextPlain, false, ""},
	}
	for _, tt := range reads {
		t.Run(tt.name, func(t *testing.T) {
			m := &richMemoryClipboard{system: tt.system, memoryClipboard: memoryClipboard{text: tt.system.Text, has: tt.system.HasText}}
			var got *ClipboardData
			called := false
			NewClipboard(m).GetData(context.Background(), tt.format, func(d *ClipboardData, err error) {
				got, called = d, true
				if err != nil {
					t.Errorf("error: %v", err)
				}
			})
			m.run()
			if !called {
				t.Fatal("done was not called")
			}
			if tt.wantNil != (got == nil) || got != nil && got.HTML != tt.wantHTML {
				t.Errorf("data = %+v, want nil %v and HTML %q", got, tt.wantNil, tt.wantHTML)
			}
		})
	}

	t.Run("the primary selection reads HTML too", func(t *testing.T) {
		m := &richMemoryClipboard{primary: ui.ClipboardContent{HTML: fragment}}
		var got *ClipboardData
		NewClipboard(m).GetPrimaryData(context.Background(), ClipboardKTextHTML, func(d *ClipboardData, _ error) { got = d })
		m.run()
		if got == nil || got.HTML != fragment {
			t.Errorf("data = %+v", got)
		}
	})

	t.Run("a text-only backend cannot take HTML", func(t *testing.T) {
		m := &memoryClipboard{}
		var setErr, getErr error
		c := NewClipboard(m)
		c.SetData(context.Background(), ClipboardData{Text: "t", HTML: fragment}, func(err error) { setErr = err })
		c.GetData(context.Background(), ClipboardKTextHTML, func(_ *ClipboardData, err error) { getErr = err })
		m.run()
		for name, err := range map[string]error{"set": setErr, "get": getErr} {
			appErr, ok := errors.FromError(err)
			if !ok || !appErr.HasRoute("ClipboardUnavailable") {
				t.Errorf("%s: %v, want ClipboardUnavailable", name, err)
			}
		}
	})
}

func TestClipboardContentHelpers(t *testing.T) {
	tests := []struct {
		name    string
		content ui.ClipboardContent
		empty   bool
		has     map[string]bool
	}{
		{"empty", ui.ClipboardContent{}, true, map[string]bool{ui.ClipboardFormatHTML: false, ui.ClipboardFormatText: false}},
		{"HTML only", ui.ClipboardContent{HTML: "<b>x</b>"}, false, map[string]bool{ui.ClipboardFormatHTML: true, ui.ClipboardFormatText: false}},
		{"text and HTML", ui.ClipboardContent{Text: "x", HasText: true, HTML: "<b>x</b>"}, false, map[string]bool{ui.ClipboardFormatHTML: true, ui.ClipboardFormatText: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.content.IsEmpty() != tt.empty {
				t.Errorf("IsEmpty = %v", tt.content.IsEmpty())
			}
			for format, want := range tt.has {
				if tt.content.Has(format) != want {
					t.Errorf("Has(%s) = %v, want %v", format, tt.content.Has(format), want)
				}
			}
		})
	}
	// The conversions keep the HTML.
	d := ClipboardData{Text: "t", HTML: "<i>t</i>"}
	if got := dataOf(d.content()); !reflect.DeepEqual(got, d) {
		t.Errorf("round trip = %+v, want %+v", got, d)
	}
}
