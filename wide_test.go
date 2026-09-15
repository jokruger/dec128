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
