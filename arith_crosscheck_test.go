package dec128

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/jokruger/dec128/state"
)

// Oracle for the additive operations and comparison: math/big over the exact values.
// A Dec128 is coef * 10^-scale with a sign; the oracle works at the common scale.

var (
	bigTen   = big.NewInt(10)
	big2p128 = new(big.Int).Lsh(big.NewInt(1), 128)
)

func bigPow10(n uint8) *big.Int {
	return new(big.Int).Exp(bigTen, big.NewInt(int64(n)), nil)
}

// bigAtScale returns the signed integer coefficient of d expressed at the given scale
// (which must not be lower than d's own).
func bigAtScale(d Dec128, scale uint8) *big.Int {
	c := new(big.Int).SetUint64(d.coef.Hi)
	c.Lsh(c, 64)
	c.Or(c, new(big.Int).SetUint64(d.coef.Lo))
	c.Mul(c, bigPow10(scale-d.scale))
	if d.state == state.Neg {
		c.Neg(c)
	}
	return c
}

func randDec(r *rand.Rand) Dec128 {
	var d Dec128
	switch r.Intn(5) {
	case 0:
		d.coef.Lo = uint64(r.Intn(100000))
	case 1:
		d.coef.Lo = r.Uint64()
	case 2:
		d.coef.Lo, d.coef.Hi = r.Uint64(), r.Uint64()>>uint(r.Intn(64))
	case 3:
		d.coef.Lo, d.coef.Hi = r.Uint64(), r.Uint64()
	default:
		// exact powers of ten and their neighbors, so that carries across limbs and trailing zeros both appear
		d.coef = Pow10Uint128[r.Intn(39)]
		if r.Intn(2) == 0 && !d.coef.IsZero() {
			d.coef.Lo -= 1
		}
	}
	d.scale = uint8(r.Intn(int(MaxScale) + 1))
	if r.Intn(2) == 0 && !d.coef.IsZero() {
		d.state = state.Neg
	}
	return d
}

func TestAddSubCompareAgainstBig(t *testing.T) {
	r := rand.New(rand.NewSource(20260907))
	for range 50000 {
		x, y := randDec(r), randDec(r)
		scale := max(x.scale, y.scale)
		bx, by := bigAtScale(x, scale), bigAtScale(y, scale)

		checkFit(t, "Add", x, y, x.Add(y), new(big.Int).Add(bx, by), scale, ROUND_TOWARD_ZERO)
		checkFit(t, "Sub", x, y, x.Sub(y), new(big.Int).Sub(bx, by), scale, ROUND_TOWARD_ZERO)

		if got, want := x.Compare(y), bx.Cmp(by); got != want {
			t.Errorf("Compare(%s, %s) = %d, want %d", x.StringFixed(), y.StringFixed(), got, want)
		}
		if got, want := x.Equal(y), bx.Cmp(by) == 0; got != want {
			t.Errorf("Equal(%s, %s) = %v, want %v", x.StringFixed(), y.StringFixed(), got, want)
		}
		if !x.Equal(x) || x.Compare(x) != 0 {
			t.Errorf("%s must equal itself", x.StringFixed())
		}
	}
}

func TestAddSubIdentities(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	for range 20000 {
		x, y := randDec(r), randDec(r)
		s := x.Add(y)
		if s.IsNaN() {
			continue
		}
		// (x + y) - y == x and x + y == y + x, numerically, whenever the sum was exact
		// (a reduced sum has lost digits and the way back is only approximate).
		if s.scale == max(x.scale, y.scale) {
			if back := s.Sub(y); !back.Equal(x) {
				t.Errorf("(%s + %s) - %s = %s, want %s", x.StringFixed(), y.StringFixed(), y.StringFixed(), back.StringFixed(), x.StringFixed())
			}
		}
		if !s.Equal(y.Add(x)) {
			t.Errorf("%s + %s is not commutative", x.StringFixed(), y.StringFixed())
		}
		if !x.Sub(y).Equal(y.Sub(x).Neg()) {
			t.Errorf("%s - %s != -(%s - %s)", x.StringFixed(), y.StringFixed(), y.StringFixed(), x.StringFixed())
		}
	}
}

