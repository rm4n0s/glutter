package widgets

import (
	"context"
	"reflect"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// LocalizationsDelegate is a factory for a set of localized resources of type T,
// to be loaded by a Localizations widget.
//
// Deviation: Load runs on the UI thread and returns the resources at once, so
// the resources of a locale are always there in the first build; Flutter's load
// returns a Future and the widget builds nothing until it completes (SPEC
// §12.10).
//
// Flutter: widgets/localizations.dart LocalizationsDelegate
type LocalizationsDelegate interface {
	// IsSupported is whether the resources are available for the locale.
	//
	// Flutter: widgets/localizations.dart LocalizationsDelegate.isSupported
	IsSupported(locale ui.Locale) bool
	// Load makes the resources for the locale.
	//
	// Flutter: widgets/localizations.dart LocalizationsDelegate.load
	Load(ctx context.Context, locale ui.Locale) (any, error)
	// ShouldReload is whether the resources must be loaded again when this
	// delegate replaces old, an earlier delegate of the same kind.
	//
	// Flutter: widgets/localizations.dart LocalizationsDelegate.shouldReload
	ShouldReload(old LocalizationsDelegate) bool
	// ResourceType is the type of the resources that the delegate loads: the
	// key of the Localizations widget's map.
	//
	// Flutter: widgets/localizations.dart LocalizationsDelegate.type
	ResourceType() reflect.Type
}

// FuncLocalizationsDelegate is a LocalizationsDelegate made of functions, for
// resources of type T.
//
// Flutter: widgets/localizations.dart LocalizationsDelegate
type FuncLocalizationsDelegate[T any] struct {
	// Supported says whether a locale is supported; nil means all of them.
	Supported func(locale ui.Locale) bool
	// LoadFunc loads the resources.
	LoadFunc func(ctx context.Context, locale ui.Locale) (T, error)
	// Reload says whether the resources are loaded again when the delegate
	// replaces the old one; nil means never.
	Reload func(old LocalizationsDelegate) bool
	// Name is the text of String.
	Name string
}

// IsSupported is whether the locale is supported.
//
// Flutter: widgets/localizations.dart LocalizationsDelegate.isSupported
func (d *FuncLocalizationsDelegate[T]) IsSupported(locale ui.Locale) bool {
	return d.Supported == nil || d.Supported(locale)
}

// Load loads the resources.
//
// Flutter: widgets/localizations.dart LocalizationsDelegate.load
func (d *FuncLocalizationsDelegate[T]) Load(ctx context.Context, locale ui.Locale) (any, error) {
	return d.LoadFunc(ctx, locale)
}

// ShouldReload is whether the resources are loaded again.
//
// Flutter: widgets/localizations.dart LocalizationsDelegate.shouldReload
func (d *FuncLocalizationsDelegate[T]) ShouldReload(old LocalizationsDelegate) bool {
	return d.Reload != nil && d.Reload(old)
}

// ResourceType is the type T.
//
// Flutter: widgets/localizations.dart LocalizationsDelegate.type
func (d *FuncLocalizationsDelegate[T]) ResourceType() reflect.Type { return reflect.TypeFor[T]() }

// String names the delegate.
//
// Flutter: widgets/localizations.dart LocalizationsDelegate.toString
func (d *FuncLocalizationsDelegate[T]) String() string {
	if d.Name != "" {
		return d.Name
	}
	return "LocalizationsDelegate[" + reflect.TypeFor[T]().String() + "]"
}

// WidgetsLocalizations is the set of localized strings that the widgets layer
// uses.
//
// Flutter: widgets/localizations.dart WidgetsLocalizations
type WidgetsLocalizations interface {
	// TextDirection is the direction of the language.
	TextDirection() ui.TextDirection
	// ReorderItemToStart is the label of the action that moves an item to the start.
	ReorderItemToStart() string
	// ReorderItemToEnd is the label of the action that moves an item to the end.
	ReorderItemToEnd() string
	// ReorderItemUp is the label of the action that moves an item up.
	ReorderItemUp() string
	// ReorderItemDown is the label of the action that moves an item down.
	ReorderItemDown() string
	// ReorderItemLeft is the label of the action that moves an item left.
	ReorderItemLeft() string
	// ReorderItemRight is the label of the action that moves an item right.
	ReorderItemRight() string
	// SearchResultsFound is the announcement made when search results appear.
	SearchResultsFound() string
	// NoResultsFound is the announcement made when a search finds nothing.
	NoResultsFound() string
	// CopyButtonLabel is the label of the copy button of the text toolbar.
	CopyButtonLabel() string
	// CutButtonLabel is the label of the cut button.
	CutButtonLabel() string
	// PasteButtonLabel is the label of the paste button.
	PasteButtonLabel() string
	// SelectAllButtonLabel is the label of the select all button.
	SelectAllButtonLabel() string
	// LookUpButtonLabel is the label of the look up button.
	LookUpButtonLabel() string
	// SearchWebButtonLabel is the label of the search web button.
	SearchWebButtonLabel() string
	// ShareButtonLabel is the label of the share button.
	ShareButtonLabel() string
	// RadioButtonUnselectedLabel is the semantics label of an unselected radio.
	RadioButtonUnselectedLabel() string
}

// DefaultWidgetsLocalizations is the English WidgetsLocalizations that is used
// when no other delegate provides them.
//
// Flutter: widgets/localizations.dart DefaultWidgetsLocalizations
type DefaultWidgetsLocalizations struct{}

// TextDirection is left to right.
//
// Flutter: widgets/localizations.dart DefaultWidgetsLocalizations.textDirection
func (DefaultWidgetsLocalizations) TextDirection() ui.TextDirection { return ui.TextDirectionLTR }

// ReorderItemToStart is "Move to the start".
//
// Flutter: widgets/localizations.dart DefaultWidgetsLocalizations.reorderItemToStart
func (DefaultWidgetsLocalizations) ReorderItemToStart() string { return "Move to the start" }

// ReorderItemToEnd is "Move to the end".
//
// Flutter: widgets/localizations.dart DefaultWidgetsLocalizations.reorderItemToEnd
func (DefaultWidgetsLocalizations) ReorderItemToEnd() string { return "Move to the end" }

// ReorderItemUp is "Move up".
//
// Flutter: widgets/localizations.dart DefaultWidgetsLocalizations.reorderItemUp
func (DefaultWidgetsLocalizations) ReorderItemUp() string { return "Move up" }

// ReorderItemDown is "Move down".
//
// Flutter: widgets/localizations.dart DefaultWidgetsLocalizations.reorderItemDown
func (DefaultWidgetsLocalizations) ReorderItemDown() string { return "Move down" }

// ReorderItemLeft is "Move left".
//
// Flutter: widgets/localizations.dart DefaultWidgetsLocalizations.reorderItemLeft
func (DefaultWidgetsLocalizations) ReorderItemLeft() string { return "Move left" }

// ReorderItemRight is "Move right".
//
// Flutter: widgets/localizations.dart DefaultWidgetsLocalizations.reorderItemRight
func (DefaultWidgetsLocalizations) ReorderItemRight() string { return "Move right" }

// SearchResultsFound is "Search results found".
//
// Flutter: widgets/localizations.dart DefaultWidgetsLocalizations.searchResultsFound
func (DefaultWidgetsLocalizations) SearchResultsFound() string { return "Search results found" }

// NoResultsFound is "No results found".
//
// Flutter: widgets/localizations.dart DefaultWidgetsLocalizations.noResultsFound
func (DefaultWidgetsLocalizations) NoResultsFound() string { return "No results found" }

// CopyButtonLabel is "Copy".
//
// Flutter: widgets/localizations.dart DefaultWidgetsLocalizations.copyButtonLabel
func (DefaultWidgetsLocalizations) CopyButtonLabel() string { return "Copy" }

// CutButtonLabel is "Cut".
//
// Flutter: widgets/localizations.dart DefaultWidgetsLocalizations.cutButtonLabel
func (DefaultWidgetsLocalizations) CutButtonLabel() string { return "Cut" }

// PasteButtonLabel is "Paste".
//
// Flutter: widgets/localizations.dart DefaultWidgetsLocalizations.pasteButtonLabel
func (DefaultWidgetsLocalizations) PasteButtonLabel() string { return "Paste" }

// SelectAllButtonLabel is "Select all".
//
// Flutter: widgets/localizations.dart DefaultWidgetsLocalizations.selectAllButtonLabel
func (DefaultWidgetsLocalizations) SelectAllButtonLabel() string { return "Select all" }

// LookUpButtonLabel is "Look Up".
//
// Flutter: widgets/localizations.dart DefaultWidgetsLocalizations.lookUpButtonLabel
func (DefaultWidgetsLocalizations) LookUpButtonLabel() string { return "Look Up" }

// SearchWebButtonLabel is "Search Web".
//
// Flutter: widgets/localizations.dart DefaultWidgetsLocalizations.searchWebButtonLabel
func (DefaultWidgetsLocalizations) SearchWebButtonLabel() string { return "Search Web" }

// ShareButtonLabel is "Share".
//
// Flutter: widgets/localizations.dart DefaultWidgetsLocalizations.shareButtonLabel
func (DefaultWidgetsLocalizations) ShareButtonLabel() string { return "Share" }

// RadioButtonUnselectedLabel is "Not selected".
//
// Flutter: widgets/localizations.dart DefaultWidgetsLocalizations.radioButtonUnselectedLabel
func (DefaultWidgetsLocalizations) RadioButtonUnselectedLabel() string { return "Not selected" }

// DefaultWidgetsLocalizationsDelegate is the delegate that loads
// DefaultWidgetsLocalizations for every locale. It is a read-only value.
//
// Flutter: widgets/localizations.dart DefaultWidgetsLocalizations.delegate
var DefaultWidgetsLocalizationsDelegate LocalizationsDelegate = &FuncLocalizationsDelegate[WidgetsLocalizations]{
	LoadFunc: func(context.Context, ui.Locale) (WidgetsLocalizations, error) {
		return DefaultWidgetsLocalizations{}, nil
	},
	Name: "DefaultWidgetsLocalizations.delegate(en_US)",
}

// WidgetsLocalizationsOf is the WidgetsLocalizations of the closest
// Localizations above the context. It panics with the tag
// WidgetsLocalizationsMissing when there are none.
//
// Flutter: widgets/localizations.dart WidgetsLocalizations.of
func WidgetsLocalizationsOf(bc BuildContext) WidgetsLocalizations {
	DebugCheckHasWidgetsLocalizations(bc)
	l, ok := LocalizationsOf[WidgetsLocalizations](bc)
	if !ok {
		panic(errors.New(tagWidgetsLocalizationsMissing, "no WidgetsLocalizations found; add DefaultWidgetsLocalizationsDelegate to the delegates of the Localizations widget"))
	}
	return l
}

// DebugCheckHasWidgetsLocalizations panics with the tag
// WidgetsLocalizationsMissing when there are no WidgetsLocalizations above the
// context. It returns true so it can be used in a condition.
//
// Flutter: widgets/debug.dart debugCheckHasWidgetsLocalizations
func DebugCheckHasWidgetsLocalizations(bc BuildContext) bool {
	if foundation.DebugMode {
		if _, ok := LocalizationsOf[WidgetsLocalizations](bc); !ok {
			panic(errors.New(tagWidgetsLocalizationsMissing,
				"no WidgetsLocalizations found; the widget needs the localized strings of the widgets library; the app is missing a Localizations widget with DefaultWidgetsLocalizationsDelegate (WidgetsApp and MaterialApp add one)"))
		}
	}
	return true
}

// localizationsScope is the inherited widget that gives the loaded resources.
type localizationsScope struct {
	InheritedWidget
	locale          ui.Locale
	state           *localizationsState
	typeToResources map[reflect.Type]any
	generation      int
}

func (w *localizationsScope) UpdateShouldNotify(old Widget) bool {
	return w.generation != old.(*localizationsScope).generation
}

// Localizations defines the Locale for its child and the localized resources
// that the child depends on.
//
// Flutter: widgets/localizations.dart Localizations
type Localizations struct {
	StatefulWidget
	locale             ui.Locale
	delegates          []LocalizationsDelegate
	child              Widget
	isApplicationLevel bool
}

// LocalizationsOpts are the optional parameters of NewLocalizations.
//
// Flutter: widgets/localizations.dart Localizations.new
type LocalizationsOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// IsApplicationLevel is whether this is the Localizations of the app, which
	// does not mark its semantics subtree with the locale.
	IsApplicationLevel bool
}

