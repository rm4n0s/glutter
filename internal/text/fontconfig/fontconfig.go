// Package fontconfig lets a text.Collection use libfontconfig to match fonts, as
// Flutter's Linux engine does: the family names, the user's UI font, and the emoji,
// CJK and per-language preferences then follow the fontconfig settings. The binding
// itself is in internal/platform/linux/fontconfig (SPEC §8.1: C libraries are
// called only under internal/platform).
//
// Flutter: no counterpart
package fontconfig

import (
	"context"

	"github.com/rm4n0s/glutter/internal/platform/linux/fontconfig"
	"github.com/rm4n0s/glutter/internal/text"
)

// Matcher is a text.Matcher over libfontconfig.
//
// Flutter: no counterpart
type Matcher struct {
	cfg *fontconfig.Config
}

// Open loads libfontconfig and the user's configuration. It fails like
// fontconfig.Open: with FontconfigLoadFailed when the library is not there, so the
// caller can use the font scan of the text.Collection instead.
//
// Flutter: no counterpart
func Open(ctx context.Context) (*Matcher, error) {
	cfg, err := fontconfig.Open(ctx)
	if err != nil {
		return nil, err
	}
	return &Matcher{cfg: cfg}, nil
}

// Match implements text.Matcher.
//
// Flutter: no counterpart
func (m *Matcher) Match(ctx context.Context, family string, weight int, italic bool, lang string) ([]text.FontFile, error) {
	files, err := m.cfg.Match(ctx, family, weight, italic, lang)
	if err != nil {
		return nil, err
	}
	out := make([]text.FontFile, len(files))
	for i, f := range files {
		out[i] = text.FontFile(f)
	}
	return out, nil
}

// Close releases the configuration. The Matcher must not be used after.
//
// Flutter: no counterpart
func (m *Matcher) Close() { m.cfg.Close() }
