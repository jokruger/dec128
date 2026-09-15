package dec128

import (
	"math"
	"math/big"
	"math/rand"
	"strconv"
	"testing"

	"github.com/jokruger/dec128/state"
)

// NthRootRound has two oracles, and between them they leave nothing to a hand-computed expectation.
//
// The first is SqrtRound, which is a different implementation of the same function at n == 2 - a Newton square root
// over a fixed 256-bit register against an exact integer root in math/big - so agreeing with it over random operands
// in every mode is a strong check on the rounding decision, ties included.
//
// The second is the post-condition itself: R is the correctly rounded root at a scale iff R is the integer nearest
// (under the mode) to the real root of the radicand, which is decided exactly by comparing R^n and (2R+1)^n with the
// radicand in math/big. That is checkable for every n without computing a root at all.

// rootPostCondition asserts that got is the correctly rounded n-th root of d at the given scale under mode. It
// rebuilds the comparison from the definition rather than from the implementation: with the radicand written as the
// fraction a/b, R is the root iff R^n*b <= a < (R+1)^n*b, and the rounding decision is the one roundBig would take
// on the remainder of that interval.
func rootPostCondition(t *testing.T, d Dec128, n int, scale uint8, mode RoundingMode, got Dec128) {
	t.Helper()

	// The exponent reaches n*scale, far past what bigPow10 takes, so the power is built here.
	pow10 := func(e int) *big.Int {
		return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(e)), nil)
	}
	a, b := new(big.Int).Abs(bigAtScale(d, d.scale)), big.NewInt(1)
	if e := n*int(scale) - int(d.scale); e >= 0 {
		a.Mul(a, pow10(e))
	} else {
		b = pow10(-e)
	}

	// the largest R with R^n*b <= a, by bisection over the 128-bit range
	nb := big.NewInt(int64(n))
	fits := func(x *big.Int) bool {
		v := new(big.Int).Exp(x, nb, nil)
		v.Mul(v, b)
		return v.Cmp(a) <= 0
	}
	lo, hi := new(big.Int), new(big.Int).Set(big2p128)
	for new(big.Int).Sub(hi, lo).Cmp(big.NewInt(1)) > 0 {
		mid := new(big.Int).Add(lo, hi)
		mid.Rsh(mid, 1)
		if fits(mid) {
			lo.Set(mid)
		} else {
			hi.Set(mid)
		}
	}
	trunc := lo

	exact := new(big.Int).Exp(trunc, nb, nil)
	exact.Mul(exact, b)
	inexact := exact.Cmp(a) != 0

	want := new(big.Int).Set(trunc)
	if inexact {
		if mode == ROUND_NAN {
			if got.state != state.Inexact {
				t.Fatalf("NthRootRound(%s, %d, %d, ROUND_NAN) = %v, want NaN(Inexact)", d.StringFixed(), n, scale, got)
			}
			return
		}
		// above half: (2R+1)^n * b < 2^n * a
		lhs := new(big.Int).Lsh(trunc, 1)
		lhs.Add(lhs, big.NewInt(1))
		lhs.Exp(lhs, nb, nil)
		lhs.Mul(lhs, b)
		rhs := new(big.Int).Lsh(a, uint(n))
		c := lhs.Cmp(rhs)
		// roundBig decides on a remainder against a divisor; here the same decision is expressed directly
		up := false
		switch mode {
		case ROUND_TOWARD_ZERO, ROUND_NAN:
		case ROUND_DOWN:
			up = d.IsNegative()
		case ROUND_UP:
			up = !d.IsNegative()
		case ROUND_AWAY_FROM_ZERO:
			up = true
		case ROUND_HALF_TOWARD_ZERO:
			up = c < 0
		case ROUND_HALF_AWAY_FROM_ZERO:
			up = c <= 0
		case ROUND_BANK:
			up = c < 0 || (c == 0 && trunc.Bit(0) == 1)
		}
		if up {
			want.Add(want, big.NewInt(1))
		}
	}

	switch {
	case want.Cmp(big2p128) >= 0:
		if got.state != state.Overflow {
			t.Fatalf("NthRootRound(%s, %d, %d, %s) = %v, want NaN(Overflow)", d.StringFixed(), n, scale, mode, got)
		}
	case got.IsNaN():
		t.Fatalf("NthRootRound(%s, %d, %d, %s) = NaN(%v), want %s", d.StringFixed(), n, scale, mode, got.ErrorDetails(), want)
	case got.scale != scale:
		t.Fatalf("NthRootRound(%s, %d, %d, %s) = %s at scale %d", d.StringFixed(), n, scale, mode, got.StringFixed(), got.scale)
	case new(big.Int).Abs(bigAtScale(got, scale)).Cmp(want) != 0:
		t.Fatalf("NthRootRound(%s, %d, %d, %s) = %s, want coefficient %s",
			d.StringFixed(), n, scale, mode, got.StringFixed(), want)
	case (got.state == state.Neg) != (d.IsNegative() && want.Sign() != 0):
		t.Fatalf("NthRootRound(%s, %d, %d, %s) = %s: wrong sign", d.StringFixed(), n, scale, mode, got.StringFixed())
	}
}

