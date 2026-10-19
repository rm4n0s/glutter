package widgets_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/actions_test.dart.
// Not ported: the cases of Dart's variance and casts ("Contravariant", "cannot
// cast Action"), which Go's type system settles at compile time, and the cases
// of the button tester (button_tester.dart), which is a Material widget.

func containerKey() *GlobalKey[State] { return NewGlobalKey[State]() }

func TestActionDispatcherInvokesActionsWhenAsked(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(NewContainer(nil))
	invoked := false
	dispatcher := NewActionDispatcher()
	result := dispatcher.InvokeAction(newTestAction(func(Intent) any {
		invoked = true
		return invoked
	}), testIntentV{}, nil)
	if result != true || !invoked {
		t.Errorf("result = %v, invoked = %v; want true, true", result, invoked)
	}
}

// actionsCollector is the collect function of the Actions group.
type actionsCollector struct {
	intent     Intent
	action     Action
	dispatcher ActionDispatcher
}

func (c *actionsCollector) collect(action Action, intent Intent, dispatcher ActionDispatcher) {
	c.intent, c.action, c.dispatcher = intent, action, dispatcher
}

func trueAction(invoked *bool) *testAction {
	return newTestAction(func(Intent) any {
		*invoked = true
		return true
	})
}

func testActions(action Action) ActionMap {
	return ActionMap{IntentTypeOf[testIntentV](): action}
}

func TestActionsCanInvokeActionsWithDefaultDispatcher(t *testing.T) {
	for _, tc := range []struct {
		name   string
		invoke func(bc BuildContext, intent Intent) any
	}{
		{"invoke", ActionsInvoke},
		{"maybeInvoke", ActionsMaybeInvoke},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			key := containerKey()
			invoked := false
			ts.PumpWidget(NewActions(testActions(trueAction(&invoked)), NewContainer(nil, ContainerOpts{Key: key})))
			ts.Pump()
			result := tc.invoke(key.CurrentContext(), testIntentV{})
			if result != true || !invoked {
				t.Errorf("result = %v, invoked = %v; want true, true", result, invoked)
			}
		})
	}
}

func TestMaybeInvokeReturnsNilWhenNoActionIsFound(t *testing.T) {
	ts := gluttertest.New(t)
	key := containerKey()
	invoked := false
	ts.PumpWidget(NewActions(testActions(trueAction(&invoked)), NewContainer(nil, ContainerOpts{Key: key})))
	ts.Pump()
	if result := ActionsMaybeInvoke(key.CurrentContext(), DoNothingIntent{}); result != nil || invoked {
		t.Errorf("result = %v, invoked = %v; want nil, false", result, invoked)
	}
}

func TestInvokePanicsWhenNoActionIsFound(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	key := containerKey()
	invoked := false
	ts.PumpWidget(NewActions(testActions(trueAction(&invoked)), NewContainer(nil, ContainerOpts{Key: key})))
	ts.Pump()
	requireTag(t, "ActionNotFound", func() { ActionsInvoke(key.CurrentContext(), DoNothingIntent{}) })
	if invoked {
		t.Error("no action should have been invoked")
	}
}

func TestActionsCanInvokeActionsWithCustomDispatcher(t *testing.T) {
	ts := gluttertest.New(t)
	key := containerKey()
	invoked := false
	var c actionsCollector
	intent := testIntentV{}
	ts.PumpWidget(NewActions(testActions(trueAction(&invoked)), NewContainer(nil, ContainerOpts{Key: key}),
		ActionsOpts{Dispatcher: newTestDispatcher(c.collect)}))
	ts.Pump()
	result := ActionsInvoke(key.CurrentContext(), intent)
	if result != true || !invoked || c.intent != Intent(intent) {
		t.Errorf("result = %v, invoked = %v, collected intent = %v", result, invoked, c.intent)
	}
}

func TestActionsCanInvokeActionsInAncestorDispatcher(t *testing.T) {
	for _, tc := range []struct {
		name        string
		lowerHasOne bool
	}{
		{"a lower dispatcher is specified", true},
		{"a lower dispatcher isn't specified", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			key := containerKey()
			invoked := false
			var c actionsCollector
			testAct := trueAction(&invoked)
			var lowerOpts ActionsOpts
			if tc.lowerHasOne {
				lowerOpts.Dispatcher = newTestDispatcher(c.collect)
			}
			ts.PumpWidget(NewActions(testActions(testAct),
				NewActions(ActionMap{}, NewContainer(nil, ContainerOpts{Key: key}), lowerOpts),
				ActionsOpts{Dispatcher: newTestDispatcher1(c.collect)}))
			ts.Pump()
			intent := testIntentV{}
			result := ActionsInvoke(key.CurrentContext(), intent)
			if result != true || !invoked {
				t.Errorf("result = %v, invoked = %v; want true, true", result, invoked)
			}
			if c.intent != Intent(intent) || c.action != Action(testAct) {
				t.Errorf("collected %v / %v", c.intent, c.action)
			}
			if _, ok := c.dispatcher.(*testDispatcher1); !ok {
				t.Errorf("the dispatcher is %T, want *testDispatcher1", c.dispatcher)
			}
		})
	}
}

func TestActionsWidgetCanBeFoundWithOf(t *testing.T) {
	ts := gluttertest.New(t)
	key := containerKey()
	var c actionsCollector
	dispatcher := newTestDispatcher1(c.collect)
	ts.PumpWidget(NewActions(ActionMap{}, NewContainer(nil, ContainerOpts{Key: key}), ActionsOpts{Dispatcher: dispatcher}))
	ts.Pump()
	if got := ActionsOf(key.CurrentContext()); got != ActionDispatcher(dispatcher) {
		t.Errorf("ActionsOf = %v, want the dispatcher of the widget", got)
	}
}

