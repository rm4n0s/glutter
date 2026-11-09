//go:build darwin && !ios

package darwin

// uiFramework is the framework that Core Text's fonts must not be made before: AppKit's NSFont is
// toll-free bridged to CTFont, so a font that Core Text makes before AppKit is loaded is a plain
// CoreFoundation object, and AppKit, which keeps and shares the system fonts, later sends it NSFont's
// messages and raises an exception (found in step 33.6).
const uiFramework = "AppKit"
