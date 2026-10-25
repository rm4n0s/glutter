//go:build windows && glutter_wine

package windows

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/draw"
	"image/png"
	"io"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// These tests use the real clipboard under Wine (SPEC §25.7, §25.13). Another program is a second
// window of the process, and a second process: this test binary again, started with the name of its
// part in GLUTTER_CLIP_HELPER (TestWineClipboardHelper).

// messageWindow makes a window of the system class STATIC that never shows, to own what it puts on
// the clipboard.
func messageWindow(t testing.TB) uintptr {
	t.Helper()
	instance, err := win32.GetModuleHandle("")
	if err != nil {
		t.Fatal(err)
	}
	hwnd, err := win32.CreateWindowEx(0, "STATIC", "", 0, 0, 0, 0, 0, 0, 0, instance, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = win32.DestroyWindow(hwnd) })
	return hwnd
}

// seq runs the steps one after the other on the UI thread: each calls next when it is done.
func seq(steps ...func(next func())) {
	var run func(i int)
	run = func(i int) {
		if i < len(steps) {
			steps[i](func() { run(i + 1) })
		}
	}
	run(0)
}

// clipRun runs the loop of a platform with no window and calls fn on the UI thread; fn calls finish.
func clipRun(t *testing.T, limit time.Duration, fn func(p *Platform, c *clipboard, finish func())) {
	t.Helper()
	p, r := newTestPlatform(t)
	runFor(t, p, r, limit, func(cancel context.CancelFunc) {
		fn(p, p.Clipboard().(*clipboard), cancel)
	})
}

// wineImage is a small PNG with see-through pixels.
func wineImage(t testing.TB) []byte { return encodePNG(t, testImage(6, 4)) }

// pixelsOf decodes a PNG into its NRGBA pixels.
func pixelsOf(t testing.TB, data []byte) [][4]uint8 {
	t.Helper()
	decoded, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewNRGBA(decoded.Bounds())
	draw.Draw(img, img.Bounds(), decoded, decoded.Bounds().Min, draw.Src)
	var out [][4]uint8
	for y := range img.Bounds().Dy() {
		for x := range img.Bounds().Dx() {
			c := img.NRGBAAt(x, y)
			out = append(out, [4]uint8{c.R, c.G, c.B, c.A})
		}
	}
	return out
}

func TestWineClipboardRoundTrip(t *testing.T) {
	png := wineImage(t)
	tests := []struct {
		name    string
		content ui.ClipboardContent
	}{
		{"text", ui.ClipboardContent{Text: "Hello, wörld! 日本語 𠮷", HasText: true}},
		{"text with line breaks", ui.ClipboardContent{Text: "one\ntwo\n\nthree", HasText: true}},
		{"an empty text", ui.ClipboardContent{HasText: true}},
		{"one file", ui.ClipboardContent{URIs: []string{"file:///C:/Users/me/a%20b.txt"}}},
		{"several files and a network path", ui.ClipboardContent{URIs: []string{"file:///C:/x/1.txt", "file:///D:/y/%C3%A9.md", "file://server/share/d/3.bin"}}},
		{"an image", ui.ClipboardContent{PNG: png}},
		{"text, files and an image are one copy", ui.ClipboardContent{Text: "all", HasText: true, URIs: []string{"file:///C:/z.txt"}, PNG: png}},
	}
	clipRun(t, 60*time.Second, func(p *Platform, c *clipboard, finish func()) {
		var steps []func(next func())
		for _, tt := range tests {
			steps = append(steps, func(next func()) {
				c.SetContent(context.Background(), ui.ClipboardKindSystem, tt.content, func(err error) {
					if err != nil {
						t.Errorf("%s: SetContent: %v", tt.name, err)
						next()
						return
					}
					read := func(format string, want ui.ClipboardContent, then func()) {
						c.GetContent(context.Background(), ui.ClipboardKindSystem, format, func(got ui.ClipboardContent, err error) {
							if err != nil {
								t.Errorf("%s: GetContent(%s): %v", tt.name, format, err)
							} else if format == ui.ClipboardFormatPNG {
								if len(want.PNG) == 0 && len(got.PNG) != 0 || len(want.PNG) != 0 && !reflect.DeepEqual(pixelsOf(t, got.PNG), pixelsOf(t, want.PNG)) {
									t.Errorf("%s: the image that came back differs", tt.name)
								}
							} else if !reflect.DeepEqual(got, want) {
								t.Errorf("%s: GetContent(%s) = %+v, want %+v", tt.name, format, got, want)
							}
							then()
						})
					}
					read(ui.ClipboardFormatText, ui.ClipboardContent{Text: tt.content.Text, HasText: tt.content.HasText}, func() {
						read(ui.ClipboardFormatURIList, ui.ClipboardContent{URIs: tt.content.URIs}, func() {
							read(ui.ClipboardFormatPNG, ui.ClipboardContent{PNG: tt.content.PNG}, next)
						})
					})
				})
			})
		}
		steps = append(steps, func(next func()) { finish() })
		seq(steps...)
	})
}

