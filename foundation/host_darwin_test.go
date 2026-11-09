//go:build darwin && !ios

package foundation

// hostWant is the platform that the framework takes as the host on this build (host_darwin.go).
const hostWant = TargetPlatformMacOS
