package dec128

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/jokruger/dec128/state"
)

// Sequential arithmetic at the working scales a ledger actually uses - 2 for currency amounts, 6 for rates - under
// every rounding mode, checked step by step against a math/big oracle.
//
// The single-operation oracles in arith_crosscheck_test.go verify the scale rule once, on fresh operands. These tests
// verify that it stays correct along a chain, where every step consumes the previous step's rounded result: a
// coefficient that has been reduced once, a scale that has moved, a sign that has flipped. That is where a
// round-to-fit bug would show up in production and not in a table test.
//
// All of them restore the process-global configuration they change.

// roundQuot rounds the exact rational num/den (den > 0) to an integer with mode and reports whether any digits were
// discarded. It is the signed wrapper around roundBig, which works on magnitudes.
func roundQuot(num, den *big.Int, mode RoundingMode) (*big.Int, bool) {
	neg := num.Sign() < 0
	q, r := new(big.Int).QuoRem(new(big.Int).Abs(num), den, new(big.Int))
	out := roundBig(q, r, den, neg, mode)
	if neg && out.Sign() != 0 {
		out = new(big.Int).Neg(out)
	}
	return out, r.Sign() != 0
}

// chainOperand returns a Dec128 at the given scale whose magnitude spans the whole range, so that a chain reaches the
// 128-bit ceiling often enough to exercise the reduction path without ending on the first step.
func chainOperand(r *rand.Rand, scale uint8) Dec128 {
	var d Dec128
	switch r.Intn(4) {
	case 0:
		d.coef.Lo = uint64(r.Intn(1_000_000)) + 1
	case 1:
		d.coef.Lo = r.Uint64()
	case 2:
		d.coef.Lo, d.coef.Hi = r.Uint64(), r.Uint64()>>uint(32+r.Intn(32))
	default:
		d.coef.Lo, d.coef.Hi = r.Uint64(), r.Uint64()
	}
	d.scale = scale
	if r.Intn(2) == 0 && !d.coef.IsZero() {
		d.state = state.Neg
	}
	return d
}