func TestWineClipboardTextUsesCRLFOnTheClipboardAndTheOtherFormatsAreThere(t *testing.T) {
	clipRun(t, 30*time.Second, func(p *Platform, c *clipboard, finish func()) {
		defer finish()
		content := ui.ClipboardContent{Text: "a\nb", HasText: true, URIs: []string{"file:///C:/f.txt"}, PNG: wineImage(t)}
		if err := c.core.write(p.service.Load(), content); err != nil {
			t.Fatal(err)
		}
		api := c.core.api
		raw, present, err := rawFormat(api, win32.CFUnicodeText)
		if err != nil || !present || string(raw) != string(textBytes(utf16Of("a\r\nb\x00"))) {
			t.Errorf("CF_UNICODETEXT = %x, %v, %v", raw, present, err)
		}
		for name, f := range map[string]uint32{"CF_HDROP": win32.CFHDrop, "CF_DIBV5": win32.CFDIBV5} {
			if !api.Has(f) {
				t.Errorf("%s is not on the clipboard", name)
			}
		}
		png, _ := api.PNGFormat()
		if !api.Has(png) {
			t.Error("the registered format PNG is not on the clipboard")
		}
		// The system makes CF_DIB and CF_BITMAP from CF_DIBV5 for the programs that ask for them.
		if !api.Has(win32.CFDIB) {
			t.Log("this Windows did not synthesize CF_DIB")
		}
		if owner := win32.GetClipboardOwner(); owner != p.service.Load() {
			t.Errorf("the owner of the clipboard is %#x, want the service window %#x", owner, p.service.Load())
		}
	})
}

// rawFormat reads a format with the clipboard opened, the way a reader would.
func rawFormat(api clipboardAPI, format uint32) ([]byte, bool, error) {
	if busy, err := api.Open(0); err != nil {
		_ = busy
		return nil, false, err
	}
	defer api.Close()
	return api.Get(format)
}

func TestWineClipboardAnotherWindowOfTheProcess(t *testing.T) {
	other := messageWindow(t)
	png := wineImage(t)
	clipRun(t, 60*time.Second, func(p *Platform, c *clipboard, finish func()) {
		otherCore := &clipboardCore{api: &winClipboardAPI{}, sleep: time.Sleep}
		expect := func(format string, want ui.ClipboardContent, next func()) {
			c.GetContent(context.Background(), ui.ClipboardKindSystem, format, func(got ui.ClipboardContent, err error) {
				if err != nil {
					t.Errorf("GetContent(%s): %v", format, err)
				} else if format == ui.ClipboardFormatPNG {
					if !reflect.DeepEqual(pixelsOf(t, got.PNG), pixelsOf(t, want.PNG)) {
						t.Error("the image from the other window differs")
					}
				} else if !reflect.DeepEqual(got, want) {
					t.Errorf("GetContent(%s) = %+v, want %+v", format, got, want)
				}
				next()
			})
		}
		seq(
			func(next func()) {
				if err := otherCore.write(other, ui.ClipboardContent{Text: "from the other window\nline 2", HasText: true, URIs: []string{"file:///C:/other.txt"}, PNG: png}); err != nil {
					t.Fatal(err)
				}
				if win32.GetClipboardOwner() != other {
					t.Errorf("the owner is %#x, want the other window %#x", win32.GetClipboardOwner(), other)
				}
				expect(ui.ClipboardFormatText, ui.ClipboardContent{Text: "from the other window\nline 2", HasText: true}, func() {
					expect(ui.ClipboardFormatURIList, ui.ClipboardContent{URIs: []string{"file:///C:/other.txt"}}, func() {
						expect(ui.ClipboardFormatPNG, ui.ClipboardContent{PNG: png}, next)
					})
				})
			},
			func(next func()) {
				// Our write replaces what the other window put there, and the other window reads it.
				c.SetText(context.Background(), "ours", func(err error) {
					if err != nil {
						t.Fatal(err)
					}
					got, err := otherCore.read(ui.ClipboardFormatText)
					if err != nil || got.Text != "ours" {
						t.Errorf("the other window read %+v, %v", got, err)
					}
					if got, _ := otherCore.read(ui.ClipboardFormatPNG); len(got.PNG) != 0 {
						t.Error("the image of the other window survived a write of text")
					}
					next()
				})
			},
			func(next func()) { finish() },
		)
	})
}

