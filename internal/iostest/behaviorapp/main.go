// Command behaviorapp is the app of the framework-behavior tests of step 35.9: a Cupertino app that shows one thing in the mode that the
// environment names, and prints a line when it happens, so that the Simulator test reads it:
//
//	scroll  a list that bounces (BouncingScrollPhysics is the default on iOS) and prints its offset when it changes;
//	back    a page pushed over a home page, which prints a line when it is popped (the back swipe from the left edge);
//	select  a text field with a text, which prints its selection when it changes (double tap, long press, and the handles).
package main

import (
	"fmt"
	"log"
	"os"

	c "github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/internal/iostest/behaviorlines"
)

func main() {
	var app *c.CupertinoApp
	switch os.Getenv(behaviorlines.ModeEnv) {
	case "back":
		app = c.NewCupertinoApp(c.CupertinoAppOpts{
			DebugShowCheckedModeBanner: c.Some(false),
			Routes: map[string]c.WidgetBuilder{
				"/":       func(c.BuildContext) c.Widget { return c.NewCupertinoPageScaffold(c.NewCenter(c.NewText("Home"))) },
				"/detail": func(c.BuildContext) c.Widget { return NewDetail() },
			},
			InitialRoute: "/detail",
		})
	case "select":
		controller := c.NewTextEditingController("Double tap a word to select it")
		last := ""
		controller.AddListener(func() {
			s := controller.Selection()
			if line := fmt.Sprintf("%d %d", s.BaseOffset, s.ExtentOffset); line != last {
				last = line
				fmt.Printf("%s %s\n", behaviorlines.Selection, line)
			}
		})
		app = c.NewCupertinoApp(c.CupertinoAppOpts{
			DebugShowCheckedModeBanner: c.Some(false),
			Home:                       c.NewCupertinoPageScaffold(c.NewSafeArea(c.NewPadding(c.NewEdgeInsetsAll(24), c.NewCupertinoTextField(c.CupertinoTextFieldOpts{Controller: controller})))),
		})
	default:
		controller := c.NewScrollController()
		last := 0.0
		controller.AddListener(func() {
			if off := controller.Offset(); off != last {
				last = off
				fmt.Printf("%s %.1f\n", behaviorlines.Offset, off)
			}
		})
		app = c.NewCupertinoApp(c.CupertinoAppOpts{
			DebugShowCheckedModeBanner: c.Some(false),
			Home: c.NewCupertinoPageScaffold(c.NewListViewBuilder(func(_ c.BuildContext, i int) c.Widget {
				return c.NewPadding(c.NewEdgeInsetsAll(16), c.NewText(fmt.Sprintf("Row %d", i)))
			}, c.ListViewOpts{Controller: controller, ItemCount: c.Some(40)})),
		})
	}
	if err := c.RunApp(app, c.RunAppOpts{AppID: "dev.glutter.behaviorapp"}); err != nil {
		log.Fatal(err)
	}
}

// Detail is the page that is pushed over the home page in the "back" mode.
type Detail struct{ c.StatefulWidget }

// NewDetail creates the page.
func NewDetail() *Detail { return &Detail{StatefulWidget: c.NewStatefulWidgetBase()} }

// CreateState creates the state.
func (d *Detail) CreateState() c.State { return &detailState{} }

type detailState struct{ c.StateOf[*Detail] }

func (s *detailState) InitState() { fmt.Println(behaviorlines.Pushed) }

func (s *detailState) Dispose() { fmt.Println(behaviorlines.Popped) }

func (s *detailState) Build(c.BuildContext) c.Widget {
	return c.NewCupertinoPageScaffold(c.NewCenter(c.NewText("Detail")), c.CupertinoPageScaffoldOpts{
		NavigationBar: c.NewCupertinoNavigationBar(c.CupertinoNavigationBarOpts{Middle: c.NewText("Detail")}),
	})
}
