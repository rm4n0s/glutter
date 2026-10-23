package services

import (
	"regexp"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/opts"
	"github.com/rm4n0s/glutter/ui"
)

// MaxLengthEnforcement is how a LengthLimitingTextInputFormatter enforces its
// limit. The zero value, MaxLengthEnforcementUnspecified, means the default of
// the platform (LengthLimitingTextInputFormatterGetDefaultMaxLengthEnforcement).
//
// Flutter: services/text_formatter.dart MaxLengthEnforcement
type MaxLengthEnforcement int

// The values of MaxLengthEnforcement follow Flutter's declaration order.
const (
	// MaxLengthEnforcementUnspecified is the zero value: the platform default.
	//
	// Flutter: no counterpart
	MaxLengthEnforcementUnspecified MaxLengthEnforcement = iota
	// MaxLengthEnforcementNone doesn't enforce the limit.
	//
	// Flutter: services/text_formatter.dart MaxLengthEnforcement.none
	MaxLengthEnforcementNone
	// MaxLengthEnforcementEnforced enforces the limit at all times, composing text
	// included.
	//
	// Flutter: services/text_formatter.dart MaxLengthEnforcement.enforced
	MaxLengthEnforcementEnforced
	// MaxLengthEnforcementTruncateAfterCompositionEnds lets the composing text go
	// over the limit, and enforces it when the composition ends.
	//
	// Flutter: services/text_formatter.dart MaxLengthEnforcement.truncateAfterCompositionEnds
	MaxLengthEnforcementTruncateAfterCompositionEnds
)

// TextInputFormatter is a way to format the changes that the user makes to a
// text field, before the field takes them.
//
// Flutter: services/text_formatter.dart TextInputFormatter
type TextInputFormatter interface {
	// FormatEditUpdate is called when the text of a field changes. It returns the
	// value the field takes: the new value, as it is, changed, or the old one.
	//
	// Flutter: services/text_formatter.dart TextInputFormatter.formatEditUpdate
	FormatEditUpdate(oldValue, newValue TextEditingValue) TextEditingValue
}

// TextInputFormatFunction is the function of a TextInputFormatter made with
// NewTextInputFormatterWithFunction.
//
// Flutter: services/text_formatter.dart TextInputFormatFunction
type TextInputFormatFunction func(oldValue, newValue TextEditingValue) TextEditingValue

type simpleTextInputFormatter struct{ formatFunction TextInputFormatFunction }

func (f simpleTextInputFormatter) FormatEditUpdate(oldValue, newValue TextEditingValue) TextEditingValue {
	return f.formatFunction(oldValue, newValue)
}

// NewTextInputFormatterWithFunction wraps a function as a TextInputFormatter.
//
// Flutter: services/text_formatter.dart TextInputFormatter.withFunction
func NewTextInputFormatterWithFunction(formatFunction TextInputFormatFunction) TextInputFormatter {
	return simpleTextInputFormatter{formatFunction}
}

// mutableTextRange is a mutable, half-open range [base, extent) within a string.
type mutableTextRange struct{ base, extent int }

func mutableFromComposingRange(r ui.TextRange) *mutableTextRange {
	if r.IsValid() && !r.IsCollapsed() {
		return &mutableTextRange{r.Start, r.End}
	}
	return nil
}

func mutableFromTextSelection(s TextSelection) *mutableTextRange {
	if s.IsValid() {
		return &mutableTextRange{s.BaseOffset, s.ExtentOffset}
	}
	return nil
}

// textEditingValueAccumulator is the intermediate state of a
// FilteringTextInputFormatter while it formats a new user input.
type textEditingValueAccumulator struct {
	inputValue      TextEditingValue
	stringBuffer    strings.Builder
	selection       *mutableTextRange
	composingRegion *mutableTextRange
}

func newTextEditingValueAccumulator(inputValue TextEditingValue) *textEditingValueAccumulator {
	return &textEditingValueAccumulator{
		inputValue:      inputValue,
		selection:       mutableFromTextSelection(inputValue.Selection),
		composingRegion: mutableFromComposingRange(inputValue.Composing),
	}
}

