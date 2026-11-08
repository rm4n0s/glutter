//go:build linux && !android

package x11

import (
	"image"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// fakeX is the Xlib, the MIT-SHM extension and the System V shared memory of one display, for the
// tests of the window that shows pixels: it records the requests, makes the XImages in C memory
// and lets the test answer with completion events. No server and no window are needed.
type fakeX struct {
	t *testing.T
	p *Platform
	w *window

	// What the server and the libraries are like.
	display     string // what XDisplayString says
	noExt       bool   // libXext cannot be loaded
	noShm       bool   // the server has no MIT-SHM
	noSysv      bool   // libc has no System V shared memory
	attachFails bool   // the server answers XShmAttach with an error
	shmgetFails bool
	shmatFails  bool
	createFails bool // XCreateImage and XShmCreateImage return NULL
	noGC        bool
	// layout of the images the server makes.
	depth, bits, order int32
	red, green, blue   uint64

	// What happened.
	puts      []put
	nextSeg   uint64
	attached  map[uint64]bool // the segments the server has
	detached  []uint64
	images    int // XImages alive
	destroyed int
	segments  int // shared segments alive in this process
	removed   int // shmctl IPC_RMID calls
	gcs       int
	syncs     int
	flushes   int
	badData   int // images destroyed while they still pointed at memory that Xlib would free
}

// put is a PutImage or a ShmPutImage request.
type put struct {
	shm           bool
	x, y, w, h    int
	sendEvent     bool
	seg           uint64
	srcX, srcY    int
	dstX, dstY    int
	image, window uintptr
}

func newFakeX(t *testing.T) *fakeX {
	f := &fakeX{t: t, display: ":0", depth: 24, bits: 32, red: 0xFF0000, green: 0xFF00, blue: 0xFF, attached: map[uint64]bool{}}
	p := &Platform{display: 1, root: 1, windows: map[uint64]*window{}, getenv: func(string) string { return "" }, event: dl.Calloc(1, xEventSize)}
	t.Cleanup(func() { dl.Free(p.event) })
	p.x = &xlib{
		displayString: func(uintptr) uintptr { return dl.CString(f.display) }, // a few bytes of C memory per test
		defaultVisual: func(uintptr, int32) uintptr { return 0x7000 },
		defaultDepth:  func(uintptr, int32) int32 { return 24 },
		createGC: func(uintptr, uint64, uint64, uintptr) uintptr {
			if f.noGC {
				return 0
			}
			f.gcs++
			return 0x6000
		},
		freeGC: func(uintptr, uintptr) int32 { f.gcs--; return 0 },
		createImage: func(_, _ uintptr, _ uint32, _, _ int32, _ uintptr, w, h uint32, _, _ int32) uintptr {
			return f.image(int32(w), int32(h), 0)
		},
		putImage: func(_ uintptr, window uint64, _ uintptr, img uintptr, sx, sy, dx, dy int32, w, h uint32) int32 {
			f.puts = append(f.puts, put{x: int(dx), y: int(dy), w: int(w), h: int(h), srcX: int(sx), srcY: int(sy), dstX: int(dx), dstY: int(dy), image: img, window: uintptr(window)})
			return 0
		},
		destroyImage: func(img uintptr) int32 {
			im := view[xImage](img)
			if im.data != 0 || im.obdata != 0 {
				f.badData++
			}
			f.destroyed++
			f.images--
			dl.Free(img)
			return 0
		},
		sync:  func(uintptr, int32) int32 { f.syncs++; return 0 },
		flush: func(uintptr) int32 { f.flushes++; return 0 },
	}
	p.loadExt = func() (*xext, error) {
		if f.noExt {
			return nil, errors.New(tagTestNoLibrary, "libXext is not there")
		}
		return &xext{
			queryExtension: func(uintptr) int32 {
				if f.noShm {
					return 0
				}
				return 1
			},
			getEventBase: func(uintptr) int32 { return 65 },
			createImage: func(_, _ uintptr, _ uint32, _ int32, _, info uintptr, w, h uint32) uintptr {
				return f.image(int32(w), int32(h), info)
			},
			attach: func(_, info uintptr) int32 {
				if f.attachFails {
					p.xErrors++
					return 1
				}
				f.nextSeg++
				view[xShmSegmentInfo](info).shmseg = 0x500000 + f.nextSeg
				f.attached[0x500000+f.nextSeg] = true
				return 1
			},
			detach: func(_, info uintptr) int32 {
				seg := view[xShmSegmentInfo](info).shmseg
				delete(f.attached, seg)
				f.detached = append(f.detached, seg)
				return 1
			},
			putImage: func(_ uintptr, window uint64, _ uintptr, img uintptr, sx, sy, dx, dy int32, w, h uint32, send int32) int32 {
				info := view[xImage](img).obdata
				f.puts = append(f.puts, put{shm: true, x: int(dx), y: int(dy), w: int(w), h: int(h), srcX: int(sx), srcY: int(sy), dstX: int(dx), dstY: int(dy),
					sendEvent: send != 0, seg: view[xShmSegmentInfo](info).shmseg, image: img, window: uintptr(window)})
				return 1
			},
		}, nil
	}
	p.loadSysv = func() (*sysv, error) {
		if f.noSysv {
			return nil, errors.New(tagTestNoLibrary, "libc has no System V shared memory")
		}
		sizes := map[uintptr]uintptr{}
		return &sysv{
			shmget: func(_ int32, size uintptr, flags int32) int32 {
				if f.shmgetFails || flags&ipcCreat == 0 {
					return -1
				}
				return int32(size)
			},
			shmat: func(id int32, _ uintptr, _ int32) uintptr {
				if f.shmatFails {
					return ^uintptr(0)
				}
				mem := dl.Calloc(1, uintptr(id))
				sizes[mem] = uintptr(id)
				f.segments++
				return mem
			},
			shmdt: func(addr uintptr) int32 {
				delete(sizes, addr)
				f.segments--
				dl.Free(addr)
				return 0
			},
			shmctl: func(_, cmd int32, _ uintptr) int32 {
				if cmd == ipcRmid {
					f.removed++
				}
				return 0
			},
		}, nil
	}
	f.p = p
	f.w = &window{p: p, xid: 42, mapped: true, visual: 0x7100, depth: 24}
	p.windows[42] = f.w
	return f
}

// image makes the XImage of the fake server in C memory.
func (f *fakeX) image(w, h int32, info uintptr) uintptr {
	if f.createFails {
		return 0
	}
	img := dl.Calloc(1, 136)
	im := view[xImage](img)
	im.width, im.height, im.format, im.depth = w, h, zPixmap, f.depth
	im.bitsPerPixel, im.byteOrder, im.bytesPerLine = f.bits, f.order, 4*w
	im.redMask, im.greenMask, im.blueMask = f.red, f.green, f.blue
	im.obdata = info
	f.images++
	return img
}

// complete is the server's completion event for a segment, sent through the event queue as Xlib
// gives it to the platform.
func (f *fakeX) complete(seg uint64) {
	f.t.Helper()
	ev := view[xShmCompletionEvent](f.p.event)
	*ev = xShmCompletionEvent{typ: 65 + shmCompletion, shmseg: seg}
	f.p.handleEvent()
}

func rects(r ...image.Rectangle) []image.Rectangle { return r }

func TestDisplayIsLocal(t *testing.T) {
	tests := []struct {
		name    string
		display string
		want    bool
	}{
		{"the first display", ":0", true},
		{"a screen of it", ":0.1", true},
		{"a nested server", ":12", true},
		{"a unix socket named", "unix:0", true},
		{"no name", "", true},
		{"a host", "buildbox:0", false},
		{"the proxy of ssh -X", "localhost:10.0", false},
		{"an address", "192.168.1.5:0", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := displayIsLocal(tt.display); got != tt.want {
				t.Errorf("displayIsLocal(%q) = %v, want %v", tt.display, got, tt.want)
			}
		})
	}
}

