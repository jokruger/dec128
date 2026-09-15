package dec128

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/jokruger/dec128/state"
)

// bigOf is the exact rational value of d as a big.Rat, for the accumulator oracles: a sum of products has no exact
// integer scale of its own until every term is known.
func bigOf(d Dec128) *big.Rat {
	return new(big.Rat).SetFrac(bigAtScale(d, d.scale), bigPow10(d.scale))
}

func ratAtScale(v *big.Rat, scale uint8) *big.Int {
	n := new(big.Int).Mul(v.Num(), bigPow10(scale))
	return n.Quo(n, v.Denom())
}

func TestAccumulatorAgainstBig(t *testing.T) {
	r := rand.New(rand.NewSource(20260922))
	for range 3000 {
		acc := NewAccumulator(uint8(r.Intn(int(MaxScale) + 1)))
		exact := new(big.Rat)
		scale := uint8(0)

		terms := 1 + r.Intn(8)
		for range terms {
			switch r.Intn(4) {
			case 0:
				d := randDec(r)
				acc.Add(d)
				exact.Add(exact, bigOf(d))
				scale = max(scale, d.scale)
			case 1:
				d := randDec(r)
				acc.Sub(d)
				exact.Sub(exact, bigOf(d))
				scale = max(scale, d.scale)
			case 2:
				x, y := randDec(r), randDec(r)
				acc.AddMul(x, y)
				exact.Add(exact, new(big.Rat).Mul(bigOf(x), bigOf(y)))
				scale = max(scale, x.scale+y.scale)
			default:
				x, y := randDec(r), randDec(r)
				acc.SubMul(x, y)
				exact.Sub(exact, new(big.Rat).Mul(bigOf(x), bigOf(y)))
				scale = max(scale, x.scale+y.scale)
			}
		}
		if acc.Count() != terms {
			t.Fatalf("Count = %d, want %d", acc.Count(), terms)
		}

		// The accumulator's working scale is the largest any term needed, but never below the one it was built with;
		// the exact total expressed there is an integer.
		scale = max(scale, acc.scale)
		want := ratAtScale(exact, scale)
		if acc.IsNaN() {
			// the only failure a random run can produce is a register that filled up
			if acc.ErrorDetails() == nil {
				t.Fatal("IsNaN with no error")
			}
			continue
		}

		mode := allModes[r.Intn(len(allModes))]
		target := uint8(r.Intn(int(MaxScale) + 1))
		checkAt(t, "Accumulator.Total", acc.Total(target, mode), want, scale, target, mode)
	}
}

// The post-condition the type exists for: the total does not depend on the order of the terms, and a sum of products
// is exact where the same sum with rounded terms is not.
func TestAccumulatorOrderIndependence(t *testing.T) {
	r := rand.New(rand.NewSource(20260923))
	for range 2000 {
		n := 2 + r.Intn(8)
		xs, ys := make([]Dec128, n), make([]Dec128, n)
		for i := range xs {
			xs[i], ys[i] = randDec(r), randDec(r)
		}

		acc := NewAccumulator(2)
		for i := range xs {
			acc.AddMul(xs[i], ys[i])
		}
		first := acc.Total(MaxScale, ROUND_BANK)

		perm := r.Perm(n)
		shuffled := NewAccumulator(2)
		for _, i := range perm {
			shuffled.AddMul(xs[i], ys[i])
		}
		if got := shuffled.Total(MaxScale, ROUND_BANK); got != first {
			t.Fatalf("order changed the total: %v vs %v", first, got)
		}
	}
}

