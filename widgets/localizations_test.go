package widgets_test

import (
	"context"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

type greetings struct{ hello string }

func greetingsDelegate(loads *int) LocalizationsDelegate {
	return &FuncLocalizationsDelegate[*greetings]{
		Supported: func(l ui.Locale) bool { return l.LanguageCode == "en" || l.LanguageCode == "fr" },
		LoadFunc: func(_ context.Context, l ui.Locale) (*greetings, error) {
			if loads != nil {
				*loads++
			}
			if l.LanguageCode == "fr" {
				return &greetings{"bonjour"}, nil
			}
			return &greetings{"hello"}, nil
		},
		Reload: func(old LocalizationsDelegate) bool { return false },
		Name:   "greetings",
	}
}

func localized(locale ui.Locale, delegates []LocalizationsDelegate, child Widget) Widget {
	return NewLocalizations(locale, append([]LocalizationsDelegate{DefaultWidgetsLocalizationsDelegate}, delegates...), child)
}

func TestLocalizationsProvidesResourcesAndLocale(t *testing.T) {
	tests := []struct {
		name   string
		locale ui.Locale
		want   string
	}{
		{"english", ui.NewLocale("en", "US"), "hello"},
		{"french", ui.NewLocale("fr"), "bonjour"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			var got *greetings
			var gotLocale ui.Locale
			var gotDir ui.TextDirection
			ts.PumpWidget(localized(tt.locale, []LocalizationsDelegate{greetingsDelegate(nil)}, NewBuilder(func(bc BuildContext) Widget {
				got, _ = LocalizationsOf[*greetings](bc)
				gotLocale = LocalizationsLocaleOf(bc)
				gotDir = DirectionalityOf(bc)
				return newLeaf()
			})))
			if got == nil || got.hello != tt.want {
				t.Errorf("resources = %v, want %q", got, tt.want)
			}
			if gotLocale != tt.locale {
				t.Errorf("locale = %v, want %v", gotLocale, tt.locale)
			}
			if gotDir != ui.TextDirectionLTR {
				t.Errorf("direction = %v, want ltr", gotDir)
			}
		})
	}
}

func TestLocalizationsLoadsTheFirstDelegateOfEachType(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	first, second := 0, 0
	var got *greetings
	ts.PumpWidget(localized(ui.NewLocale("en"), []LocalizationsDelegate{greetingsDelegate(&first), greetingsDelegate(&second)},
		NewBuilder(func(bc BuildContext) Widget { got, _ = LocalizationsOf[*greetings](bc); return newLeaf() })))
	if first != 1 || second != 0 || got == nil {
		t.Errorf("loads = %d and %d, resources = %v; want 1 and 0", first, second, got)
	}
}

func TestLocalizationsSkipsUnsupportedLocales(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	found := true
	ts.PumpWidget(localized(ui.NewLocale("de"), []LocalizationsDelegate{greetingsDelegate(nil)},
		NewBuilder(func(bc BuildContext) Widget { _, found = LocalizationsOf[*greetings](bc); return newLeaf() })))
	if found {
		t.Error("the unsupported locale loaded resources")
	}
}

func TestLocalizationsReloadsWhenTheLocaleChanges(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	loads := 0
	var got *greetings
	child := NewBuilder(func(bc BuildContext) Widget { got, _ = LocalizationsOf[*greetings](bc); return newLeaf() })
	d := greetingsDelegate(&loads)
	ts.PumpWidget(localized(ui.NewLocale("en"), []LocalizationsDelegate{d}, child))
	ts.PumpWidget(localized(ui.NewLocale("en"), []LocalizationsDelegate{d}, child))
	if loads != 1 {
		t.Errorf("loads = %d after an equal update, want 1", loads)
	}
	ts.PumpWidget(localized(ui.NewLocale("fr"), []LocalizationsDelegate{d}, child))
	if loads != 2 || got.hello != "bonjour" {
		t.Errorf("loads = %d, resources = %v after the locale changed", loads, got)
	}
}

func TestLocalizationsDependentsRebuildOnReload(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	builds := 0
	child := NewBuilder(func(bc BuildContext) Widget { builds++; LocalizationsLocaleOf(bc); return newLeaf() })
	d := greetingsDelegate(nil)
	ts.PumpWidget(localized(ui.NewLocale("en"), []LocalizationsDelegate{d}, child))
	ts.PumpWidget(localized(ui.NewLocale("fr"), []LocalizationsDelegate{d}, child))
	if builds != 2 {
		t.Errorf("builds = %d, want 2", builds)
	}
}

