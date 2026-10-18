package platform

import "runtime"

// init runs on the main goroutine, so locking the OS thread here keeps the main
// goroutine on the main OS thread for the life of the process. That thread is
// the UI thread (docs/SPEC.md §8.4).
func init() {
	runtime.LockOSThread()
}
