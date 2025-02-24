package dec128

import (
	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// MaxBytes is the maximum number of bytes that can be used to represent a Dec128 in binary form.
// The actual number of bytes used can be less than this.
const MaxBytes = 18

// MaxPrecision is the maximum number of digits after the decimal point that can be represented.
// MaxPrecision = 19
const MaxPrecision = uint8(uint128.MaxSafeStrLen64)

// MaxStrLen is the maximum number of characters that can be in a string representation of a Dec128.
// MaxStrLen = uint128.MaxStrLen + dot + sign
const MaxStrLen = uint128.MaxStrLen + 2

var (
	Zero        = Dec128{}
	One         = FromInt64(1)
	NegativeOne = FromInt64(-1)

	Decimal0    = Zero
	Decimal1    = One
	Decimal2    = FromInt64(2)
	Decimal3    = FromInt64(3)
	Decimal4    = FromInt64(4)
	Decimal5    = FromInt64(5)
	Decimal6    = FromInt64(6)
	Decimal7    = FromInt64(7)
	Decimal8    = FromInt64(8)
	Decimal9    = FromInt64(9)
	Decimal10   = FromInt64(10)
	Decimal100  = FromInt64(100)
	Decimal365  = FromInt64(365)
	Decimal366  = FromInt64(366)
	Decimal1000 = FromInt64(1000)

	ZeroStr          = "0"
	ZeroStrBytes     = []byte(ZeroStr)
	ZeroJsonStrBytes = []byte(`"0"`)

	NaNStr          = "NaN"
	NaNStrBytes     = []byte(NaNStr)
	NaNJsonStrBytes = []byte(`"NaN"`)

	Pow10Uint64  = uint128.Pow10Uint64
	Pow10Uint128 = uint128.Pow10Uint128

	defaultPrecision = MaxPrecision
)

// SetDefaultPrecision sets the default precision for all Dec128 instances, where precision is the number of digits after the decimal point.
func SetDefaultPrecision(prec uint8) {
	if prec > MaxPrecision {
		panic(state.PrecisionOutOfRange.String())
	}
	defaultPrecision = prec
}
