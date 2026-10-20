package wayland

import (
	"bytes"
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/fakecomp"
	"github.com/rm4n0s/glutter/ui"
)

type contentResult struct {
	content ui.ClipboardContent
	err     error
}

func (e *keyEnv) getContent(kind ui.ClipboardKind, format string) contentResult {
	e.t.Helper()
	res := make(chan contentResult, 1)
	e.p.Post(func() {
		e.p.Clipboard().GetContent(context.Background(), kind, format, func(c ui.ClipboardContent, err error) { res <- contentResult{c, err} })
	})
	select {
	case r := <-res:
		return r
	case <-time.After(testTimeout):
		e.t.Fatal("GetContent never called back")
		return contentResult{}
	}
}

func (e *keyEnv) setContent(kind ui.ClipboardKind, c ui.ClipboardContent) error {
	e.t.Helper()
	res := make(chan error, 1)
	e.p.Post(func() { e.p.Clipboard().SetContent(context.Background(), kind, c, func(err error) { res <- err }) })
	select {
	case err := <-res:
		e.settle()
		return err
	case <-time.After(testTimeout):
		e.t.Fatal("SetContent never called back")
		return nil
	}
}

var pngBytes = []byte("\x89PNG\r\n\x1a\nfake image bytes")

func TestClipboardReadsImagesAndFiles(t *testing.T) {
	e := clipboardEnv(t, fakecomp.Config{})
	e.srv.OfferSelectionData(map[string][]byte{
		"image/png":     pngBytes,
		"text/uri-list": []byte("# a comment\r\nfile:///home/me/a.txt\r\nfile:///home/me/b%20c.png\r\n\r\n"),
		"UTF8_STRING":   []byte("a.txt\nb c.png"),
	})
	e.settle()
	if r := e.getContent(ui.ClipboardKindSystem, ui.ClipboardFormatPNG); r.err != nil || !bytes.Equal(r.content.PNG, pngBytes) {
		t.Errorf("PNG = %+v", r)
	}
	r := e.getContent(ui.ClipboardKindSystem, ui.ClipboardFormatURIList)
	if want := []string{"file:///home/me/a.txt", "file:///home/me/b%20c.png"}; r.err != nil || !slices.Equal(r.content.URIs, want) {
		t.Errorf("URIs = %+v, want %v", r, want)
	}
	if r := e.getContent(ui.ClipboardKindSystem, ui.ClipboardFormatText); r.err != nil || !r.content.HasText || r.content.Text != "a.txt\nb c.png" {
		t.Errorf("text = %+v", r)
	}
}

func TestClipboardFormatThatTheOwnerLacksIsEmpty(t *testing.T) {
	e := clipboardEnv(t, fakecomp.Config{})
	e.srv.OfferSelectionData(map[string][]byte{"UTF8_STRING": []byte("only text")})
	e.settle()
	for _, format := range []string{ui.ClipboardFormatPNG, ui.ClipboardFormatURIList, "application/x-unknown"} {
		if r := e.getContent(ui.ClipboardKindSystem, format); r.err != nil || !r.content.IsEmpty() {
			t.Errorf("%s = %+v, want empty", format, r)
		}
	}
	if strings.Contains(e.srv.Joined(), "wl_data_offer.receive mime=image/png") {
		t.Error("the app asked for a format that the owner does not offer")
	}
}

