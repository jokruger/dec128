package uint128

import (
	"math/bits"

	"github.com/jokruger/dec128/state"
)

// Add returns ui + other and an error if the result overflows.
func (ui Uint128) Add(other Uint128) (Uint128, state.State) {
	lo, carry := bits.Add64(ui.Lo, other.Lo, 0)
	hi, carry := bits.Add64(ui.Hi, other.Hi, carry)

	if carry > 0 {
		return Zero, state.Overflow
	}

	return Uint128{lo, hi}, state.OK
}

// Add64 returns ui + other and an error if the result overflow.
func (ui Uint128) Add64(other uint64) (Uint128, state.State) {
	lo, carry := bits.Add64(ui.Lo, other, 0)
	hi, carry := bits.Add64(ui.Hi, 0, carry)

	if carry > 0 {
		return Zero, state.Overflow
	}

	return Uint128{lo, hi}, state.OK
}

// Sub returns ui - other and an error if the result underflow.
func (ui Uint128) Sub(other Uint128) (Uint128, state.State) {
	lo, borrow := bits.Sub64(ui.Lo, other.Lo, 0)
	hi, borrow := bits.Sub64(ui.Hi, other.Hi, borrow)

	if borrow > 0 {
		return Zero, state.Underflow
	}

	return Uint128{lo, hi}, state.OK
}

// Sub64 returns ui - other and an error if the result underflow.
func (ui Uint128) Sub64(other uint64) (Uint128, state.State) {
	lo, borrow := bits.Sub64(ui.Lo, other, 0)
	hi, borrow := bits.Sub64(ui.Hi, 0, borrow)

	if borrow > 0 {
		return Zero, state.Underflow
	}

	return Uint128{lo, hi}, state.OK
}

// Mul returns ui * other and an error if the result overflows.
func (ui Uint128) Mul(other Uint128) (Uint128, state.State) {
	hi, lo := bits.Mul64(ui.Lo, other.Lo)
	p0, p1 := bits.Mul64(ui.Hi, other.Lo)
	p2, p3 := bits.Mul64(ui.Lo, other.Hi)
	hi, c0 := bits.Add64(hi, p1, 0)
	hi, c1 := bits.Add64(hi, p3, c0)

	if (ui.Hi > 0 && other.Hi > 0) || p0 > 0 || p2 > 0 || c1 > 0 {
		return Zero, state.Overflow
	}

	return Uint128{lo, hi}, state.OK
}

// MulCarry returns ui * other and carry.
func (ui Uint128) MulCarry(other Uint128) (Uint128, Uint128) {
	if ui.Hi == 0 && other.Hi == 0 {
		hi, lo := bits.Mul64(ui.Lo, other.Lo)
		return Uint128{Lo: lo, Hi: hi}, Zero
	}

	hi, lo := bits.Mul64(ui.Lo, other.Lo)
	p0, p1 := bits.Mul64(ui.Hi, other.Lo)
	p2, p3 := bits.Mul64(ui.Lo, other.Hi)

	// calculate hi + p1 + p3
	// total carry = carry(hi+p1) + carry(hi+p1+p3)
	hi, c0 := bits.Add64(hi, p1, 0)
	hi, c1 := bits.Add64(hi, p3, 0)
	c1 += c0

	// calculate upper part of out carry
	e0, e1 := bits.Mul64(ui.Hi, other.Hi)
	d, d0 := bits.Add64(p0, p2, 0)
	d, d1 := bits.Add64(d, c1, 0)
	e2, e3 := bits.Add64(d, e1, 0)

	return Uint128{Lo: lo, Hi: hi}, Uint128{Lo: e2, Hi: e0 + d0 + d1 + e3}
}

// Mul64 returns ui * other and an error if the result overflows.
func (ui Uint128) Mul64(other uint64) (Uint128, state.State) {
	hi, lo := bits.Mul64(ui.Lo, other)
	p0, p1 := bits.Mul64(ui.Hi, other)
	hi, c0 := bits.Add64(hi, p1, 0)

	if p0 > 0 || c0 > 0 {
		return Zero, state.Overflow
	}

	return Uint128{lo, hi}, state.OK
}

// MulAdd64 returns (ui * other1 + other2) and an error if the result overflows.
func (ui Uint128) MulAdd64(other1 uint64, other2 uint64) (Uint128, state.State) {
	hi, lo := bits.Mul64(ui.Lo, other1)
	p0, p1 := bits.Mul64(ui.Hi, other1)
	hi, c0 := bits.Add64(hi, p1, 0)

	if p0 > 0 || c0 > 0 {
		return Zero, state.Overflow
	}

	lo, c1 := bits.Add64(lo, other2, 0)
	hi, c1 = bits.Add64(hi, 0, c1)

	if c1 > 0 {
		return Zero, state.Overflow
	}

	return Uint128{lo, hi}, state.OK
}

// Div returns ui / other and an error if the divisor is zero.
func (ui Uint128) Div(other Uint128) (Uint128, state.State) {
	q, _, s := ui.QuoRem(other)
	return q, s
}

// Div64 returns ui / other and an error if the divisor is zero.
func (ui Uint128) Div64(other uint64) (Uint128, state.State) {
	q, _, s := ui.QuoRem64(other)
	return q, s
}

// Mod returns ui % other and an error if the divisor is zero.
func (ui Uint128) Mod(other Uint128) (Uint128, state.State) {
	_, r, s := ui.QuoRem(other)
	return r, s
}

// Mod64 returns ui % other and an error if the divisor is zero.
func (ui Uint128) Mod64(other uint64) (uint64, state.State) {
	_, r, s := ui.QuoRem64(other)
	return r, s
}

// QuoRem returns ui / other and ui % other and an error if the divisor is zero.
func (ui Uint128) QuoRem(other Uint128) (Uint128, Uint128, state.State) {
	if other.IsZero() {
		return Zero, Zero, state.DivisionByZero
	}

	var q Uint128
	var r Uint128
	var s state.State

	if other.Hi == 0 {
		var r64 uint64
		q, r64, s = ui.QuoRem64(other.Lo)
		if s >= state.Error {
			return Zero, Zero, s
		}
		r = FromUint64(r64)
	} else {
		n := uint(bits.LeadingZeros64(other.Hi))
		v1 := other.Lsh(n)
		u1 := ui.Rsh(1)
		tq, _ := bits.Div64(u1.Hi, u1.Lo, v1.Hi)
		tq >>= 63 - n
		if tq > 0 {
			tq--
		}
		q = FromUint64(tq)
		var m Uint128
		m, s = other.Mul64(tq)
		if s >= state.Error {
			return Zero, Zero, s
		}
		r, s = ui.Sub(m)
		if s >= state.Error {
			return Zero, Zero, s
		}
		if r.Compare(other) >= 0 {
			q, s = q.Add64(1)
			if s >= state.Error {
				return Zero, Zero, s
			}
			r, s = r.Sub(other)
			if s >= state.Error {
				return Zero, Zero, s
			}
		}
	}

	return q, r, state.OK
}

// QuoRem64 returns ui / other and ui % other and an error if the divisor is zero.
func (ui Uint128) QuoRem64(other uint64) (Uint128, uint64, state.State) {
	if other == 0 {
		return Zero, 0, state.DivisionByZero
	}

	var q Uint128
	var r uint64

	if ui.Hi < other {
		q.Lo, r = bits.Div64(ui.Hi, ui.Lo, other)
	} else {
		q.Hi, r = bits.Div64(0, ui.Hi, other)
		q.Lo, r = bits.Div64(r, ui.Lo, other)
	}

	return q, r, state.OK
}
