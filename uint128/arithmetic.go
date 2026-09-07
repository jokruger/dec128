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
	hi, c1 := bits.Add64(hi, p3, 0)

	// c0 and c1 are overflows out of bit 127, not carries to fold back in.
	if (ui.Hi > 0 && other.Hi > 0) || p0 > 0 || p2 > 0 || c0 > 0 || c1 > 0 {
		return Zero, state.Overflow
	}

	return Uint128{lo, hi}, state.OK
}

// MulCarry returns the 256-bit product ui * other as its low and high 128-bit halves (lo, hi).
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
		// other is not zero and other.Hi == 0, so other.Lo will be > 0
		var r64 uint64
		q, r64, _ = ui.QuoRem64(other.Lo)

		// unreachable because QuoRem64 cannot be error for arg > 0
		//if s >= state.Error {
		//	return Zero, Zero, s
		//}

		r = FromUint64(r64)
		return q, r, state.OK
	}

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

// AddCarry returns ui + other modulo 2^128 and the carry out of bit 127 (0 or 1).
// Unlike Add it never reports an error: the caller decides what a carry means.
func (ui Uint128) AddCarry(other Uint128) (Uint128, uint64) {
	lo, carry := bits.Add64(ui.Lo, other.Lo, 0)
	hi, carry := bits.Add64(ui.Hi, other.Hi, carry)
	return Uint128{Lo: lo, Hi: hi}, carry
}

// SubBorrow returns ui - other modulo 2^128 and the borrow out of bit 127 (0 or 1).
// A borrow of 1 means ui < other and the difference wrapped; its two's complement is other - ui.
func (ui Uint128) SubBorrow(other Uint128) (Uint128, uint64) {
	lo, borrow := bits.Sub64(ui.Lo, other.Lo, 0)
	hi, borrow := bits.Sub64(ui.Hi, other.Hi, borrow)
	return Uint128{Lo: lo, Hi: hi}, borrow
}

// Mul64Carry returns the full 192-bit product ui * other as the low 128 bits and the high 64 bits.
// It never overflows: the product of a 128-bit and a 64-bit value fits in 192 bits.
func (ui Uint128) Mul64Carry(other uint64) (Uint128, uint64) {
	p1, p0 := bits.Mul64(ui.Lo, other)
	hi, m := bits.Mul64(ui.Hi, other)
	mid, carry := bits.Add64(p1, m, 0)
	return Uint128{Lo: p0, Hi: mid}, hi + carry
}

// QuoRemPow10 returns ui / 10^k and ui % 10^k for 0 <= k <= 19.
// The remainder always fits in a uint64 because 10^19 < 2^64.
// It returns state.ScaleOutOfRange for k > 19.
func (ui Uint128) QuoRemPow10(k uint8) (Uint128, uint64, state.State) {
	switch {
	case k > MaxSafeStrLen64:
		return Zero, 0, state.ScaleOutOfRange
	case k == 0:
		return ui, 0, state.OK
	case ui.Hi == 0:
		q, r := quoRem64Pow10(ui.Lo, k)
		return Uint128{Lo: q}, r, state.OK
	}
	q, r := quoRem128Pow10(ui, k)
	return q, r, state.OK
}