// roundBig applies mode to a truncated quotient q with remainder r (0 <= r < divisor),
// for a result of the given sign, in math/big arithmetic.
func roundBig(q, r, divisor *big.Int, neg bool, mode RoundingMode) *big.Int {
	if r.Sign() == 0 {
		return q
	}
	twoR := new(big.Int).Lsh(r, 1)
	c := twoR.Cmp(divisor)
	up := false
	switch mode {
	case ROUND_TOWARD_ZERO, ROUND_NAN:
	case ROUND_DOWN:
		up = neg
	case ROUND_UP:
		up = !neg
	case ROUND_AWAY_FROM_ZERO:
		up = true
	case ROUND_HALF_TOWARD_ZERO:
		up = c > 0
	case ROUND_HALF_AWAY_FROM_ZERO:
		up = c >= 0
	case ROUND_BANK:
		up = c > 0 || (c == 0 && q.Bit(0) == 1)
	}
	if up {
		return new(big.Int).Add(q, big.NewInt(1))
	}
	return q
}

// fitOracle is the reference implementation of the scale rule for an exact signed
// coefficient at the given scale: the largest scale not above min(scale, MaxScale) at
// which the rounded coefficient fits 128 bits. It returns the expected coefficient and
// scale, or a NaN state. mode picks the direction of the rounding, policy decides
// whether the loss is allowed at all.
func fitOracle(exact *big.Int, scale uint8, mode RoundingMode, policy LossPolicy) (coef *big.Int, resScale uint8, nan state.State) {
	neg := exact.Sign() < 0
	abs := new(big.Int).Abs(exact)
	for t := int(min(scale, MaxScale)); t >= 0; t-- {
		k := uint8(int(scale) - t)
		d := bigPow10(k)
		q, r := new(big.Int).QuoRem(abs, d, new(big.Int))
		q = roundBig(q, r, d, neg, mode)
		if q.Cmp(big2p128) < 0 {
			if r.Sign() != 0 {
				switch policy {
				case LossNaNOnInexact:
					return nil, 0, state.Inexact
				case LossNaNOnUnderflow:
					if q.Sign() == 0 {
						return nil, 0, state.Underflow
					}
				}
			}
			return q, uint8(t), state.OK
		}
	}
	return nil, 0, state.Overflow
}

func checkFit(t *testing.T, op string, x, y, got Dec128, exact *big.Int, scale uint8, mode RoundingMode) {
	t.Helper()
	wantCoef, wantScale, wantNaN := fitOracle(exact, scale, mode, CurrentLossPolicy())
	if wantNaN != state.OK {
		if got.state != wantNaN {
			t.Errorf("%s(%s, %s) [%s] = %v, want NaN(%s)", op, x.StringFixed(), y.StringFixed(), mode, got, wantNaN)
		}
		return
	}
	if got.IsNaN() {
		t.Errorf("%s(%s, %s) [%s] = NaN(%v), want %s at scale %d", op, x.StringFixed(), y.StringFixed(), mode, got.ErrorDetails(), wantCoef, wantScale)
		return
	}
	if got.scale != wantScale {
		t.Errorf("%s(%s, %s) [%s] = %s: scale %d, want %d", op, x.StringFixed(), y.StringFixed(), mode, got.StringFixed(), got.scale, wantScale)
		return
	}
	gotAbs := new(big.Int).Abs(bigAtScale(got, got.scale))
	if gotAbs.Cmp(wantCoef) != 0 {
		t.Errorf("%s(%s, %s) [%s] = %s, want coefficient %s at scale %d", op, x.StringFixed(), y.StringFixed(), mode, got.StringFixed(), wantCoef, wantScale)
		return
	}
	wantNeg := exact.Sign() < 0 && wantCoef.Sign() != 0
	if (got.state == state.Neg) != wantNeg {
		t.Errorf("%s(%s, %s) [%s] = %s: wrong sign", op, x.StringFixed(), y.StringFixed(), mode, got.StringFixed())
	}
}

