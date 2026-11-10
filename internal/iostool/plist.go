package iostool

import (
	"bytes"
	"encoding/xml"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/rm4n0s/errors"
)

// The defaults of an app's identity and the lowest iOS that runs it (D36: iOS 15 is the minimum).
const (
	// DefaultVersion is the version of an app that does not say.
	//
	// Flutter: no counterpart
	DefaultVersion = "1.0.0"
	// MinOSVersion is the MinimumOSVersion of every bundle.
	//
	// Flutter: no counterpart
	MinOSVersion = "15.0"
)

// The orientations that an Info.plist can allow (the values of UISupportedInterfaceOrientations).
const (
	// OrientationPortrait is UIInterfaceOrientationPortrait.
	//
	// Flutter: no counterpart
	OrientationPortrait = "UIInterfaceOrientationPortrait"
	// OrientationPortraitUpsideDown is UIInterfaceOrientationPortraitUpsideDown.
	//
	// Flutter: no counterpart
	OrientationPortraitUpsideDown = "UIInterfaceOrientationPortraitUpsideDown"
	// OrientationLandscapeLeft is UIInterfaceOrientationLandscapeLeft.
	//
	// Flutter: no counterpart
	OrientationLandscapeLeft = "UIInterfaceOrientationLandscapeLeft"
	// OrientationLandscapeRight is UIInterfaceOrientationLandscapeRight.
	//
	// Flutter: no counterpart
	OrientationLandscapeRight = "UIInterfaceOrientationLandscapeRight"
)

// Info is what goes into the Info.plist of an app bundle.
//
// Flutter: no counterpart
type Info struct {
	// Executable is the name of the file in the bundle (CFBundleExecutable).
	Executable string
	// ID is the bundle identifier (CFBundleIdentifier), in reverse DNS form.
	ID string
	// Name is what the home screen calls the app (CFBundleName and CFBundleDisplayName).
	Name string
	// Version is CFBundleShortVersionString and CFBundleVersion.
	Version string
	// Simulator says that the bundle is for the Simulator (CFBundleSupportedPlatforms is iPhoneSimulator) and not for devices.
	Simulator bool
	// Orientations are the orientations the app supports, the values of the Orientation constants; all four on an iPad and the
	// first three on an iPhone when empty.
	Orientations []string
	// Icons says that the bundle has the icon files that Icons makes, and names them in the plist (Write sets it).
	Icons bool
	// Usage are the usage strings that the system shows when the app first asks for something: the keys are those of the plist
	// ("NSMicrophoneUsageDescription"), the values the text.
	Usage map[string]string
}

// DefaultUsage are the usage strings that every bundle has: the system asks for the microphone and the motion sensors only when an
// app uses them, and shows the string then.
//
// Flutter: no counterpart
func DefaultUsage(name string) map[string]string {
	return map[string]string{
		"NSMicrophoneUsageDescription": name + " records sound when you use a feature that needs the microphone.",
		"NSMotionUsageDescription":     name + " reads the motion sensors when you use a feature that needs them.",
	}
}

var idPattern = regexp.MustCompile(`^[A-Za-z0-9-]+(\.[A-Za-z0-9-]+)+$`)

// DefaultID is the identifier of an app that has none: "dev.glutter." and the name in letters, digits and hyphens.
//
// Flutter: no counterpart
func DefaultID(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
			b.WriteRune(r)
		case r == ' ' || r == '_' || r == '.':
			b.WriteByte('-')
		}
	}
	id := strings.Trim(b.String(), "-")
	if id == "" {
		id = "app"
	}
	return "dev.glutter." + id
}

