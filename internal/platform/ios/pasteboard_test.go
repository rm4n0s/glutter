package ios

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

// fakeBoard is a pasteBackend that has what a test gives it.
type fakeBoard struct {
	mu         sync.Mutex
	hasStrings bool
	text       string
	allowed    bool
	html       string
	png        []byte
	urls       []string
	refuse     bool
	written    [][]pasteItem
	// gate, when it is not nil, holds String until the test closes it: the paste prompt that nobody answers.
	gate chan struct{}
}

func (f *fakeBoard) HasStrings() bool { return f.hasStrings }
func (f *fakeBoard) String() (string, bool) {
	if f.gate != nil {
		<-f.gate
	}
	return f.text, f.allowed
}
func (f *fakeBoard) HTML() (string, bool) { return f.html, f.html != "" }
func (f *fakeBoard) PNG() []byte          { return f.png }
func (f *fakeBoard) URLs() []string       { return f.urls }
func (f *fakeBoard) Write(items []pasteItem) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.refuse {
		return false
	}
	f.written = append(f.written, items)
	return true
}

// ui is a UI thread that the test runs by hand: the functions that were posted wait in a channel.
type uiThread struct{ ch chan func() }

func newUIThread() *uiThread      { return &uiThread{ch: make(chan func(), 16)} }
func (u *uiThread) post(f func()) { u.ch <- f }
func (u *uiThread) next(t *testing.T) {
	t.Helper()
	select {
	case f := <-u.ch:
		f()
	case <-time.After(5 * time.Second):
		t.Fatal("nothing was posted to the UI thread")
	}
}