func TestShmIsChosenOnlyWhenTheServerCanUseIt(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(f *fakeX)
		env    map[string]string
		wantOn bool
		reason string
		path   string
	}{
		{"a local server with the extension", func(*fakeX) {}, nil, true, "", "MIT-SHM"},
		{"the debug variable", func(*fakeX) {}, map[string]string{NoSHMEnv: "1"}, false, NoSHMEnv, "XPutImage"},
		{"the debug variable at 0 is ignored", func(*fakeX) {}, map[string]string{NoSHMEnv: "0"}, true, "", "MIT-SHM"},
		{"a remote display", func(f *fakeX) { f.display = "buildbox:0" }, nil, false, "not on this machine", "XPutImage"},
		{"the ssh -X proxy", func(f *fakeX) { f.display = "localhost:10.0" }, nil, false, "not on this machine", "XPutImage"},
		{"a server with no extension", func(f *fakeX) { f.noShm = true }, nil, false, "no MIT-SHM", "XPutImage"},
		{"no libXext", func(f *fakeX) { f.noExt = true }, nil, false, "libXext", "XPutImage"},
		{"no System V shared memory", func(f *fakeX) { f.noSysv = true }, nil, false, "System V", "XPutImage"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeX(t)
			tt.setup(f)
			f.p.getenv = func(k string) string { return tt.env[k] }
			if _, err := f.w.AcquirePixels(8, 8); err != nil {
				t.Fatal(err)
			}
			s := f.p.shm
			if s.on != tt.wantOn || !strings.Contains(s.reason, tt.reason) {
				t.Errorf("on = %v, reason %q; want %v and %q", s.on, s.reason, tt.wantOn, tt.reason)
			}
			if got := f.w.PixelPath(); got != tt.path {
				t.Errorf("PixelPath = %q, want %q", got, tt.path)
			}
			if tt.wantOn != (f.segments == 1) {
				t.Errorf("%d shared segments for the first buffer, want %v", f.segments, tt.wantOn)
			}
			f.w.closePixels()
			f.assertClean()
		})
	}
}

