// Package fontconfig binds libfontconfig (through purego, SPEC §8.1) to match a
// font family, weight, slant and language to the font files of the system, in the
// fallback order the user's fontconfig settings give. The Linux engine of Flutter
// asks fontconfig the same way.
//
// Flutter: no counterpart (engine/src/flutter/third_party/txt uses Skia's
// SkFontMgr_fontconfig)
package fontconfig
