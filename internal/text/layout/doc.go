// Package layout lays out paragraphs: it turns the styled text of a
// ui.ParagraphBuilder into lines of shaped glyphs with go-text. The steps are
// those of SPEC §10:
//
//  1. the text is split into runs by bidi level, script, style and font (with
//     fallback through text.Collection);
//  2. the runs are shaped with HarfBuzz;
//  3. the lines are broken (UAX #14) honoring the width, the maximum number of
//     lines and the ellipsis;
//  4. each line is reordered for bidi;
//  5. the text is aligned and justified, and the lines take their height from the
//     fonts, the style heights and the strut;
//  6. the metrics (baselines, line metrics, intrinsic widths) are computed.
//
// Offsets into the text are UTF-8 byte offsets (SPEC §12.10).
//
// Flutter: engine/src/flutter/third_party/txt (libtxt) and Skia's skparagraph module
package layout
