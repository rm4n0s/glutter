package androidtool

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rm4n0s/errors"
)

// sdkTree makes a temporary SDK with the given files (paths below the SDK).
func sdkTree(t *testing.T, files ...string) string {
	t.Helper()
	home := t.TempDir()
	for _, f := range files {
		p := filepath.Join(home, f)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, nil, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return home
}

func TestFindSDK(t *testing.T) {
	full := []string{"ndk/29.0.1/source.properties", "ndk/30.0.16248370/source.properties", "ndk/9.0.0/source.properties",
		"build-tools/36.0.0/aapt2", "build-tools/37.0.0/aapt2", "build-tools/37.0.0-rc1/x",
		"platforms/android-36/android.jar", "platforms/android-37.2/android.jar", "platforms/android-38/empty"}
	tests := []struct {
		name    string
		files   []string
		env     func(home string) map[string]string
		wantTag string
		check   func(t *testing.T, home string, s *SDK)
	}{
		{name: "the newest of each", files: full, env: func(h string) map[string]string { return map[string]string{"ANDROID_HOME": h} },
			check: func(t *testing.T, h string, s *SDK) {
				if s.NDK != filepath.Join(h, "ndk", "30.0.16248370") {
					t.Errorf("NDK = %s", s.NDK)
				}
				if s.BuildTools != filepath.Join(h, "build-tools", "37.0.0") {
					t.Errorf("BuildTools = %s", s.BuildTools)
				}
				if s.Platform != filepath.Join(h, "platforms", "android-37.2") {
					t.Errorf("Platform = %s (android-38 has no android.jar)", s.Platform)
				}
			}},
		{name: "ANDROID_SDK_ROOT", files: full, env: func(h string) map[string]string { return map[string]string{"ANDROID_SDK_ROOT": h} }},
		{name: "ANDROID_NDK_HOME wins", files: full, env: func(h string) map[string]string {
			return map[string]string{"ANDROID_HOME": h, "ANDROID_NDK_HOME": filepath.Join(h, "ndk", "29.0.1")}
		}, check: func(t *testing.T, h string, s *SDK) {
			if s.NDK != filepath.Join(h, "ndk", "29.0.1") {
				t.Errorf("NDK = %s", s.NDK)
			}
		}},
		{name: "no SDK", env: func(string) map[string]string { return nil }, wantTag: TagAndroidSDKNotFound},
		{name: "an SDK that is not a directory", env: func(h string) map[string]string { return map[string]string{"ANDROID_HOME": filepath.Join(h, "gone")} }, wantTag: TagAndroidSDKNotFound},
		{name: "no NDK", files: []string{"build-tools/37.0.0/aapt2", "platforms/android-37.2/android.jar"},
			env: func(h string) map[string]string { return map[string]string{"ANDROID_HOME": h} }, wantTag: TagAndroidNDKNotFound},
		{name: "no build-tools", files: []string{"ndk/30.0.1/x", "platforms/android-37.2/android.jar"},
			env: func(h string) map[string]string { return map[string]string{"ANDROID_HOME": h} }, wantTag: TagAndroidBuildToolsNotFound},
		{name: "build-tools without aapt2", files: []string{"ndk/30.0.1/x", "build-tools/37.0.0/zipalign", "platforms/android-37.2/android.jar"},
			env: func(h string) map[string]string { return map[string]string{"ANDROID_HOME": h} }, wantTag: TagAndroidBuildToolsNotFound},
		{name: "no platform", files: []string{"ndk/30.0.1/x", "build-tools/37.0.0/aapt2"},
			env: func(h string) map[string]string { return map[string]string{"ANDROID_HOME": h} }, wantTag: TagAndroidPlatformNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := sdkTree(t, tt.files...)
			env := tt.env(home)
			got, err := FindSDK(func(k string) string { return env[k] })
			if tt.wantTag != "" {
				appErr, ok := errors.FromError(err)
				if !ok {
					t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
				}
				if !appErr.HasRoute("." + tt.wantTag) {
					t.Errorf("route %s, want it to end with %s", appErr.Route(), tt.wantTag)
				}
				if s, _ := appErr.Metadata["install"].(string); s == "" {
					t.Errorf("the error has no install command: %v", appErr.Metadata)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tt.check != nil {
				tt.check(t, home, got)
			}
		})
	}
}

func TestFindRoutes(t *testing.T) {
	_, err := FindHome(func(string) string { return "" })
	if appErr, _ := errors.FromError(err); appErr == nil || !appErr.HasRoute("FindHome."+TagAndroidSDKNotFound) {
		t.Errorf("FindHome: %v", err)
	}
	_, err = FindNDK(func(string) string { return "" })
	if appErr, _ := errors.FromError(err); appErr == nil || !appErr.HasRoute("FindNDK."+TagAndroidNDKNotFound) {
		t.Errorf("FindNDK: %v", err)
	}
	home := t.TempDir()
	_, err = FindBuildTools(home)
	if appErr, _ := errors.FromError(err); appErr == nil || !appErr.HasRoute("FindBuildTools."+TagAndroidBuildToolsNotFound) {
		t.Errorf("FindBuildTools: %v", err)
	}
	_, err = FindPlatform(home)
	if appErr, _ := errors.FromError(err); appErr == nil || !appErr.HasRoute("FindPlatform."+TagAndroidPlatformNotFound) {
		t.Errorf("FindPlatform: %v", err)
	}
}

func TestVersionOrder(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{{"37.0.0", "36.9.9", 1}, {"30.0.16248370", "30.0.2", 1}, {"37", "37.0.0", 0}, {"9.0", "10.0", -1}, {"37.2", "37.0", 1}}
	for _, tt := range tests {
		a, _ := parseVersion(tt.a)
		b, _ := parseVersion(tt.b)
		if got := compareVersions(a, b); got != tt.want {
			t.Errorf("compare(%s, %s) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
	if _, ok := parseVersion("37.0.0-rc1"); ok {
		t.Error("a release candidate suffix should not parse")
	}
}

func TestClangAndTools(t *testing.T) {
	s := &SDK{Home: "/h", NDK: "/ndk", BuildTools: "/bt", Platform: "/p"}
	if got, want := s.Clang("arm64-v8a"), "/ndk/toolchains/llvm/prebuilt/linux-x86_64/bin/aarch64-linux-android26-clang"; got != want {
		t.Errorf("Clang(arm64-v8a) = %s, want %s", got, want)
	}
	if got, want := s.Clang("x86_64"), "/ndk/toolchains/llvm/prebuilt/linux-x86_64/bin/x86_64-linux-android26-clang"; got != want {
		t.Errorf("Clang(x86_64) = %s, want %s", got, want)
	}
	if s.AndroidJar() != "/p/android.jar" || s.Tool("aapt2") != "/bt/aapt2" {
		t.Errorf("AndroidJar %s, Tool %s", s.AndroidJar(), s.Tool("aapt2"))
	}
}
