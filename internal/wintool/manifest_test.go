package wintool

import (
	"encoding/xml"
	"io"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

func TestManifestGolden(t *testing.T) {
	got, err := Manifest(ManifestConfig{AppID: "org.glutter.counter", Name: `Count & "go"`, Version: Version{1, 2, 3, 4}, Arch: "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	const want = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
  <assemblyIdentity type="win32" name="org.glutter.counter" version="1.2.3.4" processorArchitecture="amd64" />
  <description>Count &amp; &#34;go&#34;</description>
  <dependency>
    <dependentAssembly>
      <assemblyIdentity type="win32" name="Microsoft.Windows.Common-Controls" version="6.0.0.0" processorArchitecture="*" publicKeyToken="6595b64144ccf1df" language="*" />
    </dependentAssembly>
  </dependency>
  <compatibility xmlns="urn:schemas-microsoft-com:compatibility.v1">
    <application>
      <supportedOS Id="{8e0f7a12-bfb3-4fe8-b9a5-48fd50a15a9a}" />
    </application>
  </compatibility>
  <trustInfo xmlns="urn:schemas-microsoft-com:asm.v3">
    <security>
      <requestedPrivileges>
        <requestedExecutionLevel level="asInvoker" uiAccess="false" />
      </requestedPrivileges>
    </security>
  </trustInfo>
  <application xmlns="urn:schemas-microsoft-com:asm.v3">
    <windowsSettings>
      <dpiAware xmlns="http://schemas.microsoft.com/SMI/2005/WindowsSettings">true/pm</dpiAware>
      <dpiAwareness xmlns="http://schemas.microsoft.com/SMI/2016/WindowsSettings">PerMonitorV2</dpiAwareness>
      <longPathAware xmlns="http://schemas.microsoft.com/SMI/2016/WindowsSettings">true</longPathAware>
      <activeCodePage xmlns="http://schemas.microsoft.com/SMI/2019/WindowsSettings">UTF-8</activeCodePage>
    </windowsSettings>
  </application>
</assembly>
`
	if string(got) != want {
		t.Errorf("manifest:\n%s\nwant:\n%s", got, want)
	}
}

func TestManifestIsWellFormedForBothArchitectures(t *testing.T) {
	for _, arch := range Archs {
		got, err := Manifest(ManifestConfig{AppID: "a.b-c_d", Name: "x < y", Version: DefaultVersion, Arch: arch.Name})
		if err != nil {
			t.Fatal(err)
		}
		dec := xml.NewDecoder(strings.NewReader(string(got)))
		for {
			if _, err := dec.Token(); err != nil {
				if !errors.Is(err, io.EOF) {
					t.Errorf("%s: the manifest is not well-formed XML: %v\n%s", arch.Name, err, got)
				}
				break
			}
		}
		if !strings.Contains(string(got), `processorArchitecture="`+arch.Name+`"`) {
			t.Errorf("%s: the manifest does not name the architecture:\n%s", arch.Name, got)
		}
	}
}

func TestManifestErrors(t *testing.T) {
	tests := []struct {
		name string
		c    ManifestConfig
		meta string
	}{
		{"an empty application ID", ManifestConfig{Name: "x", Arch: "amd64"}, "appID"},
		{"a space in the application ID", ManifestConfig{AppID: "a b", Name: "x", Arch: "amd64"}, "appID"},
		{"a quote in the application ID", ManifestConfig{AppID: `a"b`, Name: "x", Arch: "amd64"}, "appID"},
		{"a blank name", ManifestConfig{AppID: "a.b", Name: "  ", Arch: "amd64"}, "appID"},
		{"an unknown architecture", ManifestConfig{AppID: "a.b", Name: "x", Arch: "386"}, "arch"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Manifest(tt.c)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute("Manifest.WindowsManifestInvalid") {
				t.Errorf("route %s, want Manifest.WindowsManifestInvalid", appErr.Route())
			}
			if _, has := appErr.Metadata[tt.meta]; !has {
				t.Errorf("metadata %v has no %q", appErr.Metadata, tt.meta)
			}
		})
	}
}

func TestDefaultAppIDAndName(t *testing.T) {
	tests := []struct{ pkg, id, name string }{
		{"./examples/counter", "org.glutter.counter", "counter"},
		{"./examples/dev/my-app", "org.glutter.my_app", "my-app"},
		{"github.com/x/My.App", "org.glutter.my_app", "My.App"},
		{"./examples/3d", "org.glutter.a3d", "3d"},
		{".", "org.glutter.app", "Glutter"},
		{"", "org.glutter.app", "Glutter"},
	}
	for _, tt := range tests {
		if got := DefaultAppID(tt.pkg); got != tt.id {
			t.Errorf("DefaultAppID(%q) = %q, want %q", tt.pkg, got, tt.id)
		}
		if got := DefaultName(tt.pkg); got != tt.name {
			t.Errorf("DefaultName(%q) = %q, want %q", tt.pkg, got, tt.name)
		}
		if _, err := Manifest(ManifestConfig{AppID: DefaultAppID(tt.pkg), Name: DefaultName(tt.pkg), Arch: "amd64"}); err != nil {
			t.Errorf("the defaults of %q give %v", tt.pkg, err)
		}
	}
}
