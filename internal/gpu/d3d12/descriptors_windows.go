//go:build windows

package d3d12

import (
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// cpuDescriptorsPerHeap is the number of descriptors in a CPU-only heap (SPEC §9.8 "Descriptors"); a
// heap is added when the ones there are full. 1024 is inside the 2048 a sampler heap may have.
const cpuDescriptorsPerHeap = 1024

// cpuHeaps are the CPU-only descriptor heaps of one kind (render target views, shader resource
// views, samplers): the descriptors of textures and samplers, which a bind group copies into the
// shader-visible heaps of a frame slot (step 28.9). A descriptor is a number of the pool; its
// handle is the start of its heap plus slot times the increment.
type cpuHeaps struct {
	kind  uint32
	pool  *slotPool
	heaps []cpuHeap
	incr  uintptr
}

// cpuHeap is one descriptor heap and the CPU handle of its start.
type cpuHeap struct {
	obj   win32.Object
	start uintptr
}

func newCPUHeaps(kind uint32) *cpuHeaps {
	return &cpuHeaps{kind: kind, pool: newSlotPool(cpuDescriptorsPerHeap)}
}

// alloc takes a descriptor and returns its number and its CPU handle, making a heap when the last
// one is full.
func (h *cpuHeaps) alloc(d *Device) (n int, handle uintptr, err error) {
	n, newHeap := h.pool.alloc()
	if newHeap {
		if err := h.addHeap(d); err != nil {
			h.pool.release(n)
			return 0, 0, err
		}
	}
	return n, h.handle(n), nil
}

// handle is the CPU descriptor handle of a number.
func (h *cpuHeaps) handle(n int) uintptr {
	heap, slot := h.pool.split(n)
	return h.heaps[heap].start + uintptr(slot)*h.incr
}

// free gives a descriptor back.
func (h *cpuHeaps) free(n int) { h.pool.release(n) }

// addHeap makes a CPU-only heap (no shader-visible flag) of cpuDescriptorsPerHeap descriptors.
func (h *cpuHeaps) addHeap(d *Device) error {
	desc := d3d12DescriptorHeapDesc{Type: h.kind, NumDescriptors: cpuDescriptorsPerHeap}
	var out uintptr
	if err := d.hr(d.dev, "ID3D12Device::CreateDescriptorHeap", deviceCreateDescriptorHeap,
		uintptr(unsafe.Pointer(&desc)), uintptr(unsafe.Pointer(&iidDescriptorHeap)), uintptr(unsafe.Pointer(&out))); err != nil {
		return err
	}
	obj := d.own(win32.ObjectFromPtr(out))
	var start uintptr // D3D12_CPU_DESCRIPTOR_HANDLE: returned through a hidden pointer (SPEC §25.2)
	obj.Call(heapGetCPUHandleStart, uintptr(unsafe.Pointer(&start)))
	if h.incr == 0 {
		h.incr = d.dev.Call(deviceGetDescriptorIncrementSize, uintptr(h.kind))
	}
	h.heaps = append(h.heaps, cpuHeap{obj: obj, start: start})
	return nil
}

// destroy releases the heaps.
func (h *cpuHeaps) destroy(d *Device) {
	for _, hp := range h.heaps {
		d.disown(hp.obj)
	}
	h.heaps = nil
}

// Defaults for the size of a block of a shader-visible heap, in descriptors: a bind group takes a few
// descriptors, so a block holds thousands of tables. A heap of 65536 is 2 MB and inside the million a
// resource heap may have at resource binding tier 1; 2048 is the most a sampler heap may have.
const (
	viewBlockSize    = 65536
	samplerBlockSize = 2048
	// maxHeapBlocks is the most blocks of one kind that a frame slot makes. The memory of a heap does
	// not depend on its size, so a frame that needs more than this fails (GPUOutOfMemory).
	maxHeapBlocks = 8
)

// shaderHeap is one block of a shader-visible heap: the heap and the handles of its start.
type shaderHeap struct {
	obj win32.Object
	cpu uintptr // the CPU handle of the first descriptor, which bind groups write
	gpu uint64  // the GPU handle of the first descriptor, which the root arguments name
}

// shaderHeaps are the shader-visible heaps of one kind (resources or samplers) of a frame slot, in
// blocks (SPEC §9.8 "Descriptors"): the plain-Go blockAlloc says where a range goes and when a block
// is new, and a heap is made for each block. All blocks are reused when the slot is.
type shaderHeaps struct {
	kind  uint32
	size  int
	incr  uintptr
	alloc blockAlloc
	heaps []shaderHeap
}

func newShaderHeaps(kind uint32, size int, incr uintptr) *shaderHeaps {
	return &shaderHeaps{kind: kind, size: size, incr: incr, alloc: newBlockAlloc(size, maxHeapBlocks)}
}

// take reserves n descriptors and returns their block and first descriptor, making the heap of a new
// block. It fails with GPUOutOfMemory when the heap cannot be made, or when n does not fit in one
// block.
func (h *shaderHeaps) take(d *Device, n int) (block, offset int, err error) {
	block, offset, grew, ok := h.alloc.alloc(n)
	if !ok {
		return 0, 0, errors.New(tagGPUOutOfMemory, "the bind groups of the frame need more descriptors than the shader-visible heap of its slot may hold",
			"descriptors", n, "block", h.size, "blocks", len(h.heaps), "max_blocks", maxHeapBlocks)
	}
	if grew {
		desc := d3d12DescriptorHeapDesc{Type: h.kind, NumDescriptors: uint32(h.size), Flags: descriptorHeapShaderVisible}
		var out uintptr
		if err := d.hr(d.dev, "ID3D12Device::CreateDescriptorHeap", deviceCreateDescriptorHeap,
			uintptr(unsafe.Pointer(&desc)), uintptr(unsafe.Pointer(&iidDescriptorHeap)), uintptr(unsafe.Pointer(&out))); err != nil {
			h.alloc.rollback()
			return 0, 0, err
		}
		obj := d.own(win32.ObjectFromPtr(out))
		var cpu uintptr // the handles are returned through a hidden pointer (SPEC §25.2)
		var gpu uint64
		obj.Call(heapGetCPUHandleStart, uintptr(unsafe.Pointer(&cpu)))
		obj.Call(heapGetGPUHandleStart, uintptr(unsafe.Pointer(&gpu)))
		h.heaps = append(h.heaps, shaderHeap{obj: obj, cpu: cpu, gpu: gpu})
	}
	return block, offset, nil
}

// cpuHandle is the CPU handle of a descriptor of a block.
func (h *shaderHeaps) cpuHandle(block, offset int) uintptr {
	return h.heaps[block].cpu + uintptr(offset)*h.incr
}

// gpuHandle is the GPU handle of a descriptor of a block.
func (h *shaderHeaps) gpuHandle(block, offset int) uint64 {
	return h.heaps[block].gpu + uint64(offset)*uint64(h.incr)
}

// latest is the block that the next range goes to; -1 before the first range.
func (h *shaderHeaps) latest() int {
	if len(h.heaps) == 0 {
		return -1
	}
	return h.alloc.cur
}

// reset makes the blocks empty again: the frame that used them is done.
func (h *shaderHeaps) reset() { h.alloc.reset() }

// destroy releases the heaps.
func (h *shaderHeaps) destroy(d *Device) {
	for _, hp := range h.heaps {
		d.disown(hp.obj)
	}
	h.heaps = nil
}