// InfoPlist writes the Info.plist of the bundle as XML, with the keys in a fixed order (so a build is the same file every time) and
// the usage strings sorted by key. The launch screen is a dictionary (UILaunchScreen), so that no storyboard needs compiling; the
// scene manifest has no configurations, because the app delegate answers for the scene (SPEC §30.2); the icons are the PNG files that
// Icons makes, named in CFBundleIcons. It fails with IOSBundleInvalid for an identity that a bundle cannot have.
//
// Flutter: no counterpart
func InfoPlist(i Info) ([]byte, error) {
	switch {
	case i.Executable == "" || strings.ContainsAny(i.Executable, "/\x00"):
		return nil, errors.New(tagIOSBundleInvalid, "the executable needs a name without a slash", "executable", i.Executable)
	case i.Name == "" || strings.ContainsAny(i.Name, "/:\x00"):
		return nil, errors.New(tagIOSBundleInvalid, "the name of the app is empty or has a slash or a colon", "name", i.Name)
	case !idPattern.MatchString(i.ID):
		return nil, errors.New(tagIOSBundleInvalid, "the bundle identifier is letters, digits and hyphens in at least two parts divided by dots", "id", i.ID)
	case i.Version == "" || strings.ContainsAny(i.Version, "<>&\x00"):
		return nil, errors.New(tagIOSBundleInvalid, "the version is empty or has a character that a plist cannot hold", "version", i.Version)
	}
	for _, o := range i.Orientations {
		switch o {
		case OrientationPortrait, OrientationPortraitUpsideDown, OrientationLandscapeLeft, OrientationLandscapeRight:
		default:
			return nil, errors.New(tagIOSBundleInvalid, "an orientation is not one of the four of UIKit", "orientation", o)
		}
	}
	phone := i.Orientations
	if len(phone) == 0 {
		phone = []string{OrientationPortrait, OrientationLandscapeLeft, OrientationLandscapeRight}
	}
	pad := i.Orientations
	if len(pad) == 0 {
		pad = []string{OrientationPortrait, OrientationPortraitUpsideDown, OrientationLandscapeLeft, OrientationLandscapeRight}
	}
	var b plistWriter
	b.begin()
	b.str("CFBundleDevelopmentRegion", "en")
	b.str("CFBundleExecutable", i.Executable)
	b.str("CFBundleIdentifier", i.ID)
	b.str("CFBundleInfoDictionaryVersion", "6.0")
	b.str("CFBundleName", i.Name)
	b.str("CFBundleDisplayName", i.Name)
	b.str("CFBundlePackageType", "APPL")
	b.str("CFBundleShortVersionString", i.Version)
	b.str("CFBundleVersion", i.Version)
	if i.Simulator {
		b.array("CFBundleSupportedPlatforms", "iPhoneSimulator")
	} else {
		b.array("CFBundleSupportedPlatforms", "iPhoneOS")
	}
	b.str("MinimumOSVersion", MinOSVersion)
	b.ints("UIDeviceFamily", 1, 2)
	b.boolean("LSRequiresIPhoneOS", true)
	b.boolean("CADisableMinimumFrameDurationOnPhone", true)
	b.boolean("UIApplicationSupportsIndirectInputEvents", true)
	b.boolean("UIStatusBarHidden", false)
	b.str("UIStatusBarStyle", "UIStatusBarStyleDefault")
	b.emptyDict("UILaunchScreen")
	b.key("UIApplicationSceneManifest")
	b.buf.WriteString("<dict>\n")
	b.boolean("UIApplicationSupportsMultipleScenes", false)
	b.emptyDict("UISceneConfigurations")
	b.buf.WriteString("</dict>\n")
	b.array("UISupportedInterfaceOrientations", phone...)
	b.array("UISupportedInterfaceOrientations~ipad", pad...)
	if i.Icons {
		b.icons("CFBundleIcons", IconBase+"60x60")
		b.icons("CFBundleIcons~ipad", IconBase+"60x60", IconBase+"76x76", IconBase+"83.5x83.5")
	}
	keys := make([]string, 0, len(i.Usage))
	for k := range i.Usage {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b.str(k, i.Usage[k])
	}
	b.end()
	return b.bytes(), nil
}

// Entitlements writes the entitlements file of a build for a device: the application identifier (the team and the bundle
// identifier) and the permission to be debugged, which a development profile gives.
//
// Flutter: no counterpart
func Entitlements(team, id string) []byte {
	var b plistWriter
	b.begin()
	b.str("application-identifier", team+"."+id)
	b.str("com.apple.developer.team-identifier", team)
	b.boolean("get-task-allow", true)
	b.end()
	return b.bytes()
}

// plistWriter writes an XML property list.
type plistWriter struct{ buf bytes.Buffer }

func (w *plistWriter) begin() {
	w.buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	w.buf.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n")
	w.buf.WriteString(`<plist version="1.0">` + "\n<dict>\n")
}

func (w *plistWriter) end() { w.buf.WriteString("</dict>\n</plist>\n") }

func (w *plistWriter) bytes() []byte { return w.buf.Bytes() }

func (w *plistWriter) key(k string) { w.buf.WriteString("\t<key>" + escape(k) + "</key>\n") }

func (w *plistWriter) str(k, v string) {
	w.key(k)
	w.buf.WriteString("\t<string>" + escape(v) + "</string>\n")
}

func (w *plistWriter) boolean(k string, v bool) {
	w.key(k)
	if v {
		w.buf.WriteString("\t<true/>\n")
		return
	}
	w.buf.WriteString("\t<false/>\n")
}

func (w *plistWriter) emptyDict(k string) {
	w.key(k)
	w.buf.WriteString("\t<dict/>\n")
}

func (w *plistWriter) array(k string, v ...string) {
	w.key(k)
	w.buf.WriteString("\t<array>\n")
	for _, s := range v {
		w.buf.WriteString("\t\t<string>" + escape(s) + "</string>\n")
	}
	w.buf.WriteString("\t</array>\n")
}

func (w *plistWriter) ints(k string, v ...int) {
	w.key(k)
	w.buf.WriteString("\t<array>\n")
	for _, n := range v {
		w.buf.WriteString("\t\t<integer>" + strconv.Itoa(n) + "</integer>\n")
	}
	w.buf.WriteString("\t</array>\n")
}

// icons writes CFBundleIcons: a dictionary whose primary icon lists the files.
func (w *plistWriter) icons(k string, files ...string) {
	w.key(k)
	w.buf.WriteString("\t<dict>\n\t\t<key>CFBundlePrimaryIcon</key>\n\t\t<dict>\n\t\t\t<key>CFBundleIconFiles</key>\n\t\t\t<array>\n")
	for _, f := range files {
		w.buf.WriteString("\t\t\t\t<string>" + escape(f) + "</string>\n")
	}
	w.buf.WriteString("\t\t\t</array>\n\t\t\t<key>CFBundleIconName</key>\n\t\t\t<string>" + IconBase + "</string>\n\t\t</dict>\n\t</dict>\n")
}

func escape(s string) string {
	var b bytes.Buffer
	// EscapeText only fails when the writer does, and a bytes.Buffer never does.
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagIOSBundleInvalid = TagIOSBundleInvalid

	// TagIOSBundleInvalid is the tag of the error for an app identity that a bundle cannot have: an identifier that is not in reverse
	// DNS form, a name with a slash, an empty version, an orientation that UIKit does not have, or a bundle that cannot be written.
	//
	// Flutter: no counterpart
	TagIOSBundleInvalid = "IOSBundleInvalid"
)
