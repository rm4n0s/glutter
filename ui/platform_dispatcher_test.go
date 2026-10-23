package ui

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
)

// fakeHost records what the dispatcher asks of the engine.
type fakeHost struct {
	scheduled int
	rendered  []renderedScene
	exits     int
	posted    []func()
	err       error
}

type renderedScene struct {
	view  ViewID
	scene *Scene
}

func (h *fakeHost) ScheduleFrame() { h.scheduled++ }
func (h *fakeHost) RenderView(v ViewID, s *Scene) error {
	h.rendered = append(h.rendered, renderedScene{v, s})
	return h.err
}
func (h *fakeHost) RequestAppExit() { h.exits++ }
func (h *fakeHost) Post(fn func())  { h.posted = append(h.posted, fn) }

func (h *fakeHost) runPosted() {
	for len(h.posted) > 0 {
		fn := h.posted[0]
		h.posted = h.posted[1:]
		fn()
	}
}

func TestDispatcherViews(t *testing.T) {
	d := NewPlatformDispatcher(&fakeHost{})
	if d.ImplicitView() != nil || len(d.Views()) != 0 {
		t.Fatal("a new dispatcher has no views")
	}
	cfg := ViewConfiguration{
		PhysicalSize:        Size{Width: 800, Height: 600},
		DevicePixelRatio:    2,
		ViewPadding:         ViewPadding{Top: 10},
		ViewInsets:          ViewPadding{Bottom: 20},
		SystemGestureInsets: ViewPadding{Left: 5},
		Padding:             ViewPadding{Top: 10, Bottom: -20},
		Display:             Display{ID: 3, Size: Size{Width: 1920, Height: 1080}, DevicePixelRatio: 2, RefreshRate: 60},
	}
	v2 := d.AddView(2, ViewConfiguration{})
	v0 := d.AddView(ImplicitViewID, cfg)
	if got := d.Views(); !reflect.DeepEqual(got, []*View{v0, v2}) {
		t.Errorf("Views() = %v, want the views in ID order", got)
	}
	if d.ImplicitView() != v0 || d.View(2) != v2 || d.View(9) != nil {
		t.Error("view lookup")
	}
	tests := []struct {
		name string
		got  any
		want any
	}{
		{"id", v0.ViewID(), ImplicitViewID},
		{"dispatcher", v0.PlatformDispatcher(), d},
		{"size", v0.PhysicalSize(), Size{Width: 800, Height: 600}},
		{"dpr", v0.DevicePixelRatio(), 2.0},
		{"view padding", v0.ViewPadding(), ViewPadding{Top: 10}},
		{"view insets", v0.ViewInsets(), ViewPadding{Bottom: 20}},
		{"gesture insets", v0.SystemGestureInsets(), ViewPadding{Left: 5}},
		{"padding", v0.Padding(), ViewPadding{Top: 10, Bottom: -20}},
		{"display", v0.Display().ID, 3},
		{"constraints are tight", v0.PhysicalConstraints(), ViewConstraints{MinWidth: 800, MaxWidth: 800, MinHeight: 600, MaxHeight: 600}},
		{"dpr defaults to 1", v2.DevicePixelRatio(), 1.0},
	}
	for _, tt := range tests {
		if !reflect.DeepEqual(tt.got, tt.want) {
			t.Errorf("%s: got %v, want %v", tt.name, tt.got, tt.want)
		}
	}
	custom := d.AddView(5, ViewConfiguration{PhysicalConstraints: ViewConstraints{MinWidth: 1, MaxWidth: 2, MinHeight: 3, MaxHeight: 4}})
	if got := custom.PhysicalConstraints(); got.MaxHeight != 4 {
		t.Errorf("explicit constraints = %v", got)
	}
	d.RemoveView(2)
	if d.View(2) != nil {
		t.Error("RemoveView")
	}
}

func TestDispatcherMetricsChanged(t *testing.T) {
	d := NewPlatformDispatcher(&fakeHost{})
	v := d.AddView(ImplicitViewID, ViewConfiguration{})
	calls := 0
	d.OnMetricsChanged = func() { calls++ }
	d.UpdateViewConfiguration(ImplicitViewID, ViewConfiguration{PhysicalSize: Size{Width: 100, Height: 50}, DevicePixelRatio: 1.5})
	if calls != 1 || v.PhysicalSize().Width != 100 || v.DevicePixelRatio() != 1.5 {
		t.Errorf("calls %d, size %v, dpr %v", calls, v.PhysicalSize(), v.DevicePixelRatio())
	}
	d.UpdateViewConfiguration(42, ViewConfiguration{}) // no such view
	if calls != 1 {
		t.Errorf("a view that does not exist called OnMetricsChanged")
	}
}

