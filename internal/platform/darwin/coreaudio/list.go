package coreaudio

import (
	"encoding/binary"
	"strings"

	"github.com/rm4n0s/glutter/internal/platform"
)

// DeviceRecord is what the system says of one audio device.
//
// Flutter: no counterpart
type DeviceRecord struct {
	// Object is the AudioObjectID of the device.
	Object uint32
	// UID is the identifier that outlives the run of the system, which is the ID of the device for Open.
	UID  string
	Name string
	// OutChannels and InChannels are the channels of the streams that the device plays and records.
	OutChannels, InChannels int
	// Rate is the nominal sample rate of the device.
	Rate float64
}

// aggregatePrefix is the UID of the aggregate devices that Apple's frameworks make for themselves: they are not for an app
// (SDL leaves them out as well).
const aggregatePrefix = "CADefaultDeviceAggregate"

// BuildDevices makes the list of the devices of a kind: those that have a stream that goes that way, with the default of the system
// marked (defaultObject is the AudioObjectID of the default device of the kind) and the format that the device likes best.
//
// Flutter: no counterpart
func BuildDevices(kind platform.AudioKind, records []DeviceRecord, defaultObject uint32) []platform.AudioDeviceInfo {
	var out []platform.AudioDeviceInfo
	for _, r := range records {
		ch := r.OutChannels
		if kind == platform.AudioCapture {
			ch = r.InChannels
		}
		if ch <= 0 || strings.HasPrefix(r.UID, aggregatePrefix) {
			continue
		}
		info := platform.AudioDeviceInfo{ID: r.UID, Name: r.Name, Kind: kind, Default: r.Object == defaultObject && defaultObject != 0}
		if r.Rate >= minRate && r.Rate <= maxRate {
			info.Format = platform.AudioFormat{Rate: int(r.Rate + 0.5), Channels: min(ch, maxChannels)}
		}
		out = append(out, info)
	}
	return out
}

// ChannelCount returns the number of channels of an AudioBufferList, which is what the stream configuration of a device is: the
// count of buffers, and for each the channels it holds. The list is 8 bytes of head and 16 bytes for each buffer, in the byte
// order of the machine. A list that is cut short counts only the buffers that are whole.
//
// Flutter: no counterpart
func ChannelCount(list []byte) int {
	if len(list) < 4 {
		return 0
	}
	n := int(binary.LittleEndian.Uint32(list))
	total := 0
	for i := range n {
		off := 8 + 16*i
		if off+4 > len(list) {
			break
		}
		total += int(binary.LittleEndian.Uint32(list[off:]))
	}
	return total
}
