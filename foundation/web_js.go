//go:build js && wasm

package foundation

// IsWeb is whether the program runs on the web: true under GOOS=js GOARCH=wasm.
//
// Flutter: foundation/constants.dart kIsWeb
const IsWeb = true
