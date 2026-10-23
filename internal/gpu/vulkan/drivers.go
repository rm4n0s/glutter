package vulkan

import "strings"

// The deny list of drivers that Flutter's Impeller keeps (SPEC §24.4): `auto` on Android does not
// choose Vulkan on a device whose driver is known to misbehave, and draws with GLES instead. It is
// plain Go, so that its table test runs on every machine.

// Vendor is who made the driver: the vendor ID of VkPhysicalDeviceProperties, as Impeller's
// VendorVK knows them.
//
// Flutter: impeller/renderer/backend/vulkan/driver_info_vk.h VendorVK
type Vendor int

// The vendors that Impeller tells apart.
//
// Flutter: impeller/renderer/backend/vulkan/driver_info_vk.h VendorVK
const (
	VendorUnknown Vendor = iota
	VendorGoogle
	VendorQualcomm
	VendorARM
	VendorImgTec
	VendorAMD
	VendorNvidia
	VendorIntel
	VendorMesa
	VendorApple
	VendorHuawei
	VendorSamsung
)

// vendorOf identifies the vendor from its PCI or Khronos ID.
//
// Flutter: impeller/renderer/backend/vulkan/driver_info_vk.cc IdentifyVendor
func vendorOf(id uint32) Vendor {
	switch id {
	case 0x1AE0:
		return VendorGoogle
	case 0x168C, 0x17CB, 0x1969, 0x5143:
		return VendorQualcomm
	case 0x13B5:
		return VendorARM
	case 0x1010:
		return VendorImgTec
	case 0x1002, 0x1022:
		return VendorAMD
	case 0x10DE:
		return VendorNvidia
	case 0x8086:
		return VendorIntel
	case 0x106B:
		return VendorApple
	case 0x19E5:
		return VendorHuawei
	case 0x144D:
		return VendorSamsung
	case 0x10005: // VK_VENDOR_ID_MESA
		return VendorMesa
	}
	return VendorUnknown
}

// DriverInfo is what the deny list looks at: the properties of the physical device that the choice
// took.
//
// Flutter: impeller/renderer/backend/vulkan/driver_info_vk.h DriverInfoVK
type DriverInfo struct {
	// Name is deviceName: "Adreno (TM) 650", "Mali-G78", "PowerVR CXT 48-1536".
	Name string
	// VendorID, DeviceID and DriverVersion are the fields of VkPhysicalDeviceProperties.
	VendorID, DeviceID, DriverVersion uint32
	// APIVersion is the packed Vulkan version the device supports.
	APIVersion uint32
}

// adrenoOrder lists the Adreno models in the order of Impeller's AdrenoGPU enum, oldest first, so
// that "at most the 650" is a comparison of positions.
//
// Flutter: impeller/renderer/backend/vulkan/driver_info_vk.h AdrenoGPU
var adrenoOrder = []string{
	"504", "505", "506", "508", "509", "510", "512", "530", "540",
	"605", "608", "610", "612", "613", "615", "616", "618", "619L", "619", "620", "630",
	"640", "642L", "642", "643L", "644", "650", "660", "663", "675", "680", "685", "690", "695",
	"702", "710", "720", "725", "730", "732", "735", "740", "750", "X145", "X185",
}

// powerVRSeries lists the PowerVR series in the order of Impeller's PowerVRGPU enum, worst first.
//
// Flutter: impeller/renderer/backend/vulkan/driver_info_vk.h PowerVRGPU
var powerVRSeries = []string{"BXE", "BXM", "BXS", "BXT", "CXT", "DXT"}

const (
	pixel10DeviceID         = 0x71061212
	pixel10MinDriverVersion = 6794074 // PowerVR DDK 25.1
	vulkan13                = 0x00403000
)

// adrenoIndex is the position of the Adreno model in adrenoOrder, or -1 when the name is not
// "Adreno (TM) <model>" of a model Impeller knows (a newer, unknown model is not denied).
func adrenoIndex(name string) int {
	_, model, ok := strings.Cut(name, "Adreno (TM) ")
	if !ok {
		return -1
	}
	for i, m := range adrenoOrder {
		if m == model {
			return i
		}
	}
	return -1
}

// powerVRIndex is the position of the PowerVR series whose name the device name contains in
// powerVRSeries, or -1 for a name of no known series.
func powerVRIndex(name string) int {
	for i, s := range powerVRSeries {
		if strings.Contains(name, s) {
			return i
		}
	}
	return -1
}

// IsKnownBadDriver says whether the driver is on the list of drivers that Vulkan is not used with
// on Android: the Pixel 10 with a PowerVR driver older than 25.1, Adreno 650 and older, the Huawei
// Maleoon, Samsung's Xclipse below Vulkan 1.3 and PowerVR below the CXT series.
//
// Deviation: Impeller reads the PowerVR series only when the vendor is kPowerVR, a value that its
// IdentifyVendor never returns (the vendor ID 0x1010 is kImgTec), so its PowerVR rules never apply;
// here they apply to the vendor ID 0x1010, which is what they say. SPEC §12.10.
//
// Flutter: impeller/renderer/backend/vulkan/driver_info_vk.cc DriverInfoVK::IsKnownBadDriver
func (d DriverInfo) IsKnownBadDriver() (bad bool, reason string) {
	vendor := vendorOf(d.VendorID)
	if vendor == VendorImgTec && d.DeviceID == pixel10DeviceID && d.DriverVersion < pixel10MinDriverVersion {
		return true, "the Pixel 10 driver is older than PowerVR 25.1"
	}
	if vendor == VendorQualcomm {
		if i := adrenoIndex(d.Name); i >= 0 && i <= adrenoIndex("Adreno (TM) 650") {
			return true, "Adreno 650 and older need workarounds that Glutter does not have"
		}
	}
	switch vendor {
	case VendorHuawei:
		return true, "the Maleoon GPUs are known to misdraw"
	case VendorSamsung:
		if d.APIVersion < vulkan13 {
			return true, "the first Xclipse GPUs (below Vulkan 1.3) are known to misdraw"
		}
	case VendorImgTec:
		if i := powerVRIndex(d.Name); i >= 0 && i < powerVRIndex("CXT") {
			return true, "PowerVR below the CXT series has poor Vulkan drivers"
		}
	}
	return false, ""
}
