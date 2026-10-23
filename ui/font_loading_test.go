package ui

import (
	"testing"

	"github.com/rm4n0s/errors"
)

// loadingEngine is a text engine that records the fonts it is given.
type loadingEngine struct {
	nopFactory
	loaded []string
	err    error
}

func (e *loadingEngine) LoadFontFromList(list []byte, family string) error {
	if e.err != nil {
		return e.err
	}
	e.loaded = append(e.loaded, family+":"+string(list))
	return nil
}

func TestPlatformDispatcherLoadFontFromList(t *testing.T) {
	tests := []struct {
		name       string
		engine     ParagraphFactory
		wantRoute  string
		wantLoaded []string
		wantNotify int
	}{
		{"an engine that loads fonts", &loadingEngine{}, "", []string{"Mono:data"}, 1},
		{"no engine", nil, "PlatformDispatcher.LoadFontFromList.FontLoadUnsupported", nil, 0},
		{"an engine that cannot load fonts", nopFactory{}, "PlatformDispatcher.LoadFontFromList.FontLoadUnsupported", nil, 0},
		{"an engine that fails", &loadingEngine{err: errors.New(tagFontDataInvalid, "no")}, "FontDataInvalid", nil, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewPlatformDispatcher(nil)
			if tt.engine != nil {
				d.SetTextEngine(tt.engine)
			}
			notified := 0
			d.OnFontsChanged = func() { notified++ }
			err := d.LoadFontFromList([]byte("data"), "Mono")
			if tt.wantRoute == "" {
				if err != nil {
					t.Fatalf("LoadFontFromList: %v", err)
				}
			} else {
				appErr, ok := errors.FromError(err)
				if !ok {
					t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
				}
				if !appErr.HasRoute(tt.wantRoute) {
					t.Errorf("unexpected failure path: %s", appErr.Route())
				}
			}
			if notified != tt.wantNotify {
				t.Errorf("OnFontsChanged called %d times, want %d", notified, tt.wantNotify)
			}
			if e, ok := tt.engine.(*loadingEngine); ok {
				if len(e.loaded) != len(tt.wantLoaded) || (len(e.loaded) > 0 && e.loaded[0] != tt.wantLoaded[0]) {
					t.Errorf("loaded = %v, want %v", e.loaded, tt.wantLoaded)
				}
			}
		})
	}
}

func TestPlatformDispatcherLoadFontFromListWithoutACallback(t *testing.T) {
	d := NewPlatformDispatcher(nil)
	d.SetTextEngine(&loadingEngine{})
	if err := d.LoadFontFromList([]byte("data"), "Mono"); err != nil {
		t.Errorf("LoadFontFromList without OnFontsChanged: %v", err)
	}
}

// queryEngine is a text engine that answers font queries.
type queryEngine struct{ nopFactory }

func (queryEngine) ResolveFont(family string, weight FontWeight, style FontStyle) (ResolvedFont, bool) {
	if family == "none" {
		return ResolvedFont{}, false
	}
	return ResolvedFont{Family: family + "!", Weight: int(weight), Italic: style == FontStyleItalic, Origin: FontOriginLoaded}, true
}

func (queryEngine) InstalledFontFamilies() []string { return []string{"A", "B"} }

func TestPlatformDispatcherFontQueries(t *testing.T) {
	t.Run("an engine that answers", func(t *testing.T) {
		d := NewPlatformDispatcher(nil)
		d.SetTextEngine(queryEngine{})
		got, ok := d.ResolveFont("Mono", FontWeightW700, FontStyleItalic)
		if !ok || got != (ResolvedFont{Family: "Mono!", Weight: 700, Italic: true, Origin: FontOriginLoaded}) {
			t.Errorf("ResolveFont = %+v, %v", got, ok)
		}
		if _, ok := d.ResolveFont("none", FontWeightW400, FontStyleNormal); ok {
			t.Error("ResolveFont found a family the engine does not have")
		}
		if got := d.InstalledFontFamilies(); len(got) != 2 || got[0] != "A" {
			t.Errorf("InstalledFontFamilies = %v", got)
		}
	})
	for name, engine := range map[string]ParagraphFactory{"no engine": nil, "an engine that cannot answer": nopFactory{}} {
		t.Run(name, func(t *testing.T) {
			d := NewPlatformDispatcher(nil)
			if engine != nil {
				d.SetTextEngine(engine)
			}
			if got, ok := d.ResolveFont("Mono", FontWeightW400, FontStyleNormal); ok || got != (ResolvedFont{}) {
				t.Errorf("ResolveFont = %+v, %v; want nothing", got, ok)
			}
			if got := d.InstalledFontFamilies(); got != nil {
				t.Errorf("InstalledFontFamilies = %v; want nil", got)
			}
		})
	}
}

func TestPlatformDispatcherDesktopFonts(t *testing.T) {
	d := NewPlatformDispatcher(nil)
	if f, s := d.UIFont(); f != "" || s != 0 {
		t.Errorf("UIFont = %q, %v before the platform says", f, s)
	}
	d.UpdatePlatformConfiguration(PlatformConfiguration{UIFontFamily: "Cantarell", UIFontSize: 11, MonospaceFontFamily: "Adwaita Mono", MonospaceFontSize: 12})
	if f, s := d.UIFont(); f != "Cantarell" || s != 11 {
		t.Errorf("UIFont = %q, %v", f, s)
	}
	if f, s := d.MonospaceFont(); f != "Adwaita Mono" || s != 12 {
		t.Errorf("MonospaceFont = %q, %v", f, s)
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagFontDataInvalid = "FontDataInvalid"
)
