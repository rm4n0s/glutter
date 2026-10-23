package engine

import "github.com/rm4n0s/errors"

// androidVulkanMinSDK is the first Android release that `auto` chooses Vulkan on: Android 10, API
// level 29 (SPEC §24.4).
const androidVulkanMinSDK = 29

// checkAndroidVulkanSDK fails with AndroidVulkanTooOld when `auto` would choose Vulkan on a device
// below API level 29; Vulkan asked for by name is tried on any level that has the loader. The
// chooser logs the error as the reason that it went on to the next candidate.
func checkAndroidVulkanSDK(sdk int, named bool) error {
	if named || sdk >= androidVulkanMinSDK {
		return nil
	}
	return errors.New(tagAndroidVulkanTooOld, "Vulkan is chosen on Android 10 (API 29) and later; this device is older",
		"sdk", sdk, "minimum", androidVulkanMinSDK)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagAndroidVulkanTooOld = "AndroidVulkanTooOld"
)
