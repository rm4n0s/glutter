//go:build glutter_release

// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package foundation

// DebugMode is true in a debug build, the default. Debug-only checks (Flutter's
// asserts, SPEC §12.9) and diagnostics run only then.
//
// Flutter: foundation/constants.dart kDebugMode
const DebugMode = false

// ProfileMode is true in a build made with the tag glutter_profile.
//
// Flutter: foundation/constants.dart kProfileMode
const ProfileMode = false

// ReleaseMode is true in a build made with the tag glutter_release. It wins
// when both tags are given.
//
// Flutter: foundation/constants.dart kReleaseMode
const ReleaseMode = true
