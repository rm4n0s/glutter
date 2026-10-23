package androiddex

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/androidtool"
)

func TestRunFailures(t *testing.T) {
	err := Run([]string{"-nope"})
	if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("Run."+tagAndroidDexBadFlags) {
		t.Errorf("bad flags: %v", err)
	}
	t.Setenv("ANDROID_HOME", "")
	t.Setenv("ANDROID_SDK_ROOT", "")
	err = Run(nil)
	if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("Run->FindHome."+androidtool.TagAndroidSDKNotFound) {
		t.Errorf("no SDK: %v", err)
	}
	home := t.TempDir()
	t.Setenv("ANDROID_HOME", home)
	err = Run(nil)
	if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("Run->FindBuildTools."+androidtool.TagAndroidBuildToolsNotFound) {
		t.Errorf("no build tools: %v", err)
	}
}
