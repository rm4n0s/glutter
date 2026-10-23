package rendering

import (
	"math"
	"strconv"
	"strings"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
)

const placeholderIndexTagPrefix = "PlaceholderSpanIndexSemanticsTag("

// NewPlaceholderSpanIndexSemanticsTag creates the tag that marks the semantics
// of the placeholder with the index among the placeholders of a paragraph.
//
// Deviation: Flutter's is a SemanticsTag subclass with an index field, declared
// in painting; a SemanticsTag here is a name, so the index is in the name and
// PlaceholderSpanIndexOfSemanticsTag reads it back. It lives in rendering,
// because painting can not import semantics (SPEC §12.10).
//
// Flutter: painting/placeholder_span.dart PlaceholderSpanIndexSemanticsTag
func NewPlaceholderSpanIndexSemanticsTag(index int) semantics.SemanticsTag {
	return semantics.NewSemanticsTag(placeholderIndexTagPrefix + strconv.Itoa(index) + ")")
}

// PlaceholderSpanIndexOfSemanticsTag is the index of a tag made by
// NewPlaceholderSpanIndexSemanticsTag; false for any other tag.
//
// Flutter: painting/placeholder_span.dart PlaceholderSpanIndexSemanticsTag.index
func PlaceholderSpanIndexOfSemanticsTag(tag semantics.SemanticsTag) (int, bool) {
	name := tag.Name()
	if !strings.HasPrefix(name, placeholderIndexTagPrefix) || !strings.HasSuffix(name, ")") {
		return 0, false
	}
	index, err := strconv.Atoi(name[len(placeholderIndexTagPrefix) : len(name)-1])
	return index, err == nil
}

// paragraphSemantics is the semantics state of a RenderParagraph.
type paragraphSemantics struct {
	semanticsInfo          []painting.InlineSpanSemanticsInformation
	cachedAttributedLabels []semantics.AttributedString
	cachedCombinedInfos    []painting.InlineSpanSemanticsInformation
	haveCachedLabels       bool
	haveCombinedInfos      bool
	// cachedChildNodes keeps the nodes made by AssembleSemanticsNode, so that
	// they can be reused: the nodes of the text spans have stable IDs.
	cachedChildNodes []*semantics.SemanticsNode
}

func (r *RenderParagraph) combinedSemanticsInfos() []painting.InlineSpanSemanticsInformation {
	if !r.psem.haveCombinedInfos {
		r.psem.cachedCombinedInfos = painting.CombineSemanticsInfo(r.psem.semanticsInfo)
		r.psem.haveCombinedInfos = true
	}
	return r.psem.cachedCombinedInfos
}

// DescribeSemanticsConfiguration gives the label of the text, or makes the
// paragraph a semantic boundary when a span has a recognizer or an identifier.
//
// Flutter: rendering/paragraph.dart RenderParagraph.describeSemanticsConfiguration
func (r *RenderParagraph) DescribeSemanticsConfiguration(config *semantics.SemanticsConfiguration) {
	r.RenderBox.DescribeSemanticsConfiguration(config)
	r.psem.semanticsInfo = painting.InlineSpanGetSemanticsInformation(r.Text())
	needsAssembleSemanticsNode := false
	needsChildConfigurationsDelegate := false
	for _, info := range r.psem.semanticsInfo {
		if info.Recognizer != nil || info.SemanticsIdentifier != "" {
			needsAssembleSemanticsNode = true
			break
		}
		needsChildConfigurationsDelegate = needsChildConfigurationsDelegate || info.IsPlaceholder
	}
	switch {
	case needsAssembleSemanticsNode:
		config.ExplicitChildNodes = true
		config.SetIsSemanticBoundary(true)
	case needsChildConfigurationsDelegate:
		config.SetChildConfigurationsDelegate(r.childSemanticsConfigurationsDelegate)
	default:
		if !r.psem.haveCachedLabels {
			var buffer strings.Builder
			offset := 0
			var attributes []ui.StringAttribute
			for _, info := range r.psem.semanticsInfo {
				label := info.Text
				if info.SemanticsLabelSet {
					label = info.SemanticsLabel
				}
				for _, a := range info.StringAttributes {
					orig := a.Range()
					attributes = append(attributes, a.CopyWithRange(ui.NewTextRange(offset+orig.Start, offset+orig.End)))
				}
				buffer.WriteString(label)
				offset += len(label)
			}
			r.psem.cachedAttributedLabels = []semantics.AttributedString{semantics.NewAttributedString(buffer.String(), attributes...)}
			r.psem.haveCachedLabels = true
		}
		config.SetAttributedLabel(r.psem.cachedAttributedLabels[0])
		config.SetTextDirection(r.TextDirection())
	}
}

