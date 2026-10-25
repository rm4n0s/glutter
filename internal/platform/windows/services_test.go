package windows

import (
	"context"
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

func TestLimitText(t *testing.T) {
	long := func(n int) string { return strings.Repeat("a", n) }
	tests := []struct {
		name string
		in   string
		max  int
		want string
	}{
		{"fits", "hello", 63, "hello"},
		{"exactly the limit", long(63), 63, long(63)},
		{"one over is cut with an ellipsis", long(64), 63, long(62) + "…"},
		{"a tip of 127", long(127), 127, long(127)},
		{"a tip of 128", long(128), 127, long(126) + "…"},
		{"a body of 255", long(255), 255, long(255)},
		{"a body of 256", long(256), 255, long(254) + "…"},
		{"empty", "", 63, ""},
		{"a NUL is dropped", "a\x00b", 63, "ab"},
		{"a surrogate pair is never cut in the middle", long(61) + "𠮷𠮷", 63, long(61) + "…"},
		{"a pair that fits stays", long(60) + "𠮷", 63, long(60) + "𠮷"},
		{"line breaks stay", "one\ntwo", 63, "one\ntwo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := limitText(tt.in, tt.max)
			if got != tt.want {
				t.Errorf("limitText(%q, %d) = %q, want %q", tt.in, tt.max, got, tt.want)
			}
			if n := len(utf16.Encode([]rune(got))); n > tt.max {
				t.Errorf("the result has %d units, more than %d", n, tt.max)
			}
		})
	}
}

func TestCheckURI(t *testing.T) {
	tests := []struct {
		name string
		uri  string
		want string // the tag, "" for accepted
	}{
		{"https", "https://example.org/a?b=c", ""},
		{"http with a port", "http://localhost:8080/", ""},
		{"mailto", "mailto:me@example.org?subject=hi", ""},
		{"a custom scheme", "glutter-test://open/this", ""},
		{"tel", "tel:+123", ""},
		{"a file: URI of a text file", "file:///C:/Users/me/notes.txt", ""},
		{"a file: URI of a folder", "file:///C:/Users/me/", ""},
		{"a file: URI of an image", "file:///C:/pics/a.PNG", ""},
		{"a drive path", `C:\Windows\system32\calc.exe`, "OpenURISchemeBlocked"},
		{"a drive path with a slash", "c:/x", "OpenURISchemeBlocked"},
		{"javascript", "javascript:alert(1)", "OpenURISchemeBlocked"},
		{"javascript in capitals", "JavaScript:alert(1)", "OpenURISchemeBlocked"},
		{"javascript with a tab inside the scheme", "java\tscript:alert(1)", "OpenURISchemeBlocked"},
		{"vbscript", "vbscript:msgbox(1)", "OpenURISchemeBlocked"},
		{"a file: URI of an .exe", "file:///C:/tools/setup.exe", "OpenURISchemeBlocked"},
		{"a file: URI of an .EXE in capitals", "file:///C:/tools/SETUP.EXE", "OpenURISchemeBlocked"},
		{"a file: URI of a .bat", "file:///C:/x/run.bat", "OpenURISchemeBlocked"},
		{"a file: URI of a .lnk", "file:///C:/x/app.lnk", "OpenURISchemeBlocked"},
		{"a file: URI of a .ps1", "file:///C:/x/a.ps1", "OpenURISchemeBlocked"},
		{"a file: URI of a .js", "file:///C:/x/a.js", "OpenURISchemeBlocked"},
		{"a file: URI of a .hta", "file:///C:/x/a.hta", "OpenURISchemeBlocked"},
		{"a file: URI with an escaped dot", "file:///C:/x/a%2Eexe", "OpenURISchemeBlocked"},
		{"a file: URI with a trailing dot", "file:///C:/x/a.exe.", "OpenURISchemeBlocked"},
		{"a file: URI with a trailing space", "file:///C:/x/a.exe%20", "OpenURISchemeBlocked"},
		{"a file: URI with a backslash", `file:///C:\x\a.exe`, "OpenURISchemeBlocked"},
		{"a file: URI of an .exe with a query", "file:///C:/x/a.exe?x=1", "OpenURISchemeBlocked"},
		{"a network file: URI of an .exe", "file://server/share/a.exe", "OpenURISchemeBlocked"},
		{"an empty string", "", "OpenURIInvalid"},
		{"a control character", "https://example.org/\x01", "OpenURIInvalid"},
		{"a tab after the scheme", "https://example.org/\tx", "OpenURIInvalid"},
		{"a DEL", "https://example.org/\x7f", "OpenURIInvalid"},
		{"a line feed", "https://example.org/\nx", "OpenURIInvalid"},
		{"a NUL", "https://example.org/\x00", "OpenURIInvalid"},
		{"no scheme", "example.org/page", "OpenURIInvalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkURI(tt.uri)
			if tt.want == "" {
				if err != nil {
					t.Fatalf("checkURI(%q) = %v, want nil", tt.uri, err)
				}
				return
			}
			assertTagRoute(t, err, tt.want, "checkURI."+tt.want)
		})
	}
}

