package windows

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

func assertTagRoute(t *testing.T, err error, tag, route string) {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if appErr.Tag != tag || !appErr.HasRoute(route) {
		t.Errorf("got %s via %s, want %s via %s", appErr.Tag, appErr.Route(), tag, route)
	}
}

func TestTextToClipboard(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "abc", "abc\x00"},
		{"empty", "", "\x00"},
		{"a line feed is a CRLF", "a\nb", "a\r\nb\x00"},
		{"a CRLF stays one", "a\r\nb", "a\r\nb\x00"},
		{"a lone CR is a line break", "a\rb", "a\r\nb\x00"},
		{"CR CR LF is two breaks", "a\r\r\nb", "a\r\n\r\nb\x00"},
		{"two line feeds", "\n\n", "\r\n\r\n\x00"},
		{"the text ends at its NUL", "ab\x00cd", "ab\x00"},
		{"a surrogate pair", "a𠮷b", "a\U00020BB7b\x00"},
		{"invalid UTF-8 is U+FFFD", "a\xffb", "a\uFFFDb\x00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := textToClipboard(tt.in)
			want := utf16Of(tt.want)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("textToClipboard(%q) = %x, want %x", tt.in, got, want)
			}
		})
	}
}

// utf16Of is the UTF-16 of s, with the NUL it holds kept.
func utf16Of(s string) []uint16 {
	var out []uint16
	for _, r := range s {
		if r >= 0x10000 {
			r -= 0x10000
			out = append(out, uint16(0xD800+(r>>10)), uint16(0xDC00+(r&0x3FF)))
			continue
		}
		out = append(out, uint16(r))
	}
	return out
}

func TestTextFromClipboard(t *testing.T) {
	le := func(units ...uint16) []byte { return textBytes(units) }
	tests := []struct {
		name string
		in   []byte
		want string
	}{
		{"plain", le('a', 'b', 0), "ab"},
		{"no terminating NUL", le('a', 'b'), "ab"},
		{"CRLF is a line feed", le('a', '\r', '\n', 'b', 0), "a\nb"},
		{"a lone CR is a line feed", le('a', '\r', 'b', 0), "a\nb"},
		{"a lone LF stays", le('a', '\n', 'b', 0), "a\nb"},
		{"CR CR LF is two", le('a', '\r', '\r', '\n', 'b', 0), "a\n\nb"},
		{"the text ends at its NUL and the memory may hold more", le('a', 0, 'z', 'z'), "a"},
		{"a surrogate pair", le(0xD842, 0xDFB7, 0), "𠮷"},
		{"a lone high surrogate is U+FFFD", le('a', 0xD842, 'b', 0), "a\uFFFDb"},
		{"a lone low surrogate is U+FFFD", le(0xDFB7, 'b', 0), "\uFFFDb"},
		{"an odd byte at the end is dropped", append(le('a', 'b'), 0x41), "ab"},
		{"empty", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := textFromClipboard(tt.in); got != tt.want {
				t.Errorf("textFromClipboard = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFileURIsAndWindowsPaths(t *testing.T) {
	tests := []struct {
		uri, path string
		// back is the URI that the path becomes; empty when it is the URI itself.
		back string
	}{
		{"file:///C:/Users/me/a.txt", `C:\Users\me\a.txt`, ""},
		{"file:///C:/Users/me/a%20b.txt", `C:\Users\me\a b.txt`, ""},
		{"file:///c:/x/%C3%A9.txt", `c:\x\é.txt`, "file:///c:/x/%C3%A9.txt"},
		{"file://server/share/dir/a.txt", `\\server\share\dir\a.txt`, ""},
		{"file://localhost/C:/a.txt", `C:\a.txt`, "file:///C:/a.txt"},
		{"file:///home/me/a.txt", `\home\me\a.txt`, "file:///home/me/a.txt"},
		{"file:///C:/100%25/a%23b.txt", `C:\100%\a#b.txt`, ""},
		{"FILE:///C:/a.txt", `C:\a.txt`, "file:///C:/a.txt"},
	}
	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			got, ok := pathFromURI(tt.uri)
			if !ok || got != tt.path {
				t.Errorf("pathFromURI(%q) = %q, %v; want %q", tt.uri, got, ok, tt.path)
			}
			want := tt.back
			if want == "" {
				want = tt.uri
			}
			if back := uriFromPath(tt.path); back != want {
				t.Errorf("uriFromPath(%q) = %q, want %q", tt.path, back, want)
			}
		})
	}
	for _, uri := range []string{"http://example.com/a.txt", "mailto:a@b.c", "file:", "file://", "%zz", ""} {
		if p, ok := pathFromURI(uri); ok {
			t.Errorf("pathFromURI(%q) = %q, want a failure", uri, p)
		}
	}
	long := map[string]string{
		`\\?\C:\Users\me\a.txt`:        "file:///C:/Users/me/a.txt",
		`\\?\UNC\server\share\a.txt`:   "file://server/share/a.txt",
		`C:\Documents and Settings\x`:  "file:///C:/Documents%20and%20Settings/x",
		`\\server\share\with space.md`: "file://server/share/with%20space.md",
	}
	for path, want := range long {
		if got := uriFromPath(path); got != want {
			t.Errorf("uriFromPath(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestDropFilesRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		paths []string
	}{
		{"one path", []string{`C:\a.txt`}},
		{"several paths", []string{`C:\a.txt`, `D:\Documents\b c.md`, `C:\é\日本語.txt`}},
		{"a network path", []string{`\\server\share\a.txt`, `C:\b.txt`}},
		{"a path outside the basic plane", []string{`C:\𠮷.txt`}},
		{"no path", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := dropFilesFromPaths(tt.paths)
			if got := binary.LittleEndian.Uint32(data); got != dropFilesHeaderSize {
				t.Errorf("pFiles = %d, want %d", got, dropFilesHeaderSize)
			}
			if wide := binary.LittleEndian.Uint32(data[16:]); wide != 1 {
				t.Errorf("fWide = %d, want 1", wide)
			}
			if end := data[len(data)-4:]; !bytes.Equal(end, []byte{0, 0, 0, 0}) && len(tt.paths) > 0 {
				t.Errorf("the list does not end with two NULs: % x", end)
			}
			got, err := pathsFromDropFiles(data)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.paths) {
				t.Errorf("paths = %q, want %q", got, tt.paths)
			}
		})
	}
}

func TestPathsFromDropFilesTakesWhatOtherProgramsMake(t *testing.T) {
	head := func(pFiles uint32, wide uint32) []byte {
		b := make([]byte, 20)
		binary.LittleEndian.PutUint32(b, pFiles)
		binary.LittleEndian.PutUint32(b[16:], wide)
		return b
	}
	tests := []struct {
		name string
		data []byte
		want []string
	}{
		{"ANSI paths", append(head(20, 0), []byte("C:\\a.txt\x00C:\\caf\xe9.txt\x00\x00")...), []string{`C:\a.txt`, "C:\\caf\u00e9.txt"}},
		{"a list that lacks its last NUL", append(head(20, 1), textBytes(utf16Of(`C:\a.txt`))...), []string{`C:\a.txt`}},
		{"the paths start later than the header", append(append(head(24, 1), 0, 0, 0, 0), textBytes(utf16Of("C:\\a\x00\x00"))...), []string{`C:\a`}},
		{"an empty list", append(head(20, 1), 0, 0), nil},
		{"an odd byte at the end", append(append(head(20, 1), textBytes(utf16Of("C:\\a\x00\x00"))...), 7), []string{`C:\a`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pathsFromDropFiles(tt.data)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("paths = %q, want %q", got, tt.want)
			}
		})
	}
	damaged := []struct {
		name string
		data []byte
	}{
		{"shorter than the header", make([]byte, 10)},
		{"pFiles before the end of the header", head(4, 1)},
		{"pFiles past the end of the data", head(99, 1)},
	}
	for _, tt := range damaged {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pathsFromDropFiles(tt.data)
			assertTagRoute(t, err, "ClipboardReadFailed", "pathsFromDropFiles.ClipboardReadFailed")
		})
	}
}

