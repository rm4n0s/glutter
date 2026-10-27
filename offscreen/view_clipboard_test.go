package offscreen_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/offscreen"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The text that the parent's fake platform holds, so that a read or a write of the app's
// clipboard shows: the view's field would paste it, or the test would find it replaced.
const (
	parentClipboard = "PARENT CLIPBOARD"
	parentPrimary   = "PARENT PRIMARY"
	fieldText       = "hello world"
)

// textClip is a ui.ClipboardHost with no rich methods. Its done functions are posted to the
// UI thread of the parent, as an engine's are, or kept for the test to call (hold).
type textClip struct {
	post func(func())

	mu    sync.Mutex
	text  string
	has   bool
	sets  int
	gets  int
	hold  bool
	kept  []func()
	calls []string
}

func (c *textClip) SetClipboardText(_ context.Context, text string, done func(error)) {
	c.mu.Lock()
	c.text, c.has = text, true
	c.sets++
	c.calls = append(c.calls, "set text")
	c.mu.Unlock()
	if done != nil {
		c.deliver(func() { done(nil) })
	}
}

func (c *textClip) GetClipboardText(_ context.Context, done func(string, bool, error)) {
	c.mu.Lock()
	text, has := c.text, c.has
	c.gets++
	c.calls = append(c.calls, "get text")
	c.mu.Unlock()
	c.deliver(func() { done(text, has, nil) })
}

// deliver posts fn, or keeps it when the test holds the results back.
func (c *textClip) deliver(fn func()) {
	c.mu.Lock()
	if c.hold || c.post == nil {
		c.kept = append(c.kept, fn)
		c.mu.Unlock()
		return
	}
	post := c.post
	c.mu.Unlock()
	post(fn)
}

// attach gives the fake the way to the UI thread. The tree of a view is mounted inside
// NewView, before the test has the parent, so a result that comes earlier waits here.
func (c *textClip) attach(post func(func())) {
	c.mu.Lock()
	c.post = post
	early := c.kept
	if !c.hold {
		c.kept = nil
	}
	c.mu.Unlock()
	if c.hold {
		return
	}
	for _, fn := range early {
		post(fn)
	}
}

// release calls the results that were held back, on the calling goroutine (the UI thread).
func (c *textClip) release() {
	c.mu.Lock()
	kept := c.kept
	c.kept = nil
	c.mu.Unlock()
	for _, fn := range kept {
		fn()
	}
}

func (c *textClip) snapshot() (text string, has bool, sets, gets int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.text, c.has, c.sets, c.gets
}

func (c *textClip) callLog() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.calls...)
}

// richClip is a textClip that also implements ui.RichClipboardHost: it keeps the system
// clipboard and the primary selection.
type richClip struct {
	*textClip
	primary ui.ClipboardContent
	system  ui.ClipboardContent
}

func (c *richClip) SetClipboard(_ context.Context, kind ui.ClipboardKind, content ui.ClipboardContent, done func(error)) {
	c.mu.Lock()
	if kind == ui.ClipboardKindPrimary {
		c.primary = content
	} else {
		c.system = content
		c.text, c.has = content.Text, content.HasText
	}
	c.calls = append(c.calls, "set "+kind.String())
	c.mu.Unlock()
	if done != nil {
		c.deliver(func() { done(nil) })
	}
}

func (c *richClip) GetClipboard(_ context.Context, kind ui.ClipboardKind, format string, done func(ui.ClipboardContent, error)) {
	c.mu.Lock()
	content := c.primary
	if kind != ui.ClipboardKindPrimary {
		content = c.system
	}
	c.calls = append(c.calls, "get "+kind.String()+" "+format)
	c.mu.Unlock()
	var out ui.ClipboardContent
	switch format {
	case ui.ClipboardFormatText:
		out.Text, out.HasText = content.Text, content.HasText
	case ui.ClipboardFormatPNG:
		out.PNG = content.PNG
	case ui.ClipboardFormatURIList:
		out.URIs = content.URIs
	}
	c.deliver(func() { done(out, nil) })
}

