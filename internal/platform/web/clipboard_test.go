package web

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

// fakeAPI is a Clipboard API whose promises the test settles by hand.
type fakeAPI struct {
	reads  []func(string, error)
	writes []string
	wdone  []func(error)
}

func (f *fakeAPI) ReadText(done func(string, error)) { f.reads = append(f.reads, done) }
func (f *fakeAPI) WriteText(text string, done func(error)) {
	f.writes = append(f.writes, text)
	f.wdone = append(f.wdone, done)
}

// posts collects what is posted to the UI goroutine; the context of a read posts from a goroutine of its
// own.
type posts struct {
	mu  sync.Mutex
	fns []func()
}

func (p *posts) post(fn func()) {
	p.mu.Lock()
	p.fns = append(p.fns, fn)
	p.mu.Unlock()
}

func (p *posts) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.fns)
}

func (p *posts) run() {
	p.mu.Lock()
	fns := p.fns
	p.fns = nil
	p.mu.Unlock()
	for _, fn := range fns {
		fn()
	}
}

func tagOf(t *testing.T, err error) string {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	return appErr.Tag
}

func TestClipboardWithoutAnAPIFailsThroughPost(t *testing.T) {
	var p posts
	c := clipboard{post: p.post}
	var errs []error
	c.SetText(context.Background(), "x", func(err error) { errs = append(errs, err) })
	c.GetText(context.Background(), func(_ string, ok bool, err error) {
		if ok {
			t.Error("GetText said it read text")
		}
		errs = append(errs, err)
	})
	c.SetContent(context.Background(), ui.ClipboardKindSystem, ui.ClipboardContent{Text: "x", HasText: true}, func(err error) { errs = append(errs, err) })
	c.GetContent(context.Background(), ui.ClipboardKindSystem, ui.ClipboardFormatText, func(_ ui.ClipboardContent, err error) { errs = append(errs, err) })
	c.SetText(context.Background(), "y", nil) // no callback: nothing to post
	if len(errs) != 0 || p.count() != 4 {
		t.Fatalf("answered before the call returned (%d answers) or posted %d, want 0 and 4", len(errs), p.count())
	}
	p.run()
	for i, err := range errs {
		if got := tagOf(t, err); got != "WebClipboardUnavailable" {
			t.Errorf("answer %d = %s, want WebClipboardUnavailable", i, got)
		}
	}
}

// TestClipboardReadIsAnsweredFromThePromise: the read returns at once; the answer is posted when the
// promise settles, never before; the UI goroutine does not wait.
func TestClipboardReadIsAnsweredFromThePromise(t *testing.T) {
	var p posts
	api := &fakeAPI{}
	c := clipboard{post: p.post, api: api}
	var got []string
	var oks []bool
	c.GetText(context.Background(), func(text string, ok bool, err error) {
		if err != nil {
			t.Errorf("error: %v", err)
		}
		got, oks = append(got, text), append(oks, ok)
	})
	if len(api.reads) != 1 || p.count() != 0 || len(got) != 0 {
		t.Fatalf("reads=%d posted=%d answers=%d: the call must only start the read", len(api.reads), p.count(), len(got))
	}
	api.reads[0]("pasted text", nil) // the promise settles later, on the browser's callback
	if len(got) != 0 {
		t.Fatal("answered on the callback of the promise instead of on the UI goroutine")
	}
	p.run()
	if len(got) != 1 || got[0] != "pasted text" || !oks[0] {
		t.Errorf("got %q %v, want the text", got, oks)
	}

	// An empty clipboard has no text.
	c.GetText(context.Background(), func(text string, ok bool, err error) { got, oks = append(got, text), append(oks, ok) })
	api.reads[1]("", nil)
	p.run()
	if oks[1] {
		t.Error("an empty clipboard has text")
	}
}

func TestClipboardReadFailure(t *testing.T) {
	var p posts
	api := &fakeAPI{}
	c := clipboard{post: p.post, api: api}
	var err error
	c.GetText(context.Background(), func(_ string, _ bool, e error) { err = e })
	api.reads[0]("", errClipboardReadFailed("NotAllowedError: denied"))
	p.run()
	if got := tagOf(t, err); got != "WebClipboardReadFailed" {
		t.Errorf("tag = %s, want WebClipboardReadFailed", got)
	}
}