// TestChainPinnedScaleAgainstBig walks chains of MulRound, DivRound, Add and Sub with every value pinned to one
// working scale, comparing each step against math/big. MulRound and DivRound must return exactly the requested scale
// with an exact rounding decision; Add and Sub are exact at that scale until the coefficient no longer fits, at which
// point the scale rule takes over and fitOracle is the reference.
func TestChainPinnedScaleAgainstBig(t *testing.T) {
	defer SetArithmeticRounding(ArithmeticRounding())

	const steps = 24
	for _, ws := range []uint8{2, 6, 19} {
		for _, mode := range allModes {
			SetArithmeticRounding(mode) // Add and Sub read the global; MulRound and DivRound take mode per call
			r := rand.New(rand.NewSource(int64(ws)*100 + int64(mode)))
			var fullChains, reductions, inexactSteps int

			for chain := 0; chain < 400; chain++ {
				acc := chainOperand(r, ws)
				oracle := bigAtScale(acc, ws) // exact signed coefficient of acc at scale ws
				step := 0

				for ; step < steps; step++ {
					b := chainOperand(r, ws)
					bc := bigAtScale(b, ws)
					var got Dec128
					var want *big.Int
					var inexact bool
					var op string

					switch r.Intn(4) {
					case 0:
						op = "MulRound"
						got = acc.MulRound(b, ws, mode)
						want, inexact = roundQuot(new(big.Int).Mul(oracle, bc), bigPow10(ws), mode)
					case 1:
						op = "DivRound"
						if b.coef.IsZero() {
							continue
						}
						got = acc.DivRound(b, ws, mode)
						// The rounding direction follows the sign of the quotient, not of the dividend, so the
						// numerator handed to roundQuot carries sign(acc) XOR sign(b).
						num := new(big.Int).Mul(oracle, bigPow10(ws))
						if bc.Sign() < 0 {
							num.Neg(num)
						}
						want, inexact = roundQuot(num, new(big.Int).Abs(bc), mode)
					case 2:
						op = "Add"
						got = acc.Add(b)
						want = new(big.Int).Add(oracle, bc)
					default:
						op = "Sub"
						got = acc.Sub(b)
						want = new(big.Int).Sub(oracle, bc)
					}

					if op == "Add" || op == "Sub" {
						// The exact sum has scale ws; the scale rule applies only if it no longer fits.
						checkFit(t, op, acc, b, got, want, ws, mode)
						if got.IsNaN() || got.scale != ws {
							if !got.IsNaN() {
								reductions++
							}
							break
						}
						oracle = want
						acc = got
						continue
					}

					// MulRound and DivRound: exactly the requested scale, or a documented NaN. A rounded value that
					// does not fit is Overflow even under ROUND_NAN, which is the order the implementations check in.
					switch {
					case new(big.Int).Abs(want).Cmp(big2p128) >= 0:
						if got.state != state.Overflow {
							t.Fatalf("ws=%d %v chain %d step %d: %s(%s, %s) = %v, want NaN(Overflow)",
								ws, mode, chain, step, op, acc.StringFixed(), b.StringFixed(), got)
						}
						reductions++
					case inexact && mode == ROUND_NAN:
						if got.state != state.Inexact {
							t.Fatalf("ws=%d %v chain %d step %d: %s(%s, %s) = %v, want NaN(Inexact)",
								ws, mode, chain, step, op, acc.StringFixed(), b.StringFixed(), got)
						}
						inexactSteps++
					case got.IsNaN():
						t.Fatalf("ws=%d %v chain %d step %d: %s(%s, %s) = NaN(%v), want %s at scale %d",
							ws, mode, chain, step, op, acc.StringFixed(), b.StringFixed(), got.ErrorDetails(), want, ws)
					case got.scale != ws:
						t.Fatalf("ws=%d %v chain %d step %d: %s(%s, %s) = %s at scale %d, want scale %d",
							ws, mode, chain, step, op, acc.StringFixed(), b.StringFixed(), got.StringFixed(), got.scale, ws)
					case bigAtScale(got, ws).Cmp(want) != 0:
						t.Fatalf("ws=%d %v chain %d step %d: %s(%s, %s) = %s, want coefficient %s at scale %d",
							ws, mode, chain, step, op, acc.StringFixed(), b.StringFixed(), got.StringFixed(), want, ws)
					default:
						if inexact {
							inexactSteps++
						}
						oracle = want
						acc = got
						continue
					}
					break // a NaN ends the chain
				}
				if step == steps {
					fullChains++
				}
			}

			// Guard against the test going vacuous if the operand mix ever changes. ROUND_NAN is exempt from the
			// full-chain requirement: it refuses the first inexact step, so its chains are short by design.
			if reductions == 0 || (fullChains == 0 && mode != ROUND_NAN) {
				t.Errorf("ws=%d %v: chains too shallow (full=%d reductions=%d inexact=%d)", ws, mode, fullChains, reductions, inexactSteps)
			}
			t.Logf("ws=%2d %-26v full chains=%3d reductions/NaN=%4d inexact steps=%5d", ws, mode, fullChains, reductions, inexactSteps)
		}
	}
}

