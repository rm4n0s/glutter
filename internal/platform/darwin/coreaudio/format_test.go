package coreaudio

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
)

func TestStreamDescriptionRoundTrips(t *testing.T) {
	tests := []platform.AudioFormat{{Rate: 48000, Channels: 2}, {Rate: 44100, Channels: 1}, {Rate: 96000, Channels: 8}, {Rate: 1000, Channels: 1}}
	for _, f := range tests {
		sd := NewStreamDescription(f)
		if sd.FormatID != 0x6C70636D || sd.FramesPerPacket != 1 || sd.BytesPerFrame != uint32(4*f.Channels) || sd.BytesPerPacket != sd.BytesPerFrame || sd.BitsPerChannel != 32 {
			t.Errorf("%+v: description %+v", f, sd)
		}
		if sd.FormatFlags != formatFlagIsFloat|formatFlagIsPacked {
			t.Errorf("%+v: flags %#x", f, sd.FormatFlags)
		}
		if got, ok := sd.Format(); !ok || got != f {
			t.Errorf("%+v: Format() = %+v, %v", f, got, ok)
		}
	}
}

func TestStreamDescriptionFormatRefusesWhatIsNotOurs(t *testing.T) {
	good := NewStreamDescription(platform.AudioFormat{Rate: 48000, Channels: 2})
	mods := map[string]func(s *StreamDescription){
		"not linear PCM":  func(s *StreamDescription) { s.FormatID = 0x61616320 },
		"not float":       func(s *StreamDescription) { s.FormatFlags &^= formatFlagIsFloat },
		"big endian":      func(s *StreamDescription) { s.FormatFlags |= formatFlagIsBigEndian },
		"non interleaved": func(s *StreamDescription) { s.FormatFlags |= formatFlagNonInterleave },
		"not packed":      func(s *StreamDescription) { s.FormatFlags &^= formatFlagIsPacked },
		"16 bits":         func(s *StreamDescription) { s.BitsPerChannel = 16 },
		"no channels":     func(s *StreamDescription) { s.ChannelsPerFrame = 0 },
		"nine channels":   func(s *StreamDescription) { s.ChannelsPerFrame = 9; s.BytesPerFrame = 36 },
		"padded frame":    func(s *StreamDescription) { s.BytesPerFrame = 12 },
		"no rate":         func(s *StreamDescription) { s.SampleRate = 0 },
	}
	for name, mod := range mods {
		t.Run(name, func(t *testing.T) {
			s := good
			mod(&s)
			if f, ok := s.Format(); ok {
				t.Errorf("Format() = %+v", f)
			}
		})
	}
	// A rate that is a fraction is rounded.
	s := good
	s.SampleRate = 44099.6
	if f, ok := s.Format(); !ok || f.Rate != 44100 {
		t.Errorf("Format() = %+v, %v", f, ok)
	}
}

func TestCheckFormat(t *testing.T) {
	tests := []struct {
		name string
		f    platform.AudioFormat
		ok   bool
	}{
		{"the backend's choice", platform.AudioFormat{}, true},
		{"stereo 48k", platform.AudioFormat{Rate: 48000, Channels: 2}, true},
		{"only a rate", platform.AudioFormat{Rate: 22050}, true},
		{"only channels", platform.AudioFormat{Channels: 6}, true},
		{"the largest", platform.AudioFormat{Rate: 768000, Channels: 8}, true},
		{"too many channels", platform.AudioFormat{Channels: 9}, false},
		{"negative channels", platform.AudioFormat{Channels: -1}, false},
		{"a rate that is too low", platform.AudioFormat{Rate: 999}, false},
		{"a rate that is too high", platform.AudioFormat{Rate: 768001}, false},
		{"a negative rate", platform.AudioFormat{Rate: -5}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckFormat(tt.f)
			if tt.ok {
				if err != nil {
					t.Errorf("CheckFormat = %v", err)
				}
				return
			}
			appErr, ok := errors.FromError(err)
			if !ok || !appErr.HasRoute("CheckFormat.AudioFormatUnsupported") {
				t.Errorf("error = %v", err)
			}
		})
	}
}

func TestBufferFrames(t *testing.T) {
	tests := []struct{ asked, rate, want int }{{0, 48000, 480}, {0, 44100, 441}, {0, 4000, 64}, {1024, 48000, 1024}, {0, 96000, 960}}
	for _, tt := range tests {
		if got := bufferFrames(tt.asked, tt.rate); got != tt.want {
			t.Errorf("bufferFrames(%d, %d) = %d, want %d", tt.asked, tt.rate, got, tt.want)
		}
	}
}

func TestFourCC(t *testing.T) {
	if fourcc("lpcm") != 0x6C70636D || fourcc("dIn ") != 0x64496E20 {
		t.Errorf("fourcc = %#x, %#x", fourcc("lpcm"), fourcc("dIn "))
	}
}

func TestLayoutTag(t *testing.T) {
	tests := map[int]uint32{1: 100<<16 | 1, 2: 101<<16 | 2, 3: 147<<16 | 3, 6: 147<<16 | 6, 8: 147<<16 | 8}
	for n, want := range tests {
		if got := layoutTag(n); got != want {
			t.Errorf("layoutTag(%d) = %#x, want %#x", n, got, want)
		}
	}
}
