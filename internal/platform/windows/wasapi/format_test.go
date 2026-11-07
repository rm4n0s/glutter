package wasapi

import (
	"testing"
	"time"
	"unsafe"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

func TestChannelMask(t *testing.T) {
	// The layouts are SDL's: the channels of a frame come in the order of the bits of the mask, which is the order of SDL's layouts.
	tests := []struct {
		channels int
		want     uint32
	}{
		{0, 0}, {1, 0x4}, {2, 0x3}, {3, 0xB}, {4, 0x33}, {5, 0x3B}, {6, 0x3F}, {7, 0x70F}, {8, 0x63F}, {9, 0}, {-1, 0},
	}
	for _, tt := range tests {
		if got := channelMask(tt.channels); got != tt.want {
			t.Errorf("channelMask(%d) = %#x, want %#x", tt.channels, got, tt.want)
		}
	}
	for n := 1; n <= maxChannels; n++ {
		mask := channelMask(n)
		bits := 0
		for ; mask != 0; mask &= mask - 1 {
			bits++
		}
		if bits != n {
			t.Errorf("the mask of %d channels has %d speakers", n, bits)
		}
	}
}

func TestWireFormat(t *testing.T) {
	tests := []struct {
		f           platform.AudioFormat
		blockAlign  uint16
		bytesPerSec uint32
	}{
		{platform.AudioFormat{Rate: 48000, Channels: 2}, 8, 384000},
		{platform.AudioFormat{Rate: 44100, Channels: 1}, 4, 176400},
		{platform.AudioFormat{Rate: 96000, Channels: 8}, 32, 3072000},
	}
	for _, tt := range tests {
		w := wireFormat(tt.f)
		if w.FormatTag != win32.WaveFormatTagExtensible || w.CbSize != 22 || w.BitsPerSample != 32 || w.ValidBitsPerSample != 32 {
			t.Errorf("%+v: the header is wrong: %+v", tt.f, w)
		}
		if int(w.Channels) != tt.f.Channels || int(w.SamplesPerSec) != tt.f.Rate || w.BlockAlign != tt.blockAlign || w.AvgBytesPerSec != tt.bytesPerSec {
			t.Errorf("%+v: %+v", tt.f, w)
		}
		if w.SubFormat != win32.KSDataFormatSubTypeIEEEFloat || w.ChannelMask != channelMask(tt.f.Channels) {
			t.Errorf("%+v: the sub-format or the mask is wrong: %+v", tt.f, w)
		}
	}
	if size := unsafe.Sizeof(win32.WaveFormatExtensible{}); size != 40 {
		t.Errorf("the extensible format is %d bytes, want 40", size)
	}
}

func TestChooseFormat(t *testing.T) {
	mix := platform.AudioFormat{Rate: 48000, Channels: 2}
	tests := []struct {
		name      string
		want, mix platform.AudioFormat
		out       platform.AudioFormat
	}{
		{"nothing asked: the mix format", platform.AudioFormat{}, mix, mix},
		{"the rate only", platform.AudioFormat{Rate: 44100}, mix, platform.AudioFormat{Rate: 44100, Channels: 2}},
		{"the channels only", platform.AudioFormat{Channels: 6}, mix, platform.AudioFormat{Rate: 48000, Channels: 6}},
		{"everything asked", platform.AudioFormat{Rate: 22050, Channels: 1}, mix, platform.AudioFormat{Rate: 22050, Channels: 1}},
		{"a rate that is out of the limits", platform.AudioFormat{Rate: 5, Channels: 2}, mix, mix},
		{"more channels than there are layouts", platform.AudioFormat{Rate: 48000, Channels: 12}, mix, mix},
		{"no mix format either", platform.AudioFormat{}, platform.AudioFormat{}, platform.AudioFormat{Rate: 48000, Channels: 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := chooseFormat(tt.want, tt.mix); got != tt.out {
				t.Errorf("got %+v, want %+v", got, tt.out)
			}
		})
	}
}

func TestPeriodFrames(t *testing.T) {
	tests := []struct {
		name                         string
		want, fundamental, lo, hi, n uint32
	}{
		{"nothing wanted: the smallest period", 0, 32, 128, 4800, 128},
		{"a multiple of the fundamental", 480, 32, 128, 4800, 480},
		{"rounded to the nearest multiple", 500, 32, 128, 4800, 512},
		{"rounded down", 490, 32, 128, 4800, 480},
		{"below the smallest", 64, 32, 128, 4800, 128},
		{"above the largest", 10000, 32, 128, 4800, 4800},
		{"the largest is not a multiple", 10000, 32, 128, 4810, 4800},
		{"no fundamental", 500, 0, 128, 4800, 500},
		{"no limits", 500, 0, 0, 0, 500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := periodFrames(tt.want, tt.fundamental, tt.lo, tt.hi); got != tt.n {
				t.Errorf("periodFrames(%d, %d, %d, %d) = %d, want %d", tt.want, tt.fundamental, tt.lo, tt.hi, got, tt.n)
			}
		})
	}
}

func TestDurations(t *testing.T) {
	if got := framesDuration(480, 48000); got != 10*time.Millisecond {
		t.Errorf("480 frames at 48000 are %v", got)
	}
	if got := framesDuration(441, 44100); got != 10*time.Millisecond {
		t.Errorf("441 frames at 44100 are %v", got)
	}
	if got := framesDuration(10, 0); got != 0 {
		t.Errorf("a rate of 0 gave %v", got)
	}
}

func TestFlow(t *testing.T) {
	if flowOf(platform.AudioPlayback) != win32.ERender || flowOf(platform.AudioCapture) != win32.ECapture {
		t.Error("the flows are wrong")
	}
}

func TestClassify(t *testing.T) {
	tests := []struct {
		hr   win32.HRESULT
		want failure
	}{
		{win32.AudclntEUnsupportedFormat, failFormat},
		{win32.EAccessDenied, failPermission},
		{win32.AudclntEDeviceInvalidated, failLost},
		{win32.AudclntEResourcesInvalidated, failLost},
		{win32.AudclntEServiceNotRunning, failLost},
		{win32.AudclntEDeviceInUse, failOpen},
		{win32.ENotFound, failOpen},
		{win32.EFail, failOpen},
	}
	for _, tt := range tests {
		if got := classify(tt.hr); got != tt.want {
			t.Errorf("classify(%s) = %d, want %d", tt.hr, got, tt.want)
		}
	}
}
