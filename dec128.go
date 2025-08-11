// Package dec128 provides 128-bit fixed-point decimal type, operations and constants.
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
	exp   uint8
	state state.State
}

// New creates a new Dec128 from a uint64 coefficient, uint8 exponent, and negative flag.
// In case of errors it returns NaN with the error.
func New(coef uint128.Uint128, exp uint8, neg bool) Dec128 {
	if exp > MaxPrecision {
		return NaN(state.PrecisionOutOfRange)
	}

	if neg {
		return Dec128{coef: coef, exp: exp, state: state.Neg}
	}

	return Dec128{coef: coef, exp: exp}
}

// NaN returns a Dec128 with the given error.
func NaN(reason state.State) Dec128 {
	if reason < state.Error {
		reason = state.Error
	}
	return Dec128{state: reason}
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
	if d.state >= state.Error || d.coef.IsZero() {
		return 0
	}

	if d.state == state.Neg {
		return -1
	}

	return 1
}

// Precision returns the precision of the Dec128.
func (d Dec128) Precision() uint8 {
	return d.exp
}

// Rescale returns a new Dec128 with the given precision.
// If the Dec128 is NaN, it returns itself.
// In case of errors it returns NaN with the error.
func (d Dec128) Rescale(prec uint8) Dec128 {
	if d.state >= state.Error || d.exp == prec {
		return d
	}

	if prec > MaxPrecision {
		return Dec128{state: state.PrecisionOutOfRange}
	}

	if prec > d.exp {
		// scale up
		diff := prec - d.exp
		coef, s := d.coef.Mul64(Pow10Uint64[diff])
		if s >= state.Error {
			return Dec128{state: s}
		}
		return Dec128{coef: coef, exp: prec, state: d.state}
	}

	// scale down
	diff := d.exp - prec
	coef, s := d.coef.Div64(Pow10Uint64[diff])
	if s >= state.Error {
		return Dec128{state: s}
	}
	return Dec128{coef: coef, exp: prec, state: d.state}
}

// Equal returns true if the Dec128 is equal to the other Dec128.
func (d Dec128) Equal(other Dec128) bool {
	if d.state != other.state {
		return false
	}

	if d.state >= state.Error {
		return true
	}

	if d.exp == other.exp {
		return d.coef.Equal(other.coef)
	}

	if d.coef.IsZero() && other.coef.IsZero() {
		return true
	}

	prec := max(d.exp, other.exp)
	a := d.Rescale(prec)
	b := other.Rescale(prec)
	if !a.IsNaN() && !b.IsNaN() {
		return a.coef.Equal(b.coef)
	}

	return false
}

// Compare returns -1 if the Dec128 is less than the other Dec128, 0 if they are equal, and 1 if the Dec128 is greater than the other Dec128.
// NaN is considered less than any valid Dec128.
func (d Dec128) Compare(other Dec128) int {
	if d.state >= state.Error && other.state >= state.Error {
		return 0
	}

	if d.state >= state.Error {
		return -1
	}

	if other.state >= state.Error {
		return 1
	}

	sneg := d.IsNegative()
	oneg := other.IsNegative()

	if sneg && !oneg {
		return -1
	}

	if !sneg && oneg {
		return 1
	}

	if d.coef.IsZero() && other.coef.IsZero() {
		return 0
	}

	if d.exp == other.exp {
		if sneg {
			return -d.coef.Compare(other.coef)
		}
		return d.coef.Compare(other.coef)
	}

	prec := max(d.exp, other.exp)
	a := d.Rescale(prec)
	if a.IsNaN() {
		return 1
	}
	b := other.Rescale(prec)
	if b.IsNaN() {
		return -1
	}

	if sneg {
		return -a.coef.Compare(b.coef)
	}

	return a.coef.Compare(b.coef)
}

// Canonical returns a new Dec128 with the canonical representation.
// If the Dec128 is NaN, it returns itself.
func (d Dec128) Canonical() Dec128 {
	if d.state >= state.Error {
		return Dec128{state: d.state}
	}

	if d.IsZero() {
		return Zero
	}

	if d.exp == 0 {
		return d
	}

	coef := d.coef
	exp := d.exp
	for {
		t, r, s := coef.QuoRem64(10)
		if s >= state.Error || r > 0 {
			break
		}
		coef = t
		exp--
		if exp == 0 {
			break
		}
	}

	return Dec128{coef: coef, exp: exp, state: d.state}
}

// Exponent returns the exponent of the Dec128.
func (d Dec128) Exponent() uint8 {
	return d.exp
}

// Coefficient returns the coefficient of the Dec128.
func (d Dec128) Coefficient() uint128.Uint128 {
	return d.coef
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
	return Dec128{coef: d.coef, exp: d.exp, state: d.state}
}

// Scan implements the sql.Scanner interface.
func (d *Dec128) Scan(src any) error {
	var err error
	switch v := src.(type) {
	case string:
		*d = FromString(v)
		if d.IsNaN() {
			err = d.ErrorDetails()
		}
	case int:
		*d = FromInt64(int64(v))
	case int64:
		*d = FromInt64(v)
	case nil:
		*d = Zero
	default:
		err = fmt.Errorf("can't scan %T to Dec128: %T is not supported", src, src)
	}

	return err
}

// Value implements the driver.Valuer interface.
func (d Dec128) Value() (driver.Value, error) {
	return d.String(), nil
}
