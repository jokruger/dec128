package dec128

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/jokruger/dec128/state"
)

// restoreConfig saves and restores every global this file touches. SetArithmeticRounding writes the loss policy too,
// so the policy must be restored last: deferred calls run in reverse order, hence this order of registration.
func restoreConfig(t *testing.T) {
	t.Helper()
	scale, mode, policy := DefaultScale(), ArithmeticRounding(), CurrentLossPolicy()
	t.Cleanup(func() {
		SetDefaultScale(scale)
		SetArithmeticRounding(mode)
		SetLossPolicy(policy)
	})
}

func TestLossPolicyNames(t *testing.T) {
	for p, want := range map[LossPolicy]string{
		LossRound:          "LossRound",
		LossNaNOnUnderflow: "LossNaNOnUnderflow",
		LossNaNOnInexact:   "LossNaNOnInexact",
		LossPolicy(99):     "LossPolicy(099)",
	} {
		if got := p.String(); got != want {
			t.Errorf("LossPolicy(%d).String() = %q, want %q", uint8(p), got, want)
		}
	}
	if !LossRound.IsValid() || !LossNaNOnInexact.IsValid() {
		t.Error("defined policies must be valid")
	}
	if (LossNaNOnInexact + 1).IsValid() || LossPolicy(255).IsValid() {
		t.Error("undefined policies must not be valid")
	}
}

func TestSetLossPolicyPanicsOnUndefined(t *testing.T) {
	restoreConfig(t)
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("SetLossPolicy(invalid) did not panic")
		}
		if err, ok := r.(error); !ok || err != state.InvalidLossPolicy.Error() {
			t.Errorf("panic = %v, want %v", r, state.InvalidLossPolicy.Error())
		}
	}()
	SetLossPolicy(LossPolicy(99))
}

func TestSetLossPolicyRoundTrips(t *testing.T) {
	restoreConfig(t)
	for _, p := range []LossPolicy{LossRound, LossNaNOnUnderflow, LossNaNOnInexact} {
		SetLossPolicy(p)
		if got := CurrentLossPolicy(); got != p {
			t.Errorf("CurrentLossPolicy() = %v, want %v", got, p)
		}
	}
}

// TestArithmeticRoundingWritesPolicy pins the deprecated spelling: SetArithmeticRounding(ROUND_NAN) must behave
// exactly as it did in v1.1, and any other mode must put the policy back to LossRound, so that code using only the
// old setter sees no change at all.
func TestArithmeticRoundingWritesPolicy(t *testing.T) {
	restoreConfig(t)

	SetArithmeticRounding(ROUND_NAN)
	if got := CurrentLossPolicy(); got != LossNaNOnInexact {
		t.Errorf("after SetArithmeticRounding(ROUND_NAN): policy = %v, want LossNaNOnInexact", got)
	}
	if got := ArithmeticRounding(); got != ROUND_NAN {
		t.Errorf("ArithmeticRounding() = %v, want ROUND_NAN", got)
	}
	if d := FromInt64(1).Div(FromInt64(3)); !d.IsNaN() || d.ErrorDetails() != state.Inexact.Error() {
		t.Errorf("1/3 under ROUND_NAN = %v (%v), want NaN(inexact)", d, d.ErrorDetails())
	}

	SetArithmeticRounding(ROUND_BANK)
	if got := CurrentLossPolicy(); got != LossRound {
		t.Errorf("after SetArithmeticRounding(ROUND_BANK): policy = %v, want LossRound", got)
	}
	if d := FromInt64(1).Div(FromInt64(3)); d.IsNaN() {
		t.Error("1/3 under ROUND_BANK must be a value")
	}
}

// lossCase is an operation that discards digits, in the three shapes the policy distinguishes: a product that keeps
// its significant digits, a product and a quotient that lose all of them, and a root.
type lossCase struct {
	name       string
	op         func() Dec128
	underflows bool // the rounded result is zero although no operand was
}

func lossCases(t *testing.T) []lossCase {
	t.Helper()
	f := func(s string) Dec128 {
		d := FromString(s)
		if d.IsNaN() {
			t.Fatalf("FromString(%q) = NaN(%v)", s, d.ErrorDetails())
		}
		return d
	}
	tiny := f("0.0000000001") // 1e-10; squared it is 1e-20, below the smallest representable value
	small := f("0.000000001") // 1e-9; squared it is 1e-18, representable exactly
	return []lossCase{
		{"Mul inexact", func() Dec128 { return f("1.2345678901").Mul(f("1.2345678901")) }, false},
		{"Mul underflow", func() Dec128 { return tiny.Mul(tiny) }, true},
		{"Mul exact", func() Dec128 { return small.Mul(small) }, false},
		{"Div inexact", func() Dec128 { return FromInt64(1).Div(FromInt64(3)) }, false},
		{"Div underflow", func() Dec128 { return FromInt64(1).Div(f("1000000000000000000000000000000")) }, true},
		{"Sqrt inexact", func() Dec128 { return FromInt64(2).Sqrt() }, false},
	}
}

