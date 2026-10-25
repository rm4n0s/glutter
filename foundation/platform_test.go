package foundation

import "testing"

func TestTargetPlatformNames(t *testing.T) {
	tests := []struct {
		p    TargetPlatform
		name string
		str  string
	}{
		{TargetPlatformAndroid, "android", "TargetPlatform.android"},
		{TargetPlatformFuchsia, "fuchsia", "TargetPlatform.fuchsia"},
		{TargetPlatformIOS, "iOS", "TargetPlatform.iOS"},
		{TargetPlatformLinux, "linux", "TargetPlatform.linux"},
		{TargetPlatformMacOS, "macOS", "TargetPlatform.macOS"},
		{TargetPlatformWindows, "windows", "TargetPlatform.windows"},
		{TargetPlatformUnspecified, "unspecified", "TargetPlatform.unspecified"},
		{TargetPlatform(42), "TargetPlatform(42)", "TargetPlatform(42)"},
	}
	for _, tt := range tests {
		if got := tt.p.Name(); got != tt.name {
			t.Errorf("%d: Name = %q, want %q", int(tt.p), got, tt.name)
		}
		if got := tt.p.String(); got != tt.str {
			t.Errorf("%d: String = %q, want %q", int(tt.p), got, tt.str)
		}
	}
}

func TestDefaultTargetPlatform(t *testing.T) {
	if !DebugMode {
		t.Skip("the override works in debug builds only")
	}
	if got := DefaultTargetPlatform(); got != hostWant {
		t.Errorf("default = %v, want %v, the host of this build", got, hostWant)
	}
	if got := DebugDefaultTargetPlatformOverride(); got.IsSome() {
		t.Errorf("override = %v, want none", got)
	}
	tests := []TargetPlatform{TargetPlatformMacOS, TargetPlatformWindows, TargetPlatformAndroid, TargetPlatformLinux}
	for _, p := range tests {
		SetDebugDefaultTargetPlatformOverride(Some(p))
		if got := DefaultTargetPlatform(); got != p {
			t.Errorf("with override %v: default = %v", p, got)
		}
		if got := DebugDefaultTargetPlatformOverride(); got != Some(p) {
			t.Errorf("override = %v, want %v", got, p)
		}
	}
	SetDebugDefaultTargetPlatformOverride(None[TargetPlatform]())
	if got := DebugDefaultTargetPlatformOverride(); got.IsSome() {
		t.Errorf("override not removed: %v", got)
	}
}

func TestBuildModes(t *testing.T) {
	n := 0
	for _, b := range []bool{DebugMode, ProfileMode, ReleaseMode} {
		if b {
			n++
		}
	}
	if n != 1 {
		t.Errorf("exactly one of DebugMode, ProfileMode and ReleaseMode must be true, got %v %v %v", DebugMode, ProfileMode, ReleaseMode)
	}
}

func TestPrecisionErrorTolerance(t *testing.T) {
	if PrecisionErrorTolerance != 1e-10 {
		t.Errorf("PrecisionErrorTolerance = %v", PrecisionErrorTolerance)
	}
}

func TestDebugFormatDouble(t *testing.T) {
	tests := []struct {
		in   Opt[float64]
		want string
	}{
		{None[float64](), "null"},
		{Some(0.0), "0.0"},
		{Some(1.25), "1.3"},
		{Some(1.0 / 3), "0.3"},
		{Some(-2.0), "-2.0"},
		{Some(1e21), "1000000000000000000000.0"},
	}
	for _, tt := range tests {
		if got := DebugFormatDouble(tt.in); got != tt.want {
			t.Errorf("DebugFormatDouble(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
