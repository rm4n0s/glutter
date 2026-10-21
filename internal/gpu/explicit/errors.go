package explicit

import "github.com/rm4n0s/errors"

// SwapchainOutOfDate is the tag of the error that Swapchain.Acquire returns when the
// swapchain no longer matches its surface (the window was resized, or the platform asks
// for a new one): the caller configures it again and retries once (SPEC §9.3).
//
// Flutter: no counterpart
const SwapchainOutOfDate = "SwapchainOutOfDate"

// IsSwapchainOutOfDate reports whether err is the error of an out-of-date swapchain.
//
// Flutter: no counterpart
func IsSwapchainOutOfDate(err error) bool {
	appErr, ok := errors.FromError(err)
	return ok && appErr.Tag == SwapchainOutOfDate
}

// GPUDevicePending is the tag of the error that a Renderer's open function returns when the backend
// has no device at the moment and is getting one without blocking: WebGPU asks the browser for a new
// device after the old one was lost, and the UI goroutine never waits for a promise (SPEC §9.5,
// §23.3). The renderer draws nothing for that frame and returns no error; the backend tells its
// owner when the device is there, and the next frame opens it.
//
// Flutter: no counterpart
const GPUDevicePending = "GPUDevicePending"

// IsDevicePending reports whether err is the error of a device that is on its way.
//
// Flutter: no counterpart
func IsDevicePending(err error) bool {
	appErr, ok := errors.FromError(err)
	return ok && appErr.Tag == GPUDevicePending
}