func (a *textEditingValueAccumulator) finalize() TextEditingValue {
	v := TextEditingValue{Text: a.stringBuffer.String()}
	if a.composingRegion == nil || a.composingRegion.base == a.composingRegion.extent {
		v.Composing = ui.TextRangeEmpty
	} else {
		v.Composing = ui.TextRange{Start: a.composingRegion.base, End: a.composingRegion.extent}
	}
	if a.selection == nil {
		v.Selection = NewTextSelectionCollapsed(-1)
	} else {
		// Try to preserve the selection affinity and isDirectional. This may not
		// make sense if the selection has changed.
		v.Selection = NewTextSelection(a.selection.base, a.selection.extent, TextSelectionOpts{
			Affinity: a.inputValue.Selection.Affinity, IsDirectional: a.inputValue.Selection.IsDirectional,
		})
	}
	return v
}

// FilteringTextInputFormatter is a TextInputFormatter that prevents the
// insertion of characters matching (deny) or not matching (allow) a pattern.
//
// Deviation: the pattern is a *regexp.Regexp, which has Go's RE2 syntax and
// matches UTF-8 text; Dart's Pattern can also be a string (SPEC §12.10).
//
// Flutter: services/text_formatter.dart FilteringTextInputFormatter
type FilteringTextInputFormatter struct {
	// FilterPattern is the pattern that the text is filtered with.
	//
	// Flutter: services/text_formatter.dart FilteringTextInputFormatter.filterPattern
	FilterPattern *regexp.Regexp
	// Allow is whether the matches of the pattern are the allowed text (true) or
	// the banned text (false).
	//
	// Flutter: services/text_formatter.dart FilteringTextInputFormatter.allow
	Allow bool
	// ReplacementString replaces the banned text; empty removes it.
	//
	// Flutter: services/text_formatter.dart FilteringTextInputFormatter.replacementString
	ReplacementString string
}

// FilteringTextInputFormatterOpts are the optional parameters of the
// constructors of FilteringTextInputFormatter.
//
// Flutter: services/text_formatter.dart FilteringTextInputFormatter.new
type FilteringTextInputFormatterOpts struct {
	// ReplacementString replaces the banned text; it defaults to the empty
	// string, which removes the text.
	ReplacementString string
}

// NewFilteringTextInputFormatter creates a formatter that allows (allow true) or
// denies the text that matches the pattern.
//
// Flutter: services/text_formatter.dart FilteringTextInputFormatter.new
func NewFilteringTextInputFormatter(filterPattern *regexp.Regexp, allow bool, o ...FilteringTextInputFormatterOpts) *FilteringTextInputFormatter {
	return &FilteringTextInputFormatter{filterPattern, allow, opts.First(o).ReplacementString}
}

// NewFilteringTextInputFormatterAllow creates a formatter that only allows the
// text that matches the pattern.
//
// Flutter: services/text_formatter.dart FilteringTextInputFormatter.allow
func NewFilteringTextInputFormatterAllow(filterPattern *regexp.Regexp, o ...FilteringTextInputFormatterOpts) *FilteringTextInputFormatter {
	return NewFilteringTextInputFormatter(filterPattern, true, o...)
}

// NewFilteringTextInputFormatterDeny creates a formatter that denies the text
// that matches the pattern.
//
// Flutter: services/text_formatter.dart FilteringTextInputFormatter.deny
func NewFilteringTextInputFormatterDeny(filterPattern *regexp.Regexp, o ...FilteringTextInputFormatterOpts) *FilteringTextInputFormatter {
	return NewFilteringTextInputFormatter(filterPattern, false, o...)
}

// FormatEditUpdate returns the new value with the banned text removed or
// replaced, and the selection and the composing range moved to follow the text
// they were on.
//
// Flutter: services/text_formatter.dart FilteringTextInputFormatter.formatEditUpdate
func (f *FilteringTextInputFormatter) FormatEditUpdate(_, newValue TextEditingValue) TextEditingValue {
	state := newTextEditingValueAccumulator(newValue)
	previousEnd := 0
	for _, m := range f.FilterPattern.FindAllStringIndex(newValue.Text, -1) {
		// Compute the non-match region between this match and the previous one.
		// Depending on the value of allow, either the match region or the
		// non-match region is the banned pattern.
		f.processRegion(f.Allow, previousEnd, m[0], state)
		f.processRegion(!f.Allow, m[0], m[1], state)
		previousEnd = m[1]
	}
	// Handle the last non-matching region between the last match region and the
	// end of the text.
	f.processRegion(f.Allow, previousEnd, len(newValue.Text), state)
	return state.finalize()
}

