//go:build darwin

package objcx

import (
	"reflect"
	"sync"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
)

// blockTypes are the function types that NewBlock has made a layout for. purego makes one callback for
// each function type, not for each block, so only a new type costs one of the callbacks of the process.
var blockTypes sync.Map // reflect.Type -> struct{}

// NewBlock makes a block of the function fn, whose first argument is the block itself (objc.Block) and
// then the arguments of the block; its result, if any, is the block's return value. It returns +1
// reference: release it with Block.Release when nothing uses it, or copy it with Block.Copy when
// the callee keeps it. Blocks of the same function type share one callback, so a block that an app
// makes for each call does not wear out the limit (SPEC §29.2); the first block of a type counts one
// callback in package dl.
//
// Flutter: no counterpart
func NewBlock(fn any) objc.Block {
	ty := reflect.TypeOf(fn)
	if _, seen := blockTypes.LoadOrStore(ty, struct{}{}); !seen {
		dl.ReserveCallbacks(1)
	}
	return objc.NewBlock(fn)
}
