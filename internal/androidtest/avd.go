package androidtest

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rm4n0s/errors"
)

// AVD describes a virtual device the harness makes.
//
// Flutter: no counterpart
type AVD struct {
	// Name is the name of the device, such as "glutter-37".
	Name string
	// Package is the sdkmanager package of its system image, such as
	// "system-images;android-37.2;google_apis_ps16k;x86_64".
	Package string
	// API is the Android API level of the image, such as "37".
	API string
}

// AVDs are the devices the device tests use (SPEC §24.13), by name.
//
// Deviation: SPEC §24.13 names `system-images;android-37.0;google_apis;x86_64` for glutter-37,
// but on the emulator 37.2.12 that image does not boot: the system server crashes in a loop with
// "Invalid feature flag: android.security.aapm_feature_disable_insecure_wifi_autojoin" in
// AdvancedProtectionService (checked on 2026-10-24). Both API 37 devices use the 16 KB image.
//
// Flutter: no counterpart
var AVDs = map[string]AVD{
	"glutter-37":     {Name: "glutter-37", Package: "system-images;android-37.2;google_apis_ps16k;x86_64", API: "37"},
	"glutter-37-16k": {Name: "glutter-37-16k", Package: "system-images;android-37.2;google_apis_ps16k;x86_64", API: "37"},
	"glutter-26":     {Name: "glutter-26", Package: "system-images;android-26;google_apis;x86_64", API: "26"},
}

// sysdir is the image's directory below the SDK: "system-images/android-37.2/google_apis_ps16k/x86_64/".
func (a AVD) sysdir() string { return strings.ReplaceAll(a.Package, ";", "/") + "/" }

// tag is the image's tag id, "google_apis_ps16k".
func (a AVD) tag() string { return strings.Split(a.Package, ";")[2] }

// target is "android-37.2".
func (a AVD) target() string { return strings.Split(a.Package, ";")[1] }

// avdConfig is config.ini of a virtual device: a Pixel-like phone with a 1080x2400 screen,
// 420 dpi and touch, the keyboard off (so that the soft keyboard shows), no SD card and no
// device frame.
func (a AVD) avdConfig() string {
	kv := map[string]string{
		"AvdId":                     a.Name,
		"PlayStore.enabled":         "no",
		"abi.type":                  "x86_64",
		"avd.ini.displayname":       a.Name,
		"avd.ini.encoding":          "UTF-8",
		"disk.dataPartition.size":   "6G",
		"hw.accelerometer":          "yes",
		"hw.audioInput":             "no",
		"hw.audioOutput":            "no",
		"hw.battery":                "yes",
		"hw.camera.back":            "none",
		"hw.camera.front":           "none",
		"hw.cpu.arch":               "x86_64",
		"hw.cpu.ncore":              "4",
		"hw.dPad":                   "no",
		"hw.device.manufacturer":    "Google",
		"hw.device.name":            "pixel_6",
		"hw.gps":                    "no",
		"hw.gpu.enabled":            "yes",
		"hw.gpu.mode":               "auto",
		"hw.initialOrientation":     "portrait",
		"hw.keyboard":               "no",
		"hw.lcd.density":            "420",
		"hw.lcd.height":             "2400",
		"hw.lcd.width":              "1080",
		"hw.mainKeys":               "no",
		"hw.ramSize":                "3072",
		"hw.sdCard":                 "no",
		"hw.sensors.orientation":    "yes",
		"hw.sensors.proximity":      "no",
		"hw.trackBall":              "no",
		"image.sysdir.1":            a.sysdir(),
		"runtime.network.latency":   "none",
		"runtime.network.speed":     "full",
		"showDeviceFrame":           "no",
		"skin.dynamic":              "yes",
		"tag.display":               "Google APIs",
		"tag.id":                    a.tag(),
		"target":                    a.target(),
		"vm.heapSize":               "256",
		"fastboot.forceColdBoot":    "yes",
		"hw.camera.virtualscene":    "no",
		"hw.gltransport":            "pipe",
		"hw.lcd.vsync":              "60",
		"kernel.newDeviceNaming":    "autodetect",
		"kernel.supportsYaffs2":     "autodetect",
		"userdata.useQcow2":         "no",
		"test.monitorAdb":           "0",
		"test.quitAfterBootTimeOut": "-1",
	}
	keys := make([]string, 0, len(kv))
	for k := range kv {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s=%s\n", k, kv[k])
	}
	return b.String()
}

// ensureAVD writes the virtual device below root/avd if it is not there: <name>.ini and
// <name>.avd/config.ini. The emulator makes the rest at its first start. It fails with
// AndroidTestImageMissing when the system image is not installed in the SDK.
func ensureAVD(home, root string, a AVD) error {
	if _, err := os.Stat(filepath.Join(home, a.sysdir(), "system.img")); err != nil {
		return errors.NewErr(TagAndroidTestImageMissing, err).SetMetadata("image", a.Package, "install", `sdkmanager "`+a.Package+`"`)
	}
	dir := filepath.Join(root, "avd", a.Name+".avd")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return errors.NewErr(TagAndroidTestAVDFailed, err).SetMetadata("dir", dir)
	}
	ini := fmt.Sprintf("avd.ini.encoding=UTF-8\npath=%s\npath.rel=avd/%s.avd\ntarget=%s\n", dir, a.Name, a.target())
	if err := os.WriteFile(filepath.Join(root, "avd", a.Name+".ini"), []byte(ini), 0o644); err != nil {
		return errors.NewErr(TagAndroidTestAVDFailed, err).SetMetadata("avd", a.Name)
	}
	cfg := filepath.Join(dir, "config.ini")
	if err := os.WriteFile(cfg, []byte(a.avdConfig()), 0o644); err != nil {
		return errors.NewErr(TagAndroidTestAVDFailed, err).SetMetadata("path", cfg)
	}
	return nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// TagAndroidTestImageMissing is the tag of the error for a system image that is not
	// installed; the metadata "install" has the sdkmanager command.
	TagAndroidTestImageMissing = "AndroidTestImageMissing"
	// TagAndroidTestAVDFailed is the tag of the error for a virtual device that cannot be written.
	TagAndroidTestAVDFailed = "AndroidTestAVDFailed"
)
