package dec128

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// The 384-bit register underneath MulAddRound and Accumulator, checked limb by limb against math/big. The decimal
// layer above it is checked separately; this file is only about the integer arithmetic.

var big2p384 = new(big.Int).Lsh(big.NewInt(1), 384)

func (a wide) big() *big.Int {
	r := new(big.Int)
	for i := wideLimbs - 1; i >= 0; i-- {
		r.Lsh(r, 64)
		r.Or(r, new(big.Int).SetUint64(a[i]))
	}
	return r
}

// randWide returns a magnitude with a random number of live limbs, so that the narrow cases appear as often as the
// wide ones.
func randWide(r *rand.Rand) wide {
	var a wide
	live := r.Intn(wideLimbs + 1)
	for i := range live {
		a[i] = r.Uint64()
	}
	if live > 0 && r.Intn(4) == 0 {
		a[live-1] >>= uint(r.Intn(64))
	}
	return a
}

func TestWideArithmeticAgainstBig(t *testing.T) {
	r := rand.New(rand.NewSource(20260917))
	for range 20000 {
		a, b := randWide(r), randWide(r)
		ba, bb := a.big(), b.big()

		sum, over := a.add(b)
		want := new(big.Int).Add(ba, bb)
		if over != (want.Cmp(big2p384) >= 0) {
			t.Fatalf("add(%s, %s): overflow %v", ba, bb, over)
		}
		if got := sum.big(); got.Cmp(new(big.Int).Mod(want, big2p384)) != 0 {
			t.Fatalf("add(%s, %s) = %s", ba, bb, got)
		}

		if c := a.compare(b); c != ba.Cmp(bb) {
			t.Fatalf("compare(%s, %s) = %d", ba, bb, c)
		}
		if ba.Cmp(bb) >= 0 {
			if got := a.sub(b).big(); got.Cmp(new(big.Int).Sub(ba, bb)) != 0 {
				t.Fatalf("sub(%s, %s) = %s", ba, bb, got)
			}
		}

		m := r.Uint64()
		prod, over := a.mul64(m)
		want = new(big.Int).Mul(ba, new(big.Int).SetUint64(m))
		if over != (want.Cmp(big2p384) >= 0) {
			t.Fatalf("mul64(%s, %d): overflow %v", ba, m, over)
		}
		if got := prod.big(); got.Cmp(new(big.Int).Mod(want, big2p384)) != 0 {
			t.Fatalf("mul64(%s, %d) = %s", ba, m, got)
		}

		coef, ok := a.uint128()
		if ok != (ba.Cmp(big2p128) < 0) {
			t.Fatalf("uint128(%s): ok %v", ba, ok)
		}
		if ok && coef.BigInt().Cmp(ba) != 0 {
			t.Fatalf("uint128(%s) = %s", ba, coef.BigInt())
		}
	}
}

func TestWidePow10AgainstBig(t *testing.T) {
	r := rand.New(rand.NewSource(20260918))
	for range 20000 {
		a := randWide(r)
		ba := a.big()
		k := uint8(r.Intn(4*int(MaxScale) + 1))

		q, above, tie, inexact := a.quoCmpHalfPow10(k)
		wantQ, wantR := new(big.Int).QuoRem(ba, bigPow10(k), new(big.Int))
		twice := new(big.Int).Lsh(wantR, 1)
		if q.big().Cmp(wantQ) != 0 {
			t.Fatalf("quoCmpHalfPow10(%s, %d) = %s, want %s", ba, k, q.big(), wantQ)
		}
		if above != (twice.Cmp(bigPow10(k)) > 0) || tie != (twice.Cmp(bigPow10(k)) == 0 && k > 0) || inexact != (wantR.Sign() != 0) {
			t.Fatalf("quoCmpHalfPow10(%s, %d): above %v tie %v inexact %v, remainder %s of %s",
				ba, k, above, tie, inexact, wantR, bigPow10(k))
		}

		// mulPow10 is two multiplications above 10^MaxScale, so on overflow it returns whatever the first step
		// produced: the value is only defined when it did not carry out.
		// mulPow10 serves the alignment of operands, so it goes no wider than two scales
		mk := uint8(r.Intn(2*int(MaxScale) + 1))
		prod, over := a.mulPow10(mk)
		want := new(big.Int).Mul(ba, bigPow10(mk))
		if over != (want.Cmp(big2p384) >= 0) {
			t.Fatalf("mulPow10(%s, %d): overflow %v", ba, mk, over)
		}
		if got := prod.big(); !over && got.Cmp(want) != 0 {
			t.Fatalf("mulPow10(%s, %d) = %s, want %s", ba, mk, got, want)
		}
	}
}

