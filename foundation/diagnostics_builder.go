package foundation

import (
	"strings"
	"unicode/utf16"
)

// wordWrapParseMode is the state of the word wrapping loop.
type wordWrapParseMode int

const (
	// modeInSpace is at the start of a break point (or of a line); no break
	// is possible until the next one.
	modeInSpace wordWrapParseMode = iota
	// modeInWord looks for a good break point.
	modeInWord
	// modeAtBreak is at the start of a break point.
	modeAtBreak
)

// prefixedStringBuilder builds the text of a tree: it puts a prefix before
// every line and wraps the lines that may wrap. Lengths and offsets count
// UTF-16 code units, as Dart's do.
//
// Flutter: foundation/diagnostics.dart _PrefixedStringBuilder
type prefixedStringBuilder struct {
	prefixLineOne string
	// prefixOtherLines is the prefix of the lines after the first.
	prefixOtherLines string
	// nextPrefixOtherLines replaces prefixOtherLines at the next line break.
	nextPrefixOtherLines    string
	hasNextPrefixOtherLines bool
	wrapWidth               int
	buffer                  strings.Builder
	currentLine             string
	// wrappableRanges holds start and end offsets of the stretches of the
	// current line that may wrap.
	wrappableRanges []int
	numLines        int
}

func newPrefixedStringBuilder(prefixLineOne, prefixOtherLines string, wrapWidth int) *prefixedStringBuilder {
	return &prefixedStringBuilder{prefixLineOne: prefixLineOne, prefixOtherLines: prefixOtherLines, wrapWidth: wrapWidth}
}

// currentPrefixOtherLines is the prefix of the lines after the first, which
// the next line break may change.
func (b *prefixedStringBuilder) currentPrefixOtherLines() string {
	if b.hasNextPrefixOtherLines {
		return b.nextPrefixOtherLines
	}
	return b.prefixOtherLines
}

// setPrefixOtherLines sets the prefix of the lines after the first.
func (b *prefixedStringBuilder) setPrefixOtherLines(prefix string) {
	b.prefixOtherLines = prefix
	b.hasNextPrefixOtherLines = false
}

// incrementPrefixOtherLines adds suffix to the prefix of the lines after the
// first, at once if the current line is empty or updateCurrentLine is true,
// and at the next line break otherwise.
func (b *prefixedStringBuilder) incrementPrefixOtherLines(suffix string, updateCurrentLine bool) {
	if b.currentLine == "" || updateCurrentLine {
		b.prefixOtherLines = b.currentPrefixOtherLines() + suffix
		b.hasNextPrefixOtherLines = false
	} else {
		b.nextPrefixOtherLines = b.currentPrefixOtherLines() + suffix
		b.hasNextPrefixOtherLines = true
	}
}

// requiresMultipleLines is whether the text written so far takes more than
// one line.
func (b *prefixedStringBuilder) requiresMultipleLines() bool {
	return b.numLines > 1 ||
		(b.numLines == 1 && b.currentLine != "") ||
		(strLen(b.currentLine)+strLen(b.currentPrefix()) > b.wrapWidth)
}

// isCurrentLineEmpty is whether nothing is written on the current line.
func (b *prefixedStringBuilder) isCurrentLineEmpty() bool { return b.currentLine == "" }

// finalizeLine ends the current line, wrapping it if it has stretches that
// may wrap.
func (b *prefixedStringBuilder) finalizeLine(addTrailingLineBreak bool) {
	firstLine := b.buffer.Len() == 0
	text := b.currentLine
	b.currentLine = ""
	if len(b.wrappableRanges) == 0 {
		// Fast path. There were no wrappable spans of text.
		b.writeLine(text, addTrailingLineBreak)
		return
	}
	startOffset := strLen(b.prefixOtherLines)
	if firstLine {
		startOffset = strLen(b.prefixLineOne)
	}
	lines := wordWrapLine(text, b.wrappableRanges, b.wrapWidth, startOffset, strLen(b.prefixOtherLines))
	for i, line := range lines {
		b.writeLine(line, addTrailingLineBreak || i+1 < len(lines))
	}
	b.wrappableRanges = b.wrappableRanges[:0]
}