// TestChainArithmeticRoundingAtSmallScales exercises the implicit path: plain Mul, Add and Sub on scale-2 and scale-6
// operands, where the exact result routinely needs more than 128 bits and the scale is reduced with the mode set by
// SetArithmeticRounding. Every step is checked against fitOracle, and the accumulator carries the reduced result into
// the next step.
func TestChainArithmeticRoundingAtSmallScales(t *testing.T) {
	defer SetArithmeticRounding(ArithmeticRounding())

	for _, ws := range []uint8{2, 6} {
		for _, mode := range allModes {
			SetArithmeticRounding(mode)
			r := rand.New(rand.NewSource(int64(ws)*1000 + int64(mode)))
			var steps, reduced, nans int

			for chain := 0; chain < 300; chain++ {
				acc := chainOperand(r, ws)
				for range 12 {
					b := chainOperand(r, ws)
					var got Dec128
					var exact *big.Int
					var scale uint8
					var op string

					switch r.Intn(3) {
					case 0:
						op, scale = "Mul", acc.scale+b.scale
						got = acc.Mul(b)
						exact = new(big.Int).Mul(bigAtScale(acc, acc.scale), bigAtScale(b, b.scale))
					case 1:
						op, scale = "Add", max(acc.scale, b.scale)
						got = acc.Add(b)
						exact = new(big.Int).Add(bigAtScale(acc, scale), bigAtScale(b, scale))
					default:
						op, scale = "Sub", max(acc.scale, b.scale)
						got = acc.Sub(b)
						exact = new(big.Int).Sub(bigAtScale(acc, scale), bigAtScale(b, scale))
					}

					checkFit(t, op, acc, b, got, exact, scale, mode)
					steps++
					if got.IsNaN() {
						nans++
						break
					}
					if got.scale < min(scale, MaxScale) {
						reduced++
					}
					// A zero result is never negative, whatever the operands were.
					if got.coef.IsZero() && got.state == state.Neg {
						t.Fatalf("ws=%d %v: %s produced a negative zero", ws, mode, op)
					}
					acc = got
				}
			}

			if reduced == 0 {
				t.Errorf("ws=%d %v: no step was reduced (steps=%d nans=%d)", ws, mode, steps, nans)
			}
			t.Logf("ws=%d %-26v steps=%5d reduced=%4d nan=%4d", ws, mode, steps, reduced, nans)
		}
	}
}

