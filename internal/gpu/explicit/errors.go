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