func (c *richClip) primaryContent() ui.ClipboardContent {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.primary
}

// clipKind is a fake clipboard of the view, text only or rich.
type clipKind struct {
	name string
	rich bool
}

var clipKinds = []clipKind{{"text only", false}, {"rich", true}}

// clipHarness is a harness whose view has a text field and a clipboard of its own.
type clipHarness struct {
	*harness
	controller *widgets.TextEditingController
	text       *textClip
	rich       *richClip
}

// skipUnlessDebug skips a test that needs the platform override, a debug-only feature.
func skipUnlessDebug(t *testing.T) {
	t.Helper()
	if !foundation.DebugMode {
		t.Skip("needs the debug-only platform override of the framework")
	}
}

// newClipHarness makes a parent whose clipboard and primary selection hold the sentinels and
// a view with a focused text field and the fake clipboard (none for a nil kind). The target
// platform is Linux, which has a primary selection.
func newClipHarness(t *testing.T, k *clipKind) *clipHarness {
	t.Helper()
	skipUnlessDebug(t)
	foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(foundation.TargetPlatformLinux))
	t.Cleanup(func() {
		foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]())
	})
	ch := &clipHarness{controller: widgets.NewTextEditingController(fieldText)}
	t.Cleanup(ch.controller.Dispose)
	var o offscreen.ViewOpts
	if k != nil {
		tc := &textClip{}
		ch.text = tc
		o.Clipboard = tc
		if k.rich {
			rc := &richClip{textClip: tc}
			ch.rich = rc
			o.Clipboard = rc
		}
	}
	field := material.NewTextField(material.TextFieldOpts{Autofocus: true, Controller: ch.controller})
	ch.harness = newHarness(t, scaffoldApp(widgets.NewColumn([]widgets.Widget{widgets.NewSizedBox(field, widgets.SizedBoxOpts{Height: foundation.Some(64.0)})})), ui.Size{Width: 400, Height: 300}, o)
	// The fakes hand their results to the UI thread through the parent binding.
	if ch.text != nil {
		ch.text.attach(ch.parent.Binding.Post)
	}
	ch.parent.SetClipboardText(parentClipboard)
	ch.parent.SetPrimaryText(parentPrimary)
	ch.step(0)
	// The field takes the focus.
	for range 3 {
		ch.step(500 * time.Millisecond)
	}
	return ch
}

// press sends a key down and up to the dispatcher of the view, as an engine does.
func (h *clipHarness) press(logical services.LogicalKeyboardKey, character string) {
	h.t.Helper()
	d := h.view.Binding().PlatformDispatcher()
	physical, _ := services.PhysicalKeyOf(logical)
	d.DispatchKeyData(ui.KeyData{TimeStamp: h.parent.Elapsed(), Type: ui.KeyEventTypeDown, Logical: logical.KeyID(), Physical: physical.USBHidUsage(), Character: character})
	d.DispatchKeyData(ui.KeyData{TimeStamp: h.parent.Elapsed(), Type: ui.KeyEventTypeUp, Logical: logical.KeyID(), Physical: physical.USBHidUsage()})
}

// shortcut presses Control and the key together.
func (h *clipHarness) shortcut(logical services.LogicalKeyboardKey) {
	h.t.Helper()
	d := h.view.Binding().PlatformDispatcher()
	ctrl := services.LogicalKeyboardKeyControlLeft
	ctrlPhysical, _ := services.PhysicalKeyOf(ctrl)
	d.DispatchKeyData(ui.KeyData{TimeStamp: h.parent.Elapsed(), Type: ui.KeyEventTypeDown, Logical: ctrl.KeyID(), Physical: ctrlPhysical.USBHidUsage()})
	h.press(logical, "")
	d.DispatchKeyData(ui.KeyData{TimeStamp: h.parent.Elapsed(), Type: ui.KeyEventTypeUp, Logical: ctrl.KeyID(), Physical: ctrlPhysical.USBHidUsage()})
	h.run()
}

