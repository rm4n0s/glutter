package services

import (
	"context"
	"io"
	"testing"
	"testing/fstest"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

// fontSink is a text engine that records the fonts it is given, or fails on one.
type fontSink struct {
	got    []string
	failOn string
}

func (*fontSink) BuildParagraph(*ui.ParagraphContent) ui.ParagraphLayout { return nil }

func (s *fontSink) LoadFontFromList(list []byte, family string) error {
	if string(list) == s.failOn {
		return errors.New(tagFontDataInvalid, "not a font", "family", family)
	}
	s.got = append(s.got, family+":"+string(list))
	return nil
}

func sinkDispatcher() (*ui.PlatformDispatcher, *fontSink) {
	d := ui.NewPlatformDispatcher(nil)
	s := &fontSink{}
	d.SetTextEngine(s)
	return d, s
}

func TestFontLoaderLoad(t *testing.T) {
	bundle := NewFSAssetBundle(fstest.MapFS{"a.ttf": {Data: []byte("A")}, "b.ttf": {Data: []byte("B")}})
	tests := []struct {
		name  string
		build func(l *FontLoader)
		want  []string
		// tag is the tag of the failure, empty when Load succeeds.
		tag string
	}{
		{"no fonts", func(*FontLoader) {}, nil, ""},
		{"bytes, a function and assets, in the order they were added", func(l *FontLoader) {
			l.AddFontBytes([]byte("1"))
			l.AddFont(func(context.Context) ([]byte, error) { return []byte("2"), nil })
			l.AddFontAsset(bundle, "a.ttf")
			l.AddFontAsset(bundle, "b.ttf")
		}, []string{"Mono:1", "Mono:2", "Mono:A", "Mono:B"}, ""},
		{"a missing asset stops the loading", func(l *FontLoader) {
			l.AddFontBytes([]byte("1"))
			l.AddFontAsset(bundle, "none.ttf")
			l.AddFontBytes([]byte("3"))
		}, []string{"Mono:1"}, "AssetNotFound"},
		{"a function that fails with an untagged error", func(l *FontLoader) {
			l.AddFont(func(context.Context) ([]byte, error) { return nil, io.ErrClosedPipe })
		}, nil, "FontLoadFailed"},
		{"the engine refuses a font", func(l *FontLoader) {
			l.AddFontBytes([]byte("1"))
			l.AddFontBytes([]byte("bad"))
			l.AddFontBytes([]byte("3"))
		}, []string{"Mono:1"}, "FontDataInvalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, sink := sinkDispatcher()
			sink.failOn = "bad"
			l := NewFontLoader("Mono")
			tt.build(l)
			err := l.Load(context.Background(), d)
			if tt.tag == "" && err != nil {
				t.Fatalf("Load: %v", err)
			}
			if tt.tag != "" {
				appErr, ok := errors.FromError(err)
				if !ok || appErr.Tag != tt.tag {
					t.Errorf("err = %v, want tag %s", err, tt.tag)
				}
			}
			if len(sink.got) != len(tt.want) {
				t.Fatalf("loaded %v, want %v", sink.got, tt.want)
			}
			for i := range tt.want {
				if sink.got[i] != tt.want[i] {
					t.Errorf("loaded[%d] = %s, want %s", i, sink.got[i], tt.want[i])
				}
			}
		})
	}
}

func TestFontLoaderFamily(t *testing.T) {
	if got := NewFontLoader("Mono").Family(); got != "Mono" {
		t.Errorf("Family = %q", got)
	}
}

func TestFontLoaderCanceled(t *testing.T) {
	d, sink := sinkDispatcher()
	l := NewFontLoader("Mono")
	l.AddFontBytes([]byte("1"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := l.Load(ctx, d)
	appErr, ok := errors.FromError(err)
	if !ok || !appErr.HasRoute("FontLoader.Load.FontLoadCanceled") || len(sink.got) != 0 {
		t.Errorf("err = %v, loaded %v", err, sink.got)
	}
}

func TestFontLoaderReused(t *testing.T) {
	tests := []struct {
		name string
		do   func(l *FontLoader, d *ui.PlatformDispatcher)
	}{
		{"Load twice", func(l *FontLoader, d *ui.PlatformDispatcher) { _ = l.Load(context.Background(), d) }},
		{"AddFontBytes after Load", func(l *FontLoader, _ *ui.PlatformDispatcher) { l.AddFontBytes(nil) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, _ := sinkDispatcher()
			l := NewFontLoader("Mono")
			if err := l.Load(context.Background(), d); err != nil {
				t.Fatal(err)
			}
			defer func() {
				appErr, ok := errors.FromError(asError(recover()))
				if !ok || appErr.Tag != tagFontLoaderReused {
					t.Errorf("panic = %v, want FontLoaderReused", appErr)
				}
			}()
			tt.do(l, d)
			t.Error("no panic")
		})
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagFontDataInvalid = "FontDataInvalid"
)