func TestAccumulatorExactness(t *testing.T) {
	// A third of a cent, three hundred times: exact in the accumulator, lost term by term.
	third := FromString("0.0033333333333333333")
	acc := NewAccumulator(2)
	running := Zero
	for range 300 {
		acc.Add(third)
		running = running.AddRound(third, 2, ROUND_BANK)
	}
	if got := acc.Total(2, ROUND_BANK); got.StringFixed() != "1.00" {
		t.Errorf("accumulated = %s, want 1.00", got.StringFixed())
	}
	if running.StringFixed() != "0.00" {
		t.Errorf("term-by-term = %s, want 0.00 (this is the loss the accumulator avoids)", running.StringFixed())
	}

	// A sum of products stays exact even when no single product is representable.
	tiny := FromString("0.0000000001") // squares to 10^-20, below the smallest quantum
	acc = NewAccumulator(0)
	for range 100 {
		acc.AddMul(tiny, tiny)
	}
	if got := acc.Total(MaxScale, ROUND_HALF_AWAY_FROM_ZERO); got.StringFixed() != "0.0000000000000000010" {
		t.Errorf("sum of unrepresentable products = %s, want 0.0000000000000000010", got.StringFixed())
	}
	if got := FromInt64(100).Mul(tiny.Mul(tiny)); !got.IsZero() {
		t.Errorf("rounding first gives %s, expected the loss", got.StringFixed())
	}

	// Totalling does not consume the accumulator
	acc2 := NewAccumulator(2)
	acc2.Add(FromString("1.5"))
	if a, b := acc2.Total(2, ROUND_BANK), acc2.Total(2, ROUND_BANK); a != b {
		t.Errorf("Total is not repeatable: %v vs %v", a, b)
	}
	acc2.Add(FromString("1.5"))
	if got := acc2.Total(2, ROUND_BANK); got.StringFixed() != "3.00" {
		t.Errorf("continued = %s, want 3.00", got.StringFixed())
	}
}

func TestAccumulatorZeroValueAndScale(t *testing.T) {
	// the zero value is an empty accumulator at working scale 0
	var acc Accumulator
	if acc.Count() != 0 || acc.IsNaN() || acc.Total(0, ROUND_BANK) != Zero {
		t.Error("the zero value must be an empty accumulator")
	}
	acc.Add(FromString("1.25"))
	if got := acc.Total(2, ROUND_BANK); got.StringFixed() != "1.25" {
		t.Errorf("zero value total = %s", got.StringFixed())
	}

	// the constructor's scale is a floor, so the total pads rather than truncates
	acc4 := NewAccumulator(4)
	acc4.Add(One)
	if got := acc4.Total(4, ROUND_BANK); got.StringFixed() != "1.0000" {
		t.Errorf("floor scale total = %s", got.StringFixed())
	}
	if acc4.scale != 4 {
		t.Errorf("working scale = %d, want 4", acc4.scale)
	}
	// ... and a term that needs more places raises it
	acc4.Add(FromString("0.0000000001"))
	if acc4.scale != 10 {
		t.Errorf("working scale after a deeper term = %d, want 10", acc4.scale)
	}

	if got := NewAccumulator(MaxScale+1).Total(0, ROUND_BANK); got.state != state.ScaleOutOfRange {
		t.Errorf("NewAccumulator above MaxScale = %v", got)
	}
}

