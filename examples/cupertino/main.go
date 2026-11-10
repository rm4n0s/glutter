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
// The app is a CupertinoTabScaffold with three tabs, each a CupertinoTabView with its own history. The first
// tab is a list under a CupertinoSliverNavigationBar whose large title collapses into the middle as the list
// scrolls; a pushed page flies its bar from the large title to the back label. The second tab shows the controls
// of step 34.4: a switch, a checkbox, radios, a slider, activity indicators and the two segmented controls. As the
// other controls of the library (M34) come with the later steps, the rows of the lists are text under a
// GestureDetector. The fourth tab, Popups (step 34.6), shows an alert dialog, an action sheet, a context menu that
// opens when its card is pressed and held, and a menu that grows from a button (and from a long press that swipes
// over its items).
package main

import (
	"fmt"
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

// tabBar is the tab bar of the app: three tabs with an icon and a label.
func tabBar() *c.CupertinoTabBar {
	item := func(icon c.IconData, label string) c.BottomNavigationBarItem {
		return c.NewBottomNavigationBarItem(c.NewIcon(icon), c.BottomNavigationBarItemOpts{Label: label})
	}
	return c.NewCupertinoTabBar([]c.BottomNavigationBarItem{
		item(c.CupertinoIcons.Home(), "Home"), item(c.CupertinoIcons.SliderHorizontal3(), "Controls"),
		item(c.CupertinoIcons.TextCursor(), "Text"), item(c.CupertinoIcons.ChatBubble(), "Popups"), item(c.CupertinoIcons.Settings(), "Settings"),
	})
}

// newHomePage is the tab scaffold, the home of the app.
func newHomePage() c.Widget {
	return c.NewCupertinoTabScaffold(tabBar(), func(_ c.BuildContext, index int) c.Widget {
		return c.NewCupertinoTabView(c.CupertinoTabViewOpts{
			DefaultTitle: c.Some([]string{"Home", "Controls", "Text", "Popups", "Settings"}[index]),
			Builder: func(bc c.BuildContext) c.Widget {
				switch index {
				case 0:
					return homeTab(bc)
				case 1:
					return controlsTab()
				case 2:
					return textTab()
				case 3:
					return popupsTab(bc)
				}
				return listTab("Settings", "Setting", 12)
			},
		})
	})
}

// homeTab is the list of the first tab, under the sliver bar with a large title.
func homeTab(bc c.BuildContext) c.Widget {
	rows := []c.Widget{
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
	}
	return c.NewCupertinoPageScaffold(c.NewCustomScrollView([]c.Widget{
		c.NewCupertinoSliverNavigationBar(c.CupertinoSliverNavigationBarOpts{LargeTitle: c.NewText("Cupertino")}),
		c.NewSliverList(c.NewSliverChildListDelegate(rows)),
	}))
}

// listTab is a tab with a long list under a sliver bar.
func listTab(title, label string, count int) c.Widget {
	var rows []c.Widget
	for i := 1; i <= count; i++ {
		rows = append(rows, row(fmt.Sprintf("%s %d", label, i), func() {}))
	}
	return c.NewCupertinoPageScaffold(c.NewCustomScrollView([]c.Widget{
		c.NewCupertinoSliverNavigationBar(c.CupertinoSliverNavigationBarOpts{LargeTitle: c.NewText(title)}),
		c.NewSliverList(c.NewSliverChildListDelegate(rows)),
	}))
}

// textTab is the page of the text: the fields in a list under the sliver bar. The first field has a selected word.
func textTab() c.Widget {
	return c.NewStatefulBuilder(func(_ c.BuildContext, _ c.StateSetter) c.Widget {
		pad := func(w c.Widget) c.Widget { return c.NewPadding(c.NewEdgeInsetsSymmetric(6, 16), w) }
		icon := func(d c.IconData) c.Widget {
			return c.NewPadding(c.NewEdgeInsetsSymmetric(0, 6), c.NewIcon(d, c.IconOpts{Color: c.Some(c.CupertinoColors.SystemGrey.Color)}))
		}
		rows := []c.Widget{
			pad(c.NewCupertinoSearchTextField()),
			pad(c.NewCupertinoTextField(c.CupertinoTextFieldOpts{Placeholder: c.Some("Name"), Prefix: icon(c.CupertinoIcons.PersonCropCircleFill()),
				ClearButtonMode: c.OverlayVisibilityModeEditing})),
			pad(c.NewCupertinoTextField(c.CupertinoTextFieldOpts{Placeholder: c.Some("Password"), ObscureText: true, Prefix: icon(c.CupertinoIcons.LockFill()),
				ClearButtonMode: c.OverlayVisibilityModeEditing})),
			pad(c.NewCupertinoTextField(c.CupertinoTextFieldOpts{Controller: c.NewTextEditingController("Double tap a word to select it"),
				Placeholder: c.Some("Notes"), MaxLines: c.Some(4), MinLines: c.Some(3), Padding: c.NewEdgeInsetsAll(10)})),
			pad(c.NewCupertinoTextFieldBorderless(c.CupertinoTextFieldOpts{Placeholder: c.Some("A field with no border")})),
		}
		return c.NewCupertinoPageScaffold(c.NewCustomScrollView([]c.Widget{
			c.NewCupertinoSliverNavigationBar(c.CupertinoSliverNavigationBarOpts{LargeTitle: c.NewText("Text")}),
			c.NewSliverList(c.NewSliverChildListDelegate(rows)),
		}))
	})
}

// controlsState is what the controls of the second tab show.
type controlsState struct {
	on      bool
	checked c.Opt[bool]
	radio   c.Opt[int]
	slider  float64
	segment string
	sliding string
}

// controlsTab is the page of the controls: a row for each, with its label and the control, in a list under the
// sliver bar. The state lives in the closure, one for each time the tab is made.
func controlsTab() c.Widget {
	st := &controlsState{on: true, checked: c.Some(true), radio: c.Some(1), slider: 0.4, segment: "Week", sliding: "Week"}
	return c.NewStatefulBuilder(func(_ c.BuildContext, setState c.StateSetter) c.Widget {
		set := func(f func()) { setState(f) }
		labelled := func(label string, control c.Widget) c.Widget {
			return c.NewPadding(c.NewEdgeInsetsSymmetric(8, 20), c.NewRow([]c.Widget{
				c.NewText(label, c.TextOpts{Style: c.NewTextStyle(c.TextStyleOpts{FontSize: c.Some(17.0), Color: c.Some(c.CupertinoColors.Label.Color)})}),
				control,
			}, c.RowOpts{MainAxisAlignment: c.MainAxisAlignmentSpaceBetween}))
		}
		segments := func() []c.CupertinoSegment[string] {
			var out []c.CupertinoSegment[string]
			for _, l := range []string{"Day", "Week", "Month"} {
				out = append(out, c.CupertinoSegment[string]{Value: l, Child: c.NewText(l)})
			}
			return out
		}
		rows := []c.Widget{
			labelled("Switch", c.NewCupertinoSwitch(st.on, func(v bool) { set(func() { st.on = v }) })),
			labelled("Checkbox", c.NewCupertinoCheckbox(st.checked, func(v c.Opt[bool]) { set(func() { st.checked = v }) })),
			labelled("Radio", c.NewRadioGroup(func(v c.Opt[int]) { set(func() { st.radio = v }) }, c.NewRow([]c.Widget{
				c.NewCupertinoRadio(1), c.NewSizedBox(nil, c.SizedBoxOpts{Width: c.Some(16.0)}), c.NewCupertinoRadio(2),
			}, c.RowOpts{MainAxisSize: c.MainAxisSizeMin}), c.RadioGroupOpts[int]{GroupValue: st.radio})),
			labelled("Slider", c.NewCupertinoSlider(st.slider, func(v float64) { set(func() { st.slider = v }) })),
			labelled("Activity", c.NewRow([]c.Widget{
				c.NewCupertinoActivityIndicator(), c.NewSizedBox(nil, c.SizedBoxOpts{Width: c.Some(16.0)}),
				c.NewSizedBox(c.NewCupertinoLinearActivityIndicator(st.slider), c.SizedBoxOpts{Width: c.Some(80.0)}),
			}, c.RowOpts{MainAxisSize: c.MainAxisSizeMin})),
			c.NewPadding(c.NewEdgeInsetsSymmetric(8, 0), c.NewCupertinoSegmentedControl(func(v string) { set(func() { st.segment = v }) }, segments(),
				c.CupertinoSegmentedControlOpts[string]{GroupValue: c.Some(st.segment)})),
			c.NewPadding(c.NewEdgeInsetsSymmetric(8, 12), c.NewCupertinoSlidingSegmentedControl(func(v string) { set(func() { st.sliding = v }) }, segments(),
				c.CupertinoSlidingSegmentedControlOpts[string]{GroupValue: c.Some(st.sliding)})),
		}
		return c.NewCupertinoPageScaffold(c.NewCustomScrollView([]c.Widget{
			c.NewCupertinoSliverNavigationBar(c.CupertinoSliverNavigationBarOpts{LargeTitle: c.NewText("Controls")}),
			c.NewSliverList(c.NewSliverChildListDelegate(rows)),
		}))
	})
}

// popupsTab is the page of the popups of step 34.6: an alert dialog, an action sheet, a context menu that opens
// when its card is pressed and held, and a menu that grows from a button.
func popupsTab(bc c.BuildContext) c.Widget {
	dismiss := func(bc c.BuildContext) func() { return func() { c.NavigatorOf(bc).Pop() } }
	card := c.NewContainer(c.NewCenter(c.NewText("Press and hold me", c.TextOpts{Style: c.NewTextStyle(c.TextStyleOpts{Color: c.Some(c.NewColor(0xFFFFFFFF))})})),
		c.ContainerOpts{Width: c.Some(220.0), Height: c.Some(90.0), Decoration: c.NewBoxDecoration(c.BoxDecorationOpts{
			Color: c.Some(c.CupertinoColors.SystemOrange.Color), BorderRadius: c.NewBorderRadiusCircular(12)})})
	anchor := func(controller *c.MenuController) c.Widget {
		return c.NewCupertinoMenuAnchor([]c.Widget{
			c.NewCupertinoMenuItem(c.NewText("Share"), c.CupertinoMenuItemOpts{OnPressed: func() {}, Leading: c.NewIcon(c.CupertinoIcons.Share())}),
			c.NewCupertinoMenuItem(c.NewText("Copy link"), c.CupertinoMenuItemOpts{OnPressed: func() {}, Leading: c.NewIcon(c.CupertinoIcons.DocOnDoc()),
				Subtitle: c.NewText("To the clipboard")}),
			c.NewCupertinoMenuDivider(),
			c.NewCupertinoMenuItem(c.NewText("Delete"), c.CupertinoMenuItemOpts{OnPressed: func() {}, IsDestructiveAction: true, Leading: c.NewIcon(c.CupertinoIcons.Delete())}),
		}, c.CupertinoMenuAnchorOpts{Controller: controller, EnableLongPressToOpen: true, Child: c.NewCupertinoButtonFilled(func() {
			if controller.IsOpen() {
				controller.Close()
			} else {
				controller.Open()
			}
		}, c.NewText("Menu"))})
	}
	controller := c.NewMenuController()
	rows := []c.Widget{
		row("Show an alert dialog", func() {
			c.ShowCupertinoDialog(bc, func(bc c.BuildContext) c.Widget {
				return c.NewCupertinoAlertDialog(c.CupertinoAlertDialogOpts{Title: c.NewText("Delete the photo?"), Content: c.NewText("It is removed from all of your devices."),
					Actions: []c.Widget{
						c.NewCupertinoDialogAction(c.NewText("Keep"), c.CupertinoDialogActionOpts{IsDefaultAction: true, OnPressed: dismiss(bc)}),
						c.NewCupertinoDialogAction(c.NewText("Delete"), c.CupertinoDialogActionOpts{IsDestructiveAction: true, OnPressed: dismiss(bc)}),
					}})
			})
		}),
		row("Show an action sheet", func() {
			c.ShowCupertinoModalPopup(bc, func(bc c.BuildContext) c.Widget {
				return c.NewCupertinoActionSheet(c.CupertinoActionSheetOpts{Title: c.NewText("Favorite dessert"), Message: c.NewText("Select the best dessert from the options below."),
					Actions: []c.Widget{
						c.NewCupertinoActionSheetAction(dismiss(bc), c.NewText("Profiteroles"), c.CupertinoActionSheetActionOpts{IsDefaultAction: true}),
						c.NewCupertinoActionSheetAction(dismiss(bc), c.NewText("Cannolis")),
						c.NewCupertinoActionSheetAction(dismiss(bc), c.NewText("Trifle"), c.CupertinoActionSheetActionOpts{IsDestructiveAction: true}),
					},
					CancelButton: c.NewCupertinoActionSheetAction(dismiss(bc), c.NewText("Cancel"))})
			})
		}),
		c.NewPadding(c.NewEdgeInsetsSymmetric(12, 20), c.NewCupertinoContextMenu([]c.Widget{
			c.NewCupertinoContextMenuAction(c.NewText("Copy"), c.CupertinoContextMenuActionOpts{IsDefaultAction: true, TrailingIcon: c.Some(c.CupertinoIcons.DocOnDoc())}),
			c.NewCupertinoContextMenuAction(c.NewText("Share"), c.CupertinoContextMenuActionOpts{TrailingIcon: c.Some(c.CupertinoIcons.Share())}),
			c.NewCupertinoContextMenuAction(c.NewText("Delete"), c.CupertinoContextMenuActionOpts{IsDestructiveAction: true, TrailingIcon: c.Some(c.CupertinoIcons.Delete())}),
		}, card)),
		c.NewPadding(c.NewEdgeInsetsSymmetric(12, 20), c.NewAlign(anchor(controller), c.AlignOpts{Alignment: c.AlignmentCenterLeft})),
	}
	return c.NewCupertinoPageScaffold(c.NewCustomScrollView([]c.Widget{
		c.NewCupertinoSliverNavigationBar(c.CupertinoSliverNavigationBarOpts{LargeTitle: c.NewText("Popups")}),
		c.NewSliverList(c.NewSliverChildListDelegate(rows)),
	}))
}
