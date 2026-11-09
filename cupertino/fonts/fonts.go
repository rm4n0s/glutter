// Package fonts holds the font that the cupertino package bundles: CupertinoIcons, the
// font of the cupertino_icons package (version 2.0.0, MIT license). Its license is next
// to it and in third_party/cupertino_icons/LICENSE. The font is in a program only when
// the program imports package cupertino, and cupertino.CupertinoIcons loads it into an
// app's text engine the first time an icon of it is built (widgets.IconFont), so an app
// that never draws one pays no memory for it.
//
// San Francisco, the font of the Cupertino text styles, is never bundled: Apple's
// license allows it on Apple platforms only, and the engine finds it there (SPEC §31.2).
//
// Flutter: the CupertinoIcons font of the cupertino_icons package
package fonts

import _ "embed"

// CupertinoIconsFamily is the family name of the CupertinoIcons font inside its package.
//
// Flutter: cupertino/icons.dart CupertinoIcons.iconFont
const CupertinoIconsFamily = "CupertinoIcons"

// CupertinoIconsPackage is the package that holds the CupertinoIcons font.
//
// Flutter: cupertino/icons.dart CupertinoIcons.iconFontPackage
const CupertinoIconsPackage = "cupertino_icons"

//go:embed CupertinoIcons.ttf
var cupertinoIcons []byte

//go:embed CupertinoIcons-LICENSE.txt
var cupertinoIconsLicense string

// CupertinoIcons returns the CupertinoIcons font. The caller must not change the bytes.
//
// Flutter: cupertino/icons.dart CupertinoIcons
func CupertinoIcons() []byte { return cupertinoIcons }

// CupertinoIconsLicense returns the license text of the CupertinoIcons font.
//
// Flutter: no counterpart
func CupertinoIconsLicense() string { return cupertinoIconsLicense }
