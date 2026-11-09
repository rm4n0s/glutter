package mactool

import (
	"os"
	"path/filepath"

	"github.com/rm4n0s/errors"
)

// Bundle is what goes into an app bundle.
//
// Flutter: no counterpart
type Bundle struct {
	Info Info
	// Executable is the Mach-O file, which is written with mode 0755 as Contents/MacOS/<Info.Executable>.
	Executable []byte
	// Icon is the .icns file; nil leaves the bundle without one.
	Icon []byte
}

// Layout names the paths of a bundle that is at app (a directory ending in ".app").
//
// Flutter: no counterpart
type Layout struct {
	App, Contents, MacOS, Resources, Plist, Executable, Icon string
}

// LayoutOf names the paths of the bundle of the info at app.
//
// Flutter: no counterpart
func LayoutOf(app string, i Info) Layout {
	l := Layout{App: app, Contents: filepath.Join(app, "Contents")}
	l.MacOS = filepath.Join(l.Contents, "MacOS")
	l.Resources = filepath.Join(l.Contents, "Resources")
	l.Plist = filepath.Join(l.Contents, "Info.plist")
	l.Executable = filepath.Join(l.MacOS, i.Executable)
	if i.IconFile != "" {
		l.Icon = filepath.Join(l.Resources, i.IconFile+".icns")
	}
	return l
}

// Write makes the bundle at app, replacing a bundle that is there. It writes Contents/MacOS/<executable>, Contents/Info.plist,
// Contents/PkgInfo and, with an icon, Contents/Resources/<IconFile>.icns. A file at app that is not a bundle is not removed: it is
// MacOSBundleInvalid. It fails with MacOSBundleInvalid for an identity that a bundle cannot have and for a path that cannot be
// written.
//
// Flutter: no counterpart
func Write(app string, b Bundle) (Layout, error) {
	l := LayoutOf(app, b.Info)
	if filepath.Ext(app) != ".app" {
		return l, errors.New(tagMacOSBundleInvalid, "the bundle's name ends in .app", "path", app)
	}
	if len(b.Executable) == 0 {
		return l, errors.New(tagMacOSBundleInvalid, "the bundle has no executable")
	}
	if b.Icon != nil && b.Info.IconFile == "" {
		return l, errors.New(tagMacOSBundleInvalid, "an icon needs the name of its file in the plist")
	}
	plist, err := InfoPlist(b.Info)
	if err != nil {
		return l, err
	}
	if st, err := os.Stat(app); err == nil {
		if !st.IsDir() || !isBundle(l) {
			return l, errors.New(tagMacOSBundleInvalid, "the path exists and is not a bundle, so it is not replaced", "path", app)
		}
		if err := os.RemoveAll(app); err != nil {
			return l, errors.NewErr(tagMacOSBundleInvalid, err).SetMetadata("path", app)
		}
	}
	for _, d := range []string{l.MacOS, l.Resources} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return l, errors.NewErr(tagMacOSBundleInvalid, err).SetMetadata("path", d)
		}
	}
	files := []struct {
		path string
		data []byte
		mode os.FileMode
	}{
		{l.Executable, b.Executable, 0o755},
		{l.Plist, plist, 0o644},
		{filepath.Join(l.Contents, "PkgInfo"), []byte("APPL????"), 0o644},
	}
	if b.Icon != nil {
		files = append(files, struct {
			path string
			data []byte
			mode os.FileMode
		}{l.Icon, b.Icon, 0o644})
	}
	for _, f := range files {
		if err := os.WriteFile(f.path, f.data, f.mode); err != nil {
			return l, errors.NewErr(tagMacOSBundleInvalid, err).SetMetadata("path", f.path)
		}
	}
	return l, nil
}

// isBundle says whether a directory is the bundle of a layout: it has a Contents directory with an Info.plist.
func isBundle(l Layout) bool {
	st, err := os.Stat(l.Plist)
	return err == nil && !st.IsDir()
}
