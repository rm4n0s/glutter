package windows

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// TestKeyTablesSpotChecks checks 40 keys of the generated tables against the values of Flutter's
// flutter_key_map.g.cc.
func TestKeyTablesSpotChecks(t *testing.T) {
	physical := []struct {
		scan uint32
		want uint64
	}{
		{0x01, 0x70029}, {0x02, 0x7001e}, {0x0e, 0x7002a}, {0x0f, 0x7002b}, {0x10, 0x70014}, {0x1c, 0x70028},
		{0x1d, 0x700e0}, {0x1e, 0x70004}, {0x2a, 0x700e1}, {0x36, 0x700e5}, {0x38, 0x700e2}, {0x39, 0x7002c},
		{0x3a, 0x70039}, {0x3b, 0x7003a}, {0x47, 0x7005f}, {0x4f, 0x70059}, {0x52, 0x70062}, {0x56, 0x70064},
		{0xe01d, 0x700e4}, {0xe038, 0x700e6}, {0xe047, 0x7004a}, {0xe049, 0x7004b}, {0xe04b, 0x70050},
		{0xe04d, 0x7004f}, {0xe051, 0x7004e}, {0xe053, 0x7004c}, {0xe05b, 0x700e3}, {0xe045, 0x70053},
	}
	for _, tt := range physical {
		if got, ok := scanCodeToPhysical[tt.scan]; !ok || got != tt.want {
			t.Errorf("scanCodeToPhysical[%#x] = %#x, %v; want %#x", tt.scan, got, ok, tt.want)
		}
	}
	logical := []struct {
		vk   uint32
		want uint64
	}{
		{win32.VKBack, 0x100000008}, {win32.VKTab, 0x100000009}, {win32.VKReturn, 0x10000000d},
		{win32.VKEscape, 0x10000001b}, {win32.VKSpace, 0x20}, {win32.VKLeft, 0x100000302}, {win32.VKDelete, 0x10000007f},
		{win32.VKF4, 0x100000804}, {win32.VKLShift, 0x200000102}, {win32.VKShift, 0x200000102},
		{win32.VKRControl, 0x200000101}, {win32.VKCapital, 0x100000104}, {0xba, 0x3b}, {0xdb, 0x5b},
	}
	for _, tt := range logical {
		if got, ok := virtualKeyToLogical[tt.vk]; !ok || got != tt.want {
			t.Errorf("virtualKeyToLogical[%#x] = %#x, %v; want %#x", tt.vk, got, ok, tt.want)
		}
	}
	if _, ok := virtualKeyToLogical['A']; ok {
		t.Error("the letters are not in the table of virtual keys: their logical key is the character")
	}
	if got := scanCodeToLogical[0xe035]; got != 0x20000022f {
		t.Errorf("scanCodeToLogical[0xe035] = %#x, want the numpad divide", got)
	}
}

// TestKeyTablesAgainstFlutter compares the three maps of the generated file with the ones of
// flutter_key_map.g.cc (skipped where reference/ is missing).
func TestKeyTablesAgainstFlutter(t *testing.T) {
	path := filepath.Join("..", "..", "..", "reference", "flutter", "engine", "src", "flutter", "shell", "platform", "windows", "flutter_key_map.g.cc")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("reference/ is missing, so Flutter's map cannot be read: run scripts/fetch-reference.sh (%v)", err)
	}
	parts := regexp.MustCompile(`KeyboardKeyEmbedderHandler::(\w+) =`).Split(string(raw), -1)
	names := regexp.MustCompile(`KeyboardKeyEmbedderHandler::(\w+) =`).FindAllStringSubmatch(string(raw), -1)
	entry := regexp.MustCompile(`\{(0x[0-9a-f]+), (0x[0-9a-f]+)\}`)
	maps := map[string]map[uint32]uint64{}
	for i, n := range names {
		m := map[uint32]uint64{}
		for _, e := range entry.FindAllStringSubmatch(parts[i+1], -1) {
			k, err1 := strconv.ParseUint(e[1], 0, 32)
			v, err2 := strconv.ParseUint(e[2], 0, 64)
			if err1 != nil || err2 != nil {
				t.Fatalf("bad entry %v", e)
			}
			m[uint32(k)] = v
		}
		maps[n[1]] = m
	}
	for name, ours := range map[string]map[uint32]uint64{
		"windowsToPhysicalMap_": scanCodeToPhysical,
		"windowsToLogicalMap_":  virtualKeyToLogical,
		"scanCodeToLogicalMap_": scanCodeToLogical,
	} {
		theirs := maps[name]
		if len(theirs) == 0 {
			t.Errorf("%s was not found in flutter_key_map.g.cc", name)
			continue
		}
		if len(theirs) != len(ours) {
			t.Errorf("%s has %d entries, ours %d", name, len(theirs), len(ours))
		}
		for k, v := range theirs {
			if ours[k] != v {
				t.Errorf("%s[%#x] = %#x in Flutter, %#x here", name, k, v, ours[k])
			}
		}
	}
}

// TestLayoutGoalsTable checks the goals: every key of the goals has a scan code and a Latin
// logical key (a letter or a digit is mandatory), and the key of Z is the goal of its scan code.
func TestLayoutGoalsTable(t *testing.T) {
	if len(layoutGoals) != 49 {
		t.Errorf("%d layout goals, want the 49 of layout_goals.json", len(layoutGoals))
	}
	seen := map[uint32]bool{}
	for _, g := range layoutGoals {
		if seen[g.scanCode] {
			t.Errorf("scan code %#x is twice in the goals", g.scanCode)
		}
		seen[g.scanCode] = true
		if _, ok := scanCodeToPhysical[g.scanCode]; !ok {
			t.Errorf("the goal %#x has no physical key", g.scanCode)
		}
		letterOrDigit := g.logical >= 'a' && g.logical <= 'z' || g.logical >= '0' && g.logical <= '9'
		if letterOrDigit != g.mandatory {
			t.Errorf("goal %#x: logical %#x, mandatory %v: the letters and digits are the mandatory goals", g.scanCode, g.logical, g.mandatory)
		}
	}
	for _, g := range layoutGoals {
		if g.scanCode == 0x2c && g.logical != 'z' {
			t.Errorf("the goal of scan code 0x2c is %#x, want z", g.logical)
		}
	}
}

func TestVirtualKeyNames(t *testing.T) {
	for name, want := range map[string]uint32{"RETURN": 0x0d, "BACK": 0x08, "LEFT": 0x25, "LSHIFT": 0xa0, "SHIFT": 0x10, "F10": 0x79, "OEM_1": 0xba, "PRIOR": 0x21} {
		if got := virtualKeyNames[name]; got != want {
			t.Errorf("virtualKeyNames[%q] = %#x, want %#x", name, got, want)
		}
	}
}