func TestWideEdges(t *testing.T) {
	maxWide := wide{^uint64(0), ^uint64(0), ^uint64(0), ^uint64(0), ^uint64(0), ^uint64(0)}

	// the two multiplication steps of mulPow10 each have to report their own carry out
	if _, over := maxWide.mulPow10(MaxScale + 1); !over {
		t.Error("mulPow10 must overflow on the first step")
	}
	if _, over := (wide{0, 0, 0, 0, 0, 1}).mulPow10(MaxScale + 1); !over {
		t.Error("mulPow10 must overflow on the second step")
	}
	if got, over := maxWide.mulPow10(0); over || got != maxWide {
		t.Error("mulPow10 by 10^0 must be the identity")
	}
	if got, above, tie, inexact := maxWide.quoCmpHalfPow10(0); got != maxWide || above || tie || inexact {
		t.Error("quoCmpHalfPow10 by 10^0 must be the identity")
	}

	// conversions round-trip
	u := uint128.Uint128{Lo: 7, Hi: 11}
	if got, ok := wideFrom128(u).uint128(); !ok || got != u {
		t.Errorf("wideFrom128 round trip = %v", got)
	}
	if got := wideFrom256(u, uint128.Uint128{Lo: 3}); got != (wide{7, 11, 3, 0, 0, 0}) {
		t.Errorf("wideFrom256 = %v", got)
	}
}

// The three ways wide.at can fail on the padding and carry paths. MulAddRound cannot reach all of them - it returns
// early on an exact cancellation - so they are exercised on the register directly.
func TestWideAtEdges(t *testing.T) {
	// a zero total pads to any scale and is never negative
	if got := (wide{}).at(0, 2, state.Neg, ROUND_BANK); got.state != state.Default || got.scale != 2 || !got.IsZero() {
		t.Errorf("zero at scale 2 = %v, state %s, scale %d", got, got.state, got.scale)
	}

	// a coefficient that fits but does not survive being padded
	if got := wideFrom128(uint128.Max).at(0, 1, state.Default, ROUND_BANK); got.state != state.Overflow {
		t.Errorf("padding overflow = %v, want NaN(Overflow)", got)
	}

	// rounding up carries out of 128 bits: (2^128-1)*10 + 5 at one place, rounded to none
	a, _ := wideFrom128(uint128.Max).mul64(10)
	a, _ = a.add(wideFrom128(uint128.Uint128{Lo: 5}))
	if got := a.at(1, 0, state.Default, ROUND_HALF_AWAY_FROM_ZERO); got.state != state.Overflow {
		t.Errorf("carry overflow = %v, want NaN(Overflow)", got)
	}
	// ... and truncating the same value does not
	if got := a.at(1, 0, state.Default, ROUND_TOWARD_ZERO); got.IsNaN() || got.coef != uint128.Max {
		t.Errorf("truncated = %v", got)
	}
}