// NewLocalizations creates the widget. The child may be nil. In debug builds it
// panics with the tag LocalizationsNoWidgetsDelegate when no delegate loads
// WidgetsLocalizations.
//
// Flutter: widgets/localizations.dart Localizations.new
func NewLocalizations(locale ui.Locale, delegates []LocalizationsDelegate, child Widget, opts ...LocalizationsOpts) *Localizations {
	o := firstOpts(opts)
	if foundation.DebugMode {
		found := false
		for _, d := range delegates {
			if d.ResourceType() == reflect.TypeFor[WidgetsLocalizations]() {
				found = true
			}
		}
		if !found {
			panic(errors.New(tagLocalizationsNoWidgetsDelegate, "the delegates of Localizations must include one for WidgetsLocalizations, such as DefaultWidgetsLocalizationsDelegate"))
		}
	}
	return &Localizations{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), locale, delegates, child, o.IsApplicationLevel}
}

// NewLocalizationsOverride creates a Localizations that has the delegates of the
// closest one above the context, preceded by the given delegates, and the given
// locale, or the one above when it is Zero.
//
// Flutter: widgets/localizations.dart Localizations.override
func NewLocalizationsOverride(bc BuildContext, locale foundation.Opt[ui.Locale], delegates []LocalizationsDelegate, child Widget, opts ...LocalizationsOpts) *Localizations {
	scope, ok := DependOn[*localizationsScope](bc)
	if !ok {
		panic(errors.New(tagLocalizationsMissing, "a Localizations ancestor was not found"))
	}
	merged := append(append([]LocalizationsDelegate{}, delegates...), scope.state.Widget().delegates...)
	return NewLocalizations(locale.OrElse(scope.locale), merged, child, opts...)
}