// TestOverflowRoundingAtScale2And6 pins down the reduction of a product that does not fit, at the small working scales,
// with values where the mode changes the answer. The expected strings were computed independently in Python with
// decimal/int arithmetic, not by running this package.
func TestOverflowRoundingAtScale2And6(t *testing.T) {
	defer SetArithmeticRounding(ArithmeticRounding())

	type expect struct {
		mode RoundingMode
		want string
	}
	cases := []struct {
		name    string
		a, b    string
		expects []expect
	}{
		{
			// exact product is 42 digits at scale 4, so 4 digits go and the result lands at scale 0
			name: "scale2/reduce-to-0",
			a:    "51731704880396946889085588584256.60",
			b:    "1067740.77",
			expects: []expect{
				{ROUND_TOWARD_ZERO, "55236050402407793977001350950857351961"},
				{ROUND_DOWN, "55236050402407793977001350950857351961"},
				{ROUND_BANK, "55236050402407793977001350950857351962"},
				{ROUND_HALF_AWAY_FROM_ZERO, "55236050402407793977001350950857351962"},
				{ROUND_UP, "55236050402407793977001350950857351962"},
				{ROUND_AWAY_FROM_ZERO, "55236050402407793977001350950857351962"},
			},
		},
		{
			// exact tie at the reduction point with an even kept digit: ROUND_BANK keeps, half-away steps up
			name: "scale2/exact-tie-even",
			a:    "3820144601146672809592787927022.25",
			b:    "555567.54",
			expects: []expect{
				{ROUND_TOWARD_ZERO, "2122348338503338192010353590357450957.76"},
				{ROUND_BANK, "2122348338503338192010353590357450957.76"},
				{ROUND_HALF_TOWARD_ZERO, "2122348338503338192010353590357450957.76"},
				{ROUND_HALF_AWAY_FROM_ZERO, "2122348338503338192010353590357450957.77"},
				{ROUND_UP, "2122348338503338192010353590357450957.77"},
				{ROUND_AWAY_FROM_ZERO, "2122348338503338192010353590357450957.77"},
			},
		},
		{
			// exact product is 47 digits at scale 12: 9 digits go and the result lands at scale 3
			name: "scale6/reduce-to-3",
			a:    "610289584300879434569122178571.929818",
			b:    "63407.620001",
			expects: []expect{
				{ROUND_TOWARD_ZERO, "38697010051918418437274642267030190.744"},
				{ROUND_DOWN, "38697010051918418437274642267030190.744"},
				{ROUND_BANK, "38697010051918418437274642267030190.745"},
				{ROUND_HALF_AWAY_FROM_ZERO, "38697010051918418437274642267030190.745"},
				{ROUND_UP, "38697010051918418437274642267030190.745"},
			},
		},
		{
			name: "scale6/exact-tie-even",
			a:    "1540451650692680715476411491.151065",
			b:    "18.075090",
			expects: []expect{
				{ROUND_TOWARD_ZERO, "27843802226918766273500530579.5897034708"},
				{ROUND_BANK, "27843802226918766273500530579.5897034708"},
				{ROUND_HALF_TOWARD_ZERO, "27843802226918766273500530579.5897034708"},
				{ROUND_HALF_AWAY_FROM_ZERO, "27843802226918766273500530579.5897034709"},
				{ROUND_AWAY_FROM_ZERO, "27843802226918766273500530579.5897034709"},
			},
		},
	}

	for _, c := range cases {
		a, b := FromString(c.a), FromString(c.b)
		if a.IsNaN() || b.IsNaN() {
			t.Fatalf("%s: operands did not parse: %v / %v", c.name, a.ErrorDetails(), b.ErrorDetails())
		}
		for _, e := range c.expects {
			SetArithmeticRounding(e.mode)
			if got := a.Mul(b).StringFixed(); got != e.want {
				t.Errorf("%s [%v]: %s * %s = %s, want %s", c.name, e.mode, c.a, c.b, got, e.want)
			}
			// The same reduction through the negated operands, with the directed modes mirrored.
			mirror := e.mode
			switch e.mode {
			case ROUND_DOWN:
				mirror = ROUND_UP
			case ROUND_UP:
				mirror = ROUND_DOWN
			}
			SetArithmeticRounding(mirror)
			if got := a.Neg().Mul(b).StringFixed(); got != "-"+e.want {
				t.Errorf("%s [%v negated]: -%s * %s = %s, want -%s", c.name, mirror, c.a, c.b, got, e.want)
			}
		}
	}
}

