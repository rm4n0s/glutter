package windows

import "testing"

// The first two cases are the tests of sequential_id_generator_unittests.cc; the others cover
// ReleaseId and the wrap-around.
func TestSequentialIDGenerator(t *testing.T) {
	type step struct {
		op     string // "get", "release", "releaseID", "has"
		arg    uint32
		want   uint32
		wantOK bool
	}
	tests := []struct {
		name     string
		min, max uint32
		steps    []step
	}{
		{"RemoveMultipleNumbers", 4, 128, []step{
			{"get", 45, 4, false}, {"get", 55, 5, false}, {"get", 15, 6, false},
			{"release", 45, 0, false}, {"has", 45, 0, false},
			{"release", 15, 0, false}, {"has", 15, 0, false},
			{"get", 55, 5, false}, {"get", 12, 4, false},
			{"release", 12, 0, false}, {"release", 55, 0, false},
			{"get", 0, 4, false},
		}},
		{"MaybeRemoveNumbers", 0, 128, []step{
			{"get", 42, 0, false}, {"release", 42, 0, false}, {"has", 42, 0, false}, {"release", 42, 0, false},
		}},
		{"a number keeps its id", 0, 8, []step{
			{"get", 900, 0, false}, {"get", 901, 1, false}, {"get", 900, 0, false}, {"has", 901, 0, true},
		}},
		{"releaseID frees the id", 0, 8, []step{
			{"get", 7, 0, false}, {"get", 8, 1, false}, {"releaseID", 0, 0, false}, {"has", 7, 0, false},
			{"get", 9, 0, false}, {"releaseID", 5, 0, false}, {"has", 8, 0, true},
		}},
		{"when all are taken it starts again at the lowest", 0, 2, []step{
			{"get", 10, 0, false}, {"get", 11, 1, false}, {"get", 12, 0, false},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := newSequentialIDGenerator(tt.min, tt.max)
			for i, s := range tt.steps {
				switch s.op {
				case "get":
					if got := g.generatedID(s.arg); got != s.want {
						t.Fatalf("step %d: generatedID(%d) = %d, want %d", i, s.arg, got, s.want)
					}
				case "release":
					g.releaseNumber(s.arg)
				case "releaseID":
					g.releaseID(s.arg)
				case "has":
					if got := g.hasGeneratedIDFor(s.arg); got != s.wantOK {
						t.Fatalf("step %d: hasGeneratedIDFor(%d) = %v, want %v", i, s.arg, got, s.wantOK)
					}
				}
			}
		})
	}
}