// testImage is an image with every pixel different and some of them see-through.
func testImage(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(10 + x*7), G: uint8(20 + y*11), B: uint8(30 + x*y), A: uint8(255 - (x+y)*9%200)})
		}
	}
	return img
}

func encodePNG(t testing.TB, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestImageSurvivesTheDIBRoundTrip(t *testing.T) {
	for _, size := range []image.Point{{1, 1}, {3, 2}, {7, 5}, {16, 16}} {
		img := testImage(size.X, size.Y)
		dib, err := dibV5FromPNG(encodePNG(t, img))
		if err != nil {
			t.Fatal(err)
		}
		if got := binary.LittleEndian.Uint32(dib); got != 124 {
			t.Errorf("header size = %d, want 124", got)
		}
		if got := len(dib); got != 124+size.X*size.Y*4 {
			t.Errorf("%v: DIB is %d bytes, want %d", size, got, 124+size.X*size.Y*4)
		}
		// The bottom row comes first, as BGRA.
		first := dib[124 : 124+4]
		c := img.NRGBAAt(0, size.Y-1)
		if first[0] != c.B || first[1] != c.G || first[2] != c.R || first[3] != c.A {
			t.Errorf("%v: first pixel of the DIB = %v, want BGRA of %v", size, first, c)
		}
		back, err := pngFromDIB(dib)
		if err != nil {
			t.Fatal(err)
		}
		got, err := png.Decode(bytes.NewReader(back))
		if err != nil {
			t.Fatal(err)
		}
		for y := range size.Y {
			for x := range size.X {
				want := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
				have := color.NRGBAModel.Convert(got.At(x, y)).(color.NRGBA)
				if want != have {
					t.Fatalf("%v: pixel (%d,%d) = %v, want %v", size, x, y, have, want)
				}
			}
		}
	}
}

// dib24 is a 24-bit CF_DIB (a BITMAPINFOHEADER and rows padded to four bytes, bottom row first) of
// the two rows of colors.
func dib24(top, bottom []color.RGBA) []byte {
	w := len(top)
	stride := (w*3 + 3) &^ 3
	b := make([]byte, 40, 40+2*stride)
	binary.LittleEndian.PutUint32(b[0:], 40)
	binary.LittleEndian.PutUint32(b[4:], uint32(w))
	binary.LittleEndian.PutUint32(b[8:], 2)
	binary.LittleEndian.PutUint16(b[12:], 1)
	binary.LittleEndian.PutUint16(b[14:], 24)
	for _, row := range [][]color.RGBA{bottom, top} {
		line := make([]byte, stride)
		for x, c := range row {
			line[3*x], line[3*x+1], line[3*x+2] = c.B, c.G, c.R
		}
		b = append(b, line...)
	}
	return b
}

func TestPNGFromOtherDIBs(t *testing.T) {
	red, green, blue, white := color.RGBA{255, 0, 0, 255}, color.RGBA{0, 255, 0, 255}, color.RGBA{0, 0, 255, 255}, color.RGBA{255, 255, 255, 255}
	pixels := func(data []byte) [][]color.RGBA {
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		var rows [][]color.RGBA
		for y := range img.Bounds().Dy() {
			var row []color.RGBA
			for x := range img.Bounds().Dx() {
				row = append(row, color.RGBAModel.Convert(img.At(x, y)).(color.RGBA))
			}
			rows = append(rows, row)
		}
		return rows
	}
	t.Run("24 bits per pixel, a width that needs padding", func(t *testing.T) {
		out, err := pngFromDIB(dib24([]color.RGBA{red, green, blue}, []color.RGBA{white, blue, red}))
		if err != nil {
			t.Fatal(err)
		}
		want := [][]color.RGBA{{red, green, blue}, {white, blue, red}}
		if got := pixels(out); !reflect.DeepEqual(got, want) {
			t.Errorf("pixels = %v, want %v", got, want)
		}
	})
	t.Run("8 bits per pixel with a palette", func(t *testing.T) {
		b := make([]byte, 40, 40+8+8)
		binary.LittleEndian.PutUint32(b[0:], 40)
		binary.LittleEndian.PutUint32(b[4:], 2)
		binary.LittleEndian.PutUint32(b[8:], 2)
		binary.LittleEndian.PutUint16(b[12:], 1)
		binary.LittleEndian.PutUint16(b[14:], 8)
		binary.LittleEndian.PutUint32(b[32:], 2) // two colors
		b = append(b, 0, 0, 255, 0 /* red, as BGR + pad */, 255, 0, 0, 0 /* blue */)
		b = append(b, 1, 0, 0, 0 /* bottom row: blue, red */, 0, 1, 0, 0 /* top row: red, blue */)
		out, err := pngFromDIB(b)
		if err != nil {
			t.Fatal(err)
		}
		want := [][]color.RGBA{{red, blue}, {blue, red}}
		if got := pixels(out); !reflect.DeepEqual(got, want) {
			t.Errorf("pixels = %v, want %v", got, want)
		}
	})
	t.Run("32 bits per pixel with the header of 40 bytes has no alpha", func(t *testing.T) {
		b := make([]byte, 40, 44)
		binary.LittleEndian.PutUint32(b[0:], 40)
		binary.LittleEndian.PutUint32(b[4:], 1)
		binary.LittleEndian.PutUint32(b[8:], 1)
		binary.LittleEndian.PutUint16(b[12:], 1)
		binary.LittleEndian.PutUint16(b[14:], 32)
		b = append(b, 0, 0, 255, 0) // red with the unused byte zero
		out, err := pngFromDIB(b)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := pixels(out), [][]color.RGBA{{red}}; !reflect.DeepEqual(got, want) {
			t.Errorf("pixels = %v, want %v", got, want)
		}
	})
}

func TestImageFailuresHaveTheirRoutes(t *testing.T) {
	t.Run("data that is not a PNG cannot be written", func(t *testing.T) {
		_, err := dibV5FromPNG([]byte("not a png"))
		assertTagRoute(t, err, "ClipboardWriteFailed", "dibV5FromPNG.ClipboardWriteFailed")
	})
	t.Run("a PNG that is cut cannot be written", func(t *testing.T) {
		data := encodePNG(t, testImage(8, 8))
		_, err := dibV5FromPNG(data[:len(data)-20])
		assertTagRoute(t, err, "ClipboardWriteFailed", "dibV5FromPNG.ClipboardWriteFailed")
	})
	damaged := map[string][]byte{
		"no header":          make([]byte, 10),
		"a header too large": append(binary.LittleEndian.AppendUint32(nil, 5000), make([]byte, 60)...),
	}
	for name, dib := range damaged {
		t.Run(name, func(t *testing.T) {
			_, err := pngFromDIB(dib)
			assertTagRoute(t, err, "ClipboardReadFailed", "pngFromDIB.ClipboardReadFailed")
		})
	}
	t.Run("an empty bitmap", func(t *testing.T) {
		b := make([]byte, 40)
		binary.LittleEndian.PutUint32(b, 40)
		_, err := pngFromDIB(b)
		assertTagRoute(t, err, "ClipboardReadFailed", "pngFromDIB.ClipboardReadFailed")
	})
	t.Run("a compressed bitmap", func(t *testing.T) {
		b := make([]byte, 60)
		binary.LittleEndian.PutUint32(b[0:], 40)
		binary.LittleEndian.PutUint32(b[4:], 2)
		binary.LittleEndian.PutUint32(b[8:], 2)
		binary.LittleEndian.PutUint16(b[12:], 1)
		binary.LittleEndian.PutUint16(b[14:], 8)
		binary.LittleEndian.PutUint32(b[16:], 1) // BI_RLE8
		_, err := pngFromDIB(b)
		assertTagRoute(t, err, "ClipboardReadFailed", "pngFromDIB.ClipboardReadFailed")
	})
}

// fakeClipboard is the clipboard of Windows as a map: it counts the opens and fails the first
// busyOpens of them as a busy clipboard.
type fakeClipboard struct {
	mu        sync.Mutex
	data      map[uint32][]byte
	order     []uint32
	opens     int
	busyOpens int
	openErr   error
	closes    int
	empties   int
	owner     uintptr
	open      bool
	setErr    error
	getErr    error
}

const (
	fakePNGFormat  = 0xC000
	fakeHTMLFormat = 0xC001
)

func newFakeClipboard() *fakeClipboard { return &fakeClipboard{data: map[uint32][]byte{}} }

func (f *fakeClipboard) Open(owner uintptr) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.opens++
	if f.openErr != nil {
		return false, f.openErr
	}
	if f.opens <= f.busyOpens {
		return true, errors.New(win32.TagWin32CallFailed, "a Win32 function failed", "function", "OpenClipboard", "code", 5)
	}
	f.owner, f.open = owner, true
	return false, nil
}

