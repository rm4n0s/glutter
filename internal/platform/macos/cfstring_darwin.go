//go:build darwin && !ios

package macos

import "github.com/rm4n0s/glutter/internal/platform/darwin/cf"

// cfString makes a CFString (+1) and returns it as the uintptr that the C functions of the backend take.
func cfString(s string) uintptr { return uintptr(cf.NewString(s)) }

// releaseCF releases a CF object that cfString made.
func releaseCF(r uintptr) { cf.Release(cf.Ref(r)) }
