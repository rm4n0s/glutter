package androidfonts

import (
	"bytes"
	"encoding/xml"
	"io"
	"log/slog"
	"math"
	"strconv"
	"strings"

	"github.com/rm4n0s/errors"
)

// This file is the parser of /system/etc/fonts.xml and of fonts_customization.xml. It is the
// "lmpParser" of Skia's SkFontMgr_android_parser.cpp (the format of Android 5 and later: the
// versions 21 to 23 and the files of API 26 to 37), with the same handlers for the same elements:
// familyset (or fonts-modification), family, family-list, font, axis and alias. The format
// before Android 5 (nameset and fileset) is not read.

// fontVariant is Skia's FontVariant.
type fontVariant uint32

const (
	variantDefault fontVariant = 0x01
	variantCompact fontVariant = 0x02
	variantElegant fontVariant = 0x04
)

// fileStyle is Skia's FontFileInfo::Style.
type fileStyle int

const (
	styleAuto fileStyle = iota
	styleNormal
	styleItalic
)

// axis is one <axis tag stylevalue/> of a font: the design position the font file is used at.
type axis struct {
	Tag   string
	Value float64
}

// fontFileInfo is one <font> of a family: Skia's FontFileInfo.
type fontFileInfo struct {
	// FileName is the name of the file, relative to the folder of the family.
	FileName string
	// Index is the face in a font collection.
	Index int
	// Weight is the OpenType weight, 0 when the element gives none.
	Weight int
	Style  fileStyle
	// Axes are the variation settings of the file; the matcher does not use them.
	Axes []axis
	// PostScriptName is the postScriptName attribute (Android 10 and later); the matcher does not use it.
	PostScriptName string
}

// fontFamily is one <family>, or the part of one that is a fallback for another family: Skia's FontFamily.
type fontFamily struct {
	// Names are the names of the family in lower case, then its aliases.
	Names     []string
	Fonts     []fontFileInfo
	Languages []string
	Variant   fontVariant
	// Order is -1; Skia uses it only for the old format.
	Order int
	// IsFallbackFont is true for a family with no name.
	IsFallbackFont bool
	// FallbackFor is the family that a font of this one stands in for (fallbackFor), "" for none.
	FallbackFor string
	BasePath    string

	// fallbackFamilies are the families made by the fonts with fallbackFor, in the order they
	// appeared (Skia keeps them in a hash map, which has no order).
	fallbackFamilies []*fontFamily
}

func newFontFamily(basePath string, isFallbackFont bool) *fontFamily {
	return &fontFamily{Variant: variantDefault, Order: -1, IsFallbackFont: isFallbackFont, BasePath: basePath}
}

// fallbackFamily returns the family for the fonts of f that have fallbackFor, making it the first time.
func (f *fontFamily) fallbackFamily(fallbackFor string) *fontFamily {
	for _, other := range f.fallbackFamilies {
		if other.FallbackFor == fallbackFor {
			return other
		}
	}
	other := newFontFamily(f.BasePath, true)
	other.Languages = append([]string(nil), f.Languages...)
	other.Variant = f.Variant
	other.Order = f.Order
	other.FallbackFor = fallbackFor
	f.fallbackFamilies = append(f.fallbackFamilies, other)
	return other
}

// element is the kind of an open element: the handler of Skia's TagHandler stack.
type element int

const (
	elTop element = iota
	elFamilySet
	elFamilyList
	elFamily
	elFont
	elAxis
	elAlias
)

// configParser is the state of the parse of one file: Skia's FamilyData.
type configParser struct {
	logger   *slog.Logger
	filename string
	data     []byte
	dec      *xml.Decoder

	families   *[]*fontFamily
	basePath   string
	isFallback bool
	version    int

	current     *fontFamily
	font        *fontFamily // the family that has the current font (the fallback family, when fallbackFor is set)
	fontIndex   int
	familyList  string // the name of the enclosing <family-list>
	stack       []element
	skipDepth   int
	sawRootElem bool
}

// warn logs a problem at the position of the decoder: Skia's SK_FONTCONFIGPARSER_WARNING.
func (p *configParser) warn(msg string, args ...any) {
	p.logger.Warn(msg, append([]any{"file", p.filename, "line", p.line()}, args...)...)
}

