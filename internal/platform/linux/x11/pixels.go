//go:build linux && !android

package x11

import (
	"image"
	"log/slog"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
	"github.com/rm4n0s/glutter/internal/platform/linux/pixelpool"
)

// This file makes a window a gpu.PixelTarget, for the software renderer (SPEC §9.6): the renderer
// draws into a buffer of the window's pool and the window sends the damaged rectangles of it to
// the server. With MIT-SHM the buffer is a segment of shared memory that the server reads
// (XShmPutImage) and the pool gives it out again once the server's completion event says that the
// reads are done; without it (a remote display, a server with no extension, GLUTTER_X11_NOSHM=1) the
// buffer is memory of the process and XPutImage copies the rectangles into the connection.

var _ gpu.PixelTarget = (*window)(nil)

// NoSHMEnv is the variable that turns MIT-SHM off, so that the pixels go with XPutImage. It is a
// debug aid, for the display tests and for finding out whether the extension is at fault.
const NoSHMEnv = "GLUTTER_X11_NOSHM"

// maxPixelBytes bounds one buffer, so that a bad size cannot ask for gigabytes.
const maxPixelBytes = 1 << 30

// The System V constants of shmget and shmctl.
const (
	ipcCreat = 0o1000 // IPC_CREAT
	ipcRmid  = 0      // IPC_RMID
)

// shmState is what is known of MIT-SHM on the display: whether the pixels of a window can go
// through it, and if not, why not. It is made when the first window shows pixels.
type shmState struct {
	ext       *xext
	sysv      *sysv
	eventBase int32
	on        bool
	reason    string
}

// displayIsLocal says whether a display name is of a server on this machine, which can attach a
// segment of shared memory that this process made: ":0", ":0.0" and "unix:0" are, "host:0" is not
// (a remote X server, an ssh -X proxy).
func displayIsLocal(name string) bool {
	return name == "" || strings.HasPrefix(name, ":") || strings.HasPrefix(name, "unix:")
}

// shmOf looks for MIT-SHM the first time it is asked, and logs at Info level which way the pixels
// go.
func (p *Platform) shmOf() *shmState {
	if p.shm != nil {
		return p.shm
	}
	s := &shmState{}
	p.shm = s
	name := dl.GoString(p.x.displayString(p.display))
	switch {
	case p.getenv(NoSHMEnv) == "1":
		s.reason = NoSHMEnv + "=1"
	case !displayIsLocal(name):
		s.reason = "the display is not on this machine"
	default:
		ext, err := p.loadExt()
		if err != nil {
			s.reason = "libXext is not available"
			break
		}
		if ext.queryExtension(p.display) == 0 {
			s.reason = "the server has no MIT-SHM"
			break
		}
		sv, err := p.loadSysv()
		if err != nil {
			s.reason = "System V shared memory is not available"
			break
		}
		s.ext, s.sysv, s.eventBase, s.on = ext, sv, ext.getEventBase(p.display), true
	}
	if s.on {
		slog.Info("X11 pixels go to the server through shared memory", "path", "MIT-SHM", "display", name)
	} else {
		slog.Info("X11 pixels go to the server in requests", "path", "XPutImage", "reason", s.reason, "display", name)
	}
	return s
}

// onShmCompletion handles the server's word that it has read the segment of a request: the buffer
// can be drawn into again.
func (p *Platform) onShmCompletion(shmseg uint64) {
	if release := p.shmSegs[shmseg]; release != nil {
		release()
	}
}

// pixelState is what a window keeps for the software renderer: the buffers, the one that was handed
// out and not shown yet, and the graphics context of the requests.
type pixelState struct {
	pool *pixelpool.Pool
	cur  *pixelpool.Slot
	gc   uintptr
	// shmOK is whether a new buffer is tried with MIT-SHM; a server that does not attach a segment
	// turns it off for good.
	shmOK bool
}

