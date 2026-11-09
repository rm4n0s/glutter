//go:build ios

package darwin

// uiFramework is the framework that Core Text's fonts must not be made before: UIKit's UIFont is
// bridged to CTFont as AppKit's NSFont is on macOS (see uiframework_macos.go).
const uiFramework = "UIKit"