// At n == 2 the two implementations must agree bit for bit, which is the check that covers the rounding decision and
// the tie cases most thoroughly, because SqrtRound reaches them by a different route.
func TestNthRootRoundAgreesWithSqrt(t *testing.T) {
	r := rand.New(rand.NewSource(20261010))
	for range 40000 {
		d := randDec(r).Abs()
		scale := uint8(r.Intn(int(MaxScale) + 1))
		mode := allModes[r.Intn(len(allModes))]
		if got, want := d.NthRootRound(2, scale, mode), d.SqrtRound(scale, mode); got != want {
			t.Fatalf("NthRootRound(%s, 2, %d, %s) = %v, SqrtRound = %v", d.StringFixed(), scale, mode, got, want)
		}
	}
}

func TestNthRootRoundPostCondition(t *testing.T) {
	r := rand.New(rand.NewSource(20261011))
	for range 6000 {
		d := randDec(r)
		n := 1 + r.Intn(8)
		if d.IsNegative() && n%2 == 0 {
			d = d.Abs()
		}
		scale := uint8(r.Intn(int(MaxScale) + 1))
		mode := allModes[r.Intn(len(allModes))]
		got := d.NthRootRound(n, scale, mode)
		if n == 1 {
			if want := d.RescaleRound(scale, mode); got != want {
				t.Fatalf("NthRootRound(%s, 1, %d, %s) = %v, RescaleRound = %v", d.StringFixed(), scale, mode, got, want)
			}
			continue
		}
		if d.coef.IsZero() {
			if !got.IsZero() || got.scale != scale {
				t.Fatalf("root of zero = %v at scale %d", got, got.scale)
			}
			continue
		}
		rootPostCondition(t, d, n, scale, mode, got)
	}
}

// The degrees a schedule actually uses, over the bases it uses them on.
func TestNthRootRoundRateConversion(t *testing.T) {
	for _, n := range []int{2, 3, 4, 12, 52, 360, 365, maxRootDegree} {
		for _, s := range []string{"1.05", "1.126825", "0.98", "2", "1.0000000000000000001"} {
			d := FromString(s)
			for _, scale := range []uint8{2, 10, MaxScale} {
				got := d.NthRootRound(n, scale, ROUND_HALF_AWAY_FROM_ZERO)
				rootPostCondition(t, d, n, scale, ROUND_HALF_AWAY_FROM_ZERO, got)

				// and the round trip: raising the root back to the power n returns to the neighbourhood of d
				if !got.IsNaN() && !got.IsZero() {
					back := got.PowIntRound(int64(n), scale, ROUND_HALF_AWAY_FROM_ZERO)
					if !back.IsNaN() {
						want := d.RescaleRound(scale, ROUND_HALF_AWAY_FROM_ZERO)
						// n roundings of a quantum each is the most the round trip can drift
						tol := QuantumAtScale(scale).MulInt(n + 1)
						if diff := back.SubRound(want, scale, ROUND_TOWARD_ZERO).Abs(); diff.Compare(tol) > 0 {
							t.Errorf("(%s^(1/%d))^%d at scale %d = %s, want about %s (off by %s)",
								s, n, n, scale, back.StringFixed(), want.StringFixed(), diff.StringFixed())
						}
					}
				}
			}
		}
	}
}