// run lets the posted results, the frames and the post-frame callbacks of the view run.
func (h *clipHarness) run() {
	h.t.Helper()
	for range 3 {
		h.step(pace)
	}
}

// fieldValue is the text of the field in the view.
func (h *clipHarness) fieldValue() string { return h.controller.Text() }

// parentUntouched fails the test when the clipboard or the primary selection of the app
// changed: the view never reached them.
func (h *clipHarness) parentUntouched() {
	h.t.Helper()
	if text, ok := h.parent.ClipboardText(); !ok || text != parentClipboard {
		h.t.Errorf("the app's clipboard is %q (%v), want it untouched (%q)", text, ok, parentClipboard)
	}
	if text, ok := h.parent.PrimaryText(); !ok || text != parentPrimary {
		h.t.Errorf("the app's primary selection is %q (%v), want it untouched (%q)", text, ok, parentPrimary)
	}
	if png, uris := h.parent.ClipboardPNG(), h.parent.ClipboardURIs(); png != nil || uris != nil {
		h.t.Errorf("the app's clipboard has %v and %v, want no image and no URIs", png, uris)
	}
}

func TestViewClipboardCopyCutPaste(t *testing.T) {
	for _, k := range clipKinds {
		tests := []struct {
			name string
			// keys are the shortcuts, in order, after the field has the focus.
			keys []services.LogicalKeyboardKey
			// seed is put on the view's clipboard before the keys.
			seed string
			// want is the text of the field and of the view's clipboard after the keys.
			wantField, wantClip string
		}{
			{"copy", []services.LogicalKeyboardKey{services.LogicalKeyboardKeyKeyA, services.LogicalKeyboardKeyKeyC}, "", fieldText, fieldText},
			{"cut", []services.LogicalKeyboardKey{services.LogicalKeyboardKeyKeyA, services.LogicalKeyboardKeyKeyX}, "", "", fieldText},
			{"paste", []services.LogicalKeyboardKey{services.LogicalKeyboardKeyKeyA, services.LogicalKeyboardKeyKeyV}, "FROM THE VIEW CLIPBOARD", "FROM THE VIEW CLIPBOARD", "FROM THE VIEW CLIPBOARD"},
		}
		for _, tt := range tests {
			t.Run(k.name+"/"+tt.name, func(t *testing.T) {
				h := newClipHarness(t, &k)
				if tt.seed != "" {
					h.text.mu.Lock()
					h.text.text, h.text.has = tt.seed, true
					h.text.mu.Unlock()
					if h.rich != nil {
						h.rich.mu.Lock()
						h.rich.system = ui.ClipboardContent{Text: tt.seed, HasText: true}
						h.rich.mu.Unlock()
					}
				}
				for _, key := range tt.keys {
					h.shortcut(key)
				}
				if got := h.fieldValue(); got != tt.wantField {
					t.Errorf("the field holds %q, want %q", got, tt.wantField)
				}
				if text, has, _, _ := h.text.snapshot(); text != tt.wantClip || !has {
					t.Errorf("the view's clipboard holds %q (%v), want %q", text, has, tt.wantClip)
				}
				h.parentUntouched()
			})
		}
	}
}

// selectAllAndMiddleClick selects the text of the field (a selection fills the primary
// selection on Linux) and clicks the middle button on the text.
func (h *clipHarness) middleClick(x, y float64) {
	h.t.Helper()
	d := h.view.Binding().PlatformDispatcher()
	p := ui.PointerData{Kind: ui.PointerDeviceKindMouse, Device: 7, PhysicalX: x, PhysicalY: y}
	add, down, up := p, p, p
	add.Change = ui.PointerChangeAdd
	down.Change, down.Buttons = ui.PointerChangeDown, gestures.TertiaryButton
	up.Change = ui.PointerChangeUp
	d.DispatchPointerDataPacket(ui.PointerDataPacket{Data: []ui.PointerData{add}})
	d.DispatchPointerDataPacket(ui.PointerDataPacket{Data: []ui.PointerData{down}})
	d.DispatchPointerDataPacket(ui.PointerDataPacket{Data: []ui.PointerData{up}})
	h.run()
}

