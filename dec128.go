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
func (self Dec128) IsZero() bool {
	return self.state < state.Error && self.coef.IsZero()
}

// IsNegative returns true if the Dec128 is negative and false otherwise.
// If the Dec128 is NaN, it returns false.
func (self Dec128) IsNegative() bool {
	return self.state == state.Neg && !self.coef.IsZero()
}

// IsPositive returns true if the Dec128 is positive and false otherwise.
// If the Dec128 is NaN, it returns false.
func (self Dec128) IsPositive() bool {
	return self.state != state.Neg && self.state < state.Error && !self.coef.IsZero()
}

// IsNaN returns true if the Dec128 is NaN.
func (self Dec128) IsNaN() bool {
	return self.state >= state.Error
}

// ErrorDetails returns the error details of the Dec128.
// If the Dec128 is not NaN, it returns nil.
func (self Dec128) ErrorDetails() error {
	if self.state < state.Error {
		return nil
	}
	return self.state.Error()
}

// Sign returns -1 if the Dec128 is negative, 0 if it is zero, and 1 if it is positive.
func (self Dec128) Sign() int {
	if self.state >= state.Error || self.coef.IsZero() {
		return 0
	}

	if self.state == state.Neg {
		return -1
	}

	return 1
}

// Precision returns the precision of the Dec128.
func (self Dec128) Precision() uint8 {
	return self.exp
}

// Rescale returns a new Dec128 with the given precision.
// If the Dec128 is NaN, it returns itself.
// In case of errors it returns NaN with the error.
func (self Dec128) Rescale(prec uint8) Dec128 {
	if self.state >= state.Error || self.exp == prec {
		return self
	}

	if prec > MaxPrecision {
		return Dec128{state: state.PrecisionOutOfRange}
	}

	if prec > self.exp {
		// scale up
		diff := prec - self.exp
		coef, s := self.coef.Mul64(Pow10Uint64[diff])
		if s >= state.Error {
			return Dec128{state: s}
		}
		return Dec128{coef: coef, exp: prec, state: self.state}
	}

	// scale down
	diff := self.exp - prec
	coef, s := self.coef.Div64(Pow10Uint64[diff])
	if s >= state.Error {
		return Dec128{state: s}
	}
	return Dec128{coef: coef, exp: prec, state: self.state}
}

// Equal returns true if the Dec128 is equal to the other Dec128.
func (self Dec128) Equal(other Dec128) bool {
	if self.state != other.state {
		return false
	}

	if self.state >= state.Error {
		return true
	}

	if self.exp == other.exp {
		return self.coef.Equal(other.coef)
	}

	if self.coef.IsZero() && other.coef.IsZero() {
		return true
	}

	prec := max(self.exp, other.exp)
	a := self.Rescale(prec)
	b := other.Rescale(prec)
	if !a.IsNaN() && !b.IsNaN() {
		return a.coef.Equal(b.coef)
	}

	return false
}

// Compare returns -1 if the Dec128 is less than the other Dec128, 0 if they are equal, and 1 if the Dec128 is greater than the other Dec128.
// NaN is considered less than any valid Dec128.
func (self Dec128) Compare(other Dec128) int {
	if self.state >= state.Error && other.state >= state.Error {
		return 0
	}

	if self.state >= state.Error {
		return -1
	}

	if other.state >= state.Error {
		return 1
	}

	sneg := self.IsNegative()
	oneg := other.IsNegative()

	if sneg && !oneg {
		return -1
	}

	if !sneg && oneg {
		return 1
	}

	if self.coef.IsZero() && other.coef.IsZero() {
		return 0
	}

	if self.exp == other.exp {
		if sneg {
			return -self.coef.Compare(other.coef)
		}
		return self.coef.Compare(other.coef)
	}

	prec := max(self.exp, other.exp)
	a := self.Rescale(prec)
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
func (self Dec128) Canonical() Dec128 {
	if self.state >= state.Error {
		return Dec128{state: self.state}
	}

	if self.IsZero() {
		return Zero
	}

	if self.exp == 0 {
		return self
	}

	coef := self.coef
	exp := self.exp
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

	return Dec128{coef: coef, exp: exp, state: self.state}
}

// Exponent returns the exponent of the Dec128.
func (self Dec128) Exponent() uint8 {
	return self.exp
}

// Coefficient returns the coefficient of the Dec128.
func (self Dec128) Coefficient() uint128.Uint128 {
	return self.coef
}

// LessThan returns true if the Dec128 is less than the other Dec128.
func (self Dec128) LessThan(other Dec128) bool {
	return self.Compare(other) < 0
}

// LessThanOrEqual returns true if the Dec128 is less than or equal to the other Dec128.
func (self Dec128) LessThanOrEqual(other Dec128) bool {
	return self.Compare(other) <= 0
}

// GreaterThan returns true if the Dec128 is greater than the other Dec128.
func (self Dec128) GreaterThan(other Dec128) bool {
	return self.Compare(other) > 0
}

// GreaterThanOrEqual returns true if the Dec128 is greater than or equal to the other Dec128.
func (self Dec128) GreaterThanOrEqual(other Dec128) bool {
	return self.Compare(other) >= 0
}

// Copy returns a copy of the Dec128.
func (self Dec128) Copy() Dec128 {
	return Dec128{coef: self.coef, exp: self.exp, state: self.state}
}

// Scan implements the sql.Scanner interface.
func (self *Dec128) Scan(src any) error {
	var err error
	switch v := src.(type) {
	case string:
		*self = FromString(v)
		if self.IsNaN() {
			err = self.ErrorDetails()
		}
	case int:
		*self = FromInt64(int64(v))
	case int64:
		*self = FromInt64(v)
	case nil:
		*self = Zero
	default:
		err = fmt.Errorf("can't scan %T to Dec128: %T is not supported", src, src)
	}

	return err
}

// Value implements the driver.Valuer interface.
func (self Dec128) Value() (driver.Value, error) {
	return self.String(), nil
}
