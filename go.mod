module github.com/rm4n0s/glutter

go 1.27

require (
	github.com/ebitengine/purego v0.11.2
	github.com/go-text/typesetting v0.3.4
	github.com/godbus/dbus/v5 v5.2.4
	github.com/rm4n0s/errors v0.2.0
	golang.org/x/image v0.47.0
	golang.org/x/text v0.43.0
)

require golang.org/x/sys v0.49.0 // indirect

// A copy of go-text/typesetting v0.3.4 that parses the 'gvar' table of a variable font one glyph at a time, when the glyph is
// first drawn with variations, instead of every glyph when the font is opened (SPEC §10, step 37.1; third_party/typesetting).
replace github.com/go-text/typesetting => ./third_party/typesetting
