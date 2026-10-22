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
		{"an engine that fails", &loadingEngine{err: errors.New("FontDataInvalid", "no")}, "FontDataInvalid", nil, 0},
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
