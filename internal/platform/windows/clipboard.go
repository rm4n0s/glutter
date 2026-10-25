package windows

import (
	"bytes"
	"context"
	"encoding/binary"
	"image"
	"image/draw"
	"image/png"
	"net/url"
	"runtime"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
	"golang.org/x/image/bmp"
)

// This file is the plain-Go part of the clipboard of Windows (SPEC §25.7): the conversions between
// the app's content and the clipboard formats (CF_UNICODETEXT, CF_HDROP, the registered format PNG
// and CF_DIBV5), the retry of a busy clipboard, and the read on a thread of its own. The calls to
// Windows are clipboard_windows.go, behind clipboardAPI, so that all of it is tested on any host.

// The retry of a busy clipboard: OpenClipboard fails while another window has it open, which is
// usually for a moment (Flutter's ScopedClipboard::Open tries 3 times; the clipboard managers and
// remote desktops of today hold it longer).
const (
	clipboardTries         = 10
	clipboardRetryInterval = 5 * time.Millisecond
	// clipboardMaxImagePixels is the largest image that is converted, so that a hostile or damaged
	// header cannot ask for gigabytes.
	clipboardMaxImagePixels = 1 << 28
)

// clipboardAPI is the clipboard of Windows as the code of this file uses it. clipboard_windows.go
// implements it with the Win32 calls; the tests use a fake.
type clipboardAPI interface {
	// Open opens the clipboard with the window as the owner (0 to read). busy says that another
	// window has it open, so that trying again can work.
	Open(owner uintptr) (busy bool, err error)
	// Close closes the clipboard that Open opened.
	Close()
	// Empty empties the clipboard and makes the owner of Open its owner.
	Empty() error
	// Set puts the data on the clipboard in the format.
	Set(format uint32, data []byte) error
	// Get returns a copy of the data of the format; present is false when the clipboard has none.
	Get(format uint32) (data []byte, present bool, err error)
	// Has says whether the clipboard has the format, without opening it.
	Has(format uint32) bool
	// PNGFormat is the number of the registered clipboard format "PNG".
	PNGFormat() (uint32, error)
}

// clipboardCore is the logic of the clipboard over the API of Windows.
type clipboardCore struct {
	api clipboardAPI
	// sleep waits between the tries of a busy clipboard.
	sleep func(time.Duration)
}

// textToClipboard turns the app's text into the UTF-16 of CF_UNICODETEXT, with a terminating NUL:
// every line break (\n, \r\n or a lone \r) is \r\n, and the text ends at its first NUL, since a
// reader of the format could not see further (SPEC §25.7). Invalid UTF-8 becomes U+FFFD.
func textToClipboard(s string) []uint16 {
	if i := strings.IndexByte(s, 0); i >= 0 {
		s = s[:i]
	}
	out := make([]uint16, 0, len(s)+len(s)/16+1)
	var buf [2]uint16
	for i := 0; i < len(s); {
		r, n := utf8.DecodeRuneInString(s[i:])
		i += n
		switch {
		case r == '\r':
			if i < len(s) && s[i] == '\n' {
				i++
			}
			out = append(out, '\r', '\n')
		case r == '\n':
			out = append(out, '\r', '\n')
		case r >= 0x10000:
			r1, r2 := utf16.EncodeRune(r)
			buf[0], buf[1] = uint16(r1), uint16(r2)
			out = append(out, buf[:]...)
		default:
			out = append(out, uint16(r))
		}
	}
	return append(out, 0)
}

// textBytes is the UTF-16 as the little-endian bytes of the clipboard.
func textBytes(u []uint16) []byte {
	b := make([]byte, 2*len(u))
	for i, c := range u {
		binary.LittleEndian.PutUint16(b[2*i:], c)
	}
	return b
}