func (r *RenderParagraph) childSemanticsConfigurationsDelegate(childConfigs []*semantics.SemanticsConfiguration) *semantics.ChildSemanticsConfigurationsResult {
	builder := semantics.NewChildSemanticsConfigurationsResultBuilder()
	placeholderIndex, childConfigsIndex, attributedLabelCacheIndex := 0, 0, 0
	for _, info := range r.combinedSemanticsInfos() {
		if info.IsPlaceholder {
			// Every child configuration of this placeholder merges up.
			for childConfigsIndex < len(childConfigs) && childConfigBelongsToPlaceholder(childConfigs[childConfigsIndex], placeholderIndex) {
				builder.MarkAsMergeUp(childConfigs[childConfigsIndex])
				childConfigsIndex++
			}
			placeholderIndex++
		} else {
			builder.MarkAsMergeUp(r.createSemanticsConfigForTextInfo(info, attributedLabelCacheIndex))
			attributedLabelCacheIndex++
		}
	}
	return builder.Build()
}

func childConfigBelongsToPlaceholder(childConfig *semantics.SemanticsConfiguration, placeholderIndex int) bool {
	for tag := range childConfig.TagsForChildren() {
		if index, ok := PlaceholderSpanIndexOfSemanticsTag(tag); ok {
			return index == placeholderIndex
		}
	}
	return false
}

func (r *RenderParagraph) createSemanticsConfigForTextInfo(textInfo painting.InlineSpanSemanticsInformation, cacheIndex int) *semantics.SemanticsConfiguration {
	if !r.psem.haveCachedLabels {
		r.psem.haveCachedLabels = true
		r.psem.cachedAttributedLabels = nil
	}
	if cacheIndex >= len(r.psem.cachedAttributedLabels) {
		label := textInfo.Text
		if textInfo.SemanticsLabelSet {
			label = textInfo.SemanticsLabel
		}
		r.psem.cachedAttributedLabels = append(r.psem.cachedAttributedLabels, semantics.NewAttributedString(label, textInfo.StringAttributes...))
	}
	config := semantics.NewSemanticsConfiguration()
	config.SetTextDirection(r.TextDirection())
	config.SetAttributedLabel(r.psem.cachedAttributedLabels[cacheIndex])
	return config
}

