//go:build 386 || arm || mips || mipsle

package dec128

import "github.com/jokruger/dec128/state"

// intRangeError reports whether an int64 fits the platform's int: on a 32-bit platform a value outside the int32 range
// is an overflow rather than a silent wrap.
func intRangeError(i int64) error {
	if int64(int(i)) != i {
		return state.Overflow.Error()
	}
	return nil
}