func TestLocalizationsOverride(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	var gotLocale ui.Locale
	var got *greetings
	ts.PumpWidget(localized(ui.NewLocale("en"), []LocalizationsDelegate{greetingsDelegate(nil)},
		NewBuilder(func(bc BuildContext) Widget {
			return NewLocalizationsOverride(bc, foundation.Some(ui.NewLocale("fr")), nil, NewBuilder(func(bc BuildContext) Widget {
				got, _ = LocalizationsOf[*greetings](bc)
				gotLocale = LocalizationsLocaleOf(bc)
				return newLeaf()
			}))
		})))
	if gotLocale != ui.NewLocale("fr") || got == nil || got.hello != "bonjour" {
		t.Errorf("locale = %v, resources = %v", gotLocale, got)
	}
}

func TestLocalizationsMissing(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	maybe, ok := ui.Locale{}, true
	ts.PumpWidget(NewBuilder(func(bc BuildContext) Widget { maybe, ok = LocalizationsMaybeLocaleOf(bc); return newLeaf() }))
	if ok || !maybe.IsZero() {
		t.Errorf("MaybeLocaleOf = %v, %v", maybe, ok)
	}
	tests := []struct {
		name string
		fn   func(bc BuildContext)
		tag  string
	}{
		{"locale", func(bc BuildContext) { LocalizationsLocaleOf(bc) }, "LocalizationsMissing"},
		{"widgets localizations", func(bc BuildContext) { WidgetsLocalizationsOf(bc) }, "WidgetsLocalizationsMissing"},
		{"override", func(bc BuildContext) { NewLocalizationsOverride(bc, foundation.None[ui.Locale](), nil, nil) }, "LocalizationsMissing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			ts.PumpWidget(NewBuilder(func(bc BuildContext) Widget { tt.fn(bc); return newLeaf() }))
			_ = requireErrorTag(t, ts.TakeError(), tt.tag)
		})
	}
}

func TestLocalizationsNeedsAWidgetsDelegate(t *testing.T) {
	skipUnlessDebug(t)
	err := panicOf(func() { NewLocalizations(ui.NewLocale("en"), []LocalizationsDelegate{greetingsDelegate(nil)}, nil) })
	if err == nil {
		t.Fatal("no panic")
	}
	_ = requireErrorTag(t, err.(error), "LocalizationsNoWidgetsDelegate")
}

func TestLocalizationsLoadFailure(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	boom := errors.New("Boom", "boom")
	bad := &FuncLocalizationsDelegate[*greetings]{LoadFunc: func(context.Context, ui.Locale) (*greetings, error) { return nil, boom }}
	ts.PumpWidget(localized(ui.NewLocale("en"), []LocalizationsDelegate{bad}, newLeaf()))
	appErr := requireErrorTag(t, ts.TakeError(), "LocalizationsLoadFailed")
	if !errors.Is(appErr, boom) {
		t.Errorf("error %v does not wrap the cause", appErr)
	}
}

func TestDefaultWidgetsLocalizations(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	var l WidgetsLocalizations
	ts.PumpWidget(localized(ui.NewLocale("en"), nil, NewBuilder(func(bc BuildContext) Widget { l = WidgetsLocalizationsOf(bc); return newLeaf() })))
	tests := []struct {
		name      string
		got, want string
	}{
		{"copy", l.CopyButtonLabel(), "Copy"},
		{"cut", l.CutButtonLabel(), "Cut"},
		{"paste", l.PasteButtonLabel(), "Paste"},
		{"select all", l.SelectAllButtonLabel(), "Select all"},
		{"move up", l.ReorderItemUp(), "Move up"},
		{"to start", l.ReorderItemToStart(), "Move to the start"},
		{"no results", l.NoResultsFound(), "No results found"},
		{"unselected", l.RadioButtonUnselectedLabel(), "Not selected"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
		}
	}
	if l.TextDirection() != ui.TextDirectionLTR {
		t.Error("the default direction is not ltr")
	}
}

func TestImageConfigurationHasTheLocale(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	var c painting.ImageConfiguration
	ts.PumpWidget(localized(ui.NewLocale("fr", "CA"), nil, NewBuilder(func(bc BuildContext) Widget {
		c = CreateLocalImageConfiguration(bc, foundation.None[ui.Size]())
		return newLeaf()
	})))
	if got := c.Locale.OrElse(ui.Locale{}); got != ui.NewLocale("fr", "CA") {
		t.Errorf("locale = %v", got)
	}
}

func TestRichTextTakesTheLocaleFromLocalizations(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(localized(ui.NewLocale("ja"), nil, NewText("x")))
	if got := richOf(ts).Locale().OrElse(ui.Locale{}); got != ui.NewLocale("ja") {
		t.Errorf("paragraph locale = %v, want ja", got)
	}
}