// line is the line of the parser in the file, from 1.
func (p *configParser) line() int {
	off := min(int(p.dec.InputOffset()), len(p.data))
	return 1 + bytes.Count(p.data[:off], []byte{'\n'})
}

// parseConfig parses the data of a configuration file and appends its families to families. basePath
// is the folder of the font files, isFallback whether the file is a fallback file (Skia's isFallback). It
// returns the version of the file (the version attribute of familyset; 21 for fonts-modification). It
// fails with AndroidFontsXMLInvalid when the XML is not well formed, has no root element or declares an
// entity (Skia stops at an entity declaration: expat CVE-2013-0340); the families of the
// file are then not added.
//
// Flutter: no counterpart (Skia: src/ports/SkFontMgr_android_parser.cpp parse_config_file)
func parseConfig(logger *slog.Logger, filename string, data []byte, families *[]*fontFamily, basePath string, isFallback bool) (int, error) {
	p := &configParser{
		logger: logger, filename: filename, data: data, dec: xml.NewDecoder(bytes.NewReader(data)),
		families: families, basePath: basePath, isFallback: isFallback, stack: []element{elTop},
	}
	before := len(*families)
	for {
		tok, err := p.dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			*families = (*families)[:before]
			return 0, errors.NewErr(TagXMLInvalid, err).SetMetadata("file", filename, "line", p.line())
		}
		switch t := tok.(type) {
		case xml.StartElement:
			p.sawRootElem = true
			p.startElement(t)
		case xml.EndElement:
			p.endElement()
		case xml.CharData:
			// Only the text directly in a <font> counts: the file name.
			if p.skipDepth == 0 && p.stack[len(p.stack)-1] == elFont {
				p.font.Fonts[p.fontIndex].FileName += string(t)
			}
		case xml.Directive:
			if bytes.Contains(t, []byte("<!ENTITY")) {
				*families = (*families)[:before]
				return 0, errors.New(TagXMLInvalid, "an entity declaration was found, the file is not read", "file", filename, "line", p.line())
			}
		}
	}
	if !p.sawRootElem {
		return 0, errors.New(TagXMLInvalid, "the file has no element", "file", filename)
	}
	return p.version, nil
}

// startElement is Skia's start_element_handler: the parent says which handler the element has, and an
// element that no handler takes is skipped with its content.
func (p *configParser) startElement(se xml.StartElement) {
	if p.skipDepth > 0 {
		p.skipDepth++
		return
	}
	parent := p.stack[len(p.stack)-1]
	child, ok := p.childOf(parent, se)
	if !ok {
		p.warn("element not recognized, skipping", "element", se.Name.Local)
		p.skipDepth = 1
		return
	}
	p.stack = append(p.stack, child)
	switch child {
	case elFamilyList:
		p.startFamilyList(se)
	case elFamily:
		p.startFamily(se)
	case elFont:
		p.startFont(se)
	case elAxis:
		p.startAxis(se)
	case elAlias:
		p.startAlias(se)
	case elTop, elFamilySet:
	}
}

// endElement is Skia's end_element_handler.
func (p *configParser) endElement() {
	if p.skipDepth > 0 {
		p.skipDepth--
		return
	}
	switch p.stack[len(p.stack)-1] {
	case elFamilyList:
		p.familyList = ""
	case elFamily:
		*p.families = append(*p.families, p.current)
		p.current = nil
	case elFont:
		f := &p.font.Fonts[p.fontIndex]
		f.FileName = strings.Trim(f.FileName, " \n\r\t")
	case elTop, elFamilySet, elAxis, elAlias:
	}
	p.stack = p.stack[:len(p.stack)-1]
}

