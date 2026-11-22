package mactool

import (
	"bytes"
	"encoding/xml"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/rm4n0s/errors"
)

// The defaults of an app's identity and the lowest macOS that runs it (D34: macOS 13 is the minimum since step 51.1, the
// go toolchain's: Go 1.27's linker builds its own objects for macOS 13.0).
const (
	// DefaultVersion is the version of an app that does not say.
	//
	// Flutter: no counterpart
	DefaultVersion = "1.0.0"
	// MinSystemVersion is the LSMinimumSystemVersion of a bundle that names none (Info.MinSystemVersion), and the lowest
	// one a bundle can name.
	//
	// Flutter: no counterpart
	MinSystemVersion = "13.0"
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
	// MinSystemVersion is the oldest macOS that runs the app (LSMinimumSystemVersion), "major.minor" or
	// "major.minor.patch"; MinSystemVersion (the constant) when empty.
	MinSystemVersion string
	// URLSchemes are the URL schemes the app handles ("magnet"; CFBundleURLTypes): a click on such a link starts the app, or
	// reaches the running one, as a ui.OpenRequest. A scheme is letters, digits, "+", "-" and ".", and starts with a letter.
	URLSchemes []string
	// DocumentTypes are the types of the files the app opens (CFBundleDocumentTypes): a double click on such a file, or a drop
	// on the app's icon, starts the app, or reaches the running one, as a ui.OpenRequest.
	DocumentTypes []DocumentType
}

// DocumentType is a type of file that an app opens: one entry of the plist's CFBundleDocumentTypes.
//
// Flutter: no counterpart
type DocumentType struct {
	// Name is what the Finder calls a file of the type ("BitTorrent file"; CFBundleTypeName).
	Name string
	// Extensions are the file extensions of the type, without the dot ("torrent"; CFBundleTypeExtensions): letters, digits,
	// "_", "+" and "-". "*" alone is every file.
	Extensions []string
	// Role is what the app does with the file (CFBundleTypeRole): RoleViewer, also when empty, or RoleEditor.
	Role string
}

// The roles of a DocumentType.
const (
	// RoleViewer is the role of an app that reads the files of a type; it is also the role of the app for its URL schemes.
	//
	// Flutter: no counterpart
	RoleViewer = "Viewer"
	// RoleEditor is the role of an app that reads and writes the files of a type.
	//
	// Flutter: no counterpart
	RoleEditor = "Editor"
)

// DefaultUsage are the usage strings that every bundle has: the system asks for the microphone only when an app records, and
// shows the string then.
//
// Flutter: no counterpart
func DefaultUsage(name string) map[string]string {
	return map[string]string{"NSMicrophoneUsageDescription": name + " records sound when you use a feature that needs the microphone."}
}

var (
	idPattern      = regexp.MustCompile(`^[A-Za-z0-9-]+(\.[A-Za-z0-9-]+)+$`)
	versionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+(\.[0-9]+)?$`)
	// schemePattern is the scheme of RFC 3986; extensionPattern an extension without its dot, or "*" for every file.
	schemePattern    = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*$`)
	extensionPattern = regexp.MustCompile(`^([A-Za-z0-9_+-]+|\*)$`)
)

// badScheme is the first of the schemes that is not a URL scheme or is there twice (a scheme has no case), and whether there is one.
func badScheme(schemes []string) (string, bool) {
	seen := map[string]bool{}
	for _, s := range schemes {
		k := strings.ToLower(s)
		if !schemePattern.MatchString(s) || seen[k] {
			return s, true
		}
		seen[k] = true
	}
	return "", false
}

// badDocumentType says what is wrong with the first of the types that a plist cannot hold, and whether there is one.
func badDocumentType(types []DocumentType) (DocumentType, string, bool) {
	for _, t := range types {
		switch {
		case strings.TrimSpace(t.Name) == "" || strings.ContainsRune(t.Name, 0):
			return t, "a document type needs a name", true
		case len(t.Extensions) == 0:
			return t, "a document type needs a file extension", true
		case t.Role != "" && t.Role != RoleViewer && t.Role != RoleEditor:
			return t, "the role of a document type is " + RoleViewer + " or " + RoleEditor, true
		}
		for _, e := range t.Extensions {
			if !extensionPattern.MatchString(e) {
				return t, "a file extension is letters, digits, _, + and -, without the dot, or * for every file", true
			}
		}
	}
	return DocumentType{}, "", false
}

