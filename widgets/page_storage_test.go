// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets_test

import (
	"fmt"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/widgets/page_storage_test.dart. The app has
// no Navigator yet (step 10.2), so the tests make the PageStorage themselves.

func TestPageStorageReadAndWrite(t *testing.T) {
	ts := gluttertest.New(t)
	builderKey := NewPageStorageKey("builderKey")
	var setState StateSetter
	storedValue := 0
	ts.PumpWidget(ltr(NewPageStorage(NewPageStorageBucket(), NewStatefulBuilder(func(bc BuildContext, setter StateSetter) Widget {
		PageStorageOf(bc).WriteState(bc, storedValue)
		setState = setter
		return NewCenter(NewText("storedValue: " + fmt.Sprint(storedValue)))
	}, WidgetOpts{Key: builderKey}))))

	element := ts.Element(gluttertest.ByKey(builderKey))
	if PageStorageOf(element) == nil {
		t.Fatal("no PageStorage")
	}
	if got := PageStorageOf(element).ReadState(element); got != storedValue {
		t.Errorf("read %v, want %v", got, storedValue)
	}
	setState(func() { storedValue = 1 })
	ts.Pump()
	if got := PageStorageOf(element).ReadState(element); got != storedValue {
		t.Errorf("read %v, want %v", got, storedValue)
	}
}

func TestPageStorageReadAndWriteByIdentifier(t *testing.T) {
	ts := gluttertest.New(t)
	bucket := NewPageStorageBucket()
	var setState StateSetter
	storedValue := 0
	frame := func(key foundation.Key) Widget {
		return ltr(NewPageStorage(bucket, NewStatefulBuilder(func(bc BuildContext, setter StateSetter) Widget {
			PageStorageOf(bc).WriteState(bc, storedValue, PageStorageOpts{Identifier: 123})
			setState = setter
			return NewCenter(NewText("storedValue"))
		}, WidgetOpts{Key: key})))
	}
	key := foundation.NewValueKey("Key one")
	ts.PumpWidget(frame(key))
	element := ts.Element(gluttertest.ByKey(key))
	if got := PageStorageOf(element).ReadState(element); got != nil {
		t.Errorf("read %v without an identifier, want nothing", got)
	}
	if got := PageStorageOf(element).ReadState(element, PageStorageOpts{Identifier: 123}); got != storedValue {
		t.Errorf("read %v by identifier, want %v", got, storedValue)
	}

	// A new StatefulBuilder with a different key and the same identifier.
	key = foundation.NewValueKey("Key two")
	ts.PumpWidget(frame(key))
	element = ts.Element(gluttertest.ByKey(key))
	if got := PageStorageOf(element).ReadState(element); got != nil {
		t.Errorf("read %v without an identifier, want nothing", got)
	}
	if got := PageStorageOf(element).ReadState(element, PageStorageOpts{Identifier: 123}); got != storedValue {
		t.Errorf("read %v by identifier, want %v", got, storedValue)
	}
	setState(func() { storedValue = 1 })
	ts.Pump()
	if got := PageStorageOf(element).ReadState(element, PageStorageOpts{Identifier: 123}); got != storedValue {
		t.Errorf("read %v by identifier, want %v", got, storedValue)
	}
}

func TestPageStorageEntriesAreIdentifiedByTheKeysOfTheAncestors(t *testing.T) {
	ts := gluttertest.New(t)
	bucket := NewPageStorageBucket()
	var inner BuildContext
	frame := func(outer string) Widget {
		return ltr(NewPageStorage(bucket, NewSizedBox(NewSizedBox(NewBuilder(func(bc BuildContext) Widget {
			inner = bc
			return NewSizedBox(nil)
		}), SizedBoxOpts{Key: NewPageStorageKey("inner")}), SizedBoxOpts{Key: NewPageStorageKey(outer)})))
	}
	ts.PumpWidget(frame("a"))
	bucket.WriteState(inner, 7)
	if got := bucket.ReadState(inner); got != 7 {
		t.Fatalf("read %v, want 7", got)
	}
	ts.PumpWidget(frame("b")) // another outer key is another entry
	if got := bucket.ReadState(inner); got != nil {
		t.Errorf("read %v under another key, want nothing", got)
	}
	ts.PumpWidget(frame("a")) // and the first one is still there
	if got := bucket.ReadState(inner); got != 7 {
		t.Errorf("read %v, want 7", got)
	}
}

func TestPageStorageStoresNothingWithoutKeys(t *testing.T) {
	ts := gluttertest.New(t)
	bucket := NewPageStorageBucket()
	var bc BuildContext
	ts.PumpWidget(ltr(NewPageStorage(bucket, NewBuilder(func(c BuildContext) Widget {
		bc = c
		return NewSizedBox(nil)
	}))))
	bucket.WriteState(bc, 7)
	if got := bucket.ReadState(bc); got != nil {
		t.Errorf("read %v for a context with no PageStorageKey, want nothing", got)
	}
	if got := bucket.ReadState(bc, PageStorageOpts{Identifier: "x"}); got != nil {
		t.Errorf("read %v for an identifier never written", got)
	}
}