func (f *fakeClipboard) Close() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closes++
	f.open = false
}

func (f *fakeClipboard) Empty() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.empties++
	f.data, f.order = map[uint32][]byte{}, nil
	return nil
}

func (f *fakeClipboard) Set(format uint32, data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.setErr != nil {
		return f.setErr
	}
	f.data[format] = bytes.Clone(data)
	f.order = append(f.order, format)
	return nil
}

func (f *fakeClipboard) Get(format uint32) ([]byte, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, false, f.getErr
	}
	d, ok := f.data[format]
	return bytes.Clone(d), ok, nil
}

func (f *fakeClipboard) Has(format uint32) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.data[format]
	return ok
}

func (f *fakeClipboard) PNGFormat() (uint32, error) { return fakePNGFormat, nil }

func (f *fakeClipboard) HTMLFormat() (uint32, error) { return fakeHTMLFormat, nil }

func newCore(api clipboardAPI) (*clipboardCore, *[]time.Duration) {
	var sleeps []time.Duration
	return &clipboardCore{api: api, sleep: func(d time.Duration) { sleeps = append(sleeps, d) }}, &sleeps
}

func TestWriteAndReadAContentWithEveryFormat(t *testing.T) {
	fake := newFakeClipboard()
	core, _ := newCore(fake)
	img := encodePNG(t, testImage(4, 3))
	content := ui.ClipboardContent{
		Text: "a\nb", HasText: true, PNG: img, HTML: "<b>a</b><br>b \u00e9\u4e16",
		URIs: []string{"file:///C:/Users/me/a%20b.txt", "file://server/share/c.txt"},
	}
	if err := core.write(0x1234, content); err != nil {
		t.Fatal(err)
	}
	if fake.owner != 0x1234 || fake.empties != 1 || fake.closes != 1 || fake.open {
		t.Errorf("owner %#x, %d empties, %d closes, open %v", fake.owner, fake.empties, fake.closes, fake.open)
	}
	wantOrder := []uint32{win32.CFUnicodeText, fakeHTMLFormat, win32.CFHDrop, fakePNGFormat, win32.CFDIBV5}
	if !reflect.DeepEqual(fake.order, wantOrder) {
		t.Errorf("formats set in the order %v, want %v", fake.order, wantOrder)
	}
	if got := textFromClipboard(fake.data[win32.CFUnicodeText]); got != "a\nb" {
		t.Errorf("text = %q", got)
	}
	if !bytes.Equal(fake.data[fakePNGFormat], img) {
		t.Error("the PNG format does not hold the PNG that was given")
	}
	for format, want := range map[string]ui.ClipboardContent{
		ui.ClipboardFormatText:    {Text: "a\nb", HasText: true},
		ui.ClipboardFormatPNG:     {PNG: img},
		ui.ClipboardFormatHTML:    {HTML: "<b>a</b><br>b \u00e9\u4e16"},
		ui.ClipboardFormatURIList: {URIs: []string{"file:///C:/Users/me/a%20b.txt", "file://server/share/c.txt"}},
		"":                        {Text: "a\nb", HasText: true},
		"application/x-other":     {},
	} {
		t.Run("read "+format, func(t *testing.T) {
			before := fake.opens
			got, err := core.read(format)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("content = %+v, want %+v", got, want)
			}
			if format == "application/x-other" && fake.opens != before {
				t.Error("a format nobody knows opened the clipboard")
			}
			if fake.owner != 0 && format != "application/x-other" && fake.open {
				t.Error("the clipboard was left open")
			}
		})
	}
}