// wordWrapLine wraps a line at spaces that are inside the wrapRanges (pairs of
// start and end offsets), so that lines are at most width long.
func wordWrapLine(message string, wrapRanges []int, width, startOffset, otherLineOffset int) []string {
	if strLen(message)+startOffset < width {
		// Nothing to do. The line doesn't wrap.
		return []string{message}
	}
	msg := utf16.Encode([]rune(message))
	n := len(msg)
	var wrappedLine []string
	startForLengthCalculations := -startOffset
	mode := modeInSpace
	var lastWordStart int
	lastWordEnd := -1 // Dart's null
	hasLastWordEnd := false
	index := 0
	start := 0
	currentChunk := 0

	// noWrap is called with increasing indexes.
	noWrap := func(index int) bool {
		for {
			if currentChunk >= len(wrapRanges) {
				return true
			}
			if index < wrapRanges[currentChunk+1] {
				break // Found nearest chunk.
			}
			currentChunk += 2
		}
		return index < wrapRanges[currentChunk]
	}

	for {
		switch mode {
		case modeInSpace: // at start of break point (or start of line); can't break until next break
			for index < n && msg[index] == ' ' {
				index++
			}
			lastWordStart = index
			mode = modeInWord
		case modeInWord: // looking for a good break point
			for index < n && (msg[index] != ' ' || noWrap(index)) {
				index++
			}
			mode = modeAtBreak
		case modeAtBreak: // at start of break point
			if index-startForLengthCalculations > width || index == n {
				// we are over the width line, so break
				if index-startForLengthCalculations <= width || !hasLastWordEnd {
					// we should use this point, because either it doesn't actually go over the
					// end (last line), or it does, but there was no earlier break point
					lastWordEnd = index
				}
				wrappedLine = append(wrappedLine, string(utf16.Decode(msg[start:lastWordEnd])))
				if lastWordEnd >= n {
					return wrappedLine
				}
				// just yielded a line
				if lastWordEnd == index {
					// we broke at current position
					// eat all the wrappable spaces, then set our start point
					// Even if some of the spaces are not wrappable that is ok.
					for index < n && msg[index] == ' ' {
						index++
					}
					start = index
					mode = modeInWord
				} else {
					// we broke at the previous break point, and we're at the start of a new one
					start = lastWordStart
					mode = modeAtBreak
				}
				startForLengthCalculations = start - otherLineOffset
				hasLastWordEnd = false
			} else {
				// save this break point, we're not yet over the line width
				lastWordEnd = index
				hasLastWordEnd = true
				// skip to the end of this break point
				mode = modeInSpace
			}
		}
	}
}

// write adds text to the current line. Line breaks in the text end lines.
// The text may wrap when allowWrap is true.
func (b *prefixedStringBuilder) write(s string, allowWrap bool) {
	if s == "" {
		return
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if i > 0 {
			b.finalizeLine(true)
			b.updatePrefix()
		}
		if line == "" {
			continue
		}
		if allowWrap {
			wrapStart := strLen(b.currentLine)
			wrapEnd := wrapStart + strLen(line)
			if n := len(b.wrappableRanges); n > 0 && b.wrappableRanges[n-1] == wrapStart {
				// Extend last range.
				b.wrappableRanges[n-1] = wrapEnd
			} else {
				b.wrappableRanges = append(b.wrappableRanges, wrapStart, wrapEnd)
			}
		}
		b.currentLine += line
	}
}

// updatePrefix makes the prefix that was set for the next line the current
// one.
func (b *prefixedStringBuilder) updatePrefix() {
	if b.hasNextPrefixOtherLines {
		b.prefixOtherLines = b.nextPrefixOtherLines
		b.hasNextPrefixOtherLines = false
	}
}

// writeLine adds a whole line, with the prefix.
func (b *prefixedStringBuilder) writeLine(line string, includeLineBreak bool) {
	line = b.currentPrefix() + line
	b.buffer.WriteString(trimRight(line))
	if includeLineBreak {
		b.buffer.WriteByte('\n')
	}
	b.numLines++
}

// currentPrefix is the prefix of the line being written.
func (b *prefixedStringBuilder) currentPrefix() string {
	if b.buffer.Len() == 0 {
		return b.prefixLineOne
	}
	return b.prefixOtherLines
}

// writeRawLines adds lines as they are, ending the current line first.
func (b *prefixedStringBuilder) writeRawLines(lines string) {
	if lines == "" {
		return
	}
	if b.currentLine != "" {
		b.finalizeLine(true)
	}
	b.buffer.WriteString(lines)
	if !strings.HasSuffix(lines, "\n") {
		b.buffer.WriteByte('\n')
	}
	b.numLines++
	b.updatePrefix()
}

// writeStretched adds text and repeats its last character until the line is
// targetLineLength long. The line does not wrap.
func (b *prefixedStringBuilder) writeStretched(text string, targetLineLength int) {
	b.write(text, false)
	currentLineLength := strLen(b.currentLine) + strLen(b.currentPrefix())
	targetLength := targetLineLength - currentLineLength
	if targetLength > 0 {
		runes := []rune(text)
		b.currentLine += strings.Repeat(string(runes[len(runes)-1]), targetLength)
	}
	// Mark the entire line as not wrappable.
	b.wrappableRanges = b.wrappableRanges[:0]
}

// build returns the text.
func (b *prefixedStringBuilder) build() string {
	if b.currentLine != "" {
		b.finalizeLine(false)
	}
	return b.buffer.String()
}
