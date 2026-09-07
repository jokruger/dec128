package dec128

import (
	"database/sql/driver"
	"fmt"

	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// Dec128 represents a 128-bit fixed-point decimal number.
type Dec128 struct {
	coef  uint128.Uint128
	scale uint8
	state state.State
}

// New creates a new Dec128 from a uint64 coefficient, uint8 scale, and negative flag.
// In case of errors it returns NaN with the error.
func New(coef uint128.Uint128, scale uint8, neg bool) Dec128 {
	if scale > MaxScale {
		return NaN(state.ScaleOutOfRange)
	}

	if coef.IsZero() {
		return Dec128{coef: coef, scale: scale}
	}

	if neg {
		return Dec128{coef: coef, scale: scale, state: state.Neg}
	}

	return Dec128{coef: coef, scale: scale}
}

// IsZero returns true if the Dec128 is zero.
// If the Dec128 is NaN, it returns false.
func (d Dec128) IsZero() bool {
	return d.state < state.Error && d.coef.IsZero()
}

// IsNegative returns true if the Dec128 is negative and false otherwise.
// If the Dec128 is NaN, it returns false.
func (d Dec128) IsNegative() bool {
	return d.state == state.Neg && !d.coef.IsZero()
}

// IsPositive returns true if the Dec128 is positive and false otherwise.
// If the Dec128 is NaN, it returns false.
func (d Dec128) IsPositive() bool {
	return d.state != state.Neg && d.state < state.Error && !d.coef.IsZero()
}

// IsNaN returns true if the Dec128 is NaN.
func (d Dec128) IsNaN() bool {
	return d.state >= state.Error
}

// ErrorDetails returns the error details of the Dec128.
// If the Dec128 is not NaN, it returns nil.
func (d Dec128) ErrorDetails() error {
	if d.state < state.Error {
		return nil
	}
	return d.state.Error()
}

// Sign returns -1 if the Dec128 is negative, 0 if it is zero, and 1 if it is positive.
func (d Dec128) Sign() int {
	switch {
	case d.state >= state.Error || d.coef.IsZero():
		return 0
	case d.state == state.Neg:
		return -1
	default:
		return 1
	}
}

// Coefficient returns the coefficient of the Dec128.
func (d Dec128) Coefficient() uint128.Uint128 {
	return d.coef
}

// Scale returns the scale of the Dec128.
func (d Dec128) Scale() uint8 {
	return d.scale
}

// Exponent returns the exponent of the Dec128.
func (d Dec128) Exponent() uint8 {
	return d.scale
}

// Deprecated: Use Scale() instead.
func (d Dec128) Precision() uint8 {
	return d.scale
}

// Rescale returns a new Dec128 with the given scale.
// If the Dec128 is NaN, it returns itself. In case of errors it returns NaN with the error.
func (d Dec128) Rescale(scale uint8) Dec128 {
	if d.state >= state.Error || d.scale == scale {
		return d
	}

	if scale > MaxScale {
		return Dec128{state: state.ScaleOutOfRange}
	}

	if scale > d.scale {
		// scale up
		diff := scale - d.scale
		coef, s := d.coef.Mul64(Pow10Uint64[diff])
		if s >= state.Error {
			return Dec128{state: s}
		}
		return Dec128{coef: coef, scale: scale, state: d.state}
	}

	// scale down: diff is in 1..MaxScale, so QuoRemPow10 cannot fail
	coef, _, _ := d.coef.QuoRemPow10(d.scale - scale)

	return Dec128{coef: coef, scale: scale, state: d.state}
}

// ToScale is an alias for Rescale.
func (d Dec128) ToScale(scale uint8) Dec128 {
	return d.Rescale(scale)
}

// Equal returns true if the Dec128 is equal to the other Dec128.
func (d Dec128) Equal(other Dec128) bool {
	switch {
	case d.state != other.state:
		return false
	case d.state >= state.Error:
		return true
	case d.scale == other.scale:
		return d.coef.Equal(other.coef)
	}
	return d.equalSlow(other)
}

// equalSlow compares numerically across scales by widening to 192 bits.
func (d Dec128) equalSlow(other Dec128) bool {
	if d.coef.IsZero() && other.coef.IsZero() {
		return true
	}
	a, b, _ := alignOperands(d, other)
	return compareWidened(a, b) == 0
}

// Compare returns -1 if the Dec128 is less than the other Dec128, 0 if they are equal, and 1 if the Dec128 is greater
// than the other Dec128. NaN is considered less than any valid Dec128.
func (d Dec128) Compare(other Dec128) int {
	// Fast path: same scale and same sign, so the coefficients compare directly.
	if d.scale == other.scale && d.state == other.state && d.state < state.Error {
		c := d.coef.Compare(other.coef)
		if d.state == state.Neg {
			return -c
		}
		return c
	}
	return d.compareSlow(other)
}

// compareSlow handles NaN, zero, differing signs and differing scales. Operands are aligned by widening to 192 bits,
// so a coefficient that would not fit 128 bits after scaling still compares correctly instead of overflowing.
func (d Dec128) compareSlow(other Dec128) int {
	switch {
	case d.state >= state.Error && other.state >= state.Error:
		return 0
	case d.state >= state.Error:
		return -1
	case other.state >= state.Error:
		return 1
	case d.coef.IsZero() && other.coef.IsZero():
		return 0
	}

	sneg := d.IsNegative()
	oneg := other.IsNegative()

	switch {
	case sneg && !oneg:
		return -1
	case !sneg && oneg:
		return 1
	}

	a, b, _ := alignOperands(d, other)
	c := compareWidened(a, b)
	if sneg {
		return -c
	}

	return c
}

// Canonical returns a new Dec128 with the canonical representation.
// If the Dec128 is NaN, it returns itself.
func (d Dec128) Canonical() Dec128 {
	switch {
	case d.state >= state.Error:
		return Dec128{state: d.state}
	case d.IsZero():
		return Zero
	case d.scale == 0:
		return d
	}

	coef, scale := stripZeros(d.coef, d.scale, 0)
	return Dec128{coef: coef, scale: scale, state: d.state}
}

// LessThan returns true if the Dec128 is less than the other Dec128.
func (d Dec128) LessThan(other Dec128) bool {
	return d.Compare(other) < 0
}

// LessThanOrEqual returns true if the Dec128 is less than or equal to the other Dec128.
func (d Dec128) LessThanOrEqual(other Dec128) bool {
	return d.Compare(other) <= 0
}

// GreaterThan returns true if the Dec128 is greater than the other Dec128.
func (d Dec128) GreaterThan(other Dec128) bool {
	return d.Compare(other) > 0
}

// GreaterThanOrEqual returns true if the Dec128 is greater than or equal to the other Dec128.
func (d Dec128) GreaterThanOrEqual(other Dec128) bool {
	return d.Compare(other) >= 0
}

// Copy returns a copy of the Dec128.
func (d Dec128) Copy() Dec128 {
	return Dec128{coef: d.coef, scale: d.scale, state: d.state}
}

// Scan implements sql.Scanner. Text (string or []byte) is parsed with FromString, so the PostgreSQL spellings NaN,
// Infinity and -Infinity scan to NaN values: only input that is not a number at all is an error. A value that a column
// can hold but this type cannot (too many digits, too many decimals) scans to a NaN carrying the reason, without error,
// so that IsNaN and ErrorDetails see it like any other failed value. Integers convert exactly, and a SQL NULL scans to
// the value configured with SetNullValue.
func (d *Dec128) Scan(src any) error {
	var err error
	switch v := src.(type) {
	case string:
		*d = FromString(v)
		if d.state == state.InvalidFormat {
			err = d.ErrorDetails()
		}
	case []byte:
		// most drivers hand back a numeric/decimal column as bytes; FromString is generic over string | []byte, so this
		// costs no conversion and no allocation
		*d = FromString(v)
		if d.state == state.InvalidFormat {
			err = d.ErrorDetails()
		}
	case int:
		*d = FromInt64(int64(v))
	case int64:
		*d = FromInt64(v)
	case uint64:
		*d = DecodeFromUint64(v, 0)
	case nil:
		*d = nullDec
		if d.state >= state.Error && d.state != state.Null {
			err = d.ErrorDetails()
		}
	default:
		err = fmt.Errorf("can't scan %T to Dec128: %T is not supported", src, src)
	}

	return err
}

// Value implements driver.Valuer. A NULL value (see SetNullValue) becomes SQL NULL; any other value is rendered as text
// in the fixed form, so that its scale reaches the database (1.50 stays 1.50). SetTrimOutput(true) restores the trimmed
// form. A NaN is rendered as "NaN", which PostgreSQL accepts for a numeric column.
func (d Dec128) Value() (driver.Value, error) {
	if d.state == state.Null {
		return nil, nil
	}
	if trimOutput {
		return d.String(), nil
	}
	return d.StringFixed(), nil
}

// NextUp returns the next representable Dec128 greater than the current value.
func (d Dec128) NextUp() Dec128 {
	return d.Add(QuantumAtScale(d.scale))
}

// NextDown returns the next representable Dec128 less than the current value.
func (d Dec128) NextDown() Dec128 {
	return d.Sub(QuantumAtScale(d.scale))
}