func TestActionCanBeFoundWithFind(t *testing.T) {
	ts := gluttertest.New(t)
	key := containerKey()
	var c actionsCollector
	dispatcher := newTestDispatcher1(c.collect)
	invoked := false
	act := trueAction(&invoked)
	build := func() Widget {
		return NewActions(testActions(act),
			NewActions(ActionMap{}, NewContainer(nil, ContainerOpts{Key: key})),
			ActionsOpts{Dispatcher: dispatcher})
	}
	// The second build checks that finding works the same after a rebuild.
	for range 2 {
		ts.PumpWidget(build())
		ts.Pump()
		if got := ActionsFind[testIntentV](key.CurrentContext()); got != Action(act) {
			t.Errorf("ActionsFind = %v, want the test action", got)
		}
		if foundation.DebugMode {
			requireTag(t, "ActionNotFound", func() { ActionsFind[DoNothingIntent](key.CurrentContext()) })
		}
		if got := ActionsMaybeFind[DoNothingIntent](key.CurrentContext()); got != nil {
			t.Errorf("ActionsMaybeFind = %v, want nil", got)
		}
	}
}

func TestActionsHandlerForwardsTheIntentTypeAndFindsTheBoundAction(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/191045.
	ts := gluttertest.New(t)
	key := containerKey()
	invoked := false
	ts.PumpWidget(NewActions(testActions(trueAction(&invoked)), NewContainer(nil, ContainerOpts{Key: key})))
	handler := ActionsHandler(key.CurrentContext(), testIntentV{})
	if handler == nil {
		t.Fatal("the handler should not be nil")
	}
	handler()
	if !invoked {
		t.Error("the handler should have invoked the action")
	}
}

func TestActionsHandlerReturnsNil(t *testing.T) {
	for _, tc := range []struct {
		name    string
		actions func() ActionMap
	}{
		{"the action is disabled", func() ActionMap {
			a := newTestAction(func(Intent) any { return nil })
			a.enabled = false
			return testActions(a)
		}},
		{"no matching action is found", func() ActionMap { return ActionMap{} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			key := containerKey()
			ts.PumpWidget(NewActions(tc.actions(), NewContainer(nil, ContainerOpts{Key: key})))
			if got := ActionsHandler(key.CurrentContext(), testIntentV{}); got != nil {
				t.Error("the handler should be nil")
			}
		})
	}
}

func TestActionsInvokeReturnsTheValueOfActionInvoke(t *testing.T) {
	ts := gluttertest.New(t)
	key := containerKey()
	sentinel := &struct{ x int }{1}
	invoked := false
	act := newTestAction(func(Intent) any {
		invoked = true
		return sentinel
	})
	var c actionsCollector
	ts.PumpWidget(NewActions(testActions(act), NewContainer(nil, ContainerOpts{Key: key}),
		ActionsOpts{Dispatcher: newTestDispatcher(c.collect)}))
	ts.Pump()
	result := ActionsInvoke(key.CurrentContext(), testIntentV{})
	if result != any(sentinel) || !invoked {
		t.Errorf("result = %v, invoked = %v; want the sentinel, true", result, invoked)
	}
}

func TestContextActionCanReturnNil(t *testing.T) {
	ts := gluttertest.New(t)
	key := containerKey()
	var c actionsCollector
	act := newTestContextAction()
	intent := testIntentV{}
	ts.PumpWidget(NewActions(testActions(act), NewContainer(nil, ContainerOpts{Key: key}),
		ActionsOpts{Dispatcher: newTestDispatcher1(c.collect)}))
	ts.Pump()
	result := ActionsInvoke(key.CurrentContext(), intent)
	if result != nil {
		t.Errorf("result = %v, want nil", result)
	}
	if c.intent != Intent(intent) || c.action != Action(act) {
		t.Errorf("collected %v / %v", c.intent, c.action)
	}
	if _, ok := c.dispatcher.(*testDispatcher1); !ok {
		t.Errorf("the dispatcher is %T, want *testDispatcher1", c.dispatcher)
	}
	if len(act.capturedContexts) != 1 || act.capturedContexts[0] != key.CurrentContext() {
		t.Errorf("captured contexts = %v, want the one of the container", act.capturedContexts)
	}
}

func TestDisabledActionsStopPropagationToAnAncestor(t *testing.T) {
	ts := gluttertest.New(t)
	key := containerKey()
	invoked := false
	var c actionsCollector
	enabled := trueAction(&invoked)
	disabled := trueAction(&invoked)
	disabled.enabled = false
	ts.PumpWidget(NewActions(testActions(enabled),
		NewActions(testActions(disabled), NewContainer(nil, ContainerOpts{Key: key}),
			ActionsOpts{Dispatcher: newTestDispatcher(c.collect)}),
		ActionsOpts{Dispatcher: newTestDispatcher1(c.collect)}))
	ts.Pump()
	result := ActionsInvoke(key.CurrentContext(), testIntentV{})
	if result != nil || invoked || c.intent != nil || c.action != nil || c.dispatcher != nil {
		t.Errorf("result = %v, invoked = %v, collected = %v / %v / %v; want nothing", result, invoked, c.intent, c.action, c.dispatcher)
	}
}