// versionAtLeast reports whether v is a version of macOS ("14.0", "13.5.1") that is not under floor.
func versionAtLeast(v, floor string) bool {
	if !versionPattern.MatchString(v) {
		return false
	}
	a, b := strings.Split(v, "."), strings.Split(floor, ".")
	for k := range max(len(a), len(b)) {
		x, y := 0, 0
		if k < len(a) {
			x, _ = strconv.Atoi(a[k])
		}
		if k < len(b) {
			y, _ = strconv.Atoi(b[k])
		}
		if x != y {
			return x > y
		}
	}
	return true
}

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
	case i.MinSystemVersion != "" && !versionAtLeast(i.MinSystemVersion, MinSystemVersion):
		return nil, errors.New(tagMacOSBundleInvalid, "the minimum macOS is a version such as 14.0, not under "+MinSystemVersion,
			"min", i.MinSystemVersion)
	}
	if s, bad := badScheme(i.URLSchemes); bad {
		return nil, errors.New(tagMacOSBundleInvalid, "a URL scheme is letters, digits, +, - and ., starts with a letter, and is named once", "scheme", s)
	}
	if t, why, bad := badDocumentType(i.DocumentTypes); bad {
		return nil, errors.New(tagMacOSBundleInvalid, why, "name", t.Name, "extensions", strings.Join(t.Extensions, ","), "role", t.Role)
	}
	minOS := i.MinSystemVersion
	if minOS == "" {
		minOS = MinSystemVersion
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
	b.str("LSMinimumSystemVersion", minOS)
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
	// The schemes are one URL type, named after the bundle; each document type is an entry of its own.
	if len(i.URLSchemes) > 0 {
		b.key("CFBundleURLTypes")
		b.open("array")
		b.open("dict")
		b.str("CFBundleTypeRole", RoleViewer)
		b.str("CFBundleURLName", i.ID)
		b.strs("CFBundleURLSchemes", i.URLSchemes)
		b.close("dict")
		b.close("array")
	}
	if len(i.DocumentTypes) > 0 {
		b.key("CFBundleDocumentTypes")
		b.open("array")
		for _, t := range i.DocumentTypes {
			role := t.Role
			if role == "" {
				role = RoleViewer
			}
			b.open("dict")
			b.strs("CFBundleTypeExtensions", t.Extensions)
			b.str("CFBundleTypeName", t.Name)
			b.str("CFBundleTypeRole", role)
			b.str("LSHandlerRank", "Default")
			b.close("dict")
		}
		b.close("array")
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

// plistWriter writes an XML property list: the keys of the top dictionary, and arrays and dictionaries inside it (open and
// close), each level one tab deeper.
type plistWriter struct {
	buf   bytes.Buffer
	depth int
}

func (w *plistWriter) begin() {
	w.buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
`)
	w.depth = 1
}

func (w *plistWriter) end() { w.buf.WriteString("</dict>\n</plist>\n") }

func (w *plistWriter) bytes() []byte { return w.buf.Bytes() }

func (w *plistWriter) indent() {
	for range w.depth {
		w.buf.WriteByte('\t')
	}
}

func (w *plistWriter) key(k string) {
	w.indent()
	w.buf.WriteString("<key>")
	w.text(k)
	w.buf.WriteString("</key>\n")
}

// value writes a string that is the value of a key or an element of an array.
func (w *plistWriter) value(v string) {
	w.indent()
	w.buf.WriteString("<string>")
	w.text(v)
	w.buf.WriteString("</string>\n")
}

func (w *plistWriter) str(k, v string) {
	w.key(k)
	w.value(v)
}

// strs writes a key whose value is an array of strings.
func (w *plistWriter) strs(k string, vs []string) {
	w.key(k)
	w.open("array")
	for _, v := range vs {
		w.value(v)
	}
	w.close("array")
}

// open begins an array or a dictionary (the element's name), and close ends it.
func (w *plistWriter) open(element string) {
	w.indent()
	w.buf.WriteString("<" + element + ">\n")
	w.depth++
}

func (w *plistWriter) close(element string) {
	w.depth--
	w.indent()
	w.buf.WriteString("</" + element + ">\n")
}

func (w *plistWriter) boolean(k string, v bool) {
	w.key(k)
	w.indent()
	if v {
		w.buf.WriteString("<true/>\n")
	} else {
		w.buf.WriteString("<false/>\n")
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
