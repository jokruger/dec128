package dec128

import (
	"math/big"

	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// The math/big bridge: the way in and the way out for values that do not fit, or did not come from, a Dec128.
//
// A Dec128 is exactly a rational - the sign, the coefficient and 10 raised to the scale, with nothing approximated -
// so the conversion to big.Rat is lossless in one direction and, for any rational whose denominator divides a power of
// ten within range, lossless in the other. That is the whole justification for these three: not that math/big is
// faster or better, but that the identity is exact and the package should say so rather than leave every caller to
// rediscover it.
//
// The conversion a caller would otherwise write is four lines, and the third is a trap:
//
//	n := d.Coefficient().BigInt() // the magnitude, and only the magnitude
//	if d.IsNegative() {           // forget this and the value is silently |d|
//		n.Neg(n)
//	}
//	r := new(big.Rat).SetFrac(n, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(d.Scale())), nil))
//
// Coefficient returns a uint128, so the sign cannot travel with it; every other extraction in the package carries the
// sign for the caller, and these do too.
//
// These are the only methods here that hand back a pointer. A Dec128 is 24 immutable comparable bytes and everything
// else keeps to that; a big.Rat is a mutable value behind a pointer, and the one returned is always freshly allocated
// so that a caller mutating it cannot reach back into anything. They allocate, and they read no process-global
// configuration.

// Rat returns the exact value of d as a big.Rat.
//
// Nothing is rounded or approximated: the result is the sign, the coefficient and 10 raised to the scale, in lowest
// terms because that is what big.Rat.SetFrac leaves. It is the conversion to reach for when a calculation needs more
// than 39 significant digits, when a value has to cross into another decimal library, or when a test wants an
// independent oracle - big.Rat is what this package's own cross-checks are written against.
//
// A NaN has no rational value and returns the error its state carries. Zero returns a zero Rat whatever the scale,
// because a Rat has no scale to carry: 0.00 and 0 both come back as 0/1, and the scale is lost. That is the one thing
// this conversion does not preserve, and Scale is there for a caller who needs it.
func (d Dec128) Rat() (*big.Rat, error) {
	if d.state >= state.Error {
		return nil, d.state.Error()
	}

	n := d.coef.BigInt()
	if d.state == state.Neg {
		n.Neg(n)
	}

	return new(big.Rat).SetFrac(n, pow10big(int(d.scale))), nil
}

// BigInt returns the integer part of d as a big.Int, truncated toward zero.
//
// It is the integer part and not the coefficient: BigInt of 1.75 is 1, and of -1.75 is -1. The coefficient is
// Coefficient, which returns a uint128 and has its own BigInt method when a big one is wanted.
//
// This is the only exit for the integer part of a large value. Int64 covers the range to 9223372036854775807 and
// returns state.Overflow above it, while a Dec128 reaches 340282366920938463463374607431768211455, so for a value
// past the int64 range the alternative is to format it and parse it back.
//
// A NaN returns the error its state carries.
func (d Dec128) BigInt() (*big.Int, error) {
	if d.state >= state.Error {
		return nil, d.state.Error()
	}

	n := d.coef.BigInt()
	if d.scale != 0 {
		n.Quo(n, pow10big(int(d.scale)))
	}
	if d.state == state.Neg {
		n.Neg(n)
	}

	return n, nil
}

// FromRat returns r at exactly the given scale, rounding with mode.
//
// It is the way back in, and the reason it takes a scale and a mode rather than guessing is that most rationals are
// not decimals: 1/3 has no exact form at any scale, and the caller has to say where to stop and which way to go. A
// rational that is exactly representable - 1/4 at scale 2, or anything whose denominator divides 10^scale - comes back
// exact, and under ROUND_NAN anything else is NaN(Inexact).
//
// The value of taking the rational rather than a formatted string is that it rounds once. A caller who accumulates in
// big.Rat and converts through a string rounds at the formatting and again at the parse; this makes the single
// decision the rest of the package makes.
//
// A nil r is NaN(InvalidFormat), a scale above MaxScale NaN(ScaleOutOfRange), an undefined mode
// NaN(InvalidRoundingMode), and a value too large for the coefficient NaN(Overflow). A zero denominator cannot occur:
// big.Rat does not represent one.
//
// It allocates, and it reads no process-global configuration.
func FromRat(r *big.Rat, scale uint8, mode RoundingMode) Dec128 {
	switch {
	case r == nil:
		return Dec128{state: state.InvalidFormat}
	case scale > MaxScale:
		return Dec128{state: state.ScaleOutOfRange}
	case !mode.IsValid():
		return Dec128{state: state.InvalidRoundingMode}
	}

	st := state.Default
	if r.Sign() < 0 {
		st = state.Neg
	}

	// The coefficient is |num| * 10^scale / den, and the decision is made on the remainder of that one division.
	num := new(big.Int).Abs(r.Num())
	num.Mul(num, pow10big(int(scale)))
	den := r.Denom()
	q, rem := new(big.Int).QuoRem(num, den, new(big.Int))

	// Overflow before Inexact, as every other reduction in the package orders them.
	coef, s := uint128.FromBigInt(q)
	if s >= state.Error {
		return Dec128{state: state.Overflow}
	}

	if rem.Sign() != 0 {
		if mode == ROUND_NAN {
			return Dec128{state: state.Inexact}
		}
		twice := new(big.Int).Lsh(rem, 1)
		c := twice.Cmp(den)
		if roundDecision(c > 0, c == 0, q.Bit(0) == 1, st, mode) {
			if coef, s = uint128.FromBigInt(q.Add(q, bigOne)); s >= state.Error {
				return Dec128{state: state.Overflow}
			}
		}
	}

	if coef.IsZero() {
		return Dec128{scale: scale} // a zero is never negative
	}

	return Dec128{coef: coef, scale: scale, state: st}
}

// bigOne is the increment a rounding decision applies, kept rather than allocated per call.
var bigOne = big.NewInt(1)
