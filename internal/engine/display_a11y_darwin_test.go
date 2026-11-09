//go:build darwin && !ios && glutter_display

package engine_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform/darwin/mainthread"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// axProbe is a widget that gives the test the binding, which runs functions on the UI thread.
type axProbe struct {
	m.StatelessWidget
	got func(b *widgets.WidgetsBinding)
}

func (p *axProbe) Build(bc m.BuildContext) m.Widget {
	p.got(m.WidgetsBindingOf(bc))
	return m.NewSizedBox(nil, m.SizedBoxOpts{})
}

// axLines is the tree below an accessibility object as lines "role|label|value|actions", depth first.
func axLines(obj objc.ID, depth int, out *[]string) {
	str := func(getter string) string { return objcx.String(obj.Send(objcx.Sel(getter))) }
	val := ""
	if v := obj.Send(objcx.Sel("accessibilityValue")); v != 0 {
		if objc.Send[bool](v, objcx.Sel("isKindOfClass:"), objc.ID(objcx.MustClass("NSString"))) {
			val = objcx.String(v)
		} else {
			val = "#" + objcx.String(v.Send(objcx.Sel("stringValue")))
		}
	}
	names := obj.Send(objcx.Sel("accessibilityActionNames"))
	var acts []string
	for i := range objcx.ArrayCount(names) {
		acts = append(acts, objcx.String(objcx.ArrayAt(names, i)))
	}
	*out = append(*out, strings.Repeat(" ", depth)+str("accessibilityRole")+"|"+str("accessibilityLabel")+"|"+val+"|"+strings.Join(acts, ","))
	kids := obj.Send(objcx.Sel("accessibilityChildren"))
	for i := range objcx.ArrayCount(kids) {
		axLines(objcx.ArrayAt(kids, i), depth+1, out)
	}
}

// TestDisplayAccessibilityTreeOfAnApp runs a Material app, asks the view for its accessibility children as VoiceOver
// does, and reads the tree that comes back: the roles, labels and values of a button, a switch, a slider and a text
// field, and the actions of the button (SPEC §29.10).
func TestDisplayAccessibilityTreeOfAnApp(t *testing.T) {
	t.Setenv("GLUTTER_MACOS_ACCESSORY", "1")
	t.Setenv("GLUTTER_DISPLAYLINK", "cv")
	var binding *widgets.WidgetsBinding
	var mu sync.Mutex
	probe := &axProbe{got: func(b *widgets.WidgetsBinding) { mu.Lock(); binding = b; mu.Unlock() }}
	home := m.NewScaffold(m.ScaffoldOpts{Body: m.NewColumn([]m.Widget{
		m.NewFilledButton(func() {}, m.NewText("Save")),
		m.NewSwitch(true, func(bool) {}),
		m.NewSlider(0.4, func(float64) {}),
		m.NewTextField(m.TextFieldOpts{Decoration: ptr(m.NewInputDecoration(m.InputDecorationOpts{LabelText: m.Some("Name")}))}),
		probe,
	})})
	app := m.NewMaterialApp(m.MaterialAppOpts{Home: home, DebugShowCheckedModeBanner: m.Some(false)})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var lines []string
	done := make(chan struct{})
	go func() {
		defer cancel()
		defer close(done)
		var b *widgets.WidgetsBinding
		for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
			mu.Lock()
			b = binding
			mu.Unlock()
			if b != nil {
				break
			}
		}
		if b == nil {
			t.Error("the app did not build")
			return
		}
		// The first question of an assistive technology, on the UI thread: it turns the semantics tree on.
		window := func() objc.ID {
			list := objc.ID(objcx.MustClass("NSApplication")).Send(objcx.Sel("sharedApplication")).Send(objcx.Sel("windows"))
			for i := range objcx.ArrayCount(list) {
				w := objcx.ArrayAt(list, i)
				if objcx.String(w.Send(objcx.Sel("title"))) == "a11y" {
					return w
				}
			}
			return 0
		}
		asked := make(chan struct{})
		b.Post(func() {
			if w := window(); w != 0 {
				_ = w.Send(objcx.Sel("contentView")).Send(objcx.Sel("accessibilityChildren"))
			}
			close(asked)
		})
		<-asked
		// The tree arrives with the next frames; ask again until the root has children.
		for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(200 * time.Millisecond) {
			read := make(chan []string, 1)
			b.Post(func() {
				var out []string
				if w := window(); w != 0 {
					roots := w.Send(objcx.Sel("contentView")).Send(objcx.Sel("accessibilityChildren"))
					for i := range objcx.ArrayCount(roots) {
						axLines(objcx.ArrayAt(roots, i), 0, &out)
					}
				}
				read <- out
			})
			out := <-read
			joined := strings.Join(out, "\n")
			if strings.Contains(joined, "Save") && strings.Contains(joined, "Name") {
				lines = out
				return
			}
			lines = out
		}
	}()
	var err error
	mainthread.Do(func() {
		err = m.RunAppContext(ctx, app, widgets.RunAppOpts{Window: widgets.WindowConfig{Title: "a11y", Size: ui.Size{Width: 400, Height: 500}}})
	})
	<-done
	if err != nil {
		t.Fatalf("RunApp: %v", err)
	}
	all := strings.Join(lines, "\n")
	t.Log("the tree:\n" + all)
	for _, want := range []string{"AXButton|Save|", "AXCheckBox|", "AXSlider|", "AXTextField|Name|"} {
		if !strings.Contains(all, want) {
			t.Errorf("the tree has no %q:\n%s", want, all)
		}
	}
	if !strings.Contains(all, "AXButton|Save||AXPress") {
		t.Errorf("the button has no press action:\n%s", all)
	}
	if !strings.Contains(all, "AXSlider|") || !strings.Contains(all, "AXIncrement") {
		t.Errorf("the slider cannot increment:\n%s", all)
	}
}

func ptr[T any](v T) *T { return &v }
