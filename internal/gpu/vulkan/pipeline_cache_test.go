//go:build linux

package vulkan

import (
	"encoding/binary"
	"testing"
)

func TestPipelineCacheHeader(t *testing.T) {
	info := physicalInfo{VendorID: 0x10DE, DeviceID: 0x2684, PipelineCacheUUID: [uuidSize]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}}
	header := func(change func(h []byte)) []byte {
		h := make([]byte, pipelineCacheHeaderSize+8) // a header and some data
		binary.LittleEndian.PutUint32(h[0:], pipelineCacheHeaderSize)
		binary.LittleEndian.PutUint32(h[4:], pipelineCacheHeaderVersionOne)
		binary.LittleEndian.PutUint32(h[8:], info.VendorID)
		binary.LittleEndian.PutUint32(h[12:], info.DeviceID)
		copy(h[16:], info.PipelineCacheUUID[:])
		if change != nil {
			change(h)
		}
		return h
	}
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{"the device's own cache", header(nil), true},
		{"a bigger header than the one known", header(func(h []byte) { binary.LittleEndian.PutUint32(h[0:], pipelineCacheHeaderSize+8) }), true},
		{"another vendor", header(func(h []byte) { binary.LittleEndian.PutUint32(h[8:], 0x1002) }), false},
		{"another device", header(func(h []byte) { binary.LittleEndian.PutUint32(h[12:], 1) }), false},
		{"another driver build", header(func(h []byte) { h[16]++ }), false},
		{"another version of the header", header(func(h []byte) { binary.LittleEndian.PutUint32(h[4:], 2) }), false},
		{"a header size that is too small", header(func(h []byte) { binary.LittleEndian.PutUint32(h[0:], 8) }), false},
		{"cut short", header(nil)[:pipelineCacheHeaderSize-1], false},
		{"empty", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validPipelineCacheHeader(tt.data, info); got != tt.want {
				t.Errorf("validPipelineCacheHeader = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPipelineCacheKey(t *testing.T) {
	a := physicalInfo{VendorID: 1, DeviceID: 2, PipelineCacheUUID: [uuidSize]byte{1}}
	tests := []struct {
		name string
		b    physicalInfo
		same bool
	}{
		{"the same device", a, true},
		{"another device of the vendor", physicalInfo{VendorID: 1, DeviceID: 3, PipelineCacheUUID: a.PipelineCacheUUID}, false},
		{"another vendor", physicalInfo{VendorID: 2, DeviceID: 2, PipelineCacheUUID: a.PipelineCacheUUID}, false},
		{"another driver build", physicalInfo{VendorID: 1, DeviceID: 2, PipelineCacheUUID: [uuidSize]byte{2}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pipelineCacheKey(a) == pipelineCacheKey(tt.b); got != tt.same {
				t.Errorf("same key = %v, want %v", got, tt.same)
			}
		})
	}
}
