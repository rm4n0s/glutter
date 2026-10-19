package services

import (
	"context"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/rm4n0s/errors"
)

// AssetBundle is a collection of resources used by the application: images,
// fonts and other files, read by their key (a slash separated path).
//
// Flutter: services/asset_bundle.dart AssetBundle
type AssetBundle interface {
	// Load returns the bytes of the asset. It fails with the tag AssetNotFound if
	// there is no asset with the key.
	//
	// Flutter: services/asset_bundle.dart AssetBundle.load
	Load(ctx context.Context, key string) ([]byte, error)
	// Variants returns the resolution variants of the asset: the files of the
	// same name in the sibling directories named like "2.0x", with the scale each
	// stands for, in increasing order of scale. The asset itself is not in the
	// list. It returns nil when there are none.
	//
	// Flutter: services/asset_manifest.dart AssetManifest.getAssetVariants
	Variants(ctx context.Context, key string) ([]AssetVariant, error)
}

// AssetVariant is a resolution variant of an asset.
//
// Flutter: services/asset_manifest.dart AssetMetadata
type AssetVariant struct {
	// Key is the key of the variant, such as "images/2.0x/logo.png".
	Key string
	// Scale is the device pixel ratio the variant is meant for.
	Scale float64
}

// FSAssetBundle is an AssetBundle over a file system, such as an embed.FS or
// the one in RunAppOpts.Assets.
//
// Flutter: services/asset_bundle.dart PlatformAssetBundle
type FSAssetBundle struct{ fsys fs.FS }

// NewFSAssetBundle creates a bundle that reads the assets from the file system.
//
// Flutter: services/asset_bundle.dart PlatformAssetBundle.new
func NewFSAssetBundle(fsys fs.FS) *FSAssetBundle { return &FSAssetBundle{fsys: fsys} }

// Load reads the asset. It fails with AssetNotFound if the key is not a file of
// the file system, and with AssetReadFailed if it can't be read.
//
// Flutter: services/asset_bundle.dart AssetBundle.load
func (b *FSAssetBundle) Load(ctx context.Context, key string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr("AssetReadCanceled", err).SetMetadata("key", key)
	}
	if !fs.ValidPath(key) {
		return nil, errors.New("AssetNotFound", "Unable to load asset: the key is not a valid path", "key", key)
	}
	data, err := fs.ReadFile(b.fsys, key)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, errors.New("AssetNotFound", "Unable to load asset", "key", key)
		}
		return nil, errors.NewErr("AssetReadFailed", err).SetMetadata("key", key)
	}
	return data, nil
}

// Variants finds the "<scale>x" siblings of the asset.
//
// Flutter: services/asset_manifest.dart AssetManifest.getAssetVariants
func (b *FSAssetBundle) Variants(ctx context.Context, key string) ([]AssetVariant, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.NewErr("AssetReadCanceled", err).SetMetadata("key", key)
	}
	dir, file := path.Split(key)
	dir = strings.TrimSuffix(dir, "/")
	if dir == "" {
		dir = "."
	}
	entries, err := fs.ReadDir(b.fsys, dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, errors.NewErr("AssetReadFailed", err).SetMetadata("key", key)
	}
	var out []AssetVariant
	for _, e := range entries {
		scale, ok := parseScaleDirectory(e.Name())
		if !e.IsDir() || !ok {
			continue
		}
		variant := path.Join(dir, e.Name(), file)
		if info, err := fs.Stat(b.fsys, variant); err == nil && !info.IsDir() {
			out = append(out, AssetVariant{Key: variant, Scale: scale})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Scale < out[j].Scale })
	return out, nil
}

// parseScaleDirectory parses a directory name such as "2.0x" or "3x".
func parseScaleDirectory(name string) (float64, bool) {
	num, ok := strings.CutSuffix(name, "x")
	if !ok || num == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(num, 64)
	if err != nil || v <= 0 {
		return 0, false
	}
	return v, true
}