// helperContent is what the helper process writes or reads.
type helperContent struct {
	Text    string
	HasText bool
	PNG     string // base64
	URIs    []string
}

func (h helperContent) content() ui.ClipboardContent {
	png, _ := base64.StdEncoding.DecodeString(h.PNG)
	return ui.ClipboardContent{Text: h.Text, HasText: h.HasText, PNG: png, URIs: h.URIs}
}

func helperOf(c ui.ClipboardContent) helperContent {
	return helperContent{Text: c.Text, HasText: c.HasText, PNG: base64.StdEncoding.EncodeToString(c.PNG), URIs: c.URIs}
}

// TestWineClipboardHelper is the program that the tests of another process start: it does nothing
// unless GLUTTER_CLIP_HELPER names its part. Its output lines start with "CLIPHELPER ".
func TestWineClipboardHelper(t *testing.T) {
	mode := os.Getenv("GLUTTER_CLIP_HELPER")
	if mode == "" {
		t.Skip("only the helper process of the other tests runs this")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	say := func(format string, v ...any) {
		out, _ := json.Marshal(append([]any{format}, v...))
		os.Stdout.WriteString("CLIPHELPER " + string(out) + "\n") //nolint:errcheck // the parent reads the pipe
	}
	owner := messageWindow(t)
	core := &clipboardCore{api: &winClipboardAPI{}, sleep: time.Sleep}
	stdin := bufio.NewReader(os.Stdin)
	switch mode {
	case "write":
		var in helperContent
		if err := json.Unmarshal([]byte(os.Getenv("GLUTTER_CLIP_PAYLOAD")), &in); err != nil {
			t.Fatal(err)
		}
		if err := core.write(owner, in.content()); err != nil {
			t.Fatal(err)
		}
		say("ready")
		_, _ = io.Copy(io.Discard, stdin) // until the parent closes the pipe
	case "read":
		say("ready")
		if _, err := stdin.ReadString('\n'); err != nil {
			t.Fatal(err)
		}
		var out helperContent
		for _, f := range []string{ui.ClipboardFormatText, ui.ClipboardFormatURIList, ui.ClipboardFormatPNG} {
			got, err := core.read(f)
			if err != nil {
				t.Fatal(err)
			}
			if got.HasText {
				out.Text, out.HasText = got.Text, true
			}
			if len(got.PNG) > 0 {
				out.PNG = base64.StdEncoding.EncodeToString(got.PNG)
			}
			if len(got.URIs) > 0 {
				out.URIs = got.URIs
			}
		}
		b, _ := json.Marshal(out)
		say("result", string(b))
	case "render-on-demand":
		// The data is rendered when somebody asks for it; this program never answers, as a hung
		// application.
		if err := win32.OpenClipboard(owner); err != nil {
			t.Fatal(err)
		}
		if err := win32.EmptyClipboard(); err != nil {
			t.Fatal(err)
		}
		if err := win32.SetClipboardData(win32.CFUnicodeText, 0); err != nil {
			t.Fatal(err)
		}
		win32.CloseClipboard()
		say("ready")
		_, _ = io.Copy(io.Discard, stdin) // blocked without a message loop
	default:
		t.Fatalf("unknown part %q", mode)
	}
}

// helperProcess is the test binary started for one part.
type helperProcess struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
	lines *bufio.Reader
	errs  *lockedBuffer
}

