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
