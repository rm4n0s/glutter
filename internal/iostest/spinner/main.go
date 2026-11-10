// Command spinner is the app of the Metal background test of step 35.3: a progress indicator that animates for ever, so that the app
// draws a frame on every tick of the display until it is sent to the background.
package main

import (
	"log"

	m "github.com/rm4n0s/glutter/material"
)

func main() {
	app := m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "Spinner",
		DebugShowCheckedModeBanner: m.Some(false),
		Home:                       m.NewScaffold(m.ScaffoldOpts{Body: m.NewCenter(m.NewCircularProgressIndicator())}),
	})
	if err := m.RunApp(app, m.RunAppOpts{AppID: "dev.glutter.spinner"}); err != nil {
		log.Fatal(err)
	}
}