// lockedBuffer collects the output of the helper that is not a message.
type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

func startHelper(t *testing.T, mode string, payload helperContent) *helperProcess {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(payload)
	cmd := exec.Command(exe, "-test.run=^TestWineClipboardHelper$", "-test.v=false")
	cmd.Env = append(os.Environ(), "GLUTTER_CLIP_HELPER="+mode, "GLUTTER_CLIP_PAYLOAD="+string(data))
	h := &helperProcess{cmd: cmd, errs: &lockedBuffer{}}
	cmd.Stderr = h.errs
	if h.stdin, err = cmd.StdinPipe(); err != nil {
		t.Fatal(err)
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	h.lines = bufio.NewReader(out)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	var once sync.Once
	t.Cleanup(func() { once.Do(func() { h.stop() }) })
	return h
}

// helperResult is a message of the helper, or why there is none.
type helperResult struct {
	msg []any
	err error
}

// next starts reading the output of the helper for a message; the result comes on the channel. It
// is safe to call from the test goroutine only, one at a time.
func (h *helperProcess) next(t *testing.T, limit time.Duration) <-chan helperResult {
	t.Helper()
	out := make(chan helperResult, 1)
	go func() {
		got := make(chan helperResult, 1)
		go func() {
			for {
				line, err := h.lines.ReadString('\n')
				if rest, ok := strings.CutPrefix(line, "CLIPHELPER "); ok {
					var msg []any
					jerr := json.Unmarshal([]byte(rest), &msg)
					got <- helperResult{msg, jerr}
					return
				}
				if line != "" && !strings.HasPrefix(line, "CLIPHELPER ") {
					_, _ = h.errs.Write([]byte(line))
				}
				if err != nil {
					got <- helperResult{nil, err}
					return
				}
			}
		}()
		select {
		case r := <-got:
			out <- r
		case <-time.After(limit):
			out <- helperResult{nil, context.DeadlineExceeded}
		}
	}()
	return out
}

// wait reads lines until one that is a helper message, with the limit as the time allowed.
func (h *helperProcess) wait(t *testing.T, limit time.Duration) []any {
	t.Helper()
	r := <-h.next(t, limit)
	if r.err != nil {
		t.Fatalf("the helper process: %v\n%s", r.err, h.errs.String())
	}
	return r.msg
}

func (h *helperProcess) say(line string) { _, _ = io.WriteString(h.stdin, line+"\n") }

// stop lets the helper end and waits for it, killing it when it does not.
func (h *helperProcess) stop() {
	_ = h.stdin.Close()
	done := make(chan struct{})
	go func() { _ = h.cmd.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		_ = h.cmd.Process.Kill()
		<-done
	}
}

func TestWineClipboardAnotherProcessWrites(t *testing.T) {
	png := wineImage(t)
	want := ui.ClipboardContent{Text: "from another process\r\nwith CRLF is read as LF", HasText: true, URIs: []string{"file:///C:/proc/a%20b.txt", "file://server/share/c.txt"}, PNG: png}
	h := startHelper(t, "write", helperOf(want))
	if msg := h.wait(t, 40*time.Second); msg[0] != "ready" {
		t.Fatalf("helper said %v", msg)
	}
	clipRun(t, 60*time.Second, func(p *Platform, c *clipboard, finish func()) {
		var got ui.ClipboardContent
		read := func(format string, next func()) {
			c.GetContent(context.Background(), ui.ClipboardKindSystem, format, func(g ui.ClipboardContent, err error) {
				if err != nil {
					t.Errorf("GetContent(%s): %v", format, err)
				}
				if g.HasText {
					got.Text, got.HasText = g.Text, true
				}
				if len(g.PNG) > 0 {
					got.PNG = g.PNG
				}
				if len(g.URIs) > 0 {
					got.URIs = g.URIs
				}
				next()
			})
		}
		seq(
			func(next func()) { read(ui.ClipboardFormatText, next) },
			func(next func()) { read(ui.ClipboardFormatURIList, next) },
			func(next func()) { read(ui.ClipboardFormatPNG, next) },
			func(next func()) {
				if got.Text != "from another process\nwith CRLF is read as LF" || !got.HasText {
					t.Errorf("text = %q", got.Text)
				}
				if !reflect.DeepEqual(got.URIs, want.URIs) {
					t.Errorf("URIs = %q, want %q", got.URIs, want.URIs)
				}
				if !reflect.DeepEqual(pixelsOf(t, got.PNG), pixelsOf(t, png)) {
					t.Error("the image of the other process differs")
				}
				finish()
			},
		)
	})
}

func TestWineClipboardAnotherProcessReads(t *testing.T) {
	png := wineImage(t)
	h := startHelper(t, "read", helperContent{})
	if msg := h.wait(t, 40*time.Second); msg[0] != "ready" {
		t.Fatalf("helper said %v", msg)
	}
	want := ui.ClipboardContent{Text: "to another\nprocess", HasText: true, URIs: []string{"file:///C:/to/x.txt"}, PNG: png}
	clipRun(t, 90*time.Second, func(p *Platform, c *clipboard, finish func()) {
		c.SetContent(context.Background(), ui.ClipboardKindSystem, want, func(err error) {
			if err != nil {
				t.Error(err)
				finish()
				return
			}
			// The loop goes on while the other program reads: the owner of the data answers it.
			result := h.next(t, 60*time.Second)
			h.say("go")
			go func() {
				r := <-result
				p.Post(func() {
					defer finish()
					if r.err != nil || len(r.msg) < 2 || r.msg[0] != "result" {
						t.Errorf("the other process: %v %v", r.msg, r.err)
						return
					}
					var got helperContent
					if err := json.Unmarshal([]byte(r.msg[1].(string)), &got); err != nil {
						t.Error(err)
						return
					}
					if got.Text != want.Text || !got.HasText || !reflect.DeepEqual(got.URIs, want.URIs) || !reflect.DeepEqual(pixelsOf(t, got.content().PNG), pixelsOf(t, png)) {
						t.Errorf("the other process read %+v", got)
					}
				})
			}()
		})
	})
}

// holder keeps the clipboard open on a thread of its own until it is released.
type holder struct {
	release chan struct{}
	done    chan struct{}
}

func holdClipboard(t *testing.T) *holder {
	t.Helper()
	h := &holder{release: make(chan struct{}), done: make(chan struct{})}
	ready := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer close(h.done)
		instance, _ := win32.GetModuleHandle("")
		hwnd, err := win32.CreateWindowEx(0, "STATIC", "", 0, 0, 0, 0, 0, 0, 0, instance, 0)
		if err != nil {
			ready <- err
			return
		}
		defer func() { _ = win32.DestroyWindow(hwnd) }()
		if err := win32.OpenClipboard(hwnd); err != nil {
			ready <- err
			return
		}
		ready <- nil
		<-h.release
		win32.CloseClipboard()
	}()
	if err := <-ready; err != nil {
		t.Fatal(err)
	}
	var once sync.Once
	t.Cleanup(func() { once.Do(h.let) })
	return h
}

