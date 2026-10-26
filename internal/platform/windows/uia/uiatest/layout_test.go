//go:build !windows

package uiatest

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32/layouttest"
)

// TestLayoutAgainstTheHeaders checks the names and the positions of the methods of the client interfaces
// against the mingw-w64 headers (SPEC §25.2). It runs on the Linux host and is skipped, with the install
// command, when the compiler is missing.
func TestLayoutAgainstTheHeaders(t *testing.T) {
	var vtables []layouttest.VTable
	for _, iface := range Interfaces {
		v := layouttest.VTable{CName: iface.Name + "Vtbl"}
		for i, m := range iface.Methods {
			v.Methods = append(v.Methods, layouttest.Method{Name: m, Index: 3 + i})
		}
		vtables = append(vtables, v)
	}
	layouttest.Check(t, layouttest.Spec{
		Headers: []string{"windows.h", "unknwn.h", "oleauto.h", "uiautomationcore.h", "uiautomationclient.h"},
		VTables: vtables,
	})
}