// assertClean checks that nothing of the window is left: images, segments, graphics contexts.
func (f *fakeX) assertClean() {
	f.t.Helper()
	if f.images != 0 || f.segments != 0 || f.gcs != 0 || len(f.p.shmSegs) != 0 || len(f.attached) != 0 {
		f.t.Errorf("left over: %d images, %d segments, %d graphics contexts, %d registered segments, %d attached at the server",
			f.images, f.segments, f.gcs, len(f.p.shmSegs), len(f.attached))
	}
	if f.badData != 0 {
		f.t.Errorf("%d images were destroyed while they still pointed at memory that Xlib frees", f.badData)
	}
}

func TestPixelPathBeforeTheFirstBuffer(t *testing.T) {
	f := newFakeX(t)
	if got := f.w.PixelPath(); got != "" {
		t.Errorf("PixelPath = %q before the first buffer, want none", got)
	}
}

func TestPresentSendsOnlyTheDamagedRectangles(t *testing.T) {
	tests := []struct {
		name   string
		damage []image.Rectangle
		want   []image.Rectangle // what the server is asked to copy
	}{
		{"one rectangle", rects(image.Rect(10, 20, 74, 84)), rects(image.Rect(10, 20, 74, 84))},
		{"two rectangles, in order", rects(image.Rect(0, 0, 64, 64), image.Rect(128, 64, 192, 128)), rects(image.Rect(0, 0, 64, 64), image.Rect(128, 64, 192, 128))},
		{"no damage says the whole window may have changed", nil, rects(image.Rect(0, 0, 256, 128))},
		{"a rectangle past the edge is cut", rects(image.Rect(200, 100, 300, 200)), rects(image.Rect(200, 100, 256, 128))},
		{"a rectangle outside is dropped", rects(image.Rect(300, 0, 400, 50), image.Rect(0, 0, 8, 8)), rects(image.Rect(0, 0, 8, 8))},
	}
	for _, mode := range []struct {
		name  string
		shm   bool
		setup func(*fakeX)
	}{{"MIT-SHM", true, func(*fakeX) {}}, {"XPutImage", false, func(f *fakeX) { f.p.getenv = func(string) string { return "1" } }}} {
		for _, tt := range tests {
			t.Run(mode.name+"/"+tt.name, func(t *testing.T) {
				f := newFakeX(t)
				if !mode.shm {
					f.p.getenv = func(k string) string {
						if k == NoSHMEnv {
							return "1"
						}
						return ""
					}
				}
				if _, err := f.w.AcquirePixels(256, 128); err != nil {
					t.Fatal(err)
				}
				if err := f.w.PresentPixels(tt.damage); err != nil {
					t.Fatal(err)
				}
				if len(f.puts) != len(tt.want) {
					t.Fatalf("%d requests, want %d: %+v", len(f.puts), len(tt.want), f.puts)
				}
				for i, p := range f.puts {
					got := image.Rect(p.x, p.y, p.x+p.w, p.y+p.h)
					if got != tt.want[i] || p.srcX != p.dstX || p.srcY != p.dstY {
						t.Errorf("request %d copies %v from (%d,%d) to (%d,%d), want %v at the same place", i, got, p.srcX, p.srcY, p.dstX, p.dstY, tt.want[i])
					}
					if p.shm != mode.shm || p.window != 42 {
						t.Errorf("request %d: shm = %v, window %d", i, p.shm, p.window)
					}
					// Only the last request of a frame asks for the completion event: the server
					// handles the requests in order, and the first event would give the buffer back
					// while the others still read it.
					if want := mode.shm && i == len(f.puts)-1; p.sendEvent != want {
						t.Errorf("request %d asks for the completion event = %v, want %v", i, p.sendEvent, want)
					}
				}
				if f.flushes == 0 {
					t.Error("the requests were not flushed")
				}
				f.w.closePixels()
				f.assertClean()
			})
		}
	}
}