func (f *FilteringTextInputFormatter) processRegion(isBannedRegion bool, regionStart, regionEnd int, state *textEditingValueAccumulator) {
	var replacementString string
	switch {
	case !isBannedRegion:
		replacementString = state.inputValue.Text[regionStart:regionEnd]
	case regionStart != regionEnd:
		replacementString = f.ReplacementString
	}
	state.stringBuffer.WriteString(replacementString)
	if len(replacementString) == regionEnd-regionStart {
		// We don't have to adjust the indices if the replaced string and the
		// replacement string have the same length.
		return
	}
	adjustIndex := func(originalIndex int) int {
		// The length added by adding the replacementString.
		replacedLength := len(replacementString)
		if originalIndex <= regionStart && originalIndex < regionEnd {
			replacedLength = 0
		}
		// The length removed by removing the replacementRange.
		removedLength := min(max(originalIndex, regionStart), regionEnd) - regionStart
		return replacedLength - removedLength
	}
	if state.selection != nil {
		state.selection.base += adjustIndex(state.inputValue.Selection.BaseOffset)
		state.selection.extent += adjustIndex(state.inputValue.Selection.ExtentOffset)
	}
	if state.composingRegion != nil {
		state.composingRegion.base += adjustIndex(state.inputValue.Composing.Start)
		state.composingRegion.extent += adjustIndex(state.inputValue.Composing.End)
	}
}

// FilteringTextInputFormatterSingleLine is a formatter that denies the line
// breaks, to keep the text of a field on one line.
//
// Flutter: services/text_formatter.dart FilteringTextInputFormatter.singleLineFormatter
var FilteringTextInputFormatterSingleLine TextInputFormatter = NewFilteringTextInputFormatterDeny(regexp.MustCompile(`\n`))

// FilteringTextInputFormatterDigitsOnly is a formatter that allows the digits 0
// to 9 only.
//
// Flutter: services/text_formatter.dart FilteringTextInputFormatter.digitsOnly
var FilteringTextInputFormatterDigitsOnly TextInputFormatter = NewFilteringTextInputFormatterAllow(regexp.MustCompile(`[0-9]`))

// LengthLimitingTextInputFormatter limits the number of characters (grapheme
// clusters) of a text field.
//
// Flutter: services/text_formatter.dart LengthLimitingTextInputFormatter
type LengthLimitingTextInputFormatter struct {
	// MaxLength is the limit; none, or -1, is no limit.
	//
	// Flutter: services/text_formatter.dart LengthLimitingTextInputFormatter.maxLength
	MaxLength foundation.Opt[int]
	// MaxLengthEnforcement is how the limit is enforced; unspecified is the
	// default of the platform.
	//
	// Flutter: services/text_formatter.dart LengthLimitingTextInputFormatter.maxLengthEnforcement
	MaxLengthEnforcement MaxLengthEnforcement
}

// LengthLimitingTextInputFormatterOpts are the optional parameters of
// NewLengthLimitingTextInputFormatter.
//
// Flutter: services/text_formatter.dart LengthLimitingTextInputFormatter.new
type LengthLimitingTextInputFormatterOpts struct {
	// MaxLengthEnforcement defaults to the default of the platform.
	MaxLengthEnforcement MaxLengthEnforcement
}

// NewLengthLimitingTextInputFormatter creates a formatter with a limit; a
// limit that is none or -1 is no limit. Any other limit that is not above 0
// panics with the tag MaxLengthInvalid in a debug build.
//
// Flutter: services/text_formatter.dart LengthLimitingTextInputFormatter.new
func NewLengthLimitingTextInputFormatter(maxLength foundation.Opt[int], o ...LengthLimitingTextInputFormatterOpts) *LengthLimitingTextInputFormatter {
	if n, ok := maxLength.Get(); ok && n != -1 && n <= 0 && foundation.DebugMode {
		panic(errors.New(tagMaxLengthInvalid, "the limit of a LengthLimitingTextInputFormatter must be above 0, or -1", "maxLength", n))
	}
	return &LengthLimitingTextInputFormatter{maxLength, opts.First(o).MaxLengthEnforcement}
}