// Locale is the locale of the child.
//
// Flutter: widgets/localizations.dart Localizations.locale
func (w *Localizations) Locale() ui.Locale { return w.locale }

// Delegates are the factories of the localized resources.
//
// Flutter: widgets/localizations.dart Localizations.delegates
func (w *Localizations) Delegates() []LocalizationsDelegate { return w.delegates }

// IsApplicationLevel is whether this is the Localizations of the app.
//
// Flutter: widgets/localizations.dart Localizations.isApplicationLevel
func (w *Localizations) IsApplicationLevel() bool { return w.isApplicationLevel }

// CreateState creates the state.
//
// Flutter: widgets/localizations.dart Localizations.createState
func (w *Localizations) CreateState() State { return &localizationsState{} }

// LocalizationsLocaleOf is the locale of the closest Localizations above the
// context. It panics with the tag LocalizationsMissing when there is none.
//
// Flutter: widgets/localizations.dart Localizations.localeOf
func LocalizationsLocaleOf(bc BuildContext) ui.Locale {
	l, ok := LocalizationsMaybeLocaleOf(bc)
	if !ok {
		panic(errors.New(tagLocalizationsMissing, "requested the Locale of a context that does not include a Localizations ancestor; the context must be that of a descendant of a Localizations widget"))
	}
	return l
}