// wide.fit claims to be fitWide over the wider register, so it is held to the same reference implementation of the
// scale rule - at scales above MaxScale as well, which no current caller reaches but the function is written for.
func TestWideFitAgainstOracle(t *testing.T) {
	r := rand.New(rand.NewSource(20260924))
	policies := []LossPolicy{LossRound, LossNaNOnUnderflow, LossNaNOnInexact}
	for range 20000 {
		a := randWide(r)
		scale := uint8(r.Intn(2*int(MaxScale) + 1))
		mode := allModes[r.Intn(len(allModes))]
		policy := policies[r.Intn(len(policies))]

		st := state.Default
		exact := a.big()
		if r.Intn(2) == 0 && exact.Sign() != 0 {
			st = state.Neg
			exact = new(big.Int).Neg(exact)
		}

		got := a.fit(scale, st, mode, policy, false)
		wantCoef, wantScale, wantNaN := fitOracle(exact, scale, mode, policy)
		if wantNaN != state.OK {
			if got.state != wantNaN {
				t.Fatalf("fit(%s, %d, %s, %s) = %v, want NaN(%s)", exact, scale, mode, policy, got, wantNaN)
			}
			continue
		}
		if got.IsNaN() {
			t.Fatalf("fit(%s, %d, %s, %s) = NaN(%v), want %s at scale %d", exact, scale, mode, policy, got.ErrorDetails(), wantCoef, wantScale)
		}
		if got.scale != wantScale || got.coef.BigInt().Cmp(wantCoef) != 0 {
			t.Fatalf("fit(%s, %d, %s, %s) = %s at scale %d, want %s at scale %d",
				exact, scale, mode, policy, got.coef.BigInt(), got.scale, wantCoef, wantScale)
		}
		if wantNeg := exact.Sign() < 0 && wantCoef.Sign() != 0; (got.state == state.Neg) != wantNeg {
			t.Fatalf("fit(%s, %d, %s, %s) = %v: wrong sign", exact, scale, mode, policy, got)
		}
	}

	if (wide{}).bitLen() != 0 {
		t.Error("a zero magnitude has no bits")
	}
}

// A register that is exactly zero with the sticky bit set is the case of a value whose every digit was dropped before
// it arrived - a guarded power that underflowed its working scale. It is not the same as an exact zero: a directed
// mode has to reach the first unit, and the loss policy rules on the result that decision produced, which is how the
// merely-tiny case behaves on the way through reduceAt.
func TestWideFitZeroWithSticky(t *testing.T) {
	for _, c := range []struct {
		mode  RoundingMode
		st    state.State
		coef  uint64
		state state.State
	}{
		{ROUND_UP, state.Default, 1, state.Default},
		{ROUND_UP, state.Neg, 0, state.Default},
		{ROUND_DOWN, state.Default, 0, state.Default},
		{ROUND_DOWN, state.Neg, 1, state.Neg},
		{ROUND_AWAY_FROM_ZERO, state.Default, 1, state.Default},
		{ROUND_AWAY_FROM_ZERO, state.Neg, 1, state.Neg},
		{ROUND_TOWARD_ZERO, state.Default, 0, state.Default},
		{ROUND_TOWARD_ZERO, state.Neg, 0, state.Default},
		{ROUND_HALF_AWAY_FROM_ZERO, state.Default, 0, state.Default},
		{ROUND_HALF_TOWARD_ZERO, state.Default, 0, state.Default},
		{ROUND_BANK, state.Default, 0, state.Default},
		{ROUND_NAN, state.Default, 0, state.Default},
	} {
		got := (wide{}).fit(MaxScale, c.st, c.mode, LossRound, true)
		if got.IsNaN() || got.coef.Lo != c.coef || got.scale != MaxScale || got.state != c.state {
			t.Errorf("zero with sticky under %s (%s) = %v at scale %d state %s, want coefficient %d state %s",
				c.mode, c.st, got, got.scale, got.state, c.coef, c.state)
		}
	}

	// The scale is capped like any other result, and an exact zero is unaffected by the mode.
	if got := (wide{}).fit(2*MaxScale, state.Default, ROUND_UP, LossRound, true); got.scale != MaxScale || got.coef.Lo != 1 {
		t.Errorf("zero with sticky above MaxScale = %v at scale %d", got, got.scale)
	}
	for _, mode := range allModes {
		if got := (wide{}).fit(4, state.Neg, mode, LossRound, false); !got.IsZero() || got.scale != 4 {
			t.Errorf("an exact zero under %s = %v at scale %d", mode, got, got.scale)
		}
	}

	// The policy sees the rounded result: a direction that reached the first unit did not lose everything, one that
	// stayed at zero did, and LossNaNOnInexact refuses either way because a digit was dropped.
	for _, c := range []struct {
		mode   RoundingMode
		policy LossPolicy
		want   state.State
	}{
		{ROUND_UP, LossNaNOnUnderflow, state.OK},
		{ROUND_TOWARD_ZERO, LossNaNOnUnderflow, state.Underflow},
		{ROUND_BANK, LossNaNOnUnderflow, state.Underflow},
		{ROUND_UP, LossNaNOnInexact, state.Inexact},
		{ROUND_TOWARD_ZERO, LossNaNOnInexact, state.Inexact},
	} {
		got := (wide{}).fit(MaxScale, state.Default, c.mode, c.policy, true)
		if c.want == state.OK {
			if got.IsNaN() {
				t.Errorf("zero with sticky under %s/%s = NaN(%v), want a value", c.mode, c.policy, got.ErrorDetails())
			}
			continue
		}
		if got.state != c.want {
			t.Errorf("zero with sticky under %s/%s = %v, want NaN(%s)", c.mode, c.policy, got, c.want)
		}
	}

	// An exact zero is never refused by a policy, because nothing was lost to refuse.
	for _, policy := range []LossPolicy{LossRound, LossNaNOnUnderflow, LossNaNOnInexact} {
		if got := (wide{}).fit(2, state.Default, ROUND_BANK, policy, false); got.IsNaN() {
			t.Errorf("an exact zero under %s = NaN(%v)", policy, got.ErrorDetails())
		}
	}
}

