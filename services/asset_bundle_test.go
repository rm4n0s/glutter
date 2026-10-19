package services

import (
	"context"
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/rm4n0s/errors"
)

func TestFSAssetBundleLoad(t *testing.T) {
	b := NewFSAssetBundle(fstest.MapFS{"images/a.png": {Data: []byte("png")}, "root.txt": {Data: []byte("r")}})
	tests := []struct {
		key       string
		want      string
		wantRoute string
	}{
		{"images/a.png", "png", ""},
		{"root.txt", "r", ""},
		{"images/missing.png", "", "FSAssetBundle.Load.AssetNotFound"},
		{"../escape", "", "FSAssetBundle.Load.AssetNotFound"},
		{"/abs.png", "", "FSAssetBundle.Load.AssetNotFound"},
		{"images", "", "FSAssetBundle.Load.AssetReadFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, err := b.Load(context.Background(), tt.key)
			if tt.wantRoute == "" {
				if err != nil || string(got) != tt.want {
					t.Errorf("Load = %q, %v", got, err)
				}
				return
			}
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T", err)
			}
			if !appErr.HasRoute(tt.wantRoute) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := b.Load(ctx, "root.txt")
	appErr, ok := errors.FromError(err)
	if !ok || !appErr.HasRoute("FSAssetBundle.Load.AssetReadCanceled") {
		t.Errorf("cancelled load: %v", err)
	}
}

func TestFSAssetBundleVariants(t *testing.T) {
	b := NewFSAssetBundle(fstest.MapFS{
		"images/a.png":        {},
		"images/2.0x/a.png":   {},
		"images/3x/a.png":     {},
		"images/1.5x/a.png":   {},
		"images/2.0x/b.png":   {},
		"images/big/a.png":    {},
		"images/0x/a.png":     {},
		"a.png":               {},
		"2.0x/a.png":          {},
		"images/x/2.0x/z.png": {},
	})
	tests := []struct {
		key  string
		want []AssetVariant
	}{
		{"images/a.png", []AssetVariant{{"images/1.5x/a.png", 1.5}, {"images/2.0x/a.png", 2}, {"images/3x/a.png", 3}}},
		{"images/b.png", []AssetVariant{{"images/2.0x/b.png", 2}}},
		{"images/c.png", nil},
		{"a.png", []AssetVariant{{"2.0x/a.png", 2}}},
		{"nodir/a.png", nil},
	}
	for _, tt := range tests {
		got, err := b.Variants(context.Background(), tt.key)
		if err != nil || !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Variants(%q) = %v, %v, want %v", tt.key, got, err, tt.want)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := b.Variants(ctx, "a.png"); err == nil {
		t.Error("a cancelled context fails")
	}
}

func TestParseScaleDirectory(t *testing.T) {
	tests := []struct {
		name string
		want float64
		ok   bool
	}{{"2.0x", 2, true}, {"3x", 3, true}, {"1.5x", 1.5, true}, {"x", 0, false}, {"0x", 0, false}, {"2.0", 0, false}, {"ax", 0, false}, {"-1x", 0, false}}
	for _, tt := range tests {
		got, ok := parseScaleDirectory(tt.name)
		if got != tt.want || ok != tt.ok {
			t.Errorf("parseScaleDirectory(%q) = %v, %v", tt.name, got, ok)
		}
	}
}
