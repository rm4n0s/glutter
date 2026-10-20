// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package foundation

import (
	"strings"

	"github.com/go-text/typesetting/segmenter"
)

// zeroWidthJoiner is U+200D, which joins the emoji of a sequence.
const zeroWidthJoiner = '‍'

// isExtendedPictographic approximates the Unicode Extended_Pictographic property:
// the blocks of the emoji and the symbols that have an emoji presentation.
func isExtendedPictographic(r rune) bool {
	switch {
	case r >= 0x1F000 && r <= 0x1FAFF,
		r >= 0x2190 && r <= 0x21FF,
		r >= 0x2300 && r <= 0x23FF,
		r >= 0x2600 && r <= 0x27BF,
		r >= 0x2900 && r <= 0x297F,
		r >= 0x2B00 && r <= 0x2BFF,
		r == 0x00A9, r == 0x00AE, r == 0x203C, r == 0x2049, r == 0x2122, r == 0x2139,
		r == 0x24C2, r == 0x25AA, r == 0x25AB, r == 0x25B6, r == 0x25C0,
		r == 0x3030, r == 0x303D, r == 0x3297, r == 0x3299:
		return true
	}
	return false
}

// graphemeClusters splits the text into extended grapheme clusters.
//
// Deviation: go-text/typesetting v0.3.4 ends the cluster of an emoji sequence after
// the first ZWJ sequence when another follows directly, so this joins a cluster that
// ends with a ZWJ after a pictograph to the cluster that starts with a pictograph
// (rule GB11).
func graphemeClusters(text string) []string {
	var seg segmenter.Segmenter
	seg.InitWithString(text)
	var out []string
	for it := seg.GraphemeIterator(); it.Next(); {
		c := string(it.Grapheme().Text)
		if n := len(out); n > 0 && joinsEmojiSequence(out[n-1], c) {
			out[n-1] += c
			continue
		}
		out = append(out, c)
	}
	return out
}

func joinsEmojiSequence(prev, next string) bool {
	if !strings.HasSuffix(prev, string(zeroWidthJoiner)) {
		return false
	}
	first := []rune(prev)[0]
	following := []rune(next)[0]
	return isExtendedPictographic(first) && isExtendedPictographic(following)
}

// GraphemeCount is the number of user-perceived characters (extended grapheme
// clusters) of the text: Dart's text.characters.length.
//
// Flutter: characters/characters.dart Characters.length
func GraphemeCount(text string) int {
	return len(graphemeClusters(text))
}

// GraphemeOffset is the byte offset where the n-th character (grapheme cluster)
// of the text ends; the length of the text when it has fewer than n.
//
// Flutter: characters/characters.dart Characters.take
func GraphemeOffset(text string, n int) int {
	bytes := 0
	for _, c := range graphemeClusters(text) {
		if n <= 0 {
			break
		}
		bytes += len(c)
		n--
	}
	return bytes
}

// GraphemeBoundaries are the byte offsets of the boundaries between the
// characters (grapheme clusters) of the text, in order, including 0 and the
// length of the text (only 0 for the empty text).
//
// Flutter: characters/characters.dart CharacterRange.at
func GraphemeBoundaries(text string) []int {
	b := []int{0}
	n := 0
	for _, c := range graphemeClusters(text) {
		n += len(c)
		b = append(b, n)
	}
	return b
}
