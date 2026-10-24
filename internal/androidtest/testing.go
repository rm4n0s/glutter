package androidtest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

var (
	sharedMu sync.Mutex
	shared   = map[string]*Emulator{}
)

// Shared returns the emulator of a virtual device for a device test, starting it the first time
// a test of the package asks: booting takes tens of seconds, so the tests of a package share one,
// and Run stops it when they are done. A missing
// emulator or image fails the test: a device test that silently skipped would prove nothing.
//
// Flutter: no counterpart
func Shared(t testing.TB, avd string) *Emulator {
	t.Helper()
	sharedMu.Lock()
	defer sharedMu.Unlock()
	if e, ok := shared[avd]; ok {
		return e
	}
	// A shared emulator outlives the test that started it, so it gets a context of its own.
	e, err := Start(context.Background(), Options{AVD: avd})
	if err != nil {
		t.Fatalf("starting %s: %v", avd, err)
	}
	shared[avd] = e
	return e
}

// ReleaseShared stops the emulator of the AVD that Shared started, when its tests are done, so that
// fewer emulators (about 5 GB of memory each) run at once; a later Shared starts it again.
//
// Flutter: no counterpart
func ReleaseShared(avd string) {
	sharedMu.Lock()
	defer sharedMu.Unlock()
	if e, ok := shared[avd]; ok {
		e.Stop()
		delete(shared, avd)
	}
}

// StopShared stops the emulators that Shared started.
//
// Flutter: no counterpart
func StopShared() {
	sharedMu.Lock()
	defer sharedMu.Unlock()
	for name, e := range shared {
		e.Stop()
		delete(shared, name)
	}
}

// Run is for the TestMain of a package of device tests: it runs the tests, stops the emulators
// they started, and returns the exit code.
//
//	func TestMain(m *testing.M) { os.Exit(androidtest.Run(m)) }
//
// Flutter: no counterpart
func Run(m *testing.M) int {
	code := m.Run()
	StopShared()
	if root, err := ModuleRoot(); err == nil {
		if left := Leftovers(filepath.Join(root, ".android-test")); len(left) > 0 {
			fmt.Fprintf(os.Stderr, "androidtest: processes are still running after the tests: %v\n", left)
			return 1
		}
	}
	return code
}