func TestPasteItemsOf(t *testing.T) {
	tests := []struct {
		name string
		in   ui.ClipboardContent
		want []pasteItem
	}{
		{"nothing", ui.ClipboardContent{}, nil},
		{"text", ui.ClipboardContent{Text: "hi", HasText: true}, []pasteItem{{{UTI: typeText, Kind: pasteString, Text: "hi"}}}},
		{"an empty text is a text", ui.ClipboardContent{HasText: true}, []pasteItem{{{UTI: typeText, Kind: pasteString}}}},
		{"text with html and an image", ui.ClipboardContent{Text: "a", HasText: true, HTML: "<b>a</b>", PNG: []byte{1, 2}},
			[]pasteItem{{{UTI: typeText, Kind: pasteString, Text: "a"}, {UTI: typeHTML, Kind: pasteString, Text: "<b>a</b>"}, {UTI: typePNG, Kind: pasteData, Data: []byte{1, 2}}}}},
		{"each uri is an item", ui.ClipboardContent{URIs: []string{"file:///a", "https://b/"}},
			[]pasteItem{{{UTI: typeURL, Kind: pasteURL, Text: "file:///a"}}, {{UTI: typeURL, Kind: pasteURL, Text: "https://b/"}}}},
		{"text and a uri", ui.ClipboardContent{Text: "a", HasText: true, URIs: []string{"https://b/"}},
			[]pasteItem{{{UTI: typeText, Kind: pasteString, Text: "a"}}, {{UTI: typeURL, Kind: pasteURL, Text: "https://b/"}}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := pasteItemsOf(tc.in); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("pasteItemsOf = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestPasteboardWrites(t *testing.T) {
	ctx := context.Background()
	t.Run("text goes to the backend and done comes after the call", func(t *testing.T) {
		u, b := newUIThread(), &fakeBoard{}
		pb := pasteboard{post: u.post, back: b}
		var got error = &errors.Error{Tag: tagTestNotCalled}
		called := false
		pb.SetText(ctx, "hello", func(err error) { got, called = err, true })
		if called {
			t.Fatal("done was called before SetText returned")
		}
		u.next(t)
		if got != nil {
			t.Fatalf("SetText: %v", got)
		}
		if want := [][]pasteItem{{{{UTI: typeText, Kind: pasteString, Text: "hello"}}}}; !reflect.DeepEqual(b.written, want) {
			t.Errorf("written %+v, want %+v", b.written, want)
		}
	})
	t.Run("the primary selection does not exist", func(t *testing.T) {
		u, b := newUIThread(), &fakeBoard{}
		pb := pasteboard{post: u.post, back: b}
		var got error
		pb.SetContent(ctx, ui.ClipboardKindPrimary, ui.ClipboardContent{Text: "x", HasText: true}, func(err error) { got = err })
		u.next(t)
		appErr, ok := errors.FromError(got)
		if !ok || appErr.Tag != tagClipboardPrimaryUnavailable || !appErr.HasRoute("pasteboard.SetContent."+tagClipboardPrimaryUnavailable) {
			t.Errorf("error %v, want ClipboardPrimaryUnavailable", got)
		}
		if len(b.written) != 0 {
			t.Errorf("the primary selection wrote %v", b.written)
		}
	})
	t.Run("a pasteboard that refuses", func(t *testing.T) {
		u, b := newUIThread(), &fakeBoard{refuse: true}
		pb := pasteboard{post: u.post, back: b}
		var got error
		pb.SetText(ctx, "x", func(err error) { got = err })
		u.next(t)
		appErr, ok := errors.FromError(got)
		if !ok || appErr.Tag != tagIOSClipboardUnavailable || !appErr.HasRoute("pasteboard.SetContent."+tagIOSClipboardUnavailable) {
			t.Errorf("error %v, want IOSClipboardUnavailable", got)
		}
	})
	t.Run("no done is fine", func(t *testing.T) {
		pb := pasteboard{post: newUIThread().post, back: &fakeBoard{}}
		pb.SetText(ctx, "x", nil)
	})
}

func TestPasteboardReads(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name   string
		board  *fakeBoard
		format string
		want   ui.ClipboardContent
	}{
		{"text", &fakeBoard{hasStrings: true, text: "hi", allowed: true}, ui.ClipboardFormatText, ui.ClipboardContent{Text: "hi", HasText: true}},
		{"an empty text is a text", &fakeBoard{hasStrings: true, text: "", allowed: true}, ui.ClipboardFormatText, ui.ClipboardContent{HasText: true}},
		{"no text", &fakeBoard{}, ui.ClipboardFormatText, ui.ClipboardContent{}},
		{"html", &fakeBoard{html: "<i>x</i>"}, ui.ClipboardFormatHTML, ui.ClipboardContent{HTML: "<i>x</i>"}},
		{"no html", &fakeBoard{hasStrings: true, text: "x", allowed: true}, ui.ClipboardFormatHTML, ui.ClipboardContent{}},
		{"an image", &fakeBoard{png: []byte{9, 8}}, ui.ClipboardFormatPNG, ui.ClipboardContent{PNG: []byte{9, 8}}},
		{"urls", &fakeBoard{urls: []string{"https://a/"}}, ui.ClipboardFormatURIList, ui.ClipboardContent{URIs: []string{"https://a/"}}},
		{"text is the default format", &fakeBoard{hasStrings: true, text: "d", allowed: true}, "", ui.ClipboardContent{Text: "d", HasText: true}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u := newUIThread()
			pb := pasteboard{post: u.post, back: tc.board}
			var got ui.ClipboardContent
			var gotErr error
			pb.GetContent(ctx, ui.ClipboardKindSystem, tc.format, func(c ui.ClipboardContent, err error) { got, gotErr = c, err })
			u.next(t)
			if gotErr != nil || !reflect.DeepEqual(got, tc.want) {
				t.Errorf("read %+v, %v; want %+v", got, gotErr, tc.want)
			}
		})
	}
	t.Run("GetText is the text", func(t *testing.T) {
		u := newUIThread()
		pb := pasteboard{post: u.post, back: &fakeBoard{hasStrings: true, text: "t", allowed: true}}
		var text string
		var ok bool
		pb.GetText(ctx, func(s string, k bool, err error) { text, ok = s, k })
		u.next(t)
		if text != "t" || !ok {
			t.Errorf("GetText = %q, %v", text, ok)
		}
	})
}

func TestPasteDenied(t *testing.T) {
	u := newUIThread()
	pb := pasteboard{post: u.post, back: &fakeBoard{hasStrings: true, allowed: false}}
	var got error
	var gotOK = true
	pb.GetText(context.Background(), func(_ string, ok bool, err error) { gotOK, got = ok, err })
	u.next(t)
	appErr, ok := errors.FromError(got)
	if !ok || appErr.Tag != tagPasteDenied || !appErr.HasRoute("readPasteboard."+tagPasteDenied) {
		t.Errorf("error %v, want PasteDenied", got)
	}
	if gotOK {
		t.Error("a denied paste has text")
	}
}

func TestReadWaitsOffTheUIThreadAndEnds(t *testing.T) {
	u := newUIThread()
	gate := make(chan struct{})
	pb := pasteboard{post: u.post, back: &fakeBoard{hasStrings: true, text: "late", allowed: true, gate: gate}}
	ctx, cancel := context.WithCancel(context.Background())
	var got error
	called := make(chan struct{})
	pb.GetContent(ctx, ui.ClipboardKindSystem, ui.ClipboardFormatText, func(_ ui.ClipboardContent, err error) { got = err; close(called) })
	// GetContent returned although the prompt is not answered: the UI thread never waited.
	cancel()
	u.next(t)
	<-called
	appErr, ok := errors.FromError(got)
	if !ok || appErr.Tag != tagIOSClipboardUnavailable || !appErr.HasRoute("pasteboard.await."+tagIOSClipboardUnavailable) {
		t.Errorf("error %v (route %v), want IOSClipboardUnavailable", got, appErr.Route())
	}
	// The answer that comes later is dropped: nothing more is posted.
	close(gate)
	select {
	case <-u.ch:
		t.Error("a late answer was posted")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestPrimarySelectionRead(t *testing.T) {
	u := newUIThread()
	pb := pasteboard{post: u.post, back: &fakeBoard{}}
	var got error
	pb.GetContent(context.Background(), ui.ClipboardKindPrimary, ui.ClipboardFormatText, func(_ ui.ClipboardContent, err error) { got = err })
	u.next(t)
	appErr, ok := errors.FromError(got)
	if !ok || appErr.Tag != tagClipboardPrimaryUnavailable || !appErr.HasRoute("pasteboard.GetContent."+tagClipboardPrimaryUnavailable) {
		t.Errorf("error %v, want ClipboardPrimaryUnavailable", got)
	}
}

// The tag of the placeholder error of a test (rm4n0s/errors, R5).
const tagTestNotCalled = "TestNotCalled"
