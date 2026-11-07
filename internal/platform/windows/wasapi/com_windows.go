//go:build windows

package wasapi

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// call calls the method at index of a COM object and returns its HRESULT. The pointers that the arguments hold are converted in the
// argument list of the call, which keeps them alive for it.
//
//go:uintptrescapes
func call(o win32.Object, index int, args ...uintptr) win32.HRESULT {
	return win32.FromResult(o.Call(index, args...))
}

// release drops the reference to a COM object, if there is one.
func release(o win32.Object) {
	if !o.IsNil() {
		o.Release()
	}
}

// hresultError is the error of a COM method of the audio API that failed. It is one table, like win32.HResultError: the tag is
// AudioFormatUnsupported for a format that the device does not take, AudioCapturePermissionDenied for the access to a microphone
// that the user's settings refuse, AudioDeviceLost when the client was invalidated and AudioDeviceOpenFailed for any other
// failure; the metadata are "method" and "hresult". It is listed in errcheck.allow.
func hresultError(method string, hr win32.HRESULT) error {
	switch classify(hr) {
	case failFormat:
		return errors.New(platform.TagAudioFormatUnsupported, "the audio device does not take the format", "method", method, "hresult", hr.String())
	case failPermission:
		return errors.New(platform.TagAudioCapturePermissionDenied, "the use of the microphone is not allowed", "method", method, "hresult", hr.String())
	case failLost:
		return errors.New(platform.TagAudioDeviceLost, "the audio device is gone", "method", method, "hresult", hr.String())
	}
	return errors.New(platform.TagAudioDeviceOpenFailed, "the audio device could not be opened", "method", method, "hresult", hr.String())
}
