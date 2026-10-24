package androidtool

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
)

// ManifestConfig is what the generated AndroidManifest.xml says (SPEC §24.1).
//
// Flutter: no counterpart
type ManifestConfig struct {
	// AppID is the application ID, such as "org.glutter.counter".
	AppID string
	// Label is the name under the icon.
	Label string
	// Permissions are the permissions the app asks for, such as "android.permission.INTERNET".
	Permissions []string
	// Debuggable sets android:debuggable, which adb's run-as needs; debug builds set it.
	Debuggable bool
	// HasIcon says that the APK has the resource @mipmap/ic_launcher.
	HasIcon bool
}

// ConfigChanges are the configuration changes the activity handles itself, so that turning the
// device or changing the theme does not recreate it: Flutter's list.
//
// Flutter: shell/platform/android/io/flutter/embedding/android/FlutterActivity.java (the manifest meta-data it documents)
const ConfigChanges = "orientation|keyboardHidden|keyboard|screenSize|smallestScreenSize|locale|layoutDirection|fontScale|screenLayout|density|uiMode"

// ActivityClass is the activity of the embedding.
const ActivityClass = "io.github.rm4n0s.glutter.GlutterActivity"

// ReceiverClass is the broadcast receiver of the embedding: it hears the swipe of a notification
// (SPEC §24.11). It is not exported, so only the system's PendingIntent reaches it.
const ReceiverClass = "io.github.rm4n0s.glutter.NotificationReceiver"

// DefaultPermissions are the permissions every generated manifest asks for, because a service of the
// toolkit needs them: POST_NOTIFICATIONS, the runtime permission of Android 13 and later that
// desktop.ShowNotification asks the user for (SPEC §24.11). Permissions in ManifestConfig are added to
// them.
//
// Flutter: no counterpart
var DefaultPermissions = []string{"android.permission.POST_NOTIFICATIONS"}

// permissions returns the default permissions and then the config's own, each once.
func (c ManifestConfig) permissions() []string {
	var out []string
	seen := map[string]bool{}
	for _, p := range append(append([]string(nil), DefaultPermissions...), c.Permissions...) {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}

// Manifest returns the text of AndroidManifest.xml: one activity, the receiver of the notifications'
// delete intents, the minimum and target API levels of D24, and the permissions asked for (the
// DefaultPermissions and the config's).
//
// Flutter: no counterpart
func Manifest(c ManifestConfig) []byte {
	var b bytes.Buffer
	esc := func(s string) string {
		var q bytes.Buffer
		_ = xml.EscapeText(&q, []byte(s)) // a bytes.Buffer never fails to write
		return q.String()                 // EscapeText also escapes the quotes, as &#34;
	}
	fmt.Fprintf(&b, "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n")
	fmt.Fprintf(&b, "<manifest xmlns:android=\"http://schemas.android.com/apk/res/android\"\n")
	fmt.Fprintf(&b, "    package=\"%s\"\n", esc(c.AppID))
	fmt.Fprintf(&b, "    android:versionCode=\"1\"\n")
	fmt.Fprintf(&b, "    android:versionName=\"1.0\">\n")
	fmt.Fprintf(&b, "  <uses-sdk android:minSdkVersion=\"%d\" android:targetSdkVersion=\"%d\" />\n", MinSDK, TargetSDK)
	for _, p := range c.permissions() {
		fmt.Fprintf(&b, "  <uses-permission android:name=\"%s\" />\n", esc(p))
	}
	fmt.Fprintf(&b, "  <application\n")
	fmt.Fprintf(&b, "      android:label=\"%s\"\n", esc(c.Label))
	if c.HasIcon {
		fmt.Fprintf(&b, "      android:icon=\"@mipmap/ic_launcher\"\n")
	}
	fmt.Fprintf(&b, "      android:hasCode=\"true\"\n")
	fmt.Fprintf(&b, "      android:extractNativeLibs=\"false\"\n")
	fmt.Fprintf(&b, "      android:debuggable=\"%t\"\n", c.Debuggable)
	fmt.Fprintf(&b, "      android:allowBackup=\"false\"\n")
	fmt.Fprintf(&b, "      android:theme=\"@android:style/Theme.Material.NoActionBar\">\n")
	fmt.Fprintf(&b, "    <activity\n")
	fmt.Fprintf(&b, "        android:name=\"%s\"\n", ActivityClass)
	fmt.Fprintf(&b, "        android:exported=\"true\"\n")
	fmt.Fprintf(&b, "        android:launchMode=\"singleTop\"\n")
	fmt.Fprintf(&b, "        android:configChanges=\"%s\"\n", ConfigChanges)
	fmt.Fprintf(&b, "        android:windowSoftInputMode=\"adjustResize\"\n")
	fmt.Fprintf(&b, "        android:hardwareAccelerated=\"true\">\n")
	fmt.Fprintf(&b, "      <intent-filter>\n")
	fmt.Fprintf(&b, "        <action android:name=\"android.intent.action.MAIN\" />\n")
	fmt.Fprintf(&b, "        <category android:name=\"android.intent.category.LAUNCHER\" />\n")
	fmt.Fprintf(&b, "      </intent-filter>\n")
	fmt.Fprintf(&b, "    </activity>\n")
	fmt.Fprintf(&b, "    <receiver\n")
	fmt.Fprintf(&b, "        android:name=\"%s\"\n", ReceiverClass)
	fmt.Fprintf(&b, "        android:exported=\"false\" />\n")
	fmt.Fprintf(&b, "  </application>\n")
	fmt.Fprintf(&b, "</manifest>\n")
	return b.Bytes()
}

// DefaultAppID derives an application ID from a package path: "./examples/dev/android-hello"
// is "org.glutter.android_hello". Characters that an ID cannot have become underscores, and
// a segment that does not start with a letter gets a leading "a".
//
// Flutter: no counterpart
func DefaultAppID(pkg string) string {
	base := pkg
	if i := strings.LastIndexAny(pkg, `/\`); i >= 0 {
		base = pkg[i+1:]
	}
	var b strings.Builder
	for _, r := range strings.ToLower(base) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	name := b.String()
	if name == "" || name == "_" {
		name = "app"
	}
	if name[0] < 'a' || name[0] > 'z' {
		name = "a" + name
	}
	return "org.glutter." + name
}

// DefaultLabel is the base name of a package path, such as "android-hello".
//
// Flutter: no counterpart
func DefaultLabel(pkg string) string {
	if i := strings.LastIndexAny(pkg, `/\`); i >= 0 {
		pkg = pkg[i+1:]
	}
	if pkg == "" || pkg == "." {
		return "Glutter"
	}
	return pkg
}