func TestAFrameWithNoBufferSendsNothing(t *testing.T) {
	f := newFakeX(t)
	if _, err := f.w.AcquirePixels(64, 64); err != nil {
		t.Fatal(err)
	}
	if err := f.w.PresentPixels(rects(image.Rect(0, 0, 64, 64))); err != nil {
		t.Fatal(err)
	}
	before := len(f.puts)
	if err := f.w.PresentPixels(nil); err != nil { // a frame that changed nothing: no buffer was acquired
		t.Fatal(err)
	}
	if len(f.puts) != before {
		t.Errorf("a frame with no damage sent %d requests", len(f.puts)-before)
	}
	// And a window that never drew has nothing to present.
	g := newFakeX(t)
	if err := g.w.PresentPixels(nil); err != nil || len(g.puts) != 0 {
		t.Errorf("PresentPixels on a window with no buffers: %v, %d requests", err, len(g.puts))
	}
	f.w.closePixels()
	f.assertClean()
}

func TestAWindowThatIsNotOnTheScreenSendsNothing(t *testing.T) {
	for _, tt := range []struct {
		name  string
		setup func(w *window)
	}{
		{"not mapped", func(w *window) { w.mapped = false }},
		{"closed", func(w *window) { w.closed = true }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeX(t)
			if _, err := f.w.AcquirePixels(64, 64); err != nil {
				t.Fatal(err)
			}
			tt.setup(f.w)
			if err := f.w.PresentPixels(rects(image.Rect(0, 0, 8, 8))); err != nil || len(f.puts) != 0 {
				t.Errorf("PresentPixels: %v, %d requests; want none", err, len(f.puts))
			}
			f.w.closed = false
			f.w.closePixels()
			f.assertClean()
		})
	}
}