// let releases the clipboard and waits until it is closed.
func (h *holder) let() {
	select {
	case <-h.release:
	default:
		close(h.release)
	}
	<-h.done
}

func TestWineClipboardBusyIsTriedAgain(t *testing.T) {
	clipRun(t, 30*time.Second, func(p *Platform, c *clipboard, finish func()) {
		h := holdClipboard(t)
		sleeps := 0
		c.core.sleep = func(d time.Duration) {
			sleeps++
			if sleeps == 3 { // the other program lets go during the third wait
				h.let()
			}
			time.Sleep(d)
		}
		c.SetText(context.Background(), "after the wait", func(err error) {
			defer finish()
			if err != nil {
				t.Fatalf("SetText: %v", err)
			}
			if sleeps != 3 {
				t.Errorf("slept %d times, want 3", sleeps)
			}
			got, present, err := rawFormat(c.core.api, win32.CFUnicodeText)
			if err != nil || !present || textFromClipboard(got) != "after the wait" {
				t.Errorf("the clipboard holds %q, %v, %v", got, present, err)
			}
		})
	})
}

func TestWineClipboardStaysBusy(t *testing.T) {
	clipRun(t, 30*time.Second, func(p *Platform, c *clipboard, finish func()) {
		h := holdClipboard(t)
		sleeps := 0
		c.core.sleep = func(d time.Duration) { sleeps++; time.Sleep(d) }
		c.SetText(context.Background(), "never", func(err error) {
			assertTagRoute(t, err, "ClipboardBusy", "clipboard.SetContent->clipboardCore.write->clipboardCore.open.ClipboardBusy")
			if sleeps != 9 {
				t.Errorf("slept %d times, want 9 (10 tries)", sleeps)
			}
			// A read of a format that is there is busy too (the other format-less read does not open it).
			h.let()
			c.SetText(context.Background(), "now", func(err error) {
				if err != nil {
					t.Errorf("after the release: %v", err)
				}
				h2 := holdClipboard(t)
				c.GetText(context.Background(), func(text string, ok bool, err error) {
					assertTagRoute(t, err, "ClipboardBusy", "clipboardCore.read->clipboardCore.open.ClipboardBusy")
					h2.let()
					finish()
				})
			})
		})
	})
}

