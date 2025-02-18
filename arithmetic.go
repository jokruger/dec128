package dec128

import "github.com/jokruger/dec128/state"

// Add returns the sum of the Dec128 and the other Dec128.
// If any of the Dec128 is NaN, the result will be NaN.
// In case of overflow, the result will be NaN.
func (self Dec128) Add(other Dec128) Dec128 {
	if self.state >= state.Error {
		return self
	}

	if other.state >= state.Error {
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

	return Dec128{state: state.Overflow}
}

// AddInt64 returns the sum of the Dec128 and the int.
// If Dec128 is NaN, the result will be NaN.
// In case of overflow, the result will be NaN.
func (self Dec128) AddInt64(other int64) Dec128 {
	return self.Add(FromInt64(other))
}

// Sub returns the difference of the Dec128 and the other Dec128.
// If any of the Dec128 is NaN, the result will be NaN.
// In case of overflow/underflow, the result will be NaN.
func (self Dec128) Sub(other Dec128) Dec128 {
	if self.state >= state.Error {
		return self
	}

	if other.state >= state.Error {
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

	return Dec128{state: state.Overflow}
}

// SubInt64 returns the difference of the Dec128 and the int.
// If Dec128 is NaN, the result will be NaN.
// In case of overflow/underflow, the result will be NaN.
func (self Dec128) SubInt64(other int64) Dec128 {
	return self.Sub(FromInt64(other))
}

// Mul returns self * other.
// If any of the Dec128 is NaN, the result will be NaN.
// In case of overflow, the result will be NaN.
func (self Dec128) Mul(other Dec128) Dec128 {
	if self.state >= state.Error {
		return self
	}

	if other.state >= state.Error {
		return other
	}

	if self.coef.IsZero() || other.coef.IsZero() {
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

	return Dec128{state: state.Overflow}
}

// MulInt64 returns self * other.
// If Dec128 is NaN, the result will be NaN.
// In case of overflow, the result will be NaN.
func (self Dec128) MulInt64(other int64) Dec128 {
	return self.Mul(FromInt64(other))
}

// Div returns self / other.
// If any of the Dec128 is NaN, the result will be NaN.
// In case of overflow, underflow, or division by zero, the result will be NaN.
func (self Dec128) Div(other Dec128) Dec128 {
	if self.state >= state.Error {
		return self
	}

	if other.state >= state.Error {
		return other
	}

	if other.coef.IsZero() {
		return Dec128{state: state.DivisionByZero}
	}

	if self.coef.IsZero() {
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

	return Dec128{state: state.Overflow}
}

// DivInt64 returns self / other.
// If Dec128 is NaN, the result will be NaN.
// In case of overflow, underflow, or division by zero, the result will be NaN.
func (self Dec128) DivInt64(other int64) Dec128 {
	return self.Div(FromInt64(other))
}

// Mod returns self % other.
// If any of the Dec128 is NaN, the result will be NaN.
// In case of overflow, underflow, or division by zero, the result will be NaN.
func (self Dec128) Mod(other Dec128) Dec128 {
	if self.state >= state.Error {
		return self
	}

	if other.state >= state.Error {
		return other
	}

	if other.coef.IsZero() {
		return Dec128{state: state.DivisionByZero}
	}

	if self.coef.IsZero() {
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

	return Dec128{state: state.Overflow}
}

// ModInt64 returns self % other.
// If Dec128 is NaN, the result will be NaN.
// In case of overflow, underflow, or division by zero, the result will be NaN.
func (self Dec128) ModInt64(other int64) Dec128 {
	return self.Mod(FromInt64(other))
}

// QuoRem returns the quotient and remainder of the division of Dec128 by other Dec128.
// If any of the Dec128 is NaN, the result will be NaN.
// In case of overflow, underflow, or division by zero, the result will be NaN.
func (self Dec128) QuoRem(other Dec128) (Dec128, Dec128) {
	if self.state >= state.Error {
		return self, self
	}

	if other.state >= state.Error {
		return other, other
	}

	if other.coef.IsZero() {
		return Dec128{state: state.DivisionByZero}, Dec128{state: state.DivisionByZero}
	}

	if self.coef.IsZero() {
		return Zero, Zero
	}

	q, r, ok := self.tryQuoRem(other)
	if ok {
		return q, r
	}

	a := self.Canonical()
	b := other.Canonical()
	q, r, ok = a.tryQuoRem(b)
	if ok {
		return q, r
	}

	return Dec128{state: state.Overflow}, Dec128{state: state.Overflow}
}

// QuoRemInt64 returns the quotient and remainder of the division of Dec128 by int.
// If Dec128 is NaN, the result will be NaN.
// In case of overflow, underflow, or division by zero, the result will be NaN.
func (self Dec128) QuoRemInt64(other int64) (Dec128, Dec128) {
	return self.QuoRem(FromInt64(other))
}

// Abs returns |d|
// If Dec128 is NaN, the result will be NaN.
func (self Dec128) Abs() Dec128 {
	if self.state >= state.Error {
		return self
	}
	return Dec128{coef: self.coef, exp: self.exp}
}

// Neg returns -d
// If Dec128 is NaN, the result will be NaN.
func (self Dec128) Neg() Dec128 {
	if self.state >= state.Error {
		return self
	}

	if self.state == state.Neg {
		return Dec128{coef: self.coef, exp: self.exp}
	}

	return Dec128{coef: self.coef, exp: self.exp, state: state.Neg}
}

// Sqrt returns the square root of the Dec128.
// If Dec128 is NaN, the result will be NaN.
// If Dec128 is negative, the result will be NaN.
// In case of overflow, the result will be NaN.
func (self Dec128) Sqrt() Dec128 {
	if self.state >= state.Error {
		return self
	}

	if self.coef.IsZero() {
		return Zero
	}

	if self.state == state.Neg {
		return Dec128{state: state.SqrtNegative}
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

	return Dec128{state: state.Overflow}
}

// PowInt returns Dec128 raised to the power of n.
func (self Dec128) PowInt(n int) Dec128 {
	if self.state >= state.Error {
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
