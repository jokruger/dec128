package uint128

import (
	"errors"
	"math"
)

const (
	MaxStrLen       = 40 // maximum number of digits in a 128-bit unsigned integer
	MaxStrLen64     = 20 // maximum number of digits in a 64-bit unsigned integer
	MaxSafeStrLen64 = 19 // maximum number of digits that can be safely parsed as a 64-bit unsigned integer

	MaxStr   = "340282366920938463463374607431768211455"
	MaxStr64 = "18446744073709551615"
)

var (
	Zero = Uint128{}

	Max   = Uint128{math.MaxUint64, math.MaxUint64}
	Max64 = Uint128{math.MaxUint64, 0}

	ErrDivByZero     = errors.New("division by zero")
	ErrNegative      = errors.New("negative value")
	ErrOverflow      = errors.New("overflow")
	ErrUnderflow     = errors.New("underflow")
	ErrNotEnough     = errors.New("not enough bytes")
	ErrInvalidFormat = errors.New("invalid format")
)