func TestDispatcherFrames(t *testing.T) {
	host := &fakeHost{}
	d := NewPlatformDispatcher(host)
	var log []string
	var times []time.Duration
	d.OnBeginFrame = func(t time.Duration) { log = append(log, "begin"); times = append(times, t) }
	d.OnDrawFrame = func() { log = append(log, "draw") }
	d.ScheduleFrame()
	d.ScheduleFrame()
	if host.scheduled != 2 {
		t.Errorf("scheduled = %d, want 2 (the platform merges them)", host.scheduled)
	}
	for _, ft := range []time.Duration{16 * time.Millisecond, 33 * time.Millisecond, 20 * time.Millisecond, 50 * time.Millisecond} {
		d.BeginFrame(ft)
		d.DrawFrame()
	}
	if want := []string{"begin", "draw", "begin", "draw", "begin", "draw", "begin", "draw"}; !reflect.DeepEqual(log, want) {
		t.Errorf("log = %v", log)
	}
	// The third time went back, so it is raised to the second.
	want := []time.Duration{16 * time.Millisecond, 33 * time.Millisecond, 33 * time.Millisecond, 50 * time.Millisecond}
	if !reflect.DeepEqual(times, want) {
		t.Errorf("times = %v, want %v", times, want)
	}
	// With no callbacks nothing happens.
	bare := NewPlatformDispatcher(nil)
	bare.BeginFrame(time.Second)
	bare.DrawFrame()
	bare.ScheduleFrame()
	bare.RequestAppExit()
}

func TestViewRender(t *testing.T) {
	host := &fakeHost{}
	d := NewPlatformDispatcher(host)
	v := d.AddView(7, ViewConfiguration{})
	scene := &Scene{}
	if err := v.Render(scene); err != nil {
		t.Fatal(err)
	}
	if len(host.rendered) != 1 || host.rendered[0].view != 7 || host.rendered[0].scene != scene {
		t.Errorf("rendered = %v", host.rendered)
	}
	host.err = errors.New(tagEGLContextLost, "lost")
	err := v.Render(scene)
	appErr, ok := errors.FromError(err)
	if !ok || !appErr.HasRoute("EGLContextLost") {
		t.Errorf("Render did not return the host's error: %v", err)
	}
	nohost := NewPlatformDispatcher(nil).AddView(0, ViewConfiguration{})
	err = nohost.Render(scene)
	appErr, ok = errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T", err)
	}
	if !appErr.HasRoute("View.Render.PlatformHostMissing") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
}

func TestDispatcherInput(t *testing.T) {
	d := NewPlatformDispatcher(nil)
	if d.DispatchKeyData(KeyData{}) {
		t.Error("a key event nobody listens to is not handled")
	}
	d.DispatchPointerDataPacket(PointerDataPacket{}) // nobody listens: nothing happens
	var packets []PointerDataPacket
	var keys []KeyData
	d.OnPointerDataPacket = func(p PointerDataPacket) { packets = append(packets, p) }
	d.OnKeyData = func(k KeyData) bool { keys = append(keys, k); return k.Character == "a" }
	d.DispatchPointerDataPacket(PointerDataPacket{Data: []PointerData{{PhysicalX: 1}, {PhysicalX: 2}}})
	if len(packets) != 1 || len(packets[0].Data) != 2 || packets[0].Data[1].PhysicalX != 2 {
		t.Errorf("packets = %v", packets)
	}
	tests := []struct {
		key  KeyData
		want bool
	}{
		{KeyData{Character: "a"}, true},
		{KeyData{Character: "b"}, false},
	}
	for _, tt := range tests {
		if got := d.DispatchKeyData(tt.key); got != tt.want {
			t.Errorf("DispatchKeyData(%v) = %v, want %v", tt.key.Character, got, tt.want)
		}
	}
	if len(keys) != 2 {
		t.Errorf("keys = %v", keys)
	}
}

