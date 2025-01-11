package dec128

import (
	"github.com/jokruger/dec128/errors"
)

// Add returns the sum of the Dec128 and the other Dec128.
// If any of the Dec128 is NaN, the result will be NaN.
// In case of overflow, the result will be NaN.
func (self Dec128) Add(other Dec128) Dec128 {
	if self.err != errors.None {
		return self
	}

	if other.err != errors.None {
		return other
	}

	r, ok := self.tryAdd(other)
	if ok {
		return r
	}

	a := self.Canonical()
	b := other.Canonical()
	r, ok = a.tryAdd(b)
	if ok {
		return r
	}

	return NaN(errors.Overflow)
}

// Sub returns the difference of the Dec128 and the other Dec128.
// If any of the Dec128 is NaN, the result will be NaN.
// In case of overflow/underflow, the result will be NaN.
func (self Dec128) Sub(other Dec128) Dec128 {
	if self.err != errors.None {
		return self
	}

	if other.err != errors.None {
		return other
	}

	r, ok := self.trySub(other)
	if ok {
		return r
	}

	a := self.Canonical()
	b := other.Canonical()
	r, ok = a.trySub(b)
	if ok {
		return r
	}

	return NaN(errors.Overflow)
}

// Mul returns self * other.
// If any of the Dec128 is NaN, the result will be NaN.
// In case of overflow, the result will be NaN.
func (self Dec128) Mul(other Dec128) Dec128 {
	if self.err != errors.None {
		return self
	}

	if other.err != errors.None {
		return other
	}

	if self.IsZero() || other.IsZero() {
		return Zero
	}

	r, ok := self.tryMul(other)
	if ok {
		return r
	}

	a := self.Canonical()
	b := other.Canonical()
	r, ok = a.tryMul(b)
	if ok {
		return r
	}

	return NaN(errors.Overflow)
}

// Div returns self / other.
// If any of the Dec128 is NaN, the result will be NaN.
// In case of overflow, underflow, or division by zero, the result will be NaN.
func (self Dec128) Div(other Dec128) Dec128 {
	if self.err != errors.None {
		return self
	}

	if other.err != errors.None {
		return other
	}

	if other.IsZero() {
		return NaN(errors.DivisionByZero)
	}

	if self.IsZero() {
		return Zero
	}

	r, ok := self.tryDiv(other)
	if ok {
		return r
	}

	a := self.Canonical()
	b := other.Canonical()
	r, ok = a.tryDiv(b)
	if ok {
		return r
	}

	return NaN(errors.Overflow)
}

// Mod returns self % other.
// If any of the Dec128 is NaN, the result will be NaN.
// In case of overflow, underflow, or division by zero, the result will be NaN.
func (self Dec128) Mod(other Dec128) Dec128 {
	if self.err != errors.None {
		return self
	}

	if other.err != errors.None {
		return other
	}

	if other.IsZero() {
		return NaN(errors.DivisionByZero)
	}

	if self.IsZero() {
		return Zero
	}

	_, r, ok := self.tryQuoRem(other)
	if ok {
		return r
	}

	a := self.Canonical()
	b := other.Canonical()
	_, r, ok = a.tryQuoRem(b)
	if ok {
		return r
	}

	return NaN(errors.Overflow)
}

// Abs returns |d|
// If Dec128 is NaN, the result will be NaN.
func (self Dec128) Abs() Dec128 {
	if self.err != errors.None {
		return self
	}
	return Dec128{coef: self.coef, exp: self.exp}
}

// Neg returns -d
// If Dec128 is NaN, the result will be NaN.
func (self Dec128) Neg() Dec128 {
	if self.err != errors.None {
		return self
	}
	return Dec128{coef: self.coef, exp: self.exp, neg: !self.neg}
}

// Sqrt returns the square root of the Dec128.
// If Dec128 is NaN, the result will be NaN.
// If Dec128 is negative, the result will be NaN.
// In case of overflow, the result will be NaN.
func (self Dec128) Sqrt() Dec128 {
	if self.err != errors.None {
		return self
	}

	if self.IsZero() {
		return Zero
	}

	if self.neg {
		return NaN(errors.SqrtNegative)
	}

	if self.Equal(One) {
		return One
	}

	r, ok := self.trySqrt()
	if ok {
		return r
	}

	a := self.Canonical()
	r, ok = a.trySqrt()
	if ok {
		return r
	}

	return NaN(errors.Overflow)
}

// PowInt returns Dec128 raised to the power of n.
func (self Dec128) PowInt(n int) Dec128 {
	if self.err != errors.None {
		return self
	}

	if n < 0 {
		return One.Div(self.PowInt(-n))
	}

	if n == 0 {
		return One
	}

	if n == 1 {
		return self
	}

	if (n & 1) == 0 {
		return self.Mul(self).PowInt(n / 2)
	}

	return self.Mul(self).PowInt((n - 1) / 2).Mul(self)
}