// TestClipboardReadEndsWithItsContext: a read that the browser leaves waiting for the user is ended by
// the context, once, and a promise that settles later is ignored.
func TestClipboardReadEndsWithItsContext(t *testing.T) {
	var p posts
	api := &fakeAPI{}
	c := clipboard{post: p.post, api: api}
	ctx, cancel := context.WithCancel(context.Background())
	answers := 0
	var err error
	c.GetText(ctx, func(_ string, _ bool, e error) { answers++; err = e })
	cancel()
	for deadline := time.Now().Add(2 * time.Second); p.count() == 0 && time.Now().Before(deadline); {
		time.Sleep(time.Millisecond)
	}
	api.reads[0]("late", nil)
	p.run()
	if answers != 1 || tagOf(t, err) != "ClipboardReadTimeout" {
		t.Errorf("answers = %d, err = %v, want one ClipboardReadTimeout", answers, err)
	}

	// A context that is over before the call: nothing is started.
	api2 := &fakeAPI{}
	c = clipboard{post: p.post, api: api2}
	c.GetText(ctx, func(_ string, _ bool, e error) { err = e })
	if len(api2.reads) != 0 {
		t.Error("a read was started with a context that is over")
	}
	p.run()
	if tagOf(t, err) != "ClipboardReadTimeout" {
		t.Errorf("err = %v", err)
	}
}

func TestClipboardWrite(t *testing.T) {
	var p posts
	api := &fakeAPI{}
	c := clipboard{post: p.post, api: api}
	var errs []error
	c.SetText(context.Background(), "copied", func(err error) { errs = append(errs, err) })
	c.SetContent(context.Background(), ui.ClipboardKindSystem, ui.ClipboardContent{Text: "again", HasText: true}, func(err error) { errs = append(errs, err) })
	c.SetContent(context.Background(), ui.ClipboardKindUnspecified, ui.ClipboardContent{}, func(err error) { errs = append(errs, err) })
	if len(api.writes) != 3 || api.writes[0] != "copied" || api.writes[1] != "again" || api.writes[2] != "" {
		t.Fatalf("writes = %q, want the text of each call, an empty content clearing the clipboard", api.writes)
	}
	api.wdone[0](nil)
	api.wdone[1](errClipboardWriteFailed("NotAllowedError: no user action"))
	api.wdone[2](nil)
	if len(errs) != 0 {
		t.Fatal("answered before the UI goroutine")
	}
	p.run()
	if errs[0] != nil || errs[2] != nil || tagOf(t, errs[1]) != "WebClipboardWriteFailed" {
		t.Errorf("answers = %v", errs)
	}
	c.SetText(context.Background(), "no callback", nil)
	api.wdone[3](nil)
	p.run()
}

func TestClipboardKindsAndFormats(t *testing.T) {
	var p posts
	api := &fakeAPI{}
	c := clipboard{post: p.post, api: api}
	var err error
	c.SetContent(context.Background(), ui.ClipboardKindPrimary, ui.ClipboardContent{Text: "x", HasText: true}, func(e error) { err = e })
	p.run()
	if tagOf(t, err) != "ClipboardPrimaryUnavailable" {
		t.Errorf("setting the primary selection: %v", err)
	}
	c.SetContent(context.Background(), ui.ClipboardKindSystem, ui.ClipboardContent{PNG: []byte{1}}, func(e error) { err = e })
	p.run()
	if tagOf(t, err) != "WebClipboardFormatUnsupported" {
		t.Errorf("setting an image: %v", err)
	}
	var content ui.ClipboardContent
	c.GetContent(context.Background(), ui.ClipboardKindPrimary, ui.ClipboardFormatText, func(_ ui.ClipboardContent, e error) { err = e })
	p.run()
	if tagOf(t, err) != "ClipboardPrimaryUnavailable" {
		t.Errorf("reading the primary selection: %v", err)
	}
	err = errors.New("Marker", "left over")
	c.GetContent(context.Background(), ui.ClipboardKindSystem, ui.ClipboardFormatPNG, func(cc ui.ClipboardContent, e error) { content, err = cc, e })
	p.run()
	if err != nil || !content.IsEmpty() || len(api.reads) != 0 {
		t.Errorf("reading an image: %+v, %v, %d reads: want an empty content without touching the clipboard", content, err, len(api.reads))
	}
	c.GetContent(context.Background(), ui.ClipboardKindSystem, "", func(cc ui.ClipboardContent, e error) { content, err = cc, e })
	api.reads[0]("hello", nil)
	p.run()
	if err != nil || content.Text != "hello" || !content.HasText {
		t.Errorf("reading text: %+v, %v", content, err)
	}
	c.GetContent(context.Background(), ui.ClipboardKindSystem, ui.ClipboardFormatText, func(cc ui.ClipboardContent, e error) { content, err = cc, e })
	api.reads[1]("", errClipboardReadFailed("x"))
	p.run()
	if tagOf(t, err) != "WebClipboardReadFailed" || !content.IsEmpty() {
		t.Errorf("a failed read: %+v, %v", content, err)
	}
	c.GetContent(context.Background(), ui.ClipboardKindSystem, ui.ClipboardFormatText, nil) // no callback: nothing happens
}
