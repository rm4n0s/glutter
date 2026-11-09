package widgets

// ScrollableWidgetBuilder builds a widget that scrolls with the controller it is given: a
// scrollable that a sheet drags (the sheet moves instead of the content when the content is at its
// start).
//
// Flutter: widgets/draggable_scrollable_sheet.dart ScrollableWidgetBuilder
type ScrollableWidgetBuilder func(bc BuildContext, scrollController ScrollController) Widget
