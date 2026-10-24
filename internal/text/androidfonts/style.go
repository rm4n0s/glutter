package androidfonts

// face is one font of a family that exists on the device: a <font> element and its path.
type face struct {
	path   string
	index  int
	weight int
	italic bool
}

// styleScores is the table of SkFontStyleSet::matchStyleCSS3 for the slant: the row is the wanted slant
// (upright, italic), the column the slant of the font. Glutter has no oblique.
var styleScores = [2][2]int{
	/* upright */ {3, 1},
	/* italic  */ {1, 3},
}

// weightScore says how well a font of weight current fits the wanted weight, bigger is better, in the
// order of CSS Fonts level 3 (which Skia's matchStyleCSS3 follows): the weight itself; for a wanted
// weight below 400 the lighter weights, nearest first, then the heavier ones, nearest first; above 500 the
// heavier, then the lighter; for 400 first 500 (and the weights between), for 500 first 400, and then the
// lighter and the heavier as for a light weight.
func weightScore(wanted, current int) int {
	switch {
	case current == wanted:
		return 10000
	case wanted == 400 && current > 400 && current <= 500:
		return 9999 - (current - 400)
	case wanted == 500 && current >= 400 && current < 500:
		return 9999 - (500 - current)
	case wanted >= 400 && wanted <= 500:
		if current < wanted {
			return 5000 + current // the lighter weights, the nearest first
		}
		return 3000 - current // then the heavier ones, the nearest first
	case wanted < 400:
		if current < wanted {
			return 5000 + current
		}
		return 3000 - current
	default: // wanted > 500
		if current > wanted {
			return 5000 - current
		}
		return 3000 + current
	}
}

// matchStyle returns the font of the family that fits the wanted weight and slant best, the first of
// them when several do: SkFontStyleSet::matchStyleCSS3 (src/core/SkFontMgr.cpp; the widths of Android's
// fonts are all normal, so the width does not count). It returns nil for a family with no font.
//
// Flutter: no counterpart (Skia: src/core/SkFontMgr.cpp SkFontStyleSet::matchStyleCSS3)
func matchStyle(fonts []face, weight int, italic bool) *face {
	var best *face
	bestScore := -1
	slant := 0
	if italic {
		slant = 1
	}
	for i := range fonts {
		f := &fonts[i]
		cur := 0
		if f.italic {
			cur = 1
		}
		score := styleScores[slant][cur]*100000 + weightScore(weight, f.weight)
		if score > bestScore {
			best, bestScore = f, score
		}
	}
	return best
}