// childOf is the "tag" function of each TagHandler: the handler of a nested element.
func (p *configParser) childOf(parent element, se xml.StartElement) (element, bool) {
	name := se.Name.Local
	switch parent {
	case elTop:
		switch name {
		case "familyset":
			// 'version' (non-negative integer) [default 0]. The old format (before version 21) is
			// not read separately: its elements are not known, so they are skipped.
			for _, a := range se.Attr {
				if a.Name.Local == "version" {
					if v, ok := parseNonNegativeInt(a.Value); ok {
						p.version = v
					}
				}
			}
			return elFamilySet, true
		case "fonts-modification":
			// The file of a device builder (fonts_customization.xml): the standard children are the
			// ones of familyset. customizationType and supportedAxes are not used, as in Skia.
			p.version = 21
			return elFamilySet, true
		}
	case elFamilySet:
		switch name {
		case "family":
			return elFamily, true
		case "family-list":
			return elFamilyList, true
		case "alias":
			return elAlias, true
		}
	case elFamilyList:
		if name == "family" {
			return elFamily, true
		}
	case elFamily:
		if name == "font" {
			return elFont, true
		}
	case elFont:
		if name == "axis" {
			return elAxis, true
		}
	}
	return 0, false
}

// startFamilyList is Skia's lmpParser::familyListHandler start: the name of the list is the
// name of the families in it.
func (p *configParser) startFamilyList(se xml.StartElement) {
	for _, a := range se.Attr {
		if a.Name.Local == "name" {
			p.familyList = asciiLower(a.Value)
		}
	}
}

// startFamily is Skia's lmpParser::familyHandler start.
func (p *configParser) startFamily(se xml.StartElement) {
	// 'name' (string) [optional]
	// 'lang' (space separated string) [default ""]
	// 'variant' ("elegant", "compact") [default "default"]
	// If there is no name, this is a fallback only font.
	f := newFontFamily(p.basePath, true)
	p.current = f
	if p.familyList != "" {
		f.Names = append(f.Names, p.familyList)
		f.IsFallbackFont = false
	}
	for _, a := range se.Attr {
		switch a.Name.Local {
		case "name":
			f.Names = append(f.Names, asciiLower(a.Value))
			f.IsFallbackFont = false
		case "lang":
			f.Languages = append(f.Languages, splitLanguages(a.Value)...)
		case "variant":
			switch a.Value {
			case "elegant":
				f.Variant = variantElegant
			case "compact":
				f.Variant = variantCompact
			}
		}
	}
}

// startFont is Skia's lmpParser::fontHandler start. A font with fallbackFor goes to a family of its own,
// which copies the languages and the variant of the family it is in.
func (p *configParser) startFont(se xml.StartElement) {
	// 'weight' (non-negative integer) [default 0]
	// 'style' ("normal", "italic") [default "auto"]
	// 'index' (non-negative integer) [default 0]
	// The character data should be a filename.
	var file fontFileInfo
	fallbackFor := ""
	for _, a := range se.Attr {
		switch a.Name.Local {
		case "weight":
			if v, ok := parseNonNegativeInt(a.Value); ok {
				file.Weight = v
			} else {
				p.warn("invalid weight", "value", a.Value)
			}
		case "style":
			switch a.Value {
			case "normal":
				file.Style = styleNormal
			case "italic":
				file.Style = styleItalic
			}
		case "index":
			if v, ok := parseNonNegativeInt(a.Value); ok {
				file.Index = v
			} else {
				p.warn("invalid index", "value", a.Value)
			}
		case "fallbackFor":
			// fallbackFor specifies a family fallback and should have been on family.
			fallbackFor = a.Value
		case "postScriptName":
			file.PostScriptName = a.Value
		}
	}
	target := p.current
	if fallbackFor != "" {
		target = p.current.fallbackFamily(fallbackFor)
	}
	target.Fonts = append(target.Fonts, file)
	p.font, p.fontIndex = target, len(target.Fonts)-1
}

// startAxis is Skia's lmpParser::axisHandler start.
func (p *configParser) startAxis(se xml.StartElement) {
	file := &p.font.Fonts[p.fontIndex]
	var (
		tag                string
		value              float64
		tagValid, valValid bool
	)
	for _, a := range se.Attr {
		switch a.Name.Local {
		case "tag":
			if len(a.Value) != 4 {
				p.warn("invalid axis tag", "value", a.Value)
				continue
			}
			tag, tagValid = a.Value, true
			for _, other := range file.Axes {
				if other.Tag == tag {
					tagValid = false
					p.warn("axis specified more than once", "tag", tag)
				}
			}
		case "stylevalue":
			if v, ok := parseFixed16(a.Value); ok {
				value, valValid = v, true
			} else {
				p.warn("invalid axis stylevalue", "value", a.Value)
			}
		}
	}
	if tagValid && valValid {
		file.Axes = append(file.Axes, axis{Tag: tag, Value: value})
	}
}

