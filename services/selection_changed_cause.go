package services

// SelectionChangedCause is the reason the selection of a text field changed.
//
// Flutter: services/text_input.dart SelectionChangedCause
type SelectionChangedCause int

// The values of SelectionChangedCause follow Flutter's declaration order.
const (
	// SelectionChangedCauseUnspecified is the zero value: no reason is known.
	//
	// Flutter: no counterpart
	SelectionChangedCauseUnspecified SelectionChangedCause = iota
	// SelectionChangedCauseTap is a tap.
	//
	// Flutter: services/text_input.dart SelectionChangedCause.tap
	SelectionChangedCauseTap
	// SelectionChangedCauseDoubleTap is a double tap.
	//
	// Flutter: services/text_input.dart SelectionChangedCause.doubleTap
	SelectionChangedCauseDoubleTap
	// SelectionChangedCauseLongPress is a long press.
	//
	// Flutter: services/text_input.dart SelectionChangedCause.longPress
	SelectionChangedCauseLongPress
	// SelectionChangedCauseForcePress is a force press.
	//
	// Flutter: services/text_input.dart SelectionChangedCause.forcePress
	SelectionChangedCauseForcePress
	// SelectionChangedCauseKeyboard is a key.
	//
	// Flutter: services/text_input.dart SelectionChangedCause.keyboard
	SelectionChangedCauseKeyboard
	// SelectionChangedCauseToolbar is a button of the toolbar.
	//
	// Flutter: services/text_input.dart SelectionChangedCause.toolbar
	SelectionChangedCauseToolbar
	// SelectionChangedCauseDrag is a drag.
	//
	// Flutter: services/text_input.dart SelectionChangedCause.drag
	SelectionChangedCauseDrag
	// SelectionChangedCauseStylusHandwriting is a stylus handwriting.
	//
	// Flutter: services/text_input.dart SelectionChangedCause.stylusHandwriting
	SelectionChangedCauseStylusHandwriting
)

// String is the name of the cause, like Dart's enum toString.
//
// Flutter: services/text_input.dart SelectionChangedCause
func (c SelectionChangedCause) String() string {
	switch c {
	case SelectionChangedCauseTap:
		return "SelectionChangedCause.tap"
	case SelectionChangedCauseDoubleTap:
		return "SelectionChangedCause.doubleTap"
	case SelectionChangedCauseLongPress:
		return "SelectionChangedCause.longPress"
	case SelectionChangedCauseForcePress:
		return "SelectionChangedCause.forcePress"
	case SelectionChangedCauseKeyboard:
		return "SelectionChangedCause.keyboard"
	case SelectionChangedCauseToolbar:
		return "SelectionChangedCause.toolbar"
	case SelectionChangedCauseDrag:
		return "SelectionChangedCause.drag"
	case SelectionChangedCauseStylusHandwriting:
		return "SelectionChangedCause.stylusHandwriting"
	}
	return "SelectionChangedCause.unspecified"
}
