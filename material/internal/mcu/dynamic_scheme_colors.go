package mcu

// PrimaryPaletteKeyColor is the ARGB color primaryPaletteKeyColor of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.primaryPaletteKeyColor
func (s *DynamicScheme) PrimaryPaletteKeyColor() uint32 { return s.GetArgb(PrimaryPaletteKeyColor) }

// SecondaryPaletteKeyColor is the ARGB color secondaryPaletteKeyColor of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.secondaryPaletteKeyColor
func (s *DynamicScheme) SecondaryPaletteKeyColor() uint32 { return s.GetArgb(SecondaryPaletteKeyColor) }

// TertiaryPaletteKeyColor is the ARGB color tertiaryPaletteKeyColor of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.tertiaryPaletteKeyColor
func (s *DynamicScheme) TertiaryPaletteKeyColor() uint32 { return s.GetArgb(TertiaryPaletteKeyColor) }

// NeutralPaletteKeyColor is the ARGB color neutralPaletteKeyColor of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.neutralPaletteKeyColor
func (s *DynamicScheme) NeutralPaletteKeyColor() uint32 { return s.GetArgb(NeutralPaletteKeyColor) }

// NeutralVariantPaletteKeyColor is the ARGB color neutralVariantPaletteKeyColor of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.neutralVariantPaletteKeyColor
func (s *DynamicScheme) NeutralVariantPaletteKeyColor() uint32 {
	return s.GetArgb(NeutralVariantPaletteKeyColor)
}

// Background is the ARGB color background of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.background
func (s *DynamicScheme) Background() uint32 { return s.GetArgb(Background) }

// OnBackground is the ARGB color onBackground of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.onBackground
func (s *DynamicScheme) OnBackground() uint32 { return s.GetArgb(OnBackground) }

// Surface is the ARGB color surface of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.surface
func (s *DynamicScheme) Surface() uint32 { return s.GetArgb(Surface) }

// SurfaceDim is the ARGB color surfaceDim of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.surfaceDim
func (s *DynamicScheme) SurfaceDim() uint32 { return s.GetArgb(SurfaceDim) }

// SurfaceBright is the ARGB color surfaceBright of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.surfaceBright
func (s *DynamicScheme) SurfaceBright() uint32 { return s.GetArgb(SurfaceBright) }

// SurfaceContainerLowest is the ARGB color surfaceContainerLowest of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.surfaceContainerLowest
func (s *DynamicScheme) SurfaceContainerLowest() uint32 { return s.GetArgb(SurfaceContainerLowest) }

// SurfaceContainerLow is the ARGB color surfaceContainerLow of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.surfaceContainerLow
func (s *DynamicScheme) SurfaceContainerLow() uint32 { return s.GetArgb(SurfaceContainerLow) }

// SurfaceContainer is the ARGB color surfaceContainer of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.surfaceContainer
func (s *DynamicScheme) SurfaceContainer() uint32 { return s.GetArgb(SurfaceContainer) }

// SurfaceContainerHigh is the ARGB color surfaceContainerHigh of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.surfaceContainerHigh
func (s *DynamicScheme) SurfaceContainerHigh() uint32 { return s.GetArgb(SurfaceContainerHigh) }

// SurfaceContainerHighest is the ARGB color surfaceContainerHighest of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.surfaceContainerHighest
func (s *DynamicScheme) SurfaceContainerHighest() uint32 { return s.GetArgb(SurfaceContainerHighest) }

// OnSurface is the ARGB color onSurface of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.onSurface
func (s *DynamicScheme) OnSurface() uint32 { return s.GetArgb(OnSurface) }

// SurfaceVariant is the ARGB color surfaceVariant of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.surfaceVariant
func (s *DynamicScheme) SurfaceVariant() uint32 { return s.GetArgb(SurfaceVariant) }

// OnSurfaceVariant is the ARGB color onSurfaceVariant of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.onSurfaceVariant
func (s *DynamicScheme) OnSurfaceVariant() uint32 { return s.GetArgb(OnSurfaceVariant) }

// InverseSurface is the ARGB color inverseSurface of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.inverseSurface
func (s *DynamicScheme) InverseSurface() uint32 { return s.GetArgb(InverseSurface) }

// InverseOnSurface is the ARGB color inverseOnSurface of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.inverseOnSurface
func (s *DynamicScheme) InverseOnSurface() uint32 { return s.GetArgb(InverseOnSurface) }

// Outline is the ARGB color outline of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.outline
func (s *DynamicScheme) Outline() uint32 { return s.GetArgb(Outline) }

// OutlineVariant is the ARGB color outlineVariant of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.outlineVariant
func (s *DynamicScheme) OutlineVariant() uint32 { return s.GetArgb(OutlineVariant) }

// Shadow is the ARGB color shadow of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.shadow
func (s *DynamicScheme) Shadow() uint32 { return s.GetArgb(Shadow) }

// Scrim is the ARGB color scrim of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.scrim
func (s *DynamicScheme) Scrim() uint32 { return s.GetArgb(Scrim) }

// SurfaceTint is the ARGB color surfaceTint of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.surfaceTint
func (s *DynamicScheme) SurfaceTint() uint32 { return s.GetArgb(SurfaceTint) }

// Primary is the ARGB color primary of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.primary
func (s *DynamicScheme) Primary() uint32 { return s.GetArgb(Primary) }

// OnPrimary is the ARGB color onPrimary of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.onPrimary
func (s *DynamicScheme) OnPrimary() uint32 { return s.GetArgb(OnPrimary) }