func TestAnSHMBufferComesBackWhenTheServerHasReadIt(t *testing.T) {
	f := newFakeX(t)
	acquire := func() (gpu.PixelBuffer, uint64) {
		t.Helper()
		b, err := f.w.AcquirePixels(64, 64)
		if err != nil {
			t.Fatal(err)
		}
		return b, f.w.pix.cur.Buf.(*xBuffer).shmseg
	}
	b1, seg1 := acquire()
	if b1.Age != 0 || b1.Format != gpu.PixelFormatBGRX8888 || b1.Stride != 256 || b1.Width != 64 || b1.Height != 64 || len(b1.Pix) != 256*64 {
		t.Errorf("first buffer = age %d, format %v, stride %d, %dx%d, %d bytes", b1.Age, b1.Format, b1.Stride, b1.Width, b1.Height, len(b1.Pix))
	}
	if err := f.w.PresentPixels(rects(image.Rect(0, 0, 64, 64))); err != nil {
		t.Fatal(err)
	}
	// The server has not read the first buffer: the next frame gets another segment, of no age.
	b2, seg2 := acquire()
	if seg2 == seg1 || b2.Age != 0 || &b2.Pix[0] == &b1.Pix[0] {
		t.Errorf("second buffer is the first again, or has an age (%d)", b2.Age)
	}
	if err := f.w.PresentPixels(rects(image.Rect(0, 0, 8, 8))); err != nil {
		t.Fatal(err)
	}
	// The completion event of the first frame arrives through the event loop's dispatch.
	f.complete(seg1)
	b3, seg3 := acquire()
	if seg3 != seg1 || b3.Age != 2 {
		t.Errorf("after the completion of the first buffer: segment %#x age %d, want %#x and age 2", seg3, b3.Age, seg1)
	}
	if err := f.w.PresentPixels(rects(image.Rect(0, 0, 8, 8))); err != nil {
		t.Fatal(err)
	}
	f.complete(0xDEAD) // an event of a segment that is not ours (or was freed) does nothing
	f.complete(seg2)
	f.complete(seg3)
	b4, seg4 := acquire()
	if seg4 != seg3 || b4.Age != 1 {
		t.Errorf("the buffer written last comes first: segment %#x age %d, want %#x and age 1", seg4, b4.Age, seg3)
	}
	f.w.closePixels()
	f.assertClean()
	if f.removed != 2 {
		t.Errorf("%d segments were removed from the system, want 2 (each as soon as the server had it)", f.removed)
	}
}

func TestAnEventThatIsNotTheCompletionDoesNotGiveTheBufferBack(t *testing.T) {
	f := newFakeX(t)
	if _, err := f.w.AcquirePixels(8, 8); err != nil {
		t.Fatal(err)
	}
	first := f.w.pix.cur.Buf.(*xBuffer).shmseg
	if err := f.w.PresentPixels(rects(image.Rect(0, 0, 8, 8))); err != nil {
		t.Fatal(err)
	}
	// An event one past the completion's type is not ShmCompletion, whatever its segment says.
	*view[xShmCompletionEvent](f.p.event) = xShmCompletionEvent{typ: 65 + shmCompletion + 1, shmseg: first}
	f.p.handleEvent()
	if _, err := f.w.AcquirePixels(8, 8); err != nil {
		t.Fatal(err)
	}
	if got := f.w.pix.cur.Buf.(*xBuffer).shmseg; got == first {
		t.Error("a buffer that the server still reads was handed out again")
	}
	f.w.closePixels()
	f.assertClean()
}

func TestPlainBuffersAreReusedAtOnce(t *testing.T) {
	f := newFakeX(t)
	f.p.getenv = func(k string) string {
		if k == NoSHMEnv {
			return "1"
		}
		return ""
	}
	var first *byte
	for i, wantAge := range []int{0, 1, 1} {
		b, err := f.w.AcquirePixels(64, 32)
		if err != nil {
			t.Fatal(err)
		}
		if b.Age != wantAge {
			t.Errorf("frame %d: age %d, want %d", i, b.Age, wantAge)
		}
		if first == nil {
			first = &b.Pix[0]
		} else if &b.Pix[0] != first {
			t.Errorf("frame %d: a new buffer, want the one that XPutImage has already copied", i)
		}
		if err := f.w.PresentPixels(rects(image.Rect(0, 0, 4, 4))); err != nil {
			t.Fatal(err)
		}
	}
	if f.images != 1 || f.segments != 0 {
		t.Errorf("%d images and %d segments for a window that sends requests, want 1 and 0", f.images, f.segments)
	}
	f.w.closePixels()
	f.assertClean()
}

