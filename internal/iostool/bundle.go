package iostool

import (
	"os"
	"path/filepath"

	"github.com/rm4n0s/errors"
)

// Bundle is what Write puts into an app bundle.
//
// Flutter: no counterpart
type Bundle struct {
	// Info is the Info.plist.
	Info Info
	// Executable is the program.
	Executable []byte
	// Icons are the icon files by name (the result of Icons); none for no icon.
	Icons map[string][]byte
	// Profile is the provisioning profile of a build for a device (embedded.mobileprovision); none for the Simulator.
	Profile []byte
}

// Layout has the paths of a bundle. An iOS bundle is flat: the executable, the Info.plist and the resources sit in the directory of
// the app, unlike a bundle of macOS.
//
// Flutter: no counterpart
type Layout struct {
	// App is the directory of the bundle ("Counter.app").
	App string
	// Executable is the program.
	Executable string
	// Plist is the Info.plist.
	Plist string
	// Profile is embedded.mobileprovision.
	Profile string
}

// LayoutOf is where the files of a bundle of the app go.
//
// Flutter: no counterpart
func LayoutOf(app string, i Info) Layout {
	return Layout{App: app, Executable: filepath.Join(app, i.Executable), Plist: filepath.Join(app, "Info.plist"), Profile: filepath.Join(app, "embedded.mobileprovision")}
}

// Write makes the bundle at app, which must end in ".app", replacing what is there. It fails with IOSBundleInvalid when the path is
// not a bundle's, the plist is invalid or a file cannot be written.
//
// Flutter: no counterpart
func Write(app string, b Bundle) (Layout, error) {
	if filepath.Ext(app) != ".app" {
		return Layout{}, errors.New(tagIOSBundleInvalid, "a bundle is a directory whose name ends in .app", "path", app)
	}
	b.Info.Icons = len(b.Icons) > 0
	plist, err := InfoPlist(b.Info)
	if err != nil {
		return Layout{}, err
	}
	l := LayoutOf(app, b.Info)
	if err := os.RemoveAll(app); err != nil {
		return l, errors.NewErr(tagIOSBundleInvalid, err).SetMetadata("path", app)
	}
	if err := os.MkdirAll(app, 0o755); err != nil {
		return l, errors.NewErr(tagIOSBundleInvalid, err).SetMetadata("path", app)
	}
	write := func(path string, data []byte, mode os.FileMode) error {
		if err := os.WriteFile(path, data, mode); err != nil {
			return errors.NewErr(tagIOSBundleInvalid, err).SetMetadata("path", path)
		}
		return nil
	}
	if err := write(l.Plist, plist, 0o644); err != nil {
		return l, err
	}
	if err := write(l.Executable, b.Executable, 0o755); err != nil {
		return l, err
	}
	for name, data := range b.Icons {
		if err := write(filepath.Join(app, name), data, 0o644); err != nil {
			return l, err
		}
	}
	if len(b.Profile) > 0 {
		if err := write(l.Profile, b.Profile, 0o644); err != nil {
			return l, err
		}
	}
	return l, nil
}
