package dec128

import (
	"github.com/jokruger/dec128/errors"
	"github.com/jokruger/dec128/uint128"
)

type Dec128 struct {
	coef uint128.Uint128
	exp  uint8
	err  errors.Error
	neg  bool
}

// New creates a new Dec128 from a uint64 coefficient, uint8 exponent, and negative flag.
func New(coef uint128.Uint128, exp uint8, neg bool) Dec128 {
	if exp > MaxPrecision {
		return NaN(errors.PrecisionOutOfRange)
	}

	if coef.IsZero() && exp == 0 {
		return Zero
	}

	return Dec128{coef: coef, exp: exp, neg: neg}
}

// NaN returns a Dec128 with the given error.
func NaN(reason errors.Error) Dec128 {
	if reason == errors.None {
		return Dec128{err: errors.NotANumber}
	}
	return Dec128{err: reason}
}

// IsZero returns true if the Dec128 is zero.
// If the Dec128 is NaN, it returns false.
func (self Dec128) IsZero() bool {
	return self.err == errors.None && self.coef.IsZero()
}

// IsNegative returns true if the Dec128 is negative and false otherwise.
// If the Dec128 is NaN, it returns false.
func (self Dec128) IsNegative() bool {
	return self.neg && self.err == errors.None && !self.coef.IsZero()
}

// IsPosistive returns true if the Dec128 is positive and false otherwise.
// If the Dec128 is NaN, it returns false.
func (self Dec128) IsPosistive() bool {
	return !self.neg && self.err == errors.None && !self.coef.IsZero()
}

// IsNaN returns true if the Dec128 is NaN.
func (self Dec128) IsNaN() bool {
	return self.err != errors.None
}

// ErrorDetails returns the error details of the Dec128.
// If the Dec128 is not NaN, it returns nil.
func (self Dec128) ErrorDetails() error {
	return self.err.Value()
}

// Sign returns -1 if the Dec128 is negative, 0 if it is zero, and 1 if it is positive.
func (self Dec128) Sign() int {
	if self.err != errors.None || self.IsZero() {
		return 0
	}

	if self.IsNegative() {
		return -1
	}

	return 1
}

// Precision returns the precision of the Dec128.
func (self Dec128) Precision() uint8 {
	return self.exp
}

// Rescale returns a new Dec128 with the given precision.
func (self Dec128) Rescale(prec uint8) Dec128 {
	if self.err != errors.None {
		return self
	}

	if self.exp == prec {
		return self
	}

	if prec > MaxPrecision {
		return NaN(errors.PrecisionOutOfRange)
	}

	if prec > self.exp {
		// scale up
		diff := prec - self.exp
		coef, err := self.coef.Mul64(Pow10Uint64[diff])
		if err != errors.None {
			return NaN(err)
		}
		return Dec128{coef: coef, exp: prec, neg: self.neg}
	}

	// scale down
	diff := self.exp - prec
	coef, err := self.coef.Div64(Pow10Uint64[diff])
	if err != errors.None {
		return NaN(err)
	}
	return Dec128{coef: coef, exp: prec, neg: self.neg}
}

// Equal returns true if the Dec128 is equal to the other Dec128.
func (self Dec128) Equal(other Dec128) bool {
	if self.err != errors.None && other.err != errors.None {
		return true
	}

	if self.err != errors.None || other.err != errors.None {
		return false
	}

	if self.neg != other.neg {
		return false
	}

	if self.exp == other.exp {
		return self.coef.Equal(other.coef)
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
	if self.err != errors.None && other.err != errors.None {
		return 0
	}

	if self.err != errors.None {
		return -1
	}

	if other.err != errors.None {
		return 1
	}

	if self.neg && !other.neg {
		return -1
	}

	if !self.neg && other.neg {
		return 1
	}

	if self.exp == other.exp {
		if self.neg {
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

	if a.neg {
		return -a.coef.Compare(b.coef)
	}

	return a.coef.Compare(b.coef)
}

// Canonical returns a new Dec128 with the canonical representation.
func (self Dec128) Canonical() Dec128 {
	if self.err != errors.None {
		return Dec128{err: self.err}
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
		t, r, err := coef.QuoRem64(10)
		if err != errors.None || r != 0 {
			break
		}
		coef = t
		exp--
		if exp == 0 {
			break
		}
	}

	return Dec128{coef: coef, exp: exp, neg: self.neg}
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
