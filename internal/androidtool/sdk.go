package androidtool

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/rm4n0s/errors"
)

// The versions the installation instructions name (SPEC §24.1).
const (
	// NDKPackage is the sdkmanager package of the NDK that the tool was tested with.
	NDKPackage = "ndk;30.0.16248370"
	// BuildToolsPackage is the sdkmanager package of the build tools.
	BuildToolsPackage = "build-tools;37.0.0"
	// PlatformPackage is the sdkmanager package of the platform whose android.jar is built against.
	PlatformPackage = "platforms;android-37.2"
	// MinSDK is the minimum API level (D24).
	MinSDK = 26
	// TargetSDK is the target API level.
	TargetSDK = 37
)

// SDK is the Android SDK and NDK that a build uses.
//
// Flutter: no counterpart
type SDK struct {
	// Home is the SDK ($ANDROID_HOME).
	Home string
	// NDK is the NDK, $ANDROID_NDK_HOME or the newest ndk/<version> of the SDK.
	NDK string
	// BuildTools is the newest build-tools/<version> that has aapt2.
	BuildTools string
	// Platform is the newest platforms/android-<level> that has android.jar.
	Platform string
}

// AndroidJar is the platform's android.jar.
func (s *SDK) AndroidJar() string { return filepath.Join(s.Platform, "android.jar") }

// Tool is the path of a tool of the build tools: aapt2, zipalign, apksigner, d8.
func (s *SDK) Tool(name string) string { return filepath.Join(s.BuildTools, name) }

// Adb is the SDK's adb when it has one, else "adb" (found on the PATH when it runs).
func (s *SDK) Adb() string {
	p := filepath.Join(s.Home, "platform-tools", "adb")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return "adb"
}

// Clang is the NDK's clang for an ABI ("arm64-v8a" or "x86_64") at the minimum API level.
func (s *SDK) Clang(abi string) string {
	return filepath.Join(s.NDK, "toolchains", "llvm", "prebuilt", "linux-x86_64", "bin", ABITriple(abi)+strconv.Itoa(MinSDK)+"-clang")
}

// ABI describes one Android ABI: the name Android uses and the Go architecture.
//
// Flutter: no counterpart
type ABI struct {
	// Name is the directory under lib/ in the APK.
	Name string
	// GOARCH is the architecture Go builds for.
	GOARCH string
	// Short is the name the -abi flag takes.
	Short string
}

// ABIs are the supported ABIs (SPEC §24.1), in the order they are built.
var ABIs = []ABI{
	{Name: "arm64-v8a", GOARCH: "arm64", Short: "arm64"},
	{Name: "x86_64", GOARCH: "amd64", Short: "amd64"},
}

// ABITriple is the NDK's triple of an ABI name.
func ABITriple(abi string) string {
	if abi == "x86_64" {
		return "x86_64-linux-android"
	}
	return "aarch64-linux-android"
}

// FindSDK finds the SDK, the NDK, the build tools and the platform from the environment
// (getenv is os.Getenv, or a stand-in in tests). The first piece that is missing is a tagged
// error whose metadata "install" names the sdkmanager package to install.
//
// Flutter: no counterpart
func FindSDK(getenv func(string) string) (*SDK, error) {
	home, err := FindHome(getenv)
	if err != nil {
		return nil, err
	}
	sdk := &SDK{Home: home}
	if sdk.NDK, err = FindNDK(getenv); err != nil {
		return nil, err
	}
	if sdk.BuildTools, err = FindBuildTools(home); err != nil {
		return nil, err
	}
	if sdk.Platform, err = FindPlatform(home); err != nil {
		return nil, err
	}
	return sdk, nil
}

// FindHome is $ANDROID_HOME, else $ANDROID_SDK_ROOT. It fails with AndroidSDKNotFound when
// neither names a directory.
//
// Flutter: no counterpart
func FindHome(getenv func(string) string) (string, error) {
	for _, key := range []string{"ANDROID_HOME", "ANDROID_SDK_ROOT"} {
		if dir := getenv(key); dir != "" {
			if st, err := os.Stat(dir); err == nil && st.IsDir() {
				return dir, nil
			}
		}
	}
	return "", errors.New(TagAndroidSDKNotFound, "the Android SDK was not found: set ANDROID_HOME to it",
		"install", `unzip "Command line tools only" from developer.android.com into $ANDROID_HOME/cmdline-tools/latest`)
}

