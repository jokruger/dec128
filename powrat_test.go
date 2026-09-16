package dec128

import (
	"math"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// Oracles for PowRational. The exact value of d^(p/q) is irrational, so the oracle cannot compute it and compare;
// it verifies the definition instead. R is the correctly rounded coefficient exactly when, with
// W = |d|^p * 10^(s*q) taken as an exact rational, floor(V) is the largest k with k^q <= W and the half-way test
// (2k+1)^q vs 2^q * W picks R from k and k+1 under the mode.
//
// The floor is found by bisection rather than by Newton, so the oracle shares no algorithm with the implementation.

// oracleFloorRoot returns the largest k >= 0 with k^q * den <= num, by bisection over the 129-bit range a valid
// coefficient can reach, or nil when the value is out of that range.
func oracleFloorRoot(num, den *big.Int, q uint64) *big.Int {
	qb := big.NewInt(int64(q))
	le := func(k *big.Int) bool {
		t := new(big.Int).Exp(k, qb, nil)
		return t.Mul(t, den).Cmp(num) <= 0
	}

	hi := new(big.Int).Lsh(big.NewInt(1), 129)
	if le(hi) {
		return nil // beyond anything a coefficient can hold; the caller expects an overflow
	}
	lo := big.NewInt(0)
	for new(big.Int).Sub(hi, lo).Cmp(big.NewInt(1)) > 0 {
		mid := new(big.Int).Add(lo, hi)
		mid.Rsh(mid, 1)
		if le(mid) {
			lo = mid
		} else {
			hi = mid
		}
	}
	return lo
}

// checkPowRational verifies a PowRational result against the definition.
func checkPowRational(t *testing.T, tag string, got Dec128, d Dec128, p, q int64, s uint8, mode RoundingMode) {
	t.Helper()

	// W = |d|^p * 10^(s*q) as num/den
	w := new(big.Rat).SetFrac(bigAtScale(d, d.scale), bigPow10(d.scale))
	if w.Sign() < 0 {
		w.Neg(w)
	}
	pow := new(big.Rat).SetInt64(1)
	for range int(math.Abs(float64(p))) {
		pow.Mul(pow, w)
	}
	if p < 0 {
		pow.Inv(pow)
	}
	pow.Mul(pow, new(big.Rat).SetInt(bigPow10(s)))
	for range int(q) - 1 {
		pow.Mul(pow, new(big.Rat).SetInt(bigPow10(s)))
	}

	k := oracleFloorRoot(pow.Num(), pow.Denom(), uint64(q))
	if k == nil {
		if got.ErrorDetails() != state.Overflow.Error() {
			t.Fatalf("%s: got %s, want NaN(Overflow)", tag, got.StringFixed())
		}
		return
	}

	qb := big.NewInt(q)
	exact := new(big.Int).Exp(k, qb, nil)
	exact.Mul(exact, pow.Denom())
	isExact := exact.Cmp(pow.Num()) == 0

	if !isExact && mode == ROUND_NAN {
		if got.ErrorDetails() != state.Inexact.Error() {
			t.Fatalf("%s: got %s, want NaN(Inexact)", tag, got.ErrorDetails())
		}
		return
	}

	want := new(big.Int).Set(k)
	if !isExact {
		// above half: (2k+1)^q * den < 2^q * num
		lhs := new(big.Int).Lsh(k, 1)
		lhs.Add(lhs, big.NewInt(1))
		lhs.Exp(lhs, qb, nil)
		lhs.Mul(lhs, pow.Denom())
		rhs := new(big.Int).Lsh(pow.Num(), uint(q))
		c := lhs.Cmp(rhs)

		neg := d.state == state.Neg && p%2 != 0
		st := state.Default
		if neg {
			st = state.Neg
		}
		if roundDecision(c < 0, c == 0, k.Bit(0) == 1, st, mode) {
			want.Add(want, big.NewInt(1))
		}
	}

	if want.Cmp(big2p128) >= 0 {
		if got.ErrorDetails() != state.Overflow.Error() {
			t.Fatalf("%s: coefficient %s does not fit, want NaN(Overflow), got %s", tag, want, got.StringFixed())
		}
		return
	}
	if got.IsNaN() {
		t.Fatalf("%s: got NaN(%v), want %s at scale %d", tag, got.ErrorDetails(), want, s)
	}

	wantNeg := d.state == state.Neg && p%2 != 0 && want.Sign() != 0
	switch {
	case got.scale != s:
		t.Fatalf("%s: scale = %d, want %d", tag, got.scale, s)
	case bigOfUint128(got.coef).Cmp(want) != 0:
		t.Fatalf("%s: coefficient = %s, want %s", tag, bigOfUint128(got.coef), want)
	case (got.state == state.Neg) != wantNeg:
		t.Fatalf("%s: sign = %v, want negative = %v", tag, got.state, wantNeg)
	}
}

// TestPowRationalAgainstDefinition is the main oracle. The exponents are kept small so that the bisection, which
// raises a 129-bit number to the q-th power at every step, stays cheap.
func TestPowRationalAgainstDefinition(t *testing.T) {
	r := rand.New(rand.NewSource(20260923))
	var overflows, exacts, negatives int

	for i := range 3000 {
		d := randPowBase(r)
		if d.coef.IsZero() || d.IsNaN() {
			continue
		}
		p := int64(1 + r.Intn(6))
		if r.Intn(3) == 0 {
			p = -p
		}
		q := int64(1 + r.Intn(6))
		pa := p
		if pa < 0 {
			pa = -pa
		}
		if gcdUint64(uint64(pa), uint64(q)) != 1 {
			continue // the oracle works on the reduced fraction, as the implementation does
		}
		if d.state == state.Neg && q%2 == 0 {
			continue
		}
		s := uint8(r.Intn(int(MaxScale) + 1))
		mode := allModes[r.Intn(len(allModes))]

		got := d.PowRational(p, q, s, mode)
		checkPowRational(t, "PowRational", got, d, p, q, s, mode)

		switch {
		case got.IsNaN():
			overflows++
		case d.state == state.Neg:
			negatives++
		}
		if got := d.PowRational(p, q, s, ROUND_NAN); !got.IsNaN() {
			exacts++
		}
		_ = i
	}

	if overflows == 0 || exacts == 0 || negatives == 0 {
		t.Errorf("operand mix no longer reaches the edges (nan=%d exact=%d negative=%d)", overflows, exacts, negatives)
	}
}

// TestPowRationalMatchesNthRoot pins the two spellings of the same operation together: NthRootRound is PowRational
// with a numerator of one, and a degree of one is a rescale in both.
func TestPowRationalMatchesNthRoot(t *testing.T) {
	r := rand.New(rand.NewSource(20260924))
	for range 4000 {
		d := randPowBase(r)
		n := 1 + r.Intn(40)
		s := uint8(r.Intn(int(MaxScale) + 1))
		mode := allModes[r.Intn(len(allModes))]

		if got, want := d.PowRational(1, int64(n), s, mode), d.NthRootRound(n, s, mode); got != want {
			t.Fatalf("PowRational(1, %d) = %s (%v), NthRootRound(%d) = %s (%v)",
				n, got.StringFixed(), got.state, n, want.StringFixed(), want.state)
		}
	}

	// SqrtRound is the same value at n == 2, which is the claim NthRootRound's documentation makes.
	for range 2000 {
		d := randPowBase(r).Abs()
		s := uint8(r.Intn(int(MaxScale) + 1))
		mode := allModes[r.Intn(len(allModes))]
		if got, want := d.PowRational(1, 2, s, mode), d.SqrtRound(s, mode); got != want {
			t.Fatalf("PowRational(1, 2) = %s, SqrtRound = %s for %s", got.StringFixed(), want.StringFixed(), d.StringFixed())
		}
	}
}

// TestPowRationalMatchesPowInt checks q == 1 against PowIntRound wherever that one is exact: there the faithful
// result and the correctly rounded one are the same value, so the two must agree.
func TestPowRationalMatchesPowInt(t *testing.T) {
	r := rand.New(rand.NewSource(20260925))
	var compared int

	for range 4000 {
		d := randPowBase(r)
		n := int64(1 + r.Intn(24))
		if r.Intn(3) == 0 {
			n = -n
		}
		s := uint8(r.Intn(int(MaxScale) + 1))

		// ROUND_NAN returns a value only when the power is exact at that scale
		if d.PowIntRound(n, s, ROUND_NAN).IsNaN() {
			continue
		}
		got, want := d.PowRational(n, 1, s, ROUND_HALF_AWAY_FROM_ZERO), d.PowIntRound(n, s, ROUND_HALF_AWAY_FROM_ZERO)
		if got != want {
			t.Fatalf("PowRational(%d, 1) = %s, PowIntRound(%d) = %s for %s",
				n, got.StringFixed(), n, want.StringFixed(), d.StringFixed())
		}
		compared++
	}

	if compared == 0 {
		t.Error("no exact powers in the corpus; the comparison never ran")
	}
}

// TestPowRationalReduces pins the reduction, which is what makes an even-looking denominator work on a negative base
// and what keeps a fraction written large from being refused by the bound.
func TestPowRationalReduces(t *testing.T) {
	two := FromInt64(2)
	half := two.PowRational(1, 2, MaxScale, ROUND_HALF_AWAY_FROM_ZERO)

	if got := two.PowRational(2, 4, MaxScale, ROUND_HALF_AWAY_FROM_ZERO); got != half {
		t.Errorf("2^(2/4) = %s, 2^(1/2) = %s", got.StringFixed(), half.StringFixed())
	}
	if got := two.PowRational(8192, 16384, MaxScale, ROUND_HALF_AWAY_FROM_ZERO); got != half {
		t.Errorf("2^(8192/16384) = %s, 2^(1/2) = %s", got.StringFixed(), half.StringFixed())
	}

	// A negative base: the reduced denominator is the whole of the answer. 2/4 reduces to 1/2, an even denominator
	// and so a domain error; 2/6 reduces to 1/3, so the result is the cube root itself and not the sixth root of the
	// square - -2, not 2. That is the convention reduction implies, and it is worth a test because the other reading
	// is the one a reader expects.
	negEight := FromInt64(-8)
	if got := negEight.PowRational(2, 4, 2, ROUND_BANK); got.ErrorDetails() != state.DomainError.Error() {
		t.Errorf("(-8)^(2/4) = %v, want NaN(DomainError)", got.ErrorDetails())
	}
	if got := negEight.PowRational(2, 6, 2, ROUND_BANK); !got.Equal(FromInt64(-2)) {
		t.Errorf("(-8)^(2/6) = %s, want -2 (2/6 reduces to 1/3)", got.StringFixed())
	}
	if got := negEight.PowRational(1, 3, 2, ROUND_BANK); !got.Equal(FromInt64(-2)) {
		t.Errorf("(-8)^(1/3) = %s, want -2", got.StringFixed())
	}

	// A fraction that reduces to one is a rescale, sign and all.
	if got := FromString("-1.5").PowRational(3, 3, 3, ROUND_BANK); !got.Equal(FromString("-1.5")) || got.scale != 3 {
		t.Errorf("(-1.5)^(3/3) = %s at scale %d, want -1.500", got.StringFixed(), got.scale)
	}
}

// TestPowRationalCapRefuses pins the bound: at maxRootDegree the operation runs, past it it is refused, and the
// reduction happens first so that a large fraction with a small value is still accepted.
func TestPowRationalCapRefuses(t *testing.T) {
	d := FromString("1.0001")

	if got := d.PowRational(1, maxRootDegree, MaxScale, ROUND_BANK); got.IsNaN() {
		t.Errorf("q == maxRootDegree must be computed, got NaN(%v)", got.ErrorDetails())
	}
	if got := d.PowRational(maxRootDegree, 1, MaxScale, ROUND_BANK); got.IsNaN() {
		t.Errorf("p == maxRootDegree must be computed, got NaN(%v)", got.ErrorDetails())
	}

	for _, c := range []struct {
		name string
		p, q int64
	}{
		{"q past the bound", 1, maxRootDegree + 1},
		{"p past the bound", maxRootDegree + 1, 1},
		{"both past the bound", maxRootDegree + 1, maxRootDegree + 3},
		{"negative p past the bound", -(maxRootDegree + 1), 1},
		{"int64 extremes", math.MaxInt64, 1},
		{"MinInt64", math.MinInt64, 2},
		{"q zero", 1, 0},
		{"q negative", 1, -2},
	} {
		if got := d.PowRational(c.p, c.q, 2, ROUND_BANK); got.ErrorDetails() != state.DomainError.Error() {
			t.Errorf("%s: got %v, want NaN(DomainError)", c.name, got.ErrorDetails())
		}
	}

	// NthRootRound shares the bound.
	if got := d.NthRootRound(maxRootDegree+1, 2, ROUND_BANK); got.ErrorDetails() != state.DomainError.Error() {
		t.Errorf("NthRootRound past the bound: got %v, want NaN(DomainError)", got.ErrorDetails())
	}
}

// TestNthRootRoundWiderDegrees covers the degrees the old bound of 1024 refused and this one allows, against the
// definition. These are the whole-schedule conversions: a factor over 30 years of daily rests is a 10950th root.
func TestNthRootRoundWiderDegrees(t *testing.T) {
	d := FromString("1.5")
	for _, n := range []int{1025, 1560, 3650, 10950, 14600, maxRootDegree} {
		got := d.NthRootRound(n, MaxScale, ROUND_HALF_AWAY_FROM_ZERO)
		if got.IsNaN() {
			t.Fatalf("NthRootRound(%d) = NaN(%v), want a value", n, got.ErrorDetails())
		}
		// The root of a value above one is above one and below it, which brackets every one of these.
		if got.Compare(One) <= 0 || got.Compare(d) >= 0 {
			t.Fatalf("NthRootRound(%d) = %s, which is not between 1 and 1.5", n, got.StringFixed())
		}
		// Raising it back gets close to the radicand: within a relative 1e-15 is ample for the check.
		back := got.PowIntRound(int64(n), MaxScale, ROUND_HALF_AWAY_FROM_ZERO)
		if back.IsNaN() {
			t.Fatalf("raising the %dth root back overflowed", n)
		}
		diff := back.Sub(d).Abs()
		if diff.Compare(FromString("0.000000000000001")) > 0 {
			t.Fatalf("NthRootRound(%d) raised back gives %s, want 1.5", n, back.StringFixed())
		}
	}
}

// TestPowRationalCost holds the bound to the budget the constant's comment records: the worst coprime pair at
// maxRootDegree, on the widest operand, taken to the finest scale.
func TestPowRationalCost(t *testing.T) {
	if testing.Short() {
		t.Skip("the cost measurement is the slowest test in the suite")
	}
	d := Dec128{coef: uint128.Max, scale: MaxScale}

	start := time.Now()
	got := d.PowRational(maxRootDegree-1, maxRootDegree, MaxScale, ROUND_HALF_AWAY_FROM_ZERO)
	elapsed := time.Since(start)

	if got.IsNaN() && got.ErrorDetails() != state.Overflow.Error() {
		t.Fatalf("got NaN(%v)", got.ErrorDetails())
	}
	// The measured figure is ~240ms; the budget is loose enough for a slower machine and tight enough that a
	// regression in the bound or in intNthRoot shows up rather than merely making the suite slow.
	if elapsed > 5*time.Second {
		t.Errorf("the worst pair at maxRootDegree took %v, which is beyond the budget the constant documents", elapsed)
	}
	t.Logf("p=%d q=%d took %v", maxRootDegree-1, maxRootDegree, elapsed)
}

// TestPowRationalShortCircuitAgrees is the guard on the float64 magnitude estimate: every case it decides is re-run
// through the exact path and must give the same value. A float64 may choose a shortcut here; it may never choose a
// returned value, because that is what would make the result depend on the architecture.
func TestPowRationalShortCircuitAgrees(t *testing.T) {
	r := rand.New(rand.NewSource(20260926))
	var shortcuts int

	for range 4000 {
		d := randPowBase(r)
		if d.coef.IsZero() || d.IsNaN() {
			continue
		}
		pm := uint64(1 + r.Intn(60))
		qm := uint64(1 + r.Intn(8))
		if g := gcdUint64(pm, qm); g != 1 {
			pm, qm = pm/g, qm/g
		}
		negExp := r.Intn(2) == 0
		s := uint8(r.Intn(int(MaxScale) + 1))
		mode := allModes[r.Intn(len(allModes))]
		if d.state == state.Neg && qm%2 == 0 {
			continue
		}
		st := state.Default
		if d.state == state.Neg && pm%2 == 1 {
			st = state.Neg
		}

		withShortcut := powRatAt(d.coef, d.scale, negExp, pm, qm, s, st, mode)
		exact := powRatExact(d.coef, d.scale, negExp, pm, qm, s, st, mode)
		if withShortcut != exact {
			t.Fatalf("shortcut changed the result for %s^(%v%d/%d) at scale %d under %v: %s (%v) vs %s (%v)",
				d.StringFixed(), map[bool]string{true: "-", false: ""}[negExp], pm, qm, s, mode,
				withShortcut.StringFixed(), withShortcut.state, exact.StringFixed(), exact.state)
		}

		e10 := float64(pm) / float64(qm)
		if negExp {
			e10 = -e10
		}
		if lg := e10*(log10Coef(d.coef)-float64(d.scale)) + float64(s); lg > 40 || lg < -2 {
			shortcuts++
		}
	}

	if shortcuts == 0 {
		t.Error("the corpus never reached the shortcut; the comparison was vacuous")
	}
}

// TestPowRationalContract pins the argument validation and the special values.
func TestPowRationalContract(t *testing.T) {
	two := FromInt64(2)

	cases := []struct {
		name string
		got  Dec128
		want state.State
	}{
		{"NaN propagates", NaN(state.Underflow).PowRational(1, 2, 2, ROUND_BANK), state.Underflow},
		{"scale above MaxScale", two.PowRational(1, 2, MaxScale+1, ROUND_BANK), state.ScaleOutOfRange},
		{"undefined mode", two.PowRational(1, 2, 2, RoundingMode(99)), state.InvalidRoundingMode},
		{"zero to a negative power", Zero.PowRational(-1, 2, 2, ROUND_BANK), state.DivisionByZero},
		{"even root of a negative", FromInt64(-4).PowRational(1, 2, 2, ROUND_BANK), state.DomainError},
		{"inexact under ROUND_NAN", two.PowRational(1, 2, 2, ROUND_NAN), state.Inexact},
		{"overflow", FromInt64(10).PowRational(64, 1, 0, ROUND_BANK), state.Overflow},
	}
	for _, c := range cases {
		if c.got.ErrorDetails() != c.want.Error() {
			t.Errorf("%s: got %v, want %v", c.name, c.got.ErrorDetails(), c.want.Error())
		}
	}

	// p == 0 is one at the requested scale for every base, including zero and including one whose powers overflow.
	for _, base := range []Dec128{two, Zero, FromInt64(-3), MaxAtScale(0)} {
		if got := base.PowRational(0, 7, 3, ROUND_BANK); !got.Equal(One) || got.scale != 3 {
			t.Errorf("%s^0 = %s at scale %d, want 1.000", base.StringFixed(), got.StringFixed(), got.scale)
		}
	}

	// A zero base with a positive power is zero at the requested scale and is never negative.
	if got := Zero.PowRational(3, 2, 4, ROUND_BANK); got.IsNaN() || !got.IsZero() || got.scale != 4 || got.state == state.Neg {
		t.Errorf("0^(3/2) = %s at scale %d state %v, want 0.0000", got.StringFixed(), got.scale, got.state)
	}

	// Exact cases, where the correct rounding is the exact value in every mode.
	for _, mode := range allModes {
		if got := FromInt64(8).PowRational(2, 3, 0, mode); !got.Equal(FromInt64(4)) {
			t.Errorf("8^(2/3) under %v = %s, want 4", mode, got.StringFixed())
		}
		if got := FromInt64(4).PowRational(3, 2, 0, mode); !got.Equal(FromInt64(8)) {
			t.Errorf("4^(3/2) under %v = %s, want 8", mode, got.StringFixed())
		}
		// a negative exponent: 4^(-3/2) is 1/8, which is exact at three places
		if got := FromInt64(4).PowRational(-3, 2, 3, mode); !got.Equal(FromString("0.125")) {
			t.Errorf("4^(-3/2) under %v = %s, want 0.125", mode, got.StringFixed())
		}
	}
}

// TestPowRationalCarriesOutOfTheCoefficient pins the branch a random corpus does not reach: a floor that fills all
// 128 bits and a rounding that pushes it past them. Unlike a root, a rational power can land there, which is why the
// increment is checked at all.
//
// The operands are built backwards from the answer. With p/q = 2/3, a scale of 15 and three decimal places in the
// base, the radicand is coef^2 * 10^39, so a coefficient whose square puts that between (2^128-1)^3 and 2^128 cubed
// gives a floor of exactly the largest coefficient there is, with something left over.
func TestPowRationalCarriesOutOfTheCoefficient(t *testing.T) {
	d := FromString("198499385884174662381231575360242982.828")
	if d.IsNaN() || d.scale != 3 {
		t.Fatalf("the constructed base does not parse: %v at scale %d", d.ErrorDetails(), d.scale)
	}

	// Truncating keeps the largest coefficient, which is what makes the case a carry and not an ordinary overflow.
	if got := d.PowRational(2, 3, 15, ROUND_TOWARD_ZERO); !got.Equal(MaxAtScale(15)) {
		t.Fatalf("truncated: got %s, want %s", got.StringFixed(), MaxAtScale(15).StringFixed())
	}
	// Rounding away from zero carries past it.
	if got := d.PowRational(2, 3, 15, ROUND_AWAY_FROM_ZERO); got.ErrorDetails() != state.Overflow.Error() {
		t.Fatalf("rounded away: got %s (%v), want NaN(Overflow)", got.StringFixed(), got.ErrorDetails())
	}
	// ROUND_NAN sees an inexact result rather than an overflow, because the floor does fit.
	if got := d.PowRational(2, 3, 15, ROUND_NAN); got.ErrorDetails() != state.Inexact.Error() {
		t.Fatalf("ROUND_NAN: got %s (%v), want NaN(Inexact)", got.StringFixed(), got.ErrorDetails())
	}
}