func TestCanListenToEnabledStateOfActions(t *testing.T) {
	ts := gluttertest.New(t)
	key := containerKey()
	var invoked1, invoked2, invoked3 bool
	action1, action2, action3 := trueAction(&invoked1), trueAction(&invoked2), trueAction(&invoked3)
	enabled1, enabled2, enabled3 := true, true, true
	action1.AddActionListener(func(a Action) { enabled1 = a.IsEnabled(testIntentV{}) })
	action1.setEnabled(false)
	if enabled1 {
		t.Fatal("enabled1 should be false")
	}
	action2.AddActionListener(func(a Action) { enabled2 = a.IsEnabled(secondTestIntent{}) })
	action2.setEnabled(false)
	if enabled2 {
		t.Fatal("enabled2 should be false")
	}
	action3.AddActionListener(func(a Action) { enabled3 = a.IsEnabled(thirdTestIntent{}) })
	action3.setEnabled(false)
	if enabled3 {
		t.Fatal("enabled3 should be false")
	}

	ts.PumpWidget(NewActions(ActionMap{
		IntentTypeOf[testIntentV](): action1, IntentTypeOf[secondTestIntent](): action2,
	}, NewActions(ActionMap{IntentTypeOf[thirdTestIntent](): action3}, NewContainer(nil, ContainerOpts{Key: key}))))

	result := ActionsMaybeInvoke(key.CurrentContext(), testIntentV{})
	if enabled1 || result != nil || invoked1 {
		t.Errorf("enabled1 = %v, result = %v, invoked1 = %v; want false, nil, false", enabled1, result, invoked1)
	}
	action1.setEnabled(true)
	result = ActionsInvoke(key.CurrentContext(), testIntentV{})
	if !enabled1 || result != true || !invoked1 {
		t.Errorf("enabled1 = %v, result = %v, invoked1 = %v; want true, true, true", enabled1, result, invoked1)
	}

	var enabledChanged *bool
	ts.PumpWidget(NewActions(ActionMap{
		IntentTypeOf[testIntentV](): action1, IntentTypeOf[secondTestIntent](): action2,
	}, NewActionListener(func(a Action) {
		v := a.IsEnabled(thirdTestIntent{})
		enabledChanged = &v
	}, action2, NewActions(ActionMap{IntentTypeOf[thirdTestIntent](): action3}, NewContainer(nil, ContainerOpts{Key: key})))))
	ts.Pump()
	result = ActionsMaybeInvoke(key.CurrentContext(), secondTestIntent{})
	if enabledChanged != nil || enabled2 || result != nil || invoked2 {
		t.Errorf("enabledChanged = %v, enabled2 = %v, result = %v, invoked2 = %v", enabledChanged, enabled2, result, invoked2)
	}
	action2.setEnabled(true)
	if enabledChanged == nil || !*enabledChanged {
		t.Errorf("enabledChanged = %v, want true", enabledChanged)
	}
	result = ActionsInvoke(key.CurrentContext(), secondTestIntent{})
	if !enabled2 || result != true || !invoked2 {
		t.Errorf("enabled2 = %v, result = %v, invoked2 = %v; want true, true, true", enabled2, result, invoked2)
	}

	counts := func(a1, a2, a3 int) {
		t.Helper()
		if len(action1.listeners) != a1 || len(action2.listeners) != a2 || len(action3.listeners) != a3 {
			t.Errorf("listeners = %d, %d, %d; want %d, %d, %d", len(action1.listeners), len(action2.listeners), len(action3.listeners), a1, a2, a3)
		}
	}
	ts.PumpWidget(NewActions(ActionMap{IntentTypeOf[testIntentV](): action1},
		NewActions(ActionMap{IntentTypeOf[thirdTestIntent](): action3}, NewContainer(nil, ContainerOpts{Key: key}))))
	counts(2, 1, 2)

	ts.PumpWidget(NewActions(ActionMap{IntentTypeOf[testIntentV](): action1, IntentTypeOf[thirdTestIntent](): action3},
		NewContainer(nil, ContainerOpts{Key: key})))
	counts(2, 1, 2)

	ts.PumpWidget(NewActions(ActionMap{IntentTypeOf[testIntentV](): action1}, NewContainer(nil, ContainerOpts{Key: key})))
	counts(2, 1, 1)

	ts.PumpWidget(NewContainer(nil))
	ts.Pump()
	counts(1, 1, 1)
}

// testButtonOpts are the parameters of newTestButton.
type testButtonOpts struct {
	focusNode *FocusNode
	autofocus bool
	onPressed func()
}

// newTestButton is TestButton (button_tester.dart): a very basic button for
// widget tests, whose Semantics node is not built.
func newTestButton(child Widget, opts testButtonOpts) Widget {
	return NewFocusableActionDetector(
		NewGestureDetector(child, GestureDetectorOpts{OnTap: opts.onPressed}),
		FocusableActionDetectorOpts{Disabled: opts.onPressed == nil, FocusNode: opts.focusNode, Autofocus: opts.autofocus})
}

func sizedBox100(key *GlobalKey[State]) Widget {
	return NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(100.0), Key: key})
}

// detectorFixture is the setUp of the FocusableActionDetector group.
type detectorFixture struct {
	invoked, hovering, focusing bool
	focusNode                   *FocusNode
	testAct                     *testAction
}

func newDetectorFixture(t *testing.T) *detectorFixture {
	f := &detectorFixture{focusNode: NewFocusNode(FocusNodeOpts{DebugLabel: "Test Node"})}
	t.Cleanup(f.focusNode.Dispose)
	f.testAct = newTestAction(func(Intent) any {
		f.invoked = true
		return true
	})
	return f
}

type pumpDetectorOpts struct {
	disabled, directional, noCallbacks bool
}

func (f *detectorFixture) pump(ts *gluttertest.Tester, key *GlobalKey[State], o pumpDetectorOpts) {
	mode := NavigationModeTraditional
	if o.directional {
		mode = NavigationModeDirectional
	}
	opts := FocusableActionDetectorOpts{
		Disabled:  o.disabled,
		FocusNode: f.focusNode,
		Shortcuts: ShortcutMap{{NewSingleActivator(services.LogicalKeyboardKeyEnter), testIntentV{}}},
		Actions:   ActionMap{IntentTypeOf[testIntentV](): f.testAct},
	}
	if !o.noCallbacks {
		opts.OnShowHoverHighlight = func(v bool) { f.hovering = v }
		opts.OnShowFocusHighlight = func(v bool) { f.focusing = v }
	}
	ts.PumpWidget(NewMediaQuery(NewMediaQueryData(MediaQueryDataOpts{NavigationMode: mode}),
		NewCenter(NewActions(ActionMap{}, NewFocusableActionDetector(sizedBox100(key), opts),
			ActionsOpts{Dispatcher: newTestDispatcher1(nil)}))))
	ts.Pump()
}

