//go:build linux

package vulkan

import (
	"testing"

	"github.com/rm4n0s/errors"
)

// testGPU returns a device that works: Vulkan 1.3 with both features and a graphics queue.
func testGPU(index int, name string, typ uint32) physicalInfo {
	return physicalInfo{
		Index: index, Name: name, Type: typ, APIVersion: apiVersion13,
		DynamicRendering: true, Synchronization2: true, MaxImageDimension2D: 16384,
		QueueFlags: []uint32{queueGraphics},
	}
}

func TestChooseDevice(t *testing.T) {
	old := testGPU(1, "Old GPU", deviceTypeIntegrated)
	old.APIVersion = apiVersion11
	old.Extensions = map[string]bool{extDynamicRendering: true, extSynchronization2: true}
	noExt := old
	noExt.Extensions = map[string]bool{extDynamicRendering: true}
	noFeature := testGPU(2, "No Feature", deviceTypeDiscrete)
	noFeature.DynamicRendering = false
	small := testGPU(3, "Small", deviceTypeDiscrete)
	small.MaxImageDimension2D = 4096
	noGraphics := testGPU(4, "Compute", deviceTypeDiscrete)
	noGraphics.QueueFlags = []uint32{0x2, 0x4}
	secondFamily := testGPU(5, "Two Families", deviceTypeDiscrete)
	secondFamily.QueueFlags = []uint32{0x4, queueGraphics | 0x2}

	tests := []struct {
		name       string
		list       []physicalInfo
		opts       chooseOpts
		wantDevice int
		wantFamily int
		wantTag    string
	}{
		{name: "the first device in the loader's order", list: []physicalInfo{testGPU(0, "AMD Radeon", deviceTypeDiscrete), testGPU(1, "Intel", deviceTypeIntegrated)}, wantDevice: 0},
		{name: "the first one that works", list: []physicalInfo{noFeature, testGPU(7, "Works", deviceTypeIntegrated)}, wantDevice: 7},
		{name: "Vulkan 1.1 with the two extensions", list: []physicalInfo{old}, wantDevice: 1},
		{name: "a queue family that is not the first", list: []physicalInfo{secondFamily}, wantDevice: 5, wantFamily: 1},
		{name: "a CPU device is skipped when it is not asked for", list: []physicalInfo{testGPU(0, "llvmpipe", deviceTypeCPU), testGPU(1, "Intel", deviceTypeIntegrated)}, wantDevice: 1},
		{name: "a CPU device is used when Vulkan is asked for", list: []physicalInfo{testGPU(0, "llvmpipe", deviceTypeCPU)}, opts: chooseOpts{AllowCPU: true}, wantDevice: 0},
		{name: "a CPU device first is still first when allowed", list: []physicalInfo{testGPU(0, "llvmpipe", deviceTypeCPU), testGPU(1, "Intel", deviceTypeIntegrated)}, opts: chooseOpts{AllowCPU: true}, wantDevice: 0},
		{name: "the selector as an index", list: []physicalInfo{testGPU(0, "A", deviceTypeDiscrete), testGPU(1, "B", deviceTypeDiscrete)}, opts: chooseOpts{Selector: "1"}, wantDevice: 1},
		{name: "the selector as part of a name, in any case", list: []physicalInfo{testGPU(0, "AMD Radeon 8060S", deviceTypeDiscrete), testGPU(1, "Intel Iris Xe", deviceTypeIntegrated)}, opts: chooseOpts{Selector: "iris"}, wantDevice: 1},
		{name: "the selector can name a CPU device when allowed", list: []physicalInfo{testGPU(0, "Intel", deviceTypeIntegrated), testGPU(1, "llvmpipe (LLVM 19)", deviceTypeCPU)}, opts: chooseOpts{Selector: "llvmpipe", AllowCPU: true}, wantDevice: 1},
		{name: "a present check chooses the family", list: []physicalInfo{secondFamily}, opts: chooseOpts{CanPresent: func(d, f int) bool { return f == 1 }}, wantDevice: 5, wantFamily: 1},

		{name: "no devices", list: nil, wantTag: "VulkanNoDevice"},
		{name: "only a CPU device", list: []physicalInfo{testGPU(0, "llvmpipe", deviceTypeCPU)}, wantTag: "VulkanNoDevice"},
		{name: "no graphics queue", list: []physicalInfo{noGraphics}, wantTag: "VulkanNoDevice"},
		{name: "no family can present", list: []physicalInfo{secondFamily}, opts: chooseOpts{CanPresent: func(int, int) bool { return false }}, wantTag: "VulkanNoDevice"},
		{name: "the selector matches nothing", list: []physicalInfo{testGPU(0, "A", deviceTypeDiscrete)}, opts: chooseOpts{Selector: "nvidia"}, wantTag: "VulkanNoDevice"},
		{name: "the selector names a CPU device that is not allowed", list: []physicalInfo{testGPU(0, "llvmpipe", deviceTypeCPU)}, opts: chooseOpts{Selector: "0"}, wantTag: "VulkanNoDevice"},
		{name: "Vulkan 1.0 only", list: []physicalInfo{{Index: 0, Name: "Ancient", APIVersion: 0x00400000, QueueFlags: []uint32{queueGraphics}}}, wantTag: "VulkanUnsupported"},
		{name: "1.1 without both extensions", list: []physicalInfo{noExt}, wantTag: "VulkanUnsupported"},
		{name: "no dynamic rendering feature", list: []physicalInfo{noFeature}, wantTag: "VulkanUnsupported"},
		{name: "textures under the design limit", list: []physicalInfo{small}, wantTag: "VulkanUnsupported"},
		{name: "the selector names a device that lacks the features", list: []physicalInfo{noFeature, testGPU(3, "Works", deviceTypeDiscrete)}, opts: chooseOpts{Selector: "No Feature"}, wantTag: "VulkanUnsupported"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device, family, err := chooseDevice(tt.list, tt.opts)
			if tt.wantTag != "" {
				appErr, ok := errors.FromError(err)
				if !ok {
					t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
				}
				if appErr.Tag != tt.wantTag || !appErr.HasRoute("chooseDevice."+tt.wantTag) {
					t.Errorf("got %s via %s, want %s via chooseDevice", appErr.Tag, appErr.Route(), tt.wantTag)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if device != tt.wantDevice || family != tt.wantFamily {
				t.Errorf("chose device %d family %d, want %d and %d", device, family, tt.wantDevice, tt.wantFamily)
			}
		})
	}
}

func TestChooseDeviceErrorsListWhatWasRejected(t *testing.T) {
	_, _, err := chooseDevice([]physicalInfo{testGPU(0, "llvmpipe", deviceTypeCPU)}, chooseOpts{})
	appErr, _ := errors.FromError(err)
	rejected, _ := appErr.Metadata["rejected"].([]string)
	if len(rejected) != 1 || rejected[0] == "" {
		t.Errorf("rejected = %v, want one line that says why", appErr.Metadata["rejected"])
	}
}

func TestCheckDriver(t *testing.T) {
	if err := checkDriver(physicalInfo{Name: "Adreno (TM) 660", VendorID: 0x168C}); err != nil {
		t.Errorf("a driver that is not denied failed: %v", err)
	}
	err := checkDriver(physicalInfo{Name: "Adreno (TM) 630", VendorID: 0x168C, DeviceID: 7, DriverVersion: 9})
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T", err)
	}
	if !appErr.HasRoute("checkDriver.VulkanDriverDenied") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
	if appErr.Metadata["device"] != "Adreno (TM) 630" || appErr.Metadata["driver_version"] != 9 {
		t.Errorf("metadata: %v", appErr.Metadata)
	}
}
