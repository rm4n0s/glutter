//go:build js

package examples

import "testing"

// buildForWeb has nothing to run under Node (SPEC §17): the go command cannot start from a
// WebAssembly program, and the examples are built for js/wasm by the test on the host.
func buildForWeb(t *testing.T, name string) {
	t.Helper()
	t.Skip("builds the example with the go command, which a program running under Node cannot start")
}
