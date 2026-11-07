//go:build windows

package wasapi

import (
	"testing"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

func TestHResultError(t *testing.T) {
	tests := []struct {
		hr  win32.HRESULT
		tag string
	}{
		{win32.AudclntEUnsupportedFormat, platform.TagAudioFormatUnsupported},
		{win32.EAccessDenied, platform.TagAudioCapturePermissionDenied},
		{win32.AudclntEDeviceInvalidated, platform.TagAudioDeviceLost},
		{win32.AudclntEResourcesInvalidated, platform.TagAudioDeviceLost},
		{win32.AudclntEServiceNotRunning, platform.TagAudioDeviceLost},
		{win32.AudclntEDeviceInUse, platform.TagAudioDeviceOpenFailed},
		{win32.ENotFound, platform.TagAudioDeviceOpenFailed},
		{win32.EFail, platform.TagAudioDeviceOpenFailed},
	}
	for _, tt := range tests {
		err := hresultError("IAudioClient::Initialize", tt.hr)
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != tt.tag || !appErr.HasRoute("hresultError."+tt.tag) {
			t.Errorf("%s: got %v, want tag %s", tt.hr, err, tt.tag)
			continue
		}
		if appErr.Metadata["method"] != "IAudioClient::Initialize" || appErr.Metadata["hresult"] != tt.hr.String() {
			t.Errorf("%s: the metadata is %v", tt.hr, appErr.Metadata)
		}
	}
}

func TestMixFormatHeadIsTheStartOfTheFormat(t *testing.T) {
	// The head is read from a WAVEFORMATEX of 18 bytes or from the extensible one; it must sit at the start of both.
	w := wireFormat(platform.AudioFormat{Rate: 44100, Channels: 6})
	head := (*waveHead)(unsafe.Pointer(&w))
	if head.Tag != win32.WaveFormatTagExtensible || head.Channels != 6 || head.Rate != 44100 {
		t.Errorf("head %+v", *head)
	}
}
