package layout

import "testing"

// CBDTEngine is an engine with Roboto and the test color bitmap font as the family CBDTFamily, for the
// tests of other packages' users of the engine (the external test package).
func CBDTEngine(t *testing.T) *Engine {
	t.Helper()
	return cbdtEngine(t, testStrikes, func(err error) { t.Errorf("font error: %v", err) })
}

// CBDTFamily is the family of the test color bitmap font: 'A', U+1F600 and U+1F601 are squares of color.
const CBDTFamily = cbdtFamily
