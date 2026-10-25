//go:build windows

package win32

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

func TestSystemDirectoryAndFileVersion(t *testing.T) {
	dir, err := SystemDirectory()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(strings.ToLower(dir), "system32") {
		t.Errorf("SystemDirectory = %q, want the System32 directory", dir)
	}
	v, err := FileVersionInfo(filepath.Join(dir, "kernel32.dll"))
	if err != nil {
		t.Fatal(err)
	}
	if v.Major == 0 {
		t.Errorf("the version of kernel32.dll is %s", v)
	}
	t.Logf("kernel32.dll %s", v)
}

func TestFileVersionInfoErrors(t *testing.T) {
	dir, err := SystemDirectory()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, path, tag, route string
	}{
		{"a file that is not there", filepath.Join(dir, "glutter-no-such-file.dll"), TagWin32CallFailed, "FileVersionInfo->Win32Error.Win32CallFailed"},
		{"a path with a NUL", "a\x00b", TagUTF16EmbeddedNUL, "UTF16FromString.UTF16EmbeddedNUL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FileVersionInfo(tt.path)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != tt.tag || !appErr.HasRoute(tt.route) {
				t.Errorf("got %s via %s, want %s via %s", appErr.Tag, appErr.Route(), tt.tag, tt.route)
			}
		})
	}
}
