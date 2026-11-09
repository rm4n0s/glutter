//go:build darwin && !ios && glutter_display

package macos

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"reflect"
	"testing"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/ui"
)

// The clipboard tests use a pasteboard of their own (pasteboardWithUniqueName), so that they never read or
// change the clipboard of the user.

func withClipboard(t *testing.T, body func(c *clipboard, s *session)) {
	t.Helper()
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		s.onUI(func() {
			s.p.clip.pb = objcx.Retain(class("NSPasteboard").Send(sel("pasteboardWithUniqueName")))
		})
		body(&s.p.clip, s)
		s.onUI(func() {
			s.p.clip.board().Send(sel("releaseGlobally"))
			objcx.Release(s.p.clip.pb)
			s.p.clip.pb = 0
		})
	})
}

func read(t *testing.T, s *session, c *clipboard, format string) ui.ClipboardContent {
	t.Helper()
	var got ui.ClipboardContent
	var gotErr error
	s.onUI(func() {
		c.GetContent(context.Background(), ui.ClipboardKindSystem, format, func(content ui.ClipboardContent, err error) { got, gotErr = content, err })
	})
	s.onUI(func() {}) // the answer was posted: it runs before this
	if gotErr != nil {
		t.Fatalf("GetContent(%s): %v", format, gotErr)
	}
	return got
}

func write(t *testing.T, s *session, c *clipboard, content ui.ClipboardContent) {
	t.Helper()
	var werr error
	called := false
	s.onUI(func() {
		c.SetContent(context.Background(), ui.ClipboardKindSystem, content, func(err error) { werr, called = err, true })
		if called {
			t.Error("done was called before SetContent returned")
		}
	})
	s.onUI(func() {}) // the answer was posted: it runs before this
	s.onUI(func() {
		if !called {
			t.Error("done was never called")
		}
	})
	if werr != nil {
		t.Fatalf("SetContent: %v", werr)
	}
}

func testPNG(t *testing.T) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 4, 3))
	for y := 0; y < 3; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.NRGBA{R: uint8(60 * x), G: uint8(80 * y), B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestPasteboardRoundTripsEveryFormat(t *testing.T) {
	withClipboard(t, func(c *clipboard, s *session) {
		img := testPNG(t)
		want := ui.ClipboardContent{
			Text: "héllo 😀\nworld", HasText: true,
			HTML: "<b>héllo</b>",
			PNG:  img,
			URIs: []string{"file:///tmp/a%20b.txt", "file:///tmp/c.txt"},
		}
		write(t, s, c, want)
		if got := read(t, s, c, ui.ClipboardFormatText); got.Text != want.Text || !got.HasText {
			t.Errorf("text = %q, %v", got.Text, got.HasText)
		}
		if got := read(t, s, c, ui.ClipboardFormatHTML); got.HTML != want.HTML {
			t.Errorf("html = %q, want %q", got.HTML, want.HTML)
		}
		if got := read(t, s, c, ui.ClipboardFormatPNG); !bytes.Equal(got.PNG, img) {
			t.Errorf("the PNG changed: %d bytes, want %d", len(got.PNG), len(img))
		}
		if got := read(t, s, c, ui.ClipboardFormatURIList); !reflect.DeepEqual(got.URIs, want.URIs) {
			t.Errorf("uris = %q, want %q", got.URIs, want.URIs)
		}
	})
}

func TestPasteboardWithoutTheFormatIsEmpty(t *testing.T) {
	withClipboard(t, func(c *clipboard, s *session) {
		write(t, s, c, ui.ClipboardContent{Text: "only text", HasText: true})
		for _, f := range []string{ui.ClipboardFormatHTML, ui.ClipboardFormatPNG, ui.ClipboardFormatURIList} {
			if got := read(t, s, c, f); !got.IsEmpty() {
				t.Errorf("%s: content = %+v, want empty", f, got)
			}
		}
		write(t, s, c, ui.ClipboardContent{}) // an empty content clears
		if got := read(t, s, c, ui.ClipboardFormatText); got.HasText {
			t.Errorf("after a clear the text is %q", got.Text)
		}
	})
}

func TestPasteboardTextIsReadAgainWhenItChanges(t *testing.T) {
	withClipboard(t, func(c *clipboard, s *session) {
		get := func() (string, bool) {
			var text string
			var ok bool
			s.onUI(func() { c.GetText(context.Background(), func(tx string, o bool, _ error) { text, ok = tx, o }) })
			s.onUI(func() {})
			return text, ok
		}
		if _, ok := get(); ok {
			t.Error("an empty pasteboard has text")
		}
		var count int64
		s.onUI(func() {
			c.board().Send(sel("clearContents"))
			ns, ty := objcx.NewString("from another app"), objcx.NewString(typeText)
			c.board().Send(sel("setString:forType:"), ns, ty)
			objcx.Release(ns)
			objcx.Release(ty)
			count = send[int64](c.board(), "changeCount")
		})
		if text, ok := get(); !ok || text != "from another app" {
			t.Errorf("after a write by another app: %q, %v", text, ok)
		}
		var cached string
		s.onUI(func() { cached = c.text })
		if cached != "from another app" {
			t.Errorf("the text was not kept for change count %d", count)
		}
		write(t, s, c, ui.ClipboardContent{Text: "ours", HasText: true})
		if text, _ := get(); text != "ours" {
			t.Errorf("after our write: %q", text)
		}
	})
}

func TestPasteboardTIFFIsReadAsPNG(t *testing.T) {
	withClipboard(t, func(c *clipboard, s *session) {
		img := testPNG(t)
		s.onUI(func() {
			data := newData(img)
			defer objcx.Release(data)
			rep := class("NSBitmapImageRep").Send(sel("alloc")).Send(sel("initWithData:"), data)
			defer objcx.Release(rep)
			tiff := send[objc.ID](rep, "TIFFRepresentation")
			c.board().Send(sel("clearContents"))
			ty := objcx.NewString(typeTIFF)
			c.board().Send(sel("setData:forType:"), tiff, ty)
			objcx.Release(ty)
		})
		got := read(t, s, c, ui.ClipboardFormatPNG)
		decoded, err := png.Decode(bytes.NewReader(got.PNG))
		if err != nil {
			t.Fatalf("the TIFF did not become a PNG: %v (%d bytes)", err, len(got.PNG))
		}
		if b := decoded.Bounds(); b.Dx() != 4 || b.Dy() != 3 {
			t.Errorf("the image is %v, want 4 x 3", b)
		}
	})
}
