package androidtool

import (
	"strings"
	"testing"
)

func TestManifest(t *testing.T) {
	got := string(Manifest(ManifestConfig{AppID: "org.glutter.counter", Label: `Count & "go"`, Permissions: []string{"android.permission.INTERNET"}, Debuggable: true, HasIcon: true}))
	const want = `<?xml version="1.0" encoding="utf-8"?>
<manifest xmlns:android="http://schemas.android.com/apk/res/android"
    package="org.glutter.counter"
    android:versionCode="1"
    android:versionName="1.0">
  <uses-sdk android:minSdkVersion="26" android:targetSdkVersion="37" />
  <uses-permission android:name="android.permission.INTERNET" />
  <application
      android:label="Count &amp; &#34;go&#34;"
      android:icon="@mipmap/ic_launcher"
      android:hasCode="true"
      android:extractNativeLibs="false"
      android:debuggable="true"
      android:allowBackup="false"
      android:theme="@android:style/Theme.Material.NoActionBar">
    <activity
        android:name="io.github.rm4n0s.glutter.GlutterActivity"
        android:exported="true"
        android:launchMode="singleTop"
        android:configChanges="orientation|keyboardHidden|keyboard|screenSize|smallestScreenSize|locale|layoutDirection|fontScale|screenLayout|density|uiMode"
        android:windowSoftInputMode="adjustResize"
        android:hardwareAccelerated="true">
      <intent-filter>
        <action android:name="android.intent.action.MAIN" />
        <category android:name="android.intent.category.LAUNCHER" />
      </intent-filter>
    </activity>
  </application>
</manifest>
`
	if got != want {
		t.Errorf("manifest:\n%s\nwant:\n%s", got, want)
	}
}

func TestManifestRelease(t *testing.T) {
	got := string(Manifest(ManifestConfig{AppID: "a.b", Label: "x"}))
	for _, bad := range []string{"android:icon", "uses-permission"} {
		if strings.Contains(got, bad) {
			t.Errorf("the manifest has %q:\n%s", bad, got)
		}
	}
	if !strings.Contains(got, `android:debuggable="false"`) {
		t.Errorf("a release manifest is debuggable:\n%s", got)
	}
}

func TestDefaultAppIDAndLabel(t *testing.T) {
	tests := []struct{ pkg, id, label string }{
		{"./examples/dev/android-hello", "org.glutter.android_hello", "android-hello"},
		{"./examples/counter", "org.glutter.counter", "counter"},
		{"github.com/x/My.App", "org.glutter.my_app", "My.App"},
		{"./examples/3d", "org.glutter.a3d", "3d"},
		{".", "org.glutter.app", "Glutter"},
	}
	for _, tt := range tests {
		if got := DefaultAppID(tt.pkg); got != tt.id {
			t.Errorf("DefaultAppID(%q) = %q, want %q", tt.pkg, got, tt.id)
		}
		if got := DefaultLabel(tt.pkg); got != tt.label {
			t.Errorf("DefaultLabel(%q) = %q, want %q", tt.pkg, got, tt.label)
		}
		if id := DefaultAppID(tt.pkg); !ApplicationIDValid(id) {
			t.Errorf("DefaultAppID(%q) = %q is not valid", tt.pkg, id)
		}
	}
	for _, bad := range []string{"", "single", "a..b", "1a.b", "a.b-c", "a.b c"} {
		if ApplicationIDValid(bad) {
			t.Errorf("ApplicationIDValid(%q) = true", bad)
		}
	}
}
