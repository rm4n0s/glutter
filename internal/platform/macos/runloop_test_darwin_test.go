//go:build darwin && !ios && glutter_display

package macos

import (
	"time"

	"github.com/rm4n0s/glutter/internal/platform/darwin/cf"
)

// cfModeNamed is the run loop mode of the name, as a CFString that lives as long as the test binary.
func cfModeNamed(name string) cf.Ref { return cf.NewString(name) }

// runLoopOnce runs the current run loop in the mode for at most d, as a menu's tracking does.
func runLoopOnce(mode cf.Ref, d time.Duration) { cf.RunInMode(mode, d, false) }
