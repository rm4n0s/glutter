package web

// The paths under which the web server of glutter-web and of the browser tests collects
// what the page sends (SPEC §23.8, §17).
const (
	// ScreenshotPath is where the engine's self-capture posts its PNG.
	//
	// Flutter: no counterpart
	ScreenshotPath = "/__glutter/screenshot"
	// ResultsPath is where a test program posts its results.
	//
	// Flutter: no counterpart
	ResultsPath = "/__glutter/results"
)
