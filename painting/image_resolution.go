// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package painting

import (
	"sort"

	"github.com/rm4n0s/glutter/services"
)

const (
	lowDprLimit       = 2.0
	naturalResolution = 1.0
)

// AssetImage fetches an image from an AssetBundle, having determined the exact
// image to use based on the context: the bundle has the main asset and, in
// sibling directories named like "2.0x", variants of other resolutions, and the
// one closest to the device pixel ratio of the configuration is used.
//
// Flutter: painting/image_resolution.dart AssetImage
type AssetImage struct {
	// AssetName is the key of the main asset.
	AssetName string
	// Bundle is the bundle to load the asset from; nil uses the configuration's,
	// then the binding's root bundle.
	Bundle services.AssetBundle
	// Package is the package the asset is in; "" for none.
	Package string
}

// AssetImageOpts are the optional parameters of NewAssetImage.
//
// Flutter: painting/image_resolution.dart AssetImage.new
type AssetImageOpts struct {
	// Bundle is the bundle to load the asset from.
	Bundle services.AssetBundle
	// Package is the package the asset is in.
	Package string
}

// NewAssetImage creates a provider for the asset.
//
// Flutter: painting/image_resolution.dart AssetImage.new
func NewAssetImage(assetName string, opts ...AssetImageOpts) *AssetImage {
	o := firstOpts(opts)
	return &AssetImage{AssetName: assetName, Bundle: o.Bundle, Package: o.Package}
}

// KeyName is the name of the asset with the package prefix.
//
// Flutter: painting/image_resolution.dart AssetImage.keyName
func (a *AssetImage) KeyName() string { return keyNameFor(a.AssetName, a.Package) }

// ObtainKey chooses the variant of the asset for the device pixel ratio of the
// configuration.
//
// Flutter: painting/image_resolution.dart AssetImage.obtainKey
func (a *AssetImage) ObtainKey(binding *PaintingBinding, configuration ImageConfiguration) (any, error) {
	bundle, err := chooseBundle(binding, a.Bundle, configuration)
	if err != nil {
		return nil, err
	}
	variants, err := bundle.Variants(binding.Context(), a.KeyName())
	if err != nil {
		return nil, err
	}
	chosen := chooseVariant(a.KeyName(), configuration, variants)
	return AssetBundleImageKey{bundle, chosen.Key, chosen.Scale}, nil
}

func chooseVariant(mainKey string, config ImageConfiguration, variants []services.AssetVariant) services.AssetVariant {
	main := services.AssetVariant{Key: mainKey, Scale: naturalResolution}
	if len(variants) == 0 || config.DevicePixelRatio.IsNone() {
		return main
	}
	byDpr := map[float64]services.AssetVariant{main.Scale: main}
	for _, v := range variants {
		byDpr[v.Scale] = v
	}
	keys := make([]float64, 0, len(byDpr))
	for k := range byDpr {
		keys = append(keys, k)
	}
	sort.Float64s(keys)
	return findBestVariant(keys, byDpr, config.DevicePixelRatio.Value())
}

func findBestVariant(sorted []float64, byDpr map[float64]services.AssetVariant, value float64) services.AssetVariant {
	if v, ok := byDpr[value]; ok {
		return v
	}
	lower, upper := -1.0, -1.0
	hasLower, hasUpper := false, false
	for _, k := range sorted {
		if k < value {
			lower, hasLower = k, true
		} else if k > value && !hasUpper {
			upper, hasUpper = k, true
		}
	}
	if !hasLower {
		return byDpr[upper]
	}
	if !hasUpper {
		return byDpr[lower]
	}
	if value < lowDprLimit || value > (lower+upper)/2 {
		return byDpr[upper]
	}
	return byDpr[lower]
}

// LoadImage loads the asset of the key.
//
// Flutter: painting/image_provider.dart AssetBundleImageProvider.loadImage
func (a *AssetImage) LoadImage(binding *PaintingBinding, key any, decode ImageDecoderCallback) *ImageStreamCompleter {
	return assetBundleLoadImage(binding, a, key.(AssetBundleImageKey), decode)
}

// String describes the provider.
//
// Flutter: painting/image_resolution.dart AssetImage.toString
func (a *AssetImage) String() string {
	b := "null"
	if a.Bundle != nil {
		b = bundleString(a.Bundle)
	}
	return `AssetImage(bundle: ` + b + `, name: "` + a.KeyName() + `")`
}
