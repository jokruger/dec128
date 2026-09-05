package dec128

import (
	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// MaxBytes is the maximum number of bytes that can be used to represent a Dec128 in binary form.
// The actual number of bytes used can be less than this.
const MaxBytes = 18

// MaxScale is the maximum number of digits after the decimal point that can be represented.
// MaxScale = 19
const MaxScale = uint8(uint128.MaxSafeStrLen64)

// Deprecated: Use MaxScale instead.
const MaxPrecision = MaxScale

// MaxStrLen is the maximum number of characters that can be in a string representation of a Dec128.
// MaxStrLen = uint128.MaxStrLen + dot + sign
const MaxStrLen = uint128.MaxStrLen + 2

// MaxSciStrLen is the maximum number of characters that can be in a scientific notation
// representation of a Dec128.
// MaxSciStrLen = uint128.MaxStrLen + sign + dot + marker + exponent sign + 2 exponent digits
const MaxSciStrLen = uint128.MaxStrLen + 6

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

	ZeroSciStr      = "0e+0"
	ZeroSciStrBytes = []byte(ZeroSciStr)

	NaNStr          = "NaN"
	NaNStrBytes     = []byte(NaNStr)
	NaNJsonStrBytes = []byte(`"NaN"`)

	Pow10Uint64  = uint128.Pow10Uint64
	Pow10Uint128 = uint128.Pow10Uint128

	// defaultScale is the scale Div, QuoRem and Sqrt fall back to. It is a floor, not a
	// cap: an operand with a larger scale keeps its own. It deliberately leaves headroom
	// below MaxScale so that a quotient can still be multiplied afterwards - at MaxScale
	// every division lands on the ceiling and the next Mul with a fractional operand
	// overflows to NaN.
	defaultScale = uint8(6)
)

// SetDefaultScale sets the scale that Div and Sqrt fall back to. It does not affect how
// values are parsed or constructed: FromString("1.5") has scale 1 whatever this is set
// to, and QuoRem does not consult it either.
//
// The value is a floor, not a cap - an operand that already has a larger scale keeps it.
// It must not exceed MaxScale; this is the only function in the package that panics, and
// only here, at configuration time.
//
// Leaving headroom below MaxScale matters: at MaxScale every quotient lands on the
// ceiling, and multiplying one by any operand with a fractional part then overflows to
// NaN. Use DivAtScale or SqrtAtScale to choose the scale per call instead.
//
// This is process-global state. Set it once during initialisation, before any decimal is
// used; calling it while other goroutines are calculating is a data race.
func SetDefaultScale(scale uint8) {
	if scale > MaxScale {
		panic(state.ScaleOutOfRange.Error())
	}
	defaultScale = scale
}

// DefaultScale returns the scale that Div and Sqrt fall back to.
func DefaultScale() uint8 {
	return defaultScale
}

// Deprecated: Use SetDefaultScale instead.
func SetDefaultPrecision(prec uint8) {
	SetDefaultScale(prec)
}
