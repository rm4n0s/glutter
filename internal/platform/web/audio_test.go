package web

import (
	"reflect"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
)

func TestMediaErrorTag(t *testing.T) {
	tests := []struct{ name, want string }{
		{"NotAllowedError", platform.TagAudioCapturePermissionDenied},
		{"SecurityError", platform.TagAudioCapturePermissionDenied},
		{"PermissionDeniedError", platform.TagAudioCapturePermissionDenied},
		{"OverconstrainedError", platform.TagAudioFormatUnsupported},
		{"NotSupportedError", platform.TagAudioFormatUnsupported},
		{"IndexSizeError", platform.TagAudioFormatUnsupported},
		{"NotFoundError", platform.TagAudioDeviceOpenFailed},
		{"NotReadableError", platform.TagAudioDeviceOpenFailed},
		{"AbortError", platform.TagAudioDeviceOpenFailed},
		{"", platform.TagAudioDeviceOpenFailed},
	}
	for _, tt := range tests {
		if got := MediaErrorTag(tt.name); got != tt.want {
			t.Errorf("MediaErrorTag(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestAudioDevicesOf(t *testing.T) {
	list := []MediaDevice{
		{Kind: "videoinput", ID: "cam"},
		{Kind: "audioinput", ID: "mic1", Label: "USB microphone"},
		{Kind: "audioinput", ID: "default", Label: "Default - USB microphone"},
		{Kind: "audiooutput", ID: "spk1"},
		{Kind: "audiooutput", ID: "spk2", Label: "Headphones"},
	}
	tests := []struct {
		name string
		list []MediaDevice
		kind platform.AudioKind
		want []platform.AudioDeviceInfo
	}{
		{"inputs, the default has the ID default", list, platform.AudioCapture, []platform.AudioDeviceInfo{
			{ID: "mic1", Name: "USB microphone", Kind: platform.AudioCapture},
			{ID: "default", Name: "Default - USB microphone", Kind: platform.AudioCapture, Default: true},
		}},
		{"outputs, no default: the first, and a name from the number", list, platform.AudioPlayback, []platform.AudioDeviceInfo{
			{ID: "spk1", Name: "Audio output 1", Kind: platform.AudioPlayback, Default: true},
			{ID: "spk2", Name: "Headphones", Kind: platform.AudioPlayback},
		}},
		{"no devices of the kind", []MediaDevice{{Kind: "videoinput", ID: "cam"}}, platform.AudioCapture, nil},
		{"an empty list", nil, platform.AudioPlayback, nil},
		{"unlabelled microphone", []MediaDevice{{Kind: "audioinput", ID: "m"}}, platform.AudioCapture, []platform.AudioDeviceInfo{
			{ID: "m", Name: "Microphone 1", Kind: platform.AudioCapture, Default: true},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AudioDevicesOf(tt.list, tt.kind); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestDefaultAudioDevice(t *testing.T) {
	for _, kind := range []platform.AudioKind{platform.AudioPlayback, platform.AudioCapture} {
		d := DefaultAudioDevice(kind)
		if d.ID != "default" || !d.Default || d.Kind != kind || d.Name == "" {
			t.Errorf("DefaultAudioDevice(%v) = %+v", kind, d)
		}
	}
}

func TestPlaybackChannels(t *testing.T) {
	tests := []struct{ asked, max, want int }{
		{0, 0, 2}, {-1, 8, 2}, {1, 2, 1}, {2, 2, 2}, {6, 2, 2}, {6, 0, 6}, {8, 6, 6}, {0, 1, 1},
	}
	for _, tt := range tests {
		if got := PlaybackChannels(tt.asked, tt.max); got != tt.want {
			t.Errorf("PlaybackChannels(%d, %d) = %d, want %d", tt.asked, tt.max, got, tt.want)
		}
	}
}

func TestAudioTarget(t *testing.T) {
	tests := []struct{ frames, rate, want int }{
		{0, 48000, 4800}, {0, 44100, 4410}, {0, 8000, 800}, {1024, 48000, 1024}, {-5, 48000, 4800},
	}
	for _, tt := range tests {
		if got := AudioTarget(tt.frames, tt.rate); got != tt.want {
			t.Errorf("AudioTarget(%d, %d) = %d, want %d", tt.frames, tt.rate, got, tt.want)
		}
	}
	if AudioDefaultBuffer != 100*time.Millisecond {
		t.Errorf("the default buffer is %v, SPEC §22 says 100 ms", AudioDefaultBuffer)
	}
}
