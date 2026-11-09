package darwin

import (
	"github.com/rm4n0s/errors"
)

// Surface is memory that a window shows and the window server reads: an IOSurface (SPEC §29.5). The pool
// works over this interface, so that its rules (the age of a buffer, the buffers in use, a new size) are
// tested on every host with a fake.
//
// Flutter: no counterpart
type Surface interface {
	// Size is the size of the surface in pixels.
	Size() (width, height int)
	// Stride is the length of a row in bytes.
	Stride() int
	// Lock makes the memory writable by the CPU and returns it (stride x height bytes). It stays valid
	// until Unlock.
	Lock() ([]byte, error)
	// Unlock ends the CPU access.
	Unlock()
	// InUse says whether somebody else (the window server) still reads the surface.
	InUse() bool
	// Release gives the surface up. The window server keeps what it shows until it is done with it.
	Release()
}

// SurfacePool is the set of surfaces that one window draws into (SPEC §29.5, §9.6): at most Max, handed
// out so that a surface that the window shows, or that the window server still reads, is never written
// to. Each buffer has an age, the number of presentations since it was shown, so that the renderer
// redraws only what the last frames changed. It belongs to the UI thread.
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterSurfaceManager.mm FlutterSurfaceManager
// (its back and front buffers; here a pool of up to eight)
type SurfacePool struct {
	max   int
	make  func(width, height int) (Surface, error)
	slots []*poolSlot
	// current is the slot that the window shows now, nil before the first presentation.
	current *poolSlot
	// acquired is the slot that Acquire gave out and that is not presented yet.
	acquired *poolSlot
	// seq counts the presentations.
	seq uint64
	// w and h are the size of the surfaces in the pool.
	w, h int
}

type poolSlot struct {
	surface Surface
	// shown is the number of the presentation that showed the slot, 0 when it never did (or its content
	// is not a frame).
	shown  uint64
	locked bool
}

// Buffer is a surface of the pool that is locked for writing.
//
// Flutter: no counterpart
type Buffer struct {
	// Pix is the pixels: rows of Stride bytes, the top row first.
	Pix []byte
	// Stride is the length of a row in bytes.
	Stride int
	// Width and Height are the size in pixels.
	Width, Height int
	// Age is as in gpu.PixelBuffer: 0 for undefined content, n for the frame of n presentations ago.
	Age int
	// Surface is the surface behind the buffer, for the layer that shows it.
	Surface Surface
}

// NewSurfacePool makes a pool of at most max surfaces, made by the function (an IOSurface in the app).
//
// Flutter: no counterpart
func NewSurfacePool(max int, makeSurface func(width, height int) (Surface, error)) *SurfacePool {
	if max < 2 {
		max = 2
	}
	return &SurfacePool{max: max, make: makeSurface}
}

// Len is the number of surfaces that the pool holds.
func (p *SurfacePool) Len() int { return len(p.slots) }

// Acquire gives a locked buffer of the size that nothing reads now. The buffer that was acquired and
// not presented is given up first (its content is not a frame any more). A new size releases every
// surface of the old size. Among the free surfaces it picks the one shown last, which has the
// smallest age, so that the least has to be redrawn; with none free it makes a new surface up to Max
// and then fails with IOSurfacePoolExhausted.
//
// Flutter: no counterpart
func (p *SurfacePool) Acquire(width, height int) (Buffer, error) {
	if p.acquired != nil {
		p.giveUp(p.acquired)
	}
	if width != p.w || height != p.h {
		p.resize(width, height)
	}
	slot := p.pick()
	if slot == nil {
		if len(p.slots) >= p.max {
			return Buffer{}, errors.New(TagIOSurfacePoolExhausted, "every surface of the pool is shown or read by the window server",
				"surfaces", len(p.slots))
		}
		s, err := p.make(width, height)
		if err != nil {
			return Buffer{}, err
		}
		slot = &poolSlot{surface: s}
		p.slots = append(p.slots, slot)
	}
	pix, err := slot.surface.Lock()
	if err != nil {
		return Buffer{}, err
	}
	slot.locked = true
	p.acquired = slot
	age := 0
	if slot.shown != 0 {
		age = int(p.seq-slot.shown) + 1
	}
	return Buffer{Pix: pix, Stride: slot.surface.Stride(), Width: width, Height: height, Age: age, Surface: slot.surface}, nil
}

// pick is the free slot shown last, or the one that never was shown; nil when none is free. A slot is
// free when it is not the one shown now, is not acquired and the window server does not read it.
func (p *SurfacePool) pick() *poolSlot {
	var best *poolSlot
	for _, s := range p.slots {
		if s == p.current || s == p.acquired || s.surface.InUse() {
			continue
		}
		if best == nil || s.shown > best.shown {
			best = s
		}
	}
	return best
}

// giveUp unlocks a slot that was acquired and will not be presented: what it holds is not a frame.
func (p *SurfacePool) giveUp(s *poolSlot) {
	if s.locked {
		s.surface.Unlock()
		s.locked = false
	}
	s.shown = 0
	p.acquired = nil
}

// resize drops every surface: a window of a new size shows no old frame.
func (p *SurfacePool) resize(width, height int) {
	for _, s := range p.slots {
		if s.locked {
			s.surface.Unlock()
		}
		s.surface.Release()
	}
	p.slots, p.current, p.acquired = nil, nil, nil
	p.w, p.h = width, height
}

// Present unlocks the buffer that Acquire gave out and makes it the shown one. It returns its surface,
// which the caller puts into the layer, and ok false when nothing was acquired (the window shows what
// it showed).
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterSurfaceManager.mm FlutterSurfaceManager present
func (p *SurfacePool) Present() (shown Surface, ok bool) {
	s := p.acquired
	if s == nil {
		return nil, false
	}
	s.surface.Unlock()
	s.locked = false
	p.seq++
	s.shown = p.seq
	p.current = s
	p.acquired = nil
	return s.surface, true
}

// Current is the surface that the window shows now, or nil.
func (p *SurfacePool) Current() Surface {
	if p.current == nil {
		return nil
	}
	return p.current.surface
}

// Close releases every surface.
//
// Flutter: no counterpart
func (p *SurfacePool) Close() {
	p.resize(0, 0)
}

// TagIOSurfacePoolExhausted is the tag of the error of a pool whose surfaces are all in use.
const TagIOSurfacePoolExhausted = "IOSurfacePoolExhausted"
