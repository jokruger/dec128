package dec128

import (
	"github.com/jokruger/dec128/errors"
	"github.com/jokruger/dec128/uint128"
)

// MaxPrecision is the maximum number of digits after the decimal point that can be represented.
// MaxPrecision = 19
const MaxPrecision = uint8(uint128.MaxSafeStrLen64)

// MaxStrLen is the maximum number of characters that can be in a string representation of a Dec128.
// MaxStrLen = uint128.MaxStrLen + dot + sign
const MaxStrLen = uint128.MaxStrLen + 2

var (
	Zero = Dec128{}

	Decimal0    = FromInt(0)
	Decimal1    = FromInt(1)
	Decimal2    = FromInt(2)
	Decimal3    = FromInt(3)
	Decimal4    = FromInt(4)
	Decimal5    = FromInt(5)
	Decimal6    = FromInt(6)
	Decimal7    = FromInt(7)
	Decimal8    = FromInt(8)
	Decimal9    = FromInt(9)
	Decimal10   = FromInt(10)
	Decimal100  = FromInt(100)
	Decimal365  = FromInt(365)
	Decimal366  = FromInt(366)
	Decimal1000 = FromInt(1000)

	ZeroStr      = "0"
	ZeroStrBytes = []byte(ZeroStr)

	NaNStr      = "NaN"
	NaNStrBytes = []byte(NaNStr)

	Pow10Uint64  = uint128.Pow10Uint64
	Pow10Uint128 = uint128.Pow10Uint128

	defaultPrecision = MaxPrecision
)

// SetDefaultPrecision sets the default precision for all Dec128 instances, where precision is the number of digits after the decimal point.
func SetDefaultPrecision(prec uint8) {
	if prec > MaxPrecision {
		panic(errors.PrecisionOutOfRange.Value())
	}
	defaultPrecision = prec
}