func TestReadsTheImageWhenThereIsNoPNG(t *testing.T) {
	fake := newFakeClipboard()
	dib, err := dibV5FromPNG(encodePNG(t, testImage(5, 4)))
	if err != nil {
		t.Fatal(err)
	}
	fake.data[win32.CFDIBV5] = dib
	core, _ := newCore(fake)
	got, err := core.read(ui.ClipboardFormatPNG)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(got.PNG))
	if err != nil || cfg.Width != 5 || cfg.Height != 4 {
		t.Errorf("PNG = %+v, %v; want 5x4", cfg, err)
	}
	// With neither the PNG nor DIBV5, a plain CF_DIB (what Paint of old puts) is read.
	fake.data = map[uint32][]byte{win32.CFDIB: dib24([]color.RGBA{{1, 2, 3, 255}}, []color.RGBA{{4, 5, 6, 255}})}
	got, err = core.read(ui.ClipboardFormatPNG)
	if err != nil || len(got.PNG) == 0 {
		t.Errorf("CF_DIB: %d bytes, %v", len(got.PNG), err)
	}
}

func TestAnEmptyContentClearsTheClipboardAndNothingElseIsChanged(t *testing.T) {
	fake := newFakeClipboard()
	core, _ := newCore(fake)
	if err := core.write(1, ui.ClipboardContent{Text: "x", HasText: true}); err != nil {
		t.Fatal(err)
	}
	if err := core.write(1, ui.ClipboardContent{}); err != nil {
		t.Fatal(err)
	}
	if len(fake.data) != 0 || fake.empties != 2 {
		t.Errorf("%d formats left after %d empties", len(fake.data), fake.empties)
	}
	// An empty string is a text.
	if err := core.write(1, ui.ClipboardContent{HasText: true}); err != nil {
		t.Fatal(err)
	}
	got, err := core.read(ui.ClipboardFormatText)
	if err != nil || !got.HasText || got.Text != "" {
		t.Errorf("content = %+v, %v; want an empty text", got, err)
	}
	// A content that cannot be written changes nothing.
	err = core.write(1, ui.ClipboardContent{Text: "new", HasText: true, URIs: []string{"http://example.com/a"}})
	assertTagRoute(t, err, "WindowsClipboardFormatUnsupported", "clipboardCore.write.WindowsClipboardFormatUnsupported")
	if got, _ := core.read(ui.ClipboardFormatText); got.Text != "" || !got.HasText {
		t.Errorf("a failed write changed the clipboard: %+v", got)
	}
	err = core.write(1, ui.ClipboardContent{PNG: []byte("nope")})
	assertTagRoute(t, err, "ClipboardWriteFailed", "clipboardCore.write->dibV5FromPNG.ClipboardWriteFailed")
}