var allModes = []RoundingMode{
	ROUND_TOWARD_ZERO, ROUND_NAN, ROUND_DOWN, ROUND_UP, ROUND_AWAY_FROM_ZERO,
	ROUND_HALF_TOWARD_ZERO, ROUND_HALF_AWAY_FROM_ZERO, ROUND_BANK,
}

func TestMulAgainstBig(t *testing.T) {
	defer SetArithmeticRounding(ArithmeticRounding())
	r := rand.New(rand.NewSource(20260908))
	for _, mode := range allModes {
		SetArithmeticRounding(mode)
		for range 8000 {
			x, y := randDec(r), randDec(r)
			exact := new(big.Int).Mul(bigAtScale(x, x.scale), bigAtScale(y, y.scale))
			checkFit(t, "Mul", x, y, x.Mul(y), exact, x.scale+y.scale, mode)
		}
	}
}

func TestMulRoundAgainstBig(t *testing.T) {
	r := rand.New(rand.NewSource(20260909))
	for range 40000 {
		x, y := randDec(r), randDec(r)
		mode := allModes[r.Intn(len(allModes))]
		scale := uint8(r.Intn(int(MaxScale) + 1))
		got := x.MulRound(y, scale, mode)

		exact := new(big.Int).Mul(bigAtScale(x, x.scale), bigAtScale(y, y.scale))
		needed := x.scale + y.scale
		neg := exact.Sign() < 0
		abs := new(big.Int).Abs(exact)

		var want *big.Int
		var wantNaN state.State
		if scale >= needed {
			want = new(big.Int).Mul(abs, bigPow10(scale-needed))
			if want.Cmp(big2p128) >= 0 {
				wantNaN = state.Overflow
			}
		} else {
			d := bigPow10(needed - scale)
			q, rem := new(big.Int).QuoRem(abs, d, new(big.Int))
			want = roundBig(q, rem, d, neg, mode)
			switch {
			case want.Cmp(big2p128) >= 0:
				wantNaN = state.Overflow
			case mode == ROUND_NAN && rem.Sign() != 0:
				wantNaN = state.Inexact
			}
		}

		switch {
		case wantNaN != state.OK:
			if got.state != wantNaN {
				t.Errorf("MulRound(%s, %s, %d, %s) = %v, want NaN(%s)", x.StringFixed(), y.StringFixed(), scale, mode, got, wantNaN)
			}
		case got.IsNaN():
			t.Errorf("MulRound(%s, %s, %d, %s) = NaN(%v), want %s", x.StringFixed(), y.StringFixed(), scale, mode, got.ErrorDetails(), want)
		case got.scale != scale:
			t.Errorf("MulRound(%s, %s, %d, %s): scale %d", x.StringFixed(), y.StringFixed(), scale, mode, got.scale)
		case new(big.Int).Abs(bigAtScale(got, scale)).Cmp(want) != 0:
			t.Errorf("MulRound(%s, %s, %d, %s) = %s, want coefficient %s", x.StringFixed(), y.StringFixed(), scale, mode, got.StringFixed(), want)
		case (got.state == state.Neg) != (neg && want.Sign() != 0):
			t.Errorf("MulRound(%s, %s, %d, %s) = %s: wrong sign", x.StringFixed(), y.StringFixed(), scale, mode, got.StringFixed())
		}
	}
}