func TestPlatformConfiguration(t *testing.T) {
	en := NewLocale("en", "US")
	de := NewLocale("de")
	red := NewColor(0xFFFF0000)
	tests := []struct {
		name string
		cfg  PlatformConfiguration
		want []string
	}{
		{"same as default", PlatformConfiguration{}, nil},
		{"locales", PlatformConfiguration{Locales: []Locale{en, de}}, []string{"locale", "config"}},
		{"text scale", PlatformConfiguration{TextScaleFactor: 1.5}, []string{"scale", "config"}},
		{"text scale 1 is the default", PlatformConfiguration{TextScaleFactor: 1}, nil},
		{"dark", PlatformConfiguration{PlatformBrightness: BrightnessDark}, []string{"brightness", "config"}},
		{"light is the default", PlatformConfiguration{PlatformBrightness: BrightnessLight}, nil},
		{"accessibility", PlatformConfiguration{AccessibilityFeatures: AccessibilityFeatures{BoldText: true}}, []string{"accessibility", "config"}},
		{"accent color", PlatformConfiguration{AccentColor: &red}, []string{"config"}},
		{"interface font", PlatformConfiguration{UIFontFamily: "Cantarell", UIFontSize: 11}, []string{"config"}},
		{"interface font size", PlatformConfiguration{UIFontSize: 11}, []string{"config"}},
		{"monospace font", PlatformConfiguration{MonospaceFontFamily: "Adwaita Mono", MonospaceFontSize: 11}, []string{"config"}},
		{"monospace font size", PlatformConfiguration{MonospaceFontSize: 12}, []string{"config"}},
		{"all", PlatformConfiguration{Locales: []Locale{en}, TextScaleFactor: 2, PlatformBrightness: BrightnessDark,
			AccessibilityFeatures: AccessibilityFeatures{HighContrast: true}}, []string{"locale", "scale", "brightness", "accessibility", "config"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewPlatformDispatcher(nil)
			var log []string
			d.OnLocaleChanged = func() { log = append(log, "locale") }
			d.OnTextScaleFactorChanged = func() { log = append(log, "scale") }
			d.OnPlatformBrightnessChanged = func() { log = append(log, "brightness") }
			d.OnAccessibilityFeaturesChanged = func() { log = append(log, "accessibility") }
			d.OnPlatformConfigurationChanged = func() { log = append(log, "config") }
			d.UpdatePlatformConfiguration(tt.cfg)
			if !reflect.DeepEqual(log, tt.want) {
				t.Errorf("callbacks = %v, want %v", log, tt.want)
			}
			// Sending the same configuration again changes nothing.
			log = nil
			d.UpdatePlatformConfiguration(tt.cfg)
			if len(log) != 0 {
				t.Errorf("the same configuration called %v", log)
			}
		})
	}
}

func TestPlatformConfigurationGetters(t *testing.T) {
	d := NewPlatformDispatcher(nil)
	if d.Locale() != (Locale{}) || len(d.Locales()) != 0 || d.TextScaleFactor() != 1 ||
		d.PlatformBrightness() != BrightnessLight || d.AccessibilityFeatures() != (AccessibilityFeatures{}) || d.AccentColor() != nil {
		t.Error("defaults")
	}
	accent := NewColor(0xFF112233)
	d.UpdatePlatformConfiguration(PlatformConfiguration{
		Locales:               []Locale{NewLocale("fr", "CA"), NewLocale("en")},
		TextScaleFactor:       1.25,
		PlatformBrightness:    BrightnessDark,
		AccessibilityFeatures: AccessibilityFeatures{DisableAnimations: true},
		AccentColor:           &accent,
	})
	if d.Locale() != NewLocale("fr", "CA") || len(d.Locales()) != 2 || d.TextScaleFactor() != 1.25 ||
		d.PlatformBrightness() != BrightnessDark || !d.AccessibilityFeatures().DisableAnimations || *d.AccentColor() != accent {
		t.Errorf("getters after update: %v %v %v", d.Locales(), d.TextScaleFactor(), d.PlatformBrightness())
	}
	// The slice the caller passed and the one the dispatcher returns are copies.
	d.Locales()[0] = NewLocale("xx")
	if d.Locale() != NewLocale("fr", "CA") {
		t.Error("Locales must return a copy")
	}
	d.SetDisplays([]Display{{ID: 1}})
	if got := d.Displays(); len(got) != 1 || got[0].ID != 1 {
		t.Errorf("Displays = %v", got)
	}
}

