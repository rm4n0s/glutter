// Command android-spell is a development example (step 26.4): a Material page with one text field
// whose spell check is on, and that logs what the spell checker answers, so that the device tests can
// type a misspelled word with the dev events and read the result. It uses the public API only. On the
// desktop and the web the platform has no spell checker, and the field logs that.
//
// The misspelled words of the field are underlined in red, and a tap on one opens the toolbar with the
// suggestions. GLUTTER_DEMO is a list of words separated by commas:
//
//	text   puts the text "helo wrld" in the field at the start, with the cursor at its end
//
// Run it on a device or the emulator with:
//
//	go run ./cmd/glutter-android run ./examples/dev/android-spell
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"

	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

func main() {
	err := m.RunApp(newApp(), m.RunAppOpts{
		AppID:  "dev.glutter.androidspell",
		Window: m.WindowConfig{Title: "Android spell check", Size: m.Size{Width: 480, Height: 640}},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func newApp() m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "Android spell check",
		DebugShowCheckedModeBanner: m.Some(false),
		Home:                       &page{StatefulWidget: m.NewStatefulWidgetBase()},
	})
}

type page struct{ m.StatefulWidget }

func (p *page) CreateState() m.State { return &pageState{} }

type pageState struct {
	m.StateOf[*page]
	controller *m.TextEditingController
	config     *m.SpellCheckConfiguration
}

func (s *pageState) InitState() {
	s.controller = m.NewTextEditingController()
	for _, word := range strings.Split(os.Getenv("GLUTTER_DEMO"), ",") {
		if strings.TrimSpace(word) == "text" {
			s.controller.SetText("helo wrld")
		}
	}
	d := m.WidgetsBindingOf(s.Context()).PlatformDispatcher()
	s.config = m.NewSpellCheckConfiguration(m.SpellCheckConfigurationOpts{
		SpellCheckService: loggingService{inner: services.NewDefaultSpellCheckService(d), dispatcher: d},
	})
}

// loggingService asks the default spell check service and logs its answer.
type loggingService struct {
	inner      services.SpellCheckService
	dispatcher *ui.PlatformDispatcher
}

func (l loggingService) FetchSpellCheckSuggestions(ctx context.Context, locale ui.Locale, text string, done services.SpellCheckSuggestionsCallback) {
	l.inner.FetchSpellCheckSuggestions(ctx, locale, text, func(spans []services.SuggestionSpan, ok bool, err error) {
		var b strings.Builder
		for _, span := range spans {
			fmt.Fprintf(&b, "%d-%d%v ", span.Range.Start, span.Range.End, span.Suggestions)
		}
		slog.Info("spellcheck", "supported", l.dispatcher.SpellCheckSupported(), "text", text, "ok", ok, "err", err, "spans", strings.TrimSpace(b.String()))
		done(spans, ok, err)
	})
}

func (s *pageState) Build(m.BuildContext) m.Widget {
	return m.NewScaffold(m.ScaffoldOpts{
		AppBar: m.NewAppBar(m.AppBarOpts{Title: m.NewText("Android spell check")}),
		Body: m.NewColumn([]m.Widget{
			m.NewPadding(m.NewEdgeInsetsAll(8), m.NewTextField(m.TextFieldOpts{
				Autofocus:               true,
				Controller:              s.controller,
				Decoration:              &m.InputDecoration{LabelText: m.Some("Spell check")},
				SpellCheckConfiguration: s.config,
				OnChanged: func(text string) {
					slog.Info("field changed", "text", text, "composing", fmt.Sprintf("%d-%d", s.controller.Value().Composing.Start, s.controller.Value().Composing.End))
				},
			})),
		}),
	})
}