func TestMulRoundEdges(t *testing.T) {
	a, b := FromString("1234.5678"), FromString("8765.4321")
	cases := []struct {
		scale uint8
		mode  RoundingMode
		want  string
	}{
		{2, ROUND_HALF_AWAY_FROM_ZERO, "10821520.22"},
		{2, ROUND_TOWARD_ZERO, "10821520.22"},
		{0, ROUND_BANK, "10821520"},
		{8, ROUND_BANK, "10821520.22374638"},    // exact
		{10, ROUND_BANK, "10821520.2237463800"}, // padded
	}
	for _, c := range cases {
		if got := a.MulRound(b, c.scale, c.mode); got.StringFixed() != c.want {
			t.Errorf("MulRound(%d, %s) = %s, want %s", c.scale, c.mode, got.StringFixed(), c.want)
		}
	}
	if got := FromString("1.005").MulRound(One, 2, ROUND_BANK); got.StringFixed() != "1.00" {
		t.Errorf("tie to even: got %s", got.StringFixed())
	}
	if got := FromString("1.015").MulRound(One, 2, ROUND_BANK); got.StringFixed() != "1.02" {
		t.Errorf("tie to even: got %s", got.StringFixed())
	}
	if got := FromString("-1.005").MulRound(One, 2, ROUND_HALF_AWAY_FROM_ZERO); got.StringFixed() != "-1.01" {
		t.Errorf("negative half away: got %s", got.StringFixed())
	}
	if got := FromString("-1.001").MulRound(One, 2, ROUND_DOWN); got.StringFixed() != "-1.01" {
		t.Errorf("negative floor: got %s", got.StringFixed())
	}
	if got := FromString("1.001").MulRound(One, 2, ROUND_NAN); got.state != state.Inexact {
		t.Errorf("ROUND_NAN inexact: got %v", got)
	}
	if got := a.MulRound(b, MaxScale+1, ROUND_BANK); got.state != state.ScaleOutOfRange {
		t.Errorf("scale out of range: got %v", got)
	}
	if got := a.MulRound(b, 2, RoundingMode(77)); got.state != state.InvalidRoundingMode {
		t.Errorf("invalid mode: got %v", got)
	}
	if got := NaN(state.DivisionByZero).MulRound(b, 2, ROUND_BANK); got.state != state.DivisionByZero {
		t.Errorf("NaN propagation: got %v", got)
	}
	// A negative product that rounds to zero is a plain zero at the requested scale.
	if got := FromString("-0.001").MulRound(FromString("0.001"), 2, ROUND_BANK); got.StringFixed() != "0.00" || got.IsNegative() {
		t.Errorf("negative underflow to zero: got %s", got.StringFixed())
	}
}

func TestAddSubAllModesAgainstBig(t *testing.T) {
	defer SetArithmeticRounding(ArithmeticRounding())
	r := rand.New(rand.NewSource(20260910))
	for _, mode := range allModes {
		SetArithmeticRounding(mode)
		for range 6000 {
			x, y := randDec(r), randDec(r)
			scale := max(x.scale, y.scale)
			bx, by := bigAtScale(x, scale), bigAtScale(y, scale)
			checkFit(t, "Add", x, y, x.Add(y), new(big.Int).Add(bx, by), scale, mode)
			checkFit(t, "Sub", x, y, x.Sub(y), new(big.Int).Sub(bx, by), scale, mode)
		}
	}
}

func TestZeroKeepsScale(t *testing.T) {
	a := FromString("1.50")
	if got := a.Sub(a); got.StringFixed() != "0.00" || got.IsNegative() {
		t.Errorf("1.50 - 1.50 = %s, want 0.00", got.StringFixed())
	}
	if got := a.Add(a.Neg()); got.StringFixed() != "0.00" {
		t.Errorf("1.50 + -1.50 = %s, want 0.00", got.StringFixed())
	}
	if got := FromString("1.5").Sub(FromString("1.500")); got.StringFixed() != "0.000" {
		t.Errorf("1.5 - 1.500 = %s, want 0.000", got.StringFixed())
	}
	if got := FromString("0.00").Mul(FromString("5.0")); got.StringFixed() != "0.000" || got.IsNegative() {
		t.Errorf("0.00 * 5.0 = %s, want 0.000", got.StringFixed())
	}
	if got := FromString("-0.00").Mul(FromString("5.0")); got.IsNegative() {
		t.Errorf("zero product must not be negative")
	}
}

