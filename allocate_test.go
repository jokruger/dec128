package dec128

import (
	"math/rand"
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