func TestFocusableActionDetectorKeepsTrackOfFocusAndHoverEvenWhenDisabled(t *testing.T) {
	ts := gluttertest.New(t)
	f := newDetectorFixture(t)
	ts.Binding.FocusManager().SetHighlightStrategy(FocusHighlightStrategyAlwaysTraditional)
	key := containerKey()

	f.pump(ts, key, pumpDetectorOpts{})
	f.focusNode.RequestFocus()
	ts.Pump()
	ts.Hover(ts.Center(gluttertest.ByKey(key)))
	ts.Pump()
	ts.SendKeyEvent(services.LogicalKeyboardKeyEnter)
	if !f.hovering || !f.focusing || !f.invoked {
		t.Errorf("hovering = %v, focusing = %v, invoked = %v; want all true", f.hovering, f.focusing, f.invoked)
	}

	f.invoked = false
	f.pump(ts, key, pumpDetectorOpts{disabled: true})
	if f.hovering || f.focusing {
		t.Errorf("hovering = %v, focusing = %v; want false, false", f.hovering, f.focusing)
	}
	ts.SendKeyEvent(services.LogicalKeyboardKeyEnter)
	ts.Pump()
	if f.invoked {
		t.Error("a disabled detector should not invoke its actions")
	}
	f.pump(ts, key, pumpDetectorOpts{})
	if f.focusing || !f.hovering {
		t.Errorf("focusing = %v, hovering = %v; want false, true", f.focusing, f.hovering)
	}
	f.pump(ts, key, pumpDetectorOpts{disabled: true})
	if f.focusing || f.hovering {
		t.Errorf("focusing = %v, hovering = %v; want false, false", f.focusing, f.hovering)
	}
	ts.Hover(ui.Offset{})
	f.pump(ts, key, pumpDetectorOpts{})
	if f.hovering || f.focusing {
		t.Errorf("hovering = %v, focusing = %v; want false, false", f.hovering, f.focusing)
	}
}

func TestFocusableActionDetectorShowsFocusHighlightWhenFocusedAndDisabled(t *testing.T) {
	ts := gluttertest.New(t)
	f := newDetectorFixture(t)
	ts.Binding.FocusManager().SetHighlightStrategy(FocusHighlightStrategyAlwaysTraditional)
	key := containerKey()

	f.pump(ts, key, pumpDetectorOpts{})
	ts.Pump()
	if f.focusing {
		t.Fatal("focusing should be false")
	}
	f.pump(ts, key, pumpDetectorOpts{})
	f.focusNode.RequestFocus()
	ts.Pump()
	if !f.focusing {
		t.Fatal("focusing should be true")
	}

	f.focusing = false
	for range 2 {
		f.pump(ts, key, pumpDetectorOpts{disabled: true})
		f.focusNode.RequestFocus()
		ts.Pump()
		if f.focusing {
			t.Fatal("a disabled detector should not show the focus")
		}
	}
	// In directional navigation, focus should show, even if disabled.
	f.pump(ts, key, pumpDetectorOpts{disabled: true, directional: true})
	f.focusNode.RequestFocus()
	ts.Pump()
	if !f.focusing {
		t.Error("focus should show in directional navigation, even if disabled")
	}
}

func TestFocusableActionDetectorCanBeUsedWithoutCallbacks(t *testing.T) {
	ts := gluttertest.New(t)
	f := newDetectorFixture(t)
	ts.Binding.FocusManager().SetHighlightStrategy(FocusHighlightStrategyAlwaysTraditional)
	key := containerKey()
	o := pumpDetectorOpts{noCallbacks: true}

	f.pump(ts, key, o)
	f.focusNode.RequestFocus()
	ts.Pump()
	ts.Hover(ts.Center(gluttertest.ByKey(key)))
	ts.Pump()
	ts.SendKeyEvent(services.LogicalKeyboardKeyEnter)
	if f.hovering || f.focusing || !f.invoked {
		t.Errorf("hovering = %v, focusing = %v, invoked = %v; want false, false, true", f.hovering, f.focusing, f.invoked)
	}

	f.invoked = false
	o.disabled = true
	f.pump(ts, key, o)
	ts.SendKeyEvent(services.LogicalKeyboardKeyEnter)
	ts.Pump()
	if f.invoked || f.hovering || f.focusing {
		t.Errorf("invoked = %v, hovering = %v, focusing = %v; want all false", f.invoked, f.hovering, f.focusing)
	}
	o.disabled = false
	f.pump(ts, key, o)
	o.disabled = true
	f.pump(ts, key, o)
	ts.Hover(ui.Offset{})
	o.disabled = false
	f.pump(ts, key, o)
	if f.hovering || f.focusing {
		t.Errorf("hovering = %v, focusing = %v; want false, false", f.hovering, f.focusing)
	}
}

func TestFocusableActionDetectorCanPreventItsDescendantsFromBeingFocusable(t *testing.T) {
	ts := gluttertest.New(t)
	buttonNode := NewFocusNode(FocusNodeOpts{DebugLabel: "Test"})
	defer buttonNode.Dispose()
	build := func(opts FocusableActionDetectorOpts) Widget {
		return testWidgetsApp(testWidgetsAppOpts{home: NewFocusableActionDetector(
			newTestButton(navTextExt("Test"), testButtonOpts{focusNode: buttonNode, onPressed: func() {}}), opts)})
	}
	ts.PumpWidget(build(FocusableActionDetectorOpts{}))
	if buttonNode.HasFocus() {
		t.Fatal("the button should not have the focus yet")
	}
	buttonNode.RequestFocus()
	ts.Pump()
	if !buttonNode.HasFocus() {
		t.Fatal("the button should be focusable")
	}

	ts.PumpWidget(build(FocusableActionDetectorOpts{DescendantsAreNotFocusable: true}))
	if buttonNode.HasFocus() {
		t.Fatal("the button should lose the focus")
	}
	buttonNode.RequestFocus()
	ts.Pump()
	if buttonNode.HasFocus() {
		t.Error("the button should not be focusable")
	}
}

