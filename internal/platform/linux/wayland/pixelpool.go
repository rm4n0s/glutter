//go:build linux && !android

package wayland

import (
	"github.com/rm4n0s/errors"
)

// maxPoolBuffers is the most buffers one window has at a time. A compositor holds the buffer on
// display and, for a moment, the one before it, so a window has two or three; more mean that
// the compositor does not release them, and this bound keeps the memory from growing then.
const maxPoolBuffers = 8

// keepPoolBuffers is how many idle buffers of the right size a window keeps: three, so that a
// frame is never drawn into the buffer on display or the one the compositor still reads.
const keepPoolBuffers = 3

// poolBuffer is the memory of one buffer of a pool: the real one is a shmBuffer.
type poolBuffer interface {
	// pixels is the memory of the buffer.
	pixels() []byte
	// free gives the buffer back; it is safe to call twice.
	free()
}

// poolSlot is a buffer of a pixelPool and what is known of it.
type poolSlot struct {
	buf           poolBuffer
	width, height int
	// busy is whether the compositor may read the buffer: from the commit that attached it to
	// wl_buffer.release.
	busy bool
	// shown is the number of the presentation that last wrote the buffer, 0 when it holds no frame.
	shown uint64
	pool  *pixelPool
}

// pixelPool is the buffers of one window that a CPU renderer draws into (SPEC §9.6): one is never
// handed out while the compositor reads it, and each remembers which frame it holds, so that the
// renderer can redraw only what changed since (buffer age). Only the UI thread uses it.
type pixelPool struct {
	// make creates a buffer of the size. release must be called when the compositor is done with it.
	make func(width, height int, release func()) (poolBuffer, error)
	// slots are the buffers, in the order they were made.
	slots []*poolSlot
	// presented counts the presentations that used a buffer.
	presented uint64
	// width and height are the size of the last acquire: a buffer of another size is freed as soon
	// as the compositor lets go of it.
	width, height int
	closed        bool
}

func newPixelPool(make func(width, height int, release func()) (poolBuffer, error)) *pixelPool {
	return &pixelPool{make: make}
}

// acquire returns a buffer of the size that the compositor does not read, and its age: 0 for
// one that holds no frame, n for the frame of n presentations ago. Of the idle buffers it takes the
// one that was written last, which has the smallest age. When every buffer is busy a new one is made,
// up to maxPoolBuffers; it fails with ShmPoolExhausted after that, and with ShmBufferFailed when the
// memory cannot be made.
func (p *pixelPool) acquire(width, height int) (*poolSlot, int, error) {
	p.width, p.height = width, height
	var best *poolSlot
	kept := p.slots[:0]
	for _, s := range p.slots {
		if !s.busy && (s.width != width || s.height != height) {
			s.buf.free()
			continue
		}
		kept = append(kept, s)
		if !s.busy && s.width == width && s.height == height && (best == nil || s.shown > best.shown) {
			best = s
		}
	}
	clear(p.slots[len(kept):])
	p.slots = kept
	if best == nil {
		if len(p.slots) >= maxPoolBuffers {
			return nil, 0, errors.New(tagShmPoolExhausted, "the compositor holds every buffer of the window",
				"buffers", len(p.slots), "width", width, "height", height)
		}
		s := &poolSlot{width: width, height: height, pool: p}
		buf, err := p.make(width, height, s.released)
		if err != nil {
			return nil, 0, err
		}
		s.buf = buf
		p.slots = append(p.slots, s)
		best = s
	}
	if best.shown == 0 {
		return best, 0, nil
	}
	return best, int(p.presented + 1 - best.shown), nil
}

// present marks the slot as shown by the next presentation and held by the compositor.
func (p *pixelPool) present(s *poolSlot) {
	p.presented++
	s.shown = p.presented
	s.busy = true
}

// released is the compositor's wl_buffer.release. An idle buffer of a size that is no longer
// asked for, or that is more than the pool keeps, is freed; the pool being closed frees all.
func (s *poolSlot) released() {
	s.busy = false
	p := s.pool
	if p.closed || s.width != p.width || s.height != p.height {
		p.drop(s)
		return
	}
	idle := 0
	for _, o := range p.slots {
		if !o.busy {
			idle++
		}
	}
	if idle > keepPoolBuffers {
		p.drop(p.oldestIdle())
	}
}

// oldestIdle is the idle buffer that was written longest ago.
func (p *pixelPool) oldestIdle() *poolSlot {
	var old *poolSlot
	for _, s := range p.slots {
		if !s.busy && (old == nil || s.shown < old.shown) {
			old = s
		}
	}
	return old
}

// drop frees a slot and takes it out of the list.
func (p *pixelPool) drop(s *poolSlot) {
	if s == nil {
		return
	}
	s.buf.free()
	for i, o := range p.slots {
		if o == s {
			p.slots = append(p.slots[:i], p.slots[i+1:]...)
			return
		}
	}
}

// close frees every buffer. A buffer that the compositor still holds is freed too: the
// compositor has its own mapping of the memory.
func (p *pixelPool) close() {
	p.closed = true
	for _, s := range p.slots {
		s.buf.free()
	}
	p.slots = nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagShmPoolExhausted = "ShmPoolExhausted"
)