// xBuffer is a buffer of the pool: an XImage over memory that is either a shared segment of MIT-SHM
// or memory of the process.
type xBuffer struct {
	p      *Platform
	img    uintptr
	data   []byte
	stride int
	// A shared segment has the XShmSegmentInfo that the extension takes, the address this process
	// attached the segment at, and the XID the server knows it by (the key of Platform.shmSegs).
	info   uintptr
	addr   uintptr
	shmseg uint64
	// A plain buffer has its memory, which XCreateImage did not allocate.
	mem   uintptr
	freed bool
}

var _ pixelpool.Buffer = (*xBuffer)(nil)

// Pixels is the memory of the buffer.
func (b *xBuffer) Pixels() []byte { return b.data }

// Free gives the buffer back: the server detaches the segment, this process unmaps it, and the
// image is destroyed. A segment was already removed, so it is gone once both have let go.
func (b *xBuffer) Free() {
	if b.freed {
		return
	}
	b.freed = true
	p := b.p
	im := view[xImage](b.img)
	im.data, im.obdata = 0, 0 // XDestroyImage would free what these point to
	if b.info != 0 {
		if p.display != 0 {
			p.shm.ext.detach(p.display, b.info)
		}
		p.shm.sysv.shmdt(b.addr)
		delete(p.shmSegs, b.shmseg)
		dl.Free(b.info)
	} else {
		dl.Free(b.mem)
	}
	p.x.destroyImage(b.img)
}

// pixelLayoutOK says whether an image of the window's visual has the layout that the renderer writes:
// 32 bits for a pixel in the order blue, green, red and a byte that the server ignores (24 bits of
// color, or 32 with an alpha that the renderer sets to 255), the first byte lowest (SPEC §9.6).
func pixelLayoutOK(im *xImage) bool {
	return (im.depth == 24 || im.depth == 32) && im.bitsPerPixel == 32 && im.byteOrder == lsbFirst &&
		im.redMask == 0xFF0000 && im.greenMask == 0xFF00 && im.blueMask == 0xFF
}

// pixelsOf returns the window's state for pixels, made at the first frame.
func (w *window) pixelsOf() (*pixelState, error) {
	if w.pix != nil {
		return w.pix, nil
	}
	p := w.p
	if w.visual == 0 {
		w.visual, w.depth = p.x.defaultVisual(p.display, p.screen), p.x.defaultDepth(p.display, p.screen)
	}
	st := &pixelState{shmOK: p.shmOf().on}
	st.gc = p.x.createGC(p.display, w.xid, 0, 0)
	if st.gc == 0 {
		return nil, errors.New(tagX11PixelsFailed, "XCreateGC failed", "op", "XCreateGC")
	}
	st.pool = pixelpool.New(func(width, height int, release func()) (pixelpool.Buffer, error) {
		b, err := w.newPixelBuffer(st, width, height, release)
		if err != nil {
			return nil, err // not b: a nil *xBuffer in the interface would not be nil
		}
		return b, nil
	})
	w.pix = st
	return st, nil
}

// newPixelBuffer makes a buffer of the size: a shared segment when the server takes one, else
// memory of the process. It fails with X11PixelsFailed when the memory or the image cannot be made
// and with X11PixelFormatUnsupported when the visual of the window is not the layout that the
// renderer writes.
func (w *window) newPixelBuffer(st *pixelState, width, height int, release func()) (*xBuffer, error) {
	if width <= 0 || height <= 0 || 4*width*height > maxPixelBytes {
		return nil, errors.New(tagX11PixelsFailed, "the buffer needs a size that fits in memory", "op", "size", "width", width, "height", height)
	}
	var b *xBuffer
	if st.shmOK {
		var attached bool
		var err error
		if b, attached, err = w.newSharedBuffer(width, height, release); err != nil {
			return nil, err
		}
		if !attached {
			st.shmOK = false
			slog.Info("the server did not attach the shared memory: X11 pixels go in requests", "path", "XPutImage", "reason", "XShmAttach failed")
		}
	}
	if b == nil {
		var err error
		if b, err = w.newPlainBuffer(width, height); err != nil {
			return nil, err
		}
	}
	if im := view[xImage](b.img); !pixelLayoutOK(im) {
		depth, bits, order, red, green, blue := im.depth, im.bitsPerPixel, im.byteOrder, im.redMask, im.greenMask, im.blueMask
		b.Free()
		return nil, errors.New(tagX11PixelFormatUnsupported, "the visual of the window does not hold a pixel as 32 bits in the order blue, green, red",
			"depth", depth, "bits_per_pixel", bits, "byte_order", order, "red_mask", red, "green_mask", green, "blue_mask", blue)
	}
	return b, nil
}

