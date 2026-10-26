package androidtool

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// The prefixes of the temporary directories that the builder makes: the work directory of Build
// (libapp.so for each ABI, the unaligned and the aligned APK: over 100 MB for two ABIs) and the one of
// CompileDex.
const (
	apkWorkPrefix = "glutter-apk-"
	dexWorkPrefix = "glutter-dex-"
)

// staleTempAge is how old (by its modification time) a work directory must be before the next build
// removes it as the leftover of a build that was killed. A build takes minutes, and the time of a
// directory changes when an entry is added to it, so two hours is long after any build that still runs.
const staleTempAge = 2 * time.Hour

// sweepStaleTempDirs removes the directories of tmp whose name starts with one of the prefixes and
// that were last changed before now-age, and returns their paths. A build removes its work directory
// itself, on every path that returns (a deferred RemoveAll), but a process that is killed (SIGKILL,
// the memory guard, a test binary that ends with SIGINT or a timeout) never runs its defers: this
// sweep, at the start of the next build, is what keeps such leftovers from piling up in /tmp, which is
// memory. Files, and directories that are newer or have other names, are left alone. Errors are not
// reported: a directory that cannot be removed now is tried again by the next build.
func sweepStaleTempDirs(tmp string, now time.Time, age time.Duration, prefixes ...string) []string {
	entries, err := os.ReadDir(tmp)
	if err != nil {
		return nil
	}
	var removed []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		matches := false
		for _, p := range prefixes {
			if strings.HasPrefix(e.Name(), p) {
				matches = true
				break
			}
		}
		if !matches {
			continue
		}
		info, err := e.Info()
		if err != nil || now.Sub(info.ModTime()) < age {
			continue
		}
		path := filepath.Join(tmp, e.Name())
		if os.RemoveAll(path) == nil {
			removed = append(removed, path)
		}
	}
	return removed
}