func TestViewClipboardPrimarySelection(t *testing.T) {
	for _, k := range clipKinds {
		t.Run(k.name, func(t *testing.T) {
			h := newClipHarness(t, &k)
			// Selecting all the text puts it on the primary selection of a Linux desktop. (The
			// field asked the clipboard whether it can paste when it took the focus, before.)
			before := len(h.text.callLog())
			h.shortcut(services.LogicalKeyboardKeyKeyA)
			h.run()
			if k.rich {
				if got := h.rich.primaryContent(); !got.HasText || got.Text != fieldText {
					t.Errorf("the view's primary selection is %+v, want %q", got, fieldText)
				}
			} else if log := h.text.callLog()[before:]; len(log) != 0 {
				t.Errorf("a selection called a text-only clipboard: %v", log)
			}
			h.parentUntouched()

			// A middle click pastes the primary selection of the view, if it has one.
			if k.rich {
				h.rich.mu.Lock()
				h.rich.primary = ui.ClipboardContent{Text: "MIDDLE", HasText: true}
				h.rich.mu.Unlock()
			}
			h.middleClick(60, 56+28)
			got := h.fieldValue()
			hasMiddle := containsText(got, "MIDDLE")
			if hasMiddle != k.rich {
				t.Errorf("after a middle click the field holds %q; MIDDLE pasted = %v, want %v", got, hasMiddle, k.rich)
			}
			if containsText(got, parentPrimary) || containsText(got, parentClipboard) {
				t.Errorf("the field pasted from the app: %q", got)
			}
			h.parentUntouched()
		})
	}
}

func containsText(s, sub string) bool { return strings.Contains(s, sub) }

func TestViewClipboardPasteCheck(t *testing.T) {
	for _, k := range clipKinds {
		tests := []struct {
			name string
			seed string
			want widgets.ClipboardStatus
		}{
			{"the view has text", "text", widgets.ClipboardStatusPasteable},
			{"the view has none", "", widgets.ClipboardStatusNotPasteable},
		}
		for _, tt := range tests {
			t.Run(k.name+"/"+tt.name, func(t *testing.T) {
				h := newClipHarness(t, &k)
				h.text.mu.Lock()
				h.text.text, h.text.has = tt.seed, tt.seed != ""
				h.text.mu.Unlock()
				if h.rich != nil {
					h.rich.mu.Lock()
					h.rich.system = ui.ClipboardContent{Text: tt.seed, HasText: tt.seed != ""}
					h.rich.mu.Unlock()
				}
				n := widgets.NewClipboardStatusNotifier(h.view.Binding())
				t.Cleanup(n.Dispose)
				n.Update()
				h.run()
				if got := n.Value(); got != tt.want {
					t.Errorf("the status is %v, want %v (the app's clipboard holds %q)", got, tt.want, parentClipboard)
				}
				var has, called bool
				h.view.Binding().Clipboard().HasStrings(context.Background(), func(b bool, err error) {
					has, called = b, err == nil
				})
				h.run()
				if !called || has != (tt.seed != "") {
					t.Errorf("HasStrings = %v (answered without an error: %v), want %v", has, called, tt.seed != "")
				}
				h.parentUntouched()
			})
		}
	}
}