func TestABusyClipboardIsTriedAgain(t *testing.T) {
	tests := []struct {
		name       string
		busyOpens  int
		wantOpens  int
		wantSleeps int
		wantBusy   bool
	}{
		{"free at once", 0, 1, 0, false},
		{"busy twice", 2, 3, 2, false},
		{"busy nine times", 9, 10, 9, false},
		{"busy for ever", 1000, 10, 9, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := newFakeClipboard()
			fake.busyOpens = tt.busyOpens
			core, sleeps := newCore(fake)
			err := core.write(1, ui.ClipboardContent{Text: "x", HasText: true})
			if fake.opens != tt.wantOpens || len(*sleeps) != tt.wantSleeps {
				t.Errorf("%d opens and %d sleeps, want %d and %d", fake.opens, len(*sleeps), tt.wantOpens, tt.wantSleeps)
			}
			for _, d := range *sleeps {
				if d != 5*time.Millisecond {
					t.Errorf("slept %v between tries, want 5ms", d)
				}
			}
			if tt.wantBusy {
				assertTagRoute(t, err, "ClipboardBusy", "clipboardCore.write->clipboardCore.open.ClipboardBusy")
				if appErr, _ := errors.FromError(err); appErr.Metadata["tries"] != 10 {
					t.Errorf("metadata = %v", appErr.Metadata)
				}
				if fake.closes != 0 {
					t.Error("a clipboard that was never opened was closed")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if fake.closes != 1 {
				t.Errorf("%d closes, want 1", fake.closes)
			}
		})
	}
	t.Run("a read waits too", func(t *testing.T) {
		fake := newFakeClipboard()
		fake.data[win32.CFUnicodeText] = textBytes(textToClipboard("x"))
		fake.busyOpens = 3
		core, sleeps := newCore(fake)
		got, err := core.read(ui.ClipboardFormatText)
		if err != nil || got.Text != "x" || len(*sleeps) != 3 {
			t.Errorf("content %+v, error %v, %d sleeps", got, err, len(*sleeps))
		}
		fake.opens, fake.busyOpens = 0, 1000
		_, err = core.read(ui.ClipboardFormatText)
		assertTagRoute(t, err, "ClipboardBusy", "clipboardCore.read->clipboardCore.open.ClipboardBusy")
	})
}

func TestAFailureThatIsNotBusyIsReturnedAtOnce(t *testing.T) {
	fake := newFakeClipboard()
	fake.openErr = errors.New(win32.TagWin32CallFailed, "a Win32 function failed", "function", "OpenClipboard", "code", 1400)
	core, sleeps := newCore(fake)
	err := core.write(1, ui.ClipboardContent{Text: "x", HasText: true})
	assertTagRoute(t, err, win32.TagWin32CallFailed, "")
	if fake.opens != 1 || len(*sleeps) != 0 {
		t.Errorf("%d opens and %d sleeps, want 1 and 0", fake.opens, len(*sleeps))
	}
}

func TestAFailureToSetClosesTheClipboard(t *testing.T) {
	fake := newFakeClipboard()
	fake.setErr = errors.New(win32.TagWin32CallFailed, "a Win32 function failed", "function", "SetClipboardData", "code", 8)
	core, _ := newCore(fake)
	err := core.write(1, ui.ClipboardContent{Text: "x", HasText: true})
	assertTagRoute(t, err, win32.TagWin32CallFailed, "")
	if fake.closes != 1 || fake.open {
		t.Errorf("%d closes, open %v", fake.closes, fake.open)
	}
}

func TestADamagedFormatIsReadFailed(t *testing.T) {
	fake := newFakeClipboard()
	core, _ := newCore(fake)
	fake.data[win32.CFHDrop] = []byte{1, 2, 3}
	_, err := core.read(ui.ClipboardFormatURIList)
	assertTagRoute(t, err, "ClipboardReadFailed", "clipboardCore.read->pathsFromDropFiles.ClipboardReadFailed")
	if fake.closes != 1 {
		t.Errorf("%d closes, want 1", fake.closes)
	}
	fake.data = map[uint32][]byte{win32.CFDIBV5: {1, 2, 3}}
	_, err = core.read(ui.ClipboardFormatPNG)
	assertTagRoute(t, err, "ClipboardReadFailed", "clipboardCore.read->pngFromDIB.ClipboardReadFailed")
	fake.data = map[uint32][]byte{fakePNGFormat: []byte("not a png")}
	_, err = core.read(ui.ClipboardFormatPNG)
	assertTagRoute(t, err, "ClipboardReadFailed", "clipboardCore.read.ClipboardReadFailed")
	fake.getErr = errors.New(win32.TagWin32CallFailed, "a Win32 function failed", "function", "GlobalLock", "code", 6)
	fake.data = map[uint32][]byte{win32.CFUnicodeText: nil}
	_, err = core.read(ui.ClipboardFormatText)
	assertTagRoute(t, err, win32.TagWin32CallFailed, "")
}

// poster runs the functions that were posted when run is called, as the UI thread would.
type poster struct {
	fns chan func()
}

func newPoster() *poster { return &poster{fns: make(chan func(), 16)} }

func (p *poster) post(fn func()) { p.fns <- fn }

// next runs one posted function, or fails the test after a second.
func (p *poster) next(t *testing.T) {
	t.Helper()
	select {
	case fn := <-p.fns:
		fn()
	case <-time.After(time.Second):
		t.Fatal("nothing was posted to the UI thread")
	}
}

func TestAsyncReadGivesTheResultOnTheUIThread(t *testing.T) {
	p := newPoster()
	var got []string
	asyncRead(context.Background(), p.post, func() (string, error) { return "text", nil }, func(v string, err error) {
		if err != nil {
			t.Errorf("error %v", err)
		}
		got = append(got, v)
	})
	if len(got) != 0 {
		t.Fatal("done was called before the UI thread ran it")
	}
	p.next(t)
	if !reflect.DeepEqual(got, []string{"text"}) {
		t.Errorf("results = %q", got)
	}
}

func TestAsyncReadWithAnEndedContextIsReadTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p := newPoster()
	called := false
	var gotErr error
	asyncRead(ctx, p.post, func() (string, error) { called = true; return "late", nil }, func(_ string, err error) { gotErr = err })
	if gotErr != nil {
		t.Fatal("done was called before the UI thread ran it")
	}
	p.next(t)
	assertTagRoute(t, gotErr, "ClipboardReadTimeout", "asyncRead[...].ClipboardReadTimeout")
	if called {
		t.Error("a read with an ended context was started")
	}
	if !errors.Is(gotErr, context.Canceled) {
		t.Errorf("the error does not keep the cause: %v", gotErr)
	}
}

