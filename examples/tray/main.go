// Command tray is a small note-taking window that lives in the system tray. The
// window has a field for a note and a status line; the tray icon has a menu with
// "Show window", "New notification", "Open file…", a separator and "Quit". Closing
// the window does not end the app: the window is minimized and the app stays in the
// tray (a click on the icon shows it again). "New notification" shows a desktop
// notification with an "Open" action that brings the window back, and "Open file…"
// opens the file dialog of the desktop (desktop.OpenFiles) and shows the chosen path.
//
// The tray uses the StatusNotifierItem protocol: KDE Plasma, XFCE and Cinnamon show
// it, and GNOME shows it with the AppIndicator extension. Without it the app still
// works and reports TrayUnavailable once. It uses only the public packages.
//
// Run it with: go run ./examples/tray
//
// Environment: GLUTTER_BACKEND=x11|wayland, GLUTTER_SCREENSHOT=<file.png>.
package main

import (
	"context"
	"image"
	"image/color"
	"image/draw"
	"log"
	"strings"

	"github.com/rm4n0s/glutter/desktop"
	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/ui"
)

func main() {
	err := m.RunApp(NewTrayApp(), m.RunAppOpts{
		AppID:  "dev.glutter.tray",
		Window: m.WindowConfig{Title: "Tray", Size: m.Size{Width: 480, Height: 360}},
	})
	if err != nil {
		log.Fatal(err)
	}
}

// TrayApp is the Material app around the page.
type TrayApp struct{ m.StatelessWidget }

// NewTrayApp creates the app.
func NewTrayApp() *TrayApp { return &TrayApp{} }

func (a *TrayApp) Build(m.BuildContext) m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "Tray",
		Theme:                      m.NewThemeData(m.ThemeDataOpts{ColorSchemeSeed: m.Some(m.Colors.Teal.Primary)}),
		DebugShowCheckedModeBanner: m.Some(false),
		Home:                       NewTrayPage(),
	})
}

// TrayPage is the window: a note field and a status line, with the tray icon.
type TrayPage struct{ m.StatefulWidget }

// NewTrayPage creates the page.
func NewTrayPage(opts ...m.WidgetOpts) *TrayPage {
	return &TrayPage{StatefulWidget: m.NewStatefulWidgetBase(opts...)}
}

func (p *TrayPage) CreateState() m.State { return &trayPageState{} }

type trayPageState struct {
	m.StateOf[*TrayPage]
	binding   *m.WidgetsBinding
	note      *m.TextEditingController
	lifecycle *m.AppLifecycleListener
	icon      ui.WindowIcon
	status    string
	// quitting is true once the user chose "Quit": closing the window then ends the app.
	quitting bool
}

func (s *trayPageState) InitState() {
	s.binding = m.WidgetsBindingOf(s.Context())
	s.note = m.NewTextEditingController()
	s.status = "Close the window to hide it in the tray."
	icon, err := drawIcon()
	if err != nil {
		s.status = "The tray icon failed: " + err.Error()
	}
	s.icon = icon
	s.lifecycle = m.NewAppLifecycleListener(s.binding, m.AppLifecycleListenerOpts{OnExitRequested: s.onExitRequested})
}

func (s *trayPageState) Dispose() {
	s.lifecycle.Dispose()
	s.note.Dispose()
}

// onExitRequested is the close button of the window: the window is hidden, and the
// app goes on, unless the user chose "Quit".
func (s *trayPageState) onExitRequested() ui.AppExitResponse {
	if s.quitting {
		return ui.AppExitResponseExit
	}
	s.hideWindow()
	return ui.AppExitResponseCancel
}

// window is the controller of the window of the app.
func (s *trayPageState) window() *m.RegularWindowController {
	c, _ := s.binding.WindowingOwner().ImplicitWindowController()
	return c
}

// hideWindow minimizes the window: there is no hidden window in the platform layer
// yet, so the window goes to the task bar instead of away from it.
func (s *trayPageState) hideWindow() {
	if c := s.window(); c != nil {
		c.SetMinimized(true)
	}
}

