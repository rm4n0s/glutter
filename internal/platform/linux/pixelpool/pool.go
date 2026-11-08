//go:build linux && !android

// Package pixelpool is the buffers of one window that a CPU renderer draws into: the part of the
// Wayland and X11 backends that hands a buffer out only when the display server no longer reads
// it, and says how old its content is (SPEC §9.6; steps 32.1 and 32.2).
package pixelpool

import (
	"github.com/rm4n0s/errors"
)

// maxBuffers is the most buffers one window has at a time. A server holds the buffer on display
// and, for a moment, the one before it, so a window has two or three; more mean that the server
// does not release them, and this bound keeps the memory from growing then.
const maxBuffers = 8

// keepBuffers is how many idle buffers of the right size a window keeps: three, so that a frame
// is never drawn into the buffer on display or the one the server still reads.
const keepBuffers = 3

// Buffer is the memory of one buffer of a pool: a wl_shm buffer on Wayland, a shared-memory image
// on X11.
type Buffer interface {
	// Pixels is the memory of the buffer.
	Pixels() []byte
	// Free gives the buffer back; it is safe to call twice.
	Free()
}

// Slot is a buffer of a Pool and what is known of it.
type Slot struct {
	// Buf is the buffer.
	Buf Buffer
	// Width and Height are its size in pixels.
	Width, Height int
	// busy is whether the server may read the buffer: from the presentation that used it to the
	// server's release.
	busy bool
	// shown is the number of the presentation that last wrote the buffer, 0 when it holds no frame.
	shown uint64
	pool  *Pool
}

// Pool is the buffers of one window that a CPU renderer draws into (SPEC §9.6): one is never
// handed out while the server reads it, and each remembers which frame it holds, so that the
// renderer can redraw only what changed since (buffer age). Only the UI thread uses it.
type Pool struct {
	// make creates a buffer of the size. release must be called when the server is done with it.
	make func(width, height int, release func()) (Buffer, error)
	// slots are the buffers, in the order they were made.
	slots []*Slot
	// presented counts the presentations that used a buffer.
	presented uint64
	// width and height are the size of the last acquire: a buffer of another size is freed as soon
	// as the server lets go of it.
	width, height int
	closed        bool
}

// New makes a pool whose buffers come from make.
func New(make func(width, height int, release func()) (Buffer, error)) *Pool {
	return &Pool{make: make}
}

// Acquire returns a buffer of the size that the server does not read, and its age: 0 for one that
// holds no frame, n for the frame of n presentations ago. Of the idle buffers it takes the one that
// was written last, which has the smallest age. When every buffer is busy a new one is made, up to
// maxBuffers; it fails with ShmPoolExhausted after that, and with the error of make when the memory
// cannot be made.
func (p *Pool) Acquire(width, height int) (*Slot, int, error) {
	p.width, p.height = width, height
	var best *Slot
	kept := p.slots[:0]
	for _, s := range p.slots {
		if !s.busy && (s.Width != width || s.Height != height) {
			s.Buf.Free()
			continue
		}
		kept = append(kept, s)
		if !s.busy && s.Width == width && s.Height == height && (best == nil || s.shown > best.shown) {
			best = s
		}
	}
	clear(p.slots[len(kept):])
	p.slots = kept
	if best == nil {
		if len(p.slots) >= maxBuffers {
			return nil, 0, errors.New(tagShmPoolExhausted, "the server holds every buffer of the window",
				"buffers", len(p.slots), "width", width, "height", height)
		}
		s := &Slot{Width: width, Height: height, pool: p}
		buf, err := p.make(width, height, s.Released)
		if err != nil {
			return nil, 0, err
		}
		s.Buf = buf
		p.slots = append(p.slots, s)
		best = s
	}
	if best.shown == 0 {
		return best, 0, nil
	}
	return best, int(p.presented + 1 - best.shown), nil
}

// Present marks the slot as shown by the next presentation and held by the server.
func (p *Pool) Present(s *Slot) {
	p.presented++
	s.shown = p.presented
	s.busy = true
}

// Released is the server's word that it no longer reads the buffer (wl_buffer.release, the
// completion event of MIT-SHM). An idle buffer of a size that is no longer asked for, or that is
// more than the pool keeps, is freed; the pool being closed frees all.
func (s *Slot) Released() {
	s.busy = false
	p := s.pool
	if p.closed || s.Width != p.width || s.Height != p.height {
		p.drop(s)
		return
	}
	idle := 0
	for _, o := range p.slots {
		if !o.busy {
			idle++
		}
	}
	if idle > keepBuffers {
		p.drop(p.oldestIdle())
	}
}

// oldestIdle is the idle buffer that was written longest ago.
func (p *Pool) oldestIdle() *Slot {
	var old *Slot
	for _, s := range p.slots {
		if !s.busy && (old == nil || s.shown < old.shown) {
			old = s
		}
	}
	return old
}

// drop frees a slot and takes it out of the list.
func (p *Pool) drop(s *Slot) {
	if s == nil {
		return
	}
	s.Buf.Free()
	for i, o := range p.slots {
		if o == s {
			p.slots = append(p.slots[:i], p.slots[i+1:]...)
			return
		}
	}
}

// Close frees every buffer. A buffer that the server still holds is freed too: the server has its
// own mapping of the memory.
func (p *Pool) Close() {
	p.closed = true
	for _, s := range p.slots {
		s.Buf.Free()
	}
	p.slots = nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagShmPoolExhausted = "ShmPoolExhausted"
)