func TestAccumulatorFailures(t *testing.T) {
	// a NaN term is sticky and the first one wins
	acc := NewAccumulator(2)
	acc.Add(One)
	acc.Add(NaN(state.DivisionByZero))
	acc.Add(NaN(state.Overflow))
	acc.AddMul(NaN(state.NotConverged), One)
	if !acc.IsNaN() || acc.ErrorDetails() != state.DivisionByZero.Error() {
		t.Errorf("sticky NaN = %v", acc.ErrorDetails())
	}
	if got := acc.Total(2, ROUND_BANK); got.state != state.DivisionByZero {
		t.Errorf("Total after a NaN term = %v", got)
	}
	if acc.Count() != 4 {
		t.Errorf("Count = %d, want 4 (failed terms still count)", acc.Count())
	}

	// each operand of AddMul is checked, in order
	for _, tc := range []struct {
		what string
		run  func(*Accumulator)
		want state.State
	}{
		{"AddMul x", func(a *Accumulator) { a.AddMul(NaN(state.Underflow), One) }, state.Underflow},
		{"AddMul y", func(a *Accumulator) { a.AddMul(One, NaN(state.DomainError)) }, state.DomainError},
		{"Sub", func(a *Accumulator) { a.Sub(NaN(state.Inexact)) }, state.Inexact},
		{"SubMul", func(a *Accumulator) { a.SubMul(One, NaN(state.NotConverged)) }, state.NotConverged},
	} {
		a := NewAccumulator(0)
		tc.run(a)
		if a.Total(0, ROUND_BANK).state != tc.want {
			t.Errorf("%s = %v, want NaN(%s)", tc.what, a.Total(0, ROUND_BANK), tc.want)
		}
	}

	// argument errors on Total
	acc = NewAccumulator(2)
	acc.Add(One)
	if got := acc.Total(MaxScale+1, ROUND_BANK); got.state != state.ScaleOutOfRange {
		t.Errorf("Total scale = %v", got)
	}
	if got := acc.Total(2, RoundingMode(99)); got.state != state.InvalidRoundingMode {
		t.Errorf("Total mode = %v", got)
	}
	// a total too large for a coefficient at the requested scale
	acc = NewAccumulator(0)
	acc.Add(MaxAtScale(0))
	acc.Add(MaxAtScale(0))
	if got := acc.Total(0, ROUND_BANK); got.state != state.Overflow {
		t.Errorf("Total overflow = %v", got)
	}
	// ... but it is still exactly representable in the register, so a NaN never appears until Total
	if acc.IsNaN() {
		t.Error("the register held the sum; only Total should fail")
	}
}

// Filling the register is the accumulator's own overflow, and it is sticky. The largest term there is - the square
// of the largest coefficient, aligned to twice MaxScale - is just under a quarter of the register, so four of them
// fill it.
func TestAccumulatorRegisterOverflow(t *testing.T) {
	m := MaxAtScale(0)
	quantum := QuantumAtScale(MaxScale)

	// the add path: put the working scale at 2*MaxScale first, then add the widest terms there are
	acc := NewAccumulator(0)
	acc.AddMul(quantum, quantum)
	if acc.scale != 2*MaxScale {
		t.Fatalf("working scale = %d, want %d", acc.scale, 2*MaxScale)
	}
	for range 3 {
		acc.AddMul(m, m)
	}
	if acc.IsNaN() {
		t.Fatalf("three terms must still fit: %v", acc.ErrorDetails())
	}
	acc.AddMul(m, m)
	if !acc.IsNaN() || acc.ErrorDetails() != state.Overflow.Error() {
		t.Fatalf("the register must saturate rather than wrap: %v", acc.ErrorDetails())
	}
	// once failed it stays failed, and further terms change nothing
	acc.Add(One)
	if got := acc.Total(0, ROUND_BANK); got.state != state.Overflow {
		t.Errorf("Total after a full register = %v", got)
	}

	// the growth path: a nearly full register at scale 0, raised by 2*MaxScale places
	acc = NewAccumulator(0)
	for range 4 {
		acc.AddMul(m, m)
	}
	if acc.IsNaN() {
		t.Fatalf("four terms at scale 0 must fit: %v", acc.ErrorDetails())
	}
	acc.AddMul(quantum, quantum)
	if !acc.IsNaN() || acc.ErrorDetails() != state.Overflow.Error() {
		t.Errorf("growing a full register must overflow: %v", acc.ErrorDetails())
	}

	// ... and the same on the negative side, which grow checks separately
	acc = NewAccumulator(0)
	for range 4 {
		acc.SubMul(m, m)
	}
	acc.AddMul(quantum, quantum)
	if !acc.IsNaN() || acc.ErrorDetails() != state.Overflow.Error() {
		t.Errorf("growing a full negative register must overflow: %v", acc.ErrorDetails())
	}

	// and the negative add path
	acc = NewAccumulator(0)
	acc.AddMul(quantum, quantum)
	for range 4 {
		acc.SubMul(m, m)
	}
	if !acc.IsNaN() || acc.ErrorDetails() != state.Overflow.Error() {
		t.Errorf("a full negative register must overflow: %v", acc.ErrorDetails())
	}
}

func TestAccumulatorErrorDetailsWhenHealthy(t *testing.T) {
	acc := NewAccumulator(2)
	acc.Add(One)
	if err := acc.ErrorDetails(); err != nil {
		t.Errorf("ErrorDetails on a healthy accumulator = %v", err)
	}
}