func (s *trayPageState) showWindow() {
	if c := s.window(); c != nil {
		c.SetMinimized(false)
	}
}

func (s *trayPageState) quit() {
	s.quitting = true
	s.binding.ExitApplication(ui.AppExitTypeRequired)
}

func (s *trayPageState) setStatus(text string) {
	s.SetState(func() { s.status = text })
}

// newNotification shows a notification whose "Open" action, and a click on the
// notification itself, bring the window back.
func (s *trayPageState) newNotification() {
	bc := s.Context()
	note := strings.TrimSpace(s.note.Text())
	if note == "" {
		note = "Your note is empty."
	}
	s.Go(func(ctx context.Context) {
		_, err := desktop.ShowNotification(ctx, bc, "Glutter tray", desktop.NotificationOpts{
			Body:    note,
			Icon:    "dialog-information",
			Actions: []desktop.NotificationAction{{ID: "default", Label: "Open"}, {ID: "open", Label: "Open"}},
			OnAction: func(id string) {
				if id == "open" || id == "default" {
					s.showWindow()
				}
			},
		})
		if err != nil {
			s.setStatus("Notification failed: " + err.Error())
			return
		}
		s.setStatus("Notification shown.")
	})
}

// openFile asks the desktop for a file and shows its path.
func (s *trayPageState) openFile() {
	bc := s.Context()
	s.Go(func(ctx context.Context) {
		res, err := desktop.OpenFiles(ctx, bc, desktop.OpenFilesOpts{Title: "Open a file"})
		switch {
		case err != nil:
			s.setStatus("Open failed: " + err.Error())
		case res.Cancelled || len(res.Paths) == 0:
			s.setStatus("Open cancelled.")
		default:
			s.setStatus("Opened: " + res.Paths[0])
		}
		// The window shows the path, so it comes forward.
		s.binding.Post(s.showWindow)
	})
}

func (s *trayPageState) Build(bc m.BuildContext) m.Widget {
	body := m.NewPadding(m.NewEdgeInsetsAll(16), m.NewColumn([]m.Widget{
		m.NewTextField(m.TextFieldOpts{
			Controller: s.note,
			Decoration: &m.InputDecoration{LabelText: m.Some("Note")},
			MaxLines:   m.Some(4),
		}),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(16.0)}),
		m.NewText(s.status, m.TextOpts{Style: m.ThemeOf(bc).TextTheme.BodyMedium}),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStretch}))
	return desktop.NewSystemTray(s.icon, m.NewScaffold(m.ScaffoldOpts{
		AppBar: m.NewAppBar(m.AppBarOpts{Title: m.NewText("Tray")}),
		Body:   body,
	}), desktop.SystemTrayOpts{
		Title:      "Glutter tray",
		Tooltip:    "Glutter tray example",
		OnActivate: s.showWindow,
		Menu: []desktop.TrayMenuEntry{
			desktop.NewTrayMenuItem("Show window", s.showWindow),
			desktop.NewTrayMenuItem("New notification", s.newNotification),
			desktop.NewTrayMenuItem("Open file…", s.openFile),
			desktop.NewTrayMenuSeparator(),
			desktop.NewTrayMenuItem("Quit", s.quit),
		},
	})
}

// drawIcon draws the icon of the tray: a teal disc with a white note on it, 64 pixels
// wide, which the tray scales to its sizes.
func drawIcon() (ui.WindowIcon, error) {
	const size = 64
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	teal := color.NRGBA{R: 0x00, G: 0x89, B: 0x7B, A: 0xFF}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx, dy := float64(x)+0.5-size/2, float64(y)+0.5-size/2
			if dx*dx+dy*dy <= (size/2-2)*(size/2-2) {
				img.SetNRGBA(x, y, teal)
			}
		}
	}
	draw.Draw(img, image.Rect(20, 16, 44, 48), image.NewUniform(color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}), image.Point{}, draw.Src)
	for line := 0; line < 3; line++ {
		y := 22 + line*8
		draw.Draw(img, image.Rect(24, y, 40, y+2), image.NewUniform(teal), image.Point{}, draw.Src)
	}
	return desktop.TrayIconFromRGBA(size, img.Pix)
}
