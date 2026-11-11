//go:build glutter_profile && !debug_mode

package foundation

// DebugMode is true in a build made with the tag debug_mode. Debug-only checks
// (Flutter's asserts, SPEC §12.9) and diagnostics run only then. It wins when
// both debug_mode and glutter_profile are given.
//
// Flutter: foundation/constants.dart kDebugMode
const DebugMode = false

// ProfileMode is true in a build made with the tag glutter_profile and not
// debug_mode.
//
// Flutter: foundation/constants.dart kProfileMode
const ProfileMode = true

// ReleaseMode is true in a build made without the tags debug_mode and
// glutter_profile, the default: a plain go build, go run or go install.
//
// Flutter: foundation/constants.dart kReleaseMode
const ReleaseMode = false
