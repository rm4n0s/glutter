//go:build windows

package d3d12

import (
	"unsafe"

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