func TestNthRootRoundEdges(t *testing.T) {
	for _, c := range []struct {
		in    string
		n     int
		scale uint8
		mode  RoundingMode
		want  string
	}{
		{"8", 3, 0, ROUND_BANK, "2"},
		{"8", 3, 2, ROUND_BANK, "2.00"},
		{"-8", 3, 0, ROUND_BANK, "-2"},
		{"-8", 5, 4, ROUND_BANK, "-1.5157"},
		{"0.008", 3, 1, ROUND_BANK, "0.2"},
		{"1000", 3, 2, ROUND_BANK, "10.00"},
		{"2", 2, MaxScale, ROUND_BANK, "1.4142135623730950488"},
		{"1", 12, 4, ROUND_BANK, "1.0000"},
		{"0", 5, 2, ROUND_BANK, "0.00"},
		{"-0.00", 5, 2, ROUND_BANK, "0.00"},
		// a tie, which a root can genuinely produce: the square root of 0.25 is 0.5 exactly
		{"0.25", 2, 0, ROUND_BANK, "0"},
		{"0.25", 2, 0, ROUND_HALF_AWAY_FROM_ZERO, "1"},
		{"0.25", 2, 0, ROUND_HALF_TOWARD_ZERO, "0"},
		{"0.25", 2, 0, ROUND_UP, "1"},
		{"0.25", 2, 0, ROUND_TOWARD_ZERO, "0"},
		{"0.25", 2, 1, ROUND_BANK, "0.5"},
		// the largest coefficient, whose root is still comfortably inside one
		{"340282366920938463463374607431768211455", 2, 0, ROUND_TOWARD_ZERO, "18446744073709551615"},
	} {
		got := FromString(c.in).NthRootRound(c.n, c.scale, c.mode)
		if got.IsNaN() || got.StringFixed() != c.want {
			t.Errorf("NthRootRound(%s, %d, %d, %s) = %v (%v), want %s",
				c.in, c.n, c.scale, c.mode, got, got.ErrorDetails(), c.want)
		}
	}

	// an exact root is exact under ROUND_NAN, and an inexact one is refused
	if got := FromString("8").NthRootRound(3, 0, ROUND_NAN); got.StringFixed() != "2" {
		t.Errorf("exact root under ROUND_NAN = %v", got)
	}
	if got := FromString("9").NthRootRound(3, 4, ROUND_NAN); got.state != state.Inexact {
		t.Errorf("inexact root under ROUND_NAN = %v", got)
	}
}

func TestNthRootRoundArgumentErrors(t *testing.T) {
	d := FromString("8")
	for _, c := range []struct {
		what  string
		got   Dec128
		state state.State
	}{
		{"n zero", d.NthRootRound(0, 2, ROUND_BANK), state.DomainError},
		{"n negative", d.NthRootRound(-3, 2, ROUND_BANK), state.DomainError},
		{"n above the cap", d.NthRootRound(maxRootDegree+1, 2, ROUND_BANK), state.DomainError},
		{"n at MaxInt", d.NthRootRound(math.MaxInt, 2, ROUND_BANK), state.DomainError},
		{"n at MinInt", d.NthRootRound(math.MinInt, 2, ROUND_BANK), state.DomainError},
		{"even root of a negative", FromString("-8").NthRootRound(2, 2, ROUND_BANK), state.DomainError},
		{"even root of a negative, high n", FromString("-8").NthRootRound(360, 2, ROUND_BANK), state.DomainError},
		{"scale", d.NthRootRound(2, MaxScale+1, ROUND_BANK), state.ScaleOutOfRange},
		{"mode", d.NthRootRound(2, 2, RoundingMode(99)), state.InvalidRoundingMode},
		{"NaN operand", NaN(state.NotConverged).NthRootRound(2, 2, ROUND_BANK), state.NotConverged},
		// n == 1 is a rescale, so it is the one degree that can overflow
		{"n one, overflow", MaxAtScale(0).NthRootRound(1, 1, ROUND_BANK), state.Overflow},
	} {
		if c.got.state != c.state {
			t.Errorf("%s = %v, want NaN(%s)", c.what, c.got, c.state)
		}
	}

	// the argument checks come before the domain check, so a NaN operand wins over a bad n
	if got := NaN(state.Overflow).NthRootRound(0, 2, ROUND_BANK); got.state != state.Overflow {
		t.Errorf("NaN with a bad n = %v", got)
	}
}

