package dec128

import (
	"math/bits"

	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// wide is an unsigned magnitude of up to 384 bits, little-endian by 64-bit limb.
//
// It exists for the two operations whose exact intermediate is wider than the 256 bits fitWide works in. A product
// reaches 2^256, and aligning it to a common scale with an addend can multiply it by a further 10^19; an accumulator
// that aligns products to 2*MaxScale places reaches 2^382. Everything narrower keeps using widened (192 bits, for
// Add and Sub) or the (lo, hi Uint128) pair (256 bits, for Mul), which are cheaper; this type is for the cases that
// those cannot hold.
//
// Division by a power of ten goes through the reciprocals in package uint128 like every other such division in the
// library, one limb at a time, so no hardware divide is used here either.
type wide [wideLimbs]uint64

const wideLimbs = 6

// wideFrom128 widens a coefficient.
func wideFrom128(u uint128.Uint128) wide {
	return wide{u.Lo, u.Hi}
}

// wideFrom256 widens the 256-bit value hi*2^128 + lo, as MulCarry returns it.
func wideFrom256(lo, hi uint128.Uint128) wide {
	return wide{lo.Lo, lo.Hi, hi.Lo, hi.Hi}
}

// uint128 narrows a to a coefficient, reporting whether it fits.
func (a wide) uint128() (uint128.Uint128, bool) {
	if a[2]|a[3]|a[4]|a[5] != 0 {
		return uint128.Zero, false
	}
	return uint128.Uint128{Lo: a[0], Hi: a[1]}, true
}

// compare orders two magnitudes.
func (a wide) compare(b wide) int {
	for i := wideLimbs - 1; i >= 0; i-- {
		if a[i] != b[i] {
			if a[i] < b[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}

// add returns a + b and whether it carried out of 384 bits.
func (a wide) add(b wide) (wide, bool) {
	var r wide
	var carry uint64
	for i := range r {
		r[i], carry = bits.Add64(a[i], b[i], carry)
	}
	return r, carry != 0
}

// sub returns a - b and requires a >= b; the result wraps otherwise.
func (a wide) sub(b wide) wide {
	var r wide
	var borrow uint64
	for i := range r {
		r[i], borrow = bits.Sub64(a[i], b[i], borrow)
	}
	return r
}

// mul64 returns a * m and whether it carried out of 384 bits.
func (a wide) mul64(m uint64) (wide, bool) {
	var r wide
	var carry uint64
	for i := range r {
		hi, lo := bits.Mul64(a[i], m)
		var c uint64
		r[i], c = bits.Add64(lo, carry, 0)
		// hi is at most 2^64-2 for a full-width product, so it absorbs the carry without wrapping
		carry = hi + c
	}
	return r, carry != 0
}

// mulPow10 returns a * 10^k for k up to 2*MaxScale, and whether it carried out of 384 bits. A single limb holds at
// most 10^MaxScale, so a larger k is two multiplications.
func (a wide) mulPow10(k uint8) (wide, bool) {
	if k > MaxScale {
		var over bool
		if a, over = a.mul64(Pow10Uint64[MaxScale]); over {
			return a, true
		}
		k -= MaxScale
	}
	if k == 0 {
		return a, false
	}
	return a.mul64(Pow10Uint64[k])
}

// quoRemPow10Small returns a / 10^k and a % 10^k for 1 <= k <= MaxScale, with one Moller-Granlund step per limb over
// the normalized dividend.
func (a wide) quoRemPow10Small(k uint8) (q wide, r uint64) {
	// k is in range by the precondition, so the reciprocal is always there
	dn, v, shift, _ := uint128.Pow10Reciprocal(k)
	s := uint(shift)

	// Normalize by shifting left by s. The bits that leave the top limb are the first partial remainder, and they are
	// below 2^s <= 2^60 < dn, which is what the division step requires. For s == 0 every shift by 64 is zero in Go,
	// so the normalization is a no-op as intended.
	//
	// The division starts at the highest nonzero limb rather than at the top of the register: a leading zero limb
	// contributes a zero quotient limb and leaves the remainder alone, and the powers and the accumulator work in a
	// register several limbs wider than the values they usually hold.
	top := wideLimbs - 1
	for top > 0 && a[top] == 0 {
		top--
	}
	r = a[top] >> (64 - s)
	for i := top; i >= 0; i-- {
		u := a[i] << s
		if i > 0 {
			u |= a[i-1] >> (64 - s)
		}
		q[i], r = uint128.Div2By1Unsafe(r, u, dn, v)
	}

	return q, r >> s
}

// quoRem64 returns a / v and a % v for a nonzero v, with one 128-by-64 division per limb from the top down. Each
// step's remainder is below v, which is what bits.Div64 requires of its high word.
//
// The division starts at the highest nonzero limb, as quoRemPow10Small and quoRem128 do: a leading zero limb divides
// to a zero quotient digit and leaves the remainder at zero, so skipping it changes nothing. It is worth the scan
// here because each step is a hardware divide rather than a reciprocal multiply, and the register is usually far
// wider than the value in it - a fused multiply-divide over money-sized operands occupies one limb of six.
func (a wide) quoRem64(v uint64) (q wide, r uint64) {
	top := wideLimbs - 1
	for top > 0 && a[top] == 0 {
		top--
	}
	for i := top; i >= 0; i-- {
		q[i], r = bits.Div64(r, a[i], v)
	}
	return q, r
}

// quoRem128 returns a / v and a % v for a nonzero v, with one 192-by-128 division per limb from the top down.
//
// Each step divides r*2^64 + a[i] by v, where r is the remainder the step above left and is therefore below v. The
// dividend is then below v * 2^64 <= 2^192, so the quotient digit fits a single limb and the division can never report
// an overflow. The remainder that falls out at the end is the exact one against the exact divisor, which is what the
// rounding decision in roundUp takes.
//
// The division starts at the highest nonzero limb rather than at the top of the register, as quoRemPow10Small does:
// a leading zero limb contributes a zero quotient digit and leaves the remainder alone, so a numerator that fits 256
// bits - which is every fused multiply-divide whose scales leave the alignment small - costs four divisions and not
// six.
func (a wide) quoRem128(v uint128.Uint128) (q wide, r uint128.Uint128) {
	top := wideLimbs - 1
	for top > 0 && a[top] == 0 {
		top--
	}
	for i := top; i >= 0; i-- {
		// the 256-bit dividend r*2^64 + a[i], as the (lo, hi) pair QuoRem256By128 takes
		d, rem, _ := uint128.QuoRem256By128(uint128.Uint128{Lo: a[i], Hi: r.Lo}, uint128.Uint128{Lo: r.Hi}, v)
		q[i], r = d.Lo, rem
	}
	return q, r
}

// quoCmpHalfPow10 returns a / 10^k truncated, together with the two facts a rounding decision needs about what was
// dropped: whether it is above half of 10^k and whether it is exactly half, plus whether it was nonzero at all.
//
// k may be wider than a single divisor. The division then runs in steps of at most MaxScale, and only the last step's
// remainder takes part in the comparison: the steps go from the low digits up, so that remainder is the most
// significant part of what was dropped and the earlier ones are no more than a sticky bit.
func (a wide) quoCmpHalfPow10(k uint8) (q wide, above, tie, inexact bool) {
	return a.quoCmpHalfPow10Sticky(k, false)
}

// quoCmpHalfPow10Sticky is quoCmpHalfPow10 with digits the caller has already dropped folded into the comparison.
// sticky says that what fell below the register before this call was nonzero, which is what the remainder of a
// division into the register is. Those digits are worth less than one unit in the register's last place, so all they
// can do is turn an exact tie into a value just above one and make an otherwise exact reduction inexact.
func (a wide) quoCmpHalfPow10Sticky(k uint8, sticky bool) (q wide, above, tie, inexact bool) {
	if k == 0 {
		return a, false, false, sticky
	}

	last := min(k, MaxScale)
	for low := k - last; low > 0; {
		step := min(low, MaxScale)
		var r uint64
		a, r = a.quoRemPow10Small(step)
		sticky = sticky || r != 0
		low -= step
	}

	q, r := a.quoRemPow10Small(last)
	half := Pow10Uint64[last] / 2
	return q, r > half || (r == half && sticky), r == half && !sticky, sticky || r != 0
}

// fits192 reports whether the magnitude fits in the low three limbs, which is the width the guarded powers keep their
// running product at so that a squaring still fits the register.
func (a wide) fits192() bool {
	return a[3]|a[4]|a[5] == 0
}

// mul192 returns a * b for two magnitudes that fit 192 bits, as the 384-bit value that just fits the register.
// Schoolbook, three limbs by three: every partial accumulation is a 64-bit product plus two 64-bit addends, which is
// at most 2^128-1, so the running carry never overflows a limb.
func (a wide) mul192(b wide) wide {
	var r wide
	for i := range 3 {
		var carry uint64
		for j := range 3 {
			hi, lo := bits.Mul64(a[i], b[j])
			var c uint64
			lo, c = bits.Add64(lo, carry, 0)
			hi += c
			lo, c = bits.Add64(lo, r[i+j], 0)
			hi += c
			r[i+j] = lo
			carry = hi
		}
		r[i+3] = carry
	}
	return r
}

// bitLen returns the position of the highest set bit, or 0 for a zero magnitude.
func (a wide) bitLen() int {
	for i := wideLimbs - 1; i >= 0; i-- {
		if a[i] != 0 {
			return i*64 + bits.Len64(a[i])
		}
	}
	return 0
}

// reduceAt returns the coefficient of a * 10^-needed at exactly the scale target, rounded with mode for a result of
// sign st. overflow reports that the coefficient does not fit in 128 bits, inexact that nonzero digits were
// discarded; the caller decides what each of those means, exactly as it does for reduceWide.
func (a wide) reduceAt(
	needed uint8,
	target uint8,
	st state.State,
	mode RoundingMode,
) (q uint128.Uint128, overflow bool, inexact bool) {
	if target >= needed {
		// padding with zeros, which is exact whenever it fits at all
		coef, ok := a.uint128()
		if !ok {
			return uint128.Zero, true, false
		}
		if coef.IsZero() {
			return uint128.Zero, false, false
		}
		coef, s := coef.Mul(Pow10Uint128[target-needed])
		if s >= state.Error {
			return uint128.Zero, true, false
		}
		return coef, false, false
	}

	qw, above, tie, inexact := a.quoCmpHalfPow10(needed - target)
	q, ok := qw.uint128()
	if !ok {
		return uint128.Zero, true, false
	}

	if inexact && roundDecision(above, tie, q.Lo&1 == 1, st, mode) {
		var carry uint64
		if q, carry = q.AddCarry(uint128.One); carry != 0 {
			return uint128.Zero, true, inexact
		}
	}

	return q, false, inexact
}

// at brings the exact magnitude a, which stands at the scale needed, to exactly the scale target, rounding with mode
// for a result of sign st. It is the single rounding of the fused operations: NaN(Overflow) when the coefficient does
// not fit at target, NaN(Inexact) when mode is ROUND_NAN and a nonzero digit would be discarded.
func (a wide) at(needed, target uint8, st state.State, mode RoundingMode) Dec128 {
	q, overflow, inexact := a.reduceAt(needed, target, st, mode)
	switch {
	case overflow:
		return Dec128{state: state.Overflow}
	case inexact && mode == ROUND_NAN:
		return Dec128{state: state.Inexact}
	case q.IsZero():
		return Dec128{scale: target} // a zero is never negative
	}

	return Dec128{coef: q, scale: target, state: st}
}

// fit applies the scale rule to the exact magnitude a, which stands at the given scale: the largest scale not above
// min(scale, MaxScale) whose rounded coefficient fits in 128 bits, with the digits below it discarded under mode and
// policy. It is fitWide over the wide register, for the callers whose intermediate does not fit in 256 bits; the
// oracle in arith_crosscheck_test.go holds the two to the same rule. Callers handle the exact, in-range case
// themselves.
//
// sticky carries a loss the caller already took, so that the policy rules on the whole computation rather than on
// this last reduction: a guarded power that dropped digits while squaring is inexact even if the final reduction
// happens to drop only zeros.
func (a wide) fit(scale uint8, st state.State, mode RoundingMode, policy LossPolicy, sticky bool) Dec128 {
	if a == (wide{}) {
		rs := min(scale, MaxScale)
		if !sticky {
			return Dec128{scale: rs} // an exact zero
		}
		// Everything the value had was dropped before it reached here, so it stands below the quantum of every
		// scale this type has and only a directed mode reaches the first unit. reduceAt takes that same decision
		// for a magnitude that is merely small rather than gone, and the policy then rules on the rounded result;
		// doing it in that order here is what makes the two agree.
		var coef uint128.Uint128
		if roundDecision(false, false, false, st, mode) {
			coef = uint128.One
		}
		if s := lossState(coef.IsZero(), policy); s != state.OK {
			return Dec128{state: s}
		}
		if coef.IsZero() {
			return Dec128{scale: rs} // a zero is never negative
		}
		return Dec128{coef: coef, scale: rs, state: st}
	}

	// Digits that must go because of the scale cap, then those that must go because of the width: the quotient by
	// 10^k has at least bitLen - bitLen10[k] bits, so no smaller k can fit. The estimate is optimistic by at most one
	// digit, and the loop moves on when the reduction reports overflow.
	k := 0
	if scale > MaxScale {
		k = int(scale) - int(MaxScale)
	}
	bl := a.bitLen()
	for k < len(bitLen10) && bl-int(bitLen10[k]) > 128 {
		k++
	}

	for k <= int(scale) {
		q, overflow, inexact := a.reduceAt(scale, scale-uint8(k), st, mode)
		if overflow {
			k++
			continue
		}
		if inexact || sticky {
			if s := lossState(q.IsZero(), policy); s != state.OK {
				return Dec128{state: s}
			}
		}
		if q.IsZero() {
			st = state.Default
		}
		return Dec128{coef: q, scale: scale - uint8(k), state: st}
	}

	return Dec128{state: state.Overflow}
}

// mulAddSlow is the general case of MulAddRound: a product that is not itself representable, so the fused sum needs
// the wide register. The product stands at prodScale with sign st, and c is the addend.
func mulAddSlow(lo, hi uint128.Uint128, prodScale uint8, st state.State, c Dec128, scale uint8, mode RoundingMode) Dec128 {
	needed := max(prodScale, c.scale)

	// Neither alignment can carry out of 384 bits: the product is below 2^256 and is shifted by at most 10^MaxScale,
	// and the addend is below 2^128 and is shifted by at most 10^(2*MaxScale) < 2^127.
	p, _ := wideFrom256(lo, hi).mulPow10(needed - prodScale)
	q, _ := wideFrom128(c.coef).mulPow10(needed - c.scale)

	if st == c.state {
		// Same sign: the sum of a value below 2^320 and one below 2^255 cannot carry out either.
		sum, _ := p.add(q)
		return sum.at(needed, scale, st, mode)
	}

	// Opposite signs: the difference of the magnitudes, signed by the larger one.
	switch p.compare(q) {
	case 0:
		return Dec128{scale: scale} // a zero is never negative
	case 1:
		return p.sub(q).at(needed, scale, st, mode)
	default:
		return q.sub(p).at(needed, scale, c.state, mode)
	}
}

// quotientAt completes a division whose quotient and exact remainder are already in hand: it narrows the quotient to a
// coefficient at the given scale and takes the single rounding against the true divisor v. It is the tail shared by
// MulDivRound and MulDivRoundInt64, so that the two spellings of a fused multiply-divide cannot disagree about a
// rounding decision.
func quotientAt(q wide, r, v uint128.Uint128, scale uint8, st state.State, mode RoundingMode) Dec128 {
	coef, ok := q.uint128()
	if !ok {
		return Dec128{state: state.Overflow}
	}
	if !r.IsZero() && mode == ROUND_NAN {
		return Dec128{state: state.Inexact}
	}
	if up, _ := roundUp(coef, r, v, st, mode); up {
		var carry uint64
		if coef, carry = coef.AddCarry(uint128.One); carry != 0 {
			return Dec128{state: state.Overflow}
		}
	}
	if coef.IsZero() {
		return Dec128{scale: scale} // a zero is never negative
	}

	return Dec128{coef: coef, scale: scale, state: st}
}

// meanFromWide divides the exact magnitude m, which stands at the working scale ws with sign st, by count and brings
// the quotient to exactly the scale requested, taking one rounding decision for the division and the reduction
// together rather than one each. It is the tail shared by Accumulator.Mean and AvgRound, so that the running mean and
// the one-shot mean cannot disagree.
//
// count must be positive; the callers return NaN(DivisionByZero) for an empty set of terms before reaching here.
func meanFromWide(m wide, ws uint8, count int, st state.State, scale uint8, mode RoundingMode) Dec128 {
	// Pad before dividing when the mean is wanted at a finer scale than the terms were kept at, so that the division
	// produces those places instead of the reduction having to discard them.
	//
	// The padding cannot carry out of the register. A term is below 2^256 and there are fewer than 2^63 of them,
	// because the count is an int, so the total at working scale zero is below 2^319; a working scale of ws carries
	// a further 10^ws, and padding to scale multiplies by 10^(scale-ws), so what is held is below 2^319 * 10^scale,
	// and scale is at most MaxScale: 2^319 * 10^19 < 2^383.
	if scale > ws {
		m, _ = m.mulPow10(scale - ws)
		ws = scale
	}

	q, r := m.quoRem64(uint64(count))

	// What the rounding decision has to weigh is r/count of the register's last place, plus the ws-scale digits the
	// reduction drops. When there are no digits to drop the remainder is the whole of it and decides on its own;
	// otherwise those digits are the more significant part and the remainder is no more than a sticky bit.
	var qw wide
	var above, tie, inexact bool
	if k := ws - scale; k == 0 {
		half := uint64(count) / 2
		qw, above, tie, inexact = q, r > half, uint64(count)%2 == 0 && r == half, r != 0
	} else {
		qw, above, tie, inexact = q.quoCmpHalfPow10Sticky(k, r != 0)
	}

	coef, ok := qw.uint128()
	if !ok {
		return Dec128{state: state.Overflow}
	}
	if inexact {
		if mode == ROUND_NAN {
			return Dec128{state: state.Inexact}
		}
		if roundDecision(above, tie, coef.Lo&1 == 1, st, mode) {
			var carry uint64
			if coef, carry = coef.AddCarry(uint128.One); carry != 0 {
				return Dec128{state: state.Overflow}
			}
		}
	}
	if coef.IsZero() {
		return Dec128{scale: scale} // a zero is never negative
	}

	return Dec128{coef: coef, scale: scale, state: st}
}