func TestOpenURIOnlyCallsTheShellAfterTheChecks(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		wantTag string
		called  bool
	}{
		{"https is opened", "https://example.org/", "", true},
		{"a drive path is refused first", `C:\x.exe`, "OpenURISchemeBlocked", false},
		{"javascript is refused first", "javascript:alert(1)", "OpenURISchemeBlocked", false},
		{"an .exe is refused first", "file:///C:/a.exe", "OpenURISchemeBlocked", false},
		{"an empty string is refused first", "", "OpenURIInvalid", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, sh := newTestDesktop()
			err := d.OpenURI(context.Background(), tt.uri)
			if tt.wantTag == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else {
				assertTagRoute(t, err, tt.wantTag, "Desktop.OpenURI->checkURI."+tt.wantTag)
			}
			if got := len(sh.executed) == 1; got != tt.called {
				t.Errorf("ShellExecute called = %v (%v), want %v", got, sh.executed, tt.called)
			}
		})
	}
}

func TestOpenURIGivesTheShellsAnswerBack(t *testing.T) {
	d, sh := newTestDesktop()
	sh.executeErr = errors.New(tagOpenURIUnavailable, "no program is associated", "uri", "glutter-nothing://x")
	assertTagRoute(t, d.OpenURI(context.Background(), "glutter-nothing://x"), tagOpenURIUnavailable, "")
}

func TestOpenURIEndsWithItsContext(t *testing.T) {
	d, sh := newTestDesktop()
	block := make(chan struct{})
	defer close(block)
	sh.executeErr = nil
	slow := &blockingShell{fakeShell: sh, block: block}
	d.shell = slow
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	assertTagRoute(t, d.OpenURI(ctx, "https://example.org/"), tagOpenURIUnavailable, "Desktop.OpenURI."+tagOpenURIUnavailable)
}

// blockingShell is a fake shell whose ShellExecute waits.
type blockingShell struct {
	*fakeShell
	block chan struct{}
}

func (b *blockingShell) ShellExecute(string) error {
	<-b.block
	return nil
}

func TestPortalHandle(t *testing.T) {
	tests := []struct {
		hwnd uintptr
		want string
	}{
		{0, "win32:0"},
		{0x1A2B, "win32:1a2b"},
		{0x30064, "win32:30064"},
		{0xFFFFFFFF, "win32:ffffffff"},
	}
	for _, tt := range tests {
		got := portalHandleOf(tt.hwnd)
		if got != tt.want {
			t.Errorf("portalHandleOf(%#x) = %q, want %q", tt.hwnd, got, tt.want)
		}
		back, ok := hwndOfHandle(got)
		if !ok || back != tt.hwnd {
			t.Errorf("hwndOfHandle(%q) = %#x, %v; want %#x", got, back, ok, tt.hwnd)
		}
	}
	for _, bad := range []string{"", "x11:1a", "wayland:abc", "win32:", "win32:xyz", "win32:12345678901234567"} {
		if _, ok := hwndOfHandle(bad); ok {
			t.Errorf("hwndOfHandle(%q) accepted", bad)
		}
	}
	if v, ok := hwndOfHandle("win32:AbC"); !ok || v != 0xabc {
		t.Errorf("capitals: %#x, %v", v, ok)
	}
}

func TestDecodeIconMessage(t *testing.T) {
	// With NOTIFYICON_VERSION_4: the event is the low word of lParam, the icon the high word, and
	// wParam holds the anchor as two signed 16-bit numbers.
	id, ev, at := decodeIconMessage(uintptr(uint32(uint16(0xFFFB))|uint32(uint16(300))<<16), uintptr(win32.NINSelect)|uintptr(trayIconID)<<16)
	if id != trayIconID || ev != win32.NINSelect || at != (point{X: -5, Y: 300}) {
		t.Errorf("version 4: id %d, event %#x, at %v", id, ev, at)
	}
	id, ev, _ = decodeIconMessage(0x10, uintptr(win32.WMContextMenu)|uintptr(balloonIconID)<<16)
	if id != balloonIconID || ev != win32.WMContextMenu {
		t.Errorf("version 4 context menu: id %d, event %#x", id, ev)
	}
	// Without it: wParam is the icon and lParam the event.
	id, ev, at = decodeIconMessage(uintptr(trayIconID), uintptr(win32.WMMButtonUp))
	if id != trayIconID || ev != win32.WMMButtonUp || at != (point{}) {
		t.Errorf("legacy: id %d, event %#x, at %v", id, ev, at)
	}
}

func TestShowNotificationEndsWithItsContext(t *testing.T) {
	d, sh := newTestDesktop()
	sh.deferPost = true // the UI thread does not answer
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := d.ShowNotification(ctx, mobileSpec("hi"))
	assertTagRoute(t, err, tagNotificationsUnavailable, "Desktop.ShowNotification."+tagNotificationsUnavailable)
	err = d.CloseNotification(ctx, 1)
	assertTagRoute(t, err, tagNotificationsUnavailable, "Desktop.CloseNotification."+tagNotificationsUnavailable)
}
