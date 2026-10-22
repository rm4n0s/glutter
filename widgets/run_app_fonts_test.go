package widgets

import (
	"context"
	"math"
	"testing"
	"testing/fstest"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/renderingtest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
)

// monoAdvance is the advance of a glyph of Go Mono, in em.
const monoAdvance = 1229.0 / 2048.0

// freshEngine is a text engine of its own, so that loading a font does not change the shared one.
func freshEngine(t *testing.T) ui.ParagraphFactory {
	t.Helper()
	e, err := renderingtest.NewTextEngine()
	if err != nil {
		t.Fatal(err)
	}
	return e
}

// monoWidth is the width of three letters in the family at 10 pixels, with Roboto as the fallback: 3 *
// 0.6 * 10 when the family is there.
func monoWidth(d *ui.PlatformDispatcher, family string) float64 {
	b := ui.NewParagraphBuilder(d.TextEngine(), nil)
	b.PushStyle(ui.NewTextStyle(ui.TextStyleOpts{FontFamily: family, FontFamilyFallback: []string{"Roboto"}, FontSize: ui.Ptr(10.0)}))
	b.AddText("iWi")
	p := b.Build()
	p.Layout(ui.NewParagraphConstraints(1000))
	return p.LongestLine()
}

func TestLoadFontSources(t *testing.T) {
	assets := fstest.MapFS{"fonts/GoMono.ttf": {Data: gomono.TTF}, "fonts/notafont.ttf": {Data: []byte("nope")}}
	tests := []struct {
		name    string
		sources []FontSource
		assets  fstest.MapFS
		// route is the route the error must have; empty means success.
		route string
		// mono says whether "Go Mono" has the font afterwards.
		mono bool
	}{
		{"bytes", []FontSource{{Family: "Go Mono", Data: gomono.TTF}}, nil, "", true},
		{"an asset", []FontSource{{Family: "Go Mono", Asset: "fonts/GoMono.ttf"}}, assets, "", true},
		{"the family inside the font", []FontSource{{Data: gomono.TTF}}, nil, "", true},
		{"regular and bold of one family", []FontSource{{Family: "Go Mono", Data: gomono.TTF}, {Family: "Go Mono", Data: gomonobold.TTF}}, nil, "", true},
		{"no sources", nil, nil, "", false},
		{"both Data and Asset", []FontSource{{Family: "Go Mono", Data: gomono.TTF, Asset: "fonts/GoMono.ttf"}}, assets, "loadFontSources.FontSourceInvalid", false},
		{"neither Data nor Asset", []FontSource{{Family: "Go Mono"}}, assets, "loadFontSources.FontSourceInvalid", false},
		{"an asset without Assets", []FontSource{{Family: "Go Mono", Asset: "fonts/GoMono.ttf"}}, nil, "loadFontSources.FontSourceInvalid", false},
		{"a missing asset", []FontSource{{Family: "Go Mono", Asset: "fonts/none.ttf"}}, assets, "AssetNotFound", false},
		{"data that is no font", []FontSource{{Family: "Go Mono", Asset: "fonts/notafont.ttf"}}, assets, "FontDataInvalid", false},
		{"the sources before a bad one stay loaded", []FontSource{{Family: "Go Mono", Data: gomono.TTF}, {Family: "Bad", Data: []byte("nope")}}, nil, "FontDataInvalid", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := ui.NewPlatformDispatcher(nil)
			d.SetTextEngine(freshEngine(t))
			var assetFS fstest.MapFS
			if tt.assets != nil {
				assetFS = tt.assets
			}
			var err error
			if assetFS != nil {
				err = loadFontSources(context.Background(), d, tt.sources, assetFS)
			} else {
				err = loadFontSources(context.Background(), d, tt.sources, nil)
			}
			if tt.route == "" {
				if err != nil {
					t.Fatalf("loadFontSources: %v", err)
				}
			} else {
				appErr, ok := errors.FromError(err)
				if !ok {
					t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
				}
				if !appErr.HasRoute(tt.route) {
					t.Errorf("unexpected failure path: %s", appErr.Route())
				}
			}
			got := monoWidth(d, "Go Mono")
			want := 3 * monoAdvance * 10
			if tt.mono && math.Abs(got-want) > 0.01 {
				t.Errorf("width = %v, want %v: the family is not loaded", got, want)
			}
			if !tt.mono && math.Abs(got-want) < 0.5 {
				t.Errorf("width = %v: the family is loaded", got)
			}
		})
	}
}

func TestMissingAssetTag(t *testing.T) {
	d := ui.NewPlatformDispatcher(nil)
	d.SetTextEngine(freshEngine(t))
	err := loadFontSources(context.Background(), d, []FontSource{{Family: "X", Asset: "none.ttf"}}, fstest.MapFS{})
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "AssetNotFound" {
		t.Errorf("err = %v, want AssetNotFound", err)
	}
}

// findParagraph returns the first RenderParagraph below the render views.
func findParagraph(b *WidgetsBinding) *rendering.RenderParagraph {
	var found *rendering.RenderParagraph
	var visit rendering.RenderObjectVisitor
	visit = func(child rendering.RenderObject) {
		if p, ok := child.(*rendering.RenderParagraph); ok && found == nil {
			found = p
		}
		child.VisitChildren(visit)
	}
	for _, v := range b.Renderer().RenderViews() {
		v.VisitChildren(visit)
	}
	return found
}

func TestLoadFontFromListLaysTheTextOutAgain(t *testing.T) {
	bt := newBindingTree(t, bindingTreeOpts{textEngine: freshEngine(t)})
	style := painting.NewTextStyle(painting.TextStyleOpts{FontFamily: "Go Mono", FontSize: foundation.Some(20.0)})
	bt.attach(NewDirectionality(ui.TextDirectionLTR, NewAlign(NewText("iiiiWWWW", TextOpts{Style: style}), AlignOpts{Alignment: painting.AlignmentTopLeft})))
	bt.frame()
	p := findParagraph(bt.b)
	if p == nil {
		t.Fatal("no RenderParagraph in the tree")
	}
	before := p.Size().Width
	if err := bt.d.LoadFontFromList(gomono.TTF, "Go Mono"); err != nil {
		t.Fatalf("LoadFontFromList: %v", err)
	}
	if !p.DebugNeedsLayout() {
		t.Error("the paragraph was not marked as needing layout")
	}
	bt.frame()
	want := 8 * monoAdvance * 20
	if after := p.Size().Width; math.Abs(after-want) > 0.5 || math.Abs(before-want) < 0.5 {
		t.Errorf("width before = %v, after = %v, want %v after", before, after, want)
	}
}

func TestLoadFontFromListWithoutAnEngine(t *testing.T) {
	d := ui.NewPlatformDispatcher(nil)
	appErr, ok := errors.FromError(d.LoadFontFromList(gomono.TTF, "X"))
	if !ok || !appErr.HasRoute("PlatformDispatcher.LoadFontFromList.FontLoadUnsupported") {
		t.Errorf("err = %v", appErr)
	}
}
