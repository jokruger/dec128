package dec128

import (
	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// This file implements the scale rule: compute the exact result, then make it fit.
//
// An exact intermediate can be up to 256 bits wide (a product) and carry up to 38 decimal places (the sum of two
// scales). The representable form is a 128-bit coefficient with at most MaxScale places. fitWide finds the largest
// scale not above min(needed, MaxScale) at which the coefficient fits, discards the digits below it with the given
// rounding mode, and returns NaN only when the integer part itself does not fit at scale 0.

// bitLen10 holds bits.Len(10^k) for k = 0..38: the width of each power of ten, used to estimate how many digits must go
// before any division is attempted.
var bitLen10 = [39]uint8{
	1, 4, 7, 10, 14, 17, 20, 24, 27, 30,
	34, 37, 40, 44, 47, 50, 54, 57, 60, 64,
	67, 70, 74, 77, 80, 84, 87, 90, 94, 97,
	100, 103, 107, 110, 113, 117, 120, 123, 127,
}

// bitLen256 returns the bit length of the 256-bit value hi*2^128 + lo.
func bitLen256(lo, hi uint128.Uint128) int {
	if !hi.IsZero() {
		return 128 + hi.BitLen()
	}
	return lo.BitLen()
}

// signOf returns the sign state of a product or quotient of two operands with the given sign states, both of which must
// be valid (below state.Error).
func signOf(a, b state.State) state.State {
	if a == b {
		return state.Default
	}
	return state.Neg
}

// roundDecision reports whether a truncated magnitude must be incremented under mode, given that nonzero digits were
// discarded: above is "the discarded part exceeds half", tie is "it is exactly half", odd is the parity of the kept
// coefficient and st its sign.
func roundDecision(above, tie, odd bool, st state.State, mode RoundingMode) bool {
	switch mode {
	case ROUND_TOWARD_ZERO, ROUND_NAN:
		return false
	case ROUND_DOWN:
		return st == state.Neg
	case ROUND_UP:
		return st != state.Neg
	case ROUND_AWAY_FROM_ZERO:
		return true
	case ROUND_HALF_TOWARD_ZERO:
		return above
	case ROUND_HALF_AWAY_FROM_ZERO:
		return above || tie
	default: // ROUND_BANK
		return above || (tie && odd)
	}
}

// roundUp reports whether a truncated quotient q with remainder r (0 <= r < divisor) must be incremented under mode for
// a result with sign st, and whether the discarded digits were nonzero. It never doubles the remainder: the half-way
// test compares r with divisor/2, and an odd divisor cannot produce an exact tie.
func roundUp(q, r, divisor uint128.Uint128, st state.State, mode RoundingMode) (up bool, inexact bool) {
	if mode == ROUND_TOWARD_ZERO {
		return false, !r.IsZero() // the default: truncate, nothing to decide
	}
	if r.IsZero() {
		return false, false
	}
	// The half-way comparison is only needed by the half modes; the directed modes decide on the sign alone.
	if mode < ROUND_HALF_TOWARD_ZERO {
		return roundDecision(false, false, false, st, mode), true
	}
	c := r.Compare(divisor.Rsh(1))
	return roundDecision(c > 0, c == 0 && divisor.Lo&1 == 0, q.Lo&1 == 1, st, mode), true
}

// compare256 compares two 256-bit values given as (lo, hi) pairs.
func compare256(aLo, aHi, bLo, bHi uint128.Uint128) int {
	if c := aHi.Compare(bHi); c != 0 {
		return c
	}
	return aLo.Compare(bLo)
}

// reduceWide divides the 256-bit value (lo, hi) by 10^k and rounds the quotient with mode. overflow reports that the
// rounded quotient does not fit in 128 bits; inexact reports that nonzero digits were discarded (the caller decides
// whether that is an error, as it is under ROUND_NAN). Requires 1 <= k <= 38.
func reduceWide(lo, hi uint128.Uint128, k uint8, st state.State, mode RoundingMode) (q uint128.Uint128, overflow bool, inexact bool) {
	divisor := Pow10Uint128[k]
	var r uint128.Uint128
	if k <= MaxScale {
		// one-limb divisor: four reciprocal steps instead of a hardware division
		var r64 uint64
		var ok bool
		if q, r64, ok = uint128.QuoRem256ByPow10(lo, hi, k); !ok {
			return uint128.Zero, true, false
		}
		r = uint128.Uint128{Lo: r64}
	} else {
		var s state.State
		if q, r, s = uint128.QuoRem256By128(lo, hi, divisor); s >= state.Error {
			return uint128.Zero, true, false
		}
	}

	up, inexact := roundUp(q, r, divisor, st, mode)
	if up {
		var carry uint64
		q, carry = q.AddCarry(uint128.One)
		if carry != 0 {
			return uint128.Zero, true, inexact
		}
	}

	return q, false, inexact
}

// fitWide returns the Dec128 nearest to (lo, hi) * 10^-scale, signed by st, at the largest scale not above
// min(scale, MaxScale) whose coefficient fits in 128 bits. Callers handle the exact, in-range case
// (hi == 0 and scale <= MaxScale) themselves. Digits below that scale are discarded with mode. The result is
// NaN(Overflow) when the integer part does not fit even at scale 0, and NaN(Inexact) when mode is ROUND_NAN and a
// nonzero digit would be discarded. A zero result is never negative.
func fitWide(lo, hi uint128.Uint128, scale uint8, st state.State, mode RoundingMode) Dec128 {
	// Digits that must go because of the scale cap.
	k := 0
	if scale > MaxScale {
		k = int(scale) - int(MaxScale)
	}

	// Digits that must go because of the width: the quotient by 10^k has at least bitLen - bitLen10[k] bits, so no
	// smaller k can fit. The estimate is optimistic by at most one digit; the loop below moves on when the division
	// reports overflow.
	bl := bitLen256(lo, hi)
	for k < len(bitLen10) && bl-int(bitLen10[k]) > 128 {
		k++
	}

	if k == 0 {
		// unreachable with hi == 0: every caller returns an exact in-range result itself before calling fitWide, so a
		// k of 0 here means the optimistic estimate landed on 0 for a value just above 2^128 and at least one digit
		// must go.
		//if hi.IsZero() {
		//	if lo.IsZero() {
		//		st = state.Default
		//	}
		//	return Dec128{coef: lo, scale: scale, state: st}
		//}
		k = 1
	}

	for k <= int(scale) {
		q, overflow, inexact := reduceWide(lo, hi, uint8(k), st, mode)
		if overflow {
			k++
			continue
		}
		if inexact && mode == ROUND_NAN {
			return Dec128{state: state.Inexact}
		}
		if q.IsZero() {
			st = state.Default
		}
		return Dec128{coef: q, scale: scale - uint8(k), state: st}
	}

	return Dec128{state: state.Overflow}
}

// stripZeros removes trailing zeros from an exact result, lowering its scale from scale down to, but not below, ideal.
// This is the GDA "ideal exponent" rule for the results of inexact operations: 1/2 is 0.5 and 1.00/2 is 0.50,
// while 1/3 keeps every digit it was given. Divisibility is tested in chunks of up to eight digits so that a quotient
// computed at scale 19 needs only a few divisions to come back to a short form.
func stripZeros(q uint128.Uint128, scale, ideal uint8) (uint128.Uint128, uint8) {
	for scale > ideal {
		k := min(scale-ideal, 8)
		for {
			q2, r, _ := q.QuoRemPow10(k)
			if r == 0 {
				q, scale = q2, scale-k
				break
			}
			if k == 1 {
				return q, scale
			}
			k /= 2
		}
	}
	return q, scale
}