// LengthLimitingTextInputFormatterGetDefaultMaxLengthEnforcement is the
// enforcement of the platform: the limit is enforced at all times on Android and
// Windows, and after the composition ends elsewhere. With no platform given, it
// is the default target platform.
//
// Flutter: services/text_formatter.dart LengthLimitingTextInputFormatter.getDefaultMaxLengthEnforcement
func LengthLimitingTextInputFormatterGetDefaultMaxLengthEnforcement(platform ...foundation.TargetPlatform) MaxLengthEnforcement {
	p := foundation.DefaultTargetPlatform()
	if len(platform) > 0 {
		p = platform[0]
	}
	switch p {
	case foundation.TargetPlatformAndroid, foundation.TargetPlatformWindows:
		return MaxLengthEnforcementEnforced
	}
	return MaxLengthEnforcementTruncateAfterCompositionEnds
}

// LengthLimitingTextInputFormatterTruncate truncates the value to maxLength
// characters, moving the selection and the composing range into the text.
//
// Flutter: services/text_formatter.dart LengthLimitingTextInputFormatter.truncate
func LengthLimitingTextInputFormatterTruncate(value TextEditingValue, maxLength int) TextEditingValue {
	truncated := value.Text
	if graphemeCount(value.Text) > maxLength {
		truncated = value.Text[:graphemeOffset(value.Text, maxLength)]
	}
	selection := value.Selection.CopyWith(TextSelectionCopy{
		BaseOffset:   foundation.Some(min(value.Selection.Start(), len(truncated))),
		ExtentOffset: foundation.Some(min(value.Selection.End(), len(truncated))),
	})
	composing := ui.TextRangeEmpty
	if !value.Composing.IsCollapsed() && len(truncated) > value.Composing.Start {
		composing = ui.TextRange{Start: value.Composing.Start, End: min(value.Composing.End, len(truncated))}
	}
	return TextEditingValue{Text: truncated, Selection: selection, Composing: composing}
}

// FormatEditUpdate returns the new value when it fits the limit, and otherwise
// the old value or the new one truncated, as the enforcement says.
//
// Flutter: services/text_formatter.dart LengthLimitingTextInputFormatter.formatEditUpdate
func (f *LengthLimitingTextInputFormatter) FormatEditUpdate(oldValue, newValue TextEditingValue) TextEditingValue {
	maxLength, ok := f.MaxLength.Get()
	if !ok || maxLength == -1 || graphemeCount(newValue.Text) <= maxLength {
		return newValue
	}
	enforcement := f.MaxLengthEnforcement
	if enforcement == MaxLengthEnforcementUnspecified {
		enforcement = LengthLimitingTextInputFormatterGetDefaultMaxLengthEnforcement()
	}
	switch enforcement {
	case MaxLengthEnforcementNone:
		return newValue
	case MaxLengthEnforcementEnforced:
		// If already at the maximum and tried to enter even more, and has no
		// selection, keep the old value.
		if graphemeCount(oldValue.Text) == maxLength && oldValue.Selection.IsCollapsed() {
			return oldValue
		}
		// Enforced to return a truncated value.
		return LengthLimitingTextInputFormatterTruncate(newValue, maxLength)
	case MaxLengthEnforcementTruncateAfterCompositionEnds:
		// If already at the maximum and tried to enter even more, and the old
		// value is not composing, keep the old value.
		if graphemeCount(oldValue.Text) == maxLength && !oldValue.Composing.IsValid() {
			return oldValue
		}
		// Temporarily exempt newValue from the limit if it has a composing text
		// going and no enforcement to the composing value, until the composing is
		// finished.
		if newValue.Composing.IsValid() {
			return newValue
		}
		return LengthLimitingTextInputFormatterTruncate(newValue, maxLength)
	}
	return newValue
}

// graphemeCount is the number of characters (grapheme clusters) of the text: Dart's
// text.characters.length.
func graphemeCount(text string) int { return foundation.GraphemeCount(text) }

// graphemeOffset is the byte offset where the n-th character of the text ends.
func graphemeOffset(text string, n int) int { return foundation.GraphemeOffset(text, n) }

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagMaxLengthInvalid = "MaxLengthInvalid"
)