// TestOverflowRoundingExactTies drives the reduction onto an exact half-way point through the public Mul, for both
// parities of the digit that is kept and for both signs. This is the case where ROUND_BANK, ROUND_HALF_AWAY_FROM_ZERO
// and the directed modes must all disagree, so it is the sharpest check on the tie detection in roundUp.
//
// Construction: a = 0.000005 at scale 6 and b = (2Q+1) * 10^-6, so the exact product coefficient is 10*Q + 5 at
// scale 12 - a 40-digit value whose reduction by one digit leaves Q with a remainder of exactly half.
func TestOverflowRoundingExactTies(t *testing.T) {
	defer SetArithmeticRounding(ArithmeticRounding())

	cases := []struct {
		name string
		b    string
		// expected magnitudes at scale 11, keyed by mode, for a positive product
		keep, step string
	}{
		{
			name: "kept-digit-even", // Q = 10^38, last digit 0
			b:    "200000000000000000000000000000000.000001",
			keep: "1000000000000000000000000000.00000000000",
			step: "1000000000000000000000000000.00000000001",
		},
		{
			name: "kept-digit-odd", // Q = 10^38 + 1, last digit 1
			b:    "200000000000000000000000000000000.000003",
			keep: "1000000000000000000000000000.00000000001",
			step: "1000000000000000000000000000.00000000002",
		},
	}

	a := FromString("0.000005")
	if a.IsNaN() {
		t.Fatal("operand did not parse")
	}

	for _, c := range cases {
		b := FromString(c.b)
		if b.IsNaN() {
			t.Fatalf("%s: %v", c.name, b.ErrorDetails())
		}
		odd := c.name == "kept-digit-odd"

		// mode -> whether the magnitude steps up, for a positive and for a negative product
		type want struct{ pos, neg bool }
		table := map[RoundingMode]want{
			ROUND_TOWARD_ZERO:         {false, false},
			ROUND_HALF_TOWARD_ZERO:    {false, false},
			ROUND_BANK:                {odd, odd},
			ROUND_HALF_AWAY_FROM_ZERO: {true, true},
			ROUND_AWAY_FROM_ZERO:      {true, true},
			ROUND_UP:                  {true, false}, // toward +infinity
			ROUND_DOWN:                {false, true}, // toward -infinity
		}

		for mode, w := range table {
			SetArithmeticRounding(mode)

			wantPos := c.keep
			if w.pos {
				wantPos = c.step
			}
			if got := a.Mul(b); got.StringFixed() != wantPos {
				t.Errorf("%s [%v]: 0.000005 * %s = %s, want %s", c.name, mode, c.b, got.StringFixed(), wantPos)
			} else if got.Scale() != 11 {
				t.Errorf("%s [%v]: scale %d, want 11", c.name, mode, got.Scale())
			}

			wantNeg := "-" + c.keep
			if w.neg {
				wantNeg = "-" + c.step
			}
			if got := a.Neg().Mul(b).StringFixed(); got != wantNeg {
				t.Errorf("%s [%v negated]: -0.000005 * %s = %s, want %s", c.name, mode, c.b, got, wantNeg)
			}
		}

		// ROUND_NAN refuses the whole reduction rather than choosing a side.
		SetArithmeticRounding(ROUND_NAN)
		if got := a.Mul(b); got.ErrorDetails() != state.Inexact.Error() {
			t.Errorf("%s [ROUND_NAN]: got %v, want NaN(Inexact)", c.name, got.ErrorDetails())
		}
	}
}

// TestChainBankVersusTruncateDrift is the accounting property behind the two modes. Rounding a running total to two
// places at every step, truncation loses part of every inexact amount and the shortfall accumulates, while ties-to-even
// gives the digits back roughly half the time and stays near the exact total. Both chains must match their own oracle
// exactly, and truncation must never end above ties-to-even for positive amounts (rounding is monotone and
// ROUND_TOWARD_ZERO is never above ROUND_BANK, so the ordering holds by induction along the chain).
func TestChainBankVersusTruncateDrift(t *testing.T) {
	const steps = 400
	const ws = uint8(2)

	// The amounts carry four decimals, so every step has something to discard, and half of them are exact ties.
	amounts := make([]Dec128, steps)
	r := rand.New(rand.NewSource(20260910))
	for i := range amounts {
		cents := int64(r.Intn(100_000))
		if i%2 == 0 {
			cents = cents/100*100 + 50 // ...50 at four decimals: an exact half of a cent
		}
		amounts[i] = DecodeFromInt64(cents, 4)
	}

	run := func(mode RoundingMode) (Dec128, *big.Int) {
		acc := Dec128{scale: ws}
		oracle := new(big.Int)
		for _, x := range amounts {
			sum := acc.Add(x) // scale 4, exact
			if sum.IsNaN() || sum.Scale() != 4 {
				t.Fatalf("[%v] Add left the working scale: %v", mode, sum)
			}
			got := sum.RescaleRound(ws, mode)
			want, _ := roundQuot(new(big.Int).Add(new(big.Int).Mul(oracle, bigPow10(2)), bigAtScale(x, 4)), bigPow10(2), mode)
			if got.IsNaN() || got.Scale() != ws || bigAtScale(got, ws).Cmp(want) != 0 {
				t.Fatalf("[%v] step: got %s (scale %d), want coefficient %s at scale %d", mode, got.StringFixed(), got.Scale(), want, ws)
			}
			acc, oracle = got, want
		}
		return acc, oracle
	}

	tz, tzOracle := run(ROUND_TOWARD_ZERO)
	bank, bankOracle := run(ROUND_BANK)

	// The exact, never-rounded total of the same amounts.
	exact := new(big.Int)
	for _, x := range amounts {
		exact.Add(exact, bigAtScale(x, 4))
	}
	// as a coefficient at scale 2, truncated, for comparison
	exactAt2 := new(big.Int).Quo(exact, bigPow10(2))

	tzGap := new(big.Int).Sub(exactAt2, tzOracle) // cents lost to truncation
	bankGap := new(big.Int).Sub(exactAt2, bankOracle)
	t.Logf("after %d steps: exact=%s.xx  truncate=%s (short %s cents)  bank=%s (short %s cents)",
		steps, exactAt2, tzOracle, tzGap, bankOracle, bankGap)

	if tz.Compare(bank) > 0 {
		t.Errorf("truncation ended above ties-to-even: %s > %s", tz.StringFixed(), bank.StringFixed())
	}
	// Truncation must drift well past a single quantum over this many inexact steps; ties-to-even must not.
	if tzGap.Cmp(big.NewInt(50)) < 0 {
		t.Errorf("truncation drift is only %s cents over %d steps, expected a large systematic shortfall", tzGap, steps)
	}
	if new(big.Int).Abs(bankGap).Cmp(big.NewInt(5)) > 0 {
		t.Errorf("ties-to-even drifted %s cents over %d steps, expected it to stay within a few", bankGap, steps)
	}
}

