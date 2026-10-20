// Package mcu is a port of Google's material-color-utilities (the Dart
// package in reference/material-color-utilities): the HCT color space, the
// CAM16 color appearance model, tonal palettes, the dynamic color scheme
// variants and the contrast tools that Material 3 builds its color schemes with.
// Colors are ARGB values in a uint32, as 0xAARRGGBB.
//
// The package ports hct, scheme, dynamiccolor, palettes, utils, contrast, blend,
// dislike and temperature. The image quantizers and the scoring of the colors of
// an image (quantize, score) are not ported: they are needed only by
// ColorScheme.fromImageProvider, which Glutter does not have (SPEC §12.10).
//
// The code is a derivative of material-color-utilities (Apache-2.0); see NOTICE.
//
// Flutter: no counterpart
package mcu