// textFromClipboard turns the bytes of CF_UNICODETEXT into the app's text: it ends at the first
// NUL (the memory of the clipboard can be larger than the text), a lone surrogate becomes U+FFFD,
// and \r\n and a lone \r become \n.
func textFromClipboard(b []byte) string {
	u := make([]uint16, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		c := binary.LittleEndian.Uint16(b[i:])
		if c == 0 {
			break
		}
		u = append(u, c)
	}
	s := string(utf16.Decode(u))
	if !strings.ContainsRune(s, '\r') {
		return s
	}
	var sb strings.Builder
	sb.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\r' {
			if i+1 < len(s) && s[i+1] == '\n' {
				i++
			}
			sb.WriteByte('\n')
			continue
		}
		sb.WriteByte(s[i])
	}
	return sb.String()
}

// pathFromURI turns a file URI into a Windows path: file:///C:/Users/me/a%20b.txt is
// C:\Users\me\a b.txt, file://server/share/a.txt is the network path \\server\share\a.txt, and a
// path with no drive (file:///home/me/a.txt) is the rooted path \home\me\a.txt. ok is false for a
// URI that is not a file URI or cannot be read.
func pathFromURI(uri string) (path string, ok bool) {
	u, err := url.Parse(uri)
	if err != nil || !strings.EqualFold(u.Scheme, "file") || u.Path == "" {
		return "", false
	}
	p := u.Path
	if len(p) >= 3 && p[0] == '/' && isDriveLetter(p[1]) && p[2] == ':' {
		p = p[1:]
	}
	p = strings.ReplaceAll(p, "/", `\`)
	if u.Host != "" && !strings.EqualFold(u.Host, "localhost") {
		return `\\` + u.Host + p, true
	}
	return p, true
}

func isDriveLetter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

// uriFromPath turns a Windows path into a file URI, the form of the other platforms'
// text/uri-list: C:\Users\me\a b.txt is file:///C:/Users/me/a%20b.txt and \\server\share\a.txt
// is file://server/share/a.txt. The prefixes \\?\ and \\?\UNC\ of long paths are removed.
func uriFromPath(path string) string {
	switch {
	case strings.HasPrefix(path, `\\?\UNC\`):
		path = `\\` + path[len(`\\?\UNC\`):]
	case strings.HasPrefix(path, `\\?\`):
		path = path[len(`\\?\`):]
	}
	u := url.URL{Scheme: "file"}
	slashed := strings.ReplaceAll(path, `\`, "/")
	if strings.HasPrefix(slashed, "//") {
		host, rest, _ := strings.Cut(slashed[2:], "/")
		u.Host, u.Path = host, "/"+rest
	} else {
		if !strings.HasPrefix(slashed, "/") {
			slashed = "/" + slashed
		}
		u.Path = slashed
	}
	return u.String()
}

// dropFilesFromPaths makes the data of CF_HDROP: a DROPFILES whose paths are UTF-16, each ended by
// a NUL, and the list by one more NUL.
func dropFilesFromPaths(paths []string) []byte {
	head := win32.DropFiles{PFiles: dropFilesHeaderSize, FWide: 1}
	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.LittleEndian, head) // a bytes.Buffer does not fail
	for _, p := range paths {
		if i := strings.IndexByte(p, 0); i >= 0 {
			p = p[:i]
		}
		for _, c := range utf16.Encode([]rune(p)) {
			_ = binary.Write(&buf, binary.LittleEndian, c)
		}
		_ = binary.Write(&buf, binary.LittleEndian, uint16(0))
	}
	_ = binary.Write(&buf, binary.LittleEndian, uint16(0))
	return buf.Bytes()
}

// dropFilesHeaderSize is sizeof(DROPFILES).
const dropFilesHeaderSize = 20

// pathsFromDropFiles reads the paths of the data of CF_HDROP. A list whose header is damaged is
// not a list: it fails with ClipboardReadFailed. Paths that are not UTF-16 (fWide is 0) are read as
// Latin-1, since the ANSI code page is not known to the plain-Go code.
func pathsFromDropFiles(b []byte) ([]string, error) {
	if len(b) < dropFilesHeaderSize {
		return nil, errors.New(tagClipboardReadFailed, "the files of the clipboard have no header", "format", "CF_HDROP", "size", len(b))
	}
	pFiles := binary.LittleEndian.Uint32(b[0:])
	wide := binary.LittleEndian.Uint32(b[16:]) != 0
	if pFiles < dropFilesHeaderSize || int64(pFiles) > int64(len(b)) {
		return nil, errors.New(tagClipboardReadFailed, "the file list of the clipboard starts outside its data", "format", "CF_HDROP", "pFiles", int(pFiles), "size", len(b))
	}
	data := b[pFiles:]
	var paths []string
	if !wide {
		for len(data) > 0 && data[0] != 0 {
			n := bytes.IndexByte(data, 0)
			if n < 0 {
				n = len(data)
			}
			rs := make([]rune, n)
			for i := range rs {
				rs[i] = rune(data[i])
			}
			paths = append(paths, string(rs))
			if n == len(data) {
				break
			}
			data = data[n+1:]
		}
		return paths, nil
	}
	u := make([]uint16, len(data)/2)
	for i := range u {
		u[i] = binary.LittleEndian.Uint16(data[2*i:])
	}
	for len(u) > 0 && u[0] != 0 {
		n := 0
		for n < len(u) && u[n] != 0 {
			n++
		}
		paths = append(paths, string(utf16.Decode(u[:n])))
		if n == len(u) {
			break
		}
		u = u[n+1:]
	}
	return paths, nil
}

// dibV5FromPNG makes the data of CF_DIBV5 from a PNG: a BITMAPV5HEADER with the masks of 32-bit
// BGRA and the sRGB color space, then the pixels, bottom row first, not premultiplied. It fails
// with ClipboardWriteFailed for data that is not a PNG, or one too large.
func dibV5FromPNG(data []byte) ([]byte, error) {
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, errors.NewErr(tagClipboardWriteFailed, err).SetMetadata("format", "image/png")
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > clipboardMaxImagePixels {
		return nil, errors.New(tagClipboardWriteFailed, "the image is empty or too large for the clipboard", "format", "image/png", "width", cfg.Width, "height", cfg.Height)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, errors.NewErr(tagClipboardWriteFailed, err).SetMetadata("format", "image/png")
	}
	b := img.Bounds()
	nrgba := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(nrgba, nrgba.Bounds(), img, b.Min, draw.Src)
	w, h := b.Dx(), b.Dy()
	head := win32.BitmapV5Header{
		Size: 124, Width: int32(w), Height: int32(h), Planes: 1, BitCount: 32, Compression: win32.BIBitfields,
		SizeImage: uint32(w * h * 4), XPelsPerMeter: 2835, YPelsPerMeter: 2835,
		RedMask: 0x00FF0000, GreenMask: 0x0000FF00, BlueMask: 0x000000FF, AlphaMask: 0xFF000000,
		CSType: win32.LCSsRGB, Intent: win32.LCSGMImages,
	}
	var buf bytes.Buffer
	buf.Grow(124 + w*h*4)
	_ = binary.Write(&buf, binary.LittleEndian, head) // a bytes.Buffer does not fail
	row := make([]byte, w*4)
	for y := h - 1; y >= 0; y-- {
		src := nrgba.Pix[y*nrgba.Stride : y*nrgba.Stride+w*4]
		for x := range w {
			row[4*x+0], row[4*x+1], row[4*x+2], row[4*x+3] = src[4*x+2], src[4*x+1], src[4*x+0], src[4*x+3]
		}
		buf.Write(row)
	}
	return buf.Bytes(), nil
}

// pngFromDIB makes a PNG from the data of CF_DIBV5 or CF_DIB (a BITMAPINFOHEADER or a later header,
// the color table, the pixels): the data gets the header of a BMP file and golang.org/x/image/bmp
// reads it (SPEC §25.7). It fails with ClipboardReadFailed when the bitmap is damaged or of a kind
// the reader does not take (compressed, or with masks that are not BGRA).
func pngFromDIB(dib []byte) ([]byte, error) {
	if len(dib) < 40 {
		return nil, errors.New(tagClipboardReadFailed, "the bitmap of the clipboard has no header", "format", "CF_DIB", "size", len(dib))
	}
	headSize := binary.LittleEndian.Uint32(dib[0:])
	width := int32(binary.LittleEndian.Uint32(dib[4:]))
	height := int32(binary.LittleEndian.Uint32(dib[8:]))
	bpp := binary.LittleEndian.Uint16(dib[14:])
	compression := binary.LittleEndian.Uint32(dib[16:])
	clrUsed := binary.LittleEndian.Uint32(dib[32:])
	if headSize < 40 || int64(headSize) > int64(len(dib)) {
		return nil, errors.New(tagClipboardReadFailed, "the bitmap of the clipboard has a damaged header", "format", "CF_DIB", "headerSize", int(headSize), "size", len(dib))
	}
	w, h := int64(width), int64(height)
	if h < 0 {
		h = -h
	}
	if w <= 0 || w*h > clipboardMaxImagePixels {
		return nil, errors.New(tagClipboardReadFailed, "the bitmap of the clipboard is empty or too large", "format", "CF_DIB", "width", int(width), "height", int(height))
	}
	offset := 14 + int64(headSize)
	if bpp <= 8 {
		n := int64(clrUsed)
		if n == 0 {
			n = 1 << bpp
		}
		offset += 4 * n
	} else if headSize == 40 && compression == win32.BIBitfields {
		offset += 12 // the three masks that follow the header
	}
	file := make([]byte, 14, 14+len(dib))
	file[0], file[1] = 'B', 'M'
	binary.LittleEndian.PutUint32(file[2:], uint32(14+len(dib)))
	binary.LittleEndian.PutUint32(file[10:], uint32(offset))
	file = append(file, dib...)
	img, err := bmp.Decode(bytes.NewReader(file))
	if err != nil {
		return nil, errors.NewErr(tagClipboardReadFailed, err).SetMetadata("format", "CF_DIB", "bitsPerPixel", int(bpp), "compression", int(compression))
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return nil, errors.NewErr(tagClipboardReadFailed, err).SetMetadata("format", "CF_DIB")
	}
	return out.Bytes(), nil
}

// open opens the clipboard, trying again while another window has it open: up to clipboardTries
// times, clipboardRetryInterval apart. A clipboard that stays busy is ClipboardBusy.
func (c *clipboardCore) open(owner uintptr) error {
	for try := range clipboardTries {
		busy, err := c.api.Open(owner)
		if err == nil {
			return nil
		}
		if !busy {
			return err
		}
		if try < clipboardTries-1 {
			c.sleep(clipboardRetryInterval)
		}
	}
	return errors.New(tagClipboardBusy, "another program kept the clipboard open", "tries", clipboardTries, "interval", clipboardRetryInterval.String())
}

// write puts the content on the clipboard, all of it as one copy: the formats are made first, so a
// content that cannot be written changes nothing, then the clipboard is emptied and filled. An
// empty content clears the clipboard. It runs on the UI thread with the window as the owner. A file
// that is not a file URI, or an image that is not a PNG, fails with WindowsClipboardFormatUnsupported
// and ClipboardWriteFailed.
func (c *clipboardCore) write(owner uintptr, content ui.ClipboardContent) error {
	type item struct {
		format uint32
		data   []byte
	}
	var items []item
	if content.HasText {
		items = append(items, item{win32.CFUnicodeText, textBytes(textToClipboard(content.Text))})
	}
	if len(content.URIs) > 0 {
		paths := make([]string, len(content.URIs))
		for i, uri := range content.URIs {
			p, ok := pathFromURI(uri)
			if !ok {
				return errors.New(tagWindowsClipboardFormatUnsupported, "the clipboard of Windows takes the URIs of files only", "uri", uri)
			}
			paths[i] = p
		}
		items = append(items, item{win32.CFHDrop, dropFilesFromPaths(paths)})
	}
	if len(content.PNG) > 0 {
		pngFormat, err := c.api.PNGFormat()
		if err != nil {
			return err
		}
		dib, err := dibV5FromPNG(content.PNG)
		if err != nil {
			return err
		}
		items = append(items, item{pngFormat, content.PNG}, item{win32.CFDIBV5, dib})
	}
	if err := c.open(owner); err != nil {
		return err
	}
	defer c.api.Close()
	if err := c.api.Empty(); err != nil {
		return err
	}
	for _, it := range items {
		if err := c.api.Set(it.format, it.data); err != nil {
			return err
		}
	}
	return nil
}

// read reads the content of the clipboard in the format (one of the ui.ClipboardFormat constants;
// "" is the text). It runs on a thread of its own. A format the clipboard lacks reads as an empty
// content, without opening the clipboard.
func (c *clipboardCore) read(format string) (ui.ClipboardContent, error) {
	pngFormat, err := c.api.PNGFormat()
	if err != nil {
		return ui.ClipboardContent{}, err
	}
	var wanted []uint32
	switch format {
	case "", ui.ClipboardFormatText:
		wanted = []uint32{win32.CFUnicodeText}
	case ui.ClipboardFormatURIList:
		wanted = []uint32{win32.CFHDrop}
	case ui.ClipboardFormatPNG:
		wanted = []uint32{pngFormat, win32.CFDIBV5, win32.CFDIB}
	default:
		return ui.ClipboardContent{}, nil
	}
	available := false
	for _, f := range wanted {
		if c.api.Has(f) {
			available = true
			break
		}
	}
	if !available {
		return ui.ClipboardContent{}, nil
	}
	if err := c.open(0); err != nil {
		return ui.ClipboardContent{}, err
	}
	defer c.api.Close()
	for _, f := range wanted {
		if !c.api.Has(f) {
			continue
		}
		data, present, err := c.api.Get(f)
		if err != nil {
			return ui.ClipboardContent{}, err
		}
		if !present {
			continue
		}
		switch f {
		case win32.CFUnicodeText:
			return ui.ClipboardContent{Text: textFromClipboard(data), HasText: true}, nil
		case win32.CFHDrop:
			paths, err := pathsFromDropFiles(data)
			if err != nil {
				return ui.ClipboardContent{}, err
			}
			uris := make([]string, len(paths))
			for i, p := range paths {
				uris[i] = uriFromPath(p)
			}
			return ui.ClipboardContent{URIs: uris}, nil
		case win32.CFDIBV5, win32.CFDIB:
			out, err := pngFromDIB(data)
			if err != nil {
				return ui.ClipboardContent{}, err
			}
			return ui.ClipboardContent{PNG: out}, nil
		default: // the registered format PNG
			if _, err := png.DecodeConfig(bytes.NewReader(data)); err != nil {
				return ui.ClipboardContent{}, errors.NewErr(tagClipboardReadFailed, err).SetMetadata("format", "PNG")
			}
			return ui.ClipboardContent{PNG: data}, nil
		}
	}
	return ui.ClipboardContent{}, nil
}

// asyncRead runs read on a goroutine of its own, on a locked OS thread (the clipboard belongs to the
// thread that opened it, and a program that renders its data on demand answers only when its own
// message loop runs), and gives the result to done through post, on the UI thread, never before
// asyncRead has returned. A read that ctx ends first gives ClipboardReadTimeout, once: the thread
// finishes later and its result is dropped.
func asyncRead[T any](ctx context.Context, post func(func()), read func() (T, error), done func(T, error)) {
	var zero T
	finished := false // belongs to the UI thread
	finish := func(v T, err error) {
		if finished {
			return
		}
		finished = true
		done(v, err)
	}
	if ctx.Err() != nil {
		err := errors.NewErr(tagClipboardReadTimeout, ctx.Err())
		post(func() { finish(zero, err) })
		return
	}
	stop := context.AfterFunc(ctx, func() {
		err := errors.NewErr(tagClipboardReadTimeout, ctx.Err())
		post(func() { finish(zero, err) })
	})
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		v, err := read()
		stop()
		post(func() { finish(v, err) })
	}()
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagClipboardBusy                     = "ClipboardBusy"
	tagClipboardReadFailed               = "ClipboardReadFailed"
	tagClipboardReadTimeout              = "ClipboardReadTimeout"
	tagClipboardWriteFailed              = "ClipboardWriteFailed"
	tagWindowsClipboardFormatUnsupported = "WindowsClipboardFormatUnsupported"
)