func TestFocusableActionDetectorCanPreventItsDescendantsFromBeingTraversable(t *testing.T) {
	ts := gluttertest.New(t)
	node1 := NewFocusNode(FocusNodeOpts{DebugLabel: "Button Node 1"})
	node2 := NewFocusNode(FocusNodeOpts{DebugLabel: "Button Node 2"})
	skipTraversal := NewFocusNode(FocusNodeOpts{SkipTraversal: true})
	defer node1.Dispose()
	defer node2.Dispose()
	defer skipTraversal.Dispose()
	build := func(opts FocusableActionDetectorOpts) Widget {
		opts.FocusNode = skipTraversal
		return testWidgetsApp(testWidgetsAppOpts{home: NewFocusableActionDetector(NewColumn([]Widget{
			newTestButton(navTextExt("Node 1"), testButtonOpts{focusNode: node1, onPressed: func() {}}),
			newTestButton(navTextExt("Node 2"), testButtonOpts{focusNode: node2, onPressed: func() {}}),
		}), opts)})
	}
	ts.PumpWidget(build(FocusableActionDetectorOpts{}))
	node1.RequestFocus()
	ts.Pump()
	if !node1.HasFocus() || node2.HasFocus() {
		t.Fatalf("node1.HasFocus = %v, node2.HasFocus = %v; want true, false", node1.HasFocus(), node2.HasFocus())
	}
	ts.Binding.FocusManager().PrimaryFocus().NextFocus()
	ts.Pump()
	if node1.HasFocus() || !node2.HasFocus() {
		t.Fatalf("node1.HasFocus = %v, node2.HasFocus = %v; want false, true", node1.HasFocus(), node2.HasFocus())
	}

	ts.PumpWidget(build(FocusableActionDetectorOpts{DescendantsAreNotTraversable: true}))
	node1.RequestFocus()
	ts.Pump()
	if !node1.HasFocus() || node2.HasFocus() {
		t.Fatalf("node1.HasFocus = %v, node2.HasFocus = %v; want true, false", node1.HasFocus(), node2.HasFocus())
	}
	ts.Binding.FocusManager().PrimaryFocus().NextFocus()
	ts.Pump()
	if node1.HasFocus() || node2.HasFocus() {
		t.Errorf("node1.HasFocus = %v, node2.HasFocus = %v; want false, false", node1.HasFocus(), node2.HasFocus())
	}
}

func TestFocusableActionDetectorCanExcludeFocusSemantics(t *testing.T) {
	ts := gluttertest.New(t)
	build := func(opts FocusableActionDetectorOpts) Widget {
		return testWidgetsApp(testWidgetsAppOpts{home: NewFocusableActionDetector(NewColumn([]Widget{
			newTestButton(navTextExt("Button 1"), testButtonOpts{onPressed: func() {}}),
			newTestButton(navTextExt("Button 2"), testButtonOpts{onPressed: func() {}}),
		}), opts)})
	}
	firstFocus := func() *Focus {
		return ts.Widgets(gluttertest.Descendant(gluttertest.ByType[*FocusableActionDetector](), gluttertest.ByType[*Focus]()))[0].(*Focus)
	}
	ts.PumpWidget(build(FocusableActionDetectorOpts{}))
	if !firstFocus().IncludeSemantics() {
		t.Error("the Focus should include semantics by default")
	}
	ts.PumpWidget(build(FocusableActionDetectorOpts{ExcludeFocusSemantics: true}))
	if firstFocus().IncludeSemantics() {
		t.Error("the Focus should not include semantics")
	}
}

// navTextExt is a Text of a fixed size (the test environment has no fonts).
func navTextExt(s string) Widget {
	return NewSizedBox(NewText(s, TextOpts{TextDirection: ui.TextDirectionLTR}),
		SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(20.0)})
}

func TestFocusableActionDetectorChangesMouseCursorWhenHovered(t *testing.T) {
	ts := gluttertest.New(t)
	build := func(cursor services.MouseCursor) Widget {
		return NewMouseRegion(NewFocusableActionDetector(NewContainer(nil), FocusableActionDetectorOpts{
			MouseCursor:          cursor,
			OnShowHoverHighlight: func(bool) {},
			OnShowFocusHighlight: func(bool) {},
		}), MouseRegionOpts{Cursor: services.SystemMouseCursors.Forbidden})
	}
	lastCursor := func() string {
		got := ts.ActivatedCursors()
		if len(got) == 0 {
			return ""
		}
		return got[len(got)-1].Kind
	}
	ts.PumpWidget(build(services.SystemMouseCursors.Text))
	ts.Hover(ui.Offset{Dx: 1, Dy: 1})
	ts.Pump()
	if got := lastCursor(); got != "text" {
		t.Errorf("cursor = %q, want text", got)
	}
	// Test default.
	ts.PumpWidget(build(nil))
	if got := lastCursor(); got != "forbidden" {
		t.Errorf("cursor = %q, want forbidden", got)
	}
}

func TestActionSubclasses(t *testing.T) {
	t.Run("CallbackAction passes correct intent when invoked", func(t *testing.T) {
		var passed Intent
		action := NewCallbackAction[testIntent](func(i testIntent) any {
			passed = i
			return true
		})
		intent := testIntentV{}
		action.Invoke(intent)
		if passed != Intent(intent) {
			t.Errorf("passed intent = %v, want %v", passed, intent)
		}
	})

	t.Run("VoidCallbackAction", func(t *testing.T) {
		called := false
		NewVoidCallbackAction().Invoke(VoidCallbackIntent{Callback: func() { called = true }})
		if !called {
			t.Error("the callback should have been called")
		}
	})

	t.Run("the default ToKeyEventResult delegates to ConsumesKey", func(t *testing.T) {
		for consumes, want := range map[bool]KeyEventResult{false: KeyEventResultSkipRemainingHandlers, true: KeyEventResultHandled} {
			got := newDefaultToKeyEventResultAction(consumes).ToKeyEventResult(defaultToKeyEventResultIntent{}, nil)
			if got != want {
				t.Errorf("consumesKey %v: result = %v, want %v", consumes, got, want)
			}
		}
	})
}