// intNthRoot is the integer core, and its post-condition is checkable on its own: x^n <= v < (x+1)^n.
func TestIntNthRoot(t *testing.T) {
	r := rand.New(rand.NewSource(20261012))
	for range 4000 {
		v := new(big.Int)
		switch r.Intn(4) {
		case 0:
			v.SetUint64(uint64(r.Intn(1000)))
		case 1:
			v.SetUint64(r.Uint64())
		case 2:
			v.SetBytes([]byte{byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256))})
			v.Lsh(v, uint(r.Intn(400)))
		default:
			// an exact power, so that the boundary itself is hit
			v.Exp(big.NewInt(int64(2+r.Intn(30))), big.NewInt(int64(2+r.Intn(20))), nil)
		}
		n := 2 + r.Intn(40)

		x := intNthRoot(v, n)
		nb := big.NewInt(int64(n))
		if lo := new(big.Int).Exp(x, nb, nil); lo.Cmp(v) > 0 {
			t.Fatalf("intNthRoot(%s, %d) = %s is too high", v, n, x)
		}
		up := new(big.Int).Add(x, big.NewInt(1))
		if hi := up.Exp(up, nb, nil); hi.Cmp(v) <= 0 {
			t.Fatalf("intNthRoot(%s, %d) = %s is too low", v, n, x)
		}
	}

	if got := intNthRoot(new(big.Int), 3); got.Sign() != 0 {
		t.Errorf("intNthRoot(0, 3) = %s", got)
	}
	// v < 2^n, where the root is one without iterating
	if got := intNthRoot(big.NewInt(5), 8); got.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("intNthRoot(5, 8) = %s, want 1", got)
	}
}

// The descent cannot stop above the root, so the only correction is upward, and it is what makes the result
// independent of the starting value. Here it is driven from starts that are deliberately wrong, so that the climb is
// exercised rather than assumed.
//
// Note how asymmetric the two sides are: a start above the root descends geometrically and costs nothing, while one
// below it climbs a step at a time, so a start below the root of a wide value would take 2^100 steps. That is the
// whole reason nthRootGuess adds a margin on the high side, and it is why the below-root starts here are used only
// on values whose roots are small.
func TestIntNthRootFromAnyStart(t *testing.T) {
	r := rand.New(rand.NewSource(20261029))
	one := big.NewInt(1)

	for range 3000 {
		v := new(big.Int).SetUint64(r.Uint64())
		v.Lsh(v, uint(r.Intn(200)))
		if v.Sign() == 0 {
			continue
		}
		n := 2 + r.Intn(12)
		want := intNthRoot(v, n)

		for _, x := range []*big.Int{
			new(big.Int).Set(want),                   // exactly the answer
			new(big.Int).Add(want, one),              // one above, the landing the correction exists for
			new(big.Int).Add(want, big.NewInt(1000)), // far above
			new(big.Int).Lsh(want, 4),                // sixteen times the root
			new(big.Int).Lsh(want, 20),               // a million times it
			bitLenGuess(v, n),                        // the coarse fallback
			nthRootGuess(v, n),
		} {
			got := intNthRootFrom(v, n, new(big.Int).Set(x))
			if got.Cmp(want) != 0 {
				t.Fatalf("intNthRootFrom(%s, %d, start %s) = %s, want %s", v, n, x, got, want)
			}
		}
	}

	// Starts at and below the root, over values whose roots are small enough for the climb to be quick. A zero
	// start has no reciprocal to iterate on at all, so the whole answer comes from the correction.
	for range 3000 {
		v := big.NewInt(int64(1 + r.Intn(1000000)))
		n := 2 + r.Intn(10)
		want := intNthRoot(v, n)
		for _, x := range []*big.Int{
			new(big.Int),
			big.NewInt(1),
			big.NewInt(2),
			new(big.Int).Rsh(want, 1),
			new(big.Int).Sub(want, one),
		} {
			if x.Sign() < 0 {
				continue
			}
			got := intNthRootFrom(v, n, new(big.Int).Set(x))
			if got.Cmp(want) != 0 {
				t.Fatalf("intNthRootFrom(%s, %d, start %s) = %s, want %s", v, n, x, got, want)
			}
		}
	}

	for _, c := range []struct{ v, n, want int64 }{
		{1000, 3, 10},
		{1023, 10, 1},
		{1024, 10, 2},
		{9765625, 10, 5},
		{1, 5, 1},
		{2, 2, 1},
	} {
		got := intNthRootFrom(big.NewInt(c.v), int(c.n), new(big.Int))
		if got.Cmp(big.NewInt(c.want)) != 0 {
			t.Errorf("intNthRootFrom(%d, %d, start 0) = %s, want %d", c.v, c.n, got, c.want)
		}
	}
}

