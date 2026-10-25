//go:build windows

package d3d12

import (
	"runtime"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// rootSig is a root signature object and the plan it was made from. The pipelines whose bindings
// have the same shape share one (SPEC §9.8).
type rootSig struct {
	key  string
	obj  win32.Object
	plan *rootPlan
}

// rootSignature returns the root signature of the plan, made and kept by the shape of its bindings: the
// description is serialized with D3D12SerializeRootSignature at version 1.0 and given to
// CreateRootSignature. It fails with GPUOutOfMemory, GPUDeviceLost or D3D12CallFailed; for a failed
// serialization the compiler's messages are in the metadata "messages".
func (d *Device) rootSignature(plan *rootPlan) (*rootSig, error) {
	key := plan.Key()
	if rs, ok := d.roots[key]; ok {
		return rs, nil
	}
	desc := plan.description()
	sd := d3d12RootSignatureDesc{NumParameters: uint32(len(desc.Params)), Flags: desc.Flags}
	for i := range desc.Params {
		if len(desc.Ranges[i]) > 0 {
			desc.Params[i].Union[1] = uint64(uintptr(unsafe.Pointer(&desc.Ranges[i][0])))
		}
	}
	if len(desc.Params) > 0 {
		sd.Parameters = uintptr(unsafe.Pointer(&desc.Params[0]))
	}
	var blob, errs uintptr
	r, _ := d.libs.serializeRootSignature.Call(uintptr(unsafe.Pointer(&sd)), uintptr(rootSignatureVersion1),
		uintptr(unsafe.Pointer(&blob)), uintptr(unsafe.Pointer(&errs)))
	runtime.KeepAlive(desc) // the pointers in sd are addresses of its slices, which the GC does not move
	errBlob, blobObj := win32.ObjectFromPtr(errs), win32.ObjectFromPtr(blob)
	if err := check("D3D12SerializeRootSignature", win32.FromResult(r)); err != nil {
		if !errBlob.IsNil() {
			if appErr, ok := errors.FromError(err); ok {
				_ = appErr.SetMetadata("messages", blobText(errBlob)) // adds to the error in place
			}
			errBlob.Release()
		}
		if !blobObj.IsNil() {
			blobObj.Release()
		}
		return nil, err
	}
	if !errBlob.IsNil() {
		errBlob.Release()
	}
	defer blobObj.Release()
	var out uintptr
	if err := d.hr(d.dev, "ID3D12Device::CreateRootSignature", deviceCreateRootSignature, 0,
		blobObj.Call(blobGetBufferPointer), blobObj.Call(blobGetBufferSize),
		uintptr(unsafe.Pointer(&iidRootSignature)), uintptr(unsafe.Pointer(&out))); err != nil {
		return nil, err
	}
	rs := &rootSig{key: key, obj: d.own(win32.ObjectFromPtr(out)), plan: plan}
	d.roots[key] = rs
	return rs, nil
}