// FindNDK is $ANDROID_NDK_HOME, else the newest ndk/<version> of the SDK. It fails with
// AndroidNDKNotFound when there is none.
//
// Flutter: no counterpart
func FindNDK(getenv func(string) string) (string, error) {
	if dir := getenv("ANDROID_NDK_HOME"); dir != "" {
		if st, err := os.Stat(dir); err == nil && st.IsDir() {
			return dir, nil
		}
	}
	for _, key := range []string{"ANDROID_HOME", "ANDROID_SDK_ROOT"} {
		if home := getenv(key); home != "" {
			if dir := newest(filepath.Join(home, "ndk"), "", ""); dir != "" {
				return dir, nil
			}
		}
	}
	return "", errors.New(TagAndroidNDKNotFound, "the Android NDK was not found: set ANDROID_NDK_HOME, or install it in the SDK",
		"install", `sdkmanager "`+NDKPackage+`"`)
}

// FindBuildTools is the newest build-tools/<version> of the SDK that has aapt2. It fails with
// AndroidBuildToolsNotFound when there is none.
//
// Flutter: no counterpart
func FindBuildTools(home string) (string, error) {
	if dir := newest(filepath.Join(home, "build-tools"), "", "aapt2"); dir != "" {
		return dir, nil
	}
	return "", errors.New(TagAndroidBuildToolsNotFound, "the SDK has no build-tools with aapt2",
		"home", home, "install", `sdkmanager "`+BuildToolsPackage+`"`)
}

// FindPlatform is the newest platforms/android-<level> of the SDK that has android.jar. It
// fails with AndroidPlatformNotFound when there is none.
//
// Flutter: no counterpart
func FindPlatform(home string) (string, error) {
	if dir := newest(filepath.Join(home, "platforms"), "android-", "android.jar"); dir != "" {
		return dir, nil
	}
	return "", errors.New(TagAndroidPlatformNotFound, "the SDK has no platform with android.jar",
		"home", home, "install", `sdkmanager "`+PlatformPackage+`"`)
}

// newest is the subdirectory of dir with the highest version that holds the file `has`
// (any when empty). A name is "<prefix><number>(.<number>)*"; names that are not are skipped.
func newest(dir, prefix, has string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	type candidate struct {
		path    string
		version []int
	}
	var all []candidate
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), prefix) {
			continue
		}
		v, ok := parseVersion(strings.TrimPrefix(e.Name(), prefix))
		if !ok {
			continue
		}
		path := filepath.Join(dir, e.Name())
		if has != "" {
			if _, err := os.Stat(filepath.Join(path, has)); err != nil {
				continue
			}
		}
		all = append(all, candidate{path, v})
	}
	if len(all) == 0 {
		return ""
	}
	sort.Slice(all, func(i, j int) bool { return compareVersions(all[i].version, all[j].version) < 0 })
	return all[len(all)-1].path
}

// parseVersion reads "37.0.0" as [37 0 0].
func parseVersion(s string) ([]int, bool) {
	var v []int
	for _, part := range strings.Split(s, ".") {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return nil, false
		}
		v = append(v, n)
	}
	return v, len(v) > 0
}

// compareVersions is -1, 0 or 1; a missing part counts as 0.
func compareVersions(a, b []int) int {
	for i := 0; i < len(a) || i < len(b); i++ {
		var x, y int
		if i < len(a) {
			x = a[i]
		}
		if i < len(b) {
			y = b[i]
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// TagAndroidSDKNotFound is the tag of the error for a missing SDK ($ANDROID_HOME); the
	// metadata "install" says how to get it. SPEC §24.1.
	TagAndroidSDKNotFound = "AndroidSDKNotFound"
	// TagAndroidNDKNotFound is the tag of the error for a missing NDK; the metadata "install"
	// has the sdkmanager command.
	TagAndroidNDKNotFound = "AndroidNDKNotFound"
	// TagAndroidBuildToolsNotFound is the tag of the error for missing build tools; the
	// metadata "install" has the sdkmanager command.
	TagAndroidBuildToolsNotFound = "AndroidBuildToolsNotFound"
	// TagAndroidPlatformNotFound is the tag of the error for a missing platform; the metadata
	// "install" has the sdkmanager command.
	TagAndroidPlatformNotFound = "AndroidPlatformNotFound"
)
