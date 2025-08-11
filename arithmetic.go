package dec128

import "github.com/jokruger/dec128/state"

// Add returns the sum of the Dec128 and the other Dec128.
// If any of the Dec128 is NaN, the result will be NaN.
// In case of overflow, the result will be NaN.
func (d Dec128) Add(other Dec128) Dec128 {
	// Return immediately if either value is in an error state.
	if d.state >= state.Error {
		return d
	}
	if other.state >= state.Error {
		return other
	}

	// Try a fast-path add on the non‑canonical forms.
	if r, ok := d.tryAdd(other); ok {
		return r
	}

	// Canonicalize both values and try again.
	if r, ok := d.Canonical().tryAdd(other.Canonical()); ok {
		return r
	}

	// If addition could not be performed without overflow, return an overflow Dec128.
	return Dec128{state: state.Overflow}
}

// AddInt returns the sum of the Dec128 and the int.
// If Dec128 is NaN, the result will be NaN.
// In case of overflow, the result will be NaN.
func (d Dec128) AddInt(other int) Dec128 {
	return d.AddInt64(int64(other))
}

// AddInt64 returns the sum of the Dec128 and the int.
// If Dec128 is NaN, the result will be NaN.
// In case of overflow, the result will be NaN.
func (d Dec128) AddInt64(other int64) Dec128 {
	return d.Add(FromInt64(other))
}

// Sub returns the difference of the Dec128 and the other Dec128.
// If any of the Dec128 is NaN, the result will be NaN.
// In case of overflow/underflow, the result will be NaN.
func (d Dec128) Sub(other Dec128) Dec128 {
	// Return immediately if either value is in an error state.
	if d.state >= state.Error {
		return d
	}
	if other.state >= state.Error {
		return other
	}

	// Try a fast-path sub on the non‑canonical forms.
	if r, ok := d.trySub(other); ok {
		return r
	}

	// Canonicalize both values and try again.
	if r, ok := d.Canonical().trySub(other.Canonical()); ok {
		return r
	}

	// If subtraction could not be performed without overflow, return an overflow Dec128.
	return Dec128{state: state.Overflow}
}

// SubInt returns the difference of the Dec128 and the int.
// If Dec128 is NaN, the result will be NaN.
// In case of overflow/underflow, the result will be NaN.
func (d Dec128) SubInt(other int) Dec128 {
	return d.SubInt64(int64(other))
}

// SubInt64 returns the difference of the Dec128 and the int.
// If Dec128 is NaN, the result will be NaN.
// In case of overflow/underflow, the result will be NaN.
func (d Dec128) SubInt64(other int64) Dec128 {
	return d.Sub(FromInt64(other))
}

// Mul returns d * other.
// If any of the Dec128 is NaN, the result will be NaN.
// In case of overflow, the result will be NaN.
func (d Dec128) Mul(other Dec128) Dec128 {
	if d.state >= state.Error {
		return d
	}

	if other.state >= state.Error {
		return other
	}

	if d.coef.IsZero() || other.coef.IsZero() {
		return Zero
	}

	r, ok := d.tryMul(other)
	if ok {
		return r
	}

	// Fallback is unreachable; retained only for future changes.
	//a := d.Canonical()
	//b := other.Canonical()
	//r, ok = a.tryMul(b)
	//if ok {
	//	return r
	//}

	return Dec128{state: state.Overflow}
}

// MulInt returns d * other.
// If Dec128 is NaN, the result will be NaN.
// In case of overflow, the result will be NaN.
func (d Dec128) MulInt(other int) Dec128 {
	return d.MulInt64(int64(other))
}

// MulInt64 returns d * other.
// If Dec128 is NaN, the result will be NaN.
// In case of overflow, the result will be NaN.
func (d Dec128) MulInt64(other int64) Dec128 {
	return d.Mul(FromInt64(other))
}

// Div returns d / other.
// If any of the Dec128 is NaN, the result will be NaN.
// In case of overflow, underflow, or division by zero, the result will be NaN.
func (d Dec128) Div(other Dec128) Dec128 {
	if d.state >= state.Error {
		return d
	}

	if other.state >= state.Error {
		return other
	}

	if other.coef.IsZero() {
		return Dec128{state: state.DivisionByZero}
	}

	if d.coef.IsZero() {
		return Zero
	}

	r, ok := d.tryDiv(other)
	if ok {
		return r
	}

	a := d.Canonical()
	b := other.Canonical()
	r, ok = a.tryDiv(b)
	if ok {
		return r
	}

	return Dec128{state: state.Overflow}
}

// DivInt returns d / other.
// If Dec128 is NaN, the result will be NaN.
// In case of overflow, underflow, or division by zero, the result will be NaN.
func (d Dec128) DivInt(other int) Dec128 {
	return d.DivInt64(int64(other))
}

// DivInt64 returns d / other.
// If Dec128 is NaN, the result will be NaN.
// In case of overflow, underflow, or division by zero, the result will be NaN.
func (d Dec128) DivInt64(other int64) Dec128 {
	return d.Div(FromInt64(other))
}