func TestAppLifecycleState(t *testing.T) {
	d := NewPlatformDispatcher(nil)
	var states []AppLifecycleState
	d.OnAppLifecycleStateChanged = func(s AppLifecycleState) { states = append(states, s) }
	if d.AppLifecycleState() != AppLifecycleStateUnspecified {
		t.Error("initial state")
	}
	for _, s := range []AppLifecycleState{AppLifecycleStateResumed, AppLifecycleStateResumed, AppLifecycleStateInactive, AppLifecycleStateHidden} {
		d.SetAppLifecycleState(s)
	}
	want := []AppLifecycleState{AppLifecycleStateResumed, AppLifecycleStateInactive, AppLifecycleStateHidden}
	if !reflect.DeepEqual(states, want) || d.AppLifecycleState() != AppLifecycleStateHidden {
		t.Errorf("states = %v", states)
	}
}

func TestPost(t *testing.T) {
	host := &fakeHost{}
	ran := 0
	NewPlatformDispatcher(host).Post(func() { ran++ })
	if ran != 0 || len(host.posted) != 1 {
		t.Fatalf("a posted function must wait for the engine: ran %d, posted %d", ran, len(host.posted))
	}
	host.posted[0]()
	NewPlatformDispatcher(nil).Post(func() { ran++ }) // no engine: at once
	if ran != 2 {
		t.Errorf("ran = %d, want 2", ran)
	}
}

func TestRequestAppExit(t *testing.T) {
	host := &fakeHost{}
	NewPlatformDispatcher(host).RequestAppExit()
	if host.exits != 1 {
		t.Errorf("exits = %d", host.exits)
	}
}

