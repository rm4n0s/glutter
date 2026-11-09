package cupertino

import (
	"github.com/rm4n0s/glutter/cupertino/fonts"
	"github.com/rm4n0s/glutter/widgets"
)

// CupertinoIconsFont is the font family of the Cupertino icons.
//
// Flutter: cupertino/icons.dart CupertinoIcons.iconFont
const CupertinoIconsFont = fonts.CupertinoIconsFamily

// CupertinoIconsFontPackage is the package that holds the font of the Cupertino icons. A
// TextStyle that names the family with this package asks the engine for
// "packages/cupertino_icons/CupertinoIcons", the name the font is loaded under.
//
// Flutter: cupertino/icons.dart CupertinoIcons.iconFontPackage
const CupertinoIconsFontPackage = fonts.CupertinoIconsPackage

// cupertinoIconFont is the font file of the icons, which an Icon loads into the text
// engine the first time it builds one of them. The icons are the only references to it, so a
// program that never calls one of the methods of CupertinoIcons does not link the font
// (a test builds one and looks at its size).
var cupertinoIconFont = &widgets.IconFont{
	Family: "packages/" + CupertinoIconsFontPackage + "/" + CupertinoIconsFont,
	Data:   fonts.CupertinoIcons,
}
