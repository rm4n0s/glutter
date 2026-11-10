package widgets

import "time"

// SelectableDayPredicate is the signature of a function that says whether a day can be selected in a date picker.
//
// Flutter: widgets/date.dart SelectableDayPredicate
type SelectableDayPredicate func(day time.Time) bool