// PrimaryContainer is the ARGB color primaryContainer of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.primaryContainer
func (s *DynamicScheme) PrimaryContainer() uint32 { return s.GetArgb(PrimaryContainer) }

// OnPrimaryContainer is the ARGB color onPrimaryContainer of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.onPrimaryContainer
func (s *DynamicScheme) OnPrimaryContainer() uint32 { return s.GetArgb(OnPrimaryContainer) }

// InversePrimary is the ARGB color inversePrimary of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.inversePrimary
func (s *DynamicScheme) InversePrimary() uint32 { return s.GetArgb(InversePrimary) }

// Secondary is the ARGB color secondary of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.secondary
func (s *DynamicScheme) Secondary() uint32 { return s.GetArgb(Secondary) }

// OnSecondary is the ARGB color onSecondary of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.onSecondary
func (s *DynamicScheme) OnSecondary() uint32 { return s.GetArgb(OnSecondary) }

// SecondaryContainer is the ARGB color secondaryContainer of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.secondaryContainer
func (s *DynamicScheme) SecondaryContainer() uint32 { return s.GetArgb(SecondaryContainer) }

// OnSecondaryContainer is the ARGB color onSecondaryContainer of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.onSecondaryContainer
func (s *DynamicScheme) OnSecondaryContainer() uint32 { return s.GetArgb(OnSecondaryContainer) }

// Tertiary is the ARGB color tertiary of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.tertiary
func (s *DynamicScheme) Tertiary() uint32 { return s.GetArgb(Tertiary) }

// OnTertiary is the ARGB color onTertiary of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.onTertiary
func (s *DynamicScheme) OnTertiary() uint32 { return s.GetArgb(OnTertiary) }

// TertiaryContainer is the ARGB color tertiaryContainer of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.tertiaryContainer
func (s *DynamicScheme) TertiaryContainer() uint32 { return s.GetArgb(TertiaryContainer) }

// OnTertiaryContainer is the ARGB color onTertiaryContainer of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.onTertiaryContainer
func (s *DynamicScheme) OnTertiaryContainer() uint32 { return s.GetArgb(OnTertiaryContainer) }

// Error is the ARGB color error of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.error
func (s *DynamicScheme) Error() uint32 { return s.GetArgb(Error) }

// OnError is the ARGB color onError of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.onError
func (s *DynamicScheme) OnError() uint32 { return s.GetArgb(OnError) }

// ErrorContainer is the ARGB color errorContainer of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.errorContainer
func (s *DynamicScheme) ErrorContainer() uint32 { return s.GetArgb(ErrorContainer) }

// OnErrorContainer is the ARGB color onErrorContainer of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.onErrorContainer
func (s *DynamicScheme) OnErrorContainer() uint32 { return s.GetArgb(OnErrorContainer) }

// PrimaryFixed is the ARGB color primaryFixed of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.primaryFixed
func (s *DynamicScheme) PrimaryFixed() uint32 { return s.GetArgb(PrimaryFixed) }

// PrimaryFixedDim is the ARGB color primaryFixedDim of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.primaryFixedDim
func (s *DynamicScheme) PrimaryFixedDim() uint32 { return s.GetArgb(PrimaryFixedDim) }

// OnPrimaryFixed is the ARGB color onPrimaryFixed of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.onPrimaryFixed
func (s *DynamicScheme) OnPrimaryFixed() uint32 { return s.GetArgb(OnPrimaryFixed) }

// OnPrimaryFixedVariant is the ARGB color onPrimaryFixedVariant of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.onPrimaryFixedVariant
func (s *DynamicScheme) OnPrimaryFixedVariant() uint32 { return s.GetArgb(OnPrimaryFixedVariant) }

// SecondaryFixed is the ARGB color secondaryFixed of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.secondaryFixed
func (s *DynamicScheme) SecondaryFixed() uint32 { return s.GetArgb(SecondaryFixed) }

// SecondaryFixedDim is the ARGB color secondaryFixedDim of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.secondaryFixedDim
func (s *DynamicScheme) SecondaryFixedDim() uint32 { return s.GetArgb(SecondaryFixedDim) }

// OnSecondaryFixed is the ARGB color onSecondaryFixed of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.onSecondaryFixed
func (s *DynamicScheme) OnSecondaryFixed() uint32 { return s.GetArgb(OnSecondaryFixed) }

// OnSecondaryFixedVariant is the ARGB color onSecondaryFixedVariant of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.onSecondaryFixedVariant
func (s *DynamicScheme) OnSecondaryFixedVariant() uint32 { return s.GetArgb(OnSecondaryFixedVariant) }

// TertiaryFixed is the ARGB color tertiaryFixed of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.tertiaryFixed
func (s *DynamicScheme) TertiaryFixed() uint32 { return s.GetArgb(TertiaryFixed) }

// TertiaryFixedDim is the ARGB color tertiaryFixedDim of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.tertiaryFixedDim
func (s *DynamicScheme) TertiaryFixedDim() uint32 { return s.GetArgb(TertiaryFixedDim) }

// OnTertiaryFixed is the ARGB color onTertiaryFixed of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.onTertiaryFixed
func (s *DynamicScheme) OnTertiaryFixed() uint32 { return s.GetArgb(OnTertiaryFixed) }

// OnTertiaryFixedVariant is the ARGB color onTertiaryFixedVariant of the scheme.
//
// Flutter: material-color-utilities dynamiccolor/dynamic_scheme.dart DynamicScheme.onTertiaryFixedVariant
func (s *DynamicScheme) OnTertiaryFixedVariant() uint32 { return s.GetArgb(OnTertiaryFixedVariant) }