// TestChainDivAtSmallDefaultScales walks chains of the scale-choosing Div with the default scale set to 2 and to 6.
// Div reduces the scale when the quotient does not fit; along a chain the reduced result feeds the next division, so
// this checks both that the reduction is correct and that it picks the largest scale that fits, every time.
func TestChainDivAtSmallDefaultScales(t *testing.T) {
	defer SetDefaultScale(DefaultScale())
	defer SetArithmeticRounding(ArithmeticRounding())

	for _, def := range []uint8{2, 6} {
		SetDefaultScale(def)
		totalReductions := 0
		for _, mode := range allModes {
			SetArithmeticRounding(mode)
			r := rand.New(rand.NewSource(int64(def)*10_000 + int64(mode)))
			var steps, reductions int

			for chain := 0; chain < 300; chain++ {
				acc := chainOperand(r, def)
				for range 10 {
					b := chainOperand(r, def)
					if r.Intn(3) == 0 {
						// a small divisor makes the quotient overflow at a small default scale, which is the
						// scale-reduction path this test is here for
						b = DecodeFromInt64(int64(r.Intn(99))+1, def)
					}
					if b.coef.IsZero() || acc.coef.IsZero() {
						break
					}
					got := acc.Div(b)
					steps++

					// The largest scale not above the default at which the rounded quotient fits.
					best := -1
					for s := int(def); s >= 0; s-- {
						if q := acc.DivRound(b, uint8(s), mode); !q.IsNaN() {
							best = s
							break
						}
					}

					if mode == ROUND_NAN {
						// ROUND_NAN reports Inexact rather than choosing a scale; only a valid result is comparable.
						if !got.IsNaN() {
							if w := acc.DivRound(b, got.Scale(), mode); !w.Equal(got) {
								t.Fatalf("def=%d %v: Div(%s, %s) = %s, DivRound at scale %d = %s",
									def, mode, acc.StringFixed(), b.StringFixed(), got.StringFixed(), got.Scale(), w.StringFixed())
							}
						}
					} else {
						switch {
						case best < 0:
							if got.state != state.Overflow {
								t.Fatalf("def=%d %v: Div(%s, %s) = %v, want NaN(Overflow)",
									def, mode, acc.StringFixed(), b.StringFixed(), got)
							}
						case got.IsNaN():
							t.Fatalf("def=%d %v: Div(%s, %s) = NaN(%v) but scale %d fits",
								def, mode, acc.StringFixed(), b.StringFixed(), got.ErrorDetails(), best)
						default:
							if want := acc.DivRound(b, uint8(best), mode); !got.Equal(want) {
								t.Fatalf("def=%d %v: Div(%s, %s) = %s, largest fitting DivRound(%d) = %s",
									def, mode, acc.StringFixed(), b.StringFixed(), got.StringFixed(), best, want.StringFixed())
							}
							if best < int(def) {
								reductions++
							}
						}
					}

					if got.IsNaN() {
						break
					}
					if got.coef.IsZero() && got.state == state.Neg {
						t.Fatalf("def=%d %v: Div produced a negative zero", def, mode)
					}
					acc = got
				}
			}
			totalReductions += reductions
			t.Logf("def=%d %-26v steps=%5d reduced=%4d", def, mode, steps, reductions)
		}
		// The per-mode split is seed noise; what matters is that the reduction path was walked at this scale.
		if totalReductions == 0 {
			t.Errorf("def=%d: no division ever needed a reduced scale", def)
		}
	}
}

