package mactool

import (
	"bytes"
	"encoding/xml"
	"regexp"
	"sort"
	"strings"

	"github.com/rm4n0s/errors"
)

// The defaults of an app's identity and the lowest macOS that runs it (D34: macOS 12 is the minimum).
const (
	// DefaultVersion is the version of an app that does not say.
	//
	// Flutter: no counterpart
	DefaultVersion = "1.0.0"
	// MinSystemVersion is the LSMinimumSystemVersion of every bundle.
	//
	// Flutter: no counterpart
	MinSystemVersion = "12.0"
)

// Info is what goes into the Info.plist of an app bundle.
//
// Flutter: no counterpart
type Info struct {
	// Executable is the name of the file in Contents/MacOS (CFBundleExecutable).
	Executable string
	// ID is the bundle identifier (CFBundleIdentifier), in reverse DNS form.
	ID string
	// Name is what the Dock and the menu bar call the app (CFBundleName and CFBundleDisplayName).
	Name string
	// Version is CFBundleShortVersionString and CFBundleVersion.
	Version string
	// IconFile is the name of the icon file in Contents/Resources (CFBundleIconFile); empty for none.
	IconFile string
	// Usage are the usage strings that the system shows when the app first asks for something: the keys are those of the plist
	// ("NSMicrophoneUsageDescription"), the values the text.
	Usage map[string]string
}

// DefaultUsage are the usage strings that every bundle has: the system asks for the microphone only when an app records, and
// shows the string then.
//
// Flutter: no counterpart
func DefaultUsage(name string) map[string]string {
	return map[string]string{"NSMicrophoneUsageDescription": name + " records sound when you use a feature that needs the microphone."}
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

// InfoPlist writes the Info.plist of the bundle as XML, with the keys in a fixed order (so a build is the same file every time)
// and the usage strings sorted by key. It fails with MacOSBundleInvalid for an identity that a bundle cannot have.
//
// Flutter: no counterpart
func InfoPlist(i Info) ([]byte, error) {
	switch {
	case i.Executable == "" || strings.ContainsAny(i.Executable, "/\x00"):
		return nil, errors.New(tagMacOSBundleInvalid, "the executable needs a name without a slash", "executable", i.Executable)
	case i.Name == "" || strings.ContainsAny(i.Name, "/:\x00"):
		return nil, errors.New(tagMacOSBundleInvalid, "the name of the app is empty or has a slash or a colon", "name", i.Name)
	case !idPattern.MatchString(i.ID):
		return nil, errors.New(tagMacOSBundleInvalid, "the bundle identifier is letters, digits and hyphens in at least two parts divided by dots", "id", i.ID)
	case i.Version == "" || strings.ContainsAny(i.Version, "<>&\x00"):
		return nil, errors.New(tagMacOSBundleInvalid, "the version is empty or has a character that a plist cannot hold", "version", i.Version)
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
	if i.IconFile != "" {
		b.str("CFBundleIconFile", i.IconFile)
	}
	b.str("LSMinimumSystemVersion", MinSystemVersion)
	b.str("NSPrincipalClass", "NSApplication")
	b.boolean("NSHighResolutionCapable", true)
	b.boolean("NSSupportsAutomaticGraphicsSwitching", true)
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

// Entitlements writes the entitlements file that the hardened runtime needs for what the app does: the microphone.
//
// Flutter: no counterpart
func Entitlements() []byte {
	var b plistWriter
	b.begin()
	b.boolean("com.apple.security.device.audio-input", true)
	b.end()
	return b.bytes()
}

// plistWriter writes a flat XML property list.
type plistWriter struct{ buf bytes.Buffer }

func (w *plistWriter) begin() {
	w.buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
`)
}

func (w *plistWriter) end() { w.buf.WriteString("</dict>\n</plist>\n") }

func (w *plistWriter) bytes() []byte { return w.buf.Bytes() }

func (w *plistWriter) key(k string) {
	w.buf.WriteString("\t<key>")
	w.text(k)
	w.buf.WriteString("</key>\n")
}

func (w *plistWriter) str(k, v string) {
	w.key(k)
	w.buf.WriteString("\t<string>")
	w.text(v)
	w.buf.WriteString("</string>\n")
}

func (w *plistWriter) boolean(k string, v bool) {
	w.key(k)
	if v {
		w.buf.WriteString("\t<true/>\n")
	} else {
		w.buf.WriteString("\t<false/>\n")
	}
}

// text writes escaped text. EscapeText cannot fail on a bytes.Buffer.
func (w *plistWriter) text(s string) {
	_ = xml.EscapeText(&w.buf, []byte(s))
}

// The tags of the errors of this package (rm4n0s/errors, R5).
const tagMacOSBundleInvalid = TagMacOSBundleInvalid

// TagMacOSBundleInvalid is the tag of the error for a name, identifier or version that a bundle cannot have, a layout that cannot be
// written, or a bundle with a problem.
//
// Flutter: no counterpart
const TagMacOSBundleInvalid = "MacOSBundleInvalid"
