// Command cupertino is the example of the Cupertino library (SPEC §31): an iOS styled app with a list of
// pages that push a page (with the parallax slide and the back swipe of iOS), present a sheet (that is
// dragged down to dismiss it) and show a modal popup. It uses only package cupertino, imported as c.
//
// Run it with: go run ./examples/cupertino
//
// On macOS: go run ./cmd/glutter-macos run ./examples/cupertino
//
// Environment: GLUTTER_SCREENSHOT=<file.png>, GLUTTER_SCREENSHOT_AFTER=<duration>; in debug builds
// GLUTTER_DEV_EVENTS=<script> replays input after the first frame (SPEC §29.6).
//
// As the first steps of the library (M34) are the app, the routes, the sheet, the page scaffold and the
// scrollbar, the rows of the list are text under a GestureDetector: the CupertinoListTile and the buttons
// come with the later steps.
package main

import (
	"log"

	c "github.com/rm4n0s/glutter/cupertino"
)

func main() {
	err := c.RunApp(NewMyApp(), c.RunAppOpts{
		AppID:  "dev.glutter.cupertino",
		Window: c.WindowConfig{Title: "Cupertino", Size: c.Size{Width: 390, Height: 760}},
	})
	if err != nil {
		log.Fatal(err)
	}
}

// MyApp is the app: a CupertinoApp with the home page.
type MyApp struct{ c.StatelessWidget }

// NewMyApp creates the app.
func NewMyApp() *MyApp { return &MyApp{} }

// Build builds the CupertinoApp.
func (a *MyApp) Build(c.BuildContext) c.Widget {
	// The engine paints what no widget covers with a development color: black stands behind the app, as the
	// root of an iOS app does, so that the gap above a sheet is black.
	return c.NewColoredBox(c.NewColor(0xFF000000), c.NewCupertinoApp(c.CupertinoAppOpts{
		Title:                      "Cupertino",
		Theme:                      c.NewCupertinoThemeData(c.CupertinoThemeDataOpts{Brightness: c.BrightnessLight}),
		DebugShowCheckedModeBanner: c.Some(false),
		Home:                       newHomePage(),
	}))
}

// row is a line of the list: text that taps.
func row(label string, onTap func()) c.Widget {
	return c.NewGestureDetector(
		c.NewPadding(c.NewEdgeInsetsSymmetric(14, 20),
			c.NewText(label, c.TextOpts{Style: c.NewTextStyle(c.TextStyleOpts{FontSize: c.Some(17.0), Color: c.Some(c.CupertinoColors.ActiveBlue.Color)})})),
		c.GestureDetectorOpts{OnTap: onTap, Behavior: c.HitTestBehaviorOpaque})
}

// page is a page with a title and a text, for the page that is pushed and for the sheet.
func page(title, text string) c.Widget {
	return c.NewCupertinoPageScaffold(c.NewBuilder(func(bc c.BuildContext) c.Widget {
		textTheme := c.CupertinoThemeOf(bc).TextTheme()
		return c.NewSizedBoxExpand(c.NewSafeArea(c.NewPadding(c.NewEdgeInsetsAll(20), c.NewColumn([]c.Widget{
			c.NewText(title, c.TextOpts{Style: textTheme.NavLargeTitleTextStyle()}),
			c.NewSizedBox(nil, c.SizedBoxOpts{Height: c.Some(12.0)}),
			c.NewText(text, c.TextOpts{Style: textTheme.TextStyle()}),
		}, c.ColumnOpts{CrossAxisAlignment: c.CrossAxisAlignmentStart}))))
	}))
}

type homePage struct{ c.StatelessWidget }

func newHomePage() *homePage { return &homePage{} }

func (p *homePage) Build(bc c.BuildContext) c.Widget {
	textTheme := c.CupertinoThemeOf(bc).TextTheme()
	// The pages fill their space (as a list does): the page of a route is laid out loosely when a sheet covers it.
	return c.NewCupertinoPageScaffold(c.NewSizedBoxExpand(c.NewSafeArea(c.NewColumn([]c.Widget{
		c.NewPadding(c.NewEdgeInsetsAll(20), c.NewText("Cupertino", c.TextOpts{Style: textTheme.NavLargeTitleTextStyle()})),
		row("Push a page", func() {
			c.NavigatorPush(bc, c.NewCupertinoPageRoute(func(c.BuildContext) c.Widget {
				return page("A page", "Swipe from the left edge to go back.")
			}))
		}),
		row("Show a sheet", func() {
			c.ShowCupertinoSheet(bc, c.ShowCupertinoSheetOpts{ShowDragHandle: true, Builder: func(c.BuildContext) c.Widget {
				return page("A sheet", "Drag down to dismiss it.")
			}})
		}),
		row("Show a popup", func() {
			c.ShowCupertinoModalPopup(bc, func(c.BuildContext) c.Widget {
				return c.NewColoredBox(c.CupertinoColors.SystemBackground.Color, c.NewSizedBox(page("A popup", "Tap outside to dismiss it."),
					c.SizedBoxOpts{Height: c.Some(260.0)}))
			})
		}),
	}, c.ColumnOpts{CrossAxisAlignment: c.CrossAxisAlignmentStart}))))
}