func TestAServerThatDoesNotAttachTheSegmentGetsRequests(t *testing.T) {
	f := newFakeX(t)
	f.attachFails = true
	b, err := f.w.AcquirePixels(64, 64)
	if err != nil {
		t.Fatal(err)
	}
	if got := f.w.PixelPath(); got != "XPutImage" {
		t.Errorf("PixelPath = %q after the server refused the segment, want XPutImage", got)
	}
	if f.segments != 0 || f.removed != 1 || f.images != 1 || f.destroyed != 1 {
		t.Errorf("after the refusal: %d segments, %d removed, %d images, %d destroyed; want 0, 1, 1, 1 (the segment is removed and unmapped, its image destroyed)",
			f.segments, f.removed, f.images, f.destroyed)
	}
	if err := f.w.PresentPixels(rects(image.Rect(0, 0, 64, 64))); err != nil {
		t.Fatal(err)
	}
	if len(f.puts) != 1 || f.puts[0].shm || b.Age != 0 {
		t.Errorf("requests %+v, age %d: want one PutImage and no age", f.puts, b.Age)
	}
	// It is not asked again: the next buffer of the window is plain at once.
	f.attachFails = false
	if _, err := f.w.AcquirePixels(128, 64); err != nil {
		t.Fatal(err)
	}
	if len(f.attached) != 0 || f.nextSeg != 0 {
		t.Errorf("the window tried MIT-SHM again after the server refused it")
	}
	f.w.closePixels()
	f.assertClean()
}

func TestAResizeFreesTheBuffersOfTheOldSize(t *testing.T) {
	f := newFakeX(t)
	for i, size := range [][2]int{{64, 64}, {64, 64}, {128, 64}} {
		b, err := f.w.AcquirePixels(size[0], size[1])
		if err != nil {
			t.Fatal(err)
		}
		if b.Width != size[0] || b.Height != size[1] || b.Stride != 4*size[0] {
			t.Errorf("frame %d: %dx%d stride %d", i, b.Width, b.Height, b.Stride)
		}
		if err := f.w.PresentPixels(rects(image.Rect(0, 0, size[0], size[1]))); err != nil {
			t.Fatal(err)
		}
		if i == 1 {
			for seg := range f.attached {
				f.complete(seg)
			}
		}
	}
	// Of the two 64x64 segments the server has read, the one still idle was freed when the new size
	// was asked for, and the other is freed when its completion arrives.
	f.complete(0x500001)
	f.complete(0x500002)
	if len(f.p.shmSegs) != 1 || f.segments != 1 {
		t.Errorf("%d registered segments and %d mapped after the resize, want 1 and 1", len(f.p.shmSegs), f.segments)
	}
	f.w.closePixels()
	f.assertClean()
}

func TestClosingTheWindowGivesEverythingBack(t *testing.T) {
	f := newFakeX(t)
	for i := 0; i < 3; i++ {
		if _, err := f.w.AcquirePixels(64, 64); err != nil {
			t.Fatal(err)
		}
		if err := f.w.PresentPixels(rects(image.Rect(0, 0, 8, 8))); err != nil {
			t.Fatal(err)
		}
	}
	if f.segments != 3 || len(f.detached) != 0 {
		t.Fatalf("%d segments and %d detached before the close, want 3 and 0", f.segments, len(f.detached))
	}
	f.w.closePixels() // buffers the server still reads are given back too: it has its own mapping
	if len(f.detached) != 3 {
		t.Errorf("%d segments detached at the server, want 3", len(f.detached))
	}
	f.w.closePixels() // twice does nothing
	f.complete(0x500001)
	f.assertClean()
}

