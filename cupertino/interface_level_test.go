package cupertino_test

import (
	"testing"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
)

// The interface level has no test file of its own in Flutter (colors_test.dart covers it);
// these are the cases of its Of and maybeOf, and of its notifications.

func TestCupertinoUserInterfaceLevelOfFailsWithoutOne(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	var maybe bool
	ts.PumpWidget(cupertino.NewBuilder(func(bc cupertino.BuildContext) cupertino.Widget {
		_, maybe = cupertino.CupertinoUserInterfaceLevelMaybeOf(bc)
		cupertino.CupertinoUserInterfaceLevelOf(bc)
		return cupertino.NewSizedBox(nil)
	}))
	if maybe {
		t.Error("MaybeOf found a level in a tree that has none")
	}
	appErr := requireErrorTag(t, firstError(t, ts), "CupertinoUserInterfaceLevelMissing")
	if appErr.Message == "" {
		t.Error("the error has no message")
	}
}

func TestCupertinoUserInterfaceLevelOfFindsTheLevelAndNotifies(t *testing.T) {
	ts := gluttertest.New(t)
	builds := 0
	var got cupertino.CupertinoUserInterfaceLevelData
	probe := cupertino.NewBuilder(func(bc cupertino.BuildContext) cupertino.Widget {
		builds++
		got = cupertino.CupertinoUserInterfaceLevelOf(bc)
		return cupertino.NewSizedBox(nil)
	})
	tests := []struct {
		name       string
		level      cupertino.CupertinoUserInterfaceLevelData
		want       cupertino.CupertinoUserInterfaceLevelData
		wantBuilds int
	}{
		{"base", cupertino.CupertinoUserInterfaceLevelBase, cupertino.CupertinoUserInterfaceLevelBase, 1},
		{"the same level does not rebuild the dependents", cupertino.CupertinoUserInterfaceLevelBase, cupertino.CupertinoUserInterfaceLevelBase, 1},
		{"elevated rebuilds them", cupertino.CupertinoUserInterfaceLevelElevated, cupertino.CupertinoUserInterfaceLevelElevated, 2},
		{"unspecified is the base level", cupertino.CupertinoUserInterfaceLevelDataUnspecified, cupertino.CupertinoUserInterfaceLevelBase, 3},
	}
	for _, tt := range tests {
		ts.PumpWidget(cupertino.NewCupertinoUserInterfaceLevel(tt.level, probe))
		if got != tt.want || builds != tt.wantBuilds {
			t.Errorf("%s: level %v after %d builds, want %v after %d", tt.name, got.Name(), builds, tt.want.Name(), tt.wantBuilds)
		}
	}
}

func TestCupertinoUserInterfaceLevelDiagnostics(t *testing.T) {
	skipUnlessDebug(t) // diagnostics are empty in a release build
	b := foundation.NewDiagnosticPropertiesBuilder()
	cupertino.NewCupertinoUserInterfaceLevel(cupertino.CupertinoUserInterfaceLevelElevated, cupertino.NewSizedBox(nil)).DebugFillProperties(b)
	if len(b.Properties) != 1 {
		t.Fatalf("properties = %v", b.Properties)
	}
	if got := b.Properties[0].ToDescription(nil); got != "elevated" {
		t.Errorf("description = %q, want elevated", got)
	}
	if cupertino.CupertinoUserInterfaceLevelData(9).Name() != "CupertinoUserInterfaceLevelData(9)" {
		t.Error("an unknown level is not named by its number")
	}
}
