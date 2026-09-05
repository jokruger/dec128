package dec128

import "github.com/jokruger/dec128/state"

// Add returns the sum of the Dec128 and the other Dec128.
// If any of the Dec128 is NaN, the result will be NaN.
// In case of overflow, the result will be NaN.
func (d Dec128) Add(other Dec128) Dec128 {
	// Return immediately if either value is in an error state.
	switch {
	case d.state >= state.Error:
		return d
	case other.state >= state.Error:
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
	switch {
	case d.state >= state.Error:
		return d
	case other.state >= state.Error:
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
	switch {
	case d.state >= state.Error:
		return d
	case other.state >= state.Error:
		return other
	case d.coef.IsZero() || other.coef.IsZero():
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
// The scale of the result is the larger of d's scale and the package default set by
// SetDefaultScale; use DivAtScale to choose it per call instead.
// If any of the Dec128 is NaN, the result will be NaN.
// In case of overflow, underflow, or division by zero, the result will be NaN.
func (d Dec128) Div(other Dec128) Dec128 {
	return d.DivAtScale(other, defaultScale)
}

// DivAtScale returns d / other, using scale as the minimum scale of the result instead
// of the package default set by SetDefaultScale. Like the default, scale is a floor and
// not a cap: if d already has a larger scale, that one is kept.
// If any of the Dec128 is NaN, the result will be NaN.
// If scale is greater than MaxScale, the result will be NaN.
// In case of overflow, underflow, or division by zero, the result will be NaN.
func (d Dec128) DivAtScale(other Dec128, scale uint8) Dec128 {
	switch {
	case d.state >= state.Error:
		return d
	case other.state >= state.Error:
		return other
	case scale > MaxScale:
		return Dec128{state: state.ScaleOutOfRange}
	case other.coef.IsZero():
		return Dec128{state: state.DivisionByZero}
	case d.coef.IsZero():
		return Zero
	}

	r, ok := d.tryDivAtScale(other, scale)
	if ok {
		return r
	}

	a := d.Canonical()
	b := other.Canonical()
	r, ok = a.tryDivAtScale(b, scale)
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
	switch {
	case d.state >= state.Error:
		return d
	case other.state >= state.Error:
		return other
	case other.coef.IsZero():
		return Dec128{state: state.DivisionByZero}
	case d.coef.IsZero():
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
// The quotient is always an integer and the remainder takes the larger scale of the two
// operands, so unlike Div this does not depend on SetDefaultScale.
// If any of the Dec128 is NaN, the result will be NaN.
// In case of overflow, underflow, or division by zero, the result will be NaN.
func (d Dec128) QuoRem(other Dec128) (Dec128, Dec128) {
	switch {
	case d.state >= state.Error:
		return d, d
	case other.state >= state.Error:
		return other, other
	case other.coef.IsZero():
		return Dec128{state: state.DivisionByZero}, Dec128{state: state.DivisionByZero}
	case d.coef.IsZero():
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
	return Dec128{coef: d.coef, scale: d.scale}
}

// Neg returns -d
// If Dec128 is NaN, the result will be NaN.
func (d Dec128) Neg() Dec128 {
	switch {
	case d.state >= state.Error:
		return d
	case d.state == state.Neg:
		return Dec128{coef: d.coef, scale: d.scale}
	case d.coef.IsZero():
		return d
	default:
		return Dec128{coef: d.coef, scale: d.scale, state: state.Neg}
	}
}

// Sqrt returns the square root of the Dec128.
// The scale of the result is the package default set by SetDefaultScale; use
// SqrtAtScale to choose it per call instead.
// If Dec128 is NaN, the result will be NaN.
// If Dec128 is negative, the result will be NaN.
// In case of overflow, the result will be NaN.
func (d Dec128) Sqrt() Dec128 {
	return d.SqrtAtScale(defaultScale)
}

// SqrtAtScale returns the square root of the Dec128 at the given scale instead of the
// package default set by SetDefaultScale.
// If the Dec128 is NaN, the result will be NaN.
// If the Dec128 is negative, the result will be NaN.
// If scale is greater than MaxScale, the result will be NaN.
// In case of overflow, the result will be NaN.
func (d Dec128) SqrtAtScale(scale uint8) Dec128 {
	switch {
	case d.state >= state.Error:
		return d
	case scale > MaxScale:
		return Dec128{state: state.ScaleOutOfRange}
	case d.coef.IsZero():
		return Zero
	case d.state == state.Neg:
		return Dec128{state: state.SqrtNegative}
	case d.Equal(One):
		return One
	}

	r, ok := d.trySqrtAtScale(scale)
	if ok {
		return r
	}

	// Canonical() fallback is unreachable here; retained only for future changes.
	//a := d.Canonical()
	//r, ok = a.trySqrtAtScale(scale)
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
	switch {
	case d.state >= state.Error:
		return d
	case n < 0:
		return One.Div(d.PowInt64(-n))
	case n == 0:
		return One
	case n == 1:
		return d
	case (n & 1) == 0:
		return d.Mul(d).PowInt64(n / 2)
	default:
		return d.Mul(d).PowInt64((n - 1) / 2).Mul(d)
	}
}