func TestClosingThePlatformFirstDoesNotTalkToTheServer(t *testing.T) {
	f := newFakeX(t)
	if _, err := f.w.AcquirePixels(64, 64); err != nil {
		t.Fatal(err)
	}
	f.p.display = 0 // the connection is closed: no request can be made
	f.w.closePixels()
	if len(f.detached) != 0 || f.gcs != 1 {
		t.Errorf("detached %d segments and the graphics contexts alive are %d with the display closed", len(f.detached), f.gcs)
	}
	f.gcs = 0 // the server frees these with the connection
	clear(f.attached)
	f.assertClean()
}

func TestPixelLayoutOK(t *testing.T) {
	good := xImage{depth: 24, bitsPerPixel: 32, byteOrder: lsbFirst, redMask: 0xFF0000, greenMask: 0xFF00, blueMask: 0xFF}
	tests := []struct {
		name string
		edit func(*xImage)
		want bool
	}{
		{"24 bits of color in 32", func(*xImage) {}, true},
		{"32 bits with an alpha", func(i *xImage) { i.depth = 32 }, true},
		{"16 bits", func(i *xImage) { i.depth, i.bitsPerPixel = 16, 16 }, false},
		{"24 bits packed in 24", func(i *xImage) { i.bitsPerPixel = 24 }, false},
		{"30 bits", func(i *xImage) { i.depth = 30 }, false},
		{"the first byte highest", func(i *xImage) { i.byteOrder = 1 }, false},
		{"red and blue swapped", func(i *xImage) { i.redMask, i.blueMask = 0xFF, 0xFF0000 }, false},
		{"another green", func(i *xImage) { i.greenMask = 0xFF0000 }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			im := good
			tt.edit(&im)
			if got := pixelLayoutOK(&im); got != tt.want {
				t.Errorf("pixelLayoutOK = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPixelErrorRoutes(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(f *fakeX)
		w, h   int
		tag    string
		route  string
		verify func(t *testing.T, f *fakeX)
	}{
		{"a window that is closed", func(f *fakeX) { f.w.closed = true }, 8, 8, tagX11PixelWindowClosed,
			"window.AcquirePixels.X11PixelWindowClosed", nil},
		{"no graphics context", func(f *fakeX) { f.noGC = true }, 8, 8, tagX11PixelsFailed,
			"window.AcquirePixels->window.pixelsOf.X11PixelsFailed", nil},
		{"a size that does not fit in memory", func(*fakeX) {}, 20000, 20000, tagX11PixelsFailed,
			"window.AcquirePixels->Pool.Acquire->window.pixelsOf.func1->window.newPixelBuffer.X11PixelsFailed", nil},
		{"no size", func(*fakeX) {}, 0, 8, tagX11PixelsFailed,
			"window.AcquirePixels->Pool.Acquire->window.pixelsOf.func1->window.newPixelBuffer.X11PixelsFailed", nil},
		{"XShmCreateImage fails", func(f *fakeX) { f.createFails = true }, 8, 8, tagX11PixelsFailed,
			"window.AcquirePixels->Pool.Acquire->window.pixelsOf.func1->window.newPixelBuffer->window.newSharedBuffer.X11PixelsFailed",
			func(t *testing.T, f *fakeX) {
				if f.segments != 0 || f.images != 0 {
					t.Errorf("%d segments and %d images left", f.segments, f.images)
				}
			}},
		{"XCreateImage fails", func(f *fakeX) { f.createFails = true; f.noShm = true }, 8, 8, tagX11PixelsFailed,
			"window.AcquirePixels->Pool.Acquire->window.pixelsOf.func1->window.newPixelBuffer->window.newPlainBuffer.X11PixelsFailed", nil},
		{"shmget fails", func(f *fakeX) { f.shmgetFails = true }, 8, 8, tagX11PixelsFailed,
			"window.AcquirePixels->Pool.Acquire->window.pixelsOf.func1->window.newPixelBuffer->window.newSharedBuffer.X11PixelsFailed",
			func(t *testing.T, f *fakeX) {
				if f.images != 0 {
					t.Errorf("%d images left", f.images)
				}
			}},
		{"shmat fails", func(f *fakeX) { f.shmatFails = true }, 8, 8, tagX11PixelsFailed,
			"window.AcquirePixels->Pool.Acquire->window.pixelsOf.func1->window.newPixelBuffer->window.newSharedBuffer.X11PixelsFailed",
			func(t *testing.T, f *fakeX) {
				if f.images != 0 || f.removed != 1 {
					t.Errorf("%d images left, %d segments removed: want 0 and 1 (the segment that was made is removed)", f.images, f.removed)
				}
			}},
		{"a visual of 16 bits", func(f *fakeX) { f.depth, f.bits, f.red, f.green, f.blue = 16, 16, 0xF800, 0x7E0, 0x1F }, 8, 8, tagX11PixelFormatUnsupported,
			"window.AcquirePixels->Pool.Acquire->window.pixelsOf.func1->window.newPixelBuffer.X11PixelFormatUnsupported",
			func(t *testing.T, f *fakeX) {
				if f.images != 0 || f.segments != 0 {
					t.Errorf("%d images and %d segments left after the format was refused", f.images, f.segments)
				}
			}},
		{"a server that is big-endian", func(f *fakeX) { f.order = 1 }, 8, 8, tagX11PixelFormatUnsupported,
			"window.AcquirePixels->Pool.Acquire->window.pixelsOf.func1->window.newPixelBuffer.X11PixelFormatUnsupported",
			func(t *testing.T, f *fakeX) {
				if f.images != 0 || f.segments != 0 {
					t.Errorf("%d images and %d segments left", f.images, f.segments)
				}
			}},
		{"a visual with red and blue swapped, without MIT-SHM", func(f *fakeX) { f.noShm = true; f.red, f.blue = 0xFF, 0xFF0000 }, 8, 8, tagX11PixelFormatUnsupported,
			"window.AcquirePixels->Pool.Acquire->window.pixelsOf.func1->window.newPixelBuffer.X11PixelFormatUnsupported",
			func(t *testing.T, f *fakeX) {
				if f.images != 0 {
					t.Errorf("%d images left", f.images)
				}
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeX(t)
			tt.setup(f)
			_, err := f.w.AcquirePixels(tt.w, tt.h)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != tt.tag || !appErr.HasRoute(tt.route) {
				t.Errorf("tag %s, route %s; want %s and %s", appErr.Tag, appErr.Route(), tt.tag, tt.route)
			}
			if tt.verify != nil {
				tt.verify(t, f)
			}
			f.w.closed = false
			f.w.closePixels()
			f.gcs = 0
			f.assertClean()
		})
	}
}

func TestAFailedBufferIsAskedForAgain(t *testing.T) {
	f := newFakeX(t)
	f.shmgetFails = true
	if _, err := f.w.AcquirePixels(8, 8); err == nil {
		t.Fatal("AcquirePixels succeeded with no memory")
	}
	f.shmgetFails = false
	if _, err := f.w.AcquirePixels(8, 8); err != nil {
		t.Fatalf("AcquirePixels after the memory came back: %v", err)
	}
	f.w.closePixels()
	f.assertClean()
}

func TestNewTakesOneOptionsValue(t *testing.T) {
	defer func() {
		v := recover()
		err, ok := v.(error)
		if !ok {
			t.Fatalf("panic value %v is not an error", v)
		}
		assertRoute(t, err, "New.MultipleOptions")
	}()
	_, _ = New("", Options{}, Options{})
}

// tagTestNoLibrary is the tag of the error that a library that cannot be loaded gives.
const tagTestNoLibrary = "TestNoLibrary"