// Mod returns d % other.
// If any of the Dec128 is NaN, the result will be NaN.
// In case of overflow, underflow, or division by zero, the result will be NaN.
func (d Dec128) Mod(other Dec128) Dec128 {
	if d.state >= state.Error {
		return d
	}

	if other.state >= state.Error {
		return other
	}

	if other.coef.IsZero() {
		return Dec128{state: state.DivisionByZero}
	}

	if d.coef.IsZero() {
		return Zero
	}

	_, r, ok := d.tryQuoRem(other)
	if ok {
		return r
	}

	a := d.Canonical()
	b := other.Canonical()
	_, r, ok = a.tryQuoRem(b)
	if ok {
		return r
	}

	return Dec128{state: state.Overflow}
}

// ModInt returns d % other.
// If Dec128 is NaN, the result will be NaN.
// In case of overflow, underflow, or division by zero, the result will be NaN.
func (d Dec128) ModInt(other int) Dec128 {
	return d.ModInt64(int64(other))
}

// ModInt64 returns d % other.
// If Dec128 is NaN, the result will be NaN.
// In case of overflow, underflow, or division by zero, the result will be NaN.
func (d Dec128) ModInt64(other int64) Dec128 {
	return d.Mod(FromInt64(other))
}

// QuoRem returns the quotient and remainder of the division of Dec128 by other Dec128.
// If any of the Dec128 is NaN, the result will be NaN.
// In case of overflow, underflow, or division by zero, the result will be NaN.
func (d Dec128) QuoRem(other Dec128) (Dec128, Dec128) {
	if d.state >= state.Error {
		return d, d
	}

	if other.state >= state.Error {
		return other, other
	}

	if other.coef.IsZero() {
		return Dec128{state: state.DivisionByZero}, Dec128{state: state.DivisionByZero}
	}

	if d.coef.IsZero() {
		return Zero, Zero
	}

	q, r, ok := d.tryQuoRem(other)
	if ok {
		return q, r
	}

	a := d.Canonical()
	b := other.Canonical()
	q, r, ok = a.tryQuoRem(b)
	if ok {
		return q, r
	}

	return Dec128{state: state.Overflow}, Dec128{state: state.Overflow}
}

// QuoRemInt returns the quotient and remainder of the division of Dec128 by int.
// If Dec128 is NaN, the result will be NaN.
// In case of overflow, underflow, or division by zero, the result will be NaN.
func (d Dec128) QuoRemInt(other int) (Dec128, Dec128) {
	return d.QuoRemInt64(int64(other))
}

// QuoRemInt64 returns the quotient and remainder of the division of Dec128 by int.
// If Dec128 is NaN, the result will be NaN.
// In case of overflow, underflow, or division by zero, the result will be NaN.
func (d Dec128) QuoRemInt64(other int64) (Dec128, Dec128) {
	return d.QuoRem(FromInt64(other))
}

// Abs returns |d|
// If Dec128 is NaN, the result will be NaN.
func (d Dec128) Abs() Dec128 {
	if d.state >= state.Error {
		return d
	}
	return Dec128{coef: d.coef, exp: d.exp}
}

// Neg returns -d
// If Dec128 is NaN, the result will be NaN.
func (d Dec128) Neg() Dec128 {
	if d.state >= state.Error {
		return d
	}

	if d.state == state.Neg {
		return Dec128{coef: d.coef, exp: d.exp}
	}

	return Dec128{coef: d.coef, exp: d.exp, state: state.Neg}
}

// Sqrt returns the square root of the Dec128.
// If Dec128 is NaN, the result will be NaN.
// If Dec128 is negative, the result will be NaN.
// In case of overflow, the result will be NaN.
func (d Dec128) Sqrt() Dec128 {
	if d.state >= state.Error {
		return d
	}

	if d.coef.IsZero() {
		return Zero
	}

	if d.state == state.Neg {
		return Dec128{state: state.SqrtNegative}
	}

	if d.Equal(One) {
		return One
	}

	r, ok := d.trySqrt()
	if ok {
		return r
	}

	// Fallback is unreachable; retained only for future changes.
	//a := d.Canonical()
	//r, ok = a.trySqrt()
	//if ok {
	//	return r
	//}

	return Dec128{state: state.Overflow}
}

// PowInt returns Dec128 raised to the power of n.
func (d Dec128) PowInt(n int) Dec128 {
	return d.PowInt64(int64(n))
}

// PowInt64 returns Dec128 raised to the power of n.
func (d Dec128) PowInt64(n int64) Dec128 {
	if d.state >= state.Error {
		return d
	}

	if n < 0 {
		return One.Div(d.PowInt64(-n))
	}

	if n == 0 {
		return One
	}

	if n == 1 {
		return d
	}

	if (n & 1) == 0 {
		return d.Mul(d).PowInt64(n / 2)
	}

	return d.Mul(d).PowInt64((n - 1) / 2).Mul(d)
}
