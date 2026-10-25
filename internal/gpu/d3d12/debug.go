package d3d12

import (
	"encoding/binary"
	"strings"
	"unsafe"
)

// debugMessage is one message of the debug layer's info queue, read out of the buffer that
// ID3D12InfoQueue::GetMessage filled.
type debugMessage struct {
	Severity uint32
	ID       uint32
	Text     string
}

// isError says whether the message is an error of the application (corruption or error), which the
// device logs; warnings and information are left in the queue.
func (m debugMessage) isError() bool {
	return m.Severity == messageSeverityCorruption || m.Severity == messageSeverityError
}

// parseDebugMessage reads a D3D12_MESSAGE and its text out of the buffer that GetMessage filled: the
// structure comes first and the text follows it, NUL-terminated, in the same buffer (the pointer of
// the structure points to it). It returns false for a buffer too short for its own lengths.
func parseDebugMessage(buf []byte) (debugMessage, bool) {
	const size = int(unsafe.Sizeof(d3d12Message{}))
	if len(buf) < size {
		return debugMessage{}, false
	}
	// The members are read at their offsets, which the layout test checks: Severity at 4, ID at 8 and
	// the length at 24 (the pointer at 16 points into buf and is not followed).
	n := int(binary.LittleEndian.Uint64(buf[24:32]))
	if n < 0 || size+n > len(buf) {
		return debugMessage{}, false
	}
	return debugMessage{
		Severity: binary.LittleEndian.Uint32(buf[4:8]),
		ID:       binary.LittleEndian.Uint32(buf[8:12]),
		Text:     strings.TrimRight(string(buf[size:size+n]), "\x00"),
	}, true
}