// newSharedBuffer makes a buffer over a segment of System V shared memory that the server attaches
// (XShmAttach). attached is false, with no error, when the server refused the segment (a request
// error: it is on another machine, or may not map it): the caller then sends the pixels in requests.
// release is called when the server's completion event says that it has read the buffer.
func (w *window) newSharedBuffer(width, height int, release func()) (b *xBuffer, attached bool, err error) {
	p, s := w.p, w.p.shm
	info := dl.Calloc(1, 32)
	img := s.ext.createImage(p.display, w.visual, uint32(w.depth), zPixmap, 0, info, uint32(width), uint32(height))
	if img == 0 {
		dl.Free(info)
		return nil, false, errors.New(tagX11PixelsFailed, "XShmCreateImage failed", "op", "XShmCreateImage", "width", width, "height", height)
	}
	im := view[xImage](img)
	stride := int(im.bytesPerLine)
	size := stride * height
	// drop gives the image and the info back, for a segment that was not made or not attached.
	drop := func() {
		im.data, im.obdata = 0, 0
		p.x.destroyImage(img)
		dl.Free(info)
	}
	if size <= 0 || size > maxPixelBytes {
		drop()
		return nil, false, errors.New(tagX11PixelsFailed, "the buffer needs a size that fits in memory", "op", "size", "width", width, "height", height)
	}
	id := s.sysv.shmget(0, uintptr(size), ipcCreat|0o600)
	if id < 0 {
		drop()
		return nil, false, errors.New(tagX11PixelsFailed, "shmget failed", "op", "shmget", "size", size)
	}
	addr := s.sysv.shmat(id, 0, 0)
	if addr == ^uintptr(0) {
		s.sysv.shmctl(id, ipcRmid, 0)
		drop()
		return nil, false, errors.New(tagX11PixelsFailed, "shmat failed", "op", "shmat", "size", size)
	}
	seg := view[xShmSegmentInfo](info)
	seg.shmid, seg.shmaddr, seg.readOnly = id, addr, 0
	im.data = addr
	before := p.xErrors
	s.ext.attach(p.display, info)
	p.x.sync(p.display, 0) // the server has answered, or it has reported an error for the request
	// The segment goes away when the server and this process have let go of it, a crash included.
	s.sysv.shmctl(id, ipcRmid, 0)
	if p.xErrors != before {
		s.sysv.shmdt(addr)
		drop()
		return nil, false, nil
	}
	if p.shmSegs == nil {
		p.shmSegs = map[uint64]func(){}
	}
	p.shmSegs[seg.shmseg] = release
	return &xBuffer{p: p, img: img, data: dl.Bytes(addr, size), stride: stride, info: info, addr: addr, shmseg: seg.shmseg}, true, nil
}

// newPlainBuffer makes a buffer in memory of the process, whose rectangles XPutImage copies into
// the connection.
func (w *window) newPlainBuffer(width, height int) (*xBuffer, error) {
	p := w.p
	img := p.x.createImage(p.display, w.visual, uint32(w.depth), zPixmap, 0, 0, uint32(width), uint32(height), 32, 0)
	if img == 0 {
		return nil, errors.New(tagX11PixelsFailed, "XCreateImage failed", "op", "XCreateImage", "width", width, "height", height)
	}
	im := view[xImage](img)
	stride := int(im.bytesPerLine)
	mem := dl.Calloc(1, uintptr(stride*height))
	if mem == 0 {
		p.x.destroyImage(img)
		return nil, errors.New(tagX11PixelsFailed, "no memory for the buffer", "op", "calloc", "width", width, "height", height)
	}
	im.data = mem
	return &xBuffer{p: p, img: img, data: dl.Bytes(mem, stride*height), stride: stride, mem: mem}, nil
}

