//go:build darwin

package dl

import (
	"github.com/ebitengine/purego"
	"github.com/rm4n0s/errors"
)

// FrameworkDir is where the system frameworks are (on macOS 11 and later their files are not on the disk,
// but dlopen finds them in the shared cache by this path).
const FrameworkDir = "/System/Library/Frameworks"

// Lib is an open C library or framework.
//
// Flutter: no counterpart
type Lib struct {
	name   string
	handle uintptr
}

// Open loads the first of paths that dlopen accepts, with RTLD_NOW|RTLD_LOCAL. Give absolute paths
// (SPEC §29.2). If no path loads, Open fails with LibraryLoadFailed and the metadata "names" and
// "dlerror" (the message of the last failed attempt).
//
// Flutter: no counterpart
func Open(paths ...string) (*Lib, error) {
	var lastErr error
	for _, path := range paths {
		handle, err := purego.Dlopen(path, purego.RTLD_NOW|purego.RTLD_LOCAL)
		if err == nil {
			return &Lib{name: path, handle: handle}, nil
		}
		lastErr = err
	}
	dlerror := ""
	if lastErr != nil {
		dlerror = lastErr.Error()
	}
	return nil, errors.New(tagLibraryLoadFailed, "could not load the library", "names", paths, "dlerror", dlerror)
}

// OpenFramework loads the system framework of that name, such as "AppKit" or "Metal", from
// /System/Library/Frameworks/<name>.framework/<name>. It fails with MacOSFrameworkMissing and the
// metadata "path" and "dlerror" when the framework is not there (a version of the system that does not
// have it, or iOS asked for an AppKit-only framework).
//
// Flutter: no counterpart
func OpenFramework(name string) (*Lib, error) {
	path := FrameworkPath(name)
	handle, err := purego.Dlopen(path, purego.RTLD_NOW|purego.RTLD_LOCAL)
	if err != nil {
		return nil, errors.New(tagMacOSFrameworkMissing, "could not load the framework", "path", path, "dlerror", err.Error())
	}
	return &Lib{name: path, handle: handle}, nil
}

// FrameworkPath is the path of the binary of a system framework: FrameworkDir/<name>.framework/<name>.
//
// Flutter: no counterpart
func FrameworkPath(name string) string {
	return FrameworkDir + "/" + name + ".framework/" + name
}

// Name returns the path that was loaded.
//
// Flutter: no counterpart
func (l *Lib) Name() string { return l.name }

// Handle returns the dlopen handle, for the calls of purego that need it (objc and the like).
//
// Flutter: no counterpart
func (l *Lib) Handle() uintptr { return l.handle }

// Bind sets fptr, a pointer to a Go func variable, to call the C function symbol. The func's type
// describes the C signature as purego defines it (a structure may be passed and returned by value on
// darwin, on arm64 and amd64). It fails with SymbolMissing (metadata "library", "symbol") when the
// library has no such symbol. A fptr that is not a pointer to a func is a programmer error and panics.
//
// Flutter: no counterpart
func (l *Lib) Bind(fptr any, symbol string) error {
	if _, err := purego.Dlsym(l.handle, symbol); err != nil {
		return errors.NewErr(tagSymbolMissing, err).SetMetadata("library", l.name, "symbol", symbol)
	}
	purego.RegisterLibFunc(fptr, l.handle, symbol)
	return nil
}

// Has reports whether the library has the symbol, without failing.
//
// Flutter: no counterpart
func (l *Lib) Has(symbol string) bool {
	_, err := purego.Dlsym(l.handle, symbol)
	return err == nil
}

// Symbol returns the address of a symbol (a C function, or a constant such as an NSString* that a
// framework exports), or SymbolMissing.
//
// Flutter: no counterpart
func (l *Lib) Symbol(symbol string) (uintptr, error) {
	addr, err := purego.Dlsym(l.handle, symbol)
	if err != nil {
		return 0, errors.NewErr(tagSymbolMissing, err).SetMetadata("library", l.name, "symbol", symbol)
	}
	return addr, nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagLibraryLoadFailed     = "LibraryLoadFailed"
	tagMacOSFrameworkMissing = "MacOSFrameworkMissing"
	tagSymbolMissing         = "SymbolMissing"
)
