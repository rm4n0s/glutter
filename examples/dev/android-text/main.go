// Command android-text is a development example (step 25.8): a Material page with a single line text
// field, a password field and a multiline field, which logs what the framework hears of each (the
// changes, the action of the keyboard, the keyboard's height) so that the device tests can type into
// them with the dev events, read the clipboard and check the keyboard. It uses the public API only and
// runs on every platform.
//
// GLUTTER_DEMO is a list of words separated by commas:
//
//	single, password, multi   the field that has the focus from the start (single is the default)
//	copy                      puts "from the app" on the clipboard once the app has started
//	paste                     reads the clipboard and logs it
//
// Run it on a device or the emulator with:
//
//	go run ./cmd/glutter-android run ./examples/dev/android-text
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"

	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/services"
)

func main() {
	err := m.RunApp(newApp(), m.RunAppOpts{
		AppID:  "dev.glutter.androidtext",
		Window: m.WindowConfig{Title: "Android text", Size: m.Size{Width: 480, Height: 640}},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func newApp() m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "Android text",
		DebugShowCheckedModeBanner: m.Some(false),
		Home:                       &page{StatefulWidget: m.NewStatefulWidgetBase()},
	})
}

type page struct{ m.StatefulWidget }

func (p *page) CreateState() m.State { return &pageState{} }

type pageState struct {
	m.StateOf[*page]
	demo map[string]bool
}

func (s *pageState) InitState() {
	s.demo = map[string]bool{}
	for _, word := range strings.Split(os.Getenv("GLUTTER_DEMO"), ",") {
		s.demo[strings.TrimSpace(word)] = true
	}
	b := m.WidgetsBindingOf(s.Context())
	if s.demo["copy"] {
		time.AfterFunc(2*time.Second, func() {
			b.PlatformDispatcher().Post(func() {
				b.Clipboard().SetData(context.Background(), services.NewClipboardData("from the app"), func(err error) {
					slog.Info("clipboard copied", "err", err)
				})
			})
		})
	}
	if s.demo["paste"] {
		time.AfterFunc(2*time.Second, func() {
			b.PlatformDispatcher().Post(func() {
				b.Clipboard().GetData(context.Background(), "text/plain", func(data *services.ClipboardData, err error) {
					text := ""
					if data != nil {
						text = data.Text
					}
					slog.Info("clipboard pasted", "text", text, "err", err)
				})
			})
		})
	}
}

// field is one text field that logs under its name.
func (s *pageState) field(name, label string, focus bool, opts m.TextFieldOpts) m.Widget {
	opts.Autofocus = focus
	opts.Decoration = &m.InputDecoration{LabelText: m.Some(label)}
	c := m.NewTextEditingController()
	opts.Controller = c
	opts.OnChanged = func(text string) {
		slog.Info("field changed", "name", name, "text", text, "selection", fmt.Sprintf("%d-%d", c.Selection().BaseOffset, c.Selection().ExtentOffset))
	}
	opts.OnSubmitted = func(text string) { slog.Info("field submitted", "name", name, "text", text) }
	return m.NewPadding(m.NewEdgeInsetsAll(8), m.NewTextField(opts))
}

func (s *pageState) Build(bc m.BuildContext) m.Widget {
	mq := m.MediaQueryOf(bc)
	slog.Info("keyboard", "viewInsetsBottom", fmt.Sprintf("%.3f", mq.ViewInsets.Bottom), "dpr", mq.DevicePixelRatio,
		"padding", fmt.Sprintf("%.3f,%.3f", mq.Padding.Top, mq.Padding.Bottom))
	multi := m.Some(5)
	return m.NewScaffold(m.ScaffoldOpts{
		AppBar: m.NewAppBar(m.AppBarOpts{Title: m.NewText("Android text")}),
		Body: m.NewColumn([]m.Widget{
			s.field("single", "Single", s.demo["single"] || (!s.demo["password"] && !s.demo["multi"]), m.TextFieldOpts{}),
			s.field("password", "Password", s.demo["password"], m.TextFieldOpts{ObscureText: true}),
			s.field("multi", "Multi", s.demo["multi"], m.TextFieldOpts{MaxLines: multi, MinLines: multi,
				KeyboardType: m.Some(services.TextInputTypeMultiline)}),
		}),
	})
}