// Mean divides the exact total by the number of terms and brings the quotient to the requested scale, and the two
// together take one rounding decision. The oracle is the exact rational mean rounded once.
func TestAccumulatorMeanAgainstBig(t *testing.T) {
	r := rand.New(rand.NewSource(20261023))
	for range 4000 {
		acc := NewAccumulator(uint8(r.Intn(int(MaxScale) + 1)))
		exact := new(big.Rat)
		terms := 1 + r.Intn(8)
		for range terms {
			if r.Intn(3) == 0 {
				x, y := randDec(r), randDec(r)
				acc.AddMul(x, y)
				exact.Add(exact, new(big.Rat).Mul(bigOf(x), bigOf(y)))
			} else {
				d := randDec(r)
				acc.Add(d)
				exact.Add(exact, bigOf(d))
			}
		}
		if acc.IsNaN() {
			continue
		}
		exact.Quo(exact, new(big.Rat).SetInt64(int64(terms)))

		scale := uint8(r.Intn(int(MaxScale) + 1))
		mode := allModes[r.Intn(len(allModes))]
		got := acc.Mean(scale, mode)

		// the correctly rounded mean at that scale, from the rational
		num := new(big.Int).Mul(exact.Num(), bigPow10(scale))
		q, rem := new(big.Int).QuoRem(new(big.Int).Abs(num), exact.Denom(), new(big.Int))
		want := roundBig(q, rem, exact.Denom(), exact.Sign() < 0, mode)

		// A result that does not fit is an overflow whatever else is true of it: that is the order reduceAt
		// reports in and wantAt expects.
		switch {
		case want.Cmp(big2p128) >= 0:
			if got.state != state.Overflow {
				t.Fatalf("Mean(%d, %s) = %v, want NaN(Overflow)", scale, mode, got)
			}
		case mode == ROUND_NAN && rem.Sign() != 0:
			if got.state != state.Inexact {
				t.Fatalf("Mean(%d, ROUND_NAN) = %v, want NaN(Inexact) for %s", scale, got, exact.FloatString(25))
			}
		case got.IsNaN():
			t.Fatalf("Mean(%d, %s) = NaN(%v), want %s (exact %s)",
				scale, mode, got.ErrorDetails(), want, exact.FloatString(25))
		case got.scale != scale:
			t.Fatalf("Mean(%d, %s) = %s at scale %d", scale, mode, got.StringFixed(), got.scale)
		case new(big.Int).Abs(bigAtScale(got, scale)).Cmp(want) != 0:
			t.Fatalf("Mean(%d, %s) = %s, want coefficient %s (exact %s)",
				scale, mode, got.StringFixed(), want, exact.FloatString(25))
		case (got.state == state.Neg) != (exact.Sign() < 0 && want.Sign() != 0):
			t.Fatalf("Mean(%d, %s) = %s: wrong sign", scale, mode, got.StringFixed())
		}
	}
}

