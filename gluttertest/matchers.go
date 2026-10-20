// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package gluttertest

import (
	"fmt"

	"github.com/rm4n0s/errors"
)

// The matchers are plain functions that return nil when the finder finds what is
// expected and a tagged error when it does not, to be given to t.Error:
//
//	if err := gluttertest.FindsOneWidget(tester, f); err != nil {
//		t.Error(err)
//	}
//
// The error says what the finder was and how many widgets it found. Its tag is
// FindsNothingFailed, FindsNWidgetsFailed or FindsAtLeastNWidgetsFailed.

// FindsNothing checks that the finder finds nothing.
//
// Flutter: flutter_test/lib/src/matchers.dart findsNothing
func FindsNothing(t *Tester, f Finder) error {
	if n := t.Count(f); n != 0 {
		return errors.New("FindsNothingFailed", fmt.Sprintf("expected nothing, found %d: %s", n, f.Describe()),
			"finder", f.Describe(), "found", n)
	}
	return nil
}

// FindsOneWidget checks that the finder finds exactly one widget.
//
// Flutter: flutter_test/lib/src/matchers.dart findsOneWidget
func FindsOneWidget(t *Tester, f Finder) error { return FindsNWidgets(t, f, 1) }

// FindsWidgets checks that the finder finds at least one widget.
//
// Flutter: flutter_test/lib/src/matchers.dart findsWidgets
func FindsWidgets(t *Tester, f Finder) error { return FindsAtLeastNWidgets(t, f, 1) }

// FindsNWidgets checks that the finder finds exactly n widgets.
//
// Flutter: flutter_test/lib/src/matchers.dart findsNWidgets
func FindsNWidgets(t *Tester, f Finder, n int) error {
	if got := t.Count(f); got != n {
		return errors.New("FindsNWidgetsFailed", fmt.Sprintf("expected %d widget(s), found %d: %s", n, got, f.Describe()),
			"finder", f.Describe(), "want", n, "found", got)
	}
	return nil
}

// FindsAtLeastNWidgets checks that the finder finds n widgets or more.
//
// Flutter: flutter_test/lib/src/matchers.dart findsAtLeastNWidgets
func FindsAtLeastNWidgets(t *Tester, f Finder, n int) error {
	if got := t.Count(f); got < n {
		return errors.New("FindsAtLeastNWidgetsFailed", fmt.Sprintf("expected at least %d widget(s), found %d: %s", n, got, f.Describe()),
			"finder", f.Describe(), "want", n, "found", got)
	}
	return nil
}
