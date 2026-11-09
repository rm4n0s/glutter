package cupertino_test

import (
	"testing"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
)

// Ported from packages/flutter/test/cupertino/icon_theme_data_test.dart: IconTheme.of gives
// the data that was set, and the data of CupertinoIconThemeData resolves its dynamic color.
func TestIconThemeOfWorks(t *testing.T) {
	data := cupertino.NewIconThemeData(cupertino.IconThemeDataOpts{
		Size: foundation.Some(16.0), Fill: foundation.Some(0.0), Weight: foundation.Some(400.0), Grade: foundation.Some(0.0),
		OpticalSize: foundation.Some(48.0), Color: foundation.Some(ui.NewColor(0xAAAAAAAA)), Opacity: foundation.Some(0.5),
		ApplyTextScaling: foundation.Some(true),
	})
	ts := gluttertest.New(t)
	var retrieved cupertino.IconThemeData
	ts.PumpWidget(cupertino.NewIconTheme(data, cupertino.NewBuilder(func(bc cupertino.BuildContext) cupertino.Widget {
		retrieved = cupertino.IconThemeOf(bc)
		return cupertino.NewSizedBox(nil)
	})))
	if !retrieved.Equal(data) {
		t.Errorf("retrieved %v, want %v", retrieved, data)
	}

	ts.PumpWidget(cupertino.NewIconTheme(cupertino.NewCupertinoIconThemeData(cupertino.CupertinoIconThemeDataOpts{Color: cupertino.CupertinoColors.SystemBlue}),
		cupertino.NewMediaQuery(cupertino.NewMediaQueryData(cupertino.MediaQueryDataOpts{PlatformBrightness: ui.BrightnessDark}),
			cupertino.NewBuilder(func(bc cupertino.BuildContext) cupertino.Widget {
				retrieved = cupertino.IconThemeOf(bc)
				return cupertino.NewSizedBox(nil)
			}))))
	if got, _ := retrieved.Color().Get(); got != cupertino.CupertinoColors.SystemBlue.DarkColor {
		t.Errorf("color = %v, want %v", got, cupertino.CupertinoColors.SystemBlue.DarkColor)
	}
}

func TestCupertinoIconThemeDataColorFollowsCopyWithAndMerge(t *testing.T) {
	dyn := cupertino.NewCupertinoIconThemeData(cupertino.CupertinoIconThemeDataOpts{Color: cupertino.CupertinoColors.SystemRed})
	plain := dyn.CopyWith(cupertino.IconThemeDataOpts{Color: foundation.Some(ui.NewColor(0xFF010203))})
	if got, _ := plain.Color().Get(); got != ui.NewColor(0xFF010203) {
		t.Errorf("a color given to CopyWith is %v", got)
	}
	if got, _ := dyn.CopyWith(cupertino.IconThemeDataOpts{Size: foundation.Some(30.0)}).Color().Get(); got != cupertino.CupertinoColors.SystemRed.Color {
		t.Errorf("CopyWith without a color changed it to %v", got)
	}
	merged := cupertino.NewIconThemeData().Merge(&dyn)
	if got, _ := merged.Color().Get(); got != cupertino.CupertinoColors.SystemRed.Color {
		t.Errorf("a merged dynamic color is %v", got)
	}
	if dyn.Equal(plain) {
		t.Error("a dynamic and a plain color are equal")
	}
}