// TestViewClipboardRichCalls checks each call of the services clipboard against both kinds
// of host: where it goes, and what the widget side hears.
func TestViewClipboardRichCalls(t *testing.T) {
	png := []byte("\x89PNG-bytes")
	type outcome struct {
		data     *services.ClipboardData
		err      error
		answered bool
	}
	tests := []struct {
		name string
		call func(c *services.Clipboard, ctx context.Context, done func(outcome))
		// calls are the calls that each kind of host must see.
		textCalls, richCalls []string
		// want checks the outcome for each kind.
		wantText, wantRich func(o outcome) bool
	}{
		{
			name: "set the clipboard text",
			call: func(c *services.Clipboard, ctx context.Context, done func(outcome)) {
				c.SetData(ctx, services.NewClipboardData("x"), func(err error) { done(outcome{err: err, answered: true}) })
			},
			textCalls: []string{"set text"}, richCalls: []string{"set text"},
			wantText: func(o outcome) bool { return o.answered && o.err == nil },
			wantRich: func(o outcome) bool { return o.answered && o.err == nil },
		},
		{
			name: "set an image",
			call: func(c *services.Clipboard, ctx context.Context, done func(outcome)) {
				c.SetData(ctx, services.ClipboardData{PNG: png}, func(err error) { done(outcome{err: err, answered: true}) })
			},
			textCalls: nil, richCalls: []string{"set ClipboardKind.system"},
			wantText: func(o outcome) bool { return o.answered && o.err == nil },
			wantRich: func(o outcome) bool { return o.answered && o.err == nil },
		},
		{
			name: "set the primary selection",
			call: func(c *services.Clipboard, ctx context.Context, done func(outcome)) {
				c.SetPrimaryData(ctx, services.NewClipboardData("x"), func(err error) { done(outcome{err: err, answered: true}) })
			},
			textCalls: nil, richCalls: []string{"set ClipboardKind.primary"},
			wantText: func(o outcome) bool { return o.answered && o.err == nil },
			wantRich: func(o outcome) bool { return o.answered && o.err == nil },
		},
		{
			name: "get the clipboard text",
			call: func(c *services.Clipboard, ctx context.Context, done func(outcome)) {
				c.GetData(ctx, services.ClipboardKTextPlain, func(d *services.ClipboardData, err error) { done(outcome{d, err, true}) })
			},
			textCalls: []string{"get text"}, richCalls: []string{"get text"},
			wantText: func(o outcome) bool { return o.answered && o.err == nil && o.data != nil && o.data.Text == "seed" },
			wantRich: func(o outcome) bool { return o.answered && o.err == nil && o.data != nil && o.data.Text == "seed" },
		},
		{
			name: "get an image",
			call: func(c *services.Clipboard, ctx context.Context, done func(outcome)) {
				c.GetData(ctx, services.ClipboardKImagePNG, func(d *services.ClipboardData, err error) { done(outcome{d, err, true}) })
			},
			textCalls: nil, richCalls: []string{"get ClipboardKind.system image/png"},
			wantText: func(o outcome) bool { return o.answered && o.err == nil && o.data == nil },
			wantRich: func(o outcome) bool { return o.answered && o.err == nil && o.data == nil },
		},
		{
			name: "get the primary selection",
			call: func(c *services.Clipboard, ctx context.Context, done func(outcome)) {
				c.GetPrimaryData(ctx, services.ClipboardKTextPlain, func(d *services.ClipboardData, err error) { done(outcome{d, err, true}) })
			},
			textCalls: nil, richCalls: []string{"get ClipboardKind.primary text/plain"},
			wantText: func(o outcome) bool { return o.answered && o.err == nil && o.data == nil },
			wantRich: func(o outcome) bool { return o.answered && o.err == nil && o.data == nil },
		},
	}
	for _, k := range clipKinds {
		for _, tt := range tests {
			t.Run(k.name+"/"+tt.name, func(t *testing.T) {
				h := newClipHarness(t, &k)
				h.text.mu.Lock()
				h.text.text, h.text.has = "seed", true
				h.text.mu.Unlock()
				if h.rich != nil {
					h.rich.mu.Lock()
					h.rich.system = ui.ClipboardContent{Text: "seed", HasText: true}
					h.rich.mu.Unlock()
				}
				before := len(h.text.callLog())
				var got outcome
				tt.call(h.view.Binding().Clipboard(), context.Background(), func(o outcome) { got = o })
				h.run()
				want, check := tt.textCalls, tt.wantText
				if k.rich {
					want, check = tt.richCalls, tt.wantRich
				}
				if log := h.text.callLog()[before:]; !equalStrings(log, want) {
					t.Errorf("the host saw %v, want %v", log, want)
				}
				if !check(got) {
					t.Errorf("the widget side heard %+v", got)
				}
				h.parentUntouched()
			})
		}
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestViewClipboardNilUsesTheApp(t *testing.T) {
	h := newClipHarness(t, nil)
	// The paste check reads the app's clipboard.
	n := widgets.NewClipboardStatusNotifier(h.view.Binding())
	t.Cleanup(n.Dispose)
	n.Update()
	h.run()
	if got := n.Value(); got != widgets.ClipboardStatusPasteable {
		t.Errorf("the status is %v, want pasteable (the app's clipboard holds text)", got)
	}
	// Copy goes to the app's clipboard, and the selection to its primary selection.
	h.shortcut(services.LogicalKeyboardKeyKeyA)
	h.shortcut(services.LogicalKeyboardKeyKeyC)
	if text, ok := h.parent.ClipboardText(); !ok || text != fieldText {
		t.Errorf("the app's clipboard holds %q (%v), want %q", text, ok, fieldText)
	}
	if text, ok := h.parent.PrimaryText(); !ok || text != fieldText {
		t.Errorf("the app's primary selection holds %q (%v), want %q", text, ok, fieldText)
	}
	// Paste reads it.
	h.parent.SetClipboardText("FROM THE APP")
	h.shortcut(services.LogicalKeyboardKeyKeyA)
	h.shortcut(services.LogicalKeyboardKeyKeyV)
	if got := h.fieldValue(); got != "FROM THE APP" {
		t.Errorf("the field holds %q, want the app's clipboard", got)
	}
}

func TestViewClipboardDoneAfterCloseDoesNothing(t *testing.T) {
	for _, k := range clipKinds {
		tests := []struct {
			name string
			call func(c *services.Clipboard, done func())
		}{
			{"set text", func(c *services.Clipboard, done func()) {
				c.SetData(context.Background(), services.NewClipboardData("x"), func(error) { done() })
			}},
			{"get text", func(c *services.Clipboard, done func()) {
				c.GetData(context.Background(), services.ClipboardKTextPlain, func(*services.ClipboardData, error) { done() })
			}},
			{"has strings", func(c *services.Clipboard, done func()) {
				c.HasStrings(context.Background(), func(bool, error) { done() })
			}},
			{"set primary", func(c *services.Clipboard, done func()) {
				c.SetPrimaryData(context.Background(), services.NewClipboardData("x"), func(error) { done() })
			}},
			{"get primary", func(c *services.Clipboard, done func()) {
				c.GetPrimaryData(context.Background(), services.ClipboardKTextPlain, func(*services.ClipboardData, error) { done() })
			}},
		}
		for _, tt := range tests {
			t.Run(k.name+"/"+tt.name, func(t *testing.T) {
				h := newClipHarness(t, &k)
				h.text.mu.Lock()
				h.text.hold = true
				h.text.mu.Unlock()
				calls := 0
				tt.call(h.view.Binding().Clipboard(), func() { calls++ })
				h.run()
				// Before Close the result arrives.
				h.text.release()
				h.run()
				if calls != 1 {
					t.Fatalf("the result before Close arrived %d times, want 1", calls)
				}
				calls = 0
				tt.call(h.view.Binding().Clipboard(), func() { calls++ })
				h.view.Close()
				h.text.release()
				h.parent.Pump(pace)
				if calls != 0 {
					t.Errorf("a result after Close reached the widget side %d times, want 0", calls)
				}
				h.parentUntouched()
			})
		}
	}
}