// AcquirePixels returns the buffer of the next frame: one of the window's pool that the server
// does not read, of width x height pixels in the order blue, green, red, with its age. The server
// reads only the rectangles of the damage, so the picture would be right with any age; the pool
// keeps the age true all the same, so that a buffer holds a whole frame, as on Wayland.
//
// Flutter: no counterpart
func (w *window) AcquirePixels(width, height int) (gpu.PixelBuffer, error) {
	if w.closed {
		return gpu.PixelBuffer{}, errors.New(tagX11PixelWindowClosed, "the window is closed", "width", width, "height", height)
	}
	st, err := w.pixelsOf()
	if err != nil {
		return gpu.PixelBuffer{}, err
	}
	slot, age, err := st.pool.Acquire(width, height)
	if err != nil {
		return gpu.PixelBuffer{}, err
	}
	st.cur = slot
	b := slot.Buf.(*xBuffer)
	return gpu.PixelBuffer{Pix: b.data, Stride: b.stride, Width: width, Height: height, Format: gpu.PixelFormatBGRX8888, Age: age}, nil
}

// PresentPixels sends each damaged rectangle of the buffer that AcquirePixels returned last to the
// server: XShmPutImage for a shared segment, of which only the last asks for the completion event,
// since the server handles the requests in order, and XPutImage for a plain buffer, which is free
// again at once. With no buffer acquired nothing is sent: a frame that changed nothing needs no
// request, and the window's frame timer goes on by itself.
//
// Flutter: no counterpart
func (w *window) PresentPixels(damage []image.Rectangle) error {
	st := w.pix
	if st == nil {
		return nil
	}
	slot := st.cur
	st.cur = nil
	if slot == nil || w.closed || !w.mapped {
		return nil // a window that is not on the screen shows nothing; it is drawn whole when shown
	}
	p, b := w.p, slot.Buf.(*xBuffer)
	bounds := image.Rect(0, 0, slot.Width, slot.Height)
	if len(damage) == 0 {
		damage = []image.Rectangle{bounds}
	}
	rects := make([]image.Rectangle, 0, len(damage))
	for _, r := range damage {
		if r = r.Intersect(bounds); !r.Empty() {
			rects = append(rects, r)
		}
	}
	for i, r := range rects {
		x, y, rw, rh := int32(r.Min.X), int32(r.Min.Y), uint32(r.Dx()), uint32(r.Dy())
		if b.info != 0 {
			send := int32(0)
			if i == len(rects)-1 {
				send = 1
			}
			p.shm.ext.putImage(p.display, w.xid, st.gc, b.img, x, y, x, y, rw, rh, send)
		} else {
			p.x.putImage(p.display, w.xid, st.gc, b.img, x, y, x, y, rw, rh)
		}
	}
	st.pool.Present(slot)
	if b.info == 0 {
		slot.Released() // XPutImage has copied the rectangles into the connection
	}
	p.x.flush(p.display)
	return nil
}

// PixelPath says how the pixels of the window reach the server, "MIT-SHM" or "XPutImage", once the
// first buffer was asked for; before that it is empty.
//
// Flutter: no counterpart
func (w *window) PixelPath() string {
	switch {
	case w.p.shm == nil:
		return ""
	case w.p.shm.on && (w.pix == nil || w.pix.shmOK):
		return "MIT-SHM"
	}
	return "XPutImage"
}

// closePixels gives back the buffers and the graphics context of the window, before the window goes.
func (w *window) closePixels() {
	st := w.pix
	if st == nil {
		return
	}
	w.pix = nil
	st.pool.Close()
	if w.p.display != 0 {
		w.p.x.freeGC(w.p.display, st.gc)
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagX11PixelFormatUnsupported = "X11PixelFormatUnsupported"
	tagX11PixelsFailed           = "X11PixelsFailed"
	tagX11PixelWindowClosed      = "X11PixelWindowClosed"
)
