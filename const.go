package dec128

import (
	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// MaxBytes is the maximum number of bytes that can be used to represent a Dec128 in binary form. The actual number of
// bytes used can be less than this.
const MaxBytes = 18

// MaxScale is the maximum number of digits after the decimal point that can be represented.
const MaxScale = uint8(uint128.MaxSafeStrLen64) // 19

// MaxPrecision is the former name of MaxScale.
//
// Deprecated: Use MaxScale instead.
const MaxPrecision = MaxScale

// MaxStrLen is the maximum number of characters that can be in a string representation of a Dec128.
const MaxStrLen = uint128.MaxStrLen + 2

// MaxSciStrLen is the maximum number of characters that can be in a scientific notation representation of a Dec128.
const MaxSciStrLen = uint128.MaxStrLen + 6

// Zero is the value 0 at scale 0. It is also the zero value of the type, so an uninitialised Dec128 is a valid zero.
var Zero = Dec128{}

// One is the value 1.
var One = FromInt64(1)

// NegativeOne is the value -1.
var NegativeOne = FromInt64(-1)

// Small integer constants that recur in financial formulas: digits, percentages and day counts.
var (
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
)

// The textual forms of zero and NaN as String, StringSci and MarshalJSON produce them. The byte slices are shared:
// treat them as read-only.
var (
	ZeroStr          = "0"
	ZeroStrBytes     = []byte(ZeroStr)
	ZeroJsonStrBytes = []byte(`"0"`)

	ZeroSciStr      = "0e+0"
	ZeroSciStrBytes = []byte(ZeroSciStr)

	NaNStr          = "NaN"
	NaNStrBytes     = []byte(NaNStr)
	NaNJsonStrBytes = []byte(`"NaN"`)
)

// Pow10Uint64 holds 10^0..10^19 and Pow10Uint128 holds 10^0..10^38, re-exported from package uint128. Treat them as
// read-only.
var (
	Pow10Uint64  = uint128.Pow10Uint64
	Pow10Uint128 = uint128.Pow10Uint128
)

var defaultScale = MaxScale

// SetDefaultScale sets the scale that Div and Sqrt compute to when no scale is given; DivRound and SqrtRound take it
// per call instead. It panics when the scale is above MaxScale. This is process-global configuration: set it once
// during initialization, because changing it while other goroutines calculate is a data race.
func SetDefaultScale(scale uint8) {
	if scale > MaxScale {
		panic(state.ScaleOutOfRange.Error())
	}
	defaultScale = scale
}

// DefaultScale returns the scale that Div and Sqrt compute to (see SetDefaultScale).
func DefaultScale() uint8 {
	return defaultScale
}

// SetDefaultPrecision is the former name of SetDefaultScale.
//
// Deprecated: Use SetDefaultScale instead.
func SetDefaultPrecision(prec uint8) {
	SetDefaultScale(prec)
}
