package win32

import (
	"encoding/binary"
	"unsafe"
)

// ParseRawInputHID splits the RAWINPUT structure of a HID report (what GetRawInputData gives): the header, then RAWHID with the
// size of one report, their number and the reports one after another. It answers false when the data is not HID input or does
// not hold the reports it says.
//
// Flutter: no counterpart
func ParseRawInputHID(buf []byte) (device uintptr, reports [][]byte, ok bool) {
	hs := int(unsafe.Sizeof(RawInputHeader{}))
	if len(buf) < hs+8 {
		return 0, nil, false
	}
	if binary.NativeEndian.Uint32(buf[0:]) != RIMTypeHID {
		return 0, nil, false
	}
	if hs == 24 {
		device = uintptr(binary.NativeEndian.Uint64(buf[8:]))
	} else {
		device = uintptr(binary.NativeEndian.Uint32(buf[8:]))
	}
	size := int(binary.NativeEndian.Uint32(buf[hs:]))
	count := int(binary.NativeEndian.Uint32(buf[hs+4:]))
	data := buf[hs+8:]
	if size <= 0 || count <= 0 || size*count > len(data) {
		return 0, nil, false
	}
	for i := 0; i < count; i++ {
		reports = append(reports, data[i*size:(i+1)*size])
	}
	return device, reports, true
}
