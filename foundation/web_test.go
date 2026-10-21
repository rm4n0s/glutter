package foundation

import "testing"

// TestIsWebFollowsTheBuild checks IsWeb against the build: false in the normal test
// run, true under js/wasm (webTarget is set by web_js_test.go).
func TestIsWebFollowsTheBuild(t *testing.T) {
	if IsWeb != webTarget {
		t.Errorf("IsWeb = %v, want %v for this build", IsWeb, webTarget)
	}
}
