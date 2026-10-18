package dl

import (
	"os"
	"path/filepath"

	"github.com/ebitengine/purego"
	"github.com/rm4n0s/errors"
)

// Lib is an open C library.
//
// Flutter: no counterpart
type Lib struct {
	name   string
	handle uintptr
}

// Open loads the first of names that dlopen accepts, with RTLD_NOW|RTLD_LOCAL.
// Give the versioned name first (such as "libEGL.so.1"), because the
// unversioned symlink exists only when the -devel package is installed. If no
// name loads, Open fails with LibraryLoadFailed and the metadata "names" and
// "dlerror" (the message of the last failed attempt).
//
// Flutter: no counterpart
func Open(names ...string) (*Lib, error) {
	var lastErr error
	for _, name := range names {
		handle, err := purego.Dlopen(name, purego.RTLD_NOW|purego.RTLD_LOCAL)
		if err == nil {
			return &Lib{name: name, handle: handle}, nil
		}
		lastErr = err
	}
	dlerror := ""
	if lastErr != nil {
		dlerror = lastErr.Error()
	}
	return nil, errors.New("LibraryLoadFailed", "could not load the library",
		"names", names, "dlerror", dlerror)
}

// OpenOverride is like Open, except that when the environment variable env
// holds an absolute path, only that path is tried. This is for unusual setups,
// such as GLUTTER_EGL_LIBRARY on NixOS. A set but failing override is an error
// (it does not fall back to names), and a value that is not an absolute path
// fails with LibraryOverrideInvalid.
//
// Flutter: no counterpart
func OpenOverride(env string, names ...string) (*Lib, error) {
	path := os.Getenv(env)
	if path == "" {
		return Open(names...)
	}
	if !filepath.IsAbs(path) {
		return nil, errors.New("LibraryOverrideInvalid", "the library override is not an absolute path",
			"env", env, "path", path)
	}
	return Open(path)
}

// Name returns the name or path that was loaded.
//
// Flutter: no counterpart
func (l *Lib) Name() string { return l.name }

// Bind sets fptr, a pointer to a Go func variable, to call the C function
// symbol. The func's type describes the C signature as purego defines it. It
// fails with SymbolMissing (metadata "library", "symbol") when the library
// has no such symbol. A fptr that is not a pointer to a func is a programmer
// error and panics.
//
// Flutter: no counterpart
func (l *Lib) Bind(fptr any, symbol string) error {
	if _, err := purego.Dlsym(l.handle, symbol); err != nil {
		return errors.NewErr("SymbolMissing", err).SetMetadata("library", l.name, "symbol", symbol)
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
