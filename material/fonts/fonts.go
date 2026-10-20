// Package fonts holds the fonts that Glutter bundles with the program: Roboto, the
// first family of Flutter's Linux typography, and Material Icons. scripts/fetch-fonts.sh
// downloads them (the pins and hashes are in VERSIONS), and the licenses sit next to
// them: Roboto is under the SIL Open Font License; the Material Icons font is the one
// of the Flutter SDK, with the license text that ships with it (Creative Commons
// Attribution 4.0).
//
// Flutter: packages/flutter/lib/src/material/typography.dart (the Roboto family) and
// the MaterialIcons font of the Flutter SDK
package fonts

import _ "embed"

// The names of the bundled families, as a TextStyle's fontFamily refers to them.
const (
	// RobotoFamily is the family of the bundled Roboto font.
	//
	// Flutter: material/typography.dart Typography
	RobotoFamily = "Roboto"
	// MaterialIconsFamily is the family of the bundled Material Icons font.
	//
	// Flutter: widgets/icon_data.dart IconData.fontFamily
	MaterialIconsFamily = "MaterialIcons"
)

//go:embed Roboto-Variable.ttf
var roboto []byte

//go:embed MaterialIcons-Regular.otf
var materialIcons []byte

//go:embed Roboto-LICENSE.txt
var robotoLicense string

//go:embed MaterialIcons-LICENSE.txt
var materialIconsLicense string

// Roboto returns the Roboto variable font (Roboto 3, with the axes ital, wdth and
// wght). The caller must not change the bytes.
//
// Flutter: material/typography.dart Typography
func Roboto() []byte { return roboto }

// MaterialIcons returns the Material Icons font. The caller must not change the
// bytes.
//
// Flutter: widgets/icon_data.dart IconData
func MaterialIcons() []byte { return materialIcons }

// RobotoLicense returns the license text of the Roboto font.
//
// Flutter: no counterpart
func RobotoLicense() string { return robotoLicense }

// MaterialIconsLicense returns the license text of the Material Icons font.
//
// Flutter: no counterpart
func MaterialIconsLicense() string { return materialIconsLicense }

// Register adds the bundled families to a font collection, which is anything with
// the AddBundled method of internal/text's Collection. It stops at the first error.
//
// Flutter: no counterpart
func Register(c interface {
	AddBundled(family string, data []byte) error
}) error {
	if err := c.AddBundled(RobotoFamily, roboto); err != nil {
		return err
	}
	return c.AddBundled(MaterialIconsFamily, materialIcons)
}
