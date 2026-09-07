//go:build !(386 || arm || mips || mipsle)

package dec128

// intRangeError reports whether an int64 fits the platform's int. On a 64-bit platform it always does.
func intRangeError(int64) error {
	return nil
}