// quoRem64 divides the register by a word, which is what an accumulator's mean needs.
func TestWideQuoRem64AgainstBig(t *testing.T) {
	r := rand.New(rand.NewSource(20261019))
	for range 40000 {
		a := randWide(r)
		var v uint64
		switch r.Intn(3) {
		case 0:
			v = uint64(1 + r.Intn(1000))
		case 1:
			v = r.Uint64() | 1
		default:
			v = 1 << uint(r.Intn(64))
		}

		q, rem := a.quoRem64(v)
		wantQ, wantR := new(big.Int).QuoRem(a.big(), new(big.Int).SetUint64(v), new(big.Int))
		if q.big().Cmp(wantQ) != 0 || rem != wantR.Uint64() {
			t.Fatalf("%s / %d = %s rem %d, want %s rem %s", a.big(), v, q.big(), rem, wantQ, wantR)
		}
	}

	// dividing by one is the identity, and a zero register stays zero
	a := randWide(r)
	if q, rem := a.quoRem64(1); q != a || rem != 0 {
		t.Errorf("division by one changed the value")
	}
	if q, rem := (wide{}).quoRem64(7); q != (wide{}) || rem != 0 {
		t.Errorf("zero / 7 = %v rem %d", q.big(), rem)
	}
}

// The sticky form of the reduction must agree with the plain one when nothing was dropped, and must turn an exact tie
// into a value above it when something was.
func TestWideQuoCmpHalfPow10Sticky(t *testing.T) {
	r := rand.New(rand.NewSource(20261020))
	for range 20000 {
		a := randWide(r)
		k := uint8(r.Intn(2*int(MaxScale) + 1))

		q0, above0, tie0, inexact0 := a.quoCmpHalfPow10(k)
		q1, above1, tie1, inexact1 := a.quoCmpHalfPow10Sticky(k, false)
		if q0 != q1 || above0 != above1 || tie0 != tie1 || inexact0 != inexact1 {
			t.Fatalf("the sticky form disagrees with the plain one at k=%d", k)
		}

		q2, above2, tie2, inexact2 := a.quoCmpHalfPow10Sticky(k, true)
		if q2 != q0 {
			t.Fatalf("a sticky bit changed the quotient at k=%d", k)
		}
		if !inexact2 {
			t.Fatalf("a sticky bit must make the reduction inexact at k=%d", k)
		}
		// a tie becomes "above half", and nothing else moves
		if tie0 {
			if tie2 || !above2 {
				t.Fatalf("a tie with a sticky bit is above half at k=%d: above %v tie %v", k, above2, tie2)
			}
		} else if above2 != above0 || tie2 != tie0 {
			t.Fatalf("a sticky bit moved a non-tie at k=%d", k)
		}
	}

	// at k == 0 the sticky bit is the whole of what was dropped: below half, and inexact
	a := randWide(r)
	q, above, tie, inexact := a.quoCmpHalfPow10Sticky(0, true)
	if q != a || above || tie || !inexact {
		t.Errorf("k=0 with a sticky bit: q changed %v, above %v, tie %v, inexact %v", q != a, above, tie, inexact)
	}
}
