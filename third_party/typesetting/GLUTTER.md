# Glutter's copy of go-text/typesetting

This directory is `github.com/go-text/typesetting` v0.3.4 (Unlicense OR BSD-3-Clause, see `LICENSE`), without its tests, test
data and `testutils`, used through the `replace` directive of Glutter's `go.mod`. It is changed in one place (SPEC D39, step 37.1):

- `font/opentype/tables/xvar_src.go`: `Gvar.parseGlyphVariationDatas` keeps the bytes of the `gvar` table and no longer parses the
  variation data of every glyph; `Gvar.GlyphCount` and `Gvar.GlyphVariation(glyph)` parse one glyph's data when it is asked for.
  `GlyphVariationDatas` stays empty.
- `font/variations.go`: `gvar` keeps the table and unpacks the deltas of a glyph the first time that glyph is drawn with variations
  (`gvarGlyphs.glyphVariations`, under a mutex, kept for the next time). Opening a variable font such as macOS's `SFNS.ttf` held
  about 31 MB for this table alone; it holds almost nothing now. The outlines are the same: the outlines of every glyph of
  `Roboto-Variable.ttf` and `SFNS.ttf` at four settings of the axes hash to the same value with v0.3.4 and with this copy.
- Two files (`harfbuzz/aat_layout_trak.go`, `internal/unicodedata/vertical_orientation.go`) were put through `gofmt` so that
  `gofmt -l .` of Glutter is clean.

To go back to the module: delete the `replace` line of `go.mod` and this directory. Glutter's test
`internal/text.TestOpeningAVariableFontKeepsLittle` fails with the unchanged module.