func TestAccumulatorMeanEdges(t *testing.T) {
	// the mean of nothing is not zero
	empty := NewAccumulator(2)
	if got := empty.Mean(2, ROUND_BANK); got.state != state.DivisionByZero {
		t.Errorf("the mean of an empty accumulator = %v, want NaN(DivisionByZero)", got)
	}

	acc := NewAccumulator(2)
	acc.Add(One)
	acc.Add(FromInt64(2))
	acc.Add(FromInt64(4))
	for _, c := range []struct {
		scale uint8
		mode  RoundingMode
		want  string
	}{
		{MaxScale, ROUND_TOWARD_ZERO, "2.3333333333333333333"},
		{2, ROUND_BANK, "2.33"},
		{2, ROUND_AWAY_FROM_ZERO, "2.34"},
		{0, ROUND_BANK, "2"},
		{0, ROUND_AWAY_FROM_ZERO, "3"},
		{4, ROUND_HALF_AWAY_FROM_ZERO, "2.3333"},
	} {
		if got := acc.Mean(c.scale, c.mode); got.StringFixed() != c.want {
			t.Errorf("Mean(%d, %s) = %s, want %s", c.scale, c.mode, got.StringFixed(), c.want)
		}
	}

	// an exact mean is exact under ROUND_NAN, an inexact one is refused
	exact := NewAccumulator(0)
	exact.Add(One)
	exact.Add(FromInt64(3))
	if got := exact.Mean(0, ROUND_NAN); got.StringFixed() != "2" {
		t.Errorf("an exact mean under ROUND_NAN = %v", got)
	}
	if got := acc.Mean(2, ROUND_NAN); got.state != state.Inexact {
		t.Errorf("an inexact mean under ROUND_NAN = %v", got)
	}

	// a mean at a finer scale than the terms is padded before the division, so the places are produced, not invented
	half := NewAccumulator(0)
	half.Add(One)
	half.Add(Zero)
	if got := half.Mean(MaxScale, ROUND_BANK); got.StringFixed() != "0.5000000000000000000" {
		t.Errorf("Mean at a finer scale = %s", got.StringFixed())
	}

	// the sign of a negative mean, and a mean that cancels to zero
	neg := NewAccumulator(2)
	neg.Sub(One)
	neg.Sub(FromInt64(2))
	if got := neg.Mean(2, ROUND_BANK); got.StringFixed() != "-1.50" {
		t.Errorf("a negative mean = %s", got.StringFixed())
	}
	cancel := NewAccumulator(2)
	cancel.Add(One)
	cancel.Sub(One)
	if got := cancel.Mean(4, ROUND_BANK); got.StringFixed() != "0.0000" || got.state != state.Default {
		t.Errorf("a cancelling mean = %v, state %s", got, got.state)
	}

	// the argument checks, and a failure recorded earlier
	if got := acc.Mean(MaxScale+1, ROUND_BANK); got.state != state.ScaleOutOfRange {
		t.Errorf("Mean with a bad scale = %v", got)
	}
	if got := acc.Mean(2, RoundingMode(99)); got.state != state.InvalidRoundingMode {
		t.Errorf("Mean with a bad mode = %v", got)
	}
	failed := NewAccumulator(2)
	failed.Add(NaN(state.DomainError))
	if got := failed.Mean(2, ROUND_BANK); got.state != state.DomainError {
		t.Errorf("Mean of a failed accumulator = %v", got)
	}
	if got := NewAccumulator(MaxScale+1).Mean(2, ROUND_BANK); got.state != state.ScaleOutOfRange {
		t.Errorf("Mean of an accumulator born out of range = %v", got)
	}

	// a total whose mean does not fit a coefficient at the requested scale
	full := NewAccumulator(0)
	for range 3 {
		full.AddMul(MaxAtScale(0), MaxAtScale(0))
	}
	if got := full.Mean(MaxScale, ROUND_BANK); got.state != state.Overflow {
		t.Errorf("the mean of a full register at a fine scale = %v", got)
	}

	// A mean that fits when truncated but carries past the coefficient when rounded up. The terms are one product
	// of three times the largest value and two ones, so the exact mean is the largest value and two thirds.
	carry := NewAccumulator(0)
	carry.AddMul(FromInt64(3), MaxAtScale(0))
	carry.Add(One)
	carry.Add(One)
	if got := carry.Mean(0, ROUND_TOWARD_ZERO); !got.Equal(MaxAtScale(0)) {
		t.Errorf("the truncated mean = %v, want the largest value", got)
	}
	for _, mode := range []RoundingMode{ROUND_HALF_AWAY_FROM_ZERO, ROUND_BANK, ROUND_UP, ROUND_AWAY_FROM_ZERO} {
		if got := carry.Mean(0, mode); got.state != state.Overflow {
			t.Errorf("the mean rounded up under %s = %v, want NaN(Overflow)", mode, got)
		}
	}

	// Mean and Total agree where the count is one, and Mean does not disturb the running total
	one := NewAccumulator(2)
	one.Add(FromString("1.23"))
	if got, want := one.Mean(2, ROUND_BANK), one.Total(2, ROUND_BANK); got != want {
		t.Errorf("the mean of one term = %v, Total = %v", got, want)
	}
	if got := one.Mean(2, ROUND_BANK); got.StringFixed() != "1.23" {
		t.Errorf("Mean changed the running total: %s", got.StringFixed())
	}
	// a term added with AddMul counts as one term
	prod := NewAccumulator(0)
	prod.AddMul(FromInt64(3), FromInt64(4))
	prod.AddMul(FromInt64(1), FromInt64(2))
	if got := prod.Mean(0, ROUND_BANK); got.StringFixed() != "7" {
		t.Errorf("the mean of two products = %s, want 7", got.StringFixed())
	}
}

