package coreaudio

import (
	"encoding/binary"
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/internal/platform"
)

func TestBuildDevices(t *testing.T) {
	records := []DeviceRecord{
		{Object: 10, UID: "BuiltInSpeakerDevice", Name: "MacBook Pro Speakers", OutChannels: 2, Rate: 48000},
		{Object: 11, UID: "BuiltInMicrophoneDevice", Name: "MacBook Pro Microphone", InChannels: 1, Rate: 48000},
		{Object: 12, UID: "AppleUSBAudioEngine:Acme:Headset", Name: "Headset", OutChannels: 2, InChannels: 1, Rate: 44100},
		{Object: 13, UID: "CADefaultDeviceAggregate-123-0", Name: "CADefaultDeviceAggregate", OutChannels: 2, InChannels: 2, Rate: 48000},
		{Object: 14, UID: "Surround", Name: "Surround", OutChannels: 12, Rate: 96000},
		{Object: 15, UID: "NoRate", Name: "No rate", OutChannels: 2},
		{Object: 16, UID: "Dead", Name: "Dead", Rate: 48000},
	}
	out := BuildDevices(platform.AudioPlayback, records, 12)
	want := []platform.AudioDeviceInfo{
		{ID: "BuiltInSpeakerDevice", Name: "MacBook Pro Speakers", Kind: platform.AudioPlayback, Format: platform.AudioFormat{Rate: 48000, Channels: 2}},
		{ID: "AppleUSBAudioEngine:Acme:Headset", Name: "Headset", Kind: platform.AudioPlayback, Default: true, Format: platform.AudioFormat{Rate: 44100, Channels: 2}},
		{ID: "Surround", Name: "Surround", Kind: platform.AudioPlayback, Format: platform.AudioFormat{Rate: 96000, Channels: 8}},
		{ID: "NoRate", Name: "No rate", Kind: platform.AudioPlayback},
	}
	if !reflect.DeepEqual(out, want) {
		t.Errorf("playback = %+v\nwant     %+v", out, want)
	}
	in := BuildDevices(platform.AudioCapture, records, 11)
	if len(in) != 2 || in[0].ID != "BuiltInMicrophoneDevice" || !in[0].Default || in[1].Default || in[1].Format.Channels != 1 {
		t.Errorf("capture = %+v", in)
	}
	// No default: nothing is marked.
	for _, d := range BuildDevices(platform.AudioPlayback, records, 0) {
		if d.Default {
			t.Errorf("%s is the default with no default", d.ID)
		}
	}
	if got := BuildDevices(platform.AudioPlayback, nil, 1); got != nil {
		t.Errorf("no records gave %+v", got)
	}
}

// audioBufferList makes the bytes of an AudioBufferList with the channels of each buffer.
func audioBufferList(declared int, channels ...int) []byte {
	b := make([]byte, 8+16*len(channels))
	binary.LittleEndian.PutUint32(b, uint32(declared))
	for i, c := range channels {
		binary.LittleEndian.PutUint32(b[8+16*i:], uint32(c))
		binary.LittleEndian.PutUint32(b[8+16*i+4:], uint32(c*4096))
	}
	return b
}

func TestChannelCount(t *testing.T) {
	tests := []struct {
		name string
		list []byte
		want int
	}{
		{"nothing", nil, 0},
		{"a head cut short", []byte{1, 0}, 0},
		{"no buffers", audioBufferList(0), 0},
		{"one stereo stream", audioBufferList(1, 2), 2},
		{"two streams", audioBufferList(2, 2, 6), 8},
		{"a count that is more than the bytes", audioBufferList(5, 2, 2), 4},
		{"a buffer that is cut", audioBufferList(1, 2)[:10], 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ChannelCount(tt.list); got != tt.want {
				t.Errorf("ChannelCount = %d, want %d", got, tt.want)
			}
		})
	}
}
