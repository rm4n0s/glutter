//go:build !windows && !(darwin && !ios)

package foundation

// hostWant is the platform that the framework takes as the host on this build (web_other.go).
const hostWant = TargetPlatformLinux
