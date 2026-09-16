package dec128

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/jokruger/dec128/state"
)

// Oracles for the global-free twins of Sum, SumSlice and Avg. The exact total of a list of terms is a rational, so
// the oracle is the same big.Rat one the fused operations use.

func ratSum(xs []Dec128) *big.Rat {
	total := new(big.Rat)
	for _, d := range xs {
		total.Add(total, bigOf(d))
	}
	return total
}

// TestSumRoundAgainstBig checks SumRound and SumSliceRound against the exact total, at a scale and a mode per call.
func TestSumRoundAgainstBig(t *testing.T) {
	r := rand.New(rand.NewSource(20260919))
	var overflows, inexacts int

	for range 20000 {
		xs := make([]Dec128, 1+r.Intn(8))
		for i := range xs {
			xs[i] = randDec(r)
		}
		scale := uint8(r.Intn(int(MaxScale) + 1))
		mode := allModes[r.Intn(len(allModes))]

		exact := ratSum(xs)
		got := SumRound(scale, mode, xs[0], xs[1:]...)
		checkMulDiv(t, "SumRound", got, exact, scale, mode)
		if got.IsNaN() {
			overflows++
		}
		if _, ok := roundRat(exact, scale, mode); !ok {
			inexacts++
		}

		if slice := SumSliceRound(xs, scale, mode); slice != got {
			t.Fatalf("SumSliceRound = %s, SumRound = %s", slice.StringFixed(), got.StringFixed())
		}
	}

	if overflows == 0 || inexacts == 0 {
		t.Errorf("operand mix no longer reaches the edges (nan=%d inexact=%d)", overflows, inexacts)
	}
}

// TestAvgRoundAgainstBig checks the mean against the exact total divided by the count - one rounding, not two.
func TestAvgRoundAgainstBig(t *testing.T) {
	r := rand.New(rand.NewSource(20260920))

	for range 20000 {
		xs := make([]Dec128, 1+r.Intn(8))
		for i := range xs {
			xs[i] = randDec(r)
		}
		scale := uint8(r.Intn(int(MaxScale) + 1))
		mode := allModes[r.Intn(len(allModes))]

		exact := ratSum(xs)
		exact.Quo(exact, new(big.Rat).SetInt64(int64(len(xs))))

		checkMulDiv(t, "AvgRound", AvgRound(scale, mode, xs[0], xs[1:]...), exact, scale, mode)
	}
}

// TestAvgRoundMatchesAccumulatorMean holds the one-shot mean and the running one to the same result, which is what
// the shared meanFromWide tail exists for: the two must not drift apart.
func TestAvgRoundMatchesAccumulatorMean(t *testing.T) {
	r := rand.New(rand.NewSource(20260921))

	for range 20000 {
		xs := make([]Dec128, 1+r.Intn(8))
		for i := range xs {
			xs[i] = randDec(r)
		}
		scale := uint8(r.Intn(int(MaxScale) + 1))
		mode := allModes[r.Intn(len(allModes))]

		acc := NewAccumulator(0)
		for _, d := range xs {
			acc.Add(d)
		}

		if got, want := AvgRound(scale, mode, xs[0], xs[1:]...), acc.Mean(scale, mode); got != want {
			t.Fatalf("AvgRound = %s (%v), Accumulator.Mean = %s (%v)",
				got.StringFixed(), got.state, want.StringFixed(), want.state)
		}
		if got, want := SumRound(scale, mode, xs[0], xs[1:]...), acc.Total(scale, mode); got != want {
			t.Fatalf("SumRound = %s (%v), Accumulator.Total = %s (%v)",
				got.StringFixed(), got.state, want.StringFixed(), want.state)
		}
	}
}

// TestSumRoundOrderIndependence pins the property the separate positive and negative registers are there for: the
// total is exact until the single rounding, so shuffling the terms cannot change a digit.
func TestSumRoundOrderIndependence(t *testing.T) {
	r := rand.New(rand.NewSource(20260922))

	for range 5000 {
		xs := make([]Dec128, 2+r.Intn(7))
		for i := range xs {
			xs[i] = randDec(r)
		}
		scale := uint8(r.Intn(int(MaxScale) + 1))
		mode := allModes[r.Intn(len(allModes))]
		want := SumSliceRound(xs, scale, mode)
		wantAvg := AvgRound(scale, mode, xs[0], xs[1:]...)

		shuffled := append([]Dec128(nil), xs...)
		r.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })

		if got := SumSliceRound(shuffled, scale, mode); got != want {
			t.Fatalf("SumSliceRound depends on order: %s vs %s", got.StringFixed(), want.StringFixed())
		}
		if got := AvgRound(scale, mode, shuffled[0], shuffled[1:]...); got != wantAvg {
			t.Fatalf("AvgRound depends on order: %s vs %s", got.StringFixed(), wantAvg.StringFixed())
		}
	}
}