func TestLossPolicyBehavior(t *testing.T) {
	restoreConfig(t)

	for _, c := range lossCases(t) {
		exact := c.name == "Mul exact"

		SetLossPolicy(LossRound)
		got := c.op()
		if got.IsNaN() {
			t.Errorf("%s under LossRound = NaN(%v), want a value", c.name, got.ErrorDetails())
		}
		if c.underflows && !got.IsZero() {
			t.Errorf("%s under LossRound = %s, want zero", c.name, got.StringFixed())
		}

		SetLossPolicy(LossNaNOnUnderflow)
		got = c.op()
		switch {
		case c.underflows:
			if !got.IsNaN() || got.ErrorDetails() != state.Underflow.Error() {
				t.Errorf("%s under LossNaNOnUnderflow = %v (%v), want NaN(underflow)", c.name, got, got.ErrorDetails())
			}
		default:
			if got.IsNaN() {
				t.Errorf("%s under LossNaNOnUnderflow = NaN(%v), want a value", c.name, got.ErrorDetails())
			}
		}

		SetLossPolicy(LossNaNOnInexact)
		got = c.op()
		switch {
		case exact:
			if got.IsNaN() {
				t.Errorf("%s under LossNaNOnInexact = NaN(%v), want a value", c.name, got.ErrorDetails())
			}
		default:
			if !got.IsNaN() || got.ErrorDetails() != state.Inexact.Error() {
				t.Errorf("%s under LossNaNOnInexact = %v (%v), want NaN(inexact)", c.name, got, got.ErrorDetails())
			}
		}
	}
}

// TestLossPolicyUnderflowIsNaN checks that an underflow propagates like any other NaN and is told apart from an
// overflow by its state, which is the whole point of giving it its own code.
func TestLossPolicyUnderflowIsNaN(t *testing.T) {
	restoreConfig(t)
	SetLossPolicy(LossNaNOnUnderflow)

	tiny := FromString("0.0000000001")
	u := tiny.Mul(tiny)
	if !u.IsNaN() || u.ErrorDetails() != state.Underflow.Error() {
		t.Fatalf("underflow = %v (%v)", u, u.ErrorDetails())
	}
	if u.IsZero() || u.IsNegative() || u.IsPositive() {
		t.Error("a NaN is neither zero, negative nor positive")
	}
	if got := u.Add(FromInt64(1)).Mul(FromInt64(2)); !got.IsNaN() || got.ErrorDetails() != state.Underflow.Error() {
		t.Errorf("underflow did not propagate: %v (%v)", got, got.ErrorDetails())
	}
	// an overflow is still an overflow, not an underflow
	big := MaxAtScale(0)
	if o := big.Mul(big); !o.IsNaN() || o.ErrorDetails() != state.Overflow.Error() {
		t.Errorf("max^2 = %v (%v), want NaN(overflow)", o, o.ErrorDetails())
	}
}

// TestSqrtUnderflowAtCoarseScale reaches the zero-root branch, which needs a default scale too coarse to hold the
// root: sqrt(x) > x for 0 < x < 1, so at MaxScale every representable radicand has a representable root.
func TestSqrtUnderflowAtCoarseScale(t *testing.T) {
	restoreConfig(t)
	SetDefaultScale(0)

	half := FromString("0.25")

	SetLossPolicy(LossRound)
	if got := half.Sqrt(); !got.IsZero() {
		t.Errorf("sqrt(0.25) at scale 0 under LossRound = %s, want 0", got.StringFixed())
	}

	SetLossPolicy(LossNaNOnUnderflow)
	if got := half.Sqrt(); !got.IsNaN() || got.ErrorDetails() != state.Underflow.Error() {
		t.Errorf("sqrt(0.25) at scale 0 under LossNaNOnUnderflow = %v (%v), want NaN(underflow)", got, got.ErrorDetails())
	}
}

// TestPerCallRoundingIgnoresPolicy pins the boundary: the *Round methods take their scale and mode per call and read
// no global, the loss policy included.
func TestPerCallRoundingIgnoresPolicy(t *testing.T) {
	restoreConfig(t)
	SetLossPolicy(LossNaNOnInexact)

	one, three := FromInt64(1), FromInt64(3)
	if got := one.DivRound(three, 4, ROUND_BANK); got.IsNaN() || got.StringFixed() != "0.3333" {
		t.Errorf("DivRound(1/3, 4, ROUND_BANK) = %v (%v), want 0.3333", got, got.ErrorDetails())
	}
	if got := FromInt64(2).SqrtRound(4, ROUND_BANK); got.IsNaN() || got.StringFixed() != "1.4142" {
		t.Errorf("SqrtRound(2, 4, ROUND_BANK) = %v (%v), want 1.4142", got, got.ErrorDetails())
	}
	x := FromString("1.2345678901")
	if got := x.MulRound(x, 4, ROUND_BANK); got.IsNaN() || got.StringFixed() != "1.5242" {
		t.Errorf("MulRound(x, 4, ROUND_BANK) = %v (%v), want 1.5242", got, got.ErrorDetails())
	}
	// ROUND_NAN as a per-call argument still refuses, independently of the policy
	SetLossPolicy(LossRound)
	if got := one.DivRound(three, 4, ROUND_NAN); !got.IsNaN() || got.ErrorDetails() != state.Inexact.Error() {
		t.Errorf("DivRound(1/3, 4, ROUND_NAN) = %v (%v), want NaN(inexact)", got, got.ErrorDetails())
	}
}

// TestMulAgainstBigUnderPolicies re-runs the scale-rule oracle across every combination of direction and policy, so
// that the policy branches are held to the same math/big reference as the rounding modes.
func TestMulAgainstBigUnderPolicies(t *testing.T) {
	restoreConfig(t)
	r := rand.New(rand.NewSource(20260911))
	for _, policy := range []LossPolicy{LossRound, LossNaNOnUnderflow, LossNaNOnInexact} {
		for _, mode := range allModes {
			SetArithmeticRounding(mode)
			SetLossPolicy(policy) // second: SetArithmeticRounding writes the policy too
			for range 3000 {
				x, y := randDec(r), randDec(r)
				exact := new(big.Int).Mul(bigAtScale(x, x.scale), bigAtScale(y, y.scale))
				checkFit(t, "Mul", x, y, x.Mul(y), exact, x.scale+y.scale, mode)
			}
		}
	}
}