func describeProperties(p *foundation.DiagnosticPropertiesBuilder) []string {
	var out []string
	for _, n := range p.Properties {
		if !n.IsFiltered(foundation.DiagnosticLevelInfo) {
			out = append(out, n.String())
		}
	}
	return out
}

func TestActionsDiagnostics(t *testing.T) {
	skipUnlessDebug(t)

	t.Run("default Intent debugFillProperties", func(t *testing.T) {
		builder := foundation.NewDiagnosticPropertiesBuilder()
		testIntentV{}.DebugFillProperties(builder)
		if got := describeProperties(builder); len(got) != 0 {
			t.Errorf("description = %q, want none", got)
		}
	})

	t.Run("default Actions debugFillProperties", func(t *testing.T) {
		builder := foundation.NewDiagnosticPropertiesBuilder()
		NewActions(ActionMap{}, NewContainer(nil), ActionsOpts{Dispatcher: NewActionDispatcher()}).DebugFillProperties(builder)
		got := describeProperties(builder)
		if len(got) != 2 || !strings.HasPrefix(got[0], "dispatcher: ") || got[1] != "actions: {}" {
			t.Errorf("description = %q", got)
		}
	})

	t.Run("Actions implements debugFillProperties", func(t *testing.T) {
		builder := foundation.NewDiagnosticPropertiesBuilder()
		NewActions(testActions(newTestAction(func(Intent) any { return nil })), NewContainer(nil),
			ActionsOpts{Key: foundation.NewValueKey("foo"), Dispatcher: NewActionDispatcher()}).DebugFillProperties(builder)
		got := describeProperties(builder)
		if len(got) != 2 || !strings.HasPrefix(got[0], "dispatcher: ") || !strings.HasPrefix(got[1], "actions: {testIntentV: ") {
			t.Errorf("description = %q", got)
		}
	})
}

// overrideLevel makes the action of one level of an overridable chain; bc is the
// context of the level.
type overrideLevel func(bc BuildContext) Action

// overridable makes a level whose action is overridable from its own context.
func overridable(a Action) overrideLevel {
	return func(bc BuildContext) Action { return NewOverridableAction(a, bc) }
}

// overrideChain builds nested Actions widgets, one per level, each bound to the
// LogIntent, and returns the contexts: contexts[i] is the context of the Builder
// that makes level i, and the last one is the invoking context, below all of them.
func overrideChain(ts *gluttertest.Tester, levels ...overrideLevel) []BuildContext {
	contexts := make([]BuildContext, len(levels)+1)
	var build func(i int) Widget
	build = func(i int) Widget {
		return builderOf(func(bc BuildContext) Widget {
			contexts[i] = bc
			if i == len(levels) {
				return NewSizedBoxShrink(nil)
			}
			return NewActions(ActionMap{IntentTypeOf[logIntent](): levels[i](bc)}, build(i+1))
		})
	}
	ts.PumpWidget(build(0))
	return contexts
}

func requireInvocations(t *testing.T, log []string, want ...string) {
	t.Helper()
	if !slices.Equal(log, want) {
		t.Errorf("log = %q, want %q", log, want)
	}
}

var threeLevelLog = []string{
	"action1.invokeAsOverride-pre-super",
	"action2.invokeAsOverride-pre-super",
	"action3.invoke",
	"action2.invokeAsOverride-post-super",
	"action1.invokeAsOverride-post-super",
}