// divOracle returns |a|/|b| at the given scale, rounded with mode, in big arithmetic:
// the rounded coefficient, the sign, and whether it was inexact.
func divOracle(a, b Dec128, scale uint8, mode RoundingMode) (coef *big.Int, neg bool, inexact bool) {
	neg = (a.state == state.Neg) != (b.state == state.Neg)
	num := new(big.Int).Mul(new(big.Int).Abs(bigAtScale(a, a.scale)), bigPow10(scale+b.scale))
	den := new(big.Int).Mul(new(big.Int).Abs(bigAtScale(b, b.scale)), bigPow10(a.scale))
	q, r := new(big.Int).QuoRem(num, den, new(big.Int))
	return roundBig(q, r, den, neg, mode), neg, r.Sign() != 0
}

func checkQuotient(t *testing.T, op string, a, b, got Dec128, scale uint8, mode RoundingMode, want *big.Int, neg, inexact bool) {
	t.Helper()
	switch {
	case want.Cmp(big2p128) >= 0:
		if got.state != state.Overflow {
			t.Errorf("%s(%s, %s, %d, %s) = %v, want NaN(Overflow)", op, a.StringFixed(), b.StringFixed(), scale, mode, got)
		}
	case mode == ROUND_NAN && inexact:
		if got.state != state.Inexact {
			t.Errorf("%s(%s, %s, %d, %s) = %v, want NaN(Inexact)", op, a.StringFixed(), b.StringFixed(), scale, mode, got)
		}
	case got.IsNaN():
		t.Errorf("%s(%s, %s, %d, %s) = NaN(%v), want %s", op, a.StringFixed(), b.StringFixed(), scale, mode, got.ErrorDetails(), want)
	case got.scale != scale:
		t.Errorf("%s(%s, %s, %d, %s): scale %d", op, a.StringFixed(), b.StringFixed(), scale, mode, got.scale)
	case new(big.Int).Abs(bigAtScale(got, scale)).Cmp(want) != 0:
		t.Errorf("%s(%s, %s, %d, %s) = %s, want coefficient %s", op, a.StringFixed(), b.StringFixed(), scale, mode, got.StringFixed(), want)
	case (got.state == state.Neg) != (neg && want.Sign() != 0):
		t.Errorf("%s(%s, %s, %d, %s) = %s: wrong sign", op, a.StringFixed(), b.StringFixed(), scale, mode, got.StringFixed())
	}
}

func TestDivRoundAgainstBig(t *testing.T) {
	r := rand.New(rand.NewSource(20260911))
	for range 40000 {
		a, b := randDec(r), randDec(r)
		if b.IsZero() {
			continue
		}
		mode := allModes[r.Intn(len(allModes))]
		scale := uint8(r.Intn(int(MaxScale) + 1))
		want, neg, inexact := divOracle(a, b, scale, mode)
		checkQuotient(t, "DivRound", a, b, a.DivRound(b, scale, mode), scale, mode, want, neg, inexact)
	}
}

func TestDivAgainstBig(t *testing.T) {
	defer SetDefaultScale(DefaultScale())
	defer SetArithmeticRounding(ArithmeticRounding())
	r := rand.New(rand.NewSource(20260912))
	for _, def := range []uint8{MaxScale, 6, 0} {
		SetDefaultScale(def)
		for _, mode := range allModes {
			SetArithmeticRounding(mode)
			for range 3000 {
				a, b := randDec(r), randDec(r)
				if b.IsZero() || a.IsZero() {
					continue
				}
				got := a.Div(b)
				// Expected: the largest scale <= default at which the rounded quotient fits.
				var want *big.Int
				var neg, inexact bool
				scale := int(def)
				for ; scale >= 0; scale-- {
					want, neg, inexact = divOracle(a, b, uint8(scale), mode)
					if want.Cmp(big2p128) < 0 {
						break
					}
				}
				if scale < 0 {
					if got.state != state.Overflow {
						t.Errorf("Div(%s, %s) [%s, default %d] = %v, want NaN(Overflow)", a.StringFixed(), b.StringFixed(), mode, def, got)
					}
					continue
				}
				if !inexact && want.Sign() != 0 {
					// exact: ideal scale, trailing zeros removed down to max(s1-s2, 0)
					ideal := 0
					if a.scale > b.scale {
						ideal = int(a.scale - b.scale)
					}
					for scale > ideal && new(big.Int).Rem(want, bigTen).Sign() == 0 {
						want.Quo(want, bigTen)
						scale--
					}
				}
				checkQuotient(t, "Div", a, b, got, uint8(scale), mode, want, neg, inexact)
			}
		}
	}
}

