package winfonts

// styleScores is the table of SkFontStyleSet::matchStyleCSS3 for the slant: the row is the wanted slant
// (upright, italic), the column the slant of the font.
var styleScores = [2][2]int{
	/* upright */ {3, 1},
	/* italic  */ {1, 3},
}

// weightScore says how well a font of weight current fits the wanted weight, bigger is better, in the
// order of CSS Fonts level 3: the weight itself; for a wanted weight below 400 the lighter weights,
// nearest first, then the heavier ones, nearest first; above 500 the heavier, then the lighter; for
// 400 first 500 (and the weights between), for 500 first 400, and then the lighter and the heavier
// as for a light weight.
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
			return 5000 + current
		}
		return 3000 - current
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

// nearest returns the face that fits the wanted weight and slant best, the first of them when
// several do (SkFontStyleSet::matchStyleCSS3). It returns nil for a family with no face.
//
// Flutter: no counterpart (Skia: src/core/SkFontMgr.cpp SkFontStyleSet::matchStyleCSS3)
func nearest(faces []Face, weight int, italic bool) *Face {
	var best *Face
	bestScore := -1
	slant := 0
	if italic {
		slant = 1
	}
	for i := range faces {
		f := &faces[i]
		cur := 0
		if f.Italic {
			cur = 1
		}
		score := styleScores[slant][cur]*100000 + weightScore(weight, f.Weight)
		if score > bestScore {
			best, bestScore = f, score
		}
	}
	return best
}
