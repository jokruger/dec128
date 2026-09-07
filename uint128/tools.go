package uint128

import (
	"encoding/binary"
	"math/bits"

	"github.com/jokruger/dec128/state"
)

// PutBytes writes the Uint128 to the byte slice bs in little-endian order.
func (ui Uint128) PutBytes(bs []byte) state.State {
	if len(bs) < 16 {
		return state.NotEnoughBytes
	}

	binary.LittleEndian.PutUint64(bs[:8], ui.Lo)
	binary.LittleEndian.PutUint64(bs[8:], ui.Hi)

	return state.OK
}

// PutBytesBigEndian writes the Uint128 to the byte slice bs in big-endian order.
func (ui Uint128) PutBytesBigEndian(bs []byte) state.State {
	if len(bs) < 16 {
		return state.NotEnoughBytes
	}

	binary.BigEndian.PutUint64(bs[:8], ui.Hi)
	binary.BigEndian.PutUint64(bs[8:], ui.Lo)

	return state.OK
}

// AppendBytes appends the Uint128 to the byte slice bs in little-endian order.
func (ui Uint128) AppendBytes(bs []byte) []byte {
	bs = binary.LittleEndian.AppendUint64(bs, ui.Lo)
	bs = binary.LittleEndian.AppendUint64(bs, ui.Hi)
	return bs
}

// AppendBytesBigEndian appends the Uint128 to the byte slice bs in big-endian order.
func (ui Uint128) AppendBytesBigEndian(bs []byte) []byte {
	bs = binary.BigEndian.AppendUint64(bs, ui.Hi)
	bs = binary.BigEndian.AppendUint64(bs, ui.Lo)
	return bs
}

// ReverseBytes returns the Uint128 with the byte order reversed.
func (ui Uint128) ReverseBytes() Uint128 {
	return Uint128{bits.ReverseBytes64(ui.Hi), bits.ReverseBytes64(ui.Lo)}
}

// QuoRem256By128 divides the 256-bit value carry*2^128 + u by v and returns the quotient, the remainder and a state:
// state.DivisionByZero when v is zero and state.Overflow when the quotient does not fit in 128 bits (carry >= v).
func QuoRem256By128(u Uint128, carry Uint128, v Uint128) (Uint128, Uint128, state.State) {
	switch {
	case carry.IsZero():
		return Uint128{Lo: u.Lo, Hi: u.Hi}.QuoRem(v)
	case v.Hi == 0 && carry.Hi == 0:
		q, r, e := QuoRem192By64(u, carry.Lo, v.Lo)
		return q, Uint128{Lo: r}, e
	case carry.Compare(v) >= 0:
		// now we have u192 / u128 or u256 / u128
		// obviously the result won't fit into u128
		return Zero, Zero, state.Overflow
	}

	// perform u256 / u128, where carry < u128
	// based on divllu from https://github.com/ridiculousfish/libdivide
	// algorithm is explained in this blog post: https://ridiculousfish.com/blog/posts/labor-of-division-episode-iv.html
	// normalize v
	n := bits.LeadingZeros64(v.Hi)

	// 0 <= n <= 63, so it's safe to convert to uint
	v = v.Lsh(uint(n))

	// shift u to the left by n bits (n < 64)
	a := [4]uint64{}
	a[0] = u.Lo << n
	a[1] = u.Lo>>(64-n) | u.Hi<<n
	a[2] = u.Hi>>(64-n) | carry.Lo<<n
	a[3] = carry.Lo>>(64-n) | carry.Hi<<n

	// q = a / v. The window [a3,a2] is below v because carry < v, so a3 <= v.Hi; when a3 is zero and a2 is still at least
	// v.Hi the top digit can be nonzero and the four-limb form is needed as well.
	aLen := 3
	if a[3] > 0 || a[2] >= v.Hi {
		aLen = 4
	}

	q := [2]uint64{}

	for i := aLen - 3; i >= 0; i-- {
		u2, u1, u0 := a[i+2], a[i+1], a[i]

		// Trial quotient tq = [u2,u1] / v.Hi, at most two above the true digit (Knuth, Algorithm D, step D3). When u2
		// equals v.Hi that division would overflow a limb, and the trial digit is the largest one instead; u2 > v.Hi
		// cannot happen because the window is below v * 2^64.
		var tq uint64
		if u2 >= v.Hi {
			tq = ^uint64(0)
		} else {
			tq, _ = bits.Div64(u2, u1, v.Hi)
		}

		// p = tq * v as the 192-bit value [pHi, pLo]; while it exceeds the window the digit is one too large.
		// The loop runs at most twice.
		pLo, pHi := v.Mul64Carry(tq)
		for pHi > u2 || (pHi == u2 && (pLo.Hi > u1 || (pLo.Hi == u1 && pLo.Lo > u0))) {
			tq--
			var borrow uint64
			pLo, borrow = pLo.SubBorrow(v)
			pHi -= borrow
		}

		q[i] = tq

		// remainder [u2,u1,u0] - p, which fits in 128 bits now that tq is the true digit
		r0, borrow := bits.Sub64(u0, pLo.Lo, 0)
		r1, _ := bits.Sub64(u1, pLo.Hi, borrow)
		a[i+1], a[i] = r1, r0
	}

	// 0 <= n <= 63, so it's safe to convert to uint
	r := Uint128{Lo: a[0], Hi: a[1]}.Rsh(uint(n))

	return Uint128{Lo: q[0], Hi: q[1]}, r, state.OK
}

// QuoRem192By64 divides the 192-bit value carry*2^128 + u by the 64-bit v and returns the 128-bit quotient q and the
// remainder r with u = q*v + r. It returns state.Overflow when carry >= v, because the quotient would not fit.
func QuoRem192By64(u Uint128, carry uint64, v uint64) (Uint128, uint64, state.State) {
	if carry >= v {
		return Zero, 0, state.Overflow
	}

	// can't panic because we already check u.carry < v (u.carry.hi == 0 && u.carry.lo < v)
	hi, rem := bits.Div64(carry, u.Hi, v)

	// can't panic because rem < v
	lo, r := bits.Div64(rem, u.Lo, v)

	return Uint128{Lo: lo, Hi: hi}, r, state.OK
}

// SubUnsafe returns u - v assuming u >= v; the result wraps and is incorrect otherwise. It exists for callers that
// have already established the ordering and want to skip the check.
func SubUnsafe(u Uint128, v Uint128) Uint128 {
	lo, borrow := bits.Sub64(u.Lo, v.Lo, 0)
	hi, _ := bits.Sub64(u.Hi, v.Hi, borrow)
	return Uint128{Lo: lo, Hi: hi}
}
