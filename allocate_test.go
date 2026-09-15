package dec128

import (
	"math/big"
	"math/rand"
	"slices"
	"strings"
	"testing"

	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// randCoef returns a coefficient with up to the given number of decimal digits.
func randCoef(r *rand.Rand, digits int) uint128.Uint128 {
	c := uint128.Uint128{Lo: r.Uint64(), Hi: r.Uint64()}
	q, _, _, _ := wideFrom128(c).quoCmpHalfPow10(uint8(39 - r.Intn(digits+1)))
	c, _ = q.uint128()
	return c
}

// sumShares is the assertion the whole file exists for: the parts add up to the whole, exactly, with no epsilon.
func sumShares(t *testing.T, shares []Dec128, want Dec128, what string) {
	t.Helper()
	var acc Accumulator
	for _, s := range shares {
		acc.Add(s)
	}
	got := acc.Total(want.scale, ROUND_TOWARD_ZERO)
	if !got.Equal(want) {
		t.Fatalf("%s: shares sum to %s, want %s", what, got.StringFixed(), want.StringFixed())
	}
}

func TestAllocateExamples(t *testing.T) {
	// The canonical one: a penny that will not divide three ways.
	shares, ok := FromString("0.05").Split(3, 2)
	if !ok {
		t.Fatal("Split failed")
	}
	if len(shares) != 3 || shares[0].StringFixed() != "0.02" || shares[1].StringFixed() != "0.02" || shares[2].StringFixed() != "0.01" {
		t.Errorf("0.05 split three ways = %v", shares)
	}
	sumShares(t, shares, FromString("0.05"), "penny split")

	// A fee split by weights, where the largest remainder decides the odd cent.
	amount := FromString("100.00")
	ratios := []Dec128{FromInt64(1), FromInt64(1), FromInt64(1)}
	shares, ok = amount.Allocate(ratios, 2)
	if !ok || shares[0].StringFixed() != "33.34" || shares[1].StringFixed() != "33.33" || shares[2].StringFixed() != "33.33" {
		t.Errorf("100.00 three ways = %v", shares)
	}
	sumShares(t, shares, amount, "three-way fee")

	// Weighted, with the weights at different scales.
	shares, ok = FromString("1000.00").Allocate([]Dec128{FromString("0.5"), FromString("0.25"), FromString("0.25")}, 2)
	if !ok || shares[0].StringFixed() != "500.00" || shares[1].StringFixed() != "250.00" || shares[2].StringFixed() != "250.00" {
		t.Errorf("weighted split = %v", shares)
	}

	// A negative amount splits the same way, with every share negative.
	shares, ok = FromString("-0.05").Split(3, 2)
	if !ok || shares[0].StringFixed() != "-0.02" || shares[2].StringFixed() != "-0.01" {
		t.Errorf("negative split = %v", shares)
	}
	sumShares(t, shares, FromString("-0.05"), "negative split")

	// A zero weight gets nothing, and the zero is not negative.
	shares, ok = FromString("-1.00").Allocate([]Dec128{One, Zero}, 2)
	if !ok || shares[1].StringFixed() != "0.00" || shares[1].state != state.Default {
		t.Errorf("zero weight = %v (state %s)", shares, shares[1].state)
	}

	// Allocating at a finer scale than the amount's is allowed; the extra places are zeros.
	shares, ok = FromString("1.00").Allocate([]Dec128{One, FromInt64(2)}, 4)
	if !ok || shares[0].StringFixed() != "0.3333" || shares[1].StringFixed() != "0.6667" {
		t.Errorf("finer scale = %v", shares)
	}
	sumShares(t, shares, FromString("1.00"), "finer scale")
}

func TestAllocateRejects(t *testing.T) {
	d := FromString("1.00")
	for _, c := range []struct {
		what   string
		amount Dec128
		ratios []Dec128
		scale  uint8
	}{
		{"no ratios", d, nil, 2},
		{"NaN amount", NaN(state.Overflow), []Dec128{One}, 2},
		{"NaN ratio", d, []Dec128{One, NaN(state.Overflow)}, 2},
		{"negative ratio", d, []Dec128{One, FromInt64(-1)}, 2},
		{"all ratios zero", d, []Dec128{Zero, Zero}, 2},
		{"scale above MaxScale", d, []Dec128{One}, MaxScale + 1},
		{"scale below the amount's", d, []Dec128{One}, 1},
		{"amount too wide for the scale", MaxAtScale(0), []Dec128{One}, 2},
		{"ratio too wide at the common scale", d, []Dec128{MaxAtScale(0), QuantumAtScale(MaxScale)}, 2},
		{"ratios total too wide", d, []Dec128{MaxAtScale(0), MaxAtScale(0)}, 2},
	} {
		if got, ok := c.amount.Allocate(c.ratios, c.scale); ok || got != nil {
			t.Errorf("%s: got %v, ok %v", c.what, got, ok)
		}
	}

	for _, c := range []struct {
		what  string
		d     Dec128
		n     int
		scale uint8
	}{
		{"zero parts", d, 0, 2},
		{"negative parts", d, -3, 2},
		{"NaN", NaN(state.Overflow), 3, 2},
		{"scale below the amount's", d, 3, 1},
	} {
		if got, ok := c.d.Split(c.n, c.scale); ok || got != nil {
			t.Errorf("%s: got %v, ok %v", c.what, got, ok)
		}
	}

	// a failed split leaves the caller's slice alone
	dst := []Dec128{One}
	if got, ok := d.AppendAllocate(dst, nil, 2); ok || len(got) != 1 || got[0] != One {
		t.Errorf("AppendAllocate must not touch dst on failure: %v", got)
	}
	if got, ok := d.AppendSplit(dst, 0, 2); ok || len(got) != 1 {
		t.Errorf("AppendSplit must not touch dst on failure: %v", got)
	}
}

// The invariants: the shares sum to the amount exactly, each is within one quantum of its exact proportion, and the
// tie rule makes the outcome depend on nothing but the ratios.
func TestAllocateInvariants(t *testing.T) {
	r := rand.New(rand.NewSource(20261003))
	buf := make([]Dec128, 0, 64)

	for range 20000 {
		scale := uint8(r.Intn(int(MaxScale) + 1))
		amount := Dec128{
			coef:  randCoef(r, 38),
			scale: uint8(r.Intn(int(scale) + 1)),
		}
		if r.Intn(2) == 0 && !amount.coef.IsZero() {
			amount.state = state.Neg
		}

		n := 1 + r.Intn(40)
		ratios := make([]Dec128, n)
		for i := range ratios {
			ratios[i] = Dec128{
				coef:  randCoef(r, 12),
				scale: uint8(r.Intn(4)),
			}
		}

		buf = buf[:0]
		shares, ok := amount.AppendAllocate(buf, ratios, scale)
		if !ok {
			continue
		}
		if len(shares) != n {
			t.Fatalf("got %d shares for %d ratios", len(shares), n)
		}
		sumShares(t, shares, amount, "random allocation")

		// running it again must give the same answer, whatever slice it appends to
		again, _ := amount.Allocate(ratios, scale)
		for i := range shares {
			if shares[i] != again[i] {
				t.Fatalf("allocation is not deterministic at index %d: %v vs %v", i, shares[i], again[i])
			}
		}

		// no share is more than one quantum from its exact proportion
		total := Sum(ratios[0], ratios[1:]...)
		for i := range shares {
			exact := amount.MulRound(ratios[i], MaxScale, ROUND_TOWARD_ZERO).DivRound(total, scale, ROUND_TOWARD_ZERO)
			if exact.IsNaN() {
				continue
			}
			if diff := shares[i].SubRound(exact, scale, ROUND_TOWARD_ZERO).Abs(); diff.Compare(QuantumAtScale(scale)) > 0 {
				t.Fatalf("share %d is %s, exact proportion is about %s: off by %s",
					i, shares[i].StringFixed(), exact.StringFixed(), diff.StringFixed())
			}
		}
	}
}

func TestSplitInvariants(t *testing.T) {
	r := rand.New(rand.NewSource(20261004))
	for range 20000 {
		scale := uint8(r.Intn(int(MaxScale) + 1))
		amount := Dec128{coef: randCoef(r, 30), scale: uint8(r.Intn(int(scale) + 1))}
		if r.Intn(2) == 0 && !amount.coef.IsZero() {
			amount.state = state.Neg
		}
		n := 1 + r.Intn(50)

		shares, ok := amount.Split(n, scale)
		if !ok {
			continue
		}
		sumShares(t, shares, amount, "random split")

		// the shares differ by at most one quantum, and the larger ones come first
		for i := 1; i < len(shares); i++ {
			if shares[i-1].Abs().Compare(shares[i].Abs()) < 0 {
				t.Fatalf("share %d is larger than share %d: %v", i, i-1, shares)
			}
			if d := shares[i-1].Abs().SubRound(shares[i].Abs(), scale, ROUND_TOWARD_ZERO); d.Compare(QuantumAtScale(scale)) > 0 {
				t.Fatalf("shares %d and %d differ by %s", i-1, i, d.StringFixed())
			}
		}
	}
}

// Beyond the stack scratch the method takes one allocation for its working set and must still be correct.
func TestAllocateManyWays(t *testing.T) {
	ratios := make([]Dec128, allocScratch*3)
	for i := range ratios {
		ratios[i] = FromInt64(int64(i%7) + 1)
	}
	amount := FromString("1000000.00")
	shares, ok := amount.Allocate(ratios, 2)
	if !ok || len(shares) != len(ratios) {
		t.Fatalf("wide allocation failed: ok %v, %d shares", ok, len(shares))
	}
	sumShares(t, shares, amount, "96-way allocation")
}

// The residual convention: every share but one is its proportion rounded the agreed way, and the named one takes what
// is left. What has to hold is the same post-condition as Allocate - the shares sum to the whole exactly - plus the
// stronger claim about the others, that each is its own proportion under the mode.
func TestAllocateResidualExamples(t *testing.T) {
	for _, c := range []struct {
		amount string
		n, res int
		mode   RoundingMode
		want   string
	}{
		// the odd cent goes where it is told, not to the largest remainder
		{"100.00", 3, 2, ROUND_HALF_AWAY_FROM_ZERO, "33.33 33.33 33.34"},
		{"100.00", 3, 0, ROUND_HALF_AWAY_FROM_ZERO, "33.34 33.33 33.33"},
		{"100.00", 3, 1, ROUND_HALF_AWAY_FROM_ZERO, "33.33 33.34 33.33"},
		{"100.00", 3, 2, ROUND_TOWARD_ZERO, "33.33 33.33 33.34"},
		{"100.00", 3, 2, ROUND_UP, "33.34 33.34 33.32"},
		// the last installment of a schedule absorbs the rounding of the others
		{"1000.00", 7, 6, ROUND_HALF_AWAY_FROM_ZERO, "142.86 142.86 142.86 142.86 142.86 142.86 142.84"},
		{"1000.00", 7, 6, ROUND_TOWARD_ZERO, "142.85 142.85 142.85 142.85 142.85 142.85 142.90"},
		// a few quanta split many ways, where the others round away from zero and the residual turns negative
		{"0.01", 3, 2, ROUND_UP, "0.01 0.01 -0.01"},
		{"0.01", 3, 2, ROUND_TOWARD_ZERO, "0.00 0.00 0.01"},
		{"0.05", 3, 2, ROUND_TOWARD_ZERO, "0.01 0.01 0.03"},
		{"-0.05", 3, 0, ROUND_BANK, "-0.01 -0.02 -0.02"},
		{"-0.01", 3, 0, ROUND_DOWN, "0.01 -0.01 -0.01"},
		// one share is the whole
		{"1.23", 1, 0, ROUND_BANK, "1.23"},
	} {
		shares, ok := FromString(c.amount).SplitResidual(c.n, 2, c.res, c.mode)
		if !ok {
			t.Errorf("SplitResidual(%s, %d, %d, %s) failed", c.amount, c.n, c.res, c.mode)
			continue
		}
		got := make([]string, len(shares))
		for i, s := range shares {
			got[i] = s.StringFixed()
		}
		if strings.Join(got, " ") != c.want {
			t.Errorf("SplitResidual(%s, %d, res %d, %s) = %s, want %s",
				c.amount, c.n, c.res, c.mode, strings.Join(got, " "), c.want)
		}
		sumShares(t, shares, FromString(c.amount), c.amount)
	}

	// AllocateResidual with equal ratios is SplitResidual
	for _, amount := range []string{"100.00", "0.01", "-0.05", "1000.00"} {
		for res := range 3 {
			for _, mode := range allModes {
				if mode == ROUND_NAN {
					continue
				}
				a, ok1 := FromString(amount).SplitResidual(3, 2, res, mode)
				b, ok2 := FromString(amount).AllocateResidual([]Dec128{One, One, One}, 2, res, mode)
				if ok1 != ok2 {
					t.Fatalf("%s: SplitResidual ok %v, AllocateResidual ok %v", amount, ok1, ok2)
				}
				if ok1 && !slices.Equal(a, b) {
					t.Fatalf("%s under %s res %d: SplitResidual %v, AllocateResidual %v", amount, mode, res, a, b)
				}
			}
		}
	}

	// weights, at a finer scale than the amount
	shares, ok := FromString("1000.00").AllocateResidual(
		[]Dec128{FromString("0.5"), FromString("0.3"), FromString("0.2")}, 2, 2, ROUND_HALF_AWAY_FROM_ZERO)
	if !ok || shares[0].StringFixed() != "500.00" || shares[1].StringFixed() != "300.00" || shares[2].StringFixed() != "200.00" {
		t.Errorf("weighted residual allocation = %v", shares)
	}
	// a weight of zero gets nothing and is not a negative zero
	shares, ok = FromString("-1.00").AllocateResidual([]Dec128{One, Zero}, 2, 0, ROUND_BANK)
	if !ok || shares[1].StringFixed() != "0.00" || shares[1].state != state.Default {
		t.Errorf("zero weight = %v (state %s)", shares, shares[1].state)
	}
}

func TestAllocateResidualRejects(t *testing.T) {
	d := FromString("1.00")
	ratios := []Dec128{One, One}

	for _, c := range []struct {
		what string
		run  func() ([]Dec128, bool)
	}{
		{"residual below range", func() ([]Dec128, bool) { return d.AllocateResidual(ratios, 2, -1, ROUND_BANK) }},
		{"residual above range", func() ([]Dec128, bool) { return d.AllocateResidual(ratios, 2, 2, ROUND_BANK) }},
		{"no ratios", func() ([]Dec128, bool) { return d.AllocateResidual(nil, 2, 0, ROUND_BANK) }},
		{"undefined mode", func() ([]Dec128, bool) { return d.AllocateResidual(ratios, 2, 0, RoundingMode(99)) }},
		{"NaN amount", func() ([]Dec128, bool) { return NaN(state.Overflow).AllocateResidual(ratios, 2, 0, ROUND_BANK) }},
		{"NaN ratio", func() ([]Dec128, bool) {
			return d.AllocateResidual([]Dec128{One, NaN(state.Overflow)}, 2, 0, ROUND_BANK)
		}},
		{"negative ratio", func() ([]Dec128, bool) {
			return d.AllocateResidual([]Dec128{One, NegativeOne}, 2, 0, ROUND_BANK)
		}},
		{"zero ratios", func() ([]Dec128, bool) { return d.AllocateResidual([]Dec128{Zero, Zero}, 2, 0, ROUND_BANK) }},
		{"scale above MaxScale", func() ([]Dec128, bool) { return d.AllocateResidual(ratios, MaxScale+1, 0, ROUND_BANK) }},
		{"scale below the amount's", func() ([]Dec128, bool) { return d.AllocateResidual(ratios, 1, 0, ROUND_BANK) }},
		{"ROUND_NAN on an inexact split", func() ([]Dec128, bool) {
			return FromString("1.00").AllocateResidual([]Dec128{One, One, One}, 2, 0, ROUND_NAN)
		}},
		{"split: n zero", func() ([]Dec128, bool) { return d.SplitResidual(0, 2, 0, ROUND_BANK) }},
		{"split: n negative", func() ([]Dec128, bool) { return d.SplitResidual(-3, 2, 0, ROUND_BANK) }},
		{"split: residual out of range", func() ([]Dec128, bool) { return d.SplitResidual(3, 2, 3, ROUND_BANK) }},
		{"split: undefined mode", func() ([]Dec128, bool) { return d.SplitResidual(3, 2, 0, RoundingMode(99)) }},
		{"split: ROUND_NAN on an inexact split", func() ([]Dec128, bool) {
			return FromString("1.00").SplitResidual(3, 2, 0, ROUND_NAN)
		}},
	} {
		if got, ok := c.run(); ok {
			t.Errorf("%s: ok is true, shares %v", c.what, got)
		}
	}

	// ROUND_NAN accepts a split that is exact
	if got, ok := FromString("0.99").SplitResidual(3, 2, 0, ROUND_NAN); !ok || got[0].StringFixed() != "0.33" {
		t.Errorf("ROUND_NAN on an exact split = %v, ok %v", got, ok)
	}
	if got, ok := FromString("1.00").AllocateResidual([]Dec128{One, One, FromInt64(2)}, 2, 0, ROUND_NAN); !ok || len(got) != 3 {
		t.Errorf("ROUND_NAN on an exact allocation = %v, ok %v", got, ok)
	}

	// dst is handed back untouched when the split fails
	dst := []Dec128{One, FromInt64(2)}
	out, ok := d.AppendAllocateResidual(dst, nil, 2, 0, ROUND_BANK)
	if ok || len(out) != 2 || out[0] != One || out[1] != FromInt64(2) {
		t.Errorf("dst after a failure = %v, ok %v", out, ok)
	}
	out, ok = d.AppendSplitResidual(dst, 0, 2, 0, ROUND_BANK)
	if ok || len(out) != 2 {
		t.Errorf("dst after a failed split = %v, ok %v", out, ok)
	}
}

func TestAllocateResidualInvariants(t *testing.T) {
	r := rand.New(rand.NewSource(20261021))
	buf := make([]Dec128, 0, 64)

	for range 8000 {
		scale := uint8(r.Intn(int(MaxScale) + 1))
		amount := Dec128{coef: randCoef(r, 38), scale: uint8(r.Intn(int(scale) + 1))}
		if r.Intn(2) == 0 && !amount.coef.IsZero() {
			amount.state = state.Neg
		}
		n := 1 + r.Intn(40)
		ratios := make([]Dec128, n)
		for i := range ratios {
			ratios[i] = Dec128{coef: randCoef(r, 12), scale: uint8(r.Intn(4))}
		}
		res := r.Intn(n)
		mode := allModes[r.Intn(len(allModes))]

		buf = buf[:0]
		shares, ok := amount.AppendAllocateResidual(buf, ratios, scale, res, mode)
		if !ok {
			continue
		}
		if len(shares) != n {
			t.Fatalf("got %d shares for %d ratios", len(shares), n)
		}

		// the post-condition the convention exists for
		sumShares(t, shares, amount, "random residual allocation")

		// rerunning it gives the same answer, wherever it appends
		again, _ := amount.AllocateResidual(ratios, scale, res, mode)
		if !slices.Equal(shares, again) {
			t.Fatalf("residual allocation is not deterministic: %v vs %v", shares, again)
		}

		// Every share but the residual is within a quantum of its own exact proportion, and the residual is within
		// len(ratios)-1 quanta of its own - the rounding of all the others, which is what it absorbs. The
		// proportions are taken as exact rationals: a two-step reference through MulRound and DivRound rounds twice
		// itself, and at these magnitudes that is already worth more than a quantum.
		ratTotal := new(big.Rat)
		for _, w := range ratios {
			ratTotal.Add(ratTotal, new(big.Rat).SetFrac(bigAtScale(w, w.scale), bigPow10(w.scale)))
		}
		quantaRat := new(big.Rat).SetInt(new(big.Int).Abs(bigAtScale(amount, scale)))
		exactShare := func(i int) *big.Rat {
			p := new(big.Rat).SetFrac(bigAtScale(ratios[i], ratios[i].scale), bigPow10(ratios[i].scale))
			p.Quo(p, ratTotal)
			return p.Mul(p, quantaRat)
		}

		for i := range shares {
			got := new(big.Rat).SetInt(new(big.Int).Abs(bigAtScale(shares[i], scale)))
			if shares[i].IsNegative() != amount.IsNegative() && !shares[i].IsZero() {
				got.Neg(got) // the residual can come out on the other side of zero
			}
			diff := new(big.Rat).Sub(got, exactShare(i))
			diff.Abs(diff)

			allowed := big.NewRat(1, 1)
			if i == res {
				allowed = new(big.Rat).SetInt64(int64(n))
			}
			if diff.Cmp(allowed) > 0 {
				t.Fatalf("share %d of %d is %s quanta from its proportion (allowed %s): amount %s, scale %d, mode %s",
					i, n, diff.FloatString(3), allowed.RatString(), amount.StringFixed(), scale, mode)
			}
		}
	}
}

func TestSplitResidualInvariants(t *testing.T) {
	r := rand.New(rand.NewSource(20261022))
	for range 20000 {
		scale := uint8(r.Intn(int(MaxScale) + 1))
		amount := Dec128{coef: randCoef(r, 30), scale: uint8(r.Intn(int(scale) + 1))}
		if r.Intn(2) == 0 && !amount.coef.IsZero() {
			amount.state = state.Neg
		}
		n := 1 + r.Intn(50)
		res := r.Intn(n)
		mode := allModes[r.Intn(len(allModes))]

		shares, ok := amount.SplitResidual(n, scale, res, mode)
		if !ok {
			continue
		}
		sumShares(t, shares, amount, "random residual split")

		// every share but the residual is the same value, and it is the quotient rounded with mode
		want := amount.DivRound(FromInt64(int64(n)), scale, mode)
		for i := range shares {
			if i == res {
				continue
			}
			if !want.IsNaN() && !shares[i].Equal(want) {
				t.Fatalf("share %d of %s split %d ways under %s is %s, want %s",
					i, amount.StringFixed(), n, mode, shares[i].StringFixed(), want.StringFixed())
			}
			if i != res && i > 0 && res != i-1 && !shares[i].Equal(shares[i-1]) {
				t.Fatalf("shares %d and %d differ: %v", i-1, i, shares)
			}
		}
	}
}

// Beyond the stack scratch the allocation takes a working slice and must still be correct.
func TestAllocateResidualManyWays(t *testing.T) {
	ratios := make([]Dec128, allocScratch*3)
	for i := range ratios {
		ratios[i] = FromInt64(int64(i%7) + 1)
	}
	amount := FromString("1000000.00")
	for _, res := range []int{0, 1, len(ratios) / 2, len(ratios) - 1} {
		shares, ok := amount.AllocateResidual(ratios, 2, res, ROUND_HALF_AWAY_FROM_ZERO)
		if !ok || len(shares) != len(ratios) {
			t.Fatalf("wide residual allocation failed: ok %v, %d shares", ok, len(shares))
		}
		sumShares(t, shares, amount, "96-way residual allocation")
	}
}

// The largest amount and the largest weights, where the rounded shares can sum past the coefficient the whole fits in.
func TestAllocateResidualAtTheLimit(t *testing.T) {
	m := MaxAtScale(0)
	for _, res := range []int{0, 1, 2} {
		for _, mode := range []RoundingMode{ROUND_UP, ROUND_TOWARD_ZERO, ROUND_HALF_AWAY_FROM_ZERO, ROUND_AWAY_FROM_ZERO} {
			shares, ok := m.AllocateResidual([]Dec128{One, One, One}, 0, res, mode)
			if !ok {
				t.Fatalf("the largest coefficient three ways under %s failed", mode)
			}
			sumShares(t, shares, m, "max three ways")
		}
	}
	// the same at the finest scale, and as a split
	q := MaxAtScale(MaxScale)
	for _, mode := range []RoundingMode{ROUND_UP, ROUND_TOWARD_ZERO} {
		shares, ok := q.SplitResidual(7, MaxScale, 6, mode)
		if !ok {
			t.Fatalf("the largest coefficient at scale %d split seven ways under %s failed", MaxScale, mode)
		}
		sumShares(t, shares, q, "max split seven")
	}
}