func TestParseLocale(t *testing.T) {
	tests := []struct {
		in   string
		want Locale
		ok   bool
	}{
		{"en", Locale{LanguageCode: "en"}, true},
		{"en-US", Locale{LanguageCode: "en", CountryCode: "US"}, true},
		{"en_US", Locale{LanguageCode: "en", CountryCode: "US"}, true},
		{"en_US.UTF-8", Locale{LanguageCode: "en", CountryCode: "US"}, true},
		{"de_DE@euro", Locale{LanguageCode: "de", CountryCode: "DE"}, true},
		{"zh-Hans-CN", Locale{LanguageCode: "zh", ScriptCode: "Hans", CountryCode: "CN"}, true},
		{"ZH-hant-tw", Locale{LanguageCode: "zh", ScriptCode: "Hant", CountryCode: "TW"}, true},
		{"es-419", Locale{LanguageCode: "es", CountryCode: "419"}, true},
		{"fil", Locale{LanguageCode: "fil"}, true},
		{"C", Locale{}, false},
		{"POSIX", Locale{}, false},
		{"C.UTF-8", Locale{}, false},
		{"", Locale{}, false},
		{"e", Locale{}, false},
		{"1234", Locale{}, false},
		{"english", Locale{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, ok := ParseLocale(tt.in)
			if got != tt.want || ok != tt.ok {
				t.Errorf("ParseLocale(%q) = %v, %v; want %v, %v", tt.in, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestPlatformTypeStrings(t *testing.T) {
	tests := []struct{ got, want string }{
		{BrightnessDark.String(), "BrightnessDark"},
		{Brightness(9).String(), "Brightness(9)"},
		{AccessibilityFeatures{}.String(), "AccessibilityFeatures[]"},
		{AccessibilityFeatures{BoldText: true, ReduceMotion: true}.String(), "AccessibilityFeatures[boldText, reduceMotion]"},
		{Display{ID: 1, Size: Size{Width: 1920, Height: 1080}, DevicePixelRatio: 1.25, RefreshRate: 59.94}.String(),
			"Display(id: 1, size: 1920.0x1080.0, dpr: 1.3, refreshRate: 59.9)"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("got %q, want %q", tt.got, tt.want)
		}
	}
}

func TestRequestClose(t *testing.T) {
	tests := []struct {
		name     string
		response func(ViewID) AppExitResponse
		want     bool
	}{
		{"no hook lets the window close", nil, true},
		{"exit", func(ViewID) AppExitResponse { return AppExitResponseExit }, true},
		{"unspecified counts as exit", func(ViewID) AppExitResponse { return AppExitResponseUnspecified }, true},
		{"cancel keeps the window", func(ViewID) AppExitResponse { return AppExitResponseCancel }, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := NewPlatformDispatcher(nil)
			d.OnExitRequested = tc.response
			if got := d.RequestClose(3); got != tc.want {
				t.Errorf("RequestClose = %v, want %v", got, tc.want)
			}
		})
	}
	d := NewPlatformDispatcher(nil)
	var got ViewID
	d.OnExitRequested = func(v ViewID) AppExitResponse { got = v; return AppExitResponseExit }
	d.RequestClose(7)
	if got != 7 {
		t.Errorf("the hook got view %d, want 7", got)
	}
}

func TestAppExitEnumNames(t *testing.T) {
	tests := []struct{ got, want string }{
		{AppExitTypeCancelable.String(), "AppExitTypeCancelable"},
		{AppExitTypeRequired.String(), "AppExitTypeRequired"},
		{AppExitTypeUnspecified.String(), "AppExitTypeUnspecified"},
		{AppExitType(9).String(), "AppExitType(9)"},
		{AppExitResponseExit.String(), "AppExitResponseExit"},
		{AppExitResponseCancel.String(), "AppExitResponseCancel"},
		{AppExitResponseUnspecified.String(), "AppExitResponseUnspecified"},
		{AppExitResponse(9).String(), "AppExitResponse(9)"},
	}
	for _, tc := range tests {
		if tc.got != tc.want {
			t.Errorf("String = %q, want %q", tc.got, tc.want)
		}
	}
}

type cursorHost struct {
	fakeHost
	activated []string
}

func (h *cursorHost) ActivateSystemCursor(device int, kind string) {
	h.activated = append(h.activated, kind)
}

func TestActivateSystemCursor(t *testing.T) {
	host := &cursorHost{}
	NewPlatformDispatcher(host).ActivateSystemCursor(3, "click")
	if len(host.activated) != 1 || host.activated[0] != "click" {
		t.Errorf("activated = %v", host.activated)
	}
	// A host that cannot show cursors ignores the call.
	NewPlatformDispatcher(&fakeHost{}).ActivateSystemCursor(3, "click")
}

type clipboardHost struct {
	fakeHost
	text string
	has  bool
}

func (h *clipboardHost) SetClipboardText(_ context.Context, text string, done func(error)) {
	h.text, h.has = text, true
	if done != nil {
		h.Post(func() { done(nil) })
	}
}

func (h *clipboardHost) GetClipboardText(_ context.Context, done func(string, bool, error)) {
	h.Post(func() { done(h.text, h.has, nil) })
}

func TestClipboardCallsReachTheHost(t *testing.T) {
	host := &clipboardHost{}
	d := NewPlatformDispatcher(host)
	var setErr error = errors.New(tagNotCalled, "set was not called")
	d.SetClipboardText(context.Background(), "hello", func(err error) { setErr = err })
	var got string
	var gotOK bool
	d.GetClipboardText(context.Background(), func(text string, ok bool, err error) { got, gotOK = text, ok })
	host.runPosted()
	if setErr != nil || got != "hello" || !gotOK {
		t.Errorf("set error = %v, got %q ok=%v", setErr, got, gotOK)
	}
}

func TestClipboardUnavailable(t *testing.T) {
	host := &fakeHost{}
	d := NewPlatformDispatcher(host)
	var setErr, getErr error
	d.SetClipboardText(context.Background(), "x", func(err error) { setErr = err })
	d.GetClipboardText(context.Background(), func(_ string, ok bool, err error) { getErr = err })
	if setErr != nil || getErr != nil {
		t.Fatal("the results must not arrive before the calls return")
	}
	host.runPosted()
	routes := map[string]string{"set": "PlatformDispatcher.SetClipboardText.ClipboardUnavailable", "get": "PlatformDispatcher.GetClipboardText.ClipboardUnavailable"}
	for name, err := range map[string]error{"set": setErr, "get": getErr} {
		appErr, ok := errors.FromError(err)
		if !ok {
			t.Errorf("%s: expected *errors.Error, got %T", name, err)
			continue
		}
		if !appErr.HasRoute(routes[name]) {
			t.Errorf("%s: unexpected failure path: %s", name, appErr.Route())
		}
	}
}

type richClipboardHost struct {
	fakeHost
	system, primary ClipboardContent
}

func (h *richClipboardHost) SetClipboard(_ context.Context, kind ClipboardKind, c ClipboardContent, done func(error)) {
	if kind == ClipboardKindPrimary {
		h.primary = c
	} else {
		h.system = c
	}
	if done != nil {
		h.Post(func() { done(nil) })
	}
}

func (h *richClipboardHost) GetClipboard(_ context.Context, kind ClipboardKind, _ string, done func(ClipboardContent, error)) {
	c := h.system
	if kind == ClipboardKindPrimary {
		c = h.primary
	}
	h.Post(func() { done(c, nil) })
}

func TestRichClipboardCallsReachTheHost(t *testing.T) {
	host := &richClipboardHost{}
	d := NewPlatformDispatcher(host)
	var setErr error = errors.New(tagNotCalled, "set was not called")
	d.SetClipboard(context.Background(), ClipboardKindPrimary, ClipboardContent{Text: "picked", HasText: true}, func(err error) { setErr = err })
	var got ClipboardContent
	d.GetClipboard(context.Background(), ClipboardKindPrimary, ClipboardFormatText, func(c ClipboardContent, err error) { got = c })
	host.runPosted()
	if setErr != nil || got.Text != "picked" || !got.Has(ClipboardFormatText) || host.system.HasText {
		t.Errorf("set error = %v, got %+v, system %+v", setErr, got, host.system)
	}
}

func TestRichClipboardFallsBackToTheTextHost(t *testing.T) {
	host := &clipboardHost{}
	d := NewPlatformDispatcher(host)
	d.SetClipboard(context.Background(), ClipboardKindSystem, ClipboardContent{Text: "plain", HasText: true}, nil)
	var got ClipboardContent
	d.GetClipboard(context.Background(), ClipboardKindUnspecified, ClipboardFormatText, func(c ClipboardContent, err error) { got = c })
	host.runPosted()
	if host.text != "plain" || got.Text != "plain" || !got.HasText {
		t.Errorf("host text = %q, got %+v", host.text, got)
	}
}

func TestRichClipboardUnavailable(t *testing.T) {
	tests := []struct {
		name string
		set  ClipboardContent
		kind ClipboardKind
		fmt  string
	}{
		{"primary text", ClipboardContent{Text: "x", HasText: true}, ClipboardKindPrimary, ClipboardFormatText},
		{"system image", ClipboardContent{PNG: []byte("p")}, ClipboardKindSystem, ClipboardFormatPNG},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			host := &clipboardHost{}
			d := NewPlatformDispatcher(host)
			var setErr, getErr error
			d.SetClipboard(context.Background(), tc.kind, tc.set, func(err error) { setErr = err })
			d.GetClipboard(context.Background(), tc.kind, tc.fmt, func(_ ClipboardContent, err error) { getErr = err })
			host.runPosted()
			routes := map[string]string{"set": "PlatformDispatcher.SetClipboard.ClipboardUnavailable", "get": "PlatformDispatcher.GetClipboard.ClipboardUnavailable"}
			for name, err := range map[string]error{"set": setErr, "get": getErr} {
				appErr, ok := errors.FromError(err)
				if !ok || !appErr.HasRoute(routes[name]) {
					t.Errorf("%s: err = %v, want ClipboardUnavailable", name, err)
				}
			}
		})
	}
}

func TestClipboardContentFormats(t *testing.T) {
	c := ClipboardContent{PNG: []byte("p"), URIs: []string{"file:///a"}}
	if c.IsEmpty() || c.Has(ClipboardFormatText) || !c.Has(ClipboardFormatPNG) || !c.Has(ClipboardFormatURIList) || c.Has("x") {
		t.Errorf("formats of %+v are wrong", c)
	}
	if !(ClipboardContent{}).IsEmpty() || (ClipboardContent{HasText: true}).IsEmpty() {
		t.Error("IsEmpty is wrong")
	}
	if ClipboardKindPrimary.String() != "ClipboardKind.primary" || ClipboardKindUnspecified.String() != "ClipboardKind.system" || ClipboardKind(9).String() != "ClipboardKind.invalid" {
		t.Error("ClipboardKind.String is wrong")
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagEGLContextLost = "EGLContextLost"
	tagNotCalled      = "NotCalled"
)
