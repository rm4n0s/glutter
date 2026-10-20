package material_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func TestDefaultMaterialLocalizationsStrings(t *testing.T) {
	l := DefaultMaterialLocalizations{}
	date := time.Date(2020, time.March, 4, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"drawer tooltip", l.OpenAppDrawerTooltip(), "Open navigation menu"},
		{"back tooltip", l.BackButtonTooltip(), "Back"},
		{"ok", l.OkButtonLabel(), "OK"},
		{"cancel", l.CancelButtonLabel(), "Cancel"},
		{"select all", l.SelectAllButtonLabel(), "Select all"},
		{"am", l.AnteMeridiemAbbreviation(), "AM"},
		{"pm", l.PostMeridiemAbbreviation(), "PM"},
		{"about tile", l.AboutListTileTitle("Foo"), "About Foo"},
		{"scrim hint", l.ScrimOnTapHint("Menu"), "Close Menu"},
		{"tab label", l.TabLabel(2, 5), "Tab 2 of 5"},
		{"no rows selected", l.SelectedRowCountTitle(0), "No items selected"},
		{"one row selected", l.SelectedRowCountTitle(1), "1 item selected"},
		{"rows selected", l.SelectedRowCountTitle(5), "5 items selected"},
		{"page rows", l.PageRowsInfoTitle(1, 10, 100, false), "1–10 of 100"},
		{"page rows approximate", l.PageRowsInfoTitle(1, 10, 100, true), "1–10 of about 100"},
		{"no characters", l.RemainingTextFieldCharacterCount(0), "No characters remaining"},
		{"one character", l.RemainingTextFieldCharacterCount(1), "1 character remaining"},
		{"characters", l.RemainingTextFieldCharacterCount(10), "10 characters remaining"},
		{"one license", l.LicensesPackageDetailText(1), "1 license."},
		{"licenses", l.LicensesPackageDetailText(3), "3 licenses."},
		{"decimal small", l.FormatDecimal(999), "999"},
		{"decimal thousands", l.FormatDecimal(1234567), "1,234,567"},
		{"decimal negative", l.FormatDecimal(-12345), "-12,345"},
		{"hour 12", l.FormatHour(TimeOfDay{14, 5}, false), "2"},
		{"hour 24", l.FormatHour(TimeOfDay{14, 5}, true), "14"},
		{"hour midnight", l.FormatHour(TimeOfDay{0, 5}, false), "12"},
		{"minute", l.FormatMinute(TimeOfDay{14, 5}), "05"},
		{"time 12", l.FormatTimeOfDay(TimeOfDay{14, 5}, false), "2:05 PM"},
		{"time 24", l.FormatTimeOfDay(TimeOfDay{14, 5}, true), "14:05"},
		{"time am", l.FormatTimeOfDay(TimeOfDay{9, 30}, false), "9:30 AM"},
		{"year", l.FormatYear(date), "2020"},
		{"compact date", l.FormatCompactDate(date), "03/04/2020"},
		{"short date", l.FormatShortDate(date), "Mar 4, 2020"},
		{"medium date", l.FormatMediumDate(date), "Wed, Mar 4"},
		{"full date", l.FormatFullDate(date), "Wednesday, March 4, 2020"},
		{"month year", l.FormatMonthYear(date), "March 2020"},
		{"short month day", l.FormatShortMonthDay(date), "Mar 4"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
	if l.TimeOfDayFormat(true) != TimeOfDayFormatHHColonMM || l.TimeOfDayFormat(false) != TimeOfDayFormatHColonMMSpaceA {
		t.Error("wrong time of day formats")
	}
	if l.FirstDayOfWeekIndex() != 0 || len(l.NarrowWeekdays()) != 7 || l.ScriptCategory() != ScriptCategoryEnglishLike {
		t.Error("wrong week or script data")
	}
}

func TestDefaultMaterialLocalizationsParseCompactDate(t *testing.T) {
	l := DefaultMaterialLocalizations{}
	tests := []struct {
		in   string
		want time.Time
		ok   bool
	}{
		{"03/04/2020", time.Date(2020, 3, 4, 0, 0, 0, 0, time.Local), true},
		{"02/29/2020", time.Date(2020, 2, 29, 0, 0, 0, 0, time.Local), true},
		{"02/30/2020", time.Time{}, false},
		{"13/01/2020", time.Time{}, false},
		{"3/4/2020", time.Date(2020, 3, 4, 0, 0, 0, 0, time.Local), true},
		{"hello", time.Time{}, false},
		{"", time.Time{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, ok := l.ParseCompactDate(tt.in)
			if ok != tt.ok || (ok && !got.Equal(tt.want)) {
				t.Errorf("ParseCompactDate(%q) = %v, %v; want %v, %v", tt.in, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestMaterialLocalizationsOf(t *testing.T) {
	tests := []struct {
		name       string
		delegates  []widgets.LocalizationsDelegate
		locale     ui.Locale
		wantFound  bool
		wantTipFor string
	}{
		{"english", []widgets.LocalizationsDelegate{widgets.DefaultWidgetsLocalizationsDelegate, DefaultMaterialLocalizationsDelegate}, ui.NewLocale("en", "US"), true, "Back"},
		{"other language is not supported", []widgets.LocalizationsDelegate{widgets.DefaultWidgetsLocalizationsDelegate, DefaultMaterialLocalizationsDelegate}, ui.NewLocale("fr", "FR"), false, ""},
		{"no delegate", []widgets.LocalizationsDelegate{widgets.DefaultWidgetsLocalizationsDelegate}, ui.NewLocale("en", "US"), false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			var found bool
			var tip string
			ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR,
				widgets.NewLocalizations(tt.locale, tt.delegates, widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
					l, ok := MaterialLocalizationsMaybeOf(bc)
					found = ok
					if ok {
						tip = l.BackButtonTooltip()
					}
					return widgets.NewSizedBox(nil)
				}))))
			ts.Pump()
			if found != tt.wantFound || tip != tt.wantTipFor {
				t.Errorf("found = %v, tooltip = %q; want %v, %q", found, tip, tt.wantFound, tt.wantTipFor)
			}
		})
	}
}

func TestDebugCheckHasMaterialLocalizations(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	var err error
	ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		err = recoverError(func() { DebugCheckHasMaterialLocalizations(bc) })
		return widgets.NewSizedBox(nil)
	})))
	wantRoute(t, err, "DebugCheckHasMaterialLocalizations.MaterialLocalizationsMissing")
}
