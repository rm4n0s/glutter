//go:build windows

package windows

import (
	"context"
	"sync"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// winClipboardAPI is clipboardAPI over the Win32 functions of the clipboard.
type winClipboardAPI struct {
	pngOnce  sync.Once
	png      uint32
	pngErr   error
	htmlOnce sync.Once
	html     uint32
	htmlErr  error
}

var _ clipboardAPI = (*winClipboardAPI)(nil)

// Open opens the clipboard; ERROR_ACCESS_DENIED means that another window has it open.
func (a *winClipboardAPI) Open(owner uintptr) (busy bool, err error) {
	if err := win32.OpenClipboard(owner); err != nil {
		return win32.ClipboardBusyError(err), err
	}
	return false, nil
}

// Close closes the clipboard.
func (a *winClipboardAPI) Close() { win32.CloseClipboard() }

// Empty empties the clipboard.
func (a *winClipboardAPI) Empty() error { return win32.EmptyClipboard() }

// Set puts a copy of the data on the clipboard; the memory belongs to the clipboard once the call
// works.
func (a *winClipboardAPI) Set(format uint32, data []byte) error {
	h, err := win32.NewGlobal(data)
	if err != nil {
		return err
	}
	if err := win32.SetClipboardData(format, h); err != nil {
		_ = win32.GlobalFree(h) // the clipboard did not take it; the failure of SetClipboardData is the one to report
		return err
	}
	return nil
}

// Get copies the data of the format.
func (a *winClipboardAPI) Get(format uint32) ([]byte, bool, error) {
	h := win32.GetClipboardData(format)
	if h == 0 {
		return nil, false, nil
	}
	data, err := win32.GlobalBytes(h)
	if err != nil {
		return nil, false, err
	}
	return data, true, nil
}

// Has says whether the clipboard has the format.
func (a *winClipboardAPI) Has(format uint32) bool { return win32.IsClipboardFormatAvailable(format) }

// PNGFormat registers the clipboard format "PNG" the first time.
func (a *winClipboardAPI) PNGFormat() (uint32, error) {
	a.pngOnce.Do(func() { a.png, a.pngErr = win32.RegisterClipboardFormat("PNG") })
	return a.png, a.pngErr
}

// HTMLFormat registers the clipboard format "HTML Format" (CF_HTML) the first time.
func (a *winClipboardAPI) HTMLFormat() (uint32, error) {
	a.htmlOnce.Do(func() { a.html, a.htmlErr = win32.RegisterClipboardFormat("HTML Format") })
	return a.html, a.htmlErr
}

// clipboard is the clipboard of Windows (SPEC §25.7). A write runs on the UI thread with the
// service window as the owner and is complete when the call returns; a read runs on a thread of its
// own, since the program that owns the data can render it on demand. Every answer is given later on
// the UI thread, never before the call has returned.
//
// Flutter: shell/platform/windows/platform_handler.cc PlatformHandler
type clipboard struct {
	p    *Platform
	core clipboardCore
}

var _ platform.Clipboard = (*clipboard)(nil)

// newClipboard makes the clipboard of the platform.
func newClipboard(p *Platform) *clipboard {
	return &clipboard{p: p, core: clipboardCore{api: &winClipboardAPI{}, sleep: func(d time.Duration) { time.Sleep(d) }}}
}

// SetText puts the text on the clipboard.
//
// Flutter: shell/platform/windows/platform_handler.cc PlatformHandler::SetPlainText
func (c *clipboard) SetText(ctx context.Context, text string, done func(error)) {
	c.SetContent(ctx, ui.ClipboardKindSystem, ui.ClipboardContent{Text: text, HasText: true}, done)
}

// GetText reads the text of the clipboard; the text is empty and ok false when it has none. A read
// that ctx ends first fails with ClipboardReadTimeout, once.
//
// Flutter: shell/platform/windows/platform_handler.cc PlatformHandler::GetPlainText
func (c *clipboard) GetText(ctx context.Context, done func(string, bool, error)) {
	if done == nil {
		return
	}
	c.GetContent(ctx, ui.ClipboardKindSystem, ui.ClipboardFormatText, func(content ui.ClipboardContent, err error) {
		done(content.Text, content.HasText, err)
	})
}

// SetContent puts the content on the clipboard as one copy: its text, its HTML ("HTML Format"), its files (CF_HDROP) and its
// image (PNG and CF_DIBV5). The primary selection does not exist (ClipboardPrimaryUnavailable);
// a URI that is not a file URI is WindowsClipboardFormatUnsupported; a clipboard that stays busy
// is ClipboardBusy.
//
// Flutter: shell/platform/windows/platform_handler.cc PlatformHandler::SetPlainText
func (c *clipboard) SetContent(_ context.Context, kind ui.ClipboardKind, content ui.ClipboardContent, done func(error)) {
	var err error
	if kind == ui.ClipboardKindPrimary {
		err = errors.New(tagClipboardPrimaryUnavailable, "Windows has no primary selection")
	} else {
		err = c.core.write(c.p.service.Load(), content)
	}
	if done != nil {
		c.p.Post(func() { done(err) })
	}
}

// GetContent reads the clipboard in the format (the text when it is not given). The content is
// empty when the clipboard has no such format.
//
// Flutter: shell/platform/windows/platform_handler.cc PlatformHandler::GetPlainText
func (c *clipboard) GetContent(ctx context.Context, kind ui.ClipboardKind, format string, done func(ui.ClipboardContent, error)) {
	if done == nil {
		return
	}
	if kind == ui.ClipboardKindPrimary {
		err := errors.New(tagClipboardPrimaryUnavailable, "Windows has no primary selection")
		c.p.Post(func() { done(ui.ClipboardContent{}, err) })
		return
	}
	asyncRead(ctx, c.p.Post, func() (ui.ClipboardContent, error) { return c.core.read(format) }, done)
}

// Clipboard returns the clipboard of Windows.
//
// Flutter: shell/platform/windows/platform_handler.cc PlatformHandler
func (p *Platform) Clipboard() platform.Clipboard {
	if p.clip == nil {
		p.clip = newClipboard(p)
	}
	return p.clip
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagClipboardPrimaryUnavailable = "ClipboardPrimaryUnavailable"
)
