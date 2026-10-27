package gluttertest

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/rm4n0s/errors"
)

func TestOptsAssets(t *testing.T) {
	tests := []struct {
		name    string
		assets  fstest.MapFS
		key     string
		want    string
		wantErr bool
	}{
		{"an asset of the file system", fstest.MapFS{"icons/a.svg": {Data: []byte("<svg/>")}}, "icons/a.svg", "<svg/>", false},
		{"a missing asset", fstest.MapFS{"icons/a.svg": {Data: []byte("<svg/>")}}, "icons/b.svg", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := New(t, Opts{Assets: tt.assets})
			data, err := ts.Binding.Painting().AssetBundle().Load(context.Background(), tt.key)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, want error: %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if appErr, ok := errors.FromError(err); !ok || appErr.Tag != "AssetNotFound" {
					t.Errorf("err = %v, want the tag AssetNotFound", err)
				}
			}
			if string(data) != tt.want {
				t.Errorf("data = %q, want %q", data, tt.want)
			}
		})
	}
	t.Run("no file system, no bundle", func(t *testing.T) {
		ts := New(t)
		if b := ts.Binding.Painting().AssetBundle(); b != nil {
			t.Errorf("bundle = %v, want none", b)
		}
	})
}