func TestSqrtRoundAgainstBig(t *testing.T) {
	r := rand.New(rand.NewSource(20260913))
	for range 30000 {
		d := randDec(r)
		if d.IsZero() {
			continue
		}
		d.state = state.Default
		mode := allModes[r.Intn(len(allModes))]
		scale := uint8(r.Intn(int(MaxScale) + 1))
		got := d.SqrtRound(scale, mode)

		// value = coef / 10^s; sqrt(value) * 10^scale = sqrt(coef * 10^(2*scale) / 10^s)
		num := new(big.Int).Mul(bigAtScale(d, d.scale), bigPow10(2*scale))
		den := bigPow10(d.scale)
		q := new(big.Int).Sqrt(new(big.Int).Quo(num, den)) // floor(sqrt(floor(x))) == floor(sqrt(x))
		exact := new(big.Int).Mul(new(big.Int).Mul(q, q), den).Cmp(num) == 0
		var want *big.Int
		if exact {
			want = q
		} else {
			// above half iff x > (q + 1/2)^2 iff 4*num > (2q+1)^2 * den; a tie is impossible
			lhs := new(big.Int).Lsh(num, 2)
			twoQ1 := new(big.Int).Add(new(big.Int).Lsh(q, 1), big.NewInt(1))
			rhs := new(big.Int).Mul(new(big.Int).Mul(twoQ1, twoQ1), den)
			above := lhs.Cmp(rhs) > 0
			up := false
			switch mode {
			case ROUND_UP, ROUND_AWAY_FROM_ZERO:
				up = true
			case ROUND_HALF_TOWARD_ZERO, ROUND_HALF_AWAY_FROM_ZERO, ROUND_BANK:
				up = above
			}
			want = q
			if up {
				want = new(big.Int).Add(q, big.NewInt(1))
			}
		}
		switch {
		case mode == ROUND_NAN && !exact:
			if got.state != state.Inexact {
				t.Errorf("SqrtRound(%s, %d, %s) = %v, want NaN(Inexact)", d.StringFixed(), scale, mode, got)
			}
		case got.IsNaN():
			t.Errorf("SqrtRound(%s, %d, %s) = NaN(%v), want %s", d.StringFixed(), scale, mode, got.ErrorDetails(), want)
		case got.scale != scale || got.IsNegative():
			t.Errorf("SqrtRound(%s, %d, %s) = %s: bad scale or sign", d.StringFixed(), scale, mode, got.StringFixed())
		case bigAtScale(got, scale).Cmp(want) != 0:
			t.Errorf("SqrtRound(%s, %d, %s) = %s, want coefficient %s", d.StringFixed(), scale, mode, got.StringFixed(), want)
		}
	}
}

func TestFitCarryOut(t *testing.T) {
	// 101 * 3369132345751865974884897103284833776.79 = (2^128-1) * 10^2 + 79 at scale 2:
	// dropping two digits leaves the maximum coefficient with a remainder above half, so
	// rounding half away carries out of 128 bits and one more digit must go.
	a := FromString("101")
	b := FromString("3369132345751865974884897103284833776.79")
	exact := new(big.Int).Mul(bigAtScale(a, a.scale), bigAtScale(b, b.scale))
	defer SetArithmeticRounding(ArithmeticRounding())
	for _, mode := range []RoundingMode{ROUND_HALF_AWAY_FROM_ZERO, ROUND_UP, ROUND_TOWARD_ZERO} {
		SetArithmeticRounding(mode)
		checkFit(t, "Mul", a, b, a.Mul(b), exact, a.scale+b.scale, mode)
	}
	// Asked for scale 0 exactly, the carried result does not fit: Overflow.
	if got := a.MulRound(b, 0, ROUND_HALF_AWAY_FROM_ZERO); got.state != state.Overflow {
		t.Errorf("MulRound carry-out must overflow, got %v", got)
	}
	if got := a.MulRound(b, 0, ROUND_TOWARD_ZERO); got.IsNaN() || got.String() != "340282366920938463463374607431768211455" {
		t.Errorf("MulRound truncated must be the maximum, got %v", got)
	}
	if got := a.MulRound(NaN(state.Null), 0, ROUND_TOWARD_ZERO); got.state != state.Null {
		t.Errorf("NaN in the second operand must propagate, got %v", got)
	}
}