// TestChainStaysWithinOneQuantum is the coarse safety net: whatever the mode and whatever the chain, a result that is
// not NaN must be within one quantum of its own scale of the exact value it stands for, and must carry the right sign.
// A silent wraparound or a lost carry would break this even where the exact oracles above happen to agree.
func TestChainStaysWithinOneQuantum(t *testing.T) {
	defer SetArithmeticRounding(ArithmeticRounding())

	for _, ws := range []uint8{2, 6, 19} {
		for _, mode := range allModes {
			SetArithmeticRounding(mode)
			r := rand.New(rand.NewSource(int64(ws)*77 + int64(mode)))
			for chain := 0; chain < 500; chain++ {
				acc := chainOperand(r, ws)
				exact := new(big.Rat).SetFrac(bigAtScale(acc, ws), bigPow10(ws))

				for range 8 {
					b := chainOperand(r, ws)
					bx := new(big.Rat).SetFrac(bigAtScale(b, ws), bigPow10(ws))
					var got Dec128
					next := new(big.Rat)

					switch r.Intn(4) {
					case 0:
						got, next = acc.Mul(b), next.Mul(exact, bx)
					case 1:
						got, next = acc.Add(b), next.Add(exact, bx)
					case 2:
						got, next = acc.Sub(b), next.Sub(exact, bx)
					default:
						if bx.Sign() == 0 {
							continue
						}
						got, next = acc.MulRound(b, ws, mode), next.Mul(exact, bx)
					}
					if got.IsNaN() {
						break
					}

					// |got - next| must be below one quantum at got's scale.
					gotRat := new(big.Rat).SetFrac(bigAtScale(got, got.Scale()), bigPow10(got.Scale()))
					diff := new(big.Rat).Abs(new(big.Rat).Sub(gotRat, next))
					quantum := new(big.Rat).SetFrac(big.NewInt(1), bigPow10(got.Scale()))
					if diff.Cmp(quantum) >= 0 {
						t.Fatalf("ws=%d %v: %s is %s away from the exact %s (quantum %s)",
							ws, mode, got.StringFixed(), diff.FloatString(25), next.FloatString(25), quantum.FloatString(25))
					}
					if got.Sign() != 0 && next.Sign() != 0 && got.Sign() != next.Sign() {
						t.Fatalf("ws=%d %v: sign of %s does not match exact %s", ws, mode, got.StringFixed(), next.FloatString(10))
					}
					// Carry the value the chain actually holds, not the unrounded ideal: the bound is on each
					// individual step, and compounding it across steps would only measure how many steps ran.
					acc, exact = got, gotRat
				}
			}
		}
	}
}
