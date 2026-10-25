//go:build windows

package d3d12

import (
	"unsafe"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// heapProps are the properties of a committed resource's heap (SPEC §9.8 "Memory").
func heapProps(heapType uint32) d3d12HeapProperties {
	return d3d12HeapProperties{Type: heapType, CreationNodeMask: 1, VisibleNodeMask: 1}
}

// bufferDesc describes a buffer resource of size bytes.
func bufferDesc(size int) d3d12ResourceDesc {
	return d3d12ResourceDesc{
		Dimension: resourceDimensionBuffer, Width: uint64(size), Height: 1, DepthOrArraySize: 1, MipLevels: 1,
		Format: dxgiFormatUnknown, SampleDesc: dxgiSampleDesc{Count: 1}, Layout: textureLayoutRowMajor,
	}
}

// createBuffer makes a committed buffer in a heap of the type, in the state the heap needs: an upload
// heap's resources are GENERIC_READ and a readback heap's COPY_DEST. It fails with GPUOutOfMemory
// and the other tags of check.
func (d *Device) createBuffer(heapType uint32, size int) (win32.Object, error) {
	hp := heapProps(heapType)
	desc := bufferDesc(size)
	state := resourceStateGenericRead
	if heapType == heapTypeReadback {
		state = resourceStateCopyDest
	}
	var out uintptr
	if err := d.hr(d.dev, "ID3D12Device::CreateCommittedResource", deviceCreateCommittedResource,
		uintptr(unsafe.Pointer(&hp)), 0, uintptr(unsafe.Pointer(&desc)), uintptr(state), 0,
		uintptr(unsafe.Pointer(&iidResource)), uintptr(unsafe.Pointer(&out))); err != nil {
		return win32.Object{}, err
	}
	d.counts.bytes += int64(size)
	return d.own(win32.ObjectFromPtr(out)), nil
}

// releaseBuffer releases a buffer that createBuffer made, of the size it was made with.
func (d *Device) releaseBuffer(res win32.Object, size int) {
	d.counts.bytes -= int64(size)
	d.disown(res)
}

// mapBuffer maps the whole buffer for good and returns the address. The range of the CPU's reads is
// empty for an upload buffer, which the CPU only writes.
func (d *Device) mapBuffer(res win32.Object, read bool, size int) (uintptr, error) {
	var rng d3d12Range
	if read {
		rng.End = uintptr(size)
	}
	var ptr uintptr
	if err := d.hr(res, "ID3D12Resource::Map", resourceMap, 0, uintptr(unsafe.Pointer(&rng)), uintptr(unsafe.Pointer(&ptr))); err != nil {
		return 0, err
	}
	return ptr, nil
}

// unmapBuffer unmaps the buffer; written is the range the CPU wrote (empty for a readback buffer,
// which it only read).
func unmapBuffer(res win32.Object, written int) {
	rng := d3d12Range{End: uintptr(written)}
	res.Call(resourceUnmap, 0, uintptr(unsafe.Pointer(&rng)))
}

// staging is the buffer that texture uploads are filled through: 16 MiB in an upload heap, half for
// each frame slot, made when the first upload needs it (SPEC §9.8 "Memory").
type staging struct {
	res  win32.Object
	ptr  uintptr
	ring *stagingRing
}

func (s *staging) destroy(d *Device) {
	if !s.res.IsNil() {
		d.releaseBuffer(s.res, stagingSize)
		s.res = win32.Object{}
	}
}

// stage reserves n bytes for a texture upload in the staging memory of the slot of the frame that
// will run the copy (commandSlot), at an offset that is a multiple of the 512 bytes of
// D3D12_TEXTURE_DATA_PLACEMENT_ALIGNMENT. It returns the buffer, the offset in it and the address to
// write the data at. Data that does not fit in the slot's half gets an upload buffer of its own,
// released when the GPU is done with the frame that copies from it.
func (d *Device) stage(n int) (win32.Object, int, uintptr, error) {
	slot := d.commandSlot()
	if err := d.prepare(slot); err != nil { // a slot that is being recorded is ready already
		return win32.Object{}, 0, 0, err
	}
	if n <= d.staging.ring.half() {
		if err := d.ensureStaging(); err != nil {
			return win32.Object{}, 0, 0, err
		}
		if off, ok := d.staging.ring.reserve(slot, n, textureDataPlacementAlignment); ok {
			return d.staging.res, off, d.staging.ptr + uintptr(off), nil
		}
	}
	size := max(n, 1)
	res, err := d.createBuffer(heapTypeUpload, size)
	if err != nil {
		return win32.Object{}, 0, 0, err
	}
	ptr, err := d.mapBuffer(res, false, size)
	if err != nil {
		d.releaseBuffer(res, size)
		return win32.Object{}, 0, 0, err
	}
	d.counts.oneOff++
	d.retireWithNextFrame(func() {
		d.counts.oneOff--
		d.releaseBuffer(res, size)
	})
	return res, 0, ptr, nil
}

// ensureStaging makes the staging buffer when the first upload needs it.
func (d *Device) ensureStaging() error {
	if !d.staging.res.IsNil() {
		return nil
	}
	res, err := d.createBuffer(heapTypeUpload, stagingSize)
	if err != nil {
		return err
	}
	ptr, err := d.mapBuffer(res, false, stagingSize)
	if err != nil {
		d.releaseBuffer(res, stagingSize)
		return err
	}
	d.staging.res, d.staging.ptr = res, ptr
	return nil
}
