package engine

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
)

func TestCheckAndroidVulkanSDK(t *testing.T) {
	tests := []struct {
		name    string
		sdk     int
		named   bool
		wantErr bool
	}{
		{"Android 8.0 under auto", 26, false, true},
		{"Android 9 under auto", 28, false, true},
		{"Android 10 under auto", 29, false, false},
		{"Android 17 under auto", 37, false, false},
		{"Android 8.0 asked for by name", 26, true, false},
		{"an unknown level under auto", 0, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkAndroidVulkanSDK(tt.sdk, tt.named)
			if (err != nil) != tt.wantErr {
				t.Fatalf("checkAndroidVulkanSDK(%d, %v) = %v, want error %v", tt.sdk, tt.named, err, tt.wantErr)
			}
			if err == nil {
				return
			}
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T", err)
			}
			if !appErr.HasRoute("checkAndroidVulkanSDK.AndroidVulkanTooOld") {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
			if appErr.Metadata["sdk"] != tt.sdk || appErr.Metadata["minimum"] != 29 {
				t.Errorf("metadata: %v", appErr.Metadata)
			}
		})
	}
}

// The list of Android is vulkan, gles, soft (SPEC §24.4). `auto` on a device below API 29 logs why
// Vulkan was not used and draws with gles; Vulkan asked for by name does not fall back.
func TestAutoOnAnOldAndroidUsesGLESAndSaysWhy(t *testing.T) {
	handle := gpu.SurfaceHandle{Kind: gpu.SurfaceKindFake, Window: 1}
	tooOld := checkAndroidVulkanSDK(26, false)
	candidates := func() ([]rendererCandidate, *[]*namedRenderer) {
		cands, made := fakeCandidates()
		cands[0].New = func(*Config, *slog.Logger) renderer {
			r := &namedRenderer{fakeRenderer: &fakeRenderer{err: tooOld}, name: "a: device of vulkan"}
			*made = append(*made, r)
			return r
		}
		return cands, made
	}
	t.Run("auto", func(t *testing.T) {
		cands, made := candidates()
		log, buf := recordingLog()
		c, err := newChooser(&Config{}, log, "", cands)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.Draw(handle, 10, 10, nil, false); err != nil {
			t.Fatal(err)
		}
		if got := c.Describe(); got != "b: device of gles" {
			t.Errorf("Describe = %q, want gles", got)
		}
		if len(*made) != 2 {
			t.Errorf("%d renderers were made, want vulkan and gles", len(*made))
		}
		for _, want := range []string{"renderer not usable", "renderer=vulkan", "tag=AndroidVulkanTooOld", "renderer chosen", "device of gles"} {
			if !strings.Contains(buf.String(), want) {
				t.Errorf("the log lacks %q:\n%s", want, buf)
			}
		}
	})
	t.Run("by name", func(t *testing.T) {
		cands, _ := candidates()
		log, _ := recordingLog()
		c, err := newChooser(&Config{}, log, "vulkan", cands)
		if err != nil {
			t.Fatal(err)
		}
		_, err = c.Draw(handle, 10, 10, nil, false)
		if appErr, ok := errors.FromError(err); !ok || appErr.Tag != tagAndroidVulkanTooOld {
			t.Errorf("Draw = %v, want AndroidVulkanTooOld", err)
		}
	})
}