// LocalizationsMaybeLocaleOf is the locale of the closest Localizations above
// the context, if there is one.
//
// Flutter: widgets/localizations.dart Localizations.maybeLocaleOf
func LocalizationsMaybeLocaleOf(bc BuildContext) (ui.Locale, bool) {
	scope, ok := DependOn[*localizationsScope](bc)
	if !ok || !scope.state.hasLocale {
		return ui.Locale{}, false
	}
	return scope.state.locale, true
}

// LocalizationsOf is the resources of type T of the closest Localizations above
// the context, if it has them.
//
// Flutter: widgets/localizations.dart Localizations.of
func LocalizationsOf[T any](bc BuildContext) (T, bool) {
	var zero T
	scope, ok := DependOn[*localizationsScope](bc)
	if !ok {
		return zero, false
	}
	v, ok := scope.state.typeToResources[reflect.TypeFor[T]()].(T)
	if !ok {
		return zero, false
	}
	return v, true
}

type localizationsState struct {
	StateOf[*Localizations]
	typeToResources map[reflect.Type]any
	locale          ui.Locale
	hasLocale       bool
	generation      int
}

func (s *localizationsState) InitState() {
	s.typeToResources = map[reflect.Type]any{}
	s.load(s.Widget().locale)
}

func (s *localizationsState) DidUpdateWidget(old *Localizations) {
	if s.Widget().locale != old.locale || s.anyDelegatesShouldReload(old) {
		s.load(s.Widget().locale)
	}
}

