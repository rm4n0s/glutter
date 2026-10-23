package vulkan

import "testing"

// The cases of impeller/renderer/backend/vulkan/driver_info_vk_unittests.cc (the Adreno list, the
// Mali names, the Maleoon vendor), and one for each other rule of IsKnownBadDriver.
func TestIsKnownBadDriver(t *testing.T) {
	const (
		qualcomm, arm, huawei, samsung, imgtec, mesa, google = 0x168C, 0x13B5, 0x19E5, 0x144D, 0x1010, 0x10005, 0x1AE0
		v11, v13                                             = 0x00401000, 0x00403000
	)
	tests := []struct {
		name    string
		d       DriverInfo
		wantBad bool
	}{
		{"adreno 504", DriverInfo{Name: "Adreno (TM) 504", VendorID: qualcomm}, true},
		{"adreno 505", DriverInfo{Name: "Adreno (TM) 505", VendorID: qualcomm}, true},
		{"adreno 506", DriverInfo{Name: "Adreno (TM) 506", VendorID: qualcomm}, true},
		{"adreno 508", DriverInfo{Name: "Adreno (TM) 508", VendorID: qualcomm}, true},
		{"adreno 509", DriverInfo{Name: "Adreno (TM) 509", VendorID: qualcomm}, true},
		{"adreno 512", DriverInfo{Name: "Adreno (TM) 512", VendorID: qualcomm}, true},
		{"adreno 530", DriverInfo{Name: "Adreno (TM) 530", VendorID: qualcomm}, true},
		{"adreno 610", DriverInfo{Name: "Adreno (TM) 610", VendorID: qualcomm}, true},
		{"adreno 620", DriverInfo{Name: "Adreno (TM) 620", VendorID: qualcomm}, true},
		{"adreno 630", DriverInfo{Name: "Adreno (TM) 630", VendorID: qualcomm}, true},
		{"adreno 640", DriverInfo{Name: "Adreno (TM) 640", VendorID: qualcomm}, true},
		{"adreno 650 (the last denied)", DriverInfo{Name: "Adreno (TM) 650", VendorID: qualcomm}, true},
		{"adreno 660", DriverInfo{Name: "Adreno (TM) 660", VendorID: qualcomm}, false},
		{"adreno 750", DriverInfo{Name: "Adreno (TM) 750", VendorID: qualcomm}, false},
		{"adreno unknown newer model", DriverInfo{Name: "Adreno (TM) 830", VendorID: qualcomm}, false},
		{"another Qualcomm vendor ID", DriverInfo{Name: "Adreno (TM) 540", VendorID: 0x5143}, true},
		{"a name that is not Adreno's", DriverInfo{Name: "Qualcomm GPU", VendorID: qualcomm}, false},
		{"mali G52", DriverInfo{Name: "Mali-G52", VendorID: arm}, false},
		{"mali with more in the name", DriverInfo{Name: "Mali-G54-MORE STUFF", VendorID: arm}, false},
		{"maleoon", DriverInfo{Name: "Maleoon 910", VendorID: huawei}, true},
		{"xclipse below Vulkan 1.3", DriverInfo{Name: "Samsung Xclipse 920", VendorID: samsung, APIVersion: v11}, true},
		{"xclipse with Vulkan 1.3", DriverInfo{Name: "Samsung Xclipse 940", VendorID: samsung, APIVersion: v13}, false},
		{"pixel 10, old driver", DriverInfo{Name: "PowerVR D-Series DXT-48-1536", VendorID: imgtec, DeviceID: 0x71061212, DriverVersion: 6794073}, true},
		{"pixel 10, driver 25.1", DriverInfo{Name: "PowerVR D-Series DXT-48-1536", VendorID: imgtec, DeviceID: 0x71061212, DriverVersion: 6794074}, false},
		{"another device with that driver version", DriverInfo{Name: "PowerVR D-Series DXT-48-1536", VendorID: imgtec, DeviceID: 1, DriverVersion: 1}, false},
		{"powervr rogue", DriverInfo{Name: "PowerVR Rogue GE8320", VendorID: imgtec}, false}, // no known series: not denied
		{"powervr BXM", DriverInfo{Name: "PowerVR B-Series BXM-8-256", VendorID: imgtec}, true},
		{"powervr BXT", DriverInfo{Name: "PowerVR B-Series BXT-32-1024", VendorID: imgtec}, true},
		{"powervr CXT", DriverInfo{Name: "PowerVR C-Series CXT-48-1536", VendorID: imgtec}, false},
		{"mesa", DriverInfo{Name: "llvmpipe (LLVM 19.1.7, 256 bits)", VendorID: mesa}, false},
		{"the emulator's SwiftShader", DriverInfo{Name: "SwiftShader Device (Subzero)", VendorID: google}, false},
		{"unknown vendor", DriverInfo{Name: "Adreno (TM) 504", VendorID: 0xABCD}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bad, why := tt.d.IsKnownBadDriver()
			if bad != tt.wantBad {
				t.Errorf("IsKnownBadDriver() = %v (%q), want %v", bad, why, tt.wantBad)
			}
			if bad && why == "" {
				t.Error("a denied driver has no reason")
			}
		})
	}
}

func TestVendorOf(t *testing.T) {
	tests := []struct {
		id   uint32
		want Vendor
	}{
		{0x1AE0, VendorGoogle}, {0x168C, VendorQualcomm}, {0x17CB, VendorQualcomm}, {0x1969, VendorQualcomm}, {0x5143, VendorQualcomm},
		{0x13B5, VendorARM}, {0x1010, VendorImgTec}, {0x1002, VendorAMD}, {0x1022, VendorAMD}, {0x10DE, VendorNvidia},
		{0x8086, VendorIntel}, {0x106B, VendorApple}, {0x19E5, VendorHuawei}, {0x144D, VendorSamsung}, {0x10005, VendorMesa}, {1, VendorUnknown},
	}
	for _, tt := range tests {
		if got := vendorOf(tt.id); got != tt.want {
			t.Errorf("vendorOf(%#x) = %v, want %v", tt.id, got, tt.want)
		}
	}
}