func TestClipboardSetContentOffersEveryFormat(t *testing.T) {
	e := clipboardEnv(t, fakecomp.Config{})
	content := ui.ClipboardContent{Text: "a.txt", HasText: true, PNG: pngBytes, URIs: []string{"file:///home/me/a.txt", "file:///home/me/b.txt"}}
	if err := e.setContent(ui.ClipboardKindSystem, content); err != nil {
		t.Fatal(err)
	}
	mimes := e.srv.ClientSelectionMimes(false)
	for _, want := range []string{"text/plain;charset=utf-8", "text/plain", "UTF8_STRING", "image/png", "text/uri-list"} {
		if !slices.Contains(mimes, want) {
			t.Errorf("the source lacks %q: %v", want, mimes)
		}
	}
	if data, err := e.srv.ReadClientSelection(false, "image/png"); err != nil || !bytes.Equal(data, pngBytes) {
		t.Errorf("image/png = %q, %v", data, err)
	}
	if data, err := e.srv.ReadClientSelection(false, "text/uri-list"); err != nil || string(data) != "file:///home/me/a.txt\r\nfile:///home/me/b.txt\r\n" {
		t.Errorf("text/uri-list = %q, %v", data, err)
	}
	if data, err := e.srv.ReadClientSelection(false, "UTF8_STRING"); err != nil || string(data) != "a.txt" {
		t.Errorf("UTF8_STRING = %q, %v", data, err)
	}
	// The app reads what it owns without a round trip, in each format.
	if r := e.getContent(ui.ClipboardKindSystem, ui.ClipboardFormatPNG); !bytes.Equal(r.content.PNG, pngBytes) {
		t.Errorf("own PNG = %+v", r)
	}
	if strings.Contains(e.srv.Joined(), "wl_data_offer.receive") {
		t.Error("the app asked the compositor for its own content")
	}
}

func TestClipboardOnlyOffersTheFormatsItHas(t *testing.T) {
	e := clipboardEnv(t, fakecomp.Config{})
	if err := e.setContent(ui.ClipboardKindSystem, ui.ClipboardContent{PNG: pngBytes}); err != nil {
		t.Fatal(err)
	}
	if mimes := e.srv.ClientSelectionMimes(false); !slices.Equal(mimes, []string{"image/png"}) {
		t.Errorf("mimes = %v, want only image/png", mimes)
	}
}

func TestPrimarySelectionReadsTheSelectionOfAnotherApp(t *testing.T) {
	e := clipboardEnv(t, fakecomp.Config{PrimarySelection: true})
	r := e.getContent(ui.ClipboardKindPrimary, ui.ClipboardFormatText)
	if r.err != nil || !r.content.IsEmpty() {
		t.Fatalf("empty primary = %+v", r)
	}
	e.srv.OfferPrimary(map[string][]byte{"text/plain;charset=utf-8": []byte("selected words"), "UTF8_STRING": []byte("selected words")})
	e.settle()
	r = e.getContent(ui.ClipboardKindPrimary, ui.ClipboardFormatText)
	if r.err != nil || r.content.Text != "selected words" || !r.content.HasText {
		t.Errorf("primary = %+v", r)
	}
	if !strings.Contains(e.srv.Joined(), "zwp_primary_selection_offer_v1.receive mime=text/plain;charset=utf-8") {
		t.Errorf("no receive on the primary offer:\n%s", e.srv.Joined())
	}
	// The clipboard is another selection: it stays empty.
	if r := e.getContent(ui.ClipboardKindSystem, ui.ClipboardFormatText); r.err != nil || !r.content.IsEmpty() {
		t.Errorf("the clipboard = %+v, want empty", r)
	}
	e.srv.ClearPrimary()
	e.settle()
	if r := e.getContent(ui.ClipboardKindPrimary, ui.ClipboardFormatText); r.err != nil || !r.content.IsEmpty() {
		t.Errorf("primary after the clear = %+v", r)
	}
}

func TestPrimarySelectionReadsALateSelection(t *testing.T) {
	e := clipboardEnv(t, fakecomp.Config{PrimarySelection: true})
	e.srv.OfferPrimary(map[string][]byte{"UTF8_STRING": []byte("later")})
	e.settle()
	if r := e.getContent(ui.ClipboardKindPrimary, ui.ClipboardFormatText); r.content.Text != "later" {
		t.Errorf("primary = %+v", r)
	}
}

