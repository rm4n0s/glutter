package material

import (
	"testing"

	"github.com/rm4n0s/errors"
)

func TestFormatHourUnsupportedFormat(t *testing.T) {
	defer func() {
		r := recover()
		err, ok := r.(error)
		if !ok {
			t.Fatalf("expected a panic with an error, got %v", r)
		}
		appErr, ok := errors.FromError(err)
		if !ok || !appErr.HasRoute("formatHourInFormat.MaterialLocalizationsTimeFormat") {
			t.Errorf("unexpected failure: %v", err)
		}
	}()
	DefaultMaterialLocalizations{}.formatHourInFormat(TimeOfDay{1, 2}, TimeOfDayFormatFrenchCanadian)
}