// TestDivRoundingCarry constructs D = (9*(2^128-1) + 5) / 10, so that D*10/9 is exactly
// 2^128-1 with remainder 5 (above half of 9). Rounding half away then carries out of
// 128 bits: DivRound at scale 1 must overflow, and Div must fall back one more scale
// even though the bit-length estimate said scale 1 fits.
func TestDivRoundingCarry(t *testing.T) {
	m := new(big.Int).Sub(big2p128, big.NewInt(1))
	dBig := new(big.Int).Mul(m, big.NewInt(9))
	dBig.Add(dBig, big.NewInt(5))
	dBig.Quo(dBig, big.NewInt(10))
	d := FromString(dBig.String())
	nine := FromInt64(9)
	if d.IsNaN() {
		t.Fatal("D must be representable")
	}

	if got := d.DivRound(nine, 1, ROUND_HALF_AWAY_FROM_ZERO); got.state != state.Overflow {
		t.Errorf("DivRound carry-out must overflow, got %v", got)
	}
	if got := d.DivRound(nine, 1, ROUND_TOWARD_ZERO); got.IsNaN() || bigAtScale(got, 1).Cmp(m) != 0 {
		t.Errorf("DivRound truncated must be 2^128-1 at scale 1, got %v", got)
	}

	defer SetArithmeticRounding(ArithmeticRounding())
	defer SetDefaultScale(DefaultScale())
	SetDefaultScale(MaxScale)
	SetArithmeticRounding(ROUND_HALF_AWAY_FROM_ZERO)
	got := d.Div(nine)
	want, neg, inexact := divOracle(d, nine, 0, ROUND_HALF_AWAY_FROM_ZERO)
	checkQuotient(t, "Div", d, nine, got, 0, ROUND_HALF_AWAY_FROM_ZERO, want, neg, inexact)
}

func TestIdealScale(t *testing.T) {
	defer SetDefaultScale(DefaultScale())
	SetDefaultScale(MaxScale)
	cases := []struct{ a, b, want string }{
		{"1", "2", "0.5"},
		{"1.00", "2", "0.50"},
		{"1", "0.5", "2"},
		{"1.0", "0.5", "2"},
		{"10", "4", "2.5"},
		{"6.00", "2.0", "3.0"},
		{"100", "4", "25"},
		{"1", "3", "0.3333333333333333333"},
		{"0", "3", "0"},
		{"0.00", "3", "0.00"},
		{"0.000", "0.5", "0.00"},
		{"-1.50", "2", "-0.75"},
		{"3.000", "1.5", "2.00"},
	}
	for _, c := range cases {
		got := FromString(c.a).Div(FromString(c.b))
		if got.StringFixed() != c.want {
			t.Errorf("%s / %s = %s, want %s", c.a, c.b, got.StringFixed(), c.want)
		}
	}
	sqrtCases := []struct{ in, want string }{
		{"4", "2"},
		{"4.00", "2.0"},
		{"1.00", "1.0"},
		{"0.0100", "0.10"},
		{"0.01", "0.1"},
		{"2.00", "1.4142135623730950488"},
		{"0", "0"},
		{"0.000", "0.00"},
		{"100000000000000000000000000000000000000", "10000000000000000000"},
	}
	for _, c := range sqrtCases {
		got := FromString(c.in).Sqrt()
		if got.StringFixed() != c.want {
			t.Errorf("sqrt(%s) = %s, want %s", c.in, got.StringFixed(), c.want)
		}
	}
	// SqrtRound and DivRound never strip: the caller asked for that scale.
	if got := FromString("4").SqrtRound(3, ROUND_BANK); got.StringFixed() != "2.000" {
		t.Errorf("SqrtRound must keep the requested scale, got %s", got.StringFixed())
	}
	if got := FromString("1").DivRound(FromString("2"), 3, ROUND_BANK); got.StringFixed() != "0.500" {
		t.Errorf("DivRound must keep the requested scale, got %s", got.StringFixed())
	}
	// Under ROUND_NAN an inexact root is refused and an exact one is stripped as usual.
	defer SetArithmeticRounding(ArithmeticRounding())
	SetArithmeticRounding(ROUND_NAN)
	if got := FromString("2").Sqrt(); got.state != state.Inexact {
		t.Errorf("sqrt(2) under ROUND_NAN must be NaN(Inexact), got %v", got)
	}
	if got := FromString("9.0").Sqrt(); got.StringFixed() != "3.0" {
		t.Errorf("sqrt(9.0) under ROUND_NAN = %s, want 3.0", got.StringFixed())
	}
	if got := FromString("-4").Sqrt(); got.state != state.SqrtNegative {
		t.Errorf("sqrt(-4) must be NaN(SqrtNegative), got %v", got)
	}
}