func TestPrimarySelectionSetContentOwnsIt(t *testing.T) {
	e := clipboardEnv(t, fakecomp.Config{PrimarySelection: true})
	if err := e.setContent(ui.ClipboardKindPrimary, ui.ClipboardContent{Text: "picked", HasText: true}); err != nil {
		t.Fatal(err)
	}
	log := e.srv.Joined()
	for _, want := range []string{"zwp_primary_selection_device_manager_v1.create_source", "zwp_primary_selection_source_v1.offer mime=UTF8_STRING", "zwp_primary_selection_device_v1.set_selection source="} {
		if !strings.Contains(log, want) {
			t.Errorf("the request log lacks %q:\n%s", want, log)
		}
	}
	if strings.Contains(log, "wl_data_device.set_selection") {
		t.Error("the primary selection went to the clipboard")
	}
	if data, err := e.srv.ReadClientSelection(true, "UTF8_STRING"); err != nil || string(data) != "picked" {
		t.Errorf("the source sent %q, %v", data, err)
	}
	if r := e.getContent(ui.ClipboardKindPrimary, ui.ClipboardFormatText); r.content.Text != "picked" {
		t.Errorf("own primary = %+v", r)
	}
	if strings.Contains(log, "zwp_primary_selection_offer_v1.receive") {
		t.Error("the app asked the compositor for its own primary selection")
	}
	// Another app takes it: the source is cancelled and destroyed.
	e.srv.CancelClientSelection(true)
	e.srv.OfferPrimary(map[string][]byte{"UTF8_STRING": []byte("theirs")})
	e.settle()
	if !strings.Contains(e.srv.Joined(), "zwp_primary_selection_source_v1.destroy") {
		t.Error("the cancelled source was not destroyed")
	}
	if r := e.getContent(ui.ClipboardKindPrimary, ui.ClipboardFormatText); r.content.Text != "theirs" {
		t.Errorf("primary after the loss = %+v", r)
	}
}

func TestPrimarySelectionReplacesItsSource(t *testing.T) {
	e := clipboardEnv(t, fakecomp.Config{PrimarySelection: true})
	for _, text := range []string{"one", "two"} {
		if err := e.setContent(ui.ClipboardKindPrimary, ui.ClipboardContent{Text: text, HasText: true}); err != nil {
			t.Fatal(err)
		}
	}
	if n := strings.Count(e.srv.Joined(), "zwp_primary_selection_source_v1.destroy"); n != 1 {
		t.Errorf("destroyed %d sources, want 1", n)
	}
	if data, err := e.srv.ReadClientSelection(true, "UTF8_STRING"); err != nil || string(data) != "two" {
		t.Errorf("the source sent %q, %v", data, err)
	}
}

func TestPrimarySelectionWithoutACompositorManager(t *testing.T) {
	e := clipboardEnv(t, fakecomp.Config{})
	for name, err := range map[string]error{
		"get": e.getContent(ui.ClipboardKindPrimary, ui.ClipboardFormatText).err,
		"set": e.setContent(ui.ClipboardKindPrimary, ui.ClipboardContent{Text: "x", HasText: true}),
	} {
		appErr, ok := errors.FromError(err)
		if !ok || !appErr.HasRoute("ClipboardPrimaryUnavailable") {
			t.Errorf("%s: err = %v, want ClipboardPrimaryUnavailable", name, err)
		}
	}
}

func TestURIListHelpers(t *testing.T) {
	tests := []struct {
		name string
		data string
		want []string
	}{
		{"crlf", "file:///a\r\nfile:///b\r\n", []string{"file:///a", "file:///b"}},
		{"lf and comments", "# c\nfile:///a\n\n#x\nfile:///b", []string{"file:///a", "file:///b"}},
		{"empty", "", nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseURIList([]byte(tc.data)); !slices.Equal(got, tc.want) {
				t.Errorf("parseURIList = %q, want %q", got, tc.want)
			}
		})
	}
	if got := formatURIList([]string{"file:///a", "file:///b"}); got != "file:///a\r\nfile:///b\r\n" {
		t.Errorf("formatURIList = %q", got)
	}
}