// TestSumRoundContract pins the argument validation and the empty and NaN cases.
func TestSumRoundContract(t *testing.T) {
	one, two := FromInt64(1), FromInt64(2)
	nan := NaN(state.Overflow)

	cases := []struct {
		name string
		got  Dec128
		want state.State
	}{
		{"scale above MaxScale", SumRound(MaxScale+1, ROUND_BANK, one, two), state.ScaleOutOfRange},
		{"undefined mode", SumRound(2, RoundingMode(99), one, two), state.InvalidRoundingMode},
		{"first NaN wins", SumRound(2, ROUND_BANK, nan, NaN(state.Inexact)), state.Overflow},
		{"NaN in the tail", SumRound(2, ROUND_BANK, one, two, NaN(state.Inexact)), state.Inexact},
		{"slice scale above MaxScale", SumSliceRound([]Dec128{one}, MaxScale+1, ROUND_BANK), state.ScaleOutOfRange},
		{"slice undefined mode", SumSliceRound([]Dec128{one}, 2, RoundingMode(99)), state.InvalidRoundingMode},
		{"avg scale above MaxScale", AvgRound(MaxScale+1, ROUND_BANK, one, two), state.ScaleOutOfRange},
		{"avg undefined mode", AvgRound(2, RoundingMode(99), one, two), state.InvalidRoundingMode},
		{"avg NaN term", AvgRound(2, ROUND_BANK, one, nan), state.Overflow},
	}
	for _, c := range cases {
		if c.got.ErrorDetails() != c.want.Error() {
			t.Errorf("%s: got %v, want %v", c.name, c.got.ErrorDetails(), c.want.Error())
		}
	}

	// An empty slice is zero at the requested scale, not Zero at scale 0: the caller asked for a scale.
	empty := SumSliceRound(nil, 4, ROUND_BANK)
	if empty.IsNaN() || !empty.IsZero() || empty.scale != 4 {
		t.Errorf("SumSliceRound(nil, 4) = %s at scale %d, want 0.0000", empty.StringFixed(), empty.scale)
	}

	// Cancellation is exact and its zero is never negative.
	if got := SumRound(2, ROUND_BANK, FromString("1.5"), FromString("-1.50")); !got.IsZero() || got.state == state.Neg {
		t.Errorf("1.5 + -1.50 = %s state %v, want a non-negative zero", got.StringFixed(), got.state)
	}
}

// TestSumRoundMatchesSum pins Sum against the twins. Sum repeats sumTotal's body instead of calling it, to keep a
// measured 4% on its hot path (see the comment on sumTotal), so the two copies of the alignment-and-cancellation
// logic have to be held together by a test rather than by sharing code.
//
// Under the default configuration Sum returns the exact total at the aligned scale whenever it fits, which is
// exactly what SumRound returns when asked for that scale. Where it does not fit Sum applies the scale rule and the
// comparison does not apply, so those cases are counted rather than checked, to show the corpus reaches both.
func TestSumRoundMatchesSum(t *testing.T) {
	defer SetDefaultScale(DefaultScale())
	defer SetArithmeticRounding(ArithmeticRounding())
	defer SetLossPolicy(CurrentLossPolicy())

	SetDefaultScale(MaxScale)
	SetArithmeticRounding(ROUND_TOWARD_ZERO) // writes the loss policy too, so it goes first
	SetLossPolicy(LossRound)

	r := rand.New(rand.NewSource(20260927))
	var checked, reduced int

	for range 20000 {
		xs := make([]Dec128, 1+r.Intn(8))
		aligned := uint8(0)
		for i := range xs {
			xs[i] = randDec(r)
			aligned = max(aligned, xs[i].scale)
		}

		sum := Sum(xs[0], xs[1:]...)
		if sum.IsNaN() || sum.scale != aligned {
			reduced++ // the scale rule had to step in, so there is nothing to compare against
			continue
		}
		if got := SumRound(aligned, ROUND_TOWARD_ZERO, xs[0], xs[1:]...); got != sum {
			t.Fatalf("Sum = %s (scale %d), SumRound at the aligned scale = %s (scale %d)",
				sum.StringFixed(), sum.scale, got.StringFixed(), got.scale)
		}
		checked++
	}

	if checked == 0 || reduced == 0 {
		t.Errorf("the corpus no longer reaches both paths (compared=%d reduced=%d)", checked, reduced)
	}

	// The NaN term and its order are part of what the two copies must agree on.
	one, nan1, nan2 := FromInt64(1), NaN(state.Overflow), NaN(state.Inexact)
	if got, want := SumRound(2, ROUND_BANK, one, nan1, nan2).ErrorDetails(), Sum(one, nan1, nan2).ErrorDetails(); got != want {
		t.Errorf("NaN order: SumRound gives %v, Sum gives %v", got, want)
	}
	if got, want := SumRound(2, ROUND_BANK, nan2, nan1).ErrorDetails(), Sum(nan2, nan1).ErrorDetails(); got != want {
		t.Errorf("NaN order: SumRound gives %v, Sum gives %v", got, want)
	}
}