func TestActionOverriding(t *testing.T) {
	t.Run("Basic usage", func(t *testing.T) {
		ts := gluttertest.New(t)
		var log []string
		contexts := overrideChain(ts,
			overridable(newLogInvocationAction("action1")),
			overridable(newLogInvocationAction("action2")),
			overridable(newLogInvocationAction("action3")))
		ActionsInvoke(contexts[3], logIntent{log: &log})
		requireInvocations(t, log, threeLevelLog...)

		// Invoke from a different (higher) context.
		log = nil
		ActionsInvoke(contexts[2], logIntent{log: &log})
		requireInvocations(t, log, "action1.invokeAsOverride-pre-super", "action2.invoke", "action1.invokeAsOverride-post-super")

		log = nil
		ActionsInvoke(contexts[1], logIntent{log: &log})
		requireInvocations(t, log, "action1.invoke")
	})

	t.Run("Does not break after use", func(t *testing.T) {
		ts := gluttertest.New(t)
		var log []string
		contexts := overrideChain(ts,
			overridable(newLogInvocationAction("action1")),
			overridable(newLogInvocationAction("action2")),
			overridable(newLogInvocationAction("action3")))
		// Invoke a bunch of times and verify it still produces the same result.
		for _, i := range []int{3, 1, 3, 2, 2, 2, 1} {
			ActionsInvoke(contexts[i], logIntent{log: &log})
		}
		log = nil
		ActionsInvoke(contexts[3], logIntent{log: &log})
		requireInvocations(t, log, threeLevelLog...)
	})

	t.Run("Does not override if not overridable", func(t *testing.T) {
		ts := gluttertest.New(t)
		var log []string
		contexts := overrideChain(ts,
			overridable(newLogInvocationAction("action1")),
			func(BuildContext) Action { return newLogInvocationAction("action2") },
			overridable(newLogInvocationAction("action3")))
		ActionsInvoke(contexts[3], logIntent{log: &log})
		requireInvocations(t, log, "action2.invokeAsOverride-pre-super", "action3.invoke", "action2.invokeAsOverride-post-super")
	})

	t.Run("The final override controls isEnabled", func(t *testing.T) {
		ts := gluttertest.New(t)
		var log []string
		contexts := overrideChain(ts,
			overridable(newLogInvocationAction("action1")),
			overridable(newLogInvocationAction("action2", logOpts{disabled: true})),
			overridable(newLogInvocationAction("action3")))
		ActionsInvoke(contexts[3], logIntent{log: &log})
		requireInvocations(t, log, threeLevelLog...)

		log = nil
		contexts = overrideChain(ts,
			overridable(newLogInvocationAction("action1", logOpts{disabled: true})),
			overridable(newLogInvocationAction("action2")),
			overridable(newLogInvocationAction("action3")))
		ActionsInvoke(contexts[3], logIntent{log: &log})
		requireInvocations(t, log)
	})

	t.Run("The override can choose to defer isActionEnabled to the overridable", func(t *testing.T) {
		ts := gluttertest.New(t)
		var log []string
		contexts := overrideChain(ts,
			overridable(newLogInvocationAction("action1", logOpts{deferIsEnabled: true})),
			overridable(newLogInvocationAction("action2", logOpts{disabled: true})),
			overridable(newLogInvocationAction("action3")))
		ActionsInvoke(contexts[3], logIntent{log: &log})
		// Nothing, since the final override defers its isActionEnabled state to
		// action2, which is disabled.
		requireInvocations(t, log)

		contexts = overrideChain(ts,
			overridable(newLogInvocationAction("action1")),
			overridable(newLogInvocationAction("action2", logOpts{deferIsEnabled: true})),
			overridable(newLogInvocationAction("action3", logOpts{disabled: true})))
		ActionsInvoke(contexts[3], logIntent{log: &log})
		// The final override (action1) is enabled so all 3 actions are enabled.
		requireInvocations(t, log, threeLevelLog...)
	})

	t.Run("Throws on infinite recursions", func(t *testing.T) {
		skipUnlessDebug(t)
		ts := gluttertest.New(t)
		var log []string
		var setState StateSetter
		var action2Lookup BuildContext
		var invoking BuildContext
		ts.PumpWidget(builderOf(func(context1 BuildContext) Widget {
			return NewActions(ActionMap{IntentTypeOf[logIntent](): NewOverridableAction(newLogInvocationAction("action1"), context1)},
				NewStatefulBuilder(func(_ BuildContext, ss StateSetter) Widget {
					setState = ss
					actions := ActionMap{}
					if action2Lookup != nil {
						actions[IntentTypeOf[logIntent]()] = NewOverridableAction(newLogInvocationAction("action2"), action2Lookup)
					}
					return NewActions(actions, builderOf(func(context3 BuildContext) Widget {
						return NewActions(ActionMap{IntentTypeOf[logIntent](): NewOverridableAction(newLogInvocationAction("action3"), context3)},
							builderOf(func(context4 BuildContext) Widget {
								invoking = context4
								return NewSizedBoxShrink(nil)
							}))
					}))
				}))
		}))
		// Let action2 look up its override using a context below itself, so it will
		// find action3 as its override.
		if err := ts.TakeError(); err != nil {
			t.Fatalf("error: %v", err)
		}
		setState(func() { action2Lookup = invoking })
		ts.Pump()
		if err := ts.TakeError(); err != nil {
			t.Fatalf("error: %v", err)
		}
		requireTag(t, "ActionOverrideRecursion", func() { ActionsInvoke(invoking, logIntent{log: &log}) })
	})

	t.Run("Throws on invoking invalid override", func(t *testing.T) {
		skipUnlessDebug(t)
		ts := gluttertest.New(t)
		var log []string
		var invoking BuildContext
		ts.PumpWidget(builderOf(func(BuildContext) Widget {
			return NewActions(ActionMap{IntentTypeOf[logIntent](): newTestContextAction()},
				builderOf(func(bc BuildContext) Widget {
					return NewActions(ActionMap{IntentTypeOf[logIntent](): NewOverridableAction(newLogInvocationAction("action1"), bc)},
						builderOf(func(bc1 BuildContext) Widget {
							invoking = bc1
							return NewSizedBoxShrink(nil)
						}))
				}))
		}))
		requireTag(t, "ActionIntentMismatch", func() { ActionsInvoke(invoking, logIntent{log: &log}) })
	})

	t.Run("Make an overridable action overridable", func(t *testing.T) {
		ts := gluttertest.New(t)
		var log []string
		var invoking BuildContext
		ts.PumpWidget(builderOf(func(context1 BuildContext) Widget {
			return NewActions(ActionMap{IntentTypeOf[logIntent](): NewOverridableAction(newLogInvocationAction("action1"), context1)},
				builderOf(func(context2 BuildContext) Widget {
					return NewActions(ActionMap{IntentTypeOf[logIntent](): NewOverridableAction(newLogInvocationAction("action2"), context2)},
						builderOf(func(context3 BuildContext) Widget {
							nested := NewOverridableAction(NewOverridableAction(NewOverridableAction(
								newLogInvocationAction("action3"), context1), context2), context3)
							return NewActions(ActionMap{IntentTypeOf[logIntent](): nested},
								builderOf(func(context4 BuildContext) Widget {
									invoking = context4
									return NewSizedBoxShrink(nil)
								}))
						}))
				}))
		}))
		ActionsInvoke(invoking, logIntent{log: &log})
		requireInvocations(t, log, threeLevelLog...)
	})

	t.Run("Overriding Actions can change the intent", func(t *testing.T) {
		ts := gluttertest.New(t)
		var log, newLog []string
		contexts := overrideChain(ts,
			overridable(newLogInvocationAction("action1")),
			overridable(newLogInvocationAction("action2", logOpts{newLog: &newLog})),
			overridable(newLogInvocationAction("action3")))
		ActionsInvoke(contexts[3], logIntent{log: &log})
		requireInvocations(t, log, "action1.invokeAsOverride-pre-super", "action1.invokeAsOverride-post-super")
		requireInvocations(t, newLog, "action2.invokeAsOverride-pre-super", "action3.invoke", "action2.invokeAsOverride-post-super")
	})

	t.Run("Override non-context overridable Actions with a ContextAction", func(t *testing.T) {
		ts := gluttertest.New(t)
		var log []string
		var invokeContext BuildContext
		contexts := overrideChain(ts,
			// The default Action is a ContextAction.
			overridable(newLogInvocationContextAction("action1", &invokeContext)),
			overridable(newLogInvocationAction("action2", logOpts{disabled: true})),
			overridable(newLogInvocationAction("action3")))
		ActionsInvoke(contexts[3], logIntent{log: &log})
		requireInvocations(t, log, threeLevelLog...)
		// Action1 is a ContextAction and action2 and action3 are not. They should
		// not lose information.
		if invokeContext == nil || invokeContext != contexts[3] {
			t.Errorf("invoke context = %v, want the invoking context", invokeContext)
		}
	})

	t.Run("Override a ContextAction with a regular Action", func(t *testing.T) {
		ts := gluttertest.New(t)
		var log []string
		var invokeContext BuildContext
		contexts := overrideChain(ts,
			overridable(newLogInvocationAction("action1")),
			overridable(newLogInvocationContextAction("action2", &invokeContext, true)),
			overridable(newLogInvocationAction("action3")))
		ActionsInvoke(contexts[3], logIntent{log: &log})
		requireInvocations(t, log, threeLevelLog...)
		// Action2 is a ContextAction and action1 and action3 are regular actions.
		// Invoking action2 from action3 should still supply a non-nil context.
		if invokeContext == nil || invokeContext != contexts[3] {
			t.Errorf("invoke context = %v, want the invoking context", invokeContext)
		}
	})
}

