package material

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

var (
	keyUmbraOpacity      = ui.NewColor(0x33000000) // alpha = 0.2
	keyPenumbraOpacity   = ui.NewColor(0x24000000) // alpha = 0.14
	ambientShadowOpacity = ui.NewColor(0x1F000000) // alpha = 0.12
)

// ElevationToShadow maps the standard Material elevations (0, 1, 2, 3, 4, 6, 8,
// 9, 12, 16 and 24) to the shadows of that elevation, based on
// https://material.io/design/environment/elevation.html. Currently only the
// elevations that are bound to one or more widgets are defined. The empty list
// depicts no elevation. It is read-only.
//
// Flutter: material/shadows.dart kElevationToShadow
var ElevationToShadow = map[int][]painting.BoxShadow{
	0:  {},
	1:  {painting.NewBoxShadow(painting.BoxShadowOpts{Color: foundation.Some(keyUmbraOpacity), Offset: ui.Offset{Dx: 0.0, Dy: 2.0}, BlurRadius: 1.0, SpreadRadius: -1.0}), painting.NewBoxShadow(painting.BoxShadowOpts{Color: foundation.Some(keyPenumbraOpacity), Offset: ui.Offset{Dx: 0.0, Dy: 1.0}, BlurRadius: 1.0}), painting.NewBoxShadow(painting.BoxShadowOpts{Color: foundation.Some(ambientShadowOpacity), Offset: ui.Offset{Dx: 0.0, Dy: 1.0}, BlurRadius: 3.0})},
	2:  {painting.NewBoxShadow(painting.BoxShadowOpts{Color: foundation.Some(keyUmbraOpacity), Offset: ui.Offset{Dx: 0.0, Dy: 3.0}, BlurRadius: 1.0, SpreadRadius: -2.0}), painting.NewBoxShadow(painting.BoxShadowOpts{Color: foundation.Some(keyPenumbraOpacity), Offset: ui.Offset{Dx: 0.0, Dy: 2.0}, BlurRadius: 2.0}), painting.NewBoxShadow(painting.BoxShadowOpts{Color: foundation.Some(ambientShadowOpacity), Offset: ui.Offset{Dx: 0.0, Dy: 1.0}, BlurRadius: 5.0})},
	3:  {painting.NewBoxShadow(painting.BoxShadowOpts{Color: foundation.Some(keyUmbraOpacity), Offset: ui.Offset{Dx: 0.0, Dy: 3.0}, BlurRadius: 3.0, SpreadRadius: -2.0}), painting.NewBoxShadow(painting.BoxShadowOpts{Color: foundation.Some(keyPenumbraOpacity), Offset: ui.Offset{Dx: 0.0, Dy: 3.0}, BlurRadius: 4.0}), painting.NewBoxShadow(painting.BoxShadowOpts{Color: foundation.Some(ambientShadowOpacity), Offset: ui.Offset{Dx: 0.0, Dy: 1.0}, BlurRadius: 8.0})},
	4:  {painting.NewBoxShadow(painting.BoxShadowOpts{Color: foundation.Some(keyUmbraOpacity), Offset: ui.Offset{Dx: 0.0, Dy: 2.0}, BlurRadius: 4.0, SpreadRadius: -1.0}), painting.NewBoxShadow(painting.BoxShadowOpts{Color: foundation.Some(keyPenumbraOpacity), Offset: ui.Offset{Dx: 0.0, Dy: 4.0}, BlurRadius: 5.0}), painting.NewBoxShadow(painting.BoxShadowOpts{Color: foundation.Some(ambientShadowOpacity), Offset: ui.Offset{Dx: 0.0, Dy: 1.0}, BlurRadius: 10.0})},
	6:  {painting.NewBoxShadow(painting.BoxShadowOpts{Color: foundation.Some(keyUmbraOpacity), Offset: ui.Offset{Dx: 0.0, Dy: 3.0}, BlurRadius: 5.0, SpreadRadius: -1.0}), painting.NewBoxShadow(painting.BoxShadowOpts{Color: foundation.Some(keyPenumbraOpacity), Offset: ui.Offset{Dx: 0.0, Dy: 6.0}, BlurRadius: 10.0}), painting.NewBoxShadow(painting.BoxShadowOpts{Color: foundation.Some(ambientShadowOpacity), Offset: ui.Offset{Dx: 0.0, Dy: 1.0}, BlurRadius: 18.0})},
	8:  {painting.NewBoxShadow(painting.BoxShadowOpts{Color: foundation.Some(keyUmbraOpacity), Offset: ui.Offset{Dx: 0.0, Dy: 5.0}, BlurRadius: 5.0, SpreadRadius: -3.0}), painting.NewBoxShadow(painting.BoxShadowOpts{Color: foundation.Some(keyPenumbraOpacity), Offset: ui.Offset{Dx: 0.0, Dy: 8.0}, BlurRadius: 10.0, SpreadRadius: 1.0}), painting.NewBoxShadow(painting.BoxShadowOpts{Color: foundation.Some(ambientShadowOpacity), Offset: ui.Offset{Dx: 0.0, Dy: 3.0}, BlurRadius: 14.0, SpreadRadius: 2.0})},
	9:  {painting.NewBoxShadow(painting.BoxShadowOpts{Color: foundation.Some(keyUmbraOpacity), Offset: ui.Offset{Dx: 0.0, Dy: 5.0}, BlurRadius: 6.0, SpreadRadius: -3.0}), painting.NewBoxShadow(painting.BoxShadowOpts{Color: foundation.Some(keyPenumbraOpacity), Offset: ui.Offset{Dx: 0.0, Dy: 9.0}, BlurRadius: 12.0, SpreadRadius: 1.0}), painting.NewBoxShadow(painting.BoxShadowOpts{Color: foundation.Some(ambientShadowOpacity), Offset: ui.Offset{Dx: 0.0, Dy: 3.0}, BlurRadius: 16.0, SpreadRadius: 2.0})},
	12: {painting.NewBoxShadow(painting.BoxShadowOpts{Color: foundation.Some(keyUmbraOpacity), Offset: ui.Offset{Dx: 0.0, Dy: 7.0}, BlurRadius: 8.0, SpreadRadius: -4.0}), painting.NewBoxShadow(painting.BoxShadowOpts{Color: foundation.Some(keyPenumbraOpacity), Offset: ui.Offset{Dx: 0.0, Dy: 12.0}, BlurRadius: 17.0, SpreadRadius: 2.0}), painting.NewBoxShadow(painting.BoxShadowOpts{Color: foundation.Some(ambientShadowOpacity), Offset: ui.Offset{Dx: 0.0, Dy: 5.0}, BlurRadius: 22.0, SpreadRadius: 4.0})},
	16: {painting.NewBoxShadow(painting.BoxShadowOpts{Color: foundation.Some(keyUmbraOpacity), Offset: ui.Offset{Dx: 0.0, Dy: 8.0}, BlurRadius: 10.0, SpreadRadius: -5.0}), painting.NewBoxShadow(painting.BoxShadowOpts{Color: foundation.Some(keyPenumbraOpacity), Offset: ui.Offset{Dx: 0.0, Dy: 16.0}, BlurRadius: 24.0, SpreadRadius: 2.0}), painting.NewBoxShadow(painting.BoxShadowOpts{Color: foundation.Some(ambientShadowOpacity), Offset: ui.Offset{Dx: 0.0, Dy: 6.0}, BlurRadius: 30.0, SpreadRadius: 5.0})},
	24: {painting.NewBoxShadow(painting.BoxShadowOpts{Color: foundation.Some(keyUmbraOpacity), Offset: ui.Offset{Dx: 0.0, Dy: 11.0}, BlurRadius: 15.0, SpreadRadius: -7.0}), painting.NewBoxShadow(painting.BoxShadowOpts{Color: foundation.Some(keyPenumbraOpacity), Offset: ui.Offset{Dx: 0.0, Dy: 24.0}, BlurRadius: 38.0, SpreadRadius: 3.0}), painting.NewBoxShadow(painting.BoxShadowOpts{Color: foundation.Some(ambientShadowOpacity), Offset: ui.Offset{Dx: 0.0, Dy: 9.0}, BlurRadius: 46.0, SpreadRadius: 8.0})},
}
