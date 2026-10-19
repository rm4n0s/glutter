package wayland

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/fakecomp"
)

type textResult struct {
	text string
	ok   bool
	err  error
}

// getText reads the clipboard on the UI thread and waits for the result.
func (e *keyEnv) getText(ctx context.Context) textResult {
	e.t.Helper()
	res := make(chan textResult, 1)
	e.p.Post(func() {
		e.p.Clipboard().GetText(ctx, func(text string, ok bool, err error) { res <- textResult{text, ok, err} })
	})
	select {
	case r := <-res:
		return r
	case <-time.After(testTimeout):
		e.t.Fatal("GetText never called back")
		return textResult{}
	}
}

// setText sets the clipboard on the UI thread and waits for the callback.
func (e *keyEnv) setText(text string) error {
	e.t.Helper()
	res := make(chan error, 1)
	e.p.Post(func() { e.p.Clipboard().SetText(context.Background(), text, func(err error) { res <- err }) })
	select {
	case err := <-res:
		e.settle()
		return err
	case <-time.After(testTimeout):
		e.t.Fatal("SetText never called back")
		return nil
	}
}

func clipboardEnv(t *testing.T, cfg fakecomp.Config) *keyEnv {
	t.Helper()
	e := startKeyboardNoFocus(t, cfg)
	e.srv.KeyboardEnter(e.surface) // the serial of the enter is the last input serial
	e.settle()
	return e
}

func TestClipboardReadsTheSelection(t *testing.T) {
	e := clipboardEnv(t, fakecomp.Config{Selection: "héllo wörld\nsecond line"})
	r := e.getText(context.Background())
	if r.err != nil || !r.ok || r.text != "héllo wörld\nsecond line" {
		t.Errorf("GetText = %+v", r)
	}
	if got := e.srv.Joined(); !strings.Contains(got, "wl_data_offer.receive mime=text/plain;charset=utf-8 fd=true") {
		t.Errorf("the request log lacks the receive:\n%s", got)
	}
}

func TestClipboardWithoutASelection(t *testing.T) {
	e := clipboardEnv(t, fakecomp.Config{})
	if r := e.getText(context.Background()); r.err != nil || r.ok || r.text != "" {
		t.Errorf("GetText with no selection = %+v", r)
	}
}

func TestClipboardSelectionWithoutText(t *testing.T) {
	e := clipboardEnv(t, fakecomp.Config{})
	e.srv.OfferSelection("not text", "image/png", "application/x-thing")
	e.settle()
	if r := e.getText(context.Background()); r.err != nil || r.ok {
		t.Errorf("GetText with an image selection = %+v", r)
	}
}

func TestClipboardFollowsTheSelection(t *testing.T) {
	e := clipboardEnv(t, fakecomp.Config{Selection: "first"})
	if r := e.getText(context.Background()); r.text != "first" {
		t.Fatalf("GetText = %+v, want first", r)
	}
	// Another application takes the clipboard; the old offer is destroyed.
	e.srv.OfferSelection("second", "UTF8_STRING", "text/plain")
	e.settle()
	r := e.getText(context.Background())
	if r.err != nil || !r.ok || r.text != "second" {
		t.Errorf("GetText = %+v, want second", r)
	}
	if got := e.srv.Joined(); !strings.Contains(got, "wl_data_offer.receive mime=UTF8_STRING") || !strings.Contains(got, "wl_data_offer.destroy") {
		t.Errorf("the request log lacks the UTF8_STRING receive or the destroy:\n%s", got)
	}
	// The clipboard is emptied.
	e.srv.ClearSelection()
	e.settle()
	if r := e.getText(context.Background()); r.err != nil || r.ok {
		t.Errorf("GetText after the clear = %+v", r)
	}
}

func TestClipboardSetTextOwnsTheSelection(t *testing.T) {
	e := clipboardEnv(t, fakecomp.Config{})
	if err := e.setText("copied ✓"); err != nil {
		t.Fatal(err)
	}
	log := e.srv.Joined()
	for _, want := range []string{
		"wl_data_device_manager.create_data_source",
		"wl_data_source.offer mime=text/plain;charset=utf-8",
		"wl_data_source.offer mime=UTF8_STRING",
		"wl_data_device.set_selection source=",
		"serial=1", // the serial of the keyboard enter
	} {
		if !strings.Contains(log, want) {
			t.Errorf("the request log lacks %q:\n%s", want, log)
		}
	}
	// The compositor asks the source for the data: it gets what was set.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()
	e.srv.Send(e.srv.ID("wl_data_source"), 1, "text/plain;charset=utf-8", fakecomp.FD(w.Fd())) // send(mime, fd)
	e.settle()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(r)
	if err != nil || string(data) != "copied ✓" {
		t.Errorf("the source sent %q, %v", data, err)
	}
	// The app reads its own selection without a round trip.
	if got := e.getText(context.Background()); got.err != nil || !got.ok || got.text != "copied ✓" {
		t.Errorf("GetText = %+v", got)
	}
	if strings.Contains(e.srv.Joined(), "wl_data_offer.receive") {
		t.Error("the app asked the compositor for its own text")
	}
}

func TestClipboardSetTextReplacesTheSource(t *testing.T) {
	e := clipboardEnv(t, fakecomp.Config{})
	if err := e.setText("one"); err != nil {
		t.Fatal(err)
	}
	if err := e.setText("two"); err != nil {
		t.Fatal(err)
	}
	log := e.srv.Joined()
	if strings.Count(log, "wl_data_device.set_selection") != 2 || strings.Count(log, "wl_data_source.destroy") != 1 {
		t.Errorf("want two set_selection and one destroy of the old source:\n%s", log)
	}
	if got := e.getText(context.Background()); got.text != "two" {
		t.Errorf("GetText = %+v, want two", got)
	}
}

func TestClipboardLosesTheSelection(t *testing.T) {
	e := clipboardEnv(t, fakecomp.Config{})
	if err := e.setText("mine"); err != nil {
		t.Fatal(err)
	}
	// Another application takes the clipboard: the compositor cancels the source.
	e.srv.Send(e.srv.ID("wl_data_source"), 2) // cancelled
	e.srv.OfferSelection("theirs", "text/plain;charset=utf-8")
	e.settle()
	if !strings.Contains(e.srv.Joined(), "wl_data_source.destroy") {
		t.Error("the cancelled source was not destroyed")
	}
	if got := e.getText(context.Background()); got.err != nil || got.text != "theirs" {
		t.Errorf("GetText = %+v, want theirs", got)
	}
}

func TestClipboardReadFailures(t *testing.T) {
	t.Run("an owner that never writes times out", func(t *testing.T) {
		e := clipboardEnv(t, fakecomp.Config{Selection: "never sent", StallSelection: true})
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		r := e.getText(ctx)
		appErr, ok := errors.FromError(r.err)
		if !ok || !appErr.HasRoute("readClipboard.ClipboardReadTimeout") {
			t.Errorf("GetText = %+v, want ClipboardReadTimeout", r)
		}
	})
}