// AssembleSemanticsNode makes a node for each text span that has a recognizer
// or an identifier, and puts the nodes of the placeholders in order. A node of
// a span is reused when possible, so the IDs are stable.
//
// Flutter: rendering/paragraph.dart RenderParagraph.assembleSemanticsNode
func (r *RenderParagraph) AssembleSemanticsNode(node *semantics.SemanticsNode, config *semantics.SemanticsConfiguration, children []*semantics.SemanticsNode) {
	var newChildren []*semantics.SemanticsNode
	currentDirection := r.TextDirection()
	ordinal := 0.0
	start := 0
	placeholderIndex, childIndex := 0, 0
	child := r.FirstChild()
	var newChildCache []*semantics.SemanticsNode
	oldCache := r.psem.cachedChildNodes
	for _, info := range r.combinedSemanticsInfos() {
		selection := ui.NewTextRange(start, start+len(info.Text))
		start += len(info.Text)

		if info.IsPlaceholder {
			// A placeholder may have 0 or more semantics nodes: all are
			// annotated with the index of the placeholder.
			for len(children) > childIndex && children[childIndex].IsTagged(NewPlaceholderSpanIndexSemanticsTag(placeholderIndex)) {
				childNode := children[childIndex]
				// The offset is nil if the render object is truncated.
				if textParentData(child).Offset.IsSome() {
					newChildren = append(newChildren, childNode)
				}
				childIndex++
			}
			child = r.ChildAfter(child)
			placeholderIndex++
			continue
		}
		initialDirection := currentDirection
		rects := r.GetBoxesForSelection(selection)
		if len(rects) == 0 {
			continue
		}
		rect := rects[0].ToRect()
		currentDirection = rects[0].Direction
		for _, textBox := range rects[1:] {
			rect = rect.ExpandToInclude(textBox.ToRect())
			currentDirection = textBox.Direction
		}
		// A text box may have infinite dimensions, which must not go up to the
		// bridges.
		c := r.BoxConstraints()
		rect = ui.NewRectFromLTWH(math.Max(0, rect.Left), math.Max(0, rect.Top),
			math.Min(rect.Width(), c.MaxWidth), math.Min(rect.Height(), c.MaxHeight))
		// Round the rectangle to make the API testable, and pad it so that the
		// accessibility rectangles do not overlap with the text.
		currentRect := ui.NewRectFromLTRB(math.Floor(rect.Left)-4, math.Floor(rect.Top)-4, math.Ceil(rect.Right)+4, math.Ceil(rect.Bottom)+4)
		configuration := semantics.NewSemanticsConfiguration()
		configuration.SetSortKey(semantics.NewOrdinalSortKey(ordinal))
		ordinal++
		configuration.SetTextDirection(initialDirection)
		configuration.SetIdentifier(info.SemanticsIdentifier)
		label := info.Text
		if info.SemanticsLabelSet {
			label = info.SemanticsLabel
		}
		configuration.SetAttributedLabel(semantics.NewAttributedString(label, info.StringAttributes...))
		switch rec := info.Recognizer.(type) {
		case nil:
		case *gestures.TapGestureRecognizer:
			if rec.OnTap != nil {
				configuration.SetOnTap(semantics.VoidCallback(rec.OnTap))
				configuration.SetIsLink(true)
			}
		case *gestures.DoubleTapGestureRecognizer:
			if rec.OnDoubleTap != nil {
				configuration.SetOnTap(semantics.VoidCallback(rec.OnDoubleTap))
				configuration.SetIsLink(true)
			}
		case *gestures.LongPressGestureRecognizer:
			if rec.OnLongPress != nil {
				configuration.SetOnLongPress(semantics.VoidCallback(rec.OnLongPress))
			}
		default:
			if foundation.DebugMode {
				panic(errors.New(tagParagraphRecognizerUnsupported, foundation.RuntimeTypeName(info.Recognizer)+" is not supported",
					"recognizer", foundation.RuntimeTypeName(info.Recognizer)))
			}
		}
		if clip, ok := node.ParentPaintClipRect.Get(); ok {
			paintRect := clip.Intersect(currentRect)
			configuration.SetIsHidden(paintRect.IsEmpty() && !currentRect.IsEmpty())
		}
		var newChild *semantics.SemanticsNode
		if len(oldCache) > 0 {
			newChild, oldCache = oldCache[0], oldCache[1:]
		} else {
			key := foundation.NewUniqueKey()
			newChild = semantics.NewSemanticsNode(semantics.SemanticsNodeOpts{Key: key, ShowOnScreen: r.createShowOnScreenFor(key)})
		}
		newChild.UpdateWith(configuration, nil)
		newChild.SetRect(currentRect)
		newChildCache = append(newChildCache, newChild)
		newChildren = append(newChildren, newChild)
	}
	r.psem.cachedChildNodes = newChildCache
	node.UpdateWith(config, newChildren)
}

func (r *RenderParagraph) createShowOnScreenFor(key foundation.Key) func() {
	return func() {
		for _, n := range r.psem.cachedChildNodes {
			if n.Key == key {
				r.ShowOnScreen(ShowOnScreenOpts{Descendant: r.self, Rect: foundation.Some(n.Rect())})
				return
			}
		}
	}
}

// ClearSemantics forgets the nodes of the spans too.
//
// Flutter: rendering/paragraph.dart RenderParagraph.clearSemantics
func (r *RenderParagraph) ClearSemantics() {
	r.RenderBox.ClearSemantics()
	r.psem.cachedChildNodes = nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagParagraphRecognizerUnsupported = "ParagraphRecognizerUnsupported"
)