func (s *localizationsState) anyDelegatesShouldReload(old *Localizations) bool {
	delegates, oldDelegates := s.Widget().delegates, old.delegates
	if len(delegates) != len(oldDelegates) {
		return true
	}
	for i, d := range delegates {
		if reflect.TypeOf(d) != reflect.TypeOf(oldDelegates[i]) || d.ShouldReload(oldDelegates[i]) {
			return true
		}
	}
	return false
}

// load loads the resources of the locale with the first delegate that supports
// it for each resource type.
func (s *localizationsState) load(locale ui.Locale) {
	delegates := s.Widget().delegates
	if len(delegates) == 0 {
		s.locale, s.hasLocale = locale, true
		return
	}
	loaded := map[reflect.Type]any{}
	for _, d := range delegates {
		t := d.ResourceType()
		if _, done := loaded[t]; done || !d.IsSupported(locale) {
			continue
		}
		v, err := d.Load(context.Background(), locale)
		if err != nil {
			panic(errors.NewErr(tagLocalizationsLoadFailed, err).SetMetadata("delegate", t.String(), "locale", locale.String()))
		}
		loaded[t] = v
	}
	s.typeToResources = loaded
	s.locale, s.hasLocale = locale, true
	s.generation++
}

func (s *localizationsState) textDirection() ui.TextDirection {
	if r, ok := s.typeToResources[reflect.TypeFor[WidgetsLocalizations]()].(WidgetsLocalizations); ok {
		return r.TextDirection()
	}
	return ui.TextDirectionLTR
}

func (s *localizationsState) Build(bc BuildContext) Widget {
	if !s.hasLocale {
		return NewSizedBoxShrink(nil)
	}
	w := s.Widget()
	var child Widget = NewSizedBoxShrink(nil)
	if w.child != nil {
		child = w.child
	}
	so := SemanticsOpts{Container: !w.isApplicationLevel, TextDirection: s.textDirection()}
	if !w.isApplicationLevel {
		so.LocaleForSubtree = foundation.Some(w.locale)
	}
	return NewSemantics(&localizationsScope{
		InheritedWidget: NewInheritedWidgetBase(NewDirectionality(s.textDirection(), child)),
		locale:          s.locale, state: s, typeToResources: s.typeToResources, generation: s.generation,
	}, so)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagLocalizationsLoadFailed        = "LocalizationsLoadFailed"
	tagLocalizationsMissing           = "LocalizationsMissing"
	tagLocalizationsNoWidgetsDelegate = "LocalizationsNoWidgetsDelegate"
	tagWidgetsLocalizationsMissing    = "WidgetsLocalizationsMissing"
)