func TestAsyncReadThatTheContextEndsAnswersOnce(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	p := newPoster()
	release := make(chan struct{})
	started := make(chan struct{})
	var results []error
	asyncRead(ctx, p.post, func() (string, error) { close(started); <-release; return "late", nil }, func(_ string, err error) { results = append(results, err) })
	<-started
	cancel()
	p.next(t) // the timeout
	close(release)
	p.next(t) // the late result of the thread, which is dropped
	if len(results) != 1 {
		t.Fatalf("done was called %d times, want once", len(results))
	}
	assertTagRoute(t, results[0], "ClipboardReadTimeout", "ClipboardReadTimeout")
}

func TestAsyncReadKeepsTheErrorOfTheRead(t *testing.T) {
	p := newPoster()
	want := errors.New(tagClipboardBusy, "busy")
	var got error
	asyncRead(context.Background(), p.post, func() (int, error) { return 0, want }, func(_ int, err error) { got = err })
	p.next(t)
	if got != error(want) {
		t.Errorf("error = %v, want the error of the read untouched", got)
	}
}

func TestHTMLToClipboardWritesTheDescription(t *testing.T) {
	tests := []struct {
		name     string
		fragment string
		want     string // the fragment that a reader of the description gets
	}{
		{"ASCII", "<b>bold</b>", "<b>bold</b>"},
		{"non-ASCII", "<p>Gr\u00fc\u00dfe \u4e16\u754c \U0001F600</p>", "<p>Gr\u00fc\u00dfe \u4e16\u754c \U0001F600</p>"},
		{"no <html>, no <body>", "just <i>words</i>", "just <i>words</i>"},
		{"empty", "", ""},
		{"cut at a NUL", "a<b>b\x00c</b>", "a<b>b"},
		{"invalid UTF-8", "a\xffb", "a\ufffdb"},
		{"CRLF inside", "<pre>a\r\nb</pre>", "<pre>a\r\nb</pre>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := htmlToClipboard(tt.fragment)
			if len(data) == 0 || data[len(data)-1] != 0 {
				t.Fatalf("the data does not end with a NUL: %q", data)
			}
			if !strings.HasPrefix(string(data), "Version:0.9\r\nStartHTML:0000000105\r\nEndHTML:") {
				t.Fatalf("header = %q", data[:min(len(data), 120)])
			}
			if string(data[cfHTMLHeaderSize-2:cfHTMLHeaderSize]) != "\r\n" {
				t.Errorf("the header is not %d bytes long: %q", cfHTMLHeaderSize, data[:cfHTMLHeaderSize+10])
			}
			fields := map[string]int{}
			for _, line := range strings.Split(string(data[:cfHTMLHeaderSize]), "\r\n") {
				if name, value, ok := strings.Cut(line, ":"); ok && name != "Version" {
					n, err := strconv.Atoi(value)
					if err != nil {
						t.Fatalf("%q: %v", line, err)
					}
					fields[name] = n
				}
			}
			body := data[:len(data)-1] // the NUL is counted by no offset
			if fields["StartHTML"] != cfHTMLHeaderSize || fields["EndHTML"] != len(body) {
				t.Errorf("StartHTML %d EndHTML %d, want %d and %d", fields["StartHTML"], fields["EndHTML"], cfHTMLHeaderSize, len(body))
			}
			frag := string(body[fields["StartFragment"]:fields["EndFragment"]])
			if frag != tt.want {
				t.Errorf("the bytes between StartFragment and EndFragment = %q, want %q", frag, tt.want)
			}
			if !strings.HasSuffix(string(body[:fields["StartFragment"]]), "<!--StartFragment-->") ||
				!strings.HasPrefix(string(body[fields["EndFragment"]:]), "<!--EndFragment-->") {
				t.Errorf("the markers are not around the fragment: %q", body[cfHTMLHeaderSize:])
			}
			if got := htmlFromClipboard(data); got != tt.want {
				t.Errorf("the round trip = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHTMLFromClipboardOfOtherPrograms(t *testing.T) {
	// A description as Chrome writes it: offsets, the markers, a NUL at the end.
	chrome := func() []byte {
		doc := "<html>\r\n<body>\r\n<!--StartFragment--><span>caf\u00e9 \u4e16</span><!--EndFragment-->\r\n</body>\r\n</html>"
		head := "Version:1.0\r\nStartHTML:%010d\r\nEndHTML:%010d\r\nStartFragment:%010d\r\nEndFragment:%010d\r\nSourceURL:https://example.com/\r\n"
		hl := len(fmt.Sprintf(head, 0, 0, 0, 0))
		sf := hl + strings.Index(doc, "<span>")
		ef := hl + strings.Index(doc, "<!--EndFragment-->")
		return []byte(fmt.Sprintf(head, hl, hl+len(doc), sf, ef) + doc + "\x00")
	}()
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{"Chrome", chrome, "<span>caf\u00e9 \u4e16</span>"},
		{"offsets of -1 for the document, markers present",
			[]byte("Version:0.9\nStartHTML:-1\nEndHTML:-1\nStartFragment:-1\nEndFragment:-1\n<html><body><!--StartFragment--><u>x</u><!--EndFragment--></body></html>"), "<u>x</u>"},
		{"no fragment offsets, no markers: the inside of <body>",
			[]byte("Version:0.9\r\nStartHTML:0000000000\r\nEndHTML:0000000000\r\n<HTML><BODY bgcolor=red><p>hi</p></BODY></HTML>"), "<p>hi</p>"},
		{"no fragment offsets, no markers, no <body>: the document",
			[]byte("Version:0.9\r\n<p>hi</p>"), "<p>hi</p>"},
		{"no description at all: the data is the markup", []byte("<b>raw</b>\x00"), "<b>raw</b>"},
		{"no description, markers",
			[]byte("<html><!--StartFragment-->in<!--EndFragment--></html>"), "in"},
		{"offsets outside the data: the markers are used",
			[]byte("Version:0.9\r\nStartHTML:0000000105\r\nEndHTML:0000009999\r\nStartFragment:0000000130\r\nEndFragment:0000009998\r\n<!--StartFragment-->ok<!--EndFragment-->"), "ok"},
		{"offsets inside the description are not used",
			[]byte("Version:0.9\r\nStartFragment:0000000001\r\nEndFragment:0000000005\r\n<!--StartFragment-->ok<!--EndFragment-->"), "ok"},
		{"fragment offsets win over markers in the text",
			[]byte("Version:0.9\r\nStartFragment:0000000063\r\nEndFragment:0000000066\r\n<p>abc<!--StartFragment-->no<!--EndFragment--></p>"), "<p>"},
		{"the fragment is cut inside a multibyte character: invalid UTF-8 becomes U+FFFD",
			[]byte("Version:0.9\r\nStartFragment:0000000063\r\nEndFragment:0000000064\r\n\xc3\xa9\xc3\xa9"), "\ufffd"},
		{"empty", nil, ""},
		{"only a NUL", []byte{0}, ""},
		{"a line that is not a field ends the description",
			[]byte("<p>a: b</p>"), "<p>a: b</p>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := htmlFromClipboard(tt.data); got != tt.want {
				t.Errorf("fragment = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReadAndWriteHTMLOnTheClipboard(t *testing.T) {
	const fragment = "<b>na\u00efve</b> \u4e16\u754c"
	tests := []struct {
		name    string
		content ui.ClipboardContent
		format  string
		want    ui.ClipboardContent
		formats []uint32 // what is on the clipboard after the write
	}{
		{"text and HTML", ui.ClipboardContent{Text: "na\u00efve", HasText: true, HTML: fragment}, ui.ClipboardFormatHTML,
			ui.ClipboardContent{HTML: fragment}, []uint32{win32.CFUnicodeText, fakeHTMLFormat}},
		{"text and HTML, the text", ui.ClipboardContent{Text: "na\u00efve", HasText: true, HTML: fragment}, ui.ClipboardFormatText,
			ui.ClipboardContent{Text: "na\u00efve", HasText: true}, []uint32{win32.CFUnicodeText, fakeHTMLFormat}},
		{"HTML only", ui.ClipboardContent{HTML: fragment}, ui.ClipboardFormatHTML,
			ui.ClipboardContent{HTML: fragment}, []uint32{fakeHTMLFormat}},
		{"HTML only has no text", ui.ClipboardContent{HTML: fragment}, ui.ClipboardFormatText,
			ui.ClipboardContent{}, []uint32{fakeHTMLFormat}},
		{"text only has no HTML", ui.ClipboardContent{Text: "t", HasText: true}, ui.ClipboardFormatHTML,
			ui.ClipboardContent{}, []uint32{win32.CFUnicodeText}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := newFakeClipboard()
			core, _ := newCore(fake)
			if err := core.write(1, tt.content); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(fake.order, tt.formats) {
				t.Errorf("formats = %v, want %v", fake.order, tt.formats)
			}
			opens := fake.opens
			got, err := core.read(tt.format)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("read = %+v, want %+v", got, tt.want)
			}
			if tt.want.IsEmpty() && fake.opens != opens {
				t.Error("a format the clipboard lacks opened the clipboard")
			}
		})
	}
}
