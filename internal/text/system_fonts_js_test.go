//go:build js

package text

// systemFontsSkip is why the tests that scan the fonts of the machine skip: go-text's
// fontscan has no font directories on js.
const systemFontsSkip = "fontscan has no system font directories on js"
