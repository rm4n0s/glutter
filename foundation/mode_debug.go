//go:build !glutter_release && !glutter_profile

package foundation

// DebugMode is true in a debug build, the default. Debug-only checks (Flutter's
// asserts, SPEC §12.9) and diagnostics run only then.
//
// Flutter: foundation/constants.dart kDebugMode
const DebugMode = true

// ProfileMode is true in a build made with the tag glutter_profile.
//
// Flutter: foundation/constants.dart kProfileMode
const ProfileMode = false

// ReleaseMode is true in a build made with the tag glutter_release.
//
// Flutter: foundation/constants.dart kReleaseMode
const ReleaseMode = false
