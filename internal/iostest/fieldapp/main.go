// Command fieldapp is the app of the text input tests of step 35.5: a text field that takes the focus and prints a line with its text each
// time it changes, so that the Simulator test reads what the framework's field got from the text input (UITextInput) of the script.
package main

import (
	"fmt"
	"log"

	"github.com/rm4n0s/glutter/internal/iostest/inputlines"
	m "github.com/rm4n0s/glutter/material"
)

func main() {
	field := m.NewTextField(m.TextFieldOpts{
		Autofocus: true,
		OnChanged: func(text string) { fmt.Printf("%s %q\n", inputlines.Field, text) },
	})
	last := -1.0
	body := m.NewBuilder(func(bc m.BuildContext) m.Widget {
		// The bottom inset of the view, in logical pixels, each time it changes: the soft keyboard's (read above the Scaffold, which
		// takes the inset away from its body).
		if inset := m.MediaQueryOf(bc).ViewInsets.Bottom; inset != last {
			last = inset
			fmt.Printf("%s %d\n", inputlines.Insets, int(inset))
		}
		return m.NewScaffold(m.ScaffoldOpts{Body: m.NewPadding(m.NewEdgeInsetsAll(24), field)})
	})
	app := m.NewMaterialApp(m.MaterialAppOpts{Title: "Field", DebugShowCheckedModeBanner: m.Some(false), Home: body})
	if err := m.RunApp(app, m.RunAppOpts{AppID: "dev.glutter.fieldapp"}); err != nil {
		log.Fatal(err)
	}
}
