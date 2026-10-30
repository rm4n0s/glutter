//go:build linux && !android && glutter_display

package x11

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
)

// startClipboard connects a platform without windows to the X server and runs its
// loop. Two of them are two clients of the server.
func startClipboard(t *testing.T) *Platform {
	t.Helper()
	if os.Getenv("DISPLAY") == "" {
		t.Skip("DISPLAY is not set: run the X11 display tests under scripts/headless-x11.sh")
	}
	p, err := New("")
	if err != nil {
		t.Fatalf("could not connect to the X server: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx, platform.NopHandler{}) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Run: %v", err)
			}
		case <-time.After(displayTimeout):
			t.Error("Run did not return")
		}
		if err := p.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	if p.Clipboard() == nil {
		t.Fatal("the platform has no clipboard")
	}
	return p
}

type textResult struct {
	text string
	ok   bool
	err  error
}

func getText(t *testing.T, p *Platform, ctx context.Context) textResult {
	t.Helper()
	res := make(chan textResult, 1)
	p.Post(func() {
		p.Clipboard().GetText(ctx, func(text string, ok bool, err error) { res <- textResult{text, ok, err} })
	})
	select {
	case r := <-res:
		return r
	case <-time.After(displayTimeout):
		t.Fatal("GetText never called back")
		return textResult{}
	}
}

func setText(t *testing.T, p *Platform, text string) error {
	t.Helper()
	res := make(chan error, 1)
	p.Post(func() { p.Clipboard().SetText(context.Background(), text, func(err error) { res <- err }) })
	select {
	case err := <-res:
		return err
	case <-time.After(displayTimeout):
		t.Fatal("SetText never called back")
		return nil
	}
}

func TestClipboardTextPassesBetweenClients(t *testing.T) {
	owner, reader := startClipboard(t), startClipboard(t)
	for _, text := range []string{"hello", "héllo wörld ✓ 日本語", "two\nlines\n", ""} {
		if err := setText(t, owner, text); err != nil {
			t.Fatal(err)
		}
		r := getText(t, reader, context.Background())
		if r.err != nil || !r.ok || r.text != text {
			t.Errorf("reading %q gave %+v", text, r)
		}
	}
}

func TestClipboardOwnerReadsItsOwnText(t *testing.T) {
	p := startClipboard(t)
	if err := setText(t, p, "mine"); err != nil {
		t.Fatal(err)
	}
	if r := getText(t, p, context.Background()); r.err != nil || r.text != "mine" || !r.ok {
		t.Errorf("GetText = %+v", r)
	}
}

func TestClipboardFollowsTheOwner(t *testing.T) {
	a, b := startClipboard(t), startClipboard(t)
	if err := setText(t, a, "from a"); err != nil {
		t.Fatal(err)
	}
	if err := setText(t, b, "from b"); err != nil {
		t.Fatal(err)
	}
	// a lost the selection (SelectionClear) and reads b's text.
	if r := getText(t, a, context.Background()); r.err != nil || r.text != "from b" {
		t.Errorf("a read %+v, want from b", r)
	}
	if r := getText(t, b, context.Background()); r.text != "from b" {
		t.Errorf("b read %+v", r)
	}
}

func TestClipboardLargeTextUsesIncrementalTransfer(t *testing.T) {
	owner, reader := startClipboard(t), startClipboard(t)
	text := strings.Repeat("0123456789abcdef✓\n", 60000) // about 1.1 MB, above the INCR threshold
	if len(text) <= incrThreshold {
		t.Fatalf("the text is only %d bytes", len(text))
	}
	if err := setText(t, owner, text); err != nil {
		t.Fatal(err)
	}
	r := getText(t, reader, context.Background())
	if r.err != nil || !r.ok || r.text != text {
		t.Errorf("read %d bytes (ok %v, err %v), want %d", len(r.text), r.ok, r.err, len(text))
	}
}

func TestClipboardTooLargeToSet(t *testing.T) {
	p := startClipboard(t)
	err := setText(t, p, strings.Repeat("x", clipboardMaxBytes+1))
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != tagClipboardTooLarge {
		t.Errorf("SetText = %v, want ClipboardTooLarge", err)
	}
}

func TestClipboardReadersQueue(t *testing.T) {
	owner, reader := startClipboard(t), startClipboard(t)
	if err := setText(t, owner, "queued"); err != nil {
		t.Fatal(err)
	}
	const n = 5
	res := make(chan textResult, n)
	reader.Post(func() {
		for range n {
			reader.Clipboard().GetText(context.Background(), func(text string, ok bool, err error) { res <- textResult{text, ok, err} })
		}
	})
	for i := range n {
		select {
		case r := <-res:
			if r.err != nil || r.text != "queued" {
				t.Errorf("read %d gave %+v", i, r)
			}
		case <-time.After(displayTimeout):
			t.Fatalf("read %d never finished", i)
		}
	}
}

func TestClipboardCanceledRead(t *testing.T) {
	owner, reader := startClipboard(t), startClipboard(t)
	// A reader whose owner is busy: stop the owner's loop from answering by
	// canceling the read first.
	if err := setText(t, owner, "late"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := getText(t, reader, ctx)
	// Either the cancellation or the answer wins; a canceled read is an error.
	if r.err != nil {
		appErr, ok := errors.FromError(r.err)
		if !ok || appErr.Tag != tagClipboardReadTimeout {
			t.Errorf("error = %v, want ClipboardReadTimeout", r.err)
		}
	}
}