func TestActionsFailureTags(t *testing.T) {
	skipUnlessDebug(t)

	t.Run("ActionNotEnabled", func(t *testing.T) {
		disabled := newTestAction(func(Intent) any { return nil })
		disabled.enabled = false
		requireTag(t, "ActionNotEnabled", func() { NewActionDispatcher().InvokeAction(disabled, testIntentV{}, nil) })
	})

	t.Run("ActionNotInitialized", func(t *testing.T) {
		var base ActionBase
		requireTag(t, "ActionNotInitialized", func() { base.IsEnabled(testIntentV{}) })
	})

	t.Run("ActionInvokeMissing", func(t *testing.T) {
		a := &bareAction{}
		InitActionBase(&a.ActionBase, a)
		requireTag(t, "ActionInvokeMissing", func() { a.Invoke(testIntentV{}) })
		c := &bareContextAction{}
		InitContextActionBase(&c.ContextActionBase, c)
		requireTag(t, "ActionInvokeMissing", func() { c.InvokeWithContext(testIntentV{}, nil) })
	})

	t.Run("ActionOverrideSelf", func(t *testing.T) {
		ts := gluttertest.New(t)
		var setState StateSetter
		var lookup, invoking BuildContext
		ts.PumpWidget(NewStatefulBuilder(func(_ BuildContext, ss StateSetter) Widget {
			setState = ss
			actions := ActionMap{}
			if lookup != nil {
				// The action looks its override up from a context below itself, which
				// finds the action itself.
				actions[IntentTypeOf[logIntent]()] = NewOverridableAction(newLogInvocationAction("action"), lookup)
			}
			return NewActions(actions, builderOf(func(bc BuildContext) Widget {
				invoking = bc
				return NewSizedBoxShrink(nil)
			}))
		}))
		setState(func() { lookup = invoking })
		ts.Pump()
		var log []string
		requireTag(t, "ActionOverrideSelf", func() { ActionsInvoke(invoking, logIntent{log: &log}) })
	})

	t.Run("ActionContextMissing", func(t *testing.T) {
		ts := gluttertest.New(t)
		var invokeContext BuildContext
		contexts := overrideChain(ts,
			overridable(newLogInvocationAction("action1")),
			overridable(newLogInvocationContextAction("action2", &invokeContext)))
		var log []string
		// The overridable context action is invoked without a context, but it has an
		// override above it that needs one.
		action := ActionsFind[logIntent](contexts[2])
		requireTag(t, "ActionContextMissing", func() { action.Invoke(logIntent{log: &log}) })
	})

	t.Run("FocusableActionDetectorCallbackDuringBuild", func(t *testing.T) {
		ts := gluttertest.New(t)
		var setState StateSetter
		changeStrategy := false
		ts.PumpWidget(NewColumn([]Widget{
			NewFocusableActionDetector(box100(nil), FocusableActionDetectorOpts{
				OnShowFocusHighlight: func(bool) {}, OnShowHoverHighlight: func(bool) {}}),
			NewStatefulBuilder(func(_ BuildContext, ss StateSetter) Widget {
				setState = ss
				if changeStrategy {
					// A change of the highlight mode in the middle of a build.
					ts.Binding.FocusManager().SetHighlightStrategy(FocusHighlightStrategyAlwaysTouch)
				}
				return NewSizedBoxShrink(nil)
			}),
		}))
		ts.Binding.FocusManager().SetHighlightStrategy(FocusHighlightStrategyAlwaysTraditional)
		ts.Pump()
		changeStrategy = true
		setState(func() {})
		ts.Pump()
		_ = requireErrorTag(t, takeFirst(t, ts), "FocusableActionDetectorCallbackDuringBuild")
	})
}

// bareAction is an action that does not implement Invoke.
type bareAction struct{ ActionBase }

func (a *bareAction) IntentType() reflect.Type { return reflect.TypeFor[testIntent]() }

// bareContextAction is a context action that does not implement InvokeWithContext.
type bareContextAction struct{ ContextActionBase }

func (a *bareContextAction) IntentType() reflect.Type { return reflect.TypeFor[testIntent]() }
