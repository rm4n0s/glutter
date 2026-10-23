package services

import (
	"context"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

// FontLoader loads fonts into the text engine under one family name while the
// app runs: add the fonts, then call Load once. The fonts can be bytes in memory,
// the result of a function, or assets of an AssetBundle. A family can have several
// fonts (regular, bold, italic...); each font tells its own weight and slant.
//
// Flutter: services/font_loader.dart FontLoader
type FontLoader struct {
	family string
	fonts  []func(ctx context.Context) ([]byte, error)
	loaded bool
}

// NewFontLoader creates a loader for the family. An empty family takes the family
// name inside each font.
//
// Flutter: services/font_loader.dart FontLoader.new
func NewFontLoader(family string) *FontLoader { return &FontLoader{family: family} }

// Family is the family name the fonts are loaded under.
//
// Flutter: services/font_loader.dart FontLoader.family
func (l *FontLoader) Family() string { return l.family }

// AddFont adds a font that load produces when Load runs. Flutter's addFont takes a
// Future. It panics with FontLoaderReused after Load.
//
// Flutter: services/font_loader.dart FontLoader.addFont
func (l *FontLoader) AddFont(load func(ctx context.Context) ([]byte, error)) {
	l.assertNotLoaded()
	l.fonts = append(l.fonts, load)
}

// AddFontBytes adds a font that is already in memory: the content of a TrueType or
// OpenType file. It panics with FontLoaderReused after Load.
//
// Flutter: services/font_loader.dart FontLoader.addFont
func (l *FontLoader) AddFontBytes(data []byte) {
	l.AddFont(func(context.Context) ([]byte, error) { return data, nil })
}

// AddFontAsset adds a font that Load reads from the bundle (for example
// widgets.RunAppOpts.Assets through the binding's AssetBundle) under the key. It
// panics with FontLoaderReused after Load.
//
// Flutter: services/font_loader.dart FontLoader.addFont
func (l *FontLoader) AddFontAsset(bundle AssetBundle, key string) {
	l.AddFont(func(ctx context.Context) ([]byte, error) { return bundle.Load(ctx, key) })
}

func (l *FontLoader) assertNotLoaded() {
	if l.loaded {
		panic(errors.New(tagFontLoaderReused, "a FontLoader can load its fonts only once", "family", l.family))
	}
}

// Load reads the fonts in the order they were added and loads each into the
// dispatcher's text engine (PlatformDispatcher.LoadFontFromList), and stops at the
// first failure. It must be called on the UI thread. It panics with FontLoaderReused
// when Load was called before. It fails with the tag of the font's source (AssetNotFound...) or
// FontLoadFailed when a source fails, with FontLoadCanceled when ctx is done, and with the tags of
// PlatformDispatcher.LoadFontFromList.
//
// Flutter: services/font_loader.dart FontLoader.load
func (l *FontLoader) Load(ctx context.Context, d *ui.PlatformDispatcher) error {
	l.assertNotLoaded()
	l.loaded = true
	for i, load := range l.fonts {
		if err := ctx.Err(); err != nil {
			return errors.NewErr(tagFontLoadCanceled, err).SetMetadata("family", l.family, "font", i)
		}
		data, err := load(ctx)
		if err != nil {
			if _, ok := errors.FromError(err); ok {
				return err
			}
			return errors.NewErr(tagFontLoadFailed, err).SetMetadata("family", l.family, "font", i)
		}
		if err := d.LoadFontFromList(data, l.family); err != nil {
			return err
		}
	}
	return nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagFontLoadCanceled = "FontLoadCanceled"
	tagFontLoadFailed   = "FontLoadFailed"
	tagFontLoaderReused = "FontLoaderReused"
)