func TestWineClipboardReadEndedByItsContextIsReadTimeout(t *testing.T) {
	h := startHelper(t, "render-on-demand", helperContent{})
	if msg := h.wait(t, 40*time.Second); msg[0] != "ready" {
		t.Fatalf("helper said %v", msg)
	}
	clipRun(t, 60*time.Second, func(p *Platform, c *clipboard, finish func()) {
		ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
		calls := 0
		start := time.Now()
		c.GetText(ctx, func(text string, ok bool, err error) {
			calls++
			cancel()
			assertTagRoute(t, err, "ClipboardReadTimeout", "ClipboardReadTimeout")
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Errorf("the error does not keep its cause: %v", err)
			}
			if d := time.Since(start); d > 10*time.Second {
				t.Errorf("the read answered after %v: the loop waited for the thread", d)
			}
			// The other program answers late: the thread ends and its result is dropped.
			h.stop()
			p.timers.add(time.Now().Add(500*time.Millisecond), func() {
				if calls != 1 {
					t.Errorf("done was called %d times, want once", calls)
				}
				finish()
			})
		})
	})
}

func TestWineClipboardErrorsKeepTheirRoutes(t *testing.T) {
	clipRun(t, 30*time.Second, func(p *Platform, c *clipboard, finish func()) {
		ctx := context.Background()
		seq(
			func(next func()) {
				c.SetContent(ctx, ui.ClipboardKindPrimary, ui.ClipboardContent{Text: "x", HasText: true}, func(err error) {
					assertTagRoute(t, err, "ClipboardPrimaryUnavailable", "clipboard.SetContent.ClipboardPrimaryUnavailable")
					next()
				})
			},
			func(next func()) {
				c.GetContent(ctx, ui.ClipboardKindPrimary, ui.ClipboardFormatText, func(_ ui.ClipboardContent, err error) {
					assertTagRoute(t, err, "ClipboardPrimaryUnavailable", "clipboard.GetContent.ClipboardPrimaryUnavailable")
					next()
				})
			},
			func(next func()) {
				c.SetContent(ctx, ui.ClipboardKindSystem, ui.ClipboardContent{URIs: []string{"https://example.com/a"}}, func(err error) {
					assertTagRoute(t, err, "WindowsClipboardFormatUnsupported", "clipboard.SetContent->clipboardCore.write.WindowsClipboardFormatUnsupported")
					next()
				})
			},
			func(next func()) {
				c.SetContent(ctx, ui.ClipboardKindSystem, ui.ClipboardContent{PNG: []byte("not a png")}, func(err error) {
					assertTagRoute(t, err, "ClipboardWriteFailed", "clipboard.SetContent->clipboardCore.write->dibV5FromPNG.ClipboardWriteFailed")
					next()
				})
			},
			func(next func()) {
				ended, cancel := context.WithCancel(ctx)
				cancel()
				c.GetText(ended, func(_ string, _ bool, err error) {
					assertTagRoute(t, err, "ClipboardReadTimeout", "clipboard.GetText->clipboard.GetContent->asyncRead[...].ClipboardReadTimeout")
					next()
				})
			},
			func(next func()) {
				// A nil done is allowed for all of them.
				c.SetText(ctx, "x", nil)
				c.GetText(ctx, nil)
				c.GetContent(ctx, ui.ClipboardKindSystem, "", nil)
				finish()
			},
		)
	})
}
