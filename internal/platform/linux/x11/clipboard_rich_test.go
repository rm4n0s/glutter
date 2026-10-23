//go:build linux && !android && glutter_display

package x11

import (
	"bytes"
	"context"
	"slices"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/ui"
)

type contentResult struct {
	content ui.ClipboardContent
	err     error
}

func getContent(t *testing.T, p *Platform, kind ui.ClipboardKind, format string) contentResult {
	t.Helper()
	res := make(chan contentResult, 1)
	p.Post(func() {
		p.Clipboard().GetContent(context.Background(), kind, format, func(c ui.ClipboardContent, err error) { res <- contentResult{c, err} })
	})
	select {
	case r := <-res:
		return r
	case <-time.After(displayTimeout):
		t.Fatal("GetContent never called back")
		return contentResult{}
	}
}

func setContent(t *testing.T, p *Platform, kind ui.ClipboardKind, c ui.ClipboardContent) error {
	t.Helper()
	res := make(chan error, 1)
	p.Post(func() { p.Clipboard().SetContent(context.Background(), kind, c, func(err error) { res <- err }) })
	select {
	case err := <-res:
		return err
	case <-time.After(displayTimeout):
		t.Fatal("SetContent never called back")
		return nil
	}
}

func TestPrimarySelectionPassesBetweenClients(t *testing.T) {
	owner, reader := startClipboard(t), startClipboard(t)
	if err := setContent(t, owner, ui.ClipboardKindPrimary, ui.ClipboardContent{Text: "selected ✓", HasText: true}); err != nil {
		t.Fatal(err)
	}
	if r := getContent(t, reader, ui.ClipboardKindPrimary, ui.ClipboardFormatText); r.err != nil || r.content.Text != "selected ✓" || !r.content.HasText {
		t.Errorf("primary = %+v", r)
	}
	// The clipboard is another selection.
	if r := getContent(t, reader, ui.ClipboardKindSystem, ui.ClipboardFormatText); r.err != nil || !r.content.IsEmpty() {
		t.Errorf("clipboard = %+v, want empty", r)
	}
	// Another client takes the primary selection; the first one reads it.
	if err := setContent(t, reader, ui.ClipboardKindPrimary, ui.ClipboardContent{Text: "newer", HasText: true}); err != nil {
		t.Fatal(err)
	}
	if r := getContent(t, owner, ui.ClipboardKindPrimary, ui.ClipboardFormatText); r.content.Text != "newer" {
		t.Errorf("owner read %+v, want newer", r)
	}
}

func TestClipboardPassesImagesAndFiles(t *testing.T) {
	owner, reader := startClipboard(t), startClipboard(t)
	png := append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0, 1, 2, 255}, 100)...)
	uris := []string{"file:///home/me/a.txt", "file:///home/me/b%20c.png"}
	content := ui.ClipboardContent{Text: "a.txt\nb c.png", HasText: true, PNG: png, URIs: uris}
	if err := setContent(t, owner, ui.ClipboardKindSystem, content); err != nil {
		t.Fatal(err)
	}
	if r := getContent(t, reader, ui.ClipboardKindSystem, ui.ClipboardFormatPNG); r.err != nil || !bytes.Equal(r.content.PNG, png) {
		t.Errorf("PNG = %d bytes, err %v", len(r.content.PNG), r.err)
	}
	if r := getContent(t, reader, ui.ClipboardKindSystem, ui.ClipboardFormatURIList); r.err != nil || !slices.Equal(r.content.URIs, uris) {
		t.Errorf("URIs = %+v", r)
	}
	if r := getContent(t, reader, ui.ClipboardKindSystem, ui.ClipboardFormatText); r.err != nil || r.content.Text != "a.txt\nb c.png" {
		t.Errorf("text = %+v", r)
	}
}

func TestClipboardFormatThatTheOwnerLacksIsEmpty(t *testing.T) {
	owner, reader := startClipboard(t), startClipboard(t)
	if err := setContent(t, owner, ui.ClipboardKindSystem, ui.ClipboardContent{Text: "text only", HasText: true}); err != nil {
		t.Fatal(err)
	}
	for _, format := range []string{ui.ClipboardFormatPNG, ui.ClipboardFormatURIList} {
		if r := getContent(t, reader, ui.ClipboardKindSystem, format); r.err != nil || !r.content.IsEmpty() {
			t.Errorf("%s = %+v, want empty", format, r)
		}
	}
	// An image alone has no text.
	if err := setContent(t, owner, ui.ClipboardKindSystem, ui.ClipboardContent{PNG: []byte("png")}); err != nil {
		t.Fatal(err)
	}
	if r := getContent(t, reader, ui.ClipboardKindSystem, ui.ClipboardFormatText); r.err != nil || !r.content.IsEmpty() {
		t.Errorf("text of an image = %+v, want empty", r)
	}
}

func TestClipboardLargeImageUsesIncrementalTransfer(t *testing.T) {
	owner, reader := startClipboard(t), startClipboard(t)
	png := bytes.Repeat([]byte{0x89, 'P', 'N', 'G', 1, 2, 3, 4}, 200000) // 1.6 MB
	if len(png) <= incrThreshold {
		t.Fatalf("the image is only %d bytes", len(png))
	}
	if err := setContent(t, owner, ui.ClipboardKindSystem, ui.ClipboardContent{PNG: png}); err != nil {
		t.Fatal(err)
	}
	if r := getContent(t, reader, ui.ClipboardKindSystem, ui.ClipboardFormatPNG); r.err != nil || !bytes.Equal(r.content.PNG, png) {
		t.Errorf("read %d bytes, err %v, want %d", len(r.content.PNG), r.err, len(png))
	}
}

func TestClipboardOwnerReadsItsOwnContent(t *testing.T) {
	p := startClipboard(t)
	uris := []string{"file:///x"}
	if err := setContent(t, p, ui.ClipboardKindPrimary, ui.ClipboardContent{URIs: uris, PNG: []byte("p")}); err != nil {
		t.Fatal(err)
	}
	if r := getContent(t, p, ui.ClipboardKindPrimary, ui.ClipboardFormatURIList); !slices.Equal(r.content.URIs, uris) {
		t.Errorf("URIs = %+v", r)
	}
}

func TestURIListHelpers(t *testing.T) {
	if got := parseURIList([]byte("# c\r\nfile:///a\r\n\r\nfile:///b")); !slices.Equal(got, []string{"file:///a", "file:///b"}) {
		t.Errorf("parseURIList = %q", got)
	}
	if got := formatURIList([]string{"file:///a"}); got != "file:///a\r\n" {
		t.Errorf("formatURIList = %q", got)
	}
}