func TestAccumulatorReset(t *testing.T) {
	acc := NewAccumulator(4)
	acc.Add(FromString("1.5"))
	acc.AddMul(FromString("0.0000000001"), FromString("0.0000000001"))
	if acc.scale != 20 {
		t.Fatalf("the working scale should have grown to 20, got %d", acc.scale)
	}

	acc.Reset()
	if acc.Count() != 0 || acc.IsNaN() || acc.scale != 4 {
		t.Errorf("after Reset: count %d, NaN %v, scale %d, want 0, false, 4", acc.Count(), acc.IsNaN(), acc.scale)
	}
	if got := acc.Total(4, ROUND_BANK); got.StringFixed() != "0.0000" {
		t.Errorf("the total after Reset = %s", got.StringFixed())
	}

	// a reused accumulator gives what a fresh one would
	acc.Add(FromString("2.5"))
	acc.Add(FromString("1.25"))
	fresh := NewAccumulator(4)
	fresh.Add(FromString("2.5"))
	fresh.Add(FromString("1.25"))
	if got, want := acc.Total(4, ROUND_BANK), fresh.Total(4, ROUND_BANK); got != want {
		t.Errorf("a reused accumulator gave %v, a fresh one %v", got, want)
	}
	if got, want := acc.Mean(4, ROUND_BANK), fresh.Mean(4, ROUND_BANK); got != want {
		t.Errorf("a reused accumulator's mean %v, a fresh one's %v", got, want)
	}

	// Reset clears a failure that a term caused
	acc.Add(NaN(state.Overflow))
	if !acc.IsNaN() {
		t.Fatal("the accumulator should have failed")
	}
	acc.Reset()
	if acc.IsNaN() {
		t.Error("Reset must clear a failure a term caused")
	}
	acc.Add(One)
	if got := acc.Total(0, ROUND_BANK); got.StringFixed() != "1" {
		t.Errorf("the total after resetting a failure = %v", got)
	}

	// ... but not the one an impossible scale caused, which emptying it cannot fix
	born := NewAccumulator(MaxScale + 1)
	born.Reset()
	if !born.IsNaN() || born.ErrorDetails() != state.ScaleOutOfRange.Error() {
		t.Errorf("Reset on an accumulator born out of range = %v", born.ErrorDetails())
	}

	// the zero value resets to the zero value
	var zero Accumulator
	zero.Add(One)
	zero.Reset()
	if zero.Count() != 0 || zero.scale != 0 || zero.IsNaN() {
		t.Errorf("the zero value after Reset: count %d, scale %d, NaN %v", zero.Count(), zero.scale, zero.IsNaN())
	}
}

func TestSumSlice(t *testing.T) {
	// the reason it exists: Sum takes its first term separately, so an empty slice has nowhere to come from
	if got := SumSlice(nil); !got.Equal(Zero) || got.IsNaN() {
		t.Errorf("SumSlice(nil) = %v, want zero", got)
	}
	if got := SumSlice([]Dec128{}); !got.Equal(Zero) {
		t.Errorf("SumSlice(empty) = %v, want zero", got)
	}

	r := rand.New(rand.NewSource(20261024))
	for range 20000 {
		xs := make([]Dec128, 1+r.Intn(8))
		for i := range xs {
			xs[i] = randDec(r)
		}
		if got, want := SumSlice(xs), Sum(xs[0], xs[1:]...); got != want {
			t.Fatalf("SumSlice = %v, Sum = %v", got, want)
		}
	}
}
