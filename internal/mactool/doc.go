// Package mactool builds Glutter apps for macOS, for glutter-macos (SPEC §29.1): it writes the Info.plist of an app bundle, the
// .icns icon from a PNG, one Mach-O file with an arm64 and an amd64 slice (what `lipo` makes), the bundle's directories, and signs
// the bundle ad hoc with /usr/bin/codesign. All of it but the signing is plain Go with no OS tag, so it is tested on any host; it
// runs on the developer's Mac, never inside an app, and needs no Xcode, no C compiler and no lipo.
//
// Flutter: no counterpart (flutter build macos)
package mactool