func TestSumAgainstBig(t *testing.T) {
	defer SetArithmeticRounding(ArithmeticRounding())
	r := rand.New(rand.NewSource(20260919))
	for _, mode := range allModes {
		SetArithmeticRounding(mode)
		for range 3000 {
			n := 1 + r.Intn(12)
			terms := make([]Dec128, n)
			scale := uint8(0)
			for i := range terms {
				terms[i] = randDec(r)
				scale = max(scale, terms[i].scale)
			}
			exact := new(big.Int)
			for _, d := range terms {
				exact.Add(exact, bigAtScale(d, scale))
			}
			got := Sum(terms[0], terms[1:]...)
			checkFit(t, "Sum", terms[0], terms[len(terms)-1], got, exact, scale, mode)

			// order independence: a shuffled sum is identical
			r.Shuffle(n, func(i, j int) { terms[i], terms[j] = terms[j], terms[i] })
			if again := Sum(terms[0], terms[1:]...); again != got {
				t.Errorf("Sum is order dependent: %v vs %v", got, again)
			}
		}
	}
}

func TestSumExactness(t *testing.T) {
	m := MaxAtScale(0)
	// max + max - max would overflow an intermediate; the exact total is max
	if got := Sum(m, m, m.Neg()); got != m {
		t.Errorf("Sum(max, max, -max) = %v, want max", got)
	}
	if got := Sum(m, m); !got.IsNaN() {
		t.Errorf("Sum(max, max) must overflow, got %v", got)
	}
	// cancellation of tiny and huge terms is exact regardless of order
	q := QuantumAtScale(MaxScale)
	if got := Sum(q, m, m.Neg()); got != q {
		t.Errorf("Sum(q, max, -max) = %v, want the quantum", got)
	}
	if got := Sum(m, q, m.Neg()); got != q {
		t.Errorf("Sum(max, q, -max) = %v, want the quantum", got)
	}
	// NaN propagates, first in order
	if got := Sum(One, NaN(state.DivisionByZero), NaN(state.Overflow)); got.state != state.DivisionByZero {
		t.Errorf("Sum NaN propagation: %v", got)
	}
	if got := Sum(NaN(state.Null), One); got.state != state.Null {
		t.Errorf("Sum NaN first argument: %v", got)
	}
	// zero keeps the common scale and is not negative
	if got := Sum(FromString("1.50"), FromString("-1.5")); got.StringFixed() != "0.00" || got.IsNegative() {
		t.Errorf("Sum to zero: %v", got)
	}
	if got := Avg(FromString("1"), FromString("2")); got.StringFixed() != "1.5" {
		t.Errorf("Avg(1, 2) = %v, want 1.5", got)
	}
}
