package d3d12

import (
	"encoding/binary"
	"testing"
	"unsafe"
)

// message builds the buffer that GetMessage fills: the D3D12_MESSAGE and the text after it.
func message(severity, id uint32, text string, length int) []byte {
	const size = int(unsafe.Sizeof(d3d12Message{}))
	buf := make([]byte, size+len(text)+1)
	binary.LittleEndian.PutUint32(buf[4:], severity)
	binary.LittleEndian.PutUint32(buf[8:], id)
	binary.LittleEndian.PutUint64(buf[24:], uint64(length))
	copy(buf[size:], text)
	return buf
}

func TestParseDebugMessage(t *testing.T) {
	tests := []struct {
		name    string
		buf     []byte
		want    debugMessage
		ok      bool
		isError bool
	}{
		{"an error", message(messageSeverityError, 12, "bad barrier", len("bad barrier")+1), debugMessage{messageSeverityError, 12, "bad barrier"}, true, true},
		{"corruption", message(messageSeverityCorruption, 1, "oops", 5), debugMessage{messageSeverityCorruption, 1, "oops"}, true, true},
		{"a warning is not an error", message(messageSeverityWarning, 3, "slow", 5), debugMessage{messageSeverityWarning, 3, "slow"}, true, false},
		{"an empty text", message(messageSeverityError, 4, "", 1), debugMessage{messageSeverityError, 4, ""}, true, true},
		{"a buffer shorter than the structure", make([]byte, 10), debugMessage{}, false, false},
		{"a length past the end of the buffer", message(messageSeverityError, 4, "x", 99), debugMessage{}, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseDebugMessage(tt.buf)
			if ok != tt.ok || got != tt.want {
				t.Errorf("parseDebugMessage = %+v, %v; want %+v, %v", got, ok, tt.want, tt.ok)
			}
			if ok && got.isError() != tt.isError {
				t.Errorf("isError = %v, want %v", got.isError(), tt.isError)
			}
		})
	}
}
