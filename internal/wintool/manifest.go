package wintool

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/rm4n0s/errors"
)

// ManifestConfig is what the generated application manifest says (SPEC §25.1).
//
// Flutter: no counterpart
type ManifestConfig struct {
	// AppID is the name of the assembly identity, such as "org.glutter.counter": letters, digits,
	// dots, dashes and underscores.
	AppID string
	// Name is the description of the application.
	Name string
	// Version is the version of the assembly identity (four numbers).
	Version Version
	// Arch is the processor architecture: "amd64" or "arm64".
	Arch string
}

// Manifest returns the text of the application manifest (SPEC §25.1): the assembly identity, version
// 6 of the Common Controls, the supported operating systems (Windows 10 and 11), the execution level
// asInvoker, and the Windows settings: per-monitor DPI awareness version 2 (`true/pm` for older
// builds), the UTF-8 code page and long paths. It fails with WindowsManifestInvalid when the
// application ID has a character a manifest cannot have, the name is empty or the architecture is not
// known.
//
// Flutter: no counterpart (windows/runner/runner.exe.manifest)
func Manifest(c ManifestConfig) ([]byte, error) {
	if c.AppID == "" || strings.Trim(c.AppID, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789._-") != "" {
		return nil, errors.New(tagWindowsManifestInvalid, "the application ID of the manifest is empty or has a character other than letters, digits, dot, dash and underscore", "appID", c.AppID)
	}
	if strings.TrimSpace(c.Name) == "" {
		return nil, errors.New(tagWindowsManifestInvalid, "the name of the application is empty", "appID", c.AppID)
	}
	if _, ok := FindArch(c.Arch); !ok {
		return nil, errors.New(tagWindowsManifestInvalid, "the processor architecture is not amd64 or arm64", "arch", c.Arch)
	}
	var esc bytes.Buffer
	if err := xml.EscapeText(&esc, []byte(c.Name)); err != nil {
		return nil, errors.NewErr(tagWindowsManifestInvalid, err).SetMetadata("name", c.Name)
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, "<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"yes\"?>\n")
	fmt.Fprintf(&b, "<assembly xmlns=\"urn:schemas-microsoft-com:asm.v1\" manifestVersion=\"1.0\">\n")
	fmt.Fprintf(&b, "  <assemblyIdentity type=\"win32\" name=\"%s\" version=\"%s\" processorArchitecture=\"%s\" />\n", c.AppID, c.Version, c.Arch)
	fmt.Fprintf(&b, "  <description>%s</description>\n", esc.String())
	fmt.Fprintf(&b, "  <dependency>\n")
	fmt.Fprintf(&b, "    <dependentAssembly>\n")
	fmt.Fprintf(&b, "      <assemblyIdentity type=\"win32\" name=\"Microsoft.Windows.Common-Controls\" version=\"6.0.0.0\" processorArchitecture=\"*\" publicKeyToken=\"6595b64144ccf1df\" language=\"*\" />\n")
	fmt.Fprintf(&b, "    </dependentAssembly>\n")
	fmt.Fprintf(&b, "  </dependency>\n")
	fmt.Fprintf(&b, "  <compatibility xmlns=\"urn:schemas-microsoft-com:compatibility.v1\">\n")
	fmt.Fprintf(&b, "    <application>\n")
	fmt.Fprintf(&b, "      <supportedOS Id=\"%s\" />\n", SupportedOSWindows10)
	fmt.Fprintf(&b, "    </application>\n")
	fmt.Fprintf(&b, "  </compatibility>\n")
	fmt.Fprintf(&b, "  <trustInfo xmlns=\"urn:schemas-microsoft-com:asm.v3\">\n")
	fmt.Fprintf(&b, "    <security>\n")
	fmt.Fprintf(&b, "      <requestedPrivileges>\n")
	fmt.Fprintf(&b, "        <requestedExecutionLevel level=\"asInvoker\" uiAccess=\"false\" />\n")
	fmt.Fprintf(&b, "      </requestedPrivileges>\n")
	fmt.Fprintf(&b, "    </security>\n")
	fmt.Fprintf(&b, "  </trustInfo>\n")
	fmt.Fprintf(&b, "  <application xmlns=\"urn:schemas-microsoft-com:asm.v3\">\n")
	fmt.Fprintf(&b, "    <windowsSettings>\n")
	fmt.Fprintf(&b, "      <dpiAware xmlns=\"http://schemas.microsoft.com/SMI/2005/WindowsSettings\">true/pm</dpiAware>\n")
	fmt.Fprintf(&b, "      <dpiAwareness xmlns=\"http://schemas.microsoft.com/SMI/2016/WindowsSettings\">PerMonitorV2</dpiAwareness>\n")
	fmt.Fprintf(&b, "      <longPathAware xmlns=\"http://schemas.microsoft.com/SMI/2016/WindowsSettings\">true</longPathAware>\n")
	fmt.Fprintf(&b, "      <activeCodePage xmlns=\"http://schemas.microsoft.com/SMI/2019/WindowsSettings\">UTF-8</activeCodePage>\n")
	fmt.Fprintf(&b, "    </windowsSettings>\n")
	fmt.Fprintf(&b, "  </application>\n")
	fmt.Fprintf(&b, "</assembly>\n")
	return b.Bytes(), nil
}

// SupportedOSWindows10 is the GUID of the `supportedOS` element that names Windows 10 and 11.
//
// Flutter: no counterpart
const SupportedOSWindows10 = "{8e0f7a12-bfb3-4fe8-b9a5-48fd50a15a9a}"

// DefaultAppID derives an application ID from a package path: "./examples/counter" is
// "org.glutter.counter". Characters that an ID cannot have become underscores, and a name that does
// not start with a letter gets a leading "a".
//
// Flutter: no counterpart
func DefaultAppID(pkg string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(baseName(pkg)) {
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

// DefaultName is the base name of a package path, such as "counter"; "Glutter" for ".".
//
// Flutter: no counterpart
func DefaultName(pkg string) string {
	base := baseName(pkg)
	if base == "" || base == "." {
		return "Glutter"
	}
	return base
}

// baseName is the last element of a package path or directory.
func baseName(pkg string) string {
	if i := strings.LastIndexAny(pkg, `/\`); i >= 0 {
		return pkg[i+1:]
	}
	return pkg
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagWindowsManifestInvalid = TagWindowsManifestInvalid

	// TagWindowsManifestInvalid is the tag of the error for a manifest that cannot be made from the
	// configuration (an application ID, a name or an architecture it cannot have).
	TagWindowsManifestInvalid = "WindowsManifestInvalid"
)
