package macos

import (
	"bytes"
	"image/png"
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

func TestPlanFilters(t *testing.T) {
	tests := []struct {
		name    string
		filters []platform.FileFilter
		want    filterPlan
		open    []string
		save    []string
	}{
		{"none", nil, filterPlan{}, nil, nil},
		{"images and text", []platform.FileFilter{
			{Name: "Images", Patterns: []string{"*.png", "*.JPG", "*.png"}},
			{Name: "Text", Patterns: []string{"*.txt", "readme.md"}}},
			filterPlan{Groups: []filterGroup{{"Images", []string{"png", "jpg"}}, {"Text", []string{"txt", "md"}}}},
			[]string{"png", "jpg", "txt", "md"}, []string{"png", "jpg"}},
		{"all files", []platform.FileFilter{{Name: "Everything", Patterns: []string{"*"}}, {Name: "PNG", Patterns: []string{"*.png"}}},
			filterPlan{AllFiles: true, Groups: []filterGroup{{"Everything", nil}, {"PNG", []string{"png"}}}}, nil, nil},
		{"star dot star", []platform.FileFilter{{Name: "Any", Patterns: []string{"*.*"}}},
			filterPlan{AllFiles: true, Groups: []filterGroup{{"Any", nil}}}, nil, nil},
		{"a name with no extension and a wildcard in it name no type", []platform.FileFilter{{Name: "Odd", Patterns: []string{"Makefile", "img.*x"}}},
			filterPlan{Groups: []filterGroup{{"Odd", nil}}}, nil, nil},
		{"the last extension of a double one", []platform.FileFilter{{Name: "Archives", Patterns: []string{"*.tar.gz"}}},
			filterPlan{Groups: []filterGroup{{"Archives", []string{"gz"}}}}, []string{"gz"}, []string{"gz"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := planFilters(tc.filters)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("plan = %+v, want %+v", got, tc.want)
			}
			if o := got.extensions(); !reflect.DeepEqual(o, tc.open) {
				t.Errorf("extensions = %v, want %v", o, tc.open)
			}
			if s := got.saveExtensions(); !reflect.DeepEqual(s, tc.save) {
				t.Errorf("saveExtensions = %v, want %v", s, tc.save)
			}
		})
	}
	plan := planFilters([]platform.FileFilter{{Name: "Images", Patterns: []string{"*.png"}}, {Name: "Text", Patterns: []string{"*.txt"}}})
	for file, want := range map[string]string{"/a/b.PNG": "Images", "c.txt": "Text", "d.bin": "", "noext": ""} {
		if got := plan.filterNameFor(file); got != want {
			t.Errorf("filterNameFor(%q) = %q, want %q", file, got, want)
		}
	}
}

func TestPlanTrayMenu(t *testing.T) {
	var pressed []string
	press := func(s string) func() { return func() { pressed = append(pressed, s) } }
	items := []platform.TrayMenuItem{
		{Separator: true}, // first: left out
		{Label: "Open", OnPressed: press("open")},
		{Label: "Hidden", Hidden: true, OnPressed: press("hidden")},
		{Separator: true},
		{Separator: true}, // next to another: left out
		{Label: "Sound", Checkable: true, Checked: true, OnPressed: press("sound")},
		{Label: "Off", Disabled: true, OnPressed: press("off")},
		{Label: "Mode", Children: []platform.TrayMenuItem{
			{Label: "A", Radio: true, Checked: true, OnPressed: press("a")},
			{Label: "B", Radio: true, OnPressed: press("b")},
		}, OnPressed: press("mode")},
		{Separator: true}, // last: left out
	}
	plan, handlers := planTrayMenu(items)
	want := []trayEntry{
		{ID: 1, Label: "Open"},
		{ID: 2, Separator: true},
		{ID: 3, Label: "Sound", Checked: true},
		{ID: 4, Label: "Off", Disabled: true},
		{ID: 5, Label: "Mode", Children: []trayEntry{{ID: 6, Label: "A", Checked: true}, {ID: 7, Label: "B"}}},
	}
	if !reflect.DeepEqual(plan, want) {
		t.Errorf("plan = %+v\nwant   %+v", plan, want)
	}
	for _, id := range []int{1, 3, 6, 7} {
		h, ok := handlers[id]
		if !ok {
			t.Errorf("no handler for %d", id)
			continue
		}
		h()
	}
	for _, id := range []int{2, 4, 5} {
		if _, ok := handlers[id]; ok {
			t.Errorf("an entry that is a separator, disabled or a submenu has a handler: %d", id)
		}
	}
	if !reflect.DeepEqual(pressed, []string{"open", "sound", "a", "b"}) {
		t.Errorf("pressed = %v", pressed)
	}
}

func TestTrayTip(t *testing.T) {
	tests := []struct {
		spec platform.TraySpec
		want string
	}{
		{platform.TraySpec{}, ""},
		{platform.TraySpec{Tooltip: "Title"}, "Title"},
		{platform.TraySpec{Tooltip: "Title", TooltipBody: "Body"}, "Title\nBody"},
		{platform.TraySpec{Title: "App"}, "App"},
		{platform.TraySpec{TooltipBody: "Only body"}, "Only body"},
	}
	for _, tc := range tests {
		if got := trayTip(tc.spec); got != tc.want {
			t.Errorf("trayTip(%+v) = %q, want %q", tc.spec, got, tc.want)
		}
	}
}

func TestTrayIconPNG(t *testing.T) {
	px := func(size int, c byte) ui.WindowIconImage {
		b := make([]byte, size*size*4)
		for i := 0; i < len(b); i += 4 {
			b[i], b[i+1], b[i+2], b[i+3] = c, 0, 0, 255
		}
		return ui.WindowIconImage{Size: size, RGBA: b}
	}
	if data, _, err := trayIconPNG(ui.WindowIcon{}); data != nil || err != nil {
		t.Errorf("no picture: %v, %v", data, err)
	}
	icon := ui.WindowIcon{Images: []ui.WindowIconImage{px(16, 1), px(32, 2), px(64, 3)}}
	icon.Images[1].Scale = 2 // 16 points at twice the pixels: as near to 18 as the 16
	data, points, err := trayIconPNG(icon)
	if err != nil || len(data) == 0 {
		t.Fatalf("trayIconPNG = %d bytes, %v", len(data), err)
	}
	if points != 16 {
		t.Errorf("points = %v, want 16 (the 32-pixel picture at scale 2, which has the most pixels of those nearest)", points)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil || img.Bounds().Dx() != 32 {
		t.Errorf("the picture is %v, %v; want the 32-pixel one", img.Bounds(), err)
	}
	if r, _, _, _ := img.At(0, 0).RGBA(); r>>8 != 2 {
		t.Errorf("the picture is not the second one (red %d)", r>>8)
	}
}