// startAlias is Skia's lmpParser::aliasHandler start.
func (p *configParser) startAlias(se xml.StartElement) {
	// 'name' (string) introduces a new family name.
	// 'to' (string) specifies which (previous) family to alias
	// 'weight' (non-negative integer) [optional]
	// If it *does not* have a weight, 'name' is an alias for the entire 'to' family.
	// If it *does* have a weight, 'name' is a new family consisting of
	// the font(s) with 'weight' from the 'to' family.
	var aliasName, to string
	weight := 0
	for _, a := range se.Attr {
		switch a.Name.Local {
		case "name":
			aliasName = asciiLower(a.Value)
		case "to":
			to = asciiLower(a.Value)
		case "weight":
			if v, ok := parseNonNegativeInt(a.Value); ok {
				weight = v
			} else {
				p.warn("invalid weight", "value", a.Value)
			}
		}
	}
	// Assumes that the named family is already declared.
	target := p.findFamily(to)
	if target == nil {
		p.warn("alias target not found", "to", to)
		return
	}
	if weight != 0 {
		family := newFontFamily(target.BasePath, p.isFallback)
		family.Names = append(family.Names, aliasName)
		for _, f := range target.Fonts {
			if f.Weight == weight {
				f.Axes = append([]axis(nil), f.Axes...)
				family.Fonts = append(family.Fonts, f)
			}
		}
		*p.families = append(*p.families, family)
	} else {
		target.Names = append(target.Names, aliasName)
	}
}

// findFamily is Skia's find_family: the first family that has the name.
func (p *configParser) findFamily(name string) *fontFamily {
	for _, candidate := range *p.families {
		for _, n := range candidate.Names {
			if n == name {
				return candidate
			}
		}
	}
	return nil
}

// asciiLower lower-cases the ASCII letters only (Skia's SkAutoAsciiToLC).
func asciiLower(s string) string {
	for i := 0; i < len(s); i++ {
		if c := s[i]; 'A' <= c && c <= 'Z' {
			b := []byte(s)
			for j := i; j < len(b); j++ {
				if 'A' <= b[j] && b[j] <= 'Z' {
					b[j] += 'a' - 'A'
				}
			}
			return string(b)
		}
	}
	return s
}

// splitLanguages is Skia's parse_space_separated_languages. Deviation: a comma also separates, because
// Android 8 to 10 write lang="zh-Hant,zh-Bopo".
func splitLanguages(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool {
		return r == ' ' || r == '\n' || r == '\r' || r == '\t' || r == ','
	})
}

// parseNonNegativeInt is Skia's parse_non_negative_integer for an int (32 bits): digits only, no overflow.
func parseNonNegativeInt(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || '9' < s[i] {
			return 0, false
		}
	}
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, false
	}
	return int(v), true
}

// parseFixed16 is Skia's parse_fixed<16>: -?((digits(.digits)?)|(.digits)), no exponent, at most 32767.9999.
// The fraction is cut at 1/65536 as the fixed point value is.
func parseFixed16(s string) (float64, bool) {
	neg := strings.HasPrefix(s, "-")
	body := strings.TrimPrefix(s, "-")
	if body == "" {
		return 0, false
	}
	intPart, frac, hasDot := strings.Cut(body, ".")
	if hasDot && frac == "" {
		return 0, false
	}
	for _, part := range []string{intPart, frac} {
		for i := 0; i < len(part); i++ {
			if part[i] < '0' || '9' < part[i] {
				return 0, false
			}
		}
	}
	n := 0.0
	if intPart != "" {
		v, err := strconv.ParseUint(intPart, 10, 32)
		if err != nil || v > math.MaxInt32>>16 {
			return 0, false
		}
		n = float64(v)
	}
	if frac != "" {
		f, err := strconv.ParseFloat("0."+frac, 64)
		if err != nil {
			return 0, false
		}
		n += math.Floor(f*65536) / 65536
	}
	if neg {
		n = -n
	}
	return n, true
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// TagXMLInvalid is the tag of a font configuration file that is not well formed XML, has no
	// element or declares an entity.
	//
	// Flutter: no counterpart
	TagXMLInvalid = "AndroidFontsXMLInvalid"
)