// The coarse guess is the fallback nthRootGuess uses when the float64 logarithms cannot be trusted, so it has to be
// above the root for every input on its own account.
func TestNthRootGuesses(t *testing.T) {
	r := rand.New(rand.NewSource(20261030))
	for range 20000 {
		v := new(big.Int).SetUint64(r.Uint64() | 1)
		v.Lsh(v, uint(r.Intn(400)))
		n := 2 + r.Intn(60)

		root := intNthRoot(v, n)
		for _, g := range []*big.Int{nthRootGuess(v, n), bitLenGuess(v, n)} {
			if g.Cmp(root) < 0 {
				t.Fatalf("a guess of %s is below the root %s of %s ** (1/%d)", g, root, v, n)
			}
		}
		// and the real guess is close, which is what keeps the iteration short
		ratio := new(big.Float).Quo(new(big.Float).SetInt(nthRootGuess(v, n)), new(big.Float).SetInt(root))
		if f, _ := ratio.Float64(); f > 1.01 && root.Cmp(big.NewInt(1000)) > 0 {
			t.Fatalf("the guess for %s ** (1/%d) is %.4f times the root", v, n, f)
		}
	}
}

// The bound the narrowing in NthRootRound relies on: for every n >= 2 the rounded root fits a coefficient, so the
// conversion out of math/big cannot overflow. The worst case is the square root of the largest value at the finest
// scale, rounded away from zero.
func TestNthRootFitsACoefficient(t *testing.T) {
	worst := new(big.Int)
	note := ""
	for _, d := range []Dec128{
		MaxAtScale(0), MaxAtScale(MaxScale), MaxAtScale(1), MaxAtScale(9),
		FromString("340282366920938463463374607431768211455"),
		FromString("34028236692093846346.3374607431768211455"),
	} {
		for n := 2; n <= 8; n++ {
			for _, scale := range []uint8{0, 2, 10, MaxScale} {
				got := d.NthRootRound(n, scale, ROUND_AWAY_FROM_ZERO)
				if got.IsNaN() {
					t.Fatalf("%s ** (1/%d) at scale %d = NaN(%v), but every root of n >= 2 must fit",
						d.StringFixed(), n, scale, got.ErrorDetails())
				}
				if c := got.coef.BigInt(); c.Cmp(worst) > 0 {
					worst.Set(c)
					note = d.StringFixed() + " ** (1/" + strconv.Itoa(n) + ") at scale " + strconv.Itoa(int(scale))
				}
			}
		}
	}
	if worst.Cmp(big2p128) >= 0 {
		t.Fatalf("the largest root coefficient %s reached 2^128", worst)
	}
	t.Logf("the largest root coefficient is %s, from %s; 2^128 is %s", worst, note, big2p128)

	// and over random values, where the same bound must hold
	r := rand.New(rand.NewSource(20261031))
	for range 40000 {
		d := randDec(r).Abs()
		if d.coef.IsZero() {
			continue
		}
		n := 2 + r.Intn(6)
		if got := d.NthRootRound(n, MaxScale, ROUND_AWAY_FROM_ZERO); got.IsNaN() {
			t.Fatalf("%s ** (1/%d) at scale %d = NaN(%v)", d.StringFixed(), n, MaxScale, got.ErrorDetails())
		}
	}
}
