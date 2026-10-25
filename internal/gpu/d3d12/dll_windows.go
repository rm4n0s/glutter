//go:build windows

package d3d12

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// entryNames are the libraries and the functions the device calls by name (SPEC §9.8 "Loading").
// Everything else is a method of a COM object, called through its vtable. A test changes a name to
// see the error of a library or a function that is missing.
type entryNames struct {
	d3d12, dxgi                                                            string
	createDevice, serializeRootSignature, getDebugInterface, createFactory string
}

// defaultEntryNames are the names of the real libraries and functions.
func defaultEntryNames() entryNames {
	return entryNames{
		d3d12: "d3d12.dll", dxgi: "dxgi.dll",
		createDevice: "D3D12CreateDevice", serializeRootSignature: "D3D12SerializeRootSignature",
		getDebugInterface: "D3D12GetDebugInterface", createFactory: "CreateDXGIFactory2",
	}
}

// libraries are d3d12.dll and dxgi.dll, loaded from System32, and the functions of them that are
// called by name.
type libraries struct {
	d3d12, dxgi *win32.Library

	createDevice, serializeRootSignature, getDebugInterface, createFactory *win32.Proc
}

// loadLibraries loads d3d12.dll and dxgi.dll from System32 only and finds the functions. A library
// or a function that is missing is D3D12LoadFailed, with the metadata "library" and "symbol" (empty
// for a library), and "cause", the tag of the error of win32 that said so: the device's loader
// names its own tag (SPEC §9.8); the loader of the shader compiler keeps win32's (SPEC §9.8, "Decided
// in step 28.8").
func loadLibraries(n entryNames) (*libraries, error) {
	l := &libraries{}
	for _, e := range []libraryEntry{
		{n.d3d12, &l.d3d12, []symbolEntry{{n.createDevice, &l.createDevice}, {n.serializeRootSignature, &l.serializeRootSignature}, {n.getDebugInterface, &l.getDebugInterface}}},
		{n.dxgi, &l.dxgi, []symbolEntry{{n.createFactory, &l.createFactory}}},
	} {
		lib, err := win32.LoadSystemDLL(e.name)
		if err != nil {
			return nil, errors.New(TagD3D12LoadFailed, "a library of Direct3D 12 cannot be loaded from System32",
				"library", e.name, "symbol", "", "cause", causeOf(err))
		}
		*e.lib = lib
		for _, s := range e.symbols {
			proc, err := lib.Proc(s.name)
			if err != nil {
				return nil, errors.New(TagD3D12LoadFailed, "a function of Direct3D 12 is missing from its library",
					"library", e.name, "symbol", s.name, "cause", causeOf(err))
			}
			*s.proc = proc
		}
	}
	return l, nil
}

// libraryEntry is a library to load and the functions to find in it; symbolEntry is a function and
// where to put it.
type libraryEntry struct {
	name    string
	lib     **win32.Library
	symbols []symbolEntry
}

type symbolEntry struct {
	name string
	proc **win32.Proc
}

// causeOf is the tag of an error of this module, or its text.
func causeOf(err error) string {
	if appErr, ok := errors.FromError(err); ok {
		return appErr.Tag
	}
	return err.Error()
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// TagD3D12LoadFailed is the tag of the error for d3d12.dll or dxgi.dll that is not in System32, and
	// for a function that one of them lacks; the metadata are "library", "symbol" and "cause".
	TagD3D12LoadFailed = "D3D12LoadFailed"
)
