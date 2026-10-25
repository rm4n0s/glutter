package d3d12

import (
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// The memory of the device (SPEC §9.8 "Memory"): committed resources only. This file is the plain
// Go part, which is tested without Windows: the staging ring, the layout of a copy between a texture
// and a buffer, and the slots of the CPU descriptor heaps.

// stagingSize is the size of the staging ring, in an upload heap: half for each frame slot.
const stagingSize = 16 << 20

// alignUp rounds n up to a multiple of alignment.
func alignUp(n, alignment int) int {
	if alignment <= 1 {
		return n
	}
	return (n + alignment - 1) / alignment * alignment
}

// stagingRing hands out ranges of the staging buffer. Each frame slot has its own half, which is
// free again when the slot is reused (its frame is done on the GPU), so the head of a slot runs from
// the start of the half to its end and then wraps back to the start with the next frame of the slot.
// A request that does not fit in a half is not served: the device makes an upload buffer of its own,
// which the frame releases.
type stagingRing struct {
	size  int
	heads [explicit.FramesInFlight]int
}

func newStagingRing(size int) *stagingRing { return &stagingRing{size: size} }

// half is the size of the part of one slot.
func (r *stagingRing) half() int { return r.size / explicit.FramesInFlight }

// reserve takes n bytes in the half of the slot, at an offset that is a multiple of alignment (the
// offset in the whole buffer: 512 for the placement of a texture copy, SPEC §9.8). ok is false when the
// half has no room left, and the head does not move.
func (r *stagingRing) reserve(slot, n, alignment int) (offset int, ok bool) {
	base := slot * r.half()
	head := alignUp(base+r.heads[slot], alignment) - base
	if head+n > r.half() {
		return 0, false
	}
	r.heads[slot] = head + n
	return base + head, true
}

// reset makes the half of the slot free again: the frame that used it is done.
func (r *stagingRing) reset(slot int) { r.heads[slot] = 0 }

// used is how many bytes of the slot's half are taken, the alignment included.
func (r *stagingRing) used(slot int) int { return r.heads[slot] }

// copyLayout is how the rows of a rectangle lie in a buffer, as Direct3D 12 wants them for a copy
// between a buffer and a texture: every row starts at a multiple of RowPitchAlignment.
type copyLayout struct {
	// RowBytes is the bytes of pixels in a row; RowPitch is the distance between two rows, RowBytes
	// rounded up to the alignment.
	RowBytes, RowPitch int
	// Rows is the number of rows.
	Rows int
	// Size is the bytes the copy takes in the buffer: all rows but the last at the pitch, and the last
	// row as it is, which is what GetCopyableFootprints reports as the total.
	Size int
}

// layoutOf computes the layout of a rectangle of width x height pixels of a format. The pitch is
// Limits.AlignedRowPitch, which is the 256 bytes of D3D12_TEXTURE_DATA_PITCH_ALIGNMENT.
func layoutOf(limits explicit.Limits, width, height, bytesPerPixel int) copyLayout {
	l := copyLayout{RowBytes: width * bytesPerPixel, Rows: height}
	l.RowPitch = limits.AlignedRowPitch(width, bytesPerPixel)
	if height > 0 {
		l.Size = l.RowPitch*(height-1) + l.RowBytes
	}
	return l
}

// slotPool hands out the numbers of the descriptors of a CPU heap kind (render target views, shader
// resource views, samplers). The heaps have perHeap descriptors each; the number n is in heap
// n / perHeap at slot n % perHeap, and the device adds a heap when need says so. A freed number is
// reused before a new one is made.
type slotPool struct {
	perHeap int
	next    int
	free    []int
	live    int
}

func newSlotPool(perHeap int) *slotPool { return &slotPool{perHeap: perHeap} }

// alloc returns a number, and whether a heap has to be added for it (the number is the first of a
// heap that does not exist yet).
func (p *slotPool) alloc() (n int, newHeap bool) {
	p.live++
	if k := len(p.free); k > 0 {
		n = p.free[k-1]
		p.free = p.free[:k-1]
		return n, false
	}
	n = p.next
	p.next++
	return n, n%p.perHeap == 0
}

// release gives a number back.
func (p *slotPool) release(n int) {
	p.live--
	p.free = append(p.free, n)
}

// split returns the heap and the slot of a number.
func (p *slotPool) split(n int) (heap, slot int) { return n / p.perHeap, n % p.perHeap }

// heaps is how many heaps the numbers made so far need.
func (p *slotPool) heaps() int { return (p.next + p.perHeap - 1) / p.perHeap }
