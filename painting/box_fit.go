package painting

import (
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// BoxFit is how a box should be inscribed into another box. The zero value is
// BoxFitUnspecified: the default of the widget.
//
// Flutter: painting/box_fit.dart BoxFit
type BoxFit int

// The values of BoxFit follow Flutter's declaration order.
const (
	// BoxFitUnspecified is the zero value: the default of the widget.
	BoxFitUnspecified BoxFit = iota
	// BoxFitFill fills the target box by distorting the source's aspect ratio.
	//
	// Flutter: painting/box_fit.dart BoxFit.fill
	BoxFitFill
	// BoxFitContain is as large as possible while still containing the source
	// entirely within the target box.
	//
	// Flutter: painting/box_fit.dart BoxFit.contain
	BoxFitContain
	// BoxFitCover is as small as possible while still covering the entire
	// target box.
	//
	// Flutter: painting/box_fit.dart BoxFit.cover
	BoxFitCover
	// BoxFitFitWidth makes sure the full width of the source is shown,
	// regardless of whether this means the source overflows the target box
	// vertically.
	//
	// Flutter: painting/box_fit.dart BoxFit.fitWidth
	BoxFitFitWidth
	// BoxFitFitHeight makes sure the full height of the source is shown,
	// regardless of whether this means the source overflows the target box
	// horizontally.
	//
	// Flutter: painting/box_fit.dart BoxFit.fitHeight
	BoxFitFitHeight
	// BoxFitNone aligns the source within the target box (by default, centering)
	// and discards any part of the source that is outside the box.
	//
	// Flutter: painting/box_fit.dart BoxFit.none
	BoxFitNone
	// BoxFitScaleDown aligns the source within the target box and, if the
	// source is larger, scales it down to contain it, like BoxFitContain.
	//
	// Flutter: painting/box_fit.dart BoxFit.scaleDown
	BoxFitScaleDown
)

var boxFitNames = [...]string{"unspecified", "fill", "contain", "cover", "fitWidth", "fitHeight", "none", "scaleDown"}

// Name returns the Dart enum name, such as "scaleDown".
//
// Flutter: painting/box_fit.dart BoxFit.name
func (f BoxFit) Name() string {
	if f < 0 || int(f) >= len(boxFitNames) {
		return "unspecified"
	}
	return boxFitNames[f]
}

// String returns the name Dart prints, such as "BoxFit.contain".
//
// Flutter: painting/box_fit.dart BoxFit.toString
func (f BoxFit) String() string { return "BoxFit." + f.Name() }

// FittedSizes is the pair of sizes that applyBoxFit computes.
//
// Flutter: painting/box_fit.dart FittedSizes
type FittedSizes struct {
	// Source is the size of the part of the input to show.
	Source ui.Size
	// Destination is the size of the part of the output to show it on.
	Destination ui.Size
}

// ApplyBoxFit apportions the input size and output size by the fit: the part of
// the input to show and the part of the output to draw it on. An unspecified fit
// is treated as BoxFitFill.
//
// Flutter: painting/box_fit.dart applyBoxFit
func ApplyBoxFit(fit BoxFit, inputSize, outputSize ui.Size) FittedSizes {
	if inputSize.Height <= 0.0 || inputSize.Width <= 0.0 || outputSize.Height <= 0.0 || outputSize.Width <= 0.0 {
		return FittedSizes{}
	}
	var source, destination ui.Size
	switch fit {
	case BoxFitContain:
		source = inputSize
		if outputSize.Width/outputSize.Height > source.Width/source.Height {
			destination = ui.Size{Width: source.Width * outputSize.Height / source.Height, Height: outputSize.Height}
		} else {
			destination = ui.Size{Width: outputSize.Width, Height: source.Height * outputSize.Width / source.Width}
		}
	case BoxFitCover:
		if outputSize.Width/outputSize.Height > inputSize.Width/inputSize.Height {
			source = ui.Size{Width: inputSize.Width, Height: inputSize.Width * outputSize.Height / outputSize.Width}
		} else {
			source = ui.Size{Width: inputSize.Height * outputSize.Width / outputSize.Height, Height: inputSize.Height}
		}
		destination = outputSize
	case BoxFitFitWidth:
		if outputSize.Width/outputSize.Height > inputSize.Width/inputSize.Height {
			source = ui.Size{Width: inputSize.Width, Height: inputSize.Width * outputSize.Height / outputSize.Width}
			destination = outputSize
		} else {
			source = inputSize
			destination = ui.Size{Width: outputSize.Width, Height: source.Height * outputSize.Width / source.Width}
		}
	case BoxFitFitHeight:
		if outputSize.Width/outputSize.Height > inputSize.Width/inputSize.Height {
			source = inputSize
			destination = ui.Size{Width: source.Width * outputSize.Height / source.Height, Height: outputSize.Height}
		} else {
			source = ui.Size{Width: inputSize.Height * outputSize.Width / outputSize.Height, Height: inputSize.Height}
			destination = outputSize
		}
	case BoxFitNone:
		source = ui.Size{Width: math.Min(inputSize.Width, outputSize.Width), Height: math.Min(inputSize.Height, outputSize.Height)}
		destination = source
	case BoxFitScaleDown:
		source = inputSize
		destination = inputSize
		aspectRatio := inputSize.Width / inputSize.Height
		if destination.Height > outputSize.Height {
			destination = ui.Size{Width: outputSize.Height * aspectRatio, Height: outputSize.Height}
		}
		if destination.Width > outputSize.Width {
			destination = ui.Size{Width: outputSize.Width, Height: outputSize.Width / aspectRatio}
		}
	default: // BoxFitFill
		source = inputSize
		destination = outputSize
	}
	return FittedSizes{source, destination}
}
