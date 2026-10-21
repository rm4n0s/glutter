//go:build linux

package vulkan

import (
	"testing"

	"github.com/rm4n0s/errors"
)

func TestVkError(t *testing.T) {
	tests := []struct {
		name   string
		result int32
		tag    string
		route  string
	}{
		{"out of device memory", resultErrorOutOfDevice, "GPUOutOfMemory", "vkError.GPUOutOfMemory"},
		{"out of host memory", resultErrorOutOfHost, "GPUOutOfMemory", "vkError.GPUOutOfMemory"},
		{"a lost device", resultErrorDeviceLost, "VulkanDeviceLost", "vkError.VulkanDeviceLost"},
		{"initialization failed", resultErrorInitFailed, "VulkanCallFailed", "vkError.VulkanCallFailed"},
		{"a result that has no name", -999, "VulkanCallFailed", "vkError.VulkanCallFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := check("vkCreateBuffer", tt.result)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != tt.tag || !appErr.HasRoute("check->"+tt.route) {
				t.Errorf("got %s via %s, want %s via check->%s", appErr.Tag, appErr.Route(), tt.tag, tt.route)
			}
			if appErr.Metadata["call"] != "vkCreateBuffer" || appErr.Metadata["result"] != int(tt.result) {
				t.Errorf("metadata = %v", appErr.Metadata)
			}
		})
	}
	if err := check("vkCreateBuffer", resultSuccess); err != nil {
		t.Errorf("VK_SUCCESS gave %v", err)
	}
	if name := resultNames[resultErrorOutOfDevice]; name != "VK_ERROR_OUT_OF_DEVICE_MEMORY" {
		t.Errorf("name = %q", name)
	}
}
